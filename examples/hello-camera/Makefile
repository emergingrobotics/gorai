# Makefile for hello-camera example
#
# This is a simple camera example that streams video to a web dashboard.
# No containers - just run the robot directly with the gorai binary.

# ============================================================================
# Configuration
# ============================================================================

ROBOT_NAME := hello-camera
RDL_FILE := $(ROBOT_NAME).json

# ============================================================================
# Default target
# ============================================================================

.PHONY: help
help:
	@echo "hello-camera Example"
	@echo ""
	@echo "A simple camera example that streams video to a web dashboard."
	@echo ""
	@echo "Configuration:"
	@echo "  ROBOT_NAME   = $(ROBOT_NAME)"
	@echo "  RDL_FILE     = $(RDL_FILE)"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Main Targets:"
	@echo "  run           Run the robot (foreground, Ctrl+C to stop)"
	@echo "  run-background Run as a systemd service (background)"
	@echo "  stop          Stop the background service"
	@echo "  status        Show robot status"
	@echo "  logs          View robot logs"
	@echo "  validate      Validate $(RDL_FILE) configuration"
	@echo ""
	@echo "Utility Targets:"
	@echo "  clean         Clean generated files"
	@echo ""
	@echo "Prerequisites:"
	@echo "  - Camera at /dev/video0"
	@echo "  - NATS server running: nats-server or systemctl start nats-server"
	@echo "  - gorai binary in PATH"
	@echo ""
	@echo "Quick Start:"
	@echo "  make run      # Run the robot (foreground)"
	@echo "  # Open http://localhost:10101 in browser"

.DEFAULT_GOAL := help

# ============================================================================
# Main Targets
# ============================================================================

.PHONY: run
run:
	@echo "==> Running $(ROBOT_NAME)..."
	@echo "    Dashboard: http://localhost:10101"
	@echo "    Press Ctrl+C to stop"
	@echo ""
	gorai run --config $(RDL_FILE)

.PHONY: run-background
run-background:
	@echo "==> Starting $(ROBOT_NAME) as background service..."
	gorai start --config $(RDL_FILE)
	@echo ""
	@echo "Robot started! Dashboard: http://localhost:10101"
	@echo "Use 'make logs' to view logs, 'make stop' to stop"

.PHONY: stop
stop:
	@echo "==> Stopping $(ROBOT_NAME)..."
	gorai stop --config $(RDL_FILE)

.PHONY: status
status:
	@gorai status --config $(RDL_FILE)

.PHONY: logs
logs:
	gorai logs --config $(RDL_FILE) -f

.PHONY: validate
validate:
	@echo "==> Validating $(RDL_FILE)..."
	gorai validate $(ROBOT_NAME)

# ============================================================================
# Utility Targets
# ============================================================================

.PHONY: clean
clean:
	@echo "==> Cleaning up..."
	-gorai stop --config $(RDL_FILE) 2>/dev/null || true
	rm -rf .gorai/
	@echo "==> Done"
