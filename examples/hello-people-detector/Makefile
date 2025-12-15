# Makefile for hello-people-detector example
#
# A camera robot with AI-based person detection using an external service.
# The person detector runs in a container with Hailo NPU access.

# ============================================================================
# Configuration
# ============================================================================

ROBOT_NAME := hello-people-detector
RDL_FILE := $(ROBOT_NAME).json

# ============================================================================
# Default target
# ============================================================================

.PHONY: help
help:
	@echo "hello-people-detector Example"
	@echo ""
	@echo "A camera robot with AI-based person detection."
	@echo ""
	@echo "Configuration:"
	@echo "  ROBOT_NAME   = $(ROBOT_NAME)"
	@echo "  RDL_FILE     = $(RDL_FILE)"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Build Targets:"
	@echo "  build         Build the robot and person-detector container"
	@echo ""
	@echo "Run Targets:"
	@echo "  run           Run the robot (foreground, Ctrl+C to stop)"
	@echo "  run-background Run as a systemd service (background)"
	@echo "  stop          Stop the background service"
	@echo "  status        Show robot status"
	@echo "  logs          View robot logs"
	@echo ""
	@echo "Utility Targets:"
	@echo "  validate      Validate $(RDL_FILE) configuration"
	@echo "  clean         Clean generated files and containers"
	@echo ""
	@echo "Prerequisites:"
	@echo "  - Camera at /dev/video0"
	@echo "  - Hailo NPU at /dev/hailo0 (optional)"
	@echo "  - NATS server running: systemctl start nats-server"
	@echo "  - Podman installed for container runtime"
	@echo "  - gorai binary in PATH"
	@echo ""
	@echo "Quick Start:"
	@echo "  make build    # Build robot and person-detector container"
	@echo "  make run      # Run the robot (foreground)"
	@echo "  # Open http://localhost:8080 in browser"

.DEFAULT_GOAL := help

# ============================================================================
# Build Targets
# ============================================================================

.PHONY: build
build:
	@echo "==> Building $(ROBOT_NAME)..."
	gorai build --config $(RDL_FILE) --no-cache
	@echo "==> Building person-detector container..."
	podman build --no-cache -t localhost/person-detector:latest ./services/person-detector

.PHONY: rebuild
rebuild: clean-images build
	@echo "==> Full rebuild complete"

# ============================================================================
# Run Targets
# ============================================================================

.PHONY: run
run:
	@echo "==> Running $(ROBOT_NAME)..."
	@echo "    Dashboard: http://localhost:8080"
	@echo "    Press Ctrl+C to stop"
	@echo ""
	gorai run --config $(RDL_FILE)

.PHONY: run-background
run-background:
	@echo "==> Starting $(ROBOT_NAME) as background service..."
	gorai start --config $(RDL_FILE)
	@echo ""
	@echo "Robot started! Dashboard: http://localhost:8080"
	@echo "Use 'make logs' to view logs, 'make stop' to stop"

.PHONY: stop
stop:
	@echo "==> Stopping $(ROBOT_NAME)..."
	-podman stop person_detector 2>/dev/null || true
	gorai stop --config $(RDL_FILE)

.PHONY: status
status:
	@gorai status --config $(RDL_FILE)

.PHONY: logs
logs:
	gorai logs --config $(RDL_FILE) -f

# ============================================================================
# Utility Targets
# ============================================================================

.PHONY: validate
validate:
	@echo "==> Validating $(RDL_FILE)..."
	gorai validate $(ROBOT_NAME)

.PHONY: clean
clean: stop
	@echo "==> Cleaning up..."
	-podman rm person_detector 2>/dev/null || true
	rm -rf .gorai/
	@echo "==> Done"

.PHONY: clean-images
clean-images: clean
	@echo "==> Removing container images..."
	-podman rmi localhost/person-detector:latest 2>/dev/null || true
	@echo "==> Done"
