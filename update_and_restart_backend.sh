#!/usr/bin/env bash
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1"
}

echo
log "UPDATE AND RESTART"

LOG_DIR="/var/log/modeep"
LOG_FILE="$LOG_DIR/update.log"

# Create log file if it doesn't exist
if [[ ! -f "$LOG_FILE" ]]; then
    mkdir -p "$LOG_DIR"
    touch "$LOG_FILE"
fi

# Pull changes
if [[ -z "$MODEEP_BACKEND_ROOT" ]]; then
    log "Error: MODEEP_BACKEND_ROOT is not set"
    exit 1
fi

cd "$MODEEP_BACKEND_ROOT"
git stash
git pull
git stash pop

# Replace backup with current
if [[ -f "modeep" && -f "modeep.old" ]]; then
    rm modeep.old
    mv modeep modeep.old
fi

# Update dependencies, rebuild
go mod tidy
./build.sh
log "Built!"

# Stop old process
PID=$(pgrep -f modeep)
kill $PID

# Start tmux session if it doesn't exist
if ! tmux has-session -t modeep-backend 2>/dev/null; then
    tmux new-session -d -s modeep-backend
    log "Created modeep-backend tmux session"
fi

# Run fresh binary 
tmux send-keys -t modeep-backend "cd $MODEEP_BACKEND_ROOT && ./modeep" ENTER

log "Update complete and server restarted"
