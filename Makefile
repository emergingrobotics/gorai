# Gorai Makefile
#
# Common development tasks for the Gorai robotics framework.
#
# Usage:
#   make              - Show help
#   make help         - Show help
#   make test         - Run unit tests
#   make test-all     - Run all tests (unit, component, integration, module, system)
#   make build        - Build all binaries
#   make proto        - Generate Protocol Buffer code
#   make lint         - Run linters
#   make clean        - Clean build artifacts

# Default target - print all available targets
.PHONY: help
help:
	@echo "Gorai Development Makefile"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Common workflows:"
	@echo "  make build            Build the CLI"
	@echo "  make build-examples   Build all example binaries"
	@echo "  make validate-examples Validate all RDL example configs"
	@echo "  make test             Run unit tests"
	@echo "  make check            Run all checks (fmt, vet, lint, test)"
	@echo ""
	@echo "Example targets:"
	@echo "  make run-example-blinky        Run blinky LED example"
	@echo "  make run-example-gps           Run GPS tracker example"
	@echo "  make run-example-camera        Run camera streaming example"
	@echo "  make run-example-pwm-controller Run PWM controller (gorai-gsp)"
	@echo ""

# Default target
.DEFAULT_GOAL := help

# ============================================================================
# Variables
# ============================================================================

# Go settings
GO := go
GOFLAGS := -v
GOTEST := $(GO) test
GOBUILD := $(GO) build

# Build output directory
BUILD_DIR := build
BIN_DIR := bin

# Coverage output
COVERAGE_DIR := $(BUILD_DIR)/coverage
COVERAGE_FILE := $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML := $(COVERAGE_DIR)/coverage.html

# Platforms
LINUX_ARM64 := GOOS=linux GOARCH=arm64
LINUX_AMD64 := GOOS=linux GOARCH=amd64
DARWIN_ARM64 := GOOS=darwin GOARCH=arm64
DARWIN_AMD64 := GOOS=darwin GOARCH=amd64

# Test tags
COMPONENT_TAGS := -tags=component
INTEGRATION_TAGS := -tags=integration
MODULE_TAGS := -tags=module
SYSTEM_TAGS := -tags=system
HARDWARE_TAGS := -tags=hardware
ALL_TAGS := -tags=component,integration,module,system

# Timeouts
TEST_TIMEOUT := 2m
INTEGRATION_TIMEOUT := 5m
MODULE_TIMEOUT := 5m
SYSTEM_TIMEOUT := 10m

# NATS
NATS_PID_FILE := /tmp/gorai-nats.pid

# ============================================================================
# Testing
# ============================================================================

.PHONY: test
test: ## Run unit tests (fast)
	@echo "==> Running unit tests..."
	$(GOTEST) ./...

.PHONY: test-quick
test-quick: ## Run unit + component tests
	@echo "==> Running unit + component tests..."
	$(GOTEST) $(COMPONENT_TAGS) ./...

.PHONY: test-prepush
test-prepush:
	@echo "==> Running pre-push tests (unit + component + integration with race)..."
	$(GOTEST) -race $(COMPONENT_TAGS) ./...
	$(GOTEST) -race $(INTEGRATION_TAGS) -timeout=$(INTEGRATION_TIMEOUT) ./tests/integration/...

.PHONY: test-all
test-all: ## Run all tests (unit, component, integration, module, system)
	@echo "==> Running all tests..."
	$(GOTEST) $(ALL_TAGS) -timeout=$(SYSTEM_TIMEOUT) ./...

.PHONY: test-component
test-component:
	@echo "==> Running component tests..."
	$(GOTEST) $(COMPONENT_TAGS) ./components/...

.PHONY: test-integration
test-integration:
	@echo "==> Running integration tests..."
	@echo "    NOTE: tests/integration/ directory does not exist yet"

.PHONY: test-module
test-module:
	@echo "==> Running module tests..."
	@echo "    NOTE: tests/module/ directory does not exist yet"

.PHONY: test-system
test-system:
	@echo "==> Running system tests..."
	@echo "    NOTE: tests/system/ directory does not exist yet"

.PHONY: test-hardware
test-hardware:
	@echo "==> Running hardware tests..."
	@echo "    NOTE: Requires appropriate hardware connected"
	$(GOTEST) $(HARDWARE_TAGS) ./driver/...

.PHONY: test-cover
test-cover: $(COVERAGE_DIR) ## Run tests with coverage report
	@echo "==> Running tests with coverage..."
	$(GOTEST) $(COMPONENT_TAGS) -coverprofile=$(COVERAGE_FILE) ./...
	$(GO) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	$(GO) tool cover -func=$(COVERAGE_FILE) | tail -1
	@echo "==> Coverage report: $(COVERAGE_HTML)"

.PHONY: test-race
test-race:
	@echo "==> Running tests with race detector..."
	$(GOTEST) -race ./...

.PHONY: test-verbose
test-verbose:
	@echo "==> Running tests with verbose output..."
	$(GOTEST) -v ./...

.PHONY: test-short
test-short:
	@echo "==> Running short tests..."
	$(GOTEST) -short ./...

$(COVERAGE_DIR):
	@mkdir -p $(COVERAGE_DIR)

# ============================================================================
# Building
# ============================================================================

.PHONY: build
build: $(BIN_DIR) ## Build all binaries
	@echo "==> Building all binaries..."
	$(GOBUILD) $(GOFLAGS) -o $(BIN_DIR)/gorai ./cmd/gorai

.PHONY: build-linux
build-linux: $(BIN_DIR)
	@echo "==> Cross-compiling for Linux ARM64..."
	$(LINUX_ARM64) $(GOBUILD) $(GOFLAGS) -o $(BIN_DIR)/gorai-linux-arm64 ./cmd/gorai
	$(LINUX_AMD64) $(GOBUILD) $(GOFLAGS) -o $(BIN_DIR)/gorai-linux-amd64 ./cmd/gorai

.PHONY: build-pi
build-pi: $(BIN_DIR) ## Cross-compile for Raspberry Pi
	@echo "==> Cross-compiling for Raspberry Pi (Linux ARM64)..."
	$(LINUX_ARM64) $(GOBUILD) $(GOFLAGS) -o $(BIN_DIR)/gorai-pi ./cmd/gorai
	@echo "==> Binary ready: $(BIN_DIR)/gorai-pi"
	@echo "    Copy to Pi with: scp $(BIN_DIR)/gorai-pi pi@<hostname>:~/"

.PHONY: build-all-platforms
build-all-platforms: $(BIN_DIR)
	@echo "==> Building for all platforms..."
	$(LINUX_ARM64) $(GOBUILD) -o $(BIN_DIR)/gorai-linux-arm64 ./cmd/gorai
	$(LINUX_AMD64) $(GOBUILD) -o $(BIN_DIR)/gorai-linux-amd64 ./cmd/gorai
	$(DARWIN_ARM64) $(GOBUILD) -o $(BIN_DIR)/gorai-darwin-arm64 ./cmd/gorai
	$(DARWIN_AMD64) $(GOBUILD) -o $(BIN_DIR)/gorai-darwin-amd64 ./cmd/gorai
	@echo "==> All platform binaries in $(BIN_DIR)/"

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

# ============================================================================
# Installation
# ============================================================================

INSTALL_DIR := /usr/local/bin

.PHONY: install
install: ## Install binaries to /usr/local/bin (requires sudo)
	@if [ ! -f $(BIN_DIR)/gorai ]; then \
		echo "ERROR: Binary not found. Run 'make build' first, then 'sudo make install'."; \
		exit 1; \
	fi
	@echo "==> Installing binaries to $(INSTALL_DIR)..."
	@echo "    (requires sudo)"
	install -m 755 $(BIN_DIR)/gorai $(INSTALL_DIR)/gorai
	@echo "==> Installed: $(INSTALL_DIR)/gorai"

# ============================================================================
# Protocol Buffers
# ============================================================================

.PHONY: proto
proto: proto-lint
	@echo "==> Generating Protocol Buffer code..."
	@if command -v buf >/dev/null 2>&1; then \
		cd api && buf generate; \
	else \
		echo "ERROR: buf is not installed. Run 'make dev-deps' first."; \
		exit 1; \
	fi
	@echo "==> Protocol Buffer code generated in api/gen/"

.PHONY: proto-lint
proto-lint:
	@echo "==> Linting Protocol Buffer files..."
	@if command -v buf >/dev/null 2>&1; then \
		cd api && buf lint; \
	else \
		echo "ERROR: buf is not installed. Run 'make dev-deps' first."; \
		exit 1; \
	fi

.PHONY: proto-clean
proto-clean:
	@echo "==> Cleaning generated Protocol Buffer code..."
	rm -rf api/gen

.PHONY: proto-breaking
proto-breaking:
	@echo "==> Checking for breaking Protocol Buffer changes..."
	@if command -v buf >/dev/null 2>&1; then \
		cd api && buf breaking --against '.git#branch=main'; \
	else \
		echo "ERROR: buf is not installed. Run 'make dev-deps' first."; \
		exit 1; \
	fi

# ============================================================================
# Code Quality
# ============================================================================

.PHONY: lint
lint: vet ## Run all linters
	@echo "==> Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "WARNING: golangci-lint not installed. Run 'make dev-deps' first."; \
		echo "         Falling back to go vet only."; \
	fi

.PHONY: fmt
fmt: ## Format all Go code
	@echo "==> Formatting Go code..."
	$(GO) fmt ./...
	@echo "==> Done"

.PHONY: fmt-check
fmt-check:
	@echo "==> Checking Go code formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "ERROR: The following files are not formatted:"; \
		gofmt -l .; \
		echo "Run 'make fmt' to fix."; \
		exit 1; \
	fi
	@echo "==> All files are formatted correctly"

.PHONY: vet
vet: ## Run go vet
	@echo "==> Running go vet..."
	$(GO) vet ./...

.PHONY: tidy
tidy: ## Run go mod tidy
	@echo "==> Running go mod tidy..."
	$(GO) mod tidy
	@echo "==> Done"

.PHONY: check
check: fmt-check vet lint test ## Run all checks (fmt, vet, lint, test)
	@echo "==> All checks passed"

# ============================================================================
# Development
# ============================================================================

.PHONY: dev-deps
dev-deps: ## Install development dependencies
	@echo "==> Installing development dependencies..."
	@echo ""
	@echo "Installing buf (Protocol Buffers)..."
	go install github.com/bufbuild/buf/cmd/buf@latest
	@echo ""
	@echo "Installing golangci-lint..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo ""
	@echo "Installing NATS server..."
	go install github.com/nats-io/nats-server/v2@latest
	@echo ""
	@echo "Installing NATS CLI..."
	go install github.com/nats-io/natscli/nats@latest
	@echo ""
	@echo "Installing air (hot reload)..."
	go install github.com/air-verse/air@latest
	@echo ""
	@echo "==> Development dependencies installed"
	@echo ""
	@echo "Verify installation:"
	@echo "  buf --version"
	@echo "  golangci-lint --version"
	@echo "  nats-server --version"
	@echo "  nats --version"

.PHONY: nats-start
nats-start: ## Start local NATS server
	@echo "==> Starting NATS server..."
	@if [ -f $(NATS_PID_FILE) ] && kill -0 $$(cat $(NATS_PID_FILE)) 2>/dev/null; then \
		echo "NATS server already running (PID: $$(cat $(NATS_PID_FILE)))"; \
	else \
		nats-server -p 4222 -m 8222 & echo $$! > $(NATS_PID_FILE); \
		sleep 1; \
		echo "NATS server started (PID: $$(cat $(NATS_PID_FILE)))"; \
		echo "  Client port: 4222"; \
		echo "  Monitor:     http://localhost:8222"; \
	fi

.PHONY: nats-stop
nats-stop: ## Stop local NATS server
	@echo "==> Stopping NATS server..."
	@if [ -f $(NATS_PID_FILE) ]; then \
		kill $$(cat $(NATS_PID_FILE)) 2>/dev/null || true; \
		rm -f $(NATS_PID_FILE); \
		echo "NATS server stopped"; \
	else \
		echo "No NATS PID file found"; \
		pkill -x nats-server 2>/dev/null || true; \
	fi

.PHONY: nats-status
nats-status:
	@if [ -f $(NATS_PID_FILE) ] && kill -0 $$(cat $(NATS_PID_FILE)) 2>/dev/null; then \
		echo "NATS server is running (PID: $$(cat $(NATS_PID_FILE)))"; \
		curl -s http://localhost:8222/varz | head -5 || true; \
	else \
		echo "NATS server is not running"; \
	fi

.PHONY: watch
watch:
	@echo "==> Starting watch mode (runs tests on file changes)..."
	@if command -v air >/dev/null 2>&1; then \
		air -c .air.toml 2>/dev/null || \
		(echo "No .air.toml found, using default settings..." && \
		 find . -name "*.go" | grep -v vendor | head -1 >/dev/null && \
		 echo "Watching Go files... Press Ctrl+C to stop" && \
		 while true; do \
			find . -name "*.go" -newer /tmp/.gorai-watch-marker 2>/dev/null | head -1 | \
			while read f; do \
				echo "==> File changed: $$f"; \
				$(GOTEST) ./... || true; \
				touch /tmp/.gorai-watch-marker; \
			done; \
			sleep 1; \
		 done); \
	else \
		echo "Installing air for watch mode..."; \
		go install github.com/air-verse/air@latest; \
		echo "Run 'make watch' again"; \
	fi

.PHONY: watch-test
watch-test:
	@echo "==> Watching for changes and running tests..."
	@touch /tmp/.gorai-watch-marker
	@while true; do \
		find . -name "*.go" -newer /tmp/.gorai-watch-marker 2>/dev/null | head -1 | \
		while read f; do \
			clear; \
			echo "==> File changed, running tests..."; \
			$(GOTEST) ./... || true; \
			touch /tmp/.gorai-watch-marker; \
			echo ""; \
			echo "Waiting for changes..."; \
		done; \
		sleep 1; \
	done

# ============================================================================
# Cleanup
# ============================================================================

.PHONY: clean
clean: clean-coverage proto-clean ## Clean all build artifacts
	@echo "==> Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf $(BIN_DIR)
	rm -f /tmp/.gorai-watch-marker
	$(GO) clean -cache -testcache
	@echo "==> Done"

.PHONY: clean-coverage
clean-coverage:
	@echo "==> Cleaning coverage files..."
	rm -rf $(COVERAGE_DIR)

.PHONY: clean-all
clean-all: clean
	@echo "==> Deep cleaning (including module cache)..."
	$(GO) clean -modcache
	@echo "==> Done"

# ============================================================================
# Documentation
# ============================================================================

.PHONY: docs
docs:
	@echo "==> Generating documentation..."
	@echo "    View docs at: https://pkg.go.dev/github.com/gorai/gorai"
	$(GO) doc -all ./... > $(BUILD_DIR)/docs.txt 2>/dev/null || true
	@echo "==> Local docs written to $(BUILD_DIR)/docs.txt"

.PHONY: docs-serve
docs-serve:
	@echo "==> Starting local documentation server..."
	@if command -v pkgsite >/dev/null 2>&1; then \
		echo "View at: http://localhost:8080/github.com/gorai/gorai"; \
		pkgsite -http=:8080; \
	else \
		echo "Installing pkgsite..."; \
		go install golang.org/x/pkgsite/cmd/pkgsite@latest; \
		echo "Run 'make docs-serve' again"; \
	fi

# ============================================================================
# Examples
# ============================================================================

# Example directories
EXAMPLES_DIR := examples

# RDL-based examples (run via gorai CLI)
EXAMPLE_BLINKY := $(EXAMPLES_DIR)/blinky
EXAMPLE_GPS := $(EXAMPLES_DIR)/gps-tracker
EXAMPLE_CAMERA := $(EXAMPLES_DIR)/hello-camera

# Go examples in examples/
EXAMPLE_PWM_CONTROLLER := $(EXAMPLES_DIR)/pwm-controller

# --- Build All Examples ---

.PHONY: build-examples
build-examples: build-example-pwm-controller ## Build all example binaries
	@echo "==> All examples built"

.PHONY: build-example-pwm-controller
build-example-pwm-controller: $(BIN_DIR) ## Build pwm-controller example (uses gorai-gsp)
	@echo "==> Building pwm-controller..."
	cd $(EXAMPLE_PWM_CONTROLLER) && $(GOBUILD) $(GOFLAGS) -o $(CURDIR)/$(BIN_DIR)/pwm-controller .
	@echo "==> Binary ready: $(BIN_DIR)/pwm-controller"

# --- Validate Examples ---

.PHONY: validate-examples
validate-examples: build ## Validate all RDL example configurations
	@echo "==> Validating all example configurations..."
	$(BIN_DIR)/gorai validate $(EXAMPLE_BLINKY)/robot.json
	$(BIN_DIR)/gorai validate $(EXAMPLE_GPS)/robot.json
	$(BIN_DIR)/gorai validate $(EXAMPLE_CAMERA)/hello-camera.json
	@echo "==> All examples validated successfully"

.PHONY: validate-example-blinky
validate-example-blinky: build ## Validate blinky example
	$(BIN_DIR)/gorai validate $(EXAMPLE_BLINKY)/robot.json

.PHONY: validate-example-gps
validate-example-gps: build ## Validate gps-tracker example
	$(BIN_DIR)/gorai validate $(EXAMPLE_GPS)/robot.json

.PHONY: validate-example-camera
validate-example-camera: build ## Validate hello-camera example
	$(BIN_DIR)/gorai validate $(EXAMPLE_CAMERA)/hello-camera.json

# --- Run Examples ---

.PHONY: run-example-blinky
run-example-blinky: build nats-start ## Run blinky example
	@echo "==> Running blinky example..."
	@echo "    Subscribe with: nats sub 'gorai.blinky.>'"
	@echo ""
	$(BIN_DIR)/gorai run $(EXAMPLE_BLINKY)/robot.json

.PHONY: run-example-gps
run-example-gps: build nats-start ## Run gps-tracker example
	@echo "==> Running gps-tracker example..."
	@echo "    Subscribe with: nats sub 'gorai.gps-tracker.>'"
	@echo ""
	$(BIN_DIR)/gorai run $(EXAMPLE_GPS)/robot.json

.PHONY: run-example-camera
run-example-camera: build nats-start ## Run hello-camera example
	@echo "==> Running hello-camera example..."
	@echo "    Dashboard: http://localhost:8080"
	@echo ""
	$(BIN_DIR)/gorai run $(EXAMPLE_CAMERA)/hello-camera.json

.PHONY: run-example-pwm-controller
run-example-pwm-controller: build-example-pwm-controller ## Run pwm-controller example (sweep mode)
	@echo "==> Running pwm-controller example..."
	@echo "    Connect RP2040 PWM controller to /dev/ttyACM0"
	@echo ""
	$(BIN_DIR)/pwm-controller

# --- Subscribe to Example Topics ---

.PHONY: sub-example-blinky
sub-example-blinky: ## Subscribe to blinky topics
	@echo "==> Subscribing to blinky topics..."
	nats sub "gorai.blinky.>"

.PHONY: sub-example-gps
sub-example-gps: ## Subscribe to gps-tracker topics
	@echo "==> Subscribing to gps-tracker topics..."
	nats sub "gorai.gps-tracker.>"


# ============================================================================
# CI/CD
# ============================================================================

.PHONY: ci
ci: fmt-check vet test-all
	@echo "==> CI checks passed"

.PHONY: ci-quick
ci-quick: fmt-check vet test
	@echo "==> Quick CI checks passed"

# ============================================================================
# Git Hooks
# ============================================================================

.PHONY: hooks-install
hooks-install:
	@echo "==> Installing git hooks..."
	@mkdir -p .git/hooks
	@echo '#!/bin/bash' > .git/hooks/pre-commit
	@echo 'set -e' >> .git/hooks/pre-commit
	@echo 'make fmt-check' >> .git/hooks/pre-commit
	@echo 'make vet' >> .git/hooks/pre-commit
	@echo 'make test-short' >> .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo '#!/bin/bash' > .git/hooks/pre-push
	@echo 'set -e' >> .git/hooks/pre-push
	@echo 'make test-prepush' >> .git/hooks/pre-push
	@chmod +x .git/hooks/pre-push
	@echo "==> Git hooks installed"
	@echo "    pre-commit: fmt-check, vet, test-short"
	@echo "    pre-push:   test-prepush"

.PHONY: hooks-uninstall
hooks-uninstall:
	@echo "==> Removing git hooks..."
	rm -f .git/hooks/pre-commit
	rm -f .git/hooks/pre-push
	@echo "==> Git hooks removed"

# ============================================================================
# Tools
# ============================================================================

# Serial port for PWM controller (default: /dev/ttyACM0)
PWM_PORT ?= /dev/ttyACM0

.PHONY: build-tools
build-tools: build-pwm-ramp-test ## Build all tools

.PHONY: build-pwm-ramp-test
build-pwm-ramp-test: $(BIN_DIR) ## Build pwm-ramp-test tool
	@echo "==> Building pwm-ramp-test..."
	cd tools/pwm-ramp-test && $(GOBUILD) -o ../../$(BIN_DIR)/pwm-ramp-test .
	@echo "==> Binary ready: $(BIN_DIR)/pwm-ramp-test"

.PHONY: pwm-ramp-test
pwm-ramp-test: build-pwm-ramp-test ## Run PWM ramp test (use PWM_PORT to set serial port)
	@echo "==> Running PWM ramp test on $(PWM_PORT)..."
	$(BIN_DIR)/pwm-ramp-test $(PWM_PORT)

# ============================================================================
# TinyGo
# ============================================================================

.PHONY: tinygo-check
tinygo-check:
	@echo "==> Checking TinyGo compilation..."
	@if command -v tinygo >/dev/null 2>&1; then \
		echo "TinyGo version: $$(tinygo version)"; \
		echo ""; \
		echo "Checking ESP32 target..."; \
		tinygo build -target=esp32 -o /dev/null ./... 2>&1 | head -5 || true; \
		echo ""; \
		echo "Checking Pico target..."; \
		tinygo build -target=pico -o /dev/null ./... 2>&1 | head -5 || true; \
	else \
		echo "TinyGo is not installed."; \
		echo "Install from: https://tinygo.org/getting-started/install/"; \
	fi

# ============================================================================
# Version
# ============================================================================

.PHONY: version
version: ## Show version info for Go and tools
	@echo "Gorai Development Environment"
	@echo ""
	@echo "Go:       $$(go version)"
	@echo "GOOS:     $$(go env GOOS)"
	@echo "GOARCH:   $$(go env GOARCH)"
	@echo ""
	@if command -v buf >/dev/null 2>&1; then echo "buf:      $$(buf --version)"; fi
	@if command -v golangci-lint >/dev/null 2>&1; then echo "lint:     $$(golangci-lint --version 2>&1 | head -1)"; fi
	@if command -v nats-server >/dev/null 2>&1; then echo "nats:     $$(nats-server --version)"; fi
	@if command -v tinygo >/dev/null 2>&1; then echo "tinygo:   $$(tinygo version)"; fi

# =============================================================================
# Publishing
# =============================================================================

PUBLISH_IMAGE := gorai-publish
PUBLISH_DIR := publish

.PHONY: publish-container
publish-container: ## Build the publishing container
	podman build -t $(PUBLISH_IMAGE) $(PUBLISH_DIR)/container/

.PHONY: publish-all
publish-all: publish-container ## Build all documentation
	podman run --rm -v $${PWD}:/workspace:Z $(PUBLISH_IMAGE) all

.PHONY: publish-book
publish-book: publish-container ## Build the book
	podman run --rm -v $${PWD}:/workspace:Z $(PUBLISH_IMAGE) book

.PHONY: publish-website
publish-website: publish-container ## Build the website
	podman run --rm -v $${PWD}:/workspace:Z $(PUBLISH_IMAGE) website

.PHONY: serve-book
serve-book: publish-container ## Serve book with live reload (port 3000)
	podman run --rm -it -p 3000:3000 -v $${PWD}:/workspace:Z $(PUBLISH_IMAGE) book-serve

.PHONY: serve-website
serve-website: publish-container ## Serve website with live reload (port 8000)
	podman run --rm -it -p 8000:8000 -v $${PWD}:/workspace:Z $(PUBLISH_IMAGE) website-serve

.PHONY: serve-api
serve-api: publish-container ## Serve API reference (port 6060)
	podman run --rm -it -p 6060:6060 -v $${PWD}:/workspace:Z $(PUBLISH_IMAGE) api-serve

.PHONY: publish-clean
publish-clean: ## Clean build outputs
	rm -rf $(PUBLISH_DIR)/dist
