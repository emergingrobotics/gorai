# NATS.io: The Messaging Backbone of Gorai

**Version:** 1.0
**Last Updated:** 2026-02-06

## Table of Contents

1. [What is NATS?](#what-is-nats)
2. [Why Gorai Uses NATS](#why-gorai-uses-nats)
3. [Core Concepts](#core-concepts)
   - [Subjects](#subjects)
   - [Publish/Subscribe](#publishsubscribe)
   - [Request/Reply](#requestreply)
   - [Queue Groups](#queue-groups)
4. [Architecture](#architecture)
   - [Client-Server Model](#client-server-model)
   - [Clustering](#clustering)
   - [Leaf Nodes](#leaf-nodes)
5. [JetStream](#jetstream)
   - [Streams](#streams)
   - [Consumers](#consumers)
   - [Retention Policies](#retention-policies)
   - [Replay Policies](#replay-policies)
6. [Key-Value Store](#key-value-store)
7. [Object Store](#object-store)
8. [How Gorai Uses NATS](#how-gorai-uses-nats)
   - [Topic Naming Convention](#topic-naming-convention)
   - [Component Communication](#component-communication)
   - [Service Discovery (Mesh)](#service-discovery-mesh)
   - [Telemetry and Sensor Data](#telemetry-and-sensor-data)
   - [Command and Control](#command-and-control)
   - [Configuration Storage](#configuration-storage)
9. [Performance Characteristics](#performance-characteristics)
10. [Comparison with Alternatives](#comparison-with-alternatives)

---

## What is NATS?

**NATS** (Neural Autonomic Transport System) is a high-performance, open-source messaging system designed for cloud-native applications, IoT, and microservices. Originally developed at Apcera and now a CNCF (Cloud Native Computing Foundation) incubating project, NATS provides simple, secure, and scalable communication between distributed systems.

At its core, NATS is:

- **Simple**: Text-based protocol, minimal configuration, easy to understand
- **Fast**: Written in Go, capable of millions of messages per second
- **Lightweight**: Small memory footprint (~10-20MB), starts instantly
- **Secure**: Built-in TLS, authentication, and authorization
- **Resilient**: Auto-reconnection, clustering, no single point of failure

### The NATS Ecosystem

```
┌─────────────────────────────────────────────────────────────────┐
│                        NATS Ecosystem                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │  Core NATS  │  │  JetStream  │  │   Services  │              │
│  │             │  │             │  │             │              │
│  │  • Pub/Sub  │  │  • Streams  │  │  • KV Store │              │
│  │  • Req/Rep  │  │  • Consumers│  │  • Object   │              │
│  │  • Queue    │  │  • Ack/Nak  │  │    Store    │              │
│  │    Groups   │  │  • Replay   │  │  • Micro    │              │
│  │             │  │             │  │    Services │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
│         │                │                │                      │
│         └────────────────┴────────────────┘                      │
│                          │                                       │
│              ┌───────────┴───────────┐                          │
│              │    NATS Server(s)     │                          │
│              │    (nats-server)      │                          │
│              └───────────────────────┘                          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Why Gorai Uses NATS

Gorai chose NATS as its messaging backbone for several reasons:

### 1. Simplicity Over Complexity

Unlike ROS 2's DDS (Data Distribution Service) which requires understanding QoS policies, partitions, and complex discovery mechanisms, NATS offers a straightforward pub/sub model that "just works."

```go
// Publishing in NATS - it's this simple
nc.Publish("gorai.robot1.motor.command", data)

// Subscribing
nc.Subscribe("gorai.robot1.motor.>", handler)
```

### 2. Language Agnostic

NATS has official clients for 40+ languages. Gorai components written in Go, Python, C++, or JavaScript can all communicate seamlessly.

### 3. Lightweight for Embedded Systems

NATS server runs comfortably on a Raspberry Pi with minimal resources:
- ~10-20MB memory footprint
- Instant startup
- No JVM, no heavyweight dependencies

### 4. Built-in Persistence (JetStream)

When you need message durability, replay, or exactly-once delivery, JetStream provides these without requiring a separate system like Kafka or RabbitMQ.

### 5. Integrated Key-Value Store

Configuration, parameters, and state can be stored in NATS KV without adding Redis or etcd to the stack.

### 6. Operational Simplicity

One binary (`nats-server`), one configuration file, no complex orchestration. Perfect for robotics deployments where simplicity reduces failure modes.

---

## Core Concepts

### Subjects

Subjects are the addressing mechanism in NATS. They are strings with tokens separated by dots (`.`).

```
gorai.robot1.camera.front.data
  │     │      │     │     │
  │     │      │     │     └── Data type
  │     │      │     └── Instance name
  │     │      └── Component type
  │     └── Robot identifier
  └── Namespace
```

#### Wildcards

NATS supports two wildcards:

| Wildcard | Meaning | Example |
|----------|---------|---------|
| `*` | Matches exactly one token | `gorai.*.motor.command` matches `gorai.robot1.motor.command` |
| `>` | Matches one or more tokens | `gorai.robot1.>` matches everything for robot1 |

```go
// Subscribe to all motor commands for any robot
nc.Subscribe("gorai.*.motor.command", handler)

// Subscribe to everything from robot1
nc.Subscribe("gorai.robot1.>", handler)

// Subscribe to all camera data
nc.Subscribe("gorai.*.camera.*.data", handler)
```

### Publish/Subscribe

The fundamental messaging pattern. Publishers send messages to subjects; subscribers receive messages from subjects they're interested in.

```
┌──────────┐                           ┌──────────┐
│ Publisher│                           │Subscriber│
│  (IMU)   │                           │ (Logger) │
└────┬─────┘                           └────┬─────┘
     │                                      │
     │  pub: gorai.robot1.sensor.imu.data   │
     │ ──────────────────────────────────►  │
     │                                      │
     │                                 ┌────┴─────┐
     │                                 │Subscriber│
     │                                 │(Dashboard)
     │ ──────────────────────────────► └──────────┘
     │
     ▼
┌─────────────────────────────────────────────────┐
│                  NATS Server                     │
│                                                  │
│  Subject: gorai.robot1.sensor.imu.data          │
│  Subscribers: Logger, Dashboard                  │
└─────────────────────────────────────────────────┘
```

**Key characteristics:**
- **Fire-and-forget**: Publisher doesn't know if anyone received the message
- **Fan-out**: One message can go to many subscribers
- **Decoupled**: Publisher and subscriber don't know about each other
- **At-most-once**: Without JetStream, messages may be lost if no subscribers

### Request/Reply

Synchronous communication pattern built on pub/sub. The requester sends a message and waits for a response.

```
┌──────────┐                           ┌──────────┐
│ Requester│                           │ Responder│
│ (Client) │                           │ (Service)│
└────┬─────┘                           └────┬─────┘
     │                                      │
     │  req: gorai.robot1.motor.get_state   │
     │  reply-to: _INBOX.abc123             │
     │ ──────────────────────────────────►  │
     │                                      │
     │  pub: _INBOX.abc123                  │
     │  {"power": 0.5, "moving": true}      │
     │ ◄──────────────────────────────────  │
     │                                      │
```

```go
// Requester
msg, err := nc.Request("gorai.robot1.motor.get_state", nil, time.Second)
fmt.Println(string(msg.Data)) // {"power": 0.5, "moving": true}

// Responder
nc.Subscribe("gorai.robot1.motor.get_state", func(msg *nats.Msg) {
    state := getMotorState()
    msg.Respond(state)
})
```

**Key characteristics:**
- **Synchronous**: Caller blocks until response or timeout
- **Point-to-point**: Only one responder answers (usually)
- **Service pattern**: Natural fit for RPC-style communication

### Queue Groups

Load-balanced message distribution. Multiple subscribers in the same queue group share the workload—each message goes to only one subscriber in the group.

```
                                    ┌──────────┐
                                    │ Worker 1 │ ─┐
┌──────────┐                        └──────────┘  │
│ Publisher│  ──── Message A ────►  ┌──────────┐  │ Queue Group
└──────────┘                        │ Worker 2 │  │ "processors"
             ──── Message B ────►   └──────────┘  │
                                    ┌──────────┐  │
             ──── Message C ────►   │ Worker 3 │ ─┘
                                    └──────────┘

Message A → Worker 1
Message B → Worker 3
Message C → Worker 2  (round-robin distribution)
```

```go
// All workers subscribe to same subject with same queue group
nc.QueueSubscribe("gorai.tasks.process", "processors", handler)
```

**Use cases in Gorai:**
- Distributing image processing across multiple workers
- Load-balanced command handling
- Parallel sensor fusion

---

## Architecture

### Client-Server Model

NATS uses a simple client-server architecture. All clients connect to one or more NATS servers. The server handles all message routing.

```
┌─────────┐     ┌─────────┐     ┌─────────┐
│ Client  │     │ Client  │     │ Client  │
│ (Motor) │     │ (Camera)│     │(Control)│
└────┬────┘     └────┬────┘     └────┬────┘
     │               │               │
     │               │               │
     └───────────────┼───────────────┘
                     │
                     ▼
              ┌─────────────┐
              │ NATS Server │
              │  :4222      │
              └─────────────┘
```

**Client connection features:**
- **Auto-reconnect**: Clients automatically reconnect on network failure
- **Buffering**: Messages queued during disconnection (configurable)
- **Ping/Pong**: Built-in keepalive mechanism

### Clustering

For high availability and scalability, NATS servers can be clustered. Clients can connect to any server in the cluster, and messages are routed transparently.

```
                    ┌─────────────┐
         ┌─────────►│   NATS-1    │◄─────────┐
         │          │  :4222      │          │
         │          └──────┬──────┘          │
         │                 │                 │
         │    ┌────────────┼────────────┐    │
         │    │ Cluster    │ Gossip     │    │
         │    │ :6222      │            │    │
         │    │            ▼            │    │
         │    │     ┌─────────────┐     │    │
┌────────┴──┐ │     │   NATS-2    │     │ ┌──┴────────┐
│ Robot     │ │     │  :4222      │     │ │ Ground    │
│ (Client)  │ │     └──────┬──────┘     │ │ Station   │
└───────────┘ │            │            │ └───────────┘
              │            ▼            │
              │     ┌─────────────┐     │
              │     │   NATS-3    │     │
              │     │  :4222      │     │
              └────►└─────────────┘◄────┘
```

**Cluster characteristics:**
- **Full mesh**: All servers connect to all other servers
- **Interest propagation**: Subscriptions propagate across cluster
- **No single point of failure**: Any server can handle any client

### Leaf Nodes

Leaf nodes extend a NATS cluster to remote locations with selective message sharing. Perfect for robots connecting to a central server.

```
┌─────────────────────────────────────────────────────────────┐
│                    Central Hub                               │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐              │
│  │  NATS-1  │────│  NATS-2  │────│  NATS-3  │              │
│  └──────────┘    └──────────┘    └──────────┘              │
│        │                                                     │
└────────┼─────────────────────────────────────────────────────┘
         │
         │ Leaf Node Connection
         │ (Only shares gorai.robot1.>)
         │
┌────────┼─────────────────────────────────────────────────────┐
│        ▼          Robot 1 (Field)                            │
│  ┌──────────┐                                                │
│  │NATS Leaf │                                                │
│  │ Node     │                                                │
│  └──────────┘                                                │
│       │                                                      │
│  ┌────┴────┐  ┌─────────┐  ┌─────────┐                      │
│  │ Motor   │  │ Camera  │  │  IMU    │                      │
│  │ Client  │  │ Client  │  │ Client  │                      │
│  └─────────┘  └─────────┘  └─────────┘                      │
└──────────────────────────────────────────────────────────────┘
```

**Benefits for robotics:**
- Local communication stays local (low latency)
- Only relevant data shared with hub (bandwidth efficient)
- Robot operates independently if hub disconnected

---

## JetStream

JetStream is NATS' built-in persistence layer. It provides:

- **Message persistence**: Messages stored on disk
- **Replay**: Consumers can replay historical messages
- **Acknowledgments**: At-least-once and exactly-once delivery
- **Consumer groups**: Durable, load-balanced consumption

### Streams

A stream is a persistent, ordered sequence of messages matching specified subjects.

```
┌─────────────────────────────────────────────────────────────┐
│                    Stream: GORAI_SENSORS                     │
│                                                              │
│  Subjects: gorai.*.sensor.>, gorai.*.camera.>               │
│  Storage: File                                               │
│  Retention: Limits (1 hour, 10GB)                           │
│                                                              │
│  ┌─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┐         │
│  │ M1  │ M2  │ M3  │ M4  │ M5  │ M6  │ M7  │ ... │         │
│  │IMU  │Cam  │IMU  │GPS  │IMU  │Cam  │IMU  │     │         │
│  └─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┘         │
│  Seq: 1    2     3     4     5     6     7                  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Creating a stream:**

```bash
# Via CLI
nats stream add GORAI_SENSORS \
    --subjects "gorai.*.sensor.>,gorai.*.camera.>" \
    --retention limits \
    --max-age 1h \
    --max-bytes 10GB \
    --storage file
```

```go
// Via Go client
js, _ := nc.JetStream()
js.AddStream(&nats.StreamConfig{
    Name:      "GORAI_SENSORS",
    Subjects:  []string{"gorai.*.sensor.>", "gorai.*.camera.>"},
    Retention: nats.LimitsPolicy,
    MaxAge:    time.Hour,
    MaxBytes:  10 * 1024 * 1024 * 1024, // 10GB
    Storage:   nats.FileStorage,
})
```

### Consumers

Consumers read from streams. They track position and can be durable (survive restarts) or ephemeral.

```
┌─────────────────────────────────────────────────────────────┐
│                    Stream: GORAI_SENSORS                     │
│                                                              │
│  ┌─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┐         │
│  │ M1  │ M2  │ M3  │ M4  │ M5  │ M6  │ M7  │ M8  │         │
│  └─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┘         │
│           ▲                       ▲                         │
│           │                       │                         │
│    Consumer A              Consumer B                       │
│    (Logger)                (Analyzer)                       │
│    Position: 2             Position: 6                      │
│    Durable: Yes            Durable: Yes                     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Consumer types:**

| Type | Use Case | Delivery |
|------|----------|----------|
| **Push** | Server pushes messages to subscriber | Messages delivered as they arrive |
| **Pull** | Client requests batches of messages | Client controls pace |

```go
// Push consumer
js.Subscribe("gorai.*.sensor.imu.>", handler,
    nats.Durable("imu-logger"),
    nats.DeliverAll(),
)

// Pull consumer
sub, _ := js.PullSubscribe("gorai.*.sensor.>", "batch-processor")
msgs, _ := sub.Fetch(100) // Get up to 100 messages
```

### Retention Policies

Streams can retain messages based on different policies:

| Policy | Behavior | Use Case |
|--------|----------|----------|
| **Limits** | Keep until limits exceeded (age, size, count) | Sensor data, logs |
| **Interest** | Keep while consumers exist | Temporary queues |
| **WorkQueue** | Delete after acknowledgment | Task processing |

### Replay Policies

How consumers receive messages:

| Policy | Behavior | Use Case |
|--------|----------|----------|
| **DeliverAll** | Replay from beginning | Full data recovery |
| **DeliverLast** | Start from most recent | Get current state |
| **DeliverLastPerSubject** | Last per subject | State snapshots |
| **DeliverNew** | Only new messages | Real-time processing |
| **DeliverByStartSequence** | From specific sequence | Resume from checkpoint |
| **DeliverByStartTime** | From specific time | Replay last N minutes |

---

## Key-Value Store

NATS KV is a distributed key-value store built on JetStream. It provides:

- **CRUD operations**: Put, Get, Delete, Purge
- **Watch**: Real-time notifications on changes
- **History**: Retrieve previous values
- **TTL**: Optional key expiration

```
┌─────────────────────────────────────────────────────────────┐
│                  KV Bucket: GORAI_CONFIG                     │
│                                                              │
│  ┌────────────────────┬────────────────────────────────┐    │
│  │ Key                │ Value                           │    │
│  ├────────────────────┼────────────────────────────────┤    │
│  │ robot.name         │ "sentinel-1"                   │    │
│  │ motor.max_rpm      │ 3000                           │    │
│  │ camera.resolution  │ {"width": 1920, "height": 1080}│    │
│  │ pid.kp             │ 1.5                            │    │
│  │ pid.ki             │ 0.1                            │    │
│  │ pid.kd             │ 0.05                           │    │
│  └────────────────────┴────────────────────────────────┘    │
│                                                              │
│  History: 10 revisions per key                              │
│  Storage: File                                               │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Basic operations:**

```go
js, _ := nc.JetStream()

// Create bucket
kv, _ := js.CreateKeyValue(&nats.KeyValueConfig{
    Bucket:  "GORAI_CONFIG",
    History: 10,
})

// Put
kv.Put("motor.max_rpm", []byte("3000"))

// Get
entry, _ := kv.Get("motor.max_rpm")
fmt.Println(string(entry.Value())) // "3000"

// Watch for changes
watcher, _ := kv.Watch("motor.>")
for entry := range watcher.Updates() {
    fmt.Printf("%s = %s\n", entry.Key(), entry.Value())
}

// Delete
kv.Delete("motor.max_rpm")

// Get history
history, _ := kv.History("motor.max_rpm")
for _, entry := range history {
    fmt.Printf("Rev %d: %s\n", entry.Revision(), entry.Value())
}
```

**CLI operations:**

```bash
# Create bucket
nats kv add GORAI_CONFIG --history 10

# Put value
nats kv put GORAI_CONFIG motor.max_rpm 3000

# Get value
nats kv get GORAI_CONFIG motor.max_rpm

# Watch for changes
nats kv watch GORAI_CONFIG ">"

# List keys
nats kv ls GORAI_CONFIG

# Get history
nats kv history GORAI_CONFIG motor.max_rpm
```

---

## Object Store

NATS Object Store handles large binary objects (maps, ML models, firmware). Built on JetStream with chunking for objects larger than the message size limit.

```go
js, _ := nc.JetStream()

// Create object store
os, _ := js.CreateObjectStore(&nats.ObjectStoreConfig{
    Bucket: "GORAI_MODELS",
})

// Store a file
os.PutFile("yolov8n.onnx", "/path/to/model.onnx")

// Retrieve
os.GetFile("yolov8n.onnx", "/tmp/model.onnx")

// Get info
info, _ := os.GetInfo("yolov8n.onnx")
fmt.Printf("Size: %d, Chunks: %d\n", info.Size, info.Chunks)
```

---

## How Gorai Uses NATS

### Topic Naming Convention

Gorai follows a consistent subject hierarchy:

```
gorai.<robot_id>.<component_type>.<instance>.<message_type>

Examples:
gorai.robot1.motor.left.command       # Motor command
gorai.robot1.motor.left.state         # Motor state feedback
gorai.robot1.sensor.imu.data          # IMU readings
gorai.robot1.camera.front.frame       # Camera frames
gorai.robot1.behavior.patrol.status   # Behavior status
gorai.mesh.announce                   # Service discovery
```

See [Framework Specification](../specs/gorai-framework-specification.md) for complete naming rules.

### Component Communication

```
┌─────────────────────────────────────────────────────────────────┐
│                       Robot System                               │
│                                                                  │
│  ┌─────────┐      ┌─────────────┐      ┌─────────┐             │
│  │  Motor  │◄─────│ Keypress    │◄─────│Keyboard │             │
│  │Component│ cmd  │ Motor Ctrl  │ key  │ Input   │             │
│  └─────────┘      └─────────────┘      └─────────┘             │
│       │                                                         │
│       │ state                                                   │
│       ▼                                                         │
│  ┌─────────┐      ┌─────────────┐      ┌─────────┐             │
│  │Telemetry│◄─────│   Camera    │ data │Dashboard│             │
│  │ Service │      │  Component  │─────►│  (Web)  │             │
│  └─────────┘      └─────────────┘      └─────────┘             │
│                                                                  │
│                   All via NATS pub/sub                          │
└─────────────────────────────────────────────────────────────────┘
```

### Service Discovery (Mesh)

Gorai's mesh system uses NATS KV for runtime service registration:

```
┌─────────────────────────────────────────────────────────────────┐
│                    NATS JetStream KV                             │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Bucket: gorai-services (TTL: 30s)                      │    │
│  │                                                          │    │
│  │  motor-controller-abc123: {                             │    │
│  │    "name": "motor-controller",                          │    │
│  │    "type": "component",                                 │    │
│  │    "subtype": "motor",                                  │    │
│  │    "publishes": ["gorai.robot1.motor.left.state"],      │    │
│  │    "subscribes": ["gorai.robot1.motor.left.command"],   │    │
│  │    "last_seen": "2026-02-06T10:30:00Z"                  │    │
│  │  }                                                       │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  Well-Known Subjects:                                           │
│  ├── gorai.mesh.announce      → Service join/leave              │
│  └── gorai.mesh.heartbeat.*   → Periodic health checks          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Telemetry and Sensor Data

High-frequency sensor data flows through core NATS (fire-and-forget):

```go
// IMU publishing at 100Hz
for reading := range imuReadings {
    data, _ := json.Marshal(reading)
    nc.Publish("gorai.robot1.sensor.imu.data", data)
}

// Consumer subscribes
nc.Subscribe("gorai.robot1.sensor.imu.data", func(msg *nats.Msg) {
    var reading IMUReading
    json.Unmarshal(msg.Data, &reading)
    processIMU(reading)
})
```

For data that needs persistence (logging, replay), JetStream streams capture it:

```bash
# Stream captures all sensor data
nats stream add GORAI_SENSORS \
    --subjects "gorai.*.sensor.>" \
    --retention limits \
    --max-age 1h
```

### Command and Control

Commands use request/reply for acknowledgment:

```go
// Sending a motor command with confirmation
cmd := MotorCommand{Power: 0.5}
data, _ := json.Marshal(cmd)

// Request with timeout
reply, err := nc.Request("gorai.robot1.motor.left.command", data, 500*time.Millisecond)
if err != nil {
    log.Error("Motor command failed", "error", err)
    return
}

var ack CommandAck
json.Unmarshal(reply.Data, &ack)
if !ack.Success {
    log.Error("Motor rejected command", "reason", ack.Error)
}
```

### Configuration Storage

Robot configuration lives in NATS KV:

```bash
# Store configuration
nats kv put GORAI_CONFIG robot.name "sentinel-1"
nats kv put GORAI_CONFIG motor.max_rpm 3000
nats kv put GORAI_CONFIG camera.resolution '{"width":1920,"height":1080}'

# Components watch for changes
nats kv watch GORAI_CONFIG "motor.>"
```

```go
// Component watches for config changes
kv, _ := js.KeyValue("GORAI_CONFIG")
watcher, _ := kv.Watch("motor.>")

go func() {
    for entry := range watcher.Updates() {
        if entry == nil {
            continue
        }
        log.Info("Config changed", "key", entry.Key(), "value", string(entry.Value()))
        applyConfig(entry.Key(), entry.Value())
    }
}()
```

---

## Performance Characteristics

### Core NATS

| Metric | Typical Value | Notes |
|--------|---------------|-------|
| Latency | <1ms | Local, 10ms+ over WAN |
| Throughput | 10M+ msg/sec | Single server |
| Message size | Up to 64MB | Default 1MB, configurable |
| Connections | 50,000+ | Per server |

### JetStream

| Metric | Typical Value | Notes |
|--------|---------------|-------|
| Write throughput | 100K+ msg/sec | Depends on storage |
| Read throughput | 500K+ msg/sec | From memory/cache |
| Storage efficiency | ~1.1x message size | Minimal overhead |

### Raspberry Pi 5 Performance

Benchmarks on Gorai reference hardware:

| Scenario | Performance |
|----------|-------------|
| Core pub/sub | ~500K msg/sec |
| JetStream writes | ~50K msg/sec (NVMe) |
| KV operations | ~10K ops/sec |
| Memory usage | 15-30MB typical |

---

## Comparison with Alternatives

### vs. ROS 2 (DDS)

| Feature | NATS | ROS 2 DDS |
|---------|------|-----------|
| Complexity | Simple | Complex |
| Learning curve | Hours | Days/weeks |
| Resource usage | Low | High |
| Language support | 40+ | Primarily C++/Python |
| Persistence | Built-in (JetStream) | External tools |
| Discovery | Optional mesh | Automatic |
| Best for | Prosumer robotics | Enterprise/research |

### vs. MQTT

| Feature | NATS | MQTT |
|---------|------|------|
| Pattern | Pub/sub + Req/reply | Pub/sub only |
| QoS | JetStream (flexible) | 0, 1, 2 (fixed) |
| Wildcards | `*`, `>` | `+`, `#` |
| Clustering | Native | Broker-dependent |
| KV Store | Built-in | No |
| Performance | Higher | Lower |
| Best for | Microservices, robotics | IoT sensors |

### vs. ZeroMQ

| Feature | NATS | ZeroMQ |
|---------|------|--------|
| Architecture | Client-server | Peer-to-peer |
| Broker | Required | Optional |
| Persistence | JetStream | None |
| Discovery | Mesh system | Manual |
| Complexity | Low | Medium |
| Best for | Distributed systems | Low-latency P2P |

### vs. Kafka

| Feature | NATS | Kafka |
|---------|------|-------|
| Resource usage | Very low | Very high |
| Operational complexity | Simple | Complex |
| Persistence | JetStream | Native |
| Ordering | Per-subject | Per-partition |
| Best for | Robotics, edge | Big data, enterprise |

---

## Summary

NATS provides Gorai with a unified messaging infrastructure that handles:

1. **Real-time communication** - Fast pub/sub for sensor data and commands
2. **Request/reply** - Synchronous RPC-style communication
3. **Persistence** - JetStream for durable messages and replay
4. **Configuration** - KV store for parameters and settings
5. **Service discovery** - Mesh system for runtime registration
6. **Large objects** - Object store for models and maps

All of this in a single, lightweight server that runs on embedded systems. This simplicity is a key differentiator for Gorai compared to more complex robotics middleware.

---

## Further Reading

- [NATS Documentation](https://docs.nats.io/)
- [JetStream Documentation](https://docs.nats.io/nats-concepts/jetstream)
- [Gorai NATS Setup Guide](../nats/nats-setup.md)
- [Gorai NATS Authentication](./gorai-nats-auth.md)
- [Framework Specification](../specs/gorai-framework-specification.md)
