#!/usr/bin/env bash

NOW=$(date -u +%m-%d-%y_%H:%M:%S)
mkdir $NOW

sqlite3 ../modeep.db ".backup '$NOW/modeep_$NOW.bak'"
sqlite3 ../modeep.db .dump | gzip -c > $NOW/modeep_$NOW.dump.gz