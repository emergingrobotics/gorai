---
title: "Core Concepts"
description: "Understand Gorai's architecture and design"
weight: 30
---

# Core Concepts

This page introduces the key concepts you'll use throughout Gorai development.

## Resources

Everything in Gorai is a **Resource**. Resources are the fundamental building block with a consistent interface:

- **Name** — Unique identifier
- **Type** — Category (sensor, actuator, service)
- **Configure** — Accept configuration
- **Start/Stop** — Lifecycle management

## Components

Components are resources that interface with hardware:

| Type | Purpose | Examples |
|------|---------|----------|
| **Sensor** | Measure physical quantities | Temperature, IMU, GPS |
| **Actuator** | Produce physical action | Motor, servo, relay |
| **Camera** | Capture visual data | USB camera, CSI camera |

## Services

Services are resources that provide capabilities:

- **Vision** — Object detection, image processing
- **Navigation** — Path planning, localization
- **Custom** — Your application-specific services

## NATS Topics

Gorai uses NATS for all communication. Topics follow a hierarchical naming convention:

```
gorai.{robot}.{component-type}.{name}.{action}
```

Examples:
- `gorai.robot1.sensor.temperature.reading`
- `gorai.robot1.actuator.motor.command`
- `gorai.robot1.service.vision.detect`

## NWS and NWC

Gorai nodes come in two types:

- **NWS (Node With Sensors)** — Microcontrollers running TinyGo, interfacing with hardware
- **NWC (Node With Compute)** — Linux devices running full Go, handling computation

These communicate over NATS, allowing you to distribute processing across devices.

## Next Steps

- [Working with Components](/docs/guides/components/)
- [NATS Messaging Guide](/docs/guides/nats/)
