# Gorai

<img src="./images/gorai.png" width="25%">

**A lightweight, Go-based alternative to ROS 2, YARP, and Viam optimized for AI**

Gorai provides the essential capabilities of modern robotics frameworks without the complexity of DDS, the legacy constraints of C++ middleware, or mandatory cloud dependencies. Single-binary deployment, type-safe messaging, and battle-tested infrastructure.

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

## Design Principles

Drawing from [our analysis](docs/general-designs.md) of ROS 2, Viam, and YARP:

### What We Adopt

- **Resource-centric model** (from Viam): Unified abstraction for components and services
- **Named addressing** (from all): Hierarchical, human-readable identifiers
- **Transport abstraction** (from YARP): NATS as unified transport for pub/sub and request/reply
- **Configuration-driven** (from Viam): YAML/JSON config with hot reload
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

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      Application Layer                       │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────────┐ │
│  │ Nodes   │  │ Actions │  │Services │  │ AI/ML Services  │ │
│  └─────────┘  └─────────┘  └─────────┘  └─────────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                     Communication Layer                      │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                    NATS Messaging                        ││
│  │  • Topics (pub/sub)  • Request/Reply  • JetStream       ││
│  └─────────────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────────────┤
│                      Resource Layer                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────────┐  │
│  │  Motor   │  │  Camera  │  │  Sensor  │  │   Generic   │  │
│  └──────────┘  └──────────┘  └──────────┘  └─────────────┘  │
├─────────────────────────────────────────────────────────────┤
│                      Hardware Layer                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────────┐  │
│  │   GPIO   │  │   I2C    │  │   SPI    │  │   Serial    │  │
│  └──────────┘  └──────────┘  └──────────┘  └─────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Communication Patterns

Gorai uses NATS to provide three core patterns (similar to ROS 2):

| Pattern | NATS Primitive | Use Case |
|---------|----------------|----------|
| **Topics** | Pub/Sub | Continuous sensor streams, telemetry |
| **Services** | Request/Reply | Synchronous RPC, configuration |
| **Actions** | Request/Reply + Pub/Sub | Long-running tasks with feedback |

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

```yaml
# robot.yaml
components:
  - name: left_motor
    type: motor
    model: gpio
    config:
      pin: 18
      frequency: 1000
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

### AI/ML Services (Planned)

First-class support for edge AI:

- **Vision**: Object detection, classification, segmentation
- **ML Model**: Generic tensor inference with TPU/NPU acceleration
- **SLAM**: Localization and mapping
- **Navigation**: Waypoint and geospatial navigation

## Documentation

- [Design Comparison](docs/general-designs.md) - Analysis of ROS 2, Viam, and YARP
- [ROS 2 Design](docs/ros2-design.md) - ROS 2 architecture summary
- [Viam Design](docs/viam-design.md) - Viam architecture summary
- [YARP Design](docs/yarp-design.md) - YARP architecture summary

## Example Projects

### [Gorai-Sentinel](docs/project-pan-tilt.md)
Pan-tilt sensor fusion platform with camera, ToF depth sensor, and servo control. Validates multi-sensor synchronization, real-time control loops, and the action/service patterns.

**Hardware**: ~$150-350 | **Complexity**: Beginner

### [Gorai-Skimmer](docs/project-simple-boat.md)
Autonomous surface vehicle for bathymetry and water monitoring. Differential thrust propulsion, GPS navigation, Open Echo sonar, and optional underwater camera/hydrophone.

**Hardware**: ~$530 | **Complexity**: Intermediate

## Project Goals

1. **Be written in Go** (and TinyGo for microcontrollers)
2. **Use AI-assisted coding** wherever possible
3. **Use NATS.io extensively** for all communication
4. **Define appropriate data types** for robotics
5. **Enable TPU/NPU** usage to the maximum extent
6. **Have a low barrier to entry** for adoption
7. **Be as modular as possible**
8. **Apply distributed systems lessons** from cloud software
9. **Have fun!**

## License

Apache 2.0
