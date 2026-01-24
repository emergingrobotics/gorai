# Chapter 10: Building Custom Components

Now that you understand GoRAI's patterns through hello-sensor, let's build custom components from scratch.

## 10.1 When to Create a Component

Create a component when you have:

- **Hardware to abstract**: Specific sensor, motor, or device
- **Reusable functionality**: Will be used across projects
- **Standard interface compliance**: Fits existing component types
- **Need for fakes**: Testing requires simulation

Don't create a component for:
- One-off scripts
- Pure software logic (use a service instead)
- Simple utilities (use functions)

## 10.2 Component Structure

Standard layout:

```
component/
└── mycomponent/
    ├── mycomponent.go      # Interface definition
    ├── mycomponent_test.go # Unit tests
    └── fake/
        ├── fake.go         # Test double
        └── fake_test.go    # Fake tests
```

Or for a standalone driver:

```
github.com/myorg/gorai-mymotor/
├── mymotor.go
├── mymotor_test.go
├── fake/
│   └── fake.go
├── go.mod
└── README.md
```

## 10.3 Step-by-Step: Custom Motor Driver

Let's build a motor driver for a DRV8833 dual motor controller.

### 10.3.1 Define the Interface

First, understand what capabilities we need:

```go
// drv8833/drv8833.go
package drv8833

import (
    "context"
    "github.com/gorai/gorai/components/motor"
)

// DRV8833Motor implements motor.Motor for DRV8833 controller.
type DRV8833Motor interface {
    motor.Motor

    // SetDecay sets the decay mode (fast or slow).
    SetDecay(ctx context.Context, fast bool) error

    // GetFault returns true if fault pin is active.
    GetFault(ctx context.Context) (bool, error)
}
```

### 10.3.2 Implement the Driver

```go
// drv8833/motor.go
package drv8833

import (
    "context"
    "fmt"
    "sync"

    "github.com/gorai/gorai/driver/gpio"
    "github.com/gorai/gorai/pkg/resource"
)

type Config struct {
    In1Pin     int     `json:"in1_pin"`
    In2Pin     int     `json:"in2_pin"`
    PWMPin     int     `json:"pwm_pin"`
    FaultPin   int     `json:"fault_pin"`
    MaxPower   float64 `json:"max_power"`
    PWMFreqHz  int     `json:"pwm_freq_hz"`
}

type Motor struct {
    name      resource.Name
    config    Config
    in1       gpio.Pin
    in2       gpio.Pin
    pwm       gpio.PWMPin
    fault     gpio.Pin

    mu        sync.RWMutex
    power     float64
    moving    bool
    fastDecay bool
}

func New(name string, cfg Config, pins gpio.Provider) (*Motor, error) {
    if cfg.MaxPower <= 0 || cfg.MaxPower > 1.0 {
        cfg.MaxPower = 1.0
    }
    if cfg.PWMFreqHz <= 0 {
        cfg.PWMFreqHz = 20000  // 20kHz default
    }

    m := &Motor{
        name:   resource.NewComponentName("gorai", "motor", name),
        config: cfg,
    }

    var err error
    m.in1, err = pins.OutputPin(cfg.In1Pin)
    if err != nil {
        return nil, fmt.Errorf("failed to configure IN1: %w", err)
    }

    m.in2, err = pins.OutputPin(cfg.In2Pin)
    if err != nil {
        return nil, fmt.Errorf("failed to configure IN2: %w", err)
    }

    m.pwm, err = pins.PWMPin(cfg.PWMPin, cfg.PWMFreqHz)
    if err != nil {
        return nil, fmt.Errorf("failed to configure PWM: %w", err)
    }

    if cfg.FaultPin > 0 {
        m.fault, err = pins.InputPin(cfg.FaultPin)
        if err != nil {
            return nil, fmt.Errorf("failed to configure FAULT: %w", err)
        }
    }

    return m, nil
}

// Resource interface

func (m *Motor) Name() resource.Name {
    return m.name
}

func (m *Motor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
    var cfg Config
    if err := conf.Unmarshal(&cfg); err != nil {
        return err
    }
    m.mu.Lock()
    m.config.MaxPower = cfg.MaxPower
    m.mu.Unlock()
    return nil
}

func (m *Motor) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
    return nil, nil
}

func (m *Motor) Close(ctx context.Context) error {
    m.Stop(ctx)
    return nil
}

// Actuator interface

func (m *Motor) IsMoving(ctx context.Context) (bool, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.moving, nil
}

func (m *Motor) Stop(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    // Coast stop: both inputs low
    m.in1.Low()
    m.in2.Low()
    m.pwm.SetDutyCycle(0)
    m.power = 0
    m.moving = false

    return nil
}

// Motor interface

func (m *Motor) SetPower(ctx context.Context, power float64) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    // Check for faults
    if m.fault != nil && m.fault.Read() {
        return fmt.Errorf("motor fault detected")
    }

    // Clamp power
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
    } else {
        // Brake (both high with zero PWM) or coast
        if m.fastDecay {
            m.in1.Low()
            m.in2.Low()
        } else {
            m.in1.High()
            m.in2.High()
        }
    }

    // Set PWM duty cycle
    duty := uint32(power * 65535)
    if err := m.pwm.SetDutyCycle(duty); err != nil {
        return fmt.Errorf("failed to set PWM: %w", err)
    }

    m.power = power
    m.moving = power != 0

    return nil
}

// Simplified implementations for remaining methods...

func (m *Motor) SetVelocity(ctx context.Context, velocity float64) error {
    return fmt.Errorf("velocity control not supported without encoder")
}

func (m *Motor) GoTo(ctx context.Context, position, velocity float64) error {
    return fmt.Errorf("position control not supported without encoder")
}

func (m *Motor) GoFor(ctx context.Context, rpm, revolutions float64) error {
    return fmt.Errorf("GoFor not supported without encoder")
}

func (m *Motor) GetPosition(ctx context.Context) (float64, error) {
    return 0, fmt.Errorf("position reporting not supported")
}

func (m *Motor) GetVelocity(ctx context.Context) (float64, error) {
    return 0, fmt.Errorf("velocity reporting not supported")
}

func (m *Motor) ResetZeroPosition(ctx context.Context, offset float64) error {
    return fmt.Errorf("not supported")
}

func (m *Motor) IsPowered(ctx context.Context) (bool, float64, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.power != 0, m.power, nil
}

func (m *Motor) Properties(ctx context.Context) (motor.Properties, error) {
    return motor.Properties{
        PositionReporting: false,
        VelocityReporting: false,
        SupportsGoTo:      false,
    }, nil
}
```

### 10.3.3 Create the Fake

```go
// drv8833/fake/fake.go
package fake

import (
    "context"
    "sync"

    "github.com/gorai/gorai/components/motor"
    "github.com/gorai/gorai/pkg/resource"
)

type Motor struct {
    name   resource.Name
    mu     sync.RWMutex
    power  float64
    moving bool
    fault  bool
}

func New(name string) *Motor {
    return &Motor{
        name: resource.NewComponentName("test", "motor", name),
    }
}

// Test helpers

func (m *Motor) SetFault(fault bool) {
    m.mu.Lock()
    m.fault = fault
    m.mu.Unlock()
}

func (m *Motor) GetPowerSet() float64 {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.power
}

// Motor interface implementation...
func (m *Motor) SetPower(ctx context.Context, power float64) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.fault {
        return fmt.Errorf("motor fault")
    }

    m.power = power
    m.moving = power != 0
    return nil
}

// ... other methods similar to real but tracking state
```

### 10.3.4 Write Tests

```go
// drv8833/drv8833_test.go
package drv8833_test

import (
    "context"
    "testing"

    "github.com/myorg/gorai-drv8833/fake"
    "github.com/stretchr/testify/assert"
)

func TestMotor_SetPower(t *testing.T) {
    motor := fake.New("test")

    err := motor.SetPower(context.Background(), 0.5)
    assert.NoError(t, err)
    assert.Equal(t, 0.5, motor.GetPowerSet())

    moving, _ := motor.IsMoving(context.Background())
    assert.True(t, moving)
}

func TestMotor_SetPower_Fault(t *testing.T) {
    motor := fake.New("test")
    motor.SetFault(true)

    err := motor.SetPower(context.Background(), 0.5)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "fault")
}

func TestMotor_Stop(t *testing.T) {
    motor := fake.New("test")
    motor.SetPower(context.Background(), 0.5)

    err := motor.Stop(context.Background())
    assert.NoError(t, err)

    moving, _ := motor.IsMoving(context.Background())
    assert.False(t, moving)
}
```

## 10.4 Registration and Discovery

### Adding to Registry

```go
func init() {
    registry.RegisterComponent("motor", "drv8833", func(ctx context.Context, deps resource.Dependencies, conf resource.Config) (any, error) {
        var cfg Config
        if err := conf.Unmarshal(&cfg); err != nil {
            return nil, err
        }
        pins := gpio.DefaultProvider()
        return New(conf.Name, cfg, pins)
    })
}
```

### Configuration Schema

Document your configuration:

```json
{
    "name": "left_motor",
    "type": "motor",
    "model": "drv8833",
    "attributes": {
        "in1_pin": 17,
        "in2_pin": 18,
        "pwm_pin": 12,
        "fault_pin": 25,
        "max_power": 0.8,
        "pwm_freq_hz": 20000
    }
}
```


## 10.5 Network Transparency

Expose your component for remote access:

```go
// On node with hardware
motor, _ := drv8833.New("left", cfg, pins)
nws.Wrap(node, motor, "gorai.motors.left")

// On remote node
motor := nwc.Motor(remoteNode, "gorai.motors.left")
motor.SetPower(ctx, 0.5)  // Works transparently
```

*Cross-reference: See Chapter 2 for NWS/NWC concepts.*

---

Chapter 11 covers testing these components thoroughly.
