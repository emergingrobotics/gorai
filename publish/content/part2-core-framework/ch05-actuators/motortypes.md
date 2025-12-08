## 5.3 Motor Types

Different motor technologies have different characteristics. Understanding them helps you choose the right motor and write appropriate control code.

### DC Motors

Simple, cheap, high-power motors controlled by voltage/PWM:

**Characteristics**:
- Continuous rotation
- Speed proportional to voltage (roughly)
- Torque proportional to current
- Reversible by polarity swap
- Require H-bridge driver for bidirectional control

**GoRAI implementation considerations**:
```go
type DCMotor struct {
    pwmPin    gpio.PWMPin
    dir1Pin   gpio.Pin
    dir2Pin   gpio.Pin
    encoder   *Encoder  // Optional
}

func (m *DCMotor) SetPower(ctx context.Context, power float64) error {
    // Set direction
    if power >= 0 {
        m.dir1Pin.High()
        m.dir2Pin.Low()
    } else {
        m.dir1Pin.Low()
        m.dir2Pin.High()
        power = -power
    }

    // Set PWM duty cycle
    return m.pwmPin.SetDutyCycle(uint32(power * 65535))
}
```

**Typical applications**: Wheel drive, simple conveyors, fans

### Stepper Motors

Precise positioning without encoders:

**Characteristics**:
- Move in discrete steps (typically 200 steps/revolution)
- Microstepping for finer resolution (1600, 3200+ steps/rev)
- Position known by counting steps (open-loop)
- Lower top speed than DC motors
- Holding torque at rest

**GoRAI implementation considerations**:
```go
type StepperMotor struct {
    stepPin   gpio.Pin
    dirPin    gpio.Pin
    stepsPerRev int
    position  int64
}

func (m *StepperMotor) GoFor(ctx context.Context, rpm, revolutions float64) error {
    steps := int(revolutions * float64(m.stepsPerRev))
    stepDelay := time.Duration(60e9 / (rpm * float64(m.stepsPerRev)))

    // Set direction
    if steps < 0 {
        m.dirPin.Low()
        steps = -steps
    } else {
        m.dirPin.High()
    }

    for i := 0; i < steps; i++ {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            m.stepPin.High()
            time.Sleep(stepDelay / 2)
            m.stepPin.Low()
            time.Sleep(stepDelay / 2)
            m.position++
        }
    }
    return nil
}
```

**Typical applications**: 3D printers, CNC machines, camera gimbals

### Servo Motors

Position-controlled motors with built-in feedback:

**Characteristics**:
- Integrated controller, encoder, and motor
- Position or velocity commands via protocol (PWM, serial, CAN)
- High precision and repeatability
- More expensive than DC/stepper

**Types**:
- **RC Servos**: PWM control, limited rotation (180° typical)
- **Smart Servos**: Serial protocol (Dynamixel, Herkulex), full rotation
- **Industrial Servos**: CAN/EtherCAT, high power

**GoRAI implementation for RC servo**:
```go
type RCServo struct {
    pwmPin gpio.PWMPin
    minPulse time.Duration  // 1ms typical
    maxPulse time.Duration  // 2ms typical
}

func (s *RCServo) SetAngle(ctx context.Context, degrees float64) error {
    // Map degrees to pulse width
    pulse := s.minPulse + time.Duration(
        (degrees/180.0)*float64(s.maxPulse-s.minPulse))

    return s.pwmPin.SetPulseWidth(pulse)
}
```

### Brushless Motors (BLDC)

High-performance motors requiring electronic commutation:

**Characteristics**:
- Higher efficiency than brushed DC
- Higher speed capability
- Longer lifespan (no brushes to wear)
- Require ESC (Electronic Speed Controller) or FOC driver

**Control modes**:
- **PWM/Throttle**: Like DC motors, proportional control
- **Closed-loop**: Encoder feedback for precise velocity/position

**Typical applications**: Drones, high-performance wheels, industrial robots

### Choosing Motor Type

| Requirement | Best Choice |
|-------------|-------------|
| Cheap, simple drive | DC motor |
| Precise positioning, low speed | Stepper |
| Servo-like behavior, budget | DC + encoder |
| High precision positioning | Smart servo |
| High speed, efficiency | Brushless |
| Simple angle control | RC servo |
