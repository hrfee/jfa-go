#!/bin/bash

# scan all typescript and automatically add dark variants to color tags if they're not already present.

for f in $1/*.ts; do
    # FIXME: inline html
    for l in $(grep -n "~neutral\|~positive\|~urge\|~warning\|~info\|~critical" $f | sed -e 's/:.*//g'); do
    # for l in $(sed -n '/classList/=' $f); do
        line=$(sed -n "${l}p" $f)
        echo $line | grep "classList" &> /dev/null
        if [ $? -eq 0 ]; then
            echo $line | sed 's/.*classList//; s/).*//' | grep "~neutral\|~positive\|~urge\|~warning\|~info\|~critical" &> /dev/null
            if [ $? -eq 0 ]; then
                # echo "found classList @ " $l
                echo $line | grep "dark:" &>/dev/null
                if [ $? -ne 0 ]; then
                    for color in neutral positive urge warning info critical; do
                        sed -i "${l},${l}s/\"~${color}\"/\"~${color}\", \"dark:~d_${color}\"/g" $f
                    done
                fi
            else
                echo "FIX: classList found, but color tag wasn't in it"
            fi
        else
            echo $line | grep "querySelector" &> /dev/null
            if [ $? -ne 0 ]; then
                # echo "found inline in " $f " @ " $l ", " $(sed -n "${l}p" $f)
                echo $line | grep "dark:" &>/dev/null
                if [ $? -ne 0 ]; then
                    for color in neutral positive urge warning info critical; do
                        sed -i "${l},${l}s/~${color}/~${color} dark:~d_${color}/g" $f
                    done
                fi
            else
                echo $line | sed 's/.*querySelector//; s/).*//' | grep "~neutral\|~positive\|~urge\|~warning\|~info\|~critical" &> /dev/null
                if [ $? -ne 0 ]; then
                    echo $line | grep "dark:" &>/dev/null
                    if [ $? -ne 0 ]; then
                        # echo "found inline in " $f " @ " $l ", " $(sed -n "${l}p" $f)
                        for color in neutral positive urge warning info critical; do
                            sed -i "${l},${l}s/~${color}/~${color} dark:~d_${color}/g" $f
                        done
                    fi
                #else
                    #echo "FIX: querySelector found, but color tag wasn't in it: " $line
                fi
            fi
        fi
    done
done
