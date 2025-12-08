## 1.3 Who Should Use GoRAI

GoRAI isn't trying to be everything to everyone. It's designed for a specific kind of developer and a specific kind of project.

### You Should Use GoRAI If You're...

**Building new, modern robotics projects.** GoRAI shines on greenfield projects where you're not constrained by existing code. If you're starting a new robot from scratch, GoRAI lets you move fast without inheriting technical debt.

**Not dependent on ROS ecosystem packages.** The ROS ecosystem has thousands of packages—SLAM algorithms, navigation stacks, manipulation libraries. If your project critically depends on specific ROS packages with no alternatives, staying in ROS makes sense. But if you need standard capabilities (sensor interfaces, motor control, basic vision), GoRAI provides clean implementations without the baggage.

**Open to experimentation.** GoRAI is young. APIs may evolve. Best practices are still emerging. If you need a framework certified for production medical robots today, look elsewhere. If you're excited to shape a framework's future while building your robot, welcome aboard.

**Preferring Go's simplicity to C++ complexity.** If you love template metaprogramming and consider CMake a reasonable build system, GoRAI might feel constrained. But if you've ever spent an afternoon debugging a segfault or wrestling with linking errors, Go's guardrails are liberating.

**Valuing extensibility and performance.** GoRAI's architecture makes adding new components straightforward. Its Go foundation means you get native performance without unsafe memory access. When you need more speed, the profiler tells you exactly where, and optimization is tractable.

**Interested in AI-assisted development.** GoRAI's codebase is designed to work well with AI coding assistants. Clear interfaces, consistent patterns, and comprehensive specifications mean AI tools can help write components, generate tests, and explain behavior. This isn't just documentation—it's a development philosophy.

**Targeting Linux-based robot compute.** GoRAI runs on Linux: Raspberry Pi, Jetson, Orange Pi, or any ARM or x86 board. It doesn't require ROS's specific Ubuntu LTS versions—any modern Linux works. If your primary compute is Windows or macOS, GoRAI isn't the right choice.

**Wanting to use TinyGo for microcontrollers.** For low-level hardware—motor drivers, sensor interfaces, real-time control—GoRAI supports TinyGo on microcontrollers. Same language on your Raspberry Pi brain and your RP2040 motor controller. Same patterns, same skills.

### GoRAI is Not For You If...

**You need certified, production-ready software today.** GoRAI is under active development. It hasn't been validated for safety-critical applications. Medical robots, autonomous vehicles on public roads, industrial automation with human safety implications—these deserve mature, certified frameworks.

**You need specific ROS packages.** If your project depends on MoveIt for manipulation, Nav2 for navigation, or specific SLAM implementations only available in ROS, the switching cost is too high. GoRAI will eventually have equivalents, but "eventually" doesn't help today.

**Your team is deeply invested in ROS/ROS 2.** Migration costs are real. If your team knows ROS inside and out, has years of custom packages, and a deployment pipeline that works, the productivity gain from GoRAI may not justify retraining.

**You need hard real-time guarantees.** Go's garbage collector, while excellent, introduces unpredictable pauses. For microsecond-level control loops (some motor commutation, force control), dedicated real-time systems are appropriate. GoRAI works alongside these systems—the serial gateway pattern connects TinyGo microcontrollers for real-time tasks—but it doesn't replace them.
