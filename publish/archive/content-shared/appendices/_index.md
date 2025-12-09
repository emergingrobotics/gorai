# Appendices

> **Quick Reference:** Command references, protocol definitions, hardware compatibility, troubleshooting, and glossary.

## Overview

The appendices provide quick-reference material that supports the main text. Use these when you need specific details without the surrounding explanation.

## Contents

| Appendix | Description |
|----------|-------------|
| [NATS Topics Reference](topics.md) | Topic naming conventions, wildcards, examples |
| [Protocol Buffers Reference](protobuf.md) | Message definitions, headers, common types |
| [Hardware Compatibility](hardware.md) | Supported boards, sensors, actuators |
| [Troubleshooting](troubleshooting.md) | Common issues and solutions |
| [Glossary](glossary.md) | Terms and definitions |

## Quick Links

### Commands

```bash
# Start NATS
podman run -d --name nats -p 4222:4222 nats:latest

# Check NATS status
nats server ping

# Subscribe to all GoRAI topics
nats sub "gorai.>"

# Run hello-sensor
go run ./examples/hello-sensor
```

### Topic Patterns

| Pattern | Example | Use Case |
|---------|---------|----------|
| `gorai.<node>.<component>.data` | `gorai.robot1.imu.data` | Sensor readings |
| `gorai.<node>.<component>.cmd` | `gorai.robot1.motor.cmd` | Commands |
| `gorai.<node>.<service>.req` | `gorai.robot1.vision.req` | Service requests |

<!-- book-only -->
*Keep these appendices bookmarked or printed for quick reference during development.*
<!-- /book-only -->

<!-- website-only -->
!!! tip "Bookmark This"
    The appendices are designed for quick lookup during development.
<!-- /website-only -->
