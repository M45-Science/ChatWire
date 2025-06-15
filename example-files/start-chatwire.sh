#!/bin/bash

for letter in {a..r}; do
   /usr/bin/systemctl restart cw&letter&
done
