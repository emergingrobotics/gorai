# Robot Definition Language (RDL) Specification

**Version:** 3.1
**Status:** Draft
**Last Updated:** 2025-01-24

## 1. Overview

The Robot Definition Language (RDL) is a JSON configuration format that defines the software architecture of a Gorai robot. RDL specifies what components and services a robot has, how they are configured, and their dependencies.

> **Architecture:** Gorai uses a **simple binary deployment** model where robots are compiled to a single Go binary. The RDL configuration defines components and services that run within this binary, communicating via NATS messaging. No containers or Kubernetes required.
>
> **Future Extensions:** Container and K3s support is planned for production fleets. See [FUTURE-ROADMAP.md](../docs/FUTURE-ROADMAP.md) for details.

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
  "$schema": "https://gorai.dev/schemas/rdl-v3.json",
  "version": "3",
  "robot": { },
  "platform": { },
  "nats": { },
  "prometheus": { },
  "components": [ ],
  "services": [ ],
  "remotes": [ ],
  "resources": { },
  "log": { },
  "dashboard": { },
  "alerting": { }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `$schema` | string | No | JSON Schema URL for validation |
| `version` | string | Yes | RDL version ("3") |
| `robot` | object | Yes | Robot identity |
| `platform` | object | No | Hardware platform requirements |
| `nats` | object | No | NATS connection config |
| `prometheus` | object | No | Prometheus metrics config |
| `components` | array | No | Component definitions |
| `services` | array | No | Service definitions |
| `remotes` | array | No | Remote robot connections |
| `resources` | object | No | Default resource limits for containers |
| `log` | object | No | Logging configuration |
| `dashboard` | object | No | Web dashboard configuration (enabled by default) |
| `alerting` | object | No | Alert Manager configuration |

> **Note:** For the current phase, services run within the main robot binary. Container-based services are planned for future phases. See [FUTURE-ROADMAP.md](../docs/FUTURE-ROADMAP.md).

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

## 4. Platform Object

The `platform` object declares hardware requirements for the robot. This enables validation and helps users understand minimum specifications.

```json
{
  "platform": {
    "minimum": "pi4-4gb",
    "recommended": "pi5-8gb",
    "storage": "ssd-required",
    "accelerators": ["hailo8", "coral-tpu"]
  }
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `minimum` | string | No | "pi4-4gb" | Minimum compute platform |
| `recommended` | string | No | "pi5-8gb" | Recommended compute platform |
| `storage` | string | No | "ssd-required" | Storage requirements |
| `accelerators` | array | No | [] | Required AI accelerators |

### 4.1 Platform Identifiers

| Identifier | Description | RAM | Notes |
|------------|-------------|-----|-------|
| `pi4-4gb` | Raspberry Pi 4 Model B 4GB | 4 GB | Minimum supported |
| `pi4-8gb` | Raspberry Pi 4 Model B 8GB | 8 GB | Good for ML workloads |
| `pi5-4gb` | Raspberry Pi 5 4GB | 4 GB | Faster than Pi 4 |
| `pi5-8gb` | Raspberry Pi 5 8GB | 8 GB | Primary platform |
| `jetson-orin-super` | Jetson Orin Nano Super | 8 GB | 67 TOPS, CUDA, performance tier |
| `jetson-orin-nano` | NVIDIA Jetson Orin Nano | 4-8 GB | 40 TOPS, CUDA support |
| `orangepi-5b-8gb` | Orange Pi 5B 8GB | 8 GB | 6 TOPS NPU, budget AI |
| `rock5b` | Radxa Rock 5B | 4-16 GB | RK3588, 6 TOPS NPU |
| `x86-4gb` | Any x86_64 with 4GB+ | 4+ GB | Generic x86 |

### 4.2 Storage Requirements

| Value | Description |
|-------|-------------|
| `ssd-required` | External SSD mandatory |
| `ssd-recommended` | SSD recommended but SD card acceptable for testing |

> **Note:** SSD recommended for development (faster image pulls). SD cards (A2 class) acceptable for deployed robots with stable container images.

### 4.3 Accelerator Identifiers

| Identifier | Description | Performance |
|------------|-------------|-------------|
| `hailo8` | Hailo-8 NPU | 26 TOPS |
| `hailo8l` | Hailo-8L NPU | 13 TOPS |
| `coral-tpu` | Google Coral Edge TPU | 4 TOPS |
| `cuda` | NVIDIA CUDA GPU | Varies |
| `rk3588-npu` | Rockchip RK3588 NPU | 6 TOPS |

---

## 5. NATS Object

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

## 6. Components Array

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
| `log_level` | string | No | "error" | Per-component log level (trace, debug, info, warn, error, fatal) |
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

### 5.5 Per-Component Log Levels

Components can specify their own log level using the `log_level` field. This allows fine-grained control over logging verbosity without changing the global log level.

**Available log levels:** `trace`, `debug`, `info`, `warn`, `error`, `fatal`

**Default:** `error` (production-friendly, minimal logging)

Example:
```json
{
  "components": [
    {
      "name": "front_camera",
      "type": "camera",
      "model": "v4l2",
      "log_level": "debug",
      "attributes": {
        "device": "/dev/video0"
      }
    }
  ]
}
```

The log level affects:
- Internal component logging
- Driver-level logging for the component
- Any diagnostics or performance metrics

---

## 7. Services Array

Services are software capabilities that process data or make decisions.

```json
{
  "services": [
    {
      "name": "detector",
      "type": "vision",
      "model": "yolox",
      "disabled": false,
      "log_level": "info",
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
| `log_level` | string | No | "error" | Per-service log level (trace, debug, info, warn, error, fatal) |
| `external` | object | No | - | External process configuration (see 6.4) |
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

### 6.4 Per-Service Log Levels

Services can specify their own log level using the `log_level` field. This is especially useful for debugging external services without changing global logging.

**Available log levels:** `trace`, `debug`, `info`, `warn`, `error`, `fatal`

**Default:** `error` (production-friendly, minimal logging)

For **external services**, the log level is passed as the `LOG_LEVEL` environment variable. This allows service implementations (in any language) to respect the configured level.

Example:
```json
{
  "services": [
    {
      "name": "person_detector",
      "type": "vision",
      "model": "yolox",
      "log_level": "info",
      "external": {
        "enabled": true,
        "container": {
          "image": "localhost/person-detector:latest"
        }
      }
    }
  ]
}
```

The container will receive `LOG_LEVEL=INFO` in its environment. Service implementations should:
1. Read `LOG_LEVEL` from environment (e.g., `os.environ.get("LOG_LEVEL", "ERROR")`)
2. Configure their logging framework accordingly
3. Default to `ERROR` if not specified

This enables per-service debugging in production:
- Set most services to `error` for minimal noise
- Set specific services to `info` or `debug` when troubleshooting

### 6.5 External Services

Services can optionally run as separate processes, connected to the main robot via NATS. This is useful for:
- ML inference requiring specialized hardware (TPU, NPU)
- Services with different resource requirements
- Services written in different languages (e.g., Python for ML)
- Reusable service modules shared across robots

#### Service RDL Files

External services can have their own **Service RDL** file that defines their behavior independently. This enables:
- **Modularity**: Services are self-contained and reusable
- **Separation of concerns**: Service authors define behavior, robot integrators configure deployment
- **Sharing**: Services can be distributed as standalone packages
- **Independent development**: Services can be developed/tested without a full robot

A Service RDL file (conventionally named `<service>.rdl.json`) defines:
- Service metadata (name pattern, type, model)
- Input/output topic patterns (with variable substitution)
- Default attributes and configuration
- Optional container/process defaults

**Service RDL Example (`services/person-detector/person-detector.rdl.json`):**
```json
{
  "$schema": "https://gorai.dev/schemas/service-rdl-v1.json",
  "version": "1",
  "kind": "service",

  "service": {
    "type": "object_detection",
    "model": "yolox",
    "description": "YOLOX-based person detection with bounding box annotation"
  },

  "topics": {
    "subscribe": [
      {
        "name": "input",
        "pattern": "gorai.{namespace}.{input_component}.data",
        "description": "JPEG image frames to process",
        "format": "image/jpeg"
      }
    ],
    "publish": [
      {
        "name": "annotated",
        "pattern": "gorai.{namespace}.{service}.annotated",
        "description": "Annotated images with bounding boxes",
        "format": "image/jpeg"
      },
      {
        "name": "detections",
        "pattern": "gorai.{namespace}.{service}.detections",
        "description": "Detection results as JSON",
        "format": "application/json"
      }
    ]
  },

  "attributes": {
    "confidence_threshold": {
      "type": "float",
      "default": 0.5,
      "description": "Minimum confidence for detections"
    },
    "classes": {
      "type": "array",
      "default": ["person"],
      "description": "Object classes to detect"
    },
    "model_path": {
      "type": "string",
      "required": true,
      "description": "Path to the model file"
    },
    "input_component": {
      "type": "string",
      "required": true,
      "description": "Name of the camera component to subscribe to"
    }
  },

  "runtime": {
    "container": {
      "image": "localhost/{service}:latest",
      "build": {
        "context": "."
      }
    }
  }
}
```

#### Referencing Service RDL from Robot RDL

The robot RDL references external services using the `rdl` field:

**Example: Robot referencing a Service RDL**
```json
{
  "services": [
    {
      "name": "person_detector",
      "rdl": "./services/person-detector/person-detector.rdl.json",
      "attributes": {
        "input_component": "main_camera",
        "model_path": "/opt/models/yolox_s.hef",
        "confidence_threshold": 0.6
      },
      "external": {
        "enabled": true,
        "container": {
          "devices": ["/dev/hailo0"],
          "volumes": ["/opt/models:/models:ro"]
        },
        "managed": true
      }
    }
  ]
}
```

When `rdl` is specified:
1. The Service RDL file is loaded and merged with the robot's service definition
2. Topic patterns are resolved using variables from robot context
3. Attributes in robot RDL override Service RDL defaults
4. Runtime configuration in robot RDL overrides Service RDL defaults

#### Topic Pattern Variables

Service RDL uses pattern variables that are resolved at runtime:

| Variable | Description | Example |
|----------|-------------|---------|
| `{namespace}` | Robot namespace | `hello-camera` |
| `{service}` | Service name (from robot RDL) | `person_detector` |
| `{robot}` | Robot name | `hello-camera` |
| `{input_component}` | Custom variable from attributes | `main_camera` |

**Resolution example:**
```
Pattern:  gorai.{namespace}.{input_component}.data
Context:  namespace=hello-camera, input_component=main_camera
Result:   gorai.hello-camera.main_camera.data
```

#### Inline External Services (without Service RDL)

For simpler cases or one-off services, external services can still be defined inline:

**Example: Native External Service**
```json
{
  "services": [
    {
      "name": "person_detector",
      "type": "object_detection",
      "model": "hailo_yolox",
      "external": {
        "enabled": true,
        "command": "/opt/gorai/services/hailo-detector",
        "args": ["--confidence", "0.5"],
        "managed": true,
        "restart": "always",
        "env": {
          "MODEL_PATH": "/opt/models/yolox.hef"
        }
      },
      "attributes": {
        "input_topic": "gorai.myrobot.camera.data",
        "output_topic": "gorai.myrobot.person_detector.detections"
      }
    }
  ]
}
```

**Example: Containerized External Service**
```json
{
  "services": [
    {
      "name": "person_detector",
      "type": "object_detection",
      "model": "hailo_yolox",
      "external": {
        "enabled": true,
        "container": {
          "image": "localhost/hailo-detector:latest",
          "devices": ["/dev/hailo0"],
          "environment": {
            "MODEL_PATH": "/models/yolox_s.hef",
            "CONFIDENCE_THRESHOLD": "0.5"
          },
          "volumes": [
            "/opt/models:/models:ro"
          ]
        },
        "managed": true,
        "restart": "always"
      },
      "attributes": {
        "input_topic": "gorai.myrobot.camera.data",
        "output_topic": "gorai.myrobot.person_detector.detections"
      }
    }
  ]
}
```

#### Service Object Fields (Updated)

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | - | Unique service name |
| `rdl` | string | No | - | Path to Service RDL file (relative to robot config) |
| `type` | string | Yes* | - | Service type (*not required if using `rdl`) |
| `model` | string | Yes* | - | Implementation model (*not required if using `rdl`) |
| `disabled` | bool | No | false | Skip loading this service |
| `log_level` | string | No | "error" | Per-service log level (trace, debug, info, warn, error, fatal) |
| `external` | object | No | - | External process configuration (see below) |
| `attributes` | object | No | {} | Service-specific configuration |
| `depends_on` | array | No | [] | Component/service dependencies |

#### External Object Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | bool | No | false | Enable external process mode |
| `command` | string | No* | - | Path to executable (*required if not using container) |
| `args` | array | No | [] | Command line arguments |
| `container` | object | No | - | Container configuration (see below) |
| `managed` | bool | No | true | If true, robot spawns/monitors the process |
| `restart` | string | No | "always" | Restart policy: "always", "on-failure", "never" |
| `env` | object | No | {} | Environment variables for the process |

#### Container Object Fields

When using a containerized external service:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `image` | string | Yes | - | Container image name |
| `build` | object | No | - | Build configuration (see below) |
| `devices` | array | No | [] | Device mappings (e.g., "/dev/hailo0") |
| `environment` | object | No | {} | Environment variables |
| `volumes` | array | No | [] | Volume mounts (format: "host:container[:opts]") |
| `network` | string | No | "host" | Network mode |
| `privileged` | bool | No | false | Run in privileged mode |

#### Container Build Configuration

The `build` object configures how to build the container image. When present, `gorai build` will automatically build this container.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `context` | string | Yes | - | Build context directory (relative to config) |
| `containerfile` | string | No | "Containerfile" | Containerfile/Dockerfile name |
| `args` | object | No | {} | Build arguments |
| `target` | string | No | - | Multi-stage build target |
| `no_cache` | bool | No | false | Disable build cache |

**Example with build configuration:**
```json
{
  "external": {
    "enabled": true,
    "container": {
      "image": "localhost/hailo-detector:latest",
      "build": {
        "context": "./services/hailo-detector",
        "args": {
          "MODEL_VERSION": "v2.10.0"
        }
      },
      "devices": ["/dev/hailo0"]
    }
  }
}
```

**Convention-based discovery:**

If no `build` config is provided but the image starts with `localhost/`, `gorai build` will look for:
- `./services/<service-name>/Containerfile`
- `./<service-name>/Containerfile`

#### External Service Behavior

- **Managed services** (`managed: true`): The robot process spawns the external service as a child process (or container), monitors it, and restarts it according to the restart policy.
- **Unmanaged services** (`managed: false`): The robot expects the service to be running independently (e.g., started by systemd). The robot verifies connectivity via NATS.
- Native external services receive the robot config path via `--config` argument and service name via `--service` argument.
- Container services receive environment variables `GORAI_ROBOT_NAME`, `GORAI_SERVICE_NAME`, and `NATS_URL`.
- When using Service RDL, resolved topic names are passed via `GORAI_INPUT_TOPICS` and `GORAI_OUTPUT_TOPICS` environment variables.

#### Components vs Services

> **Important Architectural Distinction:**
>
> **Components** represent hardware abstractions and MUST be native Go code compiled into the robot monolith. Components have direct access to hardware devices and run in the same process as the robot.
>
> **Services** represent software capabilities and MAY be:
> - **Internal** - Native Go code in the monolith (default)
> - **External Native** - Separate process on same or different host
> - **External Container** - Container on same or different host
> - **External with Service RDL** - Modular service with self-contained definition
>
> This distinction allows compute-intensive or specialized services (ML inference, SLAM) to run separately while keeping hardware access simple and direct.

---

## 8. Remotes Array

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

## 9. Log Object

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

## 10. Prometheus Object

Prometheus is a **required dependency** for Gorai, running locally on the robot alongside NATS. It provides time-series storage, powerful queries via PromQL, and integrates with Alert Manager for alerting.

```json
{
  "prometheus": {
    "url": "http://localhost:9090",
    "metrics_port": 9091,
    "metrics_path": "/metrics",
    "scrape_interval": "5s",
    "retention": "15d",
    "labels": {
      "environment": "production"
    }
  }
}
```

### 9.1 Prometheus Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `url` | string | No | "http://localhost:9090" | Prometheus server URL |
| `metrics_port` | int | No | 9091 | Port for Gorai's /metrics endpoint |
| `metrics_path` | string | No | "/metrics" | Path for metrics endpoint |
| `scrape_interval` | duration | No | "5s" | How often Prometheus scrapes metrics |
| `retention` | duration | No | "15d" | How long Prometheus retains data |
| `labels` | object | No | {} | Additional labels added to all metrics |

### 9.2 Why Prometheus is Required

Prometheus runs locally on the robot as a peer dependency alongside NATS:

| Dependency | Purpose | Runs On |
|------------|---------|---------|
| **NATS** | Real-time messaging, pub/sub | Robot (localhost:4222) |
| **Prometheus** | Time-series storage, queries, alerting | Robot (localhost:9090) |
| **Alert Manager** | Alert routing and notifications | Robot (localhost:9093, optional) |

### 9.3 Resource Requirements

| Component | RAM | CPU | Disk |
|-----------|-----|-----|------|
| Gorai | 50-200 MB | Variable | Minimal |
| NATS | 10-50 MB | Low | Minimal |
| Prometheus | 200-500 MB | Low | ~2 GB/month |
| Alert Manager | 50 MB | Minimal | Minimal |
| **Total** | **~500 MB** | Low | ~2 GB/month |

Compatible with: Raspberry Pi 4 (4GB), Jetson Nano, Rock 5B, any x86 system.

### 9.4 Metrics Exported by Gorai

Gorai automatically exports metrics in Prometheus format:

```prometheus
# Sensor readings
gorai_sensor_value{robot="sentinel", sensor="imu", field="accel_x"} 0.02
gorai_sensor_value{robot="sentinel", sensor="battery", field="level"} 0.73

# Component state (1=running, 0=stopped, -1=error)
gorai_component_state{robot="sentinel", component="motor_left"} 1

# Inference metrics
gorai_inference_duration_seconds{robot="sentinel", model="yolox", quantile="0.99"} 0.067
gorai_detections_total{robot="sentinel", model="yolox", class="person"} 892

# System metrics
gorai_messages_total{robot="sentinel", direction="published"} 1234567
gorai_errors_total{robot="sentinel", component="camera_front"} 3
```

### 9.5 PromQL Query Examples

The dashboard and external tools can query Prometheus using PromQL:

```promql
# Current battery level
gorai_sensor_value{sensor="battery", field="level"}

# Average inference latency over 5 minutes
avg_over_time(gorai_inference_duration_seconds[5m])

# Detection rate (per second)
rate(gorai_detections_total[1m])

# Components in error state
gorai_component_state == -1

# Battery drain rate (% per hour)
deriv(gorai_sensor_value{sensor="battery", field="level"}[1h]) * 3600
```

### 9.6 Prometheus Configuration File

Gorai expects Prometheus to be configured to scrape its metrics endpoint:

**/etc/prometheus/prometheus.yml**:
```yaml
global:
  scrape_interval: 5s
  evaluation_interval: 5s

rule_files:
  - /etc/gorai/alerts.yml

scrape_configs:
  - job_name: 'gorai'
    static_configs:
      - targets: ['localhost:9091']
    relabel_configs:
      - source_labels: [__address__]
        target_label: robot
        replacement: 'sentinel'
```

---

## 11. Alerting Object

The `alerting` object configures integration with Prometheus Alert Manager for robot alerts.

```json
{
  "alerting": {
    "enabled": true,
    "alertmanager_url": "http://localhost:9093",
    "rules_file": "/etc/gorai/alerts.yml",
    "evaluation_interval": "15s"
  }
}
```

### 10.1 Alerting Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | bool | No | false | Enable Alert Manager integration |
| `alertmanager_url` | string | No | "http://localhost:9093" | Alert Manager URL |
| `rules_file` | string | No | "/etc/gorai/alerts.yml" | Path to alert rules file |
| `evaluation_interval` | duration | No | "15s" | How often to evaluate alert rules |

### 10.2 Alert Rules Example

**/etc/gorai/alerts.yml**:
```yaml
groups:
  - name: robot_alerts
    rules:
      - alert: BatteryLow
        expr: gorai_sensor_value{sensor="battery", field="level"} < 0.2
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Battery below 20%"

      - alert: BatteryCritical
        expr: gorai_sensor_value{sensor="battery", field="level"} < 0.1
        for: 30s
        labels:
          severity: critical
        annotations:
          summary: "Battery below 10% - return to base"

      - alert: MotorOverheat
        expr: gorai_sensor_value{sensor=~"motor.*", field="temperature"} > 80
        for: 30s
        labels:
          severity: critical
        annotations:
          summary: "Motor {{ $labels.sensor }} overheating"

      - alert: InferenceSlow
        expr: histogram_quantile(0.99, rate(gorai_inference_duration_seconds_bucket[5m])) > 0.2
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Inference latency > 200ms"

      - alert: ComponentError
        expr: gorai_component_state == -1
        for: 10s
        labels:
          severity: critical
        annotations:
          summary: "Component {{ $labels.component }} in error state"
```

### 10.3 Alert Manager Configuration

**/etc/alertmanager/alertmanager.yml**:
```yaml
global:
  resolve_timeout: 5m

route:
  receiver: 'local'
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 1h
  routes:
    - match:
        severity: critical
      receiver: 'critical'
    - match:
        severity: warning
      receiver: 'warning'

receivers:
  - name: 'local'
    webhook_configs:
      - url: 'http://localhost:8080/api/alerts'
        send_resolved: true

  - name: 'critical'
    webhook_configs:
      - url: 'http://localhost:8080/api/alerts'
        send_resolved: true

  - name: 'warning'
    webhook_configs:
      - url: 'http://localhost:8080/api/alerts'
        send_resolved: true
```

---

## 12. Dashboard Object

The `dashboard` object configures the embedded web dashboard. The dashboard queries **local Prometheus** for both real-time gauges and historical data, providing a unified monitoring interface.

```json
{
  "dashboard": {
    "enabled": true,
    "listen": ":8080",
    "video": {
      "enabled": true,
      "format": "mjpeg",
      "max_fps": 15,
      "quality": 80
    }
  }
}
```

### 11.1 Dashboard Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | bool | No | true | Enable/disable dashboard |
| `listen` | string | No | ":8080" | HTTP listen address |
| `video` | object | No | - | Video streaming configuration |

Note: Data retention is now configured via `prometheus.retention`, not in the dashboard.

### 11.2 Video Configuration

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | bool | No | true | Enable video streaming |
| `format` | string | No | "mjpeg" | Stream format: "mjpeg", "webrtc", "hls" |
| `max_fps` | int | No | 15 | Maximum frame rate |
| `quality` | int | No | 80 | JPEG quality (1-100) |

### 11.3 Dashboard Sections

The dashboard provides two viewing modes, both powered by Prometheus queries:

| Section | Description | Prometheus Query Type |
|---------|-------------|----------------------|
| **Gauges** | Real-time current values — latest sensor reading, component state, inference result | Instant query |
| **History** | Time-series data — graphs, trends, historical playback | Range query |

**Gauges** show current state:
- Current sensor values with units
- Component state indicators (running, stopped, error)
- Latest inference results
- Queries Prometheus instant API (~5ms latency)

**History** shows trends over time:
- Time-series graphs powered by uPlot
- Configurable time windows (5m, 1h, 24h, etc.)
- Scroll back through data up to Prometheus retention (default 15 days)
- Queries Prometheus range API (~10-50ms latency)

### 11.4 Dashboard Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ Browser                                                          │
│                                                                  │
│  Dashboard UI (HTMX + uPlot)                                    │
│   ├─ Gauges: polls /api/gauges every 5s                        │
│   ├─ History: polls /api/history on load + refresh             │
│   └─ Cameras: streams from /api/cameras/{name}/stream          │
└────────────────────────────────────────────────────────────────┘
                              │
                              │ HTTP :8080
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Gorai Dashboard Server                                           │
│                                                                  │
│  /api/gauges   → Prometheus instant query                       │
│  /api/history  → Prometheus range query                         │
│  /api/topology → Component/service registry                     │
│  /api/alerts   → Receives webhooks from Alert Manager           │
│  /api/cameras  → MJPEG streams from camera components           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ PromQL queries
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Local Prometheus (localhost:9090)                                │
│                                                                  │
│  Time-series storage, 15-day retention                          │
│  Scrapes Gorai /metrics endpoint every 5s                       │
└─────────────────────────────────────────────────────────────────┘
```

### 11.5 Disabling the Dashboard

To disable the dashboard completely:

```json
{
  "dashboard": {
    "enabled": false
  }
}
```

When disabled:
- No HTTP server starts on :8080
- Zero CPU/memory overhead
- Prometheus continues to collect metrics (can still use Grafana, etc.)

### 11.6 Dashboard Features

When enabled, the dashboard provides:

| View | URL | Description |
|------|-----|-------------|
| Home | `/` | System status, uptime, connection info, active alerts |
| Topology | `/topology` | Tree view of all components and services |
| Sensors | `/sensors` | Gauges and History for sensor data |
| Cameras | `/cameras` | Live camera feeds with inference overlays |
| Inference | `/inference` | ML model outputs and performance metrics |
| Alerts | `/alerts` | Active and resolved alerts from Alert Manager |

### 11.7 Security Considerations

The dashboard is designed for **trusted networks** (local development, robot-internal access). For production deployments exposed to untrusted networks:

1. Bind to localhost only: `"listen": "127.0.0.1:8080"`
2. Use a reverse proxy with authentication
3. Or disable the dashboard: `"enabled": false`

Future versions may add optional authentication.

---

## 13. Validation Rules

### 12.1 Structural Validation

1. JSON must be syntactically valid
2. Required fields must be present
3. Field types must match schema
4. Unknown fields are warnings (not errors)

### 12.2 Semantic Validation

1. `robot.name` must be valid identifier
2. Component/service names must be unique
3. Component `type` must be registered
4. Component `model` must be registered for type
5. Dependencies must reference existing components/services
6. No circular dependencies

### 12.3 Error Messages

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

## 14. Complete Example

```json
{
  "$schema": "https://gorai.dev/schemas/rdl-v3.json",
  "version": "3",

  "robot": {
    "name": "wheeled-robot",
    "namespace": "wr1",
    "description": "A differential drive robot with camera and LiDAR"
  },

  "nats": {
    "url": "${NATS_URL:-nats://localhost:4222}",
    "jetstream": true
  },

  "prometheus": {
    "url": "http://localhost:9090",
    "metrics_port": 9091,
    "scrape_interval": "5s",
    "retention": "15d"
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
      "external": {
        "enabled": true,
        "command": "/opt/gorai/services/yolox-detector",
        "managed": true,
        "restart": "always"
      },
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
  },

  "alerting": {
    "enabled": true,
    "alertmanager_url": "http://localhost:9093",
    "rules_file": "/etc/gorai/alerts.yml"
  },

  "dashboard": {
    "enabled": true,
    "listen": ":8080",
    "video": {
      "enabled": true,
      "format": "mjpeg",
      "max_fps": 15,
      "quality": 80
    }
  }
}
```

---

## 15. Service RDL Schema

Service RDL files define external services independently of any specific robot. This section describes the Service RDL schema.

### 14.1 Service RDL Top-Level Structure

```json
{
  "$schema": "https://gorai.dev/schemas/service-rdl-v1.json",
  "version": "1",
  "kind": "service",
  "service": { },
  "topics": { },
  "attributes": { },
  "runtime": { }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `$schema` | string | No | JSON Schema URL for validation |
| `version` | string | Yes | Service RDL version ("1") |
| `kind` | string | Yes | Must be "service" |
| `service` | object | Yes | Service metadata |
| `topics` | object | Yes | Topic subscriptions and publications |
| `attributes` | object | No | Configurable attributes with defaults |
| `runtime` | object | No | Default runtime configuration |

### 14.2 Service Object

```json
{
  "service": {
    "type": "object_detection",
    "model": "yolox",
    "description": "YOLOX-based object detection service"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Service type (vision, slam, etc.) |
| `model` | string | Yes | Implementation model |
| `description` | string | No | Human-readable description |

### 14.3 Topics Object

```json
{
  "topics": {
    "subscribe": [
      {
        "name": "input",
        "pattern": "gorai.{namespace}.{input_component}.data",
        "description": "Input data stream",
        "format": "image/jpeg"
      }
    ],
    "publish": [
      {
        "name": "output",
        "pattern": "gorai.{namespace}.{service}.result",
        "description": "Output data stream",
        "format": "application/json"
      }
    ]
  }
}
```

#### Topic Entry Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Logical name for this topic |
| `pattern` | string | Yes | NATS topic pattern with variables |
| `description` | string | No | Human-readable description |
| `format` | string | No | Data format (MIME type) |

#### Pattern Variables

Patterns can include variables in `{variable}` format:
- `{namespace}` - Robot namespace (from robot RDL)
- `{robot}` - Robot name (from robot RDL)
- `{service}` - Service name (from robot RDL service entry)
- Custom variables from service attributes

### 14.4 Attributes Object

```json
{
  "attributes": {
    "confidence_threshold": {
      "type": "float",
      "default": 0.5,
      "description": "Minimum confidence",
      "min": 0.0,
      "max": 1.0
    },
    "model_path": {
      "type": "string",
      "required": true,
      "description": "Path to model file"
    },
    "classes": {
      "type": "array",
      "default": ["person"],
      "description": "Object classes to detect"
    }
  }
}
```

#### Attribute Definition Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Data type: string, int, float, bool, array, object |
| `default` | any | No | Default value if not specified |
| `required` | bool | No | If true, must be provided in robot RDL |
| `description` | string | No | Human-readable description |
| `min` | number | No | Minimum value (for numeric types) |
| `max` | number | No | Maximum value (for numeric types) |
| `enum` | array | No | Allowed values (for string type) |

### 14.5 Runtime Object

```json
{
  "runtime": {
    "container": {
      "image": "localhost/{service}:latest",
      "build": {
        "context": "."
      },
      "environment": {
        "LOG_LEVEL": "info"
      }
    },
    "command": "/usr/local/bin/service",
    "env": {
      "LOG_LEVEL": "info"
    }
  }
}
```

The `runtime` object provides default configuration that can be overridden in the robot RDL. It follows the same structure as the `external` object in robot RDL services.

### 14.6 Complete Service RDL Example

```json
{
  "$schema": "https://gorai.dev/schemas/service-rdl-v1.json",
  "version": "1",
  "kind": "service",

  "service": {
    "type": "object_detection",
    "model": "yolox",
    "description": "YOLOX person detector with bounding box annotation"
  },

  "topics": {
    "subscribe": [
      {
        "name": "input",
        "pattern": "gorai.{namespace}.{input_component}.data",
        "description": "JPEG image frames from camera",
        "format": "image/jpeg"
      }
    ],
    "publish": [
      {
        "name": "annotated",
        "pattern": "gorai.{namespace}.{service}.annotated",
        "description": "Images with bounding boxes drawn",
        "format": "image/jpeg"
      },
      {
        "name": "detections",
        "pattern": "gorai.{namespace}.{service}.detections",
        "description": "Detection results as JSON array",
        "format": "application/json"
      }
    ]
  },

  "attributes": {
    "input_component": {
      "type": "string",
      "required": true,
      "description": "Camera component name to subscribe to"
    },
    "model_path": {
      "type": "string",
      "required": true,
      "description": "Path to YOLOX model file (.hef, .onnx, etc.)"
    },
    "confidence_threshold": {
      "type": "float",
      "default": 0.5,
      "min": 0.0,
      "max": 1.0,
      "description": "Minimum detection confidence"
    },
    "classes": {
      "type": "array",
      "default": ["person"],
      "description": "Object classes to detect and report"
    },
    "draw_boxes": {
      "type": "bool",
      "default": true,
      "description": "Draw bounding boxes on annotated output"
    },
    "draw_labels": {
      "type": "bool",
      "default": true,
      "description": "Draw class labels on bounding boxes"
    }
  },

  "runtime": {
    "container": {
      "image": "localhost/person-detector:latest",
      "build": {
        "context": "."
      },
      "network": "host"
    }
  }
}
```

---

## 16. Schema Evolution

### 15.1 Versioning

- The `version` field indicates schema version
- Major version changes may break compatibility
- Minor changes are backward compatible
- Robot RDL and Service RDL have independent versioning

### 15.2 Future Extensions

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
