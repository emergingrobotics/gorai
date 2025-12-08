# Examples

> **Working Projects:** Complete, buildable robot projects demonstrating GoRAI patterns.

## Overview

Learn by example. These projects demonstrate GoRAI concepts in complete, working systems you can build and modify.

## Available Examples

| Example | Description | Complexity | Hardware Cost |
|---------|-------------|------------|---------------|
| [Hello Sensor](hello-sensor/_index.md) | CPU temperature sensor with NATS publishing | Beginner | $0 (no hardware) |
| [Pan-Tilt Platform](pan-tilt/_index.md) | Camera platform with servo control | Beginner | ~$150-350 |
| [Surface Vehicle](skimmer/_index.md) | Autonomous boat for water monitoring | Intermediate | ~$530 |

## Hello Sensor

**The essential starting point.** A complete sensor implementation demonstrating:
- Platform-specific code with build tags
- The Sensor interface
- Fake implementations for testing
- NATS publishing

[View Hello Sensor →](hello-sensor/_index.md)

## GoRAI-Sentinel (Pan-Tilt Platform)

**Your first multi-component robot.** A pan-tilt camera platform featuring:
- Dual servo control
- Camera integration
- ToF depth sensing
- Coordinated movement

[View Pan-Tilt Platform →](pan-tilt/_index.md)

## GoRAI-Skimmer (Surface Vehicle)

**A complete autonomous robot.** An autonomous surface vehicle including:
- GPS navigation
- Motor control
- Sensor fusion
- Mission coordination

[View Surface Vehicle →](skimmer/_index.md)

## Running Examples

```bash
# Clone the repository
git clone https://github.com/gorai/gorai
cd gorai

# Run hello-sensor (no hardware required)
go run ./examples/hello-sensor -fake

# Run with live NATS monitoring
nats sub "gorai.>" &
go run ./examples/hello-sensor -fake
```

<!-- website-only -->
!!! tip "Start with Hello Sensor"
    Even if you're building something complex, start with Hello Sensor to understand the patterns.
<!-- /website-only -->
