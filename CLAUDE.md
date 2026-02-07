# Gorai

**The robotics platform for software engineers.**

Gorai is a Go-based robotics framework designed for makers, citizen scientists, students, and small organizations who need real autonomy without ROS 2's complexity. Build a robot in under an hour with a single binary and NATS messaging.

> **Documentation has moved to [../gorai-docs](../gorai-docs)** — all strategy, specifications, architecture docs, book content, plans, and guides are in the gorai-docs repository. This repo focuses on design and implementation of the core gorai system.

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
gorai validate <config>   # Validate RDL file
gorai run <config>        # Run robot (development mode)
gorai build <config>      # Build standalone binary
gorai components          # List available component types
gorai version             # Show version info

# Mesh service discovery
gorai mesh services       # List running services
gorai mesh channels       # List registered NATS channels
gorai mesh schemas        # List or show message schemas
gorai mesh watch          # Watch for services joining/leaving
gorai mesh summary        # Show mesh state summary
gorai mesh init           # Initialize predefined schemas
```

---

## Code Structure

```
gorai/
├── cmd/gorai/              # CLI commands
│   └── commands/
│       ├── mesh.go         # Mesh service discovery CLI
│       └── ...
├── pkg/                    # Core libraries
│   ├── accel/              # ML acceleration
│   ├── config/             # RDL parsing
│   ├── mesh/               # Service discovery (NATS KV)
│   │   ├── client.go       # Main client interface
│   │   ├── registration.go # Service registration + heartbeat
│   │   ├── discovery.go    # Query services and channels
│   │   ├── watcher.go      # Watch for changes
│   │   ├── schema.go       # Schema registry
│   │   └── micro.go        # NATS micro service API
│   ├── nats/               # NATS client
│   ├── runtime/            # Robot lifecycle
│   └── dashboard/          # Web dashboard
├── components/             # Component interfaces
├── driver/                 # Hardware drivers (GPIO, I2C, serial)
├── services/               # Service implementations
├── examples/               # Example robots
│   ├── blinky/             # LED blink example (RDL)
│   ├── gps-tracker/        # GPS tracking example (RDL)
│   ├── hello-camera/       # Camera streaming example (RDL)
│   └── pwm-controller/     # PWM control via gorai-gsp (Go)
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

All documentation has moved to [../gorai-docs](../gorai-docs):

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
