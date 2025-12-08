# Chapter 3: NATS Messaging

> **In This Chapter:** Master NATS—the communication backbone of GoRAI. Learn pub/sub, request/reply, QoS levels, and JetStream persistence.

## Overview

NATS is to GoRAI what DDS is to ROS 2—the messaging layer that connects everything. But NATS comes from the cloud-native world, bringing lessons learned from operating systems at massive scale. It's simple, fast, and proven.

This chapter gives you deep understanding of NATS and how GoRAI uses it. We start with fundamentals (publish/subscribe, request/reply), move to GoRAI-specific patterns (topics, services, actions), cover quality of service options, and finish with JetStream for persistence.

## What You'll Learn

After reading this chapter, you'll understand:

- Why NATS was chosen over DDS, ZeroMQ, or gRPC
- Publish/subscribe for streaming data
- Request/reply for RPC-style calls
- Actions for long-running operations
- QoS levels and when to use each
- JetStream for message persistence

## Chapter Contents

| Section | Description |
|---------|-------------|
| [Why NATS?](whynats.md) | Comparison with alternatives, cloud-native heritage |
| [Fundamentals](fundamentals.md) | Subjects, publish, subscribe, wildcards |
| [GoRAI Patterns](patterns.md) | Topics, services, actions—GoRAI's NATS conventions |
| [Quality of Service](qos.md) | BestEffort, Reliable, Retained, History |
| [JetStream](jetstream.md) | Streams, consumers, persistence |
| [NATS CLI](cli.md) | Debugging and monitoring with `nats` command |

## Key Takeaways

- **NATS subjects** use dot-delimited hierarchies: `gorai.robot1.sensors.imu`
- **Wildcards** enable flexible subscriptions: `*` (single token), `>` (multiple tokens)
- **Topics** are pub/sub for streaming data (sensor readings, telemetry)
- **Services** are request/reply for commands and queries
- **Actions** handle long-running operations with feedback and cancellation
- **QoS** choices depend on data criticality and timing requirements
- **JetStream** adds persistence, replay, and exactly-once semantics

## Prerequisites

This chapter assumes you've read:
- [Chapter 2: Architecture](../../part1-getting-started/ch02-architecture/_index.md)

Understanding nodes and resources helps contextualize how NATS connects them.

<!-- book-only -->
*This is the most important chapter in Part II. Every subsequent chapter builds on NATS concepts. Take time to understand it thoroughly.*
<!-- /book-only -->

<!-- website-only -->
!!! warning "Foundation Chapter"
    Don't skip this chapter! All component and service chapters assume you understand NATS messaging patterns.
<!-- /website-only -->
