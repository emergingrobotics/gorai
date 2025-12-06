# Gorai

<img src="./images/gorai.png" width="25%">

**A lightweight, Go-based alternative to ROS 2, YARP, and Viam optimized for AI**

Gorai provides the essential capabilities of ROS 2 without the complexity of DDS, the legacy baggage of YARP, or the licensing concerns of Viam. Single-binary deployment, type-safe messaging, and battle-tested infrastructure.

## Why Gorai?

| | Gorai | ROS 2 | Viam |
|--|------|-------|------|
| **Language** | Go | C++/Python | Go |
| **Middleware** | NATS | DDS | Custom |
| **Deployment** | Single binary | Complex | Cloud-centric |
| **License** | Apache 2.0 | Apache 2.0 | AGPL |

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

## Documentation

- [Framework Specification](docs/gorai-framework-specification.md) - Core architecture, components, and message types

## Example Projects

### [Gorai-Sentinel](docs/project-pan-tilt.md)
Pan-tilt sensor fusion platform with camera, ToF depth sensor, and servo control. Validates multi-sensor synchronization, real-time control loops, and the action/service patterns.

**Hardware**: ~$150-350 | **Complexity**: Beginner

### [Gorai-Skimmer](docs/project-simple-boat.md)
Autonomous surface vehicle for bathymetry and water monitoring. Differential thrust propulsion, GPS navigation, Open Echo sonar, and optional underwater camera/hydrophone.

**Hardware**: ~$530 | **Complexity**: Intermediate

## License

Apache 2.0
