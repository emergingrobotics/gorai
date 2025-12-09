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

Motors are the most common actuators. The Motor interface from `component/motor/motor.go` provides comprehensive control:

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

    // Move moves the servo to the specified angle (degrees).
    Move(ctx context.Context, angleDeg float64) error

    // GetPosition returns the current angle (degrees).
    GetPosition(ctx context.Context) (float64, error)
}
```

### Angle-Based Positioning

```go
// Center position
servo.Move(ctx, 90)

// Full left
servo.Move(ctx, 0)

// Full right
servo.Move(ctx, 180)
```

Angle ranges vary by servo:

- Standard RC servos: 0-180°
- Extended range: 0-270° or 0-360°
- Continuous rotation: Angle maps to speed/direction

### PWM Control

RC servos use PWM pulse width for position:

```go
type RCServo struct {
    pwm       gpio.PWMPin
    minAngle  float64        // Typically 0
    maxAngle  float64        // Typically 180
    minPulse  time.Duration  // Typically 500µs - 1ms
    maxPulse  time.Duration  // Typically 2ms - 2.5ms
}

func (s *RCServo) Move(ctx context.Context, angleDeg float64) error {
    // Clamp to valid range
    if angleDeg < s.minAngle {
        angleDeg = s.minAngle
    }
    if angleDeg > s.maxAngle {
        angleDeg = s.maxAngle
    }

    // Map angle to pulse width
    fraction := (angleDeg - s.minAngle) / (s.maxAngle - s.minAngle)
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
            if err := s.Move(ctx, a); err != nil {
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
