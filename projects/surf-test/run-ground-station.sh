#!/bin/bash
# Run script for Ground Station (PC)
# Logs to both stdout and file with timestamps

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="$SCRIPT_DIR/ground-station.json"
LOG_DIR="$SCRIPT_DIR/logs"
LOG_FILE="$LOG_DIR/ground-station-$(date +%Y%m%d-%H%M%S).log"

# Create log directory if it doesn't exist
mkdir -p "$LOG_DIR"

# Colors for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Starting Ground Station${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "Config:   ${YELLOW}$CONFIG_FILE${NC}"
echo -e "Log file: ${YELLOW}$LOG_FILE${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if RASPBERRY_PI_IP needs to be configured
if grep -q "RASPBERRY_PI_IP" "$CONFIG_FILE"; then
    echo -e "${RED}WARNING: You need to update the NATS URL in ground-station.json${NC}"
    echo -e "${YELLOW}Replace 'RASPBERRY_PI_IP' with your Raspberry Pi's IP address${NC}"
    echo ""
    read -p "Enter Raspberry Pi IP address (or press Enter to continue anyway): " PI_IP
    if [ -n "$PI_IP" ]; then
        echo -e "${GREEN}Tip: Update ground-station.json with: \"url\": \"nats://$PI_IP:4222\"${NC}"
        echo ""
    fi
fi

# Log startup info to file
{
    echo "========================================"
    echo "Ground Station Started"
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
    echo "Ground Station Stopped"
    echo "Timestamp: $(date)"
    echo "========================================"
} >> "$LOG_FILE"

