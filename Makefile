# Gorai Makefile

.DEFAULT_GOAL := help

GO := go
BIN_DIR := bin

.PHONY: help build test clean version install

help:
	@echo "Targets:"
	@echo "  build    Build the gorai CLI binary"
	@echo "  install  Install gorai to ~/bin"
	@echo "  test     Run all tests"
	@echo "  clean    Remove build artifacts"
	@echo "  version  Show version info for Go and tools"

build: $(BIN_DIR)
	@echo "==> Building all binaries..."
	$(GO) build -v -o $(BIN_DIR)/gorai ./cmd/gorai

test:
	@echo "==> Running tests..."
	$(GO) test ./...

install: build
	@mkdir -p $(HOME)/bin
	cp $(BIN_DIR)/gorai $(HOME)/bin/gorai
	@echo "==> Installed $(HOME)/bin/gorai"

clean:
	@echo "==> Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	$(GO) clean -cache -testcache
	@echo "==> Done"

version:
	@echo "Gorai Development Environment"
	@echo ""
	@echo "Go:       $$(go version)"
	@echo "GOOS:     $$(go env GOOS)"
	@echo "GOARCH:   $$(go env GOARCH)"

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)
