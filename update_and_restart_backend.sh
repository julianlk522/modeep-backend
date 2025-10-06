#!/usr/bin/env bash

LOG_FILE="/var/log/modeep/update.log"
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1"
}

echo
log "UPDATE AND RESTART"

# redirect non-explicit output to log file (append)
exec >> "$LOG_FILE" 2>&1
# "exec >> {arg}" replaces current shell process (modifying stdout file descriptor) with {arg} output for later script commands
# "2>&1" redirects stderr (file descriptor 2) to stdout (1)

# pull changes
if [ -z "$MODEEP_BACKEND_ROOT" ]; then
    log "error: MODEEP_BACKEND_ROOT is not set"
    exit 1
fi
cd "$MODEEP_BACKEND_ROOT" || { log "error: could not navigate to $MODEEP_BACKEND_ROOT"; exit 1; }
git pull

# update dependencies, rebuild
go mod tidy
./build.sh
log "build complete"

# gracefully stop running server process
PID=$(pgrep -f modeep)
kill $PID

# start tmux session if one doesn't already exist
if ! tmux has-session -t modeep-backend 2>/dev/null; then
    log "creating new modeep-backend tmux session"
    tmux new-session -d -s modeep-backend
fi

# run fresh binary 
tmux send-keys -t modeep-backend "cd $MODEEP_BACKEND_ROOT && ./modeep" ENTER

# detach
tmux detach -s modeep-backend

log "update complete and server restarted"
