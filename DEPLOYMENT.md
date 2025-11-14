# AWS EC2 Deployment Guide

This guide will help you deploy the notification microservices platform to AWS EC2.

## Table of Contents

- [Prerequisites](#prerequisites)
- [EC2 Instance Setup](#ec2-instance-setup)
- [Server Configuration](#server-configuration)
- [Application Deployment](#application-deployment)
- [Security Configuration](#security-configuration)
- [Nginx Reverse Proxy](#nginx-reverse-proxy)
- [SSL/TLS Setup](#ssltls-setup)
- [Monitoring and Maintenance](#monitoring-and-maintenance)
- [Troubleshooting](#troubleshooting)

## Prerequisites

- AWS Account
- EC2 instance (recommended: t3.medium or larger, Ubuntu 22.04 LTS)
- Domain name (optional, for SSL)
- Firebase service account JSON file
- SMTP credentials for email service

## EC2 Instance Setup

### 1. Launch EC2 Instance

1. Go to AWS Console → EC2 → Launch Instance
2. Choose **Ubuntu Server 22.04 LTS**
3. Select instance type: **t3.medium** (minimum) or **t3.large** (recommended)
4. Configure security group:
   - **SSH (22)** - Your IP only
   - **HTTP (80)** - 0.0.0.0/0
   - **HTTPS (443)** - 0.0.0.0/0
5. Create or select a key pair
6. Launch instance

### 2. Connect to Instance

```bash
ssh -i your-key.pem ubuntu@your-ec2-ip
```

### 3. Update System

```bash
sudo apt update && sudo apt upgrade -y
```

## Server Configuration

### 1. Install Docker and Docker Compose

```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Add user to docker group
sudo usermod -aG docker ubuntu

# Install Docker Compose v2
sudo apt install docker-compose-plugin -y

# Verify installation
docker --version
docker compose version

# Log out and back in for group changes to take effect
exit
```

### 2. Install Nginx (for reverse proxy)

```bash
sudo apt install nginx -y
sudo systemctl enable nginx
sudo systemctl start nginx
```

### 3. Install Git

```bash
sudo apt install git -y
```

### 4. Install Certbot (for SSL)

```bash
sudo apt install certbot python3-certbot-nginx -y
```

## Application Deployment

### 1. Clone Repository

```bash
cd /opt
sudo git clone <your-repository-url> hng-stage-four
sudo chown -R ubuntu:ubuntu hng-stage-four
cd hng-stage-four
```

### 2. Set Up Environment Variables

```bash
# Copy example file
cp .env.production.example .env.production

# Edit with your values
nano .env.production
```

**Important**: Generate strong passwords:

```bash
# Generate random passwords
openssl rand -base64 32  # For POSTGRES_PASSWORD
openssl rand -base64 32  # For RABBITMQ_PASSWORD
openssl rand -base64 32  # For REDIS_PASSWORD
```

Fill in `.env.production`:
```env
POSTGRES_USER=postgres
POSTGRES_PASSWORD=<generated-password>

RABBITMQ_USER=admin
RABBITMQ_PASSWORD=<generated-password>

REDIS_PASSWORD=<generated-password>

SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password
EMAIL_SENDER=noreply@yourdomain.com
```

### 3. Upload Firebase Service Account

```bash
# Create directory if needed
mkdir -p push-service

# Upload your Firebase service account file
# Use scp from your local machine:
# scp -i your-key.pem service-account.json ubuntu@your-ec2-ip:/opt/hng-stage-four/push-service/
```

Or manually create the file:

```bash
nano push-service/service-account.json
# Paste your Firebase service account JSON
```

### 4. Build and Start Services

```bash
# Build and start all services
docker compose -f docker-compose.prod.yml --env-file .env.production up -d --build

# Check status
docker compose -f docker-compose.prod.yml ps

# View logs
docker compose -f docker-compose.prod.yml logs -f
```

### 5. Verify Services

```bash
# Check API Gateway
curl http://localhost:3000/api/v1/health

# Check all services
curl http://localhost:3001/api/v1/health  # User Service
curl http://localhost:4000/api/v1/health  # Template Service
curl http://localhost:8000/health        # Email Service
curl http://localhost:8080/health        # Push Service
```

## Security Configuration

### 1. Configure Firewall (UFW)

```bash
# Enable UFW
sudo ufw enable

# Allow SSH
sudo ufw allow 22/tcp

# Allow HTTP and HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Check status
sudo ufw status
```

### 2. Secure Database Ports

The production docker-compose file binds database ports to `127.0.0.1` only, so they're not accessible from outside. This is already configured.

### 3. Set File Permissions

```bash
# Secure environment file
chmod 600 .env.production

# Secure Firebase credentials
chmod 600 push-service/service-account.json
```

## Nginx Reverse Proxy

### 1. Create Nginx Configuration

```bash
sudo nano /etc/nginx/sites-available/hng-stage-four
```

Add configuration:

```nginx
upstream api_gateway {
    server 127.0.0.1:3000;
}

server {
    listen 80;
    server_name your-domain.com www.your-domain.com;

    # Logging
    access_log /var/log/nginx/hng-access.log;
    error_log /var/log/nginx/hng-error.log;

    # Client body size limit
    client_max_body_size 10M;

    # API Gateway
    location / {
        proxy_pass http://api_gateway;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        proxy_read_timeout 300s;
        proxy_connect_timeout 75s;
    }

    # Health check endpoint
    location /health {
        proxy_pass http://api_gateway/api/v1/health;
        access_log off;
    }
}
```

### 2. Enable Site

```bash
# Create symlink
sudo ln -s /etc/nginx/sites-available/hng-stage-four /etc/nginx/sites-enabled/

# Remove default site
sudo rm /etc/nginx/sites-enabled/default

# Test configuration
sudo nginx -t

# Reload Nginx
sudo systemctl reload nginx
```

### 3. Test Access

```bash
# From your local machine
curl http://your-ec2-ip/api/v1/health
```

## SSL/TLS Setup

### 1. Obtain SSL Certificate

```bash
# Replace with your domain
sudo certbot --nginx -d your-domain.com -d www.your-domain.com
```

Follow the prompts:
- Enter your email
- Agree to terms
- Choose whether to redirect HTTP to HTTPS (recommended: Yes)

### 2. Auto-Renewal

Certbot sets up auto-renewal automatically. Test it:

```bash
sudo certbot renew --dry-run
```

## Monitoring and Maintenance

### 1. View Logs

```bash
# All services
docker compose -f docker-compose.prod.yml logs -f

# Specific service
docker compose -f docker-compose.prod.yml logs -f api-gateway
docker compose -f docker-compose.prod.yml logs -f email-service
docker compose -f docker-compose.prod.yml logs -f push-service
```

### 2. Check Service Status

```bash
# Container status
docker compose -f docker-compose.prod.yml ps

# Resource usage
docker stats

# Disk usage
df -h
docker system df
```

### 3. Backup Databases

Create backup script:

```bash
nano /opt/backup-databases.sh
```

```bash
#!/bin/bash
BACKUP_DIR="/opt/backups"
DATE=$(date +%Y%m%d_%H%M%S)
mkdir -p $BACKUP_DIR

# Backup user-db
docker exec hng-user-db pg_dump -U postgres user_service | gzip > $BACKUP_DIR/user_db_$DATE.sql.gz

# Backup template-db
docker exec hng-template-db pg_dump -U postgres template_service | gzip > $BACKUP_DIR/template_db_$DATE.sql.gz

# Backup push-db
docker exec hng-push-db pg_dump -U postgres push_service | gzip > $BACKUP_DIR/push_db_$DATE.sql.gz

# Backup email-db
docker exec hng-email-db pg_dump -U postgres email_service | gzip > $BACKUP_DIR/email_db_$DATE.sql.gz

# Keep only last 7 days
find $BACKUP_DIR -name "*.sql.gz" -mtime +7 -delete

echo "Backup completed: $DATE"
```

Make executable and schedule:

```bash
chmod +x /opt/backup-databases.sh

# Add to crontab (daily at 2 AM)
crontab -e
# Add: 0 2 * * * /opt/backup-databases.sh
```

### 4. Update Application

```bash
cd /opt/hng-stage-four

# Pull latest changes
git pull

# Rebuild and restart
docker compose -f docker-compose.prod.yml --env-file .env.production up -d --build

# Or restart specific service
docker compose -f docker-compose.prod.yml restart api-gateway
```

### 5. Monitor Resources

Install monitoring tools:

```bash
# Install htop
sudo apt install htop -y

# Monitor
htop
```

## Troubleshooting

### Services Not Starting

```bash
# Check logs
docker compose -f docker-compose.prod.yml logs

# Check container status
docker compose -f docker-compose.prod.yml ps

# Restart specific service
docker compose -f docker-compose.prod.yml restart <service-name>
```

### Port Already in Use

```bash
# Find process using port
sudo lsof -i :3000

# Kill process or change port in docker-compose.prod.yml
```

### Database Connection Issues

```bash
# Check database is running
docker compose -f docker-compose.prod.yml ps | grep db

# Check database logs
docker compose -f docker-compose.prod.yml logs user-db

# Test connection
docker exec -it hng-user-db psql -U postgres -d user_service
```

### Nginx 502 Bad Gateway

```bash
# Check if API Gateway is running
curl http://localhost:3000/api/v1/health

# Check Nginx error logs
sudo tail -f /var/log/nginx/hng-error.log

# Restart services
docker compose -f docker-compose.prod.yml restart api-gateway
sudo systemctl restart nginx
```

### Out of Disk Space

```bash
# Clean up Docker
docker system prune -a --volumes

# Check disk usage
df -h
docker system df
```

### SSL Certificate Issues

```bash
# Check certificate status
sudo certbot certificates

# Renew manually
sudo certbot renew

# Check Nginx SSL config
sudo nginx -t
```

## Production Checklist

- [ ] Strong passwords set in `.env.production`
- [ ] Firebase service account uploaded
- [ ] SMTP credentials configured
- [ ] Firewall (UFW) configured
- [ ] Nginx reverse proxy configured
- [ ] SSL certificate installed
- [ ] Database backups scheduled
- [ ] Monitoring set up
- [ ] Health checks working
- [ ] All services running
- [ ] Logs accessible
- [ ] Domain DNS configured (if using)

## Security Best Practices

1. **Never commit `.env.production`** to version control
2. **Use strong passwords** (32+ characters, random)
3. **Restrict SSH access** to your IP only
4. **Keep system updated**: `sudo apt update && sudo apt upgrade`
5. **Monitor logs** regularly for suspicious activity
6. **Set up CloudWatch** or similar monitoring
7. **Enable AWS Security Groups** properly
8. **Use IAM roles** instead of access keys when possible
9. **Regular backups** of databases
10. **Review and rotate** credentials periodically

## Scaling Considerations

For higher traffic, consider:

1. **Use AWS RDS** for databases instead of containers
2. **Use AWS ElastiCache** for Redis
3. **Use AWS SQS** or managed RabbitMQ
4. **Load balancer** with multiple EC2 instances
5. **Auto-scaling groups** for EC2
6. **Container orchestration** (ECS, EKS, or Kubernetes)

## Support

For issues:
1. Check logs: `docker compose -f docker-compose.prod.yml logs`
2. Check service health: `curl http://localhost:3000/api/v1/health`
3. Review this troubleshooting guide
4. Check AWS CloudWatch logs (if configured)

---

**Deployment Complete!** 🚀

Your notification platform should now be accessible at:
- HTTP: `http://your-domain.com` or `http://your-ec2-ip`
- HTTPS: `https://your-domain.com` (if SSL configured)

