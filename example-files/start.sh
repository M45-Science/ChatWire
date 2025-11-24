#!/usr/bin/env bash
set -euo pipefail

# If slash commands are modified or added use this instead:
#   ./ChatWire -deregCommands -regCommands

# Always exec so systemd can deliver signals directly to ChatWire for graceful
# shutdowns during commands such as "sudo reboot".
cd "$(dirname "$(readlink -f "$0")")"
exec ./ChatWire -proxy="http://127.0.0.1:55555"
