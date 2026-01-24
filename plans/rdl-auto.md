# RDL-Driven Robot Project Automation

## Proposal: Automated Robot Code Generation from RDL

**Author:** Claude
**Status:** Proposal
**Created:** 2024

---

## 1. Executive Summary

This proposal defines a workflow where developers create robots by:

1. Starting a new Git repository
2. Writing a Robot Definition Language (RDL) file
3. Running tools that auto-generate a complete, buildable code skeleton

The goal is to minimize boilerplate while maintaining full control and extensibility.

---

## 2. Problem Statement

### Current Pain Points

1. **Too much boilerplate**: Starting a new robot project requires creating main.go, Makefile, systemd files, deploy scripts, etc.

2. **Easy to miss imports**: Forgetting to import a component package means it won't be registered.

3. **Configuration drift**: The RDL file and the code can diverge (e.g., RDL references a component type not imported).

4. **No validation at dev time**: Errors only appear at runtime when the robot fails to start.

5. **Inconsistent project structure**: Every developer structures their robot project differently.

### Desired State

- Write RDL → Run one command → Get working robot project
- RDL is the source of truth
- Tooling validates RDL against available components
- Generated code stays in sync with RDL
- Clear separation between generated and user code

---

## 3. Design Principles

1. **RDL-First**: The RDL file is the primary artifact; code is derived from it
2. **Convention over Configuration**: Sensible defaults, override when needed
3. **No Magic**: Generated code is readable and understandable
4. **Escape Hatches**: Users can always drop down to manual code
5. **Incremental Adoption**: Works for simple robots, scales to complex ones
6. **Standard Go**: Uses go modules, not git submodules (Go-idiomatic)

---

## 4. Workflow Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    Robot Development Workflow                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Step 1: Initialize Project                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ $ gorai init myrobot                                    │    │
│  │ $ cd myrobot                                            │    │
│  │ $ git init                                              │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           │                                     │
│                           ▼                                     │
│  Step 2: Define Robot (edit RDL)                                │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ $ vim robot.json                                        │    │
│  │                                                         │    │
│  │ {                                                       │    │
│  │   "robot": { "name": "myrobot" },                       │    │
│  │   "components": [                                       │    │
│  │     { "name": "imu", "type": "ahrs", "model": "bno055" }│    │
│  │   ]                                                     │    │
│  │ }                                                       │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           │                                     │
│                           ▼                                     │
│  Step 3: Generate Code                                          │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ $ gorai generate                                        │    │
│  │ ✓ Validated robot.json                                  │    │
│  │ ✓ Generated internal/generated/imports.go               │    │
│  │ ✓ Generated deploy/myrobot.service                      │    │
│  │ ✓ Updated go.mod dependencies                           │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           │                                     │
│                           ▼                                     │
│  Step 4: (Optional) Add Custom Components                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ $ gorai add component my_sensor --type sensor           │    │
│  │ Created: components/my_sensor/sensor.go                 │    │
│  │                                                         │    │
│  │ # Edit robot.json to use it:                            │    │
│  │ { "name": "custom", "type": "sensor", "model": "my_sensor" } │
│  │                                                         │    │
│  │ $ gorai generate   # Regenerate to pick up new component│    │
│  └─────────────────────────────────────────────────────────┘    │
│                           │                                     │
│                           ▼                                     │
│  Step 5: Build and Deploy                                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ $ make build-pi                                         │    │
│  │ $ make deploy ROBOT_HOST=pi@myrobot.local               │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 5. Project Structure

### 5.1 Generated Structure

After `gorai init myrobot && gorai generate`:

```
myrobot/
├── robot.json                    # RDL - THE source of truth
├── main.go                       # Entry point (generated once, user-editable)
├── go.mod                        # Go module (generated, updated by gorai generate)
├── go.sum                        # Go checksums
├── Makefile                      # Build/deploy (generated once, user-editable)
├── README.md                     # Project readme (generated once)
│
├── internal/
│   └── generated/                # AUTO-GENERATED - regenerated by gorai generate
│       ├── doc.go                # Package documentation
│       ├── imports.go            # Component/service imports
│       └── validate.go           # Compile-time validation
│
├── components/                   # User's custom components (empty initially)
│   └── .gitkeep
│
├── services/                     # User's custom services (empty initially)
│   └── .gitkeep
│
├── deploy/
│   ├── myrobot.service           # systemd unit (regenerated)
│   └── install.sh                # Robot setup script (regenerated)
│
├── scripts/
│   └── dev.sh                    # Development helpers (generated once)
│
└── .github/
    └── workflows/
        └── build.yml             # CI workflow (generated once)
```

### 5.2 File Ownership

| File/Directory | Generated | Regenerated | User-Editable |
|----------------|-----------|-------------|---------------|
| `robot.json` | Once | Never | **Yes** (primary) |
| `main.go` | Once | Never | Yes |
| `go.mod` | Once | Updated | Yes (but managed) |
| `Makefile` | Once | Never | Yes |
| `internal/generated/*` | Yes | **Always** | **No** |
| `deploy/*.service` | Yes | **Always** | No (use overrides) |
| `components/*` | Never | Never | **Yes** (user code) |
| `services/*` | Never | Never | **Yes** (user code) |
| `.github/workflows/*` | Once | Never | Yes |

### 5.3 The `internal/generated/` Directory

This is the key innovation—a directory that's completely regenerated on every `gorai generate`:

```go
// internal/generated/doc.go
// Code generated by gorai generate from robot.json. DO NOT EDIT.
// Regenerate with: gorai generate

// Package generated contains auto-generated code derived from the robot.json
// configuration file. This package is regenerated every time `gorai generate`
// is run. Do not edit files in this package manually.
package generated
```

```go
// internal/generated/imports.go
// Code generated by gorai generate from robot.json. DO NOT EDIT.

package generated

import (
    // ============================================
    // Standard Gorai Components (from robot.json)
    // ============================================

    // Sensors
    _ "github.com/gorai/gorai/components/sensor/bno055"    // ahrs: front_imu
    _ "github.com/gorai/gorai/components/sensor/rplidar"   // lidar: front_lidar

    // Actuators
    _ "github.com/gorai/gorai/components/motor/gpio"       // motor: left_motor, right_motor
    _ "github.com/gorai/gorai/components/base/differential" // base: base

    // Cameras
    _ "github.com/gorai/gorai/components/camera/v4l2"      // camera: front_camera

    // ============================================
    // Standard Gorai Services (from robot.json)
    // ============================================

    _ "github.com/gorai/gorai/services/vision/yolox"           // vision: detector
    _ "github.com/gorai/gorai/services/slam/cartographer"      // slam: mapper
    _ "github.com/gorai/gorai/services/navigation/default"     // navigation: navigator

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
const RobotConfig = "robot.json"

// ComponentCount is the number of components defined
const ComponentCount = 5

// ServiceCount is the number of services defined
const ServiceCount = 3
```

```go
// internal/generated/validate.go
// Code generated by gorai generate from robot.json. DO NOT EDIT.

package generated

import (
    "github.com/gorai/gorai/components/motor"
    "github.com/gorai/gorai/components/sensor"
    "github.com/gorai/gorai/components/camera"
    "github.com/gorai/gorai/components/base"
    "github.com/gorai/gorai/services/vision"
    "github.com/gorai/gorai/services/slam"
    "github.com/gorai/gorai/services/navigation"
)

// These type assertions ensure that the interfaces referenced in robot.json
// actually exist at compile time. If any of these fail to compile, your
// robot.json references a components/service type that doesn't exist.

var (
    // Component interface assertions
    _ motor.Motor     = nil  // motor type exists
    _ sensor.AHRS     = nil  // ahrs type exists
    _ sensor.LiDAR    = nil  // lidar type exists
    _ camera.Camera   = nil  // camera type exists
    _ base.Base       = nil  // base type exists

    // Service interface assertions
    _ vision.Service     = nil  // vision type exists
    _ slam.Service       = nil  // slam type exists
    _ navigation.Service = nil  // navigation type exists
)
```

---

## 6. Command Reference

### 6.1 `gorai init`

Creates a new robot project.

```bash
gorai init <project-name> [flags]

Flags:
  --template <name>    Use a specific template (default: "standard")
  --no-git             Don't initialize git repository
  --no-generate        Don't run gorai generate after init

Templates:
  standard             Basic robot with examples
  minimal              Bare minimum files
  wheeled              Differential drive robot starter
  arm                  Robot arm starter
```

**Example:**

```bash
$ gorai init my-robot
Creating project: my-robot
  ✓ Created my-robot/
  ✓ Created my-robot/robot.json
  ✓ Created my-robot/main.go
  ✓ Created my-robot/Makefile
  ✓ Created my-robot/go.mod
  ✓ Created my-robot/README.md
  ✓ Created my-robot/deploy/
  ✓ Created my-robot/components/
  ✓ Created my-robot/services/
  ✓ Initialized git repository
  ✓ Running gorai generate...

Project created! Next steps:
  cd my-robot
  vim robot.json        # Define your robot
  gorai generate        # Regenerate after changes
  make build            # Build for local testing
  make build-pi         # Build for Raspberry Pi
```

### 6.2 `gorai generate`

Regenerates code from RDL.

```bash
gorai generate [flags]

Flags:
  --config <path>      Path to robot.json (default: "robot.json")
  --dry-run            Show what would be generated without writing
  --verbose            Show detailed output
  --validate-only      Only validate, don't generate
```

**Example:**

```bash
$ gorai generate
Reading robot.json...
Validating configuration...
  ✓ Robot name: my-robot
  ✓ 5 components defined
  ✓ 3 services defined
  ✓ All component types valid
  ✓ All service types valid
  ✓ No circular dependencies

Generating code...
  ✓ internal/generated/imports.go (updated)
  ✓ internal/generated/validate.go (updated)
  ✓ deploy/my-robot.service (updated)
  ✓ go.mod (updated: added github.com/gorai/gorai/components/sensor/bno055)

Generation complete!
```

### 6.3 `gorai add component`

Scaffolds a custom component.

```bash
gorai add component <name> [flags]

Flags:
  --type <type>        Component type: sensor, motor, servo, etc.
  --interface <iface>  Specific interface to implement
```

**Example:**

```bash
$ gorai add component my_sensor --type sensor
Creating custom component: my_sensor
  ✓ Created components/my_sensor/sensor.go
  ✓ Created components/my_sensor/sensor_test.go
  ✓ Created components/my_sensor/fake/fake.go

Next steps:
  1. Edit components/my_sensor/sensor.go to implement your sensor
  2. Add to robot.json:
     { "name": "my_sensor_instance", "type": "sensor", "model": "my_sensor" }
  3. Run: gorai generate
```

Generated stub:

```go
// components/my_sensor/sensor.go
package my_sensor

import (
    "context"

    "github.com/gorai/gorai/pkg/registry"
    "github.com/gorai/gorai/pkg/resource"
)

func init() {
    registry.RegisterComponent("sensor", "my_sensor", New)
}

// Config holds configuration for MySensor
type Config struct {
    // TODO: Add your configuration fields here
    SampleRate int `json:"sample_rate"`
}

// MySensor is a custom sensor implementation
type MySensor struct {
    name   resource.Name
    config Config
    // TODO: Add your fields here
}

// New creates a new MySensor from configuration
func New(ctx context.Context, deps resource.Dependencies, conf registry.Config) (interface{}, error) {
    var cfg Config
    if err := conf.Decode(&cfg); err != nil {
        return nil, err
    }

    return &MySensor{
        name:   resource.NewComponentName("myrobot", "sensor", conf.String("name")),
        config: cfg,
    }, nil
}

// Name returns the resource name
func (s *MySensor) Name() resource.Name {
    return s.name
}

// Readings returns sensor readings
func (s *MySensor) Readings(ctx context.Context) (map[string]any, error) {
    // TODO: Implement your sensor reading logic
    return map[string]any{
        "value": 0.0,
    }, nil
}

// Reconfigure updates the sensor configuration
func (s *MySensor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf registry.Config) error {
    var cfg Config
    if err := conf.Decode(&cfg); err != nil {
        return err
    }
    s.config = cfg
    return nil
}

// DoCommand handles arbitrary commands
func (s *MySensor) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
    return nil, nil
}

// Close cleans up the sensor
func (s *MySensor) Close(ctx context.Context) error {
    return nil
}
```

### 6.4 `gorai add service`

Scaffolds a custom service.

```bash
gorai add service <name> [flags]

Flags:
  --type <type>        Service type: vision, behavior, etc.
```

### 6.5 `gorai validate`

Validates RDL without generating.

```bash
gorai validate [config]

$ gorai validate robot.json
Validating robot.json...
  ✓ JSON syntax valid
  ✓ Schema valid (RDL v1)
  ✓ Robot name: my-robot
  ✓ 5 components defined
  ✓ 3 services defined
  ✓ All component types registered
  ✓ All models available
  ✓ Dependencies resolvable
  ✓ No circular dependencies

robot.json is valid!
```

### 6.6 `gorai list`

Lists available components and services.

```bash
gorai list components
gorai list services
gorai list models --type motor

$ gorai list components
Component Types:
  Sensors:
    imu          - Inertial Measurement Unit
    ahrs         - Attitude/Heading Reference System (models: bno055, bno085, fake)
    gps          - GPS/GNSS Receiver (models: neo6m, neo_m8n, fake)
    encoder      - Rotary/Linear Encoder (models: quadrature, as5600, fake)
    lidar        - Laser Scanner (models: rplidar_a1, rplidar_a2, fake)
    ...

  Actuators:
    motor        - DC/Brushless Motor (models: gpio, odrive, fake)
    servo        - Position Servo (models: pwm, dynamixel, fake)
    stepper      - Stepper Motor (models: gpio, tmc2209, fake)
    ...
```

---

## 7. Generated File Templates

### 7.1 main.go (Generated Once)

```go
// main.go
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
    _ "myrobot/internal/generated"

    // Custom components - add your component imports here
    // _ "myrobot/components/my_sensor"
)

func main() {
    configPath := flag.String("config", "robot.json", "Path to robot configuration")
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
```

### 7.2 Makefile (Generated Once)

```makefile
# Makefile
# Generated by gorai init. You may edit this file.

# ============================================
# Project Configuration
# ============================================

BINARY_NAME := myrobot
BUILD_TARGET ?= linux-arm64

# Deployment (override with environment or command line)
ROBOT_HOST ?= pi@myrobot.local
ROBOT_PATH ?= /opt/$(BINARY_NAME)

# ============================================
# Go Configuration
# ============================================

GOFLAGS := -trimpath
LDFLAGS := -s -w

# Version info (set by CI or manually)
VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS += -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# ============================================
# Targets
# ============================================

.PHONY: all build build-pi generate test lint clean deploy

all: generate build

# Generate code from robot.json
generate:
	gorai generate

# Build for current platform
build: generate
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o build/$(BINARY_NAME) .

# Build for Raspberry Pi (ARM64)
build-pi: generate
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
		-o build/linux-arm64/$(BINARY_NAME) .

# Build for all platforms
build-all: generate
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o build/linux-arm64/$(BINARY_NAME) .
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o build/linux-amd64/$(BINARY_NAME) .

# Run tests
test:
	go test ./...

# Run linter
lint:
	golangci-lint run

# Validate robot.json
validate:
	gorai validate robot.json

# Clean build artifacts
clean:
	rm -rf build/

# ============================================
# Deployment
# ============================================

# Deploy to robot
deploy: build-pi
	@echo "Deploying to $(ROBOT_HOST):$(ROBOT_PATH)"
	ssh $(ROBOT_HOST) "sudo mkdir -p $(ROBOT_PATH) && sudo chown pi:pi $(ROBOT_PATH)"
	ssh $(ROBOT_HOST) "sudo systemctl stop $(BINARY_NAME) 2>/dev/null || true"
	rsync -avz --progress build/linux-arm64/$(BINARY_NAME) robot.json $(ROBOT_HOST):$(ROBOT_PATH)/
	ssh $(ROBOT_HOST) "chmod +x $(ROBOT_PATH)/$(BINARY_NAME)"
	ssh $(ROBOT_HOST) "sudo systemctl start $(BINARY_NAME)"
	@echo "Deployment complete"

# Deploy config only
deploy-config:
	rsync -avz robot.json $(ROBOT_HOST):$(ROBOT_PATH)/
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
	./build/$(BINARY_NAME) --config robot.json

# Watch for changes and regenerate
watch:
	@echo "Watching for changes..."
	@while true; do \
		inotifywait -q -e modify robot.json components/ services/ 2>/dev/null || sleep 2; \
		echo "Change detected, regenerating..."; \
		gorai generate; \
	done
```

### 7.3 robot.json (Generated Once, User Edits)

```json
{
  "$schema": "https://gorai.dev/schemas/rdl-v1.json",
  "version": "1",

  "robot": {
    "name": "myrobot",
    "namespace": "myrobot",
    "description": "My Gorai Robot"
  },

  "nats": {
    "url": "nats://localhost:4222",
    "jetstream": true
  },

  "components": [
    {
      "name": "example_sensor",
      "type": "temperature",
      "model": "fake",
      "attributes": {
        "initial_temp": 25.0
      }
    }
  ],

  "services": [],

  "log": {
    "level": "info",
    "format": "json",
    "output": "stdout"
  },

  "dashboard": {
    "enabled": true,
    "listen": ":8080"
  }
}
```

---

## 8. Validation and Error Handling

### 8.1 Validation Stages

```
┌─────────────────────────────────────────────────────────────┐
│                    Validation Pipeline                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Stage 1: JSON Syntax                                       │
│  ├─ Valid JSON?                                             │
│  └─ Parse errors → immediate failure with line number       │
│                                                             │
│  Stage 2: Schema Validation                                 │
│  ├─ Required fields present?                                │
│  ├─ Field types correct?                                    │
│  └─ Unknown fields → warning (not error)                    │
│                                                             │
│  Stage 3: Semantic Validation                               │
│  ├─ Component types exist in registry?                      │
│  ├─ Models exist for each type?                             │
│  ├─ Dependencies reference existing resources?              │
│  └─ Circular dependencies?                                  │
│                                                             │
│  Stage 4: Custom Component Detection                        │
│  ├─ Scan components/ directory                              │
│  ├─ Find init() registrations                               │
│  └─ Match against robot.json models                         │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 8.2 Error Messages

```bash
$ gorai generate

ERROR: robot.json:15: Unknown component type "imu2"

  { "name": "front_imu", "type": "imu2", "model": "bno055" }
                                ^^^^^^

  Did you mean "imu" or "ahrs"?

  Available sensor types:
    imu, ahrs, gps, encoder, range_sensor, lidar, presence_sensor,
    thermal_array, force_sensor, force_6dof, current_sensor,
    reflectance_sensor, camera, temperature

  Run 'gorai list components' for full list.
```

```bash
$ gorai generate

ERROR: robot.json:23: Unknown model "bno056" for type "ahrs"

  { "name": "front_imu", "type": "ahrs", "model": "bno056" }
                                                  ^^^^^^^^

  Did you mean "bno055"?

  Available models for "ahrs":
    bno055   - Bosch BNO055 9-DOF AHRS
    bno085   - Bosch BNO085 9-DOF AHRS
    fake     - Fake AHRS for testing

  Or create a custom model:
    gorai add component my_ahrs --type ahrs
```

```bash
$ gorai generate

ERROR: robot.json: Circular dependency detected

  navigator → mapper → base → navigator

  Service "navigator" depends on "mapper"
  Service "mapper" depends on "base"
  Component "base" depends on "navigator" (in attributes)

  Remove one dependency to break the cycle.
```

---

## 9. Integration with Go Modules

### 9.1 Dependency Management

`gorai generate` updates go.mod based on robot.json:

```bash
$ gorai generate --verbose

Analyzing dependencies...
  robot.json requires:
    - github.com/gorai/gorai/components/sensor/bno055
    - github.com/gorai/gorai/components/motor/gpio
    - github.com/gorai/gorai/services/vision/yolox

  go.mod has:
    - github.com/gorai/gorai v0.2.0

  Action: No changes needed (subpackages included in main module)

$ cat go.mod
module myrobot

go 1.22

require github.com/gorai/gorai v0.2.0
```

### 9.2 Version Pinning

```bash
# Update gorai version
go get github.com/gorai/gorai@v0.3.0
gorai generate  # Regenerate with new version
```

### 9.3 Local Development (Optional)

For developing gorai itself:

```bash
# Use replace directive in go.mod
go mod edit -replace github.com/gorai/gorai=../gorai

# Or set GOFLAGS
export GOFLAGS="-mod=mod"
```

This is preferable to git submodules for Go projects.

---

## 10. CI/CD Integration

### 10.1 Generated GitHub Actions Workflow

```yaml
# .github/workflows/build.yml
# Generated by gorai init. You may edit this file.

name: Build and Test

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

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

      - name: Validate robot.json
        run: gorai validate robot.json

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
        run: gorai generate

      - name: Build
        run: make build-${{ matrix.target }}

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
        run: gorai generate

      - name: Test
        run: make test
```

---

## 11. Advanced Features

### 11.1 Multiple Robot Configurations

For robots with multiple configurations (dev, prod, different hardware):

```
myrobot/
├── robot.json              # Default/development
├── robot.prod.json         # Production
├── robot.sim.json          # Simulation
└── ...
```

```bash
# Generate for specific config
gorai generate --config robot.prod.json

# Build with specific config embedded
make build CONFIG=robot.prod.json
```

### 11.2 Configuration Inheritance (Future)

```json
{
  "$extends": "robot.base.json",
  "components": [
    { "name": "extra_sensor", "type": "lidar", "model": "rplidar_a2" }
  ]
}
```

### 11.3 Component Libraries (Future)

Share custom components across projects:

```bash
# In go.mod
require github.com/myorg/robot-components v1.0.0

# In robot.json
{ "name": "custom", "type": "sensor", "model": "myorg/special_sensor" }
```

---

## 12. Implementation Plan

### Phase 1: Core CLI (Week 1)

- [ ] `gorai init` - Basic project scaffolding
- [ ] `gorai generate` - Import generation
- [ ] `gorai validate` - RDL validation

### Phase 2: Templates and Stubs (Week 2)

- [ ] `gorai add component` - Component scaffolding
- [ ] `gorai add service` - Service scaffolding
- [ ] `gorai list` - List available types

### Phase 3: Advanced Features (Week 3)

- [ ] Smart dependency detection
- [ ] go.mod management
- [ ] CI/CD template generation

### Phase 4: Documentation (Week 4)

- [ ] Getting started guide
- [ ] Command reference
- [ ] Tutorial: Create your first robot

---

## 13. Alternatives Considered

### 13.1 Git Submodules

**Rejected because:**
- Not idiomatic Go
- Complex git workflow
- Harder for beginners
- Go modules solve this problem better

### 13.2 Full Code Generation (like protobuf)

**Rejected because:**
- Would regenerate user-editable files
- More complex to maintain
- Less flexible
- Users can't easily customize main.go

### 13.3 No Code Generation (Library Only)

**Rejected because:**
- Too much boilerplate for users
- Easy to forget imports
- No validation at dev time
- Inconsistent project structures

### 13.4 YAML Instead of JSON

**Rejected because:**
- JSON has native Go support
- JSON Schema is well-established
- YAML parsing has edge cases
- JSON is sufficient for RDL complexity

---

## 14. Success Criteria

1. **Time to first robot**: < 5 minutes from `gorai init` to running on Pi
2. **Zero boilerplate errors**: Generated code always compiles
3. **Clear error messages**: Validation errors explain how to fix
4. **No lost work**: User code is never overwritten
5. **Familiar workflow**: Feels like standard Go development

---

## 15. Open Questions

1. **Template repository vs CLI?** Should we also provide a GitHub template repo for click-to-start?

2. **Watch mode?** Should `gorai generate --watch` automatically regenerate on RDL changes?

3. **IDE integration?** Should we provide JSON Schema for autocomplete in editors?

4. **Versioned schemas?** How to handle RDL schema evolution?

5. **Remote components?** Should RDL support referencing components from other Go modules?
