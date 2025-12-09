# Appendices

## Appendix A: Command Reference

### GoRAI Scripts

| Script | Purpose |
|--------|---------|
| `scripts/start.sh` | Start NATS and development services |
| `scripts/stop.sh` | Stop all services |
| `scripts/hello.sh` | Run hello-sensor example |

### NATS CLI Commands

| Command | Purpose |
|---------|---------|
| `nats sub ">"` | Subscribe to all messages |
| `nats sub "gorai.>"` | Subscribe to GoRAI messages |
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


## Appendix C: Hardware Compatibility

### Supported SBCs

| Board | CPU | RAM | NPU | GPIO | Notes |
|-------|-----|-----|-----|------|-------|
| Raspberry Pi 5 | Cortex-A76 | 8GB | No | Yes | Best starter |
| Orange Pi 5 | RK3588S | 8-16GB | 6 TOPS | Yes | Best for AI |
| Jetson Orin Nano | Cortex-A78 | 8GB | GPU | Yes | CUDA support |
| BeagleBone AI-64 | TDA4VM | 4GB | 8 TOPS | Yes | Real-time PRUs |
| Rock 5B | RK3588 | 8-16GB | 6 TOPS | Yes | PCIe support |

### Supported Microcontrollers (TinyGo)

| Board | CPU | RAM | Flash | Notes |
|-------|-----|-----|-------|-------|
| RP2040 (Pico) | Cortex-M0+ | 264KB | 2MB | Dual core |
| ESP32-C3 | RISC-V | 400KB | 4MB | WiFi/BLE |
| STM32F4 | Cortex-M4 | 128KB+ | 512KB+ | Industrial |

### Common Sensors

| Sensor | Interface | GoRAI Support |
|--------|-----------|---------------|
| MPU6050 | I2C | Example available |
| BME280 | I2C/SPI | Example available |
| GPS (NMEA) | UART | Example available |
| HC-SR04 | GPIO | Example available |
| Encoders | GPIO | Included |

### Common Actuators

| Actuator | Interface | GoRAI Support |
|----------|-----------|---------------|
| DC Motors | PWM+GPIO | Included |
| Steppers | GPIO | Example available |
| RC Servos | PWM | Included |
| DRV8833 | PWM+GPIO | Example available |


## Appendix D: Troubleshooting

### NATS Connectivity

**Problem**: Can't connect to NATS server

```
failed to connect to NATS: nats: no servers available for connection
```

**Solutions**:
1. Check NATS server is running: `nats server info`
2. Check URL: default is `nats://localhost:4222`
3. Check firewall allows port 4222
4. Verify network connectivity

**Problem**: JetStream not available

```
JetStream not available
```

**Solutions**:
1. Start NATS with `-js` flag
2. Check JetStream is enabled in config

### Hardware Access Issues

**Problem**: Permission denied for GPIO

```
open /sys/class/gpio/export: permission denied
```

**Solutions**:
1. Add user to gpio group: `sudo usermod -aG gpio $USER`
2. Logout and login
3. Or run with sudo (not recommended for production)

**Problem**: I2C device not found

```
no I2C device at address 0x68
```

**Solutions**:
1. Check wiring
2. Verify I2C enabled: `sudo raspi-config`
3. Scan bus: `i2cdetect -y 1`
4. Check address in datasheet

### Build Problems

**Problem**: Module not found

```
cannot find module providing package github.com/gorai/gorai/...
```

**Solutions**:
1. Run `go mod download`
2. Check Go version ≥ 1.21
3. Verify network access to github.com

**Problem**: CGo errors

```
cgo: C compiler "gcc" not found
```

**Solutions**:
1. Install build-essential: `sudo apt install build-essential`
2. Or use pure Go alternatives


## Appendix E: Glossary

| Term | Definition |
|------|------------|
| **Actuator** | Component that performs physical actions (motors, servos) |
| **Component** | Hardware abstraction in GoRAI |
| **Fake** | Test implementation that simulates real hardware |
| **JetStream** | NATS persistence layer |
| **Node** | GoRAI process managing resources and NATS connection |
| **NPU** | Neural Processing Unit for ML inference |
| **NWC** | Network Wrapper Client - consumes remote resources |
| **NWS** | Network Wrapper Server - exposes resources |
| **QoS** | Quality of Service for message delivery |
| **Resource** | Base interface for components and services |
| **Sensor** | Component that provides readings |
| **Service** | Software capability (vision, navigation) |
| **TinyGo** | Go compiler for microcontrollers |
| **Topic** | NATS subject for pub/sub messaging |

---

*End of GoRAI Book*
