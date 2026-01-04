#!/usr/bin/env bash
set -euo pipefail

pids=$(pgrep -f "ChatWire" || true)
if [ -z "${pids}" ]; then
    echo "No ChatWire processes found." >&2
    exit 1
fi

kill -USR1 ${pids}
