# LLM Design Guide for Gorai

**Purpose:** This document contains everything an LLM needs to design new Gorai components and services without reading the entire codebase.

**For:** Claude, GPT-4, or other LLMs assisting with Gorai development.

---

## Quick Reference

### Module Path
```
github.com/gorai/gorai
```

### Key Packages
| Package | Import | Purpose |
|---------|--------|---------|
| Registry | `github.com/gorai/gorai/pkg/registry` | Compile-time component registration |
| Mesh | `github.com/gorai/gorai/pkg/mesh` | Runtime service discovery |
| NATS | `github.com/gorai/gorai/pkg/nats` | NATS client wrapper |
| Topics | `github.com/gorai/gorai/pkg/topics` | Topic naming conventions |
| Resource | `github.com/gorai/gorai/pkg/resource` | Base resource interfaces |

---

## Decision Tree: Where Does My Code Go?

```
Is it a CLI command?
  └─ Yes → cmd/gorai/commands/

Does it abstract physical hardware?
  └─ Yes → Is it low-level bus/protocol?
           └─ Yes → driver/     (GPIO, I2C, SPI, serial)
           └─ No  → components/ (motor, sensor, camera)

Is it a software algorithm/capability?
  └─ Yes → services/ (vision, navigation, behavior)

Is it shared infrastructure?
  └─ Yes → pkg/ (config, nats, mesh, registry)
```

---

## Component Pattern

Components abstract **physical hardware** (motors, sensors, cameras).

### File Structure
```
components/<subtype>/
├── <subtype>.go           # Interface definition
├── <model>/
│   └── <model>.go         # Implementation
└── fake/
    └── fake.go            # Fake for testing (required)
```

### Interface Template
```go
// components/motor/motor.go
package motor

import (
    "context"
)

// Motor controls a DC or brushless motor.
type Motor interface {
    // SetPower sets motor power from -1.0 (full reverse) to 1.0 (full forward).
    SetPower(ctx context.Context, power float64) error

    // GetPower returns the current power level.
    GetPower(ctx context.Context) (float64, error)

    // Stop immediately stops the motor.
    Stop(ctx context.Context) error

    // IsMoving returns true if the motor is currently moving.
    IsMoving(ctx context.Context) (bool, error)

    // Close releases resources.
    Close(ctx context.Context) error
}
```

### Implementation Template
```go
// components/motor/pwm/pwm.go
package pwm

import (
    "context"
    "sync"

    "github.com/gorai/gorai/components/motor"
    "github.com/gorai/gorai/pkg/registry"
)

func init() {
    registry.RegisterComponent("motor", "pwm", New)
}

// Config holds PWM motor configuration.
type Config struct {
    Pin      int     `json:"pin"`
    MinPulse float64 `json:"min_pulse,omitempty"` // default: 1000
    MaxPulse float64 `json:"max_pulse,omitempty"` // default: 2000
}

type pwmMotor struct {
    mu     sync.RWMutex
    config Config
    power  float64
}

// New creates a new PWM motor.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
    cfg := Config{
        MinPulse: 1000,
        MaxPulse: 2000,
    }
    // Parse config from conf map into cfg struct
    // ...

    return &pwmMotor{config: cfg}, nil
}

func (m *pwmMotor) SetPower(ctx context.Context, power float64) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.power = power
    // Send PWM signal...
    return nil
}

func (m *pwmMotor) GetPower(ctx context.Context) (float64, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.power, nil
}

func (m *pwmMotor) Stop(ctx context.Context) error {
    return m.SetPower(ctx, 0)
}

func (m *pwmMotor) IsMoving(ctx context.Context) (bool, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.power != 0, nil
}

func (m *pwmMotor) Close(ctx context.Context) error {
    return m.Stop(ctx)
}

// Compile-time interface check
var _ motor.Motor = (*pwmMotor)(nil)
```

### Fake Implementation Template
```go
// components/motor/fake/fake.go
package fake

import (
    "context"
    "sync"

    "github.com/gorai/gorai/components/motor"
    "github.com/gorai/gorai/pkg/registry"
)

func init() {
    registry.RegisterComponent("motor", "fake", New)
}

type fakeMotor struct {
    mu    sync.RWMutex
    power float64
}

func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
    return &fakeMotor{}, nil
}

func (m *fakeMotor) SetPower(ctx context.Context, power float64) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.power = power
    return nil
}

func (m *fakeMotor) GetPower(ctx context.Context) (float64, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.power, nil
}

func (m *fakeMotor) Stop(ctx context.Context) error {
    return m.SetPower(ctx, 0)
}

func (m *fakeMotor) IsMoving(ctx context.Context) (bool, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.power != 0, nil
}

func (m *fakeMotor) Close(ctx context.Context) error {
    return nil
}

var _ motor.Motor = (*fakeMotor)(nil)
```

---

## Service Pattern

Services are **software capabilities** (vision, navigation, behavior).

### File Structure
```
services/<subtype>/
├── <subtype>.go           # Interface definition
├── <model>/
│   └── <model>.go         # Implementation
└── fake/
    └── fake.go            # Fake for testing
```

### Interface Template
```go
// services/vision/vision.go
package vision

import (
    "context"
)

// Detection represents an object detection result.
type Detection struct {
    Label      string    `json:"label"`
    Confidence float64   `json:"confidence"`
    BoundingBox BBox     `json:"bounding_box"`
}

type BBox struct {
    X, Y, Width, Height int
}

// Vision provides object detection capabilities.
type Vision interface {
    // Detect runs object detection on an image.
    Detect(ctx context.Context, image []byte) ([]Detection, error)

    // Close releases resources.
    Close(ctx context.Context) error
}
```

### Implementation Template
```go
// services/vision/yolo/yolo.go
package yolo

import (
    "context"

    "github.com/gorai/gorai/pkg/registry"
    "github.com/gorai/gorai/services/vision"
)

func init() {
    registry.RegisterService("vision", "yolo", New)
}

type Config struct {
    ModelPath   string  `json:"model_path"`
    Confidence  float64 `json:"confidence,omitempty"` // default: 0.5
}

type yoloVision struct {
    config Config
}

func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
    cfg := Config{Confidence: 0.5}
    // Parse config...
    return &yoloVision{config: cfg}, nil
}

func (v *yoloVision) Detect(ctx context.Context, image []byte) ([]vision.Detection, error) {
    // Run inference...
    return nil, nil
}

func (v *yoloVision) Close(ctx context.Context) error {
    return nil
}

var _ vision.Vision = (*yoloVision)(nil)
```

---

## Registration Pattern

All components and services **MUST** self-register in `init()`:

```go
func init() {
    // For components
    registry.RegisterComponent("subtype", "model", New)

    // For services
    registry.RegisterService("subtype", "model", New)
}
```

### Constructor Signature (Required)
```go
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error)
```

### Accessing Dependencies
```go
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
    // Get a specific dependency by name
    cam, err := deps.Get("camera")
    if err != nil {
        return nil, err
    }
    camera := cam.(camera.Camera)

    // Get all dependencies of a type
    motors, err := deps.GetByType("motor")
    // ...
}
```

---

## NATS Topic Naming

### Format
```
gorai.<robot_id>.<component>.<message_type>
```

### Message Types
| Type | Purpose | Example |
|------|---------|---------|
| `data` | Sensor readings | `gorai.robot1.imu.data` |
| `command` | Control commands | `gorai.robot1.motor.command` |
| `state` | Component state | `gorai.robot1.motor.state` |
| `status` | Health/errors | `gorai.robot1.motor.status` |

### System Topics
```
gorai.<robot_id>.system.startup
gorai.<robot_id>.system.shutdown
gorai.<robot_id>.system.logs
gorai.<robot_id>.system.heartbeat
```

### Using the Topic Builder
```go
import "github.com/gorai/gorai/pkg/topics"

tb := topics.NewBuilder("robot-alpha")

// Component topics
tb.ComponentData("imu")      // "gorai.robot-alpha.imu.data"
tb.ComponentCommand("motor") // "gorai.robot-alpha.motor.command"
tb.ComponentState("motor")   // "gorai.robot-alpha.motor.state"

// System topics
tb.SystemStartup()           // "gorai.robot-alpha.system.startup"
tb.SystemHeartbeat()         // "gorai.robot-alpha.system.heartbeat"

// Wildcards
tb.All()                     // "gorai.robot-alpha.>"
tb.AllComponents("data")     // "gorai.robot-alpha.*.data"
```

---

## Mesh Service Discovery

For runtime service registration/discovery across independent processes.

### Registering a Service
```go
import (
    "github.com/nats-io/nats.go"
    "github.com/gorai/gorai/pkg/mesh"
)

nc, _ := nats.Connect("nats://localhost:4222")
client, _ := mesh.NewClient(nc)

reg, _ := client.Register(ctx, mesh.ServiceDescriptor{
    Name:    "motor-controller",
    Type:    mesh.TypeComponent,
    Subtype: "motor",
    Model:   "pwm",
    RobotID: "robot-alpha",
    Publishes: []string{
        "gorai.robot-alpha.motor.state",
    },
    Subscribes: []string{
        "gorai.robot-alpha.motor.command",
    },
})
defer reg.Deregister()
```

### Discovering Services
```go
// Find all motor components
motors, _ := client.FindServices(ctx, mesh.Query{
    RobotID: "robot-alpha",
    Subtype: "motor",
})

// Watch for changes
watcher, _ := client.WatchServices(ctx, mesh.Query{})
for event := range watcher.Events() {
    if event.Type == mesh.EventServiceJoined {
        fmt.Printf("Service joined: %s\n", event.Service.Name)
    }
}
```

### Registering Channels
```go
client.RegisterChannel(ctx, mesh.ChannelDescriptor{
    Subject:     "gorai.robot-alpha.motor.state",
    Schema:      "gorai.actuator.MotorState/v1",
    QoS:         mesh.QoSRetained,
    Direction:   mesh.DirectionPub,
    Description: "Motor state feedback",
    SampleRate:  "10Hz",
})
```

---

## RDL Configuration

Components are instantiated from JSON configuration.

### Format
```json
{
  "version": "3",
  "robot": {
    "name": "my-robot",
    "namespace": "gorai"
  },
  "nats": {
    "url": "nats://localhost:4222"
  },
  "components": [
    {
      "name": "left_motor",
      "type": "motor/pwm",
      "config": {
        "pin": 18,
        "min_pulse": 1000,
        "max_pulse": 2000
      }
    }
  ],
  "services": [
    {
      "name": "vision",
      "type": "vision/yolo",
      "config": {
        "model_path": "/models/yolov8n.onnx"
      },
      "depends_on": ["camera"]
    }
  ]
}
```

### Type Format
```
<subtype>/<model>
```

Examples:
- `motor/pwm` → components/motor/pwm/
- `camera/v4l2` → components/camera/v4l2/
- `vision/yolo` → services/vision/yolo/

---

## Common Interfaces

### Sensor (Read-Only Component)
```go
type Sensor interface {
    Readings(ctx context.Context) (map[string]any, error)
    Close(ctx context.Context) error
}
```

### Actuator (Controllable Component)
```go
type Actuator interface {
    IsMoving(ctx context.Context) (bool, error)
    Stop(ctx context.Context) error
    Close(ctx context.Context) error
}
```

### Reconfigurable
```go
type Reconfigurable interface {
    Reconfigure(ctx context.Context, deps Dependencies, conf Config) error
}
```

---

## Error Handling

### Standard Errors
```go
import "errors"

var (
    ErrNotFound      = errors.New("not found")
    ErrNotSupported  = errors.New("not supported")
    ErrTimeout       = errors.New("timeout")
    ErrClosed        = errors.New("closed")
    ErrInvalidConfig = errors.New("invalid configuration")
)
```

### Wrapping Errors
```go
return fmt.Errorf("failed to set power: %w", err)
```

---

## Concurrency Patterns

### Thread-Safe State
```go
type myComponent struct {
    mu    sync.RWMutex
    state State
}

func (c *myComponent) GetState() State {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.state
}

func (c *myComponent) SetState(s State) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.state = s
}
```

### Background Goroutines
```go
type myComponent struct {
    cancel context.CancelFunc
    done   chan struct{}
}

func New(...) (any, error) {
    ctx, cancel := context.WithCancel(context.Background())
    c := &myComponent{
        cancel: cancel,
        done:   make(chan struct{}),
    }
    go c.run(ctx)
    return c, nil
}

func (c *myComponent) run(ctx context.Context) {
    defer close(c.done)
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // Do periodic work
        }
    }
}

func (c *myComponent) Close(ctx context.Context) error {
    c.cancel()
    <-c.done
    return nil
}
```

---

## Testing

### Test File Location
```
components/motor/pwm/pwm_test.go
```

### Test Template
```go
package pwm

import (
    "context"
    "testing"

    "github.com/gorai/gorai/pkg/registry"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestSetPower(t *testing.T) {
    ctx := context.Background()
    conf := registry.Config{"pin": 18}

    m, err := New(ctx, nil, conf)
    require.NoError(t, err)

    motor := m.(*pwmMotor)
    defer motor.Close(ctx)

    err = motor.SetPower(ctx, 0.5)
    assert.NoError(t, err)

    power, err := motor.GetPower(ctx)
    assert.NoError(t, err)
    assert.Equal(t, 0.5, power)
}
```

---

## Checklist for New Components

- [ ] Interface defined in `components/<subtype>/<subtype>.go`
- [ ] Implementation in `components/<subtype>/<model>/<model>.go`
- [ ] Fake implementation in `components/<subtype>/fake/fake.go`
- [ ] `init()` calls `registry.RegisterComponent()`
- [ ] Constructor signature matches `func New(ctx, deps, conf) (any, error)`
- [ ] Implements the interface (compile-time check with `var _ Interface = (*impl)(nil)`)
- [ ] Thread-safe if stateful
- [ ] `Close()` cleans up resources
- [ ] Tests in `*_test.go`
- [ ] Config documented with JSON tags

## Checklist for New Services

- [ ] Interface defined in `services/<subtype>/<subtype>.go`
- [ ] Implementation in `services/<subtype>/<model>/<model>.go`
- [ ] Fake implementation in `services/<subtype>/fake/fake.go`
- [ ] `init()` calls `registry.RegisterService()`
- [ ] Constructor signature matches `func New(ctx, deps, conf) (any, error)`
- [ ] Implements the interface
- [ ] Thread-safe if stateful
- [ ] `Close()` cleans up resources
- [ ] Tests in `*_test.go`

---

## Anti-Patterns to Avoid

| Don't | Do Instead |
|-------|------------|
| Global mutable state | Pass state via constructor |
| Blocking in constructor | Start goroutines, return quickly |
| Ignoring context cancellation | Check `ctx.Done()` in loops |
| Panicking | Return errors |
| Hardcoding configuration | Use Config struct with JSON tags |
| Skipping the fake implementation | Always provide a fake for testing |
| Forgetting `Close()` | Always clean up resources |

---

## Quick Command Reference

```bash
# Build
go build ./...

# Test
go test ./...

# Run a specific test
go test -v ./components/motor/pwm -run TestSetPower

# Check for issues
go vet ./...

# Format
go fmt ./...

# Tidy dependencies
go mod tidy
```

---

---

## Gateway Integration

Gateways bridge hardware protocols (GSP/2, Modbus) to NATS. They should register with the mesh.

### Gateway Pattern

```go
// Gateway registers itself
meshClient.Register(ctx, mesh.ServiceDescriptor{
    Name:    "gsp-gateway",
    Type:    mesh.TypeService,
    Subtype: "gateway",
    Model:   "gsp",
    RobotID: robotID,
})

// Each device registers when connected
meshClient.Register(ctx, mesh.ServiceDescriptor{
    Name:    deviceID,
    Type:    mesh.TypeComponent,
    Subtype: "pwm-controller",
    Model:   "gsp-device",
    RobotID: robotID,
    Publishes: []string{"gsp." + deviceID + ".rx.sensor.>"},
    Subscribes: []string{"gsp." + deviceID + ".tx.command.>"},
})
```

### Subject Namespaces

| Layer | Prefix | Use |
|-------|--------|-----|
| Gateway | `gsp.<device>` | Raw device data |
| Gorai | `gorai.<robot>.<component>` | Normalized data |

---

## Dynamic Discovery

RDL can define discovery rules to auto-adopt devices not in the config.

### RDL Discovery Section

```json
{
  "discovery": {
    "enabled": true,
    "auto_adopt": true,
    "sources": [
      {"type": "gateway", "gateway": "gsp-gateway"},
      {"type": "mesh", "query": {"subtype": "motor"}}
    ],
    "rules": [
      {
        "match": {"capability": "PWM"},
        "adopt_as": {"type": "motor", "model": "remote-pwm"}
      }
    ]
  }
}
```

### Dynamic Dependencies

Services can depend on discovered resources:

```json
{
  "name": "patrol",
  "type": "behavior/patrol",
  "depends_on": [
    "camera",
    "@discovered:motor/*",
    "@discovered:sensor/imu/*"
  ]
}
```

| Pattern | Meaning |
|---------|---------|
| `@discovered:motor/*` | Any discovered motor |
| `@discovered:sensor/imu/*` | Any discovered IMU |
| `@discovered:*` | Any discovered resource |

### Proxy Components

Discovered devices are wrapped in proxies:

```go
type RemoteMotor struct {
    natsConn   *nats.Conn
    cmdSubject string
}

func (m *RemoteMotor) SetPower(ctx context.Context, power float64) error {
    cmd := map[string]any{"power": power}
    data, _ := json.Marshal(cmd)
    return m.natsConn.Publish(m.cmdSubject, data)
}
```

See [specs/dynamic-discovery.md](../specs/dynamic-discovery.md) for complete specification.

---

## Related Documents

| Document | Purpose |
|----------|---------|
| [CLAUDE.md](../CLAUDE.md) | Project overview for AI assistants |
| [specs/mesh-service-discovery.md](../specs/mesh-service-discovery.md) | Mesh system specification |
| [specs/dynamic-discovery.md](../specs/dynamic-discovery.md) | Dynamic discovery and auto-adoption |
| [specs/gorai-framework-specification.md](../specs/gorai-framework-specification.md) | Complete technical spec |
| [docs/PACKAGE-LOCATIONS.md](PACKAGE-LOCATIONS.md) | Where code belongs |
| [specs/robot-definition-language.md](../specs/robot-definition-language.md) | RDL JSON format |
