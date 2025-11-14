#!/bin/bash

# Deployment script for AWS EC2
# Usage: ./deploy.sh [production|development]

set -e  # Exit on error

ENVIRONMENT=${1:-production}
COMPOSE_FILE="docker-compose.yml"

if [ "$ENVIRONMENT" = "production" ]; then
    COMPOSE_FILE="docker-compose.prod.yml"
    ENV_FILE=".env.production"
    
    if [ ! -f "$ENV_FILE" ]; then
        echo "Error: $ENV_FILE not found!"
        echo "Please copy .env.production.example to $ENV_FILE and configure it."
        exit 1
    fi
else
    ENV_FILE=".env"
fi

echo "=========================================="
echo "Deploying to $ENVIRONMENT environment"
echo "Using compose file: $COMPOSE_FILE"
echo "=========================================="

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "Error: Docker is not running!"
    exit 1
fi

# Check if Docker Compose is available
if ! docker compose version > /dev/null 2>&1; then
    echo "Error: Docker Compose is not installed!"
    exit 1
fi

# Pull latest code (if in git repository)
if [ -d ".git" ]; then
    echo "Pulling latest code..."
    git pull || echo "Warning: Could not pull latest code"
fi

# Build and start services
echo "Building and starting services..."
if [ "$ENVIRONMENT" = "production" ]; then
    docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d --build
else
    docker compose -f "$COMPOSE_FILE" up -d --build
fi

# Wait for services to be healthy
echo "Waiting for services to start..."
sleep 10

# Check service health
echo "Checking service health..."
HEALTH_CHECK_URLS=(
    "http://localhost:3000/api/v1/health"
    "http://localhost:3001/api/v1/health"
    "http://localhost:4000/api/v1/health"
    "http://localhost:8000/health"
    "http://localhost:8080/health"
)

FAILED_CHECKS=0
for url in "${HEALTH_CHECK_URLS[@]}"; do
    if curl -f -s "$url" > /dev/null 2>&1; then
        echo "✓ $url - Healthy"
    else
        echo "✗ $url - Unhealthy"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
    fi
done

# Show service status
echo ""
echo "Service Status:"
docker compose -f "$COMPOSE_FILE" ps

# Show summary
echo ""
echo "=========================================="
if [ $FAILED_CHECKS -eq 0 ]; then
    echo "✓ Deployment successful!"
    echo "All services are healthy."
else
    echo "⚠ Deployment completed with $FAILED_CHECKS service(s) unhealthy"
    echo "Check logs with: docker compose -f $COMPOSE_FILE logs"
fi
echo "=========================================="

# Show useful commands
echo ""
echo "Useful commands:"
echo "  View logs:        docker compose -f $COMPOSE_FILE logs -f"
echo "  Stop services:    docker compose -f $COMPOSE_FILE down"
echo "  Restart service:  docker compose -f $COMPOSE_FILE restart <service-name>"
echo "  View status:      docker compose -f $COMPOSE_FILE ps"

