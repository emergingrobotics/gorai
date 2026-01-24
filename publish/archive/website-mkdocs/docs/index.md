# GoRAI

**A lightweight, Go-based robotics framework built on NATS.io**

*Pronounced "go-ray" (like "sting-ray")*

---

## What is GoRAI?

GoRAI is a modern robotics framework that provides:

- **NATS-based messaging** for pub/sub, request/reply, and persistence
- **Protocol Buffer serialization** for type-safe communication
- **Resource-centric architecture** with unified components/service abstraction
- **First-class AI/ML support** with hardware acceleration
- **Hot reconfiguration** without restart
- **TinyGo compatibility** for microcontrollers

## Why GoRAI?

| Aspect | GoRAI | ROS 2 | Viam |
|--------|-------|-------|------|
| **Language** | Go + TinyGo | C++/Python | Go |
| **Middleware** | NATS | DDS | gRPC |
| **Build** | Go modules | CMake + ament | Go modules |
| **AI/ML** | First-class | Package ecosystem | First-class |
| **MCU Support** | TinyGo | micro-ROS | None |

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

## Get Started

<div class="grid cards" markdown>

-   :material-download:{ .lg .middle } **Installation**

    ---

    Install GoRAI and its dependencies

    [:octicons-arrow-right-24: Install](getting-started/installation.md)

-   :material-rocket-launch:{ .lg .middle } **Quick Start**

    ---

    Build your first GoRAI node in minutes

    [:octicons-arrow-right-24: Quick Start](getting-started/quickstart.md)

-   :material-book-open-variant:{ .lg .middle } **The Book**

    ---

    Comprehensive guide to building robots

    [:octicons-arrow-right-24: Read the Book](/book/)

-   :material-github:{ .lg .middle } **Source Code**

    ---

    View and contribute on GitHub

    [:octicons-arrow-right-24: GitHub](https://github.com/gorai/gorai)

</div>

## Example Projects

### GoRAI-Sentinel

Pan-tilt sensor fusion platform with camera, ToF depth sensor, and servo control.

**Hardware**: ~$150-350 | **Complexity**: Beginner

[Learn more](examples/pan-tilt.md)

### GoRAI-Skimmer

Autonomous surface vehicle for bathymetry and water monitoring.

**Hardware**: ~$530 | **Complexity**: Intermediate

[Learn more](examples/skimmer.md)

## License

Apache 2.0
