---
title: "Working with Components"
description: "Build and use sensors, actuators, and cameras"
weight: 10
---

# Working with Components

Components are the building blocks that interface with robot hardware. They abstract physical devices behind consistent interfaces.

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

## Sensors

Sensors measure physical quantities and return readings. They implement:

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

Actuators receive commands and produce physical action. They implement:

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

## Cameras

Cameras capture visual data:

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
