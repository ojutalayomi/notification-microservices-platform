#!/bin/sh
set -e

DB_HOST=${DB_HOST:-push-db}
DB_PORT=${DB_PORT:-5432}
RABBIT_HOST=${RABBITMQ_HOST:-rabbitmq}
RABBIT_PORT=${RABBITMQ_PORT:-5672}

echo "Waiting for PostgreSQL at ${DB_HOST}:${DB_PORT}..."
until nc -z "$DB_HOST" "$DB_PORT"; do
  printf '.'
  sleep 2
done
echo "PostgreSQL is reachable."

echo "Waiting for RabbitMQ at ${RABBIT_HOST}:${RABBIT_PORT}..."
until nc -z "$RABBIT_HOST" "$RABBIT_PORT"; do
  printf '.'
  sleep 2
done
echo "RabbitMQ is reachable."

exec ./main

