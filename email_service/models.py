# models.py
import enum
from uuid import uuid4
from datetime import datetime
from sqlalchemy import Column, String, Text, Enum, Integer, ForeignKey, DateTime, JSON
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.orm import relationship
from db import Base

class EmailStatus(str, enum.Enum):
    queued = "queued"
    processing = "processing"
    sent = "sent"
    failed = "failed"
    delivered = "delivered"
    bounced = "bounced"

class EmailMessage(Base):
    __tablename__ = "email_messages"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid4)
    request_id = Column(String(100), nullable=True, index=True)  # idempotency
    user_id = Column(UUID(as_uuid=True), nullable=False, index=True)
    template_code = Column(String(255), nullable=True, index=True)
    subject = Column(String(255), nullable=True)
    body = Column(Text, nullable=True)
    to_email = Column(String(255), nullable=False, index=True)
    status = Column(Enum(EmailStatus), default=EmailStatus.queued, nullable=False, index=True)
    retry_count = Column(Integer, default=0)
    metadata = Column(JSON, nullable=True)
    error_message = Column(Text, nullable=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    sent_at = Column(DateTime, nullable=True)

    logs = relationship("EmailLog", back_populates="email_message", cascade="all, delete-orphan")

class EmailLog(Base):
    __tablename__ = "email_logs"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid4)
    email_id = Column(UUID(as_uuid=True), ForeignKey("email_messages.id"), nullable=False)
    event = Column(String(50), nullable=False)
    timestamp = Column(DateTime, default=datetime.utcnow)
    metadata = Column(JSON, nullable=True)

    email_message = relationship("EmailMessage", back_populates="logs")
