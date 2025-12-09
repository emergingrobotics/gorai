---
title: "Configuration"
description: "Configure Gorai components and services"
weight: 40
---

# Configuration Guide

Gorai supports configuration through YAML files and environment variables.

## YAML Configuration

```yaml
# config.yaml
robot:
  name: my-robot
  nats:
    url: nats://localhost:4222

components:
  - name: temp-sensor
    type: sensor
    driver: ds18b20
    pin: 4

  - name: left-motor
    type: actuator
    driver: pwm
    pins: [17, 18]
```

## Environment Variables

Override configuration with environment variables:

```bash
export GORAI_NATS_URL=nats://192.168.1.100:4222
export GORAI_LOG_LEVEL=debug
```

## Loading Configuration

```go
cfg, err := config.Load("config.yaml")
if err != nil {
    log.Fatal(err)
}

robot := gorai.New(cfg)
robot.Start()
```

## Next Steps

- [Testing Guide](../testing/)
- [Reference: Configuration](/docs/reference/configuration/)
