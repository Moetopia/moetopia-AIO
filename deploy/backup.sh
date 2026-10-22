#!/bin/bash
set -euo pipefail
umask 077
: "${MOETOPIA_BACKUP_RECIPIENT:?Set the public age recipient}"
export PGPASSWORD="$(cat /run/secrets/migration_db_password)"
nonce=$(od -An -N16 -tx1 /dev/urandom | tr -d ' \n')
name="moetopia-$(date -u +%Y%m%dT%H%M%SZ)-$nonce.dump.age"
target="/backups/$name"
trap 'rm -f -- "$target.partial"' EXIT
pg_dump --host=postgres --username=moetopia_migrator --dbname="${PGDATABASE:-moetopia}" --format=custom --no-owner --no-acl | age --recipient "$MOETOPIA_BACKUP_RECIPIENT" --output "$target.partial"
mv -n -- "$target.partial" "$target"
(cd /backups && sha256sum "$name" > "$name.sha256")
printf 'Encrypted database backup created: %s\n' "$name"
