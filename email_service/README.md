# Email Service

An email notification service that queues and sends emails.

## What This Does

1. **API receives email request** → Saves to database + Adds to queue
2. **Worker picks from queue** → Sends email via SMTP
3. **Database tracks status** → queued → processing → sent/failed
4. **Extensible**: Works with template service or orchestrators for dynamic email content.


## Setup Instructions

### Step 1: Install Dependencies

```bash
# Create virtual environment 
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install packages
pip install -r requirements.txt
```

### Step 2: Configure Environment Variables

Edit the `.env` file with your SMTP settings:

```env
# For Gmail (requires App Password, not regular password)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password
EMAIL_SENDER=noreply@yourapp.com

# Optional: Alternative SMTP configuration
# SMTP_PORT=465              # Use SSL/TLS directly (if port 587 is blocked)
# SMTP_USE_SSL=true         # Force SSL connection
# SMTP_USE_TLS=true         # Enable STARTTLS (default: true)
```

**How to get Gmail App Password:**
1. Go to Google Account → Security
2. Enable 2-Step Verification
3. Search "App passwords" and create one
4. Use that password in `.env`

**SMTP Configuration Options:**
- **Port 587 (STARTTLS)**: Default, automatically falls back to port 465 if needed
- **Port 465 (SSL/TLS)**: Direct SSL connection, use if port 587 is blocked
- **Automatic Fallback**: Service tries multiple connection methods automatically

For detailed SMTP configuration, troubleshooting, and provider-specific settings, see [SMTP_CONFIGURATION.md](../SMTP_CONFIGURATION.md)

### Step 3: Start Database & RabbitMQ

```bash
docker-compose up -d
```

This starts:
- PostgreSQL on port 5432
- RabbitMQ on port 5672
- RabbitMQ Management UI on http://localhost:15672 (guest/guest)

### Step 4: Start the API

```bash
python main.py
```

API will be available at: http://localhost:8000

Check health: http://localhost:8000/health

### Step 5: Start the Worker (in another terminal)

```bash
python worker.py
```

You should see:
```
[worker] Listening to queue: email.queue
[worker] Waiting for emails...
```

## Testing

### Send a test email

```bash
curl -X POST http://localhost:8000//email/queue \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "123e4567-e89b-12d3-a456-426614174000",
    "to_email": "test@example.com",
    "subject": "Test Email",
    "body": "<h1>Hello!</h1><p>This is a test email.</p>"
  }'
```

### Check what happens:

1. **API Terminal** - Should show:
   ```
   [api] Email saved to DB: <some-uuid>
   [queue] Published email job: <same-uuid>
   ```

2. **Worker Terminal** - Should show:
   ```
   [worker] Received email job: <uuid>
   [smtp] Connecting to smtp.gmail.com:587...
   [smtp] Email sent successfully!
   [worker] ✓ Email sent successfully!
   ```

3. **Check email status:**
   ```bash
   curl http://localhost:8000//email/<email-id>
   ```

### Check RabbitMQ UI

Go to http://localhost:15672 (guest/guest) and see:
- Queues → `email.queue`
- Message rates
- Messages waiting

## Understanding the Flow

### When you send POST /email/queue

**File: main.py**
```python
# 1. Create EmailMessage in database (status: queued)
new_email = EmailMessage(id=uuid4(), status=EmailStatus.queued, ...)
db.add(new_email)
db.commit()

# 2. Publish to RabbitMQ
publish_email_job({"email_id": str(new_email.id), ...})
```

### Worker processes it

**File: worker.py**
```python
# 1. Receive message from queue
message = json.loads(body)

# 2. Get email from database
email = db.query(EmailMessage).filter(...)

# 3. Update status to processing
email.status = EmailStatus.processing
db.commit()

# 4. Send via SMTP
send_email(email.to_email, email.subject, email.body)

# 5. Update status to sent
email.status = EmailStatus.sent
db.commit()
```

## Common Issues

### Worker not processing emails?
- Make sure worker.py is running
- Check RabbitMQ is running: `docker-compose ps`
- Check queue has messages: http://localhost:15672

### SMTP errors?
- Check your SMTP credentials in `.env`
- For Gmail, use App Password not regular password
- Test connectivity: Run `./test-smtp-connectivity.sh` from project root
- Try alternative port: Set `SMTP_PORT=465` and `SMTP_USE_SSL=true`
- Check firewall/security group: Ensure outbound port 587/465 is allowed
- See [SMTP_CONFIGURATION.md](../SMTP_CONFIGURATION.md) for detailed troubleshooting

### Database connection errors?
- Make sure PostgreSQL is running: `docker-compose ps`
- Check DATABASE_URL in `.env`
