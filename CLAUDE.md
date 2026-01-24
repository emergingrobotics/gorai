# Gorai

**Professional robotics for prosumers — without the PhD**

Gorai is a Go-based robotics framework designed for makers, citizen scientists, students, and small organizations who need real autonomy without ROS 2's complexity. Build a robot in under an hour with a single binary and NATS messaging.

## Strategic Positioning

**Target Market**: Prosumers who find Arduino too limiting and ROS 2 too complex.

**Not competing with ROS 2** in enterprise/research. We're "ROS 2 for prosumers," not "ROS 2 killer."

See [docs/vision-analysis.md](docs/vision-analysis.md) for comprehensive strategic analysis.

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
- **NATS messaging** — All component communication via NATS
- **Go core** — Components, behaviors, runtime all in Go
- **Simple deployment** — Copy binary to Pi, run with systemd
- **Low overhead** — ~20-50MB RAM (vs 512MB+ for containers)

### Future Phases (Preserved in docs/archive/future-state/)

- **Phase 2:** Optional containers for ML/vision services (Python/C++)
- **Phase 3:** K3s orchestration for fleet management

## CLI Commands

```bash
gorai validate <config>   # Validate RDL file
gorai run <config>        # Run robot (development mode)
gorai build <config>      # Build standalone binary
gorai components          # List available component types
gorai version             # Show version info
```

## Hardware Platforms

| Platform | Role | AI Performance |
|----------|------|----------------|
| **Raspberry Pi 5 (8GB)** | Primary | External (Hailo 13-26 TOPS) |
| **Raspberry Pi 5 (4GB)** | Budget | External (Hailo) |
| **Orange Pi 5B (8GB)** | Budget AI | 6 TOPS (built-in) |

**Not supported:** Pi 3, Pi Zero, Pi 4 (2GB)

See [specs/hardware-requirements.md](specs/hardware-requirements.md) for full requirements.

## Key Documentation

- [README.md](README.md) — Quick start guide
- [specs/robot-definition-language.md](specs/robot-definition-language.md) — RDL configuration
- [specs/hardware-requirements.md](specs/hardware-requirements.md) — Hardware specs
- [docs/STRATEGIC-SUMMARY.md](docs/STRATEGIC-SUMMARY.md) — Strategic decisions
- [docs/FUTURE-ROADMAP.md](docs/FUTURE-ROADMAP.md) — Container/K3s expansion plans
- [docs/PACKAGE-LOCATIONS.md](docs/PACKAGE-LOCATIONS.md) — Where code belongs
- [docs/archive/future-state/](docs/archive/future-state/) — Preserved K3s/container designs

## Code Structure

```
gorai/
├── cmd/gorai/              # CLI commands
├── pkg/                    # Core libraries
│   ├── accel/              # ML acceleration
│   ├── config/             # RDL parsing
│   ├── nats/               # NATS client
│   ├── runtime/            # Robot lifecycle
│   └── dashboard/          # Web dashboard
├── components/              # Component interfaces
├── driver/                 # Hardware drivers (GPIO, I2C, serial)
├── services/                # Service implementations
├── examples/               # Example robots
│   ├── gps-tracker/        # GPS tracking example
│   └── blinky/             # LED blink example
├── docs/                   # Documentation
│   ├── archive/future-state/  # K3s/container designs (preserved)
│   └── *.md                # Strategy and design docs
└── archive/                # Archived code for future phases
```

## Design Principles

1. **Simple binary deployment** — No containers required for basic robots
2. **NATS-based messaging** — All component communication via NATS pub/sub
3. **Progressive complexity** — Start simple, add containers/K3s when needed
4. **Go-first, pragmatic polyglot** — Go core, Python/C++ via NATS (future)
5. **Cloud-native patterns** — NATS, Prometheus, JetStream (event sourcing)

## RDL Example

```json
{
  "name": "gps-tracker",
  "description": "Simple GPS tracker",
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

## Development

```bash
# Build CLI
go build -o bin/gorai ./cmd/gorai

# Run tests
go test ./...

# Validate example
./bin/gorai validate examples/gps-tracker/robot.rdl.json

# Run example
./bin/gorai run examples/gps-tracker/robot.rdl.json
```
