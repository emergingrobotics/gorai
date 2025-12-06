# Gorai Code Organization Recommendation

## Executive Summary

**My strong recommendation: Hybrid monorepo with satellite modules.**

- One core monorepo (`github.com/gorai/gorai`) containing interfaces, core libraries, and reference implementations
- Separate repos for hardware-specific drivers, accelerator backends, and complex services
- Clear naming convention: `gorai-{category}-{name}`
- Self-registration pattern for extensibility

---

## The Core Question: Monorepo vs Multi-repo

### Why Not Pure Monorepo

A pure monorepo (everything in one repo) would be problematic for Gorai:

1. **Dependency bloat**: Users who only need GPIO motor control would pull in Coral TPU dependencies, CUDA libraries, SLAM algorithms, etc.

2. **Build complexity**: CGo dependencies for hardware (libedgetpu, HailoRT, OpenCV) would make the build matrix nightmarish

3. **Platform fragmentation**: Coral TPU only works on certain platforms; CUDA only on NVIDIA; Hailo only on specific SBCs. One repo means everyone deals with everyone's platform issues.

4. **Contribution friction**: Someone contributing a new I2C sensor driver shouldn't need to understand the SLAM service code

### Why Not Pure Multi-repo

A pure multi-repo (everything separate) would also be problematic:

1. **Version hell**: Coordinating compatible versions across 50+ repos is a nightmare (see early ROS2)

2. **API drift**: Interfaces defined in separate repos tend to diverge

3. **Discovery problem**: Hard for users to find what exists

4. **Testing gaps**: Integration testing across repos is painful

### The Hybrid Answer

**Core monorepo** for things that must stay in sync:
- All interfaces (component, service, accelerator)
- Core libraries (node, pub/sub, NATS integration)
- Protocol buffers
- CLI tool
- Reference implementations (CPU accelerator, basic drivers)
- Examples

**Satellite repos** for things that benefit from independence:
- Hardware-specific drivers requiring CGo/external deps
- Accelerator backends (TPU, NPU, CUDA)
- Complex services (SLAM, navigation with large dependencies)
- Community contributions

---

## Recommended Core Repository Structure

```
github.com/gorai/gorai/
├── go.mod                          # module github.com/gorai/gorai
├── go.sum
├── README.md
├── LICENSE                         # Apache 2.0
│
├── api/                            # Protocol Buffers (THE contract)
│   ├── proto/
│   │   └── gorai/
│   │       ├── std/
│   │       ├── geometry/
│   │       ├── sensor/
│   │       ├── control/
│   │       ├── vision/
│   │       ├── ml/
│   │       ├── nav/
│   │       └── action/
│   ├── gen/                        # Generated Go code (committed)
│   │   └── gorai/
│   └── buf.yaml
│
├── pkg/                            # Core implementation (stable)
│   ├── node/                       # Node lifecycle
│   ├── nats/                       # NATS abstraction layer
│   ├── pub/                        # Publisher
│   ├── sub/                        # Subscriber
│   ├── service/                    # Service server/client
│   ├── action/                     # Action server/client
│   ├── param/                      # Parameter store (NATS KV)
│   ├── tf/                         # Transform tree
│   ├── config/                     # Configuration loading
│   ├── registry/                   # Component/service registry
│   └── log/                        # Structured logging
│
├── component/                      # Component INTERFACES + base implementations
│   ├── component.go                # Base Component interface
│   ├── motor/
│   │   ├── motor.go                # Motor interface
│   │   └── fake/                   # Fake motor for testing
│   ├── camera/
│   │   ├── camera.go               # Camera interface
│   │   └── fake/                   # Fake camera for testing
│   ├── sensor/
│   ├── base/
│   ├── arm/
│   └── gripper/
│
├── service/                        # Service INTERFACES + simple implementations
│   ├── service.go                  # Base Service interface
│   ├── vision/
│   │   ├── vision.go               # Vision service interface
│   │   └── simple/                 # Simple OpenCV-free implementation
│   ├── mlmodel/
│   │   ├── mlmodel.go              # ML model interface
│   │   └── onnx/                   # ONNX runtime (if deps are reasonable)
│   ├── slam/
│   │   └── slam.go                 # SLAM interface only (impl separate)
│   ├── navigation/
│   │   └── navigation.go           # Navigation interface only
│   └── motion/
│       └── motion.go               # Motion planning interface
│
├── accel/                          # Acceleration INTERFACES + CPU impl
│   ├── accel.go                    # Accelerator interface
│   ├── tensor/                     # Tensor types
│   │   ├── tensor.go
│   │   └── tensor_test.go
│   └── cpu/                        # CPU reference implementation
│       └── cpu.go
│
├── driver/                         # Driver INTERFACES + pure-Go implementations
│   ├── driver.go                   # Base driver interface
│   ├── gpio/                       # GPIO (pure Go, no CGo)
│   │   ├── gpio.go
│   │   └── periph/                 # periph.io based impl
│   ├── i2c/
│   ├── spi/
│   └── serial/
│
├── nws/                            # Network Wrapper Server/Client
│   ├── nws.go                      # NWS base
│   ├── nwc.go                      # NWC base
│   └── grpc/                       # gRPC-based wrappers
│
├── cmd/
│   └── gorai/                      # CLI tool
│       ├── main.go
│       └── commands/
│
├── examples/                       # Reference examples (always tested)
│   ├── minimal/
│   ├── pubsub/
│   ├── motor/
│   ├── camera/
│   └── vision/
│
├── internal/                       # Internal packages (not importable)
│   ├── testutil/
│   └── proto/
│
└── docs/
    ├── getting-started.md
    ├── concepts.md
    └── contributing.md
```

---

## Satellite Repository Naming Convention

Use a **prefix-based naming scheme** for discoverability:

### Pattern: `gorai-{category}-{name}`

| Category | Pattern | Examples |
|----------|---------|----------|
| Drivers | `gorai-driver-{name}` | `gorai-driver-v4l2`, `gorai-driver-realsense`, `gorai-driver-rplidar` |
| Accelerators | `gorai-accel-{name}` | `gorai-accel-coral`, `gorai-accel-hailo`, `gorai-accel-cuda`, `gorai-accel-rockchip` |
| Services | `gorai-service-{name}` | `gorai-service-slam-cartographer`, `gorai-service-nav-movebase` |
| Components | `gorai-component-{name}` | `gorai-component-dynamixel`, `gorai-component-roboclaw` |
| Robots | `gorai-robot-{name}` | `gorai-robot-turtlebot`, `gorai-robot-mycobot` |
| Examples | `gorai-example-{name}` | `gorai-example-warehouse-bot` |

### Why Prefix Over Suffix

- `gorai-driver-*` sorts together in GitHub org listing
- Easy to search: `org:gorai driver-` finds all drivers
- Clear categorization at a glance
- Matches successful patterns (terraform-provider-*, prometheus-*-exporter)

### Module Names

Each satellite repo is its own Go module:

```go
// github.com/gorai/gorai-driver-v4l2/go.mod
module github.com/gorai/gorai-driver-v4l2

require github.com/gorai/gorai v0.1.0
```

Import paths are clean:
```go
import (
    "github.com/gorai/gorai/component/camera"
    "github.com/gorai/gorai-driver-v4l2"
)
```

---

## Registration Pattern

Use self-registration via `init()` for plugin-like extensibility:

### In the core repo (interface + registry):

```go
// github.com/gorai/gorai/pkg/registry/registry.go
package registry

import "sync"

type Constructor func(ctx context.Context, deps Dependencies, conf Config) (any, error)

var (
    mu         sync.RWMutex
    components = make(map[string]map[string]Constructor) // subtype -> model -> constructor
)

// RegisterComponent registers a component implementation.
func RegisterComponent(subtype, model string, ctor Constructor) {
    mu.Lock()
    defer mu.Unlock()
    if components[subtype] == nil {
        components[subtype] = make(map[string]Constructor)
    }
    components[subtype][model] = ctor
}

// Lookup finds a registered constructor.
func Lookup(subtype, model string) (Constructor, bool) {
    mu.RLock()
    defer mu.RUnlock()
    if m, ok := components[subtype]; ok {
        ctor, ok := m[model]
        return ctor, ok
    }
    return nil, false
}
```

### In a satellite repo (implementation):

```go
// github.com/gorai/gorai-driver-v4l2/v4l2.go
package v4l2

import (
    "github.com/gorai/gorai/component/camera"
    "github.com/gorai/gorai/pkg/registry"
)

func init() {
    registry.RegisterComponent("camera", "v4l2", New)
}

// New creates a new V4L2 camera.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
    // ... implementation
}
```

### User code:

```go
package main

import (
    "github.com/gorai/gorai/pkg/node"
    _ "github.com/gorai/gorai-driver-v4l2"  // Register V4L2 driver
    _ "github.com/gorai/gorai-accel-coral"  // Register Coral TPU
)

func main() {
    // Config references "model": "v4l2" and it just works
    node.Run("robot.json")
}
```

---

## What Goes Where: Decision Framework

### Put in CORE repo if:

1. It's an **interface** that others implement
2. It's **required** by most users (node, pub/sub, config)
3. It has **no CGo dependencies** or only optional ones
4. It's the **reference implementation** (CPU accelerator, fake components)
5. It's an **example** that should always be tested

### Put in SATELLITE repo if:

1. It has **heavy CGo dependencies** (OpenCV, libedgetpu, CUDA)
2. It's **platform-specific** (Coral only works on certain platforms)
3. It's **large** (SLAM algorithms, navigation stacks)
4. It could have **different maintainers** (community drivers)
5. It has **different release cadence** than core

### Concrete Examples:

| Component | Location | Reason |
|-----------|----------|--------|
| Motor interface | Core | Interface definition |
| Fake motor | Core | Testing, no deps |
| GPIO motor (periph.io) | Core | Pure Go, common |
| Dynamixel motor | Satellite | Specific protocol, niche |
| Camera interface | Core | Interface definition |
| V4L2 camera | Satellite | CGo, Linux-specific |
| RealSense camera | Satellite | Heavy deps (librealsense) |
| Vision interface | Core | Interface definition |
| Simple vision | Core | Minimal deps |
| YOLO vision | Satellite | Requires model files, ONNX |
| CPU accelerator | Core | Reference impl, no deps |
| Coral TPU | Satellite | libedgetpu CGo |
| CUDA | Satellite | NVIDIA-specific |
| ONNX runtime | Core or Satellite | Depends on dep weight |

---

## TinyGo Compatibility

For microcontroller targets, two approaches:

### Option A: Build Tags in Core

```go
// +build !tinygo

package node

// Full implementation with reflection, etc.
```

```go
// +build tinygo

package node

// Minimal implementation without reflection
```

**Pros**: Single repo
**Cons**: Complexity, easy to break TinyGo compat

### Option B: Separate TinyGo Module

```
github.com/gorai/gorai-tiny/
├── go.mod           # module github.com/gorai/gorai-tiny
├── node/            # TinyGo-compatible node
├── pub/             # TinyGo-compatible publisher
└── driver/
    └── gpio/        # TinyGo GPIO
```

**Pros**: Clean separation, explicit compatibility
**Cons**: Potential code duplication

**My recommendation**: Start with Option B. TinyGo has significant limitations (no reflection, limited stdlib). Trying to share code leads to constant breakage. A dedicated tiny module, even with some duplication, is more maintainable.

---

## Versioning Strategy

### Core Repo

Use Go modules semantic versioning:
- `v0.x.y` during initial development
- `v1.0.0` when interfaces stabilize
- `v2+` follows Go modules convention (path suffix)

### Satellite Repos

- Version independently from core
- Specify compatible core versions in `go.mod`:
  ```
  require github.com/gorai/gorai v0.5.0
  ```
- Consider version ranges when stable:
  ```
  require github.com/gorai/gorai v1.0.0
  ```

### Compatibility Matrix

Maintain a compatibility matrix in core repo:

| Satellite | v0.1.x | v0.2.x | v0.3.x |
|-----------|--------|--------|--------|
| gorai-driver-v4l2 | 0.1.0+ | 0.2.0+ | 0.3.0+ |
| gorai-accel-coral | - | 0.1.0+ | 0.2.0+ |

---

## Discovery and Documentation

### Official Registry

Maintain a curated list in core repo (`docs/ecosystem.md`):

```markdown
## Official Drivers

| Name | Platform | Status |
|------|----------|--------|
| [gorai-driver-v4l2](https://github.com/gorai/gorai-driver-v4l2) | Linux | Stable |
| [gorai-driver-realsense](https://github.com/gorai/gorai-driver-realsense) | Linux | Beta |

## Community Drivers

| Name | Maintainer | Platform |
|------|------------|----------|
| [gorai-driver-custom-lidar](https://github.com/user/gorai-driver-custom-lidar) | @user | Linux |
```

### pkg.go.dev

All repos under `github.com/gorai/*` will appear together on pkg.go.dev, making discovery natural.

### GitHub Topics

Use consistent GitHub topics:
- `gorai`
- `gorai-driver` / `gorai-accel` / `gorai-service`
- `robotics`
- `go`

---

## Anti-Patterns to Avoid

### 1. The "contrib" Graveyard

Don't create `gorai-contrib` or `gorai-community` repos. These become unmaintained dumping grounds. Instead:
- Let community publish under their own orgs
- Curate a list of known-good community packages
- Promote well-maintained packages to the official org

### 2. Over-Fragmentation

Don't create separate repos for every variation:
- Bad: `gorai-driver-gpio-rpi`, `gorai-driver-gpio-jetson`, `gorai-driver-gpio-beaglebone`
- Good: `gorai-driver-gpio` with platform-specific code paths

### 3. Interface Repos

Don't put interfaces in separate repos from core:
- Bad: `gorai-interfaces`, `gorai-api`
- Good: Interfaces in `github.com/gorai/gorai/component/*`

### 4. Premature Extraction

Don't extract to satellite repos too early. Start in core, extract when:
- Dependencies become problematic
- Different release cadence is needed
- Clear ownership boundary exists

---

## Recommended Initial Setup

### Phase 1: Core Only

Start with everything in the core repo:
```
github.com/gorai/gorai/
├── api/
├── pkg/
├── component/
├── service/
├── accel/cpu/
├── driver/gpio/
├── cmd/
└── examples/
```

### Phase 2: Extract Heavy Deps

When you add first CGo-heavy driver, extract:
```
github.com/gorai/gorai-driver-v4l2/
github.com/gorai/gorai-accel-coral/
```

### Phase 3: Community Growth

As community grows, encourage pattern:
```
github.com/someone/gorai-driver-custom/
github.com/company/gorai-robot-product/
```

Promote well-maintained ones to official org.

---

## Summary

| Aspect | Recommendation |
|--------|----------------|
| **Core structure** | Monorepo with interfaces, core libs, reference impls |
| **Satellite repos** | For CGo/platform-specific/heavy deps |
| **Naming** | `gorai-{category}-{name}` prefix pattern |
| **Discovery** | Registration via `init()`, curated list in docs |
| **Versioning** | Independent semver, compatibility matrix |
| **TinyGo** | Separate `gorai-tiny` module |
| **Community** | Own repos, curated list, promotion path |

This structure provides:
- Clean imports for users
- Minimal dependency footprint
- Easy contribution path
- Clear ownership boundaries
- Sustainable long-term growth
