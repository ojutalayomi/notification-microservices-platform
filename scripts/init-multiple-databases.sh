#!/bin/bash
# Initialize multiple databases in a single PostgreSQL instance
# This script can be run automatically by the postgres entrypoint or manually after startup

set -e

# Wait for PostgreSQL to be ready (with retries)
echo "Waiting for PostgreSQL to be ready..."
for i in {1..30}; do
    if pg_isready -U "postgres" -d postgres >/dev/null 2>&1; then
        break
    fi
    if [ $i -eq 30 ]; then
        echo "PostgreSQL failed to become ready"
        exit 1
    fi
    sleep 1
done

# Connect as the default postgres superuser (created by initdb) to set up our user
# Use trust authentication for local connections (default in postgres Docker image)
export PGPASSWORD=""

# Ensure the configured user exists and has the correct password
TARGET_USER="${POSTGRES_USER:-postgres}"
if [ "$TARGET_USER" != "postgres" ]; then
    echo "Ensuring user '$TARGET_USER' exists..."
    psql -v ON_ERROR_STOP=1 --username "postgres" --dbname postgres <<-EOSQL
        DO \$\$
        BEGIN
            IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '$TARGET_USER') THEN
                CREATE USER "$TARGET_USER" WITH SUPERUSER PASSWORD '${POSTGRES_PASSWORD}';
            ELSE
                ALTER USER "$TARGET_USER" WITH PASSWORD '${POSTGRES_PASSWORD}';
            END IF;
        END
        \$\$;
EOSQL
else
    # If using default postgres user, just set the password
    echo "Setting password for postgres user..."
    psql -v ON_ERROR_STOP=1 --username "postgres" --dbname postgres <<-EOSQL
        ALTER USER postgres WITH PASSWORD '${POSTGRES_PASSWORD}';
EOSQL
fi

# Now use the configured user for subsequent operations
export PGPASSWORD="${POSTGRES_PASSWORD}"

# Create databases if they don't exist
if [ -n "$POSTGRES_MULTIPLE_DATABASES" ]; then
    echo "Creating multiple databases: $POSTGRES_MULTIPLE_DATABASES"
    for db in $(echo $POSTGRES_MULTIPLE_DATABASES | tr ',' ' '); do
        echo "Creating database: $db"
        psql -v ON_ERROR_STOP=1 --username "${POSTGRES_USER:-postgres}" --dbname postgres -tc "SELECT 1 FROM pg_database WHERE datname = '$db'" | grep -q 1 || \
        psql -v ON_ERROR_STOP=1 --username "${POSTGRES_USER:-postgres}" --dbname postgres -c "CREATE DATABASE $db"
    done
    echo "Multiple databases created successfully"
fi


