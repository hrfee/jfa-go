#!/bin/bash

for f in $1/*.html; do
    for color in neutral positive urge warning info critical; do
        sed -i "s/~${color}/~${color} dark:~d_${color}/g" $f
    done
done
