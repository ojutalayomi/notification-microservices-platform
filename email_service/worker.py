import json
import time
import pika
from datetime import datetime
from uuid import uuid4
import pybreaker

from db import SessionLocal
from models import EmailMessage, EmailStatus
from services import send_email
from task_queue import get_connection, QUEUE_NAME, publish_email_job

# Circuit breaker configuration
breaker = pybreaker.CircuitBreaker(
    fail_max=3,           # 3 failures in a row opens the circuit
    reset_timeout=30      # seconds before trying again
)

# Retry configuration
MAX_RETRIES = 5
BASE_DELAY = 2  # seconds for exponential backoff


def process_email(email_id: str, retry_count: int = 0):
    db = SessionLocal()
    try:
        email = db.query(EmailMessage).filter(EmailMessage.id == email_id).first()
        if not email:
            print(f"[worker] Email {email_id} not found")
            return

        if email.status == EmailStatus.sent:
            print(f"[worker] Email {email_id} already sent, skipping")
            return

        email.status = EmailStatus.processing
        db.commit()
        print(f"[worker] Processing email {email_id} to {email.to_email}")

        try:
            # Circuit breaker wraps the send_email function
            breaker.call(send_email, email.to_email, email.subject, email.body)

            email.status = EmailStatus.sent
            email.sent_at = datetime.utcnow()
            db.commit()
            print(f"[worker] ✓ Email {email_id} sent successfully!")

        except pybreaker.CircuitBreakerError:
            # Circuit is OPEN, skip sending
            print(f"[worker] Circuit breaker OPEN, skipping email {email_id}")
            email.status = EmailStatus.failed
            email.error_message = "Circuit breaker open"
            db.commit()

        except Exception as e:
            # Failed sending email
            print(f"[worker] ✗ Email {email_id} failed: {e}")
            if retry_count < MAX_RETRIES:
                # exponential backoff before retry
                delay = BASE_DELAY ** retry_count
                print(f"[worker] Retrying in {delay}s (retry {retry_count + 1})")
                time.sleep(delay)
                
                # Requeue the email with incremented retry count
                publish_email_job({
                    "email_id": str(email.id),
                    "retry_count": retry_count + 1
                })

                email.status = EmailStatus.queued
                db.commit()
            else:
                email.status = EmailStatus.failed
                email.error_message = str(e)
                db.commit()
                print(f"[worker] Email {email_id} failed after {MAX_RETRIES} retries")

    finally:
        db.close()


def callback(ch, method, properties, body):
    try:
        message = json.loads(body)
        
        # Check if this is an API Gateway message or internal retry message
        if "email_id" in message:
            # Internal retry message format
            email_id = message["email_id"]
            retry_count = message.get("retry_count", 0)

            print(f"\n[worker] Received email job: {email_id} (retry {retry_count})")
            process_email(email_id, retry_count=retry_count)
        else:
            # API Gateway message format: {notification_id, user_id, email, name, template, ...}
            notification_id = message.get("notification_id")
            user_id = message.get("user_id")
            email = message.get("email")
            template = message.get("template", {})
            
            if not email or not template:
                print(f"[worker] Invalid API Gateway message format: missing email or template")
                ch.basic_ack(delivery_tag=method.delivery_tag)
                return
            
            # Extract subject and body from template
            subject = template.get("subject", "Notification")
            # Template service returns 'html_body', not 'body'
            body_content = template.get("html_body") or template.get("body", "You have a new notification")
            
            # Get template variables and data for substitution
            template_variables = template.get("variables", [])
            notification_data = message.get("data", {})
            
            # Substitute template variables (e.g., {{name}} -> actual value)
            if template_variables and notification_data:
                for var_name in template_variables:
                    placeholder = f"{{{{{var_name}}}}}"
                    value = notification_data.get(var_name, "")
                    body_content = body_content.replace(placeholder, str(value))
                    subject = subject.replace(placeholder, str(value))
            
            print(f"\n[worker] Received API Gateway notification: {notification_id} for user: {user_id}")
            print(f"[worker] Template: {template.get('name', 'unknown')}, Subject: {subject}")
            
            # Create email record in database
            db = SessionLocal()
            try:
                new_email = EmailMessage(
                    id=uuid4(),
                    user_id=user_id,
                    to_email=email,
                    subject=subject,
                    body=body_content,
                    status=EmailStatus.queued,
                    created_at=datetime.utcnow()
                )
                
                db.add(new_email)
                db.commit()
                db.refresh(new_email)
                
                print(f"[worker] Created email record: {new_email.id}")
                
                # Process the email
                process_email(str(new_email.id), retry_count=0)
                
            except Exception as e:
                print(f"[worker] Error creating email record: {e}")
                db.rollback()
            finally:
                db.close()

        ch.basic_ack(delivery_tag=method.delivery_tag)
        print(f"[worker] Message acknowledged\n")

    except Exception as e:
        print(f"[worker] Error processing message: {e}")
        ch.basic_ack(delivery_tag=method.delivery_tag)


def start_worker():
    print("=" * 50)
    print("EMAIL WORKER STARTING")
    print("=" * 50)

    connection = get_connection()
    channel = connection.channel()
    
    # Declare exchange
    channel.exchange_declare(exchange="notifications.direct", exchange_type="direct", durable=True)
    
    # Declare queue
    channel.queue_declare(queue=QUEUE_NAME, durable=True)
    
    # Bind queue to exchange with routing key "email"
    channel.queue_bind(exchange="notifications.direct", queue=QUEUE_NAME, routing_key="email")
    
    channel.basic_qos(prefetch_count=1)
    channel.basic_consume(queue=QUEUE_NAME, on_message_callback=callback)

    print(f"[worker] Listening to queue: {QUEUE_NAME} (from notifications.direct exchange)")
    print("[worker] Waiting for emails... Press CTRL+C to exit\n")

    try:
        channel.start_consuming()
    except KeyboardInterrupt:
        print("\n[worker] Shutting down...")
        channel.stop_consuming()
    finally:
        connection.close()


if __name__ == "__main__":
    start_worker()
