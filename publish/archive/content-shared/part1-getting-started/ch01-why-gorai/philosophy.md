## 1.2 Design Philosophy

GoRAI isn't just another robotics framework—it's a deliberate set of choices about how robot software should be built. These principles guide every design decision.

### Developer Experience is Non-Negotiable

**Developer experience (DX) is as important as functionality.** A framework with powerful features that nobody can learn is a framework nobody uses. We aim for wide adoption and contributions—that only happens when developers enjoy using GoRAI.

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
// A complete GoRAI node in ~20 lines
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

While ROS 2 chose DDS—a complex enterprise middleware with multiple competing implementations—GoRAI builds on NATS, a messaging system designed for cloud-native applications.

NATS brings:
- **Simplicity**: A single binary, zero configuration to start
- **Performance**: Millions of messages per second on modest hardware
- **Flexibility**: Pub/sub, request/reply, and streaming in one system
- **JetStream**: Persistence when you need it, fire-and-forget when you don't
- **Clustering**: Built-in distribution across nodes and networks

NATS was battle-tested at companies processing billions of messages daily before GoRAI adopted it. That operational maturity matters when your robot needs to work reliably.

### AI-Optimized

Modern robots increasingly rely on ML inference—object detection, pose estimation, voice recognition, path planning. GoRAI treats AI as a first-class capability rather than an afterthought.

The acceleration layer (`accel/`) provides a unified interface across different hardware:
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

GoRAI components communicate through messages, not method calls. This isn't just architecture astronautics—it has practical consequences:

- **Hot swapping**: Replace a motor driver without restarting the navigation stack
- **Distributed deployment**: Run vision processing on a GPU node, control on a Pi
- **Testing**: Inject fake components without modifying production code
- **Monitoring**: Observe any data flow with standard NATS tools

Every component implements the same `Resource` interface. Every resource can be accessed locally or remotely with the same code. The system composes naturally.

### Low Barrier to Entry

Getting started with GoRAI should take minutes, not days:

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

GoRAI aims to bring back the fun: write code, see it run on your robot, iterate quickly, and spend your time solving robotics problems rather than fighting tools.

### Community-Driven Development

GoRAI is built for contributors. We want you to:
- **File issues**: Found a bug? Confused by docs? Tell us.
- **Submit PRs**: New components, better examples, typo fixes—all welcome
- **Share your builds**: Inspire others with what you create
- **Improve documentation**: Every clarification helps the next developer

Great open-source projects grow through community contribution. We've designed GoRAI to be approachable: clear code structure, consistent patterns, comprehensive tests. Not just so you can use it—so you can contribute to it.

The barrier to your first contribution should be as low as the barrier to your first robot.
