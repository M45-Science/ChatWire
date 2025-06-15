#!/bin/bash

for letter in {a..r}; do
   /usr/bin/systemctl stop cw&letter&
done
