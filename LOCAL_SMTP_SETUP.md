# Local Development SMTP Setup

If you're testing on localhost and SMTP ports (587/465) are blocked, here are solutions for local development.

## Problem

Your local network/firewall is blocking outbound SMTP ports. This is common because:
- ISPs often block SMTP ports to prevent spam
- macOS/Windows firewalls may block outbound SMTP
- Corporate networks block SMTP ports

## Solutions for Local Development

### Option 1: Use Mailtrap (Recommended for Testing)

Mailtrap is a testing SMTP service that doesn't require real email delivery. Perfect for development!

**Setup:**

1. **Sign up for free**: https://mailtrap.io (free tier available)

2. **Get SMTP credentials** from Mailtrap inbox settings

3. **Update `.env.production` or `.env`:**
   ```env
   SMTP_HOST=smtp.mailtrap.io
   SMTP_PORT=2525
   SMTP_USER=your-mailtrap-username
   SMTP_PASS=your-mailtrap-password
   EMAIL_SENDER=noreply@test.com
   ```

4. **Restart email service:**
   ```bash
   docker compose -f docker-compose.freetier.yml restart email-service
   ```

5. **Check emails in Mailtrap inbox** - emails won't be delivered, just captured for testing

**Advantages:**
- No port blocking issues
- Free tier available
- Perfect for testing
- See all sent emails in web interface

### Option 2: Use SendGrid (Free Tier)

SendGrid offers a free tier with 100 emails/day and often works even when Gmail is blocked.

**Setup:**

1. **Sign up**: https://sendgrid.com (free tier: 100 emails/day)

2. **Create API Key**: Settings → API Keys → Create API Key

3. **Update environment:**
   ```env
   SMTP_HOST=smtp.sendgrid.net
   SMTP_PORT=587
   SMTP_USER=apikey
   SMTP_PASS=your-sendgrid-api-key
   EMAIL_SENDER=noreply@yourdomain.com
   ```

### Option 3: Use Mailgun (Free Tier)

Mailgun offers 5,000 emails/month free.

**Setup:**

1. **Sign up**: https://www.mailgun.com

2. **Get SMTP credentials** from Mailgun dashboard

3. **Update environment:**
   ```env
   SMTP_HOST=smtp.mailgun.org
   SMTP_PORT=587
   SMTP_USER=postmaster@your-domain.mailgun.org
   SMTP_PASS=your-mailgun-smtp-password
   EMAIL_SENDER=noreply@yourdomain.com
   ```

### Option 4: Configure macOS Firewall

If you're on macOS and want to use Gmail:

1. **System Settings** → **Network** → **Firewall**
2. **Options** → Allow incoming connections for Docker
3. **Check if outbound connections are blocked**

Or disable firewall temporarily for testing:
```bash
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate off
```

**Re-enable after testing:**
```bash
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate on
```

### Option 5: Use Docker Host Network Mode

This makes the container use the host's network directly:

**Update `docker-compose.freetier.yml` email-service section:**

```yaml
email-service:
  # ... existing config ...
  network_mode: "host"  # Add this line
  # Remove the ports section if using host network
```

**Note:** This may cause port conflicts if services use the same ports.

### Option 6: Use Port 465 (SSL) Instead

Sometimes port 465 works when 587 doesn't:

**Update `.env.production`:**
```env
SMTP_HOST=smtp.gmail.com
SMTP_PORT=465
SMTP_USE_SSL=true
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password
EMAIL_SENDER=your-email@gmail.com
```

Then rebuild and restart:
```bash
docker compose -f docker-compose.freetier.yml build email-service
docker compose -f docker-compose.freetier.yml restart email-service
```

## Quick Test After Configuration

```bash
# Test SMTP connectivity
./test-smtp-connectivity.sh

# Send a test notification
curl -X POST http://localhost:3000/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "58f8f5c0-8354-49b0-b297-433cebb2064f",
    "template_id": "71a258ea-e6f5-415a-9ca5-b077425f8344",
    "type": "email",
    "data": {}
  }'

# Check email service logs
docker compose -f docker-compose.freetier.yml logs email-service --tail 20 | grep smtp
```

## Recommended Approach

For **local development**, use **Mailtrap**:
- ✅ No port blocking issues
- ✅ Free tier
- ✅ Perfect for testing
- ✅ See all emails in web interface
- ✅ No real emails sent

For **production/EC2**, use **Gmail** or **SendGrid**:
- ✅ Real email delivery
- ✅ Better deliverability
- ✅ Production-ready

## Troubleshooting

If none of the above work:

1. **Check your network**: Try from a different network (mobile hotspot)
2. **Check ISP**: Some ISPs block all SMTP ports
3. **Use VPN**: VPN might bypass ISP restrictions
4. **Test from EC2**: Deploy to EC2 where ports are open

## Testing Without Real SMTP

You can also test the system flow without SMTP by mocking the email service, but that's beyond the scope of this guide.

