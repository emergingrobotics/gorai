## 5.2 Motor Interface

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
