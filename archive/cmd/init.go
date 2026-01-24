package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gorai/gorai/pkg/config"
)

// cmdInit handles the 'gorai init' command.
func cmdInit() error {
	args := os.Args[2:]

	// Parse flags
	var robotName string
	var template = "standard"
	var noGit, noGenerate bool

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return printInitUsage()
		case arg == "--template" && i+1 < len(args):
			template = args[i+1]
			i++
		case strings.HasPrefix(arg, "--template="):
			template = strings.TrimPrefix(arg, "--template=")
		case arg == "--no-git":
			noGit = true
		case arg == "--no-generate":
			noGenerate = true
		case !strings.HasPrefix(arg, "-"):
			if robotName == "" {
				robotName = arg
			}
		default:
			return fmt.Errorf("unknown flag: %s", arg)
		}
	}

	// Get robot name from args or environment
	if robotName == "" {
		robotName = os.Getenv("GORAI_ROBOT_NAME")
	}

	if robotName == "" {
		return fmt.Errorf("robot name required\n\nUsage: gorai init <robot-name> [flags]\n\nOr set GORAI_ROBOT_NAME environment variable")
	}

	// Validate robot name
	if err := validateProjectName(robotName); err != nil {
		return fmt.Errorf("invalid robot name: %w", err)
	}

	// Check if {robot-name}.json exists
	configFile := robotName + ".json"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("%s not found\n\nCreate %s first with your robot configuration, then run:\n  gorai init %s", configFile, configFile, robotName)
	}

	// Load and validate the configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load %s: %w", configFile, err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration in %s: %w", configFile, err)
	}

	// Use robot name from config if it differs (warn user)
	if cfg.Robot.Name != robotName {
		fmt.Printf("Warning: robot.name in %s is %q, but you specified %q\n", configFile, cfg.Robot.Name, robotName)
		fmt.Printf("Using %q from the config file\n\n", cfg.Robot.Name)
		robotName = cfg.Robot.Name
	}

	// Create project directory
	projectDir := robotName
	if _, err := os.Stat(projectDir); err == nil {
		return fmt.Errorf("directory %q already exists", projectDir)
	}

	fmt.Printf("Initializing project: %s (from %s)\n", robotName, configFile)

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Generate project files
	if err := generateProjectFilesFromConfig(projectDir, robotName, configFile, template); err != nil {
		os.RemoveAll(projectDir)
		return fmt.Errorf("failed to generate project files: %w", err)
	}

	// Initialize git repository
	if !noGit {
		if err := initGitRepo(projectDir); err != nil {
			fmt.Printf("  ! Warning: failed to initialize git: %v\n", err)
		} else {
			fmt.Println("  + Initialized git repository")
		}
	}

	// Run gorai generate
	if !noGenerate {
		fmt.Println("  + Running gorai generate...")
		if err := runGenerateInProject(projectDir, robotName); err != nil {
			fmt.Printf("  ! Warning: gorai generate failed: %v\n", err)
		}
	}

	// Print next steps
	fmt.Printf(`
Project initialized! Next steps:
  cd %s
  vim %s.json         # Edit your robot configuration
  gorai generate %s   # Regenerate after changes
  make build            # Build for local testing
  make build-pi         # Build for Raspberry Pi
`, robotName, robotName, robotName)

	return nil
}

func printInitUsage() error {
	fmt.Println(`gorai init - Initialize a robot project from an existing RDL file

Usage:
  gorai init <robot-name> [flags]

The command expects <robot-name>.json to exist in the current directory.

Flags:
  --template <name>    Use a specific template (default: "standard")
  --no-git             Don't initialize git repository
  --no-generate        Don't run gorai generate after init
  -h, --help           Show this help

Environment:
  GORAI_ROBOT_NAME     Default robot name if not specified

Templates:
  standard             Standard project structure
  minimal              Bare minimum files

Example:
  # Create my-robot.json first, then:
  gorai init my-robot

  # Or with environment variable:
  export GORAI_ROBOT_NAME=my-robot
  gorai init`)
	return nil
}

func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if len(name) > 63 {
		return fmt.Errorf("name too long (max 63 characters)")
	}
	// Must start with a letter
	if name[0] < 'a' || name[0] > 'z' {
		if name[0] < 'A' || name[0] > 'Z' {
			return fmt.Errorf("name must start with a letter")
		}
	}
	// Only letters, numbers, hyphens, underscores
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_') {
			return fmt.Errorf("name may only contain letters, numbers, hyphens, and underscores")
		}
	}
	return nil
}

func generateProjectFilesFromConfig(projectDir, robotName, configFile, template string) error {
	// Create directories
	dirs := []string{
		"components",
		"services",
		"deploy",
		"scripts",
		"internal/generated",
		".github/workflows",
	}
	for _, dir := range dirs {
		path := filepath.Join(projectDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		fmt.Printf("  + Created %s/\n", dir)
	}

	// Create .gitkeep files
	gitkeeps := []string{
		"components/.gitkeep",
		"services/.gitkeep",
	}
	for _, gk := range gitkeeps {
		path := filepath.Join(projectDir, gk)
		if err := os.WriteFile(path, []byte(""), 0644); err != nil {
			return fmt.Errorf("failed to create %s: %w", gk, err)
		}
	}

	// Copy the config file into the project
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", configFile, err)
	}
	destConfig := filepath.Join(projectDir, robotName+".json")
	if err := os.WriteFile(destConfig, configData, 0644); err != nil {
		return fmt.Errorf("failed to copy config: %w", err)
	}
	fmt.Printf("  + Copied %s\n", robotName+".json")

	// Create main.go
	mainGo := generateMainGo(robotName)
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(mainGo), 0644); err != nil {
		return fmt.Errorf("failed to create main.go: %w", err)
	}
	fmt.Println("  + Created main.go")

	// Create go.mod
	goMod := generateGoMod(robotName)
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goMod), 0644); err != nil {
		return fmt.Errorf("failed to create go.mod: %w", err)
	}
	fmt.Println("  + Created go.mod")

	// Create Makefile
	makefile := generateMakefile(robotName)
	if err := os.WriteFile(filepath.Join(projectDir, "Makefile"), []byte(makefile), 0644); err != nil {
		return fmt.Errorf("failed to create Makefile: %w", err)
	}
	fmt.Println("  + Created Makefile")

	// Create README.md
	readme := generateReadme(robotName)
	if err := os.WriteFile(filepath.Join(projectDir, "README.md"), []byte(readme), 0644); err != nil {
		return fmt.Errorf("failed to create README.md: %w", err)
	}
	fmt.Println("  + Created README.md")

	// Create .gitignore
	gitignore := generateGitignore()
	if err := os.WriteFile(filepath.Join(projectDir, ".gitignore"), []byte(gitignore), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}
	fmt.Println("  + Created .gitignore")

	// Create systemd service file
	service := generateSystemdService(robotName)
	if err := os.WriteFile(filepath.Join(projectDir, "deploy", robotName+".service"), []byte(service), 0644); err != nil {
		return fmt.Errorf("failed to create systemd service: %w", err)
	}
	fmt.Printf("  + Created deploy/%s.service\n", robotName)

	// Create GitHub Actions workflow
	workflow := generateGitHubWorkflow(robotName)
	if err := os.WriteFile(filepath.Join(projectDir, ".github/workflows/build.yml"), []byte(workflow), 0644); err != nil {
		return fmt.Errorf("failed to create GitHub workflow: %w", err)
	}
	fmt.Println("  + Created .github/workflows/build.yml")

	// Create internal/generated/doc.go
	docGo := generateDocGo()
	if err := os.WriteFile(filepath.Join(projectDir, "internal/generated/doc.go"), []byte(docGo), 0644); err != nil {
		return fmt.Errorf("failed to create internal/generated/doc.go: %w", err)
	}
	fmt.Println("  + Created internal/generated/doc.go")

	return nil
}

func generateMainGo(robotName string) string {
	configFile := robotName + ".json"
	return fmt.Sprintf(`// main.go
// Generated by gorai init. You may edit this file.

package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorai/gorai/pkg/config"
	"github.com/gorai/gorai/pkg/robot"

	// Generated imports - do not remove this import
	_ "%s/internal/generated"

	// Custom components - add your component imports here
	// _ "%s/components/my_sensor"
)

func main() {
	configPath := flag.String("config", "%s", "Path to robot configuration")
	flag.Parse()

	// Setup logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("Failed to load config", "error", err, "path", *configPath)
		os.Exit(1)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		slog.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create robot
	r, err := robot.New(ctx, cfg, robot.WithLogger(logger))
	if err != nil {
		slog.Error("Failed to create robot", "error", err)
		os.Exit(1)
	}

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	go func() {
		for sig := range sigCh {
			switch sig {
			case syscall.SIGTERM, syscall.SIGINT:
				slog.Info("Shutdown signal received", "signal", sig)
				cancel()
			case syscall.SIGHUP:
				slog.Info("Reload signal received")
				newCfg, err := config.Load(*configPath)
				if err != nil {
					slog.Error("Failed to reload config", "error", err)
					continue
				}
				if err := r.Reconfigure(ctx, newCfg); err != nil {
					slog.Error("Failed to reconfigure", "error", err)
				}
			}
		}
	}()

	// Start robot
	if err := r.Start(ctx); err != nil {
		slog.Error("Failed to start robot", "error", err)
		os.Exit(1)
	}

	slog.Info("Robot started", "name", cfg.Robot.Name)

	// Run until shutdown
	if err := r.Run(ctx); err != nil && err != context.Canceled {
		slog.Error("Robot error", "error", err)
	}

	// Graceful shutdown
	slog.Info("Shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := r.Stop(shutdownCtx); err != nil {
		slog.Error("Shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("Robot stopped")
}
`, robotName, robotName, configFile)
}

func generateGoMod(robotName string) string {
	return fmt.Sprintf(`module %s

go 1.22

require github.com/gorai/gorai v0.1.0
`, robotName)
}

func generateMakefile(robotName string) string {
	return fmt.Sprintf(`# Makefile
# Generated by gorai init. You may edit this file.

# ============================================
# Project Configuration
# ============================================

ROBOT_NAME := %s
BINARY_NAME := $(ROBOT_NAME)
CONFIG_FILE := $(ROBOT_NAME).json
BUILD_TARGET ?= linux-arm64

# Deployment (override with environment or command line)
ROBOT_HOST ?= pi@$(ROBOT_NAME).local
ROBOT_PATH ?= /opt/$(BINARY_NAME)

# ============================================
# Go Configuration
# ============================================

GOFLAGS := -trimpath
LDFLAGS := -s -w

# Version info (set by CI or manually)
VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +%%Y-%%m-%%dT%%H:%%M:%%SZ)

LDFLAGS += -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# ============================================
# Targets
# ============================================

.PHONY: all build build-pi generate test lint clean deploy

all: generate build

# Generate code from robot config
generate:
	gorai generate $(ROBOT_NAME)

# Build for current platform
build: generate
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o build/$(BINARY_NAME) .

# Build for Raspberry Pi (ARM64)
build-pi: generate
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
		-o build/linux-arm64/$(BINARY_NAME) .

# Build for all platforms
build-all: generate
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o build/linux-arm64/$(BINARY_NAME) .
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o build/linux-amd64/$(BINARY_NAME) .

# Run tests
test:
	go test ./...

# Run linter
lint:
	golangci-lint run

# Validate robot config
validate:
	gorai validate $(ROBOT_NAME)

# Clean build artifacts
clean:
	rm -rf build/

# ============================================
# Deployment
# ============================================

# Deploy to robot
deploy: build-pi
	@echo "Deploying to $(ROBOT_HOST):$(ROBOT_PATH)"
	ssh $(ROBOT_HOST) "sudo mkdir -p $(ROBOT_PATH) && sudo chown $$(whoami) $(ROBOT_PATH)"
	ssh $(ROBOT_HOST) "sudo systemctl stop $(BINARY_NAME) 2>/dev/null || true"
	rsync -avz --progress build/linux-arm64/$(BINARY_NAME) $(CONFIG_FILE) $(ROBOT_HOST):$(ROBOT_PATH)/
	ssh $(ROBOT_HOST) "chmod +x $(ROBOT_PATH)/$(BINARY_NAME)"
	ssh $(ROBOT_HOST) "sudo systemctl start $(BINARY_NAME)"
	@echo "Deployment complete"

# Deploy config only
deploy-config:
	rsync -avz $(CONFIG_FILE) $(ROBOT_HOST):$(ROBOT_PATH)/
	ssh $(ROBOT_HOST) "sudo systemctl reload $(BINARY_NAME) || sudo systemctl restart $(BINARY_NAME)"

# Install systemd service
deploy-service:
	scp deploy/$(BINARY_NAME).service $(ROBOT_HOST):/tmp/
	ssh $(ROBOT_HOST) "sudo mv /tmp/$(BINARY_NAME).service /etc/systemd/system/ && \
		sudo systemctl daemon-reload && \
		sudo systemctl enable $(BINARY_NAME)"

# Initial deployment (first time)
deploy-init: build-pi deploy-service deploy
	@echo "Initial deployment complete"

# ============================================
# Robot Management
# ============================================

start:
	ssh $(ROBOT_HOST) "sudo systemctl start $(BINARY_NAME)"

stop:
	ssh $(ROBOT_HOST) "sudo systemctl stop $(BINARY_NAME)"

restart:
	ssh $(ROBOT_HOST) "sudo systemctl restart $(BINARY_NAME)"

status:
	ssh $(ROBOT_HOST) "sudo systemctl status $(BINARY_NAME)"

logs:
	ssh $(ROBOT_HOST) "sudo journalctl -u $(BINARY_NAME) -f"

ssh:
	ssh $(ROBOT_HOST)

# ============================================
# Development
# ============================================

# Run locally (requires NATS)
run: build
	./build/$(BINARY_NAME) --config $(CONFIG_FILE)

# Watch for changes and regenerate
watch:
	@echo "Watching for changes..."
	@while true; do \
		inotifywait -q -e modify $(CONFIG_FILE) components/ services/ 2>/dev/null || sleep 2; \
		echo "Change detected, regenerating..."; \
		gorai generate $(ROBOT_NAME); \
	done
`, robotName)
}

func generateReadme(robotName string) string {
	return fmt.Sprintf(`# %s

A Gorai robot project.

## Getting Started

### Prerequisites

- Go 1.22 or later
- NATS server (for local development)
- Raspberry Pi or other Linux SBC (for deployment)

### Development

1. Edit `+"`%s.json`"+` to define your robot's components and services
2. Run `+"`gorai generate %s`"+` to regenerate code
3. Build with `+"`make build`"+`
4. Run locally with `+"`make run`"+`

### Deployment

1. Set your robot's hostname: `+"`export ROBOT_HOST=pi@%s.local`"+`
2. Deploy: `+"`make deploy`"+`

## Project Structure

`+"```"+`
%s/
+-- %s.json         # Robot Definition Language config
+-- main.go             # Entry point
+-- go.mod              # Go module
+-- Makefile            # Build and deployment
+-- components/         # Custom component implementations
+-- services/           # Custom service implementations
+-- internal/generated/ # Auto-generated code (do not edit)
+-- deploy/             # Deployment files (systemd, etc.)
`+"```"+`

## Commands

- `+"`make build`"+`              - Build for current platform
- `+"`make build-pi`"+`           - Build for Raspberry Pi (ARM64)
- `+"`make deploy`"+`             - Deploy to robot
- `+"`make logs`"+`               - View robot logs
- `+"`gorai validate %s`"+`   - Validate config
- `+"`gorai generate %s`"+`   - Regenerate code

## Environment Variables

- `+"`GORAI_ROBOT_NAME`"+` - Default robot name for gorai commands

## Documentation

- [Gorai Documentation](https://gorai.dev/docs)
- [RDL Specification](https://gorai.dev/docs/rdl)
`, robotName, robotName, robotName, robotName, robotName, robotName, robotName, robotName)
}

func generateGitignore() string {
	return `# Binaries
build/
*.exe
*.dll
*.so
*.dylib

# Test
*.test
coverage.out

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Logs
*.log
.logs/

# Local development
.env
.env.local
`
}

func generateSystemdService(robotName string) string {
	configFile := robotName + ".json"
	return fmt.Sprintf(`[Unit]
Description=%s Gorai Robot
After=network.target nats.service
Wants=nats.service

[Service]
Type=simple
User=pi
Group=pi
WorkingDirectory=/opt/%s
ExecStart=/opt/%s/%s --config /opt/%s/%s
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/opt/%s

# Resource limits
MemoryMax=512M
CPUQuota=80%%

[Install]
WantedBy=multi-user.target
`, robotName, robotName, robotName, robotName, robotName, configFile, robotName)
}

func generateGitHubWorkflow(robotName string) string {
	return fmt.Sprintf(`# .github/workflows/build.yml
# Generated by gorai init. You may edit this file.

name: Build and Test

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

env:
  GORAI_ROBOT_NAME: %s

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Install gorai
        run: go install github.com/gorai/gorai/cmd/gorai@latest

      - name: Validate config
        run: gorai validate %s

  build:
    needs: validate
    runs-on: ubuntu-latest
    strategy:
      matrix:
        target: [linux-arm64, linux-amd64]
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Install gorai
        run: go install github.com/gorai/gorai/cmd/gorai@latest

      - name: Generate
        run: gorai generate %s

      - name: Build
        run: |
          if [ "${{ matrix.target }}" = "linux-arm64" ]; then
            GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/${{ matrix.target }}/%s .
          else
            GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/${{ matrix.target }}/%s .
          fi

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: binary-${{ matrix.target }}
          path: build/${{ matrix.target }}/

  test:
    needs: validate
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Install gorai
        run: go install github.com/gorai/gorai/cmd/gorai@latest

      - name: Generate
        run: gorai generate %s

      - name: Test
        run: go test ./...
`, robotName, robotName, robotName, robotName, robotName, robotName)
}

func generateDocGo() string {
	return `// Code generated by gorai generate. DO NOT EDIT.
// Regenerate with: gorai generate <robot-name>

// Package generated contains auto-generated code derived from the robot
// configuration file. This package is regenerated every time ` + "`gorai generate`" + `
// is run. Do not edit files in this package manually.
package generated
`
}

func initGitRepo(projectDir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = projectDir
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func runGenerateInProject(projectDir, robotName string) error {
	// For now, just create the imports.go file
	// The full generate command will handle this when run separately
	importsGo := generateEmptyImportsGo(robotName)
	return os.WriteFile(filepath.Join(projectDir, "internal/generated/imports.go"), []byte(importsGo), 0644)
}

func generateEmptyImportsGo(robotName string) string {
	return fmt.Sprintf(`// Code generated by gorai generate from %s.json. DO NOT EDIT.

package generated

import (
	// ============================================
	// Standard Gorai Components (from %s.json)
	// ============================================

	// (will be populated by gorai generate)

	// ============================================
	// Standard Gorai Services (from %s.json)
	// ============================================

	// (will be populated by gorai generate)

	// ============================================
	// Custom Components (from components/)
	// ============================================

	// (none defined yet - add with: gorai add component <name>)

	// ============================================
	// Custom Services (from services/)
	// ============================================

	// (none defined yet - add with: gorai add service <name>)
)

// RobotConfig is the path to the configuration file
const RobotConfig = "%s.json"
`, robotName, robotName, robotName, robotName)
}
