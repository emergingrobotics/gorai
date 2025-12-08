# Introduction

Welcome to *Gorai: Building Modern Robots with Go and NATS*.

*Pronounced "Go-ray-I" (rhymes with "samurai")*

This book is your comprehensive guide to building robots with the Gorai framework—a lightweight, Go-based alternative to ROS 2, YARP, and Viam optimized for AI.

## What is Gorai?

Gorai is a robotics framework that provides:

- **NATS-based messaging** for pub/sub, request/reply, and persistence
- **Protocol Buffer serialization** for type-safe, efficient communication
- **Resource-centric architecture** with unified component/service abstraction
- **First-class AI/ML support** with hardware acceleration (RK3588 NPU, NVIDIA CUDA)
- **Hot reconfiguration** without restart
- **TinyGo compatibility** for microcontroller deployment

## What You'll Learn

By the end of this book, you'll understand:

- The Gorai mental model and architecture
- NATS messaging patterns for robotics
- Building sensors, actuators, and vision components
- Testing strategies for robot software
- AI/ML integration with hardware acceleration
- AI-assisted development workflows

## Prerequisites

- Basic Go knowledge (variables, functions, structs, interfaces)
- Command-line familiarity
- Optional: Basic electronics understanding

## How This Book is Organized

The book is divided into four parts:

**Part 1: Getting Started** (Chapters 1-2)
- Why Gorai exists and the problems it solves
- The mental model and architecture

**Part 2: Core Framework** (Chapters 3-7)
- NATS messaging patterns
- Components: sensors, actuators, vision
- Services and higher-level abstractions

**Part 3: Development** (Chapters 8-11)
- Development environment setup
- Building the Hello Sensor example
- Custom components and testing

**Part 4: Advanced Topics** (Chapters 12-15)
- AI/ML integration
- Project organization
- AI-assisted development
- Conclusion and next steps

Let's build some robots.
