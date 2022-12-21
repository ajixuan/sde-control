#!/bin/bash
set -e

dbUser=${1}
dbPass="$(cat $2)"
dbHost=${3}
databasePrefix=${4:-sde_}
expectedCount=${5:-4}
sdeDbs=($(PGPASSWORD="${dbPass}" psql --hostname --username "${dbUser}" --list --csv | grep -v -w "${databasePrefix}" | grep "${databasePrefix}" | sort))
dbCount="${#sdeDbs[@]}"

if [ ${expectedCount} -lt 2 ]; then
    echo "expcetedCount cannot be lower than 2"
    exit 2
fi

echo "dbcount is ${dbCount}, expected count is: ${expectedCount}"
cleanupCount=$(( ${dbCount} - ${expectedCount} ))

if [ ${cleanupCount} -gt 0 ]; then
    echo "number of dbs exceed limit, cleaning up last ${cleanupCount} dbs"

    i=0
    while [ ${i} -ne ${cleanupCount} ]; do
        echo "DROP DATABASE ${sdeDbs[${i}]}"
        i=$(( i + 1 ))
        # PGPASSWORD="$(cat ${dbPass})" psql --username ${dbUser} --command "DROP DATABASE ${sdeDbs[${i}]}" 
    done
fi