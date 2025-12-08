# Part II: Core Framework

> **In This Part:** Master the building blocks of GoRAI—NATS messaging, sensors, actuators, vision, services, behaviors, and coordinators.

## Overview

Part II is the heart of the book. Here you'll learn every component type in GoRAI, how they communicate over NATS, and how to compose them into working robot systems.

We start with NATS because it's the communication backbone that everything else depends on. Then we work through components from simple (sensors) to complex (coordinators), building your understanding layer by layer.

## Chapters in This Part

| Chapter | Title | What You'll Learn |
|---------|-------|-------------------|
| [Chapter 3](ch03-nats/_index.md) | NATS Messaging | Pub/sub, request/reply, QoS, JetStream |
| [Chapter 4](ch04-sensors/_index.md) | Sensors | The Sensor interface, built-in types, data types, fakes |
| [Chapter 5](ch05-actuators/_index.md) | Actuators | Motors, servos, bases, arms, control patterns |
| [Chapter 6](ch06-vision/_index.md) | Vision | Cameras, image types, data flow, OpenCV integration |
| [Chapter 7](ch07-services/_index.md) | Services | Vision service, navigation, SLAM, custom services |
| [Chapter 8](ch08-behaviors/_index.md) | Behaviors | Robot decision making, state machines, reactive behaviors |
| [Chapter 9](ch09-coordinators/_index.md) | Coordinators | Mission orchestration, multi-robot coordination |

## Key Concepts Introduced

By the end of Part II, you'll understand:

- **NATS fundamentals**: Topics, services, actions, wildcards, JetStream
- **Quality of Service**: BestEffort, Reliable, Retained, History
- **Component interfaces**: Sensor, Motor, Camera, Servo, Gripper, Base, Arm
- **Service patterns**: When to use services vs components
- **Behavioral architecture**: How robots make decisions
- **Coordination patterns**: Orchestrating complex multi-step tasks

## Prerequisites

Part II assumes you've read Part I, particularly:
- Understanding of nodes and resources (Chapter 2)
- The three-layer architecture concept
- Basic familiarity with distributed systems concepts

## Reading Order

The chapters are designed to be read in order, but you can skip around if you're focused on specific component types:

- **Building sensor-based projects?** Focus on Chapters 3, 4, and 7
- **Building mobile robots?** Focus on Chapters 3, 5, and 8-9
- **Building vision systems?** Focus on Chapters 3, 6, and 7

<!-- book-only -->
*Chapter 3 (NATS) is essential—don't skip it. All other chapters reference NATS concepts heavily.*
<!-- /book-only -->

<!-- website-only -->
!!! note "Chapter Dependencies"
    Chapter 3 (NATS Messaging) is foundational. Read it before diving into specific component chapters.
<!-- /website-only -->
