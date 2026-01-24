# Actuators

If sensors are how robots perceive, actuators are how they act. Actuators transform electrical signals into physical motion—wheels that turn, arms that reach, grippers that grasp.

Gorai provides a hierarchy of actuator interfaces from the basic `Actuator` (with `Stop()`) to specialized interfaces like `Motor`, `Servo`, and `Arm`. This chapter covers all actuator types, their interfaces, and the control patterns that make them work safely and effectively. Safety is paramount—every actuator can stop instantly, and control loops handle failures gracefully.

## The Actuator Interface

All actuators share a common base interface from `pkg/resource/resource.go`:

```go
type Actuator interface {
    Resource

    // IsMoving returns true if the actuator is currently in motion.
    IsMoving(ctx context.Context) (bool, error)

    // Stop halts all motion immediately.
    Stop(ctx context.Context) error
}
```

The Actuator interface adds two critical safety methods:

**IsMoving()**: Query motion state. Essential for:

- Waiting for motion to complete before next action
- Safety interlocks (don't close gripper while arm is moving)
- State machine transitions

**Stop()**: Emergency halt. Every actuator must implement immediate stop:

- Called during emergency shutdowns
- Triggered by safety systems
- Available for manual intervention

### Safety-First Design

Actuators can cause harm. Gorai's actuator design prioritizes safety:

```go
func (m *Motor) SetPower(ctx context.Context, power float64) error {
    // Clamp power to safe limits
    if power > m.maxPower {
        power = m.maxPower
    }
    if power < -m.maxPower {
        power = -m.maxPower
    }

    // Check safety conditions
    if m.overTemp {
        return fmt.Errorf("motor overtemperature, refusing to run")
    }

    return m.driver.SetPower(power)
}
```

### Emergency Stop Patterns

Implement robust stop behavior:

```go
func (m *Motor) Stop(ctx context.Context) error {
    // Stop is best-effort—try multiple approaches
    var errs []error

    // Try graceful stop first
    if err := m.driver.SetPower(0); err != nil {
        errs = append(errs, err)
    }

    // Engage brake if available
    if m.hasBrake {
        if err := m.driver.EngageBrake(); err != nil {
            errs = append(errs, err)
        }
    }

    // Cut power as last resort
    if len(errs) > 0 {
        m.driver.CutPower()
    }

    m.mu.Lock()
    m.moving = false
    m.mu.Unlock()

    if len(errs) > 0 {
        return fmt.Errorf("stop encountered errors: %v", errs)
    }
    return nil
}
```

## Motor Interface

Motors are the most common actuators. The Motor interface from `components/motor/motor.go` provides comprehensive control:

```go
type Motor interface {
    component.Actuator

    // SetPower sets the motor power from -1.0 (full reverse) to 1.0 (full forward).
    SetPower(ctx context.Context, power float64) error

    // SetVelocity sets target velocity (rad/s or m/s depending on motor type).
    SetVelocity(ctx context.Context, velocity float64) error

    // GoTo moves the motor to the specified absolute position at the given velocity.
    GoTo(ctx context.Context, position, velocity float64) error

    // GoFor moves the motor for the specified number of revolutions at the given RPM.
    // Positive RPM moves forward, negative moves backward.
    GoFor(ctx context.Context, rpm, revolutions float64) error

    // GetPosition returns the current position (in revolutions from zero).
    GetPosition(ctx context.Context) (float64, error)

    // GetVelocity returns the current velocity.
    GetVelocity(ctx context.Context) (float64, error)

    // ResetZeroPosition sets the current position as the zero position.
    ResetZeroPosition(ctx context.Context, offset float64) error

    // IsPowered returns whether the motor is currently receiving power
    // and the current power level.
    IsPowered(ctx context.Context) (bool, float64, error)

    // Properties returns the motor's properties.
    Properties(ctx context.Context) (Properties, error)
}
```

### Power Control: SetPower

The simplest control mode—direct power/duty cycle:

```go
// Full forward
motor.SetPower(ctx, 1.0)

// Half speed reverse
motor.SetPower(ctx, -0.5)

// Stop (coast)
motor.SetPower(ctx, 0)
```

**Power is normalized** to [-1.0, 1.0]:

- +1.0 = maximum forward voltage/PWM
- -1.0 = maximum reverse voltage/PWM
- 0 = no power (motor coasts)

**Use cases**: Direct joystick control, simple behaviors where speed precision doesn't matter.

### Velocity Control: SetVelocity

Closed-loop speed control using encoder feedback:

```go
// Rotate at 10 rad/s
motor.SetVelocity(ctx, 10.0)

// For linear actuators, this might be m/s
motor.SetVelocity(ctx, 0.5)  // 0.5 m/s
```

Velocity control requires:

- Encoder feedback
- PID controller (typically in driver or motor controller)
- Proper tuning

### Position Control: GoTo and GoFor

Move to absolute or relative positions:

```go
// Move to position 5.0 revolutions at velocity 2.0 rad/s
motor.GoTo(ctx, 5.0, 2.0)

// Move forward 2 revolutions at 60 RPM
motor.GoFor(ctx, 60, 2.0)

// Move backward 1 revolution at 30 RPM
motor.GoFor(ctx, -30, 1.0)
```

**GoTo**: Absolute position (requires knowing current position)
**GoFor**: Relative motion (useful for "move X distance" commands)

Both methods are typically blocking or provide completion feedback.

### State Queries

```go
// Current position in revolutions
pos, _ := motor.GetPosition(ctx)

// Current velocity
vel, _ := motor.GetVelocity(ctx)

// Is motor powered and at what level?
powered, level, _ := motor.IsPowered(ctx)
```

### Motor Properties

Motors report their capabilities:

```go
type Properties struct {
    // PositionReporting indicates whether the motor can report its position.
    PositionReporting bool

    // VelocityReporting indicates whether the motor can report its velocity.
    VelocityReporting bool

    // SupportsGoTo indicates whether the motor supports absolute positioning.
    SupportsGoTo bool
}
```

Check properties before using advanced features:

```go
props, _ := motor.Properties(ctx)

if !props.PositionReporting {
    // Can't use GoTo without position feedback
    // Fall back to timed motion or power control
}
```

## Motor Types

Different motor technologies have different characteristics. Understanding them helps you choose the right motor and write appropriate control code.

### DC Motors

Simple, cheap, high-power motors controlled by voltage/PWM:

**Characteristics**:

- Continuous rotation
- Speed proportional to voltage (roughly)
- Torque proportional to current
- Reversible by polarity swap
- Require H-bridge driver for bidirectional control

**Gorai implementation considerations**:

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

**Gorai implementation considerations**:

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

**Gorai implementation for RC servo**:

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

## Control Patterns

Motor control ranges from simple power commands to sophisticated closed-loop control. Understanding these patterns helps you implement appropriate control for your application.

### Open-Loop vs Closed-Loop

**Open-loop control**: Command without feedback

```go
// "Run at 50% power"
motor.SetPower(ctx, 0.5)
// No verification that actual speed matches intent
```

Pros: Simple, no sensors required
Cons: No compensation for load changes, friction, battery voltage

**Closed-loop control**: Command with feedback

```go
// "Run at 100 RPM" - controller adjusts power to maintain speed
motor.SetVelocity(ctx, 100)
// Encoder feedback allows continuous adjustment
```

Pros: Precise, handles disturbances
Cons: Requires sensors, tuning, more complex

### PID Control Basics

PID (Proportional-Integral-Derivative) is the workhorse of motor control:

```go
type PIDController struct {
    Kp, Ki, Kd float64     // Gains
    integral   float64     // Accumulated error
    lastError  float64     // Previous error
    lastTime   time.Time
}

func (p *PIDController) Compute(setpoint, measurement float64) float64 {
    now := time.Now()
    dt := now.Sub(p.lastTime).Seconds()
    p.lastTime = now

    error := setpoint - measurement

    // Proportional term
    proportional := p.Kp * error

    // Integral term (accumulated error)
    p.integral += error * dt
    integral := p.Ki * p.integral

    // Derivative term (rate of change)
    derivative := p.Kd * (error - p.lastError) / dt
    p.lastError = error

    return proportional + integral + derivative
}
```

**Tuning PID**:

1. Start with Ki = Kd = 0
2. Increase Kp until oscillation, then halve it
3. Add Ki to eliminate steady-state error
4. Add Kd to reduce overshoot

### Velocity Profiles

For smooth motion, use velocity profiles instead of instantaneous commands:

**Trapezoidal profile**: Accelerate, cruise, decelerate

```
Velocity
    ^
    │     ┌─────────────┐
    │    /               \
    │   /                 \
    │  /                   \
    └──────────────────────────> Time
      Accel   Cruise   Decel
```

```go
type TrapezoidalProfile struct {
    maxVel   float64
    maxAccel float64
}

func (t *TrapezoidalProfile) Generate(distance float64) []VelocityPoint {
    // Calculate times for each phase
    accelTime := t.maxVel / t.maxAccel
    accelDist := 0.5 * t.maxAccel * accelTime * accelTime

    if 2*accelDist > distance {
        // Triangle profile - never reach max velocity
        accelTime = math.Sqrt(distance / t.maxAccel)
        return t.triangleProfile(distance, accelTime)
    }

    cruiseDist := distance - 2*accelDist
    cruiseTime := cruiseDist / t.maxVel

    return t.trapezoidProfile(accelTime, cruiseTime)
}
```

**S-curve profile**: Smoother acceleration (jerk-limited)

- Better for delicate operations
- Reduces mechanical stress

### Position Feedback

For position control, combine velocity control with position feedback:

```go
type PositionController struct {
    velocityPID *PIDController
    positionGain float64
}

func (c *PositionController) GoTo(target float64) {
    for {
        current := c.motor.GetPosition()
        error := target - current

        if math.Abs(error) < c.tolerance {
            break  // Close enough
        }

        // Position error -> velocity setpoint
        velocitySetpoint := c.positionGain * error

        // Clamp to max velocity
        velocitySetpoint = clamp(velocitySetpoint, -c.maxVel, c.maxVel)

        // Velocity PID computes power
        currentVel := c.motor.GetVelocity()
        power := c.velocityPID.Compute(velocitySetpoint, currentVel)

        c.motor.SetPower(power)
        time.Sleep(10 * time.Millisecond)  // Control loop rate
    }
}
```

### When Control Happens

Control loops can run at different levels:

**Motor controller hardware**: Many motor drivers have built-in PID

- Fastest response (microseconds)
- Configure via registers/protocol

**TinyGo on microcontroller**: Real-time control in Go

- Fast response (100µs - 1ms)
- Custom control algorithms

**Gorai on Linux**: Higher-level control

- Slower response (1-10ms)
- Suitable for position control, trajectory tracking
- Not suitable for commutation, current control

Choose the right level for your control loop requirements.

## Servo Interface

Servos are position-controlled actuators, typically for angular positioning. They differ from motors in their control paradigm: you command angles, not speeds.

```go
type Servo interface {
    component.Actuator

    // SetAngle sets the servo to the specified angle (degrees).
    SetAngle(ctx context.Context, degrees float64) error

    // GetAngle returns the current angle (degrees).
    GetAngle(ctx context.Context) (float64, error)

    // SetSpeed sets the movement speed (0.0-1.0, where 1.0 is max speed).
    SetSpeed(ctx context.Context, speed float64) error

    // SetTorqueLimit sets the maximum torque (0.0-1.0).
    SetTorqueLimit(ctx context.Context, limit float64) error

    // GetProperties returns the servo's properties.
    GetProperties(ctx context.Context) (Properties, error)
}

type Properties struct {
    MinAngle     float64  // Typically -90 or 0
    MaxAngle     float64  // Typically 90 or 180
    IsContinuous bool     // Continuous rotation servo
    HasFeedback  bool     // Position feedback available
    Protocol     string   // "pwm", "dynamixel", "lx16a", "feetech"
}
```

### Angle-Based Positioning

```go
// Center position
servo.SetAngle(ctx, 0)  // Or 90 depending on range

// Move to 45 degrees
servo.SetAngle(ctx, 45.0)

// Move to negative angle
servo.SetAngle(ctx, -45.0)
```

Angle ranges vary by servo:

- Standard RC servos: 0-180° (or -90 to +90)
- Extended range: 0-270° or 0-360°
- Continuous rotation: Angle maps to speed/direction

### Servo Types

**RC Servos (PWM)**: Simple, cheap, no feedback

```go
props := servo.Properties{
    MinAngle:  -90,
    MaxAngle:  90,
    Protocol:  "pwm",
    HasFeedback: false,
}
```

**Smart Servos (Dynamixel, LX-16A, Feetech)**: Serial protocol, position feedback, configurable parameters

```go
props := servo.Properties{
    MinAngle:  -150,
    MaxAngle:  150,
    Protocol:  "dynamixel",
    HasFeedback: true,
}
```

### PWM Control Implementation

RC servos use PWM pulse width for position:

```go
type RCServo struct {
    pwm       gpio.PWMPin
    minAngle  float64        // Typically -90
    maxAngle  float64        // Typically 90
    minPulse  time.Duration  // Typically 500µs - 1ms
    maxPulse  time.Duration  // Typically 2ms - 2.5ms
}

func (s *RCServo) SetAngle(ctx context.Context, degrees float64) error {
    // Clamp to valid range
    if degrees < s.minAngle {
        degrees = s.minAngle
    }
    if degrees > s.maxAngle {
        degrees = s.maxAngle
    }

    // Map angle to pulse width
    fraction := (degrees - s.minAngle) / (s.maxAngle - s.minAngle)
    pulse := s.minPulse + time.Duration(fraction*float64(s.maxPulse-s.minPulse))

    return s.pwm.SetPulseWidth(pulse)
}
```

**Calibration**: Real servos rarely match spec exactly:

```go
// Measure actual endpoints and center
servo.minPulse = 600 * time.Microsecond   // Actual minimum
servo.maxPulse = 2400 * time.Microsecond  // Actual maximum
```

### Multi-Servo Coordination

Robots often have multiple servos that must move together:

```go
type ServoGroup struct {
    servos map[string]Servo
}

func (g *ServoGroup) MoveAll(ctx context.Context, positions map[string]float64) error {
    // Start all moves simultaneously
    var wg sync.WaitGroup
    errs := make(chan error, len(positions))

    for name, angle := range positions {
        servo := g.servos[name]
        wg.Add(1)
        go func(s Servo, a float64) {
            defer wg.Done()
            if err := s.SetAngle(ctx, a); err != nil {
                errs <- err
            }
        }(servo, angle)
    }

    wg.Wait()
    close(errs)

    // Collect errors
    var allErrs []error
    for err := range errs {
        allErrs = append(allErrs, err)
    }

    if len(allErrs) > 0 {
        return fmt.Errorf("servo errors: %v", allErrs)
    }
    return nil
}
```

**Synchronized motion**: For smooth coordinated movement:

```go
func (g *ServoGroup) Interpolate(from, to map[string]float64, duration time.Duration) {
    steps := int(duration / (20 * time.Millisecond))  // 50Hz update

    for i := 0; i <= steps; i++ {
        t := float64(i) / float64(steps)
        positions := make(map[string]float64)

        for name := range to {
            // Linear interpolation
            positions[name] = from[name] + t*(to[name]-from[name])
        }

        g.MoveAll(ctx, positions)
        time.Sleep(20 * time.Millisecond)
    }
}
```

## Stepper Interface

Stepper motors provide precise positioning through discrete steps, making them ideal for applications requiring accuracy without closed-loop feedback:

```go
type Stepper interface {
    component.Actuator

    // Step moves the motor by the specified number of steps.
    // Positive steps move forward, negative steps move backward.
    Step(ctx context.Context, steps int64) error

    // SetMicrostepping sets the microstepping divisor (1, 2, 4, 8, 16, 32, etc.).
    SetMicrostepping(ctx context.Context, divisor int) error

    // SetCurrent sets the run and hold current in milliamps.
    SetCurrent(ctx context.Context, runMA, holdMA int) error

    // GetPosition returns the current position in steps from zero.
    GetPosition(ctx context.Context) (int64, error)

    // ResetPosition sets the current position as zero.
    ResetPosition(ctx context.Context) error

    // Home moves to the home position using limit switch or stall detection.
    Home(ctx context.Context, direction bool) error

    // GetProperties returns the stepper's properties.
    GetProperties(ctx context.Context) (Properties, error)
}

type Properties struct {
    StepsPerRevolution int     // Typically 200 for 1.8° motors
    MaxMicrostepping   int     // Maximum supported (e.g., 256)
    MaxCurrent         int     // Maximum current in mA
    HasStallDetection  bool    // TMC drivers support this
    Driver             string  // "a4988", "drv8825", "tmc2209", "tmc5160"
}
```

### Basic Stepper Control

```go
// Move 200 steps forward (1 revolution for standard stepper)
stepper.Step(ctx, 200)

// Move 400 steps backward
stepper.Step(ctx, -400)

// Get current position
pos, _ := stepper.GetPosition(ctx)

// Reset position to zero
stepper.ResetPosition(ctx)
```

### Microstepping

Microstepping divides each full step into smaller increments:

```go
// Set 16x microstepping (3200 microsteps per revolution)
stepper.SetMicrostepping(ctx, 16)

// Now each "step" is 1/16 of a full step
stepper.Step(ctx, 3200)  // One full revolution
```

**Microstepping trade-offs**:

| Divisor | Steps/Rev | Resolution | Torque | Speed |
|---------|-----------|------------|--------|-------|
| 1 (full) | 200 | Low | High | Fast |
| 16 | 3200 | Medium | Medium | Medium |
| 256 | 51200 | Very High | Lower | Slow |

### Current Control

Proper current settings affect torque, heating, and noise:

```go
// Set 1500mA run current, 500mA hold current
stepper.SetCurrent(ctx, 1500, 500)
```

- **Run current**: Applied during motion, determines torque
- **Hold current**: Applied at rest, prevents drift but causes heating

### Homing

Home the stepper to establish a known position:

```go
// Home in negative direction until limit switch or stall
stepper.Home(ctx, false)  // false = negative direction

// Position is now 0
pos, _ := stepper.GetPosition(ctx)  // Returns 0
```

### Driver Considerations

Different drivers have different capabilities:

| Driver | Microstepping | Stall Detection | Notes |
|--------|---------------|-----------------|-------|
| A4988 | 16x | No | Basic, cheap |
| DRV8825 | 32x | No | Higher current |
| TMC2209 | 256x | Yes | Quiet, UART config |
| TMC5160 | 256x | Yes | High current, SPI |

## Thruster Interface

Thrusters are specialized actuators for underwater and aerial vehicles, providing bidirectional thrust:

```go
type Thruster interface {
    component.Actuator

    // SetThrust sets the thrust level from -1.0 (full reverse) to 1.0 (full forward).
    SetThrust(ctx context.Context, thrust float64) error

    // GetRPM returns the current motor RPM (if telemetry available).
    GetRPM(ctx context.Context) (int, error)

    // GetTemperature returns the motor/ESC temperature in °C.
    GetTemperature(ctx context.Context) (float64, error)

    // GetCurrent returns the motor current in amps.
    GetCurrent(ctx context.Context) (float64, error)

    // GetProperties returns the thruster's properties.
    GetProperties(ctx context.Context) (Properties, error)
}

type Properties struct {
    MaxThrustForward  float64  // Newtons or kgf
    MaxThrustReverse  float64  // May differ from forward
    DeadbandWidth     float64  // Thrust values below this are zero
    IsBidirectional   bool     // Can reverse?
    HasTelemetry      bool     // RPM/temp/current available?
    Protocol          string   // "pwm", "i2c", "can"
}
```

### Basic Thruster Control

```go
// Forward at 50% thrust
thruster.SetThrust(ctx, 0.5)

// Reverse at 30% thrust
thruster.SetThrust(ctx, -0.3)

// Stop (respects deadband)
thruster.SetThrust(ctx, 0)
```

### Deadband Handling

Thrusters have a deadband where small inputs produce no thrust:

```go
// Small thrust within deadband - motor stops
thruster.SetThrust(ctx, 0.03)  // Below 0.05 deadband = no motion

// Just above deadband - motor runs
thruster.SetThrust(ctx, 0.06)  // Above deadband = starts moving
```

### Telemetry

Some thrusters (like BlueRobotics with ESC telemetry) provide real-time data:

```go
rpm, _ := thruster.GetRPM(ctx)
temp, _ := thruster.GetTemperature(ctx)
current, _ := thruster.GetCurrent(ctx)

// Monitor for overheating
if temp > 70.0 {
    thruster.Stop(ctx)
    log.Warn("Thruster overtemperature!")
}
```

### Common Thruster Hardware

| Thruster | Control | Telemetry | Application |
|----------|---------|-----------|-------------|
| BlueRobotics T100/T200 | PWM/I2C | Yes (I2C) | ROVs, AUVs |
| Generic brushless | PWM | No | Drones, boats |
| Brushed thrusters | H-bridge | Current only | Small boats |

## Valve Interface

Valves control fluid or gas flow, essential for pneumatic and hydraulic systems:

```go
type Valve interface {
    component.Actuator

    // Open fully opens the valve.
    Open(ctx context.Context) error

    // Shut fully closes the valve.
    // Note: Named "Shut" to avoid conflict with resource.Resource.Close()
    Shut(ctx context.Context) error

    // SetPosition sets the valve position (0.0 = closed, 1.0 = open).
    SetPosition(ctx context.Context, position float64) error

    // GetPosition returns the current position (0.0-1.0).
    GetPosition(ctx context.Context) (float64, error)

    // IsOpen returns true if valve is fully open.
    IsOpen(ctx context.Context) (bool, error)

    // IsClosed returns true if valve is fully closed.
    IsClosed(ctx context.Context) (bool, error)
}
```

### Binary vs Proportional Valves

**Binary valves** are either open or closed:

```go
// Solenoid valve - discrete states only
valve.Open(ctx)   // Fully open
valve.Shut(ctx)   // Fully closed

// SetPosition snaps to nearest state
valve.SetPosition(ctx, 0.3)  // Closes (< 0.5)
valve.SetPosition(ctx, 0.7)  // Opens (>= 0.5)
```

**Proportional valves** support continuous positioning:

```go
// Ball valve with stepper motor - continuous control
valve.SetPosition(ctx, 0.25)  // 25% open
valve.SetPosition(ctx, 0.75)  // 75% open

pos, _ := valve.GetPosition(ctx)  // Returns actual position
```

### Valve Types

| Type | Control | Response | Application |
|------|---------|----------|-------------|
| Solenoid | Binary | Fast | Pneumatics |
| Ball valve | Proportional | Slow | Flow control |
| Butterfly | Proportional | Fast | Large pipes |
| Check valve | Passive | - | Backflow prevention |

### Safety Considerations

Valves often control critical systems (coolant, fuel, air):

```go
// Emergency close
func (v *Valve) Stop(ctx context.Context) error {
    // For valves, "stop" typically means "close for safety"
    return v.Shut(ctx)
}

// Check state before critical operations
isOpen, _ := valve.IsOpen(ctx)
if !isOpen {
    return fmt.Errorf("valve not open, cannot proceed")
}
```

## Actuator Type Summary

| Actuator | Key Methods | Control Type | Application |
|----------|-------------|--------------|-------------|
| **Motor** | SetPower, SetVelocity, GoTo | Power/Velocity/Position | Wheels, conveyors |
| **Servo** | SetAngle, SetSpeed, SetTorqueLimit | Position | Pan/tilt, joints |
| **Stepper** | Step, SetMicrostepping, Home | Step count | 3D printers, CNC |
| **Thruster** | SetThrust, GetRPM, GetTemperature | Thrust percentage | ROVs, drones |
| **Valve** | Open, Shut, SetPosition | Binary/Proportional | Pneumatics, flow |
| **Gripper** | Open, Close, Grab | Binary/Force | Pick and place |
| **Base** | SetVelocity, MoveStraight, Spin | Velocity | Mobile robots |
| **Arm** | MoveToPosition, JointPositions | Cartesian/Joint | Manipulation |

## Gripper Interface

Grippers are end effectors for grasping objects:

```go
type Gripper interface {
    component.Actuator

    // Open fully opens the gripper.
    Open(ctx context.Context) error

    // Close fully closes the gripper.
    Close(ctx context.Context) error

    // Grab closes until resistance is felt (object grasped).
    Grab(ctx context.Context) (bool, error)

    // IsOpen returns true if gripper is fully open.
    IsOpen(ctx context.Context) (bool, error)
}
```

### Basic Open/Close

```go
// Open gripper before approaching object
gripper.Open(ctx)

// Close to grasp
gripper.Close(ctx)
```

### Force Sensing

Advanced grippers detect when they've grasped something:

```go
func (g *Gripper) Grab(ctx context.Context) (bool, error) {
    // Start closing
    g.motor.SetPower(ctx, -0.5)  // Close direction

    timeout := time.After(5 * time.Second)
    ticker := time.NewTicker(10 * time.Millisecond)

    for {
        select {
        case <-ctx.Done():
            g.motor.Stop(ctx)
            return false, ctx.Err()

        case <-timeout:
            g.motor.Stop(ctx)
            return false, fmt.Errorf("grab timeout")

        case <-ticker.C:
            // Check for resistance (current spike, stall)
            current := g.motor.GetCurrent()
            if current > g.grabThreshold {
                g.motor.Stop(ctx)
                return true, nil  // Object grasped
            }

            // Check if fully closed without object
            if g.atClosedLimit() {
                g.motor.Stop(ctx)
                return false, nil  // No object
            }
        }
    }
}
```

## Base Interface (Mobile Robots)

The Base interface abstracts mobile robot locomotion:

```go
type Base interface {
    component.Actuator

    // SetVelocity sets linear and angular velocity.
    SetVelocity(ctx context.Context, linear, angular float64) error

    // MoveStraight moves the robot forward by the specified distance.
    MoveStraight(ctx context.Context, distanceMm int, velocity float64) error

    // Spin rotates the robot in place by the specified angle.
    Spin(ctx context.Context, angleDeg, velocity float64) error

    // GetVelocities returns current linear and angular velocities.
    GetVelocities(ctx context.Context) (linear, angular float64, err error)
}
```

### Drive Types

**Differential Drive**: Two independently controlled wheels

```go
// Convert linear/angular to wheel velocities
func (b *DiffDrive) setWheelVelocities(linear, angular float64) {
    // v_left = linear - angular * (wheelbase / 2)
    // v_right = linear + angular * (wheelbase / 2)
    vLeft := linear - angular*b.wheelbase/2
    vRight := linear + angular*b.wheelbase/2

    b.leftMotor.SetVelocity(ctx, vLeft/b.wheelRadius)
    b.rightMotor.SetVelocity(ctx, vRight/b.wheelRadius)
}
```

**Mecanum Wheels**: Omnidirectional movement

```go
// Four-wheel mecanum kinematics
func (b *Mecanum) setWheelVelocities(vx, vy, angular float64) {
    // Each wheel contributes differently to motion
    fl := vx - vy - angular*(b.lx+b.ly)  // Front left
    fr := vx + vy + angular*(b.lx+b.ly)  // Front right
    rl := vx + vy - angular*(b.lx+b.ly)  // Rear left
    rr := vx - vy + angular*(b.lx+b.ly)  // Rear right

    b.motors["fl"].SetVelocity(ctx, fl)
    b.motors["fr"].SetVelocity(ctx, fr)
    b.motors["rl"].SetVelocity(ctx, rl)
    b.motors["rr"].SetVelocity(ctx, rr)
}
```

**Ackermann Steering**: Car-like steering

```go
func (b *Ackermann) SetVelocity(ctx context.Context, linear, angular float64) error {
    // Convert angular velocity to steering angle
    // Using bicycle model: angular = linear * tan(steering) / wheelbase
    if linear != 0 {
        steering := math.Atan(angular * b.wheelbase / linear)
        b.steeringServo.Move(ctx, steeringToDegrees(steering))
    }

    b.driveMotor.SetVelocity(ctx, linear)
    return nil
}
```

## Arm Interface (Manipulators)

Robotic arms require sophisticated interfaces:

```go
type Arm interface {
    component.Actuator

    // EndPosition returns the current end effector pose.
    EndPosition(ctx context.Context) (*spatialmath.Pose, error)

    // MoveToPosition moves the end effector to the target pose.
    MoveToPosition(ctx context.Context, pose *spatialmath.Pose) error

    // JointPositions returns current joint angles.
    JointPositions(ctx context.Context) ([]float64, error)

    // MoveToJointPositions sets joint angles directly.
    MoveToJointPositions(ctx context.Context, positions []float64) error
}
```

### Joint Space vs Task Space

**Joint space**: Direct control of each joint angle

```go
// Move each joint to specific angle
positions := []float64{0, -45, 90, 0, 45, 0}  // degrees
arm.MoveToJointPositions(ctx, positions)
```

- Direct, predictable
- Requires knowing valid configurations
- Good for predefined poses

**Task space**: Control end effector position/orientation

```go
// Move end effector to position
pose := spatialmath.NewPoseFromPoint(r3.Vector{X: 0.3, Y: 0.1, Z: 0.4})
arm.MoveToPosition(ctx, pose)
```

- More intuitive for applications
- Requires inverse kinematics
- May have multiple solutions or none

### Forward/Inverse Kinematics Concepts

**Forward kinematics**: Joints → End effector position

```
Given: Joint angles [θ1, θ2, θ3, ...]
Find: End effector pose (x, y, z, rotation)
```

Always has a unique solution.

**Inverse kinematics**: End effector position → Joints

```
Given: Desired end effector pose
Find: Joint angles to achieve it
```

May have:

- Multiple solutions (elbow up vs elbow down)
- No solution (target unreachable)
- Singularities (infinite solutions along an axis)

### Trajectory Planning

Moving from A to B requires planning:

```go
type Trajectory struct {
    Points []TrajectoryPoint
}

type TrajectoryPoint struct {
    Time     time.Duration
    Joints   []float64
    Velocity []float64
}

func (a *Arm) ExecuteTrajectory(ctx context.Context, traj *Trajectory) error {
    start := time.Now()

    for _, point := range traj.Points {
        // Wait for point time
        elapsed := time.Since(start)
        if point.Time > elapsed {
            time.Sleep(point.Time - elapsed)
        }

        // Move to point
        if err := a.MoveToJointPositions(ctx, point.Joints); err != nil {
            return err
        }
    }
    return nil
}
```

**Trajectory types**:

- Point-to-point: Direct joint interpolation
- Cartesian: Straight line in task space
- Spline: Smooth curves through waypoints

---

With sensors and actuators covered, Chapter 7 explores vision—the intersection of sensors and AI that enables robots to perceive and understand their environment.
