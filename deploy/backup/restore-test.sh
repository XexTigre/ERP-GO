#!/bin/sh
set -eu
: "${RESTORE_DATABASE_URL:?Defina RESTORE_DATABASE_URL}"
: "${1:?Informe o dump}"
name=$(psql "$RESTORE_DATABASE_URL" -XAt --set ON_ERROR_STOP=1 -c 'SELECT current_database()')
case "$name" in *_test) ;; *) echo 'Restore exige banco descartável com sufixo _test' >&2; exit 1 ;; esac
count=$(psql "$RESTORE_DATABASE_URL" -XAt --set ON_ERROR_STOP=1 -c "SELECT count(*) FROM pg_tables WHERE schemaname NOT IN ('pg_catalog','information_schema')")
test "$count" = 0 || { echo 'Restore exige banco vazio' >&2; exit 1; }
pg_restore --dbname="$RESTORE_DATABASE_URL" --exit-on-error --no-owner --no-privileges "$1"
