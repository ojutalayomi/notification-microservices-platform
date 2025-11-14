# Free Tier Quick Start

## What's Different?

The free tier configuration uses:
- **Single PostgreSQL instance** instead of 4 separate instances (saves ~600MB RAM)
- **Memory limits** on all containers to fit in 1GB RAM
- **Alpine-based images** for smaller footprint
- **Optimized Redis** (no persistence, 64MB limit)
- **Optimized RabbitMQ** (30% memory limit)

## Quick Deploy

```bash
# 1. Launch t3.micro EC2 instance (Ubuntu 22.04)

# 2. Connect and run setup
ssh -i key.pem ubuntu@your-ip
curl -fsSL https://raw.githubusercontent.com/your-repo/hng-stage-four/main/setup-freetier.sh | bash

# 3. Log out and back in
exit
ssh -i key.pem ubuntu@your-ip

# 4. Clone and configure
cd /opt
sudo git clone <your-repo> hng-stage-four
sudo chown -R ubuntu:ubuntu hng-stage-four
cd hng-stage-four

# 5. Setup environment
cp .env.production.example .env.production
nano .env.production  # Fill in values

# 6. Upload Firebase service account
# scp -i key.pem service-account.json ubuntu@your-ip:/opt/hng-stage-four/push-service/

# 7. Deploy
docker compose -f docker-compose.freetier.yml --env-file .env.production up -d --build

# 8. Check status
docker compose -f docker-compose.freetier.yml ps
docker stats --no-stream
```

## Resource Usage

- **Total Memory**: ~1.2GB (with 2GB swap)
- **Total Storage**: ~16-22GB (within 30GB limit)
- **Startup Time**: 3-5 minutes

## Full Guide

See [DEPLOYMENT_FREETIER.md](./DEPLOYMENT_FREETIER.md) for complete instructions.

