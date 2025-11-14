#!/bin/sh
set -e

RABBIT_HOST=${RABBITMQ_HOST:-rabbitmq}
RABBIT_PORT=${RABBITMQ_PORT:-5672}

echo "Waiting for RabbitMQ at ${RABBIT_HOST}:${RABBIT_PORT}..."
until nc -z "$RABBIT_HOST" "$RABBIT_PORT"; do
  printf '.'
  sleep 2
done
echo "RabbitMQ is reachable."

exec node dist/main

