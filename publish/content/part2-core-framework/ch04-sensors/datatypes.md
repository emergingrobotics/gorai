## 4.3 Sensor Data Types (Protocol Buffers)

While `Readings()` returns dynamic maps, structured sensor data uses Protocol Buffers for efficient serialization and type safety.

### The sensor.proto Definitions

GoRAI defines standard sensor messages in `api/proto/gorai/sensor/sensor.proto`:

```protobuf
syntax = "proto3";
package gorai.sensor;

import "gorai/std/std.proto";
import "gorai/geometry/geometry.proto";

// Imu - Inertial Measurement Unit data
message Imu {
    std.Header header = 1;

    geometry.Quaternion orientation = 2;
    repeated double orientation_covariance = 3;

    geometry.Vector3 angular_velocity = 4;
    repeated double angular_velocity_covariance = 5;

    geometry.Vector3 linear_acceleration = 6;
    repeated double linear_acceleration_covariance = 7;
}

// Image - Raw camera image
message Image {
    std.Header header = 1;
    uint32 height = 2;
    uint32 width = 3;
    string encoding = 4;    // "rgb8", "bgr8", "mono8", etc.
    uint32 step = 5;        // Row length in bytes
    bytes data = 6;
}

// LaserScan - 2D laser scan
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

// Range - Single distance measurement
message Range {
    std.Header header = 1;
    uint32 radiation_type = 2;   // ULTRASOUND=0, INFRARED=1
    float field_of_view = 3;
    float min_range = 4;
    float max_range = 5;
    float range = 6;
}

// NavSatFix - GPS position
message NavSatFix {
    std.Header header = 1;

    int32 status = 2;           // STATUS_NO_FIX=-1, FIX=0, SBAS=1, GBAS=2
    uint32 service = 3;         // SERVICE_GPS=1, GLONASS=2, ...

    double latitude = 4;
    double longitude = 5;
    double altitude = 6;

    repeated double position_covariance = 7;
    uint32 position_covariance_type = 8;
}

// BatteryState - Power source status
message BatteryState {
    std.Header header = 1;
    float voltage = 2;
    float current = 3;
    float charge = 4;
    float capacity = 5;
    float design_capacity = 6;
    float percentage = 7;
    uint32 power_supply_status = 8;
    uint32 power_supply_health = 9;
    uint32 power_supply_technology = 10;
    bool present = 11;
}
```

### Timestamps and Headers

Every sensor message includes a Header:

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
```

**stamp**: When the measurement was taken (not when it was published)
**frame_id**: Coordinate frame reference (e.g., "imu_link", "camera_optical")
**seq**: Sequence number for ordering and gap detection

Usage in Go:
```go
import "google.golang.org/protobuf/types/known/timestamppb"

msg := &sensor.Imu{
    Header: &std.Header{
        Stamp:   timestamppb.Now(),
        FrameId: "imu_link",
        Seq:     atomic.AddUint32(&seq, 1),
    },
    LinearAcceleration: &geometry.Vector3{
        X: accel.X,
        Y: accel.Y,
        Z: accel.Z,
    },
    // ...
}
```

### Covariance Matrices for Uncertainty

Sensor data is uncertain. Covariance matrices express this uncertainty:

```go
// 3x3 covariance matrix as 9 elements, row-major
// [0 1 2]
// [3 4 5]
// [6 7 8]

msg.OrientationCovariance = []float64{
    0.01, 0,    0,     // Roll variance and correlations
    0,    0.01, 0,     // Pitch variance and correlations
    0,    0,    0.02,  // Yaw variance and correlations
}
```

**Diagonal elements**: Variance in each dimension
**Off-diagonal elements**: Correlation between dimensions

For uncorrelated sensors, use a diagonal matrix. For unknown covariance, use -1 in the first element as a flag.

*Cross-reference: See Chapter 3 for how sensor data flows over NATS.*
