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
# 1. Install the CLI
go install github.com/gorai/gorai/cmd/gorai@latest

# 2. Create a robot project from the template
git clone https://github.com/emergingrobotics/gorai-robot-template.git my-robot
cd my-robot

# 3. Edit robot.json, then validate and run
gorai validate robot.json
gorai run robot.json

# 4. Build for deployment
gorai build robot.json -o robot --target linux/arm64
scp robot pi@raspberrypi:~ && ssh pi@raspberrypi ./robot
```

No containers. No K8s. No external services. Just a single binary that runs on a Raspberry Pi 5 or Orange Pi 5.

---

## Hardware Products Powered by Gorai

| Product | Type | Price | Status |
|---------|------|-------|--------|
| **ORCA** | Autonomous submersible (2 motors + dive planes, 80ft depth) | Under $2,500 | Prototype |
| **Surf** | Autonomous surface vessel | Under $1,500 | Prototype |
| Drive | Land robot | TBD | Deferred |

**ORCA** is the flagship hardware project. "Autonomous submersible under $2,500" is a category with zero competition — BlueROV2 ($4,600) is purely remote-controlled, and professional AUVs start at $50,000+. ORCA runs `gorai run` on a Raspberry Pi inside a pressure housing.

---

## Component Ecosystem

Gorai uses the **Caddy model** for component packaging: your robot project is a standard Go module, and the import list in `main.go` *is* the component manifest. Each component self-registers via `init()` calling `registry.RegisterComponent()`. There is no custom package manager -- Go modules handles everything.

### How It Works

A robot's `main.go` declares which components to include via blank imports:

```go
package main

import (
    gorai "github.com/gorai/gorai/pkg/gorai"

    // Remote proxy components from GoRAI core
    _ "github.com/gorai/gorai/components/motor/remote"
    _ "github.com/gorai/gorai/components/camera/remote"

    // Third-party component from the ecosystem
    _ "github.com/someone/gorai-component-lidar/rplidar"

    // Custom component in this repo
    _ "my-robot/components/ballast"
)

func main() {
    gorai.Run()
}
```

The Go import list replaces `package.json`, `requirements.txt`, or any custom manifest. What you import is what gets compiled into the binary.

### Workflow

```bash
gorai component search lidar          # Find components in the ecosystem
gorai component add sensor/rplidar    # go get + add blank import to main.go
gorai build robot.json                # Single binary with everything included
```

### Custom Components

Custom components are ordinary Go packages in your project's repo. Write a package with an `init()` function that calls `registry.RegisterComponent()`, add a blank import in `main.go`, and it compiles into the binary alongside everything else.

### Sharing Components

Sharing a component means publishing a Go module. Extract the package into its own repo, push to GitHub, and anyone can `gorai component add` it. No registry servers, no package approval process -- standard Go module hosting.

### Why This Matters

This is a key advantage of choosing Go as the platform language. The single-binary story extends all the way to the component ecosystem: no runtime dependency resolution, no DLL hell, no version conflicts at deploy time. Every dependency is resolved at build time by the Go toolchain, and the result is one static binary you copy to the robot.

Non-Go components (Python vision pipelines, C++ SLAM) run as external services communicating via NATS. They do not compile into the binary. This is Phase 2 complexity.

For the full design, see [docs/package-dev-approach.md](docs/package-dev-approach.md).

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

For the full strategic context, see [Gorai Overarching Strategy](https://github.com/emergingrobotics/gorai-docs/tree/main/docs/overview/) in the gorai-docs repository.

---

## Prerequisites

You need **Go 1.25+** to build gorai. That's it.

**macOS:**
```bash
brew install go
```

**Ubuntu/Debian:**
```bash
sudo apt update && sudo apt install -y golang-go
```

Or download from https://go.dev/dl/ for the latest version.

NATS is embedded in the gorai binary -- no external message broker to install.

### NATS CLI (optional, for debugging)

The NATS CLI lets you subscribe to messages and debug your robot:

**macOS:**
```bash
brew install nats-io/nats-tools/nats
```

**Linux:**
```bash
go install github.com/nats-io/natscli/nats@latest
```

---

## Quick Start

### 1. Install the Gorai CLI

```bash
go install github.com/gorai/gorai/cmd/gorai@latest
```

### 2. Create a robot project from the template

```bash
# Clone the template (or click "Use this template" on GitHub)
git clone https://github.com/emergingrobotics/gorai-robot-template.git my-robot
cd my-robot

# Update the module path to your own
go mod edit -module github.com/yourorg/my-robot
```

### 3. Edit robot.json

The template includes a skeleton `robot.json`. Add components as needed:

```json
{
  "version": "2",
  "robot": {"name": "my-robot", "description": "My first robot!"},
  "nats": {"embedded": true, "url": "nats://localhost:4222"},
  "dashboard": {"enabled": true, "listen": ":8080"},
  "components": []
}
```

### 4. Validate and run

```bash
make validate   # Check configuration
make run        # Run in development mode
```

The embedded NATS server starts automatically -- no separate process needed.

### 5. Build and deploy

```bash
# Build for Raspberry Pi
make build TARGET=linux/arm64

# Deploy
make deploy DEPLOY_HOST=pi@raspberrypi
```

### Using an External NATS Server

For multi-robot deployments or shared brokers, point at an external NATS server:

```json
{
  "nats": {
    "url": "nats://nats-server.local:4222",
    "external": true
  }
}
```

When `external` is `true` or the URL points to a non-local address, gorai connects to the external server instead of starting its own.

---

## Hardware Platforms

| Platform | AI Performance | Cost | Best For |
|----------|----------------|------|----------|
| **Raspberry Pi 5 (8GB)** | External (Hailo 13-26 TOPS) | ~$160 | Primary platform, best ecosystem |
| **Raspberry Pi 5 (4GB)** | External (Hailo 13-26 TOPS) | ~$100 | Budget builds |
| **Orange Pi 5B (8GB)** | 6 TOPS (built-in NPU) | ~$145 | Budget AI builds |

**Not supported:** Pi 3, Pi Zero

See Hardware Requirements in the [gorai-docs](https://github.com/emergingrobotics/gorai-docs/tree/main/docs/specifications/) repository for details.

### Hardware Access Patterns

Gorai supports two patterns for accessing hardware. Both produce components with identical interfaces — application code doesn't know or care which is used.

**Co-processor (RP2040 via GSP/2):** An RP2040 microcontroller handles real-time hardware I/O (PWM, motor control, encoders). The RPi communicates with it over USB serial using the Gorai Serial Protocol v2. Best for timing-critical control, isolating hardware from Linux scheduler jitter, and reliability-critical applications (e.g., ORCA — motor control must not glitch underwater).

**Native RPi hardware (GPIO/I2C/SPI):** Direct access to Raspberry Pi GPIO pins, I2C buses, and SPI buses from Go code. Best for simple sensors, I2C devices, prototyping, and cost-sensitive builds where an RP2040 is unnecessary.

Both patterns can be used simultaneously — e.g., RP2040 handling motors while the Pi reads I2C sensors directly.

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
| `gorai migrate` | Migrate RDL v1 config to v2 format |

### Component Management

| Command | Description |
|---------|-------------|
| `gorai component search` | Search for third-party components |
| `gorai component info` | Show component information |
| `gorai component add` | Add component to project |

### Mesh Commands (Service Discovery)

| Command | Description |
|---------|-------------|
| `gorai mesh services` | List running services in the mesh |
| `gorai mesh channels` | List registered NATS channels |
| `gorai mesh schemas` | List or show message schemas |
| `gorai mesh watch` | Watch for services joining/leaving |
| `gorai mesh summary` | Show mesh state summary |
| `gorai mesh robots` | List robots with registered services |
| `gorai mesh init` | Initialize predefined schemas |
| `gorai mesh reset` | Reset mesh (delete all data) |

---

## Core Components

The core repo provides **component interfaces**, **remote proxies** (for multi-robot mesh access), and **fakes** (for testing). Hardware-specific implementations live in external Go modules.

### Interfaces (in core)
`arm`, `base`, `camera`, `gripper`, `input`, `link`, `motor`, `power`, `pwm`, `sensor`, `servo`, `space`, `stepper`, `thruster`, `valve`

### Remote Proxies (in core)
`camera/remote`, `gpio/remote`, `input/remote`, `motor/remote`, `pwm/remote`, `sensor/encoder/remote`

### Fakes (in core, for testing)
`camera/fake`, `motor/fake`, `pwm/fake`, `servo/fake`

### External Components (separate Go modules)
Hardware-specific implementations are installed with `gorai component add`:
- `gorai-picarx` - SunFounder PiCar-X robot kit
- `gorai-driver-hcsr04` - HC-SR04 ultrasonic distance sensor
- More in the [component registry](https://github.com/emergingrobotics/gorai-registry)

---

## Architecture

Gorai uses a message-based architecture where all components communicate via an embedded NATS server:

```
┌──────────────────────────────────────────────────────┐
│                  Your Robot Binary                    │
│                                                      │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐                │
│  │  GPS    │ │ Motor   │ │ Sensor  │    ...          │
│  │Component│ │Component│ │Component│                 │
│  └────┬────┘ └────┬────┘ └────┬────┘                 │
│       │           │           │                      │
│       └───────────┴─────┬─────┘                      │
│                         │                            │
│                   ┌─────▼─────┐                      │
│                   │  Message  │                      │
│                   │   Router  │                      │
│                   └─────┬─────┘                      │
│                         │ NATS Protocol              │
│                   ┌─────▼─────┐                      │
│                   │ Embedded  │                      │
│                   │   NATS    │  (JetStream enabled) │
│                   └───────────┘                      │
└──────────────────────────────────────────────────────┘
```

**Key principles:**
- Single binary with embedded NATS -- no external services required
- Each component runs in its own goroutine
- Internal control uses Go channels
- Inter-component communication uses NATS only
- No shared memory between components
- Message-based architecture enables remote debugging
- External NATS supported for multi-robot or shared broker deployments

---

## Examples

| Example | Description | Status |
|---------|-------------|--------|
| [blinky](examples/blinky/) | LED blink demo | Working |
| [gps-tracker](examples/gps-tracker/) | GPS tracking robot | Working |
| [gsp-pico](examples/gsp-pico/) | GSP/2 Pico co-processor | Working |
| [hello-camera](examples/hello-camera/) | Camera streaming | Working |
| [pwm-controller](examples/pwm-controller/) | PWM control via gorai-gsp | Working |

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

See the mesh service discovery specification in the [gorai-docs](https://github.com/emergingrobotics/gorai-docs/tree/main/docs/specifications/) repository for complete documentation.

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

See the dynamic discovery specification in the [gorai-docs](https://github.com/emergingrobotics/gorai-docs/tree/main/docs/specifications/) repository for complete documentation.

---

## Why Gorai?

### Cloud-Native Patterns

| Pattern | ROS 2 | Gorai | Benefit |
|---------|-------|-------|---------|
| **Message Broker** | DDS peer-to-peer | NATS server | Decoupled, easy monitoring |
| **Event Sourcing** | rosbag (manual) | JetStream (built-in) | Replay, time-travel debug |
| **Observability** | Custom diagnostics | Prometheus /metrics | Industry-standard tools |
| **Package Management** | Custom (rosdep, colcon) | Go modules | Standard tooling, no custom manager |

### Language Strategy

- **Go core** — NATS orchestration, configuration, web dashboard
- **Python services** — Vision (OpenCV), ML inference (PyTorch) — *future*
- **C++ services** — SLAM (Cartographer), point cloud processing — *future*

NATS has clients for 40+ languages. Use the best tool for each job.

---

## Project Status

**Version:** 0.1.0 (Early Development)

### Current Phase: Single Binary Deployment

The current focus is a zero-dependency, single-binary deployment model:
- Single Go binary with embedded NATS server
- No external services required -- everything in one process
- No containers, no K8s
- Runs directly on Raspberry Pi with systemd
- JetStream enabled by default for event sourcing and mesh discovery

### Component Ecosystem (Caddy Model)

GoRAI uses the [Caddy model](docs/package-dev-approach.md) for component distribution. Each hardware driver is a standalone Go module. A user's robot project has a `main.go` that imports the GoRAI core plus blank imports for each component needed. `go build` produces a single binary with exactly those components compiled in. See [REQUIREMENTS.md](REQUIREMENTS.md) for details.

### Future Roadmap

- **Non-Go services** — Python/C++ services (vision, SLAM) communicate via NATS as external processes, not compiled-in components
- **Fleet management** — Multi-robot coordination (future phase)
- **ROS 2 bridge** — Interop with ROS 2 ecosystems (future phase)

---

## Documentation

All documentation has moved to the [gorai-docs](https://github.com/emergingrobotics/gorai-docs) repository, including:

- **Getting Started** — Hardware requirements, Robot Definition Language (RDL) configuration
- **Architecture & Design** — Strategy, vision analysis, design comparisons, mesh service discovery, dynamic discovery
- **For AI Assistants / LLMs** — [CLAUDE.md](CLAUDE.md) (in this repo), [gorai-docs INDEX.md](https://github.com/emergingrobotics/gorai-docs/blob/main/INDEX.md)
- **Future State** — Roadmap, component ecosystem plans

---

## Contributing

Gorai is built with [Claude Code](https://claude.ai/claude-code). We believe AI-assisted development is the future — the project is organized for both humans and AI to reason about effectively.

### For Humans
- See [CLAUDE.md](CLAUDE.md) for contributor guidelines
- See the [gorai-docs](https://github.com/emergingrobotics/gorai-docs) repository for code organization and package locations

### For AI Assistants
- See the LLM Design Guide in the [gorai-docs](https://github.com/emergingrobotics/gorai-docs) repository — everything needed to design new components and services without reading the entire codebase

---

## License

Apache 2.0
