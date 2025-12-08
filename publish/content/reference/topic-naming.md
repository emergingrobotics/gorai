# Topic Naming Reference

NATS topic (subject) naming conventions for GoRAI.

## Hierarchical Structure

```
gorai.<domain>.<type>.<name>.<instance>
```

### Components

| Pattern | Example | Use |
|---------|---------|-----|
| `gorai.sensor.<type>.<name>` | `gorai.sensor.temperature.cpu` | Sensor data |
| `gorai.actuator.<type>.<name>` | `gorai.actuator.motor.left` | Actuator commands |
| `gorai.camera.<name>.image` | `gorai.camera.front.image` | Camera frames |

### Services

| Pattern | Example | Use |
|---------|---------|-----|
| `gorai.service.<name>.<action>` | `gorai.service.vision.detect` | Request/reply |
| `gorai.service.<name>.stream` | `gorai.service.vision.stream` | Continuous data |

### System

| Pattern | Example | Use |
|---------|---------|-----|
| `gorai.telemetry.<node>` | `gorai.telemetry.hello-sensor` | Node telemetry |
| `gorai.health.<node>` | `gorai.health.hello-sensor` | Health status |
| `gorai.config.<node>` | `gorai.config.hello-sensor` | Configuration updates |

## Wildcards

| Wildcard | Matches | Example |
|----------|---------|---------|
| `*` | Single token | `gorai.sensor.*.cpu` |
| `>` | Multiple tokens | `gorai.sensor.>` |

## Naming Rules

1. Use lowercase
2. Use hyphens for multi-word names
3. Be descriptive but concise
4. Include instance identifiers for multi-robot systems

## Reserved Prefixes

| Prefix | Reserved For |
|--------|--------------|
| `gorai.internal` | Framework internal use |
| `gorai.system` | System-level operations |
| `$JS` | JetStream operations |
