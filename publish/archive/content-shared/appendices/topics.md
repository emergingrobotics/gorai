# NATS Topics Reference

This appendix covers NATS topic naming conventions and patterns used in GoRAI.

## Topic Hierarchy

GoRAI uses a hierarchical topic structure:

```
gorai.<domain>.<component>.<type>.<instance>
```

### Standard Prefixes

| Prefix | Purpose |
|--------|---------|
| `gorai.sensor` | Sensor data publications |
| `gorai.actuator` | Actuator commands |
| `gorai.service` | Service requests/responses |
| `gorai.telemetry` | System telemetry |
| `gorai.config` | Configuration updates |
| `gorai.health` | Health status |

### Examples

| Topic | Description |
|-------|-------------|
| `gorai.sensor.temperature.cpu.main` | CPU temperature from main sensor |
| `gorai.sensor.camera.front.image` | Image from front camera |
| `gorai.actuator.motor.drive.left` | Left drive motor commands |
| `gorai.service.vision.detect` | Object detection requests |
| `gorai.telemetry.node.hello-sensor` | Node telemetry |

## Wildcards

NATS supports two wildcards:

- `*` - Matches a single token
- `>` - Matches one or more tokens

### Subscription Patterns

| Pattern | Matches |
|---------|---------|
| `gorai.sensor.>` | All sensor messages |
| `gorai.*.camera.>` | Camera messages from any domain |
| `gorai.sensor.*.cpu.*` | All CPU sensor messages |

## Command Reference

### NATS CLI Commands

| Command | Purpose |
|---------|---------|
| `nats sub ">"` | Subscribe to all messages |
| `nats sub "gorai.>"` | Subscribe to GoRAI messages |
| `nats pub TOPIC DATA` | Publish message |
| `nats request TOPIC DATA` | Request/reply |
| `nats server info` | Server information |
| `nats stream list` | List JetStream streams |
| `nats stream view NAME` | View stream messages |
| `nats consumer list STREAM` | List consumers |

## Best Practices

1. **Be specific**: Use descriptive topic names
2. **Use hierarchy**: Group related topics under common prefixes
3. **Include instance**: Add instance identifiers for multi-robot systems
4. **Document schemas**: Keep message schema documentation current
