# Why Gorai?

This chapter establishes the motivation, philosophy, and positioning of Gorai in the robotics software landscape.

## The Robotics Software Landscape

Robotics software has evolved through several generations:

- **ROS (2007)**: Pioneered the publish/subscribe model for robotics
- **ROS 2 (2017)**: Added DDS, real-time support, and security
- **YARP (2006)**: Focused on humanoid robotics with flexible transports
- **Viam (2020s)**: Modern Go-based framework with cloud integration

Each has strengths, but common pain points remain:

- C++ complexity and build systems (CMake, colcon)
- Python performance limitations at the edge
- Heavy framework overhead
- Steep learning curves
- Complex middleware abstractions

## The Gap Gorai Fills

Gorai addresses these challenges by:

- Using **Go** for simplicity, safety, and single-binary deployment
- Building on **NATS** for proven, lightweight messaging
- Prioritizing **edge AI** with first-class NPU/TPU support
- Maintaining a **low barrier to entry**
- Being **fun** to use

## Design Philosophy

Gorai follows these core principles:

1. **Go-first**: Leverage Go's simplicity, concurrency, and deployment model
2. **NATS-native**: Use battle-tested messaging infrastructure from cloud computing
3. **AI-optimized**: First-class support for edge inference on NPUs and TPUs
4. **Modular by default**: Loose coupling through messaging
5. **Low barrier to entry**: Get started in minutes, not days

## Who Should Use Gorai

Gorai is designed for developers who:

- Are building new, modern robotics projects
- Don't need ROS interoperability
- Are open to experimentation
- Want Go's simplicity over C++ complexity
- Value extensibility and performance
- Want to use AI-assisted development
- Are targeting Linux-based robot compute
- Want TinyGo for microcontrollers
