# Core Concepts

Understanding the Gorai mental model.

## Nodes

A node is a process that connects to NATS and participates in the robot system.

## Resources

Everything in Gorai is a Resource:

- **Components**: Hardware abstractions (motors, sensors, cameras)
- **Services**: Software capabilities (vision, SLAM, navigation)

## Communication Patterns

| Pattern | Use Case | NATS Primitive |
|---------|----------|----------------|
| Topics | Sensor streams | Pub/Sub |
| Services | Synchronous RPC | Request/Reply |
| Actions | Long-running tasks | Request/Reply + Pub/Sub |

## Next Steps

- [Components Guide](../guides/components.md)
- [NATS Messaging](../guides/nats.md)
