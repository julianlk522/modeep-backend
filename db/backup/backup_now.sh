#!/bin/sh

NOW=$(date -u +%m-%d-%y_%H:%M:%S)
mkdir $NOW

sqlite3 ../fitm.db ".backup '$NOW/fitm_$NOW.bak'"
sqlite3 ../fitm.db .dump | gzip -c > $NOW/fitm_$NOW.dump.gz