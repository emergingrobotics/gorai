# Why Gorai?

Before diving into code, let's establish *why* Gorai exists. The robotics software world isn't lacking options—ROS 2, YARP, Viam, and others all have their place. So why create something new?

This chapter answers that question by examining what existing frameworks do well, where they fall short, and how Gorai addresses those gaps. By the end, you'll understand Gorai's design philosophy and be able to judge whether it's the right tool for your project.

## The Robotics Software Landscape

The world of robotics software has evolved dramatically over the past two decades. What began as custom, hand-rolled solutions for individual robots has grown into a rich ecosystem of frameworks, libraries, and platforms. Yet despite this maturation, building robot software remains more difficult than it should be.

### A Brief History

**ROS (Robot Operating System)** emerged from Stanford and Willow Garage in 2007, becoming the de facto standard for research robotics. Its publish-subscribe architecture, standardized message types, and vast package ecosystem revolutionized how robots were built. However, ROS was designed for a different era—single robots, research labs, and developers comfortable with C++ build systems.

**ROS 2** arrived in 2017 to address ROS's limitations. Built on DDS (Data Distribution Service), it brought real-time capabilities, better security, and multi-robot support. But ROS 2 also brought complexity: multiple DDS implementations to choose from, a steep learning curve, and build times that can stretch into hours.

**YARP (Yet Another Robot Platform)** took a different approach, focusing on middleware for humanoid robots. It excels at connecting heterogeneous systems but requires significant investment to master its idioms and patterns.

**Viam** represents the modern cloud-connected approach: a managed platform where robots connect to cloud services for configuration, monitoring, and ML inference. It's elegant but introduces cloud dependencies that not every robot application can accept.

### Framework Comparison

Gorai learns from three generations of robotics middleware. Here's how they compare:

| Aspect | Gorai | ROS 2 | Viam | YARP |
|--------|-------|-------|------|------|
| **Language** | Go + TinyGo | C++/Python | Go | C++ |
| **Middleware** | NATS | DDS | gRPC | Custom carriers |
| **Discovery** | NATS (embedded/cluster) | DDS multicast | Cloud/local | Name server |
| **Build** | Go modules | CMake + ament + colcon | Go modules | CMake |
| **AI/ML** | First-class + TPU/NPU | Package ecosystem | First-class services | Minimal |
| **MCU Support** | TinyGo | micro-ROS | None | None |
| **Cloud** | Optional | Ecosystem | Core feature | None |
| **License** | Apache 2.0 | Apache 2.0 | AGPL | BSD-3 |

This table captures technical choices, but the real differences are in philosophy and daily experience.

### Common Pain Points

After years of working with these platforms, recurring frustrations emerge:

**C++ Complexity**: ROS and ROS 2 are fundamentally C++ frameworks. While Python bindings exist, performance-critical code requires C++. This means grappling with CMake, colcon, header dependencies, and compilation times measured in tens of minutes. Memory safety issues lurk in every pointer.

**Python Performance Limitations**: Many teams escape to Python for faster development, only to hit walls when their control loops can't keep up or their image processing saturates a single core. The "prototype in Python, rewrite in C++" cycle wastes enormous effort.

**Heavy Framework Overhead**: Modern ROS 2 installations consume gigabytes. Starting a simple node pulls in layers of middleware. The abstraction cost—both in binary size and mental overhead—grows with each release.

**Steep Learning Curves**: New developers face months of ramp-up time. Understanding launch files, parameter servers, lifecycle management, QoS profiles, and the interaction between nodes requires dedicated study. Documentation assumes familiarity with concepts that aren't explained.

**Build System Complexity**: colcon, CMake, ament, package.xml, setup.py—the tooling stack has grown organically and shows it. Cross-compilation for embedded targets requires arcane knowledge. Reproducible builds demand containerization.

### The Gap Gorai Fills

These pain points aren't inevitable. They reflect choices made in different contexts—academic research, enterprise middleware, cloud platforms—that don't always align with building practical robots.

What if we started fresh? What if we took the best ideas from distributed systems and cloud computing, combined them with Go's simplicity and performance, and designed specifically for modern robotics development?

That's the question Gorai answers.

## Design Principles

Before diving into specifics, here's a summary of what Gorai adopts from existing frameworks, what it does differently, and what it deliberately avoids.

### What We Adopt

- **Resource-centric model** (from Viam): Unified abstraction for components and services
- **Named addressing** (from all): Hierarchical, human-readable identifiers
- **Transport abstraction** (from YARP): NATS as unified transport for pub/sub and request/reply
- **Configuration-driven** (from Viam): JSON config with hot reload
- **Device interfaces** (from all): Clean separation of hardware from logic
- **NWS/NWC pattern** (from YARP): Transparent local/remote resource access

### What We Differentiate

- **NATS as core**: Simpler than DDS, more capable than gRPC for pub/sub patterns
- **TinyGo support**: Unified language from microcontrollers to cloud
- **TPU/NPU focus**: Edge AI as primary concern, not afterthought
- **No cloud dependency**: Standalone-first, cloud-optional
- **Lower barrier**: Simpler than ROS 2, more flexible than Viam

### What We Avoid

- Heavy build systems that increase barrier to entry
- Mandatory cloud connectivity
- Complex middleware abstractions that leak implementation details
- Central coordinators as single points of failure

## Design Philosophy

Gorai isn't just another robotics framework—it's a deliberate set of choices about how robot software should be built. These principles guide every design decision.

### Developer Experience is Non-Negotiable

**Developer experience (DX) is as important as functionality.** A framework with powerful features that nobody can learn is a framework nobody uses. We aim for wide adoption and contributions—that only happens when developers enjoy using Gorai.

This means:

- **Fantastic documentation**: Clear, comprehensive, with working examples
- **Tutorials that teach**: Not just what, but why
- **Errors that help**: Clear messages that point to solutions
- **Patterns that compose**: Learn once, apply everywhere
- **Fast feedback loops**: Seconds to build, not minutes

We believe great documentation and tutorials are what lower the barrier to entry for new developers. Every design decision asks: "Can a newcomer understand this?"

### Go-First

Go was designed at Google to solve exactly the problems that plague robotics development: C++ complexity, build system nightmares, and dependency hell. It compiles to native binaries in seconds, not minutes. It has built-in concurrency primitives that match how robots actually work—many things happening at once. It produces single, statically-linked binaries that deploy trivially.

```go
// A complete Gorai node in ~20 lines
func main() {
    n, _ := node.New("my_robot", node.WithNATS("nats://localhost:4222"))
    defer n.Close()

    pub := pub.New[*sensor.Temperature](n, "sensors.temp")

    for reading := range readSensor() {
        pub.Publish(context.Background(), reading)
    }
}
```

Go's type system catches errors at compile time without the ceremony of C++. Its garbage collector eliminates memory leaks without runtime overhead that matters for robotics. Its tooling—`go build`, `go test`, `go mod`—just works.

For microcontrollers, TinyGo brings the same language to resource-constrained devices. Write your robot's brain in Go, write your motor controller in Go—same language, same patterns, same mental model.

### NATS-Native

While ROS 2 chose DDS—a complex enterprise middleware with multiple competing implementations—Gorai builds on NATS, a messaging system designed for cloud-native applications.

NATS brings:

- **Simplicity**: A single binary, zero configuration to start
- **Performance**: Millions of messages per second on modest hardware
- **Flexibility**: Pub/sub, request/reply, and streaming in one system
- **JetStream**: Persistence when you need it, fire-and-forget when you don't
- **Clustering**: Built-in distribution across nodes and networks

NATS was battle-tested at companies processing billions of messages daily before Gorai adopted it. That operational maturity matters when your robot needs to work reliably.

### AI-Optimized

Modern robots increasingly rely on ML inference—object detection, pose estimation, voice recognition, path planning. Gorai treats AI as a first-class capability rather than an afterthought.

The acceleration layer provides a unified interface across different hardware:

- **NPU**: Rockchip RK3588's 6 TOPS neural processing unit
- **GPU**: NVIDIA CUDA for Jetson platforms
- **TPU**: Google Coral edge TPU
- **CPU**: Optimized fallback that works everywhere

Load a model, run inference, get results—the same code works whether you're on a laptop testing or deployed on edge hardware:

```go
acc, _ := rknn.New()
model, _ := acc.Load(ctx, "yolov5s.rknn")
outputs, _ := model.Infer(ctx, inputs)
```

### Modular by Default

Gorai components communicate through messages, not method calls. This isn't just architecture astronautics—it has practical consequences:

- **Hot swapping**: Replace a motor driver without restarting the navigation stack
- **Distributed deployment**: Run vision processing on a GPU node, control on a Pi
- **Testing**: Inject fake components without modifying production code
- **Monitoring**: Observe any data flow with standard NATS tools

Every component implements the same `Resource` interface. Every resource can be accessed locally or remotely with the same code. The system composes naturally.

### Low Barrier to Entry

Getting started with Gorai should take minutes, not days:

```bash
# Install Go (if needed)
# Install NATS (single binary)
# Clone and run
git clone https://github.com/gorai/gorai
cd gorai
go run ./examples/hello-sensor
```

No colcon builds. No CMake configuration. No ROS workspace setup. No Docker containers (unless you want them). The examples compile and run immediately.

### Fun!

This might seem frivolous, but it matters. Robotics should spark joy. When build systems frustrate and frameworks confuse, that joy disappears.

Gorai aims to bring back the fun: write code, see it run on your robot, iterate quickly, and spend your time solving robotics problems rather than fighting tools.

## Who Should Use Gorai

Gorai isn't trying to be everything to everyone. It's designed for a specific kind of developer and a specific kind of project.

### You Should Use Gorai If You're...

**Building new, modern robotics projects.** Gorai shines on greenfield projects where you're not constrained by existing code. If you're starting a new robot from scratch, Gorai lets you move fast without inheriting technical debt.

**Not dependent on ROS ecosystem packages.** The ROS ecosystem has thousands of packages—SLAM algorithms, navigation stacks, manipulation libraries. If your project critically depends on specific ROS packages with no alternatives, staying in ROS makes sense. But if you need standard capabilities (sensor interfaces, motor control, basic vision), Gorai provides clean implementations without the baggage.

**Open to experimentation.** Gorai is young. APIs may evolve. Best practices are still emerging. If you need a framework certified for production medical robots today, look elsewhere. If you're excited to shape a framework's future while building your robot, welcome aboard.

**Preferring Go's simplicity to C++ complexity.** If you love template metaprogramming and consider CMake a reasonable build system, Gorai might feel constrained. But if you've ever spent an afternoon debugging a segfault or wrestling with linking errors, Go's guardrails are liberating.

**Valuing extensibility and performance.** Gorai's architecture makes adding new components straightforward. Its Go foundation means you get native performance without unsafe memory access. When you need more speed, the profiler tells you exactly where, and optimization is tractable.

**Interested in AI-assisted development.** Gorai's codebase is designed to work well with AI coding assistants. Clear interfaces, consistent patterns, and comprehensive specifications mean AI tools can help write components, generate tests, and explain behavior.

**Targeting Linux-based robot compute.** Gorai runs on Linux: Raspberry Pi, Jetson, Orange Pi, or any ARM or x86 board. It doesn't require ROS's specific Ubuntu LTS versions—any modern Linux works.

**Wanting to use TinyGo for microcontrollers.** For low-level hardware—motor drivers, sensor interfaces, real-time control—Gorai supports TinyGo on microcontrollers. Same language on your Raspberry Pi brain and your RP2040 motor controller. Same patterns, same skills.

### Gorai is Not For You If...

**You need certified, production-ready software today.** Gorai is under active development. It hasn't been validated for safety-critical applications. Medical robots, autonomous vehicles on public roads, industrial automation with human safety implications—these deserve mature, certified frameworks.

**You need specific ROS packages.** If your project depends on MoveIt for manipulation, Nav2 for navigation, or specific SLAM implementations only available in ROS, the switching cost is too high. Gorai will eventually have equivalents, but "eventually" doesn't help today.

**Your team is deeply invested in ROS/ROS 2.** Migration costs are real. If your team knows ROS inside and out, has years of custom packages, and a deployment pipeline that works, the productivity gain from Gorai may not justify retraining.

**You need hard real-time guarantees.** Go's garbage collector, while excellent, introduces unpredictable pauses. For microsecond-level control loops (some motor commutation, force control), dedicated real-time systems are appropriate. Gorai works alongside these systems—the serial gateway pattern connects TinyGo microcontrollers for real-time tasks—but it doesn't replace them.

## What You'll Build

This book is hands-on. By the end, you'll have built real, working robot components and understand Gorai deeply enough to build your own.

### The Hello Sensor Example

Our primary teaching example is `hello-sensor`: a CPU temperature sensor that reads system thermal data and publishes it over NATS. It sounds simple, but it demonstrates everything you need to know:

- Creating a Gorai node and connecting to NATS
- Implementing the `Sensor` interface
- Platform-specific code (Linux thermal zones, macOS system calls)
- Publishing Protocol Buffer messages
- Configuration and command-line flags
- Statistics collection and diagnostics
- Graceful shutdown and resource cleanup
- Fake implementations for testing

By Chapter 12, you'll understand every line of this example and be ready to adapt it for your own sensors.

### Along the Way

Each chapter builds practical skills:

**Chapters 3**: You'll explore Gorai's architecture—the mental model for thinking about robot software as distributed systems.

**Chapter 4**: You'll run NATS, observe message flow, and understand how Gorai's communication backbone works in practice.

**Chapters 5-7**: You'll explore component interfaces—sensors, actuators, cameras—understanding the contracts that make components interchangeable.

**Chapters 8-10**: You'll learn about services, behaviors, and coordinators—higher-level abstractions for complex robot functionality.

**Chapter 11**: You'll set up a complete development environment, from Go installation to hardware connections.

**Chapter 12**: You'll implement the hello-sensor from scratch, following patterns established in the framework.

**Chapter 13**: You'll build custom components, extending Gorai for your specific needs.

**Chapter 14**: You'll write tests at every level—unit tests with fakes, component tests with embedded NATS, integration tests across systems.

**Chapters 15-17**: You'll explore advanced topics: AI/ML integration, project organization, and AI-assisted development.

## Prerequisites

Gorai is designed to be approachable, but some background knowledge will help you get the most from this book.

### Required: Basic Go Knowledge

You should be comfortable with Go fundamentals:

- **Variables and types**: `var`, `:=`, basic types (`int`, `string`, `float64`)
- **Functions**: Declaration, multiple return values, error handling
- **Structs**: Field definition, methods with receivers
- **Interfaces**: How they work, implicit satisfaction
- **Slices and maps**: Creation, access, iteration
- **Goroutines and channels**: Basic concurrent patterns
- **Packages and imports**: Go module structure

If you're new to Go, spend a few hours with the [Go Tour](https://go.dev/tour/) before diving in. The concepts translate quickly, especially if you know Python, JavaScript, or C.

### Required: Command-Line Familiarity

You'll spend time in the terminal:

- Navigating directories (`cd`, `ls`, `pwd`)
- Running commands with flags
- Understanding stdout, stderr, and exit codes
- Basic environment variables

Nothing exotic—if you've used a Unix-like terminal, you're prepared.

### Helpful: Networking Basics

Understanding helps but isn't required:

- IP addresses and ports
- TCP vs UDP (NATS uses TCP)
- Client-server vs peer-to-peer models
- What "localhost" means

The book explains what you need when you need it.

### Helpful: Basic Electronics/Hardware

If you want to connect real hardware:

- What GPIO, I2C, SPI, and UART mean
- How to read a pinout diagram
- Basic electrical safety (don't short 5V to ground)

For the first several chapters, you'll work with simulated and fake components. Hardware comes later, and we'll explain what you need.

### Development Environment

You'll need:

- A computer running Linux, macOS, or Windows (with WSL2 for Linux compatibility)
- Go 1.21 or later installed
- A text editor or IDE (VS Code with Go extension recommended)
- Git for cloning repositories
- Network access for downloading dependencies

Chapter 11 covers setup in detail. For now, confirm you can run `go version` and see output like `go version go1.22.0 linux/amd64`.

---

With these foundations in place, you're ready to understand how Gorai thinks about robotics. Chapter 3 introduces the architecture and mental model that makes everything else make sense.
