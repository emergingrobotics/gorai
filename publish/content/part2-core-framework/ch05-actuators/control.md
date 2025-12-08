## 5.4 Control Patterns

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

**GoRAI on Linux**: Higher-level control
- Slower response (1-10ms)
- Suitable for position control, trajectory tracking
- Not suitable for commutation, current control

Choose the right level for your control loop requirements.
