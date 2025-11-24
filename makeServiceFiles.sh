#!/bin/bash

# Output directory (change if needed)
OUTDIR="."
mkdir -p "$OUTDIR"

for letter in {a..r}; do
svc="chatwire-$letter.service"

cat > "$OUTDIR/$svc" <<EOF
[Unit]
Description=chatwire-$letter
Wants=network-online.target
After=network.target network-online.target

[Service]
User=fact2
WorkingDirectory=/home/fact2/cw-$letter/
ExecStart=/bin/bash start.sh

KillSignal=SIGTERM
TimeoutStopSec=60s
KillMode=process
SendSIGKILL=no

Restart=always
RestartSec=1
StartLimitBurst=3
StartLimitIntervalSec=1

[Install]
WantedBy=multi-user.target
EOF

echo "Generated: $OUTDIR/$svc"
done
