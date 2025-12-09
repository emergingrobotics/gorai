# Chapter 3: NATS - The Communication Backbone

NATS is the foundation of GoRAI's communication. Understanding NATS deeply transforms how you think about robot architecture.

## 3.1 Why NATS?

GoRAI could have built on many messaging systems: ROS 2's DDS, ZeroMQ, gRPC, MQTT, or custom protocols. NATS won for compelling reasons.

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

GoRAI uses core NATS for real-time data (sensor streams, control commands) and JetStream when durability matters (configuration updates, logged data, mission waypoints).

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
```
# Three-server cluster for high availability
nats-server -c server1.conf
nats-server -c server2.conf
nats-server -c server3.conf
```

For most robots, a single local NATS server is sufficient. The option to scale exists when needed.
