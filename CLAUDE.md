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

# Process-compose deployment
gorai compile <config>    # Compile RDL to process-compose.yaml
gorai up <config>         # Compile and run via process-compose (alias for run --compose)

# Device management
gorai device reset        # Send GSP/2 reset command to a device via NATS

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
