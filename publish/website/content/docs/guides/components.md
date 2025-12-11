---
title: "Working with Components"
description: "Build and use sensors, actuators, and other hardware abstractions"
weight: 10
---

# Working with Components

Components are Resources that interface with robot hardware. They abstract physical devices behind consistent interfaces, organized into five categories based on their relationship with the physical world.

## The Resource Interface

All components implement the base Resource interface:

```go
type Resource interface {
    Name() resource.Name
    Reconfigure(ctx context.Context, deps Dependencies, conf Config) error
    DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error)
    Close(ctx context.Context) error
}
```

## Component Categories

Components are organized by what they do:

| Category | What It Does | Interface |
|----------|--------------|-----------|
| **Sensor** | Observes the world (read-only) | `Readings()` |
| **Actuator** | Changes the world | `IsMoving()`, `Stop()` |
| **Power** | Manages energy | `GetCapacity()`, `GetLevel()` |
| **Space** | Virtual container on robot | `GetVolume()`, `GetComponents()` |
| **Link** | Extra communication channel | `IsConnected()`, `GetStats()` |

**Note**: All components assume NATS connectivity as baseline infrastructure. NATS is not a "Link"—Links exist for additional channels that NATS cannot reach (serial to MCUs, radio telemetry, etc.).

## Sensors

Sensors observe the environment without changing it. They implement:

```go
type Sensor interface {
    Resource
    Readings(ctx context.Context) (map[string]any, error)
}
```

### Implementing a Sensor

```go
type TemperatureSensor struct {
    name   resource.Name
    config Config
    reader reader.Reader
    nc     *nats.Conn

    mu          sync.RWMutex
    lastReading float64
}

func (s *TemperatureSensor) Name() resource.Name {
    return s.name
}

func (s *TemperatureSensor) Readings(ctx context.Context) (map[string]any, error) {
    reading, err := s.reader.Read(ctx, s.config.Zone)
    if err != nil {
        return nil, err
    }

    return map[string]any{
        "temperature_celsius":    reading.TemperatureC,
        "temperature_fahrenheit": celsiusToFahrenheit(reading.TemperatureC),
        "zone":                   reading.Zone,
    }, nil
}
```

### Built-in Sensor Types

| Sensor | Key Methods | Use Case |
|--------|-------------|----------|
| **IMU** | LinearAcceleration, AngularVelocity, Orientation, GetMagneticField | Motion sensing |
| **AHRS** | GetEulerAngles, GetQuaternion, GetCalibrationStatus | Sensor-fused orientation |
| **GPS** | Position, Accuracy, Fix, GetSatellitesUsed | Outdoor navigation |
| **Encoder** | Position, GetVelocity, GetResolution | Motor feedback |
| **RangeSensor** | GetRange, GetMinRange, GetMaxRange | Obstacle detection |
| **LiDAR** | GetScan, GetPointCloud, GetProperties | Mapping, SLAM |
| **PresenceSensor** | IsPresenceDetected, GetMotionState | Human detection |
| **ThermalArray** | GetTemperatureGrid, GetMinMaxTemperature | Heat mapping |
| **ForceSensor** | GetForce, Tare | Force measurement |
| **Force6DOF** | GetWrench | 6-axis force/torque |
| **CurrentSensor** | GetCurrent, GetVoltage, GetPower | Power monitoring |
| **ReflectanceSensor** | GetReflectances, GetLinePosition | Line following |

## Actuators

Actuators change the environment. They implement:

```go
type Actuator interface {
    Resource
    IsMoving(ctx context.Context) (bool, error)
    Stop(ctx context.Context) error
}
```

### Motors

Motors are the most common actuator type:

```go
type Motor interface {
    Actuator
    SetPower(ctx context.Context, power float64) error
    SetVelocity(ctx context.Context, velocity float64) error
    GoTo(ctx context.Context, position, velocity float64) error
    GetPosition(ctx context.Context) (float64, error)
    Properties(ctx context.Context) (Properties, error)
}
```

### Built-in Actuator Types

| Actuator | Key Methods | Use Case |
|----------|-------------|----------|
| **Motor** | SetPower, SetVelocity, GoTo, GoFor | Wheels, conveyors |
| **Servo** | SetAngle, SetSpeed, SetTorqueLimit | Pan/tilt, joints |
| **Stepper** | Step, SetMicrostepping, Home | 3D printers, CNC |
| **Thruster** | SetThrust, GetRPM, GetTemperature | ROVs, drones |
| **Valve** | Open, Shut, SetPosition | Pneumatics, flow control |
| **Gripper** | Open, Close, Grab | Pick and place |
| **Base** | SetVelocity, MoveStraight, Spin | Mobile robots |
| **Arm** | MoveToPosition, JointPositions | Manipulation |

### Servo Interface

```go
type Servo interface {
    Actuator
    SetAngle(ctx context.Context, degrees float64) error
    GetAngle(ctx context.Context) (float64, error)
    SetSpeed(ctx context.Context, speed float64) error
    SetTorqueLimit(ctx context.Context, limit float64) error
    GetProperties(ctx context.Context) (Properties, error)
}
```

### Stepper Interface

```go
type Stepper interface {
    Actuator
    Step(ctx context.Context, steps int64) error
    SetMicrostepping(ctx context.Context, divisor int) error
    SetCurrent(ctx context.Context, runMA, holdMA int) error
    GetPosition(ctx context.Context) (int64, error)
    ResetPosition(ctx context.Context) error
    Home(ctx context.Context, direction bool) error
}
```

### Thruster Interface

```go
type Thruster interface {
    Actuator
    SetThrust(ctx context.Context, thrust float64) error  // -1.0 to 1.0
    GetRPM(ctx context.Context) (int, error)
    GetTemperature(ctx context.Context) (float64, error)
    GetCurrent(ctx context.Context) (float64, error)
}
```

### Valve Interface

```go
type Valve interface {
    Actuator
    Open(ctx context.Context) error
    Shut(ctx context.Context) error  // Named "Shut" to avoid conflict with Close()
    SetPosition(ctx context.Context, position float64) error  // 0.0-1.0
    GetPosition(ctx context.Context) (float64, error)
    IsOpen(ctx context.Context) (bool, error)
    IsClosed(ctx context.Context) (bool, error)
}
```

## Power Components

Power components manage energy storage and distribution:

```go
type Power interface {
    Resource
    GetCapacity(ctx context.Context) (float64, error)
    GetLevel(ctx context.Context) (float64, error)
    GetVoltage(ctx context.Context) (float64, error)
    GetCurrent(ctx context.Context) (float64, error)
    IsCharging(ctx context.Context) (bool, error)
}
```

## Space Components

Space components represent **virtual containers on the robot**. A Space doesn't directly interface with hardware—it aggregates and coordinates other components that do.

Use Spaces for:
- Storage areas with doors or hatches (cargo bay, sample drawer)
- Tanks with valves and level sensors (ballast tank, fuel tank)
- Compartments with environmental controls (battery bay, equipment compartment)

```go
type Space interface {
    Resource
    GetVolume(ctx context.Context) (float64, error)
    GetBounds(ctx context.Context) (*geometry.Box, error)
    GetContents(ctx context.Context) ([]string, error)
    IsEmpty(ctx context.Context) (bool, error)
    GetComponents(ctx context.Context) ([]resource.Name, error)
}
```

### Space Example: Ballast Tank

```go
type BallastTank struct {
    name        resource.Name
    volume      float64
    fillValve   actuator.Valve   // Controls water intake
    drainValve  actuator.Valve   // Controls water release
    levelSensor sensor.Level     // Measures fill percentage
}

func (t *BallastTank) GetContents(ctx context.Context) ([]string, error) {
    level, _ := t.levelSensor.Readings(ctx)
    return []string{fmt.Sprintf("water:%.1f%%", level["percent"])}, nil
}

func (t *BallastTank) GetComponents(ctx context.Context) ([]resource.Name, error) {
    return []resource.Name{
        t.fillValve.Name(),
        t.drainValve.Name(),
        t.levelSensor.Name(),
    }, nil
}
```

## Link Components

Links provide **additional communication channels beyond NATS**. All components assume NATS connectivity—that's the baseline. Links exist for communication paths that NATS cannot reach:

- **Serial links**: Bridge to TinyGo microcontrollers without IP capability
- **Radio links**: Remote telemetry when out of WiFi range
- **CAN bus**: Vehicle systems, industrial protocols

```go
type Link interface {
    Resource
    Type() LinkType
    Direction() LinkDirection
    IsConnected(ctx context.Context) (bool, error)
    GetStats(ctx context.Context) (*LinkStats, error)
}
```

### Link Example: Serial Gateway

```go
type SerialLink struct {
    name     resource.Name
    port     string           // e.g., "/dev/ttyUSB0"
    baudRate int
    conn     serial.Port
    nc       *nats.Conn
}

// Bridges NATS messages to/from a microcontroller
func (l *SerialLink) Run(ctx context.Context) {
    // Forward NATS commands to MCU over serial
    l.nc.Subscribe("gorai.robot.motor.command", func(msg *nats.Msg) {
        l.conn.Write(encodeCommand(msg.Data))
    })

    // Publish MCU sensor data back to NATS
    go func() {
        for {
            data := l.conn.Read()
            l.nc.Publish("gorai.robot.mcu.sensors", data)
        }
    }()
}
```

## Cameras

Cameras capture visual data and are a special type that bridges sensors and vision:

```go
type Camera interface {
    Resource
    Image(ctx context.Context) (image.Image, error)
    Stream(ctx context.Context) (chan image.Image, error)
    Properties(ctx context.Context) (Properties, error)
}
```

### Camera Example

```go
func (c *USBCamera) Image(ctx context.Context) (image.Image, error) {
    frame, err := c.device.Capture()
    if err != nil {
        return nil, err
    }
    return frame, nil
}
```

## Concurrency Model

Gorai uses a **single-owner model** for components:

- Each component instance is owned by exactly one goroutine
- **No mutex protection needed** for component state
- Cross-goroutine coordination uses NATS messaging, not shared memory

This follows Go's philosophy: "Share memory by communicating, don't communicate by sharing memory."

```go
// Single-owner component - no mutex needed
type FakeMotor struct {
    name     resource.Name
    power    float64    // Only accessed by owner goroutine
    moving   bool
    position float64
}

func (m *FakeMotor) SetPower(ctx context.Context, power float64) error {
    m.power = power
    m.moving = power != 0
    return nil
}
```

## Fake Implementations

Every component should have a fake for testing:

```go
type FakeMotor struct {
    name     resource.Name
    power    float64
    moving   bool
    position float64
}

func (m *FakeMotor) SetPower(ctx context.Context, power float64) error {
    m.power = power
    m.moving = power != 0
    return nil
}

// Test helper - for use by test code (single goroutine)
func (m *FakeMotor) GetPowerSet() float64 {
    return m.power
}

func (m *FakeMotor) SetPosition(pos float64) {
    m.position = pos
}
```

Fakes should:
- Implement the same interface as real components
- Provide test helper methods (setters/getters for inspection)
- Follow the single-owner model (no mutex)
- Support error injection for testing failure cases

## Next Steps

- [Working with Services](../services/)
- [NATS Messaging Guide](../nats/)
- [Testing Guide](../testing/)
