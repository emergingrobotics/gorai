---
title: "Gorai"
description: "A lightweight, Go-based alternative to ROS 2, YARP, and Viam optimized for AI"
---

# Gorai

**A lightweight, Go-based alternative to ROS 2, YARP, and Viam optimized for AI**

*Pronounced "go-ray" (like "sting-ray")*

Gorai provides the essential capabilities of modern robotics frameworks without the complexity of DDS, the legacy constraints of C++ middleware, or mandatory cloud dependencies. Single-binary deployment, type-safe messaging, and battle-tested infrastructure.

---

## Why Gorai?

Gorai learns from three generations of robotics middleware:

| Aspect | Gorai | ROS 2 | Viam | YARP |
|--------|-------|-------|------|------|
| **Language** | Go + TinyGo | C++/Python | Go | C++ |
| **Middleware** | NATS | DDS | gRPC | Custom carriers |
| **Discovery** | NATS (embedded/cluster) | DDS multicast | Cloud/local | Name server |
| **Build** | Go modules | CMake + ament + colcon | Go modules | CMake |
| **AI/ML** | First-class + TPU/NPU | Package ecosystem | First-class services | Minimal |
| **MCU Support** | TinyGo | micro-ROS | None | None |
| **Cloud** | Optional | Ecosystem | Core feature | None |
| **License** | Apache 2.0 | Apache 2.0 | AGPL | BSD-3 |

---

## Design Principles

### What We Adopt

- **Resource-centric model** (from Viam): Unified abstraction for components and services
- **Named addressing** (from all): Hierarchical, human-readable identifiers
- **Transport abstraction** (from YARP): NATS as unified transport for pub/sub and request/reply
- **Configuration-driven** (from Viam): JSON config with hot reload
- **Device interfaces** (from all): Clean separation of hardware from logic
- **NWS/NWC pattern** (from YARP): Transparent local/remote resource access

### What We Differentiate

- **NATS as core**: Simpler than DDS, more capable than gRPC for pub/sub patterns
- **TinyGo support**: Unified language from microcontrollers to cloud
- **TPU/NPU focus**: Edge AI as primary concern, not afterthought
- **No cloud dependency**: Standalone-first, cloud-optional
- **Lower barrier**: Simpler than ROS 2, more flexible than Viam

### What We Avoid

- Heavy build systems that increase barrier to entry
- Mandatory cloud connectivity
- Complex middleware abstractions that leak implementation details
- Central coordinators as single points of failure

---

## AI-Assisted Development

Gorai is built entirely with [Claude Code](https://claude.ai/claude-code) assisting in design, implementation, and testing. Go is particularly well-suited for AI-assisted development—the language's clarity, strong typing, and consistent idioms make it easier for AI to generate correct, idiomatic code.

---

## Object Model

Think of Gorai like building with LEGO blocks for robots. Every piece in the system—whether it's a camera, a motor, an AI vision system, or a communication channel—is built from the same fundamental building block called a **Resource**.

```
                    Resource (the base block)
                         │
        ┌────────────────┼────────────────┐
        │                │                │
   Component          Service          Module
   (hardware)       (software)       (plugins)
```

Every Resource has four basic abilities:

1. **Name** - A unique ID like `gorai:component:camera/front_camera`
2. **Reconfigure** - Can update settings without restarting
3. **DoCommand** - Can receive arbitrary commands
4. **Close** - Knows how to shut itself down cleanly

### Components: The Physical Stuff

Components represent **hardware** (or virtual aggregations of hardware):

| Category | What It Does | Examples |
|----------|--------------|----------|
| **Sensor** | Observes the world (read-only) | Camera, GPS, temperature sensor |
| **Actuator** | Changes the world (does stuff) | Motor, robotic arm, gripper |
| **Power** | Manages energy | Battery, power supply |
| **Space** | Virtual container on robot | Ballast tank, cargo bay with door |
| **Link** | Extra communication channel | Serial to MCU, radio telemetry |

### Services: The Brain Power

Services are **software** that processes data or makes decisions:

- **Vision** - Understands what cameras see
- **SLAM** - Builds maps and knows where the robot is
- **Navigation** - Plans paths from A to B
- **Behavior** - AI-powered decision making

---

## Communication Patterns

Gorai uses NATS to provide three core patterns:

| Pattern | NATS Primitive | Use Case |
|---------|----------------|----------|
| **Topics** | Pub/Sub | Continuous sensor streams, telemetry |
| **Services** | Request/Reply | Synchronous RPC, configuration |
| **Actions** | Request/Reply + Pub/Sub | Long-running tasks with feedback |

Messages follow a naming pattern:

```
gorai.{robot}.{node}.{topic}

Example: gorai.sentinel.camera_front.data.compressed
```

---

## Quick Start

```go
n, _ := node.New("my_robot", node.WithNATS("nats://localhost:4222"))
defer n.Close()

// Publish sensor data
pub := pub.New[sensor.Image](n, "camera.image")
pub.Publish(ctx, &sensor.Image{Width: 640, Height: 480, Data: frame})

// Subscribe to commands
sub.New[geometry.Twist](n, "cmd_vel", func(msg *geometry.Twist) {
    drive(msg.Linear.X, msg.Angular.Z)
})

n.Spin(ctx)
```

---

## Core Features

### Resource Model

Every component (hardware) and service (software capability) is a resource:

```go
type Resource interface {
    Name() string
    Reconfigure(ctx context.Context, config Config) error
    Close(ctx context.Context) error
}
```

### Hot Reconfiguration

Update robot configuration without restart:

```json
{
  "components": [
    {
      "name": "left_motor",
      "type": "motor",
      "model": "gpio",
      "config": {
        "pin": 18,
        "frequency": 1000
      }
    }
  ]
}
```

### Device Interfaces

Clean abstraction for hardware:

```go
type Motor interface {
    Resource
    SetPower(ctx context.Context, power float64) error
    GetPosition(ctx context.Context) (float64, error)
    Stop(ctx context.Context) error
}
```

---

## AI/ML Integration

Gorai provides first-class support for edge AI with a focus on hardware acceleration.

### Services

- **Vision**: Object detection, classification, segmentation
- **ML Model**: Generic tensor inference with TPU/NPU acceleration
- **SLAM**: Localization and mapping
- **Navigation**: Waypoint and geospatial navigation

### Hardware Acceleration

| Platform | Status | Library |
|----------|--------|---------|
| Rockchip RK3588 NPU | **Working** | go-rknnlite |
| NVIDIA CUDA | **Working** | onnxruntime_go |
| Intel OpenVINO | Partial | GoCV |
| Google Coral TPU | Planned | - |
| Hailo NPU | Planned | - |

---

## Example Projects

### Gorai-Sentinel
Pan-tilt sensor fusion platform with camera, ToF depth sensor, and servo control.

**Hardware**: ~$150-350 | **Complexity**: Beginner

### Gorai-Skimmer
Autonomous surface vehicle for bathymetry and water monitoring.

**Hardware**: ~$530 | **Complexity**: Intermediate

---

## Get Started

- **[Installation](/docs/getting-started/installation/)** — Set up your development environment
- **[Quick Start](/docs/getting-started/quickstart/)** — Build your first component
- **[Core Concepts](/docs/getting-started/concepts/)** — Understand the architecture

## Resources

- **[Documentation](/docs/)** — Guides and reference
- **[Examples](/examples/)** — Working code samples
- **[GitHub](https://github.com/emergingrobotics/gorai)** — Source code and issues

---

## License

Apache 2.0
