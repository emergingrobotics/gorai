# Mesh Service Discovery Specification

This document specifies the Gorai mesh service discovery system, which enables runtime service registration and discovery across independent processes.

## Overview

The mesh provides a NATS-native service discovery mechanism that allows:

1. **Runtime Registration**: Services register themselves when they start
2. **Channel Discovery**: Find available NATS subjects and their schemas
3. **Cross-Binary Discovery**: Independent processes discover each other
4. **Health Monitoring**: Automatic heartbeat and status tracking
5. **Schema Registry**: Message format documentation and validation

## Architecture

### Storage Layer (NATS KV)

The mesh uses three NATS JetStream KV buckets:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         NATS JetStream                                   │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────┐     │
│  │  gorai-services    (TTL: 30s, requires heartbeat)              │     │
│  │  Key: <robot_id>/<service_name>/<instance_id>                  │     │
│  │  Value: ServiceDescriptor JSON                                  │     │
│  └────────────────────────────────────────────────────────────────┘     │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────┐     │
│  │  gorai-channels    (persistent, no TTL)                        │     │
│  │  Key: <subject>                                                 │     │
│  │  Value: ChannelDescriptor JSON                                  │     │
│  └────────────────────────────────────────────────────────────────┘     │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────┐     │
│  │  gorai-schemas     (persistent, versioned)                     │     │
│  │  Key: <name>/<version>                                          │     │
│  │  Value: SchemaDescriptor JSON                                   │     │
│  └────────────────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────────────┘
```

### Communication Layer (NATS Subjects)

Well-known subjects for mesh operations:

| Subject | Purpose |
|---------|---------|
| `gorai.mesh.announce` | Service join/leave announcements |
| `gorai.mesh.heartbeat.<service-id>` | Per-service heartbeats |
| `$SRV.gorai-registry.*` | NATS micro service query API |

### Service Lifecycle

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Start     │────▶│  Register   │────▶│  Heartbeat  │
│             │     │  (KV Put)   │     │  (10s loop) │
└─────────────┘     └─────────────┘     └──────┬──────┘
                           │                    │
                           ▼                    │
                    ┌─────────────┐             │
                    │  Announce   │             │
                    │  (Join)     │             │
                    └─────────────┘             │
                                               │
                    ┌─────────────┐     ┌──────▼──────┐
                    │  Announce   │◀────│   Stop      │
                    │  (Leave)    │     │             │
                    └─────────────┘     └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │ Deregister  │
                    │ (KV Delete) │
                    └─────────────┘
```

## Data Types

### ServiceDescriptor

Describes a running service instance:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "motor-controller",
  "type": "component",
  "subtype": "motor",
  "model": "pwm",
  "robot_id": "robot-alpha",
  "version": "1.0.0",
  "endpoints": [
    {
      "name": "SetPower",
      "subject": "gorai.robot-alpha.motor.left.rpc",
      "description": "Set motor power level"
    }
  ],
  "publishes": [
    "gorai.robot-alpha.left_motor.state"
  ],
  "subscribes": [
    "gorai.robot-alpha.left_motor.command"
  ],
  "metadata": {
    "firmware": "2.1.0"
  },
  "host": "raspberrypi",
  "pid": 1234,
  "started_at": "2024-01-15T10:30:00Z",
  "last_seen": "2024-01-15T10:35:00Z",
  "status": "healthy"
}
```

**Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique instance UUID |
| `name` | string | Yes | Human-readable service name |
| `type` | string | Yes | "component" or "service" |
| `subtype` | string | Yes | Resource subtype (motor, camera, behavior) |
| `model` | string | Yes | Implementation model (pwm, fake, v4l2) |
| `robot_id` | string | Yes | Robot this service belongs to |
| `version` | string | No | Semantic version |
| `endpoints` | array | No | RPC endpoints exposed |
| `publishes` | array | No | Channels this service writes to |
| `subscribes` | array | No | Channels this service reads from |
| `metadata` | object | No | Custom key-value tags |
| `host` | string | No | Hostname where service runs |
| `pid` | int | No | Process ID |
| `started_at` | string | Yes | ISO 8601 start time |
| `last_seen` | string | Yes | ISO 8601 last heartbeat |
| `status` | string | Yes | healthy, degraded, stale, unknown |

### ChannelDescriptor

Describes a NATS subject/channel:

```json
{
  "subject": "gorai.robot-alpha.imu.data",
  "schema": "gorai.sensor.IMUReading/v1",
  "qos": "best_effort",
  "direction": "pub",
  "publisher": "550e8400-e29b-41d4-a716-446655440000",
  "description": "IMU sensor readings at 100Hz",
  "sample_rate": "100Hz",
  "robot_id": "robot-alpha",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `subject` | string | Yes | NATS subject pattern |
| `schema` | string | No | Schema key reference |
| `qos` | string | Yes | best_effort, reliable, retained, history |
| `direction` | string | Yes | pub, sub, req-rep, bidirectional |
| `publisher` | string | No | Service ID that owns this channel |
| `description` | string | No | Human-readable description |
| `sample_rate` | string | No | Expected publish rate |
| `robot_id` | string | No | Robot this channel belongs to |
| `created_at` | string | Yes | ISO 8601 creation time |
| `updated_at` | string | Yes | ISO 8601 last update |

### SchemaDescriptor

Describes a message schema:

```json
{
  "name": "gorai.sensor.IMUReading",
  "version": "v1",
  "format": "json-schema",
  "definition": {
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object",
    "properties": {
      "linear_acceleration": { "$ref": "#/definitions/Vector3" },
      "angular_velocity": { "$ref": "#/definitions/Vector3" }
    }
  },
  "description": "IMU sensor reading with acceleration and angular velocity",
  "examples": [...],
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

**Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Schema name (e.g., gorai.sensor.IMUReading) |
| `version` | string | Yes | Schema version (e.g., v1, v2) |
| `format` | string | Yes | json-schema, protobuf, avro |
| `definition` | object | Yes | The actual schema |
| `description` | string | No | Human-readable description |
| `examples` | array | No | Example messages |
| `created_at` | string | Yes | ISO 8601 creation time |
| `updated_at` | string | Yes | ISO 8601 last update |

## Service Status

Services have the following status values:

| Status | Description | Condition |
|--------|-------------|-----------|
| `healthy` | Service is responsive | Last heartbeat < 20s ago |
| `degraded` | Service has issues | Explicitly set by service |
| `stale` | Missing heartbeats | Last heartbeat 20-30s ago |
| `unknown` | Cannot determine status | Last heartbeat > 30s ago |

## QoS Levels

Channels support different quality-of-service levels:

| QoS | Transport | Persistence | Use Case |
|-----|-----------|-------------|----------|
| `best_effort` | Core NATS | None | High-frequency sensor data |
| `reliable` | JetStream | Ack required | Commands, important events |
| `retained` | JetStream | Last value | State, configuration |
| `history` | JetStream | Last N | Catch-up subscribers |

## Timing Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| Service TTL | 30s | KV entry expires without heartbeat |
| Heartbeat interval | 10s | Time between heartbeats |
| Stale threshold | 20s | When service is marked stale |

## Usage Examples

### Service Registration

```go
package main

import (
    "context"
    "github.com/nats-io/nats.go"
    "github.com/gorai/gorai/pkg/mesh"
)

func main() {
    nc, _ := nats.Connect("nats://localhost:4222")
    client, _ := mesh.NewClient(nc)

    // Register service with channels
    reg, _ := client.Register(context.Background(), mesh.ServiceDescriptor{
        Name:    "motor-controller",
        Type:    mesh.TypeComponent,
        Subtype: "motor",
        Model:   "pwm",
        RobotID: "robot-alpha",
        Publishes: []string{
            "gorai.robot-alpha.left_motor.state",
            "gorai.robot-alpha.right_motor.state",
        },
        Subscribes: []string{
            "gorai.robot-alpha.left_motor.command",
            "gorai.robot-alpha.right_motor.command",
        },
    }, mesh.WithChannels(
        mesh.ChannelDescriptor{
            Subject:     "gorai.robot-alpha.left_motor.state",
            QoS:         mesh.QoSRetained,
            Direction:   mesh.DirectionPub,
            Schema:      "gorai.actuator.MotorState/v1",
            Description: "Left motor state",
        },
    ))

    defer reg.Deregister()

    // Run service...
}
```

### Service Discovery

```go
// Find all motor components
motors, _ := client.FindServices(ctx, mesh.Query{
    RobotID: "robot-alpha",
    Type:    mesh.TypeComponent,
    Subtype: "motor",
})

for _, motor := range motors {
    fmt.Printf("Found motor: %s (status: %s)\n", motor.Name, motor.Status)
}

// Watch for new services
watcher, _ := client.WatchServices(ctx, mesh.Query{RobotID: "robot-alpha"})
for event := range watcher.Events() {
    switch event.Type {
    case mesh.EventServiceJoined:
        fmt.Printf("Service joined: %s\n", event.Service.Name)
    case mesh.EventServiceLeft:
        fmt.Printf("Service left: %s\n", event.Service.Name)
    }
}
```

### Channel Discovery

```go
// List all channels for a robot
channels, _ := client.ListChannelsByRobot(ctx, "robot-alpha")

for _, ch := range channels {
    fmt.Printf("Channel: %s [%s] - %s\n", ch.Subject, ch.QoS, ch.Description)
}

// Get schema for a channel
schema, _ := client.GetChannelSchema(ctx, "gorai.robot-alpha.imu.data")
fmt.Printf("Schema: %s\n", schema.Definition)
```

## CLI Commands

```bash
# List all registered services
gorai mesh services

# List services for a specific robot
gorai mesh services robot-alpha

# Filter by type/subtype
gorai mesh services --type component --subtype motor

# List all channels
gorai mesh channels

# List channels for a robot (JSON output)
gorai mesh channels robot-alpha --json

# Show schema
gorai mesh schemas gorai.sensor.IMUReading/v1

# Watch for changes
gorai mesh watch robot-alpha

# Show summary
gorai mesh summary

# Initialize predefined schemas
gorai mesh init

# Reset mesh data
gorai mesh reset --force
```

## NATS Micro Service API

The mesh includes a NATS micro service for remote queries:

```
Subject: gorai-registry.registry.<endpoint>
```

Available endpoints:

| Endpoint | Request | Description |
|----------|---------|-------------|
| `list-services` | (none) | List all services |
| `find-services` | Query JSON | Find matching services |
| `get-service` | `{robot_id, name}` | Get specific service |
| `list-channels` | (none) | List all channels |
| `find-channels` | ChannelQuery JSON | Find matching channels |
| `get-channel` | `{subject}` | Get specific channel |
| `list-schemas` | (none) | List all schemas |
| `get-schema` | `{name, version}` | Get specific schema |
| `summary` | (none) | Get mesh summary |
| `list-robots` | (none) | List all robots |

## Design Rationale

### Why NATS KV?

1. **Native Integration**: No additional infrastructure beyond NATS
2. **TTL Support**: Automatic cleanup of stale entries
3. **Watch Capability**: Real-time change notifications
4. **Atomic Operations**: Safe concurrent access
5. **Replication**: Built-in HA with NATS clustering

### Alternatives Considered

| Option | Pros | Cons | Decision |
|--------|------|------|----------|
| Consul | Feature-rich, battle-tested | Extra infrastructure | Rejected - overkill |
| etcd | Strong consistency | Operational complexity | Rejected - overkill |
| DNS-SD | Standard, local discovery | Poor fit for pub/sub | Rejected - wrong model |
| Custom channel | Simple | No persistence, race conditions | Rejected - too limited |
| **NATS KV** | Native, simple, sufficient | Requires JetStream | **Selected** |

## Predefined Schemas

The mesh includes predefined schemas for common message types:

- `gorai.sensor.IMUReading/v1` - IMU sensor data
- `gorai.sensor.GPSReading/v1` - GPS position data
- `gorai.actuator.MotorCommand/v1` - Motor control commands
- `gorai.actuator.MotorState/v1` - Motor state feedback
- `gorai.camera.Frame/v1` - Camera frame metadata
- `gorai.system.Heartbeat/v1` - Service heartbeat
- `gorai.system.Announcement/v1` - Service join/leave

Run `gorai mesh init` to populate these schemas.

## Implementation Location

```
pkg/mesh/
├── types.go          # Core types (ServiceDescriptor, etc.)
├── client.go         # Main client interface
├── kv.go             # NATS KV bucket management
├── registration.go   # Service registration + heartbeat
├── discovery.go      # Query services and channels
├── watcher.go        # Watch for changes
├── schema.go         # Schema registry + predefined schemas
└── micro.go          # NATS micro service query API

cmd/gorai/commands/
└── mesh.go           # CLI commands
```

## Gateway Integration

Protocol gateways (GSP/2, Modbus, CAN) bridge hardware devices to NATS and should register with the mesh.

### Gateway Self-Registration

```go
// Gateway registers itself as a service
meshClient.Register(ctx, mesh.ServiceDescriptor{
    Name:    "gsp-gateway",
    Type:    mesh.TypeService,
    Subtype: "gateway",
    Model:   "gsp",
    RobotID: robotID,
    Metadata: map[string]string{
        "protocol":  "gsp/2",
        "transport": "serial",
    },
})
```

### Device Bridge Registration

Each connected device registers as a component:

```go
// When device connects and reports capabilities
meshClient.Register(ctx, mesh.ServiceDescriptor{
    Name:    "pico-001",
    Type:    mesh.TypeComponent,
    Subtype: "pwm-controller",
    Model:   "gsp-device",
    RobotID: robotID,
    Version: "1.0.0",
    Metadata: map[string]string{
        "capabilities": "PWM,IMU,GPIO",
        "serial_port":  "/dev/ttyACM0",
    },
    Publishes: []string{
        "gsp.pico-001.rx.sensor.imu_data",
        "gsp.pico-001.rx.event.heartbeat",
    },
    Subscribes: []string{
        "gsp.pico-001.tx.command.pwm_set",
    },
})
```

### Subject Namespace Strategy

| Layer | Prefix | Example |
|-------|--------|---------|
| Gateway (raw) | `gsp.<device>` | `gsp.pico-001.rx.sensor.imu_data` |
| Gorai (normalized) | `gorai.<robot>.<component>` | `gorai.scout.imu.data` |

Gateways use their own namespace (`gsp.*`) while the mesh tracks them for discovery. Optional bridge services can translate between namespaces.

See [Dynamic Discovery Specification](dynamic-discovery.md) for complete gateway integration patterns.

---

## Related Documents

- [Dynamic Discovery](dynamic-discovery.md) — Auto-adoption and dynamic dependencies
- [NATS Configuration](../nats/nats.conf)
- [Topic Naming Conventions](../pkg/topics/topics.go)
- [Component Registry](../pkg/registry/registry.go)
- [Framework Specification](gorai-framework-specification.md)
- [GSP-NATS Gateway](../../gorai-nats-gw/docs/DESIGN.md) — Gateway design document
