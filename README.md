# Gorai

<img src="./images/gorai.png" width="25%">

**A lightweight, Go-based alternative to ROS 2, YARP, and Viam optimized for AI**

*Pronounced "go-ray" (like "sting-ray")*

Gorai provides the essential capabilities of modern robotics frameworks without the complexity of DDS, the legacy constraints of C++ middleware, or mandatory cloud dependencies. Native systemd deployment, type-safe messaging, and battle-tested infrastructure.

We originally wanted to name this project "Gort" after the iconic robot from *The Day the Earth Stood Still*, but that name has been taken for years by DevOps tooling for the excellent [GoBot](https://gobot.io/) project. Gorai is a contraction of **Go + Robot + AI**—and it evokes the [eye ray gun of Gort](https://youtu.be/K6iF5sINVns?t=92)!

## Deployment Model

A Gorai robot runs as a **single native process** managed by systemd. Components (hardware drivers) are compiled into the binary, while services (AI, navigation) can run internally or as external processes/containers.

```
Robot Deployment (v2)
┌─────────────────────────────────────────────────────────────────┐
│  Host System (Raspberry Pi, etc.)                               │
│                                                                  │
│  /opt/my-robot/                                                 │
│  └── robot.json                  (Configuration)                │
│                                                                  │
│  systemd                                                         │
│  ├── my-robot.service            (Main robot process)           │
│  └── nats-server.service         (NATS - installed natively)    │
│                                                                  │
│  Optional: External Services                                     │
│  └── hailo-detector container    (AI inference on NPU)          │
└─────────────────────────────────────────────────────────────────┘
```

### Components vs Services

| Aspect | Components | Services |
|--------|------------|----------|
| **What** | Hardware abstractions | Software capabilities |
| **Runtime** | Native Go in monolith | Internal, external process, or container |
| **Location** | Same host as robot | Same host or remote |
| **Examples** | Camera, Motor, IMU, GPIO | AI inference, SLAM, Navigation |

**Components** (sensors, actuators) MUST be native Go code compiled into the robot binary. They have direct hardware access.

**Services** (AI, navigation, SLAM) CAN be:
- Internal Go code in the monolith
- External native processes
- External containers (for Python ML, specialized hardware)
- Running on different hosts

## Quick Start

### Prerequisites

- Linux (Raspberry Pi OS, Ubuntu, Fedora)
- Go 1.22+ (for building)
- NATS server (`sudo apt install nats-server`)

### 1. Install the gorai CLI

```bash
# From source
go install github.com/gorai/gorai/cmd/gorai@latest

# Or build locally
git clone https://github.com/gorai/gorai
cd gorai
make build
sudo make install
```

### 2. Create a robot configuration

```json
{
  "$schema": "https://gorai.dev/schemas/rdl-v2.json",
  "version": "2",
  "robot": {
    "name": "my-robot",
    "description": "Example robot with camera"
  },
  "nats": {
    "url": "nats://localhost:4222"
  },
  "components": [
    {
      "name": "main_camera",
      "type": "camera",
      "model": "v4l2",
      "attributes": {
        "device": "/dev/video0",
        "width": 640,
        "height": 480
      }
    }
  ],
  "services": [
    {
      "name": "dashboard",
      "type": "dashboard",
      "model": "web",
      "attributes": {
        "listen": ":8080"
      }
    }
  ],
  "dashboard": {
    "enabled": true
  }
}
```

### 3. Run the robot

```bash
# Development: Run directly in foreground
gorai run --config robot.json

# Production: Deploy as systemd service
gorai start --config robot.json --enable
```

### 4. Manage the robot

```bash
# Check status
gorai status --config robot.json

# View logs
gorai logs --config robot.json -f

# Stop
gorai stop --config robot.json
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `gorai run` | Run robot directly in foreground |
| `gorai start` | Deploy as native systemd service |
| `gorai stop` | Stop the systemd service |
| `gorai status` | Show service status and components |
| `gorai logs` | View logs via journalctl |
| `gorai migrate` | Migrate v1 config to v2 format |
| `gorai validate` | Validate configuration file |

## Why Gorai?

Gorai learns from three generations of robotics middleware:

| Aspect | Gorai | ROS 2 | Viam | YARP |
|--------|-------|-------|------|------|
| **Language** | Go + TinyGo | C++/Python | Go | C++ |
| **Middleware** | NATS | DDS | gRPC | Custom carriers |
| **Discovery** | NATS (embedded/cluster) | DDS multicast | Cloud/local | Name server |
| **Deployment** | Native + systemd | apt/source | Cloud | apt/source |
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

### What We Differentiate

- **Monolithic core**: Components run in single process for simplicity and performance
- **External services**: AI/ML can run in containers with specialized hardware access
- **NATS as core**: Simpler than DDS, more capable than gRPC for pub/sub patterns
- **TinyGo support**: Unified language from microcontrollers to cloud
- **TPU/NPU focus**: Edge AI as primary concern, not afterthought
- **No cloud dependency**: Standalone-first, cloud-optional

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
| **Link** | Extra communication channel | Serial to MCU, radio telemetry |

### Communication via NATS

Everything communicates via NATS messaging:

```
gorai.{robot}.{node}.{topic}

Example: gorai.sentinel.camera_front.data
         │      │        │            │
         │      │        │            └─ the actual topic
         │      │        └─ which component
         │      └─ which robot
         └─ framework prefix
```

## External Services (AI/ML)

For compute-intensive workloads like ML inference, services can run as external processes or containers. External services can have their own **Service RDL** file that defines their behavior independently:

### Service RDL Pattern

External services can be modular, reusable components with their own definition files:

```
services/
+-- person-detector/
    +-- person-detector.rdl.json   # Service RDL (defines interface)
    +-- main.py                     # Service implementation
    +-- Containerfile               # Container build
```

**Service RDL** (`person-detector.rdl.json`):
```json
{
  "kind": "service",
  "service": {
    "type": "object_detection",
    "model": "yolox"
  },
  "topics": {
    "subscribe": [
      {"name": "input", "pattern": "gorai.{namespace}.{input_component}.data"}
    ],
    "publish": [
      {"name": "annotated", "pattern": "gorai.{namespace}.{service}.annotated"},
      {"name": "detections", "pattern": "gorai.{namespace}.{service}.detections"}
    ]
  },
  "attributes": {
    "confidence_threshold": {"type": "float", "default": 0.5}
  }
}
```

**Robot RDL** references the Service RDL:
```json
{
  "services": [
    {
      "name": "person_detector",
      "rdl": "./services/person-detector/person-detector.rdl.json",
      "attributes": {
        "input_component": "main_camera",
        "model_path": "/models/yolox_s.hef"
      },
      "external": {
        "enabled": true,
        "container": {
          "image": "localhost/person-detector:latest",
          "devices": ["/dev/hailo0"]
        },
        "managed": true
      }
    }
  ]
}
```

### Benefits of Service RDL

- **Modularity**: Services are self-contained packages
- **Reusability**: Same service definition works across robots
- **Separation of concerns**: Service authors define behavior, robot integrators configure deployment
- **Documentation**: Service RDL documents the interface (topics, attributes)
- **Validation**: Attributes are type-checked at load time

This pattern enables:
- Python-based ML frameworks
- Specialized hardware access (Hailo NPU, Coral TPU)
- Independent updates and scaling
- Running on different hosts

## AI/ML Integration

Gorai provides first-class support for edge AI with hardware acceleration.

### Platform Support

| Platform | Status | Library | Notes |
|----------|--------|---------|-------|
| Hailo NPU | **Working** | Container with HailoRT | 13-26 TOPS |
| Rockchip RK3588 NPU | **Working** | [go-rknnlite](https://github.com/swdee/go-rknnlite) | 6 TOPS |
| NVIDIA CUDA | **Working** | [onnxruntime_go](https://github.com/yalue/onnxruntime_go) | Requires CUDA 12.x |
| Google Coral TPU | Planned | - | CGo bindings needed |

### Inference Runtimes

- **ONNX Runtime**: Primary path for PyTorch/TensorFlow models
- **TensorFlow Lite**: Edge inference via [tflitego](https://github.com/nbortolotti/tflitego)
- **HailoRT**: For Hailo NPU inference (via container)

## Monitoring

### Prometheus Integration

Gorai exposes metrics for Prometheus:

```
gorai_sensor_value{robot="sentinel",sensor="temperature"} 42.5
gorai_component_state{robot="sentinel",component="motor_left"} 1
gorai_messages_total{robot="sentinel",direction="sent"} 15420
```

### Logging via journald

All logs go to systemd journal:

```bash
# Follow robot logs
gorai logs --config robot.json -f

# View last 100 lines
gorai logs --config robot.json --tail 100

# Direct journalctl
journalctl --user -u my-robot.service -f
```

## Documentation

- [Robot Definition Language](specs/robot-definition-language.md) - Configuration format
- [Framework Specification](specs/gorai-framework-specification.md) - Complete technical specification
- [Runtime Specification](specs/runtime.md) - Robot lifecycle and behavior
- [Deployment](specs/deployment.md) - Deployment guide
- [Hailo NPU Integration](plans/hailo.md) - AI inference with Hailo

## Example Projects

### [hello-camera](examples/hello-camera/)
Simple camera robot demonstrating V4L2 capture and web dashboard. A minimal example to get started.

### [hello-people-detector](examples/hello-people-detector/)
Camera robot with AI-based person detection using an external service. Demonstrates the **Service RDL** pattern for modular, reusable external services running on Hailo NPU.

### [Gorai-Sentinel](projects/project-pan-tilt.md)
Pan-tilt sensor fusion platform with camera, ToF depth sensor, and servo control.

## Migration from v1

If you have a v1 configuration with containers:

```bash
# Migrate configuration
gorai migrate --config old-robot.json --output robot.json

# Install NATS natively
sudo apt install nats-server
sudo systemctl enable nats-server

# Run with new config
gorai start --config robot.json --enable
```

## Project Goals

1. **Native deployment** - Single binary managed by systemd
2. **Written in Go** (and TinyGo for microcontrollers)
3. **NATS.io** for all communication
4. **Edge AI first** - TPU/NPU support as primary concern
5. **Low barrier to entry** - Simple CLI, JSON configuration
6. **Modular** - Services can be external containers when needed
7. **No cloud dependency** - Standalone-first

## AI-Assisted Development

Gorai is built entirely with [Claude Code](https://claude.ai/claude-code). Go's clarity, strong typing, and consistent idioms make it well-suited for AI-assisted development.

## License

Apache 2.0
