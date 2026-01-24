# Robot Build and Deployment Architecture Plan

## Executive Summary

This plan addresses fundamental architectural questions about how Gorai robots are built, configured, and deployed. The current skeleton CLI approach needs revision to create a clear, production-ready workflow.

**Key Decisions:**
1. The primary output is a **robot binary**, not a CLI tool
2. Robot configuration uses a **JSON Robot Definition Language (RDL)**
3. Deployment uses **cross-compilation + scp/rsync**
4. The `gorai` package is a **library**, not a framework that generates code

---

## Part 1: Problem Statement

### Current Issues

1. **Unclear Binary Purpose**: Is `gorai` a CLI tool, a runtime, or a library?
2. **Incomplete Runtime**: `gorai run config.json` exists as a stub but isn't implemented
3. **No Deployment Story**: How does code get from dev machine to robot?
4. **Missing Configuration Schema**: What exactly goes in the JSON config?
5. **Build Target Ambiguity**: What architecture are we building for?

### Design Goals

1. **Library-First**: Gorai is a Go library; users write their own `main.go`
2. **Configuration-Driven**: JSON defines what components/services to load
3. **Cross-Compile Ready**: Build on any machine, deploy to ARM64 Pi
4. **Simple Deployment**: `make deploy` should just work
5. **Production Ready**: systemd integration, logging, monitoring

---

## Part 2: Architecture Decision

### Option Analysis

| Approach | Description | Pros | Cons |
|----------|-------------|------|------|
| **A: Library** | User writes main.go, imports gorai | Go-idiomatic, flexible | More boilerplate |
| **B: Framework CLI** | `gorai run config.json` | Turnkey, less code | Less flexible, magic |
| **C: Code Generation** | `gorai init` generates project | Scaffolding helps | Maintenance burden |
| **D: Hybrid** | Library + optional runtime binary | Best of both | Complexity |

### Recommended: Option D (Hybrid)

**Rationale:**
- **Library for flexibility**: Advanced users write custom main.go
- **Runtime binary for simplicity**: Beginners use `gorai-runtime config.json`
- **Same configuration format**: Both use Robot Definition Language
- **Same component model**: Registry pattern works for both

### Binary Types

```
┌─────────────────────────────────────────────────────────────┐
│                    Gorai Ecosystem                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. gorai-runtime (optional)                                │
│     - Generic robot runtime                                 │
│     - Loads config.json and runs                            │
│     - Good for simple robots                                │
│                                                             │
│  2. Custom Robot Binary (recommended)                       │
│     - User writes main.go                                   │
│     - Imports github.com/gorai/gorai/*                      │
│     - Full control over startup/lifecycle                   │
│     - Can embed config or load from file                    │
│                                                             │
│  3. gorai (dev tool, separate concern)                      │
│     - Development utilities only                            │
│     - gorai topic list, gorai topic echo                    │
│     - gorai service call                                    │
│     - NOT for running robots                                │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## Part 3: Robot Definition Language (RDL)

### Purpose

RDL defines the **software architecture** of a robot:
- What components exist (sensors, actuators)
- What services run (vision, navigation)
- How they're configured
- Dependencies between them

RDL does **NOT** define:
- Physical robot geometry (that's URDF/SDF)
- Kinematic chains
- Visual/collision meshes

### JSON Schema

```json
{
  "$schema": "https://gorai.dev/schemas/robot-v1.json",
  "version": "1",

  "robot": {
    "name": "my-robot",
    "namespace": "mybot",
    "description": "A simple wheeled robot with camera"
  },

  "nats": {
    "url": "nats://localhost:4222",
    "jetstream": true,
    "credentials": null
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
        "max_rpm": 200
      }
    },
    {
      "name": "front_imu",
      "type": "imu",
      "model": "mpu6050",
      "attributes": {
        "i2c_bus": 1,
        "address": "0x68",
        "sample_rate": 100
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
        "fps": 30,
        "format": "mjpeg"
      }
    },
    {
      "name": "front_lidar",
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
        "nms_threshold": 0.45
      },
      "depends_on": ["front_camera"]
    },
    {
      "name": "mapper",
      "type": "slam",
      "model": "cartographer",
      "attributes": {
        "map_resolution": 0.05
      },
      "depends_on": ["front_lidar", "front_imu"]
    },
    {
      "name": "navigator",
      "type": "navigation",
      "model": "default",
      "attributes": {
        "max_velocity": 0.5,
        "goal_tolerance": 0.1
      },
      "depends_on": ["mapper"]
    }
  ],

  "remotes": [
    {
      "name": "mcu_bridge",
      "address": "nats://mcu-gateway:4222",
      "components": ["wheel_encoders", "motor_driver"]
    }
  ],

  "log": {
    "level": "info",
    "format": "json",
    "output": "stdout"
  }
}
```

### Component Types (must match code)

**Sensors:**
- `imu`, `ahrs`, `gps`, `encoder`, `range_sensor`, `lidar`
- `presence_sensor`, `thermal_array`, `force_sensor`, `force_6dof`
- `current_sensor`, `reflectance_sensor`, `camera`

**Actuators:**
- `motor`, `servo`, `stepper`, `thruster`, `valve`
- `gripper`, `arm`, `base`

**Infrastructure:**
- `power`, `space`, `link`

### Model Types (implementations)

Each component type can have multiple models:
- `motor`: `gpio`, `can`, `serial`, `fake`
- `imu`: `mpu6050`, `mpu9250`, `bno055`, `fake`
- `camera`: `v4l2`, `picamera`, `realsense`, `fake`
- `lidar`: `rplidar_a1`, `hokuyo`, `fake`

---

## Part 4: Build System

### Cross-Compilation Targets

| Target | GOOS | GOARCH | Use Case |
|--------|------|--------|----------|
| `linux-arm64` | linux | arm64 | Raspberry Pi 4/5, Orange Pi 5 |
| `linux-armv7` | linux | arm | Raspberry Pi 3, older boards |
| `linux-amd64` | linux | amd64 | x86 SBCs, servers |
| `darwin-arm64` | darwin | arm64 | Mac M1/M2 development |
| `darwin-amd64` | darwin | amd64 | Intel Mac development |

### Build Commands

```makefile
# Build for local development
make build

# Build for Raspberry Pi 4/5 (ARM64)
make build-pi
# Output: build/bin/linux-arm64/myrobot

# Build for specific target
make build-linux-arm64
make build-linux-armv7

# Build all platforms
make build-all

# Build with embedded config
make build CONFIG=robot.json
```

### Project Structure (User's Robot)

```
my-robot/
├── main.go              # Entry point
├── robot.json           # Robot definition
├── Makefile             # Build/deploy targets
├── go.mod
├── go.sum
├── components/          # Custom components (optional)
│   └── custom_sensor/
│       └── sensor.go
├── services/            # Custom services (optional)
│   └── custom_behavior/
│       └── behavior.go
└── deploy/
    ├── myrobot.service  # systemd unit
    └── install.sh       # Installation script
```

### Example main.go

```go
package main

import (
    "context"
    "flag"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/gorai/gorai/pkg/config"
    "github.com/gorai/gorai/pkg/robot"

    // Import component implementations to register them
    _ "github.com/gorai/gorai/components/motor/gpio"
    _ "github.com/gorai/gorai/components/sensor/mpu6050"
    _ "github.com/gorai/gorai/components/camera/v4l2"

    // Import custom components
    _ "myrobot/components/custom_sensor"
)

func main() {
    configPath := flag.String("config", "robot.json", "Path to robot configuration")
    flag.Parse()

    // Load configuration
    cfg, err := config.Load(*configPath)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Create robot from configuration
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    r, err := robot.New(ctx, cfg)
    if err != nil {
        log.Fatalf("Failed to create robot: %v", err)
    }
    defer r.Close(ctx)

    // Handle shutdown signals
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigCh
        log.Println("Shutting down...")
        cancel()
    }()

    // Run the robot
    log.Printf("Robot %s starting...", cfg.Robot.Name)
    if err := r.Run(ctx); err != nil && err != context.Canceled {
        log.Fatalf("Robot error: %v", err)
    }
}
```

---

## Part 5: Deployment System

### Deployment Workflow

```
┌─────────────────────┐     ┌─────────────────────┐
│   Dev Machine       │     │   Robot (Pi)        │
│                     │     │                     │
│  1. Edit code       │     │                     │
│  2. Edit robot.json │     │                     │
│  3. make build-pi   │     │                     │
│  4. make deploy ────┼────►│  /opt/myrobot/      │
│                     │     │    myrobot          │
│                     │     │    robot.json       │
│                     │     │                     │
│                     │     │  systemctl restart  │
│                     │     │    myrobot.service  │
└─────────────────────┘     └─────────────────────┘
```

### Makefile Deploy Targets

```makefile
# Deployment configuration
ROBOT_HOST ?= pi@myrobot.local
ROBOT_PATH ?= /opt/myrobot
ROBOT_NAME ?= myrobot

# Deploy to robot
deploy: build-pi
	@echo "Deploying to $(ROBOT_HOST):$(ROBOT_PATH)"
	ssh $(ROBOT_HOST) "sudo mkdir -p $(ROBOT_PATH) && sudo chown pi:pi $(ROBOT_PATH)"
	rsync -avz --progress \
		build/bin/linux-arm64/$(ROBOT_NAME) \
		robot.json \
		$(ROBOT_HOST):$(ROBOT_PATH)/
	ssh $(ROBOT_HOST) "sudo systemctl restart $(ROBOT_NAME).service || true"

# Deploy config only (no rebuild)
deploy-config:
	rsync -avz robot.json $(ROBOT_HOST):$(ROBOT_PATH)/
	ssh $(ROBOT_HOST) "sudo systemctl restart $(ROBOT_NAME).service"

# Install systemd service
deploy-service:
	scp deploy/$(ROBOT_NAME).service $(ROBOT_HOST):/tmp/
	ssh $(ROBOT_HOST) "sudo mv /tmp/$(ROBOT_NAME).service /etc/systemd/system/ && \
		sudo systemctl daemon-reload && \
		sudo systemctl enable $(ROBOT_NAME).service"

# Full initial setup
deploy-init: build-pi deploy-service deploy
	@echo "Initial deployment complete"

# SSH to robot
ssh:
	ssh $(ROBOT_HOST)

# View robot logs
logs:
	ssh $(ROBOT_HOST) "sudo journalctl -u $(ROBOT_NAME).service -f"

# Check robot status
status:
	ssh $(ROBOT_HOST) "sudo systemctl status $(ROBOT_NAME).service"
```

### systemd Service Template

```ini
# deploy/myrobot.service
[Unit]
Description=MyRobot Gorai Robot
After=network.target nats.service
Wants=nats.service

[Service]
Type=simple
User=pi
Group=pi
WorkingDirectory=/opt/myrobot
ExecStart=/opt/myrobot/myrobot --config /opt/myrobot/robot.json
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# Hardware access
SupplementaryGroups=gpio i2c spi video dialout

# Resource limits
MemoryMax=512M
CPUQuota=80%

# Security
NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths=/opt/myrobot /tmp

[Install]
WantedBy=multi-user.target
```

### NATS Server on Robot

```ini
# /etc/systemd/system/nats.service
[Unit]
Description=NATS Message Broker
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/nats-server -js -sd /var/lib/nats
Restart=always
User=nats
Group=nats

[Install]
WantedBy=multi-user.target
```

---

## Part 6: Implementation Plan

### Phase 1: Specifications (Week 1)

Create formal specification documents:

| Document | Purpose | Location |
|----------|---------|----------|
| `specs/robot-definition-language.md` | RDL JSON schema specification | New |
| `specs/build-targets.md` | Supported platforms and builds | New |
| `specs/deployment.md` | Deployment methods and requirements | New |
| `specs/runtime.md` | Robot runtime behavior specification | New |

**Deliverables:**
- [ ] RDL JSON Schema (formal, with validation)
- [ ] Build target matrix
- [ ] Deployment protocol specification
- [ ] Runtime lifecycle specification

### Phase 2: Design Documents (Week 2)

Create detailed design documents:

| Document | Purpose | Location |
|----------|---------|----------|
| `docs/design/robot-runtime.md` | How robot.New() and robot.Run() work | New |
| `docs/design/config-loading.md` | Configuration loading and validation | New |
| `docs/design/deployment-tools.md` | Deploy scripts and tooling | New |

**Deliverables:**
- [ ] Runtime architecture diagrams
- [ ] Configuration flow diagrams
- [ ] Deployment sequence diagrams

### Phase 3: Core Implementation (Weeks 3-4)

#### 3.1 Configuration System

```
pkg/config/
├── config.go          # Existing - expand
├── schema.go          # NEW: JSON schema validation
├── validate.go        # NEW: Config validation
└── loader.go          # NEW: Hot reload support
```

**Tasks:**
- [ ] Define complete config structs matching RDL
- [ ] Implement JSON schema validation
- [ ] Add configuration validation (types exist, dependencies satisfied)
- [ ] Add hot-reload capability (watch file, SIGHUP)

#### 3.2 Robot Runtime

```
pkg/robot/
├── robot.go           # NEW: Main robot struct
├── lifecycle.go       # NEW: Start/stop/reconfigure
├── component_mgr.go   # NEW: Component lifecycle management
└── service_mgr.go     # NEW: Service lifecycle management
```

**Tasks:**
- [ ] Implement `robot.New(ctx, cfg)` - creates robot from config
- [ ] Implement `robot.Run(ctx)` - runs main loop
- [ ] Implement component instantiation from registry
- [ ] Implement service instantiation with dependencies
- [ ] Implement graceful shutdown

#### 3.3 Build System Updates

**Tasks:**
- [ ] Update Makefile with deploy targets
- [ ] Create project template/example
- [ ] Create systemd service template
- [ ] Test cross-compilation for all targets

### Phase 4: CLI Updates (Week 5)

Separate concerns - the `gorai` CLI is for **development**, not running robots:

```
cmd/
├── gorai/             # Development CLI
│   └── commands/
│       ├── root.go
│       ├── topic.go   # gorai topic list/echo/pub
│       ├── service.go # gorai service list/call
│       ├── config.go  # gorai config validate
│       └── version.go
└── gorai-runtime/     # Optional generic runtime
    └── main.go        # gorai-runtime config.json
```

**Tasks:**
- [ ] Implement `gorai topic list` - list active topics
- [ ] Implement `gorai topic echo <topic>` - subscribe and print
- [ ] Implement `gorai topic pub <topic> <data>` - publish message
- [ ] Implement `gorai service list` - list services
- [ ] Implement `gorai service call <service> <request>` - call service
- [ ] Implement `gorai config validate <config.json>` - validate config
- [ ] Create `gorai-runtime` as optional generic robot binary

### Phase 5: Examples and Templates (Week 6)

#### 5.1 Project Template

Create `examples/robot-template/`:
```
robot-template/
├── main.go
├── robot.json
├── Makefile
├── go.mod.template
├── deploy/
│   ├── robot.service
│   └── install.sh
└── README.md
```

#### 5.2 Complete Examples

Update existing examples to use new architecture:
- [ ] `examples/wheeled-robot/` - differential drive robot
- [ ] `examples/arm-robot/` - robot arm with gripper
- [ ] `examples/sensor-node/` - sensor-only node

### Phase 6: Documentation (Week 7)

#### 6.1 Book Updates

| Chapter | Updates |
|---------|---------|
| Ch 2: Getting Started | New build/deploy workflow |
| Ch 3: Architecture | Robot runtime, config system |
| Ch 10: Deployment | Full deployment guide |
| Ch 12: Hello Sensor | Update to new pattern |

#### 6.2 Website Updates

| Page | Updates |
|------|---------|
| `/docs/quickstart/` | New getting started guide |
| `/docs/guides/configuration/` | RDL reference |
| `/docs/guides/deployment/` | Deployment guide |
| `/docs/reference/config-schema/` | JSON schema reference |

#### 6.3 API Documentation

- [ ] Update godoc comments
- [ ] Generate API reference
- [ ] Configuration reference

---

## Part 7: Migration Path

### For Existing Code

The hello-sensor example currently works standalone. Migration:

**Before:**
```go
func main() {
    // Manual NATS connection
    nc, _ := nats.Connect(natsURL)
    // Manual component creation
    sensor := temperature.New(...)
    // Manual publishing loop
}
```

**After:**
```go
func main() {
    cfg, _ := config.Load("robot.json")
    r, _ := robot.New(ctx, cfg)
    r.Run(ctx)
}
```

**robot.json:**
```json
{
  "robot": { "name": "hello-sensor" },
  "components": [
    {
      "name": "cpu_temp",
      "type": "temperature",
      "model": "linux_thermal",
      "attributes": { "zone": "thermal_zone0" }
    }
  ]
}
```

### Backward Compatibility

- Existing examples continue to work (library usage)
- New config-driven approach is recommended, not required
- Registry pattern unchanged
- Component interfaces unchanged

---

## Part 8: Success Criteria

### Must Have

- [ ] `make build-pi` produces working ARM64 binary
- [ ] `make deploy` transfers binary and config to robot
- [ ] Robot starts from JSON configuration
- [ ] systemd service runs robot reliably
- [ ] Configuration validation catches errors before runtime

### Should Have

- [ ] Hot reload of configuration (SIGHUP)
- [ ] `gorai config validate` catches errors
- [ ] `gorai topic echo` works for debugging
- [ ] Complete example robot project

### Nice to Have

- [ ] Web-based configuration editor
- [ ] OTA update mechanism
- [ ] Fleet deployment support
- [ ] Docker/Podman deployment option

---

## Part 9: Open Questions

1. **NATS Deployment**: Should NATS run on the robot or externally?
   - Recommendation: On robot for standalone, external for multi-robot

2. **Secrets Management**: How to handle credentials in config?
   - Recommendation: Environment variables, not in JSON

3. **Multi-Node Robots**: How to deploy to multiple SBCs?
   - Recommendation: Multiple config files, one per node

4. **Remote Configuration**: Should config be pulled from server?
   - Recommendation: Phase 2 feature, start with local files

5. **TinyGo Integration**: How do microcontroller nodes fit?
   - Recommendation: Serial gateway pattern (already designed)

---

## Part 10: Timeline Summary

| Week | Phase | Deliverables |
|------|-------|--------------|
| 1 | Specifications | RDL spec, build spec, deploy spec |
| 2 | Design | Runtime design, config design, deploy design |
| 3-4 | Implementation | Config system, robot runtime, build system |
| 5 | CLI | Dev tools, optional runtime binary |
| 6 | Examples | Project template, complete examples |
| 7 | Documentation | Book, website, API docs |

**Total: 7 weeks to production-ready build/deploy system**

---

## Appendix A: Component Type Reference

For configuration, use these exact type strings:

### Sensors
```
imu, ahrs, gps, encoder, range_sensor, lidar,
presence_sensor, thermal_array, force_sensor, force_6dof,
current_sensor, reflectance_sensor, camera, temperature
```

### Actuators
```
motor, servo, stepper, thruster, valve, gripper, arm, base
```

### Infrastructure
```
power, space, link
```

### Services
```
vision, slam, navigation, motion, behavior, coordinator, mlmodel
```

---

## Appendix B: Target Architecture Reference

| Board | Architecture | Build Target |
|-------|--------------|--------------|
| Raspberry Pi 5 | ARM64 | `linux-arm64` |
| Raspberry Pi 4 | ARM64 | `linux-arm64` |
| Raspberry Pi 3 | ARMv7 | `linux-armv7` |
| Raspberry Pi Zero 2 | ARM64 | `linux-arm64` |
| Orange Pi 5 | ARM64 | `linux-arm64` |
| Jetson Nano | ARM64 | `linux-arm64` |
| Jetson Orin | ARM64 | `linux-arm64` |
| BeagleBone | ARMv7 | `linux-armv7` |
| x86 SBC | AMD64 | `linux-amd64` |
