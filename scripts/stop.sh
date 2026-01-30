#!/bin/bash

# Definition of colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

PIDS_FILE=".project.pids"

echo -e "${BLUE}=== Stopping Project ===${NC}"

# 1. Kill Processes
if [ -f "$PIDS_FILE" ]; then
    echo -e "${BLUE}[1/2] Stopping Background Services...${NC}"
    while read -r PID; do
        if ps -p $PID > /dev/null; then
            echo "Killing process $PID"
            kill $PID 2>/dev/null || true
            # Wait a bit or force kill if needed? 
            # Usually kill is SIGTERM, which is good.
        else
            echo "Process $PID already dead"
        fi
    done < "$PIDS_FILE"
    rm "$PIDS_FILE"
    
    # Extra cleanup for Vite/Node if they spawned children
    pkill -f "vite" || true
else
    echo "No PIDs file found. Checking for strays..."
    # Optional: Safety net
    pkill -f "bin/api" || true
    pkill -f "bin/worker" || true
    pkill -f "bin/consumer" || true
    pkill -f "bin/payment" || true
    pkill -f "bin/ticket" || true
fi

# 2. Stop Docker
echo -e "${BLUE}[2/2] Stopping Infrastructure...${NC}"
docker-compose -p web_app down

# Cleanup Logs (Optional, user might want to read them)
# rm *.log

echo -e "${GREEN}All services stopped.${NC}"
