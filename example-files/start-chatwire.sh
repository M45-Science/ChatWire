#!/usr/bin/env bash
set -euo pipefail

for letter in {a..r}; do
    /usr/bin/systemctl restart "chatwire-$letter"
done
