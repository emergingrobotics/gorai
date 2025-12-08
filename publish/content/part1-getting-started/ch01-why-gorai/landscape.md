# Chapter 1: Why GoRAI?

## 1.1 The Robotics Software Landscape

The world of robotics software has evolved dramatically over the past two decades. What began as custom, hand-rolled solutions for individual robots has grown into a rich ecosystem of frameworks, libraries, and platforms. Yet despite this maturation, building robot software remains more difficult than it should be.

### A Brief History

**ROS (Robot Operating System)** emerged from Stanford and Willow Garage in 2007, becoming the de facto standard for research robotics. Its publish-subscribe architecture, standardized message types, and vast package ecosystem revolutionized how robots were built. However, ROS was designed for a different era—single robots, research labs, and developers comfortable with C++ build systems.

**ROS 2** arrived in 2017 to address ROS's limitations. Built on DDS (Data Distribution Service), it brought real-time capabilities, better security, and multi-robot support. But ROS 2 also brought complexity: multiple DDS implementations to choose from, a steep learning curve, and build times that can stretch into hours.

**YARP (Yet Another Robot Platform)** took a different approach, focusing on middleware for humanoid robots. It excels at connecting heterogeneous systems but requires significant investment to master its idioms and patterns.

**Viam** represents the modern cloud-connected approach: a managed platform where robots connect to cloud services for configuration, monitoring, and ML inference. It's elegant but introduces cloud dependencies that not every robot application can accept.

### Common Pain Points

After years of working with these platforms, recurring frustrations emerge:

**C++ Complexity**: ROS and ROS 2 are fundamentally C++ frameworks. While Python bindings exist, performance-critical code requires C++. This means grappling with CMake, colcon, header dependencies, and compilation times measured in tens of minutes. Memory safety issues lurk in every pointer.

**Python Performance Limitations**: Many teams escape to Python for faster development, only to hit walls when their control loops can't keep up or their image processing saturates a single core. The "prototype in Python, rewrite in C++" cycle wastes enormous effort.

**Heavy Framework Overhead**: Modern ROS 2 installations consume gigabytes. Starting a simple node pulls in layers of middleware. The abstraction cost—both in binary size and mental overhead—grows with each release.

**Steep Learning Curves**: New developers face months of ramp-up time. Understanding launch files, parameter servers, lifecycle management, QoS profiles, and the interaction between nodes requires dedicated study. Documentation assumes familiarity with concepts that aren't explained.

**Build System Complexity**: colcon, CMake, ament, package.xml, setup.py—the tooling stack has grown organically and shows it. Cross-compilation for embedded targets requires arcane knowledge. Reproducible builds demand containerization.

### The Gap GoRAI Fills

These pain points aren't inevitable. They reflect choices made in different contexts—academic research, enterprise middleware, cloud platforms—that don't always align with building practical robots.

What if we started fresh? What if we took the best ideas from distributed systems and cloud computing, combined them with Go's simplicity and performance, and designed specifically for modern robotics development?

That's the question GoRAI answers.
