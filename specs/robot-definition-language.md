# Robot Definition Language (RDL) Specification

**Version:** 1.0
**Status:** Draft
**Last Updated:** 2024

## 1. Overview

The Robot Definition Language (RDL) is a JSON-based configuration format that defines the software architecture of a Gorai robot. RDL specifies what components and services a robot has, how they are configured, and their dependencies.

### 1.1 Scope

RDL defines:
- Robot identity and namespace
- NATS connection configuration
- Component instances (sensors, actuators, infrastructure)
- Service instances (vision, navigation, SLAM, etc.)
- Dependencies between components and services
- Logging configuration

RDL does **NOT** define:
- Physical robot geometry (use URDF/SDF for that)
- Kinematic chains or joint limits
- Visual or collision meshes
- Simulation parameters

### 1.2 Design Principles

1. **Explicit over implicit**: All configuration is visible in the file
2. **Validation at load time**: Invalid configs fail early with clear errors
3. **Registry-driven**: Component/service types must be registered in code
4. **Dependency-aware**: Services declare dependencies, loaded in order
5. **Environment-friendly**: Secrets use environment variables, not inline

---

## 2. File Format

### 2.1 File Extension

RDL files use the `.json` extension. By convention, the main robot configuration is named `robot.json`.

### 2.2 Encoding

- UTF-8 encoding
- No BOM (Byte Order Mark)
- Standard JSON (RFC 8259)
- Comments are NOT supported (standard JSON limitation)

### 2.3 Top-Level Structure

```json
{
  "$schema": "https://gorai.dev/schemas/rdl-v1.json",
  "version": "1",
  "robot": { },
  "nats": { },
  "components": [ ],
  "services": [ ],
  "remotes": [ ],
  "log": { }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `$schema` | string | No | JSON Schema URL for validation |
| `version` | string | Yes | RDL version ("1") |
| `robot` | object | Yes | Robot identity |
| `nats` | object | No | NATS connection config |
| `components` | array | No | Component definitions |
| `services` | array | No | Service definitions |
| `remotes` | array | No | Remote robot connections |
| `log` | object | No | Logging configuration |

---

## 3. Robot Object

The `robot` object defines the robot's identity.

```json
{
  "robot": {
    "name": "my-robot",
    "namespace": "mybot",
    "description": "A wheeled robot with camera and LiDAR"
  }
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | - | Unique robot name (alphanumeric, hyphens, underscores) |
| `namespace` | string | No | Value of `name` | NATS topic namespace |
| `description` | string | No | "" | Human-readable description |

### 3.1 Name Constraints

- Must start with a letter
- May contain letters, numbers, hyphens, underscores
- Length: 1-63 characters
- Case-sensitive
- Must be unique within a NATS cluster

### 3.2 Namespace Usage

The namespace prefixes all NATS topics:
```
gorai.{namespace}.{node}.{topic}
```

Example with namespace "mybot":
```
gorai.mybot.sensors.imu.data
gorai.mybot.motors.left.command
```

---

## 4. NATS Object

The `nats` object configures the NATS connection.

```json
{
  "nats": {
    "url": "nats://localhost:4222",
    "urls": ["nats://nats1:4222", "nats://nats2:4222"],
    "jetstream": true,
    "credentials_file": "/etc/gorai/nats.creds",
    "tls": {
      "ca_file": "/etc/gorai/ca.pem",
      "cert_file": "/etc/gorai/cert.pem",
      "key_file": "/etc/gorai/key.pem"
    },
    "connect_timeout": "5s",
    "reconnect_wait": "1s",
    "max_reconnects": -1
  }
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `url` | string | No* | "nats://localhost:4222" | Single server URL |
| `urls` | array | No* | - | Multiple server URLs (cluster) |
| `jetstream` | bool | No | false | Enable JetStream |
| `credentials_file` | string | No | - | Path to credentials file |
| `tls` | object | No | - | TLS configuration |
| `connect_timeout` | duration | No | "5s" | Connection timeout |
| `reconnect_wait` | duration | No | "1s" | Wait between reconnects |
| `max_reconnects` | int | No | -1 | Max reconnect attempts (-1 = infinite) |

*Either `url` or `urls` should be provided; if both, `urls` takes precedence.

### 4.1 URL Format

```
nats://[user:password@]host:port
```

Examples:
- `nats://localhost:4222`
- `nats://user:pass@nats.example.com:4222`
- `nats://192.168.1.100:4222`

### 4.2 Environment Variable Substitution

Credentials should use environment variables:

```json
{
  "nats": {
    "url": "${NATS_URL:-nats://localhost:4222}",
    "credentials_file": "${NATS_CREDS}"
  }
}
```

Syntax:
- `${VAR}` - Required variable
- `${VAR:-default}` - Variable with default value

---

## 5. Components Array

Components are hardware abstractions (sensors, actuators, infrastructure).

```json
{
  "components": [
    {
      "name": "front_imu",
      "type": "imu",
      "model": "mpu6050",
      "disabled": false,
      "attributes": {
        "i2c_bus": 1,
        "address": "0x68",
        "sample_rate": 100
      },
      "depends_on": []
    }
  ]
}
```

### 5.1 Component Object

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | - | Unique component name |
| `type` | string | Yes | - | Component type (see 5.2) |
| `model` | string | Yes | - | Implementation model |
| `disabled` | bool | No | false | Skip loading this component |
| `attributes` | object | No | {} | Model-specific configuration |
| `depends_on` | array | No | [] | Names of dependencies |

### 5.2 Component Types

#### Sensors

| Type | Description | Common Models |
|------|-------------|---------------|
| `imu` | Inertial Measurement Unit | `mpu6050`, `mpu9250`, `lsm6ds3`, `fake` |
| `ahrs` | Attitude/Heading Reference | `bno055`, `bno085`, `fake` |
| `gps` | GPS/GNSS Receiver | `neo6m`, `neo_m8n`, `zed_f9p`, `fake` |
| `encoder` | Rotary/Linear Encoder | `quadrature`, `as5600`, `amt10x`, `fake` |
| `range_sensor` | Distance Sensor | `hcsr04`, `vl53l0x`, `vl53l1x`, `fake` |
| `lidar` | Laser Scanner | `rplidar_a1`, `rplidar_a2`, `hokuyo`, `fake` |
| `presence_sensor` | Presence Detector | `pir`, `ld2410`, `ld2450`, `fake` |
| `thermal_array` | Thermal Imager | `amg8833`, `mlx90640`, `fake` |
| `force_sensor` | Force Sensor | `hx711`, `fake` |
| `force_6dof` | 6-Axis F/T Sensor | `ati_mini45`, `fake` |
| `current_sensor` | Current/Power Monitor | `ina219`, `ina260`, `acs712`, `fake` |
| `reflectance_sensor` | Line Sensor | `qtr8rc`, `tcrt5000`, `fake` |
| `camera` | Camera | `v4l2`, `picamera`, `realsense`, `fake` |
| `temperature` | Temperature Sensor | `linux_thermal`, `ds18b20`, `bme280`, `fake` |

#### Actuators

| Type | Description | Common Models |
|------|-------------|---------------|
| `motor` | DC/Brushless Motor | `gpio`, `can`, `serial`, `odrive`, `fake` |
| `servo` | Position Servo | `pwm`, `dynamixel`, `lx16a`, `feetech`, `fake` |
| `stepper` | Stepper Motor | `gpio`, `tmc2209`, `tmc5160`, `fake` |
| `thruster` | Underwater Thruster | `pwm`, `bluerobotics`, `fake` |
| `valve` | Valve | `gpio`, `solenoid`, `motorized`, `fake` |
| `gripper` | Gripper | `servo`, `pneumatic`, `fake` |
| `arm` | Robot Arm | `custom`, `fake` |
| `base` | Mobile Base | `differential`, `mecanum`, `ackermann`, `fake` |

#### Infrastructure

| Type | Description | Common Models |
|------|-------------|---------------|
| `power` | Power Source | `battery`, `adc`, `ina219`, `fake` |
| `space` | Virtual Container | `container`, `tank`, `fake` |
| `link` | Communication Link | `serial`, `radio`, `can`, `fake` |

### 5.3 Attribute Types

Attributes are model-specific. Common attribute types:

| Type | JSON Type | Example |
|------|-----------|---------|
| Integer | number | `"pin": 17` |
| Float | number | `"max_rpm": 200.0` |
| String | string | `"device": "/dev/ttyUSB0"` |
| Boolean | boolean | `"inverted": true` |
| Duration | string | `"timeout": "5s"` |
| Address | string | `"address": "0x68"` |
| Array | array | `"pins": [5, 6, 7]` |
| Object | object | `"encoder": { "pin_a": 5, "pin_b": 6 }` |

### 5.4 Dependency Resolution

Components are instantiated in dependency order:
1. Components with no dependencies first
2. Components whose dependencies are satisfied next
3. Circular dependencies cause load failure

Example:
```json
{
  "components": [
    { "name": "imu", "type": "imu", "model": "fake" },
    { "name": "base", "type": "base", "model": "differential",
      "depends_on": ["left_motor", "right_motor"] },
    { "name": "left_motor", "type": "motor", "model": "gpio" },
    { "name": "right_motor", "type": "motor", "model": "gpio" }
  ]
}
```

Load order: `imu`, `left_motor`, `right_motor`, `base`

---

## 6. Services Array

Services are software capabilities that process data or make decisions.

```json
{
  "services": [
    {
      "name": "detector",
      "type": "vision",
      "model": "yolox",
      "disabled": false,
      "attributes": {
        "model_path": "/opt/models/yolox_s.onnx",
        "confidence_threshold": 0.5
      },
      "depends_on": ["front_camera"]
    }
  ]
}
```

### 6.1 Service Object

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | - | Unique service name |
| `type` | string | Yes | - | Service type (see 6.2) |
| `model` | string | Yes | - | Implementation model |
| `disabled` | bool | No | false | Skip loading this service |
| `attributes` | object | No | {} | Model-specific configuration |
| `depends_on` | array | No | [] | Component/service dependencies |

### 6.2 Service Types

| Type | Description | Common Models |
|------|-------------|---------------|
| `vision` | Computer Vision | `yolox`, `yolov8`, `tflite`, `custom`, `fake` |
| `slam` | SLAM/Mapping | `cartographer`, `gmapping`, `fake` |
| `navigation` | Path Planning | `default`, `custom`, `fake` |
| `motion` | Motion Planning | `default`, `custom`, `fake` |
| `behavior` | Behavior Trees | `default`, `custom`, `fake` |
| `coordinator` | Multi-Robot | `default`, `custom`, `fake` |
| `mlmodel` | ML Inference | `tflite`, `onnx`, `tpu`, `fake` |

### 6.3 Service Dependencies

Services can depend on:
- Components (by name)
- Other services (by name)

Dependencies are resolved after all components are loaded.

---

## 7. Remotes Array

Remotes connect to components/services on other robots or nodes.

```json
{
  "remotes": [
    {
      "name": "mcu_bridge",
      "address": "nats://mcu-gateway.local:4222",
      "namespace": "mcu",
      "components": ["wheel_encoders", "motor_driver"],
      "services": []
    }
  ]
}
```

### 7.1 Remote Object

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | - | Unique remote name |
| `address` | string | Yes | - | NATS URL of remote |
| `namespace` | string | No | - | Remote namespace |
| `components` | array | No | [] | Component names to import |
| `services` | array | No | [] | Service names to import |

### 7.2 Remote Resource Access

Imported resources appear as local resources with qualified names:
```
{remote_name}.{component_name}
```

Example: `mcu_bridge.wheel_encoders`

---

## 8. Log Object

Configures logging behavior.

```json
{
  "log": {
    "level": "info",
    "format": "json",
    "output": "stdout",
    "file": "/var/log/gorai/robot.log",
    "max_size_mb": 100,
    "max_backups": 3,
    "max_age_days": 7
  }
}
```

### 8.1 Log Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `level` | string | No | "info" | Log level |
| `format` | string | No | "text" | Output format |
| `output` | string | No | "stdout" | Output destination |
| `file` | string | No | - | Log file path (if output="file") |
| `max_size_mb` | int | No | 100 | Max file size before rotation |
| `max_backups` | int | No | 3 | Number of backups to keep |
| `max_age_days` | int | No | 7 | Days to keep old logs |

### 8.2 Log Levels

| Level | Description |
|-------|-------------|
| `trace` | Very verbose debugging |
| `debug` | Debugging information |
| `info` | Normal operation |
| `warn` | Warning conditions |
| `error` | Error conditions |
| `fatal` | Fatal errors (exits) |

### 8.3 Log Formats

| Format | Description |
|--------|-------------|
| `text` | Human-readable text |
| `json` | JSON lines (structured) |

### 8.4 Output Destinations

| Output | Description |
|--------|-------------|
| `stdout` | Standard output |
| `stderr` | Standard error |
| `file` | Log file (requires `file` field) |

---

## 9. Validation Rules

### 9.1 Structural Validation

1. JSON must be syntactically valid
2. Required fields must be present
3. Field types must match schema
4. Unknown fields are warnings (not errors)

### 9.2 Semantic Validation

1. `robot.name` must be valid identifier
2. Component/service names must be unique
3. Component `type` must be registered
4. Component `model` must be registered for type
5. Dependencies must reference existing components/services
6. No circular dependencies

### 9.3 Error Messages

Validation errors should include:
- File path and line number (if possible)
- Field path (e.g., `components[0].attributes.pin`)
- Expected vs actual value
- Suggestion for fix

Example:
```
robot.json:15: components[0].type: unknown component type "imu2"
  Did you mean "imu"?
  Valid types: imu, ahrs, gps, encoder, ...
```

---

## 10. Complete Example

```json
{
  "$schema": "https://gorai.dev/schemas/rdl-v1.json",
  "version": "1",

  "robot": {
    "name": "wheeled-robot",
    "namespace": "wr1",
    "description": "A differential drive robot with camera and LiDAR"
  },

  "nats": {
    "url": "${NATS_URL:-nats://localhost:4222}",
    "jetstream": true
  },

  "components": [
    {
      "name": "left_motor",
      "type": "motor",
      "model": "gpio",
      "attributes": {
        "pin_forward": 17,
        "pin_reverse": 18,
        "pin_pwm": 12,
        "max_rpm": 200,
        "encoder": {
          "pin_a": 5,
          "pin_b": 6,
          "ticks_per_rev": 1200
        }
      }
    },
    {
      "name": "right_motor",
      "type": "motor",
      "model": "gpio",
      "attributes": {
        "pin_forward": 22,
        "pin_reverse": 23,
        "pin_pwm": 13,
        "max_rpm": 200,
        "encoder": {
          "pin_a": 19,
          "pin_b": 26,
          "ticks_per_rev": 1200
        }
      }
    },
    {
      "name": "base",
      "type": "base",
      "model": "differential",
      "attributes": {
        "left_motor": "left_motor",
        "right_motor": "right_motor",
        "wheel_radius": 0.05,
        "wheel_base": 0.3
      },
      "depends_on": ["left_motor", "right_motor"]
    },
    {
      "name": "imu",
      "type": "ahrs",
      "model": "bno055",
      "attributes": {
        "i2c_bus": 1,
        "address": "0x28"
      }
    },
    {
      "name": "front_camera",
      "type": "camera",
      "model": "v4l2",
      "attributes": {
        "device": "/dev/video0",
        "width": 640,
        "height": 480,
        "fps": 30
      }
    },
    {
      "name": "lidar",
      "type": "lidar",
      "model": "rplidar_a1",
      "attributes": {
        "serial_port": "/dev/ttyUSB0",
        "baud_rate": 115200
      }
    }
  ],

  "services": [
    {
      "name": "detector",
      "type": "vision",
      "model": "yolox",
      "attributes": {
        "model_path": "/opt/models/yolox_s.onnx",
        "confidence_threshold": 0.5,
        "classes": ["person", "chair", "table"]
      },
      "depends_on": ["front_camera"]
    },
    {
      "name": "mapper",
      "type": "slam",
      "model": "cartographer",
      "attributes": {
        "map_resolution": 0.05,
        "map_update_interval": 0.5
      },
      "depends_on": ["lidar", "imu"]
    },
    {
      "name": "navigator",
      "type": "navigation",
      "model": "default",
      "attributes": {
        "max_velocity": 0.5,
        "max_angular_velocity": 1.0,
        "goal_tolerance": 0.1
      },
      "depends_on": ["mapper", "base"]
    }
  ],

  "log": {
    "level": "info",
    "format": "json",
    "output": "stdout"
  }
}
```

---

## 11. Schema Evolution

### 11.1 Versioning

- The `version` field indicates schema version
- Major version changes may break compatibility
- Minor changes are backward compatible

### 11.2 Future Extensions

Reserved for future versions:
- `modules` - Plugin/module loading
- `transforms` - TF tree configuration
- `parameters` - Runtime parameters
- `network` - Network topology
- `security` - Authentication/authorization

---

## Appendix A: JSON Schema

A formal JSON Schema for validation is available at:
```
https://gorai.dev/schemas/rdl-v1.json
```

This schema can be used with JSON validators and IDEs for autocomplete and validation.

---

## Appendix B: Duration Format

Durations use Go-style format:
- `5s` - 5 seconds
- `100ms` - 100 milliseconds
- `1m30s` - 1 minute 30 seconds
- `1h` - 1 hour

---

## Appendix C: Address Format

I2C addresses use hex notation:
- `"0x68"` - Standard hex format
- `"0x28"` - BNO055 default
- `"104"` - Decimal also accepted (equals 0x68)
