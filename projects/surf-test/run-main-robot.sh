#!/bin/bash
# Run script for Main Robot (Raspberry Pi 5)
# Logs to both stdout and file with timestamps

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="$SCRIPT_DIR/main-robot.json"
LOG_DIR="$SCRIPT_DIR/logs"
LOG_FILE="$LOG_DIR/main-robot-$(date +%Y%m%d-%H%M%S).log"

# Create log directory if it doesn't exist
mkdir -p "$LOG_DIR"

# Colors for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Starting Main Robot${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "Config:   ${YELLOW}$CONFIG_FILE${NC}"
echo -e "Log file: ${YELLOW}$LOG_FILE${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Log startup info to file
{
    echo "========================================"
    echo "Main Robot Started"
    echo "Timestamp: $(date)"
    echo "Config: $CONFIG_FILE"
    echo "Host: $(hostname)"
    echo "========================================"
    echo ""
} >> "$LOG_FILE"

# Run gorai with output to both stdout and log file
# Use unbuffer or stdbuf to prevent buffering issues
if command -v stdbuf &> /dev/null; then
    stdbuf -oL -eL gorai run "$CONFIG_FILE" 2>&1 | tee -a "$LOG_FILE"
else
    gorai run "$CONFIG_FILE" 2>&1 | tee -a "$LOG_FILE"
fi

# Log shutdown
{
    echo ""
    echo "========================================"
    echo "Main Robot Stopped"
    echo "Timestamp: $(date)"
    echo "========================================"
} >> "$LOG_FILE"

