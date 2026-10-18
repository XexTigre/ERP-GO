#!/bin/sh
set -eu
: "${MIGRATOR_PASSWORD:?Defina MIGRATOR_PASSWORD}"
: "${APP_PASSWORD:?Defina APP_PASSWORD}"
psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" --set ON_ERROR_STOP=1 \
  --set migrator_password="$MIGRATOR_PASSWORD" --set app_password="$APP_PASSWORD" <<'SQL'
SELECT 'CREATE ROLE erp_migrator LOGIN NOSUPERUSER NOBYPASSRLS'
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='erp_migrator') \gexec
SELECT 'CREATE ROLE erp_app LOGIN NOSUPERUSER NOBYPASSRLS'
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='erp_app') \gexec
ALTER ROLE erp_migrator PASSWORD :'migrator_password';
ALTER ROLE erp_app PASSWORD :'app_password';
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE, CREATE ON SCHEMA public TO erp_migrator;
GRANT USAGE ON SCHEMA public TO erp_app;
SELECT format('GRANT CONNECT, CREATE ON DATABASE %I TO erp_migrator', current_database()) \gexec
SELECT format('GRANT CONNECT ON DATABASE %I TO erp_app', current_database()) \gexec
SQL
