#!/bin/bash
# Run the hello-sensor example
# Usage: ./scripts/hello.sh [options]
#
# Options are passed through to hello-sensor, e.g.:
#   ./scripts/hello.sh -fake -fake-temp 50.0
#   ./scripts/hello.sh -interval 500ms

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

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

# Check if NATS is running
check_nats() {
    if ! nc -z localhost 4222 2>/dev/null; then
        log_warn "NATS server not detected on localhost:4222"
        log_info "Starting NATS server..."
        "$SCRIPT_DIR/start.sh"
        echo ""
    fi
}

# Main
main() {
    cd "$PROJECT_DIR"

    # Check NATS
    check_nats

    # Default to fake mode if no args provided
    if [ $# -eq 0 ]; then
        log_info "Running hello-sensor with fake reader (use -h for options)"
        echo ""
        exec go run ./examples/hello-sensor -fake
    else
        log_info "Running hello-sensor..."
        echo ""
        exec go run ./examples/hello-sensor "$@"
    fi
}

main "$@"
