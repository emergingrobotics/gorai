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
- Premature containerization (native binaries for simple robots)
- "Not Invented Here" syndrome (ROS 2 bridge planned for ecosystem access)
- Complex middleware that leaks implementation details

## Deployment Model

Gorai embraces **distributed systems thinking** from the ground up. Robots can range from simple single-process deployments to complex multi-node clusters, with the deployment strategy matching the robot's complexity.

### Deployment Tiers

**Tier 1: Simple Robots (systemd-managed processes)**

Simple robots run as one or a few processes managed by systemd. Processes can be native binaries or containers.

```
Simple Robot Deployment
┌─────────────────────────────────────────────────────────────────┐
│  Host System (Raspberry Pi, Jetson Nano, etc.)                  │
│                                                                  │
│  /opt/my-robot/                                                 │
│  └── robot.json                  (Configuration)                │
│                                                                  │
│  systemd Services                                                │
│  ├── nats-server.service         (Message broker)               │
│  ├── my-robot-core.service       (Core robot - native binary)   │
│  └── vision-detector.service     (Podman container - optional)  │
└─────────────────────────────────────────────────────────────────┘
```

**Deployment**: `scp + systemctl restart`

**When to use**:
- Single robot, simple needs
- 1-5 processes total
- Educational projects, hobby robots, single-purpose platforms
- GPS trackers, sensor platforms, basic wheeled robots

---

**Tier 2: Complex Robots (K3s single-node)**

Complex robots use K3s even on a single machine for orchestration capabilities. K3s is designed for edge computing and works great on Raspberry Pi.

```
Complex Robot Deployment (K3s single-node)
┌─────────────────────────────────────────────────────────────────┐
│  Host System (Raspberry Pi 4+, Jetson, RK3588)                  │
│                                                                  │
│  K3s (single-node Kubernetes)                                   │
│  ├── nats-server pod           (Message broker)                 │
│  ├── robot-core pod            (Go binary)                      │
│  ├── vision-yolo pod           (Python + PyTorch)               │
│  ├── slam-cartographer pod     (C++ + libraries)                │
│  └── navigation pod            (Go binary)                      │
│                                                                  │
│  Features: health checks, rolling updates, resource limits      │
└─────────────────────────────────────────────────────────────────┘
```

**Deployment**: `kubectl apply -f robot.yaml`

**When to use**:
- Multi-language services (Go + Python + C++)
- Complex ML pipelines
- Need orchestration features (health checks, restarts, updates)
- Research platforms with many moving parts
- University labs, advanced makers, sophisticated single robots

**Why K3s for single robots:**
- Automatic health monitoring and restart
- Rolling updates without downtime
- Resource limits prevent runaway processes
- Service discovery and load balancing
- Same tooling scales from 1 robot to 100
- Only 70MB binary, 512MB RAM (designed for edge/IoT)

---

**Tier 3: Fleet Management (K3s multi-node)**

Fleet deployments use K3s multi-node clusters for edge-cloud hybrid orchestration.

```
Fleet Deployment with K3s
┌─────────────────────────────────────────────────────────────────┐
│  Edge Nodes (robots in field)                                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ K3s Agent Node (Robot A)                                 │   │
│  │ ├── robot-core pod                                       │   │
│  │ ├── vision-lightweight pod                               │   │
│  │ └── navigation pod                                       │   │
│  └─────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ K3s Agent Node (Robot B)                                 │   │
│  │ ├── robot-core pod                                       │   │
│  │ └── sensor-fusion pod                                    │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼ (NATS leaf nodes)
┌─────────────────────────────────────────────────────────────────┐
│  Cloud Infrastructure                                            │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ K3s Server Node                                          │   │
│  │ ├── nats-server (supercluster)                           │   │
│  │ ├── heavy-ml-inference pods (GPU instances)              │   │
│  │ ├── fleet-management pod                                 │   │
│  │ ├── data-warehouse pod                                   │   │
│  │ └── monitoring (Prometheus, Grafana)                     │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

**Deployment**: GitOps (ArgoCD, Flux) or `kubectl apply`

**When to use**:
- Fleet of multiple robots needing centralized management
- Edge-cloud hybrid (heavy ML in cloud, real-time control on robots)
- Multi-robot coordination and swarm behaviors
- Commercial deployments (warehouses, delivery, agriculture)
- Centralized monitoring, updates, and configuration

### What is K3s?

[K3s](https://k3s.io/) is a lightweight, certified Kubernetes distribution designed for resource-constrained environments and edge computing. It's perfect for robotics:

| Feature | Kubernetes | K3s | Why for Robotics |
|---------|-----------|-----|------------------|
| **Binary size** | ~1GB | ~70MB | Fits on edge devices |
| **Memory usage** | ~1GB | ~512MB | Raspberry Pi 4 compatible |
| **Installation** | Complex | Single binary | `curl -sfL https://get.k3s.io \| sh -` |
| **Storage** | etcd (external) | SQLite (embedded) | No external dependencies |
| **Networking** | Calico/Flannel | Flannel (built-in) | Works out-of-box |
| **Load balancer** | Requires cloud | ServiceLB (built-in) | Edge load balancing |

**K3s removes**:
- Cloud-provider specific code
- In-tree storage plugins
- Legacy features

**K3s adds**:
- Embedded SQLite (or MySQL, PostgreSQL)
- Embedded Traefik ingress controller
- Embedded ServiceLB
- Helm controller

**For robotics, K3s enables**:
- Deploy same workload to edge and cloud
- Centralized management of robot fleet
- Automatic failover and restart
- Rolling updates without downtime
- Resource limits and scheduling

### Components vs Services

Components and services are **logical concepts**, not deployment requirements. How they run depends on the tier:

| Aspect | Components | Services |
|--------|------------|----------|
| **What** | Hardware abstractions | Software capabilities |
| **Typical runtime** | Native Go binary | Native, container, or remote |
| **Location** | On-robot (direct hardware access) | On-robot, cloud, or distributed |
| **Examples** | Camera, Motor, IMU, GPIO | AI inference, SLAM, Navigation |

**Components** (sensors, actuators):
- Usually run in native Go process for direct hardware access
- Can be separate processes if needed (e.g., licensed SDK in container)
- Communicate via NATS

**Services** (AI, navigation, SLAM):
- Run wherever makes sense: native binary, container, cloud
- Can be scaled independently (multiple vision service instances)
- Can leverage specialized hardware (Hailo NPU, GPU in cloud)
- NATS queue groups provide automatic load balancing

### Choosing Your Deployment Tier

```
Start simple → Add complexity only when needed

Tier 1 (systemd)
↓ (if you need: orchestration, multi-language, health monitoring, complex ML)
Tier 2 (K3s single-node)
↓ (if you need: fleet management, edge-cloud, centralized control)
Tier 3 (K3s multi-node)
```

**Decision guide:**
- **Simple robot** (sensor platform, GPS tracker) → **Tier 1 (systemd)**
- **Complex single robot** (research platform, multi-language services, heavy ML) → **Tier 2 (K3s single-node)**
- **Robot fleet** (warehouse automation, delivery fleet, coordinated swarm) → **Tier 3 (K3s multi-node)**

**Most robots should start at Tier 1.** Complexity is a liability—add it only when the benefits outweigh the costs. But when you need orchestration for a complex single robot, K3s is designed exactly for that use case.

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

### Containers: Tiered Approach

**Simple robots** (Surf, Drive, basic kits):
- Native Go binaries + systemd
- Optional Podman for vision/ML services
- Deployment: `scp + systemctl restart`

**Research platforms**:
- Podman pods for complex dependencies
- Multi-SBC coordination

**Fleet management** (>10 robots):
- K3s edge-cloud hybrid
- GitOps deployment

Start simple. Add complexity only when needed.

We use [Podman](https://podman.io/) as our reference container runtime:
- **Daemonless**: No background service required
- **Rootless**: Run containers without root privileges
- **OCI-compliant**: Images work with Docker, Kubernetes
- **Pod support**: Native multi-container coordination

See [nats/nats-setup.md](nats/nats-setup.md) for container-based NATS deployment.

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
- [Robot Definition Language](specs/robot-definition-language.md) - Configuration format
- [Runtime Specification](specs/runtime.md) - Robot lifecycle and behavior
- [Deployment](specs/deployment.md) - Deployment guide

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
