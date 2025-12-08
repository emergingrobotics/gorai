# Mental Model & Architecture

This chapter establishes the conceptual framework for understanding Gorai.

## The Big Picture

A Gorai robot system consists of three layers:

1. **Primary Compute**: Linux SBCs running standard Go (Raspberry Pi 5, Rockchip, Jetson)
2. **Secondary Nodes**: Smaller Linux boards for distributed processing
3. **Microcontrollers**: TinyGo peripherals for real-time I/O

All nodes communicate through NATS, creating a unified messaging fabric.

## Core Concepts

### Nodes

A node is a process that connects to NATS and participates in the robot system. Nodes can:

- Publish sensor data
- Subscribe to commands
- Provide services
- Execute actions

### Resources

Everything in Gorai is a Resource. The Resource interface is the foundation:

```go
type Resource interface {
    Name() string
    Reconfigure(ctx context.Context, config Config) error
    Close(ctx context.Context) error
}
```

Resources are divided into:

- **Components**: Hardware abstractions (motors, sensors, cameras)
- **Services**: Software capabilities (vision, SLAM, navigation)

### Communication Patterns

Gorai provides three messaging patterns:

| Pattern | Use Case | NATS Primitive |
|---------|----------|----------------|
| Topics | Sensor streams, telemetry | Pub/Sub |
| Services | Synchronous RPC | Request/Reply |
| Actions | Long-running tasks | Request/Reply + Pub/Sub |

## Configuration

Robots are configured via JSON:

```json
{
  "components": [
    {
      "name": "left_motor",
      "type": "motor",
      "model": "gpio",
      "config": { "pin": 18 }
    }
  ]
}
```

Configuration can be updated at runtime without restarting the robot.
