#!/usr/bin/env bash
set -euo pipefail

pids="$(pgrep -x ChatWire || true)"
if [ -z "${pids}" ]; then
    echo "No ChatWire processes found." >&2
    exit 1
fi

for pid in ${pids}; do
    kill -USR1 "${pid}"
done
