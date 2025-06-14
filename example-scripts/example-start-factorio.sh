#!/bin/bash

USER_NAME=$(whoami)

for letter in {a..r}; do
    echo > "/home/$USER_NAME/cw-$letter/.start"
done