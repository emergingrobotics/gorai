# Chapter 5: Actuators

> **In This Chapter:** Master actuator interfaces—motors, servos, grippers, mobile bases, and arms. Learn control patterns and safety considerations.

## Overview

If sensors are how robots perceive, actuators are how they act. GoRAI provides a hierarchy of actuator interfaces from the basic `Actuator` (with `Stop()`) to specialized interfaces like `Motor`, `Servo`, and `Arm`.

This chapter covers all actuator types, their interfaces, and the control patterns that make them work safely and effectively. Safety is paramount—every actuator can stop instantly, and control loops handle failures gracefully.

## What You'll Learn

After reading this chapter, you'll understand:

- The Actuator interface and safety-first design
- Motor interface: power, velocity, and position control
- Different motor types and their characteristics
- Control patterns: open-loop, closed-loop, PID
- Servo, Gripper, Base, and Arm interfaces

## Chapter Contents

| Section | Description |
|---------|-------------|
| [The Actuator Interface](actuator.md) | `IsMoving()`, `Stop()`, safety patterns |
| [Motors](motor.md) | SetPower, SetVelocity, GoTo, GoFor, Properties |
| [Motor Types](motortypes.md) | DC, stepper, servo, brushless |
| [Control Patterns](control.md) | Open-loop, closed-loop, PID basics |
| [Servos](servo.md) | Angle-based positioning, PWM control |
| [Bases & Arms](base_arm.md) | Mobile robots and manipulators |

## Key Takeaways

- **Safety first**: Every actuator has `Stop()` and `IsMoving()`
- **Motors** support multiple control modes: power, velocity, position
- **Control loops** can be open (simple) or closed (feedback-driven)
- **Servos** provide angle-based positioning
- **Bases** abstract mobile robot drivetrains (differential, mecanum, etc.)
- **Arms** handle kinematics and trajectory planning

## Prerequisites

This chapter assumes you've read:
- [Chapter 2: Architecture](../../part1-getting-started/ch02-architecture/_index.md) — Understanding components
- [Chapter 3: NATS](../ch03-nats/_index.md) — Command/response patterns

## Quick Reference

```go
// The core Actuator interface
type Actuator interface {
    Component
    IsMoving(ctx context.Context) (bool, error)
    Stop(ctx context.Context) error
}

// Motor extends Actuator
type Motor interface {
    Actuator
    SetPower(ctx context.Context, power float64) error
    GoTo(ctx context.Context, position float64) error
    GetPosition(ctx context.Context) (float64, error)
    // ...more methods
}
```

<!-- book-only -->
*Actuator code requires extra care. A bug in sensor code might give bad readings; a bug in actuator code might damage hardware or injure people. The patterns in this chapter emphasize safety.*
<!-- /book-only -->

<!-- website-only -->
!!! danger "Safety Critical"
    Actuator code can cause physical harm. Always implement proper `Stop()` handlers and test with hardware disconnected first.
<!-- /website-only -->
