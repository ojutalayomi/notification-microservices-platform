# worker.py
import os
import json
import time
import pika
from dotenv import load_dotenv
from db import SessionLocal
from models import EmailMessage, EmailStatus
from services import send_email
from queue import params, EXCHANGE, DLX_EXCHANGE, EMAIL_ROUTING_KEY
from sqlalchemy.orm import Session
from datetime import datetime

load_dotenv()
MAX_RETRIES = int(os.getenv("MAX_RETRIES", "5"))

def process_email_message(body, properties):
    payload = json.loads(body)
    email_id = payload.get("email_id")
    db: Session = SessionLocal()
    try:
        email = db.query(EmailMessage).filter(EmailMessage.id == email_id).first()
        if not email:
            print(f"[worker] email not found in db {email_id}")
            return True  # ack - nothing to do

        # idempotency check: if already sent, ack
        if email.status in (EmailStatus.sent, EmailStatus.delivered):
            print(f"[worker] email {email_id} already sent")
            return True

        # mark processing
        email.status = EmailStatus.processing
        db.commit()

        # Send the email (replace with provider integration)
        send_email(to=email.to_email, subject=email.subject or "", body=email.body or "")

        email.status = EmailStatus.sent
        email.sent_at = datetime.utcnow()
        db.commit()
        print(f"[worker] email {email_id} sent")
        return True

    except Exception as exc:
        # retry logic
        retry_count = 0
        if properties and properties.headers:
            retry_count = int(properties.headers.get("retry_count", 0))
        retry_count += 1

        email.retry_count = retry_count
        email.status = EmailStatus.failed
        email.error_message = str(exc)
        db.commit()

        if retry_count >= MAX_RETRIES:
            print(f"[worker] moving {email_id} to dead-letter after {retry_count} retries")
            return False  # signal to nack and let Rabbit route to DLQ (or handle explicit)
        else:
            backoff_seconds = (2 ** retry_count)
            print(f"[worker] retrying {email_id} in {backoff_seconds}s (attempt {retry_count})")
            time.sleep(backoff_seconds)
            # Republish with increased header
            conn = pika.BlockingConnection(params)
            ch = conn.channel()
            headers = properties.headers or {}
            headers["retry_count"] = retry_count
            ch.basic_publish(
                exchange=EXCHANGE,
                routing_key=EMAIL_ROUTING_KEY,
                body=body,
                properties=pika.BasicProperties(
                    delivery_mode=2,
                    headers=headers
                )
            )
            conn.close()
            return True
    finally:
        db.close()

def callback(ch, method, properties, body):
    try:
        ok = process_email_message(body, properties)
        if ok:
            ch.basic_ack(delivery_tag=method.delivery_tag)
        else:
            # nack without requeue to allow DLX to capture (ensure queue DLX configured)
            ch.basic_nack(delivery_tag=method.delivery_tag, requeue=False)
    except Exception as e:
        print("Fatal worker error:", e)
        ch.basic_nack(delivery_tag=method.delivery_tag, requeue=False)

def run_worker():
    conn = pika.BlockingConnection(params)
    ch = conn.channel()
    ch.basic_qos(prefetch_count=1)
    ch.basic_consume(queue="email.queue", on_message_callback=callback)
    print("[worker] waiting for messages...")
    ch.start_consuming()

if __name__ == "__main__":
    run_worker()
