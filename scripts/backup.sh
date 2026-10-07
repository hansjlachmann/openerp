#!/usr/bin/env bash
#
# Back up the production database (pg_dump custom format) and remove old automatic backups.
#
#   scripts/backup.sh
#
# Settings (environment variables):
#   BACKUP_DIR   where the dumps go (default: ~/backups — outside the git checkout)
#   KEEP_DAYS    automatic backups older than this are deleted (default: 14)
#
# Files are named openerp-auto-YYYYMMDD-HHMM.dump; only those are ever pruned, so dumps made by
# hand (e.g. before an upgrade) are kept. Run it daily from cron, e.g.:
#   15 2 * * * $HOME/openerp/scripts/backup.sh >> $HOME/backups/backup.log 2>&1
# Copying backups to another machine is a separate step (they are not safe on the same disk).

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKUP_DIR="${BACKUP_DIR:-$HOME/backups}"
KEEP_DAYS="${KEEP_DAYS:-14}"

mkdir -p "$BACKUP_DIR"
file="$BACKUP_DIR/openerp-auto-$(date +%Y%m%d-%H%M).dump"
tmp="$file.partial"

# Dump through the running db container (no published port needed)
if ! "$ROOT/scripts/prod.sh" exec -T db pg_dump -U openerp -d openerp -Fc > "$tmp"; then
	rm -f "$tmp"
	echo "$(date '+%F %T') backup FAILED" >&2
	exit 1
fi
if [[ ! -s "$tmp" ]]; then
	rm -f "$tmp"
	echo "$(date '+%F %T') backup FAILED: empty dump" >&2
	exit 1
fi
mv "$tmp" "$file"
echo "$(date '+%F %T') backup written: $file ($(du -h "$file" | cut -f1))"

# Prune old automatic backups (only after a successful one)
find "$BACKUP_DIR" -maxdepth 1 -name 'openerp-auto-*.dump' -mtime +"$KEEP_DAYS" -print -delete |
	sed 's/^/pruned: /'
