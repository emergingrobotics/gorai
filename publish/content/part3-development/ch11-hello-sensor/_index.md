# Chapter 11: Hello Sensor Deep Dive

> **In This Chapter:** Build a complete, production-quality temperature sensor from scratch. Learn patterns you'll use for every GoRAI component.

## Overview

The best way to learn GoRAI is to build something real. This chapter walks through the "hello-sensor" example in complete detail—a CPU temperature sensor that reads thermal data, publishes to NATS, and includes proper testing infrastructure.

This isn't a toy example. The patterns here—platform abstraction, fake implementations, statistics tracking, graceful shutdown—are exactly what you'll use in production code.

## What You'll Learn

After reading this chapter, you'll understand:

- Complete structure of a GoRAI component
- Platform-specific code with build tags
- The reader/sensor separation pattern
- Implementing the Sensor interface
- Creating fake implementations for testing
- Publishing data over NATS

## Chapter Contents

| Section | Description |
|---------|-------------|
| [Overview](overview.md) | What we're building and why |
| [The Reader](reader.md) | Platform abstraction for temperature reading |
| [The Sensor](sensor.md) | Component implementation with statistics |
| [Main Program](main.md) | Wiring it together with NATS |

## Key Takeaways

- **Readers** abstract platform-specific I/O
- **Sensors** implement the component interface
- **Fakes** enable testing without hardware
- **Build tags** select platform implementations
- **Graceful shutdown** handles signals properly

## Prerequisites

This chapter assumes you've read:
- [Chapter 4: Sensors](../../part2-core-framework/ch04-sensors/_index.md)
- [Chapter 10: Development Environment](../ch10-devenv/_index.md)

You should have a working Go environment with NATS available.

## Architecture Overview

```
┌─────────────────────────────────────────┐
│              hello-sensor               │
├─────────────────────────────────────────┤
│  main.go                                │
│    ├── Create node                      │
│    ├── Create reader (platform-specific)│
│    ├── Create sensor component          │
│    └── Run publish loop                 │
├─────────────────────────────────────────┤
│  reader/                                │
│    ├── reader.go (interface)            │
│    ├── linux.go (thermal zones)         │
│    ├── darwin.go (osx-cpu-temp)         │
│    └── unsupported.go (stub)            │
├─────────────────────────────────────────┤
│  sensor/                                │
│    ├── temperature.go (component)       │
│    └── fake/fake.go (test double)       │
└─────────────────────────────────────────┘
```

<!-- book-only -->
*This is the most important chapter in Part III. The patterns established here recur throughout GoRAI development. Take time to understand each piece.*
<!-- /book-only -->

<!-- website-only -->
!!! success "Reference Implementation"
    This example serves as a reference for all GoRAI component development. Bookmark it!
<!-- /website-only -->
