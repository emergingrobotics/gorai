# Gorai Framework Specification

**Version 0.1.0**

A lightweight, Go-based robotics framework built on NATS.io with first-class AI/ML support.

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Resource Model](#resource-model)
4. [Communication Layer](#communication-layer)
5. [Topic Naming Convention](#topic-naming-convention)
6. [Protocol Buffer Definitions](#protocol-buffer-definitions)
7. [Core Components](#core-components)
8. [Device Interfaces](#device-interfaces)
9. [AI/ML Services](#aiml-services)
10. [Acceleration Layer](#acceleration-layer)
11. [Configuration System](#configuration-system)
12. [Network Transparency](#network-transparency)
13. [CLI Tool](#cli-tool)
14. [Directory Structure](#directory-structure)

---

## Overview

Gorai is a robotics framework providing:

- **NATS-based messaging** for pub/sub, request/reply, and persistence
- **Protocol Buffer serialization** for type-safe, efficient communication
- **Resource-centric architecture** with unified component/service abstraction
- **First-class AI/ML support** with hardware acceleration (RK3588 NPU, NVIDIA CUDA)
- **Hot reconfiguration** without restart
- **TinyGo compatibility** for microcontroller deployment
- **OCI container support** with Podman as the reference runtime

### Target Platform

Gorai targets **Linux-based systems**:

| Platform | Support Level |
|----------|---------------|
| Linux x86_64 | Primary |
| Linux ARM64 (Raspberry Pi, Rockchip, Jetson) | Primary |
| Microcontrollers via TinyGo | Primary |
| macOS | Development only |
| Windows | Not supported |

### Container Support

Gorai supports deployment via OCI-compliant containers:

| Runtime | Support Level | Notes |
|---------|---------------|-------|
| Podman | Reference | Daemonless, rootless-capable |
| Docker | Compatible | Via OCI compliance |
| Kubernetes | Compatible | Via OCI compliance |

Podman is the reference container runtime for Gorai due to:
- **Daemonless architecture**: No background service required
- **Rootless operation**: Run containers without root privileges
- **OCI compliance**: Images work with any OCI-compliant runtime
- **Pod support**: Native multi-container pod support

### Design Document Standard

Gorai uses detailed design documents as the specification format for components and services. See [hello-sensor-design.md](hello-sensor-design.md) for the canonical example.

A complete design document includes:

| Section | Purpose |
|---------|---------|
| **Overview** | Goals, use cases, design philosophy |
| **Architecture** | Component diagrams, data flow, NATS topic structure |
| **Protocol Buffers** | Complete `.proto` definitions with field documentation |
| **Implementation** | Package structure, platform-specific code, configuration |
| **Verification** | Step-by-step testing procedures with expected outputs |
| **Test Specification** | Unit test cases, integration test scenarios |

This level of detail serves two purposes:
1. **Human documentation**: Engineers can understand the component without reading source code
2. **AI implementation blueprint**: AI coding assistants can implement the design with minimal ambiguity

When adding new components or services, create a design document following this format before implementation.

---

## Architecture

### Layer Diagram

```mermaid
block-beta
    columns 1

    block:app["Application Layer"]
        columns 4
        Nodes Actions Services AI["AI/ML Services<br/>Vision | MLModel<br/>SLAM | Navigation"]
    end

    block:comm["Communication Layer"]
        columns 1
        NATS["NATS Messaging<br/>• Topics (Pub/Sub) • Request/Reply • JetStream • KV Store"]
    end

    block:resource["Resource Layer"]
        columns 5
        Motor Camera Sensor Arm Generic
    end

    block:accel["Acceleration Layer"]
        columns 4
        RockchipNPU["Rockchip NPU ✓"] CUDA["GPU/CUDA ✓"] CoralTPU["Coral TPU*"] HailoNPU["Hailo NPU*"]
    end

    block:hw["Hardware Layer"]
        columns 5
        GPIO I2C SPI Serial USB
    end
```

### Component Interaction

```mermaid
flowchart TB
    subgraph top["Processing Pipeline"]
        Camera["Camera Node"]
        Vision["Vision Node<br/>(ML Infer)"]
        Nav["Nav Node"]
        Camera --> Vision --> Nav
    end

    subgraph nats["NATS Server"]
        JS["JetStream: sensor.image, vision.detections, nav.goal"]
    end

    subgraph bottom["Consumers"]
        Motor["Motor Node"]
        SLAM["SLAM Node"]
        Logging["Logging Node"]
    end

    Camera --> nats
    Vision --> nats
    Nav --> nats
    nats --> Motor
    nats --> SLAM
    nats --> Logging
```

---

## Resource Model

### Resource Interface

All components and services implement the base Resource interface:

```go
package resource

import "context"

// Resource is the base interface for all Gorai components and services.
type Resource interface {
    // Name returns the unique resource identifier.
    Name() Name

    // Reconfigure updates the resource with new configuration.
    // Called during hot reload without full restart.
    Reconfigure(ctx context.Context, deps Dependencies, conf Config) error

    // DoCommand executes arbitrary commands for extensibility.
    DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error)

    // Close releases all resources.
    Close(ctx context.Context) error
}

// Name identifies a resource with hierarchical naming.
type Name struct {
    Namespace string // Organization namespace (e.g., "gorai", "mycompany")
    Type      string // "component" or "service"
    Subtype   string // Specific type (e.g., "motor", "camera", "vision")
    Name      string // Instance name (e.g., "left_motor", "front_camera")
}

// String returns the full resource name.
func (n Name) String() string {
    return fmt.Sprintf("%s:%s:%s/%s", n.Namespace, n.Type, n.Subtype, n.Name)
}

// Dependencies provides access to dependent resources.
type Dependencies interface {
    Get(name Name) (Resource, error)
    GetByType(subtype string) ([]Resource, error)
}

// Config holds resource configuration.
type Config struct {
    Attributes map[string]any
    Raw        []byte // Original JSON
}
```

### Resource Registry

Resources are registered with factory functions:

```go
package resource

// Model identifies a specific implementation.
type Model struct {
    Namespace string // e.g., "gorai"
    Family    string // e.g., "builtin"
    Name      string // e.g., "gpio"
}

// Creator is a factory function for resources.
type Creator func(ctx context.Context, deps Dependencies, conf Config) (Resource, error)

// Registry holds all registered resource types.
type Registry struct {
    // ...
}

// RegisterComponent registers a component model.
func (r *Registry) RegisterComponent(api API, model Model, creator Creator)

// RegisterService registers a service model.
func (r *Registry) RegisterService(api API, model Model, creator Creator)
```

### Helper Interfaces

Resources may implement additional capability interfaces:

```go
// Sensor represents resources that provide readings.
type Sensor interface {
    Resource
    Readings(ctx context.Context) (map[string]any, error)
}

// Actuator represents resources that can move.
type Actuator interface {
    Resource
    IsMoving(ctx context.Context) (bool, error)
    Stop(ctx context.Context) error
}

// Shaped represents resources with geometry.
type Shaped interface {
    Resource
    Geometries(ctx context.Context) ([]Geometry, error)
}

// Reconfigurable indicates in-place reconfiguration support.
type Reconfigurable interface {
    Resource
    Reconfigure(ctx context.Context, deps Dependencies, conf Config) error
}
```

---

## Communication Layer

### NATS Integration

Gorai uses NATS for all communication:

| Gorai Pattern | NATS Primitive | Persistence |
|---------------|----------------|-------------|
| Topic (pub/sub) | Core NATS Publish/Subscribe | Optional (JetStream) |
| Service (RPC) | Request/Reply | No |
| Action (long-running) | Request/Reply + Publish | Feedback via pub/sub |
| Parameter | KV Store | Yes (JetStream KV) |
| State | Object Store | Yes (JetStream Object) |

### Quality of Service

| QoS Level | NATS Implementation | Use Case |
|-----------|---------------------|----------|
| `BestEffort` | Core NATS | High-frequency sensor data |
| `Reliable` | JetStream with ack | Commands, state changes |
| `Retained` | JetStream last-value | Late subscriber catch-up |
| `History(n)` | JetStream limit | Replay for debugging |
| `Persistent` | JetStream durable | Data logging |

### Message Envelope

All messages are wrapped with metadata:

```protobuf
message Envelope {
    Header header = 1;
    string type_url = 2;      // e.g., "gorai.sensor.Image"
    bytes payload = 3;        // Serialized protobuf
    map<string, string> metadata = 4;
}
```

---

## Topic Naming Convention

### Hierarchy

```
gorai.{robot}.{node}.{topic}[.{suffix}]
```

| Segment | Description | Example |
|---------|-------------|---------|
| `gorai` | Framework prefix (fixed) | `gorai` |
| `{robot}` | Robot/machine identifier | `sentinel`, `skimmer` |
| `{node}` | Node name | `camera`, `motor_left`, `vision` |
| `{topic}` | Topic name | `image`, `cmd_vel`, `detections` |
| `{suffix}` | Optional qualifier | `raw`, `compressed`, `filtered` |

### Standard Topic Patterns

#### Sensor Topics

```
gorai.{robot}.{sensor}.data              # Primary sensor output
gorai.{robot}.{sensor}.data.raw          # Unprocessed data
gorai.{robot}.{sensor}.data.compressed   # Compressed variant
gorai.{robot}.{sensor}.info              # Sensor metadata/calibration
gorai.{robot}.{sensor}.diagnostics       # Health/status
```

Examples:
```
gorai.sentinel.camera_front.data.compressed    # JPEG images
gorai.sentinel.imu.data                        # IMU readings
gorai.sentinel.lidar.data                      # Point cloud
gorai.sentinel.gps.data                        # GPS fix
```

#### Control Topics

```
gorai.{robot}.{actuator}.command         # Incoming commands
gorai.{robot}.{actuator}.state           # Current state
gorai.{robot}.{actuator}.feedback        # Control feedback
```

Examples:
```
gorai.sentinel.drive.command             # Twist commands
gorai.sentinel.drive.state               # Odometry
gorai.sentinel.arm.command               # Joint commands
gorai.sentinel.arm.state                 # Joint states
```

#### AI/ML Topics

```
gorai.{robot}.vision.detections          # Object detections
gorai.{robot}.vision.classifications     # Image classifications
gorai.{robot}.vision.segmentation        # Segmentation masks
gorai.{robot}.mlmodel.{name}.input       # Model input
gorai.{robot}.mlmodel.{name}.output      # Model output
gorai.{robot}.slam.map                   # SLAM map updates
gorai.{robot}.slam.pose                  # Localized pose
gorai.{robot}.nav.path                   # Planned path
gorai.{robot}.nav.goal                   # Navigation goal
```

#### Service Subjects

```
gorai.{robot}.{node}.{service}.request   # Service request
gorai.{robot}.{node}.{service}.response  # Service response
```

Examples:
```
gorai.sentinel.camera.set_exposure.request
gorai.sentinel.arm.get_pose.request
gorai.sentinel.vision.detect.request
```

#### Action Subjects

```
gorai.{robot}.{node}.{action}.goal       # Action goal
gorai.{robot}.{node}.{action}.cancel     # Cancel request
gorai.{robot}.{node}.{action}.feedback   # Progress feedback
gorai.{robot}.{node}.{action}.result     # Final result
gorai.{robot}.{node}.{action}.status     # Action status
```

Examples:
```
gorai.sentinel.nav.navigate_to.goal
gorai.sentinel.nav.navigate_to.feedback
gorai.sentinel.arm.move_to_pose.goal
gorai.sentinel.arm.move_to_pose.result
```

#### System Topics

```
gorai.{robot}._system.nodes              # Active nodes
gorai.{robot}._system.heartbeat          # Node heartbeats
gorai.{robot}._system.logs               # Centralized logs
gorai.{robot}._system.diagnostics        # System diagnostics
gorai.{robot}._system.tf                 # Transform tree
```

### Wildcards

NATS wildcards for subscription:

| Pattern | Matches |
|---------|---------|
| `gorai.sentinel.>` | All topics for sentinel robot |
| `gorai.*.camera.>` | All camera topics on any robot |
| `gorai.sentinel.*.data` | All sensor data topics |
| `gorai.sentinel._system.*` | All system topics |

---

## Protocol Buffer Definitions

### Package Structure

```
api/proto/gorai/
├── std/
│   └── std.proto           # Common types (Header, Time, Duration)
├── geometry/
│   └── geometry.proto      # Spatial types (Vector3, Pose, Transform)
├── sensor/
│   └── sensor.proto        # Sensor messages (Image, PointCloud, IMU)
├── control/
│   └── control.proto       # Control messages (Twist, JointCommand)
├── vision/
│   └── vision.proto        # Vision types (Detection, Classification)
├── ml/
│   └── ml.proto            # ML types (Tensor, ModelMetadata)
├── nav/
│   └── nav.proto           # Navigation (Path, Waypoint, Map)
├── action/
│   └── action.proto        # Action protocol messages
└── buf.yaml
```

### std.proto - Common Types

```protobuf
syntax = "proto3";
package gorai.std;

option go_package = "github.com/gorai-robotics/gorai/api/std";

// Header contains metadata for all stamped messages.
message Header {
    // Nanoseconds since Unix epoch (1970-01-01 00:00:00 UTC).
    int64 timestamp_ns = 1;

    // Coordinate frame this data is associated with.
    string frame_id = 2;

    // Sequence number for ordering.
    uint32 seq = 3;
}

// Time represents a point in time.
message Time {
    int64 sec = 1;
    int32 nsec = 2;
}

// Duration represents a time span.
message Duration {
    int64 sec = 1;
    int32 nsec = 2;
}

// DiagnosticStatus represents component health.
message DiagnosticStatus {
    enum Level {
        OK = 0;
        WARN = 1;
        ERROR = 2;
        STALE = 3;
    }
    Level level = 1;
    string name = 2;
    string message = 3;
    string hardware_id = 4;
    map<string, string> values = 5;
}

// KeyValue for generic key-value pairs.
message KeyValue {
    string key = 1;
    string value = 2;
}
```

### geometry.proto - Spatial Types

```protobuf
syntax = "proto3";
package gorai.geometry;

option go_package = "github.com/gorai-robotics/gorai/api/geometry";

import "gorai/std/std.proto";

// Vector3 represents a 3D vector.
message Vector3 {
    double x = 1;
    double y = 2;
    double z = 3;
}

// Point is an alias for Vector3 representing position.
message Point {
    double x = 1;
    double y = 2;
    double z = 3;
}

// Quaternion represents rotation.
message Quaternion {
    double x = 1;
    double y = 2;
    double z = 3;
    double w = 4;
}

// Pose represents position and orientation.
message Pose {
    Point position = 1;
    Quaternion orientation = 2;
}

// PoseStamped is a Pose with header.
message PoseStamped {
    gorai.std.Header header = 1;
    Pose pose = 2;
}

// PoseWithCovariance includes uncertainty.
message PoseWithCovariance {
    Pose pose = 1;
    repeated double covariance = 2; // 6x6 row-major (36 elements)
}

// PoseWithCovarianceStamped is PoseWithCovariance with header.
message PoseWithCovarianceStamped {
    gorai.std.Header header = 1;
    PoseWithCovariance pose = 2;
}

// Twist represents linear and angular velocity.
message Twist {
    Vector3 linear = 1;
    Vector3 angular = 2;
}

// TwistStamped is a Twist with header.
message TwistStamped {
    gorai.std.Header header = 1;
    Twist twist = 2;
}

// TwistWithCovariance includes uncertainty.
message TwistWithCovariance {
    Twist twist = 1;
    repeated double covariance = 2; // 6x6 row-major
}

// Accel represents linear and angular acceleration.
message Accel {
    Vector3 linear = 1;
    Vector3 angular = 2;
}

// AccelStamped is Accel with header.
message AccelStamped {
    gorai.std.Header header = 1;
    Accel accel = 2;
}

// Wrench represents force and torque.
message Wrench {
    Vector3 force = 1;
    Vector3 torque = 2;
}

// WrenchStamped is Wrench with header.
message WrenchStamped {
    gorai.std.Header header = 1;
    Wrench wrench = 2;
}

// Transform represents a coordinate transformation.
message Transform {
    Vector3 translation = 1;
    Quaternion rotation = 2;
}

// TransformStamped is a Transform between two frames.
message TransformStamped {
    gorai.std.Header header = 1;
    string child_frame_id = 2;
    Transform transform = 3;
}

// Polygon is a 2D polygon.
message Polygon {
    repeated Point32 points = 1;
}

// Point32 is a 32-bit precision point.
message Point32 {
    float x = 1;
    float y = 2;
    float z = 3;
}

// Inertia represents moment of inertia.
message Inertia {
    double m = 1;       // Mass
    Vector3 com = 2;    // Center of mass
    double ixx = 3;
    double ixy = 4;
    double ixz = 5;
    double iyy = 6;
    double iyz = 7;
    double izz = 8;
}
```

### sensor.proto - Sensor Messages

```protobuf
syntax = "proto3";
package gorai.sensor;

option go_package = "github.com/gorai-robotics/gorai/api/sensor";

import "gorai/std/std.proto";
import "gorai/geometry/geometry.proto";

// Image encodings
enum ImageEncoding {
    ENCODING_UNKNOWN = 0;
    RGB8 = 1;
    RGBA8 = 2;
    BGR8 = 3;
    BGRA8 = 4;
    MONO8 = 5;
    MONO16 = 6;
    DEPTH16 = 7;      // 16-bit depth in mm
    DEPTH32F = 8;     // 32-bit float depth in meters
    BAYER_RGGB8 = 9;
    BAYER_BGGR8 = 10;
    BAYER_GBRG8 = 11;
    BAYER_GRBG8 = 12;
    YUV422 = 13;
    NV12 = 14;
    NV21 = 15;
}

// Image represents an uncompressed image.
message Image {
    gorai.std.Header header = 1;
    uint32 height = 2;
    uint32 width = 3;
    ImageEncoding encoding = 4;
    bool is_bigendian = 5;
    uint32 step = 6;              // Row length in bytes
    bytes data = 7;
}

// CompressedImage represents a compressed image.
message CompressedImage {
    gorai.std.Header header = 1;
    string format = 2;            // "jpeg", "png", "webp", "h264", "h265"
    bytes data = 3;
}

// CameraInfo contains camera calibration data.
message CameraInfo {
    gorai.std.Header header = 1;
    uint32 height = 2;
    uint32 width = 3;
    string distortion_model = 4;  // "plumb_bob", "rational_polynomial", "equidistant"
    repeated double d = 5;        // Distortion parameters
    repeated double k = 6;        // Intrinsic matrix (3x3 row-major)
    repeated double r = 7;        // Rectification matrix (3x3)
    repeated double p = 8;        // Projection matrix (3x4)
    uint32 binning_x = 9;
    uint32 binning_y = 10;
    RegionOfInterest roi = 11;
}

// RegionOfInterest defines a sub-region.
message RegionOfInterest {
    uint32 x_offset = 1;
    uint32 y_offset = 2;
    uint32 height = 3;
    uint32 width = 4;
    bool do_rectify = 5;
}

// PointField describes a field in a PointCloud2.
message PointField {
    enum Datatype {
        INT8 = 0;
        UINT8 = 1;
        INT16 = 2;
        UINT16 = 3;
        INT32 = 4;
        UINT32 = 5;
        FLOAT32 = 6;
        FLOAT64 = 7;
    }
    string name = 1;
    uint32 offset = 2;
    Datatype datatype = 3;
    uint32 count = 4;
}

// PointCloud2 represents a 3D point cloud.
message PointCloud2 {
    gorai.std.Header header = 1;
    uint32 height = 2;
    uint32 width = 3;
    repeated PointField fields = 4;
    bool is_bigendian = 5;
    uint32 point_step = 6;        // Bytes per point
    uint32 row_step = 7;          // Bytes per row
    bytes data = 8;
    bool is_dense = 9;            // True if no invalid points
}

// LaserScan represents a 2D laser scan.
message LaserScan {
    gorai.std.Header header = 1;
    float angle_min = 2;          // Start angle (rad)
    float angle_max = 3;          // End angle (rad)
    float angle_increment = 4;    // Angular resolution (rad)
    float time_increment = 5;     // Time between measurements
    float scan_time = 6;          // Time for full scan
    float range_min = 7;          // Minimum range (m)
    float range_max = 8;          // Maximum range (m)
    repeated float ranges = 9;    // Range data (m)
    repeated float intensities = 10;
}

// Range represents a single range measurement.
message Range {
    gorai.std.Header header = 1;
    enum RadiationType {
        ULTRASOUND = 0;
        INFRARED = 1;
        LIDAR = 2;
    }
    RadiationType radiation_type = 2;
    float field_of_view = 3;      // Radians
    float min_range = 4;
    float max_range = 5;
    float range = 6;
}

// Imu represents IMU data.
message Imu {
    gorai.std.Header header = 1;
    gorai.geometry.Quaternion orientation = 2;
    repeated double orientation_covariance = 3;     // 3x3
    gorai.geometry.Vector3 angular_velocity = 4;
    repeated double angular_velocity_covariance = 5; // 3x3
    gorai.geometry.Vector3 linear_acceleration = 6;
    repeated double linear_acceleration_covariance = 7; // 3x3
}

// MagneticField represents magnetometer data.
message MagneticField {
    gorai.std.Header header = 1;
    gorai.geometry.Vector3 magnetic_field = 2;      // Tesla
    repeated double magnetic_field_covariance = 3;  // 3x3
}

// NavSatFix represents GPS data.
message NavSatFix {
    gorai.std.Header header = 1;
    enum Status {
        STATUS_NO_FIX = 0;
        STATUS_FIX = 1;
        STATUS_SBAS_FIX = 2;
        STATUS_GBAS_FIX = 3;
    }
    enum Service {
        SERVICE_GPS = 1;
        SERVICE_GLONASS = 2;
        SERVICE_COMPASS = 4;
        SERVICE_GALILEO = 8;
    }
    Status status = 2;
    uint32 service = 3;           // Bitfield of Service
    double latitude = 4;          // Degrees
    double longitude = 5;         // Degrees
    double altitude = 6;          // Meters (WGS84)
    repeated double position_covariance = 7; // 3x3 ENU
    enum CovarianceType {
        COVARIANCE_TYPE_UNKNOWN = 0;
        COVARIANCE_TYPE_APPROXIMATED = 1;
        COVARIANCE_TYPE_DIAGONAL_KNOWN = 2;
        COVARIANCE_TYPE_KNOWN = 3;
    }
    CovarianceType position_covariance_type = 8;
}

// JointState represents joint positions/velocities/efforts.
message JointState {
    gorai.std.Header header = 1;
    repeated string name = 2;
    repeated double position = 3;     // Radians or meters
    repeated double velocity = 4;     // Rad/s or m/s
    repeated double effort = 5;       // Nm or N
}

// BatteryState represents battery status.
message BatteryState {
    gorai.std.Header header = 1;
    float voltage = 2;                // Volts
    float current = 3;                // Amps (negative = discharging)
    float charge = 4;                 // Ah
    float capacity = 5;               // Ah
    float design_capacity = 6;        // Ah
    float percentage = 7;             // 0.0 - 1.0
    enum PowerSupplyStatus {
        POWER_SUPPLY_STATUS_UNKNOWN = 0;
        POWER_SUPPLY_STATUS_CHARGING = 1;
        POWER_SUPPLY_STATUS_DISCHARGING = 2;
        POWER_SUPPLY_STATUS_NOT_CHARGING = 3;
        POWER_SUPPLY_STATUS_FULL = 4;
    }
    PowerSupplyStatus power_supply_status = 8;
    enum PowerSupplyHealth {
        POWER_SUPPLY_HEALTH_UNKNOWN = 0;
        POWER_SUPPLY_HEALTH_GOOD = 1;
        POWER_SUPPLY_HEALTH_OVERHEAT = 2;
        POWER_SUPPLY_HEALTH_DEAD = 3;
        POWER_SUPPLY_HEALTH_OVERVOLTAGE = 4;
        POWER_SUPPLY_HEALTH_UNSPEC_FAILURE = 5;
        POWER_SUPPLY_HEALTH_COLD = 6;
        POWER_SUPPLY_HEALTH_WATCHDOG_TIMER_EXPIRE = 7;
        POWER_SUPPLY_HEALTH_SAFETY_TIMER_EXPIRE = 8;
    }
    PowerSupplyHealth power_supply_health = 9;
    bool present = 10;
    repeated float cell_voltage = 11;
    string location = 12;
    string serial_number = 13;
}

// Temperature represents a temperature reading.
message Temperature {
    gorai.std.Header header = 1;
    double temperature = 2;           // Celsius
    double variance = 3;
}

// FluidPressure represents pressure reading.
message FluidPressure {
    gorai.std.Header header = 1;
    double fluid_pressure = 2;        // Pascals
    double variance = 3;
}

// Illuminance represents light level.
message Illuminance {
    gorai.std.Header header = 1;
    double illuminance = 2;           // Lux
    double variance = 3;
}
```

### control.proto - Control Messages

```protobuf
syntax = "proto3";
package gorai.control;

option go_package = "github.com/gorai-robotics/gorai/api/control";

import "gorai/std/std.proto";
import "gorai/geometry/geometry.proto";

// JointCommand sends commands to joints.
message JointCommand {
    gorai.std.Header header = 1;
    repeated string name = 2;
    repeated double position = 3;     // Desired position (optional)
    repeated double velocity = 4;     // Desired velocity (optional)
    repeated double effort = 5;       // Desired effort (optional)
    repeated double kp = 6;           // Position gains (optional)
    repeated double kd = 7;           // Velocity gains (optional)
}

// JointTrajectory defines a trajectory for joints.
message JointTrajectory {
    gorai.std.Header header = 1;
    repeated string joint_names = 2;
    repeated JointTrajectoryPoint points = 3;
}

// JointTrajectoryPoint is a waypoint in a trajectory.
message JointTrajectoryPoint {
    repeated double positions = 1;
    repeated double velocities = 2;
    repeated double accelerations = 3;
    repeated double effort = 4;
    gorai.std.Duration time_from_start = 5;
}

// PanTiltCommand commands a pan-tilt unit.
message PanTiltCommand {
    gorai.std.Header header = 1;
    double pan = 2;                   // Radians
    double tilt = 3;                  // Radians
    double pan_velocity = 4;          // Rad/s (optional)
    double tilt_velocity = 5;         // Rad/s (optional)
}

// PanTiltState reports pan-tilt status.
message PanTiltState {
    gorai.std.Header header = 1;
    double pan = 2;
    double tilt = 3;
    double pan_velocity = 4;
    double tilt_velocity = 5;
    bool is_moving = 6;
}

// MotorCommand commands a single motor.
message MotorCommand {
    gorai.std.Header header = 1;
    enum Mode {
        MODE_POWER = 0;           // Direct PWM/power
        MODE_VELOCITY = 1;        // Velocity control
        MODE_POSITION = 2;        // Position control
    }
    Mode mode = 2;
    double value = 3;             // Power (-1 to 1), velocity, or position
}

// MotorState reports motor status.
message MotorState {
    gorai.std.Header header = 1;
    double position = 2;          // Encoder position
    double velocity = 3;          // Current velocity
    double current = 4;           // Motor current (amps)
    double temperature = 5;       // Temperature (celsius)
    bool is_moving = 6;
}

// GripperCommand commands a gripper.
message GripperCommand {
    gorai.std.Header header = 1;
    double position = 2;          // 0.0 = closed, 1.0 = open
    double max_effort = 3;        // Maximum force
}

// GripperState reports gripper status.
message GripperState {
    gorai.std.Header header = 1;
    double position = 2;
    double effort = 3;
    bool stalled = 4;
    bool reached_goal = 5;
}
```

### vision.proto - Vision Messages

```protobuf
syntax = "proto3";
package gorai.vision;

option go_package = "github.com/gorai-robotics/gorai/api/vision";

import "gorai/std/std.proto";
import "gorai/geometry/geometry.proto";

// Detection represents a detected object.
message Detection {
    // Bounding box (normalized 0-1 or pixel coordinates).
    BoundingBox2D bbox = 1;

    // Detection results with confidence scores.
    repeated ObjectHypothesis results = 2;

    // Optional 3D pose of detected object.
    gorai.geometry.PoseWithCovariance pose = 3;

    // Instance segmentation mask (optional).
    bytes mask = 4;
    uint32 mask_width = 5;
    uint32 mask_height = 6;

    // Tracking ID for multi-frame tracking (optional).
    string tracking_id = 7;
}

// Detections is a collection of detections.
message Detections {
    gorai.std.Header header = 1;
    repeated Detection detections = 2;

    // Source image dimensions.
    uint32 source_width = 3;
    uint32 source_height = 4;
}

// BoundingBox2D represents a 2D bounding box.
message BoundingBox2D {
    // Center of the bounding box.
    double center_x = 1;
    double center_y = 2;

    // Size of the bounding box.
    double size_x = 3;
    double size_y = 4;

    // Alternative: corner representation.
    double x_min = 5;
    double y_min = 6;
    double x_max = 7;
    double y_max = 8;
}

// BoundingBox3D represents a 3D bounding box.
message BoundingBox3D {
    gorai.geometry.Pose center = 1;
    gorai.geometry.Vector3 size = 2;
}

// ObjectHypothesis represents a classification result.
message ObjectHypothesis {
    string class_id = 1;          // Class identifier
    string class_name = 2;        // Human-readable name
    double score = 3;             // Confidence (0-1)
}

// Classification represents image classification result.
message Classification {
    gorai.std.Header header = 1;
    repeated ObjectHypothesis results = 2;
}

// Classifications for batch classification.
message Classifications {
    gorai.std.Header header = 1;
    repeated Classification classifications = 2;
}

// SemanticSegmentation represents pixel-wise classification.
message SemanticSegmentation {
    gorai.std.Header header = 1;
    uint32 height = 2;
    uint32 width = 3;

    // Class ID for each pixel (row-major).
    bytes class_map = 4;          // uint8 per pixel

    // Or 16-bit class IDs for many classes.
    bytes class_map_16 = 5;       // uint16 per pixel

    // Class definitions.
    repeated ClassInfo classes = 6;
}

// ClassInfo describes a segmentation class.
message ClassInfo {
    uint32 id = 1;
    string name = 2;
    uint32 color_rgb = 3;         // Visualization color
}

// InstanceSegmentation represents instance-level segmentation.
message InstanceSegmentation {
    gorai.std.Header header = 1;
    uint32 height = 2;
    uint32 width = 3;

    // Instance ID for each pixel.
    bytes instance_map = 4;       // uint16 per pixel

    // Instance information.
    repeated InstanceInfo instances = 5;
}

// InstanceInfo describes a segmentation instance.
message InstanceInfo {
    uint32 id = 1;
    string class_name = 2;
    double confidence = 3;
    BoundingBox2D bbox = 4;
    uint32 pixel_count = 5;
}

// Keypoints represents detected keypoints (pose estimation).
message Keypoints {
    gorai.std.Header header = 1;
    repeated KeypointGroup groups = 2;
}

// KeypointGroup is a set of related keypoints (e.g., one person).
message KeypointGroup {
    repeated Keypoint keypoints = 1;
    repeated KeypointConnection connections = 2;
    double confidence = 3;
    string tracking_id = 4;
}

// Keypoint is a single detected keypoint.
message Keypoint {
    string name = 1;              // e.g., "left_shoulder"
    double x = 2;                 // Normalized 0-1
    double y = 3;
    double z = 4;                 // For 3D pose (optional)
    double confidence = 5;
    bool is_visible = 6;
}

// KeypointConnection defines a skeleton edge.
message KeypointConnection {
    string from_keypoint = 1;
    string to_keypoint = 2;
}

// DepthImage represents depth data.
message DepthImage {
    gorai.std.Header header = 1;
    uint32 height = 2;
    uint32 width = 3;

    enum DepthEncoding {
        DEPTH_16UC1_MM = 0;       // 16-bit unsigned, millimeters
        DEPTH_32FC1_M = 1;        // 32-bit float, meters
    }
    DepthEncoding encoding = 4;

    float min_depth = 5;          // Minimum valid depth
    float max_depth = 6;          // Maximum valid depth
    bytes data = 7;
}
```

### ml.proto - Machine Learning Messages

```protobuf
syntax = "proto3";
package gorai.ml;

option go_package = "github.com/gorai-robotics/gorai/api/ml";

import "gorai/std/std.proto";

// Tensor represents an n-dimensional array.
message Tensor {
    // Tensor name (for named inputs/outputs).
    string name = 1;

    // Data type.
    enum DataType {
        DT_INVALID = 0;
        DT_FLOAT = 1;
        DT_DOUBLE = 2;
        DT_INT8 = 3;
        DT_INT16 = 4;
        DT_INT32 = 5;
        DT_INT64 = 6;
        DT_UINT8 = 7;
        DT_UINT16 = 8;
        DT_UINT32 = 9;
        DT_UINT64 = 10;
        DT_BOOL = 11;
        DT_STRING = 12;
        DT_FLOAT16 = 13;
        DT_BFLOAT16 = 14;
    }
    DataType dtype = 2;

    // Shape (dimensions).
    repeated int64 shape = 3;

    // Raw data (packed according to dtype).
    bytes data = 4;

    // Or typed data (only one should be set).
    repeated float float_data = 5;
    repeated double double_data = 6;
    repeated int32 int32_data = 7;
    repeated int64 int64_data = 8;
    repeated bytes string_data = 9;
}

// TensorList is a collection of tensors.
message TensorList {
    repeated Tensor tensors = 1;
}

// ModelMetadata describes an ML model.
message ModelMetadata {
    string name = 1;
    string version = 2;
    string framework = 3;         // "tensorflow", "onnx", "tflite", "pytorch"
    string description = 4;

    // Input specifications.
    repeated TensorSpec inputs = 5;

    // Output specifications.
    repeated TensorSpec outputs = 6;

    // Model properties.
    map<string, string> properties = 7;
}

// TensorSpec describes expected tensor format.
message TensorSpec {
    string name = 1;
    Tensor.DataType dtype = 2;
    repeated int64 shape = 3;     // -1 for dynamic dimensions
    string description = 4;
}

// InferenceRequest requests model inference.
message InferenceRequest {
    gorai.std.Header header = 1;
    string model_name = 2;
    string model_version = 3;     // Empty for latest
    repeated Tensor inputs = 4;

    // Output filtering (empty = all outputs).
    repeated string output_names = 5;

    // Request options.
    map<string, string> options = 6;
}

// InferenceResponse contains inference results.
message InferenceResponse {
    gorai.std.Header header = 1;
    string model_name = 2;
    string model_version = 3;
    repeated Tensor outputs = 4;

    // Timing information.
    int64 inference_time_ns = 5;
    int64 preprocess_time_ns = 6;
    int64 postprocess_time_ns = 7;
}

// ModelInfo provides runtime model information.
message ModelInfo {
    ModelMetadata metadata = 1;

    // Runtime status.
    enum Status {
        STATUS_UNKNOWN = 0;
        STATUS_LOADING = 1;
        STATUS_READY = 2;
        STATUS_ERROR = 3;
        STATUS_UNLOADING = 4;
    }
    Status status = 2;

    // Accelerator being used.
    string accelerator = 3;       // "cpu", "cuda", "tpu", "npu"

    // Statistics.
    int64 inference_count = 4;
    int64 avg_inference_time_ns = 5;
}
```

### nav.proto - Navigation Messages

```protobuf
syntax = "proto3";
package gorai.nav;

option go_package = "github.com/gorai-robotics/gorai/api/nav";

import "gorai/std/std.proto";
import "gorai/geometry/geometry.proto";

// Path represents a sequence of poses.
message Path {
    gorai.std.Header header = 1;
    repeated gorai.geometry.PoseStamped poses = 2;
}

// Odometry represents robot odometry.
message Odometry {
    gorai.std.Header header = 1;
    string child_frame_id = 2;
    gorai.geometry.PoseWithCovariance pose = 3;
    gorai.geometry.TwistWithCovariance twist = 4;
}

// OccupancyGrid represents a 2D occupancy map.
message OccupancyGrid {
    gorai.std.Header header = 1;
    MapMetaData info = 2;

    // Occupancy data: -1 = unknown, 0-100 = probability (0 free, 100 occupied).
    repeated int8 data = 3;
}

// MapMetaData describes map properties.
message MapMetaData {
    gorai.std.Time map_load_time = 1;
    float resolution = 2;         // Meters per cell
    uint32 width = 3;             // Cells
    uint32 height = 4;            // Cells
    gorai.geometry.Pose origin = 5;
}

// Waypoint represents a navigation waypoint.
message Waypoint {
    string id = 1;
    gorai.geometry.Pose pose = 2;
    string name = 3;
    map<string, string> properties = 4;
}

// WaypointList is a collection of waypoints.
message WaypointList {
    gorai.std.Header header = 1;
    repeated Waypoint waypoints = 2;
}

// NavigationGoal requests navigation to a pose.
message NavigationGoal {
    gorai.std.Header header = 1;
    gorai.geometry.PoseStamped target_pose = 2;

    // Tolerances.
    double xy_goal_tolerance = 3;
    double yaw_goal_tolerance = 4;

    // Behavior flags.
    bool allow_backward = 5;
    double max_velocity = 6;
}

// NavigationFeedback provides navigation progress.
message NavigationFeedback {
    gorai.std.Header header = 1;
    gorai.geometry.PoseStamped current_pose = 2;
    double distance_remaining = 3;
    gorai.std.Duration time_remaining = 4;
    int32 recovery_count = 5;
}

// NavigationResult provides navigation outcome.
message NavigationResult {
    gorai.std.Header header = 1;

    enum Status {
        STATUS_UNKNOWN = 0;
        STATUS_SUCCEEDED = 1;
        STATUS_CANCELED = 2;
        STATUS_FAILED = 3;
    }
    Status status = 2;
    string message = 3;
    gorai.geometry.PoseStamped final_pose = 4;
}

// GeoPoint represents a geographic location.
message GeoPoint {
    double latitude = 1;          // Degrees
    double longitude = 2;         // Degrees
    double altitude = 3;          // Meters (WGS84)
}

// GeoPose represents a geographic pose.
message GeoPose {
    GeoPoint position = 1;
    gorai.geometry.Quaternion orientation = 2;
}

// GeoPath represents a geographic path.
message GeoPath {
    gorai.std.Header header = 1;
    repeated GeoPose poses = 2;
}
```

### action.proto - Action Protocol Messages

```protobuf
syntax = "proto3";
package gorai.action;

option go_package = "github.com/gorai-robotics/gorai/api/action";

import "gorai/std/std.proto";
import "google/protobuf/any.proto";

// GoalID uniquely identifies an action goal.
message GoalID {
    string id = 1;
    gorai.std.Time stamp = 2;
}

// GoalStatus represents the status of a goal.
message GoalStatus {
    GoalID goal_id = 1;

    enum Status {
        STATUS_UNKNOWN = 0;
        STATUS_ACCEPTED = 1;
        STATUS_EXECUTING = 2;
        STATUS_CANCELING = 3;
        STATUS_SUCCEEDED = 4;
        STATUS_CANCELED = 5;
        STATUS_ABORTED = 6;
    }
    Status status = 2;
    string text = 3;
}

// GoalStatusArray is a list of goal statuses.
message GoalStatusArray {
    gorai.std.Header header = 1;
    repeated GoalStatus status_list = 2;
}

// CancelGoal requests goal cancellation.
message CancelGoal {
    GoalID goal_id = 1;           // Empty = cancel all
    gorai.std.Time stamp = 2;     // Cancel goals before this time
}

// CancelGoalResponse responds to cancellation.
message CancelGoalResponse {
    enum Code {
        ERROR_NONE = 0;
        ERROR_REJECTED = 1;
        ERROR_UNKNOWN_GOAL = 2;
        ERROR_GOAL_TERMINATED = 3;
    }
    Code return_code = 1;
    repeated GoalID goals_canceling = 2;
}
```

---

## Core Components

### Node

```go
package node

type Node struct {
    // ...
}

type Option func(*Node) error

// New creates a new node.
func New(name string, opts ...Option) (*Node, error)

// Options
func WithNATS(url string) Option
func WithNATSConn(nc *nats.Conn) Option
func WithNamespace(ns string) Option
func WithLogger(logger *slog.Logger) Option
func WithContext(ctx context.Context) Option

// Methods
func (n *Node) Name() string
func (n *Node) Namespace() string
func (n *Node) FullName() string                    // namespace.name
func (n *Node) NATS() *nats.Conn
func (n *Node) JetStream() nats.JetStreamContext
func (n *Node) Logger() *slog.Logger
func (n *Node) Context() context.Context
func (n *Node) Spin(ctx context.Context) error      // Block until context done
func (n *Node) SpinOnce() error                     // Process one message
func (n *Node) Close() error
```

### Publisher

```go
package pub

type Publisher[T proto.Message] struct {
    // ...
}

type Option func(*options)

// New creates a typed publisher.
func New[T proto.Message](n *node.Node, topic string, opts ...Option) *Publisher[T]

// Options
func WithQoS(qos QoS) Option
func WithRetain() Option
func WithHistory(n int) Option

// Methods
func (p *Publisher[T]) Publish(ctx context.Context, msg T) error
func (p *Publisher[T]) Topic() string
func (p *Publisher[T]) Close() error
```

### Subscriber

```go
package sub

type Subscriber[T proto.Message] struct {
    // ...
}

type Option func(*options)
type Handler[T proto.Message] func(msg T)

// New creates a typed subscriber.
func New[T proto.Message](n *node.Node, topic string, handler Handler[T], opts ...Option) *Subscriber[T]

// Options
func WithQueueGroup(name string) Option
func WithBuffer(size int) Option
func WithStartTime(t time.Time) Option
func WithDeliverAll() Option
func WithDeliverLast() Option

// Methods
func (s *Subscriber[T]) Topic() string
func (s *Subscriber[T]) Close() error
```

### Service

```go
package srv

type Server[Req, Resp proto.Message] struct {
    // ...
}

type Client[Req, Resp proto.Message] struct {
    // ...
}

type Handler[Req, Resp proto.Message] func(ctx context.Context, req Req) (Resp, error)

// NewServer creates a service server.
func NewServer[Req, Resp proto.Message](n *node.Node, name string, handler Handler[Req, Resp]) *Server[Req, Resp]

// NewClient creates a service client.
func NewClient[Req, Resp proto.Message](n *node.Node, name string) *Client[Req, Resp]

// Client methods
func (c *Client[Req, Resp]) Call(ctx context.Context, req Req, opts ...CallOption) (Resp, error)

// Call options
func WithTimeout(d time.Duration) CallOption
```

### Action

```go
package action

type Server[Goal, Feedback, Result proto.Message] struct {
    // ...
}

type Client[Goal, Feedback, Result proto.Message] struct {
    // ...
}

type FeedbackSender[Feedback proto.Message] interface {
    Send(fb Feedback) error
}

type Handler[Goal, Feedback, Result proto.Message] func(
    ctx context.Context,
    goal Goal,
    feedback FeedbackSender[Feedback],
) (Result, error)

// NewServer creates an action server.
func NewServer[Goal, Feedback, Result proto.Message](
    n *node.Node,
    name string,
    handler Handler[Goal, Feedback, Result],
) *Server[Goal, Feedback, Result]

// NewClient creates an action client.
func NewClient[Goal, Feedback, Result proto.Message](
    n *node.Node,
    name string,
) *Client[Goal, Feedback, Result]

// GoalHandle represents an active goal.
type GoalHandle[Feedback, Result proto.Message] interface {
    ID() string
    Status() Status
    Feedback() <-chan Feedback
    Result() (Result, error)
    Cancel() error
}

// Client methods
func (c *Client[G, F, R]) SendGoal(ctx context.Context, goal G) (GoalHandle[F, R], error)
func (c *Client[G, F, R]) CancelAll(ctx context.Context) error
```

### Parameter Store

```go
package param

type Store struct {
    // ...
}

// NewStore creates a parameter store backed by NATS KV.
func NewStore(n *node.Node) (*Store, error)

// Methods
func (s *Store) Set(key string, value any) error
func (s *Store) Get(key string) (any, error)
func (s *Store) Delete(key string) error
func (s *Store) Keys(pattern string) ([]string, error)
func (s *Store) Watch(pattern string, handler func(key string, value any)) error

// Typed accessors
func Get[T any](s *Store, key string) (T, error)
func GetWithDefault[T any](s *Store, key string, def T) T
```

---

## Device Interfaces

### Motor Interface

```go
package motor

type Motor interface {
    resource.Resource

    // SetPower sets motor power (-1.0 to 1.0).
    SetPower(ctx context.Context, power float64) error

    // SetVelocity sets target velocity (rad/s or m/s).
    SetVelocity(ctx context.Context, velocity float64) error

    // GoTo moves to absolute position.
    GoTo(ctx context.Context, position float64, velocity float64) error

    // GetPosition returns current position.
    GetPosition(ctx context.Context) (float64, error)

    // GetVelocity returns current velocity.
    GetVelocity(ctx context.Context) (float64, error)

    // ResetZeroPosition sets current position as zero.
    ResetZeroPosition(ctx context.Context) error

    // Stop stops the motor.
    Stop(ctx context.Context) error

    // IsMoving returns true if motor is in motion.
    IsMoving(ctx context.Context) (bool, error)

    // IsPowered returns true if motor is powered.
    IsPowered(ctx context.Context) (bool, error)

    // Properties returns motor capabilities.
    Properties(ctx context.Context) (Properties, error)
}

type Properties struct {
    PositionReporting bool
    VelocityReporting bool
    SupportsGoTo      bool
}
```

### Camera Interface

```go
package camera

type Camera interface {
    resource.Resource

    // GetImage captures an image.
    GetImage(ctx context.Context) (*sensor.Image, error)

    // GetImages captures from all streams (color, depth, etc.).
    GetImages(ctx context.Context) ([]*sensor.Image, error)

    // GetPointCloud returns 3D point cloud (if depth capable).
    GetPointCloud(ctx context.Context) (*sensor.PointCloud2, error)

    // GetProperties returns camera properties.
    GetProperties(ctx context.Context) (Properties, error)

    // Stream returns a channel of images.
    Stream(ctx context.Context) (<-chan *sensor.Image, error)
}

type Properties struct {
    Width             int
    Height            int
    FrameRate         float64
    SupportedFormats  []string
    IntrinsicParams   *sensor.CameraInfo
    DistortionParams  []float64
    SupportsDepth     bool
    SupportsPointCloud bool
}
```

### Sensor Interface

```go
package sensor

type Sensor interface {
    resource.Resource

    // Readings returns current sensor readings.
    Readings(ctx context.Context) (map[string]any, error)
}

// Specialized sensor interfaces

type IMU interface {
    Sensor
    GetAngularVelocity(ctx context.Context) (*geometry.Vector3, error)
    GetLinearAcceleration(ctx context.Context) (*geometry.Vector3, error)
    GetOrientation(ctx context.Context) (*geometry.Quaternion, error)
}

type GPS interface {
    Sensor
    GetPosition(ctx context.Context) (*GeoPoint, error)
    GetAltitude(ctx context.Context) (float64, error)
    GetLinearVelocity(ctx context.Context) (*geometry.Vector3, error)
    GetHeading(ctx context.Context) (float64, error)
    GetAccuracy(ctx context.Context) (float64, float64, error) // horizontal, vertical
}

type Encoder interface {
    Sensor
    GetPosition(ctx context.Context) (float64, error)
    GetVelocity(ctx context.Context) (float64, error)
    ResetPosition(ctx context.Context) error
}

type RangeSensor interface {
    Sensor
    GetRange(ctx context.Context) (float64, error)
    GetRanges(ctx context.Context) ([]float64, error) // For array sensors
}
```

### Base Interface (Mobile Robot)

```go
package base

type Base interface {
    resource.Resource
    resource.Actuator
    resource.Shaped

    // MoveStraight moves forward/backward.
    MoveStraight(ctx context.Context, distanceMm int, mmPerSec float64) error

    // Spin rotates in place.
    Spin(ctx context.Context, angleDeg float64, degPerSec float64) error

    // SetPower sets raw wheel powers.
    SetPower(ctx context.Context, linear, angular geometry.Vector3) error

    // SetVelocity sets velocity.
    SetVelocity(ctx context.Context, linear, angular geometry.Vector3) error

    // GetProperties returns base properties.
    GetProperties(ctx context.Context) (Properties, error)
}

type Properties struct {
    WheelCircumferenceMm float64
    WidthMm              float64
    TurningRadiusMm      float64
}
```

### Arm Interface

```go
package arm

type Arm interface {
    resource.Resource
    resource.Actuator
    resource.Shaped

    // GetEndPosition returns end effector pose.
    GetEndPosition(ctx context.Context) (*geometry.Pose, error)

    // MoveToPosition moves end effector to pose.
    MoveToPosition(ctx context.Context, pose *geometry.Pose) error

    // GetJointPositions returns current joint positions.
    GetJointPositions(ctx context.Context) ([]float64, error)

    // MoveToJointPositions moves to joint positions.
    MoveToJointPositions(ctx context.Context, positions []float64) error

    // GetKinematics returns kinematic model.
    GetKinematics(ctx context.Context) (Kinematics, error)
}

type Kinematics struct {
    DOF          int
    JointLimits  []JointLimit
    URDFModel    []byte // Optional URDF
}

type JointLimit struct {
    Min float64
    Max float64
}
```

### Gripper Interface

```go
package gripper

type Gripper interface {
    resource.Resource
    resource.Actuator
    resource.Shaped

    // Open opens the gripper.
    Open(ctx context.Context) error

    // Close closes the gripper.
    Close(ctx context.Context) error

    // Grab closes until resistance is felt.
    Grab(ctx context.Context) (bool, error) // Returns true if object grasped

    // GetPosition returns current position (0 = closed, 1 = open).
    GetPosition(ctx context.Context) (float64, error)

    // SetPosition sets position (0-1).
    SetPosition(ctx context.Context, position float64) error
}
```

---

## AI/ML Services

### Vision Service

```go
package vision

type Service interface {
    resource.Resource

    // GetDetections runs object detection.
    GetDetections(ctx context.Context, img *sensor.Image) (*vision.Detections, error)

    // GetDetectionsFromCamera runs detection on camera stream.
    GetDetectionsFromCamera(ctx context.Context, cameraName string) (*vision.Detections, error)

    // GetClassifications runs image classification.
    GetClassifications(ctx context.Context, img *sensor.Image, n int) (*vision.Classifications, error)

    // GetClassificationsFromCamera runs classification on camera.
    GetClassificationsFromCamera(ctx context.Context, cameraName string, n int) (*vision.Classifications, error)

    // GetObjectPointClouds returns 3D segmented objects.
    GetObjectPointClouds(ctx context.Context, cameraName string) ([]*sensor.PointCloud2, error)

    // GetDetectorNames returns available detector models.
    GetDetectorNames(ctx context.Context) ([]string, error)

    // GetClassifierNames returns available classifier models.
    GetClassifierNames(ctx context.Context) ([]string, error)

    // AddDetector adds a detection model.
    AddDetector(ctx context.Context, config DetectorConfig) error

    // AddClassifier adds a classification model.
    AddClassifier(ctx context.Context, config ClassifierConfig) error
}

type DetectorConfig struct {
    Name           string
    ModelPath      string
    ModelType      string            // "tflite", "onnx", "tensorflow", "pytorch"
    LabelPath      string
    Accelerator    string            // "cpu", "tpu", "npu", "cuda"
    ConfidenceThreshold float64
    MaxDetections  int
    Parameters     map[string]any
}

type ClassifierConfig struct {
    Name           string
    ModelPath      string
    ModelType      string
    LabelPath      string
    Accelerator    string
    TopK           int
    Parameters     map[string]any
}
```

### ML Model Service

```go
package mlmodel

type Service interface {
    resource.Resource

    // Infer runs model inference.
    Infer(ctx context.Context, inputs map[string]*ml.Tensor) (map[string]*ml.Tensor, error)

    // Metadata returns model metadata.
    Metadata(ctx context.Context) (*ml.ModelMetadata, error)
}

// ModelConfig configures a model.
type ModelConfig struct {
    Name        string
    ModelPath   string
    ModelType   string            // "tflite", "onnx", "tensorflow", "pytorch", "openvino"
    Accelerator string            // "cpu", "tpu", "npu", "cuda", "auto"

    // Preprocessing options.
    InputNormalization  *Normalization
    InputResize         *Resize

    // Runtime options.
    NumThreads          int
    UseXNNPack          bool
    AllowFP16           bool
    AllowInt8           bool

    // Accelerator-specific options.
    TPUDevice           string    // e.g., "usb:0"
    NPUDevice           string
    CUDADevice          int
}

type Normalization struct {
    Mean   []float32
    Stddev []float32
}

type Resize struct {
    Width  int
    Height int
    Mode   string // "bilinear", "nearest"
}
```

### SLAM Service

```go
package slam

type Service interface {
    resource.Resource

    // GetPosition returns current pose estimate.
    GetPosition(ctx context.Context) (*geometry.PoseStamped, error)

    // GetPointCloudMap returns the current map as point cloud.
    GetPointCloudMap(ctx context.Context) (*sensor.PointCloud2, error)

    // GetInternalState returns internal SLAM state for persistence.
    GetInternalState(ctx context.Context) ([]byte, error)

    // GetProperties returns SLAM properties.
    GetProperties(ctx context.Context) (Properties, error)
}

type Properties struct {
    CloudSlam       bool    // Using cloud SLAM
    MappingMode     string  // "localization_only", "mapping", "updating"
    SensorType      string  // "lidar", "camera", "rgbd"
}
```

### Navigation Service

```go
package navigation

type Service interface {
    resource.Resource

    // GetMode returns current navigation mode.
    GetMode(ctx context.Context) (Mode, error)

    // SetMode sets navigation mode.
    SetMode(ctx context.Context, mode Mode) error

    // GetLocation returns current location.
    GetLocation(ctx context.Context) (*nav.GeoPoint, error)

    // GetWaypoints returns configured waypoints.
    GetWaypoints(ctx context.Context) ([]*nav.Waypoint, error)

    // AddWaypoint adds a navigation waypoint.
    AddWaypoint(ctx context.Context, point *nav.GeoPoint) error

    // RemoveWaypoint removes a waypoint.
    RemoveWaypoint(ctx context.Context, id string) error

    // GetObstacles returns detected obstacles.
    GetObstacles(ctx context.Context) ([]*geometry.GeoObstacle, error)

    // GetPaths returns planned paths.
    GetPaths(ctx context.Context) ([]*nav.GeoPath, error)

    // GetProperties returns navigation properties.
    GetProperties(ctx context.Context) (Properties, error)
}

type Mode int

const (
    ModeManual Mode = iota
    ModeWaypoint
    ModeExplore
)

type Properties struct {
    MapType string // "gps", "local", "none"
}
```

### Motion Service

```go
package motion

type Service interface {
    resource.Resource

    // Move moves a component to a destination.
    Move(ctx context.Context, componentName string, destination *geometry.PoseStamped, worldState *WorldState) error

    // MoveOnMap moves on a SLAM map.
    MoveOnMap(ctx context.Context, componentName string, destination *geometry.Pose, slamService string) (ExecutionID, error)

    // MoveOnGlobe moves to a geographic location.
    MoveOnGlobe(ctx context.Context, componentName string, destination *nav.GeoPoint, heading float64) (ExecutionID, error)

    // StopPlan stops current motion plan.
    StopPlan(ctx context.Context, componentName string) error

    // GetPose gets pose of a component in a frame.
    GetPose(ctx context.Context, componentName string, destinationFrame string) (*geometry.PoseStamped, error)
}

type WorldState struct {
    Obstacles   []*geometry.GeometriesInFrame
    Transforms  []*geometry.TransformStamped
}

type ExecutionID string
```

---

## Acceleration Layer

### Accelerator Interface

```go
package accel

// Accelerator provides hardware acceleration for ML inference.
type Accelerator interface {
    // Name returns accelerator identifier.
    Name() string

    // Type returns accelerator type.
    Type() AcceleratorType

    // IsAvailable checks if accelerator is usable.
    IsAvailable() bool

    // Capabilities returns supported operations.
    Capabilities() Capabilities

    // LoadModel loads a model for inference.
    LoadModel(ctx context.Context, config ModelConfig) (Model, error)

    // Close releases accelerator resources.
    Close() error
}

type AcceleratorType int

const (
    AcceleratorCPU AcceleratorType = iota
    AcceleratorCUDA
    AcceleratorTPU           // Google Coral
    AcceleratorNPU           // Various NPU chips
    AcceleratorOpenVINO      // Intel
    AcceleratorROCm          // AMD
)

type Capabilities struct {
    SupportedFormats []string  // "tflite", "onnx", "tensorflow", "openvino"
    MaxBatchSize     int
    SupportsInt8     bool
    SupportsFP16     bool
    SupportsDynamic  bool      // Dynamic input shapes
    MemoryMB         int
}

type Model interface {
    // Infer runs inference.
    Infer(ctx context.Context, inputs []Tensor) ([]Tensor, error)

    // Metadata returns model info.
    Metadata() ModelMetadata

    // Close releases model resources.
    Close() error
}
```

### NPU Support (Rockchip RK3588) - WORKING

**Status:** Production-ready via [go-rknnlite](https://github.com/swdee/go-rknnlite)

```go
package npu

import "github.com/swdee/go-rknnlite"

// RK3588 has 3 NPU cores, 6 TOPS total
// Supported chips: RK3562, RK3566, RK3568, RK3576, RK3582, RK3588

runtime, _ := rknnlite.NewRuntime(modelPath, rknnlite.NPUCoreAuto)
defer runtime.Close()

outputs, _ := runtime.Inference(inputData)
```

**Requirements:**
- Linux (Armbian, Radxa OS, etc.)
- RKNN-Toolkit2 installed

### GPU Support (NVIDIA CUDA) - WORKING

**Status:** Production-ready via [onnxruntime_go](https://github.com/yalue/onnxruntime_go)

```go
package gpu

import ort "github.com/yalue/onnxruntime_go"

// Requires CUDA 12.x and cuDNN 9.x
// Requires CUDA-enabled onnxruntime library (not included by default)

cudaOpts, _ := ort.NewCUDAProviderOptions()
defer cudaOpts.Destroy()

sessionOpts, _ := ort.NewSessionOptions()
sessionOpts.AppendExecutionProviderCUDA(cudaOpts)

session, _ := ort.NewSessionWithOptions(modelPath, sessionOpts)
```

### TPU Support (Google Coral) - ASPIRATIONAL

**Status:** No Go bindings exist. CGo bindings to [libedgetpu](https://github.com/google-coral/libedgetpu) required.

```go
package tpu

// FUTURE: Requires CGo bindings to libedgetpu C API
// Available Coral TPU implementations:
// - USB Accelerator (~$60)
// - M.2/Mini PCIe Accelerator (~$25)
// - Dev Board

type CoralAccelerator struct {
    // ...
}

func NewCoralAccelerator(device string) (*CoralAccelerator, error)

// device examples:
// - "usb:0" - First USB accelerator
// - "pci:0" - First PCIe accelerator
// - ""      - Auto-detect
```

### NPU Support (Hailo) - ASPIRATIONAL

**Status:** No Go bindings exist. CGo bindings to [HailoRT](https://github.com/hailo-ai/hailort) required.

```go
package npu

// FUTURE: Requires CGo bindings to HailoRT C API
// Hailo-8L: 13 TOPS (Raspberry Pi AI Kit)
// Hailo-8: 26 TOPS

type HailoAccelerator struct {
    // ...
}

func NewHailoAccelerator(device string) (*HailoAccelerator, error)
```

### Acceleration Status Summary

| Platform | Go Support | Status | Library/Notes |
|----------|------------|--------|---------------|
| Rockchip RK3588 NPU | **Working** | Production | go-rknnlite |
| NVIDIA CUDA | **Working** | Production | onnxruntime_go (requires CUDA 12.x) |
| Intel OpenVINO | Partial | May need updates | GoCV (uses OpenVINO 2022.1) |
| Google Coral TPU | **None** | Aspirational | CGo bindings needed |
| Hailo NPU | **None** | Aspirational | CGo bindings needed |

### Model Format Support

| Format | CPU | Rockchip NPU | CUDA | Coral TPU* | Hailo NPU* |
|--------|-----|--------------|------|------------|------------|
| ONNX | Yes | Via RKNN | Yes | N/A | N/A |
| TFLite | Yes | Via RKNN | Via TensorRT | N/A | N/A |
| RKNN | No | Yes | No | N/A | N/A |

*Coral TPU and Hailo NPU require CGo bindings to be developed before model support is available.

### Model Conversion Utilities

```go
package convert

// ConvertToTFLite converts models to TFLite format.
func ConvertToTFLite(src string, dst string, opts ConvertOptions) error

// ConvertToEdgeTPU compiles TFLite for Coral TPU.
func ConvertToEdgeTPU(src string, dst string) error

// ConvertToRKNN converts to Rockchip RKNN format.
func ConvertToRKNN(src string, dst string, platform string) error

// ConvertToHEF converts to Hailo HEF format.
func ConvertToHEF(src string, dst string) error

type ConvertOptions struct {
    Quantize        bool
    QuantizationType string  // "int8", "fp16"
    RepresentativeDataset func() [][]float32
}
```

---

## Configuration System

### Configuration File Format

```json
{
  "robot": {
    "name": "sentinel",
    "namespace": "gorai"
  },

  "nats": {
    "url": "nats://localhost:4222",
    "embedded": true,
    "jetstream": true
  },

  "components": [
    {
      "name": "camera_front",
      "type": "camera",
      "model": "gorai:builtin:v4l2",
      "config": {
        "device": "/dev/video0",
        "width": 1280,
        "height": 720,
        "format": "MJPEG",
        "fps": 30
      },
      "depends_on": []
    },
    {
      "name": "motor_left",
      "type": "motor",
      "model": "gorai:builtin:gpio",
      "config": {
        "pin_pwm": 18,
        "pin_dir": 23,
        "pin_encoder_a": 24,
        "pin_encoder_b": 25,
        "encoder_ticks_per_rev": 1440,
        "max_rpm": 200
      }
    },
    {
      "name": "imu",
      "type": "movement_sensor",
      "model": "gorai:builtin:mpu6050",
      "config": {
        "i2c_bus": 1,
        "i2c_addr": "0x68",
        "sample_rate": 100
      }
    },
    {
      "name": "drive",
      "type": "base",
      "model": "gorai:builtin:differential",
      "config": {
        "left_motor": "motor_left",
        "right_motor": "motor_right",
        "wheel_circumference_mm": 220,
        "wheel_separation_mm": 300
      },
      "depends_on": ["motor_left", "motor_right"]
    }
  ],

  "services": [
    {
      "name": "vision",
      "type": "vision",
      "model": "gorai:builtin:mlvision",
      "config": {
        "detectors": [
          {
            "name": "objects",
            "model_path": "/models/ssd_mobilenet_v2.tflite",
            "label_path": "/models/coco_labels.txt",
            "accelerator": "tpu",
            "confidence_threshold": 0.5
          }
        ],
        "classifiers": [
          {
            "name": "scene",
            "model_path": "/models/mobilenet_v2.tflite",
            "label_path": "/models/imagenet_labels.txt",
            "accelerator": "tpu"
          }
        ]
      }
    },
    {
      "name": "slam",
      "type": "slam",
      "model": "gorai:builtin:cartographer",
      "config": {
        "sensors": [
          {"name": "lidar"},
          {"name": "imu"}
        ],
        "mode": "mapping",
        "map_resolution": 0.05
      }
    },
    {
      "name": "navigation",
      "type": "navigation",
      "model": "gorai:builtin:nav",
      "config": {
        "slam_service": "slam",
        "base": "drive",
        "motion_service": "motion"
      }
    },
    {
      "name": "motion",
      "type": "motion",
      "model": "gorai:builtin:motion",
      "config": {
        "slam_service": "slam"
      }
    }
  ],

  "modules": [
    {
      "name": "custom_sensor",
      "path": "/opt/gorai/modules/custom_sensor",
      "config": {
        "port": 9000
      }
    }
  ],

  "params": {
    "camera_front": {
      "exposure": "auto",
      "white_balance": "auto"
    },
    "navigation": {
      "max_velocity": 0.5,
      "goal_tolerance": 0.1
    }
  }
}
```

### Hot Reconfiguration

```go
package config

type Manager struct {
    // ...
}

// NewManager creates a configuration manager.
func NewManager(configPath string) (*Manager, error)

// Methods
func (m *Manager) Load() (*Config, error)
func (m *Manager) Watch(handler func(old, new *Config)) error
func (m *Manager) Reload() error

// Config change handling
type ChangeSet struct {
    Added    []ResourceConfig
    Removed  []ResourceConfig
    Modified []ResourceConfig
    Unchanged []ResourceConfig
}

func (m *Manager) ComputeChanges(old, new *Config) ChangeSet
```

---

## Network Transparency

### Network Wrapper Pattern

Following YARP's NWS/NWC pattern, resources can be accessed transparently over the network:

```go
package nws

// NetworkWrapperServer exposes a local resource over NATS.
type NetworkWrapperServer struct {
    // ...
}

// Wrap exposes a resource over the network.
func Wrap(n *node.Node, res resource.Resource) (*NetworkWrapperServer, error)

// NetworkWrapperClient provides remote resource access.
type NetworkWrapperClient struct {
    // ...
}

// Connect creates a client for a remote resource.
func Connect[T resource.Resource](n *node.Node, name resource.Name) (T, error)
```

### Example: Remote Camera

```go
// On robot (server side)
cam, _ := v4l2.NewCamera(config)
nws.Wrap(node, cam)  // Exposes as gorai.robot.camera_front.*

// On workstation (client side)
remoteCam, _ := nws.Connect[camera.Camera](node, resource.Name{
    Namespace: "gorai",
    Type:      "component",
    Subtype:   "camera",
    Name:      "camera_front",
})
img, _ := remoteCam.GetImage(ctx)  // Transparent RPC
```

---

## CLI Tool

### Commands

```bash
# Topic operations
gorai topic list                                    # List active topics
gorai topic echo <topic>                            # Print messages
gorai topic hz <topic>                              # Measure rate
gorai topic pub <topic> '<json>'                    # Publish message
gorai topic info <topic>                            # Topic info

# Service operations
gorai service list                                  # List services
gorai service call <service> '<json>'               # Call service
gorai service info <service>                        # Service info

# Action operations
gorai action list                                   # List actions
gorai action send <action> '<json>'                 # Send goal
gorai action cancel <action> [goal_id]              # Cancel goal

# Parameter operations
gorai param list [pattern]                          # List parameters
gorai param get <key>                               # Get value
gorai param set <key> <value>                       # Set value
gorai param delete <key>                            # Delete parameter

# Node operations
gorai node list                                     # List nodes
gorai node info <node>                              # Node details
gorai node kill <node>                              # Stop node

# Resource operations
gorai resource list                                 # List all resources
gorai resource info <name>                          # Resource details
gorai resource call <name> <method> '<json>'        # Call DoCommand

# Transform operations
gorai tf list                                       # List frames
gorai tf echo <source> <target>                     # Print transform
gorai tf tree                                       # Show TF tree

# Recording/playback
gorai bag record <topics...> -o <file>              # Record messages
gorai bag play <file>                               # Playback messages
gorai bag info <file>                               # Bag file info

# System operations
gorai launch <config.json>                          # Launch robot
gorai status                                        # System status
gorai doctor                                        # Diagnostics
```

---

## Directory Structure

```
gorai/
├── go.mod
├── go.sum
├── README.md
├── LICENSE                          # Apache 2.0
│
├── api/                             # Protocol buffer definitions
│   └── proto/
│       ├── gorai/
│       │   ├── std/
│       │   ├── geometry/
│       │   ├── sensor/
│       │   ├── control/
│       │   ├── vision/
│       │   ├── ml/
│       │   ├── nav/
│       │   └── action/
│       └── buf.yaml
│
├── pkg/                             # Core library
│   ├── node/                        # Node implementation
│   ├── pub/                         # Publisher
│   ├── sub/                         # Subscriber
│   ├── srv/                         # Service
│   ├── action/                      # Action
│   ├── param/                       # Parameter store
│   ├── tf/                          # Transform library
│   ├── resource/                    # Resource interfaces
│   ├── config/                      # Configuration
│   └── nws/                         # Network wrappers
│
├── component/                       # Component interfaces
│   ├── motor/
│   ├── camera/
│   ├── sensor/
│   ├── base/
│   ├── arm/
│   └── gripper/
│
├── service/                         # Service interfaces
│   ├── vision/
│   ├── mlmodel/
│   ├── slam/
│   ├── navigation/
│   └── motion/
│
├── accel/                           # Acceleration layer
│   ├── accel.go                     # Interface
│   ├── cpu/
│   ├── tpu/                         # Coral TPU
│   ├── npu/                         # Various NPUs
│   └── cuda/                        # NVIDIA GPU
│
├── driver/                          # Hardware drivers
│   ├── v4l2/                        # Video4Linux cameras
│   ├── gpio/                        # GPIO motors/sensors
│   ├── i2c/                         # I2C devices
│   ├── spi/                         # SPI devices
│   ├── serial/                      # Serial devices
│   └── usb/                         # USB devices
│
├── cmd/
│   └── gorai/                       # CLI tool
│       └── main.go
│
├── examples/
│   ├── minimal/                     # Minimal example
│   ├── camera/                      # Camera streaming
│   ├── motor/                       # Motor control
│   ├── vision/                      # Object detection
│   └── navigation/                  # Navigation example
│
└── docs/
    ├── getting-started.md
    ├── concepts.md
    ├── tutorials/
    └── api/
```

---

## Appendix A: NATS Quick Reference

### Running NATS

```bash
# Install
go install github.com/nats-io/nats-server/v2@latest

# Run with JetStream
nats-server -js

# Podman (OCI container)
podman run -p 4222:4222 -p 8222:8222 docker.io/library/nats:latest -js
```

### NATS CLI

```bash
# Install
go install github.com/nats-io/natscli/nats@latest

# Subscribe to all gorai topics
nats sub "gorai.>"

# Publish
nats pub gorai.test "hello"

# Request/reply
nats request gorai.service.test '{"foo": "bar"}'

# JetStream
nats stream add GORAI --subjects "gorai.>" --retention limits --max-msgs 1000000
nats stream ls
nats consumer add GORAI replay --deliver all --replay instant

# KV store
nats kv add PARAMS
nats kv put PARAMS camera.exposure 100
nats kv get PARAMS camera.exposure
nats kv watch PARAMS ">"
```

---

## Appendix B: Version History

| Version | Date | Changes |
|---------|------|---------|
| 0.1.0 | 2024-XX-XX | Initial specification |

---

## Appendix C: References

- [NATS Documentation](https://docs.nats.io/)
- [Protocol Buffers](https://protobuf.dev/)
- [TensorFlow Lite](https://www.tensorflow.org/lite)
- [ONNX Runtime](https://onnxruntime.ai/)
- [Coral Edge TPU](https://coral.ai/)
- [Hailo AI Processors](https://hailo.ai/)
