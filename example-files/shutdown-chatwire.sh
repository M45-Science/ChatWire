#!/usr/bin/env bash
set -euo pipefail

for letter in {a..r}; do
    /usr/bin/systemctl stop "chatwire-$letter"
done
