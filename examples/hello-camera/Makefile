# Makefile for hello-camera example
#
# This Makefile provides targets for initializing, generating, building,
# and running a robot from the Robot Definition Language (RDL) file.

# ============================================================================
# Configuration
# ============================================================================

ROBOT_NAME := hello-camera
RDL_FILE := $(ROBOT_NAME).json
PROJECT_DIR := $(ROBOT_NAME)

# Path to local gorai module (relative to PROJECT_DIR)
GORAI_MODULE := github.com/gorai/gorai
GORAI_LOCAL_PATH := ../../..

# Build settings
BUILD_DIR := $(PROJECT_DIR)/build
BINARY := $(BUILD_DIR)/$(ROBOT_NAME)

# NATS settings
NATS_PID_FILE := /tmp/gorai-nats.pid

# Prometheus settings
PROMETHEUS_PID_FILE := /tmp/gorai-prometheus.pid
PROMETHEUS_CONFIG := prometheus.yml
PROMETHEUS_PORT := 9090

# ============================================================================
# Default target
# ============================================================================

.PHONY: help
help:
	@echo "hello-camera Example Makefile"
	@echo ""
	@echo "Configuration:"
	@echo "  ROBOT_NAME  = $(ROBOT_NAME)"
	@echo "  RDL_FILE    = $(RDL_FILE)"
	@echo "  PROJECT_DIR = $(PROJECT_DIR)"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Setup Targets:"
	@echo "  init            Initialize project from $(RDL_FILE)"
	@echo "  generate        Generate code from $(RDL_FILE)"
	@echo "  validate        Validate $(RDL_FILE) configuration"
	@echo "  tidy            Download dependencies (go mod tidy)"
	@echo ""
	@echo "Build Targets:"
	@echo "  build           Build the robot binary"
	@echo ""
	@echo "Run Targets:"
	@echo "  run             Build and run the robot (starts NATS and Prometheus)"
	@echo "  stop            Stop robot, NATS, and Prometheus"
	@echo "  nats-start      Start local NATS server"
	@echo "  nats-stop       Stop local NATS server"
	@echo "  prom-start      Start local Prometheus server"
	@echo "  prom-stop       Stop local Prometheus server"
	@echo "  services-start  Start all services (NATS + Prometheus)"
	@echo "  services-stop   Stop all services"
	@echo ""
	@echo "Utility Targets:"
	@echo "  clean           Remove generated project and build files"
	@echo "  all             Full setup: validate, init, generate, build"
	@echo ""
	@echo "Quick Start:"
	@echo "  make all        # Setup everything"
	@echo "  make run        # Run the robot"

.DEFAULT_GOAL := help

# ============================================================================
# Setup Targets
# ============================================================================

.PHONY: init
init:
	@echo "==> Initializing project from $(RDL_FILE)..."
	gorai init $(ROBOT_NAME)
	@echo "==> Adding local gorai replace directive..."
	@cd $(PROJECT_DIR) && go mod edit -replace $(GORAI_MODULE)=$(GORAI_LOCAL_PATH)

.PHONY: generate
generate:
	@echo "==> Generating code from $(RDL_FILE)..."
	@if [ -d "$(PROJECT_DIR)" ]; then \
		cd $(PROJECT_DIR) && gorai generate $(ROBOT_NAME); \
	else \
		gorai generate $(ROBOT_NAME); \
	fi

.PHONY: validate
validate:
	@echo "==> Validating $(RDL_FILE)..."
	gorai validate $(ROBOT_NAME)

.PHONY: validate-strict
validate-strict:
	@echo "==> Validating $(RDL_FILE) (strict mode)..."
	gorai validate $(ROBOT_NAME) --strict

.PHONY: tidy
tidy:
	@if [ ! -d "$(PROJECT_DIR)" ]; then \
		echo "ERROR: Project not initialized. Run 'make init' first."; \
		exit 1; \
	fi
	@echo "==> Ensuring local gorai replace directive..."
	@cd $(PROJECT_DIR) && go mod edit -replace $(GORAI_MODULE)=$(GORAI_LOCAL_PATH)
	@echo "==> Downloading dependencies..."
	cd $(PROJECT_DIR) && go mod tidy

# ============================================================================
# Build Targets
# ============================================================================

.PHONY: build
build: tidy
	@if [ ! -d "$(PROJECT_DIR)" ]; then \
		echo "ERROR: Project not initialized. Run 'make init' first."; \
		exit 1; \
	fi
	@echo "==> Building $(ROBOT_NAME)..."
	cd $(PROJECT_DIR) && go build -o build/$(ROBOT_NAME) .

# ============================================================================
# Run Targets
# ============================================================================

.PHONY: run
run: services-start build
	@echo "==> Running $(ROBOT_NAME)..."
	cd $(PROJECT_DIR) && ./build/$(ROBOT_NAME) --config $(ROBOT_NAME).json

.PHONY: services-start
services-start: nats-start prom-start
	@echo "==> All services started"

.PHONY: nats-start
nats-start:
	@if [ -f $(NATS_PID_FILE) ] && kill -0 $$(cat $(NATS_PID_FILE)) 2>/dev/null; then \
		echo "==> NATS server already running (PID: $$(cat $(NATS_PID_FILE)))"; \
	elif nc -z localhost 4222 2>/dev/null; then \
		echo "==> NATS already running on port 4222 (external instance)"; \
	else \
		if ! command -v nats-server >/dev/null 2>&1; then \
			echo "ERROR: nats-server not found. Install with:"; \
			echo "  go install github.com/nats-io/nats-server/v2@latest"; \
			echo "  # or: sudo apt install nats-server"; \
			exit 1; \
		fi; \
		echo "==> Starting NATS server..."; \
		nats-server -p 4222 -m 8222 & echo $$! > $(NATS_PID_FILE); \
		sleep 1; \
		if ! kill -0 $$(cat $(NATS_PID_FILE)) 2>/dev/null; then \
			echo "ERROR: NATS server failed to start (port 4222 may be in use)"; \
			echo "       Run: make stop  or  pkill nats-server"; \
			rm -f $(NATS_PID_FILE); \
			exit 1; \
		fi; \
		echo "    NATS server started (PID: $$(cat $(NATS_PID_FILE)))"; \
		echo "    Client port: 4222"; \
		echo "    Monitor:     http://localhost:8222"; \
	fi

.PHONY: nats-stop
nats-stop:
	@echo "==> Stopping NATS server..."
	@if [ -f $(NATS_PID_FILE) ]; then \
		kill $$(cat $(NATS_PID_FILE)) 2>/dev/null || true; \
		rm -f $(NATS_PID_FILE); \
		echo "    NATS server stopped"; \
	elif pkill -x nats-server 2>/dev/null; then \
		echo "    NATS server stopped (was running without PID file)"; \
	else \
		echo "    No NATS server running"; \
	fi

.PHONY: prom-start
prom-start:
	@if [ -f $(PROMETHEUS_PID_FILE) ] && kill -0 $$(cat $(PROMETHEUS_PID_FILE)) 2>/dev/null; then \
		echo "==> Prometheus already running (PID: $$(cat $(PROMETHEUS_PID_FILE)))"; \
	else \
		if ! command -v prometheus >/dev/null 2>&1; then \
			echo "==> Prometheus not found, skipping (install with: apt install prometheus)"; \
		else \
			echo "==> Starting Prometheus..."; \
			if [ ! -f $(PROMETHEUS_CONFIG) ]; then \
				echo "    Creating default $(PROMETHEUS_CONFIG)..."; \
				echo "global:" > $(PROMETHEUS_CONFIG); \
				echo "  scrape_interval: 15s" >> $(PROMETHEUS_CONFIG); \
				echo "" >> $(PROMETHEUS_CONFIG); \
				echo "scrape_configs:" >> $(PROMETHEUS_CONFIG); \
				echo "  - job_name: 'gorai'" >> $(PROMETHEUS_CONFIG); \
				echo "    static_configs:" >> $(PROMETHEUS_CONFIG); \
				echo "      - targets: ['localhost:2112']" >> $(PROMETHEUS_CONFIG); \
			fi; \
			prometheus --config.file=$(PROMETHEUS_CONFIG) --web.listen-address=:$(PROMETHEUS_PORT) \
				--storage.tsdb.path=/tmp/gorai-prometheus-data >/dev/null 2>&1 & \
			echo $$! > $(PROMETHEUS_PID_FILE); \
			sleep 1; \
			echo "    Prometheus started (PID: $$(cat $(PROMETHEUS_PID_FILE)))"; \
			echo "    Web UI:  http://localhost:$(PROMETHEUS_PORT)"; \
			echo "    Scraping: localhost:2112/metrics"; \
		fi; \
	fi

.PHONY: prom-stop
prom-stop:
	@echo "==> Stopping Prometheus..."
	@if [ -f $(PROMETHEUS_PID_FILE) ]; then \
		kill $$(cat $(PROMETHEUS_PID_FILE)) 2>/dev/null || true; \
		rm -f $(PROMETHEUS_PID_FILE); \
		echo "    Prometheus stopped"; \
	else \
		echo "    No Prometheus PID file found"; \
		pkill -x prometheus 2>/dev/null || true; \
	fi

.PHONY: services-stop
services-stop: nats-stop prom-stop
	@echo "==> All services stopped"

.PHONY: stop
stop: services-stop
	@echo "==> Stopping any running $(ROBOT_NAME) processes..."
	@pkill -x $(ROBOT_NAME) 2>/dev/null && echo "    $(ROBOT_NAME) stopped" || echo "    No $(ROBOT_NAME) process found"
	@echo "==> All stopped"

# ============================================================================
# Utility Targets
# ============================================================================

.PHONY: all
all: validate init build
	@echo "==> Setup complete!"
	@echo ""
	@echo "Next steps:"
	@echo "  make run        # Run the robot"

.PHONY: clean
clean: services-stop
	@echo "==> Cleaning generated files..."
	rm -rf $(PROJECT_DIR)/
	rm -rf internal/
	rm -f $(PROMETHEUS_CONFIG)
	rm -rf /tmp/gorai-prometheus-data
	@echo "==> Done"
