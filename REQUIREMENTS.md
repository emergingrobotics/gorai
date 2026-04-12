# GoRAI Platform Requirements

**Version:** 2.0
**Date:** 2026-04-12
**Status:** Active
**Supersedes:** Embedded NATS Requirements v1.0

---

## Document Index

This document is structured for autonomous agent consumption. Each section is self-contained. Jump to the section relevant to your task.

| Section | What It Covers | When to Read |
|---------|---------------|--------------|
| [1. Vision](#1-vision) | What GoRAI is, who it's for, strategic positioning | Understanding the project |
| [2. Architecture](#2-architecture) | Single binary, embedded NATS, Caddy model overview | Understanding the system design |
| [3. Caddy Model](#3-caddy-model-component-ecosystem) | How components are packaged, distributed, imported | Building the component ecosystem |
| [4. Template Repo](#4-template-repo) | The user's robot project structure | Creating gorai-robot-template |
| [5. Core Repo Changes](#5-core-repo-changes) | What changes in gorai/gorai | Modifying the core framework |
| [6. CLI Commands](#6-cli-commands) | gorai build, run, validate, component add/search | Modifying CLI commands |
| [7. Registry](#7-component-registry) | JSON registry for component discovery | Building registry infrastructure |
| [8. Component Authoring](#8-component-authoring-contract) | How to write a GoRAI component | Writing new components |
| [9. Dependency Injection](#9-dependency-injection) | How components receive dependencies at runtime | Understanding component wiring |
| [10. Embedded NATS](#10-embedded-nats) | NATS server embedded in binary | Already implemented |
| [11. RDL](#11-robot-definition-language-rdl) | Robot configuration format | Modifying config parsing |
| [12. PiCar-X Reference](#12-picar-x-reference-implementation) | Concrete example of the full pattern | Building the first external component |
| [13. Non-Go Services](#13-non-go-services) | Python/C++ as external NATS processes | Adding non-Go service support |
| [14. Implementation Phases](#14-implementation-phases) | Ordered work plan | Planning and sequencing work |
| [15. What NOT to Build](#15-what-not-to-build) | Explicit anti-requirements | Avoiding scope creep |

---

## 1. Vision

GoRAI is **"npm for robotics"** — a platform where you describe your robot in JSON, install the components you need, and build a single binary that runs on a Raspberry Pi.

**Target users:** Software engineers, makers, citizen scientists, and small teams who want real autonomy without ROS 2's complexity. They already own a Raspberry Pi and cheap COTS hardware kits (PiCar-X, custom boats, drones).

**Core promise:** Write JSON, get a binary, deploy to a Pi.

```
robot.json  -->  gorai build  -->  single binary  -->  scp to Pi  -->  running robot
```

**Strategic positioning:** GoRAI is to ROS 2 what Express.js is to Java EE. Opinionated, simple, fast to start, scales when you need it.

---

## 2. Architecture

### 2.1 Single Binary Model

Every GoRAI robot is a single Go binary with:
- The GoRAI runtime (component lifecycle, NATS messaging, dashboard)
- An embedded NATS server (with JetStream)
- Exactly the components the robot needs (compiled in via Go imports)

There are **no containers, no external services, no package managers, no runtime dependency resolution.** The binary runs on a Raspberry Pi with systemd. That's the entire deployment.

### 2.2 How It Works Today (Monolithic)

```
gorai/gorai (single repo)
├── cmd/gorai/main.go       # Blank imports ALL components
├── components/             # ALL component implementations
├── services/               # ALL service implementations
├── pkg/                    # Core framework
└── driver/                 # ALL hardware drivers

Problem: Every gorai binary contains every component.
         PiCar-X code ships in ORCA's binary.
         Users can't add custom components without forking the repo.
```

### 2.3 How It Should Work (Caddy Model)

```
gorai/gorai (core repo)
├── cmd/gorai/              # CLI commands (validate, run, build, component)
├── pkg/                    # Core framework (config, registry, robot, nats, etc.)
└── driver/                 # Core driver interfaces (i2c, gpio, spi, serial)
    (NO component implementations here except fakes for testing)

emergingrobotics/gorai-picarx       # PiCar-X components (separate Go module)
emergingrobotics/gorai-driver-hcsr04 # HC-SR04 ultrasonic (separate Go module)
emergingrobotics/gorai-driver-robothat # Robot HAT MCU (separate Go module)

user's robot project (from template)
├── main.go                 # Blank imports exactly the components needed
├── go.mod                  # Depends on gorai core + component modules
├── robot.json              # RDL describing the robot
└── components/             # User's custom components (local packages)

Result: Each binary contains only what it needs.
        Users add components with `go get` + a blank import.
        Custom components are just Go packages in the same repo.
```

### 2.4 Key Architectural Decisions

| Decision | Rationale |
|----------|-----------|
| Single binary, no containers | Simplicity. A Raspberry Pi runs one binary via systemd. No container runtime overhead, no image management. |
| Embedded NATS | Zero external dependencies. `gorai run robot.json` works without installing anything else. |
| Go modules for components | Go already solves versioning, checksums, caching, and dependency resolution. No custom package manager. |
| Blank imports as manifest | The Go import list IS the component list. Compile-time safety. No runtime surprises. |
| Component interfaces in core | Core defines interfaces (Motor, Servo, Sensor). Implementations live in external modules. |
| RDL references type+model | The JSON config doesn't know about Go modules. It says `"type": "servo", "model": "robot-hat-pwm-servo"`. The binary's compiled-in registry resolves that. |

---

## 3. Caddy Model (Component Ecosystem)

The Caddy web server solved the same problem: a core binary with a plugin ecosystem where users choose which plugins compile into their build. GoRAI adopts this pattern exactly.

### 3.1 The Mechanism

Every GoRAI component is a Go package with an `init()` function that calls `registry.RegisterComponent()`:

```go
package servo

import "github.com/emergingrobotics/gorai/pkg/registry"

func init() {
    registry.RegisterComponent("servo", "robot-hat-pwm-servo", New)
}
```

A user's `main.go` imports the packages they need. Each import triggers `init()`, which registers the component. The compiled binary has exactly what's needed.

```go
package main

import (
    "github.com/emergingrobotics/gorai/cmd/gorai"

    // These blank imports register components at init() time
    _ "github.com/emergingrobotics/gorai-picarx/servo"
    _ "github.com/emergingrobotics/gorai-picarx/motor"
    _ "github.com/emergingrobotics/gorai-driver-hcsr04"
)

func main() {
    gorai.Run()
}
```

### 3.2 Adding a Component

```bash
gorai component add sensor/hc-sr04
# 1. Looks up "sensor/hc-sr04" in the registry JSON
# 2. Finds Go module path: github.com/emergingrobotics/gorai-driver-hcsr04
# 3. Runs: go get github.com/emergingrobotics/gorai-driver-hcsr04@latest
# 4. Adds: _ "github.com/emergingrobotics/gorai-driver-hcsr04" to main.go
```

That's it. Standard Go tooling handles everything else.

### 3.3 Requirements

**REQ-CADDY-1: Blank imports are the component manifest.**
The user's `main.go` blank import list is the sole mechanism for declaring which components are compiled into the binary. No separate manifest file, no package.json, no lock file beyond `go.sum`.

**REQ-CADDY-2: Components are standalone Go modules.**
Each component (or family of related components) is a separate Go module with its own `go.mod`, version, and release cycle. Components MUST NOT live in the core `gorai/gorai` repo (except fakes for testing).

**REQ-CADDY-3: Self-registration via init().**
Every component package MUST call `registry.RegisterComponent()` or `registry.RegisterService()` in an `init()` function. No other registration mechanism is needed.

**REQ-CADDY-4: gorai.Run() entrypoint.**
The core repo MUST export a `gorai.Run()` function (or equivalent) that a user's `main.go` calls. This function parses CLI args, loads the RDL, starts embedded NATS, starts the robot, and handles signals. The user's main.go should be ~15 lines.

**REQ-CADDY-5: go build is the build system.**
`gorai build robot.json` MUST run `go build` to produce a native Go binary. It MUST NOT invoke podman, docker, or any container tooling. Cross-compilation uses `GOOS` and `GOARCH`.

**REQ-CADDY-6: Component interfaces stay in core.**
The core repo defines component interfaces (Motor, Servo, Sensor, Base, etc.) and the registry. External modules implement these interfaces. This allows compile-time type checking across module boundaries.

---

## 4. Template Repo

### 4.1 Structure

The template repo (`emergingrobotics/gorai-robot-template`) is a GitHub template repository. Users click "Use this template" or clone it.

```
gorai-robot-template/
├── go.mod                  # module github.com/username/my-robot
├── main.go                 # Imports gorai core + blank imports for components
├── robot.json              # Skeleton RDL (user edits this)
├── components/             # User's custom components (local Go packages)
│   └── .gitkeep
├── services/               # User's custom services (local Go packages)
│   └── .gitkeep
├── Makefile                # build, run, test, validate, deploy, clean
└── .gitignore              # bin/, .env, .envrc, *~, etc.
```

### 4.2 Template main.go

```go
package main

import (
    "github.com/emergingrobotics/gorai/cmd/gorai"

    // Built-in components from GoRAI core (minimal set)
    // _ "github.com/emergingrobotics/gorai/components/serial"

    // Third-party components (add with: gorai component add <name>)
    // _ "github.com/emergingrobotics/gorai-picarx"

    // Custom components (local to this repo)
    // _ "my-robot/components/my-custom-sensor"
)

func main() {
    gorai.Run()
}
```

### 4.3 Template Makefile

```makefile
ROBOT_CONFIG ?= robot.json
BINARY_NAME  ?= $(shell jq -r '.robot.name // "robot"' $(ROBOT_CONFIG))
TARGET       ?= $(shell go env GOOS)/$(shell go env GOARCH)

.PHONY: build run test validate clean deploy

build:
	gorai build $(ROBOT_CONFIG) -o bin/$(BINARY_NAME) --target $(TARGET)

run:
	gorai run $(ROBOT_CONFIG)

test:
	go test ./...

validate:
	gorai validate $(ROBOT_CONFIG)

clean:
	rm -rf bin/

deploy: build
	scp bin/$(BINARY_NAME) $(DEPLOY_HOST):~/
	ssh $(DEPLOY_HOST) "chmod +x ~/$(BINARY_NAME)"
```

### 4.4 Requirements

**REQ-TEMPLATE-1: GitHub template repo.**
`emergingrobotics/gorai-robot-template` MUST be a GitHub template repository that users can instantiate with "Use this template".

**REQ-TEMPLATE-2: Minimal, working out of the box.**
The template MUST build and run without modification (with an empty RDL that starts embedded NATS and the dashboard, no components).

**REQ-TEMPLATE-3: go.mod references gorai core.**
The template's `go.mod` MUST require the GoRAI core module (`github.com/emergingrobotics/gorai`).

**REQ-TEMPLATE-4: main.go is tiny.**
The template `main.go` MUST be under 20 lines (excluding comments). It imports `gorai/cmd/gorai`, has blank imports, and calls `gorai.Run()`.

---

## 5. Core Repo Changes

### 5.1 What Moves Out

The following MUST be moved from `gorai/gorai` to external modules. They MUST NOT remain in the core repo's active code (they may be archived).

| Current Location | New Location | Rationale |
|------------------|-------------|-----------|
| `components/serial/` | External module | Hardware-specific GPS driver |
| `components/input/keyboard/` | Keep in core or external | Could argue either way; keyboard is generic enough |
| `components/camera/v4l2` (in driver/) | External module | Requires CGo (V4L2 bindings) |
| `components/pwm/gpiod/` | External module | Requires libgpiod, platform-specific |
| `components/*/remote/` | Keep in core | Remote proxy components are core infrastructure |
| `services/control/*` | External module (gorai-picarx or similar) | Robot-specific control logic |
| `pkg/systemd/` | Archive or remove | Container service management, not needed for launch |

### 5.2 What Stays in Core

| Package | Purpose |
|---------|---------|
| `cmd/gorai/` | CLI: validate, run, build, component |
| `pkg/config/` | RDL parsing and validation |
| `pkg/registry/` | Component/service registration |
| `pkg/robot/` | Robot lifecycle, startup, shutdown |
| `pkg/nats/` | NATS client wrapper |
| `pkg/embeddednats/` | Embedded NATS server |
| `pkg/resource/` | Resource naming and interfaces |
| `pkg/topics/` | NATS topic conventions |
| `pkg/dashboard/` | Web dashboard |
| `pkg/mesh/` | Mesh service discovery |
| `pkg/node/` | Node lifecycle |
| `pkg/pub/`, `pkg/sub/` | NATS publisher/subscriber |
| `pkg/log/` | Structured logging |
| `pkg/tf/` | Transform frames |
| `pkg/validation/` | Config validation rules |
| `components/` | Interface definitions + fakes for testing only |
| `driver/` | Interface definitions only (i2c.Bus, gpio.Pin, etc.) |

### 5.3 New: gorai.Run() Entrypoint

**REQ-CORE-1: Export a Run() function.**
The core repo MUST export a function that the user's `main.go` calls. This function:
1. Parses CLI arguments (validate, run, build, component, version, help)
2. For `run`: loads RDL, starts embedded NATS, starts robot, handles signals
3. For `build`: validates RDL, runs `go build` with cross-compilation
4. For `validate`: loads and validates RDL
5. For `component`: searches/adds components

Location: `github.com/emergingrobotics/gorai/cmd/gorai` package, exported `Run()` function.

```go
// In the core repo: cmd/gorai/run.go
package gorai

// Run is the main entrypoint for robot projects.
// Call this from your main.go after blank-importing your components.
func Run() {
    if err := commands.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

**REQ-CORE-2: Clean component interfaces.**
Every component interface (Motor, Servo, Sensor, Base, etc.) MUST be in the core repo under `components/`. These are interface-only packages with no hardware implementations. Fake implementations for testing are acceptable.

**REQ-CORE-3: Clean driver interfaces.**
Every driver interface (I2C Bus/Device, GPIO Pin/Driver, SPI, Serial) MUST be in the core repo under `driver/`. Platform-specific implementations (linux gpiod, linux i2c) may stay in core under `driver/*/linux/` since they are OS-level, not hardware-specific.

**REQ-CORE-4: Registry validates at startup.**
When `gorai run` loads an RDL, it MUST check that every `type+model` pair referenced in the RDL has a registered constructor. If not, it MUST print an actionable error:
```
Error: component type "servo" model "robot-hat-pwm-servo" not found in registry.
       This component may need to be installed. Try:
         gorai component search servo
```

**REQ-CORE-5: Remove container build logic.**
The `gorai build` command MUST run `go build` to produce a native Go binary. All podman/docker/container build logic MUST be removed.

---

## 6. CLI Commands

### 6.1 gorai run

No change from current behavior. Loads RDL, starts embedded NATS, starts components, handles signals.

### 6.2 gorai build

**REQ-CLI-BUILD-1: gorai build = go build.**
`gorai build <config> [flags]` MUST:
1. Load and validate the RDL
2. Run `go build` in the current directory (the user's robot project)
3. Produce a single native Go binary
4. Support `--target <os/arch>` for cross-compilation (sets `GOOS`/`GOARCH`)
5. Support `-o <path>` for output binary path (default: robot name from RDL)

**REQ-CLI-BUILD-2: No container tooling.**
`gorai build` MUST NOT invoke podman, docker, or any container build tool.

**REQ-CLI-BUILD-3: Error if not in a Go module.**
If the current directory does not contain a `go.mod`, `gorai build` MUST fail with:
```
Error: not in a Go module. Run 'gorai build' from your robot project directory.
       See: https://gorai.dev/docs/getting-started
```

### 6.3 gorai validate

**REQ-CLI-VALIDATE-1: Registry-aware validation.**
`gorai validate <config>` MUST check that every component `type+model` in the RDL has a registered constructor (from compiled-in blank imports). Missing components MUST produce an actionable error suggesting `gorai component search`.

### 6.4 gorai component search

**REQ-CLI-SEARCH-1: Search the registry.**
`gorai component search <query>` MUST:
1. Fetch the registry JSON (from a well-known URL or local cache)
2. Search by name, type, model, tags, and description
3. Print results: name, description, Go module path, version

```
$ gorai component search servo
  servo/robot-hat-pwm-servo   SunFounder Robot HAT PWM servo       v0.1.0
  servo/dynamixel             Dynamixel smart servo (serial)        v0.2.1
  servo/lx16a                 LewanSoul LX-16A bus servo            v0.1.0
```

### 6.5 gorai component add

**REQ-CLI-ADD-1: go get + blank import.**
`gorai component add <name>` MUST:
1. Look up `<name>` in the registry to find the Go module path
2. Run `go get <module>@latest` (or specified version)
3. Add a blank import line to `main.go`
4. Run `go mod tidy`

**REQ-CLI-ADD-2: Edit main.go safely.**
The blank import MUST be added to the correct import block in `main.go`. If no suitable import block exists, one MUST be created. The file MUST be formatted with `gofmt` after editing.

**REQ-CLI-ADD-3: Support direct module paths.**
If `<name>` looks like a Go module path (contains a dot), use it directly instead of looking up the registry:
```bash
gorai component add github.com/acme-corp/gorai-driver-custom@v1.0.0
```

### 6.6 gorai component info

**REQ-CLI-INFO-1: Show component details.**
`gorai component info <name>` MUST show: description, Go module path, version, hardware requirements, configuration schema (if available), and installation instructions.

---

## 7. Component Registry

### 7.1 Structure

The registry is a JSON file in a GitHub repo (`emergingrobotics/gorai-registry`):

```json
{
  "version": "1",
  "components": {
    "servo/robot-hat-pwm-servo": {
      "module": "github.com/emergingrobotics/gorai-picarx/servo",
      "type": "servo",
      "model": "robot-hat-pwm-servo",
      "description": "SunFounder Robot HAT PWM servo (I2C MCU at 0x14)",
      "hardware": "i2c",
      "version": "v0.1.0",
      "tags": ["sunfounder", "robot-hat", "picar-x", "servo"]
    },
    "sensor/hc-sr04": {
      "module": "github.com/emergingrobotics/gorai-driver-hcsr04",
      "type": "sensor",
      "model": "hc-sr04",
      "description": "HC-SR04 ultrasonic distance sensor (GPIO)",
      "hardware": "gpio",
      "version": "v0.1.0",
      "tags": ["ultrasonic", "distance", "range", "gpio"]
    },
    "picarx": {
      "module": "github.com/emergingrobotics/gorai-picarx",
      "type": "bundle",
      "description": "SunFounder PiCar-X (all components: servo, motor, base, grayscale)",
      "version": "v0.1.0",
      "tags": ["sunfounder", "picar-x", "car", "differential-drive"]
    }
  }
}
```

### 7.2 Requirements

**REQ-REGISTRY-1: Static JSON file.**
The registry MUST be a JSON file in a GitHub repo, fetchable via raw URL. No API server needed for launch.

**REQ-REGISTRY-2: Local caching.**
`gorai component search` MUST cache the registry locally (e.g., `~/.gorai/registry.json`) and refresh periodically (e.g., every 24 hours or on explicit `--refresh`).

**REQ-REGISTRY-3: Registry is optional.**
If the registry is unavailable, `gorai component add` MUST still work with direct Go module paths.

---

## 8. Component Authoring Contract

This section defines what a component module author must do.

### 8.1 Module Structure

A component module is a standard Go module:

```
gorai-driver-hcsr04/
├── go.mod          # module github.com/emergingrobotics/gorai-driver-hcsr04
├── go.sum
├── hcsr04.go       # Driver implementation
├── hcsr04_test.go  # Tests
├── component.go    # GoRAI component wrapper + init() registration
└── component_test.go
```

### 8.2 Registration

**REQ-AUTHOR-1: init() registration is mandatory.**
Every component package MUST register itself in `init()`:

```go
func init() {
    registry.RegisterComponent("sensor", "hc-sr04", New)
}
```

### 8.3 Constructor Signature

**REQ-AUTHOR-2: Standard constructor.**
Component constructors MUST have the signature:

```go
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error)
```

Where:
- `ctx` — Cancellation context
- `deps` — Provides access to other components by name (e.g., the MCU driver)
- `conf` — Key-value config from the RDL `attributes` map, plus `name`, `type`, `model`

### 8.4 Interface Compliance

**REQ-AUTHOR-3: Implement the correct interface.**
A servo component MUST implement `servo.Servo`. A motor MUST implement `motor.Motor`. Etc. The returned `any` from the constructor is type-asserted by the runtime to check for `Startable`, `Closeable`, and `resource.Resource` interfaces.

### 8.5 Dependencies Between Components

**REQ-AUTHOR-4: Accept interfaces, not concrete types.**
Components that depend on other components (e.g., a servo depends on an MCU driver) MUST define a local interface for the dependency, NOT import the concrete type:

```go
// In gorai-picarx/servo/servo.go

// MCUDriver is what this servo needs from the Robot HAT MCU.
// Satisfied by gorai-driver-robothat.MCU.
type MCUDriver interface {
    SetPWMFrequency(ctx context.Context, channel int, hz float64) (float64, error)
    SetPWMPulseWidth(ctx context.Context, channel int, value uint16) error
    TimerPeriod(channel int) uint16
}
```

The MCU component is obtained via `deps.Get("robot-hat-mcu")` and type-asserted to this interface. This avoids a compile-time dependency on the MCU module, though a Go module `require` is acceptable when modules are in the same family.

### 8.6 Component Lifecycle

Components may implement these optional interfaces:

```go
// Called after construction (in dependency order)
type Startable interface {
    Start(ctx context.Context) error
}

// Called at shutdown (reverse dependency order)
type Closeable interface {
    Close(ctx context.Context) error
}

// Full resource interface (name, reconfigure, commands)
type Resource interface {
    Name() resource.Name
    Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error
    DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error)
    Close(ctx context.Context) error
}
```

---

## 9. Dependency Injection

### 9.1 How It Works

The RDL declares component dependencies via `depends_on`:

```json
{
  "components": [
    {
      "name": "robot-hat-mcu",
      "type": "i2c_device",
      "model": "sunfounder-robot-hat",
      "attributes": {"i2c_bus": "/dev/i2c-1", "address": "0x14"}
    },
    {
      "name": "steering",
      "type": "servo",
      "model": "robot-hat-pwm-servo",
      "depends_on": ["robot-hat-mcu"],
      "attributes": {"pwm_channel": 2, "min_angle": -30, "max_angle": 30}
    }
  ]
}
```

The runtime:
1. Topologically sorts components by `depends_on`
2. Creates `robot-hat-mcu` first
3. When creating `steering`, passes a `deps` object where `deps.Get("robot-hat-mcu")` returns the already-created MCU instance

### 9.2 Requirements

**REQ-DI-1: Topological sort.**
The runtime MUST create components in dependency order. Circular dependencies MUST be detected and rejected at validation time.

**REQ-DI-2: deps.Get() returns the component instance.**
`deps.Get(name)` MUST return the component instance (the `any` value from the constructor). The consuming component type-asserts it to the interface it needs.

**REQ-DI-3: NATS and logger always available.**
Every component MUST have access to the NATS connection and logger via `deps.Get("nats")` and `deps.Get("logger")`, regardless of `depends_on`.

**REQ-DI-4: Missing dependency is a startup error.**
If a component's `depends_on` references a name not present in the RDL, `gorai validate` MUST catch it. If a `depends_on` component fails to start, components that depend on it MUST NOT be started.

---

## 10. Embedded NATS

Already implemented. Summary of requirements (see git history for original spec):

| Requirement | Status |
|-------------|--------|
| REQ-EMBED-1: Default to embedded NATS for local URLs | Implemented |
| REQ-EMBED-2: Bind to 127.0.0.1 only | Implemented |
| REQ-EMBED-3: Start before any components | Implemented |
| REQ-EMBED-4: JetStream enabled by default | Implemented |
| REQ-EMBED-5: Shutdown after client close | Implemented |
| REQ-EMBED-6: Port from RDL | Implemented |
| REQ-EMBED-7: TLS passthrough | Implemented |
| REQ-EXTERNAL-1: `nats.external: true` skips embedding | Implemented |
| REQ-EXTERNAL-2: Non-local URL implies external | Implemented |

See `pkg/embeddednats/` and `pkg/robot/robot.go`.

---

## 11. Robot Definition Language (RDL)

### 11.1 No Changes to RDL Format

**REQ-RDL-1: RDL format is unchanged.**
The RDL JSON format does NOT change in the Caddy model transition. The same `robot.json` that works today MUST work in the new architecture. RDL references components by `type` + `model`, which are registry keys, NOT Go module paths.

### 11.2 Minimal Valid RDL

```json
{
  "version": "2",
  "robot": {"name": "my-robot"}
}
```

This starts embedded NATS and the dashboard. No components.

### 11.3 Full RDL Example (PiCar-X)

```json
{
  "version": "2",
  "robot": {
    "name": "picar-x",
    "description": "SunFounder PiCar-X on GoRAI"
  },
  "components": [
    {
      "name": "robot-hat-mcu",
      "type": "i2c_device",
      "model": "sunfounder-robot-hat",
      "attributes": {
        "i2c_bus": "/dev/i2c-1",
        "address": "0x14",
        "reset_pin": "MCURST"
      }
    },
    {
      "name": "steering",
      "type": "servo",
      "model": "robot-hat-pwm-servo",
      "depends_on": ["robot-hat-mcu"],
      "attributes": {
        "pwm_channel": 2,
        "min_angle": -30,
        "max_angle": 30
      }
    },
    {
      "name": "left-motor",
      "type": "motor",
      "model": "robot-hat-dc-motor",
      "depends_on": ["robot-hat-mcu"],
      "attributes": {
        "pwm_channel": 13,
        "direction_pin": "D4"
      }
    },
    {
      "name": "right-motor",
      "type": "motor",
      "model": "robot-hat-dc-motor",
      "depends_on": ["robot-hat-mcu"],
      "attributes": {
        "pwm_channel": 12,
        "direction_pin": "D5",
        "invert_direction": true
      }
    },
    {
      "name": "drive",
      "type": "base",
      "model": "ackermann-differential",
      "depends_on": ["left-motor", "right-motor", "steering"],
      "attributes": {
        "left_motor": "left-motor",
        "right_motor": "right-motor",
        "steering_servo": "steering"
      }
    },
    {
      "name": "ultrasonic",
      "type": "sensor",
      "model": "hc-sr04",
      "attributes": {
        "trigger_pin": "D2",
        "echo_pin": "D3"
      }
    },
    {
      "name": "line-sensor",
      "type": "sensor",
      "model": "robot-hat-grayscale",
      "depends_on": ["robot-hat-mcu"],
      "attributes": {
        "channels": [0, 1, 2]
      }
    }
  ],
  "dashboard": {"enabled": true}
}
```

---

## 12. PiCar-X Reference Implementation

The PiCar-X port is the first concrete test of the Caddy model. It validates the entire pattern. See [docs/picar-x-go.md](docs/picar-x-go.md) for the full Go implementation.

### 12.1 Module Split

| Module | Contents | Registration Keys |
|--------|----------|-------------------|
| `gorai-driver-robothat` | MCU I2C driver, board detection, pin maps, calibration | `"i2c_device", "sunfounder-robot-hat"` |
| `gorai-driver-hcsr04` | HC-SR04 GPIO ultrasonic driver + component | `"sensor", "hc-sr04"` |
| `gorai-picarx` | Servo, motor, base, grayscale components | `"servo", "robot-hat-pwm-servo"` / `"motor", "robot-hat-dc-motor"` / `"base", "ackermann-differential"` / `"sensor", "robot-hat-grayscale"` |

### 12.2 User's main.go for PiCar-X

```go
package main

import (
    "github.com/emergingrobotics/gorai/cmd/gorai"

    // PiCar-X (registers all: servo, motor, base, grayscale, MCU)
    _ "github.com/emergingrobotics/gorai-picarx"

    // HC-SR04 ultrasonic (separate because it's not PiCar-X-specific)
    _ "github.com/emergingrobotics/gorai-driver-hcsr04"
)

func main() {
    gorai.Run()
}
```

### 12.3 Build and Deploy

```bash
gorai build robot.json -o picarx --target linux/arm64
scp picarx pi@raspberrypi:~
ssh pi@raspberrypi ./picarx
```

---

## 13. Non-Go Services

Some workloads (vision, ML, SLAM) are best written in Python or C++. These are NOT compiled into the Go binary. They run as separate processes that communicate via NATS.

### 13.1 How It Works

An RDL can reference an external service:

```json
{
  "services": [
    {
      "name": "detector",
      "type": "vision",
      "model": "yolo",
      "external": {
        "enabled": true,
        "command": "/opt/gorai-vision-yolo/detect.py",
        "restart": "on-failure"
      }
    }
  ]
}
```

The GoRAI runtime starts the external process and passes `NATS_URL` as an environment variable. The external process connects to NATS and communicates via pub/sub.

### 13.2 Requirements

**REQ-EXTERNAL-SVC-1: External services are processes, not containers.**
External services are child processes started by the GoRAI runtime. They are NOT containers. The command must be an absolute path to an executable.

**REQ-EXTERNAL-SVC-2: NATS is the integration boundary.**
External services communicate with the Go binary exclusively via NATS. There is no shared memory, no IPC, no socket files. NATS topics are the only contract.

**REQ-EXTERNAL-SVC-3: External services are optional.**
The core platform works without any external services. External services add capabilities (vision, ML) but are not required for motor control, sensors, or navigation.

---

## 14. Implementation Phases

### Phase 1: Core Repo Cleanup (Prerequisites)

1. Export `gorai.Run()` entrypoint from `cmd/gorai` package
2. Rewrite `gorai build` to run `go build` (done)
3. Remove container build logic from `build.go` (done)
4. Move hardware-specific component implementations to archive (serial/gps, pwm/gpiod, etc.)
5. Ensure component interfaces are clean and importable from external modules
6. Ensure `registry.RegisterComponent` and `registry.RegisterService` work correctly for external callers
7. Add registry-aware validation to `gorai validate` (suggest `gorai component search` for missing components)

### Phase 2: Template Repo

1. Create `gorai-robot-template` repo
2. Create template `main.go` that calls `gorai.Run()`
3. Create template `go.mod` referencing gorai core
4. Create template `Makefile` with build/run/test/validate/deploy targets
5. Create skeleton `robot.json`
6. Verify: clone template, `make build`, `make run` produces a working (empty) robot

### Phase 3: First External Component Modules

1. Create `gorai-driver-robothat` — Robot HAT MCU I2C driver
2. Create `gorai-driver-hcsr04` — HC-SR04 ultrasonic
3. Create `gorai-picarx` — PiCar-X components (depends on robothat driver)
4. Verify: user clones template, adds PiCar-X imports, builds, deploys to Pi

### Phase 4: Component CLI

1. Create `gorai-registry` repo with JSON registry file
2. Implement `gorai component search` — fetches and queries registry
3. Implement `gorai component add` — runs `go get` + edits `main.go`
4. Implement `gorai component info` — shows component details
5. Verify: `gorai component add picarx` works end-to-end

### Phase 5: Existing Components as External Modules

1. Extract GPS sensor to external module
2. Extract keyboard input to external module
3. Extract camera/V4L2 to external module
4. Extract mecanum/L298N control services to external modules
5. Core repo main.go becomes minimal (only remote proxies + fakes)

---

## 15. What NOT to Build

| Temptation | Why Not |
|-----------|---------|
| Custom package manager | Go modules already solves versioning, checksums, caching |
| Container-based component distribution | Adds container runtime dependency for a problem Go modules already solves |
| Git submodules | Terrible DX, merge conflicts, version confusion |
| Monorepo of all components | Creates coupling, slows iteration, every component must release together |
| Language-agnostic plugin system (WASM, CGo, etc.) | The Go binary boundary IS the plugin system. Non-Go code uses NATS. |
| `gorai init` wizard | The template repo is simpler and more transparent. Add later if needed. |
| gorai-component.yaml metadata file | Not needed for launch. The registry JSON is sufficient for discovery. Component metadata in Go code (comments, struct tags) is sufficient for documentation. |
| Automated non-Go service installation | `pip install` instructions in README are fine for now. Automated polyglot installation is Phase 2+ complexity. |
| Container build support | `gorai build` means `go build`. Period. If users need containers for external services, they use `podman build` directly. |

---

## Appendix A: Glossary

| Term | Definition |
|------|-----------|
| **Caddy model** | Component packaging pattern from the Caddy web server: blank imports in main.go trigger init() registration. The import list is the component manifest. |
| **RDL** | Robot Definition Language. The `robot.json` configuration file that describes a robot's components, services, and settings. |
| **Blank import** | A Go import prefixed with `_` (e.g., `_ "github.com/foo/bar"`). Executes the package's `init()` function without importing any names. |
| **Component** | A Go package that implements a GoRAI interface (Motor, Servo, Sensor, etc.) and registers itself via `init()`. Compiled into the binary. |
| **External service** | A non-Go process (Python, C++) that communicates with the GoRAI binary via NATS. Started as a child process, not compiled in. |
| **Registry** | A JSON file mapping component names to Go module paths. Used by `gorai component search/add`. |
| **Template repo** | `gorai-robot-template`. The starting point for every robot project. A minimal Go module with main.go + robot.json. |
| **Core repo** | `gorai/gorai`. Contains the runtime, CLI, interfaces, embedded NATS. Does NOT contain hardware-specific component implementations. |

## Appendix B: File Quick Reference

For agents working on specific files:

| File | What It Does | When to Edit |
|------|-------------|-------------|
| `cmd/gorai/main.go` | CLI entry point with blank imports | Moving components to external modules |
| `cmd/gorai/commands/root.go` | CLI command dispatch | Adding new CLI commands |
| `cmd/gorai/commands/build.go` | `gorai build` implementation | Changing build behavior |
| `cmd/gorai/commands/run.go` | `gorai run` implementation | Changing runtime behavior |
| `cmd/gorai/commands/component.go` | `gorai component` subcommands | Implementing search/add |
| `pkg/registry/registry.go` | Component/service registration | Changing registration API |
| `pkg/config/config.go` | RDL structs and parsing | Changing config format |
| `pkg/robot/robot.go` | Robot lifecycle (start, stop, NATS) | Changing startup sequence |
| `pkg/embeddednats/server.go` | Embedded NATS wrapper | Changing NATS behavior |
| `pkg/resource/resource.go` | Resource interface definitions | Changing component contracts |
| `components/*/` | Component interface definitions | Adding new component types |
| `driver/*/` | Driver interface definitions | Adding new driver types |
