#!/usr/bin/env bash
set -euo pipefail

USER_NAME=$(whoami)

for letter in {a..r}; do
    : > "/home/$USER_NAME/cw-$letter/.queue"
done
