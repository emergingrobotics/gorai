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

| Sensor | Readings | Use Case |
|--------|----------|----------|
| Temperature | Celsius, Fahrenheit | Thermal monitoring |
| IMU | Orientation, acceleration, angular velocity | Motion sensing |
| GPS | Latitude, longitude, altitude | Outdoor navigation |
| Encoder | Position, velocity | Motor feedback |
| Range | Distance | Obstacle detection |

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

### Implementing a Motor

```go
type DRV8833Motor struct {
    name   resource.Name
    config Config
    in1    gpio.Pin
    in2    gpio.Pin
    pwm    gpio.PWMPin

    mu     sync.RWMutex
    power  float64
    moving bool
}

func (m *DRV8833Motor) SetPower(ctx context.Context, power float64) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    // Clamp power to configured max
    if power > m.config.MaxPower {
        power = m.config.MaxPower
    }
    if power < -m.config.MaxPower {
        power = -m.config.MaxPower
    }

    // Set direction
    if power > 0 {
        m.in1.High()
        m.in2.Low()
    } else if power < 0 {
        m.in1.Low()
        m.in2.High()
        power = -power
    }

    // Set PWM duty cycle
    m.pwm.SetDutyCycle(uint32(power * 65535))
    m.power = power
    m.moving = power != 0

    return nil
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

## Fake Implementations

Every component should have a fake for testing:

```go
type FakeMotor struct {
    mu     sync.RWMutex
    power  float64
    moving bool
}

func (m *FakeMotor) SetPower(ctx context.Context, power float64) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.power = power
    m.moving = power != 0
    return nil
}

// Test helper
func (m *FakeMotor) GetPowerSet() float64 {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.power
}
```

## Next Steps

- [Working with Services](../services/)
- [NATS Messaging Guide](../nats/)
- [Testing Guide](../testing/)
