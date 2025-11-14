# AWS Free Tier Deployment Guide

This guide is optimized for AWS Free Tier (t2.micro/t3.micro instances with 1GB RAM).

## Free Tier Constraints

- **Instance**: t2.micro or t3.micro (1 vCPU, 1 GB RAM)
- **Storage**: 30 GiB EBS
- **Bandwidth**: 100 GB/month
- **750 hours/month** of instance usage

## Optimizations Applied

1. **Single PostgreSQL instance** with multiple databases (saves ~600MB RAM)
2. **Memory limits** on all containers
3. **Alpine-based images** for smaller footprint
4. **Minimal Redis persistence** (no AOF, no RDB)
5. **RabbitMQ memory limits** (30% of available)
6. **Swap space** for memory overflow protection

## Prerequisites

- AWS Account with Free Tier eligibility
- EC2 t2.micro or t3.micro instance
- Ubuntu 22.04 LTS
- Domain name (optional, for SSL)
- Firebase service account JSON file
- SMTP credentials

## Step 1: Launch EC2 Instance

1. Go to AWS Console → EC2 → Launch Instance
2. Choose **Ubuntu Server 22.04 LTS** (Free Tier eligible)
3. Select instance type: **t3.micro** (or t2.micro if t3.micro unavailable)
4. Configure storage: **20 GiB** (within free tier's 30 GiB limit)
5. Configure security group:
   - **SSH (22)** - Your IP only
   - **HTTP (80)** - 0.0.0.0/0
   - **HTTPS (443)** - 0.0.0.0/0
6. Launch instance

## Step 2: Connect and Initial Setup

```bash
# Connect to instance
ssh -i your-key.pem ubuntu@your-ec2-ip

# Update system
sudo apt update && sudo apt upgrade -y
```

## Step 3: Create Swap Space

**Critical for 1GB RAM instances!**

```bash
# Create 2GB swap file
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# Make it permanent
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab

# Verify
free -h
```

## Step 4: Install Docker and Dependencies

```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Add user to docker group
sudo usermod -aG docker ubuntu

# Install Docker Compose v2
sudo apt install docker-compose-plugin -y

# Install other dependencies
sudo apt install nginx git certbot python3-certbot-nginx htop -y

# Configure Docker log limits (important for storage)
sudo tee /etc/docker/daemon.json > /dev/null <<EOF
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
EOF

sudo systemctl restart docker

# Log out and back in for group changes
exit
```

## Step 5: Clone and Configure Application

```bash
# Connect again
ssh -i your-key.pem ubuntu@your-ec2-ip

# Clone repository
cd /opt
sudo git clone <your-repository-url> hng-stage-four
sudo chown -R ubuntu:ubuntu hng-stage-four
cd hng-stage-four

# Setup environment
cp .env.production.example .env.production
nano .env.production
```

**Generate strong passwords:**
```bash
openssl rand -base64 32  # POSTGRES_PASSWORD
openssl rand -base64 32  # RABBITMQ_PASSWORD
openssl rand -base64 32  # REDIS_PASSWORD
```

**Fill in `.env.production`:**
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

## Step 6: Upload Firebase Service Account

```bash
# From your local machine
scp -i your-key.pem service-account.json ubuntu@your-ec2-ip:/opt/hng-stage-four/push-service/
```

Or manually:
```bash
nano push-service/service-account.json
# Paste your Firebase service account JSON
```

## Step 7: Deploy Application

```bash
# Use the free tier optimized compose file
docker compose -f docker-compose.freetier.yml --env-file .env.production up -d --build

# Monitor startup (this may take 3-5 minutes on t3.micro)
docker compose -f docker-compose.freetier.yml logs -f
```

**Wait for all services to be healthy** (press Ctrl+C to exit logs)

## Step 8: Verify Services

```bash
# Check all services
curl http://localhost:3000/api/v1/health  # API Gateway
curl http://localhost:3001/api/v1/health  # User Service
curl http://localhost:4000/api/v1/health  # Template Service
curl http://localhost:8000/health          # Email Service
curl http://localhost:8080/health          # Push Service

# Check container status
docker compose -f docker-compose.freetier.yml ps

# Check resource usage
docker stats --no-stream
```

## Step 9: Setup Nginx Reverse Proxy

```bash
# Copy nginx config
sudo cp nginx.conf.example /etc/nginx/sites-available/hng-stage-four
sudo nano /etc/nginx/sites-available/hng-stage-four  # Update domain name

# Enable site
sudo ln -s /etc/nginx/sites-available/hng-stage-four /etc/nginx/sites-enabled/
sudo rm /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl reload nginx
```

## Step 10: Setup SSL (Optional)

```bash
sudo certbot --nginx -d your-domain.com -d www.your-domain.com
```

## Step 11: Configure Firewall

```bash
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

## Monitoring and Maintenance

### Check Resource Usage

```bash
# Memory usage
free -h

# Disk usage
df -h
docker system df

# Container stats
docker stats
```

### Clean Up Docker (Important for Storage)

```bash
# Remove unused images, containers, volumes
docker system prune -a --volumes

# Remove old logs
sudo journalctl --vacuum-time=7d
```

### Monitor Logs

```bash
# All services
docker compose -f docker-compose.freetier.yml logs -f

# Specific service
docker compose -f docker-compose.freetier.yml logs -f api-gateway
```

### Backup Database

```bash
# Backup script
cat > /opt/backup-db.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/opt/backups"
DATE=$(date +%Y%m%d_%H%M%S)
mkdir -p $BACKUP_DIR

# Backup all databases
docker exec hng-postgres pg_dumpall -U postgres | gzip > $BACKUP_DIR/all_dbs_$DATE.sql.gz

# Keep only last 3 days (to save space)
find $BACKUP_DIR -name "*.sql.gz" -mtime +3 -delete

echo "Backup completed: $DATE"
EOF

chmod +x /opt/backup-db.sh

# Schedule daily backup at 2 AM
(crontab -l 2>/dev/null; echo "0 2 * * * /opt/backup-db.sh") | crontab -
```

## Performance Expectations

On t3.micro with 1GB RAM:

- **Startup time**: 3-5 minutes for all services
- **Memory usage**: ~800-900MB (with swap)
- **Concurrent requests**: ~10-20 requests/second
- **Response time**: 100-500ms (depending on load)
- **Suitable for**: Development, testing, low-traffic production (<1000 requests/day)

## Troubleshooting

### Out of Memory

```bash
# Check memory
free -h

# Check swap
swapon --show

# Restart services
docker compose -f docker-compose.freetier.yml restart
```

### Out of Disk Space

```bash
# Check disk usage
df -h

# Clean Docker
docker system prune -a --volumes

# Remove old logs
sudo journalctl --vacuum-time=3d
```

### Services Not Starting

```bash
# Check logs
docker compose -f docker-compose.freetier.yml logs

# Check container status
docker compose -f docker-compose.freetier.yml ps

# Restart specific service
docker compose -f docker-compose.freetier.yml restart <service-name>
```

### Slow Performance

```bash
# Check resource usage
htop
docker stats

# Restart services
docker compose -f docker-compose.freetier.yml restart
```

## Resource Limits Summary

| Service | Memory Limit | Notes |
|---------|-------------|-------|
| PostgreSQL | 200MB | Single instance for all DBs |
| RabbitMQ | 200MB | Alpine image, 30% memory limit |
| Redis | 64MB | No persistence |
| API Gateway | 256MB | Main entry point |
| User Service | 128MB | Lightweight |
| Template Service | 128MB | Lightweight |
| Email Service | 128MB | Python/FastAPI |
| Push Service | 128MB | Go service |
| **Total** | **~1.2GB** | With swap, fits in 1GB + 2GB swap |

## Storage Management

With 30 GiB EBS:
- Base OS: ~8 GB
- Docker images: ~3-4 GB (optimized)
- Application code: ~500 MB
- Database data: ~2-5 GB
- Logs: ~1-2 GB (with rotation)
- Swap: 2 GB
- **Total**: ~16-22 GB (within 30 GiB limit)

## Optimization Tips

1. **Regular cleanup**: Run `docker system prune` weekly
2. **Log rotation**: Already configured in Docker daemon.json
3. **Monitor disk**: Set up alerts if possible
4. **Backup strategy**: Keep only 3 days of backups
5. **Disable RabbitMQ UI**: If not needed, remove port 15672 mapping

## Updating Application

```bash
cd /opt/hng-stage-four
git pull
docker compose -f docker-compose.freetier.yml --env-file .env.production up -d --build
```

## Scaling Beyond Free Tier

When you outgrow free tier:
1. Upgrade to t3.small (2GB RAM) - ~$15/month
2. Use separate database instances
3. Add more swap space
4. Consider managed services (RDS, ElastiCache)

## Support

For issues:
1. Check logs: `docker compose -f docker-compose.freetier.yml logs`
2. Check resources: `htop`, `docker stats`
3. Review this guide
4. Check AWS CloudWatch (if configured)

---

**Free Tier Deployment Complete!** 🚀

Your optimized notification platform is now running on AWS Free Tier.

