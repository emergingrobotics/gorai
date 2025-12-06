# Gort: A Modern Robotics Framework

**A lightweight, Go-based alternative to ROS 2, YARP, and Viam**

---

## Executive Summary

Gort is a modern robotics framework built on NATS.io, designed to provide the essential capabilities of ROS 2 without the complexity of DDS, the legacy baggage of YARP, or the cloud-lock-in concerns of Viam. By leveraging NATS's battle-tested messaging infrastructure, Gort offers a clean, Go-idiomatic approach to building distributed robot systems.

This document outlines the framework architecture and proposes **Gort-Sentinel**, a pan-tilt sensor fusion platform, as the first reference implementation to prove out the core concepts.

---

## Table of Contents

1. [Motivation](#motivation)
2. [Framework Architecture](#framework-architecture)
3. [Core Components](#core-components)
4. [Message Types](#message-types)
5. [Communication Patterns](#communication-patterns)
6. [First Project: Gort-Sentinel](#first-project-gort-sentinel)
7. [Development Roadmap](#development-roadmap)
8. [Technical Decisions](#technical-decisions)

---

## Motivation

### Problems with Existing Frameworks

**ROS 2**
- DDS middleware is complex to configure and debug
- Vendor fragmentation (FastDDS, Cyclone, Connext) causes interop headaches
- Heavy resource footprint for embedded systems
- Slow build times with C++ codebase
- Steep learning curve

**YARP**
- Primarily serves the iCub research community
- Limited adoption outside academia
- C++ complexity remains
- Smaller ecosystem and driver support

**Viam**
- Cloud-centric architecture raises data sovereignty concerns
- Commercial interests may diverge from open-source community needs
- Relatively young, API still evolving

### Why NATS + Go?

| Aspect | Benefit |
|--------|---------|
| **NATS messaging** | Production-proven pub/sub, request/reply, persistence (JetStream), service discovery—all built in |
| **Go language** | Fast compilation, single-binary deployment, excellent concurrency, memory safety |
| **Operational simplicity** | NATS server is a single ~15MB binary; no configuration hell |
| **Edge-native** | Leaf nodes for edge deployment, low resource footprint |
| **Cloud-ready** | NATS already used in Kubernetes, Synadia Cloud available |

---

## Framework Architecture

### High-Level Design

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Gort Application                            │
├─────────────────────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │  Camera  │  │   ToF    │  │  Motor   │  │  Fusion  │   Nodes    │
│  │   Node   │  │   Node   │  │   Node   │  │   Node   │            │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘            │
│       │             │             │             │                   │
├───────┴─────────────┴─────────────┴─────────────┴───────────────────┤
│                         Gort Core Library                           │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐   │
│  │  Node   │  │ Pub/Sub │  │ Service │  │ Action  │  │  Param  │   │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘  └─────────┘   │
├─────────────────────────────────────────────────────────────────────┤
│                         NATS + JetStream                            │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐               │
│  │ Pub/Sub │  │ Req/Rep │  │   KV    │  │ Service │               │
│  │         │  │         │  │  Store  │  │  Disc.  │               │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘               │
└─────────────────────────────────────────────────────────────────────┘
```

### Subject Namespace Convention

Gort uses a hierarchical subject naming scheme:

```
gort.{robot}.{node}.{topic}

Examples:
gort.sentinel.camera.image
gort.sentinel.tof.pointcloud
gort.sentinel.pantilt.state
gort.sentinel.fusion.depth_image
```

Service and action subjects follow the same pattern with suffixes:

```
gort.sentinel.pantilt.move_to.goal      # action goal
gort.sentinel.pantilt.move_to.feedback  # action feedback
gort.sentinel.pantilt.home.request      # service request
gort.sentinel.pantilt.home.response     # service response
```

---

## Core Components

### Node

The fundamental unit of computation. Manages lifecycle, logging, and NATS connection.

```go
package main

import (
    "context"
    "github.com/gort-robotics/gort/pkg/node"
)

func main() {
    n, err := node.New("camera_driver",
        node.WithNATS("nats://localhost:4222"),
        node.WithNamespace("sentinel"),
        node.WithLogger(slog.Default()),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer n.Close()

    // Node logic here

    n.Spin(context.Background())
}
```

**Node API**

| Method | Description |
|--------|-------------|
| `New(name, opts...)` | Create a new node |
| `Close()` | Clean shutdown |
| `Spin(ctx)` | Block and process callbacks |
| `Context()` | Get node's context |
| `Logger()` | Get structured logger |
| `NATS()` | Get underlying NATS connection |

### Publisher

Type-safe message publishing with generics.

```go
import "github.com/gort-robotics/gort/pkg/pub"

publisher := pub.New[sensor.Image](n, "camera.image",
    pub.WithQoS(pub.Reliable),
)

img := &sensor.Image{
    Header: std.NewHeader("camera_optical"),
    Width:  640,
    Height: 480,
    Data:   frameData,
}
publisher.Publish(ctx, img)
```

**Publisher Options**

| Option | Description |
|--------|-------------|
| `WithQoS(qos)` | BestEffort or Reliable delivery |
| `WithRetain()` | Keep last message for late subscribers (JetStream) |
| `WithHistory(n)` | Keep last N messages |

### Subscriber

Type-safe message subscription with callbacks.

```go
import "github.com/gort-robotics/gort/pkg/sub"

sub.New[geometry.Twist](n, "cmd_vel", func(msg *geometry.Twist) {
    log.Printf("Received velocity command: linear=%.2f, angular=%.2f",
        msg.Linear.X, msg.Angular.Z)
})
```

**Subscriber Options**

| Option | Description |
|--------|-------------|
| `WithQueueGroup(name)` | Load balance across subscribers |
| `WithBuffer(n)` | Channel buffer size |
| `WithStartTime(t)` | Replay from timestamp (JetStream) |

### Service

Synchronous request/reply pattern.

```go
import "github.com/gort-robotics/gort/pkg/srv"

// Server side
srv.NewServer[HomeRequest, HomeResponse](n, "pantilt.home",
    func(ctx context.Context, req *HomeRequest) (*HomeResponse, error) {
        err := moveToHome()
        return &HomeResponse{Success: err == nil}, err
    },
)

// Client side
client := srv.NewClient[HomeRequest, HomeResponse](n, "pantilt.home")
resp, err := client.Call(ctx, &HomeRequest{}, srv.WithTimeout(5*time.Second))
```

### Action

Long-running tasks with feedback and cancellation.

```go
import "github.com/gort-robotics/gort/pkg/action"

// Server side
action.NewServer[MoveToGoal, MoveToFeedback, MoveToResult](n, "pantilt.move_to",
    func(ctx context.Context, goal *MoveToGoal, fb action.FeedbackSender[MoveToFeedback]) (*MoveToResult, error) {
        for !atTarget() {
            if ctx.Err() != nil {
                return nil, ctx.Err() // cancelled
            }
            fb.Send(&MoveToFeedback{
                CurrentPan:  getCurrentPan(),
                CurrentTilt: getCurrentTilt(),
            })
            time.Sleep(50 * time.Millisecond)
        }
        return &MoveToResult{Success: true}, nil
    },
)

// Client side
client := action.NewClient[MoveToGoal, MoveToFeedback, MoveToResult](n, "pantilt.move_to")
handle, err := client.SendGoal(ctx, &MoveToGoal{Pan: 45.0, Tilt: -10.0})
for fb := range handle.Feedback() {
    log.Printf("Progress: pan=%.1f, tilt=%.1f", fb.CurrentPan, fb.CurrentTilt)
}
result, err := handle.Result()
```

### Parameter Store

Configuration backed by NATS KV.

```go
import "github.com/gort-robotics/gort/pkg/param"

store := param.NewStore(n)

// Set parameters
store.Set("camera.exposure", 100)
store.Set("camera.gain", 1.5)

// Get parameters
exposure, _ := param.Get[int](store, "camera.exposure")

// Watch for changes
store.Watch("camera.*", func(key string, value any) {
    log.Printf("Parameter changed: %s = %v", key, value)
})
```

---

## Message Types

### Standard Messages

Gort defines a minimal set of robotics primitives using Protocol Buffers.

```protobuf
// std.proto
syntax = "proto3";
package gort.std;

message Header {
    int64 timestamp_ns = 1;    // nanoseconds since Unix epoch
    string frame_id = 2;       // coordinate frame
    uint32 seq = 3;            // sequence number
}

message Time {
    int64 sec = 1;
    int32 nsec = 2;
}

message Duration {
    int64 sec = 1;
    int32 nsec = 2;
}
```

```protobuf
// geometry.proto
syntax = "proto3";
package gort.geometry;

import "std.proto";

message Vector3 {
    double x = 1;
    double y = 2;
    double z = 3;
}

message Quaternion {
    double x = 1;
    double y = 2;
    double z = 3;
    double w = 4;
}

message Pose {
    Vector3 position = 1;
    Quaternion orientation = 2;
}

message PoseStamped {
    gort.std.Header header = 1;
    Pose pose = 2;
}

message Twist {
    Vector3 linear = 1;
    Vector3 angular = 2;
}

message TwistStamped {
    gort.std.Header header = 1;
    Twist twist = 2;
}

message Transform {
    Vector3 translation = 1;
    Quaternion rotation = 2;
}

message TransformStamped {
    gort.std.Header header = 1;
    string child_frame_id = 2;
    Transform transform = 3;
}
```

```protobuf
// sensor.proto
syntax = "proto3";
package gort.sensor;

import "std.proto";

message Image {
    gort.std.Header header = 1;
    uint32 height = 2;
    uint32 width = 3;
    string encoding = 4;       // "rgb8", "bgr8", "mono8", "depth16", etc.
    bool is_bigendian = 5;
    uint32 step = 6;           // row length in bytes
    bytes data = 7;
}

message CompressedImage {
    gort.std.Header header = 1;
    string format = 2;         // "jpeg", "png", "h264"
    bytes data = 3;
}

message PointField {
    string name = 1;
    uint32 offset = 2;
    uint32 datatype = 3;
    uint32 count = 4;
}

message PointCloud2 {
    gort.std.Header header = 1;
    uint32 height = 2;
    uint32 width = 3;
    repeated PointField fields = 4;
    bool is_bigendian = 5;
    uint32 point_step = 6;
    uint32 row_step = 7;
    bytes data = 8;
    bool is_dense = 9;
}

message Range {
    gort.std.Header header = 1;
    uint32 radiation_type = 2; // ULTRASOUND=0, INFRARED=1
    float field_of_view = 3;   // radians
    float min_range = 4;
    float max_range = 5;
    float range = 6;
}

message Imu {
    gort.std.Header header = 1;
    gort.geometry.Quaternion orientation = 2;
    gort.geometry.Vector3 angular_velocity = 3;
    gort.geometry.Vector3 linear_acceleration = 4;
}

message JointState {
    gort.std.Header header = 1;
    repeated string name = 2;
    repeated double position = 3;
    repeated double velocity = 4;
    repeated double effort = 5;
}
```

```protobuf
// control.proto
syntax = "proto3";
package gort.control;

import "std.proto";

message JointCommand {
    gort.std.Header header = 1;
    repeated string name = 2;
    repeated double position = 3;    // desired position (optional)
    repeated double velocity = 4;    // desired velocity (optional)
    repeated double effort = 5;      // desired effort/torque (optional)
}

message PanTiltCommand {
    gort.std.Header header = 1;
    double pan = 2;                  // radians
    double tilt = 3;                 // radians
}

message PanTiltState {
    gort.std.Header header = 1;
    double pan = 2;
    double tilt = 3;
    double pan_velocity = 4;
    double tilt_velocity = 5;
}
```

---

## Communication Patterns

### Topic (Pub/Sub)

Many-to-many streaming data.

```
┌──────────┐     gort.sentinel.camera.image     ┌──────────┐
│  Camera  │ ───────────────────────────────────▶│  Fusion  │
│   Node   │                                     │   Node   │
└──────────┘                              ┌─────▶│          │
                                          │      └──────────┘
┌──────────┐     gort.sentinel.tof.depth  │      ┌──────────┐
│   ToF    │ ─────────────────────────────┴─────▶│ Logging  │
│   Node   │                                     │   Node   │
└──────────┘                                     └──────────┘
```

### Service (Request/Reply)

One server, many clients, synchronous.

```
┌──────────┐                                     ┌──────────┐
│  Client  │ ──── Request ───────────────────────▶│  Server  │
│   Node   │ ◀─── Response ──────────────────────│   Node   │
└──────────┘                                     └──────────┘
```

### Action (Goal/Feedback/Result)

Long-running tasks with progress updates.

```
┌──────────┐     Goal                            ┌──────────┐
│  Client  │ ────────────────────────────────────▶│  Server  │
│   Node   │ ◀─── Feedback (stream) ─────────────│   Node   │
│          │ ◀─── Result ────────────────────────│          │
│          │ ──── Cancel ────────────────────────▶│          │
└──────────┘                                     └──────────┘
```

---

## First Project: Gort-Sentinel

### Overview

Gort-Sentinel is a pan-tilt sensor fusion platform that combines:

- **Camera**: RGB imaging (USB webcam or CSI camera)
- **ToF Sensor**: Time-of-Flight depth measurement (VL53L5CX 8x8 array)
- **Pan-Tilt Mount**: Two-axis servo control (hobby servos or Dynamixel)

The platform provides a constrained but complete robotics problem: synchronized multi-sensor data acquisition, real-time motor control, and sensor fusion—all core challenges that validate the Gort framework.

### Why This Project?

| Aspect | Validation |
|--------|------------|
| **Multi-sensor fusion** | Tests timestamping, synchronization, transforms |
| **Real-time control** | Tests low-latency command/state loops |
| **Multiple node types** | Camera driver, ToF driver, motor driver, fusion node |
| **Actions** | Pan-tilt movement is a natural action (goal, feedback, completion) |
| **Services** | Homing, calibration are natural service calls |
| **Parameters** | Camera exposure, servo PID gains, fusion weights |
| **Visualization** | Combined RGB+depth output is immediately understandable |

### System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Gort-Sentinel                                │
│                                                                     │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐            │
│  │   Camera    │    │     ToF     │    │  Pan-Tilt   │            │
│  │   Driver    │    │   Driver    │    │   Driver    │            │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘            │
│         │                  │                  │                    │
│         │ image            │ depth_grid       │ state              │
│         │                  │                  │                    │
│         ▼                  ▼                  ▼                    │
│  ┌─────────────────────────────────────────────────────┐          │
│  │                    NATS Server                       │          │
│  └─────────────────────────────────────────────────────┘          │
│         │                  │                  ▲                    │
│         │                  │                  │ command            │
│         ▼                  ▼                  │                    │
│  ┌─────────────────────────────────┐   ┌─────┴───────┐            │
│  │          Fusion Node            │   │  Teleop /   │            │
│  │  (RGB-D reconstruction)         │   │  Planner    │            │
│  └──────────────┬──────────────────┘   └─────────────┘            │
│                 │                                                  │
│                 ▼ depth_image                                      │
│  ┌─────────────────────────────────┐                              │
│  │        Visualization            │                              │
│  │    (browser / OpenCV GUI)       │                              │
│  └─────────────────────────────────┘                              │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Hardware Bill of Materials

| Component | Suggested Part | Approx. Cost | Notes |
|-----------|----------------|--------------|-------|
| **Camera** | Logitech C920 or Raspberry Pi Camera v3 | $30-50 | 1080p, good Linux support |
| **ToF Sensor** | VL53L5CX breakout (Adafruit, Pololu, or SparkFun) | $25 | 8x8 depth grid, 4m range |
| **Servos** | 2x MG996R or Dynamixel XL330 | $20-80 | Hobby servos for prototype, Dynamixel for production |
| **Pan-Tilt Bracket** | Generic aluminum bracket | $15 | Or 3D print custom |
| **Microcontroller** | Raspberry Pi 4/5 or Jetson Nano | $50-150 | Runs NATS + all nodes |
| **Servo Controller** | PCA9685 (hobby) or U2D2 (Dynamixel) | $10-40 | I2C for hobby servos |

**Total: ~$150-350** depending on component choices.

### Node Specifications

#### Camera Node

**Purpose**: Capture RGB images from camera hardware

**Published Topics**:
| Topic | Type | Rate | Description |
|-------|------|------|-------------|
| `camera.image` | `sensor.Image` | 30 Hz | Raw RGB image |
| `camera.image/compressed` | `sensor.CompressedImage` | 30 Hz | JPEG compressed |
| `camera.info` | `sensor.CameraInfo` | 1 Hz | Intrinsics, distortion |

**Parameters**:
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `camera.device` | string | `/dev/video0` | Video device path |
| `camera.width` | int | 640 | Frame width |
| `camera.height` | int | 480 | Frame height |
| `camera.fps` | int | 30 | Frames per second |
| `camera.exposure` | int | -1 | Exposure (-1 = auto) |

**Implementation Notes**:
- Use V4L2 for Linux camera access (github.com/blackjack/webcam or custom)
- Timestamp at moment of capture, not publication
- Support both raw and compressed output (skip compression if no subscribers)

```go
// Skeleton implementation
type CameraNode struct {
    node   *node.Node
    pub    *pub.Publisher[sensor.Image]
    pubCmp *pub.Publisher[sensor.CompressedImage]
    device *v4l2.Device
}

func (c *CameraNode) Run(ctx context.Context) error {
    ticker := time.NewTicker(time.Second / 30)
    for {
        select {
        case <-ctx.Done():
            return nil
        case <-ticker.C:
            frame, ts := c.device.Capture()
            img := &sensor.Image{
                Header:   std.NewHeaderAt("camera_optical", ts),
                Width:    uint32(frame.Width),
                Height:   uint32(frame.Height),
                Encoding: "rgb8",
                Data:     frame.Data,
            }
            c.pub.Publish(ctx, img)
        }
    }
}
```

#### ToF Node

**Purpose**: Read depth data from VL53L5CX time-of-flight sensor

**Published Topics**:
| Topic | Type | Rate | Description |
|-------|------|------|-------------|
| `tof.depth_grid` | `sensor.DepthGrid` | 15 Hz | 8x8 depth array |
| `tof.pointcloud` | `sensor.PointCloud2` | 15 Hz | 3D points from depth |

**Custom Message Type**:
```protobuf
// sensor.proto (addition)
message DepthGrid {
    gort.std.Header header = 1;
    uint32 rows = 2;           // 8 for VL53L5CX
    uint32 cols = 3;           // 8 for VL53L5CX
    float field_of_view = 4;   // radians (45° for VL53L5CX)
    repeated float distances = 5;  // mm, row-major
    repeated uint32 status = 6;    // per-zone status
}
```

**Parameters**:
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `tof.i2c_bus` | int | 1 | I2C bus number |
| `tof.i2c_addr` | int | 0x29 | I2C address |
| `tof.ranging_freq` | int | 15 | Hz (1-60) |
| `tof.integration_time` | int | 20 | ms |

**Implementation Notes**:
- Use periph.io or go-i2c for I2C access
- VL53L5CX requires firmware upload on boot (~80KB)
- Handle sensor status codes (valid, sigma fail, wrap-around, etc.)

#### Pan-Tilt Node

**Purpose**: Control pan and tilt servo motors

**Published Topics**:
| Topic | Type | Rate | Description |
|-------|------|------|-------------|
| `pantilt.state` | `control.PanTiltState` | 50 Hz | Current pan/tilt angles |
| `pantilt.joint_state` | `sensor.JointState` | 50 Hz | ROS-compatible format |

**Subscribed Topics**:
| Topic | Type | Description |
|-------|------|-------------|
| `pantilt.command` | `control.PanTiltCommand` | Direct angle command |

**Services**:
| Service | Request | Response | Description |
|---------|---------|----------|-------------|
| `pantilt.home` | `Empty` | `Success` | Move to home position |
| `pantilt.set_limits` | `PanTiltLimits` | `Success` | Set software limits |

**Actions**:
| Action | Goal | Feedback | Result | Description |
|--------|------|----------|--------|-------------|
| `pantilt.move_to` | `PanTiltGoal` | `PanTiltFeedback` | `PanTiltResult` | Move to target position |
| `pantilt.scan` | `ScanGoal` | `ScanFeedback` | `ScanResult` | Execute scan pattern |

**Custom Message Types**:
```protobuf
// control.proto (additions)
message PanTiltGoal {
    double pan = 1;            // target pan (radians)
    double tilt = 2;           // target tilt (radians)
    double max_velocity = 3;   // rad/s, 0 = default
}

message PanTiltFeedback {
    double current_pan = 1;
    double current_tilt = 2;
    double error_pan = 3;
    double error_tilt = 4;
}

message PanTiltResult {
    bool success = 1;
    double final_pan = 2;
    double final_tilt = 3;
}

message ScanGoal {
    double pan_min = 1;
    double pan_max = 2;
    double tilt_min = 3;
    double tilt_max = 4;
    double step = 5;           // radians between positions
    double dwell_time = 6;     // seconds at each position
}

message ScanFeedback {
    uint32 current_step = 1;
    uint32 total_steps = 2;
    double current_pan = 3;
    double current_tilt = 4;
}

message ScanResult {
    bool success = 1;
    uint32 positions_visited = 2;
}
```

**Parameters**:
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `pantilt.pan.channel` | int | 0 | PWM channel for pan servo |
| `pantilt.tilt.channel` | int | 1 | PWM channel for tilt servo |
| `pantilt.pan.min_angle` | float | -1.57 | Minimum pan (radians) |
| `pantilt.pan.max_angle` | float | 1.57 | Maximum pan (radians) |
| `pantilt.tilt.min_angle` | float | -0.78 | Minimum tilt (radians) |
| `pantilt.tilt.max_angle` | float | 0.78 | Maximum tilt (radians) |
| `pantilt.pan.home` | float | 0.0 | Home position |
| `pantilt.tilt.home` | float | 0.0 | Home position |

**Implementation Notes**:
- PCA9685 via I2C for hobby servos (12-bit PWM, 50Hz)
- Dynamixel Protocol 2.0 for smart servos
- Implement smooth motion profiles (trapezoidal velocity)
- Track commanded vs actual position (open-loop for hobby servos)

#### Fusion Node

**Purpose**: Combine camera and ToF data into depth-enhanced images

**Subscribed Topics**:
| Topic | Type | Description |
|-------|------|-------------|
| `camera.image` | `sensor.Image` | RGB image |
| `tof.depth_grid` | `sensor.DepthGrid` | 8x8 depth |
| `pantilt.state` | `control.PanTiltState` | Current pose |

**Published Topics**:
| Topic | Type | Rate | Description |
|-------|------|------|-------------|
| `fusion.depth_image` | `sensor.Image` | 30 Hz | Depth-colorized RGB |
| `fusion.pointcloud` | `sensor.PointCloud2` | 15 Hz | Colored 3D points |
| `fusion.rgbd` | `sensor.RGBD` | 15 Hz | Aligned RGB-D |

**Custom Message Type**:
```protobuf
// sensor.proto (addition)
message RGBD {
    gort.std.Header header = 1;
    Image rgb = 2;
    Image depth = 3;         // 16-bit depth in mm
    CameraInfo camera_info = 4;
}
```

**Algorithm**:
1. **Temporal synchronization**: Use approximate time synchronization (within 33ms)
2. **Spatial alignment**: Project ToF zones into camera frame using extrinsic calibration
3. **Depth upsampling**: Interpolate 8x8 ToF grid to camera resolution
4. **Visualization**: Colorize depth using jet colormap, blend with RGB

**Parameters**:
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `fusion.sync_tolerance_ms` | int | 50 | Max time difference for pairing |
| `fusion.extrinsic.tx` | float | 0.0 | ToF→Camera X offset (m) |
| `fusion.extrinsic.ty` | float | 0.0 | ToF→Camera Y offset (m) |
| `fusion.extrinsic.tz` | float | 0.02 | ToF→Camera Z offset (m) |
| `fusion.depth_colormap` | string | "jet" | Colormap for visualization |

### Transform Tree

Gort-Sentinel uses a simple static transform tree:

```
                    base_link
                        │
                        │ (0, 0, 0.05)
                        ▼
                    pan_link
                        │
                        │ rotation: pan angle
                        ▼
                    tilt_link
                        │
                        │ rotation: tilt angle
                        ▼
                   sensor_mount
                     ╱     ╲
    (0, 0.02, 0)   ╱         ╲   (0, -0.02, 0)
                 ╱             ╲
          camera_optical    tof_optical
```

**Implementation**: For Phase 1, implement a simple static transform publisher. Full TF tree support deferred to Phase 2.

```go
// Simple transform publisher
type StaticTransformPublisher struct {
    pub *pub.Publisher[geometry.TransformStamped]
    transforms []geometry.TransformStamped
}

func (s *StaticTransformPublisher) Run(ctx context.Context) error {
    ticker := time.NewTicker(100 * time.Millisecond)
    for {
        select {
        case <-ctx.Done():
            return nil
        case <-ticker.C:
            for _, tf := range s.transforms {
                tf.Header = std.NewHeader(tf.Header.FrameId)
                s.pub.Publish(ctx, &tf)
            }
        }
    }
}
```

### CLI Tool: `gort`

Minimal command-line interface for introspection and debugging.

```bash
# List active topics
$ gort topic list
gort.sentinel.camera.image         [sensor.Image]      29.8 Hz
gort.sentinel.tof.depth_grid       [sensor.DepthGrid]  15.0 Hz
gort.sentinel.pantilt.state        [control.PanTiltState]  49.7 Hz

# Echo messages
$ gort topic echo gort.sentinel.pantilt.state
header:
  timestamp_ns: 1699574932123456789
  frame_id: "tilt_link"
  seq: 1234
pan: 0.523
tilt: -0.175
pan_velocity: 0.0
tilt_velocity: 0.0
---

# Measure rate
$ gort topic hz gort.sentinel.camera.image
average rate: 29.97 Hz
min: 32.1ms, max: 34.8ms, std: 0.8ms

# Publish a message
$ gort topic pub gort.sentinel.pantilt.command '{"pan": 0.5, "tilt": -0.2}'

# Call a service
$ gort service call gort.sentinel.pantilt.home '{}'
success: true

# List parameters
$ gort param list
gort.sentinel.camera.exposure: -1
gort.sentinel.camera.fps: 30
gort.sentinel.pantilt.pan.home: 0.0
...

# Set parameter
$ gort param set gort.sentinel.camera.exposure 50
```

### Directory Structure

```
gort/
├── go.mod
├── go.sum
├── README.md
├── LICENSE                      # Apache 2.0 or MIT
│
├── api/                         # Protocol buffer definitions
│   └── proto/
│       ├── gort/
│       │   ├── std/
│       │   │   └── std.proto
│       │   ├── geometry/
│       │   │   └── geometry.proto
│       │   ├── sensor/
│       │   │   └── sensor.proto
│       │   └── control/
│       │       └── control.proto
│       └── buf.yaml
│
├── pkg/                         # Core library
│   ├── node/
│   │   ├── node.go
│   │   ├── options.go
│   │   └── node_test.go
│   ├── pub/
│   │   ├── publisher.go
│   │   └── publisher_test.go
│   ├── sub/
│   │   ├── subscriber.go
│   │   └── subscriber_test.go
│   ├── srv/
│   │   ├── server.go
│   │   ├── client.go
│   │   └── service_test.go
│   ├── action/
│   │   ├── server.go
│   │   ├── client.go
│   │   └── action_test.go
│   ├── param/
│   │   ├── store.go
│   │   └── store_test.go
│   └── tf/
│       ├── static.go
│       └── buffer.go            # Phase 2
│
├── driver/                      # Hardware driver interfaces
│   ├── camera/
│   │   └── camera.go
│   ├── tof/
│   │   └── tof.go
│   └── servo/
│       └── servo.go
│
├── drivers/                     # Concrete implementations
│   ├── v4l2/
│   │   └── camera.go
│   ├── vl53l5cx/
│   │   └── tof.go
│   └── pca9685/
│       └── servo.go
│
├── cmd/
│   ├── gort/                    # CLI tool
│   │   └── main.go
│   └── sentinel/                # Sentinel nodes
│       ├── camera/
│       │   └── main.go
│       ├── tof/
│       │   └── main.go
│       ├── pantilt/
│       │   └── main.go
│       └── fusion/
│           └── main.go
│
├── examples/
│   └── sentinel/
│       ├── README.md
│       ├── docker-compose.yaml
│       └── launch.sh
│
└── docs/
    ├── getting-started.md
    ├── concepts.md
    ├── tutorials/
    │   └── sentinel.md
    └── api/
        └── messages.md
```

### Development Milestones

#### Milestone 1: Core Framework (Week 1-2)

**Goal**: Basic pub/sub working with one node

- [ ] Project scaffolding (go.mod, directory structure)
- [ ] Protocol buffer definitions (std, geometry, sensor)
- [ ] `pkg/node`: Node lifecycle, NATS connection
- [ ] `pkg/pub`: Generic publisher
- [ ] `pkg/sub`: Generic subscriber with callbacks
- [ ] Basic test: Two nodes exchanging messages

**Deliverable**: `examples/hello/` with publisher and subscriber

#### Milestone 2: Camera Node (Week 2-3)

**Goal**: Streaming camera images over NATS

- [ ] `driver/camera`: Camera interface
- [ ] `drivers/v4l2`: V4L2 implementation
- [ ] `cmd/sentinel/camera`: Camera node
- [ ] `cmd/gort`: Basic CLI with `topic list`, `topic echo`
- [ ] Verify ~30 fps image streaming

**Deliverable**: Camera node running, viewable via CLI

#### Milestone 3: Pan-Tilt Control (Week 3-4)

**Goal**: Servo control with command/state topics

- [ ] `driver/servo`: Servo interface
- [ ] `drivers/pca9685`: PCA9685 PWM driver
- [ ] `cmd/sentinel/pantilt`: Pan-tilt node
- [ ] `pkg/srv`: Service client/server
- [ ] `pantilt.home` service working

**Deliverable**: Pan-tilt responds to commands, reports state

#### Milestone 4: Actions (Week 4-5)

**Goal**: Long-running pan-tilt movements with feedback

- [ ] `pkg/action`: Action client/server
- [ ] `pantilt.move_to` action implemented
- [ ] Smooth motion profiles
- [ ] Cancellation support

**Deliverable**: Action demo moving pan-tilt with progress feedback

#### Milestone 5: ToF Sensor (Week 5-6)

**Goal**: VL53L5CX depth data streaming

- [ ] `driver/tof`: ToF interface
- [ ] `drivers/vl53l5cx`: VL53L5CX I2C driver
- [ ] Firmware upload on initialization
- [ ] `cmd/sentinel/tof`: ToF node
- [ ] Verify ~15 fps depth grid streaming

**Deliverable**: ToF node running, depth data visible via CLI

#### Milestone 6: Sensor Fusion (Week 6-8)

**Goal**: Combined RGB-D output

- [ ] `cmd/sentinel/fusion`: Fusion node
- [ ] Temporal synchronization
- [ ] Extrinsic calibration (manual for now)
- [ ] Depth upsampling (bilinear interpolation)
- [ ] Depth visualization (colormap overlay)
- [ ] `pkg/param`: Parameter store with NATS KV

**Deliverable**: Fusion node producing colorized depth images

#### Milestone 7: Polish (Week 8-10)

**Goal**: Documentation, testing, demo

- [ ] Unit tests for all packages (>70% coverage)
- [ ] Integration test with mock hardware
- [ ] `docs/getting-started.md`
- [ ] `docs/tutorials/sentinel.md`
- [ ] Demo video / GIF
- [ ] Clean up CLI (`gort topic pub`, `gort service call`, `gort param`)

**Deliverable**: Public-ready repository

---

## Technical Decisions

### Serialization: Protocol Buffers

**Rationale**:
- Mature, fast, multi-language
- Schema evolution with backward compatibility
- Smaller wire format than JSON
- Good tooling (buf, protoc)

**Alternatives Considered**:
- **FlatBuffers**: Zero-copy, but more complex API
- **MessagePack**: Schema-less, harder to evolve
- **Cap'n Proto**: Good, but smaller ecosystem

**Decision**: Start with protobuf. Can add FlatBuffers for performance-critical paths (images) later.

### QoS Mapping to NATS

| Gort QoS | NATS Implementation |
|----------|---------------------|
| BestEffort | Core NATS pub/sub |
| Reliable | JetStream with ack |
| Retained | JetStream with last-value retention |
| History(n) | JetStream with limit |
| Transient | Core NATS (default) |

### Image Transport

Images are large. Options:

1. **Compress in publisher**: JPEG/PNG, universally supported
2. **Shared memory**: Fast but local-only
3. **Chunking**: Split large messages
4. **Separate transport**: Use different mechanism for bulk data

**Phase 1 Decision**: Compress to JPEG for transport. Add shared memory optimization in Phase 2 for same-host subscribers.

### Time Synchronization

For sensor fusion, timestamps must be comparable.

**Phase 1**: Use system clock (`time.Now()`), assume NTP sync on all nodes.

**Phase 2**: Implement clock offset estimation via NATS request/reply.

### Error Handling

Go idioms:
- Return errors, don't panic
- Use `context.Context` for cancellation
- Wrap errors with context: `fmt.Errorf("camera capture: %w", err)`

### Logging

Use `log/slog` (Go 1.21+):
- Structured logging built into stdlib
- Zero external dependencies
- JSON output for production

```go
n.Logger().Info("frame captured",
    "width", frame.Width,
    "height", frame.Height,
    "latency_ms", time.Since(start).Milliseconds(),
)
```

### Testing

- **Unit tests**: Mock NATS with `github.com/nats-io/nats-server/v2/test`
- **Integration tests**: In-memory NATS server
- **Hardware tests**: Marked with build tag `//go:build hardware`

```go
func TestPublisherSubscriber(t *testing.T) {
    s := test.RunDefaultServer()
    defer s.Shutdown()

    n1, _ := node.New("pub", node.WithNATS(s.ClientURL()))
    n2, _ := node.New("sub", node.WithNATS(s.ClientURL()))
    defer n1.Close()
    defer n2.Close()

    received := make(chan *sensor.Image, 1)
    sub.New[sensor.Image](n2, "test.image", func(img *sensor.Image) {
        received <- img
    })

    pub := pub.New[sensor.Image](n1, "test.image")
    pub.Publish(context.Background(), &sensor.Image{Width: 640})

    select {
    case img := <-received:
        assert.Equal(t, uint32(640), img.Width)
    case <-time.After(time.Second):
        t.Fatal("timeout waiting for message")
    }
}
```

---

## Development Roadmap

### Phase 1: Gort-Sentinel (Weeks 1-10)

Prove the concept with a working robot.

| Week | Focus | Deliverable |
|------|-------|-------------|
| 1-2 | Core framework | Pub/sub working |
| 2-3 | Camera node | Streaming images |
| 3-4 | Pan-tilt control | Command/state loop |
| 4-5 | Actions | move_to with feedback |
| 5-6 | ToF sensor | Depth streaming |
| 6-8 | Fusion | RGB-D output |
| 8-10 | Polish | Docs, tests, demo |

### Phase 2: Framework Hardening (Months 3-6)

Generalize and extend.

- [ ] Full transform tree (TF equivalent)
- [ ] Bag recording/playback (JetStream)
- [ ] Shared memory transport for local nodes
- [ ] WebSocket bridge for browser visualization
- [ ] Launch system (YAML-based node graph)
- [ ] More drivers (IMU, LIDAR, Dynamixel)
- [ ] Python bindings (optional)

### Phase 3: Community Release (Month 6+)

Open source and grow.

- [ ] Public GitHub repository
- [ ] Documentation site
- [ ] Example robots
- [ ] Community drivers
- [ ] Conference talk / blog posts

---

## Comparison with Alternatives

| Feature | Gort | ROS 2 | YARP | Viam |
|---------|------|-------|------|------|
| **Language** | Go | C++/Python | C++ | Go |
| **Middleware** | NATS | DDS | Custom | Custom |
| **Deployment** | Single binary | Complex | Moderate | Cloud-centric |
| **Learning curve** | Low | High | Moderate | Low |
| **Ecosystem** | New | Massive | Niche | Growing |
| **Real-time** | Soft | Soft/Hard | Soft | Soft |
| **License** | Apache 2.0 | Apache 2.0 | LGPL | AGPL |
| **Cloud integration** | Native (NATS) | Limited | Limited | Native |

---

## Conclusion

Gort aims to be the robotics framework that should have existed: simple enough to understand in an afternoon, powerful enough to build real robots. By building on NATS's proven messaging infrastructure and Go's practical design, we avoid the complexity that has made ROS 2 adoption painful.

Gort-Sentinel provides the perfect first project: constrained enough to complete in weeks, complex enough to validate the core framework. Success means a working RGB-D pan-tilt platform and a solid foundation for the Gort ecosystem.

---

## Appendix A: NATS Quick Reference

### Running NATS

```bash
# Install
go install github.com/nats-io/nats-server/v2@latest

# Run with JetStream
nats-server -js

# Or with Docker
docker run -p 4222:4222 -p 8222:8222 nats:latest -js
```

### NATS CLI

```bash
# Install
go install github.com/nats-io/natscli/nats@latest

# Subscribe
nats sub "gort.>"

# Publish
nats pub gort.test "hello"

# Request/reply
nats request gort.service.test '{"foo": "bar"}'

# JetStream streams
nats stream ls
nats stream info GORT

# KV store
nats kv add PARAMS
nats kv put PARAMS camera.exposure 100
nats kv get PARAMS camera.exposure
```

---

## Appendix B: Hardware Setup Notes

### VL53L5CX I2C Connection

```
Raspberry Pi          VL53L5CX
-----------          ---------
3.3V (pin 1)   -->   VIN
GND (pin 6)    -->   GND
SDA (pin 3)    -->   SDA
SCL (pin 5)    -->   SCL
GPIO17 (pin 11) -->  LPn (optional, for reset)
```

Enable I2C: `sudo raspi-config` → Interfacing Options → I2C

Verify: `i2cdetect -y 1` should show device at 0x29

### PCA9685 Servo Connection

```
Raspberry Pi          PCA9685           Servos
-----------          -------           ------
3.3V (pin 1)   -->   VCC
GND (pin 6)    -->   GND
SDA (pin 3)    -->   SDA
SCL (pin 5)    -->   SCL
                     V+ (5-6V) <------ External power
                     CH0 -------------> Pan servo signal
                     CH1 -------------> Tilt servo signal
```

**Important**: Servos need external 5-6V power supply. Do not power from Pi.

---

## Appendix C: References

- [NATS Documentation](https://docs.nats.io/)
- [Protocol Buffers](https://protobuf.dev/)
- [VL53L5CX Datasheet](https://www.st.com/resource/en/datasheet/vl53l5cx.pdf)
- [PCA9685 Datasheet](https://www.nxp.com/docs/en/data-sheet/PCA9685.pdf)
- [ROS 2 Design](https://design.ros2.org/)
- [Viam Documentation](https://docs.viam.com/)
