#!/bin/sh
set -e

DB_HOST=${DB_HOST:-push-db}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_PASSWORD=${DB_PASSWORD:-postgres}
DB_NAME=${DB_NAME:-push_service}
RABBIT_HOST=${RABBITMQ_HOST:-rabbitmq}
RABBIT_PORT=${RABBITMQ_PORT:-5672}

MIGRATE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

run_migrate() {
  migrate -database "$MIGRATE_URL" -path ./migrations "$@"
}

run_migrations() {
  if [ ! -d "./migrations" ]; then
    echo "Migration directory not found!"
    exit 1
  fi

  echo "Running migrations..."

  set +e
  OUTPUT=$(run_migrate up 2>&1)
  STATUS=$?
  set -e

  if [ -n "$OUTPUT" ]; then
    echo "$OUTPUT"
  fi

  if [ "$STATUS" -ne 0 ]; then
    if echo "$OUTPUT" | grep -qi "Dirty database version"; then
      DIRTY_VERSION=$(echo "$OUTPUT" | sed -n 's/.*Dirty database version \([0-9][0-9]*\).*/\1/p')

      if [ -z "$DIRTY_VERSION" ]; then
        VERSION_OUTPUT=$(run_migrate version 2>&1 || true)
        echo "$VERSION_OUTPUT"
        DIRTY_VERSION=$(echo "$VERSION_OUTPUT" | sed -n 's/.*version \([0-9][0-9]*\).*/\1/p')
      fi

      DIRTY_VERSION=${DIRTY_VERSION:-1}
      echo "Database dirty at version ${DIRTY_VERSION}. Forcing and retrying..."
      run_migrate force "$DIRTY_VERSION"
      run_migrate up
    else
      echo "Migration command failed. Aborting."
      exit "$STATUS"
    fi
  fi
}

echo "Waiting for PostgreSQL at ${DB_HOST}:${DB_PORT}..."
until nc -z "$DB_HOST" "$DB_PORT"; do
  printf '.'
  sleep 2
done
echo "PostgreSQL is reachable."

run_migrations

echo "Waiting for RabbitMQ at ${RABBIT_HOST}:${RABBIT_PORT}..."
until nc -z "$RABBIT_HOST" "$RABBIT_PORT"; do
  printf '.'
  sleep 2
done
echo "RabbitMQ is reachable."

exec ./main
