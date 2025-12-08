# NATS: The Communication Backbone

NATS is the messaging system at the heart of Gorai. This chapter explains how Gorai uses NATS for all robot communication—from high-frequency sensor data to command-and-control messages.

## Why NATS?

Gorai chose NATS over alternatives (DDS, ZeroMQ, ROS 2's middleware) for several reasons:

| Feature | NATS | DDS | ZeroMQ |
|---------|------|-----|--------|
| **Simplicity** | Single binary, zero config | Complex, many implementations | Requires careful design |
| **Performance** | Millions msg/sec | High, but variable | Very high |
| **Persistence** | JetStream built-in | Varies by impl | External |
| **Cloud Native** | First-class | Retrofitted | Limited |
| **Learning Curve** | Hours | Days to weeks | Days |

NATS provides:

- **Simplicity**: Single binary, works out of the box
- **Performance**: Millions of messages per second on commodity hardware
- **Reliability**: At-least-once delivery with JetStream
- **Flexibility**: Pub/sub, request/reply, and streaming in one system
- **Edge-Ready**: Leaf nodes for disconnected operation

## NATS Fundamentals

### Architecture

A NATS deployment for robotics typically looks like this:

```
┌─────────────────────────────────────────────────────────────┐
│                      NATS Server                             │
│                   (running on Primary Compute)               │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                     JetStream                          │  │
│  │  (Persistence, replay, exactly-once delivery)         │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
        │              │              │              │
        ▼              ▼              ▼              ▼
   ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐
   │ Camera  │   │  Motor  │   │ Vision  │   │ Behavior│
   │  Node   │   │  Node   │   │ Service │   │ Service │
   └─────────┘   └─────────┘   └─────────┘   └─────────┘
```

### Subjects (Topics)

NATS uses **subjects** for message routing. Gorai follows a hierarchical naming convention:

```
gorai.{robot}.{type}.{subtype}.{name}.{field}
```

Examples:
```
gorai.myrobot.component.motor.left_wheel.position
gorai.myrobot.component.sensor.imu.readings
gorai.myrobot.service.vision.detector.detections
gorai.myrobot.service.behavior.navigator.state
```

### Wildcards

NATS supports two wildcard characters:

| Wildcard | Matches | Example |
|----------|---------|---------|
| `*` | Single token | `gorai.*.component.motor.*` matches any robot's motors |
| `>` | One or more tokens | `gorai.myrobot.>` matches everything for myrobot |

Useful patterns:
```
gorai.*.component.sensor.>     # All sensors on all robots
gorai.myrobot.component.>      # All components on myrobot
gorai.myrobot.*.*.*.state      # All state messages
```

### Connecting to NATS

Gorai provides a simple connection wrapper:

```go
import "github.com/gorai/gorai/pkg/nats"

// Simple connection
nc, err := nats.Connect(nats.Options{
    URL: "nats://localhost:4222",
})

// With authentication
nc, err := nats.Connect(nats.Options{
    URL:      "nats://nats.example.com:4222",
    User:     "robot",
    Password: "secret",
})

// Using a token
nc, err := nats.Connect(nats.Options{
    URL:   "nats://nats.example.com:4222",
    Token: "my-auth-token",
})
```

## Communication Patterns

Gorai uses three NATS patterns for different needs:

### Pattern 1: Publish/Subscribe (Topics)

For sensor data, telemetry, and one-to-many communication:

```go
// Publisher (sensor node)
func (s *IMUSensor) publishReadings(nc *nats.Conn) {
    readings := &pb.IMUReading{
        Timestamp:          timestamppb.Now(),
        LinearAcceleration: &pb.Vector3{X: 0.1, Y: 0.0, Z: 9.8},
        AngularVelocity:    &pb.Vector3{X: 0.0, Y: 0.01, Z: 0.0},
    }

    data, _ := proto.Marshal(readings)
    nc.Publish("gorai.myrobot.component.sensor.imu.readings", data)
}

// Subscriber (any interested node)
func subscribeToIMU(nc *nats.Conn) {
    nc.Subscribe("gorai.myrobot.component.sensor.imu.readings", func(msg *nats.Msg) {
        var readings pb.IMUReading
        proto.Unmarshal(msg.Data, &readings)
        fmt.Printf("Acceleration: %.2f m/s²\n", readings.LinearAcceleration.Z)
    })
}
```

**Use pub/sub for:**
- High-frequency sensor data (IMU at 100Hz, cameras at 30fps)
- Telemetry and logging
- State broadcasts
- Events (button pressed, obstacle detected)

### Pattern 2: Request/Reply (Services)

For synchronous operations that need a response:

```go
// Service handler (motor controller)
nc.Subscribe("gorai.myrobot.component.motor.left_wheel.get_position", func(msg *nats.Msg) {
    position := motor.GetPosition()

    response := &pb.PositionResponse{Position: position}
    data, _ := proto.Marshal(response)

    msg.Respond(data)
})

// Client (requesting position)
func getMotorPosition(nc *nats.Conn) (float64, error) {
    msg, err := nc.Request(
        "gorai.myrobot.component.motor.left_wheel.get_position",
        nil,
        time.Second,
    )
    if err != nil {
        return 0, err
    }

    var response pb.PositionResponse
    proto.Unmarshal(msg.Data, &response)
    return response.Position, nil
}
```

**Use request/reply for:**
- Getting current state (motor position, battery level)
- Configuration queries
- Single-shot commands with confirmation
- Service calls (run inference, get detections)

### Pattern 3: Actions (Long-Running Tasks)

For operations that take time and need progress updates:

```go
// Action server (navigator)
type NavigateAction struct {
    nc       *nats.Conn
    goalSub  *nats.Subscription
    cancelSub *nats.Subscription
}

func (a *NavigateAction) handleGoal(msg *nats.Msg) {
    var goal pb.NavigateGoal
    proto.Unmarshal(msg.Data, &goal)

    goalID := uuid.New().String()

    // Accept the goal
    msg.Respond([]byte(goalID))

    // Execute with feedback
    go func() {
        for progress := 0.0; progress < 1.0; progress += 0.1 {
            feedback := &pb.NavigateFeedback{
                GoalID:   goalID,
                Progress: progress,
                Distance: (1.0 - progress) * goal.Distance,
            }
            data, _ := proto.Marshal(feedback)
            a.nc.Publish("gorai.myrobot.action.navigate.feedback", data)

            time.Sleep(time.Second)
        }

        // Send result
        result := &pb.NavigateResult{GoalID: goalID, Success: true}
        data, _ := proto.Marshal(result)
        a.nc.Publish("gorai.myrobot.action.navigate.result", data)
    }()
}

// Action client
func navigateTo(nc *nats.Conn, x, y float64) error {
    goal := &pb.NavigateGoal{X: x, Y: y}
    data, _ := proto.Marshal(goal)

    // Send goal, get ID
    msg, err := nc.Request("gorai.myrobot.action.navigate.goal", data, time.Second)
    if err != nil {
        return err
    }
    goalID := string(msg.Data)

    // Subscribe to feedback
    feedbackCh := make(chan *pb.NavigateFeedback)
    nc.Subscribe("gorai.myrobot.action.navigate.feedback", func(msg *nats.Msg) {
        var fb pb.NavigateFeedback
        proto.Unmarshal(msg.Data, &fb)
        if fb.GoalID == goalID {
            feedbackCh <- &fb
        }
    })

    // Wait for result
    for fb := range feedbackCh {
        fmt.Printf("Progress: %.0f%%\n", fb.Progress*100)
        if fb.Progress >= 1.0 {
            break
        }
    }

    return nil
}
```

**Use actions for:**
- Navigation (move to waypoint)
- Manipulation (pick and place)
- Any operation taking more than a few seconds
- Operations needing progress feedback
- Cancelable tasks

## JetStream for Persistence

JetStream adds persistence, replay, and exactly-once delivery to NATS:

### Creating Streams

```go
js, _ := nc.JetStream()

// Create a stream for sensor data
_, err := js.AddStream(&nats.StreamConfig{
    Name:     "SENSORS",
    Subjects: []string{"gorai.myrobot.component.sensor.>"},
    Storage:  nats.FileStorage,
    MaxAge:   24 * time.Hour,  // Keep data for 24 hours
    MaxBytes: 1 << 30,          // 1GB max
})
```

### Publishing with Acknowledgment

```go
// Publish with delivery guarantee
ack, err := js.Publish("gorai.myrobot.component.sensor.imu.readings", data)
if err != nil {
    log.Printf("Publish failed: %v", err)
} else {
    log.Printf("Published to stream %s, seq %d", ack.Stream, ack.Sequence)
}
```

### Durable Consumers

Consumers track position in the stream, surviving restarts:

```go
// Create durable consumer
_, err := js.AddConsumer("SENSORS", &nats.ConsumerConfig{
    Durable:       "logger",
    DeliverPolicy: nats.DeliverAllPolicy,
    AckPolicy:     nats.AckExplicitPolicy,
})

// Subscribe with the durable consumer
sub, _ := js.PullSubscribe(
    "gorai.myrobot.component.sensor.imu.readings",
    "logger",
)

// Fetch and process messages
for {
    msgs, _ := sub.Fetch(10, nats.MaxWait(time.Second))
    for _, msg := range msgs {
        processMessage(msg)
        msg.Ack()
    }
}
```

### Replay and Time Travel

JetStream enables replaying historical data:

```go
// Subscribe starting from 1 hour ago
_, err := js.AddConsumer("SENSORS", &nats.ConsumerConfig{
    Durable:         "replay",
    DeliverPolicy:   nats.DeliverByStartTimePolicy,
    OptStartTime:    &time.Time{}.Add(-time.Hour),
})
```

This is powerful for:
- Debugging robot behavior
- Training ML models on historical data
- Replaying scenarios for testing

## Quality of Service

Gorai provides QoS levels for publishers and subscribers:

### Publisher QoS

```go
import "github.com/gorai/gorai/pkg/pub"

// Best-effort: fire and forget
pub.Publish(nc, "sensor.readings", data, pub.BestEffort)

// Reliable: JetStream with acknowledgment
pub.Publish(nc, "motor.commands", data, pub.Reliable)

// Retained: keeps last value for new subscribers
pub.Publish(nc, "robot.status", data, pub.Retained)

// History: keeps last N messages
pub.Publish(nc, "events.log", data, pub.History(100))
```

### Subscriber QoS

```go
import "github.com/gorai/gorai/pkg/sub"

// Best-effort: may miss messages
sub.Subscribe(nc, "sensor.readings", handler, sub.BestEffort)

// Reliable: acknowledges messages, redelivers on failure
sub.Subscribe(nc, "motor.commands", handler, sub.Reliable)

// Durable: survives restarts, never misses messages
sub.Subscribe(nc, "events.important", handler, sub.Durable("myservice"))
```

### Choosing QoS

| Data Type | Publisher QoS | Subscriber QoS |
|-----------|--------------|----------------|
| IMU data (100Hz) | BestEffort | BestEffort |
| Camera frames | BestEffort | BestEffort |
| Motor commands | Reliable | Reliable |
| Emergency stop | Reliable | Reliable |
| Configuration | Retained | Durable |
| Event logs | History(1000) | Durable |

## The Node Abstraction

Gorai's `Node` wraps NATS connections with robot-specific conveniences:

```go
import "github.com/gorai/gorai/pkg/node"

// Create a node
n, err := node.New("motor_controller",
    node.WithNATS("nats://localhost:4222"),
    node.WithNamespace("gorai"),
    node.WithLogger(slog.Default()),
)

// Access NATS connection
nc := n.NATS()

// Access JetStream
js := n.JetStream()

// Run the node (blocks until shutdown)
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go n.Spin(ctx)

// ... node logic ...

// Graceful shutdown
n.Shutdown()
```

### Node Features

| Method | Purpose |
|--------|---------|
| `Name()` | Returns node name |
| `Namespace()` | Returns namespace |
| `FullName()` | Returns `namespace.name` |
| `NATS()` | Raw NATS connection |
| `JetStream()` | JetStream context |
| `HasJetStream()` | Checks if JetStream is available |
| `IsConnected()` | Connection status |
| `Spin(ctx)` | Run until context canceled |
| `SpinOnce(ctx, timeout)` | Process pending messages once |
| `Shutdown()` | Graceful shutdown |

## CLI Tools

### nats CLI

The `nats` CLI is invaluable for debugging:

```bash
# Subscribe to all sensor data
nats sub "gorai.*.component.sensor.>"

# Publish a test message
nats pub gorai.myrobot.component.motor.left_wheel.command '{"power": 0.5}'

# Request/reply
nats request gorai.myrobot.component.motor.left_wheel.get_position ""

# View stream info
nats stream info SENSORS

# View consumer info
nats consumer info SENSORS logger

# Replay messages from stream
nats consumer next SENSORS logger --count 10
```

### Monitoring

NATS server exposes metrics:

```bash
# Server stats
nats server info

# Connection stats
nats server report connections

# JetStream stats
nats server report jetstream
```

### Installing NATS

```bash
# macOS
brew install nats-server nats-io/nats-tools/nats

# Linux (download from GitHub releases)
curl -L https://github.com/nats-io/nats-server/releases/download/v2.10.0/nats-server-v2.10.0-linux-arm64.tar.gz | tar xz
sudo mv nats-server-v2.10.0-linux-arm64/nats-server /usr/local/bin/

# Run with JetStream
nats-server -js
```

## Topic Naming Conventions

Gorai uses consistent topic naming:

### Component Topics

```
gorai.{robot}.component.{subtype}.{name}.{field}
```

| Subject Pattern | Purpose |
|-----------------|---------|
| `...motor.left_wheel.position` | Motor position readings |
| `...motor.left_wheel.velocity` | Motor velocity readings |
| `...motor.left_wheel.command` | Motor commands |
| `...sensor.imu.readings` | IMU sensor data |
| `...camera.front.image` | Camera image frames |
| `...camera.front.info` | Camera metadata |

### Service Topics

```
gorai.{robot}.service.{subtype}.{name}.{method}
```

| Subject Pattern | Purpose |
|-----------------|---------|
| `...vision.detector.detect` | Request detection |
| `...vision.detector.detections` | Detection results stream |
| `...behavior.navigator.goal` | Set behavior goal |
| `...behavior.navigator.state` | Behavior state updates |
| `...behavior.navigator.derived.pose` | Derived sensor data |

### Action Topics

```
gorai.{robot}.action.{name}.{type}
```

| Subject Pattern | Purpose |
|-----------------|---------|
| `...action.navigate.goal` | Send goal (request/reply) |
| `...action.navigate.cancel` | Cancel action |
| `...action.navigate.feedback` | Progress updates (pub/sub) |
| `...action.navigate.result` | Final result |

## Best Practices

### Message Size

Keep messages small for high-frequency data:

```go
// Good: Separate topics for different data
nc.Publish("gorai.myrobot.component.sensor.imu.acceleration", accelData)
nc.Publish("gorai.myrobot.component.sensor.imu.gyro", gyroData)

// Avoid: Large combined messages at high frequency
nc.Publish("gorai.myrobot.component.sensor.imu.all", hugeMessage)
```

### Serialization

Always use Protocol Buffers for efficiency:

```go
// Good: Protocol Buffers
data, _ := proto.Marshal(reading)
nc.Publish(topic, data)

// Avoid: JSON for high-frequency data
data, _ := json.Marshal(reading)  // Slower, larger
nc.Publish(topic, data)
```

### Error Handling

```go
// Publish with error handling
if err := nc.Publish(topic, data); err != nil {
    if errors.Is(err, nats.ErrConnectionClosed) {
        // Handle reconnection
    }
    log.Printf("Publish error: %v", err)
}

// Request with timeout
msg, err := nc.Request(topic, data, 500*time.Millisecond)
if err == nats.ErrTimeout {
    // Handle timeout (service unavailable?)
}
```

### Graceful Shutdown

```go
// Drain connection before closing
nc.Drain()

// Or with context
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
nc.DrainWithContext(ctx)
```
