# Viam Architecture and Design Analysis

## Executive Summary

Viam is a modern, open-source robotics framework written in Go that provides a unified platform for building, configuring, and operating robots. It emphasizes modularity, extensibility, and distributed systems patterns inspired by cloud infrastructure. The architecture is built around a core concept of "Resources" (components and services) with a plugin/module system that enables custom functionality.

## Overall Architecture and Design Philosophy

### Core Principles

1. **Resource-Centric Design**: Everything in Viam is a resource—either a component (hardware/actuator) or a service (software capability). Resources are identified by hierarchical triplet names and APIs.

2. **Modular and Extensible**: The framework is designed to be extended through:
   - Custom APIs (protocol buffer-based)
   - Custom Models (implementations of APIs)
   - External Modules (separate processes communicating via gRPC)
   - Registry-based package distribution

3. **Configuration-Driven**: Robots are configured declaratively via JSON configuration files, enabling dynamic reconfiguration without restarting.

4. **Distributed Systems Approach**: Leverages patterns from cloud software:
   - gRPC for inter-process and remote communication
   - Unix domain sockets for local module communication
   - Remote robot support (robots can be composed from multiple remote robots)
   - WebRTC for secure tunneling and client connectivity

5. **Go-First Implementation**: Written in Go for performance, concurrency, and easy cross-platform compilation. Provides SDKs in Python, TypeScript, C++, and Rust.

6. **AI/ML Integration Ready**: Built-in support for vision, ML models, and various AI-powered services.

## Core Abstractions and Interfaces

### 1. Resource Interface

The fundamental building block in Viam:

```go
type Resource interface {
    Name() Name
    Reconfigure(ctx context.Context, deps Dependencies, conf Config) error
    DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error)
    Close(ctx context.Context) error
}
```

Key aspects:
- All resources must implement this minimal interface
- `Reconfigure` allows safe in-place updates without full restart
- `DoCommand` provides extensibility for arbitrary operations
- `Close` ensures proper cleanup

### 2. API Abstraction

APIs define the contract for a type of resource using a hierarchical triplet:
- **Namespace**: Identifies the organization (e.g., "rdk" for Viam, custom namespaces for third parties)
- **Type**: Either "component" or "service"
- **Subtype**: Specific resource type (e.g., "motor", "arm", "vision")

Example: `rdk:component:motor`, `acme:component:gizmo`

### 3. Model Abstraction

Models represent concrete implementations of APIs:
- **Triplet Format**: `namespace:family:name` (e.g., `rdk:builtin:gpio`)
- Models can be:
  - Built-in (part of RDK)
  - Local (on the host system)
  - Registry (downloaded and managed)

### 4. Composition Interfaces

**Robot Interface**: The root interface providing:
- Resource provider functionality
- Remote robot access
- Frame system operations (kinematic chains, spatial transforms)
- Operation and session management
- Cloud metadata and connectivity

**LocalRobot Interface**: Extends Robot with:
- Configuration management
- Reconfiguration capability
- Web server lifecycle
- Module management

**RemoteRobot Interface**: Specialized for remote connections
- Connection state management

### 5. Helper Interfaces

- **Sensor**: Resources with `Readings()` capability
- **Actuator**: Resources with `IsMoving()` and `Stop()` capabilities
- **Shaped**: Resources with geometric information via `Geometries()`

## Component Model

### Standard Components

Viam provides 23+ built-in component types organized hierarchically:

1. **Motion Components**:
   - Arm: Multi-joint manipulator with forward/inverse kinematics
   - Base: Wheeled/locomotion platform
   - Gripper: End-effector for grasping
   - Motor: Individual actuator
   - Servo: Controlled angular motion
   - Gantry: Linear motion system

2. **Sensing Components**:
   - Camera: Image acquisition
   - MovementSensor: IMU, GPS, odometry
   - Encoder: Position/velocity measurement
   - Sensor: Generic sensor (temperature, pressure, etc.)
   - Button: Binary input
   - PoseTracker: Position/orientation tracking

3. **Specialized Components**:
   - AudioIn/AudioOut: Audio devices
   - PowerSensor: Power monitoring
   - Generic: Catchall for custom components
   - Switch: Binary output control
   - Input: Input device abstraction

Each component has a:
- Go interface defining methods
- Protocol Buffer definition for gRPC
- Client implementation (for remote access)
- Server implementation (for RPC exposure)

### Component Lifecycle

1. **Registration**: Component models registered with the resource registry during initialization
2. **Configuration**: Declared in JSON config with name, API, model, and parameters
3. **Construction**: Resource manager creates instances via registered Create functions
4. **Reconfiguration**: Updates handled atomically when config changes
5. **Cleanup**: Proper shutdown via Close()

## Services (Higher-Level Capabilities)

Services are computational/algorithmic resources built on top of components:

### Built-in Services

1. **Motion**: Path planning and execution
   - Move, MoveOnMap, MoveOnGlobe operations
   - Constraint satisfaction
   - Collision avoidance via motion planning

2. **Vision**: Computer vision algorithms
   - Detection (object, classification)
   - Segmentation
   - Pose estimation
   - Custom vision models

3. **ML Model**: Inference service
   - Tensor-based inference API
   - Support for multiple ML frameworks
   - Metadata about model inputs/outputs

4. **SLAM**: Simultaneous localization and mapping
   - For autonomous navigation
   - Visual and lidar-based approaches

5. **Navigation**: High-level navigation
   - Waypoint-based navigation
   - Obstacle avoidance
   - Geospatial navigation

6. **Data Management**: Data collection and sync
   - Sensor data collection
   - Cloud synchronization
   - Configurable capture policies

7. **Discovery**: Service discovery
8. **Video**: Video streaming and processing
9. **Shell**: Remote command execution (for debugging)
10. **World State Store**: State persistence for multi-robot systems
11. **Base Remote Control**: Teleoperation

## Communication Patterns

### 1. gRPC-Based Communication

- **Protocol**: gRPC over HTTP/2
- **Definition**: All APIs defined in Protocol Buffers (external repo: viamrobotics/api)
- **Code Generation**: buf tool generates gRPC server stubs and client code
- **Bidirectional Streaming**: For continuous operations (video, streaming sensor data)

### 2. Transport Options

- **Direct gRPC**: Local or network communication
- **WebRTC**: Browser-based clients and secure tunneling through cloud
- **Unix Domain Sockets**: Local module communication
- **TCP Mode**: Alternative to Unix sockets on systems that don't support them (Windows)

### 3. RPC Client Pattern

For each components/service:
- Server-side RPC handler registers gRPC service
- Client implementation created from connection
- Transparent proxying of local resources as remote clients for composition

### 4. Session Management

- **Session Manager**: Tracks active client sessions
- **Operation Manager**: Manages concurrent operations with cancellation
- **Heartbeats**: Periodic connection checks to detect client disconnection

## Module/Plugin System

### Module Architecture

Modules are external processes that:
1. Implement custom resource models (components/services)
2. Communicate with parent via gRPC
3. Run in isolated processes with their own lifecycle
4. Are automatically started/stopped by the parent

### Module Types

1. **Local Modules**: Binary files on the host system
   - Path specified in config
   - Run immediately
   - Can be shell scripts or compiled binaries

2. **Registry Modules**: Downloaded from Viam's module registry
   - Identified by module ID
   - Automatically versioned and cached
   - Support for first-run setup scripts

### Module Communication

- **Unix Domain Sockets** (Linux/macOS): Low overhead local IPC
- **TCP Sockets** (Windows, or when VIAM_TCP_SOCKETS=true): TCP localhost connection
- **Protocol**: gRPC over these transports
- **Parent Address Sharing**: Parent provides its own address to modules for dependency resolution

### Module Lifecycle Manager

- Located in `/rdk/module/modmanager`
- Handles:
  - Module process spawning with environment variables
  - Log level configuration
  - Timeout management (startup, reconfiguration)
  - Dependency graph resolution
  - Module restart on failure
  - Data directory management per module

### Resource Resolution in Modules

- Modules can depend on parent robot resources
- Parent resources are transparently proxied to modules
- Dependency validation prevents circular dependencies
- Atomic reconfiguration of module resources

## Configuration Approach

### Configuration Structure

Robots are configured via JSON with sections for:

1. **Cloud**: Integration with Viam cloud platform
   - Org ID, location ID, machine credentials
   - Cloud connectivity settings

2. **Network**: Network configuration
   - Port binding
   - TLS settings
   - FQDN for remote access

3. **Modules**: External module specifications
   - Executable path or module ID
   - Configuration parameters per module
   - Environment variables
   - Log levels

4. **Remotes**: Remote robot connections
   - Robot addresses
   - Authentication credentials
   - Connection pooling settings

5. **Components**: Component declarations
   - Component name (unique identifier)
   - API and model
   - Configuration parameters (varies by component)
   - Dependencies on other components

6. **Services**: Service declarations
   - Service name
   - API and model
   - Configuration parameters

7. **Processes**: External processes to manage
   - For running auxiliary services

8. **Packages**: Viam packages management
   - Python packages, data files, etc.

9. **Auth**: Authentication configuration
   - JWT key sources
   - Role-based access control

10. **Logging**: Logging configuration
    - Per-logger settings
    - Log levels, outputs

11. **Jobs**: Scheduled jobs
    - Cron-like scheduling for periodic tasks

### Configuration Features

- **Hot Reload**: Configuration changes trigger reconfiguration without restart
- **Validation**: Pre-flight checks before applying config
- **Diff Tracking**: Config revisions tracked for auditing
- **Placeholders**: Environment variable substitution in config
- **Weak Dependencies**: Optional component dependencies
- **Cloud Sync**: Configuration synced from Viam cloud

### Reconfiguration Logic

When config changes:
1. Resource graph is analyzed for changes
2. Unchanged resources are reused
3. Changed resources are reconfigured in-place if supported
4. Unsupported changes trigger resource recreation
5. Dependencies are respected during reconfiguration order
6. Failures are handled with rollback semantics

## Resource Management

### Resource Graph

The heart of resource management:
- **Nodes**: Each resource is a graph node
- **Edges**: Dependencies between resources
- **Thread-Safe**: Concurrent access with mutex protection
- **Efficient Lookup**: Cached lookups by simple name and API
- **Version Tracking**: Logical clocks for update tracking

### Registry Pattern

- **Resource Registry**: Central registration of APIs and Models
- **Factory Functions**: Create and CreateRPCClient functions per model
- **Attribute Converters**: Custom JSON unmarshaling for complex configs
- **Reflection Support**: Proto reflection for runtime type introspection

### Dependency Resolution

- **Explicit Dependencies**: Components/services declare what they depend on
- **Type Safety**: Generic functions for safe dependency retrieval
- **Error Handling**: Clear errors when dependencies missing or wrong type
- **Circular Detection**: Prevents resource graph cycles

## Key Technologies and Dependencies

### Core Framework
- **Go 1.25.1**: Primary language
- **gRPC & Protocol Buffers**: RPC and serialization
- **gRPC Gateway**: REST/JSON endpoint generation
- **Protocol Reflection**: Runtime type introspection

### Robotics Specific
- **golang-geo**: Geospatial calculations
- **go-gl/mathgl**: Mathematical operations
- **go-nlopt**: Optimization (used in motion planning)
- **pointcloud libraries**: 3D data processing
- **FFmpeg integration**: Video processing

### Vision & ML
- **Gorgonia/tensor**: Tensor operations for ML
- **Custom vision modules**: Classification, detection, segmentation, keypoints
- **Object detection**: Built-in models support
- **Training data capture**: viscapture module

### Communication & Networking
- **pion/webrtc**: WebRTC for browser connectivity
- **pion/mediadevices**: Media device access
- **RTSP support**: gortsplib for video streaming
- **CORS support**: rs/cors for cross-origin requests

### DevOps & Observability
- **OpenTelemetry**: Distributed tracing
- **FTDC**: Full-time data capture for diagnostics
- **Prometheus metrics**: Performance monitoring
- **pprof**: CPU and memory profiling
- **Structured logging**: golog for consistent logging

### CLI & Development
- **urfave/cli**: Command-line interface
- **buf**: Protocol buffer management
- **go-git**: Git integration for module versioning
- **License finder**: License compliance

## How AI/ML is Integrated

### 1. Vision Service
- Object detection using TensorFlow/PyTorch models
- Classification of images
- Segmentation (semantic and instance)
- Custom vision modules support
- Camera streaming integration

### 2. ML Model Service
- Generic inference API for any tensor-based model
- Input tensor marshaling/unmarshaling
- Support for multiple tensor libraries (via gorgonia/tensor)
- Metadata API for model IO specifications
- Output post-processing hooks

### 3. Motion Planning with Constraints
- Path planning considering robot geometry
- Obstacle avoidance
- Kinematic chain solving for arms
- Trajectory generation with velocity profiles

### 4. Navigation Service
- SLAM support (simultaneous localization and mapping)
- Waypoint-based navigation
- Geospatial navigation on globe
- Integration with vision for obstacle detection

### 5. Data Capture for AI Training
- Data collection from sensors/cameras
- Structured data capture interface
- Cloud sync for training data
- Training data organization

### 6. Custom ML Models
- Modules can implement custom inference
- Vision service allows pluggable detectors
- Support for ONNX, TensorFlow, PyTorch through wrappers

## Agent (Management Layer)

The Viam Agent is a separate system service that:

1. **Self-Updates**: Downloads and installs new versions of viam-server
2. **Process Management**: Manages viam-server lifecycle
3. **Systemd Integration**: Runs as a systemd service on Linux
4. **Version Tracking**: Maintains version cache for rollback
5. **Installation**: Provides `--install` flag for setup

## Design Patterns and Best Practices

### 1. Interface-Driven Development
Every resource type has:
- An interface defining the contract
- Multiple implementations
- Remote client wrappers for transparent proxying

### 2. Dependency Injection
- Resources receive dependencies at construction
- Type-safe dependency lookup
- Optional weak dependencies for flexible composition

### 3. Context-Based Lifecycle
- All operations accept context.Context
- Proper cancellation propagation
- Timeouts enforced

### 4. Error Handling
- Specific error types for common issues
- Wrapped errors with context
- Pretty-printing for complex error chains

### 5. Testability
- Fake implementations for all services
- Mock components for testing
- In-process testing support

### 6. Logging
- Structured logging throughout
- Per-component logger instances
- Debug and info level support

### 7. Hot Reloadability
- Components can reconfigure in-place
- Web UI shows resource status
- Dynamic module loading/unloading

## Extensibility Points

### 1. Custom APIs (Highest Level)
Define new resource types:
- Protocol buffer definitions
- Go interfaces
- gRPC implementations

### 2. Custom Models
Implement existing APIs:
- Built-in models for Viam's components
- Custom models in modules

### 3. Modules
Run custom code:
- Any executable (Go, Python, Rust, etc.)
- Communicate via gRPC

### 4. Vision Detectors
Plug in custom ML models:
- Implement vision.Detector interface
- Registry-based model resolution

### 5. Motion Planning
Custom constraint solvers:
- Implement motion.Solver interface

### 6. Services
New high-level capabilities:
- Implement resource.Resource interface
- Register with API registry

## Performance Considerations

1. **Resource Graph Caching**: Efficient lookup structures
2. **Connection Pooling**: Reuse of gRPC connections
3. **Lazy Module Loading**: Modules started on first use
4. **Background Workers**: Non-blocking resource reconfiguration
5. **Metrics Collection**: Low-overhead telemetry
6. **Streaming**: Bidirectional gRPC for efficient communication

## Security Features

1. **TLS by Default**: Encrypted communication
2. **JWT Authentication**: Token-based access control
3. **WebRTC Tunneling**: Secure browser connectivity
4. **Module Sandboxing**: Isolated process execution
5. **Config Validation**: Pre-flight checks prevent invalid states
6. **Cloud Credentials**: Secure credential management

## Summary

Viam is a sophisticated robotics framework that:

1. **Unifies** hardware and software through a resource abstraction
2. **Enables** modular composition via gRPC and Protocol Buffers
3. **Supports** dynamic reconfiguration without downtime
4. **Integrates** AI/ML through vision and inference services
5. **Scales** from single robot to fleet management via cloud
6. **Extensible** at every level: components, services, APIs, modules
7. **Cloud-Native**: Follows distributed systems best practices
8. **Developer-Friendly**: Multiple language SDKs, clear abstractions

The design emphasizes composition, configuration, and extensibility—making it easy to build complex robotic systems by combining pre-built and custom components, while maintaining clean separation of concerns through the resource-centric architecture.

## Relevance to Gorai

Key patterns from Viam that align with Gorai's goals:

| Viam Pattern | Gorai Application |
|--------------|-------------------|
| Resource-centric design | Core abstraction for components/services |
| gRPC + Protocol Buffers | Consider NATS + Protocol Buffers or NATS + JSON |
| Module system (external processes) | Plugin architecture via NATS microservices |
| Configuration-driven | JSON config with hot reload |
| Go-first implementation | Aligned with Gorai's Go focus |
| AI/ML services | First-class AI integration goal |
| Hot reconfiguration | Dynamic robot updates |

### Differences to Consider

1. **NATS vs gRPC**: Gorai plans to use NATS extensively; Viam uses gRPC. NATS offers simpler pub/sub and request/reply patterns but may need additional structure for streaming.

2. **TinyGo Support**: Gorai targets TinyGo for microcontrollers; Viam's RDK requires full Go runtime.

3. **Cloud Dependency**: Viam has deep cloud integration; Gorai should consider standalone operation as primary.

4. **Simplicity**: Gorai aims for lower barrier to entry; can simplify Viam's patterns where appropriate.
