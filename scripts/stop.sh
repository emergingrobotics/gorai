#!/bin/bash
# Stop Gorai core services
# Usage: ./scripts/stop.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
PID_DIR="$PROJECT_DIR/.pids"
LOG_DIR="$PROJECT_DIR/.logs"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Stop a process by PID file
stop_process() {
    local name="$1"
    local pid_file="$2"
    local timeout="${3:-5}"

    if [ ! -f "$pid_file" ]; then
        log_warn "$name: no PID file found"
        return 0
    fi

    local pid=$(cat "$pid_file")

    if ! kill -0 "$pid" 2>/dev/null; then
        log_warn "$name: process not running (stale PID file)"
        rm -f "$pid_file"
        return 0
    fi

    log_info "Stopping $name (PID: $pid)..."

    # Send SIGTERM
    kill -TERM "$pid" 2>/dev/null || true

    # Wait for graceful shutdown
    local count=0
    while kill -0 "$pid" 2>/dev/null && [ $count -lt $timeout ]; do
        sleep 1
        count=$((count + 1))
    done

    # Force kill if still running
    if kill -0 "$pid" 2>/dev/null; then
        log_warn "$name: forcing shutdown..."
        kill -KILL "$pid" 2>/dev/null || true
        sleep 1
    fi

    if kill -0 "$pid" 2>/dev/null; then
        log_error "$name: failed to stop process"
        return 1
    fi

    rm -f "$pid_file"
    log_info "$name stopped"
    return 0
}

# Stop NATS server
stop_nats() {
    stop_process "NATS server" "$PID_DIR/nats.pid" 5
}

# Clean up logs (optional)
cleanup_logs() {
    if [ "$1" = "--clean" ]; then
        log_info "Cleaning up logs..."
        rm -rf "$LOG_DIR"/*.log
        rm -rf "$PROJECT_DIR/.nats-data"
    fi
}

# Main
main() {
    log_info "Stopping Gorai services..."
    echo ""

    # Stop services in reverse order
    stop_nats

    # Handle cleanup flag
    cleanup_logs "$1"

    echo ""
    log_info "All services stopped"
}

main "$@"
