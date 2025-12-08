# Mental Model & Architecture

This chapter establishes the conceptual framework for understanding Gorai. Before diving into code, you need to understand how Gorai thinks about robots. This mental model will help everything else make sense.

## The Big Picture

A Gorai robot system consists of three hardware layers connected by NATS messaging:

```
┌─────────────────────────────────────────────────────────────┐
│                    Primary Compute                           │
│  Linux SBC running standard Go                              │
│  (Raspberry Pi 5, Orange Pi 5, Jetson Orin)                 │
│  - Runs main robot logic and AI inference                   │
│  - Connects to NATS server (often local)                    │
│  - Manages sensors, actuators, services                     │
└─────────────────────────────────────────────────────────────┘
                              │ NATS
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Secondary Nodes                           │
│  Smaller Linux boards for distributed processing            │
│  (Pi Zero 2 W, Orange Pi Zero 2)                            │
│  - Dedicated camera processing                              │
│  - Sensor fusion nodes                                      │
│  - Actuator controllers                                     │
└─────────────────────────────────────────────────────────────┘
                              │ Serial (GSP)
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Microcontrollers                          │
│  TinyGo peripherals for real-time I/O                       │
│  (RP2040, ESP32)                                            │
│  - Motor PWM generation                                     │
│  - Sensor polling                                           │
│  - Real-time control loops                                  │
└─────────────────────────────────────────────────────────────┘
```

All nodes communicate through NATS, creating a unified messaging fabric. Microcontrollers connect via the Gorai Serial Protocol (GSP), which gets bridged to NATS by a gateway process.

## The Five-Layer Architecture

Gorai's software architecture has five layers:

| Layer | Purpose | Examples |
|-------|---------|----------|
| **Application** | User code and robot logic | Your robot application |
| **Communication** | NATS messaging | Pub/sub, request/reply, JetStream |
| **Resource** | Component/service abstractions | Motors, sensors, vision, behaviors |
| **Acceleration** | Hardware-accelerated inference | RockchipNPU, CUDA, CoralTPU |
| **Hardware** | Physical device interfaces | GPIO, I2C, SPI, Serial, USB |

---

## Resources: The Foundation

**Everything in Gorai is a Resource.** This is the most important concept to understand.

A Resource is any entity that:
- Has a unique name
- Can be reconfigured at runtime
- Can be gracefully closed
- Can respond to custom commands

### The Resource Interface

The base Resource interface defines the contract all components and services must implement:

```go
type Resource interface {
    // Name returns the unique identifier for this resource
    Name() Name

    // Reconfigure updates the resource with new configuration
    // This enables hot-reloading without restarting the robot
    Reconfigure(ctx context.Context, deps Dependencies, conf Config) error

    // DoCommand provides an extensibility point for custom operations
    // Any command not covered by the standard interface goes here
    DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error)

    // Close gracefully shuts down the resource
    Close(ctx context.Context) error
}
```

### Resource Naming

Every resource has a hierarchical name with four parts:

```
namespace:type:subtype/name
```

| Part | Description | Examples |
|------|-------------|----------|
| **Namespace** | Organization scope | `gorai`, `mycompany` |
| **Type** | Resource category | `component`, `service` |
| **Subtype** | Specific kind | `motor`, `camera`, `vision` |
| **Name** | Instance identifier | `left_motor`, `front_camera` |

Examples:
- `gorai:component:motor/left_wheel`
- `gorai:component:camera/front`
- `gorai:service:vision/detector`
- `gorai:service:behavior/navigation`

Resource names convert directly to NATS topics:
- `gorai:component:motor/left_wheel` → `gorai.component.motor.left_wheel`

### Why Resources Matter

The Resource abstraction provides:

1. **Uniform Interface**: All robot parts look the same to the framework
2. **Hot Reconfiguration**: Change settings without restarting
3. **Dependency Injection**: Resources can depend on other resources
4. **Network Transparency**: Local and remote resources work identically
5. **Extensibility**: DoCommand handles anything not in the standard API

---

## Components: Hardware Abstractions

Components are Resources that abstract physical hardware. They represent things you can touch—motors, sensors, cameras.

### Component Categories

Gorai defines five fundamental component categories:

```
Resource (base interface)
└── Component
    ├── Sensor     - Observes the environment
    ├── Actuator   - Changes the environment
    ├── Link       - Communicates with other systems
    ├── Power      - Manages energy
    └── Space      - Represents physical volumes
```

### Sensors

Sensors read data from the physical world. The base sensor interface is simple:

```go
type Sensor interface {
    Resource
    Readings(ctx context.Context) (map[string]any, error)
}
```

Gorai provides specialized sensor interfaces:

| Sensor Type | Key Methods | Use Case |
|-------------|-------------|----------|
| **IMU** | `LinearAcceleration()`, `AngularVelocity()`, `Orientation()` | Inertial measurement |
| **GPS** | `Position()`, `LinearVelocity()`, `Accuracy()`, `Fix()` | Positioning |
| **Encoder** | `Position()`, `ResetPosition()`, `Properties()` | Motor feedback |
| **RangeFinder** | `Range()`, `Properties()` | Distance measurement |
| **Camera** | `Image()`, `Stream()`, `Properties()` | Vision |

**Example: IMU Sensor Interface**

```go
type IMU interface {
    Sensor

    // LinearAcceleration returns acceleration in m/s² for each axis
    LinearAcceleration(ctx context.Context) (x, y, z float64, err error)

    // AngularVelocity returns rotation rate in rad/s for each axis
    AngularVelocity(ctx context.Context) (x, y, z float64, err error)

    // Orientation returns the current orientation as a quaternion
    Orientation(ctx context.Context) (x, y, z, w float64, err error)
}
```

### Actuators

Actuators change the physical world. The base actuator interface provides stop functionality:

```go
type Actuator interface {
    Resource
    IsMoving(ctx context.Context) (bool, error)
    Stop(ctx context.Context) error
}
```

| Actuator Type | Key Methods | Use Case |
|---------------|-------------|----------|
| **Motor** | `SetPower()`, `SetVelocity()`, `GoTo()`, `GoFor()` | Rotary motion |
| **Base** | `SetVelocity()`, `MoveStraight()`, `Spin()` | Mobile platforms |
| **Arm** | `MoveToPosition()`, `MoveToJointPositions()` | Manipulation |
| **Gripper** | `Open()`, `Grab()`, `IsOpen()` | End effectors |

**Example: Motor Interface**

```go
type Motor interface {
    Actuator

    // SetPower sets motor power from -1.0 (full reverse) to 1.0 (full forward)
    SetPower(ctx context.Context, power float64) error

    // SetVelocity sets target velocity in RPM
    SetVelocity(ctx context.Context, velocity float64) error

    // GoTo moves to an absolute position at given velocity
    GoTo(ctx context.Context, position, velocity float64) error

    // GoFor rotates for a number of revolutions at given RPM
    GoFor(ctx context.Context, rpm, revolutions float64) error

    // GetPosition returns current position in revolutions
    GetPosition(ctx context.Context) (float64, error)

    // Properties returns motor capabilities
    Properties(ctx context.Context) (Properties, error)
}
```

### Links

Links provide communication between nodes and external systems:

```go
type Link interface {
    Resource

    // Type returns the link type (serial, IP, NATS, CAN, I2C, SPI)
    Type() LinkType

    // Direction returns whether communication is bidirectional or broadcast
    Direction() LinkDirection

    // IsConnected returns the current connection status
    IsConnected(ctx context.Context) (bool, error)

    // Connect establishes the connection
    Connect(ctx context.Context) error

    // Disconnect closes the connection
    Disconnect(ctx context.Context) error
}
```

| Link Type | Direction | Use Case |
|-----------|-----------|----------|
| **SerialLink** | Bidirectional | UART, RS-232, RS-485 to microcontrollers |
| **IPLink** | Bidirectional | TCP/UDP to networked devices |
| **NATSLink** | Broadcast | Pub/sub to other NATS nodes |
| **CANLink** | Broadcast | CAN bus to motor controllers |
| **I2CLink** | Bidirectional | I2C sensors and actuators |
| **SPILink** | Bidirectional | High-speed SPI devices |

### Power

Power components manage energy:

```go
type Power interface {
    Resource
    GetCapacity(ctx context.Context) (float64, error)   // Watt-hours
    GetLevel(ctx context.Context) (float64, error)       // 0.0 to 1.0
    GetVoltage(ctx context.Context) (float64, error)     // Volts
    GetCurrent(ctx context.Context) (float64, error)     // Amps
    IsCharging(ctx context.Context) (bool, error)
}
```

### Space

Space components represent physical volumes:

```go
type Space interface {
    Resource
    GetVolume(ctx context.Context) (float64, error)    // Cubic meters
    GetBounds(ctx context.Context) (Bounds, error)     // 3D bounding box
    GetContents(ctx context.Context) ([]string, error) // What's inside
    IsEmpty(ctx context.Context) (bool, error)
}
```

Space types: Container (cargo bays, hoppers), Workspace (robot envelopes), Zone (safety areas)

---

## Services: Software Capabilities

Services are Resources that provide software capabilities—algorithms, processing pipelines, AI inference. Unlike components, services don't directly represent hardware.

### Service Architecture

Services consume data from components and provide higher-level functionality:

```
┌─────────────────────────────────────────────────────────┐
│                      Services                            │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐   │
│  │ Vision  │  │  SLAM   │  │ Motion  │  │Navigation│   │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘   │
│       │            │            │            │          │
└───────┼────────────┼────────────┼────────────┼──────────┘
        │            │            │            │
┌───────┼────────────┼────────────┼────────────┼──────────┐
│       ▼            ▼            ▼            ▼          │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐   │
│  │ Camera  │  │  LiDAR  │  │  Motor  │  │   GPS   │   │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘   │
│                      Components                         │
└─────────────────────────────────────────────────────────┘
```

### Vision Service

Provides object detection and classification:

```go
type VisionService interface {
    Service

    // Detections returns detected objects in an image
    Detections(ctx context.Context, img image.Image, extra map[string]any) ([]Detection, error)

    // Classifications returns image classifications
    Classifications(ctx context.Context, img image.Image, n int, extra map[string]any) ([]Classification, error)

    // GetObjectPointClouds returns 3D segmented objects
    GetObjectPointClouds(ctx context.Context, cameraName string, extra map[string]any) ([]Object3D, error)
}

type Detection struct {
    BoundingBox Rectangle  // Where in the image
    Label       string     // What it is ("person", "cup", "obstacle")
    Confidence  float64    // 0.0 to 1.0
}
```

### ML Model Service

Generic tensor inference:

```go
type MLModelService interface {
    Service

    // Infer runs model inference on input tensors
    Infer(ctx context.Context, inputs map[string]Tensor) (map[string]Tensor, error)

    // Metadata returns model information
    Metadata(ctx context.Context) (Metadata, error)
}

type Tensor struct {
    Name     string
    Shape    []int64        // e.g., [1, 224, 224, 3]
    DataType DataType       // Float32, Int8, etc.
    Data     []byte         // Flattened tensor data
}
```

Supported frameworks: ONNX, TensorFlow Lite, RKNN (Rockchip NPU)

### SLAM Service

Simultaneous Localization and Mapping:

```go
type SLAMService interface {
    Service

    // GetPosition returns current pose estimate
    GetPosition(ctx context.Context) (Pose, error)

    // GetPointCloudMap returns the map as a point cloud
    GetPointCloudMap(ctx context.Context) ([]byte, error)

    // GetInternalState returns serialized state for persistence
    GetInternalState(ctx context.Context) ([]byte, error)
}
```

### Motion Service

Motion planning and execution:

```go
type MotionService interface {
    Service

    // Move plans and executes motion to a destination
    Move(ctx context.Context, componentName string, destination Pose, constraints *Constraints) (bool, error)

    // GetPose returns a component's pose in a reference frame
    GetPose(ctx context.Context, componentName, referenceFrame string) (Pose, error)

    // StopPlan cancels the current motion plan
    StopPlan(ctx context.Context) error
}
```

### Navigation Service

Autonomous navigation:

```go
type NavigationService interface {
    Service

    // GetMode returns current navigation mode
    GetMode(ctx context.Context) (Mode, error)  // Manual, Waypoint, Explore

    // SetMode changes navigation mode
    SetMode(ctx context.Context, mode Mode) error

    // GetWaypoints returns the waypoint list
    GetWaypoints(ctx context.Context) ([]Waypoint, error)

    // AddWaypoint adds a navigation target
    AddWaypoint(ctx context.Context, waypoint Waypoint) error
}
```

---

## Behaviors: Robot Decision Making

**Behaviors are the brain of your robot.** They implement decision-making logic—what to do, when to do it, and how to respond to the environment.

### What is a Behavior?

A Behavior is a service that:
- Makes decisions based on sensor data
- Controls actuators to achieve goals
- Can be started, stopped, and monitored
- Exposes "derived sensors" (computed data)

```go
type BehaviorService interface {
    Service

    // Start begins behavior execution
    Start(ctx context.Context) error

    // Stop halts behavior execution
    Stop(ctx context.Context) error

    // IsRunning returns execution status
    IsRunning(ctx context.Context) (bool, error)

    // GetState returns current behavior state
    GetState(ctx context.Context) (*State, error)

    // SetGoal sets the behavior's objective
    SetGoal(ctx context.Context, goal *Goal) error

    // Tick manually advances the behavior (for external scheduling)
    Tick(ctx context.Context) (*TickResult, error)

    // GetDerivedSensors returns virtual sensors this behavior exposes
    GetDerivedSensors(ctx context.Context) ([]Name, error)
}
```

### Behavior Types

Gorai supports multiple behavior implementation styles:

| Type | Description | Use Case |
|------|-------------|----------|
| **BehaviorTree** | Hierarchical decision trees | Complex sequences with fallbacks |
| **StateMachine** | Finite state machines | Clear state transitions |
| **Subsumption** | Priority-based layers | Reactive robotics |
| **Utility** | Utility-based selection | Multi-objective decisions |
| **AIAgent** | ML model-powered decisions | Learned behaviors |
| **LLMAgent** | LLM-powered reasoning | Natural language goals |

### Behavior State

Behaviors maintain rich state information:

```go
type State struct {
    Status      Status         // Idle, Running, Success, Failure, Canceled
    CurrentNode string         // Current position in behavior tree/state machine
    Variables   map[string]any // Blackboard data
    LastTick    time.Time
    TickCount   int64
    Duration    time.Duration
    Error       string
}

type Goal struct {
    ID         string
    Type       string         // "navigate", "pick", "patrol", etc.
    Target     map[string]any // Goal-specific parameters
    Priority   int
    Timeout    time.Duration
}
```

### Derived Sensors: Virtual Sensor Data

Behaviors can expose **derived sensors**—computed data that looks like sensor readings but comes from behavior processing:

| Derived Sensor | Description |
|----------------|-------------|
| `estimated_pose` | Fused position from multiple sensors |
| `tracked_objects` | Objects being tracked over time |
| `anomalies` | Detected unusual situations |
| `scene_context` | Semantic understanding of the scene |
| `navigation_status` | Current navigation progress |
| `reasoning_trace` | LLM reasoning steps |

```go
type TrackedObject struct {
    ID          string
    Class       string     // "person", "robot", "obstacle"
    Confidence  float64
    BoundingBox Rectangle
    Position3D  Vector3
    Velocity    Vector3
    Age         time.Duration
    LastSeen    time.Time
}

type EstimatedPose struct {
    Position    Vector3
    Orientation Quaternion
    Covariance  []float64  // Uncertainty
    FrameID     string
    Sources     []string   // IMU, GPS, SLAM, etc.
}
```

Other components can subscribe to derived sensors just like physical sensors:
```
gorai.service.behavior.navigator.derived.estimated_pose
gorai.service.behavior.tracker.derived.tracked_objects
```

---

## AI/ML in Behaviors

Gorai treats AI/ML as a first-class citizen in behavior design. There are two primary patterns:

### ML-Powered Behaviors (AIBehavior)

Use trained machine learning models for decision making:

```go
type AIBehavior interface {
    BehaviorService

    // GetModel returns the model being used
    GetModel(ctx context.Context) (string, error)

    // GetConfidence returns decision confidence
    GetConfidence(ctx context.Context) (float64, error)

    // GetExplanation provides interpretability
    GetExplanation(ctx context.Context) (string, error)

    // Learn provides feedback for online learning
    Learn(ctx context.Context, feedback *Feedback) error

    // GetModelMetadata returns model information
    GetModelMetadata(ctx context.Context) (*ModelMetadata, error)
}

type ModelMetadata struct {
    Name            string
    Version         string
    Framework       string  // "tensorflow", "pytorch", "onnx"
    Accelerator     string  // "cpu", "npu", "cuda"
    InputShape      []int64
    OutputShape     []int64
    LearningEnabled bool
}
```

**ML Behavior Patterns:**

| Pattern | Model Type | Use Case |
|---------|-----------|----------|
| **Reinforcement Learning** | Policy networks | Navigation, control |
| **Imitation Learning** | Behavior cloning | Manipulation from demos |
| **Classification** | Classifiers | Anomaly detection |
| **Object Detection** | YOLO, SSD | Target tracking |
| **Semantic Segmentation** | U-Net, DeepLab | Terrain analysis |

### LLM-Powered Behaviors (LLMBehavior)

Use Large Language Models for reasoning and planning:

```go
type LLMBehavior interface {
    BehaviorService

    // GetLLMProvider returns the LLM provider
    GetLLMProvider(ctx context.Context) (string, error)  // "anthropic", "openai", "local"

    // GetLLMModel returns the model name
    GetLLMModel(ctx context.Context) (string, error)

    // SendPrompt sends a prompt and gets a response
    SendPrompt(ctx context.Context, prompt string) (string, error)

    // GetReasoningTrace returns the reasoning steps
    GetReasoningTrace(ctx context.Context) ([]ReasoningStep, error)

    // SetSystemPrompt configures the system prompt
    SetSystemPrompt(ctx context.Context, prompt string) error
}

type ReasoningStep struct {
    Step        int
    Thought     string  // "I need to find the red ball"
    Action      string  // "search_area"
    ActionInput string  // "living_room"
    Observation string  // "Found red ball at coordinates (2.5, 1.2)"
    TokensUsed  int
}
```

**LLM Behavior Capabilities:**

| Capability | Description |
|------------|-------------|
| **Goal Interpretation** | "Find the red ball" → navigation commands |
| **Contextual Reasoning** | "The door is closed, I need to open it first" |
| **Dynamic Planning** | Breaking complex tasks into steps |
| **Error Recovery** | Reasoning about what went wrong |
| **Human Interaction** | Natural language conversation |
| **Tool Selection** | Choosing the right robot capability |

**Example LLM Behavior Configuration:**

```json
{
  "name": "assistant_behavior",
  "model": "gorai:service:behavior/llm_agent",
  "config": {
    "llm_provider": "anthropic",
    "llm_model": "claude-3-sonnet",
    "system_prompt": "You are a helpful robot assistant...",
    "available_tools": ["navigate_to", "pick_object", "place_object", "search_area"],
    "max_reasoning_steps": 10,
    "safety_constraints": ["never_enter_restricted_zones", "ask_before_moving_heavy_objects"]
  }
}
```

### AI Safety in Behaviors

Gorai provides safety mechanisms for AI-powered behaviors:

| Safety Feature | Description |
|----------------|-------------|
| **Action Allowlists** | Only permitted actions can execute |
| **Sensor Grounding** | Decisions must reference actual sensor data |
| **Timeout Limits** | Behaviors must respond within time limits |
| **Fallback Behaviors** | Default to safe behavior on failure |
| **Rate Limiting** | Prevent runaway inference loops |
| **On-Device Inference** | Keep data local for privacy |

---

## Coordinators: Mission Orchestration

**Coordinators orchestrate behaviors**—they don't control components directly. A Coordinator manages multi-phase missions by sequencing and parallelizing behaviors.

### The Coordinator Service

```go
type CoordinatorService interface {
    Service

    // Start begins mission execution
    Start(ctx context.Context) error

    // Stop halts the mission
    Stop(ctx context.Context) error

    // IsRunning returns execution status
    IsRunning(ctx context.Context) (bool, error)

    // GetManagedBehaviors returns behaviors being orchestrated
    GetManagedBehaviors(ctx context.Context) ([]string, error)

    // SetMission sets the high-level mission
    SetMission(ctx context.Context, mission *Mission) error

    // GetProgress returns mission progress
    GetProgress(ctx context.Context) (*Progress, error)
}
```

### Missions and Phases

A Mission is a sequence of Phases, each containing Behaviors:

```go
type Mission struct {
    ID          string
    Name        string
    Description string
    Phases      []Phase
    Priority    int
    Timeout     time.Duration
    OnFailure   FailurePolicy  // Abort, Retry, Skip, Fallback
}

type Phase struct {
    Name        string
    Behaviors   []BehaviorRef  // Behaviors to run
    Parallel    bool           // Run behaviors in parallel?
    Condition   string         // Optional precondition
    OnComplete  string         // Next phase on success
    OnFailure   string         // Next phase on failure
    Timeout     time.Duration
}

type BehaviorRef struct {
    Name     string  // Behavior service name
    Goal     *Goal   // Goal to set
    Required bool    // Phase fails if this behavior fails
}
```

**Example Mission:**

```json
{
  "id": "delivery_mission_001",
  "name": "Package Delivery",
  "phases": [
    {
      "name": "navigate_to_pickup",
      "behaviors": [
        {"name": "navigator", "goal": {"type": "goto", "target": {"location": "warehouse"}}, "required": true}
      ]
    },
    {
      "name": "pick_package",
      "behaviors": [
        {"name": "vision_tracker", "goal": {"type": "find", "target": {"object": "package"}}, "required": true},
        {"name": "manipulator", "goal": {"type": "pick", "target": {"object": "detected_package"}}, "required": true}
      ]
    },
    {
      "name": "deliver",
      "behaviors": [
        {"name": "navigator", "goal": {"type": "goto", "target": {"location": "destination"}}, "required": true},
        {"name": "manipulator", "goal": {"type": "place", "target": {"location": "delivery_spot"}}, "required": true}
      ]
    }
  ],
  "on_failure": "fallback",
  "fallback_mission": "return_to_base"
}
```

### AI-Powered Coordinators

Coordinators can use LLMs for mission generation:

```go
type AICoordinator interface {
    CoordinatorService

    // GenerateMission creates a mission from natural language
    GenerateMission(ctx context.Context, description string) (*Mission, error)

    // AdaptMission modifies the mission based on situation changes
    AdaptMission(ctx context.Context, situation string) error

    // ExplainPlan provides a natural language explanation
    ExplainPlan(ctx context.Context) (string, error)
}
```

This enables high-level commands like:
- "Deliver the package to room 302"
- "Patrol the warehouse and report any anomalies"
- "Find the lost robot and guide it back"

---

## Communication Patterns

Gorai provides three messaging patterns through NATS:

### Topics (Pub/Sub)

For sensor streams and telemetry:

```
gorai.component.motor.left_wheel.position    # Motor position updates
gorai.component.sensor.imu.readings          # IMU data stream
gorai.service.behavior.navigator.derived.pose # Estimated pose
```

**Quality of Service Levels:**

| Level | Description | Use Case |
|-------|-------------|----------|
| **BestEffort** | Fire-and-forget | High-frequency sensor data |
| **Reliable** | JetStream with acks | Commands that must arrive |
| **Retained** | Last value retained | Configuration, state |
| **History** | Keep last N messages | Replay, debugging |

### Services (Request/Reply)

For synchronous RPC:

```go
// Client sends request, waits for response
result, err := motorClient.GetPosition(ctx)
```

### Actions (Long-Running Tasks)

For operations that take time and need feedback:

```go
// Start action, get handle
handle := navigator.MoveTo(ctx, destination)

// Monitor progress
for feedback := range handle.Feedback() {
    fmt.Printf("Progress: %.0f%%\n", feedback.Progress*100)
}

// Get final result
result, err := handle.Wait(ctx)
```

---

## Configuration

Robots are configured via JSON:

```json
{
  "components": [
    {
      "name": "left_motor",
      "type": "component",
      "subtype": "motor",
      "model": "gorai:driver:gpio_motor",
      "config": {
        "pin_forward": 18,
        "pin_reverse": 19,
        "pin_pwm": 12
      }
    },
    {
      "name": "front_camera",
      "type": "component",
      "subtype": "camera",
      "model": "gorai:driver:v4l2",
      "config": {
        "device": "/dev/video0",
        "width": 640,
        "height": 480
      }
    }
  ],
  "services": [
    {
      "name": "detector",
      "type": "service",
      "subtype": "vision",
      "model": "gorai:service:vision/rknn",
      "config": {
        "model_path": "/models/yolov8n.rknn",
        "accelerator": "npu"
      },
      "depends_on": ["front_camera"]
    },
    {
      "name": "navigator",
      "type": "service",
      "subtype": "behavior",
      "model": "gorai:service:behavior/nav_behavior",
      "config": {
        "behavior_type": "behavior_tree"
      },
      "depends_on": ["detector", "left_motor", "right_motor"]
    }
  ]
}
```

### Hot Reconfiguration

Configuration can be updated at runtime without restarting:

```go
// The framework calls Reconfigure() on the resource
func (m *Motor) Reconfigure(ctx context.Context, deps Dependencies, conf Config) error {
    // Update settings from new config
    m.maxPower = conf.GetFloat("max_power", 1.0)
    return nil
}
```

This enables:
- Tuning PID parameters live
- Switching AI models without restart
- Adjusting sensor polling rates
- Changing behavior goals on the fly

---

## Putting It All Together

Here's how the pieces fit in a real robot:

```
┌─────────────────────────────────────────────────────────────────┐
│                         Coordinator                              │
│                    (Mission Orchestration)                       │
│                                                                  │
│    Mission: "Patrol and Report"                                 │
│    ├── Phase 1: patrol_area                                     │
│    │   └── Behavior: patrol_behavior                            │
│    ├── Phase 2: investigate_anomaly                             │
│    │   └── Behavior: investigation_behavior                     │
│    └── Phase 3: report_findings                                 │
│        └── Behavior: reporting_behavior                         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                          Behaviors                               │
│                    (Decision Making)                             │
│                                                                  │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │ patrol_behavior │  │  llm_reasoner   │  │ anomaly_detector│ │
│  │ (BehaviorTree)  │  │  (LLMAgent)     │  │   (AIAgent)     │ │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘ │
│           │                    │                    │           │
│           └────────────────────┴────────────────────┘           │
│                    Derived Sensors:                              │
│                    - estimated_pose                             │
│                    - tracked_objects                            │
│                    - anomalies                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                          Services                                │
│                  (Processing Pipelines)                          │
│                                                                  │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐           │
│  │ Vision  │  │  SLAM   │  │ Motion  │  │MLModel  │           │
│  │ Service │  │ Service │  │ Service │  │ Service │           │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘           │
└───────┼────────────┼────────────┼────────────┼───────────────────┘
        │            │            │            │
        ▼            ▼            ▼            ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Components                               │
│                    (Hardware Abstraction)                        │
│                                                                  │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐           │
│  │ Camera  │  │  LiDAR  │  │  Motor  │  │   IMU   │           │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘           │
│                                                                  │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐           │
│  │   GPS   │  │ Encoder │  │ Battery │  │ Serial  │           │
│  │         │  │         │  │ (Power) │  │ (Link)  │           │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘           │
└─────────────────────────────────────────────────────────────────┘
        │            │            │            │
        ▼            ▼            ▼            ▼
┌─────────────────────────────────────────────────────────────────┐
│                          Hardware                                │
│  Physical sensors, actuators, and communication interfaces       │
└─────────────────────────────────────────────────────────────────┘
```

This mental model—Resources divided into Components and Services, with Behaviors for decision making and Coordinators for mission orchestration—is the foundation for everything in Gorai.
