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

Components are organized by what they do in the physical world:

| Category | What It Does | Interface |
|----------|--------------|-----------|
| **Sensor** | Observes the world (read-only) | `Readings()` |
| **Actuator** | Changes the world | `IsMoving()`, `Stop()` |
| **Power** | Manages energy | `GetCapacity()`, `GetLevel()` |
| **Space** | Defines physical volumes | `GetVolume()`, `GetBounds()` |
| **Link** | Enables communication | `IsConnected()`, `GetStats()` |

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

Space components represent physical volumes:

```go
type Space interface {
    Resource
    GetVolume(ctx context.Context) (float64, error)
    GetBounds(ctx context.Context) (*geometry.Box, error)
    GetContents(ctx context.Context) ([]string, error)
    IsEmpty(ctx context.Context) (bool, error)
}
```

## Link Components

Links provide communication between nodes:

```go
type Link interface {
    Resource
    Type() LinkType
    Direction() LinkDirection
    IsConnected(ctx context.Context) (bool, error)
    GetStats(ctx context.Context) (*LinkStats, error)
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
