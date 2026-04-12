# Component Packaging and Distribution — Design Approach

**Date:** 2026-04-12
**Status:** Draft
**Theme:** "npm for robotics"

---

## Problem Statement

A developer wants to build a new robot using GoRAI. They need to:

1. Start from a template and get the framework
2. Write an RDL describing their robot's components and services
3. Discover which components already exist (built-in or third-party)
4. Install components they need
5. Write custom components for hardware or behaviors unique to their robot
6. Share their custom components with the community

Today, none of this has a defined workflow. This document proposes one.

---

## Constraints

- **Single binary deployment.** `gorai run` produces one binary with embedded NATS. Components must compile into it.
- **Go-first, but not Go-only.** Most components will be Go. Some (vision, ML, SLAM) will be Python or C++. The approach must handle both without forcing containers on everyone.
- **No custom package manager.** Go modules already handle versioning, checksums, and dependency resolution. Use them.
- **Custom components live alongside shared ones.** A user's robot repo will contain both third-party and custom components in the same source tree. The boundary should be clean but not burdensome.
- **Start simple.** The first version needs to work for two people building ORCA and Surf. Scalability of the registry can come later.

---

## The Caddy Model

Caddy (the Go web server) solved an almost identical problem: a core binary with a plugin ecosystem where users choose which plugins compile into their build. Each plugin is a Go module that self-registers via `init()`. GoRAI already uses this pattern with `registry.RegisterComponent()`.

The approach: a user's robot project is a Go module. It has a `main.go` that imports the GoRAI core plus whichever components the robot needs. Each import triggers `init()` registration. The binary contains exactly what the RDL references — nothing more.

---

## How It Works

### Step 1: Clone the Template Repo

```bash
# Create a new robot project
git clone https://github.com/emergingrobotics/gorai-robot-template myrobot
cd myrobot
```

Template structure:

```
myrobot/
├── go.mod              # module myrobot; requires gorai
├── main.go             # imports gorai core + component registrations
├── robot.rdl.json      # robot definition (edit this)
├── components/         # custom components (Go packages)
│   └── .gitkeep
├── services/           # custom services (Go packages)
│   └── .gitkeep
├── Makefile            # build, test, validate, deploy targets
└── .gitignore
```

### Step 2: main.go Is the Bill of Materials

```go
package main

import (
    "github.com/emergingrobotics/gorai/cmd/gorai"

    // Built-in components (from gorai core)
    _ "github.com/emergingrobotics/gorai/components/sensor/gps"
    _ "github.com/emergingrobotics/gorai/components/motor/gpio"

    // Third-party components (from the registry)
    _ "github.com/gorai-community/sensor-bno055"
    _ "github.com/gorai-community/motor-vesc"

    // Custom components (local to this repo)
    _ "myrobot/components/depth"
)

func main() {
    gorai.Run()
}
```

Each blank import (`_`) triggers the package's `init()` function, which calls `registry.RegisterComponent()`. The compiled binary has exactly the components it needs.

This is the central design decision: **the Go import list is the component manifest.** No separate package.json, no lock file beyond `go.sum`, no custom resolution logic.

### Step 3: gorai component add

```bash
gorai component add sensor/bno055
```

This command does two things:

1. Looks up `sensor/bno055` in the component registry to find the Go module path
2. Runs `go get github.com/gorai-community/sensor-bno055@latest`
3. Adds `_ "github.com/gorai-community/sensor-bno055"` to `main.go`

Standard Go tooling handles versioning, checksums, and caching. No custom package manager.

```bash
gorai component search imu
# sensor/bno055   Bosch BNO055 9-DOF AHRS (I2C)         v0.1.0
# sensor/icm20948 TDK ICM-20948 9-DOF IMU (I2C/SPI)     v0.3.2
# sensor/mpu6050  InvenSense MPU-6050 6-DOF IMU (I2C)    v0.1.1
```

### Step 4: Write the RDL

```json
{
  "version": "2",
  "robot": {"name": "orca", "description": "Autonomous submersible"},
  "components": [
    {"name": "imu", "type": "sensor", "model": "bno055",
     "attributes": {"i2c_bus": 1, "address": "0x28"}},
    {"name": "depth", "type": "sensor", "model": "ms5837",
     "attributes": {"i2c_bus": 1, "address": "0x76"}},
    {"name": "left_motor", "type": "motor", "model": "gpio",
     "attributes": {"pin_a": 17, "pin_b": 18}}
  ]
}
```

### Step 5: Validate

```bash
gorai validate robot.rdl.json
```

Validation checks the RDL against the component registry that `init()` populated at startup:

- **Component found:** `sensor/bno055` — registered, configuration valid.
- **Component missing:** `sensor/ms5837` — not registered. Checks the remote registry and suggests: `Component sensor/ms5837 not found. Run 'gorai component add sensor/ms5837' to install it.`
- **Component unknown:** `sensor/foo` — not in local registry or remote registry. User must write a custom component.

### Step 6: Build and Run

```bash
gorai build robot.rdl.json -o orca --target linux/arm64
# Under the hood: GOOS=linux GOARCH=arm64 go build -o orca .

scp orca pi@orca.local:~
ssh pi@orca.local ./orca
```

Or for development:

```bash
gorai run robot.rdl.json
```

Single binary. Everything compiled in. No runtime dependency resolution.

---

## Custom Components

A user writes a depth sensor component for their robot. It lives in the same repo:

```
myrobot/
├── components/
│   └── depth/
│       ├── depth.go        # implements sensor.Sensor interface
│       └── depth_test.go
```

```go
package depth

import (
    "github.com/emergingrobotics/gorai/pkg/registry"
)

func init() {
    registry.RegisterComponent("sensor", "ms5837", New)
}

// New creates a new MS5837 depth/pressure sensor component.
func New(cfg map[string]interface{}) (interface{}, error) {
    // Read I2C bus and address from cfg
    // Return a component implementing sensor.Sensor
}
```

The `init()` registration means this component is indistinguishable from a built-in or third-party component. The RDL references it the same way. Application code accesses it the same way.

---

## Sharing Components

When a custom component is useful enough to share:

### Extract to a Standalone Go Module

```bash
mkdir -p ~/src/gorai-sensor-ms5837
# Move depth.go and depth_test.go
# Create go.mod: module github.com/username/gorai-sensor-ms5837
# Push to GitHub
```

Others install it:

```bash
gorai component add sensor/ms5837
# resolves to: go get github.com/username/gorai-sensor-ms5837
```

The component code does not change. The `init()` registration works identically whether the package is local or a Go module dependency. This is the "npm publish" equivalent.

### Submit to the Registry

```bash
gorai component publish sensor/ms5837
# Validates the module
# Submits an entry to the gorai-registry repo (PR or API)
```

This is a future convenience. Initially, registry entries can be added manually via PR to the registry repo.

---

## The Registry

The registry is a JSON file mapping component names to Go module paths:

```json
{
  "components": {
    "sensor/bno055": {
      "module": "github.com/gorai-community/sensor-bno055",
      "type": "sensor",
      "model": "bno055",
      "description": "Bosch BNO055 9-DOF AHRS",
      "hardware_access": "i2c",
      "version": "v0.1.0",
      "tags": ["imu", "ahrs", "9dof", "compass"]
    },
    "motor/vesc": {
      "module": "github.com/gorai-community/motor-vesc",
      "type": "motor",
      "model": "vesc",
      "description": "VESC ESC motor controller via UART",
      "hardware_access": "serial",
      "version": "v0.2.1",
      "tags": ["bldc", "esc", "foc"]
    }
  },
  "services": {
    "vision/yolo": {
      "repo": "github.com/gorai-community/vision-yolo",
      "language": "python",
      "type": "vision",
      "description": "YOLO object detection service",
      "version": "v0.1.0",
      "tags": ["detection", "camera", "ml"]
    }
  }
}
```

The registry lives in a GitHub repo (`emergingrobotics/gorai-registry`). `gorai component search` fetches and caches it locally. Simple to start, can move to a hosted API later if scale demands it.

---

## Non-Go Components

Some components will be written in Python (vision, ML inference) or C++ (SLAM, point cloud processing). These cannot compile into the Go binary.

**The answer: non-Go components are external services, not compiled-in components.**

GoRAI already has this distinction:
- **Components** = Go code, compiled into the binary, registered via `init()`, communicate via in-process NATS
- **External services** = separate processes, any language, communicate via NATS over the network

A Python vision pipeline is an external service defined in the RDL:

```json
{
  "services": [
    {
      "name": "detector",
      "type": "vision",
      "model": "yolo",
      "external": {
        "enabled": true,
        "command": "python3 /opt/gorai-vision-yolo/detect.py",
        "restart": "on-failure"
      }
    }
  ]
}
```

The registry can list non-Go services alongside Go components. The CLI differentiates:

```bash
gorai component add vision/yolo
# Detected: Python service (not a Go component)
# Install with: pip install gorai-vision-yolo
# Added external service definition to robot.rdl.json
```

**This is Phase 2 complexity.** For the initial release, the entire value proposition works with Go-only components. Non-Go services can be added manually to the RDL. Automated installation for non-Go services comes later.

---

## Hardware Access in Components

Components can use either hardware access pattern. The registry metadata indicates which:

### Co-Processor Components (RP2040 via GSP/2)

```json
{
  "sensor/bno055-gsp": {
    "module": "github.com/gorai-community/sensor-bno055-gsp",
    "hardware_access": "gsp",
    "description": "BNO055 via RP2040 co-processor (GSP/2)"
  }
}
```

The component talks to the RP2040 over GSP/2, which reads the BNO055 via I2C on the microcontroller side. Best for timing-critical or reliability-critical applications.

### Native RPi Components

```json
{
  "sensor/bno055": {
    "module": "github.com/gorai-community/sensor-bno055",
    "hardware_access": "i2c",
    "description": "BNO055 via native RPi I2C"
  }
}
```

The component reads the BNO055 directly from the RPi's I2C bus. Simpler setup, no co-processor needed.

Both produce components implementing the same `sensor.Sensor` interface. Application code and RDL configuration are identical — only the `model` field differs to select the driver.

---

## Template Repo Structure

```
gorai-robot-template/
├── go.mod
│   # module github.com/username/myrobot
│   # go 1.22
│   # require github.com/emergingrobotics/gorai v0.1.0
│
├── main.go
│   # Imports gorai core + blank imports for components
│   # This is the component manifest
│
├── robot.rdl.json
│   # Skeleton RDL — user edits this
│   # Comments explaining each section
│
├── components/
│   # User's custom Go components
│   # Each subdirectory is a package with init() registration
│   └── .gitkeep
│
├── services/
│   # User's custom Go services
│   # Each subdirectory is a package with init() registration
│   └── .gitkeep
│
├── Makefile
│   # build:    gorai build robot.rdl.json -o robot
│   # run:      gorai run robot.rdl.json
│   # test:     go test ./...
│   # validate: gorai validate robot.rdl.json
│   # deploy:   scp + ssh to target
│   # clean:    rm -f robot
│
├── .gitignore
│   # .env, .envrc, *~, bin/, robot (the built binary)
│
└── README.md
    # Project-specific README (not GoRAI docs)
```

The template is a GitHub template repository. Users click "Use this template" or `git clone` it. They rename the module in `go.mod`, edit `robot.rdl.json`, and start building.

---

## The Workflow End-to-End

```
1. Clone template
   └── git clone gorai-robot-template myrobot

2. Edit RDL
   └── Define components and services in robot.rdl.json

3. Search for components
   └── gorai component search depth
       → sensor/ms5837  MS5837 depth/pressure sensor (I2C)  v0.1.0

4. Install components
   └── gorai component add sensor/ms5837
       → go get github.com/gorai-community/sensor-ms5837
       → adds blank import to main.go

5. Write custom components
   └── Create components/mywidget/mywidget.go
       → Implement interface, register via init()
       → Add blank import to main.go

6. Validate
   └── gorai validate robot.rdl.json
       → All components registered ✓
       → Configuration valid ✓

7. Build and run
   └── gorai run robot.rdl.json         (development)
       gorai build robot.rdl.json -o robot  (deployment)

8. Share (optional)
   └── Extract custom component to standalone Go module
       → Push to GitHub
       → gorai component publish sensor/mywidget
```

---

## What to Build First

### Now (supports ORCA and Surf development)

1. **Template repo** — `gorai-robot-template` with the structure above
2. **`main.go` blank import pattern** — verify it works with the existing `gorai run` entrypoint
3. **A few reference components** — GPS, IMU (BNO055), depth sensor (MS5837), motor/GPIO as standalone Go modules that can be imported

### Next (supports public launch)

4. **Registry JSON file** — `gorai-registry` repo with component/service metadata
5. **`gorai component search`** — fetches and queries the registry
6. **`gorai component add`** — runs `go get` and edits `main.go`
7. **`gorai validate` enhancement** — checks RDL against registered components, suggests missing installs

### Later (supports community growth)

8. **`gorai component publish`** — validates a module and submits a PR to the registry
9. **Non-Go service installation** — automated setup for Python/C++ external services
10. **`gorai init` wizard** — interactive RDL builder that shows available components

---

## What Not to Build

- **Git submodules** — terrible developer experience, merge conflicts, version confusion
- **Custom package manager** — Go modules already solves versioning, checksums, caching
- **Container-based component distribution** — adds a container runtime dependency for a problem that Go modules solves
- **Monorepo of all components** — components should be independently versioned Go modules. A monorepo creates coupling and slows iteration
- **Language-agnostic plugin system** — the Go binary boundary is the plugin system. Non-Go code communicates via NATS as external services. Trying to load Python into a Go binary (CGo, WASM, etc.) is complexity for no benefit

---

## Open Questions

1. **Registry namespace governance** — who controls `sensor/bno055` vs. `sensor/bno055-gsp`? First-come-first-served? Curated? Namespaced by author (`username/sensor-bno055`)?

2. **Component metadata in Go source** — should components declare their RDL schema (required attributes, types, defaults) in Go code so `gorai validate` can check configuration without running the component? Something like:

   ```go
   func init() {
       registry.RegisterComponent("sensor", "bno055", New,
           registry.WithAttribute("i2c_bus", "int", true),
           registry.WithAttribute("address", "string", true),
       )
   }
   ```

3. **Template repo vs. `gorai init`** — is the template repo sufficient, or do users need a `gorai init myrobot` command that scaffolds the project? The template repo is simpler and more transparent. `gorai init` is more discoverable but adds CLI complexity.

4. **Component testing** — should the template include a standard pattern for testing components against simulated hardware? GoRAI already has `internal/testutil/` — should this be public for component authors?
