---
title: "NATS Messaging"
description: "Learn NATS messaging patterns for Gorai"
weight: 30
---

# NATS Messaging Guide

Gorai uses NATS for all component communication. This guide covers the patterns you'll use most often.

## Quick Reference

| Pattern | Use Case | Example |
|---------|----------|---------|
| Pub/Sub | Sensor data | Temperature readings |
| Request/Reply | Commands | Motor control |
| Queue Groups | Load balancing | Multiple processors |
| JetStream | Persistence | Message replay |

## Pub/Sub

The most common pattern for sensor data.

### Publishing

```go
nc.Publish("sensors.temp.reading", []byte("25.5"))
```

### Subscribing

```go
nc.Subscribe("sensors.>", func(msg *nats.Msg) {
    fmt.Printf("Received: %s on %s\n", msg.Data, msg.Subject)
})
```

### Wildcards

- `*` — Match single token: `sensors.*.reading`
- `>` — Match multiple tokens: `sensors.>`

## Request/Reply

For commands that need acknowledgment.

### Requester

```go
msg, err := nc.Request("actuators.motor.command",
    []byte("speed:50"),
    time.Second)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Response: %s\n", msg.Data)
```

### Responder

```go
nc.Subscribe("actuators.motor.command", func(msg *nats.Msg) {
    // Process command
    speed := parseSpeed(msg.Data)
    motor.SetSpeed(speed)

    // Send response
    msg.Respond([]byte("ok"))
})
```

## Queue Groups

Distribute load across multiple consumers.

```go
// Both subscribers share the work
nc.QueueSubscribe("vision.detect", "processors", handler)
nc.QueueSubscribe("vision.detect", "processors", handler)
```

## JetStream

Persistent messaging with replay capabilities.

```go
js, _ := nc.JetStream()

// Create stream
js.AddStream(&nats.StreamConfig{
    Name:     "SENSORS",
    Subjects: []string{"sensors.>"},
})

// Publish
js.Publish("sensors.temp.reading", []byte("25.5"))

// Consume with replay
js.Subscribe("sensors.>", handler, nats.DeliverAll())
```

## Next Steps

- [Configuration Guide](../configuration/)
- [Testing Guide](../testing/)
