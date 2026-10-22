#!/bin/bash
set -euo pipefail
umask 077
: "${MOETOPIA_KEY_BACKUP_RECIPIENT:?Set a separate public age recipient for key escrow}"
nonce=$(od -An -N16 -tx1 /dev/urandom | tr -d ' \n')
name="encryption-key-$(date -u +%Y%m%dT%H%M%SZ)-$nonce.age"
target="/key-backups/$name"
trap 'rm -f -- "$target.partial"' EXIT
age --recipient "$MOETOPIA_KEY_BACKUP_RECIPIENT" --output "$target.partial" /run/secrets/encryption_key
mv -n -- "$target.partial" "$target"
(cd /key-backups && sha256sum "$name" > "$name.sha256")
printf 'Encrypted key escrow created: %s\n' "$name"
