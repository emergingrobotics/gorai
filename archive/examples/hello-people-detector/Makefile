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
	@echo "A camera robot with AI-based person detection using Hailo-8 NPU."
	@echo ""
	@echo "Configuration:"
	@echo "  ROBOT_NAME   = $(ROBOT_NAME)"
	@echo "  RDL_FILE     = $(RDL_FILE)"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Setup Targets:"
	@echo "  setup-hailo   Copy Hailo runtime files for container build"
	@echo ""
	@echo "Build Targets:"
	@echo "  build         Build the robot and person-detector container"
	@echo ""
	@echo "Run Targets:"
	@echo "  run             Run the robot (background, logs to file)"
	@echo "  run-foreground  Run in foreground (Ctrl+C to stop)"
	@echo "  stop            Stop the robot"
	@echo "  status          Show robot status"
	@echo "  logs            View gorai logs (tail -f)"
	@echo "  logs-detector   View person-detector container logs"
	@echo ""
	@echo "Utility Targets:"
	@echo "  validate      Validate $(RDL_FILE) configuration"
	@echo "  clean         Clean generated files and containers"
	@echo ""
	@echo "Prerequisites:"
	@echo "  - Camera at /dev/video0"
	@echo "  - Hailo-8 NPU at /dev/hailo0"
	@echo "  - NATS server running: systemctl start nats-server"
	@echo "  - Podman installed for container runtime"
	@echo "  - gorai binary in PATH"
	@echo "  - Hailo software: sudo apt install hailo-all"
	@echo ""
	@echo "Quick Start:"
	@echo "  make setup-hailo  # Copy Hailo runtime files (run once)"
	@echo "  make build        # Build robot and person-detector container"
	@echo "  make run          # Start the robot (background)"
	@echo "  # Open http://localhost:8080 in browser"
	@echo "  make logs-detector # View inference timing logs"
	@echo "  make stop         # Stop when done"

.DEFAULT_GOAL := help

# ============================================================================
# Setup Targets
# ============================================================================

.PHONY: setup-hailo
setup-hailo:
	@echo "==> Setting up Hailo runtime files..."
	./services/person-detector/setup-hailo.sh

# ============================================================================
# Build Targets
# ============================================================================

.PHONY: build
build:
	@if [ ! -d "./services/person-detector/hailo_runtime/lib" ]; then \
		echo "ERROR: hailo_runtime not found. Run 'make setup-hailo' first."; \
		exit 1; \
	fi
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

LOG_DIR := .gorai/logs
GORAI_LOG := $(LOG_DIR)/gorai.log

.PHONY: run
run:
	@mkdir -p $(LOG_DIR)
	@echo "==> Starting $(ROBOT_NAME) in background..."
	@gorai run --config $(RDL_FILE) > $(GORAI_LOG) 2>&1 &
	@sleep 1
	@echo ""
	@echo "Robot started!"
	@echo "  Dashboard:    http://localhost:8080"
	@echo "  Gorai logs:   $(GORAI_LOG)"
	@echo "  Detector:     make logs-detector"
	@echo ""
	@echo "Commands:"
	@echo "  make logs          - View gorai logs"
	@echo "  make logs-detector - View person-detector logs"
	@echo "  make stop          - Stop the robot"
	@echo ""

.PHONY: run-foreground
run-foreground:
	@echo "==> Running $(ROBOT_NAME) in foreground..."
	@echo "    Dashboard: http://localhost:8080"
	@echo "    Press Ctrl+C to stop"
	@echo ""
	gorai run --config $(RDL_FILE)

.PHONY: stop
stop:
	@echo "==> Stopping $(ROBOT_NAME)..."
	-podman stop person_detector 2>/dev/null || true
	-pkill -f "gorai run --config $(RDL_FILE)" 2>/dev/null || true
	@echo "Stopped."

.PHONY: status
status:
	@gorai status --config $(RDL_FILE)

.PHONY: logs
logs:
	@if [ -f $(GORAI_LOG) ]; then \
		tail -f $(GORAI_LOG); \
	else \
		echo "No log file found. Is the robot running?"; \
		echo "Run 'make run' first."; \
	fi

.PHONY: logs-detector
logs-detector:
	@echo "==> Person detector container logs (Ctrl+C to stop)..."
	podman logs -f person_detector

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
