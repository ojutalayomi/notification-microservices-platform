# Quick Deployment Guide - AWS EC2

## TL;DR - Fast Deployment

### 1. Launch EC2 Instance
- Ubuntu 22.04 LTS
- **Free Tier**: t3.micro (or t2.micro) - See [DEPLOYMENT_FREETIER.md](./DEPLOYMENT_FREETIER.md)
- **Production**: t3.medium or larger
- Security Group: SSH (22), HTTP (80), HTTPS (443)

### 2. Connect and Setup

```bash
# Connect
ssh -i your-key.pem ubuntu@your-ec2-ip

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh && sudo sh get-docker.sh
sudo usermod -aG docker ubuntu
sudo apt install docker-compose-plugin nginx git certbot python3-certbot-nginx -y
exit  # Log out and back in
```

### 3. Deploy Application

```bash
# Clone repository
cd /opt
sudo git clone <your-repo-url> hng-stage-four
sudo chown -R ubuntu:ubuntu hng-stage-four
cd hng-stage-four

# Setup environment
cp .env.production.example .env.production
nano .env.production  # Fill in your values

# Upload Firebase service account
# scp -i your-key.pem service-account.json ubuntu@your-ec2-ip:/opt/hng-stage-four/push-service/

# Deploy
# For Free Tier (t3.micro):
docker compose -f docker-compose.freetier.yml --env-file .env.production up -d --build

# For Production (t3.medium+):
./deploy.sh production
```

### 4. Setup Nginx

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

### 5. Setup SSL (if you have a domain)

```bash
sudo certbot --nginx -d your-domain.com -d www.your-domain.com
```

### 6. Configure Firewall

```bash
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

## Common Commands

```bash
# View logs
docker compose -f docker-compose.prod.yml logs -f

# Restart services
docker compose -f docker-compose.prod.yml restart

# Check status
docker compose -f docker-compose.prod.yml ps

# Update application
cd /opt/hng-stage-four
git pull
./deploy.sh production
```

## Generate Strong Passwords

```bash
openssl rand -base64 32  # Use for POSTGRES_PASSWORD
openssl rand -base64 32  # Use for RABBITMQ_PASSWORD
openssl rand -base64 32  # Use for REDIS_PASSWORD
```

## Health Checks

```bash
curl http://localhost:3000/api/v1/health  # API Gateway
curl http://localhost:3001/api/v1/health  # User Service
curl http://localhost:4000/api/v1/health  # Template Service
curl http://localhost:8000/health          # Email Service
curl http://localhost:8080/health          # Push Service
```

## Troubleshooting

```bash
# Services not starting
docker compose -f docker-compose.prod.yml logs

# Nginx 502 error
sudo tail -f /var/log/nginx/hng-error.log
docker compose -f docker-compose.prod.yml restart api-gateway

# Out of disk space
docker system prune -a --volumes
```

## Free Tier Deployment

For AWS Free Tier (t3.micro/t2.micro), see [DEPLOYMENT_FREETIER.md](./DEPLOYMENT_FREETIER.md)

For production deployment, see [DEPLOYMENT.md](./DEPLOYMENT.md)

