# NATS Messaging

NATS is the communication backbone of GoRAI.

## Why NATS?

- Simple and lightweight
- High performance
- Built-in clustering
- JetStream for persistence

## Topics (Pub/Sub)

```go
// Publisher
pub := pub.New[sensor.Temperature](n, "sensor.temperature")
pub.Publish(ctx, &sensor.Temperature{Value: 25.5})

// Subscriber
sub.New[sensor.Temperature](n, "sensor.temperature", func(msg *sensor.Temperature) {
    log.Printf("Temperature: %.1f", msg.Value)
})
```

*More content coming soon*
