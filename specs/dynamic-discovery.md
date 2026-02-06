# Dynamic Discovery Specification

**Version 0.1.0**

This specification defines how Gorai supports runtime discovery of devices and services not explicitly defined in the Robot Definition Language (RDL).

---

## Overview

Dynamic discovery enables robots to:

1. **Discover devices at runtime** — Find hardware not declared in RDL
2. **Auto-adopt resources** — Automatically integrate discovered devices
3. **Resolve dynamic dependencies** — Services can depend on discovered resources
4. **Adapt to hardware changes** — Hot-plug devices without restart

This creates a **hybrid static/dynamic** model where RDL defines structure and rules, while mesh provides runtime discovery.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Robot Runtime                                  │
│                                                                          │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────────────┐  │
│  │    RDL      │    │  Discovery  │    │      Mesh Registry          │  │
│  │  (Static)   │───►│   Manager   │◄──►│  (Runtime State)            │  │
│  └─────────────┘    └──────┬──────┘    └─────────────────────────────┘  │
│                            │                        ▲                    │
│                            ▼                        │                    │
│                    ┌───────────────┐                │                    │
│                    │ Proxy Factory │                │                    │
│                    └───────┬───────┘                │                    │
│                            │                        │                    │
│         ┌──────────────────┼──────────────────┐     │                    │
│         ▼                  ▼                  ▼     │                    │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐                    │
│  │ RemoteMotor │   │ RemoteIMU   │   │RemoteCamera │                    │
│  │   (Proxy)   │   │   (Proxy)   │   │   (Proxy)   │                    │
│  └──────┬──────┘   └──────┬──────┘   └──────┬──────┘                    │
│         │                 │                 │                            │
│         └─────────────────┼─────────────────┘                            │
│                           │ NATS                                         │
└───────────────────────────┼─────────────────────────────────────────────┘
                            │
              ┌─────────────┼─────────────┐
              ▼             ▼             ▼
       ┌───────────┐ ┌───────────┐ ┌───────────┐
       │  Gateway  │ │  Gateway  │ │  External │
       │ (GSP/2)   │ │ (Modbus)  │ │  Service  │
       └─────┬─────┘ └─────┬─────┘ └───────────┘
             │             │
        ┌────┴────┐   ┌────┴────┐
        │  Pico   │   │   PLC   │
        │ Device  │   │ Device  │
        └─────────┘   └─────────┘
```

---

## Discovery Sources

### 1. Gateway Discovery

Gateways bridge hardware protocols (GSP/2, Modbus, CAN) to NATS and register devices in the mesh.

```yaml
gateways:
  - name: "gsp-gateway"
    type: "gateway/gsp"
    config:
      discovery:
        enabled: true
        patterns: ["/dev/ttyACM*", "/dev/ttyUSB*"]
        probe_interval: "5s"
```

**Flow:**
1. Gateway probes serial ports matching patterns
2. Device responds with BOOT/capabilities
3. Gateway registers device in mesh
4. Discovery manager sees new service
5. Proxy component created

### 2. Mesh Discovery

Discover services already registered in the mesh by other processes.

```yaml
discovery:
  sources:
    - type: "mesh"
      query:
        subtype: "motor"
        status: "healthy"
```

**Flow:**
1. Discovery manager queries mesh
2. Matching services returned
3. Proxy components created for each

### 3. Network Discovery (Future)

Discover services via mDNS/DNS-SD or custom protocols.

```yaml
discovery:
  sources:
    - type: "mdns"
      service: "_gorai._tcp"
```

---

## RDL Schema Extensions

### Gateways Section

New top-level section for protocol gateways:

```json
{
  "gateways": [
    {
      "name": "device-gateway",
      "type": "gateway/gsp",
      "config": {
        "nats": {
          "subject_prefix": "gsp"
        },
        "discovery": {
          "enabled": true,
          "patterns": ["/dev/ttyACM*"],
          "probe_message": "PING",
          "probe_interval": "5s"
        },
        "devices": [
          {
            "id": "flow-001",
            "port": "/dev/ttyACM0",
            "baud": 115200
          }
        ]
      }
    }
  ]
}
```

### Discovery Section

Defines how to discover and adopt resources:

```json
{
  "discovery": {
    "enabled": true,
    "auto_adopt": true,
    "sources": [
      {
        "type": "gateway",
        "gateway": "device-gateway"
      },
      {
        "type": "mesh",
        "query": {
          "robot_id": "robot-alpha",
          "subtype": "motor"
        }
      }
    ],
    "rules": [
      {
        "match": {"capability": "PWM"},
        "adopt_as": {
          "type": "motor",
          "model": "gsp-pwm"
        },
        "config": {
          "min_pulse": 1000,
          "max_pulse": 2000
        }
      },
      {
        "match": {"capability": "IMU"},
        "adopt_as": {
          "type": "sensor",
          "subtype": "imu",
          "model": "gsp-imu"
        },
        "config": {
          "sample_rate": 100
        }
      }
    ]
  }
}
```

### Dynamic Dependencies

Services can depend on discovered resources using the `@discovered:` prefix:

```json
{
  "services": [
    {
      "name": "patrol",
      "type": "behavior/patrol",
      "depends_on": [
        "camera",
        "@discovered:motor/*",
        "@discovered:sensor/imu/*"
      ]
    }
  ]
}
```

**Dependency Patterns:**

| Pattern | Meaning |
|---------|---------|
| `@discovered:motor/*` | Any discovered motor |
| `@discovered:motor/left` | Discovered motor named "left" |
| `@discovered:sensor/imu/*` | Any discovered IMU sensor |
| `@discovered:*` | Any discovered resource |

---

## Gateway-Mesh Integration

### Gateway Registers Itself

When gateway starts, it registers as a service:

```go
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

### Device Bridges Register as Components

Each connected device registers:

```go
meshClient.Register(ctx, mesh.ServiceDescriptor{
    Name:    deviceID,           // "pico-001"
    Type:    mesh.TypeComponent,
    Subtype: detectSubtype(caps), // "pwm-controller"
    Model:   "gsp-device",
    RobotID: robotID,
    Version: fwVersion,
    Metadata: map[string]string{
        "hw_rev":      hwRev,
        "device_id":   uniqueID,
        "serial_port": port,
        "capabilities": strings.Join(caps, ","),
    },
    Publishes: []string{
        "gsp." + deviceID + ".rx.sensor.>",
    },
    Subscribes: []string{
        "gsp." + deviceID + ".tx.command.>",
    },
})
```

### Channel Registration

Each sensor/actuator capability registers its channel:

```go
meshClient.RegisterChannel(ctx, mesh.ChannelDescriptor{
    Subject:     "gsp.pico-001.rx.sensor.imu_data",
    Schema:      "gorai.sensor.IMUReading/v1",
    QoS:         mesh.QoSBestEffort,
    Direction:   mesh.DirectionPub,
    SampleRate:  "100Hz",
    Description: "IMU sensor data from Pico device",
    RobotID:     robotID,
})
```

---

## Subject Namespace Strategy

### Dual Namespace Approach (Recommended)

Keep gateway and Gorai subjects separate:

| Layer | Prefix | Example |
|-------|--------|---------|
| Gateway (raw) | `gsp.<device>` | `gsp.pico-001.rx.sensor.imu_data` |
| Gorai (normalized) | `gorai.<robot>.<component>` | `gorai.scout.imu.data` |

**Benefits:**
- Clean separation of concerns
- Raw data available for debugging
- Gorai subjects follow standard conventions

### Optional: Bridge Service

A lightweight bridge can translate between namespaces:

```go
// Subscribe to raw GSP data
nc.Subscribe("gsp.*.rx.sensor.imu_data", func(msg *nats.Msg) {
    // Transform and republish to Gorai namespace
    nc.Publish("gorai.scout."+deviceID+".data", transformed)
})
```

---

## Proxy Components

Discovered devices are wrapped in proxy components that implement standard Gorai interfaces.

### RemoteMotor Proxy

```go
type RemoteMotor struct {
    meshClient *mesh.Client
    natsConn   *nats.Conn
    descriptor mesh.ServiceDescriptor
    cmdSubject string
}

func NewRemoteMotor(mc *mesh.Client, nc *nats.Conn, desc mesh.ServiceDescriptor) (*RemoteMotor, error) {
    // Find command subject from descriptor
    cmdSubject := ""
    for _, sub := range desc.Subscribes {
        if strings.Contains(sub, "command") {
            cmdSubject = sub
            break
        }
    }

    return &RemoteMotor{
        meshClient: mc,
        natsConn:   nc,
        descriptor: desc,
        cmdSubject: cmdSubject,
    }, nil
}

func (m *RemoteMotor) SetPower(ctx context.Context, power float64) error {
    cmd := map[string]any{
        "type": "PWM_SET",
        "data": map[string]any{
            "channels": []map[string]any{
                {"channel": 0, "pulse_us": powerToPulse(power)},
            },
        },
    }
    data, _ := json.Marshal(cmd)
    return m.natsConn.Publish(m.cmdSubject, data)
}

func (m *RemoteMotor) Stop(ctx context.Context) error {
    return m.SetPower(ctx, 0)
}

func (m *RemoteMotor) IsMoving(ctx context.Context) (bool, error) {
    // Query state via request-reply
    // ...
}

func (m *RemoteMotor) Close(ctx context.Context) error {
    return nil // Nothing to close for remote
}

var _ motor.Motor = (*RemoteMotor)(nil)
```

### RemoteSensor Proxy

```go
type RemoteSensor struct {
    meshClient  *mesh.Client
    natsConn    *nats.Conn
    descriptor  mesh.ServiceDescriptor
    dataSubject string
    lastReading map[string]any
    mu          sync.RWMutex
}

func (s *RemoteSensor) Readings(ctx context.Context) (map[string]any, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.lastReading, nil
}

// Background subscription updates lastReading
func (s *RemoteSensor) subscribe(ctx context.Context) {
    s.natsConn.Subscribe(s.dataSubject, func(msg *nats.Msg) {
        var env map[string]any
        json.Unmarshal(msg.Data, &env)

        s.mu.Lock()
        s.lastReading = env["data"].(map[string]any)
        s.mu.Unlock()
    })
}
```

---

## Runtime Lifecycle

### Startup Sequence

```
1. Parse RDL
2. Connect to NATS
3. Initialize mesh client
4. Start gateways
   └─ Gateways begin device discovery
   └─ Gateways register in mesh
5. Start static components (from RDL)
6. Start discovery manager
   └─ Query mesh for existing services
   └─ Watch for new services
7. Auto-adopt discovered devices
   └─ Create proxy components
8. Resolve dynamic dependencies
   └─ Match @discovered: patterns
9. Start services (dependencies now resolved)
10. Robot running
```

### Hot-Plug Flow

```
1. New device connected to USB
2. Gateway detects new serial port
3. Gateway probes device (PING)
4. Device responds (BOOT + capabilities)
5. Gateway registers device in mesh
6. Discovery manager receives event
7. Adoption rules evaluated
8. Proxy component created
9. Dependent services notified
10. Device available for use
```

### Disconnect Flow

```
1. Device disconnected from USB
2. Gateway detects serial port gone
3. Gateway deregisters from mesh
4. Discovery manager receives event
5. Proxy component removed
6. Dependent services notified
7. Services handle degraded state
```

---

## Example: Complete Dynamic Robot

```json
{
  "version": "4",
  "robot": {
    "name": "scout",
    "namespace": "gorai"
  },
  "nats": {
    "url": "nats://localhost:4222"
  },

  "gateways": [
    {
      "name": "usb-gateway",
      "type": "gateway/gsp",
      "config": {
        "discovery": {
          "enabled": true,
          "patterns": ["/dev/ttyACM*"]
        }
      }
    }
  ],

  "discovery": {
    "enabled": true,
    "auto_adopt": true,
    "sources": [
      {"type": "gateway", "gateway": "usb-gateway"}
    ],
    "rules": [
      {
        "match": {"capability": "PWM"},
        "adopt_as": {"type": "motor", "model": "remote-pwm"}
      },
      {
        "match": {"capability": "IMU"},
        "adopt_as": {"type": "sensor", "subtype": "imu", "model": "remote-imu"}
      },
      {
        "match": {"capability": "GPS"},
        "adopt_as": {"type": "sensor", "subtype": "gps", "model": "remote-gps"}
      }
    ]
  },

  "components": [
    {
      "name": "camera",
      "type": "camera/v4l2",
      "config": {"device": "/dev/video0"}
    }
  ],

  "services": [
    {
      "name": "vision",
      "type": "vision/yolo",
      "depends_on": ["camera"]
    },
    {
      "name": "navigation",
      "type": "navigation/waypoint",
      "depends_on": [
        "@discovered:sensor/gps/*",
        "@discovered:sensor/imu/*"
      ]
    },
    {
      "name": "patrol",
      "type": "behavior/patrol",
      "config": {
        "waypoints": [[0,0], [10,0], [10,10], [0,10]]
      },
      "depends_on": [
        "vision",
        "navigation",
        "@discovered:motor/*"
      ]
    }
  ]
}
```

**Behavior:**

1. Robot starts with only camera defined statically
2. Gateway discovers Pico with PWM+IMU+GPS capabilities
3. Three proxy components created automatically
4. Navigation service starts (GPS+IMU dependencies resolved)
5. Patrol service starts (motor dependency resolved)
6. Robot begins patrol using discovered hardware

---

## CLI Commands

```bash
# List discovered resources
gorai mesh services --discovered

# Watch for discoveries
gorai mesh watch

# Force re-discovery
gorai discovery scan

# List adoption rules
gorai discovery rules

# Manually adopt a service
gorai discovery adopt <service-id> --as motor/pwm
```

---

## Implementation Packages

```
pkg/
├── discovery/
│   ├── manager.go      # Discovery manager
│   ├── source.go       # Discovery source interface
│   ├── mesh_source.go  # Mesh-based discovery
│   ├── rules.go        # Adoption rules
│   └── config.go       # Configuration types
├── proxy/
│   ├── factory.go      # Proxy component factory
│   ├── motor.go        # RemoteMotor proxy
│   ├── sensor.go       # RemoteSensor proxy
│   └── camera.go       # RemoteCamera proxy
└── mesh/
    └── ...             # Existing mesh package
```

---

## Related Documents

- [Mesh Service Discovery](mesh-service-discovery.md) — Runtime service registry
- [Robot Definition Language](robot-definition-language.md) — RDL specification
- [GSP-NATS Gateway](../../gorai-nats-gw/docs/DESIGN.md) — Gateway design
- [LLM Design Guide](../docs/LLM-DESIGN-GUIDE.md) — Component development guide
