# Chapter 8: Behaviors

> **In This Chapter:** Understand how robots make decisions. Learn about behavior trees, state machines, and reactive architectures.

## Overview

How does a robot decide what to do next? Behaviors are the decision-making layer that sits above services and components. They observe the robot's state and environment, then select appropriate actions.

This chapter introduces behavior patterns used in robotics: finite state machines for simple sequential behaviors, behavior trees for complex hierarchical decisions, and reactive architectures for real-time response.

## What You'll Learn

After reading this chapter, you'll understand:

- Why behavior abstraction matters
- Finite state machines for sequential behaviors
- Behavior trees for complex decision making
- Reactive/subsumption architectures
- Implementing behaviors in GoRAI

## Chapter Contents

This chapter covers behavioral architecture comprehensively:

1. **The Behavior Problem** — Why we need behavior abstraction
2. **Finite State Machines** — States, transitions, guards
3. **Behavior Trees** — Selectors, sequences, decorators
4. **Reactive Architectures** — Priority-based, subsumption
5. **GoRAI Behavior Patterns** — Implementation strategies
6. **Testing Behaviors** — Simulation and verification

## Key Takeaways

- **Behaviors** decide what the robot does based on state and environment
- **FSMs** work well for simple, linear sequences
- **Behavior trees** scale to complex, hierarchical decisions
- **Reactive systems** prioritize real-time responsiveness
- Choose the right pattern for your complexity level

## Prerequisites

This chapter assumes you've read:
- [Chapter 7: Services](../ch07-services/_index.md) — Services that behaviors control
- Basic programming concepts (state machines, trees)

## Quick Reference

```go
// Simple behavior interface
type Behavior interface {
    // Update is called each tick
    Update(ctx context.Context, state *RobotState) (Action, error)

    // Name identifies this behavior
    Name() string
}

// Behavior tree node
type BTNode interface {
    Tick(ctx context.Context, bb *Blackboard) Status
}

type Status int
const (
    Running Status = iota
    Success
    Failure
)
```

<!-- book-only -->
*Behavior design is as much art as science. Start simple (FSMs), add complexity only when needed (behavior trees), and always ensure safety behaviors have priority.*
<!-- /book-only -->

<!-- website-only -->
!!! tip "Start Simple"
    Begin with finite state machines. Graduate to behavior trees when FSMs become unwieldy.
<!-- /website-only -->
