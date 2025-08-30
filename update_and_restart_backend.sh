#!/usr/bin/env bash

LOG_FILE="/var/log/fitm/update.log"
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
go build --tags 'fts5' .
log "build complete"

# identify running server process
PID=$(pgrep -f fitm)

# send SIGTERM signal to gracefully stop
kill $PID

# while process exists, try to kill it
countdown=10
## (kill -0 evals to status 0 if process exists and 1 if process does not exist)
## (2>/dev/null redirects stderr to null device file to suppress)
while kill -0 $PID 2>/dev/null; do
    ## force if stuck
    if [ $countdown -le 0 ]; then
        kill -9 $PID
        break
    fi
    sleep 1
    ((countdown--))
done
log "stopped process $PID"

# start tmux session if one doesn't already exist
if ! tmux has-session -t fitm-backend 2>/dev/null; then
    log "creating new fitm-backend tmux session"
    tmux new-session -d -s fitm-backend
fi

# run fresh binary in tmux session
tmux send-keys -t fitm-backend "cd $MODEEP_BACKEND_ROOT && ./fitm" ENTER

# detach
tmux detach -s fitm-backend

log "update complete and server restarted"
