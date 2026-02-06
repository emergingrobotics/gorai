# Package Location Guide

This document describes the logic for where code belongs in the Gorai file structure.

## Directory Overview

```
gorai/
├── cmd/           # Entry points (CLI, binaries)
├── pkg/           # Core libraries (shared infrastructure)
├── components/     # Hardware abstractions (sensors, actuators)
├── driver/        # Low-level hardware access (GPIO, I2C, SPI)
├── services/       # Software capabilities (vision, navigation, ML)
├── internal/      # Private implementation details
├── api/           # API definitions (protobuf, OpenAPI)
├── examples/      # Example robots and configurations
├── docs/          # Documentation
└── archive/       # Archived code for future phases
```

## Decision Tree

```
Is it an entry point (main package)?
  └─ Yes → cmd/

Does it abstract physical hardware?
  └─ Yes → Is it low-level bus/protocol access?
           └─ Yes → driver/     (GPIO, I2C, SPI, serial)
           └─ No  → components/  (motor, sensor, camera, servo)

Is it a software capability/algorithm?
  └─ Yes → services/  (vision, navigation, SLAM, ML inference)

Is it shared infrastructure used by multiple packages?
  └─ Yes → pkg/  (config, nats, accel, registry, resource)

Is it private to this module only?
  └─ Yes → internal/
```

## Package Descriptions

### `cmd/` - Entry Points

**Purpose:** Executable entry points only. Minimal code.

**Contains:**
- CLI commands (`gorai validate`, `gorai run`, etc.)
- Main functions that wire together other packages

**Rules:**
- No business logic
- Import and orchestrate other packages
- Keep main.go minimal

**Example:**
```go
// cmd/gorai/main.go
func main() {
    if err := commands.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### `pkg/` - Core Libraries

**Purpose:** Shared infrastructure that multiple packages depend on.

**Contains:**
- `pkg/config/` - RDL parsing and configuration
- `pkg/nats/` - NATS client wrapper
- `pkg/mesh/` - Service discovery via NATS KV (runtime registration/discovery)
- `pkg/discovery/` - Dynamic discovery manager (auto-adoption, rules)
- `pkg/proxy/` - Remote component proxies (RemoteMotor, RemoteSensor)
- `pkg/accel/` - ML acceleration abstractions
- `pkg/registry/` - Component/service registry (compile-time registration)
- `pkg/resource/` - Base resource interfaces
- `pkg/dashboard/` - Web dashboard
- `pkg/pub/`, `pkg/sub/` - NATS pub/sub utilities

**Rules:**
- Must be reusable across multiple packages
- No hardware-specific code
- No robot-specific logic
- Stable interfaces

**When to add to pkg/:**
- Code is imported by 3+ packages
- It's infrastructure, not domain logic
- It could be useful in other projects

### `components/` - Hardware Abstractions

**Purpose:** Abstract hardware components for robot builders.

**Contains:**
- `components/motor/` - Motor control
- `components/sensor/` - Sensor abstractions
- `components/camera/` - Camera interfaces
- `components/servo/` - Servo control
- `components/arm/`, `components/gripper/` - Manipulators

**Structure:**
```
component/
├── component.go         # Base Component interface
├── motor/
│   ├── motor.go         # Motor interface
│   ├── dc/              # DC motor implementation
│   └── fake/            # Fake for testing
└── sensor/
    ├── sensor.go        # Sensor interface
    ├── ultrasonic/      # Ultrasonic implementation
    └── fake/            # Fake for testing
```

**Rules:**
- One interface per hardware type
- Implementations in subdirectories
- Always provide a `fake/` for testing
- Components use drivers, not raw hardware

**Component vs Driver:**
| Aspect | Component | Driver |
|--------|-----------|--------|
| Level | High | Low |
| User | Robot builder | Component implementer |
| Example | "Motor" | "PWM over GPIO" |
| Interface | Domain-specific | Protocol-specific |

### `driver/` - Hardware Access

**Purpose:** Low-level access to hardware buses and protocols.

**Contains:**
- `driver/gpio/` - GPIO pin control
- `driver/i2c/` - I2C bus communication
- `driver/spi/` - SPI bus communication
- `driver/serial/` - Serial/UART communication

**Rules:**
- Platform-specific implementations
- Minimal abstraction over hardware
- Used by components, not directly by users
- Handle Linux sysfs, character devices, etc.

**Example:**
```go
// driver/gpio/gpio.go
type Pin interface {
    High() error
    Low() error
    Read() (bool, error)
    SetPWM(dutyCycle float64) error
}
```

### `services/` - Software Capabilities

**Purpose:** Algorithms and processing pipelines that don't directly control hardware.

**Contains:**
- `services/vision/` - Computer vision processing
- `services/navigation/` - Path planning and navigation
- `services/slam/` - SLAM algorithms
- `services/mlmodel/` - ML model inference
- `services/behavior/` - Autonomous behaviors
- `services/coordinator/` - Behavior orchestration

**Rules:**
- Services consume components, not the other way around
- Services may use `pkg/accel/` for acceleration
- Stateless where possible
- Expose via NATS for inter-process use

**Service vs Component:**
| Aspect | Service | Component |
|--------|---------|-----------|
| Hardware | No direct access | Abstracts hardware |
| Examples | Vision, SLAM, ML | Camera, Motor, Sensor |
| Dependency | Uses components | Uses drivers |

### `internal/` - Private Code

**Purpose:** Implementation details not for external use.

**Rules:**
- Cannot be imported outside this module
- Use for code that shouldn't be public API
- Refactor to `pkg/` when stabilized

### `api/` - API Definitions

**Purpose:** Protocol definitions and generated code.

**Contains:**
- Protobuf definitions
- OpenAPI specs
- Generated Go code

## When to Create a Separate Repository

Create a new repo when:

1. **Independent release cycle** - The code needs to be versioned separately
2. **Different consumers** - Other projects need it without all of Gorai
3. **Language boundary** - Non-Go code (Python ML models, C drivers)
4. **Team boundary** - Different team owns it
5. **Size/complexity** - Large enough to be its own project

**Examples of separate repos:**

| Repo | Reason |
|------|--------|
| `gorai-rknn` | CGO bindings, special build requirements |
| `gorai-coral` | TPU-specific, different release cycle |
| `gorai-models` | Large ML models, different versioning |
| `gorai-ros-bridge` | ROS integration, optional dependency |

**Keep in main repo when:**
- Tightly coupled to core
- Same release cycle
- Same team
- Small/focused code

## Adding New Code: Examples

### Example 1: New Sensor Type

"I want to add a LiDAR sensor"

**Decision:** `components/sensor/lidar/`
- It's hardware abstraction → component
- It's a sensor type → under sensor

```
component/sensor/
├── sensor.go
├── lidar/
│   ├── lidar.go       # LiDAR interface
│   ├── rplidar/       # RPLidar implementation
│   └── fake/          # Fake for testing
└── ...
```

### Example 2: New ML Accelerator

"I want to add Hailo-8 NPU support"

**Decision:** `pkg/accel/hailo/`
- It's ML acceleration infrastructure → pkg/accel
- It's a new accelerator backend → subdirectory

```
pkg/accel/
├── accel.go           # Accelerator interface
├── cpu/               # CPU fallback
├── hailo/             # New Hailo implementation
└── tensor/            # Tensor utilities
```

### Example 3: Path Planning Algorithm

"I want to add A* path planning"

**Decision:** `services/navigation/astar/`
- It's an algorithm, not hardware → service
- It's navigation-related → under navigation

```
service/navigation/
├── navigation.go      # Navigation interface
├── astar/             # A* implementation
└── rrt/               # RRT implementation
```

### Example 4: Serial Protocol for GPS

"I want to add NMEA parsing for GPS"

**Decision:** `driver/serial/nmea/` or `components/sensor/gps/`
- NMEA parsing itself → `driver/serial/nmea/`
- GPS sensor abstraction → `components/sensor/gps/`

### Example 5: Shared Utility

"I want to add a rate limiter used by multiple packages"

**Decision:** `pkg/ratelimit/`
- Shared infrastructure → pkg
- Used by multiple packages → confirms pkg placement

### Example 6: Service Discovery

"I want cross-binary service discovery using NATS"

**Decision:** `pkg/mesh/`
- Shared infrastructure → pkg
- Uses NATS KV for persistent registry
- Provides client for registration, discovery, watching

```
pkg/mesh/
├── client.go           # Main client interface
├── registration.go     # Service registration + heartbeat
├── discovery.go        # Query services and channels
├── watcher.go          # Watch for changes
├── schema.go           # Schema registry
├── micro.go            # NATS micro service API
├── kv.go               # KV bucket management
└── types.go            # Core types (ServiceDescriptor, etc.)
```

### Example 7: Dynamic Device Adoption

"I want to auto-adopt devices discovered by gateways"

**Decision:** `pkg/discovery/` + `pkg/proxy/`
- Discovery manager → `pkg/discovery/`
- Proxy components → `pkg/proxy/`

```
pkg/discovery/
├── manager.go          # Discovery manager
├── source.go           # Discovery source interface
├── mesh_source.go      # Mesh-based discovery
├── rules.go            # Adoption rules
└── config.go           # Configuration types

pkg/proxy/
├── factory.go          # Proxy component factory
├── motor.go            # RemoteMotor proxy
├── sensor.go           # RemoteSensor proxy
└── camera.go           # RemoteCamera proxy
```

## Anti-Patterns

**Don't:**
- Put business logic in `cmd/`
- Put hardware code in `services/`
- Put algorithms in `driver/`
- Create `utils/` or `helpers/` packages (be specific)
- Import `internal/` from outside the module

**Do:**
- Keep packages focused on one responsibility
- Use clear, descriptive package names
- Provide interfaces at package level, implementations in subdirs
- Include `fake/` implementations for testing

## Summary Table

| Directory | Contains | Examples |
|-----------|----------|----------|
| `cmd/` | Entry points | CLI, main.go |
| `pkg/` | Shared infrastructure | config, nats, mesh, discovery, proxy, accel, registry |
| `components/` | Hardware abstractions | motor, sensor, camera |
| `driver/` | Low-level hardware | gpio, i2c, spi, serial |
| `services/` | Software capabilities | vision, navigation, slam |
| `internal/` | Private implementation | - |
| `api/` | Protocol definitions | protobuf, openapi |
| `examples/` | Example robots | gps-tracker, blinky |
| `docs/` | Documentation | - |
| `archive/` | Future-phase code | k3s, containers |
