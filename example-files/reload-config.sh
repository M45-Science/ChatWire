#!/usr/bin/env bash
set -euo pipefail

# Reload ChatWire configuration on all instances by sending SIGUSR2.
# Signals all running `ChatWire` processes.

if command -v pgrep >/dev/null 2>&1; then
  pids="$(pgrep -x ChatWire || true)"
  if [[ -z "${pids}" ]]; then
    echo "No ChatWire processes found." >&2
    exit 1
  fi
  for pid in ${pids}; do
    kill -USR2 "${pid}"
  done
  exit 0
fi

echo "pgrep is not available; unable to reload configs." >&2
exit 1
