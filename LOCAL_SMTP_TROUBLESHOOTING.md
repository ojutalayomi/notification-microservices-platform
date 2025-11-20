# Local SMTP Troubleshooting Guide

If you're testing on localhost (your PC) and SMTP connections are failing, this guide will help you resolve the issue.

## Quick Diagnosis

Run the connectivity test:
```bash
./test-smtp-connectivity.sh
```

## Common Local Issues

### 1. macOS Firewall Blocking Outbound SMTP

**Symptoms:**
- Connection timeout
- Port 587/465 not reachable from host
- Works from some networks but not others

**Solution:**

#### Check macOS Firewall Status
```bash
# Check if firewall is enabled
/usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate

# Check firewall mode
/usr/libexec/ApplicationFirewall/socketfilterfw --getblockall
```

#### Allow Outbound SMTP (if firewall is blocking)
1. Go to **System Settings** → **Network** → **Firewall**
2. Click **Options** (if firewall is on)
3. Check if Docker or Python is blocked
4. Add exception for outbound connections on ports 587/465

#### Alternative: Temporarily Disable Firewall (for testing only)
```bash
# Disable firewall (NOT recommended for production)
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate off

# Re-enable after testing
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate on
```

### 2. ISP Blocking SMTP Ports

Many ISPs block outbound SMTP ports (25, 587, 465) to prevent spam.

**Solutions:**

#### Option A: Use Port 465 (SSL/TLS)
Port 465 is sometimes less restricted:
```env
SMTP_PORT=465
SMTP_USE_SSL=true
```

#### Option B: Use a Different SMTP Provider
Use providers that offer alternative ports or web APIs:

**SendGrid:**
```env
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USER=apikey
SMTP_PASS=your-sendgrid-api-key
```

**Mailgun:**
```env
SMTP_HOST=smtp.mailgun.org
SMTP_PORT=587
SMTP_USER=postmaster@your-domain.mailgun.org
SMTP_PASS=your-mailgun-password
```

#### Option C: Use VPN
Connect to a VPN to bypass ISP restrictions.

### 3. Docker Network Configuration

Docker's network might be blocking outbound connections.

**Solution: Use Host Network Mode (for testing)**

Update `docker-compose.freetier.yml` email-service section:

```yaml
email-service:
  # ... existing config ...
  network_mode: host  # Add this line
  # Remove or comment out the networks section
  # networks:
  #   - hng-network
```

**Note:** This makes the container use the host's network directly. Only use for local testing.

### 4. macOS Network Restrictions

Some corporate or public networks block SMTP.

**Test from Terminal:**
```bash
# Test direct connection
telnet smtp.gmail.com 587

# Or with nc (netcat)
nc -zv smtp.gmail.com 587
```

If these fail, the network is blocking SMTP.

## Recommended Solutions for Local Development

### Solution 1: Use Mailtrap (Testing SMTP Service)

Mailtrap is a testing SMTP service that doesn't require real email delivery:

1. Sign up at [mailtrap.io](https://mailtrap.io) (free tier available)
2. Get your SMTP credentials
3. Update `.env.production`:

```env
SMTP_HOST=sandbox.smtp.mailtrap.io
SMTP_PORT=2525
SMTP_USER=your-mailtrap-username
SMTP_PASS=your-mailtrap-password
EMAIL_SENDER=noreply@test.com
```

**Advantages:**
- No firewall issues
- No ISP blocking
- Emails captured for testing
- Free tier available

### Solution 2: Use Gmail with App Password (Port 465)

If port 587 is blocked, try port 465:

```env
SMTP_HOST=smtp.gmail.com
SMTP_PORT=465
SMTP_USE_SSL=true
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password
EMAIL_SENDER=your-email@gmail.com
```

### Solution 3: Use Local SMTP Relay

Set up a local SMTP relay that forwards to Gmail:

1. Install Postfix or similar
2. Configure as relay
3. Point email service to `localhost:25`

### Solution 4: Test with Different Network

Try from:
- Mobile hotspot
- Different WiFi network
- VPN connection

## Testing Steps

### Step 1: Test Host Connectivity
```bash
# Test port 587
python3 -c "import socket; s = socket.socket(); s.settimeout(5); print('Port 587:', 'OPEN' if s.connect_ex(('smtp.gmail.com', 587)) == 0 else 'BLOCKED'); s.close()"

# Test port 465
python3 -c "import socket; s = socket.socket(); s.settimeout(5); print('Port 465:', 'OPEN' if s.connect_ex(('smtp.gmail.com', 465)) == 0 else 'BLOCKED'); s.close()"
```

### Step 2: Test from Container
```bash
docker exec hng-email-service python3 -c "
import socket
s = socket.socket()
s.settimeout(5)
result = s.connect_ex(('smtp.gmail.com', 587))
print('Container → Port 587:', 'OPEN' if result == 0 else 'BLOCKED')
s.close()
"
```

### Step 3: Check Docker Network
```bash
# Check Docker network configuration
docker network inspect hng-stage-four_hng-network | grep -A 10 "Config"
```

## Quick Fix: Use Mailtrap for Local Testing

The easiest solution for local development:

1. **Sign up for Mailtrap** (free): https://mailtrap.io
2. **Get credentials** from your Mailtrap inbox
3. **Update `.env.production`**:
   ```env
   SMTP_HOST=sandbox.smtp.mailtrap.io
   SMTP_PORT=2525
   SMTP_USER=your-username
   SMTP_PASS=your-password
   EMAIL_SENDER=test@example.com
   ```
4. **Restart email service**:
   ```bash
   docker compose -f docker-compose.freetier.yml restart email-service
   ```
5. **Test**: Emails will appear in your Mailtrap inbox instead of being sent

## Verify Fix

After applying a solution:

```bash
# Run full test
./test-smtp-connectivity.sh

# Send test notification
curl -X POST http://localhost:3000/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "58f8f5c0-8354-49b0-b297-433cebb2064f",
    "template_id": "71a258ea-e6f5-415a-9ca5-b077425f8344",
    "type": "email",
    "data": {}
  }'

# Check logs
docker compose -f docker-compose.freetier.yml logs email-service --tail 20 | grep smtp
```

## Summary

For **local development**, the best options are:

1. **Mailtrap** (recommended) - No firewall/ISP issues, perfect for testing
2. **Port 465** - Sometimes less restricted than 587
3. **Different SMTP provider** - SendGrid, Mailgun, etc.
4. **VPN** - Bypass ISP restrictions

For **production (EC2)**, ensure security group allows outbound port 587/465.

