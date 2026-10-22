#!/bin/bash
set -euo pipefail
umask 077
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)
cd "$root"
nonce=$(od -An -N8 -tx1 /dev/urandom | tr -d ' \n')
[[ "$nonce" =~ ^[a-f0-9]{16}$ ]]
run="$root/.data/release-$nonce"
mkdir -p "$root/.data"
mkdir -m 700 "$run" "$run/backups" "$run/key-backups"
export MOETOPIA_RELEASE_CHECK_PROJECT="moetopia-release-check-$nonce"
export MOETOPIA_RELEASE="release-check-$nonce"
export MOETOPIA_SECRETS_DIR="$run/secrets"
export MOETOPIA_BACKUPS_DIR="$run/backups"
export MOETOPIA_KEY_BACKUPS_DIR="$run/key-backups"
export MOETOPIA_RESTORE_IDENTITY_FILE="$run/restore-identity"
export MOETOPIA_OPERATOR_UID="$(id -u)"
export MOETOPIA_OPERATOR_GID="$(id -g)"
compose() { sh "$root/deploy/check-compose.sh" "$@"; }
cleanup() { compose down --volumes --remove-orphans > "$run/cleanup.log" 2>&1; }
trap cleanup EXIT
sh deploy/init-secrets.sh "$MOETOPIA_SECRETS_DIR"
compose config --quiet
compose build api account edge backup > "$run/build.log" 2>&1
compose up -d --wait postgres valkey
compose --profile maintenance run --rm migrate
compose --profile maintenance run --rm permissions > "$run/permissions.log" 2>&1
if compose run --rm --no-deps --entrypoint /app/migrate api > "$run/runtime-ddl-denied.log" 2>&1; then
  printf '%s\n' 'Runtime database role unexpectedly has migration privileges.' >&2; exit 1
fi
compose run --rm --no-deps --entrypoint /app/keys api init
compose run --rm --no-deps --entrypoint /app/firstparty api
compose run --rm --no-deps --entrypoint /app/administrator api init-client
compose up -d --wait api account edge
for path in /auth/register /auth/login; do
  [[ "$(curl --silent --show-error --insecure --resolve account.moetopia.test:58443:127.0.0.1 --output /dev/null --write-out '%{http_code}' "https://account.moetopia.test:58443$path")" == 200 ]]
done
[[ "$(curl --silent --show-error --insecure --resolve admin.moetopia.test:58444:127.0.0.1 --output /dev/null --write-out '%{http_code}' https://admin.moetopia.test:58444/applications)" == 200 ]]
curl --silent --show-error --insecure --resolve account.moetopia.test:58443:127.0.0.1 https://account.moetopia.test:58443/internal/metrics > "$run/public-metrics-response.html"
if grep -q '^moetopia_http_requests_total' "$run/public-metrics-response.html"; then printf '%s\n' 'Private metrics leaked through the public edge.' >&2; exit 1; fi
docker run --rm --label io.moetopia.validation="$nonce" --network host --read-only -v "$root/deploy/probe.mjs:/probe.mjs:ro" --entrypoint node "$(sed -n 's/^MOETOPIA_NODE_IMAGE=//p' deploy/images.env)" /probe.mjs https://account.moetopia.test:58443 100 4 --connect-host=127.0.0.1 --insecure-test-tls > "$run/probe.json"
docker run --rm --label io.moetopia.validation="$nonce" --network none --read-only --user "$MOETOPIA_OPERATOR_UID:$MOETOPIA_OPERATOR_GID" -v "$run:/work" --entrypoint age-keygen "moetopia-tools:$MOETOPIA_RELEASE" -o /work/restore-identity > "$run/keygen.log" 2>&1
docker run --rm --label io.moetopia.validation="$nonce" --network none --read-only --user "$MOETOPIA_OPERATOR_UID:$MOETOPIA_OPERATOR_GID" -v "$run:/work" --entrypoint age-keygen "moetopia-tools:$MOETOPIA_RELEASE" -o /work/key-identity >> "$run/keygen.log" 2>&1
export MOETOPIA_BACKUP_RECIPIENT
MOETOPIA_BACKUP_RECIPIENT=$(docker run --rm --network none --read-only --user "$MOETOPIA_OPERATOR_UID:$MOETOPIA_OPERATOR_GID" -v "$run:/work:ro" --entrypoint age-keygen "moetopia-tools:$MOETOPIA_RELEASE" -y /work/restore-identity)
export MOETOPIA_KEY_BACKUP_RECIPIENT
MOETOPIA_KEY_BACKUP_RECIPIENT=$(docker run --rm --network none --read-only --user "$MOETOPIA_OPERATOR_UID:$MOETOPIA_OPERATOR_GID" -v "$run:/work:ro" --entrypoint age-keygen "moetopia-tools:$MOETOPIA_RELEASE" -y /work/key-identity)
compose exec -T postgres psql -X -U moetopia_migrator -d moetopia -v ON_ERROR_STOP=1 -c "INSERT INTO sessions.web_sessions (id,user_id,application,expires_at) VALUES ('restore-drill-session','00000000-0000-4000-8000-000000000001','account',now()+interval '1 hour')" > "$run/seed.log"
compose --profile maintenance run --rm backup
compose --profile maintenance run --rm backup-key
shopt -s nullglob
archives=("$run/backups"/moetopia-*.dump.age)
keys=("$run/key-backups"/encryption-key-*.age)
[[ ${#archives[@]} == 1 && ${#keys[@]} == 1 ]]
export MOETOPIA_RESTORE_ARCHIVE="${archives[0]##*/}"
export MOETOPIA_RESTORE_DATABASE="moetopia_restore_$nonce"
printf 'postgres://moetopia_migrator:%s@postgres:5432/%s?sslmode=disable\n' "$(cat "$MOETOPIA_SECRETS_DIR/migration_db_password")" "$MOETOPIA_RESTORE_DATABASE" > "$MOETOPIA_SECRETS_DIR/restore_database_url"
chmod 444 "$MOETOPIA_SECRETS_DIR/restore_database_url"
started=$(date +%s)
compose --profile restore run --rm restore
if compose --profile restore run --rm -e MOETOPIA_RESTORE_DATABASE=moetopia restore > "$run/unsafe-restore.log" 2>&1; then
  printf '%s\n' 'Unsafe restore target was accepted.' >&2; exit 1
fi
if compose --profile restore run --rm -e MOETOPIA_ENCRYPTION_KEY_FILE= -e MOETOPIA_ENCRYPTION_KEY=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA= restore-check --invalidate-transient > "$run/wrong-key.log" 2>&1; then
  printf '%s\n' 'Wrong key was accepted.' >&2; exit 1
fi
before=$(compose exec -T postgres psql -X -U moetopia_migrator -d "$MOETOPIA_RESTORE_DATABASE" -At -c 'SELECT count(*) FROM sessions.web_sessions')
[[ "$before" == 1 ]]
compose --profile restore run --rm restore-check --invalidate-transient
after=$(compose exec -T postgres psql -X -U moetopia_migrator -d "$MOETOPIA_RESTORE_DATABASE" -At -c 'SELECT count(*) FROM sessions.web_sessions')
source_count=$(compose exec -T postgres psql -X -U moetopia_migrator -d moetopia -At -c 'SELECT count(*) FROM sessions.web_sessions')
[[ "$after" == 0 && "$source_count" == 1 ]]
elapsed=$(( $(date +%s) - started ))
docker run --rm --network none --read-only --user "$MOETOPIA_OPERATOR_UID:$MOETOPIA_OPERATOR_GID" -v "$run:/work" --entrypoint age "moetopia-tools:$MOETOPIA_RELEASE" --decrypt --identity /work/key-identity --output /work/recovered-key "/work/key-backups/${keys[0]##*/}"
cmp -s "$run/recovered-key" "$MOETOPIA_SECRETS_DIR/encryption_key"
rm -- "$run/recovered-key"
compose images --format json > "$run/images.json"
docker stats --no-stream --format '{{json .}}' $(compose ps -q) > "$run/runtime.jsonl"
printf '{"schema":16,"isolated_restore":true,"wrong_key_rejected":true,"unsafe_target_rejected":true,"transient_state_invalidated":true,"source_unchanged":true,"key_escrow_verified":true,"private_metrics_not_public":true,"restore_seconds":%d}\n' "$elapsed" > "$run/restore.json"
printf 'Release checks passed; private artifacts: %s\n' "$run"
