# main.py
import os
import json
from fastapi import FastAPI, Depends, HTTPException, status
from fastapi.responses import JSONResponse
from db import Base, engine, get_db
from sqlalchemy.orm import Session
from dotenv import load_dotenv
from models import EmailMessage, EmailStatus
from schemas import EmailCreate, EmailResponse, StandardResponse
from queue import setup_infrastructure, publish_email_job
from uuid import uuid4
from datetime import datetime

load_dotenv()

Base.metadata.create_all(bind=engine)
setup_infrastructure()

app = FastAPI(title="email_service", version="1.0.0")

@app.get("/health")
def health():
    return {"status": "ok"}

@app.post("/internal/email/queue", response_model=StandardResponse, status_code=201)
def enqueue_email(email: EmailCreate, db: Session = Depends(get_db)):
    """
    Enqueue email job coming from API Gateway or Notification Orchestrator.
    This service stores the job in DB and pushes to RabbitMQ.
    """
    # idempotency: if request_id exists, try to find existing record
    if email.request_id:
        existing = db.query(EmailMessage).filter(EmailMessage.request_id == email.request_id).first()
        if existing:
            return StandardResponse(success=True, data=EmailResponse.from_orm(existing), message="already_queued")

    new_email = EmailMessage(
        id=uuid4(),
        request_id=email.request_id,
        user_id=email.user_id,
        template_code=email.template_code,
        subject=email.subject,
        body=email.body,
        to_email=str(email.to_email),
        status=EmailStatus.queued,
        created_at=datetime.utcnow()
    )
    db.add(new_email)
    db.commit()
    db.refresh(new_email)

    # publish to rabbitmq
    publish_email_job({
        "email_id": str(new_email.id),
        "user_id": str(new_email.user_id),
        "template_code": new_email.template_code,
        "to_email": new_email.to_email,
        "subject": new_email.subject,
        "body": new_email.body,
        "metadata": email.metadata or {}
    }, request_id=email.request_id, priority=getattr(email, "priority", 0))

    return StandardResponse(success=True, data=EmailResponse.from_orm(new_email), message="queued")
