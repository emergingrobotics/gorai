# Chapter 2: Mental Model & Architecture

Understanding GoRAI's architecture isn't about memorizing components—it's about internalizing a way of thinking about robot software. This chapter establishes the mental model that makes everything else click.

## 2.1 The Big Picture

A GoRAI robot is a collection of independent processes communicating through messages. This sounds abstract, so let's make it concrete.

### System Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         NATS Message Bus                                 │
│  (topics, services, streams)                                            │
└───────┬─────────────────┬─────────────────┬─────────────────┬───────────┘
        │                 │                 │                 │
        ▼                 ▼                 ▼                 ▼
┌───────────────┐ ┌───────────────┐ ┌───────────────┐ ┌───────────────┐
│  Sensor Node  │ │  Motor Node   │ │  Vision Node  │ │  Brain Node   │
│               │ │               │ │               │ │               │
│ - Temperature │ │ - Left Motor  │ │ - Camera      │ │ - Navigation  │
│ - IMU         │ │ - Right Motor │ │ - Detector    │ │ - Planning    │
│ - GPS         │ │ - Servo       │ │               │ │               │
└───────────────┘ └───────────────┘ └───────────────┘ └───────────────┘
        │                 │                 │                 │
        ▼                 ▼                 ▼                 ▼
   ┌─────────┐      ┌──────────┐     ┌──────────┐      ┌──────────┐
   │ Sensors │      │ Motors   │     │ Camera   │      │ Software │
   │ (HW)    │      │ (HW)     │     │ (HW)     │      │ Only     │
   └─────────┘      └──────────┘     └──────────┘      └──────────┘
```

Each box is a **node**—an independent process that manages one or more **resources** (components or services). Nodes communicate exclusively through the NATS message bus. This separation has profound implications:

- **Failure isolation**: If the vision node crashes, motors keep running
- **Independent scaling**: Run vision on a GPU, control on a Pi
- **Hot updates**: Restart a node without stopping the robot
- **Clean testing**: Replace real nodes with fake ones

### The Three-Layer Model

GoRAI robots typically span three computational layers:

#### Layer 1: Primary Compute

The main robot brain—a Linux single-board computer (SBC) running Go:
- Raspberry Pi 5, Orange Pi 5, Jetson Orin Nano
- Runs high-level logic: navigation, planning, behavior
- Connects to NATS server (often running locally)
- Has network access for updates, remote monitoring, fleet coordination

#### Layer 2: Secondary Nodes

Smaller Linux boards for specialized tasks:
- Dedicated vision processing on a board with GPU/NPU
- Sensor fusion node close to physical sensors
- Isolated control loops for manipulator arms
- Each runs its own nodes, connects to the same NATS bus

#### Layer 3: Microcontrollers

TinyGo on resource-constrained devices:
- RP2040, ESP32, STM32 for real-time motor control
- Direct GPIO/PWM/ADC for hardware interfaces
- Communicate with Layer 1/2 via serial gateway
- Handle microsecond-level timing requirements

Not every robot needs all three layers. A simple robot might have a single Raspberry Pi running everything. A complex robot might have dozens of nodes across multiple boards. The architecture scales gracefully.

### Message Flow

Let's trace a concrete example: a robot detecting an obstacle and stopping.

1. **Camera publishes image**: `gorai.vision.camera.image`
2. **Detector subscribes, processes, publishes**: `gorai.vision.detector.detections`
3. **Navigation subscribes, sees obstacle, publishes velocity**: `gorai.control.cmd_vel`
4. **Motor controller subscribes, applies brake**: Hardware stops

Each step is a node doing one thing well. Each message is typed and structured. The flow is observable with standard NATS tools:

```bash
# Watch all messages in real-time
nats sub "gorai.>"
```

This observability transforms debugging. Instead of adding print statements and recompiling, you watch message flow directly.
