#!/usr/bin/env bash
#
# docker compose for production, with the server's settings from prod.env:
#
#   scripts/prod.sh pull          # download the images of APP_VERSION
#   scripts/prod.sh up -d         # start / apply changes
#   scripts/prod.sh ps | logs backend --since 5m | down
#
# Runs from the checkout folder whatever the current directory: the project name (and so the
# database volume, <folder>_postgres_data) comes from that folder.
# Never run "down -v" or "docker volume prune": they delete the database volume.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [[ ! -f prod.env ]]; then
	echo "prod.env not found in $ROOT." >&2
	echo "Create it: cp prod.env.example prod.env && chmod 600 prod.env, then fill it in (docs/operations.md)." >&2
	exit 1
fi

exec docker compose --env-file prod.env -f docker-compose.prod.yml "$@"
