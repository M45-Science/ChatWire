#!/bin/bash

if [ ! "$BASH_VERSION" ] ; then
        echo "Not launched with bash, re-running with bash."
        bash build.sh
    exit 1
fi

USER_NAME=$(whoami)

for letter in {a..r}; do
    echo > "/home/$USER_NAME/cw-$letter/.rebootcw"
done