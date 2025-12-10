# Core Concepts

This page introduces the key concepts you'll use throughout Gorai development.

## The Object Model - Explained Simply

Think of Gorai like building with LEGO blocks for robots. Every single piece in the system—whether it's a camera, a motor, an AI vision system, or a communication channel—is built from the same fundamental building block called a **Resource**.

### The Resource: The Universal Building Block

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

## Components: The Physical Stuff

Components represent **hardware** (or virtual aggregations of hardware). They're organized into 5 categories:

| Category | What It Does | Examples |
|----------|--------------|----------|
| **Sensor** | Observes the world (read-only) | Camera, GPS, temperature sensor |
| **Actuator** | Changes the world (does stuff) | Motor, robotic arm, gripper |
| **Power** | Manages energy | Battery, power supply |
| **Space** | Virtual container on robot | Ballast tank, cargo bay with door |
| **Link** | Extra communication channel | Serial to MCU, radio telemetry |

**Note**: All components assume NATS connectivity as baseline—NATS is the assumed infrastructure, not a "Link." A Space is a virtual abstraction that coordinates other components (e.g., a ballast tank has valves and level sensors). A Link provides communication to devices that can't connect to NATS directly (microcontrollers, radio links).

## Services: The Brain Power

Services are **software** that processes data or makes decisions:

- **Vision** - Understands what cameras see
- **SLAM** - Builds maps and knows where the robot is
- **Navigation** - Plans paths from A to B
- **Behavior** - AI-powered decision making

## How They Talk to Each Other

Everything communicates via **NATS** (a messaging system). Messages follow a naming pattern:

```
gorai.{robot}.{node}.{topic}

Example: gorai.sentinel.camera_front.data.compressed
         │      │        │            │
         │      │        │            └─ the actual topic
         │      │        └─ which component
         │      └─ which robot
         └─ framework prefix
```

## The Interface Philosophy

The design uses **interfaces** (contracts that define what something can do):

```go
// If something is a Sensor, it MUST be able to give Readings
type Sensor interface {
    Readings() map[string]any
}

// If something is an Actuator, it MUST be able to Stop and tell you if it's Moving
type Actuator interface {
    IsMoving() bool
    Stop()
}
```

This means you can write code that works with "any sensor" or "any actuator" without caring about the specific hardware.

## Why This Design?

1. **Uniform treatment** - Everything is a Resource, so management code works on everything
2. **Hot reconfiguration** - Change settings without rebooting the robot
3. **Discoverable** - Find components by type (`GetByType("motor")`)
4. **Extensible** - Add new components by implementing the interface

## A Concrete Example

Imagine a robot with a camera and wheels:

```
Camera (Sensor)
  └─ publishes images to: gorai.mybot.camera.data

Vision Service (Service)
  └─ subscribes to camera images
  └─ publishes detections to: gorai.mybot.vision.detections

Motor Left (Actuator)
  └─ subscribes to: gorai.mybot.drive.command
  └─ publishes state to: gorai.mybot.motor_left.state
```

All three are Resources, so they all can be reconfigured, queried, and managed the same way—but each implements different interfaces based on what it actually does.

## Communication Patterns

| Pattern | Use Case | NATS Primitive |
|---------|----------|----------------|
| Topics | Sensor streams | Pub/Sub |
| Services | Synchronous RPC | Request/Reply |
| Actions | Long-running tasks | Request/Reply + Pub/Sub |

## Next Steps

- [Components Guide](../guides/components.md)
- [NATS Messaging](../guides/nats.md)
