#!/bin/bash
set -euo pipefail
umask 077
: "${MOETOPIA_RESTORE_DATABASE:?Choose a new moetopia_restore_<16 hex> database}"
: "${MOETOPIA_RESTORE_ARCHIVE:?Choose an encrypted archive basename}"
if [[ ! "$MOETOPIA_RESTORE_DATABASE" =~ ^moetopia_restore_[a-f0-9]{16}$ ]] || [[ ! "$MOETOPIA_RESTORE_ARCHIVE" =~ ^moetopia-[0-9]{8}T[0-9]{6}Z-[a-f0-9]{32}\.dump\.age$ ]]; then
  printf '%s\n' 'Refusing an unsafe restore target or archive name.' >&2
  exit 1
fi
test -r "/backups/$MOETOPIA_RESTORE_ARCHIVE"
test -r /run/secrets/restore_identity
export PGPASSWORD="$(cat /run/secrets/migration_db_password)"
createdb --host=postgres --username=moetopia_migrator --template=template0 "$MOETOPIA_RESTORE_DATABASE"
age --decrypt --identity /run/secrets/restore_identity "/backups/$MOETOPIA_RESTORE_ARCHIVE" | pg_restore --host=postgres --username=moetopia_migrator --dbname="$MOETOPIA_RESTORE_DATABASE" --no-owner --no-acl --exit-on-error --single-transaction
printf '%s\n' 'Restored into a new isolated database. Verify the original encryption key and invalidate transient credentials before any cutover.'
