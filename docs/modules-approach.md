# Gorai Multi-Repository Component Architecture

**Version**: 1.0
**Date**: 2026-01-16
**Status**: Proposed

---

## Executive Summary

This document recommends a **hybrid multi-repository architecture** for Gorai components and services that leverages:

1. **Go Modules** for in-process components (hardware drivers, internal services)
2. **Container Registries** for external services (ML models, vision pipelines, SLAM)
3. **Service RDL Registry** for reusable service definitions

This approach maintains Gorai's configuration-driven philosophy while enabling:
- Independent component development and versioning
- Private/proprietary component repositories
- Language-agnostic external services
- Zero changes to the RDL format

**Key Insight**: The RDL doesn't need to know about repos. It references components by `type+model`. The user's robot binary controls which components are available via Go imports.

---

## Architecture Overview

### Current State (Monolithic)

```
gorai/gorai (single repo)
├── components/          # All component implementations
├── services/            # All service implementations
├── pkg/                # Core framework
└── cmd/                # CLI & runtime

User Robot Binary:
- Imports github.com/gorai/gorai
- Gets ALL components/services
- RDL selects which to instantiate
```

### Proposed State (Modular)

```
gorai/gorai                          # Core framework + base interfaces
gorai/gorai-component-motor          # Motor implementations
gorai/gorai-component-sensor         # Sensor implementations
gorai/gorai-service-vision           # Vision service implementations
acme-corp/gorai-driver-custom-lidar  # Private company driver

User Robot Binary:
- Imports only needed component repos
- Registry pattern auto-discovers available implementations
- RDL validates referenced types exist at runtime
- External services pulled from container registry
```

---

## Component Distribution Strategies

### 1. In-Process Components: Go Modules

**For**: Hardware drivers, internal services, core components written in Go

#### Repository Structure

Each component category becomes its own module:

```
github.com/gorai/gorai-component-motor
├── go.mod              # module github.com/gorai/gorai-component-motor
├── gpio/               # GPIO motor implementation
│   ├── gpio.go
│   └── gpio_test.go
├── can/                # CAN motor implementation
├── odrive/             # ODrive motor implementation
└── fake/               # Fake motor for testing
    └── fake.go

github.com/gorai/gorai-component-sensor
├── go.mod              # module github.com/gorai/gorai-component-sensor
├── imu/
│   ├── mpu6050/
│   ├── bno055/
│   └── fake/
├── gps/
├── lidar/
└── encoder/

github.com/gorai/gorai-service-vision
├── go.mod              # module github.com/gorai/gorai-service-vision
├── yolox/              # YOLOX object detection
├── yolov8/             # YOLOv8 implementation
└── tflite/             # TensorFlow Lite models
```

#### Integration Pattern

**Step 1**: User imports needed components in their robot binary

```go
// my-robot/main.go
package main

import (
    "github.com/gorai/gorai/pkg/robot"

    // Import component packages (triggers init() registration)
    _ "github.com/gorai/gorai-component-motor/gpio"
    _ "github.com/gorai/gorai-component-sensor/imu/mpu6050"
    _ "github.com/acme-corp/gorai-driver-custom-lidar"
)

func main() {
    robot.RunFromConfig("robot.json")
}
```

**Step 2**: Component packages self-register in `init()`

```go
// gorai-component-motor/gpio/gpio.go
package gpio

import "github.com/gorai/gorai/pkg/registry"

func init() {
    registry.RegisterComponent("motor", "gpio", NewGPIOMotor)
}

func NewGPIOMotor(ctx context.Context, deps resource.Dependencies, conf resource.Config) (resource.Resource, error) {
    // Implementation...
}
```

**Step 3**: RDL references components by type+model (no import knowledge needed)

```json
{
  "components": [
    {
      "name": "left_motor",
      "type": "motor",
      "model": "gpio",
      "attributes": {
        "pin_forward": 17,
        "pin_reverse": 18
      }
    }
  ]
}
```

**Step 4**: Runtime validates referenced components exist

```go
// In pkg/config/config.go
func (c *Config) Validate() error {
    for _, comp := range c.Components {
        if !registry.HasComponent(comp.Type, comp.Model) {
            return fmt.Errorf("component type=%s model=%s not available (did you forget to import it?)",
                comp.Type, comp.Model)
        }
    }
}
```

#### Versioning & Dependencies

```go
// my-robot/go.mod
module github.com/acme-corp/my-robot

go 1.22

require (
    github.com/gorai/gorai v0.3.0
    github.com/gorai/gorai-component-motor v0.2.1
    github.com/gorai/gorai-component-sensor v0.2.0
    github.com/acme-corp/gorai-driver-custom-lidar v1.0.0  // Private repo
)
```

#### Private Repository Support

**Option A**: Use `GOPRIVATE` environment variable

```bash
# Configure Go to treat acme-corp repos as private
export GOPRIVATE=github.com/acme-corp/*

# Configure Git credentials (HTTPS or SSH)
git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"

# Go commands work normally
go get github.com/acme-corp/gorai-driver-custom-lidar
go build ./...
```

**Option B**: Use `replace` directive for local development

```go
// my-robot/go.mod
module github.com/acme-corp/my-robot

require (
    github.com/gorai/gorai v0.3.0
    github.com/acme-corp/gorai-driver-custom-lidar v1.0.0
)

// Local development override
replace github.com/acme-corp/gorai-driver-custom-lidar => ../gorai-driver-custom-lidar
```

#### Advantages

✅ Native Go tooling (go get, go mod tidy, go mod vendor)
✅ Semantic versioning built-in
✅ Transitive dependency management
✅ Private repos supported via GOPRIVATE
✅ Local development via `replace` directives
✅ No manual submodule management
✅ Works seamlessly with CI/CD

#### Disadvantages

❌ Only works for Go code
❌ Requires importing packages (can't add components without recompile)
❌ Private repos need authentication setup
❌ Go module proxy caching can cause issues with private repos

---

### 2. External Services: Container Registries

**For**: Language-agnostic services (Python ML models, C++ SLAM, ROS 2 bridges)

#### Repository Structure

Each external service is its own repo with:
- Source code (any language)
- Containerfile/Dockerfile
- Service RDL definition
- Build scripts

```
github.com/gorai/gorai-service-yolox
├── Containerfile           # Multi-stage build
├── service.rdl.json        # Reusable service definition
├── requirements.txt        # Python dependencies
├── src/
│   ├── main.py            # NATS subscriber + inference loop
│   ├── model_loader.py
│   └── ...
└── models/
    └── yolox_s.onnx       # Model weights

github.com/acme-corp/custom-vision-pipeline
├── Containerfile
├── service.rdl.json
├── src/
└── models/
```

#### Integration Pattern

**Step 1**: Build and push container image

```bash
# Public image
cd gorai-service-yolox
podman build -t ghcr.io/gorai/yolox:v1.0.0 .
podman push ghcr.io/gorai/yolox:v1.0.0

# Private image
podman build -t registry.acme-corp.com/vision-pipeline:latest .
podman login registry.acme-corp.com
podman push registry.acme-corp.com/vision-pipeline:latest
```

**Step 2**: Reference container image in RDL

```json
{
  "services": [
    {
      "name": "object_detector",
      "type": "vision",
      "model": "yolox",
      "rdl": "https://raw.githubusercontent.com/gorai/gorai-service-yolox/v1.0.0/service.rdl.json",
      "attributes": {
        "confidence_threshold": 0.6
      },
      "external": {
        "enabled": true,
        "container": {
          "image": "ghcr.io/gorai/yolox:v1.0.0",
          "pull_policy": "IfNotPresent"
        }
      }
    }
  ]
}
```

**Step 3**: Runtime pulls and launches container

```go
// In pkg/runtime/external.go
func (m *Manager) StartExternalService(ctx context.Context, svc ServiceConfig) error {
    // Pull image if needed
    if svc.External.Container.PullPolicy == "Always" || !imageExists(svc.External.Container.Image) {
        if err := pullImage(ctx, svc.External.Container.Image); err != nil {
            return err
        }
    }

    // Start container with NATS connection
    return m.containerRuntime.Start(ctx, ContainerSpec{
        Image: svc.External.Container.Image,
        Env: map[string]string{
            "NATS_URL": m.natsURL,
            "SERVICE_NAME": svc.Name,
            "NAMESPACE": m.namespace,
        },
    })
}
```

#### Service RDL Distribution

Service RDL files define reusable service templates:

**Option A**: Include in component repos (recommended)

```
github.com/gorai/gorai-service-yolox
├── service.rdl.json        # Authoritative definition
└── ...

# User references via URL
"rdl": "https://raw.githubusercontent.com/gorai/gorai-service-yolox/v1.0.0/service.rdl.json"
```

**Option B**: Central registry (future enhancement)

```
registry.gorai.dev/services/yolox/v1.0.0.rdl.json
registry.acme-corp.com/services/custom-vision/latest.rdl.json

# User references via URL
"rdl": "registry://gorai.dev/services/yolox:v1.0.0"
```

#### Private Container Registry Support

**Authentication**: Standard container registry mechanisms

```bash
# Login to private registry
podman login registry.acme-corp.com

# Credentials stored in ${XDG_RUNTIME_DIR}/containers/auth.json
# Gorai runtime uses existing credentials
```

**Pull Policy**: Control image freshness

```json
{
  "external": {
    "container": {
      "image": "registry.acme-corp.com/vision:latest",
      "pull_policy": "Always"  // Options: Always, IfNotPresent, Never
    }
  }
}
```

#### Advantages

✅ Language-agnostic (Python, C++, Rust, anything)
✅ Standard distribution mechanism
✅ Well-understood authentication (registry login)
✅ Can bundle large model weights
✅ Isolation (dependencies don't conflict)
✅ Works with existing Gorai external service architecture
✅ Private registries widely supported

#### Disadvantages

❌ Larger artifacts (GB-sized images)
❌ Requires container build infrastructure
❌ Slower iteration (build + push + pull cycle)
❌ Storage costs for private registries

---

## Recommended Repository Organization

### Core Framework

```
github.com/gorai/gorai
├── components/           # Base interfaces ONLY (no implementations except fake/)
│   ├── motor/
│   │   ├── motor.go          # Motor interface
│   │   └── fake/fake.go      # Fake motor for testing
│   ├── sensor/
│   └── ...
├── services/             # Base interfaces ONLY
├── pkg/                 # Core runtime, config, registry, nats, dashboard
├── driver/              # Low-level driver interfaces (gpio, i2c, spi)
├── cmd/
│   ├── gorai/          # CLI tool
│   └── gorai-robot/    # Generic robot runtime
└── examples/
    └── hello-robot/    # Minimal example (imports public components)
```

**Dependencies**: Minimal (NATS, Chi, Prometheus client)

### Component Repositories (Go Modules)

**Public Components**:

```
github.com/gorai/gorai-component-motor
github.com/gorai/gorai-component-sensor
github.com/gorai/gorai-component-camera
github.com/gorai/gorai-component-actuator
```

**Platform-Specific Drivers**:

```
github.com/gorai/gorai-driver-hailo       # Hailo AI accelerator
github.com/gorai/gorai-driver-camera-v4l2 # V4L2 camera support
github.com/gorai/gorai-driver-lidar-rplidar
```

**Private/Company Repositories**:

```
github.com/acme-corp/gorai-driver-custom-sensor
github.com/acme-corp/gorai-component-proprietary-arm
```

### Service Repositories (Containers + Go)

**Public Services**:

```
github.com/gorai/gorai-service-vision      # YOLOX, YOLOv8, etc.
github.com/gorai/gorai-service-slam        # Cartographer, etc.
github.com/gorai/gorai-service-navigation  # Path planning
```

**Each service repo contains**:
- Go interface implementation (optional, for in-process fallback)
- Containerfile + multi-language implementation
- Service RDL definition
- Model weights or download scripts

### Platform Repositories

```
github.com/gorai/gorai-platform-k3s       # K3s deployment tools
github.com/gorai/gorai-platform-systemd   # systemd integration
```

---

## Migration Strategy

### Phase 1: Extract External Services (Low Risk)

**Target**: Services that already run in containers

```
gorai-service-vision
gorai-service-slam
gorai-service-navigation
```

**Steps**:
1. Create new repo
2. Copy service code + Containerfile
3. Add service.rdl.json
4. Build & push container to ghcr.io
5. Update examples to reference container images
6. Deprecate old service code in main repo

**Impact**: Zero breaking changes (examples reference containers, not code)

### Phase 2: Extract Platform Drivers (Medium Risk)

**Target**: Hardware-specific drivers (Hailo, V4L2, etc.)

```
gorai-driver-hailo
gorai-driver-camera-v4l2
gorai-driver-lidar-rplidar
```

**Steps**:
1. Create new repo with go.mod
2. Move driver code
3. Update main repo to remove implementation, keep interface
4. Add import to examples
5. Tag v0.1.0 release

**Impact**: Examples need updated imports (one-line change per component)

### Phase 3: Extract Core Components (Higher Risk)

**Target**: Fundamental components (motors, sensors, etc.)

```
gorai-component-motor
gorai-component-sensor
gorai-component-actuator
```

**Steps**:
1. Create new repos
2. Move implementations (keep interfaces in core)
3. Update core repo to import from new repos in cmd/gorai-robot
4. Tag synchronized releases
5. Update all examples

**Impact**: All user robots need import updates

### Phase 4: Stabilize & Document (Essential)

1. Create migration guide
2. Add `gorai migrate` CLI command to auto-update imports
3. Publish recommended starter templates
4. Update documentation

---

## Developer Experience

### Creating a New Robot (User Perspective)

**Before** (Monolithic):
```bash
mkdir my-robot && cd my-robot
go mod init github.com/me/my-robot
go get github.com/gorai/gorai
```

```go
package main
import (
    "github.com/gorai/gorai/pkg/robot"
    _ "github.com/gorai/gorai/components/motor/gpio"  // All components in one repo
    _ "github.com/gorai/gorai/components/sensor/imu"
)
func main() { robot.RunFromConfig("robot.json") }
```

**After** (Modular):
```bash
mkdir my-robot && cd my-robot
go mod init github.com/me/my-robot
go get github.com/gorai/gorai
go get github.com/gorai/gorai-component-motor
go get github.com/gorai/gorai-component-sensor
```

```go
package main
import (
    "github.com/gorai/gorai/pkg/robot"
    _ "github.com/gorai/gorai-component-motor/gpio"      // Per-category repos
    _ "github.com/gorai/gorai-component-sensor/imu/mpu6050"
)
func main() { robot.RunFromConfig("robot.json") }
```

**Improvement**: CLI scaffolding command

```bash
gorai new my-robot --template differential-drive
# Generates:
# - go.mod with correct dependencies
# - main.go with necessary imports
# - robot.json with example config
# - README.md with instructions
```

### Creating a Custom Component (Developer Perspective)

**Public Component**:

```bash
# Clone template
git clone https://github.com/gorai/gorai-component-template my-component
cd my-component

# Customize
vim mydevice/mydevice.go
# Implement resource.Resource interface
# Add init() registration

# Test
go test ./...

# Publish
git tag v0.1.0
git push origin v0.1.0

# Others can now use it
go get github.com/me/my-component
```

**Private Component**:

```bash
# Same as above, but in private repo
# Users configure GOPRIVATE:
export GOPRIVATE=github.com/acme-corp/*
go get github.com/acme-corp/gorai-driver-custom@v1.0.0
```

### Creating a Custom External Service

```bash
# Clone template
git clone https://github.com/gorai/gorai-service-template my-service
cd my-service

# Implement in any language
vim src/main.py
# - Subscribe to NATS topics
# - Publish results

# Create service RDL
vim service.rdl.json

# Build container
podman build -t ghcr.io/me/my-service:v1.0.0 .
podman push ghcr.io/me/my-service:v1.0.0

# Users reference in robot.json
"services": [{
  "name": "my_service",
  "rdl": "https://github.com/me/my-service/raw/v1.0.0/service.rdl.json",
  "external": {
    "container": {"image": "ghcr.io/me/my-service:v1.0.0"}
  }
}]
```

---

## Comparison: Submodules vs Go Modules

### Git Submodules Approach (NOT RECOMMENDED)

```
gorai/gorai
├── .gitmodules
├── components/
│   ├── motor/          → git submodule github.com/gorai/gorai-component-motor
│   ├── sensor/         → git submodule github.com/gorai/gorai-component-sensor
│   └── ...
└── services/
    └── vision/         → git submodule github.com/gorai/gorai-service-vision
```

**Why Not?**

❌ Requires `git submodule update --init --recursive` (users forget)
❌ Detached HEAD state causes confusion
❌ Difficult to track versions
❌ Doesn't integrate with Go modules
❌ Manual management overhead
❌ CI/CD complexity
❌ Universally disliked by developers

**When It Makes Sense:**

- Mixed-language projects where Go modules don't work
- Need to vendor non-Go code directly into tree
- Building monorepo with non-module dependencies

**Verdict**: Don't use submodules for Gorai. Go modules solve this better.

### Go Modules Approach (RECOMMENDED)

**Advantages**:

✅ Native Go tooling
✅ Semantic versioning
✅ Transitive dependencies handled automatically
✅ Works with private repos via GOPRIVATE
✅ `replace` for local development
✅ Widely understood by Go developers
✅ CI/CD friendly
✅ Module proxy caching (faster builds)

**Disadvantages**:

❌ Only for Go code (but that's fine—use containers for non-Go)
❌ Private repos need authentication setup (one-time per developer)
❌ Requires recompilation to change components

**Verdict**: Use Go modules for all Go components. This is the idiomatic approach.

---

## Handling Private Repositories

### For Go Modules

**Setup** (one-time per developer):

```bash
# Method 1: GOPRIVATE environment variable
echo 'export GOPRIVATE=github.com/acme-corp/*,github.com/private-org/*' >> ~/.bashrc
source ~/.bashrc

# Method 2: Git credential helper (for HTTPS)
git config --global credential.helper store
git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"

# Method 3: SSH (recommended)
git config --global url."git@github.com:".insteadOf "https://github.com/"
# Ensure SSH keys are set up with GitHub

# Verify
go env GOPRIVATE
go get github.com/acme-corp/private-component@v1.0.0
```

**CI/CD Setup**:

```yaml
# GitHub Actions
- name: Configure Go for private repos
  run: |
    git config --global url."https://${{ secrets.REPO_ACCESS_TOKEN }}@github.com/".insteadOf "https://github.com/"
    go env -w GOPRIVATE=github.com/acme-corp/*

# GitLab CI
before_script:
  - git config --global url."https://gitlab-ci-token:${CI_JOB_TOKEN}@gitlab.com/".insteadOf "https://gitlab.com/"
  - go env -w GOPRIVATE=gitlab.com/acme-corp/*
```

### For Container Registries

**Setup**:

```bash
# Login to private registry (one-time)
podman login registry.acme-corp.com
# Username: admin
# Password: [token]

# Credentials stored in ${XDG_RUNTIME_DIR}/containers/auth.json
# Gorai automatically uses these credentials

# For multiple registries
podman login ghcr.io
podman login registry.acme-corp.com
podman login quay.io
```

**CI/CD Setup**:

```yaml
# GitHub Actions
- name: Login to private registry
  run: |
    echo "${{ secrets.REGISTRY_PASSWORD }}" | podman login -u "${{ secrets.REGISTRY_USERNAME }}" --password-stdin registry.acme-corp.com

# Pull images in robot.json work automatically after login
```

---

## RDL Integration Details

### No Changes to RDL Format

The RDL format **does not change**. It continues to reference components by `type+model`:

```json
{
  "components": [
    {
      "name": "lidar",
      "type": "sensor",
      "model": "rplidar_a1"
    }
  ]
}
```

The `type+model` lookup happens at **runtime** against the **registry**. The registry is populated by `init()` functions in imported packages.

### Validation & Error Messages

**Before compilation** (optional): `gorai validate robot.json`

```bash
$ gorai validate robot.json
✓ Valid RDL structure
✓ All component types exist in registry
✗ Component type=sensor model=rplidar_a1 not found

  Did you forget to import it?
  Try: go get github.com/gorai/gorai-driver-lidar-rplidar
```

**At runtime**:

```go
// pkg/config/config.go
func (c *Config) Validate() error {
    for _, comp := range c.Components {
        if _, err := registry.LookupComponent(comp.Type, comp.Model); err != nil {
            return &ComponentNotFoundError{
                Type:  comp.Type,
                Model: comp.Model,
                Suggestion: fmt.Sprintf("Import the package that provides this component, e.g.:\nimport _ \"github.com/gorai/gorai-component-motor/gpio\""),
            }
        }
    }
    return nil
}
```

### Service RDL Files

External services can reference **remote RDL files**:

```json
{
  "services": [
    {
      "name": "detector",
      "rdl": "https://raw.githubusercontent.com/gorai/gorai-service-vision/v1.2.0/yolox/service.rdl.json",
      "attributes": {
        "confidence": 0.7
      }
    }
  ]
}
```

**Caching**: Downloaded RDL files cached in `~/.cache/gorai/rdl/`

**Versioning**: URL should include version tag (v1.2.0) for reproducibility

**Local override**: For development, use local path

```json
"rdl": "./services/my-detector/service.rdl.json"
```

---

## Implementation Checklist

### Phase 1: Infrastructure (Week 1-2)

- [ ] Create component repository template
  - go.mod with gorai dependency
  - Example implementation
  - CI/CD for testing + releasing
- [ ] Create service repository template
  - Containerfile
  - service.rdl.json
  - Multi-language example
  - CI/CD for building + pushing images
- [ ] Add validation to CLI
  - `gorai validate robot.json` checks registry
  - Helpful error messages with import suggestions
- [ ] Add scaffolding to CLI
  - `gorai new robot --template <name>`
  - `gorai new component --type motor --model mymotor`
  - `gorai new service --type vision --model mydetector`

### Phase 2: Extract Services (Week 3-4)

- [ ] Extract gorai-service-vision
- [ ] Extract gorai-service-slam
- [ ] Extract gorai-service-navigation
- [ ] Update examples to reference container images
- [ ] Deprecation notices in main repo

### Phase 3: Extract Drivers (Week 5-6)

- [ ] Extract gorai-driver-hailo
- [ ] Extract gorai-driver-camera-v4l2
- [ ] Extract gorai-driver-lidar-rplidar
- [ ] Update examples with new imports
- [ ] Update documentation

### Phase 4: Extract Core Components (Week 7-8)

- [ ] Extract gorai-component-motor
- [ ] Extract gorai-component-sensor
- [ ] Extract gorai-component-actuator
- [ ] Update all examples
- [ ] Create migration guide

### Phase 5: Developer Experience (Week 9-10)

- [ ] Write migration guide
- [ ] Add `gorai migrate` command (auto-update imports)
- [ ] Create video tutorials
- [ ] Update all documentation
- [ ] Publish blog post explaining benefits

---

## FAQ

### Q: Why not use a monorepo with Bazel/Buck?

**A**: Monorepos solve different problems (build consistency, atomic changes). Gorai users want:
- Independent component versioning
- Private proprietary drivers
- Mix-and-match from multiple sources

Monorepos centralize control. We want to **decentralize** component development.

### Q: What if users want all components in one binary?

**A**: Import them all! The framework is flexible:

```go
import (
    _ "github.com/gorai/gorai-component-motor/gpio"
    _ "github.com/gorai/gorai-component-motor/can"
    _ "github.com/gorai/gorai-component-motor/odrive"
    // ...all components
)
```

Or use the "batteries-included" meta-package:

```go
import _ "github.com/gorai/gorai-stdlib"  // Imports all official components
```

### Q: How do I test my robot without recompiling?

**A**: Use external services in containers. Change container image in RDL, restart service:

```bash
# Edit robot.json to point to new image
vim robot.json

# Restart just that service
gorai service restart detector
```

For in-process components, recompilation is required (but fast with Go).

### Q: What about model weights for ML services?

**A**: Bundle in container image or download on first run:

```dockerfile
# Option 1: Bundle weights (simple but large image)
FROM python:3.11-slim
COPY models/yolox_s.onnx /models/
COPY src/ /app/
CMD ["python", "/app/main.py"]

# Option 2: Download on startup (smaller image, slower first run)
FROM python:3.11-slim
COPY src/ /app/
RUN pip install gdown
CMD ["sh", "-c", "gdown https://drive.google.com/uc?id=XXXXX -O /models/yolox_s.onnx && python /app/main.py"]
```

### Q: Can I vendor dependencies for offline use?

**A**: Yes, Go modules support vendoring:

```bash
go mod vendor
go build -mod=vendor ./...
```

For containers, build images and save as tarballs:

```bash
podman save ghcr.io/gorai/yolox:v1.0.0 -o yolox.tar
podman load -i yolox.tar  # On offline robot
```

### Q: What's the recommended versioning scheme?

**A**: Semantic versioning (semver):

- **Major version** (v1.0.0 → v2.0.0): Breaking API changes
- **Minor version** (v1.0.0 → v1.1.0): New features, backward compatible
- **Patch version** (v1.0.0 → v1.0.1): Bug fixes

Tag releases in Git:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Users pin versions in go.mod:

```go
require github.com/gorai/gorai-component-motor v1.2.3
```

Or use version ranges:

```go
require github.com/gorai/gorai-component-motor v1.2  // Any v1.2.x
```

---

## Third-Party Component Ecosystem

### Current Gaps for External Developers

While the Go modules + containers approach enables third-party development, several gaps exist:

1. **Discovery Problem**: No way to find available components except knowing exact import paths
2. **Metadata Gap**: Components can't self-describe capabilities, compatibility, or configuration
3. **Installation UX**: Manual `go get` + import is clunky compared to modern package managers
4. **Quality Signals**: No way to assess third-party component quality, maintenance, or compatibility
5. **Documentation**: External services lack standardized configuration documentation

### Solution: Component Metadata Standard

#### For Go Components: `gorai-component.yaml`

Every third-party component repository should include metadata:

```yaml
schema_version: "1.0"
component:
  name: "advanced-imu"
  repository: "github.com/robotics-lab/gorai-component-imu"
  version: "v1.2.0"

  provides:
    - type: "sensor"
      model: "bno085"
    - type: "sensor"
      model: "icm42688"

  compatibility:
    gorai_version: ">=0.3.0, <1.0.0"
    platforms: ["linux/arm64", "linux/amd64"]
    go_version: ">=1.22"

  author: "Robotics Lab"
  license: "MIT"
  description: "High-precision 9-DOF IMU drivers with calibration"
  homepage: "https://github.com/robotics-lab/gorai-component-imu"

  hardware_requirements:
    i2c: true
    gpio: false
```

See [specs/gorai-component-schema.yaml](../specs/gorai-component-schema.yaml) for full schema definition.

#### For Container Services: Enhanced `service.rdl.json`

External services should include comprehensive metadata:

```json
{
  "schema_version": "1.0",
  "service": {
    "name": "yolox-detector",
    "type": "vision",
    "model": "yolox",
    "version": "v1.2.0",

    "container": {
      "default_image": "ghcr.io/gorai/yolox:v1.2.0",
      "resource_requirements": {
        "memory": "2Gi",
        "cpu": "1"
      }
    },

    "nats_topics": {
      "subscribes": ["camera.{name}.frame"],
      "publishes": ["vision.{name}.detections"]
    },

    "configuration": {
      "confidence_threshold": {
        "type": "float",
        "default": 0.6,
        "range": [0.0, 1.0]
      }
    }
  }
}
```

See [specs/service-rdl-schema.json](../specs/service-rdl-schema.json) for full schema definition.

### Solution: CLI Commands for Discovery

Add component management commands to the Gorai CLI:

```bash
# Search for components
gorai component search imu
gorai component search --type sensor

# Show component info
gorai component info github.com/robotics-lab/gorai-component-imu

# Install component (adds to go.mod + shows import instructions)
gorai component add github.com/robotics-lab/gorai-component-imu@v1.2.0

# List installed components
gorai component list

# Validate component (for developers)
gorai component validate ./my-component
```

See [specs/cli-component-commands.md](../specs/cli-component-commands.md) for full CLI specification.

### Solution: Optional Component Registry

Create **registry.gorai.dev** (optional, not required):

- Searchable index of community components
- Submitted via PR to registry repo
- Validates metadata and compatibility
- Shows download stats, CI status, last updated
- Quality signals (test coverage, documentation)

**Important**: Registry is opt-in for discovery. Components can be distributed anywhere:
- Public GitHub repos
- Private git repos
- Internal artifact stores
- Corporate package registries

### Third-Party Developer Workflow

```
1. Create component
   ├── gorai new component --type sensor --model my-sensor
   ├── Implement resource.Resource interface
   └── Fill out gorai-component.yaml

2. Test & validate
   ├── go test ./...
   ├── gorai component validate .
   └── Ensure CI passes

3. Distribute
   ├── Git tag: git tag v1.0.0 && git push origin v1.0.0
   ├── Optional: Submit to registry.gorai.dev
   └── Documentation: Add usage examples to README

4. Users consume
   ├── Discovery: gorai component search my-sensor
   ├── Install: gorai component add github.com/me/gorai-component-my-sensor
   ├── Import: import _ "github.com/me/gorai-component-my-sensor/driver"
   └── Configure in robot.json
```

### Benefits

**For Third-Party Developers:**
- Clear path to create components outside Gorai repo
- Standard metadata format for self-description
- Validation tools ensure quality
- Optional discoverability via registry
- Template generators reduce boilerplate

**For Users:**
- Easy discovery of community components
- Confidence in compatibility & quality
- Simple installation workflow
- Clear documentation of configuration options
- Version management via go.mod

**For Gorai Project:**
- Ecosystem growth without core repo bloat
- Community innovation and contributions
- Maintained compatibility via version constraints
- Quality bar through validation tools
- Reduced maintenance burden

See [docs/third-party-component-ecosystem.md](third-party-component-ecosystem.md) for comprehensive developer guide.

---

## Conclusion

The recommended **hybrid approach** leverages:

1. **Go Modules** for in-process components (idiomatic Go, great DX)
2. **Container Registries** for external services (language-agnostic, standard distribution)
3. **Service RDL Registry** for reusable service definitions (configuration-driven composition)
4. **Component Metadata** for self-description and discovery
5. **CLI Tooling** for installation and validation
6. **Optional Registry** for community discovery

This architecture maintains Gorai's core philosophy:
- **Configuration-driven**: RDL defines robots, not code
- **Loosely coupled**: Components communicate via NATS, not function calls
- **Pluggable**: Registry pattern enables dynamic discovery
- **Pragmatic polyglot**: Go for core, any language for services
- **Ecosystem-friendly**: Clear path for third-party contributions

It enables:
- Independent component development & versioning
- Private/proprietary components via Go modules & container registries
- Thriving third-party ecosystem with discovery & validation
- Zero changes to RDL format
- Seamless migration path from monolithic repo

**Next Steps**:
1. Review & approve this proposal
2. Create repository templates with metadata files
3. Implement CLI component commands
4. Begin Phase 1 extraction (services)
5. Launch optional registry for community discovery
6. Iterate based on community feedback
