# Chapter 4: Sensors

> **In This Chapter:** Understand the Sensor interface, built-in sensor types, data representations, and the fake pattern for testing.

## Overview

Sensors are how robots perceive the world. In GoRAI, all sensors implement a common interface that returns readings as key-value maps. This simple abstraction handles everything from temperature sensors to IMUs to GPS receivers.

This chapter covers the Sensor interface in depth, surveys the built-in sensor types, explains the Protocol Buffer data types for sensor data, and introduces the fake pattern that makes sensors testable without hardware.

## What You'll Learn

After reading this chapter, you'll understand:

- The Sensor interface and why it uses `map[string]any`
- Built-in sensor types: Temperature, IMU, GPS, Encoder, Range
- Protocol Buffer definitions for sensor data
- The fake pattern for testing sensor code

## Chapter Contents

| Section | Description |
|---------|-------------|
| [The Sensor Interface](interface.md) | `Readings()` method, standard keys, timestamps |
| [Built-in Sensors](builtin.md) | Temperature, IMU, GPS, Encoder, Range finders |
| [Data Types](datatypes.md) | Protocol Buffer definitions, headers, covariance |
| [Fake Sensors](fakes.md) | Test doubles, configurable behavior, error injection |

## Key Takeaways

- **All sensors** implement `Readings(ctx) (map[string]any, error)`
- **Standard keys** provide consistency: `temperature_c`, `acceleration_x`, `latitude`
- **Timestamps** are always included for data fusion
- **Protocol Buffers** define wire format for NATS transmission
- **Fakes** mirror real sensor behavior for testing

## Prerequisites

This chapter assumes you've read:
- [Chapter 2: Architecture](../../part1-getting-started/ch02-architecture/_index.md) — Understanding resources
- [Chapter 3: NATS](../ch03-nats/_index.md) — How sensor data flows over NATS

## Quick Reference

```go
// The core Sensor interface
type Sensor interface {
    Component
    Readings(ctx context.Context) (map[string]any, error)
}

// Example usage
readings, err := tempSensor.Readings(ctx)
if err != nil {
    return err
}
celsius := readings["temperature_c"].(float64)
```

<!-- book-only -->
*The Hello Sensor example in Chapter 11 implements a complete temperature sensor. Consider reading that chapter alongside this one for concrete examples.*
<!-- /book-only -->

<!-- website-only -->
!!! example "See It In Action"
    The [Hello Sensor Tutorial](../../part3-development/ch11-hello-sensor/_index.md) implements a complete temperature sensor from scratch.
<!-- /website-only -->
