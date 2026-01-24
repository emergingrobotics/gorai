# Project Organization

As your robot codebase grows, organization matters. This chapter covers best practices for Gorai projects.

## The Gorai Monorepo Structure

The main Gorai repository:

```
github.com/gorai/gorai/
├── api/                 # Protocol definitions
│   ├── proto/           # .proto source files
│   │   └── gorai/
│   │       ├── std/
│   │       ├── geometry/
│   │       ├── sensor/
│   │       └── vision/
│   └── gen/             # Generated Go code
├── pkg/                 # Core packages
│   ├── node/
│   ├── pub/
│   ├── sub/
│   ├── resource/
│   └── registry/
├── components/           # Component interfaces
│   ├── component.go
│   ├── motor/
│   ├── camera/
│   └── sensor/
├── services/             # Service interfaces
│   ├── service.go
│   ├── vision/
│   └── navigation/
├── driver/              # Hardware drivers
│   ├── gpio/
│   ├── i2c/
│   └── serial/
├── accel/               # Acceleration layer
│   ├── accel.go
│   └── rknn/
├── nws/                 # Network wrappers
├── examples/            # Example applications
├── internal/            # Private packages
├── scripts/             # Utility scripts
├── specs/               # Specifications
└── docs/                # Documentation
```

## When to Use the Monorepo

Use the core monorepo for:

- Framework development
- Changes affecting multiple packages
- Core components/service interfaces
- Shared protocol definitions

## When to Create Separate Repos

### Custom Components

Hardware-specific drivers:

```
github.com/myorg/gorai-drv8833/
├── drv8833.go          # Motor driver
├── drv8833_test.go
├── fake/
│   └── fake.go
├── go.mod              # Imports gorai/gorai
└── README.md
```

`go.mod`:

```
module github.com/myorg/gorai-drv8833

go 1.22

require (
    github.com/gorai/gorai v0.2.0
)
```

### Robot Applications

Complete robot packages:

```
github.com/myorg/my-robot/
├── cmd/
│   └── myrobot/
│       └── main.go
├── config/
│   ├── default.json
│   └── production.json
├── components/          # Custom components
│   └── arm/
├── services/            # Custom services
│   └── behavior/
├── internal/            # Private packages
├── scripts/
└── go.mod
```

### Driver Packages

Drivers with external dependencies:

```
github.com/gorai/gorai-driver-v4l2/
├── camera.go           # V4L2 camera driver
├── camera_test.go
├── go.mod              # CGo dependencies isolated
└── README.md
```

Why separate?

- CGo adds build complexity
- Vendor-specific licenses
- Platform-specific code
- Optional functionality

## Import Paths and Versioning

**Semantic versioning**:

```
v0.1.0  # Initial development
v0.2.0  # Breaking changes (pre-1.0)
v1.0.0  # First stable release
v1.1.0  # Backward-compatible features
v1.1.1  # Bug fixes
v2.0.0  # Breaking changes (new import path)
```

**Go module versioning**:

```go
// v0.x and v1.x
import "github.com/gorai/gorai/pkg/node"

// v2+
import "github.com/gorai/gorai/v2/pkg/node"
```

## Configuration Organization

```
config/
├── default.json         # Development defaults
├── production.json      # Production settings
├── robots/
│   ├── robot1.json      # Robot-specific
│   └── robot2.json
└── components/
    └── motor_left.json  # Component-specific
```

**Layered configuration**:

```go
// Load base, then overlay environment-specific
config := loadConfig("default.json")
config.Merge(loadConfig(os.Getenv("GORAI_ENV") + ".json"))
```

## Multi-Robot Fleet Organization

```
github.com/myorg/robot-fleet/
├── shared/              # Common code
│   ├── components/
│   └── services/
├── robots/
│   ├── scout/           # Scout robot type
│   │   ├── cmd/
│   │   │   └── scout/
│   │   └── config/
│   └── carrier/         # Carrier robot type
│       ├── cmd/
│       │   └── carrier/
│       └── config/
├── fleet-manager/       # Central coordination
│   ├── cmd/
│   │   └── manager/
│   └── api/
├── deploy/              # Deployment configs
│   ├── docker/
│   └── kubernetes/
└── go.mod
```

## Documentation Standards

### README per Package

```markdown
# package motor

Motor component interface for Gorai.

## Installation

go get github.com/gorai/gorai/components/motor

## Usage

motor := fake.NewMotor()
motor.SetPower(ctx, 0.5)

## Configuration

| Field | Type | Description |
|-------|------|-------------|
| max_power | float64 | Maximum power (0-1) |
```

### GoDoc Comments

```go
// Motor represents a controllable motor.
//
// Motors can be controlled via power (open-loop), velocity (closed-loop),
// or position (closed-loop). Use Properties to discover capabilities.
//
// Example:
//
//     motor, _ := drv8833.New("left", cfg, pins)
//     motor.SetPower(ctx, 0.5)  // 50% forward
//     motor.Stop(ctx)
//
type Motor interface {
    // SetPower sets the motor power from -1.0 (full reverse) to 1.0 (full forward).
    //
    // Power values are clamped to configured max_power.
    // Returns an error if the motor is in a fault state.
    SetPower(ctx context.Context, power float64) error

    // ...
}
```

### Example Files

```go
// motor_example_test.go
package motor_test

func ExampleMotor_SetPower() {
    motor := fake.NewMotor()

    motor.SetPower(context.Background(), 0.5)
    // Motor runs at 50% power

    motor.Stop(context.Background())
    // Motor stops

    // Output:
}
```

Run examples as tests:

```bash
go test -v -run Example
```

## Dependency Management

### Pinning Versions

```go
// go.mod
require (
    github.com/gorai/gorai v0.2.0
    github.com/nats-io/nats.go v1.31.0
)
```

### Updating Dependencies

```bash
# Update a specific dependency
go get -u github.com/gorai/gorai@v0.3.0

# Update all dependencies
go get -u ./...

# Tidy unused dependencies
go mod tidy
```

### Vendoring (Optional)

For reproducible builds:

```bash
go mod vendor
```

Then build with:

```bash
go build -mod=vendor ./...
```

## Build Organization

### Makefile Targets

```makefile
.PHONY: build test lint clean

build:
    go build ./cmd/...

test:
    go test ./...

test-all:
    go test -tags="component integration" ./...

lint:
    golangci-lint run

proto:
    buf generate

clean:
    rm -rf bin/ tmp/
```

### Cross-Compilation

Build for target platform:

```bash
# For Raspberry Pi (ARM64)
GOOS=linux GOARCH=arm64 go build -o myrobot-arm64 ./cmd/myrobot

# For Orange Pi (ARM64)
GOOS=linux GOARCH=arm64 go build -o myrobot-arm64 ./cmd/myrobot
```

### Build Tags for Platform Code

```go
//go:build linux && arm64
// +build linux,arm64

package gpio

// ARM64-specific GPIO implementation
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: CI

on: [push, pull_request]

jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]
        go: ['1.21', '1.22']
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go }}
      - run: go test ./...

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: golangci/golangci-lint-action@v3
```

## Release Process

### Version Tags

```bash
git tag v0.2.0
git push origin v0.2.0
```

### Changelog

```markdown
# Changelog

## [0.2.0] - 2024-01-15

### Added
- DRV8833 motor driver
- Ultrasonic sensor component

### Changed
- Motor interface: added Properties() method

### Fixed
- Race condition in sensor publishing
```

---

Chapter 17 explores using AI tools to accelerate your Gorai development workflow.
