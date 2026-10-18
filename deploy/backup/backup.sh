#!/bin/sh
set -eu
: "${DATABASE_URL:?Defina DATABASE_URL}"
: "${1:?Informe o arquivo de destino}"
umask 077
pg_dump --dbname="$DATABASE_URL" --format=custom --no-owner --file="$1"
