# schemas.py
from pydantic import BaseModel, EmailStr, HttpUrl
from typing import Optional, Dict, Any
from uuid import UUID
from datetime import datetime
from enum import Enum

class NotificationType(str, Enum):
    email = "email"
    push = "push"

class UserData(BaseModel):
    name: str
    link: Optional[HttpUrl]
    meta: Optional[Dict[str, Any]] = None

class NotificationCreate(BaseModel):
    notification_type: NotificationType
    user_id: UUID
    template_code: str
    variables: UserData
    request_id: str  # idempotency key
    priority: Optional[int] = 0
    metadata: Optional[Dict[str, Any]] = None

class EmailCreate(BaseModel):
    user_id: UUID
    template_code: str
    to_email: EmailStr
    subject: Optional[str]
    body: Optional[str]
    request_id: Optional[str]
    metadata: Optional[Dict[str, Any]] = None

class EmailResponse(BaseModel):
    id: UUID
    request_id: Optional[str]
    user_id: UUID
    template_code: Optional[str]
    subject: Optional[str]
    to_email: EmailStr
    status: str
    retry_count: int
    created_at: datetime
    updated_at: datetime

    class Config:
        orm_mode = True

class PaginationMeta(BaseModel):
    total: int
    limit: int
    page: int
    total_pages: int
    has_next: bool
    has_previous: bool

class StandardResponse(BaseModel):
    success: bool
    data: Optional[Any] = None
    error: Optional[str] = None
    message: str
    meta: Optional[PaginationMeta] = None
