## 2.2 Core Concepts

Three concepts form Gorai's foundation: Nodes, Resources, and the Resource Model. Master these, and the framework becomes intuitive.

> **Design Note**: These abstractions exist to make your life easier, not to impress. If something seems unnecessarily complex, file an issue—we probably got it wrong. The goal is concepts you can explain to a teammate in two minutes.

### The Object Model - Explained Simply

Before diving into details, let's establish the fundamental mental model. Think of Gorai like building with LEGO blocks for robots. Every single piece in the system—whether it's a camera, a motor, an AI vision system, or a communication channel—is built from the same fundamental building block called a **Resource**.

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

**Components** represent hardware (or virtual aggregations of hardware), organized into 5 categories:

| Category | What It Does | Examples |
|----------|--------------|----------|
| **Sensor** | Observes the world (read-only) | Camera, GPS, temperature sensor |
| **Actuator** | Changes the world (does stuff) | Motor, robotic arm, gripper |
| **Power** | Manages energy | Battery, power supply |
| **Space** | Virtual container on robot | Ballast tank, cargo bay with door |
| **Link** | Extra communication channel | Serial to MCU, radio telemetry |

**Note**: NATS is assumed infrastructure, not a "Link." A Space is a virtual abstraction coordinating other components (valves, doors, sensors). A Link bridges to devices that can't connect to NATS (microcontrollers, radios).

**Services** are software that processes data or makes decisions: Vision, SLAM, Navigation, and Behavior.

This design enables **uniform treatment** (management code works on everything), **hot reconfiguration** (change settings without rebooting), **discoverability** (find components by type), and **extensibility** (add new components by implementing interfaces).

Now let's dive into the details.

### 2.2.1 Nodes

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

The `FullName()` method returns the complete identifier:

```go
n.FullName() // Returns "robot1.sensors"
```

*Cross-reference: See Chapter 3 for how nodes communicate via NATS.*

### 2.2.2 Resources

A **Resource** is anything managed by Gorai: a motor, a camera, a navigation service, a sensor. All resources implement a common interface defined in `pkg/resource/resource.go`:

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

Creating names in code:

```go
name := resource.NewComponentName("gorai", "sensor", "cpu_temp")
// gorai:component:sensor/cpu_temp
```

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

This distinction matters for organization but not for the core interface—both are Resources.

*Cross-reference: Chapters 4-6 detail component types; Chapter 7 covers services.*

### 2.2.3 The Resource Model

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

### 2.2.4 Why These Abstractions?

You might wonder: why not just write code that directly controls hardware?

**Consistency**: Once you learn to use one sensor, you know how to use any sensor. Same for motors, cameras, services.

**Testing**: The interface boundary is where you inject fakes. `Readings()` returns a map—easy to mock. No hardware needed for unit tests.

**Documentation**: Interfaces are documentation. `type Sensor interface` tells you exactly what a sensor can do.

**Contribution**: New developers can add components by implementing known interfaces. The patterns are learnable; the codebase is approachable.

The goal isn't abstraction for its own sake—it's making the codebase navigable for contributors and maintainable over time.
