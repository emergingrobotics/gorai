# Chapter 5: Components - Actuators

Actuators transform electrical signals into physical motion. They're how your robot interacts with the world—wheels that turn, arms that reach, grippers that grasp.

## 5.1 The Actuator Interface

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

Actuators can cause harm. GoRAI's actuator design prioritizes safety:

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
