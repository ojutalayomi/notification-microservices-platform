# SMTP Configuration Guide

This guide explains how to configure SMTP settings for the Email Service, including alternative connection methods and troubleshooting.

## Environment Variables

The Email Service supports the following SMTP environment variables:

```env
# Required
SMTP_HOST=smtp.gmail.com          # SMTP server hostname
SMTP_USER=your-email@gmail.com     # SMTP username
SMTP_PASS=your-app-password        # SMTP password (App Password for Gmail)
EMAIL_SENDER=noreply@yourdomain.com # Email sender address

# Optional
SMTP_PORT=587                      # SMTP port (default: 587)
SMTP_USE_SSL=false                 # Force SSL/TLS connection (default: false)
SMTP_USE_TLS=true                  # Enable STARTTLS (default: true)
```

## Supported SMTP Ports and Methods

The Email Service automatically tries multiple connection methods based on the port:

### Port 587 (STARTTLS) - **Recommended**
- **Method**: STARTTLS (upgrades plain connection to encrypted)
- **Fallback**: Automatically tries port 465 (SSL) if STARTTLS fails
- **Use case**: Most common, works with most SMTP providers
- **Configuration**:
  ```env
  SMTP_PORT=587
  SMTP_USE_TLS=true
  ```

### Port 465 (SSL/TLS)
- **Method**: Direct SSL/TLS connection from the start
- **Use case**: When port 587 is blocked or STARTTLS fails
- **Configuration**:
  ```env
  SMTP_PORT=465
  SMTP_USE_SSL=true
  ```

### Port 25 (Standard SMTP)
- **Method**: Plain SMTP (may use STARTTLS if available)
- **Use case**: Legacy systems, often blocked by ISPs
- **Configuration**:
  ```env
  SMTP_PORT=25
  SMTP_USE_TLS=true
  ```

## Provider-Specific Configuration

### Gmail

**Requirements:**
- Enable 2-Step Verification
- Generate an App Password (not your regular password)

**Configuration:**
```env
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-16-char-app-password
EMAIL_SENDER=your-email@gmail.com
```

**How to get Gmail App Password:**
1. Go to [Google Account Security](https://myaccount.google.com/security)
2. Enable 2-Step Verification
3. Search for "App passwords"
4. Generate a new app password for "Mail"
5. Use the 16-character password in `SMTP_PASS`

### Outlook/Office 365

```env
SMTP_HOST=smtp.office365.com
SMTP_PORT=587
SMTP_USER=your-email@outlook.com
SMTP_PASS=your-password
EMAIL_SENDER=your-email@outlook.com
```

### SendGrid

```env
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USER=apikey
SMTP_PASS=your-sendgrid-api-key
EMAIL_SENDER=noreply@yourdomain.com
```

### Mailgun

```env
SMTP_HOST=smtp.mailgun.org
SMTP_PORT=587
SMTP_USER=postmaster@your-domain.mailgun.org
SMTP_PASS=your-mailgun-smtp-password
EMAIL_SENDER=noreply@yourdomain.com
```

### Amazon SES

```env
SMTP_HOST=email-smtp.us-east-1.amazonaws.com  # Use your region
SMTP_PORT=587
SMTP_USER=your-ses-smtp-username
SMTP_PASS=your-ses-smtp-password
EMAIL_SENDER=noreply@yourdomain.com
```

## Automatic Fallback

The Email Service automatically tries alternative connection methods if the primary method fails:

1. **Port 587**: Tries STARTTLS first, then falls back to port 465 (SSL)
2. **Port 465**: Uses SSL/TLS directly
3. **Other ports**: Tries STARTTLS, then falls back to port 465 if enabled

This ensures maximum compatibility with different network configurations and firewall rules.

## Testing SMTP Connectivity

Use the provided test script to diagnose SMTP connectivity issues:

```bash
# Run SMTP connectivity test
./test-smtp-connectivity.sh
```

This script tests:
- SMTP configuration
- Port connectivity from container
- Port connectivity from host
- SMTP handshake
- SMTP authentication

## Common Issues and Solutions

### Issue: Connection Timeout

**Symptoms:**
- `TimeoutError: timed out`
- `[Errno 99] Cannot assign requested address`

**Solutions:**
1. **Check firewall/security group**: Ensure outbound port 587 (or 465) is allowed
   - **EC2**: Add outbound rule in Security Group
   - **Local**: Check firewall settings
2. **Try alternative port**: Switch to port 465
   ```env
   SMTP_PORT=465
   SMTP_USE_SSL=true
   ```
3. **Test connectivity**: Run `./test-smtp-connectivity.sh`

### Issue: Authentication Failed

**Symptoms:**
- `535 Authentication failed`
- `535-5.7.8 Username and Password not accepted`

**Solutions:**
1. **Gmail**: Use App Password, not regular password
2. **Check credentials**: Verify username and password are correct
3. **Check sender**: Ensure `EMAIL_SENDER` matches the authenticated account (for Gmail)

### Issue: Port Blocked

**Symptoms:**
- `Connection refused`
- `[Errno 111] Connection refused`

**Solutions:**
1. **Try port 465**: Often less restricted
   ```env
   SMTP_PORT=465
   SMTP_USE_SSL=true
   ```
2. **Use different provider**: Consider SendGrid, Mailgun, or Amazon SES
3. **Check network**: Some ISPs block SMTP ports

### Issue: SSL/TLS Errors

**Symptoms:**
- `SSL: CERTIFICATE_VERIFY_FAILED`
- `[SSL: SSLV3_ALERT_HANDSHAKE_FAILURE]`

**Solutions:**
1. **Update certificates**: Ensure container has latest CA certificates
2. **Check provider**: Some providers require specific TLS versions
3. **Try different port**: Switch between 587 and 465

## EC2/AWS Deployment

When deploying to EC2, ensure your Security Group allows outbound SMTP:

### Security Group Configuration

1. Go to EC2 → Security Groups
2. Select your instance's security group
3. Add outbound rule:
   - **Type**: Custom TCP
   - **Port**: 587 (or 465)
   - **Destination**: 0.0.0.0/0

### Alternative: Use AWS SES

For better deliverability and no port restrictions:

```env
SMTP_HOST=email-smtp.us-east-1.amazonaws.com
SMTP_PORT=587
SMTP_USER=your-ses-smtp-username
SMTP_PASS=your-ses-smtp-password
EMAIL_SENDER=noreply@yourdomain.com
```

## Docker Compose Configuration

Add SMTP variables to your `.env.production` file:

```env
# SMTP Configuration
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password
EMAIL_SENDER=noreply@yourdomain.com

# Optional: Force SSL or disable TLS
# SMTP_USE_SSL=false
# SMTP_USE_TLS=true
```

The `docker-compose.freetier.yml` automatically passes these to the email service.

## Monitoring and Logs

Check email service logs for SMTP connection details:

```bash
# View email service logs
docker compose -f docker-compose.freetier.yml logs email-service --tail 50

# Filter for SMTP-related logs
docker compose -f docker-compose.freetier.yml logs email-service | grep -i smtp
```

Look for:
- `[smtp] Connecting to...` - Connection attempt
- `[smtp] Trying STARTTLS/SSL on port...` - Method being tried
- `[smtp] Email sent successfully!` - Success
- `[smtp] Error sending email:...` - Failure details

## Best Practices

1. **Use App Passwords**: Never use your main account password
2. **Test First**: Use `test-smtp-connectivity.sh` before deploying
3. **Monitor Logs**: Regularly check email service logs for issues
4. **Use Dedicated SMTP**: Consider SendGrid/Mailgun for production
5. **Rate Limiting**: Be aware of provider rate limits (Gmail: 500/day for free accounts)
6. **SPF/DKIM**: Configure SPF and DKIM records for better deliverability

## Troubleshooting Checklist

- [ ] SMTP credentials are correct
- [ ] Port 587/465 is not blocked by firewall
- [ ] Security group allows outbound SMTP (if on EC2)
- [ ] Using App Password for Gmail (not regular password)
- [ ] `EMAIL_SENDER` matches authenticated account (for Gmail)
- [ ] Test script shows connectivity
- [ ] Email service logs show connection attempts
- [ ] Database shows email records being created

