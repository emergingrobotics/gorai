---
title: "Gorai"
description: "A lightweight, Go-based robotics framework built on NATS.io"
---

# Gorai

**A lightweight, Go-based robotics framework built on NATS.io**

*Pronounced "go-ray" (like "sting-ray")*

---

## What is Gorai?

Gorai is a modern robotics framework that provides:

- **NATS-based messaging** for pub/sub, request/reply, and persistence
- **Go-native design** with single-binary deployment
- **Component architecture** for sensors, actuators, and services
- **First-class AI/ML support** with hardware acceleration
- **TinyGo compatibility** for microcontrollers

## Quick Start

```bash
# Install Gorai CLI
go install github.com/gorai/gorai/cmd/gorai@latest

# Create a new project
gorai init my-robot

# Run the example
cd my-robot && go run .
```

## Why Gorai?

| Aspect | Gorai | ROS 2 | Viam |
|--------|-------|-------|------|
| **Language** | Go + TinyGo | C++/Python | Go |
| **Middleware** | NATS | DDS | gRPC |
| **Deployment** | Single binary | Complex | Cloud-dependent |
| **Learning curve** | Low | High | Medium |

## Get Started

- **[Installation](/docs/getting-started/installation/)** — Set up your development environment
- **[Quick Start](/docs/getting-started/quickstart/)** — Build your first component
- **[Core Concepts](/docs/getting-started/concepts/)** — Understand the architecture

## Resources

- **[Documentation](/docs/)** — Guides and reference
- **[Examples](/examples/)** — Working code samples
- **[The Book](/book/)** — Comprehensive tutorial (PDF/ePub)
- **[GitHub](https://github.com/gorai/gorai)** — Source code and issues
