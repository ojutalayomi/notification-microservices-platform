# HNG Stage Four - Notification Microservices Platform

A distributed microservices platform for sending notifications via email and push notifications. The system is fully dockerized and can be started with a single command.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Services](#services)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Usage Guide](#usage-guide)
- [API Documentation](#api-documentation)
- [Service Endpoints](#service-endpoints)
- [How It Works](#how-it-works)
- [Development](#development)
- [Troubleshooting](#troubleshooting)

## Overview

This platform provides a unified API for sending notifications through multiple channels:
- **Email Notifications** - Delivered via SMTP
- **Push Notifications** - Delivered via Firebase Cloud Messaging (FCM)

The system uses a microservices architecture with:
- **API Gateway** - Single entry point that orchestrates requests
- **User Service** - Manages user data and notification preferences
- **Template Service** - Manages notification templates
- **Email Service** - Handles email delivery asynchronously
- **Push Service** - Handles push notification delivery asynchronously
- **Message Queue** - RabbitMQ for asynchronous message processing

## Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────┐
│         API Gateway                  │
│    (NestJS - Port 3000)              │
└──────┬──────────────┬────────────────┘
       │              │
       ▼              ▼
┌──────────┐    ┌──────────────┐
│   User  │    │   Template   │
│ Service │    │   Service    │
└─────────┘    └──────────────┘
       │              │
       └──────┬───────┘
              │
              ▼
       ┌──────────────┐
       │   RabbitMQ   │
       │  (Message    │
       │   Broker)    │
       └──┬───────┬───┘
          │       │
          ▼       ▼
    ┌─────────┐ ┌──────────┐
    │  Email  │ │   Push   │
    │ Service │ │ Service  │
    └─────────┘ └──────────┘
```

### Data Flow

1. **Client** sends notification request to **API Gateway**
2. **API Gateway** fetches user data from **User Service**
3. **API Gateway** fetches template from **Template Service**
4. **API Gateway** validates user preferences
5. **API Gateway** publishes message(s) to **RabbitMQ** queue(s)
6. **Email Service** and/or **Push Service** consume messages and deliver notifications

## Services

### API Gateway (`api-gateway/`)
- **Technology**: NestJS (TypeScript)
- **Port**: 3000
- **Role**: Orchestrates notification requests, validates user preferences, fetches templates, and publishes to message queues
- **Database**: None (uses in-memory store for notification status)

### User Service (`user-service/`)
- **Technology**: NestJS (TypeScript) + Prisma
- **Port**: 3001
- **Role**: Manages user records, email addresses, push tokens, and notification preferences
- **Database**: PostgreSQL (`user-db` on port 5433)

### Template Service (`template-service/`)
- **Technology**: NestJS (TypeScript) + Prisma
- **Port**: 4000
- **Role**: Manages notification templates (subject, body, etc.)
- **Database**: PostgreSQL (`template-db` on port 5434)

### Email Service (`email_service/`)
- **Technology**: FastAPI (Python)
- **Port**: 8000
- **Role**: Consumes email messages from RabbitMQ and sends emails via SMTP
- **Database**: PostgreSQL (`email-db` on port 5436)
- **Features**: Circuit breaker, retry mechanism, email status tracking

### Push Service (`push-service/`)
- **Technology**: Go (Gin framework)
- **Port**: 8080
- **Role**: Consumes push messages from RabbitMQ and sends push notifications via FCM
- **Database**: PostgreSQL (`push-db` on port 5435)
- **Cache**: Redis (port 6379)
- **Features**: Device token management, retry mechanism, dead letter queue

### Infrastructure Services

- **RabbitMQ**: Message broker (ports 5672, 15672 for management UI)
- **PostgreSQL**: 4 separate database instances (one per service)
- **Redis**: Caching for push service (port 6379)

## Prerequisites

- **Docker Engine 24+** with BuildKit enabled (default on recent Docker Desktop installs)
- **Docker Compose v2**
- **Firebase Service Account** (for push notifications) - Place at `push-service/service-account.json`

## Deployment Options

- **Local Development**: Use `docker-compose.yml` (see [Quick Start](#quick-start))
- **AWS Free Tier**: Use `docker-compose.freetier.yml` - Optimized for t3.micro (1GB RAM) - See [DEPLOYMENT_FREETIER.md](./DEPLOYMENT_FREETIER.md)
- **AWS Production**: Use `docker-compose.prod.yml` - Recommended for t3.medium+ - See [DEPLOYMENT.md](./DEPLOYMENT.md)

## Quick Start

### 1. One-Time Setup

Ensure you have a Firebase service account JSON file for push notifications:

```bash
# Place your Firebase service account file at:
push-service/service-account.json
```

### 2. Start All Services

```bash
# From the repository root
docker compose up --build
```

This will:
- Download all required images
- Build all service containers
- Run database migrations
- Start all services and infrastructure

### 3. Verify Services Are Running

Check service health:

```bash
# API Gateway
curl http://localhost:3000/api/v1/health

# User Service
curl http://localhost:3001/api/v1/health

# Template Service
curl http://localhost:4000/api/v1/health

# Email Service
curl http://localhost:8000/health

# Push Service
curl http://localhost:8080/health
```

### 4. Stop Services

```bash
# Stop and remove containers (keeps database data)
docker compose down

# Stop and remove everything including volumes (deletes all data)
docker compose down -v
```

## Usage Guide

### Step 1: Create a User

```bash
curl -X POST http://localhost:3001/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com",
    "email_enabled": true,
    "push_enabled": true,
    "push_token": "fcm-device-token-here"
  }'
```

Response:
```json
{
  "success": true,
  "data": {
    "id": "user-uuid",
    "name": "John Doe",
    "email": "john@example.com",
    ...
  }
}
```

### Step 2: Create a Template

```bash
curl -X POST http://localhost:4000/api/v1/templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "welcome",
    "subject": "Welcome to our platform!",
    "body": "Hi {{name}}, welcome to our amazing platform!"
  }'
```

Response:
```json
{
  "success": true,
  "data": {
    "id": "template-uuid",
    "name": "welcome",
    "subject": "Welcome to our platform!",
    "body": "Hi {{name}}, welcome to our amazing platform!"
  }
}
```

### Step 3: Send a Notification

```bash
curl -X POST http://localhost:3000/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-uuid-from-step-1",
    "template_id": "template-uuid-from-step-2",
    "type": "both",
    "data": {
      "custom_field": "value"
    }
  }'
```

Response:
```json
{
  "success": true,
  "message": "Notification queued successfully",
  "data": {
    "notification_id": "notification-uuid",
    "user_id": "user-uuid",
    "type": "both",
    "status": "queued",
    "created_at": "2024-01-01T00:00:00.000Z"
  }
}
```

### Step 4: Check Notification Status

```bash
curl http://localhost:3000/api/v1/notifications/{notification_id}
```

## API Documentation

### API Gateway Endpoints

#### Send Notification

**POST** `/api/v1/notifications`

Request Body:
```json
{
  "user_id": "string (required)",
  "template_id": "string (required)",
  "type": "email | push | both (required)",
  "data": {
    "key": "value"
  },
  "priority": "high | normal | low (optional)"
}
```

Response: `202 Accepted`
```json
{
  "success": true,
  "message": "Notification queued successfully",
  "data": {
    "notification_id": "uuid",
    "user_id": "uuid",
    "type": "both",
    "status": "queued",
    "created_at": "ISO 8601 timestamp"
  }
}
```

#### Get Notification Status

**GET** `/api/v1/notifications/:id`

Response:
```json
{
  "success": true,
  "message": "Notification status retrieved",
  "data": {
    "notification_id": "uuid",
    "user_id": "uuid",
    "type": "both",
    "status": "queued | sent | failed",
    "created_at": "ISO 8601 timestamp"
  }
}
```

#### List All Notifications

**GET** `/api/v1/notifications?page=1&limit=10`

Response:
```json
{
  "success": true,
  "message": "Notifications retrieved successfully",
  "data": [...],
  "meta": {
    "total": 100,
    "limit": 10,
    "page": 1,
    "total_pages": 10,
    "has_next": true,
    "has_previous": false
  }
}
```

### Notification Types

- `email` - Send email notification only
- `push` - Send push notification only
- `both` - Send both email and push notifications

## Service Endpoints

| Service           | URL                              | Description                    |
|-------------------|----------------------------------|--------------------------------|
| API Gateway       | http://localhost:3000/api/v1     | Main entry point               |
| User Service      | http://localhost:3001/api/v1     | User management                |
| Template Service  | http://localhost:4000/api/v1     | Template management            |
| Email Service     | http://localhost:8000/docs       | FastAPI Swagger documentation  |
| Push Service      | http://localhost:8080/swagger/  | Swagger UI for push API        |
| RabbitMQ UI       | http://localhost:15672           | RabbitMQ Management UI         |

**RabbitMQ Credentials**: `guest` / `guest`

## How It Works

### Notification Flow

1. **Client Request** → API Gateway receives POST `/api/v1/notifications`

2. **User Validation** → API Gateway calls User Service to:
   - Fetch user data (email, push_token, preferences)
   - Validate user exists
   - Check if user has enabled the requested notification type

3. **Template Fetching** → API Gateway calls Template Service to:
   - Fetch template by ID
   - Get subject and body content

4. **Message Publishing** → API Gateway publishes to RabbitMQ:
   - Exchange: `notifications.direct`
   - Routing Keys: `email` and/or `push`
   - Queues: `email.queue` and/or `push.queue`

5. **Message Processing**:
   - **Email Service** consumes from `email.queue`:
     - Creates email record in database
     - Sends email via SMTP
     - Updates status (queued → processing → sent/failed)
   
   - **Push Service** consumes from `push.queue`:
     - Gets device tokens from database
     - Sends push notification via FCM
     - Updates status and handles retries

### Message Queue Architecture

```
notifications.direct (Exchange)
├── email (routing key) → email.queue → Email Service Worker
└── push (routing key) → push.queue → Push Service Worker
```

### Error Handling

- **User not found**: Returns 404
- **User preferences disabled**: Returns 400
- **Template not found**: Uses default template
- **Queue failures**: Messages go to dead letter queue
- **Retry mechanism**: Automatic retries with exponential backoff

## Development

### Running Services Locally

To run a single service locally while keeping dependencies in Docker:

1. Comment out the service in `docker-compose.yml`
2. Start other services: `docker compose up`
3. Run the service locally with proper environment variables

### Viewing Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f api-gateway
docker compose logs -f email-service
docker compose logs -f push-service
```

### Database Migrations

- **NestJS services (User, Template)**: Prisma migrations run automatically on container start
- **Push Service**: SQL migrations in `push-service/migrations/` run automatically
- **Email Service**: Tables created automatically via SQLAlchemy

### Rebuilding After Code Changes

After modifying code, rebuild the affected services:

```bash
# Rebuild all services
docker compose up --build

# Rebuild specific service
docker compose build email-service
docker compose up email-service
```

### Environment Variables

Key environment variables are defined in `docker-compose.yml`. To override:

1. Create a `.env` file in the repository root
2. Add your overrides:
   ```
   RABBITMQ_DEFAULT_USER=myuser
   RABBITMQ_DEFAULT_PASS=mypass
   ```

## Troubleshooting

### Services Not Starting

1. **Check Docker is running**: `docker ps`
2. **Check logs**: `docker compose logs <service-name>`
3. **Check health**: `curl http://localhost:<port>/health`

### Email Not Sending

1. **Check SMTP configuration**: Email service needs SMTP credentials
2. **Check worker logs**: `docker compose logs -f email-service`
3. **Check RabbitMQ**: Verify messages in queue at http://localhost:15672
4. **Check email status**: Query email service database

### Push Notifications Not Working

1. **Verify Firebase credentials**: Check `push-service/service-account.json` exists
2. **Check device registration**: Ensure device tokens are registered
3. **Check push service logs**: `docker compose logs -f push-service`
4. **Check FCM configuration**: Verify service account has FCM permissions

### RabbitMQ Connection Issues

1. **Check RabbitMQ is healthy**: `docker compose ps rabbitmq`
2. **Check RabbitMQ UI**: http://localhost:15672 (guest/guest)
3. **Verify queue bindings**: Check exchanges and queues in RabbitMQ UI
4. **Check network**: All services should be on `hng-network`

### Database Connection Issues

1. **Check database is running**: `docker compose ps`
2. **Check connection strings**: Verify DATABASE_URL in docker-compose.yml
3. **Check database logs**: `docker compose logs <db-name>`
4. **Verify ports**: Ensure no port conflicts

### Common Issues

**Port already in use**:
```bash
# Find process using port
lsof -i :3000

# Kill process or change port in docker-compose.yml
```

**Volume permission issues**:
```bash
# Reset volumes
docker compose down -v
docker compose up --build
```

**Build cache issues**:
```bash
# Rebuild without cache
docker compose build --no-cache
docker compose up
```

## Network Architecture

All services communicate via Docker's internal network `hng-network`:

- Services reach each other by container name (e.g., `http://user-service:3001`)
- External access via localhost ports (e.g., `http://localhost:3000`)
- Databases are only accessible from within the Docker network

## Production Considerations

⚠️ **This setup is for development only. For production:**

1. **Security**:
   - Change all default passwords
   - Use secrets management
   - Enable TLS/SSL
   - Restrict network access

2. **Scalability**:
   - Use Redis for notification status (not in-memory)
   - Scale workers horizontally
   - Use connection pooling
   - Implement rate limiting

3. **Monitoring**:
   - Add health checks
   - Set up logging aggregation
   - Monitor queue depths
   - Track delivery rates

4. **Reliability**:
   - Configure proper retry policies
   - Set up dead letter queues
   - Implement circuit breakers
   - Add alerting

## License

[Add your license here]

## Contributing

[Add contribution guidelines here]

---

**Happy coding!** 🚀
