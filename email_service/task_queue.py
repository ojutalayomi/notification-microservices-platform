import os
import json
import pika
from dotenv import load_dotenv

load_dotenv()

RABBITMQ_URL = os.getenv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
QUEUE_NAME = "email_queue"

def get_connection():
    """
    Create a RabbitMQ connection.
    """
    params = pika.URLParameters(RABBITMQ_URL)
    return pika.BlockingConnection(params)


def setup_queue():
    """
    Create the email queue if it doesn't exist.
    """
    print("[queue] Setting up RabbitMQ queue...")
    conn = get_connection()
    channel = conn.channel()

    # Declare queue (creates if doesn't exist)
    channel.queue_declare(queue=QUEUE_NAME, durable=True)

    conn.close()
    print(f"[queue] Queue '{QUEUE_NAME}' is ready")


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
        exchange='',  # Default exchange
        routing_key=QUEUE_NAME,
        body=message,
        properties=pika.BasicProperties(
            delivery_mode=2,  # Persistent
        )
    )

    conn.close()
    print(f"[queue] Published email job: {email_data['email_id']}")
