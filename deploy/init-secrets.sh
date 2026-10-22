#!/bin/sh
set -eu
umask 077
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)
target="${1:-$root/deploy/secrets}"
case "$target" in /*) ;; *) target="$root/$target" ;; esac
if [ -e "$target" ] || [ -L "$target" ]; then
  printf '%s\n' 'Secret directory already exists; refusing to replace or regenerate any key.' >&2
  exit 1
fi
command -v openssl >/dev/null 2>&1
mkdir -m 700 "$target"
for name in migration_db_password app_db_password valkey_password; do
  openssl rand -hex 32 > "$target/$name"
done
openssl rand -base64 32 > "$target/encryption_key"
printf 'postgres://moetopia_migrator:%s@postgres:5432/moetopia?sslmode=disable\n' "$(cat "$target/migration_db_password")" > "$target/migration_database_url"
printf 'postgres://moetopia_app:%s@postgres:5432/moetopia?sslmode=disable\n' "$(cat "$target/app_db_password")" > "$target/app_database_url"
for name in smtp_password github_client_secret google_client_secret; do
  : > "$target/$name"
done
chmod 444 "$target"/*
printf '%s\n' 'Private secret files created. Configure the optional provider/SMTP files and back up the encryption key separately.'
