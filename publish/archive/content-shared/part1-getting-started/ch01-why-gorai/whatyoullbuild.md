## 1.4 What You'll Build

This book is hands-on. By the end, you'll have built real, working robot components and understand GoRAI deeply enough to build your own.

### The Hello Sensor Example

Our primary teaching example is `hello-sensor`: a CPU temperature sensor that reads system thermal data and publishes it over NATS. It sounds simple, but it demonstrates everything you need to know:

- Creating a GoRAI node and connecting to NATS
- Implementing the `Sensor` interface
- Platform-specific code (Linux thermal zones, macOS system calls)
- Publishing Protocol Buffer messages
- Configuration and command-line flags
- Statistics collection and diagnostics
- Graceful shutdown and resource cleanup
- Fake implementations for testing

By Chapter 9, you'll understand every line of this example and be ready to adapt it for your own sensors.

### Along the Way

Each chapter builds practical skills:

**Chapter 2-3**: You'll run NATS, observe message flow, and understand how GoRAI's distributed architecture works in practice.

**Chapter 4-7**: You'll explore component interfaces—sensors, actuators, cameras, and services—understanding the contracts that make components interchangeable.

**Chapter 8**: You'll set up a complete development environment, from Go installation to hardware connections.

**Chapter 10**: You'll implement a custom component from scratch, following patterns established in the framework.

**Chapter 11**: You'll write tests at every level—unit tests with fakes, component tests with embedded NATS, integration tests across systems.

**Chapter 12**: You'll run ML inference on accelerated hardware, integrating AI capabilities into robot behaviors.

### What You Won't Build

This book focuses on foundations. We won't build:

- A complete autonomous robot (that's a book unto itself)
- Production navigation or SLAM systems
- Detailed manipulation pipelines
- Fleet management and cloud integration

These are important topics, but they build on the foundations this book establishes. Master the fundamentals here, and those advanced topics become tractable.

*Cross-reference: The complete hello-sensor implementation is covered in Chapter 9.*
