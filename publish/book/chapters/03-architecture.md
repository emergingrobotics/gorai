# Architecture & Mental Model

Every framework has a mental model—a way of thinking about problems that, once internalized, makes everything easier. Gorai's mental model comes from distributed systems engineering: robots are networks of communicating processes, and the communication patterns matter as much as the code.

This chapter establishes the conceptual foundation you'll use throughout your Gorai development. We'll cover the three-layer architecture, the node and resource abstractions, configuration patterns, and the NWS/NWC mechanism that makes location transparent.

## The Object Model - Explained Simply

Before diving into architecture details, let's establish the fundamental mental model. Think of Gorai like building with LEGO blocks for robots. Every single piece in the system—whether it's a camera, a motor, an AI vision system, or a communication channel—is built from the same fundamental building block called a **Resource**.

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

### Components: The Physical Stuff

Components represent **hardware** (or virtual aggregations of hardware). They're organized into 5 categories:

| Category | What It Does | Examples |
|----------|--------------|----------|
| **Sensor** | Observes the world (read-only) | Camera, GPS, temperature sensor |
| **Actuator** | Changes the world (does stuff) | Motor, robotic arm, gripper |
| **Power** | Manages energy | Battery, power supply |
| **Space** | Virtual container on robot | Ballast tank, cargo bay with door |
| **Link** | Extra communication channel | Serial to MCU, radio telemetry |

**Note**: All components assume NATS connectivity as baseline—NATS is the assumed infrastructure, not a "Link." A Space is a virtual abstraction that coordinates other components (e.g., a ballast tank has valves and level sensors). A Link provides communication to devices that can't connect to NATS directly (microcontrollers, radio links).

### Services: The Brain Power

Services are **software** that processes data or makes decisions:

- **Vision** - Understands what cameras see
- **SLAM** - Builds maps and knows where the robot is
- **Navigation** - Plans paths from A to B
- **Behavior** - AI-powered decision making

### How They Talk to Each Other

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

### The Interface Philosophy

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

### Why This Design?

1. **Uniform treatment** - Everything is a Resource, so management code works on everything
2. **Hot reconfiguration** - Change settings without rebooting the robot
3. **Discoverable** - Find components by type (`GetByType("motor")`)
4. **Extensible** - Add new components by implementing the interface

### A Concrete Example

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

With this mental model established, let's see how it plays out in a real robot system.

## The Big Picture

A Gorai robot is a collection of independent processes communicating through messages. This sounds abstract, so let's make it concrete.

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

Gorai robots typically span three computational layers:

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

## Core Concepts

Three concepts form Gorai's foundation: Nodes, Resources, and the Resource Model. Master these, and the framework becomes intuitive.

### Nodes

A **Node** is the fundamental unit of execution in Gorai. It represents a process that:

- Connects to the NATS message bus
- Manages one or more resources
- Handles its own lifecycle (startup, running, shutdown)

Creating a node is straightforward:

```go
n, err := node.New("my_node",
    node.WithNATS("nats://localhost:4222"),
    node.WithNamespace("robot1"),
)
if err != nil {
    log.Fatal(err)
}
defer n.Close()
```

#### Node Lifecycle

Nodes progress through distinct phases:

1. **Creation**: `node.New()` creates the node structure
2. **Connection**: `WithNATS()` establishes NATS connection
3. **Setup**: Create publishers, subscribers, register resources
4. **Running**: `Spin()` blocks, processing messages
5. **Shutdown**: `Shutdown()` signals stop, `Close()` releases resources

```go
// The typical node lifecycle
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle signals for graceful shutdown
    go func() {
        sig := make(chan os.Signal, 1)
        signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
        <-sig
        cancel()
    }()

    // Create and configure node
    n, _ := node.New("example", node.WithNATS("nats://localhost:4222"))
    defer n.Close()

    // Setup resources, publishers, subscribers...

    // Run until context cancels
    n.Spin(ctx)
}
```

#### Namespacing for Multi-Robot Systems

When multiple robots share a NATS bus, namespacing prevents collisions:

```go
// Robot 1
n1, _ := node.New("sensors", node.WithNamespace("robot1"))
// Publishes to: robot1.sensors.*

// Robot 2
n2, _ := node.New("sensors", node.WithNamespace("robot2"))
// Publishes to: robot2.sensors.*
```

### Resources

A **Resource** is anything managed by Gorai: a motor, a camera, a navigation service, a sensor. All resources implement a common interface:

```go
type Resource interface {
    // Name returns the unique resource identifier
    Name() Name

    // Reconfigure updates the resource with new configuration
    Reconfigure(ctx context.Context, deps Dependencies, conf Config) error

    // DoCommand executes arbitrary commands for extensibility
    DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error)

    // Close releases all resources and stops background operations
    Close(ctx context.Context) error
}
```

This interface is small by design. Every resource can:

- **Be identified** by its Name
- **Be reconfigured** at runtime without restart
- **Be extended** with custom commands via DoCommand
- **Be cleaned up** when no longer needed

#### Resource Naming

Resources have structured names with four parts:

```
namespace:type:subtype/instance
```

For example:

- `gorai:component:motor/left_wheel`
- `gorai:component:sensor/cpu_temp`
- `gorai:service:vision/detector`
- `robot1:component:camera/front`

This structure enables:

- **Discovery**: Find all motors with `*:component:motor/*`
- **Organization**: Group by namespace for multi-robot fleets
- **Clarity**: Names are self-documenting

#### Components vs Services

Resources divide into two categories:

**Components** abstract hardware:

- Motors, cameras, sensors, grippers
- Have physical counterparts
- Defined in the `components/` package

**Services** provide software capabilities:

- Vision processing, navigation, SLAM
- Pure computation, no direct hardware
- Defined in the `services/` package

### The Resource Model

Gorai's resource model creates a consistent hierarchy:

```
Resource (base interface)
├── Component (hardware abstraction)
│   ├── Sensor (provides readings)
│   │   ├── IMU
│   │   ├── GPS
│   │   ├── Encoder
│   │   ├── Temperature
│   │   └── ...
│   └── Actuator (provides movement)
│       ├── Motor
│       ├── Servo
│       ├── Gripper
│       └── ...
├── Service (software capabilities)
│   ├── Vision
│   ├── Navigation
│   ├── SLAM
│   └── ...
└── Camera (special case: both sensor and image provider)
```

Each level adds capabilities:

**Sensor** adds the ability to provide readings:

```go
type Sensor interface {
    Resource
    Readings(ctx context.Context) (map[string]any, error)
}
```

**Actuator** adds motion control:

```go
type Actuator interface {
    Resource
    IsMoving(ctx context.Context) (bool, error)
    Stop(ctx context.Context) error
}
```

Specific types like `Motor` add domain-specific methods while inheriting from `Actuator`. This layered approach means code that works with any `Actuator` works with any motor, servo, or gripper—polymorphism through interfaces, the Go way.

## Concurrency Model

Gorai adopts a **single-owner model** for components—a deliberate design choice that simplifies implementation and aligns with Go's concurrency philosophy.

### Single-Owner Design

Each component instance is owned by exactly one goroutine at any time. This means:

- **No mutex protection needed** for component state
- **No concurrent access** to component methods
- **Simpler implementations** without synchronization overhead

```go
// A component implementation - no mutex needed
type FakeMotor struct {
    name     resource.Name
    power    float64       // Only accessed by owner goroutine
    moving   bool          // No concurrent access
    position float64
}

func (m *FakeMotor) SetPower(ctx context.Context, power float64) error {
    m.power = power
    m.moving = power != 0
    return nil
}
```

This contrasts with traditional thread-safe designs that wrap every field access in locks:

```go
// Traditional approach (NOT recommended in Gorai)
type ThreadSafeMotor struct {
    mu       sync.RWMutex  // Unnecessary overhead
    name     resource.Name
    power    float64
    moving   bool
}

func (m *ThreadSafeMotor) SetPower(ctx context.Context, power float64) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.power = power
    m.moving = power != 0
    return nil
}
```

### Cross-Goroutine Coordination via NATS

When coordination across goroutines is needed, use NATS messaging instead of shared memory:

```go
// Instead of sharing state directly, publish updates
func (m *Motor) SetPower(ctx context.Context, power float64) error {
    m.power = power
    m.moving = power != 0

    // Notify other components via NATS
    m.publisher.Publish(ctx, &MotorState{
        Power:  power,
        Moving: m.moving,
    })
    return nil
}

// Other components subscribe to state updates
func (c *Controller) subscribeToMotor() {
    c.subscriber.Subscribe("gorai.motor.state", func(state *MotorState) {
        // React to motor state changes
        c.handleMotorUpdate(state)
    })
}
```

This follows Go's mantra: **"Share memory by communicating, don't communicate by sharing memory."**

### Benefits of Single-Owner Model

1. **Simpler code**: No lock/unlock boilerplate, no deadlock risks
2. **Better performance**: No synchronization overhead
3. **Clearer ownership**: Explicit about who controls what
4. **Natural fit for robotics**: Components map to physical devices with single points of control
5. **Debugging**: State changes are traceable through message flow

### When to Use Mutexes

The single-owner model doesn't mean *never* use mutexes. Use them for:

- **Statistics counters** that multiple goroutines increment
- **Configuration caches** that are read frequently but written rarely
- **Connection pools** or other shared infrastructure

But component state—position, velocity, sensor readings—should follow single-owner semantics.

### The Component Hierarchy with Concurrency

```
Resource (base interface)
├── Component (hardware abstraction) ← Single owner per instance
│   ├── Sensor (read-only observations)
│   │   └── IMU, GPS, Encoder, LiDAR, ...
│   └── Actuator (motion control)
│       └── Motor, Servo, Stepper, Thruster, Valve, ...
├── Service (software capabilities) ← May use internal concurrency
│   └── Vision, Navigation, SLAM, Behavior, ...
└── Communication happens via NATS, not shared memory
```

Each Component instance has one owner. Services may internally use goroutines and synchronization, but their public interface follows the same request/response or publish/subscribe patterns.

## Distributed Architecture

Gorai is distributed by default. Even a single-board robot runs multiple nodes communicating through NATS. This section explains why and how.

### Why Distributed Matters for Robotics

Robots are inherently parallel systems:

- Sensors produce data continuously
- Actuators execute commands asynchronously
- Processing happens at different rates (vision at 30Hz, IMU at 1000Hz)
- Failures in one subsystem shouldn't cascade

Traditional monolithic architectures fight this reality. A single-threaded main loop serializes inherently parallel work. Shared memory creates coupling and race conditions. A crash anywhere stops everything.

Distributed architecture embraces the reality:

- Each node runs independently at its natural rate
- Message passing provides clean, typed interfaces between subsystems
- Failure isolation protects the system
- Horizontal scaling is natural—add nodes, add capability

### Primary Compute Responsibilities

The primary compute board (typically the most powerful SBC) usually handles:

**NATS Server**: The message broker runs here, accessible to all nodes:

```bash
# Start NATS server
nats-server -js  # -js enables JetStream
```

**High-Level Logic**: Navigation planning, behavior trees, mission management:

```go
// Brain node coordinates behavior
brain, _ := node.New("brain", node.WithNATS(natsURL))

// Subscribe to sensor fusion output
sub.New(brain, "gorai.perception.state", func(state *State) {
    decision := planner.Decide(state)
    cmdPub.Publish(ctx, decision.Commands)
})
```

**User Interfaces**: Web dashboards, API endpoints, remote control.

**Data Logging**: Recording messages for debugging and replay.

### Serial Gateway Pattern for Microcontrollers

TinyGo runs on microcontrollers (RP2040, ESP32) but can't connect directly to NATS. The serial gateway pattern bridges this gap:

```
┌──────────────────────────┐          ┌──────────────────────────┐
│     Linux Board          │          │     Microcontroller      │
│                          │          │     (TinyGo)             │
│  ┌────────────────────┐  │  Serial  │  ┌────────────────────┐  │
│  │   Serial Gateway   │◄─┼──────────┼──│   Motor Driver     │  │
│  │   (Go process)     │  │   UART   │  │   PWM/Encoder      │  │
│  └─────────┬──────────┘  │          │  └────────────────────┘  │
│            │             │          │                          │
│            ▼ NATS        │          └──────────────────────────┘
│  ┌────────────────────┐  │
│  │    Other Nodes     │  │
│  └────────────────────┘  │
└──────────────────────────┘
```

The gateway translates between NATS messages and a compact serial protocol. This pattern gives you:

- Real-time control on dedicated hardware
- NATS integration without microcontroller networking complexity
- Clean separation between real-time and non-real-time code

## Configuration & Hot Reload

Robots need configuration: motor directions, sensor calibrations, network addresses, behavioral parameters. Gorai provides a configuration system that works at runtime, not just at startup.

### JSON-Based Configuration

Configuration files are JSON, readable and editable without special tools:

```json
{
  "components": [
    {
      "name": "left_motor",
      "type": "motor",
      "model": "gpio",
      "attributes": {
        "pin_forward": 17,
        "pin_reverse": 18,
        "pin_pwm": 12,
        "max_rpm": 200
      }
    },
    {
      "name": "front_camera",
      "type": "camera",
      "model": "v4l2",
      "attributes": {
        "device": "/dev/video0",
        "width": 640,
        "height": 480,
        "fps": 30
      }
    }
  ],
  "services": [
    {
      "name": "detector",
      "type": "vision",
      "model": "yolox",
      "attributes": {
        "model_path": "/models/yolox_s.onnx",
        "confidence_threshold": 0.5
      },
      "depends_on": ["front_camera"]
    }
  ]
}
```

The structure is intentional:

- **name**: Unique identifier within type
- **type**: Component/service category (motor, camera, vision)
- **model**: Specific implementation (gpio motor, v4l2 camera, yolox detector)
- **attributes**: Implementation-specific settings
- **depends_on**: Resources this one needs (for initialization order)

### Runtime Reconfiguration

The `Reconfigure()` method on every resource enables runtime updates:

```go
// Change motor parameters without restarting
newConf := resource.NewConfig(map[string]any{
    "max_rpm": 250,  // Increase from 200
})
motor.Reconfigure(ctx, deps, newConf)
```

This matters for:

- **Tuning**: Adjust PID gains while watching behavior
- **Adaptation**: Change parameters based on conditions
- **Debugging**: Temporarily lower speeds, increase logging
- **Fleet management**: Push configuration updates to deployed robots

## Network Transparency (NWS/NWC)

One of Gorai's most powerful features is network transparency: the ability to use resources the same way whether they're local (in the same process) or remote (on another machine).

### Local vs Remote Resources

Consider a motor. When it's local, you call methods directly:

```go
motor := createMotor()
motor.SetPower(ctx, 0.5)  // Direct method call
```

When the motor runs on a different node (perhaps a microcontroller gateway), you still want the same interface. This is where NWS (Network Wrapper Server) and NWC (Network Wrapper Client) come in.

### NWS: Exposing Resources Over NATS

A Network Wrapper Server takes a local resource and exposes its methods over NATS:

```go
// On the node with the physical motor
motor := createMotor()

// Wrap it for network access
wrapper := nws.Wrap(node, motor, "gorai.motors.left_wheel")
```

Now method calls arrive as NATS messages. The wrapper subscribes to request topics, deserializes incoming requests, calls the actual resource method, serializes and returns the response.

### NWC: Consuming Remote Resources

A Network Wrapper Client creates a local proxy that forwards calls over NATS:

```go
// On a different node
motor := nwc.Motor(node, "gorai.motors.left_wheel")

// Use it like a local motor
motor.SetPower(ctx, 0.5)  // Becomes NATS request/reply
```

### Transparent Location Abstraction

The magic is that consuming code doesn't know (or care) if a resource is local or remote:

```go
func RunBehavior(motor motor.Motor) {
    // This function works with local or remote motors
    for i := 0; i < 10; i++ {
        motor.SetPower(ctx, float64(i) / 10)
        time.Sleep(100 * time.Millisecond)
    }
    motor.Stop(ctx)
}

// Works with local motor
localMotor := createMotor()
RunBehavior(localMotor)

// Works with remote motor
remoteMotor := nwc.Motor(node, "gorai.motors.left_wheel")
RunBehavior(remoteMotor)
```

### Performance Considerations

Network transparency has overhead:

- Serialization/deserialization for each call
- Network latency (microseconds locally, milliseconds across network)
- NATS message processing

For high-frequency operations (1kHz control loops), prefer local resources or the serial gateway pattern. Reserve NWS/NWC for:

- Infrequent operations (configuration, status checks)
- Operations where network latency is acceptable
- Cross-node coordination

---

With the mental model established—nodes, resources, distributed architecture, configuration, and network transparency—you're ready to understand Gorai's communication backbone. Chapter 4 dives deep into NATS.
