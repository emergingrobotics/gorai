# Gorai Object Model & Interfaces - Explained Simply

## The Big Picture

Think of Gorai like building with LEGO blocks for robots. Every single piece in the system - whether it's a camera, a motor, an AI vision system, or a communication channel - is built from the same fundamental building block called a **Resource**.

## The Resource: The Universal Building Block

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

### Important Clarifications

**NATS is assumed infrastructure, not a Link.** All components assume IP connectivity to a NATS server—that's the baseline. Links exist for *additional* communication paths that NATS cannot reach.

**Space is a virtual abstraction.** A Space represents a physical area on the robot (like a ballast tank or cargo bay) but doesn't directly interface with hardware. Instead, it coordinates other components—valves, doors, level sensors, etc.—that control or monitor that area.

**Link bridges to non-NATS devices.** Common examples:
- Serial connection to a TinyGo microcontroller
- Radio link for telemetry when out of WiFi range
- CAN bus for vehicle systems

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

## Learn More

For the complete technical specification, see [Framework Specification](../specs/gorai-framework-specification.md).
