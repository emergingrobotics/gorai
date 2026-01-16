# Gorai CLI Component & Service Commands

**Version**: 1.0
**Date**: 2026-01-16
**Status**: Proposed

---

## Overview

This specification defines CLI commands for discovering, installing, managing, and validating third-party Gorai components and services. These commands make it easy for users to find and integrate external components without manually editing `go.mod` or searching GitHub.

**Goals**:
- Make third-party components as easy to discover as npm/pip packages
- Streamline installation workflow
- Validate components before use
- Provide helpful error messages and suggestions

---

## Table of Contents

1. [Component Commands](#component-commands)
2. [Service Commands](#service-commands)
3. [Validation Commands](#validation-commands)
4. [Configuration](#configuration)
5. [Implementation Details](#implementation-details)

---

## Component Commands

Commands for managing Go components (motors, sensors, etc.).

### `gorai component search`

Search for components by keyword, type, or capability.

**Syntax**:
```bash
gorai component search <query> [flags]
```

**Flags**:
- `--type <type>` - Filter by component type (motor, sensor, camera, etc.)
- `--capability <cap>` - Filter by capability (9dof, uart, i2c, etc.)
- `--platform <platform>` - Filter by platform (linux/arm64, linux/amd64, etc.)
- `--license <license>` - Filter by license (MIT, Apache-2.0, Proprietary, etc.)
- `--limit <n>` - Limit results (default: 20)
- `--sort <field>` - Sort by: relevance, updated, downloads, stars (default: relevance)

**Examples**:

```bash
# Search for IMU sensors
$ gorai component search imu

Found 5 components:

  github.com/robotics-lab/gorai-component-imu v1.2.3
    High-precision IMU drivers with calibration
    Provides: sensor/bno085, sensor/icm42688
    License: MIT | Updated: 2026-01-10 | ★ Stable

  github.com/sensors-inc/gorai-imu-advanced v2.0.1
    Industrial-grade IMU with temperature compensation
    Provides: sensor/bmi088, sensor/bmi270
    License: Proprietary | Updated: 2025-12-15 | ★ Mature

# Search by type
$ gorai component search --type motor

Found 8 components:

  github.com/gorai/gorai-component-motor v0.3.0
    Official Gorai motor drivers
    Provides: motor/gpio, motor/pwm, motor/fake
    License: Apache-2.0 | Updated: 2026-01-12 | ★ Stable

# Search by multiple criteria
$ gorai component search lidar --capability uart --platform linux/arm64

Found 2 components:

  github.com/acme-corp/gorai-lidar-360 v1.5.0
    360° LiDAR with UART interface
    Provides: sensor/rplidar-a1, sensor/rplidar-a2
    License: Proprietary | Updated: 2026-01-05 | ★ Stable
```

**Output Format**:
- Repository URL and version
- Description (truncated to fit terminal width)
- Provided types/models
- License, last update date, maturity level
- Sorted by relevance (keyword matching) by default

**Registry Source**:
- Queries `registry.gorai.dev` API (if configured)
- Falls back to GitHub search API
- Can be configured to use private registries

---

### `gorai component info`

Display detailed information about a component.

**Syntax**:
```bash
gorai component info <repository> [version]
```

**Arguments**:
- `<repository>` - Component repository (e.g., github.com/org/repo)
- `[version]` - Optional version (defaults to latest)

**Examples**:

```bash
$ gorai component info github.com/robotics-lab/gorai-component-imu

┌────────────────────────────────────────────────────────────────┐
│ Advanced IMU Component                                         │
└────────────────────────────────────────────────────────────────┘

Repository:   github.com/robotics-lab/gorai-component-imu
Version:      v1.2.3 (latest)
Author:       Robotics Lab
License:      MIT
Maturity:     Stable

Description:
  High-precision IMU drivers with advanced calibration and sensor
  fusion algorithms for robotics applications.

Provides:
  • sensor/bno085      Bosch BNO085 9-DOF IMU with sensor fusion
  • sensor/icm42688    TDK ICM-42688-P 6-axis IMU

Compatibility:
  Gorai:      >=0.3.0, <1.0.0
  Go:         >=1.22
  Platforms:  linux/arm64, linux/amd64

Hardware Requirements:
  ✓ I2C bus required
  ✗ GPIO not required
  ✗ UART not required

Configuration Attributes:
  • i2c_bus (int, required)
      I2C bus number (e.g., 1 for /dev/i2c-1)
      Example: 1

  • i2c_address (int, optional)
      I2C device address (default: 0x4A)
      Range: 0x00 - 0x7F

  • update_rate_hz (int, optional)
      Sensor update rate in Hz (default: 100)
      Options: 50, 100, 200, 400

  • enable_calibration (bool, optional)
      Enable automatic calibration on startup (default: true)

Installation:
  gorai component add github.com/robotics-lab/gorai-component-imu@v1.2.3

Documentation: https://docs.robotics-lab.com/gorai-imu
Support:       support@robotics-lab.com
Repository:    https://github.com/robotics-lab/gorai-component-imu

Quality Metrics:
  Test Coverage:       87%
  CI Status:           ✓ Passing
  Examples:            ✓ Included
  Integration Tests:   ✓ Included

$ gorai component info github.com/robotics-lab/gorai-component-imu v1.0.0
# Shows info for specific version
```

**Data Source**:
- Fetches `gorai-component.yaml` from repository
- Displays parsed metadata in human-readable format
- Caches results locally for 24 hours

---

### `gorai component add`

Add a component to the current robot project.

**Syntax**:
```bash
gorai component add <repository>[@version] [flags]
```

**Arguments**:
- `<repository>` - Component repository
- `[@version]` - Optional version (defaults to latest)

**Flags**:
- `--import` - Automatically add import statement to main.go
- `--models <models>` - Comma-separated list of models to import
- `--dry-run` - Show what would be done without making changes

**Examples**:

```bash
# Add component (basic)
$ gorai component add github.com/robotics-lab/gorai-component-imu

✓ Added github.com/robotics-lab/gorai-component-imu v1.2.3 to go.mod
✓ Downloaded dependencies

Next steps:
  1. Add import to your robot binary (e.g., cmd/robot/main.go):

     import _ "github.com/robotics-lab/gorai-component-imu/bno085"

  2. Configure in robot.json:

     {
       "components": [
         {
           "name": "imu",
           "type": "sensor",
           "model": "bno085",
           "attributes": {
             "i2c_bus": 1,
             "i2c_address": 74
           }
         }
       ]
     }

# Add with automatic import
$ gorai component add github.com/robotics-lab/gorai-component-imu --import

✓ Added github.com/robotics-lab/gorai-component-imu v1.2.3 to go.mod
✓ Added import to cmd/robot/main.go

Available models:
  • bno085
  • icm42688

Which model do you want to import? [bno085]: bno085

✓ Added: import _ "github.com/robotics-lab/gorai-component-imu/bno085"

Next: Configure in robot.json (see example above)

# Add specific version
$ gorai component add github.com/acme-corp/gorai-lidar@v1.0.0

✓ Added github.com/acme-corp/gorai-lidar v1.0.0 to go.mod

# Add with specific models
$ gorai component add github.com/robotics-lab/gorai-component-imu --models bno085,icm42688 --import

✓ Added github.com/robotics-lab/gorai-component-imu v1.2.3
✓ Added imports:
    import _ "github.com/robotics-lab/gorai-component-imu/bno085"
    import _ "github.com/robotics-lab/gorai-component-imu/icm42688"

# Dry run
$ gorai component add github.com/example/component --dry-run

Would add:
  • github.com/example/component v2.1.0 to go.mod
  • Import statement to cmd/robot/main.go

No changes made (dry run)
```

**Behavior**:
1. Validates component exists and version is valid
2. Runs `go get <repository>@<version>`
3. Updates `go.mod` and `go.sum`
4. Optionally adds import to main.go (with user confirmation)
5. Displays next steps with example configuration

**Error Handling**:
```bash
$ gorai component add github.com/nonexistent/component

✗ Error: Component not found
  Repository: github.com/nonexistent/component

  Suggestions:
    • Check the repository URL is correct
    • Try searching: gorai component search <keyword>

$ gorai component add github.com/example/component@v99.0.0

✗ Error: Version v99.0.0 not found
  Available versions: v1.0.0, v1.1.0, v1.2.0

  Try: gorai component add github.com/example/component@v1.2.0
```

---

### `gorai component list`

List all components available in the current robot binary.

**Syntax**:
```bash
gorai component list [flags]
```

**Flags**:
- `--type <type>` - Filter by type
- `--source <source>` - Filter by source (builtin, external, local)
- `--format <format>` - Output format: table (default), json, yaml

**Examples**:

```bash
$ gorai component list

Installed Components (12):

TYPE      MODEL           SOURCE                                           VERSION
motor     gpio            github.com/gorai/gorai-component-motor           v0.3.0
motor     pwm             github.com/gorai/gorai-component-motor           v0.3.0
sensor    bno085          github.com/robotics-lab/gorai-component-imu      v1.2.3
sensor    fake            github.com/gorai/gorai (builtin)                 v0.3.0
camera    v4l2            github.com/gorai/gorai-driver-camera-v4l2        v0.2.1
...

# Filter by type
$ gorai component list --type sensor

TYPE      MODEL           SOURCE
sensor    bno085          github.com/robotics-lab/gorai-component-imu      v1.2.3
sensor    mpu6050         github.com/gorai/gorai-component-sensor          v0.3.0
sensor    fake            github.com/gorai/gorai (builtin)                 v0.3.0

# JSON output
$ gorai component list --format json
[
  {
    "type": "motor",
    "model": "gpio",
    "repository": "github.com/gorai/gorai-component-motor",
    "version": "v0.3.0",
    "source": "external"
  },
  ...
]
```

**Data Source**:
- Queries the component registry at runtime
- Shows what's actually available in the compiled binary
- Includes builtin components from core repo

---

### `gorai component remove`

Remove a component from the project.

**Syntax**:
```bash
gorai component remove <repository> [flags]
```

**Flags**:
- `--clean-imports` - Remove import statements from main.go

**Examples**:

```bash
$ gorai component remove github.com/robotics-lab/gorai-component-imu

This will remove:
  • github.com/robotics-lab/gorai-component-imu from go.mod
  • Import statements from cmd/robot/main.go (if --clean-imports)

Warning: Components using this package will stop working!

Proceed? [y/N]: y

✓ Removed github.com/robotics-lab/gorai-component-imu
✓ Cleaned imports from cmd/robot/main.go
✓ Ran go mod tidy

# Without confirmation
$ gorai component remove github.com/example/component --clean-imports -y
✓ Removed github.com/example/component
```

---

### `gorai component update`

Update components to latest versions.

**Syntax**:
```bash
gorai component update [repository] [flags]
```

**Flags**:
- `--all` - Update all components
- `--major` - Allow major version updates (breaking changes)
- `--check` - Only check for updates without applying

**Examples**:

```bash
# Update specific component
$ gorai component update github.com/robotics-lab/gorai-component-imu

Current:   v1.2.3
Available: v1.3.0 (minor update - backward compatible)

Changelog:
  • Added support for BMI270 sensor
  • Improved calibration algorithm
  • Bug fixes

Update to v1.3.0? [Y/n]: y

✓ Updated to v1.3.0
✓ Ran go mod tidy

# Check all updates
$ gorai component update --check --all

Updates available:

  github.com/robotics-lab/gorai-component-imu
    v1.2.3 → v1.3.0 (minor)

  github.com/gorai/gorai-component-motor
    v0.3.0 → v0.4.0 (minor)

Run 'gorai component update --all' to update

# Update all (non-breaking)
$ gorai component update --all

✓ Updated 2 components
  • github.com/robotics-lab/gorai-component-imu: v1.2.3 → v1.3.0
  • github.com/gorai/gorai-component-motor: v0.3.0 → v0.4.0

# Allow breaking updates
$ gorai component update --all --major

Available updates:

  github.com/example/component v1.5.0 → v2.0.0 (MAJOR - breaking changes)

⚠ Warning: Major version update may require code changes!

Proceed? [y/N]:
```

---

## Service Commands

Commands for managing container services (vision, SLAM, etc.).

### `gorai service search`

Search for available services.

**Syntax**:
```bash
gorai service search <query> [flags]
```

**Flags**:
- `--type <type>` - Filter by service type (vision, slam, navigation, etc.)
- `--accelerator <accel>` - Filter by accelerator support (cpu, cuda, hailo, etc.)
- `--platform <platform>` - Filter by platform
- `--limit <n>` - Limit results

**Examples**:

```bash
$ gorai service search vision

Found 4 services:

  ghcr.io/gorai/yolox:v1.2.0
    Real-time object detection using YOLOX
    Type: vision | Accelerators: cpu, hailo, cuda
    Latency: 50ms | Throughput: 30 fps
    License: Apache-2.0 | ★ Stable

  ghcr.io/gorai/yolov8:v2.0.0
    YOLOv8 object detection and segmentation
    Type: vision | Accelerators: cpu, cuda, coral
    Latency: 40ms | Throughput: 35 fps
    License: GPL-3.0 | ★ Stable

# Filter by accelerator
$ gorai service search --accelerator hailo

Found 3 services with Hailo support:

  ghcr.io/gorai/yolox:v1.2.0-hailo
  ghcr.io/gorai/efficientdet:v1.0.0-hailo
  ghcr.io/robotics-lab/pose-estimation:v3.1.0-hailo
```

---

### `gorai service info`

Display detailed service information.

**Syntax**:
```bash
gorai service info <image|rdl-url> [flags]
```

**Examples**:

```bash
$ gorai service info ghcr.io/gorai/yolox:v1.2.0

┌────────────────────────────────────────────────────────────────┐
│ YOLOX Object Detection Service                                 │
└────────────────────────────────────────────────────────────────┘

Name:     yolox-detector
Type:     vision
Model:    yolox
Version:  v1.2.0
License:  Apache-2.0

Description:
  Real-time object detection using YOLOX neural network with
  multiple model size variants optimized for edge devices.

Container Images:
  • default: ghcr.io/gorai/yolox:v1.2.0
  • cpu:     ghcr.io/gorai/yolox:v1.2.0-cpu (universal)
  • hailo:   ghcr.io/gorai/yolox:v1.2.0-hailo (26 TOPS)
  • cuda:    ghcr.io/gorai/yolox:v1.2.0-cuda (Jetson)

NATS Topics:
  Subscribes:
    • camera.*.frame (image/jpeg) - Camera frames for detection

  Publishes:
    • vision.{name}.detections (30 Hz) - Detection results

Configuration:
  • confidence_threshold (float)
      Minimum confidence score (default: 0.6, range: 0.0-1.0)

  • model_size (string)
      Model size: tiny|small|medium|large (default: small)

  • class_filter (array, optional)
      Only detect specific COCO classes

Performance:
  Latency:    50ms typical, 100ms max
  Throughput: 30 fps
  Startup:    5s

Resources:
  Memory: 2Gi required, 4Gi limit
  CPU:    1 core required
  GPU:    Optional

Usage Example:
  {
    "services": [
      {
        "name": "detector",
        "rdl": "https://raw.githubusercontent.com/gorai/gorai-service-vision/v1.2.0/yolox/service.rdl.json",
        "attributes": {
          "confidence_threshold": 0.7,
          "model_size": "medium"
        }
      }
    ]
  }

Documentation: https://docs.gorai.dev/services/yolox
Repository:    https://github.com/gorai/gorai-service-vision
```

---

### `gorai service pull`

Pre-pull service container images.

**Syntax**:
```bash
gorai service pull <image> [flags]
```

**Flags**:
- `--platform <platform>` - Pull for specific platform
- `--all-variants` - Pull all image variants

**Examples**:

```bash
$ gorai service pull ghcr.io/gorai/yolox:v1.2.0

Pulling ghcr.io/gorai/yolox:v1.2.0...
✓ Downloaded 543 MB

$ gorai service pull ghcr.io/gorai/yolox:v1.2.0 --all-variants

Pulling all variants:
  ✓ cpu:   ghcr.io/gorai/yolox:v1.2.0-cpu (412 MB)
  ✓ hailo: ghcr.io/gorai/yolox:v1.2.0-hailo (623 MB)
  ✓ cuda:  ghcr.io/gorai/yolox:v1.2.0-cuda (1.2 GB)
```

---

## Validation Commands

Commands for validating components and services.

### `gorai component validate`

Validate a component repository.

**Syntax**:
```bash
gorai component validate [path] [flags]
```

**Arguments**:
- `[path]` - Path to component repo (default: current directory)

**Flags**:
- `--strict` - Enable strict validation (fails on warnings)
- `--fix` - Attempt to auto-fix issues

**Examples**:

```bash
$ gorai component validate .

Validating component at /home/user/gorai-component-imu...

✓ gorai-component.yaml exists
✓ go.mod is valid
✓ Component metadata is valid
✓ All provided models register correctly:
    • sensor/bno085
    • sensor/icm42688
✓ Tests pass (87% coverage)
✓ LICENSE file exists (MIT)
✓ README.md exists
✓ Configuration examples in README match schema

Component is valid!

Quality score: 95/100
  ✓ Excellent test coverage (87%)
  ✓ CI configured
  ✓ Documentation complete
  ⚠ No integration tests found

Suggestions:
  • Add integration tests for hardware validation
  • Consider adding examples/ directory

# Validation failure
$ gorai component validate .

Validating component...

✗ gorai-component.yaml: schema validation failed
    • Missing required field: component.version
    • Invalid pattern for component.name: must be kebab-case

✗ Registration mismatch:
    • gorai-component.yaml declares sensor/bno085
    • But no init() registration found for sensor/bno085

✗ Tests failed:
    • 2 tests failed in ./bno085/

Fix these issues and try again.

# Auto-fix
$ gorai component validate --fix

Attempting to fix issues...
✓ Fixed component.yaml indentation
✓ Added missing LICENSE file
✓ Updated go.mod with correct version

Remaining issues (manual fix required):
✗ Tests still failing
```

---

### `gorai service validate`

Validate a service RDL file.

**Syntax**:
```bash
gorai service validate <rdl-file> [flags]
```

**Examples**:

```bash
$ gorai service validate service.rdl.json

Validating service.rdl.json...

✓ JSON is valid
✓ Schema validation passed
✓ Container image is accessible: ghcr.io/gorai/yolox:v1.2.0
✓ NATS topic patterns are valid
✓ Configuration schema is consistent
✓ All image variants exist:
    • cpu:   ghcr.io/gorai/yolox:v1.2.0-cpu ✓
    • hailo: ghcr.io/gorai/yolox:v1.2.0-hailo ✓
    • cuda:  ghcr.io/gorai/yolox:v1.2.0-cuda ✓

Service RDL is valid!

# Validation failure
$ gorai service validate service.rdl.json

✗ Schema validation failed:
    • service.version: does not match pattern (must start with 'v')
    • service.configuration.threshold: invalid type (expected 'float', got 'string')

✗ Container image not accessible:
    • ghcr.io/gorai/nonexistent:v1.0.0
    • HTTP 404: Not Found

Fix these issues and try again.
```

---

## Configuration

Configure CLI behavior and registry settings.

### Registry Configuration

**File**: `~/.config/gorai/config.yaml`

```yaml
# Component registry settings
registry:
  # Primary registry URL
  url: "https://registry.gorai.dev"

  # Additional registries (searched in order)
  additional:
    - url: "https://registry.acme-corp.com"
      auth:
        token_file: "~/.config/gorai/acme-token"

  # Cache settings
  cache:
    enabled: true
    ttl: 24h
    directory: "~/.cache/gorai/registry"

# Container registry settings
containers:
  # Default pull policy
  pull_policy: "IfNotPresent"

  # Private registries
  registries:
    - url: "registry.acme-corp.com"
      username: "user"
      password_file: "~/.config/gorai/registry-password"

# CLI behavior
cli:
  # Auto-import when adding components
  auto_import: false

  # Confirm destructive actions
  confirm_destructive: true

  # Output format
  default_format: "table"
```

### Environment Variables

```bash
# Override registry URL
export GORAI_REGISTRY_URL="https://registry.acme-corp.com"

# Disable registry caching
export GORAI_REGISTRY_CACHE_DISABLED=true

# Set authentication token
export GORAI_REGISTRY_TOKEN="ghp_xxxxxxxxxxxx"

# Container registry auth
export GORAI_CONTAINER_REGISTRY_USER="admin"
export GORAI_CONTAINER_REGISTRY_PASSWORD="secret"
```

---

## Implementation Details

### Component Search Implementation

1. **Query Registry API**:
   ```
   GET https://registry.gorai.dev/api/v1/components/search?q=imu&type=sensor&limit=20
   ```

2. **Fallback to GitHub**:
   If registry unavailable, use GitHub API:
   ```
   GET https://api.github.com/search/repositories?q=gorai-component+imu
   ```

3. **Local Cache**:
   Cache results in `~/.cache/gorai/search/` for 1 hour

4. **Ranking Algorithm**:
   - Keyword relevance (fuzzy matching)
   - Download count
   - Last update recency
   - Star count
   - Quality score (test coverage, CI status)

### Component Add Implementation

```go
// Pseudocode
func ComponentAdd(repo string, version string) error {
    // 1. Fetch component metadata
    metadata := fetchMetadata(repo, version)

    // 2. Validate compatibility
    if !isCompatible(metadata.Compatibility.GoraiVersion) {
        return errors.New("incompatible Gorai version")
    }

    // 3. Run go get
    cmd := exec.Command("go", "get", fmt.Sprintf("%s@%s", repo, version))
    if err := cmd.Run(); err != nil {
        return err
    }

    // 4. Optionally add import
    if autoImport {
        addImportToMainGo(repo, metadata.Provides)
    }

    // 5. Show next steps
    printNextSteps(metadata)

    return nil
}
```

### Registry API Specification

**Endpoints**:

```
GET  /api/v1/components/search?q=<query>&type=<type>&limit=<n>
GET  /api/v1/components/<repo>/<version>/metadata
GET  /api/v1/services/search?q=<query>&type=<type>
GET  /api/v1/services/<image>/rdl
POST /api/v1/components/<repo>/register  (for submitting components)
```

**Response Format**:

```json
{
  "results": [
    {
      "repository": "github.com/robotics-lab/gorai-component-imu",
      "version": "v1.2.3",
      "name": "advanced-imu",
      "description": "High-precision IMU drivers",
      "provides": [
        {"type": "sensor", "model": "bno085"},
        {"type": "sensor", "model": "icm42688"}
      ],
      "license": "MIT",
      "maturity": "stable",
      "last_updated": "2026-01-10T12:00:00Z",
      "downloads": 1523,
      "stars": 42
    }
  ],
  "total": 5,
  "page": 1
}
```

---

## Future Enhancements

### v2.0 Features

- **Interactive mode**: `gorai component search --interactive` (TUI picker)
- **Component templates**: `gorai component new --from-template advanced-sensor`
- **Dependency graph**: `gorai component deps --graph`
- **Security scanning**: `gorai component audit` (check for CVEs)
- **Performance benchmarks**: `gorai component benchmark <repo>`
- **AI-powered search**: Natural language queries
- **Component marketplace**: Browse/purchase commercial components

---

## See Also

- [specs/gorai-component-schema.yaml](gorai-component-schema.yaml) - Component metadata schema
- [specs/service-rdl-schema.json](service-rdl-schema.json) - Service RDL schema
- [docs/third-party-component-ecosystem.md](../docs/third-party-component-ecosystem.md) - Developer guide
- [docs/modules-approach.md](../docs/modules-approach.md) - Architecture overview
