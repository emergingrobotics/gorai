# Makefile for hello-camera example
#
# This Makefile uses containerized gorai CLI for orchestration.
# The gorai CLI runs in a container with podman-compose included.
# Only podman is required on the host.

# ============================================================================
# Configuration
# ============================================================================

ROBOT_NAME := hello-camera
RDL_FILE := $(ROBOT_NAME).json

# Gorai CLI container image
GORAI_IMAGE ?= localhost/gorai:latest

# Path to gorai source (for building the CLI container)
GORAI_SRC := ../..

# Podman socket (auto-detected)
PODMAN_SOCK := $(shell if [ -S /run/podman/podman.sock ]; then echo /run/podman/podman.sock; else echo /run/user/$$(id -u)/podman/podman.sock; fi)

# ============================================================================
# Gorai container command
# ============================================================================

# Run gorai CLI in container with access to podman socket
gorai = podman run --rm -it \
	--security-opt label=disable \
	-v $(PODMAN_SOCK):/run/podman/podman.sock:rw \
	-v $(PWD):/workspace:rw \
	-w /workspace \
	-e GORAI_ROBOT_NAME=$(ROBOT_NAME) \
	$(GORAI_IMAGE)

# Non-interactive version for CI/scripts
gorai_batch = podman run --rm \
	--security-opt label=disable \
	-v $(PODMAN_SOCK):/run/podman/podman.sock:rw \
	-v $(PWD):/workspace:rw \
	-w /workspace \
	-e GORAI_ROBOT_NAME=$(ROBOT_NAME) \
	$(GORAI_IMAGE)

# ============================================================================
# Default target
# ============================================================================

.PHONY: help
help:
	@echo "hello-camera Example Makefile (Containerized)"
	@echo ""
	@echo "Configuration:"
	@echo "  ROBOT_NAME   = $(ROBOT_NAME)"
	@echo "  RDL_FILE     = $(RDL_FILE)"
	@echo "  GORAI_IMAGE  = $(GORAI_IMAGE)"
	@echo "  PODMAN_SOCK  = $(PODMAN_SOCK)"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Container Orchestration:"
	@echo "  up            Start all robot containers (build + detach)"
	@echo "  down          Stop and remove all containers"
	@echo "  status        Show container status"
	@echo "  logs          Follow container logs"
	@echo "  restart       Restart all containers"
	@echo ""
	@echo "Build Targets:"
	@echo "  container     Build the gorai CLI container image"
	@echo "  build         Build robot service containers"
	@echo ""
	@echo "Utility Targets:"
	@echo "  validate      Validate $(RDL_FILE) configuration"
	@echo "  shell         Open shell in gorai container"
	@echo "  clean         Stop containers and clean generated files"
	@echo ""
	@echo "Prerequisites:"
	@echo "  1. Install podman: sudo apt install podman"
	@echo "  2. Enable socket:  systemctl --user enable --now podman.socket"
	@echo "  3. Build image:    make container"
	@echo ""
	@echo "Quick Start:"
	@echo "  make container  # Build gorai CLI container (once)"
	@echo "  make up         # Start all services"
	@echo "  make logs       # View logs"
	@echo "  make down       # Stop everything"

.DEFAULT_GOAL := help

# ============================================================================
# Container Orchestration Targets
# ============================================================================

.PHONY: up
up: _check-container _check-socket
	@echo "==> Starting $(ROBOT_NAME) containers..."
	$(gorai) start --config $(RDL_FILE) --build --detach
	@echo ""
	@echo "Robot started! View dashboard at http://localhost:8080"
	@echo "Use 'make logs' to follow logs, 'make down' to stop"

.PHONY: down
down: _check-container _check-socket
	@echo "==> Stopping $(ROBOT_NAME) containers..."
	$(gorai) stop --config $(RDL_FILE)

.PHONY: status
status: _check-container _check-socket
	@$(gorai) status --config $(RDL_FILE)

.PHONY: logs
logs: _check-container _check-socket
	$(gorai) logs --config $(RDL_FILE) --follow

.PHONY: restart
restart: down up

# ============================================================================
# Build Targets
# ============================================================================

.PHONY: container
container:
	@echo "==> Building gorai CLI container..."
	cd $(GORAI_SRC) && podman build -t $(GORAI_IMAGE) -f Containerfile.gorai .
	@echo "==> Container built: $(GORAI_IMAGE)"

.PHONY: build
build: _check-container _check-socket
	@echo "==> Building robot containers..."
	$(gorai) build --config $(RDL_FILE)

# ============================================================================
# Utility Targets
# ============================================================================

.PHONY: validate
validate: _check-container _check-socket
	@echo "==> Validating $(RDL_FILE)..."
	$(gorai) validate $(ROBOT_NAME)

.PHONY: shell
shell: _check-container _check-socket
	@echo "==> Opening shell in gorai container..."
	podman run --rm -it \
		--security-opt label=disable \
		-v $(PODMAN_SOCK):/run/podman/podman.sock:rw \
		-v $(PWD):/workspace:rw \
		-w /workspace \
		--entrypoint /bin/bash \
		$(GORAI_IMAGE)

.PHONY: clean
clean:
	@echo "==> Cleaning up..."
	-$(gorai_batch) stop --config $(RDL_FILE) 2>/dev/null || true
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

.PHONY: _check-socket
_check-socket:
	@if [ ! -S "$(PODMAN_SOCK)" ]; then \
		echo "Error: Podman socket not found at $(PODMAN_SOCK)"; \
		echo "Enable it with: systemctl --user enable --now podman.socket"; \
		exit 1; \
	fi

# ============================================================================
# Legacy targets (for backwards compatibility during transition)
# ============================================================================

.PHONY: start stop
start: up
stop: down
