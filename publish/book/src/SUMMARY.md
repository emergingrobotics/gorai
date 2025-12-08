# Summary

[Introduction](README.md)

---

# Part I: Getting Started

- [Part Overview](part1-getting-started/_index.md)
- [Why GoRAI?](part1-getting-started/ch01-why-gorai/_index.md)
    - [The Robotics Landscape](part1-getting-started/ch01-why-gorai/landscape.md)
    - [Design Philosophy](part1-getting-started/ch01-why-gorai/philosophy.md)
    - [Target Audience](part1-getting-started/ch01-why-gorai/audience.md)
    - [What You'll Build](part1-getting-started/ch01-why-gorai/whatyoullbuild.md)
    - [Prerequisites](part1-getting-started/ch01-why-gorai/prerequisites.md)
- [Architecture & Mental Model](part1-getting-started/ch02-architecture/_index.md)
    - [The Big Picture](part1-getting-started/ch02-architecture/bigpicture.md)
    - [Core Concepts](part1-getting-started/ch02-architecture/coreconcepts.md)
    - [Distributed Systems](part1-getting-started/ch02-architecture/distributed.md)
    - [Configuration](part1-getting-started/ch02-architecture/config.md)
    - [NWS/NWC Pattern](part1-getting-started/ch02-architecture/nwsnwc.md)

---

# Part II: Core Framework

- [Part Overview](part2-core-framework/_index.md)
- [NATS Messaging](part2-core-framework/ch03-nats/_index.md)
    - [Why NATS?](part2-core-framework/ch03-nats/whynats.md)
    - [Fundamentals](part2-core-framework/ch03-nats/fundamentals.md)
    - [GoRAI Patterns](part2-core-framework/ch03-nats/patterns.md)
    - [Quality of Service](part2-core-framework/ch03-nats/qos.md)
    - [JetStream](part2-core-framework/ch03-nats/jetstream.md)
    - [NATS CLI](part2-core-framework/ch03-nats/cli.md)
- [Sensors](part2-core-framework/ch04-sensors/_index.md)
    - [The Sensor Interface](part2-core-framework/ch04-sensors/interface.md)
    - [Built-in Sensors](part2-core-framework/ch04-sensors/builtin.md)
    - [Data Types](part2-core-framework/ch04-sensors/datatypes.md)
    - [Fake Sensors](part2-core-framework/ch04-sensors/fakes.md)
- [Actuators](part2-core-framework/ch05-actuators/_index.md)
    - [The Actuator Interface](part2-core-framework/ch05-actuators/actuator.md)
    - [Motors](part2-core-framework/ch05-actuators/motor.md)
    - [Motor Types](part2-core-framework/ch05-actuators/motortypes.md)
    - [Control Patterns](part2-core-framework/ch05-actuators/control.md)
    - [Servos](part2-core-framework/ch05-actuators/servo.md)
    - [Bases & Arms](part2-core-framework/ch05-actuators/base_arm.md)
- [Vision, Links & More](part2-core-framework/ch06-vision/_index.md)
    - [Camera Interface](part2-core-framework/ch06-vision/camera.md)
    - [Camera Types](part2-core-framework/ch06-vision/types.md)
    - [Data Flow](part2-core-framework/ch06-vision/dataflow.md)
    - [Computer Vision](part2-core-framework/ch06-vision/cv.md)
- [Services](part2-core-framework/ch07-services/_index.md)
- [Behaviors](part2-core-framework/ch08-behaviors/_index.md)
- [Coordinators](part2-core-framework/ch09-coordinators/_index.md)

---

# Part III: Development

- [Part Overview](part3-development/_index.md)
- [Development Environment](part3-development/ch10-devenv/_index.md)
- [Hello Sensor Deep Dive](part3-development/ch11-hello-sensor/_index.md)
    - [Overview](part3-development/ch11-hello-sensor/overview.md)
    - [The Reader](part3-development/ch11-hello-sensor/reader.md)
    - [The Sensor](part3-development/ch11-hello-sensor/sensor.md)
    - [Main Program](part3-development/ch11-hello-sensor/main.md)
- [Building Custom Components](part3-development/ch12-custom/_index.md)
- [Testing Strategies](part3-development/ch13-testing/_index.md)

---

# Part IV: Advanced Topics

- [Part Overview](part4-advanced/_index.md)
- [AI/ML Integration](part4-advanced/ch14-ai-ml/_index.md)
- [Project Organization](part4-advanced/ch15-organization/_index.md)
- [AI-Assisted Development](part4-advanced/ch16-ai-dev/_index.md)
- [Conclusion & Next Steps](part4-advanced/ch17-conclusion/_index.md)

---

# Reference

- [Appendices](appendices/_index.md)
    - [NATS Topics Reference](appendices/topics.md)
    - [Protocol Buffers Reference](appendices/protobuf.md)
    - [Hardware Compatibility](appendices/hardware.md)
    - [Troubleshooting](appendices/troubleshooting.md)
    - [Glossary](appendices/glossary.md)
- [Reference](reference/_index.md)
    - [CLI Reference](reference/cli.md)
    - [Configuration](reference/configuration.md)
    - [Topic Naming](reference/topic-naming.md)
- [Examples](examples/_index.md)
    - [Hello Sensor](examples/hello-sensor/_index.md)
    - [Pan-Tilt Platform](examples/pan-tilt/_index.md)
    - [Surface Vehicle](examples/skimmer/_index.md)
