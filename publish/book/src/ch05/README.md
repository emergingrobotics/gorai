# Components: Actuators

Actuators are components that change the physical world. This chapter covers motors, mobile bases, robotic arms, and grippers.

## The Actuator Interface

All actuators implement the base `Actuator` interface:

```go
type Actuator interface {
    resource.Resource

    // IsMoving returns true if the actuator is currently in motion
    IsMoving(ctx context.Context) (bool, error)

    // Stop immediately halts all motion
    Stop(ctx context.Context) error
}
```

The `Stop()` method is critical for safety—it must immediately halt motion regardless of what the actuator is doing.

## Motors

Motors are the most common actuators in robotics. Gorai provides a comprehensive motor interface:

```go
type Motor interface {
    Actuator

    // SetPower sets motor power from -1.0 (full reverse) to 1.0 (full forward)
    // 0.0 stops the motor
    SetPower(ctx context.Context, power float64) error

    // SetVelocity sets target velocity in RPM
    // Requires a motor with velocity control (encoder + PID)
    SetVelocity(ctx context.Context, velocity float64) error

    // GoTo moves to an absolute position (in revolutions) at given velocity (RPM)
    // Requires a motor with position control
    GoTo(ctx context.Context, position, velocity float64) error

    // GoFor rotates for a number of revolutions at given RPM
    // Positive values = forward, negative values = reverse
    GoFor(ctx context.Context, rpm, revolutions float64) error

    // GetPosition returns current position in revolutions
    // Returns error if motor doesn't support position reporting
    GetPosition(ctx context.Context) (float64, error)

    // GetVelocity returns current velocity in RPM
    // Returns error if motor doesn't support velocity reporting
    GetVelocity(ctx context.Context) (float64, error)

    // ResetZeroPosition sets current position as zero (with optional offset)
    ResetZeroPosition(ctx context.Context, offset float64) error

    // IsPowered returns whether motor is powered and the current power level
    IsPowered(ctx context.Context) (bool, float64, error)

    // Properties returns motor capabilities
    Properties(ctx context.Context) (Properties, error)
}

type Properties struct {
    PositionReporting bool  // Can report position
    VelocityReporting bool  // Can report velocity
    SupportsGoTo      bool  // Supports position control
}
```

### Motor Types

| Type | Control Method | Use Case |
|------|----------------|----------|
| **DC Motor** | PWM power | Simple drive wheels |
| **DC + Encoder** | PID velocity | Precise wheel control |
| **Servo** | Position command | Pan/tilt, joints |
| **Stepper** | Step pulses | Precise positioning |
| **BLDC** | ESC commands | High-speed, drones |

### Using Motors

**Simple Power Control:**

```go
func driveForward(motor motor.Motor, duration time.Duration) error {
    ctx := context.Background()

    // Start moving
    if err := motor.SetPower(ctx, 0.5); err != nil {
        return err
    }

    // Wait
    time.Sleep(duration)

    // Stop
    return motor.Stop(ctx)
}
```

**Velocity Control:**

```go
func maintainSpeed(motor motor.Motor, targetRPM float64) error {
    ctx := context.Background()

    // Check if motor supports velocity control
    props, _ := motor.Properties(ctx)
    if !props.VelocityReporting {
        return fmt.Errorf("motor doesn't support velocity control")
    }

    // Set target velocity (motor's internal PID handles the rest)
    return motor.SetVelocity(ctx, targetRPM)
}
```

**Position Control:**

```go
func moveToPosition(motor motor.Motor, position float64) error {
    ctx := context.Background()

    // Check if motor supports position control
    props, _ := motor.Properties(ctx)
    if !props.SupportsGoTo {
        return fmt.Errorf("motor doesn't support position control")
    }

    // Move to position at 60 RPM
    if err := motor.GoTo(ctx, position, 60); err != nil {
        return err
    }

    // Wait for move to complete
    for {
        moving, _ := motor.IsMoving(ctx)
        if !moving {
            break
        }
        time.Sleep(50 * time.Millisecond)
    }

    return nil
}
```

**Relative Movement:**

```go
func rotateRevolutions(motor motor.Motor, revolutions float64) error {
    ctx := context.Background()

    // Rotate forward 2.5 revolutions at 30 RPM
    return motor.GoFor(ctx, 30, revolutions)
}
```

## Mobile Bases

A `Base` represents a mobile robot platform with coordinated wheel control:

```go
type Base interface {
    Actuator

    // SetVelocity sets linear and angular velocity
    // linear: m/s forward (positive) or backward (negative)
    // angular: rad/s counterclockwise (positive) or clockwise (negative)
    SetVelocity(ctx context.Context, linear, angular float64) error

    // MoveStraight moves forward/backward a specified distance
    // distance: meters (positive = forward, negative = backward)
    // velocity: m/s (always positive)
    // Blocks until complete
    MoveStraight(ctx context.Context, distance, velocity float64) error

    // Spin rotates in place
    // angle: radians (positive = counterclockwise, negative = clockwise)
    // velocity: rad/s (always positive)
    // Blocks until complete
    Spin(ctx context.Context, angle, velocity float64) error

    // SetPower sets individual wheel powers (for testing/manual control)
    // Values from -1.0 to 1.0
    SetPower(ctx context.Context, left, right float64) error

    // Properties returns base capabilities
    Properties(ctx context.Context) (BaseProperties, error)
}

type BaseProperties struct {
    WheelCircumference float64  // Wheel circumference in meters
    WheelBase          float64  // Distance between wheels in meters
    MaxLinearVelocity  float64  // Max forward speed in m/s
    MaxAngularVelocity float64  // Max rotation speed in rad/s
}
```

### Base Types

| Type | Description |
|------|-------------|
| **Differential Drive** | Two independently driven wheels (most common) |
| **Holonomic** | Can move in any direction (mecanum/omni wheels) |
| **Ackermann** | Car-like steering |
| **Tracked** | Tank-style tracks |

### Using Bases

**Velocity Control:**

```go
func followPath(base base.Base) {
    ctx := context.Background()

    // Move forward at 0.5 m/s while turning slightly left
    base.SetVelocity(ctx, 0.5, 0.1)

    time.Sleep(2 * time.Second)

    // Stop
    base.Stop(ctx)
}
```

**Distance-Based Movement:**

```go
func navigateSquare(base base.Base, sideLength float64) error {
    ctx := context.Background()

    for i := 0; i < 4; i++ {
        // Move forward one side length
        if err := base.MoveStraight(ctx, sideLength, 0.3); err != nil {
            return err
        }

        // Turn 90 degrees (π/2 radians)
        if err := base.Spin(ctx, math.Pi/2, 0.5); err != nil {
            return err
        }
    }

    return nil
}
```

## Robotic Arms

Arms provide multi-DOF manipulation:

```go
type Arm interface {
    Actuator

    // EndPosition returns the current end effector pose
    EndPosition(ctx context.Context) (Pose, error)

    // MoveToPosition moves the end effector to target pose
    // Blocks until complete or error
    MoveToPosition(ctx context.Context, pose Pose) error

    // JointPositions returns current joint angles in radians
    JointPositions(ctx context.Context) ([]float64, error)

    // MoveToJointPositions moves all joints to specified angles
    // Blocks until complete or error
    MoveToJointPositions(ctx context.Context, positions []float64) error

    // ModelFrame returns the kinematic model
    ModelFrame(ctx context.Context) (ModelFrame, error)
}

type Pose struct {
    // Position in meters
    X, Y, Z float64

    // Orientation as axis-angle
    // OX, OY, OZ: rotation axis (unit vector)
    // Theta: rotation angle in radians
    OX, OY, OZ, Theta float64
}

type ModelFrame struct {
    DOF         int              // Degrees of freedom
    JointNames  []string         // Names of each joint
    JointLimits []JointLimit     // Min/max for each joint
}

type JointLimit struct {
    Min float64  // Minimum angle in radians
    Max float64  // Maximum angle in radians
}
```

### Using Arms

**Cartesian Movement:**

```go
func pickObject(arm arm.Arm, objectPose Pose) error {
    ctx := context.Background()

    // Move above the object
    approachPose := objectPose
    approachPose.Z += 0.1  // 10cm above

    if err := arm.MoveToPosition(ctx, approachPose); err != nil {
        return fmt.Errorf("failed to move to approach: %w", err)
    }

    // Move down to object
    if err := arm.MoveToPosition(ctx, objectPose); err != nil {
        return fmt.Errorf("failed to move to object: %w", err)
    }

    return nil
}
```

**Joint Space Movement:**

```go
func moveToHome(arm arm.Arm) error {
    ctx := context.Background()

    // Get model info
    model, _ := arm.ModelFrame(ctx)

    // Create home position (all joints at 0)
    homePositions := make([]float64, model.DOF)

    return arm.MoveToJointPositions(ctx, homePositions)
}
```

## Grippers

Grippers are end effectors for grasping:

```go
type Gripper interface {
    Actuator

    // Open opens the gripper to maximum width
    Open(ctx context.Context) error

    // Grab closes the gripper and returns true if object detected
    Grab(ctx context.Context) (bool, error)

    // IsOpen returns true if gripper is fully open
    IsOpen(ctx context.Context) (bool, error)
}
```

### Using Grippers

```go
func pickAndPlace(arm arm.Arm, gripper gripper.Gripper, pickPose, placePose Pose) error {
    ctx := context.Background()

    // Open gripper
    if err := gripper.Open(ctx); err != nil {
        return err
    }

    // Move to pick position
    if err := arm.MoveToPosition(ctx, pickPose); err != nil {
        return err
    }

    // Close gripper and check for object
    grabbed, err := gripper.Grab(ctx)
    if err != nil {
        return err
    }
    if !grabbed {
        return fmt.Errorf("failed to grab object")
    }

    // Move to place position
    if err := arm.MoveToPosition(ctx, placePose); err != nil {
        return err
    }

    // Release
    return gripper.Open(ctx)
}
```

## Control Patterns

### Emergency Stop

Always implement emergency stop capability:

```go
type EmergencyStop struct {
    actuators []component.Actuator
}

func (e *EmergencyStop) Stop() {
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()

    // Stop all actuators in parallel
    var wg sync.WaitGroup
    for _, a := range e.actuators {
        wg.Add(1)
        go func(actuator component.Actuator) {
            defer wg.Done()
            actuator.Stop(ctx)
        }(a)
    }
    wg.Wait()
}
```

### Velocity Ramping

Smooth acceleration prevents mechanical shock:

```go
func rampVelocity(motor motor.Motor, target float64, duration time.Duration) error {
    ctx := context.Background()

    current, _ := motor.GetVelocity(ctx)
    steps := 20
    stepDuration := duration / time.Duration(steps)
    increment := (target - current) / float64(steps)

    for i := 0; i < steps; i++ {
        current += increment
        if err := motor.SetVelocity(ctx, current); err != nil {
            return err
        }
        time.Sleep(stepDuration)
    }

    return motor.SetVelocity(ctx, target)
}
```

### Position Monitoring

Track position during movement:

```go
func moveWithMonitoring(motor motor.Motor, target float64) error {
    ctx := context.Background()

    // Start movement
    if err := motor.GoTo(ctx, target, 60); err != nil {
        return err
    }

    // Monitor until complete
    for {
        pos, _ := motor.GetPosition(ctx)
        moving, _ := motor.IsMoving(ctx)

        log.Printf("Position: %.3f / %.3f, Moving: %v", pos, target, moving)

        if !moving {
            break
        }

        time.Sleep(100 * time.Millisecond)
    }

    // Verify we reached target
    finalPos, _ := motor.GetPosition(ctx)
    if math.Abs(finalPos-target) > 0.01 {
        return fmt.Errorf("did not reach target: %.3f vs %.3f", finalPos, target)
    }

    return nil
}
```

## Configuration

### Motor Configuration

```json
{
  "name": "left_wheel",
  "type": "component",
  "subtype": "motor",
  "model": "gorai:driver:gpio_motor",
  "config": {
    "pin_pwm": 12,
    "pin_direction": 18,
    "pin_enable": 19,
    "max_rpm": 300,
    "encoder_pin_a": 20,
    "encoder_pin_b": 21,
    "ticks_per_revolution": 1200,
    "pid_p": 0.5,
    "pid_i": 0.1,
    "pid_d": 0.05
  }
}
```

### Base Configuration

```json
{
  "name": "base",
  "type": "component",
  "subtype": "base",
  "model": "gorai:component:differential_drive",
  "config": {
    "wheel_circumference_m": 0.314,
    "wheel_base_m": 0.25,
    "max_linear_velocity_m_s": 1.0,
    "max_angular_velocity_rad_s": 3.14
  },
  "depends_on": ["left_wheel", "right_wheel"]
}
```

### Arm Configuration

```json
{
  "name": "arm",
  "type": "component",
  "subtype": "arm",
  "model": "gorai:driver:dynamixel_arm",
  "config": {
    "serial_port": "/dev/ttyUSB0",
    "baud_rate": 1000000,
    "joint_ids": [1, 2, 3, 4, 5, 6],
    "joint_limits": [
      {"min": -3.14, "max": 3.14},
      {"min": -1.57, "max": 1.57},
      {"min": -2.0, "max": 2.0},
      {"min": -3.14, "max": 3.14},
      {"min": -1.57, "max": 1.57},
      {"min": -3.14, "max": 3.14}
    ]
  }
}
```

## Testing with Fakes

### Fake Motor

```go
type FakeMotor struct {
    resource.Named

    mu        sync.Mutex
    position  float64
    velocity  float64
    power     float64
    moving    bool
    powered   bool
}

func (f *FakeMotor) SetPower(ctx context.Context, power float64) error {
    f.mu.Lock()
    defer f.mu.Unlock()

    // Clamp to valid range
    if power < -1.0 {
        power = -1.0
    } else if power > 1.0 {
        power = 1.0
    }

    f.power = power
    f.powered = power != 0
    f.moving = power != 0

    // Simulate velocity based on power
    f.velocity = power * 100  // Max 100 RPM

    return nil
}

func (f *FakeMotor) GoTo(ctx context.Context, position, velocity float64) error {
    f.mu.Lock()
    defer f.mu.Unlock()

    f.moving = true
    f.powered = true
    f.velocity = velocity

    // In real fake, would simulate movement over time
    // For simple tests, just set position immediately
    f.position = position
    f.moving = false
    f.velocity = 0

    return nil
}

func (f *FakeMotor) GetPosition(ctx context.Context) (float64, error) {
    f.mu.Lock()
    defer f.mu.Unlock()
    return f.position, nil
}

func (f *FakeMotor) IsMoving(ctx context.Context) (bool, error) {
    f.mu.Lock()
    defer f.mu.Unlock()
    return f.moving, nil
}

func (f *FakeMotor) Stop(ctx context.Context) error {
    f.mu.Lock()
    defer f.mu.Unlock()

    f.power = 0
    f.velocity = 0
    f.moving = false
    f.powered = false

    return nil
}
```

### Using Fakes in Tests

```go
func TestBaseMovement(t *testing.T) {
    // Create fake motors
    leftMotor := fake.NewMotor(resource.NewName("gorai", "component", "motor", "left"))
    rightMotor := fake.NewMotor(resource.NewName("gorai", "component", "motor", "right"))

    // Create base with fake motors
    base := NewDifferentialBase(leftMotor, rightMotor)

    // Test movement
    err := base.MoveStraight(context.Background(), 1.0, 0.5)
    require.NoError(t, err)

    // Verify motors were commanded correctly
    leftPos, _ := leftMotor.GetPosition(context.Background())
    rightPos, _ := rightMotor.GetPosition(context.Background())

    // For 1 meter with 0.314m wheel circumference = ~3.18 revolutions
    assert.InDelta(t, 3.18, leftPos, 0.1)
    assert.InDelta(t, 3.18, rightPos, 0.1)
}
```

## Best Practices

### 1. Always Handle Stop

```go
func runMotor(motor motor.Motor) {
    ctx := context.Background()

    // Ensure motor stops even on panic
    defer motor.Stop(ctx)

    motor.SetPower(ctx, 0.5)

    // ... motor logic ...
}
```

### 2. Check Capabilities

```go
func configureMotor(motor motor.Motor) error {
    ctx := context.Background()
    props, _ := motor.Properties(ctx)

    if !props.SupportsGoTo {
        log.Println("Motor doesn't support position control, using power mode")
        // Fall back to simpler control
    }

    return nil
}
```

### 3. Use Timeouts

```go
func moveWithTimeout(motor motor.Motor, position float64) error {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := motor.GoTo(ctx, position, 30); err != nil {
        return err
    }

    // Wait for completion with timeout
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            motor.Stop(context.Background())
            return ctx.Err()
        case <-ticker.C:
            moving, _ := motor.IsMoving(ctx)
            if !moving {
                return nil
            }
        }
    }
}
```

### 4. Validate Commands

```go
func (m *Motor) SetPower(ctx context.Context, power float64) error {
    // Validate input
    if power < -1.0 || power > 1.0 {
        return fmt.Errorf("power must be between -1.0 and 1.0, got %f", power)
    }

    // Apply power limit from config
    if m.maxPower < 1.0 {
        if power > m.maxPower {
            power = m.maxPower
        } else if power < -m.maxPower {
            power = -m.maxPower
        }
    }

    return m.driver.SetPower(power)
}
```
