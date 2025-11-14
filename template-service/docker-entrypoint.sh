#!/bin/sh
set -e

DB_HOST=${DB_HOST:-template-db}
DB_PORT=${DB_PORT:-5432}

echo "Waiting for PostgreSQL at ${DB_HOST}:${DB_PORT}..."
until nc -z "$DB_HOST" "$DB_PORT"; do
  printf '.'
  sleep 2
done
echo "PostgreSQL is reachable."

if [ -f "prisma/schema.prisma" ]; then
  if [ -d "prisma/migrations" ] && [ "$(ls -A prisma/migrations 2>/dev/null)" ]; then
    npx prisma migrate deploy
  else
    npx prisma db push
  fi
fi

exec node dist/main

