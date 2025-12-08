## 5.5 Servo Interface

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


## 5.6 Gripper Interface

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

### Grasp Detection

Beyond simple force sensing:

```go
type GraspStatus struct {
    Grasping     bool
    ObjectWidth  float64  // Estimated from encoder position
    GripForce    float64  // From current or force sensor
}

func (g *Gripper) GetGraspStatus(ctx context.Context) (GraspStatus, error) {
    position := g.encoder.GetPosition()
    current := g.motor.GetCurrent()

    return GraspStatus{
        Grasping:    current > g.holdThreshold,
        ObjectWidth: g.positionToWidth(position),
        GripForce:   g.currentToForce(current),
    }, nil
}
```
