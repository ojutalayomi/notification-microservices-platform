import json
import time
import pika
from datetime import datetime
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
        email_id = message["email_id"]
        retry_count = message.get("retry_count", 0)

        print(f"\n[worker] Received email job: {email_id} (retry {retry_count})")
        process_email(email_id, retry_count=retry_count)

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
    channel.queue_declare(queue=QUEUE_NAME, durable=True)
    channel.basic_qos(prefetch_count=1)
    channel.basic_consume(queue=QUEUE_NAME, on_message_callback=callback)

    print(f"[worker] Listening to queue: {QUEUE_NAME}")
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
