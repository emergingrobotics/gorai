## 3.2 NATS Fundamentals

Before diving into GoRAI's patterns, let's understand NATS primitives.

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

GoRAI's `node.New()` configures these sensibly by default:

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

GoRAI follows consistent naming:

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
