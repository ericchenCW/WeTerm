#!/usr/bin/env bash
# Deploy procmon to a remote host: copy the binary, install cron entries,
# and verify the first collect runs cleanly.
#
# Usage:  ./deploy.sh <user@host> [<user@host> ...]
#
# Requires:  the local file ./procmon-linux-amd64 (build with `make build-linux`)
#            ssh + scp access to each target as a user able to write to
#            /usr/local/bin and /etc/cron.d (typically root)

set -euo pipefail

BIN_LOCAL="${BIN_LOCAL:-./procmon-linux-amd64}"
BIN_REMOTE="/usr/local/bin/procmon"
CRON_FILE="/etc/cron.d/procmon"
DATA_DIR="/var/log/procmon"

if [[ $# -lt 1 ]]; then
    echo "usage: $0 <user@host> [<user@host> ...]" >&2
    exit 2
fi

if [[ ! -f "$BIN_LOCAL" ]]; then
    echo "binary not found: $BIN_LOCAL (run 'make build-linux' first)" >&2
    exit 1
fi

for target in "$@"; do
    echo "==> $target"
    scp -q "$BIN_LOCAL" "$target:/tmp/procmon.new"

    # Stage to /tmp first so we can chmod, then mv atomically.
    # Cron file is written via heredoc; SHELL=/bin/sh keeps cron predictable
    # under any user default shell.
    ssh "$target" bash -s <<EOF
set -euo pipefail
chmod 0755 /tmp/procmon.new
install -m 0755 /tmp/procmon.new $BIN_REMOTE
rm -f /tmp/procmon.new
mkdir -p $DATA_DIR
cat > $CRON_FILE <<'CRON'
SHELL=/bin/sh
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
# Sample every minute
* * * * * root $BIN_REMOTE collect --data-dir $DATA_DIR
# Prune daily at 03:00
0 3 * * * root $BIN_REMOTE prune --data-dir $DATA_DIR --keep-days 7
CRON
chmod 0644 $CRON_FILE
# Some systems require explicit reload
if command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet cron; then
    systemctl reload cron || true
elif command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet crond; then
    systemctl reload crond || true
fi
# Smoke test
$BIN_REMOTE collect --data-dir $DATA_DIR
ls -la $DATA_DIR
EOF
    echo "==> $target done"
done

echo "all hosts deployed."
