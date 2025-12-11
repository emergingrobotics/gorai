# NATS Messaging

NATS is to Gorai what DDS is to ROS 2—the messaging layer that connects everything. But NATS comes from the cloud-native world, bringing lessons learned from operating systems at massive scale. It's simple, fast, and proven.

This chapter gives you deep understanding of NATS and how Gorai uses it. We start with why NATS was chosen, move through fundamentals (publish/subscribe, request/reply), cover Gorai-specific patterns, explore quality of service options, and finish with JetStream for persistence. This is the most important chapter in Part II—every subsequent chapter builds on these concepts.

## Why NATS?

Gorai could have built on many messaging systems: ROS 2's DDS, ZeroMQ, gRPC, MQTT, or custom protocols. NATS won for compelling reasons.

### Cloud-Native Messaging for Robotics

NATS was built for cloud infrastructure—systems with thousands of services, unreliable networks, and demanding performance requirements. These constraints mirror robotics:

- **Many producers and consumers**: Sensors publish, multiple nodes subscribe
- **Unreliable connections**: WiFi drops, nodes restart, processes crash
- **Low latency requirements**: Control loops can't wait
- **Simple operations**: No time for complex configuration

NATS brings cloud-hardened solutions to robotics problems.

### Performance Characteristics

NATS is fast. Benchmarks show:

- **18+ million messages/second** on modest hardware (single server)
- **Sub-millisecond latency** for typical messages
- **Minimal CPU overhead**: More cycles for your robot logic

For comparison, ROS 2 with DDS can struggle to saturate a gigabit link. NATS handles it trivially.

Memory usage is also lean. The NATS server runs in tens of megabytes. Clients add negligible overhead. This matters when your robot's brain is a Raspberry Pi, not a data center.

### Comparison with Alternatives

**vs ROS 2 DDS**:

- DDS is enterprise middleware designed for defense and aerospace
- Multiple implementations (CycloneDDS, FastDDS, Connext) with different behaviors
- Complex QoS configuration with dozens of parameters
- NATS: One implementation, simple config, predictable behavior

**vs ZeroMQ**:

- ZeroMQ is a library, not a broker—each node manages its own connections
- Discovery requires custom solutions
- NATS: Broker simplifies topology, built-in discovery via subjects

**vs gRPC**:

- gRPC is point-to-point, not pub/sub
- Requires knowing endpoints ahead of time
- NATS: Loose coupling, dynamic discovery, pub/sub native

**vs MQTT**:

- MQTT is designed for IoT telemetry—small messages, constrained devices
- Limited pub/sub patterns, no request/reply
- NATS: Full messaging patterns, higher performance, JetStream for persistence

### JetStream for Persistence

Core NATS is fire-and-forget: if no subscriber is listening, messages disappear. JetStream adds persistence:

- **Streams**: Store messages durably
- **Consumers**: Track what each subscriber has seen
- **Replay**: New subscribers can catch up on history
- **Acknowledgment**: Ensure messages are processed

Gorai uses core NATS for real-time data (sensor streams, control commands) and JetStream when durability matters (configuration updates, logged data, mission waypoints).

```go
// Core NATS: fast, no persistence
pub := pub.New[*sensor.IMU](node, "sensors.imu.data")

// JetStream: reliable, persisted
pub := pub.New[*config.Update](node, "config.updates",
    pub.WithQoS(pub.Reliable))
```

### Operational Simplicity

NATS runs as a single binary with zero dependencies:

```bash
# That's it. NATS is running.
nats-server -js

# Or with a config file
nats-server -c /etc/nats/nats.conf
```

No ZooKeeper, no etcd, no Kubernetes. NATS can run on a Raspberry Pi as easily as in a cloud cluster.

Clustering is straightforward when you need it:

```bash
# Three-server cluster for high availability
nats-server -c server1.conf
nats-server -c server2.conf
nats-server -c server3.conf
```

For most robots, a single local NATS server is sufficient. The option to scale exists when needed.

## NATS Fundamentals

Before diving into Gorai's patterns, let's understand NATS primitives.

### Publish/Subscribe Basics

NATS pub/sub is simple: publishers send to subjects, subscribers listen on subjects:

```
Publisher                 NATS                    Subscribers
    │                       │                          │
    │ Publish("foo", data)  │                          │
    │──────────────────────>│                          │
    │                       │────────────────────────>│ Sub("foo")
    │                       │────────────────────────>│ Sub("foo")
    │                       │                          │
```

Subjects are strings with dot-separated hierarchies:

- `sensors.imu.data`
- `motors.left.command`
- `vision.camera.front.image`

This isn't just convention—it enables wildcard subscriptions.

### Request/Reply Pattern

NATS supports RPC-style synchronous calls:

```go
// Requester
response, err := nc.Request("services.detector", request, timeout)

// Responder
nc.Subscribe("services.detector", func(msg *nats.Msg) {
    result := process(msg.Data)
    msg.Respond(result)
})
```

Under the hood, NATS creates a temporary inbox subject for the reply. This pattern is perfect for:

- Getting current sensor values
- Querying component status
- Invoking service methods

### Wildcards and Subject Hierarchies

NATS wildcards make subscriptions powerful:

**Single-level wildcard (`*`)**: Matches exactly one token

```go
// Matches: sensors.imu.data, sensors.gps.data
// Not: sensors.imu.calibration.data
nc.Subscribe("sensors.*.data", handler)
```

**Multi-level wildcard (`>`)**: Matches one or more tokens

```go
// Matches: sensors.anything, sensors.a.b.c.d
nc.Subscribe("sensors.>", handler)
```

Practical uses:

```go
// All motor commands for any motor
nc.Subscribe("motors.*.command", handler)

// Everything from robot1
nc.Subscribe("robot1.>", handler)

// All camera images from any namespace
nc.Subscribe("*.cameras.*.image", handler)
```

### Connection Management

NATS clients handle connection lifecycle automatically:

```go
nc, err := nats.Connect("nats://localhost:4222",
    nats.Name("my_node"),           // Identify in server logs
    nats.ReconnectWait(time.Second), // Retry interval
    nats.MaxReconnects(-1),          // Retry forever
)
```

The client automatically:

- Reconnects on disconnect
- Re-subscribes after reconnection
- Buffers messages during brief outages

Gorai's `node.New()` configures these sensibly by default:

```go
n, err := node.New("my_node", node.WithNATS("nats://localhost:4222"))
// Reconnection, buffering, etc. are configured automatically
```

### Observing Messages

The `nats` CLI is invaluable for debugging:

```bash
# Subscribe to everything
nats sub ">"

# Subscribe to sensor data
nats sub "sensors.>"

# Publish a test message
nats pub "test.topic" "hello world"

# Request/reply
nats request "services.echo" "ping"
```

During development, keep a terminal running `nats sub ">"` to watch all traffic. It's like `tcpdump` for your robot's nervous system.

### Subject Naming Conventions

Gorai follows consistent naming:

```
{namespace}.{type}.{name}.{suffix}

Examples:
gorai.sensors.imu.data
gorai.motors.left.command
gorai.services.detector.request
robot1.cameras.front.image
```

Where:

- **namespace**: Organization or robot identifier
- **type**: Category (sensors, motors, cameras, services)
- **name**: Specific instance
- **suffix**: Data type or operation (data, command, request, response)

This structure enables useful wildcard patterns:

```go
// All sensors from this robot
nc.Subscribe("gorai.sensors.>", handler)

// All motor commands
nc.Subscribe("gorai.motors.*.command", handler)

// Everything from robot1
nc.Subscribe("robot1.>", handler)
```

## Gorai's NATS Patterns

Gorai builds three communication patterns on NATS: Topics (pub/sub), Services (request/reply), and Actions (long-running with feedback).

### Topics (Pub/Sub)

Topics are the primary pattern for streaming data. Sensors publish continuously; interested nodes subscribe.

**Publishing sensor data**:

```go
// Create a typed publisher
pub := pub.New[*sensor.Temperature](node, "gorai.sensors.temp.data")

// In your reading loop
for reading := range temperatureReadings() {
    msg := &sensor.Temperature{
        Header:      makeHeader(),
        Temperature: reading.Celsius,
        Variance:    reading.Variance,
    }
    pub.Publish(ctx, msg)
}
```

**Subscribing to sensor data**:

```go
sub.New[*sensor.Temperature](node, "gorai.sensors.temp.data",
    func(msg *sensor.Temperature) {
        log.Printf("Temperature: %.1f°C", msg.Temperature)
    })
```

**Telemetry publishing** follows the same pattern:

```go
// Battery monitor
battPub := pub.New[*sensor.BatteryState](node, "gorai.power.battery.state")

ticker := time.NewTicker(time.Second)
for range ticker.C {
    state := readBatteryState()
    battPub.Publish(ctx, state)
}
```

**Topic naming conventions**:

```
gorai.{node}.{component}.{datatype}

Examples:
gorai.hello.cpu_temp.data
gorai.sensors.imu.data
gorai.motors.left.feedback
gorai.cameras.front.image
```

### Services (Request/Reply)

Services handle synchronous operations: "give me the current value" or "execute this command and tell me if it worked."

**Implementing a service**:

```go
// Register a handler for motor commands
nc.Subscribe("gorai.motors.left.set_power", func(msg *nats.Msg) {
    var req MotorPowerRequest
    proto.Unmarshal(msg.Data, &req)

    err := motor.SetPower(ctx, req.Power)

    resp := &MotorPowerResponse{Success: err == nil}
    if err != nil {
        resp.Error = err.Error()
    }

    data, _ := proto.Marshal(resp)
    msg.Respond(data)
})
```

**Calling a service**:

```go
req := &MotorPowerRequest{Power: 0.5}
data, _ := proto.Marshal(req)

respMsg, err := nc.Request("gorai.motors.left.set_power", data, time.Second)
if err != nil {
    return fmt.Errorf("request failed: %w", err)
}

var resp MotorPowerResponse
proto.Unmarshal(respMsg.Data, &resp)
if !resp.Success {
    return fmt.Errorf("motor error: %s", resp.Error)
}
```

**Timeout handling** is critical for robotics:

```go
// Short timeout for control commands
resp, err := nc.Request(subject, data, 100*time.Millisecond)
if err == nats.ErrTimeout {
    // Handle timeout—maybe stop motors for safety
    emergencyStop()
}
```

### Actions (Long-Running Operations)

Actions handle operations that take time and provide progress updates: navigation to a goal, arm movements, scanning routines.

The pattern involves three message types:

- **Goal**: What to do
- **Feedback**: Progress updates during execution
- **Result**: Final outcome

```
Client                           Server
  │                                │
  │ Goal: navigate to (10, 5)      │
  │───────────────────────────────>│
  │                                │ Start navigating
  │   Feedback: 20% complete       │
  │<───────────────────────────────│
  │   Feedback: 50% complete       │
  │<───────────────────────────────│
  │   Feedback: 80% complete       │
  │<───────────────────────────────│
  │                                │ Arrived
  │   Result: success              │
  │<───────────────────────────────│
  │                                │
```

**Server implementation**:

```go
server, _ := action.NewServer[*NavGoal, *NavFeedback, *NavResult](
    node, "navigation.go_to",
    func(ctx context.Context, handle *action.GoalHandle[*NavGoal, *NavFeedback, *NavResult]) {
        goal := handle.Goal()

        for !atGoal(goal.Position) {
            if handle.IsCanceling() {
                handle.SetCanceled(&NavResult{Success: false})
                return
            }

            // Move toward goal
            step := computeStep(goal.Position)
            executeStep(step)

            // Send feedback
            handle.SendFeedback(&NavFeedback{
                DistanceRemaining: distanceTo(goal.Position),
                Progress:          computeProgress(),
            })

            time.Sleep(100 * time.Millisecond)
        }

        handle.SetSucceeded(&NavResult{
            Success:       true,
            FinalPosition: currentPosition(),
        })
    },
)
```

**Client usage**:

```go
client, _ := action.NewClient[*NavGoal, *NavFeedback, *NavResult](
    node, "navigation.go_to")

goal := &NavGoal{Position: &geometry.Point{X: 10, Y: 5}}
handle, _ := client.SendGoal(ctx, goal)

// Monitor feedback
for fb := range handle.Feedback() {
    log.Printf("Progress: %.1f%%, Distance: %.2fm",
        fb.Progress*100, fb.DistanceRemaining)
}

// Get result
result, err := handle.Wait(ctx)
if result.Success {
    log.Printf("Arrived at %v", result.FinalPosition)
}
```

**Cancellation support**:

```go
// Client can cancel
handle.Cancel()

// Server checks for cancellation
if handle.IsCanceling() {
    cleanup()
    handle.SetCanceled(&NavResult{})
    return
}
```

Use actions for:

- Navigation to waypoints
- Arm trajectory execution
- Scanning/searching behaviors
- Any operation lasting more than a few seconds

## Quality of Service (QoS)

Not all messages have the same requirements. A control command must arrive immediately but can be lost if the subscriber isn't ready. A configuration update must be delivered reliably. Gorai provides QoS levels for these different needs.

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

## JetStream Features

JetStream is NATS's persistence layer. When you need messages to survive restarts, handle late subscribers, or guarantee delivery, JetStream provides the mechanisms.

### Streams and Consumers

**Streams** store messages:

```go
// Gorai creates streams automatically when using JetStream QoS
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

## The NATS CLI

The `nats` command-line tool is indispensable for Gorai development. It lets you observe, debug, and interact with the message bus directly.

### Installation

```bash
# macOS
brew install nats-io/nats-tools/nats

# Linux (via go install)
go install github.com/nats-io/natscli/nats@latest

# Or download from releases
# https://github.com/nats-io/natscli/releases
```

Verify installation:

```bash
nats --version
```

### Basic Commands

**Subscribe to messages**:

```bash
# All messages
nats sub ">"

# All sensor messages
nats sub "gorai.sensors.>"

# Specific topic
nats sub "gorai.sensors.temp.data"

# With timestamps
nats sub "gorai.>" --raw
```

**Publish messages**:

```bash
# Simple text
nats pub "test.topic" "hello world"

# JSON
nats pub "gorai.test" '{"value": 42}'

# From file
nats pub "gorai.config" --file config.json
```

**Request/Reply**:

```bash
# Send request, wait for reply
nats request "gorai.services.echo" "ping" --timeout 5s
```

### Monitoring with `nats server`

Check server status:

```bash
# Server info
nats server info

# Connection list
nats server connections

# Request counts
nats server report connections
```

Real-time monitoring:

```bash
# Watch message rates
nats server report accounts --top

# Stream activity
nats stream report
```

### JetStream Commands

**Stream management**:

```bash
# List streams
nats stream list

# Stream info
nats stream info SENSOR_DATA

# View messages in stream
nats stream view SENSOR_DATA

# Purge (delete all messages)
nats stream purge SENSOR_DATA
```

**Consumer management**:

```bash
# List consumers
nats consumer list SENSOR_DATA

# Consumer info (shows lag, pending, etc.)
nats consumer info SENSOR_DATA my_consumer

# Get next message
nats consumer next SENSOR_DATA my_consumer
```

### Debugging Robot Communication

**Watch all traffic during development**:

```bash
# Terminal 1: Watch everything
nats sub ">" --raw

# Terminal 2: Run your robot
go run ./examples/hello-sensor
```

**Filter for specific patterns**:

```bash
# Only motor commands
nats sub "gorai.motors.*.command"

# Only errors/warnings (if you publish them)
nats sub "gorai.*.error" "gorai.*.warn"
```

**Interactive testing**:

```bash
# Test a motor service manually
nats request "gorai.motors.left.set_power" \
    '{"power": 0.5}' \
    --timeout 1s

# Simulate sensor data
nats pub "gorai.sensors.fake.data" \
    '{"temperature": 42.5}'
```

**Measure latency**:

```bash
# Round-trip time to server
nats rtt

# Latency distribution
nats bench "test.latency" --pub 1000 --sub 1 --size 256
```

### Useful One-Liners

```bash
# Count messages per second on a topic
nats sub "gorai.sensors.imu.data" --count 1000 2>&1 | tail -1

# Dump last N messages from a stream
nats stream view SENSOR_DATA --last 10

# Create a quick test stream
nats stream add TEST --subjects "test.>" --storage memory

# Follow logs with pretty JSON
nats sub "gorai.logs.>" | jq .

# Export stream contents
nats stream view SENSOR_DATA --json > data.jsonl
```

---

With a solid understanding of NATS—its patterns, QoS levels, JetStream features, and debugging tools—you're ready to explore how Gorai uses these capabilities for specific component types. Chapter 5 begins with sensors.
