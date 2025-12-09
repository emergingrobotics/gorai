# The GoRAI Book

**Building Modern Robots with Go and NATS**

*By Greg Herlein & Luca Herlein*

---

This comprehensive guide takes you from first principles through advanced topics in building robotics software with GoRAI. Whether you prefer to read sequentially or jump to specific topics, you'll find everything you need here.

## Book Structure

<div class="grid cards" markdown>

-   :material-rocket-launch:{ .lg .middle } **Part I: Getting Started**

    ---

    Why GoRAI exists, core concepts, and the mental model for thinking about robot software as distributed systems.

    [:octicons-arrow-right-24: Start Here](part1-getting-started/_index.md)

-   :material-puzzle:{ .lg .middle } **Part II: Core Framework**

    ---

    Deep dive into NATS messaging, sensors, actuators, vision, services, behaviors, and coordinators.

    [:octicons-arrow-right-24: Core Concepts](part2-core-framework/_index.md)

-   :material-code-braces:{ .lg .middle } **Part III: Development**

    ---

    Hands-on development: environment setup, tutorials, building custom components, and testing strategies.

    [:octicons-arrow-right-24: Build Things](part3-development/_index.md)

-   :material-brain:{ .lg .middle } **Part IV: Advanced Topics**

    ---

    AI/ML integration, project organization, AI-assisted development, and what's next.

    [:octicons-arrow-right-24: Go Deeper](part4-advanced/_index.md)

</div>

## Quick Navigation

### Part I: Getting Started

| Chapter | Description |
|---------|-------------|
| [Why GoRAI?](part1-getting-started/ch01-why-gorai/_index.md) | The robotics landscape, design philosophy, and what you'll build |
| [Architecture](part1-getting-started/ch02-architecture/_index.md) | The big picture, core concepts, distributed systems thinking |

### Part II: Core Framework

| Chapter | Description |
|---------|-------------|
| [NATS Messaging](part2-core-framework/ch03-nats/_index.md) | Why NATS, fundamentals, patterns, QoS, JetStream |
| [Sensors](part2-core-framework/ch04-sensors/_index.md) | Sensor interface, built-in sensors, data types, fakes |
| [Actuators](part2-core-framework/ch05-actuators/_index.md) | Motors, servos, control patterns, bases and arms |
| [Vision](part2-core-framework/ch06-vision/_index.md) | Cameras, data flow, computer vision integration |
| [Services](part2-core-framework/ch07-services/_index.md) | Service architecture and implementation |
| [Behaviors](part2-core-framework/ch08-behaviors/_index.md) | Behavior-based robotics with GoRAI |
| [Coordinators](part2-core-framework/ch09-coordinators/_index.md) | Orchestrating complex robot behaviors |

### Part III: Development

| Chapter | Description |
|---------|-------------|
| [Dev Environment](part3-development/ch10-devenv/_index.md) | Setting up your development environment |
| [Hello Sensor](part3-development/ch11-hello-sensor/_index.md) | Step-by-step tutorial building your first sensor |
| [Custom Components](part3-development/ch12-custom/_index.md) | Building reusable custom components |
| [Testing](part3-development/ch13-testing/_index.md) | Testing strategies for robotics code |

### Part IV: Advanced

| Chapter | Description |
|---------|-------------|
| [AI/ML Integration](part4-advanced/ch14-ai-ml/_index.md) | Running ML models on edge hardware |
| [Project Organization](part4-advanced/ch15-organization/_index.md) | Structuring code that scales |
| [AI-Assisted Dev](part4-advanced/ch16-ai-dev/_index.md) | Using AI tools in robotics development |
| [Conclusion](part4-advanced/ch17-conclusion/_index.md) | What's next for you and GoRAI |

### Reference

| Section | Description |
|---------|-------------|
| [Appendices](appendices/_index.md) | NATS topics, Protocol Buffers, hardware, troubleshooting |
| [Examples](../examples/index.md) | Working code examples |

## How to Read This Book

=== "New to Robotics"

    Read sequentially from Part I through Part IV. Each chapter builds on the previous, introducing concepts progressively.

=== "Experienced with ROS"

    Skim Part I for philosophical differences, then dive into [Chapter 3: NATS](part2-core-framework/ch03-nats/_index.md) and [Chapter 11: Hello Sensor](part3-development/ch11-hello-sensor/_index.md). The patterns will feel familiar; the simplicity will feel liberating.

=== "Go Developer"

    Part I (architecture) and Part II (components) will orient you quickly. The code will feel natural; the domain concepts will be new.

=== "Just Want to Build"

    Start with [Chapter 10: Dev Environment](part3-development/ch10-devenv/_index.md) and [Chapter 11: Hello Sensor](part3-development/ch11-hello-sensor/_index.md). Get code running, then circle back to understand why it works.

## Download

Want to read offline?

- **PDF**: Coming soon
- **ePub**: Coming soon
- **Print on Demand**: Coming soon

---

*Pronounced "go-ray" (like "sting-ray")*
