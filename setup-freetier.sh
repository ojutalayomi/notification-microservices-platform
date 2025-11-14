#!/bin/bash

# Setup script for AWS Free Tier EC2 instance
# Run this script on a fresh Ubuntu 22.04 instance

set -e

echo "=========================================="
echo "AWS Free Tier Setup Script"
echo "=========================================="

# Check if running as root
if [ "$EUID" -eq 0 ]; then 
   echo "Please do not run as root. Run as ubuntu user."
   exit 1
fi

# Update system
echo "Updating system packages..."
sudo apt update && sudo apt upgrade -y

# Create swap space (2GB)
echo "Creating swap space..."
if [ ! -f /swapfile ]; then
    sudo fallocate -l 2G /swapfile
    sudo chmod 600 /swapfile
    sudo mkswap /swapfile
    sudo swapon /swapfile
    echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
    echo "✓ Swap space created (2GB)"
else
    echo "✓ Swap file already exists"
fi

# Install Docker
echo "Installing Docker..."
if ! command -v docker &> /dev/null; then
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    sudo usermod -aG docker $USER
    rm get-docker.sh
    echo "✓ Docker installed"
else
    echo "✓ Docker already installed"
fi

# Install Docker Compose
echo "Installing Docker Compose..."
if ! command -v docker compose &> /dev/null; then
    sudo apt install docker-compose-plugin -y
    echo "✓ Docker Compose installed"
else
    echo "✓ Docker Compose already installed"
fi

# Install other dependencies
echo "Installing dependencies..."
sudo apt install -y nginx git certbot python3-certbot-nginx htop

# Configure Docker log limits
echo "Configuring Docker log limits..."
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
echo "✓ Docker log limits configured"

# Display system info
echo ""
echo "=========================================="
echo "Setup Complete!"
echo "=========================================="
echo ""
echo "System Information:"
echo "  Memory: $(free -h | grep Mem | awk '{print $2}')"
echo "  Swap: $(free -h | grep Swap | awk '{print $2}')"
echo "  Disk: $(df -h / | tail -1 | awk '{print $4}') available"
echo ""
echo "Installed:"
echo "  ✓ Docker $(docker --version | cut -d' ' -f3 | tr -d ',')"
echo "  ✓ Docker Compose $(docker compose version | cut -d' ' -f4)"
echo "  ✓ Nginx"
echo "  ✓ Git"
echo "  ✓ Certbot"
echo ""
echo "Next Steps:"
echo "  1. Log out and back in for Docker group changes"
echo "  2. Clone your repository"
echo "  3. Configure .env.production"
echo "  4. Run: docker compose -f docker-compose.freetier.yml up -d --build"
echo ""
echo "=========================================="

