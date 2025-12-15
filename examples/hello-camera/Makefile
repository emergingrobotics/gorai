# Makefile for hello-camera example
#
# This is a simple camera example that streams video to a web dashboard.
# No containers to build - just run the robot directly.

# ============================================================================
# Configuration
# ============================================================================

ROBOT_NAME := hello-camera
RDL_FILE := $(ROBOT_NAME).json

# Gorai CLI container image (for portable execution)
GORAI_IMAGE ?= localhost/gorai:latest

# Path to gorai source (for building the CLI container)
GORAI_SRC := ../..

# Podman socket (auto-detected)
PODMAN_SOCK := $(shell if [ -S /run/podman/podman.sock ]; then echo /run/podman/podman.sock; else echo /run/user/$$(id -u)/podman/podman.sock; fi)

# ============================================================================
# Gorai container command
# ============================================================================

# Run gorai CLI in container with access to devices
gorai = podman run --rm -it \
	--security-opt label=disable \
	--device /dev/video0:/dev/video0 \
	-v $(PWD):/workspace:rw \
	-w /workspace \
	-p 8080:8080 \
	-e GORAI_ROBOT_NAME=$(ROBOT_NAME) \
	$(GORAI_IMAGE)

# Non-interactive version for CI/scripts
gorai_batch = podman run --rm \
	--security-opt label=disable \
	-v $(PWD):/workspace:rw \
	-w /workspace \
	-e GORAI_ROBOT_NAME=$(ROBOT_NAME) \
	$(GORAI_IMAGE)

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
	@echo "  GORAI_IMAGE  = $(GORAI_IMAGE)"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Main Targets:"
	@echo "  run           Run the robot (foreground, Ctrl+C to stop)"
	@echo "  run-native    Run with native gorai binary (no container)"
	@echo "  validate      Validate $(RDL_FILE) configuration"
	@echo ""
	@echo "Setup Targets:"
	@echo "  container     Build the gorai CLI container image"
	@echo "  shell         Open shell in gorai container"
	@echo "  clean         Clean generated files"
	@echo ""
	@echo "Prerequisites:"
	@echo "  - Camera at /dev/video0"
	@echo "  - NATS server running: nats-server or systemctl start nats-server"
	@echo "  - Gorai container: make container"
	@echo ""
	@echo "Quick Start:"
	@echo "  make container  # Build gorai CLI container (once)"
	@echo "  make run        # Run the robot"
	@echo "  # Open http://localhost:8080 in browser"

.DEFAULT_GOAL := help

# ============================================================================
# Main Targets
# ============================================================================

.PHONY: run
run: _check-container
	@echo "==> Running $(ROBOT_NAME)..."
	@echo "    Dashboard: http://localhost:8080"
	@echo "    Press Ctrl+C to stop"
	@echo ""
	$(gorai) run --config $(RDL_FILE)

.PHONY: run-native
run-native:
	@echo "==> Running $(ROBOT_NAME) with native gorai..."
	@echo "    Dashboard: http://localhost:8080"
	@echo "    Press Ctrl+C to stop"
	@echo ""
	gorai run --config $(RDL_FILE)

.PHONY: validate
validate: _check-container
	@echo "==> Validating $(RDL_FILE)..."
	$(gorai_batch) validate $(ROBOT_NAME)

# ============================================================================
# Setup Targets
# ============================================================================

.PHONY: container
container:
	@echo "==> Building gorai CLI container..."
	cd $(GORAI_SRC) && podman build -t $(GORAI_IMAGE) -f Containerfile.gorai .
	@echo "==> Container built: $(GORAI_IMAGE)"

.PHONY: shell
shell: _check-container
	@echo "==> Opening shell in gorai container..."
	podman run --rm -it \
		--security-opt label=disable \
		-v $(PWD):/workspace:rw \
		-w /workspace \
		--entrypoint /bin/bash \
		$(GORAI_IMAGE)

.PHONY: clean
clean:
	@echo "==> Cleaning up..."
	rm -rf .gorai/
	@echo "==> Done"

# ============================================================================
# Internal Targets
# ============================================================================

.PHONY: _check-container
_check-container:
	@if ! podman image exists $(GORAI_IMAGE) 2>/dev/null; then \
		echo "Error: Gorai container image not found: $(GORAI_IMAGE)"; \
		echo "Build it with: make container"; \
		exit 1; \
	fi
