#!/bin/bash
# Start Gorai core services
# Usage: ./scripts/start.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
PID_DIR="$PROJECT_DIR/.pids"
LOG_DIR="$PROJECT_DIR/.logs"

# Create directories
mkdir -p "$PID_DIR" "$LOG_DIR"

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

# Check if a process is running
is_running() {
    local pid_file="$1"
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            return 0
        fi
    fi
    return 1
}

# Start NATS server
start_nats() {
    local pid_file="$PID_DIR/nats.pid"
    local log_file="$LOG_DIR/nats.log"

    if is_running "$pid_file"; then
        log_warn "NATS server already running (PID: $(cat "$pid_file"))"
        return 0
    fi

    # Check if NATS is already running (not started by us)
    if nc -z localhost 4222 2>/dev/null; then
        log_warn "NATS server already running on port 4222 (external)"
        return 0
    fi

    # Check if nats-server is installed
    if ! command -v nats-server &>/dev/null; then
        log_error "nats-server not found. Install with: go install github.com/nats-io/nats-server/v2@latest"
        return 1
    fi

    log_info "Starting NATS server..."

    # Try to find an available monitoring port
    local monitor_port=8222
    for port in 8222 8223 8224 8225; do
        if ! nc -z localhost $port 2>/dev/null; then
            monitor_port=$port
            break
        fi
    done

    nats-server \
        -p 4222 \
        -m $monitor_port \
        -js \
        -sd "$PROJECT_DIR/.nats-data" \
        > "$log_file" 2>&1 &

    local pid=$!
    echo "$pid" > "$pid_file"

    # Wait for NATS to be ready
    sleep 1
    if is_running "$pid_file"; then
        log_info "NATS server started (PID: $pid)"
        log_info "  - Client port: 4222"
        log_info "  - Monitoring:  http://localhost:$monitor_port"
        log_info "  - JetStream:   enabled"
    else
        log_error "Failed to start NATS server. Check $log_file"
        cat "$log_file" | tail -5
        return 1
    fi
}

# Main
main() {
    log_info "Starting Gorai services..."
    echo ""

    # Start NATS
    if ! start_nats; then
        log_error "Failed to start NATS server"
        exit 1
    fi

    echo ""
    log_info "All services started successfully!"
    echo ""
    log_info "To stop services: ./scripts/stop.sh"
    log_info "To view NATS logs: tail -f $LOG_DIR/nats.log"
    echo ""
    log_info "Example commands:"
    log_info "  nats sub 'gorai.>'"
    log_info "  go run ./examples/hello-sensor -fake"
}

main "$@"
