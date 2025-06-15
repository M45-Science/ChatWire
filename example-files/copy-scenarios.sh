#!/bin/bash

for letter in {a..r}; do
    cp -rf ~/cw-a/factorio/data/base/scenarios ~/cw-$letter/factorio/
done
