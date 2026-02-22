# Gorai

**The robotics platform for the AI era.**

Gorai is a Go-based robotics framework designed for makers, citizen scientists, students, and small organizations who need real autonomy without ROS 2's complexity. Build a robot in under an hour with a single binary and NATS messaging.

> **Documentation has moved to [../gorai-docs](https://github.com/emergingrobotics/gorai-docs)** — all strategy, specifications, architecture docs, book content, plans, and guides are in the gorai-docs repository. This repo focuses on design and implementation of the core gorai system.

## What This Repo Contains

This repository is the **core implementation** — Go source code, component interfaces, drivers, services, CLI, runtime, and build system.

## Core Architecture

### Current Phase: Simple Binary Deployment

```
┌─────────────────────────────────────────────────────────────────┐
│                    User Experience                               │
│                                                                  │
│   robot.json (RDL)  →  gorai run  →  Robot running              │
│                                                                  │
│   Users work with: Robot Definition Language (JSON)             │
│   Output: Single binary (~10-20MB)                              │
└─────────────────────────────────────────────────────────────────┘
```

- **Single binary** — No containers, no K8s, just a Go binary
- **NATS messaging** — All component communication via NATS pub/sub
- **Go core** — Components, behaviors, runtime all in Go
- **Simple deployment** — Copy binary to Pi, run with systemd
- **Low overhead** — ~20-50MB RAM (vs 512MB+ for containers)

## Related Modules

Gorai works with companion modules for hardware communication:

### gorai-gsp (../gorai-gsp)
Go/TinyGo library implementing the **Gorai Serial Protocol v2 (GSP/2)**:
- Transport-agnostic binary protocol for host-device communication
- 40+ message types for PWM, motors, encoders, IMU, sensors, GPIO
- Works over UART, UDP, or radio links

### rp2040-pwm (../rp2040-pwm)
TinyGo firmware for RP2040-based boards providing:
- 16-channel hardware PWM for servos/ESCs
- USB serial interface using GSP/2

---

## CLI Commands

```bash
# Core commands
gorai validate <config>   # Validate RDL configuration file
gorai run <config>        # Run robot in development mode (foreground)
gorai build <config>      # Build standalone binary for deployment

# Component management
gorai components          # List available component types
gorai component search    # Search for third-party components
gorai component info      # Show component information
gorai component add       # Add component to project

# Mesh service discovery
gorai mesh services       # List running services in the mesh
gorai mesh channels       # List registered NATS channels
gorai mesh schemas        # List or show message schemas
gorai mesh watch          # Watch for services joining/leaving
gorai mesh summary        # Show mesh state summary
gorai mesh robots         # List robots with registered services
gorai mesh init           # Initialize mesh with predefined schemas
gorai mesh reset          # Reset mesh (delete all data)

# Utility commands
gorai version             # Print version information
gorai migrate             # Migrate RDL v1 config to v2 format
gorai help                # Print help message
```

---

## Code Structure

```
gorai/
├── api/                    # API definitions
│   └── proto/              # Protobuf definitions
├── cmd/gorai/              # CLI entry point
│   └── commands/           # CLI command implementations
├── components/             # Component type interfaces and implementations
│   ├── arm/                # Robotic arm
│   ├── base/               # Mobile base (differential drive, etc.)
│   ├── camera/             # Camera capture
│   ├── gripper/            # Gripper/end-effector
│   ├── input/              # Input devices (joystick, gamepad)
│   ├── link/               # Kinematic link
│   ├── motor/              # DC/brushless motor
│   ├── power/              # Power management
│   ├── pwm/                # PWM output
│   ├── sensor/             # Generic sensor
│   ├── serial/             # Serial port
│   ├── servo/              # Servo motor
│   ├── space/              # Spatial/coordinate frame
│   ├── stepper/            # Stepper motor
│   ├── thruster/           # Thruster (ROV/drone)
│   └── valve/              # Valve actuator
├── driver/                 # Hardware drivers
│   ├── camera/             # Camera drivers
│   ├── gpio/               # GPIO pin access
│   ├── i2c/                # I2C bus
│   ├── pwm/                # PWM hardware
│   ├── serial/             # Serial/UART
│   └── spi/                # SPI bus
├── examples/               # Example robots
│   ├── blinky/             # LED blink example (RDL)
│   ├── gps-tracker/        # GPS tracking example (RDL)
│   ├── hello-camera/       # Camera streaming example (RDL)
│   └── pwm-controller/     # PWM control via gorai-gsp (Go)
├── images/                 # Project images and assets
├── internal/               # Internal packages (not importable)
│   ├── proto/              # Generated protobuf Go code
│   └── testutil/           # Test helpers
├── nws/                    # NATS WebSocket bridge
├── pkg/                    # Core libraries
│   ├── accel/              # ML acceleration
│   ├── components/         # Component registry and lifecycle
│   ├── config/             # RDL parsing and validation
│   ├── dashboard/          # Web dashboard
│   ├── discovery/          # Service discovery
│   ├── gsp/                # Gorai Serial Protocol client
│   ├── hardware/           # Hardware abstraction
│   ├── log/                # Structured logging
│   ├── mesh/               # Mesh service discovery (NATS KV)
│   ├── nats/               # NATS client wrapper
│   ├── node/               # Robot node lifecycle
│   ├── param/              # Parameter server
│   ├── proxy/              # Component proxy (remote access)
│   ├── pub/                # NATS publisher helpers
│   ├── registry/           # Component/service registry
│   ├── resource/           # Resource naming and management
│   ├── robot/              # Robot instance orchestration
│   ├── services/           # Service registry and lifecycle
│   ├── sub/                # NATS subscriber helpers
│   ├── systemd/            # Systemd unit file generation
│   ├── tf/                 # Transform/coordinate frames
│   ├── topics/             # NATS topic conventions
│   └── validation/         # Config validation rules
├── scripts/                # Shell scripts (wrapper, start/stop)
├── services/               # Service implementations
│   ├── behavior/           # Behavior trees / state machines
│   ├── bridge/             # Protocol bridge
│   ├── control/            # Control loops (PID, etc.)
│   ├── coordinator/        # Multi-component coordination
│   ├── formatter/          # Data formatting
│   ├── gateway/            # External API gateway
│   ├── mlmodel/            # ML model serving
│   ├── motion/             # Motion planning
│   ├── navigation/         # Autonomous navigation
│   ├── slam/               # SLAM (mapping/localization)
│   ├── telemetry/          # Metrics and telemetry
│   └── vision/             # Computer vision pipelines
├── templates/              # Code generation templates
│   ├── component/          # Component scaffolding templates
│   └── service/            # Service scaffolding templates
├── tools/                  # Development tools
│   └── pwm-ramp-test/      # PWM testing tool
└── archive/                # Archived code for future phases
```

---

## Design Principles

1. **Simple binary deployment** — No containers required for basic robots
2. **NATS-based messaging** — All component communication via NATS pub/sub
3. **Progressive complexity** — Start simple, add containers/K3s when needed
4. **Go-first, pragmatic polyglot** — Go core, Python/C++ via NATS (future)
5. **Cloud-native patterns** — NATS, Prometheus, JetStream (event sourcing)

---

## Key Design Decisions

### Language Strategy
| Component Type | Language | Rationale |
|----------------|----------|-----------|
| Framework core | Pure Go | Concurrency, deployment, AI-assisted coding |
| Simple sensors | Pure Go | GPIO, I2C, GPS, IMU — protocol parsing |
| Vision preprocessing | Python | OpenCV ecosystem (future) |
| ML inference | Python or ONNX Runtime | PyTorch training; ONNX deployment (future) |
| Camera drivers | cgo wrappers | V4L2, RealSense SDKs in C/C++ |
| Web UI | Go templates + HTMX | Avoid separate JS frontend complexity |

### Satellite Repository Pattern
Code requiring CGo or platform-specific dependencies goes in satellite repos:
- `gorai-driver-*` — Hardware drivers with CGo
- `gorai-accel-*` — Accelerator backends (Coral, CUDA, Rockchip)
- `gorai-service-*` — Complex services
- `gorai-tiny-*` — TinyGo microcontroller code

### Registration Pattern
All implementations use self-registration via `init()`:
```go
func init() {
    registry.RegisterComponent("camera", "v4l2", New)
}
```

### Component Status Value Standard

Every component implementing `resource.Sensor` MUST include two keys in its `Readings()` return map and in any data published to NATS `.data` topics. Services publishing heartbeats SHOULD include them.

#### Required Keys

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `status_value` | varies | YES | The headline value for dashboard display |
| `status_value_type` | string | YES | One of: `"binary"`, `"number"`, `"string"` |
| `status_value_unit` | string | NO | Unit suffix for number type (e.g., `"W"`, `"kWh"`) |

#### Status Value Types

| Type | Go type | Display | Example |
|------|---------|---------|---------|
| `binary` | string (`"on"` / `"off"`) | Badge: green "ON" / gray "OFF" | Switch components |
| `number` | float64 | Formatted with unit suffix | Power meter: `1234.5 W` |
| `string` | string | Text as-is | Service health: `"healthy"` |

#### Examples

Switch component:
```go
readings["status_value"] = s.state      // "on" or "off"
readings["status_value_type"] = "binary"
```

Power meter:
```go
readings["status_value"] = pacW          // current AC power
readings["status_value_type"] = "number"
readings["status_value_unit"] = "W"
```

Service heartbeat:
```go
msg.StatusValue = "healthy"
msg.StatusValueType = "string"
```

The dashboard subscribes to `gorai.<robot>.*.data` and extracts these fields to display inline next to each component name, replacing the generic "active" label.

### Configuration Hot-Reload Standard

Gorai supports live reloading of `robot.json` while the robot is running. The robot runtime watches the config file for changes and applies safe updates without restart.

#### Structural vs Parameter Changes

Config changes fall into two categories:

| Category | Examples | Behavior |
|----------|----------|----------|
| **Structural** | Adding/removing components or services, changing component type/model, renaming components/services, changing NATS URL | **Rejected**. Log error, keep running with current config. |
| **Parameter** | Schedule offsets, poll intervals, brightness defaults, latitude/longitude, forecast intervals, retry counts, descriptions | **Applied**. Parse and validate new config, then push updated attributes to affected components/services. |

#### Invariants

1. **Parse before apply**: The new config file MUST parse and validate successfully before any changes are applied. If `config.Load()` fails, the robot keeps running with the current config and logs the error.
2. **Structural diff check**: After successful parse, the robot compares the new config's structural shape (component names/types/models, service names/types/models, NATS config) against the running config. If any structural field changed, the entire reload is rejected with a descriptive error log. No partial application.
3. **Attribute-only updates**: Only `attributes` maps within components and services are eligible for hot-reload. Top-level fields (`robot.name`, `nats.url`, `dashboard.listen`, `log.level`) are structural.
4. **Disabled flag changes are structural**: Enabling or disabling a component/service requires lifecycle management (start/stop) and is treated as structural.

#### Robot Runtime Behavior (`pkg/robot`)

The `Robot` struct watches `robot.json` using filesystem notifications (e.g., `fsnotify`). On file change:

1. **Debounce**: Wait 500ms after last write event to handle editors that write in multiple steps.
2. **Load**: Call `config.Load(path)` on the new file. If it fails, log error and return.
3. **Structural diff**: Compare new config against `r.cfg`:
   - Component count, names, types, models, disabled flags must match exactly.
   - Service count, names, types, models, disabled flags must match exactly.
   - NATS config must match.
   - Robot name/namespace must match.
   - If any mismatch: log `"config reload rejected: structural change detected"` with details of what changed. Return without applying.
4. **Attribute diff**: For each component and service, compare `Attributes` maps. Build a list of `(name, newAttributes)` pairs where attributes differ.
5. **Notify**: For each changed component/service, call `Reconfigure(ctx, deps, newConf)` on the running instance (if it implements `resource.Resource`). The component/service is responsible for applying the new attributes and adjusting behavior.
6. **Update stored config**: Replace `r.cfg` with the new config so subsequent operations use the latest values.
7. **Publish event**: Publish a `config_reloaded` event to NATS with the list of updated component/service names.

#### Component/Service Reconfigure Contract

Every component and service that supports hot-reload MUST implement `resource.Resource.Reconfigure()` with these semantics:

- **Idempotent**: Calling Reconfigure with identical config is a no-op.
- **Non-destructive**: Reconfigure MUST NOT stop polling, drop connections, or lose state. It updates internal parameters and adjusts behavior on the next cycle.
- **Thread-safe**: Reconfigure may be called concurrently with polling/processing goroutines. Use appropriate locking.
- **Error reporting**: Return an error if the new attributes are invalid. The robot logs the error but does not roll back other successful reconfigurations.

#### NATS Notification

On successful config reload, the robot publishes to `gorai.<robot>.system.config_reloaded`:

```json
{
  "timestamp": "2026-02-22T10:30:00Z",
  "updated_components": ["plug_a", "bulb_a"],
  "updated_services": ["pool-lights", "bugs"],
  "rejected": false
}
```

On rejected reload (structural change):

```json
{
  "timestamp": "2026-02-22T10:30:00Z",
  "rejected": true,
  "reason": "structural change: component 'new_plug' added"
}
```

### Startup-as-Reload

On startup, after all components and services are initialized and running, the robot MUST treat the loaded configuration as if it were a hot-reload event. This means calling `Reconfigure()` on every component and service with their current config attributes.

#### Purpose

When a robot starts (or restarts), the config file may contain parameter values that imply the robot should be in a specific state. For example, a light controller schedule may indicate that lights should currently be on based on the time of day and schedule offsets. Without startup-as-reload, the robot would start "cold" and wait for the next trigger event, potentially leaving devices in the wrong state for hours.

#### Startup Behavior

1. **Initialize**: Start all components and services normally (constructors, NATS connections, initial state).
2. **Trigger Reconfigure**: After all components and services are running, call `Reconfigure(ctx, deps, conf)` on every component and service with their current config attributes. This is identical to what happens during a hot-reload when attributes change.
3. **Services check state**: Each service's `Reconfigure()` implementation MUST evaluate the current state of its managed devices against the desired state implied by the config parameters and the current time. If a corrective action is needed (e.g., turn on a light that should be on), the service executes it immediately.
4. **Components apply defaults**: Each component's `Reconfigure()` applies any parameter defaults (e.g., poll intervals, timeouts) and begins operating with those values.

#### Structural Changes at Startup

Since startup-as-reload uses the same `Reconfigure()` path, and the "old" and "new" configs are identical (both are the loaded config), no structural diff is performed. There is no "previous" config to compare against at initial startup. The robot simply calls `Reconfigure()` on each resource with its own config.

#### Invariants

- **No double-initialization**: Components and services MUST be fully initialized before the startup Reconfigure call. The Reconfigure is additive — it adjusts parameters and checks state, it does not re-initialize.
- **Idempotent**: Since Reconfigure is required to be idempotent, calling it with the same config the component was just constructed with MUST be a safe no-op for parameters. The key difference is the state-checking behavior: services use Reconfigure as a trigger to evaluate whether corrective actions are needed.
- **Error tolerance**: If a component's startup Reconfigure fails, log the error and continue with other components. The component is still running with its constructor-provided config.

---

## Development

```bash
make build              # Build CLI binary
make build-examples     # Build all example binaries
make validate-examples  # Validate all RDL configs
make test               # Run unit tests
make test-all           # All tests (unit, component, integration, module, system)
make check              # Run all checks (fmt, vet, lint, test)
make nats-start         # Start local NATS server
```

---

## Documentation (in gorai-docs)

All documentation has moved to [../gorai-docs](https://github.com/emergingrobotics/gorai-docs):

| Category | Location in gorai-docs |
|----------|----------------------|
| Strategy & Vision | `docs/overview/` |
| Architecture & Design | `docs/architecture/` |
| Technical Specifications | `docs/specifications/` |
| Hardware Analysis | `docs/hardware/` |
| Setup & Installation Guides | `docs/guides/` |
| Ecosystem Components | `docs/ecosystem/` |
| Implementation Plans | `docs/plans/` |
| Book Chapters | `docs/book/chapters/` |
| Project Definitions | `docs/projects/` |
| Website | `website/` |

---

**Pronunciation:** "go-ray" (like "sting-ray")
