# ROS 2 Design Summary

This document summarizes the architecture and design of ROS 2 (Robot Operating System 2) based on the official documentation.

## Overview

ROS 2 is a set of software libraries and tools for building robot applications. It provides a strongly-typed, anonymous publish/subscribe system built on top of DDS (Data Distribution Service) middleware, with pluggable middleware support.

## Core Concepts

### Nodes

A **node** is the fundamental unit of computation in ROS 2. Each node should do one logical thing and can:

- Publish to named topics to deliver data to other nodes
- Subscribe to named topics to receive data from other nodes
- Act as a service client or server for request/response communication
- Act as an action client or server for long-running tasks
- Provide configurable parameters

Nodes can communicate within the same process, across processes, or across machines. Connections between nodes are established through automatic distributed discovery.

### Communication Patterns

#### Topics (Publish/Subscribe)

Topics provide continuous data streams using a publish/subscribe model:

- **Publishers** send data to a named topic
- **Subscribers** receive data from a topic
- Multiple publishers and subscribers can exist on any topic
- Anonymous: subscribers don't know/care which publisher sent data
- **Strongly-typed**: Message types are enforced at multiple levels

#### Services (Request/Response)

Services provide synchronous remote procedure calls:

- A **service server** accepts requests and returns responses
- A **service client** sends requests and waits for responses
- One server per service name; multiple clients allowed
- Designed for short-running computations

#### Actions (Long-Running Tasks)

Actions handle long-running tasks with feedback and cancellation:

- An **action server** performs the task, provides feedback, handles cancellation
- An **action client** initiates the action and monitors progress
- Three-part message: request, response, and feedback
- One server per action name; multiple clients allowed

### Interfaces (Message Types)

ROS 2 uses an Interface Definition Language (IDL) to describe data structures:

- **.msg files**: Define message structures for topics
- **.srv files**: Define service request/response pairs (separated by `---`)
- **.action files**: Define action goal/result/feedback (separated by `---`)

Built-in types include: `bool`, `byte`, `char`, `float32`, `float64`, `int8/16/32/64`, `uint8/16/32/64`, `string`, `wstring`, plus arrays (static, bounded, unbounded).

### Parameters

Parameters configure nodes at startup and runtime:

- Associated with individual nodes
- Types: `bool`, `int64`, `float64`, `string`, `byte[]`, and arrays of these
- Nodes declare parameters they accept (type-safe)
- Three callback types: pre-set, set (for validation), post-set
- Exposed via services: get, set, list, describe parameters

### Discovery

Node discovery happens automatically through the underlying middleware:

1. Nodes advertise presence to others on the same ROS domain (ROS_DOMAIN_ID)
2. Periodic re-advertisement for new entities
3. Nodes advertise when going offline
4. Connections require compatible QoS settings

## Architecture Layers

### Client Libraries

Client libraries provide user-facing APIs in various languages:

- **rclcpp**: C++ client library (C++17)
- **rclpy**: Python client library
- **rclc**: C client library (for micro-ROS)
- Community libraries: Ada, Java/Android, .NET, Node.js, Rust, Flutter/Dart

All client libraries share common functionality through `rcl` (ROS Client Library) in C, ensuring consistent behavior across languages.

### Internal API Stack

```
User Code
    ↓
Client Library (rclcpp, rclpy, etc.)
    ↓
rcl (ROS Client Library - C API)
    ↓
rmw (ROS Middleware Interface)
    ↓
DDS/RTPS Implementation (or Zenoh)
```

- **rcl**: Common ROS functionality (discovery, graph events, etc.) independent of language
- **rmw**: Minimal middleware abstraction layer for different DDS vendors
- **rosidl**: Interface definition and code generation

### Type Support

Two approaches for handling message types:

1. **Static Type Support**: Generated code for each message type per vendor; more efficient but vendor-specific
2. **Dynamic Type Support**: Generic functions using type introspection; more portable but slower; requires DDS-XTypes

## Middleware Layer

### DDS Integration

ROS 2 primarily uses DDS (Data Distribution Service) for transport:

- Multiple DDS implementations supported: Eclipse Cyclone DDS, eProsima Fast DDS, RTI Connext DDS, GurumDDS
- Middleware can be selected at runtime
- DDS provides: discovery, pub/sub, request/reply, serialization

### Zenoh Support

As an alternative to DDS:

- Uses zenoh-c bindings
- Requires Zenoh router (zenohd) for discovery
- Maps ROS concepts to Zenoh primitives (publishers, subscribers, queryables)

### Quality of Service (QoS)

QoS policies tune communication behavior:

- **History**: Keep last N samples or keep all
- **Depth**: Queue size for "keep last"
- **Reliability**: Best effort or reliable delivery
- **Durability**: Volatile or transient local (latching)
- **Deadline**: Expected maximum time between messages
- **Lifespan**: Maximum message age before expiration
- **Liveliness**: Automatic or manual health indication
- **Lease Duration**: Maximum time before considered dead

Predefined profiles: Default, Services, Sensor Data, Parameters, System Default.

QoS compatibility uses "Request vs Offered" model - connections only made if subscriber's requested QoS is compatible with publisher's offered QoS.

## Execution Model

### Executors

Executors manage callback invocation:

- **SingleThreadedExecutor**: One thread processes all callbacks
- **MultiThreadedExecutor**: Configurable thread pool for parallel processing

### Callback Groups

Organize callbacks for execution control:

- **Mutually Exclusive**: Callbacks cannot run in parallel
- **Reentrant**: Callbacks may run in parallel

Callbacks in different groups may always run in parallel.

### Composition

Components allow flexible deployment:

- Write code as **Components** (similar to ROS 1 nodelets)
- Same API for in-process and inter-process communication
- Deploy-time choice: separate processes (isolation) or single process (efficiency)
- Component containers: `component_container`, `component_container_mt`, `component_container_isolated`
- Optional intra-process communication for zero-copy in same process

## Build System

### Package Structure

Every ROS 2 package contains:

- **package.xml**: Manifest with name, version, dependencies, build type
- Build files (CMakeLists.txt for C++, setup.py for Python)

### Build Tools

- **Build Tool**: CMake (C++), setuptools (Python)
- **Build Helpers**: ament packages (ament_cmake, ament_package, ament_lint)
- **Meta-Build Tool**: colcon - topologically orders and builds packages

### Ament

Ament provides CMake infrastructure for ROS 2:

- `ament_cmake`: Core CMake functions and macros
- `ament_cmake_auto`: Automatic dependency handling
- Environment hooks for workspace setup
- Resource indexing for efficient package discovery
- Symbolic link installation (replaces catkin devel space)

## Launch System

Launch files automate running multiple nodes:

- Formats: XML, YAML, or Python
- Configure: what to run, where, arguments, remappings
- Monitor process state and react to changes
- Execute with `ros2 launch`

## Security

ROS 2 includes DDS-Security integration:

- **Security Enclave**: Policy container for one or more nodes
- **Identity files**: CA certificate, node certificate, private key
- **Permissions files**: Permissions CA, signed governance, signed permissions
- Enables: encryption, authentication, access control
- Environment variables: `ROS_SECURITY_ENABLE`, `ROS_SECURITY_STRATEGY`

## Command-Line Tools

Extensible CLI framework (`ros2` command):

- `ros2 node`: List, info about nodes
- `ros2 topic`: List, echo, publish to topics
- `ros2 service`: Call services
- `ros2 action`: Interact with actions
- `ros2 param`: Get/set parameters
- `ros2 bag`: Record/playback messages
- `ros2 launch`: Run launch files
- `ros2 component`: Load/unload components

## Key Differences from ROS 1

1. **DDS-based middleware**: Supports multiple vendors, QoS policies, security
2. **No roscore**: Decentralized discovery
3. **Unified API**: Components replace nodes vs nodelets distinction
4. **Quality of Service**: Fine-grained communication tuning
5. **Lifecycle nodes**: Managed state transitions
6. **Security**: Built-in DDS-Security support
7. **Real-time support**: Designed for deterministic execution
8. **Multi-platform**: Linux, Windows, macOS
9. **Common client library core**: rcl ensures consistency across languages
10. **Stricter typing**: Parameters declare types upfront

## High-Level Packages

Notable packages available:

- **gazebo_ros_pkgs**: Gazebo simulation integration
- **navigation2**: Autonomous navigation stack
- **rosbag2**: Message recording and playback
- **RQt**: GUI tools
- **RViz2**: 3D visualization
- **image_transport**: Efficient image streaming
