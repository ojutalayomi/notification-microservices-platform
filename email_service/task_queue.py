import os
import json
import time
import pika
from dotenv import load_dotenv

load_dotenv()

RABBITMQ_URL = os.getenv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
QUEUE_NAME = "email.queue"

def get_connection(max_attempts: int = 10, delay_seconds: int = 3):
    """
    Create a RabbitMQ connection with basic retry logic to tolerate startup delays.
    """
    params = pika.URLParameters(RABBITMQ_URL)

    attempt = 1
    while True:
        try:
            return pika.BlockingConnection(params)
        except pika.exceptions.AMQPConnectionError as exc:
            if attempt >= max_attempts:
                raise

            wait_time = delay_seconds * attempt
            print(
                f"[queue] Unable to reach RabbitMQ (attempt {attempt}/{max_attempts}): {exc}. "
                f"Retrying in {wait_time} seconds..."
            )
            time.sleep(wait_time)
            attempt += 1


def setup_queue():
    """
    Create the email queue and exchange if they don't exist.
    Sets up notifications.direct exchange and binds email.queue to it.
    """
    print("[queue] Setting up RabbitMQ queue and exchange...")
    conn = get_connection()
    channel = conn.channel()

    # Declare exchange (creates if doesn't exist)
    channel.exchange_declare(exchange="notifications.direct", exchange_type="direct", durable=True)
    
    # Declare queue (creates if doesn't exist)
    channel.queue_declare(queue=QUEUE_NAME, durable=True)
    
    # Bind queue to exchange with routing key "email"
    channel.queue_bind(exchange="notifications.direct", queue=QUEUE_NAME, routing_key="email")

    conn.close()
    print(f"[queue] Exchange 'notifications.direct' and queue '{QUEUE_NAME}' are ready")


def publish_email_job(email_data: dict):
    """
    Publish an email job to the queue with persistent delivery.

    Args:
        email_data: dict with email_id, to_email, subject, body
    """
    conn = get_connection()
    channel = conn.channel()

    message = json.dumps(email_data)

    # delivery_mode=2 makes the message persistent
    channel.basic_publish(
        exchange='notifications.direct',  # Default exchange
        routing_key=QUEUE_NAME,
        body=message,
        properties=pika.BasicProperties(
            delivery_mode=2,  # Persistent
        )
    )

    conn.close()
    print(f"[queue] Published email job: {email_data['email_id']}")
