#!/bin/bash

echo "=========================================="
echo "FULL SYSTEM FLOW TEST"
echo "=========================================="
echo ""

# Valid user ID from database
USER_ID="58f8f5c0-8354-49b0-b297-433cebb2064f"
TEMPLATE_ID="71a258ea-e6f5-415a-9ca5-b077425f8344"

echo "Step 1: Check worker is running..."
WORKER_COUNT=$(docker top hng-email-service 2>/dev/null | grep -c "worker.py" || echo "0")
if [ "$WORKER_COUNT" -gt 0 ]; then
    echo "✓ Worker is running"
else
    echo "✗ Worker not found"
    exit 1
fi

echo ""
echo "Step 2: Check RabbitMQ queue status (before)..."
docker exec hng-rabbitmq rabbitmqctl list_queues name messages consumers 2>/dev/null | grep email

echo ""
echo "Step 3: Send notification via API Gateway..."
RESPONSE=$(curl -s -X POST http://localhost:3000/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": \"$USER_ID\",
    \"template_id\": \"$TEMPLATE_ID\",
    \"type\": \"email\",
    \"data\": {}
  }")

echo "Response:"
echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"

NOTIFICATION_ID=$(echo "$RESPONSE" | jq -r '.data.notification_id' 2>/dev/null)
if [ -n "$NOTIFICATION_ID" ] && [ "$NOTIFICATION_ID" != "null" ]; then
    echo "✓ Notification queued: $NOTIFICATION_ID"
else
    echo "✗ Failed to queue notification"
    exit 1
fi

echo ""
echo "Step 4: Wait 8 seconds for processing..."
sleep 8

echo ""
echo "Step 5: Check RabbitMQ queue status (after)..."
docker exec hng-rabbitmq rabbitmqctl list_queues name messages consumers 2>/dev/null | grep email

echo ""
echo "Step 6: Check email service logs..."
echo "--- Recent worker logs ---"
docker compose -f docker-compose.freetier.yml logs email-service --tail 30 2>/dev/null | grep -E "worker|smtp|Received|Created|Processing|sent|failed" | tail -10

echo ""
echo "Step 7: Check database for email record..."
docker exec hng-postgres psql -U postgres -d email_service -c "SELECT id, to_email, subject, status, error_message, created_at FROM email_messages ORDER BY created_at DESC LIMIT 1;" 2>/dev/null

echo ""
echo "=========================================="
echo "TEST COMPLETE"
echo "=========================================="

