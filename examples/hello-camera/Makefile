# Makefile for hello-camera example
#
# This Makefile provides targets for initializing and generating code
# from the Robot Definition Language (RDL) file.

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
	@echo "hello-camera Example Makefile"
	@echo ""
	@echo "Configuration:"
	@echo "  ROBOT_NAME = $(ROBOT_NAME)"
	@echo "  RDL_FILE   = $(RDL_FILE)"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  help        Show this help message (default)"
	@echo "  init        Initialize project from $(RDL_FILE)"
	@echo "  generate    Generate code from $(RDL_FILE)"
	@echo "  validate    Validate $(RDL_FILE) configuration"
	@echo "  clean       Remove generated files"
	@echo "  all         Run validate, init, and generate"
	@echo ""
	@echo "Examples:"
	@echo "  make init       # Create project structure from RDL"
	@echo "  make generate   # Regenerate code after RDL changes"
	@echo "  make validate   # Check RDL file for errors"

.DEFAULT_GOAL := help

# ============================================================================
# RDL Targets
# ============================================================================

.PHONY: init
init:
	@echo "==> Initializing project from $(RDL_FILE)..."
	gorai init $(ROBOT_NAME)

.PHONY: generate
generate:
	@echo "==> Generating code from $(RDL_FILE)..."
	gorai generate $(ROBOT_NAME)

.PHONY: validate
validate:
	@echo "==> Validating $(RDL_FILE)..."
	gorai validate $(ROBOT_NAME)

.PHONY: validate-strict
validate-strict:
	@echo "==> Validating $(RDL_FILE) (strict mode)..."
	gorai validate $(ROBOT_NAME) --strict

# ============================================================================
# Convenience Targets
# ============================================================================

.PHONY: all
all: validate init generate
	@echo "==> All targets completed"

.PHONY: clean
clean:
	@echo "==> Cleaning generated files..."
	rm -rf $(ROBOT_NAME)/
	@echo "==> Done"
