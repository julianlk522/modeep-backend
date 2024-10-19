#!/bin/bash

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
if [ -z "$FITM_BACKEND_ROOT" ]; then
    log "error: FITM_BACKEND_ROOT is not set"
    exit 1
fi
cd "$FITM_BACKEND_ROOT" || { log "error: could not navigate to $FITM_BACKEND_ROOT"; exit 1; }
git pull

# update dependencies, rebuild
go mod tidy
go build --tags 'fts5' .
log "build complete"

# interrupt running server process(es)
PIDs=$(pgrep -f fitm)
log "found PID(s): $PIDs"
if [ -n "$PIDs" ]; then
    for PID in $PIDs; do
        log "attempting to stop process $PID"
        kill $PID
        # send SIGTERM signal to gracefully stop process
        
        # countdown process stop
        countdown=10

        # while process exists
        while kill -0 $PID 2>/dev/null; do
        ## (kill -0 evals to status 0 if process exists and 1 if process does not exist)
        ## (2>/dev/null redirects stderr to null device file to suppress)
            if [ $countdown -le 0 ]; then
                log "countdown exceeded for PID $PID. Forcing kill."
                kill -9 $PID
                break
            fi
            sleep 1
            ((countdown--))
        done
        log "stopped process $PID"
    done
fi
log "all old processes stopped"

# start tmux session if not exists already
if ! tmux has-session -t FITM 2>/dev/null; then
    log "creating new FITM tmux session"
    tmux new-session -d -s FITM
fi

# start new binary in tmux session
tmux send-keys -t FITM "cd $FITM_BACKEND_ROOT && ./fitm" ENTER

# detach
tmux detach -s FITM

log "update complete and server restarted"
