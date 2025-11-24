#!/usr/bin/env bash
set -euo pipefail

# Output directory (change if needed)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTDIR="${1:-$SCRIPT_DIR}"
mkdir -p "$OUTDIR"

SERVICE_USER="fact2"
SERVICE_HOME="/home/$SERVICE_USER"

for letter in {a..r}; do
    svc="chatwire-$letter.service"
    workdir="$SERVICE_HOME/cw-$letter"

    cat > "$OUTDIR/$svc" <<EOF2
[Unit]
Description=chatwire-$letter
Wants=network-online.target
After=network.target network-online.target

[Service]
User=$SERVICE_USER
WorkingDirectory=$workdir/
ExecStart=$workdir/start.sh

KillSignal=SIGTERM
TimeoutStopSec=300s
KillMode=process
SendSIGKILL=no

Restart=always
RestartSec=1
StartLimitBurst=3
StartLimitIntervalSec=1

[Install]
WantedBy=multi-user.target
EOF2

    echo "Generated: $OUTDIR/$svc"
done
