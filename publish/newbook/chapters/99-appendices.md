# Appendices

## Appendix A: Command Reference

### Gorai Scripts

| Script | Purpose |
|--------|---------|
| `scripts/start.sh` | Start NATS and development services |
| `scripts/stop.sh` | Stop all services |
| `scripts/hello.sh` | Run hello-sensor example |

### NATS CLI Commands

| Command | Purpose |
|---------|---------|
| `nats sub ">"` | Subscribe to all messages |
| `nats sub "gorai.>"` | Subscribe to Gorai messages |
| `nats pub TOPIC DATA` | Publish message |
| `nats request TOPIC DATA` | Request/reply |
| `nats server info` | Server information |
| `nats stream list` | List JetStream streams |
| `nats stream view NAME` | View stream messages |
| `nats consumer list STREAM` | List consumers |

### Go Commands

| Command | Purpose |
|---------|---------|
| `go build ./...` | Build all packages |
| `go test ./...` | Run unit tests |
| `go test -tags=component ./...` | Run component tests |
| `go test -race ./...` | Test with race detector |
| `go test -cover ./...` | Test with coverage |

### Makefile Targets

| Target | Purpose |
|--------|---------|
| `make build` | Build all binaries |
| `make test` | Run unit tests |
| `make test-quick` | Unit + component tests |
| `make test-all` | All test levels |
| `make proto` | Generate proto code |
| `make lint` | Run linters |


## Appendix B: Protocol Buffer Reference

### Standard Messages (gorai/std)

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

### Geometry Messages (gorai/geometry)

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

### Sensor Messages (gorai/sensor)

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

message Image {
    std.Header header = 1;
    uint32 height = 2;
    uint32 width = 3;
    string encoding = 4;
    uint32 step = 5;
    bytes data = 6;
}

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

### Working with Protobuf

Generating Go Code:

```bash
make proto
```

Using Messages in Go:

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


## Appendix C: NATS Topics Reference

### Topic Hierarchy

Gorai uses a hierarchical topic structure:

```
gorai.<domain>.<component>.<type>.<instance>
```

### Standard Prefixes

| Prefix | Purpose |
|--------|---------|
| `gorai.sensor` | Sensor data publications |
| `gorai.actuator` | Actuator commands |
| `gorai.service` | Service requests/responses |
| `gorai.telemetry` | System telemetry |
| `gorai.config` | Configuration updates |
| `gorai.health` | Health status |

### Topic Examples

| Topic | Description |
|-------|-------------|
| `gorai.sensor.temperature.cpu.main` | CPU temperature from main sensor |
| `gorai.sensor.camera.front.image` | Image from front camera |
| `gorai.actuator.motor.drive.left` | Left drive motor commands |
| `gorai.service.vision.detect` | Object detection requests |
| `gorai.telemetry.node.hello-sensor` | Node telemetry |

### Wildcards

NATS supports two wildcards:

- `*` - Matches a single token
- `>` - Matches one or more tokens

**Subscription Patterns**:

| Pattern | Matches |
|---------|---------|
| `gorai.sensor.>` | All sensor messages |
| `gorai.*.camera.>` | Camera messages from any domain |
| `gorai.sensor.*.cpu.*` | All CPU sensor messages |


## Appendix D: Hardware Compatibility

### Supported SBCs

| Board | CPU | RAM | NPU | GPIO | Notes |
|-------|-----|-----|-----|------|-------|
| Raspberry Pi 5 | Cortex-A76 | 8GB | No | Yes | Best starter |
| Orange Pi 5 | RK3588S | 8-16GB | 6 TOPS | Yes | Best for AI |
| Jetson Orin Nano | Cortex-A78 | 8GB | GPU | Yes | CUDA support |
| BeagleBone AI-64 | TDA4VM | 4GB | 8 TOPS | Yes | Real-time PRUs |
| Rock 5B | RK3588 | 8-16GB | 6 TOPS | Yes | PCIe support |

### Recommended Configuration

**For learning/development**:

- Raspberry Pi 5 (8GB)
- USB webcam
- Basic sensors (temperature, distance)

**For AI/ML workloads**:

- Orange Pi 5 or Jetson Orin Nano
- USB3 camera or CSI camera
- NPU-accelerated inference

### Supported Microcontrollers (TinyGo)

| Board | CPU | RAM | Flash | Notes |
|-------|-----|-----|-------|-------|
| RP2040 (Pico) | Cortex-M0+ | 264KB | 2MB | Dual core |
| ESP32-C3 | RISC-V | 400KB | 4MB | WiFi/BLE |
| STM32F4 | Cortex-M4 | 128KB+ | 512KB+ | Industrial |

### Common Sensors

| Sensor | Interface | Gorai Support |
|--------|-----------|---------------|
| MPU6050 | I2C | Example available |
| BME280 | I2C/SPI | Example available |
| GPS (NMEA) | UART | Example available |
| HC-SR04 | GPIO | Example available |
| Encoders | GPIO | Included |
| VL53L0X | I2C | Example available |
| Camera (USB) | V4L2 | Included |
| Camera (CSI) | MMAL/V4L2 | Platform-specific |

### Common Actuators

| Actuator | Interface | Gorai Support |
|----------|-----------|---------------|
| DC Motors | PWM+GPIO | Included |
| Steppers | GPIO | Example available |
| RC Servos | PWM | Included |
| DRV8833 | PWM+GPIO | Example available |
| PCA9685 | I2C | Example available |

### Motor Controllers

| Controller | Motors | Interface | Notes |
|------------|--------|-----------|-------|
| DRV8833 | 2 DC | PWM+GPIO | Low cost |
| TB6612FNG | 2 DC | PWM+GPIO | Higher current |
| Roboclaw | 2 DC | Serial | Encoder support |
| ODrive | 2 BLDC | USB/CAN | High performance |

### Camera Options

| Camera | Resolution | Interface | FPS | Notes |
|--------|------------|-----------|-----|-------|
| Generic USB | 1080p | USB | 30 | Easy setup |
| Pi Camera v3 | 12MP | CSI | 60 | RPi only |
| OAK-D Lite | 4K | USB | 30 | Stereo + AI |
| RealSense D435 | 1080p | USB | 90 | Depth sensing |


## Appendix E: Troubleshooting

### NATS Connectivity

**Problem**: Can't connect to NATS server

```
failed to connect to NATS: nats: no servers available for connection
```

**Solutions**:

1. Check NATS server is running: `nats server info`
2. Check URL: default is `nats://localhost:4222`
3. Check firewall allows port 4222
4. Verify network connectivity: `ping localhost`

**Problem**: JetStream not available

```
JetStream not available
```

**Solutions**:

1. Start NATS with `-js` flag: `nats-server -js`
2. Check JetStream is enabled in config file
3. Verify storage directory is writable

**Problem**: Slow message delivery

**Possible causes**:

1. Network congestion
2. Large message payloads
3. Consumer processing too slow

**Solutions**:

1. Use QoS settings appropriate for your use case
2. Compress large payloads
3. Add buffering or parallel consumers

### Hardware Access Issues

**Problem**: Permission denied for GPIO

```
open /sys/class/gpio/export: permission denied
```

**Solutions**:

1. Add user to gpio group: `sudo usermod -aG gpio $USER`
2. Logout and login for group changes to take effect
3. Or run with sudo (not recommended for production)

**Problem**: I2C device not found

```
no I2C device at address 0x68
```

**Solutions**:

1. Check wiring - verify connections
2. Verify I2C enabled: `sudo raspi-config` → Interfaces → I2C
3. Scan bus: `i2cdetect -y 1`
4. Check device address in datasheet (some have configurable addresses)
5. Check for address conflicts with other devices

**Problem**: Camera not accessible

```
cannot open video device /dev/video0
```

**Solutions**:

1. Check camera is connected: `ls /dev/video*`
2. Add user to video group: `sudo usermod -aG video $USER`
3. Check another application isn't using the camera
4. Verify driver is loaded: `lsmod | grep uvc`

### Build Problems

**Problem**: Module not found

```
cannot find module providing package github.com/gorai/gorai/...
```

**Solutions**:

1. Run `go mod download`
2. Check Go version ≥ 1.21: `go version`
3. Verify network access to github.com
4. Clear module cache: `go clean -modcache`

**Problem**: CGo errors

```
cgo: C compiler "gcc" not found
```

**Solutions**:

1. Install build tools: `sudo apt install build-essential`
2. Or use pure Go alternatives (check package documentation)
3. Set `CGO_ENABLED=0` for pure Go builds (if supported)

**Problem**: TinyGo compilation errors

```
error: could not find wasm-opt
```

**Solutions**:

1. Install binaryen: `sudo apt install binaryen`
2. Update TinyGo to latest version
3. Check target is supported: `tinygo targets`

### Runtime Issues

**Problem**: High CPU usage

**Possible causes**:

1. Tight polling loops
2. Unthrottled message processing
3. Memory allocation in hot paths

**Solutions**:

1. Add appropriate sleep/delays in loops
2. Use channel buffering
3. Profile with `go tool pprof`

**Problem**: Memory leaks

**Possible causes**:

1. Goroutines not terminating
2. Subscriptions not cleaned up
3. Accumulated data structures

**Solutions**:

1. Ensure proper context cancellation
2. Call `Close()` on all resources
3. Use `runtime/pprof` to profile memory

**Problem**: Message ordering issues

**NATS guarantees**:

- Per-publisher ordering preserved
- No global ordering across publishers

**Solutions**:

1. Use sequence numbers in messages
2. Use JetStream for ordering guarantees
3. Design for eventual consistency


## Appendix F: Glossary

| Term | Definition |
|------|------------|
| **Actuator** | Component that performs physical actions (motors, servos, relays) |
| **Base** | Actuator representing a mobile platform (differential drive, holonomic) |
| **Behavior** | High-level robot action composed of sensor reads and actuator commands |
| **Component** | Hardware abstraction in Gorai - sensors, actuators, cameras |
| **Consumer** | NATS entity that receives messages from a stream |
| **Coordinator** | Module that orchestrates behaviors and manages robot state |
| **DDS** | Data Distribution Service - middleware used by ROS 2 |
| **Fake** | Test implementation that simulates real hardware behavior |
| **Frame** | Coordinate reference for spatial data |
| **GPIO** | General Purpose Input/Output pins on SBCs |
| **I2C** | Inter-Integrated Circuit - serial communication protocol |
| **JetStream** | NATS persistence and streaming layer |
| **Node** | Gorai process managing resources and NATS connection |
| **NPU** | Neural Processing Unit - accelerator for ML inference |
| **NWC** | Network Wrapper Client - consumes remote resources over NATS |
| **NWS** | Network Wrapper Server - exposes local resources over NATS |
| **Odometry** | Estimation of robot position from sensor data |
| **Pose** | Position and orientation in 3D space |
| **Proto / Protobuf** | Protocol Buffers - binary serialization format |
| **PWM** | Pulse Width Modulation - technique for analog-like signals |
| **QoS** | Quality of Service - delivery guarantees for messages |
| **Resource** | Base interface for all Gorai components and services |
| **SBC** | Single Board Computer (Raspberry Pi, etc.) |
| **Sensor** | Component that provides readings from the physical world |
| **Service** | Software capability exposing request/reply functionality |
| **SPI** | Serial Peripheral Interface - high-speed serial protocol |
| **Stream** | JetStream persistent message storage |
| **Subject** | NATS term for topic - the address for message routing |
| **TinyGo** | Go compiler for microcontrollers |
| **Topic** | NATS subject for pub/sub messaging |
| **Transform** | Translation and rotation between coordinate frames |
| **Twist** | Linear and angular velocity |
| **UART** | Universal Asynchronous Receiver-Transmitter - serial protocol |
| **V4L2** | Video4Linux2 - Linux camera API |

### Abbreviations

| Abbreviation | Meaning |
|--------------|---------|
| AI | Artificial Intelligence |
| API | Application Programming Interface |
| BLDC | Brushless DC (motor) |
| CLI | Command Line Interface |
| CSI | Camera Serial Interface |
| DC | Direct Current |
| FPS | Frames Per Second |
| IMU | Inertial Measurement Unit |
| LiDAR | Light Detection and Ranging |
| MCU | Microcontroller Unit |
| ML | Machine Learning |
| NMEA | National Marine Electronics Association (GPS protocol) |
| RGB | Red Green Blue (color encoding) |
| SDK | Software Development Kit |
| ToF | Time of Flight (distance sensor) |
| USB | Universal Serial Bus |


## Getting Help

If you can't resolve an issue:

1. Check GitHub Issues
2. Search Discussions
3. Include in bug reports:
   - Go version
   - NATS server version
   - Hardware platform
   - Minimal reproduction code

---

*End of Gorai Book*
