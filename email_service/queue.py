# rabbitmq_queue.py
import os
import json
import pika
from dotenv import load_dotenv

load_dotenv()

RABBIT_URL = os.getenv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/")
EXCHANGE = "notifications.direct"
EMAIL_ROUTING_KEY = "email"
EMAIL_QUEUE = "email.queue"
DLX_EXCHANGE = "notifications.dlx"
DLQ_QUEUE = "failed.queue"

params = pika.URLParameters(RABBIT_URL)

def _connection():
    return pika.BlockingConnection(params)

def setup_infrastructure():
    """Call once at startup to ensure exchanges/queues exist with DLQ configured."""
    conn = _connection()
    ch = conn.channel()
    ch.exchange_declare(exchange=EXCHANGE, exchange_type='direct', durable=True)
    ch.exchange_declare(exchange=DLX_EXCHANGE, exchange_type='direct', durable=True)

    # email queue with DLX
    ch.queue_declare(queue=EMAIL_QUEUE, durable=True, arguments={
        'x-dead-letter-exchange': DLX_EXCHANGE,
        'x-dead-letter-routing-key': 'failed'
    })
    ch.queue_bind(queue=EMAIL_QUEUE, exchange=EXCHANGE, routing_key=EMAIL_ROUTING_KEY)

    # dead letter queue
    ch.queue_declare(queue=DLQ_QUEUE, durable=True)
    ch.queue_bind(queue=DLQ_QUEUE, exchange=DLX_EXCHANGE, routing_key='failed')

    conn.close()

def publish_email_job(payload: dict, request_id: str = None, priority: int = 0):
    """
    payload: dict (must include email_id or the full email data)
    set headers: request_id (for idempotency), retry_count (int)
    """
    conn = _connection()
    ch = conn.channel()
    properties = pika.BasicProperties(
        delivery_mode=2,  # persistent
        headers={"request_id": request_id or "", "retry_count": 0},
        priority=priority
    )
    ch.basic_publish(
        exchange=EXCHANGE,
        routing_key=EMAIL_ROUTING_KEY,
        body=json.dumps(payload),
        properties=properties
    )
    conn.close()
