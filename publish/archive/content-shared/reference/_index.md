# Reference

> **Technical Reference:** CLI commands, configuration options, and API specifications.

## Overview

This section provides complete technical reference documentation. Use it when you know what you're looking for and need exact syntax or options.

## Contents

| Reference | Description |
|-----------|-------------|
| [CLI Reference](cli.md) | Command-line tools and options |
| [Configuration](configuration.md) | JSON configuration format and options |
| [Topic Naming](topic-naming.md) | NATS topic naming conventions |

## Quick Reference

### GoRAI CLI

```bash
gorai version            # Show version
gorai run <config>       # Run robot from config
gorai validate <config>  # Validate configuration
gorai list-components    # Show available components
gorai list-services      # Show available services
```

### Configuration Structure

```json
{
  "name": "my-robot",
  "nats": {
    "url": "nats://localhost:4222"
  },
  "components": [
    {
      "type": "sensor.temperature",
      "name": "cpu_temp",
      "config": { ... }
    }
  ],
  "services": [
    {
      "type": "vision.detection",
      "name": "object_detector",
      "config": { ... }
    }
  ]
}
```

### Topic Naming Convention

```
gorai.<namespace>.<resource-type>.<resource-name>.<message-type>

Examples:
gorai.robot1.sensor.imu.data
gorai.robot1.motor.left_wheel.cmd
gorai.robot1.service.vision.req
```

<!-- website-only -->
!!! tip "API Documentation"
    For Go API documentation, see [pkg.go.dev/github.com/gorai/gorai](https://pkg.go.dev/github.com/gorai/gorai).
<!-- /website-only -->
