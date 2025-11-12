from fastapi import FastAPI, Depends, HTTPException
from sqlalchemy.orm import Session
from uuid import uuid4
from datetime import datetime

from db import Base, engine, get_db
from models import EmailMessage, EmailStatus
from schemas import EmailCreate, EmailResponse, StandardResponse
from task_queue import setup_queue, publish_email_job


print("[startup] Creating database tables...")
Base.metadata.create_all(bind=engine)


print("[startup] Setting up RabbitMQ queue...")
setup_queue()


app = FastAPI(
    title="Email Service",
    description="Simple email notification service",
    version="1.0.0"
)

@app.get("/health")
def health_check():
    """
    Health check endpoint - returns OK if service is running.
    """
    return {
        "status": "ok",
        "service": "email-service"
    }

@app.post("/email/queue", response_model=StandardResponse, status_code=201)
def queue_email(email: EmailCreate, db: Session = Depends(get_db)):
    """
    Queue a new email for sending.
    
    This endpoint:
    1. Saves email to database with status 'queued'
    2. Publishes job to RabbitMQ
    3. Returns the email details
    
    The worker will pick it up and actually send it.
    """
    try:
        # Create new email record
        new_email = EmailMessage(
            id=uuid4(),
            user_id=email.user_id,
            to_email=str(email.to_email),
            subject=email.subject,
            body=email.body,
            status=EmailStatus.queued,
            created_at=datetime.utcnow()
        )

        db.add(new_email)
        db.commit()
        db.refresh(new_email)

        print(f"[api] Email saved to DB: {new_email.id}")

        # Publish to queue
        publish_email_job({
            "email_id": str(new_email.id),
            "to_email": new_email.to_email,
            "subject": new_email.subject,
            "body": new_email.body
        })

        # Return response based on DB object (not request object!)
        return StandardResponse(
            success=True,
            data=EmailResponse.model_validate(new_email, from_attributes=True),
            message="Email queued successfully"
        )

    except Exception as e:
        db.rollback()
        print(f"[api] Error queueing email: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))
    
    
@app.get("/email/{email_id}", response_model=StandardResponse)
def get_email_status(email_id: str, db: Session = Depends(get_db)):
    """
    Get the status of an email by its ID.
    """
    email = db.query(EmailMessage).filter(EmailMessage.id == email_id).first()
    
    if not email:
        raise HTTPException(status_code=404, detail="Email not found")
    
    return StandardResponse(
        success=True,
        data = EmailResponse.model_validate(email, from_attributes=True)
,
        message="Email found"
    )

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)