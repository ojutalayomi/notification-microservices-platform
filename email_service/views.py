# routes/email_routes.py
from fastapi import APIRouter, Depends, HTTPException, BackgroundTasks
from sqlalchemy.orm import Session
from db import get_db
from models import EmailMessage, EmailStatus
from schemas import EmailCreate, EmailResponse
from redis_queue import enqueue_email_job
from uuid import uuid4
from datetime import datetime

router = APIRouter(prefix="/emails", tags=["Email Service"])


@router.post("/", response_model=EmailResponse)
async def create_email(email: EmailCreate, db: Session = Depends(get_db)):
    """
    Receives a new email job request and queues it for processing.
    """
    new_email = EmailMessage(
        id=uuid4(),
        user_id=email.user_id,
        template_id=email.template_id,
        subject=email.subject,
        body=email.body,
        to_email=email.to_email,
        status=EmailStatus.queued,
        created_at=datetime.utcnow()
    )
    db.add(new_email)
    db.commit()
    db.refresh(new_email)

    # Push job to Redis queue
    enqueue_email_job(str(new_email.id))

    return new_email
