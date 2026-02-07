# Gorai

<img src="./images/gorai.png" width="25%">

**The robotics platform for the AI era.**

*Pronounced "go-ray" (like "sting-ray")*

## Why Gorai?

> **Autonomy without replay is folklore.** Gorai treats action logs, state streams, and replay as first-class platform concerns — not optional add-ons.

> **Agent-compatible. Not agent-dependent.** AI-driven execution is a first-class citizen, but the platform works just as well with deterministic state machines, scripts, and rule-based planners. No autonomy method is mandatory. All are constrained.

> **Build robots like software. Run them like systems.** Gorai is a software engineer's robotics platform — opinionated, pragmatic, and operational. If you already think in APIs, distributed systems, and deployments, you'll be productive in days, not months.

---

> **Full documentation lives at [gorai-docs](https://github.com/emergingrobotics/gorai-docs).** Strategy, architecture, specifications, hardware analysis, a 20-chapter book, and implementation guides — all indexed for both humans and AI agents. Point your AI coding assistant at that repo and it will navigate 100+ documents via `CLAUDE.md` and `INDEX.md` automatically.

---

Build a robot in under an hour. Write JSON, get a binary, deploy to a Raspberry Pi.

```bash
# 1. Write a JSON file
cat > robot.rdl.json << 'EOF'
{
  "name": "my-robot",
  "nats": {"url": "nats://localhost:4222"},
  "components": [
    {"name": "gps", "type": "serial/gps", "config": {"device": "/dev/gps-sim"}}
  ]
}
EOF

# 2. Validate and run (development mode)
gorai validate robot.rdl.json
gorai run robot.rdl.json

# 3. Build for deployment
gorai build robot.rdl.json -o robot --target linux/arm64
scp robot pi@raspberrypi:~ && ssh pi@raspberrypi ./robot
```

No containers. No K8s. Just a 10-20MB binary that runs on a Raspberry Pi 5 or Orange Pi 5.

---

## Who Is Gorai For?

**Use Gorai if you:**
- Want **real autonomy** (not educational toys, not simulations)
- Need **approachable software** (productive in days, not months)
- Value **modern tooling** (AI-assisted coding, simple deployment)
- Are building **prosumer robots** (marine monitoring, land vehicles, research platforms)
- Are a **software-first team** adding physical embodiment to AI/ML or scaling from one robot to fleets
- Care more about **behavior, coordination, and operations** than low-level kinematics or middleware internals

**Use ROS 2 if you:**
- Work in **enterprise/research** robotics (warehouse automation, autonomous vehicles)
- Need the **full ROS ecosystem** (thousands of packages, simulation, SLAM libraries)
- Are in **academia** where ROS 2 is the standard
- Need **deep hardware or control-architecture experimentation** (ROS 2 is the right platform)

We're not replacing ROS 2 — we're targeting a different market. Think "ROS 2 for prosumers," with an **AI-first** design for the next wave of autonomous systems.

---

## The Future Robot Market (and Why Gorai Addresses It)

Robotics is shifting because **decision-making is moving up the stack**: autonomy is no longer only hand-authored logic. Perception, planning, and task selection are increasingly learned, probabilistic, or agentic. That shift—*physical AI*—changes what a platform must provide.

**What the future market is:**
- **Physical AI** — systems where behavior, autonomy, and coordination are more complex than motor control or kinematics
- **Software-first teams** — engineers and AI/ML practitioners who need to ship robots and fleets without becoming robotics-infrastructure experts
- **Autonomy as a spectrum** — from scripted behaviors and state machines to learned perception and agentic planners, all needing the same operational surface

**Why Gorai is built for it:**
- **AI-first by design** — we treat AI-driven execution as a first-class assumption: capability surfaces for tools, governance and safety at runtime, and auditability/replay as baseline, not add-ons
- **Start simple, scale without rewriting** — one binary today; same contracts and RDL when you add ML services, multi-robot coordination, or fleet operations
- **Clarity over maximal flexibility** — we optimize for teams whose hardest problems are autonomy, orchestration, deployment, and safety—not low-level robotics research

We are not trying to serve every robot. We are building the default platform for teams that ask: *"How do we safely decide what the robot should do next—and scale that across systems?"*

For the full strategic context, see [Gorai Overarching Strategy](docs/gorai-overarching-strategy.md).

---

## Prerequisites

Before you start, you need two things: **Go** (to build gorai) and **NATS Server** (message broker for component communication).

### 1. Install Go

**macOS:**
```bash
brew install go
```

**Ubuntu/Debian:**
```bash
sudo apt update && sudo apt install -y golang-go
```

Or download from https://go.dev/dl/ for the latest version (1.22+ required).

### 2. Install NATS Server

NATS is a lightweight message broker that gorai uses for all component communication. It must be running before you start your robot.

**macOS:**
```bash
brew install nats-server

# Start NATS (runs in foreground)
nats-server

# Or run in background
brew services start nats-server
```

**Ubuntu/Debian:**
```bash
sudo apt update && sudo apt install -y nats-server

# Start NATS and enable on boot
sudo systemctl enable --now nats-server

# Verify it's running
systemctl status nats-server
```

### 3. Install NATS CLI (optional, for debugging)

The NATS CLI lets you subscribe to messages and debug your robot.

**macOS:**
```bash
brew install nats-io/nats-tools/nats
```

**Ubuntu/Debian:**
```bash
go install github.com/nats-io/natscli/nats@latest
```

---

## Quick Start

### 1. Build Gorai CLI

```bash
# Clone the repository
git clone https://github.com/emergingrobotics/gorai.git
cd gorai

# Build CLI
go build -o bin/gorai ./cmd/gorai
```

### 2. Create your first robot

```bash
# Create robot configuration (uses GPS simulator)
cat > robot.rdl.json << 'EOF'
{
  "name": "gps-tracker",
  "description": "My first robot!",
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
EOF
```

**Note:** The GPS simulator (`/dev/gps-sim`) is used by default. This lets you test without hardware.

### 3. Validate and run

```bash
# Validate configuration
./bin/gorai validate robot.rdl.json

# Run in development mode
./bin/gorai run robot.rdl.json
```

### 4. Verify it works

In another terminal, subscribe to GPS data:

```bash
nats sub "gorai.gps-tracker.gps.nmea"
```

You'll see GPS NMEA sentences streaming over NATS.

---

## Hardware Platforms

| Platform | AI Performance | Cost | Best For |
|----------|----------------|------|----------|
| **Raspberry Pi 5 (8GB)** | External (Hailo 13-26 TOPS) | ~$160 | Primary platform, best ecosystem |
| **Raspberry Pi 5 (4GB)** | External (Hailo 13-26 TOPS) | ~$100 | Budget builds |
| **Orange Pi 5B (8GB)** | 6 TOPS (built-in NPU) | ~$145 | Budget AI builds |

**Not supported:** Pi 3, Pi Zero, Pi 4 (2GB)

See [Hardware Requirements](specs/hardware-requirements.md) for details.

---

## CLI Commands

### Core Commands

| Command | Description |
|---------|-------------|
| `gorai validate <config>` | Validate RDL configuration |
| `gorai run <config>` | Run robot in development mode |
| `gorai build <config>` | Build standalone binary |
| `gorai components` | List available component types |
| `gorai version` | Show version information |

### Mesh Commands (Service Discovery)

| Command | Description |
|---------|-------------|
| `gorai mesh services` | List running services in the mesh |
| `gorai mesh channels` | List registered NATS channels |
| `gorai mesh schemas` | List or show message schemas |
| `gorai mesh watch` | Watch for services joining/leaving |
| `gorai mesh summary` | Show mesh state summary |
| `gorai mesh init` | Initialize predefined schemas |

---

## Built-in Components

### Currently Implemented
- `serial/gps` - GPS NMEA reader (uses simulator by default)
- `gpio/input` - Digital input
- `gpio/output` - Digital output

### Coming Soon
- `motor/gpio` - DC motor control via GPIO
- `servo/gpio` - Hobby servo control
- `sensor/hcsr04` - HC-SR04 ultrasonic distance sensor
- `camera/v4l2` - USB/CSI cameras via Video4Linux

---

## Architecture

Gorai uses a message-based architecture where all components communicate via NATS:

```
┌─────────────────────────────────────────────────┐
│              Your Robot Binary                  │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐           │
│  │  GPS    │ │ Motor   │ │ Sensor  │    ...    │
│  │Component│ │Component│ │Component│           │
│  └────┬────┘ └────┬────┘ └────┬────┘           │
│       │           │           │                 │
│       └───────────┴─────┬─────┘                 │
│                         │                        │
│                   ┌─────▼─────┐                  │
│                   │  Message  │                  │
│                   │   Router  │                  │
│                   └─────┬─────┘                  │
└─────────────────────────┼───────────────────────┘
                          │ NATS Protocol
                          ▼
                   ┌─────────────┐
                   │ NATS Server │
                   └─────────────┘
```

**Key principles:**
- Each component runs in its own goroutine
- Internal control uses Go channels
- Inter-component communication uses NATS only
- No shared memory between components
- Message-based architecture enables remote debugging

---

## Examples

| Example | Description | Status |
|---------|-------------|--------|
| [gps-tracker](examples/gps-tracker/) | GPS tracking robot | Working |
| [blinky](examples/blinky/) | LED blink demo | Working |

---

## Service Discovery (Mesh)

Gorai includes a built-in service mesh for runtime discovery across independent processes. This enables:

- **Cross-binary discovery** — Modules that aren't compiled together can find each other
- **Channel registry** — Discover available NATS subjects and their schemas
- **Health monitoring** — Automatic TTL-based expiry for stale services
- **Schema documentation** — JSON Schema definitions for message types

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         NATS JetStream KV                                │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  gorai-services (TTL: 30s)     → Active service registrations   │    │
│  │  gorai-channels (persistent)   → Channel/subject descriptors    │    │
│  │  gorai-schemas  (persistent)   → Message schemas (JSON Schema)  │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

### Quick Example

```go
// Register a service
client, _ := mesh.NewClient(natsConn)
reg, _ := client.Register(ctx, mesh.ServiceDescriptor{
    Name:    "motor-controller",
    Type:    mesh.TypeComponent,
    Subtype: "motor",
    RobotID: "robot-alpha",
})
defer reg.Deregister()

// Discover other services
motors, _ := client.FindServices(ctx, mesh.Query{Subtype: "motor"})
```

### CLI Discovery

```bash
# List all running services
gorai mesh services

# Watch for changes in real-time
gorai mesh watch

# List available channels
gorai mesh channels robot-alpha
```

See [specs/mesh-service-discovery.md](specs/mesh-service-discovery.md) for complete documentation.

---

## Dynamic Discovery

Gorai supports **hybrid static/dynamic** configuration. Define structure in RDL, discover hardware at runtime.

### The Problem

Traditional approach requires declaring every device in config:

```json
{
  "components": [
    {"name": "motor1", "type": "motor/pwm", "config": {"pin": 18}},
    {"name": "motor2", "type": "motor/pwm", "config": {"pin": 19}}
  ]
}
```

What if you don't know what devices will be connected? What if devices are hot-plugged?

### The Solution

Define **discovery rules** instead of individual devices:

```json
{
  "gateways": [
    {
      "name": "usb-gateway",
      "type": "gateway/gsp",
      "config": {
        "discovery": {"enabled": true, "patterns": ["/dev/ttyACM*"]}
      }
    }
  ],

  "discovery": {
    "enabled": true,
    "auto_adopt": true,
    "rules": [
      {"match": {"capability": "PWM"}, "adopt_as": {"type": "motor"}},
      {"match": {"capability": "IMU"}, "adopt_as": {"type": "sensor", "subtype": "imu"}}
    ]
  },

  "services": [
    {
      "name": "patrol",
      "type": "behavior/patrol",
      "depends_on": ["@discovered:motor/*", "@discovered:sensor/imu/*"]
    }
  ]
}
```

**What happens:**

1. Gateway discovers Pico on USB with PWM+IMU capabilities
2. Auto-adopts as motor and IMU sensor (via rules)
3. Patrol service's `@discovered:` dependencies resolve
4. Robot starts patrolling with discovered hardware

See [specs/dynamic-discovery.md](specs/dynamic-discovery.md) for complete documentation.

---

## Why Gorai?

### Cloud-Native Patterns

| Pattern | ROS 2 | Gorai | Benefit |
|---------|-------|-------|---------|
| **Message Broker** | DDS peer-to-peer | NATS server | Decoupled, easy monitoring |
| **Event Sourcing** | rosbag (manual) | JetStream (built-in) | Replay, time-travel debug |
| **Observability** | Custom diagnostics | Prometheus /metrics | Industry-standard tools |

### Language Strategy

- **Go core** — NATS orchestration, configuration, web dashboard
- **Python services** — Vision (OpenCV), ML inference (PyTorch) — *future*
- **C++ services** — SLAM (Cartographer), point cloud processing — *future*

NATS has clients for 40+ languages. Use the best tool for each job.

---

## Project Status

**Version:** 0.1.0 (Early Development)

### Current Phase: Simple Binary Deployment

The current focus is on a simple, single-binary deployment model:
- Single Go binary (~10-20MB)
- NATS as only external dependency
- No containers, no K8s required
- Runs directly on Raspberry Pi with systemd

### Future Roadmap

For production fleets and advanced features, see [Future Roadmap](docs/FUTURE-ROADMAP.md):
- **Phase 2:** Optional containers for ML/vision services
- **Phase 3:** K3s orchestration for fleet management
- **Phase 4:** ROS 2 bridge, advanced SLAM

The K3s/container architecture is preserved in [docs/archive/future-state/](docs/archive/future-state/).

---

## Documentation

### Getting Started
- [Hardware Requirements](specs/hardware-requirements.md) — Supported platforms
- [Robot Definition Language](specs/robot-definition-language.md) — RDL configuration

### Architecture & Design
- [Gorai Overarching Strategy](docs/gorai-overarching-strategy.md) — AI-first positioning and future robot market
- [Vision Analysis](docs/vision-analysis.md) — Strategic architecture assessment
- [Design Comparison](docs/general-designs.md) — Analysis of ROS 2, Viam, YARP
- [Strategic Summary](docs/STRATEGIC-SUMMARY.md) — Key decisions and positioning
- [Mesh Service Discovery](specs/mesh-service-discovery.md) — Runtime service discovery
- [Dynamic Discovery](specs/dynamic-discovery.md) — Auto-adoption and `@discovered:` dependencies

### For AI Assistants / LLMs
- [LLM Design Guide](docs/LLM-DESIGN-GUIDE.md) — **Everything an LLM needs to build components/services**
- [CLAUDE.md](CLAUDE.md) — Project overview for AI assistants
- [gorai-docs INDEX.md](https://github.com/emergingrobotics/gorai-docs/blob/main/INDEX.md) — AI-navigable index of all documentation

### Future State
- [Future Roadmap](docs/FUTURE-ROADMAP.md) — Container/K3s expansion plans
- [K3s Architecture](docs/archive/future-state/) — Preserved K3s/container designs

---

## Contributing

Gorai is built with [Claude Code](https://claude.ai/claude-code). We believe AI-assisted development is the future — the project is organized for both humans and AI to reason about effectively.

### For Humans
- See [CLAUDE.md](CLAUDE.md) for contributor guidelines
- See [docs/PACKAGE-LOCATIONS.md](docs/PACKAGE-LOCATIONS.md) for code organization

### For AI Assistants
- See [docs/LLM-DESIGN-GUIDE.md](docs/LLM-DESIGN-GUIDE.md) — A single document containing everything needed to design new components and services without reading the entire codebase

---

## License

Apache 2.0
