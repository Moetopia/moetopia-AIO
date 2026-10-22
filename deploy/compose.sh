#!/bin/sh
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)
exec docker compose -p moetopia-production --env-file "$root/deploy/images.env" --env-file "$root/.env.production" -f "$root/compose.production.yml" "$@"
