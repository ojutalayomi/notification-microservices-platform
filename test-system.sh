#!/bin/bash

echo "=========================================="
echo "COMPREHENSIVE SYSTEM TEST"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test 1: Check all services are running
echo -e "${YELLOW}=== Test 1: Service Health Checks ===${NC}"
echo ""

echo "Checking API Gateway..."
API_GATEWAY_HEALTH=$(curl -s http://localhost:3000/health 2>/dev/null)
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ API Gateway is running${NC}"
else
    echo -e "${RED}✗ API Gateway is not responding${NC}"
fi

echo "Checking Email Service..."
EMAIL_HEALTH=$(curl -s http://localhost:8000/health 2>/dev/null)
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Email Service is running${NC}"
else
    echo -e "${RED}✗ Email Service is not responding${NC}"
fi

echo "Checking RabbitMQ..."
RABBITMQ_STATUS=$(docker exec hng-rabbitmq rabbitmqctl status 2>/dev/null | head -1)
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ RabbitMQ is running${NC}"
else
    echo -e "${RED}✗ RabbitMQ is not responding${NC}"
fi

echo ""
echo -e "${YELLOW}=== Test 2: Worker Process Check ===${NC}"
echo ""

WORKER_PROCESSES=$(docker top hng-email-service 2>/dev/null | grep -c "worker.py" || echo "0")
if [ "$WORKER_PROCESSES" -gt 0 ]; then
    echo -e "${GREEN}✓ Worker process is running (found $WORKER_PROCESSES process(es))${NC}"
    docker top hng-email-service | grep worker.py
else
    echo -e "${RED}✗ Worker process NOT found${NC}"
fi

echo ""
echo -e "${YELLOW}=== Test 3: RabbitMQ Queue Status ===${NC}"
echo ""

echo "Queue status before test:"
docker exec hng-rabbitmq rabbitmqctl list_queues name messages consumers messages_unacknowledged 2>/dev/null | grep email || echo "No email queue found"

echo ""
echo -e "${YELLOW}=== Test 4: Send Notification via API Gateway ===${NC}"
echo ""

# Use a test user and template ID
USER_ID="b725418a-4bee-4a69-b587-ad5e0fd9f865"
TEMPLATE_ID="71a258ea-e6f5-415a-9ca5-b077425f8344"

echo "Sending notification..."
RESPONSE=$(curl -s -X POST http://localhost:3000/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": \"$USER_ID\",
    \"template_id\": \"$TEMPLATE_ID\",
    \"type\": \"email\",
    \"data\": {}
  }")

echo "API Gateway Response:"
echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"

NOTIFICATION_ID=$(echo "$RESPONSE" | jq -r '.data.notification_id' 2>/dev/null)

if [ -n "$NOTIFICATION_ID" ] && [ "$NOTIFICATION_ID" != "null" ]; then
    echo -e "${GREEN}✓ Notification queued successfully: $NOTIFICATION_ID${NC}"
else
    echo -e "${RED}✗ Failed to queue notification${NC}"
fi

echo ""
echo -e "${YELLOW}=== Test 5: Wait for Processing (5 seconds) ===${NC}"
sleep 5

echo ""
echo -e "${YELLOW}=== Test 6: RabbitMQ Queue Status After ===${NC}"
echo ""

echo "Queue status after test:"
docker exec hng-rabbitmq rabbitmqctl list_queues name messages consumers messages_unacknowledged 2>/dev/null | grep email || echo "No email queue found"

echo ""
echo -e "${YELLOW}=== Test 7: API Gateway Logs ===${NC}"
echo ""

echo "Recent API Gateway logs (filtered):"
docker compose -f docker-compose.freetier.yml logs api-gateway --tail 20 2>/dev/null | grep -E "published|Notification|queued|Error|error" | tail -5

echo ""
echo -e "${YELLOW}=== Test 8: Email Service Logs ===${NC}"
echo ""

echo "Recent Email Service logs (all):"
docker compose -f docker-compose.freetier.yml logs email-service --tail 30 2>/dev/null | tail -20

echo ""
echo -e "${YELLOW}=== Test 9: Worker-Specific Logs ===${NC}"
echo ""

echo "Worker logs (filtered):"
docker compose -f docker-compose.freetier.yml logs email-service --tail 100 2>/dev/null | grep -E "worker|EMAIL WORKER|Received|Created|Processing|smtp|Listening" | tail -10

echo ""
echo -e "${YELLOW}=== Test 10: Database Check ===${NC}"
echo ""

echo "Latest email records in database:"
docker exec hng-postgres psql -U postgres -d email_service -c "SELECT id, to_email, subject, status, error_message, created_at FROM email_messages ORDER BY created_at DESC LIMIT 5;" 2>/dev/null

echo ""
echo -e "${YELLOW}=== Test 11: RabbitMQ Consumer Details ===${NC}"
echo ""

echo "Active consumers:"
docker exec hng-rabbitmq rabbitmqctl list_consumers 2>/dev/null | grep email || echo "No email queue consumers found"

echo ""
echo -e "${YELLOW}=== Test 12: Check for Errors ===${NC}"
echo ""

echo "API Gateway errors:"
docker compose -f docker-compose.freetier.yml logs api-gateway --tail 50 2>/dev/null | grep -iE "error|exception|failed" | tail -5

echo ""
echo "Email Service errors:"
docker compose -f docker-compose.freetier.yml logs email-service --tail 100 2>/dev/null | grep -iE "error|exception|failed|traceback" | tail -5

echo ""
echo "=========================================="
echo "TEST COMPLETE"
echo "=========================================="

