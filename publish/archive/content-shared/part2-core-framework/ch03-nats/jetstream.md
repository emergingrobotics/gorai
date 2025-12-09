## 3.5 JetStream Features

JetStream is NATS's persistence layer. When you need messages to survive restarts, handle late subscribers, or guarantee delivery, JetStream provides the mechanisms.

### Streams and Consumers

**Streams** store messages:
```go
// GoRAI creates streams automatically when using JetStream QoS
// But you can create them manually for advanced control
js, _ := nc.JetStream()

_, err := js.AddStream(&nats.StreamConfig{
    Name:     "SENSOR_DATA",
    Subjects: []string{"gorai.sensors.>"},
    Storage:  nats.FileStorage,
    MaxMsgs:  1000000,
    MaxAge:   24 * time.Hour,
})
```

Stream configuration options:
- **Storage**: `FileStorage` (persistent) or `MemoryStorage` (faster, volatile)
- **MaxMsgs**: Maximum messages to retain
- **MaxAge**: Maximum message age before deletion
- **MaxBytes**: Maximum storage size
- **Replicas**: Number of copies for HA (in clusters)

**Consumers** track subscriber progress:
```go
// Push consumer: messages delivered as they arrive
sub, _ := js.Subscribe("gorai.sensors.>",
    handler,
    nats.Durable("sensor_processor"),
    nats.DeliverAll(),
)

// Pull consumer: subscriber requests messages
sub, _ := js.PullSubscribe("gorai.sensors.>",
    "batch_processor",
    nats.AckExplicit(),
)
msgs, _ := sub.Fetch(100) // Get up to 100 messages
```

### Durable Subscriptions

Durable consumers remember their position across restarts:

```go
sub, _ := sub.New[*sensor.Temperature](node, "gorai.sensors.temp.data",
    handler,
    sub.WithDurable("temp_logger"),  // Named consumer
)

// After restart, continues from where it left off
```

Without durability, restarting a subscriber loses track of which messages were processed. With durability:
1. First run: Processes messages 1-100
2. Restart
3. Second run: Continues from message 101

Critical for:
- Log processors that shouldn't miss messages
- Event handlers that need exactly-once semantics
- Offline-capable nodes that catch up after reconnection

### Replay Capabilities

JetStream enables powerful replay scenarios:

**Start from beginning**:
```go
sub.New(node, topic, handler, sub.WithDeliverAll())
```

**Start from last message**:
```go
sub.New(node, topic, handler, sub.WithDeliverLast())
```

**Start from new messages only** (default):
```go
sub.New(node, topic, handler, sub.WithDeliverNew())
```

**Replay by sequence number** (advanced):
```go
js.Subscribe(subject, handler,
    nats.StartSequence(12345))
```

**Replay by time**:
```go
js.Subscribe(subject, handler,
    nats.StartTime(time.Now().Add(-1*time.Hour)))
```

Use cases:
- Debugging: Replay sensor data through a fixed algorithm
- Testing: Run the same inputs through new code
- Recovery: Re-process events after a crash
- Analysis: Historical data review

*Cross-reference: Chapter 11 covers using replay for testing.*

### Acknowledgment and Redelivery

JetStream tracks message acknowledgment:

```go
js.Subscribe(subject, func(msg *nats.Msg) {
    err := process(msg)
    if err != nil {
        msg.Nak() // Negative acknowledgment: redeliver
        return
    }
    msg.Ack() // Success: don't redeliver
})
```

Acknowledgment options:
- **Ack()**: Message processed successfully
- **Nak()**: Processing failed, redeliver soon
- **Term()**: Don't redeliver (poison message)
- **InProgress()**: Still working, extend timeout

Configuration controls redelivery:
```go
sub.WithAckWait(30 * time.Second)  // Wait before redelivering
sub.WithMaxDeliver(5)              // Give up after 5 attempts
```

### Practical JetStream Usage

**Recording robot sessions**:
```bash
# Record all messages to a stream
nats stream add RECORDING \
    --subjects "gorai.>" \
    --storage file \
    --max-age 1h

# Later, replay for analysis
nats consumer add RECORDING analyzer \
    --deliver all \
    --replay instant
```

**Mission-critical commands**:
```go
// Ensure waypoints are never lost
waypointPub := pub.New[*nav.Waypoint](node, "gorai.mission.waypoints",
    pub.WithQoS(pub.Reliable))

// On subscriber side, acknowledge after persisting
sub.New[*nav.Waypoint](node, "gorai.mission.waypoints",
    func(wp *nav.Waypoint) {
        saveToDatabase(wp)
        // JetStream auto-acks after handler returns without error
    },
    sub.WithSubQoS(sub.Reliable))
```

**Fleet telemetry collection**:
```go
// Stream from all robots
js.AddStream(&nats.StreamConfig{
    Name:     "FLEET_TELEMETRY",
    Subjects: []string{"*.telemetry.>"},  // Any robot's telemetry
    Storage:  nats.FileStorage,
    MaxAge:   7 * 24 * time.Hour,  // Keep a week
})
```
