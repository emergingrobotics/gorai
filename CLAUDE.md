# Gorai

**Professional robotics for prosumers — without the PhD**

> **Building a new component or service?** Skip to [docs/LLM-DESIGN-GUIDE.md](docs/LLM-DESIGN-GUIDE.md) for complete templates, patterns, and checklists.

Gorai is a Go-based robotics framework designed for makers, citizen scientists, students, and small organizations who need real autonomy without ROS 2's complexity. Build a robot in under an hour with a single binary and NATS messaging.

## Strategic Positioning

**Target Market**: Prosumers who find Arduino too limiting and ROS 2 too complex.

**Not competing with ROS 2** in enterprise/research. We're "ROS 2 for prosumers," not "ROS 2 killer."

### What Gorai Is
- **Prosumer robotics framework** — real autonomy between educational toys ($100-300) and enterprise platforms ($4,000+)
- **Simple binary deployment** — single Go binary, NATS messaging, no containers required
- **Go-first, pragmatically polyglot** — Go core, Python/C++ services via NATS when appropriate

### What Gorai Is NOT
- **Not a ROS 2 replacement** — different markets (prosumer vs. enterprise/research)
- **Not language-purist** — we use the best tool for each job
- **Not enterprise-focused** — we optimize for accessibility, not feature completeness

---

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

### Future Phases

- **Phase 2:** Optional containers for ML/vision services (Python/C++)
- **Phase 3:** K3s orchestration for fleet management

See [docs/archive/future-state/](docs/archive/future-state/) for preserved future designs.

---

## Related Modules

Gorai works with companion modules for hardware communication:

### gorai-gsp (../gorai-gsp)
Go/TinyGo library implementing the **Gorai Serial Protocol v2 (GSP/2)**:
- Transport-agnostic binary protocol for host-device communication
- 5-byte header with version, flags, type, sequence number, and length
- CRC-16-CCITT error detection
- Bidirectional with selective ACK
- 40+ message types for PWM, motors, encoders, IMU, sensors, GPIO
- Works over UART, UDP, or radio links

### rp2040-pwm (../rp2040-pwm)
TinyGo firmware for RP2040-based boards providing:
- 16-channel hardware PWM for servos/ESCs
- USB serial interface using GSP/2
- Configurable failsafe and pulse limits
- Persistent configuration in flash

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

## Mesh Service Discovery

The mesh system enables runtime service discovery across independent processes using NATS KV.

### Key Features
- **Runtime Registration**: Services register themselves at startup with automatic heartbeat
- **Channel Discovery**: Find available NATS subjects and their message schemas
- **Cross-Binary Discovery**: Independent processes discover each other's services
- **Health Monitoring**: Automatic TTL-based expiry for stale services
- **Schema Registry**: JSON Schema definitions for message types

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         NATS JetStream KV                                │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  gorai-services (TTL: 30s)     → Active service registrations   │    │
│  │  gorai-channels (persistent)   → Channel/subject descriptors    │    │
│  │  gorai-schemas  (persistent)   → Message schemas (JSON Schema)  │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│  Well-Known Subjects:                                                   │
│  ├── gorai.mesh.announce         → Service join/leave announcements    │
│  └── gorai.mesh.heartbeat.<id>   → Per-service heartbeats              │
└─────────────────────────────────────────────────────────────────────────┘
```

### Usage Example

```go
// Register a service
client, _ := mesh.NewClient(natsConn)
reg, _ := client.Register(ctx, mesh.ServiceDescriptor{
    Name:    "motor-controller",
    Type:    mesh.TypeComponent,
    Subtype: "motor",
    Model:   "pwm",
    RobotID: "robot-alpha",
    Publishes: []string{"gorai.robot-alpha.motor.state"},
})
defer reg.Deregister()

// Discover services
motors, _ := client.FindServices(ctx, mesh.Query{
    RobotID: "robot-alpha",
    Subtype: "motor",
})
```

See [specs/mesh-service-discovery.md](specs/mesh-service-discovery.md) for complete specification.

---

## Dynamic Discovery

Gorai supports **hybrid static/dynamic** configuration where RDL defines structure and rules, while devices are discovered at runtime.

### Key Concepts

1. **Gateways** — Bridge hardware protocols (GSP/2, Modbus) to NATS, register devices in mesh
2. **Discovery Sources** — Where to find devices (gateways, mesh queries)
3. **Adoption Rules** — How to map discovered capabilities to component types
4. **Dynamic Dependencies** — Services that depend on discovered resources (`@discovered:motor/*`)
5. **Proxy Components** — Wrappers that make remote devices look like local components

### RDL Example

```json
{
  "gateways": [
    {
      "name": "usb-gateway",
      "type": "gateway/gsp",
      "config": {"discovery": {"enabled": true, "patterns": ["/dev/ttyACM*"]}}
    }
  ],
  "discovery": {
    "enabled": true,
    "rules": [
      {"match": {"capability": "PWM"}, "adopt_as": {"type": "motor"}}
    ]
  },
  "services": [
    {"name": "patrol", "depends_on": ["@discovered:motor/*"]}
  ]
}
```

### Runtime Flow

```
Gateway discovers device → Registers in mesh → Discovery manager adopts →
Proxy component created → Service dependencies resolve → Robot operates
```

See [specs/dynamic-discovery.md](specs/dynamic-discovery.md) for complete specification.

---

## Hardware Platforms

| Platform | Role | AI Performance |
|----------|------|----------------|
| **Raspberry Pi 5 (8GB)** | Primary | External (Hailo 13-26 TOPS) |
| **Raspberry Pi 5 (4GB)** | Budget | External (Hailo) |
| **Orange Pi 5B (8GB)** | Budget AI | 6 TOPS (built-in) |

**Not supported:** Pi 3, Pi Zero, Pi 4 (2GB)

---

## Key Documentation

### Strategy & Vision
- [docs/STRATEGIC-SUMMARY.md](docs/STRATEGIC-SUMMARY.md) — **Key strategic decisions and positioning**
- [docs/vision-analysis.md](docs/vision-analysis.md) — Comprehensive strategic analysis
- [docs/FUTURE-ROADMAP.md](docs/FUTURE-ROADMAP.md) — Container/K3s expansion plans

### Specifications
- [specs/gorai-framework-specification.md](specs/gorai-framework-specification.md) — **Complete technical spec**
- [specs/robot-definition-language.md](specs/robot-definition-language.md) — RDL JSON configuration format
- [specs/code-organization.md](specs/code-organization.md) — **Module structure and satellite repos**
- [specs/mesh-service-discovery.md](specs/mesh-service-discovery.md) — **Runtime service discovery via NATS KV**
- [specs/dynamic-discovery.md](specs/dynamic-discovery.md) — **Auto-adoption and dynamic dependencies**
- [specs/gsp-v2-protocol.md](specs/gsp-v2-protocol.md) — Gorai Serial Protocol specification
- [specs/hardware-requirements.md](specs/hardware-requirements.md) — Hardware specs
- [specs/serial-interfaces.md](specs/serial-interfaces.md) — Serial communication patterns
- [specs/runtime.md](specs/runtime.md) — Robot lifecycle and runtime
- [specs/testing-approach.md](specs/testing-approach.md) — Testing strategy

### Architecture & Design
- [docs/PACKAGE-LOCATIONS.md](docs/PACKAGE-LOCATIONS.md) — Where code belongs
- [docs/LLM-DESIGN-GUIDE.md](docs/LLM-DESIGN-GUIDE.md) — **Complete guide for LLMs designing components/services**
- [docs/hardware-abstraction.md](docs/hardware-abstraction.md) — Hardware abstraction layer
- [docs/component-reference.md](docs/component-reference.md) — Component types reference
- [docs/general-designs.md](docs/general-designs.md) — ROS 2, Viam, YARP comparison

### Archived Future State
- [docs/archive/future-state/](docs/archive/future-state/) — K3s/container designs (preserved)

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
├── archive/examples/       # Archived Go examples
│   ├── hello-robot/        # NATS pub/sub example
│   └── hello-robot-production/  # Production-ready example
├── tools/                  # Development tools
│   └── pwm-ramp-test/      # PWM testing tool
├── docs/                   # Documentation
├── specs/                  # Specifications
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

## RDL Example

```json
{
  "version": "3",
  "robot": {
    "name": "gps-tracker",
    "namespace": "gorai"
  },
  "nats": {"url": "nats://localhost:4222"},
  "components": [
    {
      "name": "gps",
      "type": "serial/gps",
      "config": {
        "device": "/dev/gps-sim",
        "baud_rate": 9600
      }
    }
  ]
}
```

---

## Development

### Build Commands

```bash
# Build CLI
make build

# Build all examples
make build-examples

# Build specific example
make build-example-pwm-controller

# Validate RDL examples
make validate-examples

# Run tests
make test

# Run all checks (fmt, vet, lint, test)
make check
```

### Run Examples

```bash
# Start NATS server
make nats-start

# Run RDL-based examples
make run-example-blinky
make run-example-gps
make run-example-camera

# Run Go-based examples
make run-example-hello-robot
make run-example-pwm-controller
```

### Key Make Targets

| Target | Description |
|--------|-------------|
| `build` | Build CLI binary |
| `build-examples` | Build all example binaries |
| `validate-examples` | Validate all RDL configs |
| `test` | Run unit tests |
| `check` | Run all checks |
| `nats-start` | Start local NATS server |

---

## Testing

```bash
# Unit tests
make test

# All tests with coverage
make test-cover

# Quick tests (unit + component)
make test-quick

# All tests (unit, component, integration, module, system)
make test-all
```

See [specs/testing-approach.md](specs/testing-approach.md) and [specs/howto-run-tests.md](specs/howto-run-tests.md) for details.

---

## Contributing Guidelines

1. **Read STRATEGIC-SUMMARY.md** — understand strategic context
2. **Follow language strategy** — Go core, polyglot when appropriate
3. **Don't fight ROS 2** — we're complementary, not competitive
4. **Pragmatism over purity** — best tool for job
5. **Keep it simple** — complexity only when needed
6. **Document design decisions** — AI-assisted dev requires clarity

---

## Quick Reference

| Concept | Location |
|---------|----------|
| **LLM Design Guide** | [docs/LLM-DESIGN-GUIDE.md](docs/LLM-DESIGN-GUIDE.md) — **Start here for building components/services** |
| Framework spec | [specs/gorai-framework-specification.md](specs/gorai-framework-specification.md) |
| RDL format | [specs/robot-definition-language.md](specs/robot-definition-language.md) |
| Code organization | [specs/code-organization.md](specs/code-organization.md) |
| Mesh discovery | [specs/mesh-service-discovery.md](specs/mesh-service-discovery.md) |
| Dynamic discovery | [specs/dynamic-discovery.md](specs/dynamic-discovery.md) — **Auto-adoption, gateways, @discovered:** |
| Strategic decisions | [docs/STRATEGIC-SUMMARY.md](docs/STRATEGIC-SUMMARY.md) |
| GSP protocol | [specs/gsp-v2-protocol.md](specs/gsp-v2-protocol.md) |
| Hardware reqs | [specs/hardware-requirements.md](specs/hardware-requirements.md) |
| Future roadmap | [docs/FUTURE-ROADMAP.md](docs/FUTURE-ROADMAP.md) |

---

**Pronunciation:** "go-ray" (like "sting-ray")
