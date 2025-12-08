# Chapter 9: Coordinators

> **In This Chapter:** Learn to orchestrate complex, multi-step missions and coordinate multiple robots working together.

## Overview

Coordinators sit at the top of GoRAI's architecture, orchestrating missions that span multiple behaviors, services, and even multiple robots. They handle the big picture: task sequencing, resource allocation, failure recovery, and inter-robot coordination.

This chapter covers mission orchestration for single robots and coordination patterns for multi-robot systems.

## What You'll Learn

After reading this chapter, you'll understand:

- The role of coordinators in robot architecture
- Mission planning and task sequencing
- Resource allocation and conflict resolution
- Failure detection and recovery
- Multi-robot coordination patterns

## Chapter Contents

This chapter covers coordination comprehensively:

1. **The Coordinator Role** — Orchestrating behaviors and services
2. **Mission Planning** — Breaking goals into tasks
3. **Task Execution** — Sequencing, parallelism, dependencies
4. **Failure Recovery** — Detecting and handling failures
5. **Multi-Robot Systems** — Fleet coordination patterns
6. **Practical Examples** — Complete coordinator implementations

## Key Takeaways

- **Coordinators** orchestrate high-level missions
- **Missions** decompose into **tasks** that map to behaviors
- **Task graphs** handle dependencies and parallelism
- **Failure recovery** is built-in, not an afterthought
- **Multi-robot** coordination uses NATS for communication

## Prerequisites

This chapter assumes you've read:
- [Chapter 7: Services](../ch07-services/_index.md) — Services that coordinators orchestrate
- [Chapter 8: Behaviors](../ch08-behaviors/_index.md) — Behaviors that execute tasks

## Quick Reference

```go
// Coordinator interface
type Coordinator interface {
    // Start begins mission execution
    Start(ctx context.Context, mission Mission) error

    // Status returns current mission state
    Status(ctx context.Context) (MissionStatus, error)

    // Cancel aborts the current mission
    Cancel(ctx context.Context) error
}

// Mission defines what to accomplish
type Mission struct {
    Name        string
    Tasks       []Task
    Constraints []Constraint
}
```

<!-- book-only -->
*This chapter completes the architecture from sensors to coordinators. A well-designed robot uses all layers: components provide hardware abstraction, services provide capabilities, behaviors make decisions, and coordinators orchestrate everything toward goals.*
<!-- /book-only -->

<!-- website-only -->
!!! success "Architecture Complete"
    With coordinators, you have the complete picture: Components → Services → Behaviors → Coordinators.
<!-- /website-only -->
