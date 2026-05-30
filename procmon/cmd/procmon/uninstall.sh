#!/usr/bin/env bash
# Remove procmon from a remote host. Leaves /var/log/procmon/*.jsonl behind
# by default — pass --purge to also delete collected data.
#
# Usage:  ./uninstall.sh [--purge] <user@host> [<user@host> ...]

set -euo pipefail

PURGE=0
if [[ "${1:-}" == "--purge" ]]; then
    PURGE=1
    shift
fi

if [[ $# -lt 1 ]]; then
    echo "usage: $0 [--purge] <user@host> [<user@host> ...]" >&2
    exit 2
fi

for target in "$@"; do
    echo "==> $target"
    ssh "$target" bash -s "$PURGE" <<'EOF'
set -euo pipefail
PURGE=$1
rm -f /etc/cron.d/procmon
rm -f /usr/local/bin/procmon
if [[ "$PURGE" == "1" ]]; then
    rm -rf /var/log/procmon
fi
if command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet cron; then
    systemctl reload cron || true
elif command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet crond; then
    systemctl reload crond || true
fi
echo "removed."
EOF
done
