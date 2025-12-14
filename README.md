# Gorai

<img src="./images/gorai.png" width="25%">

**A lightweight, Go-based alternative to ROS 2, YARP, and Viam optimized for AI**

*Pronounced "go-ray" (like "sting-ray")*

Gorai provides the essential capabilities of modern robotics frameworks without the complexity of DDS, the legacy constraints of C++ middleware, or mandatory cloud dependencies. Container-native deployment, type-safe messaging, and battle-tested infrastructure.

We originally wanted to name this project "Gort" after the iconic robot from *The Day the Earth Stood Still*, but that name has been taken for years by DevOps tooling for the excellent [GoBot](https://gobot.io/) project. Gorai is a contraction of **Go + Robot + AI**—and it evokes the [eye ray gun of Gort](https://youtu.be/K6iF5sINVns?t=92)!

## Deployment Model

A Gorai robot is a **set of containers managed by systemd** via [Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html). There are no binaries to install on the robot—only container images and Quadlet unit files.

```
Robot Deployment
┌─────────────────────────────────────────────────────────────────┐
│  systemd                                                        │
│  ├── hello-camera-nats.service      (from .container file)     │
│  ├── hello-camera-gorai-core.service                           │
│  └── hello-camera-gorai-hailo.service                          │
│                                                                 │
│  Quadlet Files (~/.config/containers/systemd/)                 │
│  ├── hello-camera-network.network                              │
│  ├── hello-camera-nats.container                               │
│  ├── hello-camera-gorai-core.container                         │
│  └── hello-camera-gorai-hailo.container                        │
│                                                                 │
│  Container Images                                               │
│  ├── nats:2.10-alpine                                          │
│  ├── localhost/hello-camera-core:latest                        │
│  └── localhost/hello-camera-hailo:latest                       │
└─────────────────────────────────────────────────────────────────┘
```

### Why Containers + Quadlet?

| Benefit | Description |
|---------|-------------|
| **No installation** | Robot runs from container images, no system packages |
| **Reproducible** | Same images work on any Linux system with Podman |
| **Native systemd** | Services start at boot, restart on failure, use journald |
| **Rootless** | Run without root privileges for security |
| **Auto-updates** | Built-in `podman auto-update` with rollback |

## Quick Start

### Prerequisites

- Linux (Raspberry Pi OS, Ubuntu, Fedora, etc.)
- [Podman](https://podman.io/) 4.4+ (for Quadlet support)
- systemd (standard on most Linux distributions)

### 1. Install the gorai CLI

```bash
# From source
go install github.com/gorai/gorai/cmd/gorai@latest

# Or download a release binary
# https://github.com/gorai/gorai/releases
```

### 2. Create a robot configuration

```json
{
  "version": "1",
  "robot": {
    "name": "my-robot",
    "description": "Example robot"
  },
  "nats": {
    "url": "nats://nats:4222",
    "container": "nats"
  },
  "containers": {
    "nats": {
      "image": "docker.io/nats:2.10-alpine",
      "ports": ["4222:4222"],
      "restart": "always"
    },
    "gorai-core": {
      "image": "your-registry/my-robot-core:latest",
      "depends_on": {
        "nats": {"condition": "service_healthy"}
      },
      "devices": ["/dev/video0"],
      "restart": "always"
    }
  }
}
```

### 3. Build and deploy

```bash
# Generate Quadlet files and build container images
gorai build --config robot.json

# Start the robot (installs Quadlet files, starts services)
gorai start --config robot.json

# Check status
gorai status --config robot.json

# View logs
gorai logs --config robot.json -f

# Stop the robot
gorai stop --config robot.json
```

### 4. Enable auto-start at boot

```bash
# Enable user lingering (allows services to run without login)
loginctl enable-linger $USER

# Start with auto-enable
gorai start --config robot.json --enable
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `gorai build` | Generate Quadlet files and build container images |
| `gorai start` | Deploy and start the robot via systemd |
| `gorai stop` | Stop the robot services |
| `gorai status` | Show service status |
| `gorai logs` | View logs via journalctl |

All commands use systemd under the hood:
- `gorai start` → `systemctl --user start {robot}-*.service`
- `gorai stop` → `systemctl --user stop {robot}-*.service`
- `gorai logs` → `journalctl --user -u {service}`

## Why Gorai?

Gorai learns from three generations of robotics middleware:

| Aspect | Gorai | ROS 2 | Viam | YARP |
|--------|-------|-------|------|------|
| **Language** | Go + TinyGo | C++/Python | Go | C++ |
| **Middleware** | NATS | DDS | gRPC | Custom carriers |
| **Discovery** | NATS (embedded/cluster) | DDS multicast | Cloud/local | Name server |
| **Deployment** | Containers + Quadlet | apt/source | Cloud | apt/source |
| **AI/ML** | First-class + TPU/NPU | Package ecosystem | First-class services | Minimal |
| **MCU Support** | TinyGo | micro-ROS | None | None |
| **Cloud** | Optional | Ecosystem | Core feature | None |
| **License** | Apache 2.0 | Apache 2.0 | AGPL | BSD-3 |

## Design Principles

### What We Adopt

- **Resource-centric model** (from Viam): Unified abstraction for components and services
- **Named addressing** (from all): Hierarchical, human-readable identifiers
- **Transport abstraction** (from YARP): NATS as unified transport for pub/sub and request/reply
- **Configuration-driven** (from Viam): JSON config with hot reload
- **Device interfaces** (from all): Clean separation of hardware from logic

### What We Differentiate

- **Container-native**: Deployment via Podman containers, managed by systemd Quadlet
- **NATS as core**: Simpler than DDS, more capable than gRPC for pub/sub patterns
- **TinyGo support**: Unified language from microcontrollers to cloud
- **TPU/NPU focus**: Edge AI as primary concern, not afterthought
- **No cloud dependency**: Standalone-first, cloud-optional

### What We Avoid

- Heavy build systems that increase barrier to entry
- Mandatory cloud connectivity
- Complex middleware abstractions that leak implementation details
- Installing binaries on the robot (containers only)

## Architecture

### Object Model

Every component (hardware) and service (software capability) is a Resource:

```go
type Resource interface {
    Name() string
    Reconfigure(ctx context.Context, config Config) error
    Close(ctx context.Context) error
}
```

### Component Categories

| Category | What It Does | Examples |
|----------|--------------|----------|
| **Sensor** | Observes the world (read-only) | Camera, GPS, temperature sensor |
| **Actuator** | Changes the world (does stuff) | Motor, robotic arm, gripper |
| **Power** | Manages energy | Battery, power supply |
| **Space** | Virtual container on robot | Ballast tank, cargo bay |
| **Link** | Extra communication channel | Serial to MCU, radio telemetry |

### Communication via NATS

Everything communicates via NATS messaging:

```
gorai.{robot}.{node}.{topic}

Example: gorai.sentinel.camera_front.data.compressed
         │      │        │            │
         │      │        │            └─ the actual topic
         │      │        └─ which component
         │      └─ which robot
         └─ framework prefix
```

### Communication Patterns

| Pattern | NATS Primitive | Use Case |
|---------|----------------|----------|
| **Topics** | Pub/Sub | Continuous sensor streams, telemetry |
| **Services** | Request/Reply | Synchronous RPC, configuration |
| **Actions** | Request/Reply + Pub/Sub | Long-running tasks with feedback |

## Container Architecture

### Robot Definition Language (RDL)

Robots are defined in JSON configuration files:

```json
{
  "version": "1",
  "robot": {"name": "rover1"},
  "nats": {"url": "nats://nats:4222"},
  "containers": {
    "nats": {
      "image": "nats:2.10-alpine",
      "ports": ["4222:4222"],
      "healthcheck": {
        "test": ["CMD", "wget", "-q", "--spider", "http://localhost:8222/healthz"],
        "interval": "10s"
      }
    },
    "gorai-core": {
      "image": "localhost/rover1-core:latest",
      "depends_on": {"nats": {"condition": "service_healthy"}},
      "devices": ["/dev/video0"],
      "group_add": ["video"],
      "components": ["camera"],
      "services": ["navigation"]
    }
  },
  "components": [
    {"name": "camera", "type": "camera", "model": "v4l2", "container": "gorai-core"}
  ],
  "services": [
    {"name": "navigation", "type": "navigation", "model": "default", "container": "gorai-core"}
  ]
}
```

### Generated Quadlet Files

The `gorai build` command generates systemd Quadlet files:

**hello-camera-nats.container:**
```ini
[Unit]
Description=Gorai container hello-camera-nats

[Container]
ContainerName=hello-camera-nats
Image=nats:2.10-alpine
Network=hello-camera-network.network
PublishPort=4222:4222
HealthCmd=wget -q --spider http://localhost:8222/healthz
AutoUpdate=registry

[Service]
Restart=always
TimeoutStartSec=300

[Install]
WantedBy=default.target
```

### Building Robot Containers

Each robot container is built from a Containerfile in your project:

```dockerfile
# Containerfile.core
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /robot ./cmd/robot

FROM alpine:3.19
COPY --from=builder /robot /usr/local/bin/robot
ENTRYPOINT ["/usr/local/bin/robot"]
```

Build with:
```bash
gorai build --config robot.json
```

## AI/ML Integration

Gorai provides first-class support for edge AI with hardware acceleration.

### Platform Support

| Platform | Status | Library | Notes |
|----------|--------|---------|-------|
| Rockchip RK3588 NPU | **Working** | [go-rknnlite](https://github.com/swdee/go-rknnlite) | 6 TOPS |
| NVIDIA CUDA | **Working** | [onnxruntime_go](https://github.com/yalue/onnxruntime_go) | Requires CUDA 12.x |
| Hailo NPU | **Working** | Container with HailoRT | See examples |
| Google Coral TPU | Planned | - | CGo bindings needed |

### Inference Runtimes

- **ONNX Runtime**: Primary path for PyTorch/TensorFlow models
- **TensorFlow Lite**: Edge inference via [tflitego](https://github.com/nbortolotti/tflitego)
- **GoCV DNN**: OpenCV's neural network module

## Monitoring

### Prometheus Integration

Gorai exposes metrics for Prometheus:

```
gorai_sensor_value{robot="sentinel",sensor="temperature"} 42.5
gorai_component_state{robot="sentinel",component="motor_left"} 1
gorai_messages_total{robot="sentinel",direction="sent"} 15420
```

### Logging via journald

All container logs go to systemd journal:

```bash
# Follow all robot logs
gorai logs --config robot.json -f

# View specific container
journalctl --user -u hello-camera-gorai-core.service -f

# View last 100 lines
gorai logs --config robot.json --tail 100
```

## Documentation

- [Quadlet Specification](specs/quadlet-functionality.md) - Container orchestration details
- [Framework Specification](specs/gorai-framework-specification.md) - Complete technical specification
- [RDL Specification](specs/robot-definition-language.md) - Robot configuration format
- [Code Organization](specs/code-organization.md) - Module structure
- [Go AI Ecosystem](docs/go-ai-material.md) - ML frameworks and hardware acceleration

## Example Projects

### [hello-camera](examples/hello-camera/)
Camera robot with person detection using Hailo NPU. Demonstrates multi-container deployment with NATS, camera streaming, and ML inference.

### [Gorai-Sentinel](projects/project-pan-tilt.md)
Pan-tilt sensor fusion platform with camera, ToF depth sensor, and servo control.

### [Gorai-Skimmer](projects/project-simple-boat.md)
Autonomous surface vehicle for bathymetry and water monitoring.

## AI-Assisted Development

Gorai is built entirely with [Claude Code](https://claude.ai/claude-code). Go's clarity, strong typing, and consistent idioms make it well-suited for AI-assisted development. We treat AI-generated code the same way we treat any code: with appropriate verification through tests, reviews, and integration testing.

## Project Goals

1. **Container-native deployment** - Robots are containers managed by systemd
2. **Written in Go** (and TinyGo for microcontrollers)
3. **NATS.io** for all communication
4. **Edge AI first** - TPU/NPU support as primary concern
5. **Low barrier to entry** - Simple CLI, JSON configuration
6. **Modular** - Components and services as separate containers
7. **No cloud dependency** - Standalone-first

## License

Apache 2.0
