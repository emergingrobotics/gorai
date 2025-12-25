# Gorai

<img src="./images/gorai.png" width="25%">

**Professional robotics for prosumers — without the PhD**

*Pronounced "go-ray" (like "sting-ray")*

Gorai is a Go-based robotics framework designed for makers, citizen scientists, students, and small organizations who need real autonomy without ROS 2's complexity. Built on battle-tested cloud infrastructure (NATS, Prometheus), Gorai applies distributed systems patterns to robotics—making professional-grade software accessible to everyone.

We originally wanted to name this project "Gort" after the iconic robot from *The Day the Earth Stood Still*, but that name has been taken for years by DevOps tooling for the excellent [GoBot](https://gobot.io/) project. Gorai is a contraction of **Go + Robot + AI**—and it evokes the [eye ray gun of Gort](https://youtu.be/K6iF5sINVns?t=92)!

---

## Who Is Gorai For?

### You Should Use Gorai If:
- ✅ You want **real autonomy** (not educational toys, not simulations)
- ✅ You need **approachable software** (productive in days, not months)
- ✅ You value **modern developer experience** (AI-assisted coding, simple deployment)
- ✅ You're building **prosumer robots** (marine monitoring, land vehicles, research platforms)
- ✅ You want **cloud-native patterns** (NATS, Prometheus, containerization when needed)

### You Should Use ROS 2 If:
- ❌ You're in **enterprise/research** robotics (warehouse automation, autonomous vehicles)
- ❌ You need the **full ROS ecosystem** (thousands of packages, simulation, SLAM libraries)
- ❌ You have **months to invest** in learning complex toolchains
- ❌ You're in **academia** where ROS 2 is the standard

**We're not trying to replace ROS 2.** We're targeting a different market—prosumers who find Arduino too limiting and ROS 2 too complex. Think of Gorai as "ROS 2 for prosumers," not "ROS 2 killer."

## Why Gorai?

### The Cloud-Native Advantage

Gorai applies proven **distributed systems patterns from cloud infrastructure** to robotics:

| Cloud Pattern | ROS 2 Approach | Gorai Approach | Benefit |
|---------------|----------------|----------------|---------|
| **Message Broker** | Direct DDS peer-to-peer | NATS server | Decouples producers/consumers; easy monitoring |
| **Service Mesh** | Custom DDS discovery | NATS request/reply + queue groups | Automatic load balancing, failover |
| **Event Sourcing** | rosbag (manual) | JetStream (built-in) | Replay sensor streams, time-travel debugging |
| **Config Management** | Per-node params | NATS KV store | Global config, hot reload, version history |
| **Observability** | Custom diagnostics | Prometheus /metrics | Industry-standard dashboards, alerting |
| **Security** | DDS Security (complex) | NATS auth, TLS, JWT | Simpler, firewall-friendly |

**ROS 2's DDS is pre-cloud architecture** (designed for LANs in 2004). Gorai uses infrastructure that powers Coinbase, Mastercard, and Siemens—proven at global scale.

### Comparison Table

| Aspect | Gorai | ROS 2 | Viam | YARP |
|--------|-------|-------|------|------|
| **Target Market** | Prosumer | Enterprise/Research | Cloud-first | Research |
| **Language** | Go + TinyGo | C++/Python | Go | C++ |
| **Middleware** | NATS (cloud-native) | DDS (pre-cloud) | gRPC | Custom carriers |
| **Learning Curve** | Days | Months | Moderate | High |
| **Build System** | Go modules | CMake + ament + colcon | Go modules | CMake |
| **Deployment** | Native + systemd | Workspace sourcing | Cloud-dependent | Build artifacts |
| **AI/ML** | First-class + TPU/NPU | Package ecosystem | First-class | Minimal |
| **MCU Support** | TinyGo | micro-ROS | None | None |
| **ROS 2 Bridge** | Planned (Phase 3) | Native | None | Yes |
| **License** | Apache 2.0 | Apache 2.0 | AGPL | BSD-3 |

## Design Principles

Drawing from [our analysis](docs/general-designs.md) of ROS 2, Viam, and YARP, plus [strategic vision](docs/vision-analysis.md):

### What We Adopt

- **Resource-centric model** (from Viam): Unified abstraction for components and services
- **Named addressing** (from ROS 2, Viam, YARP): Hierarchical, human-readable identifiers
- **Message types** (from ROS 2): Proven sensor_msgs, geometry_msgs patterns
- **Transform trees** (from ROS 2 TF2): Coordinate frame management
- **Configuration-driven** (from Viam): JSON config with hot reload
- **Device interfaces** (from all): Clean separation of hardware from logic
- **NWS/NWC pattern** (from YARP): Transparent local/remote resource access

### What We Differentiate

- **NATS over DDS**: Cloud-native message broker vs. pre-cloud middleware
- **Go core + polyglot services**: Pragmatic language choices per component
- **Prometheus native**: Industry-standard observability, not custom diagnostics
- **JetStream**: Built-in event sourcing, not bolt-on database
- **Native deployment**: Components run in single process for simplicity and performance
- **External services**: AI/ML can run in containers with specialized hardware access
- **TinyGo support**: Unified language from microcontrollers to cloud
- **TPU/NPU focus**: Edge AI as primary concern, not afterthought
- **No cloud dependency**: Standalone-first, cloud-optional
- **Lower barrier**: Productive in days, not months

### What We Avoid

- PhD-level learning curves (CMake, colcon, ament, DDS QoS)
- Language purity dogma (use best tool for each job)
- Kubernetes complexity exposure (RDL abstracts it away)
- "Not Invented Here" syndrome (ROS 2 bridge planned for ecosystem access)
- Complex middleware that leaks implementation details

## Architecture: K3s-Everywhere

Gorai uses a **K3s-everywhere architecture** where all robots deploy on Kubernetes (K3s), from simple single robots to multi-robot fleets. This provides a consistent deployment model that scales without architectural changes.

### Why K3s-Everywhere?

**"AI at the edge requires capable hardware. Capable hardware can run K3s."**

- Edge AI workloads (vision, SLAM) need 4GB+ RAM regardless of orchestration
- One deployment model to learn, debug, and maintain
- Every robot is fleet-ready from day one
- Container benefits: reproducible builds, versioned artifacts, isolated dependencies
- K3s is designed for edge/IoT: 70MB binary, ~1.5GB RAM overhead

### What Users See vs What Runs

```
┌─────────────────────────────────────────────────────────────────┐
│                    User Experience                               │
│                                                                  │
│   robot.json (RDL)  →  gorai deploy  →  Robot running           │
│                                                                  │
│   Users work with: Robot Definition Language (JSON)             │
│   Users never need: kubectl, manifests, pods, deployments       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ gorai translates
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    What Actually Runs                            │
│                                                                  │
│   K3s Cluster (single-node or multi-node)                       │
│   ├── Namespace: gorai-{robot-name}                             │
│   ├── Pod: nats (message broker)                                │
│   ├── Pod: gorai-core (Go orchestration + components)           │
│   ├── Pod: {service} (vision, SLAM, navigation)                 │
│   └── ConfigMap, Services, PVCs (auto-generated)                │
└─────────────────────────────────────────────────────────────────┘
```

### Hardware Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| **Compute** | Raspberry Pi 4 (4GB) | Raspberry Pi 5 (8GB) |
| **Storage** | USB 3.0 SSD (128GB) | NVMe SSD (256GB) |
| **Cost** | ~$105 | ~$145 |

**Not supported:** Pi 3, Pi Zero, Pi 4 (2GB), SD card-only deployments

**Why SSD required:** K3s uses SQLite which requires sustained random I/O. SD cards provide only 10-30 IOPS, causing database corruption and instability.

### Single Robot Deployment

```
┌─────────────────────────────────────────────────────────────────┐
│  Raspberry Pi 5 (8GB) + NVMe SSD                                │
│                                                                  │
│  K3s (containerd)                                               │
│  ├── nats pod           (message broker)                        │
│  ├── gorai-core pod     (orchestration, components)             │
│  ├── detector pod       (vision service)                        │
│  └── navigation pod     (path planning)                         │
│                                                                  │
│  Hardware: /dev/video0, /dev/i2c-1, /dev/hailo0                 │
└─────────────────────────────────────────────────────────────────┘
```

### Fleet Deployment

```
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│     Robot 1      │  │     Robot 2      │  │     Robot 3      │
│  K3s (worker)    │  │  K3s (worker)    │  │  K3s (worker)    │
│  NATS Leaf Node  │  │  NATS Leaf Node  │  │  NATS Leaf Node  │
└────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘
         └─────────────────────┼─────────────────────┘
                               │
                    ┌──────────┴──────────┐
                    │   Control Plane      │
                    │   K3s Server         │
                    │   NATS Hub           │
                    │   ArgoCD (GitOps)    │
                    └─────────────────────┘
```

### Components vs Services

| Aspect | Components | Services |
|--------|------------|----------|
| **What** | Hardware abstractions | Software capabilities |
| **Implementation** | Native Go in gorai-core pod | Separate pods (any language) |
| **Hardware access** | Direct device passthrough | Via NATS messaging |
| **Examples** | Camera, Motor, IMU, GPIO | Vision, SLAM, Navigation |

All communication happens via NATS—components and services are logically separate but deployed as Kubernetes pods.

## Quick Start

### Prerequisites

- Raspberry Pi 4 (4GB+) or Pi 5 with external SSD
- Linux (Raspberry Pi OS 64-bit, Ubuntu)
- Internet connection for initial setup

### 1. Install Gorai and Initialize K3s

```bash
# Install gorai CLI
curl -sfL https://get.gorai.dev | sh

# Initialize K3s cluster (first time only, ~2-3 minutes)
gorai cluster init

# Verify cluster is ready
gorai cluster status
```

### 2. Create a robot configuration

```json
{
  "$schema": "https://gorai.dev/schemas/rdl-v3.json",
  "version": "3",
  "robot": {
    "name": "my-robot",
    "description": "Example robot with camera"
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
  "dashboard": {
    "enabled": true
  }
}
```

### 3. Deploy the robot

```bash
# Deploy to K3s cluster
gorai deploy robot.json

# Watch deployment progress
gorai status my-robot
```

### 4. Manage the robot

```bash
# Check status
gorai status my-robot

# View logs
gorai logs my-robot -f

# Open dashboard
gorai dashboard my-robot

# Undeploy
gorai undeploy my-robot
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `gorai cluster init` | Initialize K3s on this machine |
| `gorai cluster join` | Join an existing K3s cluster |
| `gorai cluster status` | Show cluster health and nodes |
| `gorai deploy` | Deploy robot to K3s cluster |
| `gorai undeploy` | Remove robot from cluster |
| `gorai status` | Show robot status and pods |
| `gorai logs` | View robot logs |
| `gorai dashboard` | Open web dashboard |
| `gorai validate` | Validate configuration file |
| `gorai build` | Build container images |

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

### Services

- **Vision**: Object detection, classification, segmentation
- **ML Model**: Generic tensor inference with TPU/NPU acceleration
- **SLAM**: Localization and mapping
- **Navigation**: Waypoint and geospatial navigation

### Platform

Gorai targets **Linux-based systems** including:
- x86_64 servers and workstations
- ARM64 single-board computers (Raspberry Pi, Rockchip, NVIDIA Jetson)
- Microcontrollers via TinyGo (serial bridge to Linux node)

**Raspberry Pi 5** is our reference platform for testing and verification.

### Language Strategy: Pragmatic Polyglot

**Go for core framework** (NATS orchestration, node lifecycle, configuration, web UI):
- Native concurrency perfect for robotics
- Single binary deployment
- AI-assisted coding excellent with Go's clean syntax
- Cross-compilation trivial

**Polyglot services via NATS clients** (any language can be a GoRAI service):
- **Python**: Vision (YOLO, OpenCV), ML inference (PyTorch), path planning
- **C++**: SLAM (Cartographer, ORB-SLAM), point cloud processing
- **TinyGo**: Microcontroller peripherals (motor control, safety cutoffs)

NATS has clients for 40+ languages. Use the best tool for each job.

### Container Runtime

Gorai uses **K3s with containerd** for all deployments. The RDL abstracts container orchestration—users define robots in JSON, and `gorai deploy` handles Kubernetes manifests automatically.

**Container images** are OCI-compliant and can be built with Docker, Podman, or Buildah. K3s uses containerd internally (neither Docker nor Podman daemon required on the robot).

See [specs/deployment-k3s.md](specs/deployment-k3s.md) for deployment details.

### Hardware Acceleration

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

### Core Documentation
- [Object Model Explained Simply](docs/simple-object-model.md) - Beginner-friendly introduction to Resources, Components, and Services
- [Framework Specification](specs/gorai-framework-specification.md) - Complete technical specification
- [Hello Sensor Design](specs/hello-sensor-design.md) - Example design document (CPU temperature sensor)
- [Code Organization](specs/code-organization.md) - Module structure and naming conventions
- [Robot Definition Language](specs/robot-definition-language.md) - RDL v3 configuration format
- [Deployment (K3s)](specs/deployment-k3s.md) - K3s-everywhere deployment guide
- [Hardware Requirements](specs/hardware-requirements.md) - Supported platforms and specs
- [Runtime Specification](specs/runtime.md) - Robot lifecycle and behavior

### Strategic Vision
- [Vision Analysis](docs/vision-analysis.md) - **Strategic architecture assessment**: Pure Go vs. hybrid, ROS 2 positioning, containerization strategy, market positioning
- [Strategic Summary](docs/STRATEGIC-SUMMARY.md) - Quick reference for strategic decisions
- [Design Comparison](docs/general-designs.md) - Analysis of ROS 2, Viam, and YARP
- [ROS 2 Design](docs/ros2-design.md) - ROS 2 architecture summary
- [Viam Design](docs/viam-design.md) - Viam architecture summary
- [YARP Design](docs/yarp-design.md) - YARP architecture summary

### AI/ML Integration
- [Go AI Ecosystem](docs/go-ai-material.md) - ML frameworks, inference runtimes, and hardware acceleration
- [Hailo NPU Integration](plans/hailo.md) - AI inference with Hailo

### Design Document Philosophy

The [Hello Sensor Design](specs/hello-sensor-design.md) document serves as a template for Gorai component designs. This level of detail is intentional: a well-written design document is both documentation for humans and a blueprint for AI-assisted implementation. It includes:

- Architecture diagrams and component relationships
- Complete Protocol Buffer definitions
- Platform-specific implementation details
- Verification steps and expected outputs
- Test specifications

When contributing new components or services, follow this format. The specificity enables AI coding assistants to implement and test designs with minimal ambiguity.

## Example Projects

### [hello-camera](examples/hello-camera/)
Simple camera robot demonstrating V4L2 capture and web dashboard. A minimal example to get started.

### [hello-people-detector](examples/hello-people-detector/)
Camera robot with AI-based person detection using an external service. Demonstrates the **Service RDL** pattern for modular, reusable external services running on Hailo NPU.

### [Gorai-Sentinel](projects/project-pan-tilt.md)
Pan-tilt sensor fusion platform with camera, ToF depth sensor, and servo control. Validates multi-sensor synchronization, real-time control loops, and the action/service patterns.

**Hardware**: ~$150-350 | **Complexity**: Beginner

### [Gorai-Skimmer](projects/project-simple-boat.md)
Autonomous surface vehicle for bathymetry and water monitoring. Differential thrust propulsion, GPS navigation, Open Echo sonar, and optional underwater camera/hydrophone.

**Hardware**: ~$530 | **Complexity**: Intermediate

## Migration from v2

If you have a v2 configuration with tiered deployment:

```bash
# Initialize K3s cluster (required for v3)
gorai cluster init

# Update configuration version
# Change "version": "2" to "version": "3" in robot.json

# Deploy to K3s
gorai deploy robot.json

# Remove old systemd services (if any)
sudo systemctl disable gorai-*
```

See [specs/archive/](specs/archive/) for documentation on the previous tiered deployment model.

## Project Goals

1. **Target prosumer market** — makers, citizen scientists, students, small organizations
2. **Go core framework** — with pragmatic polyglot services via NATS
3. **Cloud-native patterns** — NATS, Prometheus, JetStream (proven distributed systems)
4. **AI-assisted coding** — designed for modern AI-powered development
5. **Low barrier to entry** — productive in days, not months
6. **ROS 2 compatibility** — bridge planned (Phase 3) for ecosystem access
7. **Modular architecture** — components, services, and resources all use same interface
8. **Edge AI focus** — TPU/NPU acceleration, ONNX Runtime, TensorFlow Lite
9. **No cloud dependency** — standalone-first, cloud-optional
10. **Have fun!** — robotics should be accessible and enjoyable

## Roadmap

### Phase 1: Core Framework (Months 1-6)
- ✅ NATS-based messaging (pub/sub, request/reply, actions)
- ✅ Resource model (components, services, unified interface)
- ✅ Configuration system (JSON, hot reload)
- ✅ Basic sensors (GPS, IMU, compass) in pure Go
- ✅ Motor control (I2C, PWM) in pure Go
- ✅ Web dashboard (Go stdlib http)
- ✅ Deploy to educational kit (PiCar-X) for validation

### Phase 2: First Product - Surf (Months 6-12)
- 🔄 Marine-specific sensors (GPS, compass, depth)
- 🔄 Waypoint navigation
- 🔄 Mission planner
- 🔄 LoRa telemetry
- 🔄 Hardware design finalized
- ⏸️ ROS 2 bridge (defer to Phase 3)
- ⏸️ K3s deployment (defer to Phase 3)

### Phase 3: Ecosystem (Months 12-18)
- ⏳ ROS 2 bridge MVP (one-way: ROS2 → NATS)
- ⏳ Gazebo simulation integration
- ⏳ Drive (wheeled robot) hardware
- ⏳ Community drivers (cameras, LIDAR)
- ⏳ Podman deployment templates

### Phase 4: Advanced Features (Months 18-24)
- ⏳ SLAM integration (Cartographer via C++ wrapper)
- ⏳ K3s edge-cloud deployment
- ⏳ Fleet management dashboard
- ⏳ Advanced ML (object tracking, semantic SLAM)

**Legend**: ✅ Complete | 🔄 In Progress | ⏳ Planned | ⏸️ Deferred

## AI-Assisted Development

Gorai is built entirely with [Claude Code](https://claude.ai/claude-code). Go's clarity, strong typing, and consistent idioms make it well-suited for AI-assisted development.

We believe AI-assisted software engineering is the future, and the entire project is organized to leverage it to the maximum extent possible: detailed specifications, clear interfaces, and comprehensive documentation that both humans and AI can reason about effectively.

## License

Apache 2.0
