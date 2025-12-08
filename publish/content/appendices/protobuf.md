# Protocol Buffers Reference

GoRAI uses Protocol Buffers for message serialization. This appendix provides reference for standard message types.

## Standard Messages (gorai/std)

```protobuf
message Header {
    Timestamp stamp = 1;
    string frame_id = 2;
    uint32 seq = 3;
}

message Timestamp {
    int64 seconds = 1;
    int32 nanos = 2;
}

message Duration {
    int64 seconds = 1;
    int32 nanos = 2;
}

message DiagnosticStatus {
    uint32 level = 1;      // OK=0, WARN=1, ERROR=2, STALE=3
    string name = 2;
    string message = 3;
    string hardware_id = 4;
}
```

## Geometry Messages (gorai/geometry)

```protobuf
message Vector3 {
    double x = 1;
    double y = 2;
    double z = 3;
}

message Point {
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
    Point position = 1;
    Quaternion orientation = 2;
}

message Twist {
    Vector3 linear = 1;
    Vector3 angular = 2;
}

message Transform {
    Vector3 translation = 1;
    Quaternion rotation = 2;
}
```

## Sensor Messages (gorai/sensor)

### IMU

```protobuf
message Imu {
    std.Header header = 1;
    geometry.Quaternion orientation = 2;
    repeated double orientation_covariance = 3;
    geometry.Vector3 angular_velocity = 4;
    repeated double angular_velocity_covariance = 5;
    geometry.Vector3 linear_acceleration = 6;
    repeated double linear_acceleration_covariance = 7;
}
```

### Image

```protobuf
message Image {
    std.Header header = 1;
    uint32 height = 2;
    uint32 width = 3;
    string encoding = 4;
    uint32 step = 5;
    bytes data = 6;
}
```

### Laser Scan

```protobuf
message LaserScan {
    std.Header header = 1;
    float angle_min = 2;
    float angle_max = 3;
    float angle_increment = 4;
    float time_increment = 5;
    float scan_time = 6;
    float range_min = 7;
    float range_max = 8;
    repeated float ranges = 9;
    repeated float intensities = 10;
}
```

### GPS/NavSat

```protobuf
message NavSatFix {
    std.Header header = 1;
    int32 status = 2;
    uint32 service = 3;
    double latitude = 4;
    double longitude = 5;
    double altitude = 6;
    repeated double position_covariance = 7;
    uint32 position_covariance_type = 8;
}
```

## Working with Protobuf

### Generating Go Code

```bash
make proto
```

### Using Messages in Go

```go
import "github.com/gorai/gorai/api/sensor"

msg := &sensor.Image{
    Header: &std.Header{
        Stamp:   timestamppb.Now(),
        FrameId: "camera_front",
    },
    Width:    640,
    Height:   480,
    Encoding: "rgb8",
    Step:     640 * 3,
    Data:     frameData,
}
```
