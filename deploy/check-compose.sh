#!/bin/sh
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)
project="${MOETOPIA_RELEASE_CHECK_PROJECT:-moetopia-release-check}"
case "$project" in moetopia-release-check|moetopia-release-check-????????????????) ;; *) printf '%s\n' 'Unsafe release-check project name.' >&2; exit 1 ;; esac
exec docker compose -p "$project" --env-file "$root/deploy/images.env" --env-file "$root/deploy/check.env" -f "$root/compose.production.yml" -f "$root/compose.release-check.yml" "$@"
