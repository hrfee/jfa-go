#!/bin/bash

if [[ "$1" == "html" ]]; then
    for f in $2/*.html; do
        for color in neutral positive urge warning info critical; do
            sed -i "s/~${color}/~${color} dark:~d_${color}/g" $f
        done
    done
elif [[ "$1" == "ts" ]]; then
    for f in $2/*.ts; do
        # FIXME: inline html
        for l in $(grep -n "~neutral\|~positive\|~urge\|~warning\|~info\|~critical" $f | sed -e 's/:.*//g'); do
        # for l in $(sed -n '/classList/=' $f); do
            line=$(sed -n "${l}p" $f)
            echo $line | grep "classList" &> /dev/null
            if [ $? -eq 0 ]; then
                echo $line | sed 's/.*classList//; s/).*//' | grep "~neutral\|~positive\|~urge\|~warning\|~info\|~critical" &> /dev/null
                if [ $? -eq 0 ]; then
                    echo "found classList @ " $l
                    for color in neutral positive urge warning info critical; do
                        sed -i "${l},${l}s/\"~${color}\"/\"~${color}\", \"dark:~d_${color}\"/g" $f
                    done
                else
                    echo "FIX: classList found, but color tag wasn't in it"
                fi
            else
                echo "found inline @ " $l ", " $(sed -n "${l}p" $f)
                sed -i "${l},${l}s/~${color}/~${color} dark:~d_${color}/g" $f
            fi
        done
    done
fi
