## 3.4 Quality of Service (QoS)

Not all messages have the same requirements. A control command must arrive immediately but can be lost if the subscriber isn't ready. A configuration update must be delivered reliably. GoRAI provides QoS levels for these different needs.

### BestEffort: Core NATS

The default QoS—simple, fast, no persistence:

```go
pub := pub.New[*sensor.IMU](node, "sensors.imu.data")
// Uses core NATS, no JetStream
```

**Characteristics**:
- **Lowest latency**: Direct publish to subscribers
- **No storage**: If no subscriber is listening, message is lost
- **No acknowledgment**: Publisher doesn't know if anyone received it
- **Minimal overhead**: Just network I/O

**Use for**:
- High-frequency sensor data (IMU at 1kHz)
- Real-time control commands
- Any data where the next message supersedes the previous

**Example**: IMU data stream
```go
pub := pub.New[*sensor.Imu](node, "gorai.sensors.imu.data")

for reading := range imu.Readings() {
    pub.Publish(ctx, reading)
    // If subscribers miss one, the next arrives in 1ms anyway
}
```

### Reliable: JetStream Acknowledgment

Messages are persisted and delivery is guaranteed:

```go
pub := pub.New[*config.Update](node, "gorai.config.updates",
    pub.WithQoS(pub.Reliable))
```

**Characteristics**:
- **Persistence**: Messages stored in JetStream stream
- **Acknowledgment**: Publisher knows message was stored
- **Redelivery**: Failed deliveries are retried
- **Higher overhead**: Storage I/O, acknowledgment round-trip

**Use for**:
- Configuration updates
- Mission waypoints
- Critical commands that must not be lost
- Logging and telemetry that must be preserved

**Example**: Configuration distribution
```go
pub := pub.New[*config.RobotConfig](node, "gorai.config.robot",
    pub.WithQoS(pub.Reliable))

// When config changes, publish reliably
pub.Publish(ctx, newConfig)
// Returns only after message is persisted
```

### Retained: Last-Value Retention

Only the most recent message per subject is kept:

```go
pub := pub.New[*sensor.BatteryState](node, "gorai.power.battery",
    pub.WithRetain())
```

**Characteristics**:
- **Last value available**: New subscribers immediately get current state
- **Automatic cleanup**: Old values are replaced
- **JetStream storage**: Persisted, but only one message per subject

**Use for**:
- Current status/state
- Configuration that should be available to new subscribers
- "What is X right now?" queries

**Example**: Battery status
```go
pub := pub.New[*sensor.BatteryState](node, "gorai.power.battery",
    pub.WithRetain())

// Publish periodically
ticker := time.NewTicker(time.Second)
for range ticker.C {
    pub.Publish(ctx, readBatteryState())
}

// New subscribers immediately get the latest state
sub.New[*sensor.BatteryState](node, "gorai.power.battery", handler,
    sub.WithDeliverLast())
```

### History: Message Buffering

Keep the last N messages for late-joining subscribers:

```go
pub := pub.New[*sensor.Odometry](node, "gorai.odom.data",
    pub.WithHistory(100))  // Keep last 100 messages
```

**Characteristics**:
- **Catch-up**: New subscribers can replay recent history
- **Bounded storage**: Only N messages per subject
- **Ordered delivery**: Messages arrive in sequence

**Use for**:
- Odometry data (for pose estimation catch-up)
- Event logs
- Any stream where context from recent past matters

**Example**: Odometry with history
```go
// Publisher keeps history
pub := pub.New[*nav.Odometry](node, "gorai.odom.data",
    pub.WithHistory(100))

// Late subscriber catches up
sub.New[*nav.Odometry](node, "gorai.odom.data", handler,
    sub.WithDeliverAll())  // Get all available history first
```

### Choosing the Right QoS

| Scenario | QoS | Rationale |
|----------|-----|-----------|
| IMU at 1kHz | BestEffort | Speed matters, missing one is fine |
| Motor commands | BestEffort | Latest command supersedes previous |
| Configuration updates | Reliable | Must not be lost |
| Current battery level | Retained | New nodes need current value |
| Odometry stream | History | Localization needs recent context |
| Logged events | Reliable | Must be preserved |
| Camera images | BestEffort | Too large for persistence at frame rate |

Default to **BestEffort**. Only use JetStream QoS when you need its guarantees—the overhead is real.
