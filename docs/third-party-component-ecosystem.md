# Third-Party Component Ecosystem

**Version**: 1.0
**Date**: 2026-01-16
**Status**: Proposed

---

## Executive Summary

This document provides a comprehensive guide for third-party developers who want to create Gorai components and services **outside the main Gorai repository**. It covers:

- How to structure a third-party components/service repository
- Metadata standards for self-description and discovery
- Testing and validation requirements
- Distribution strategies (public vs private)
- User consumption workflow
- Best practices and quality guidelines

**Goal**: Enable a thriving ecosystem where anyone can create, distribute, and monetize Gorai-compatible components without needing to contribute to the core repository.

---

## Table of Contents

1. [Why Third-Party Components?](#why-third-party-components)
2. [Component Types](#component-types)
3. [Creating a Go Component](#creating-a-go-component)
4. [Creating a Container Service](#creating-a-container-service)
5. [Metadata Standards](#metadata-standards)
6. [Testing & Validation](#testing--validation)
7. [Distribution Strategies](#distribution-strategies)
8. [User Consumption Workflow](#user-consumption-workflow)
9. [Best Practices](#best-practices)
10. [Examples](#examples)

---

## Why Third-Party Components?

### Use Cases

**Commercial Hardware Vendors**:
- Distribute proprietary drivers for custom sensors, motors, or actuators
- Support customers without open-sourcing IP
- Version control separate from Gorai core

**Research Labs & Universities**:
- Share experimental algorithms and components
- Publish alongside research papers
- Maintain independent development timelines

**Internal Corporate Tools**:
- Build company-specific components (e.g., warehouse automation)
- Keep proprietary logic private
- Integrate with internal systems

**Open Source Contributors**:
- Develop niche components without core repo approval
- Experiment with new ideas
- Maintain at your own pace

---

## Component Types

### 1. Go Components (In-Process)

**Best For**:
- Hardware drivers (GPIO, I2C, SPI, CAN)
- Low-latency components (motor controllers, encoders)
- Core robot logic that needs direct memory access
- Components written in Go

**Distribution**: Go modules via git repositories

**Examples**:
- Custom motor driver for proprietary hardware
- Specialized sensor (LiDAR, IMU, GPS)
- Custom actuator or gripper controller

### 2. Container Services (External Process)

**Best For**:
- ML/AI models (object detection, SLAM, path planning)
- Non-Go languages (Python, C++, Rust)
- Heavy computation that benefits from isolation
- Services with large dependencies (TensorFlow, PyTorch)

**Distribution**: Container images via registries

**Examples**:
- Custom vision model trained on proprietary data
- SLAM algorithm with C++ dependencies
- Navigation service with path planning

---

## Creating a Go Component

### Step 1: Scaffold the Repository

Use the Gorai CLI to generate a template:

```bash
gorai new component \
  --type sensor \
  --model custom-lidar \
  --repo github.com/acme-corp/gorai-component-lidar

cd gorai-component-lidar
```

This generates:

```
gorai-component-lidar/
├── gorai-component.yaml       # Metadata file
├── go.mod                      # Go module definition
├── go.sum
├── README.md                   # Usage documentation
├── LICENSE                     # License file
├── .github/
│   └── workflows/
│       └── ci.yml              # CI/CD pipeline
├── custom-lidar/
│   ├── lidar.go                # Component implementation
│   └── lidar_test.go           # Unit tests
└── examples/
    └── basic/
        └── main.go             # Example usage
```

### Step 2: Implement the Component

Your component must implement the `resource.Resource` interface:

```go
// custom-lidar/lidar.go
package customlidar

import (
    "context"
    "fmt"

    "github.com/gorai/gorai/components/sensor"
    "github.com/gorai/gorai/pkg/resource"
    "github.com/gorai/gorai/pkg/registry"
)

// Register this component with Gorai
func init() {
    registry.RegisterComponent("sensor", "custom-lidar", NewCustomLidar)
}

type CustomLidar struct {
    resource.Named
    resource.TriviallyCloseable

    // Your hardware-specific fields
    port      string
    baudRate  int
    scanRange float64
}

// Config defines the JSON attributes from robot.json
type Config struct {
    Port      string  `json:"port"`
    BaudRate  int     `json:"baud_rate"`
    ScanRange float64 `json:"scan_range"`
}

// NewCustomLidar is the factory function
func NewCustomLidar(
    ctx context.Context,
    deps resource.Dependencies,
    conf resource.Config,
) (resource.Resource, error) {
    // Parse configuration
    var config Config
    if err := conf.DecodeAttributes(&config); err != nil {
        return nil, fmt.Errorf("failed to decode config: %w", err)
    }

    // Validate configuration
    if config.Port == "" {
        return nil, fmt.Errorf("port is required")
    }

    // Create component
    lidar := &CustomLidar{
        Named:     conf.ResourceName().AsNamed(),
        port:      config.Port,
        baudRate:  config.BaudRate,
        scanRange: config.ScanRange,
    }

    // Initialize hardware connection
    if err := lidar.connect(ctx); err != nil {
        return nil, fmt.Errorf("failed to connect: %w", err)
    }

    return lidar, nil
}

// Implement sensor.Sensor interface methods
func (l *CustomLidar) Readings(ctx context.Context, extra map[string]interface{}) (map[string]interface{}, error) {
    // Read from hardware
    distance, angle := l.scan(ctx)

    return map[string]interface{}{
        "distance_mm": distance,
        "angle_deg":   angle,
    }, nil
}

// Private hardware-specific methods
func (l *CustomLidar) connect(ctx context.Context) error {
    // Open serial port, initialize hardware, etc.
    return nil
}

func (l *CustomLidar) scan(ctx context.Context) (float64, float64) {
    // Actual hardware scanning logic
    return 0.0, 0.0
}

// DoCommand allows custom commands
func (l *CustomLidar) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
    return nil, fmt.Errorf("command not supported")
}
```

### Step 3: Fill Out Metadata

Edit `gorai-component.yaml`:

```yaml
schema_version: "1.0"
component:
  name: "custom-lidar"
  repository: "github.com/acme-corp/gorai-component-lidar"
  version: "v1.0.0"

  provides:
    - type: "sensor"
      model: "custom-lidar"

  compatibility:
    gorai_version: ">=0.3.0, <1.0.0"
    platforms: ["linux/arm64", "linux/amd64"]
    go_version: ">=1.22"

  author: "ACME Corporation"
  license: "Proprietary"
  description: "High-precision 360° LiDAR sensor driver with real-time scanning"
  homepage: "https://github.com/acme-corp/gorai-component-lidar"
  documentation: "https://docs.acme-corp.com/gorai-lidar"

  hardware_requirements:
    uart: true
    gpio: false
    i2c: false
    spi: false

  configuration:
    port:
      type: "string"
      required: true
      description: "Serial port path (e.g., /dev/ttyUSB0)"
      example: "/dev/ttyUSB0"

    baud_rate:
      type: "int"
      default: 115200
      description: "UART baud rate"

    scan_range:
      type: "float"
      default: 360.0
      description: "Scanning range in degrees"
      range: [0, 360]
```

### Step 4: Add Tests

```go
// custom-lidar/lidar_test.go
package customlidar

import (
    "context"
    "testing"

    "github.com/gorai/gorai/pkg/resource"
    "github.com/stretchr/testify/assert"
)

func TestNewCustomLidar(t *testing.T) {
    ctx := context.Background()

    conf := resource.Config{
        Attributes: map[string]interface{}{
            "port":       "/dev/ttyUSB0",
            "baud_rate":  115200,
            "scan_range": 360.0,
        },
    }

    lidar, err := NewCustomLidar(ctx, nil, conf)
    assert.NoError(t, err)
    assert.NotNil(t, lidar)
}

func TestReadings(t *testing.T) {
    // Test reading sensor data
    // Add hardware simulation or mocking
}
```

### Step 5: Validate

```bash
# Run tests
go test ./...

# Validate metadata
gorai component validate .

# Expected output:
# ✓ gorai-component.yaml is valid
# ✓ Component registers correctly
# ✓ Tests pass
# ✓ go.mod is properly configured
# ✓ README contains usage instructions
```

### Step 6: Publish

```bash
# Tag a release
git tag v1.0.0
git push origin v1.0.0

# GitHub releases are automatically created (if CI configured)
```

---

## Creating a Container Service

### Step 1: Scaffold the Repository

```bash
gorai new service \
  --type vision \
  --model custom-detector \
  --language python \
  --repo github.com/acme-corp/gorai-service-detector

cd gorai-service-detector
```

This generates:

```
gorai-service-detector/
├── service.rdl.json            # Service metadata & RDL definition
├── Containerfile               # Container build definition
├── requirements.txt            # Python dependencies
├── README.md
├── .github/
│   └── workflows/
│       └── build-push.yml      # Container build CI
├── src/
│   ├── main.py                 # NATS subscriber entrypoint
│   ├── detector.py             # Detection logic
│   └── model_loader.py         # Model initialization
├── models/
│   └── .gitkeep                # Model weights (or download script)
└── tests/
    └── test_detector.py
```

### Step 2: Implement the Service

The service communicates via NATS:

```python
# src/main.py
import os
import asyncio
import json
from nats.aio.client import Client as NATS
from detector import CustomDetector

async def main():
    # Load configuration from environment
    nats_url = os.getenv("NATS_URL", "nats://localhost:4222")
    service_name = os.getenv("SERVICE_NAME", "detector")
    confidence_threshold = float(os.getenv("CONFIDENCE_THRESHOLD", "0.6"))

    # Initialize NATS
    nc = NATS()
    await nc.connect(nats_url)

    # Initialize detector
    detector = CustomDetector(
        model_path="models/custom_model.onnx",
        confidence_threshold=confidence_threshold
    )

    # Subscribe to camera frames
    async def message_handler(msg):
        try:
            # Parse incoming frame
            frame_data = json.loads(msg.data)
            image_bytes = frame_data["image"]

            # Run detection
            detections = detector.detect(image_bytes)

            # Publish results
            result = {
                "timestamp": frame_data["timestamp"],
                "detections": detections
            }
            await nc.publish(f"vision.{service_name}.detections", json.dumps(result).encode())
        except Exception as e:
            print(f"Error processing frame: {e}")

    # Subscribe to input topic
    await nc.subscribe(f"camera.*.frame", cb=message_handler)

    print(f"Service {service_name} started")

    # Keep running
    await asyncio.Event().wait()

if __name__ == "__main__":
    asyncio.run(main())
```

```python
# src/detector.py
import numpy as np
import onnxruntime as ort

class CustomDetector:
    def __init__(self, model_path: str, confidence_threshold: float):
        self.session = ort.InferenceSession(model_path)
        self.confidence_threshold = confidence_threshold

    def detect(self, image_bytes: bytes) -> list:
        # Preprocess image
        image = self._preprocess(image_bytes)

        # Run inference
        outputs = self.session.run(None, {"input": image})

        # Post-process
        detections = self._postprocess(outputs)

        return detections

    def _preprocess(self, image_bytes: bytes):
        # Convert bytes to numpy array, resize, normalize
        return np.zeros((1, 3, 640, 640))  # Example

    def _postprocess(self, outputs):
        # Parse model outputs, filter by confidence
        return [
            {
                "class": "person",
                "confidence": 0.92,
                "bbox": [100, 100, 200, 300]
            }
        ]
```

### Step 3: Create Containerfile

```dockerfile
# Containerfile
FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy source code
COPY src/ /app/src/

# Copy or download model weights
COPY models/ /app/models/
# Alternative: Download on startup
# RUN wget https://example.com/model.onnx -O /app/models/custom_model.onnx

# Expose port for health checks (optional)
EXPOSE 8080

# Run service
CMD ["python", "src/main.py"]
```

### Step 4: Fill Out Service RDL

Edit `service.rdl.json`:

```json
{
  "schema_version": "1.0",
  "service": {
    "name": "custom-detector",
    "type": "vision",
    "model": "custom-detector",
    "version": "v1.0.0",
    "repository": "github.com/acme-corp/gorai-service-detector",

    "description": "Custom object detection model trained on proprietary dataset",
    "author": "ACME Corporation",
    "license": "Proprietary",
    "homepage": "https://github.com/acme-corp/gorai-service-detector",

    "compatibility": {
      "gorai_version": ">=0.3.0",
      "platforms": ["linux/arm64", "linux/amd64"],
      "accelerators": ["cpu", "hailo"]
    },

    "container": {
      "default_image": "ghcr.io/acme-corp/custom-detector:v1.0.0",
      "resource_requirements": {
        "memory": "2Gi",
        "cpu": "1",
        "gpu": "optional"
      },
      "health_check": {
        "type": "http",
        "endpoint": "http://localhost:8080/health",
        "interval": "30s"
      }
    },

    "nats_topics": {
      "subscribes": [
        {
          "pattern": "camera.*.frame",
          "description": "Incoming camera frames for detection"
        }
      ],
      "publishes": [
        {
          "pattern": "vision.{name}.detections",
          "description": "Detection results with bounding boxes"
        }
      ]
    },

    "configuration": {
      "confidence_threshold": {
        "type": "float",
        "default": 0.6,
        "range": [0.0, 1.0],
        "description": "Minimum confidence score for detections",
        "env_var": "CONFIDENCE_THRESHOLD"
      },
      "model_variant": {
        "type": "string",
        "enum": ["small", "medium", "large"],
        "default": "medium",
        "description": "Model size variant (affects speed/accuracy tradeoff)",
        "env_var": "MODEL_VARIANT"
      }
    }
  }
}
```

### Step 5: Build & Test

```bash
# Build container locally
podman build -t custom-detector:local .

# Test locally with NATS
podman run --rm \
  -e NATS_URL=nats://host.docker.internal:4222 \
  -e SERVICE_NAME=test_detector \
  -e CONFIDENCE_THRESHOLD=0.7 \
  custom-detector:local
```

### Step 6: Publish

Configure CI/CD to build and push containers:

```yaml
# .github/workflows/build-push.yml
name: Build and Push Container

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract version
        id: version
        run: echo "VERSION=${GITHUB_REF#refs/tags/}" >> $GITHUB_OUTPUT

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          platforms: linux/amd64,linux/arm64
          push: true
          tags: |
            ghcr.io/acme-corp/custom-detector:${{ steps.version.outputs.VERSION }}
            ghcr.io/acme-corp/custom-detector:latest
```

---

## Metadata Standards

### gorai-component.yaml Schema

See [specs/gorai-component-schema.yaml](../specs/gorai-component-schema.yaml) for the full JSON Schema definition.

**Required Fields**:
- `schema_version`: Metadata format version (currently "1.0")
- `component.name`: Component name (kebab-case)
- `component.repository`: Git repository URL
- `component.version`: Semantic version (e.g., "v1.2.3")
- `component.provides`: List of `{type, model}` pairs this component registers
- `component.compatibility.gorai_version`: Gorai version constraint

**Recommended Fields**:
- `component.author`: Author name or organization
- `component.license`: License identifier (SPDX format)
- `component.description`: One-line description
- `component.homepage`: Repository or product page URL
- `component.configuration`: Configuration attribute schemas

### service.rdl.json Schema

See [specs/service-rdl-schema.json](../specs/service-rdl-schema.json) for the full JSON Schema definition.

**Required Fields**:
- `schema_version`: RDL format version
- `service.name`: Service name
- `service.type`: Service type (vision, slam, navigation, etc.)
- `service.model`: Model identifier
- `service.version`: Semantic version
- `service.container.default_image`: Default container image reference
- `service.nats_topics`: Topics subscribed/published

**Recommended Fields**:
- `service.configuration`: Configuration attribute schemas
- `service.container.resource_requirements`: Memory/CPU requirements
- `service.compatibility`: Platform and accelerator support

---

## Testing & Validation

### For Go Components

```bash
# Unit tests
go test ./... -v -race -coverprofile=coverage.out

# Check coverage
go tool cover -html=coverage.out

# Validate metadata
gorai component validate .

# Integration test (requires running robot)
gorai validate examples/basic/robot.json
```

### For Container Services

```bash
# Build container
podman build -t myservice:test .

# Run integration tests
podman run --rm \
  -e NATS_URL=nats://localhost:4222 \
  myservice:test pytest tests/

# Validate service RDL
gorai service validate service.rdl.json

# Test end-to-end with robot
gorai run examples/robot.json
```

### Quality Checklist

- [ ] All tests pass with >80% coverage
- [ ] Metadata file validates successfully
- [ ] README includes usage examples
- [ ] LICENSE file included
- [ ] CI/CD configured (tests + releases)
- [ ] Documentation covers configuration options
- [ ] Examples demonstrate common use cases
- [ ] Semantic versioning used for releases

---

## Distribution Strategies

### Public Open Source

**Hosting**: GitHub, GitLab, etc.

**Container Registry**: ghcr.io, Docker Hub, quay.io

**Discovery**: Submit to registry.gorai.dev

**Example**:
```bash
# Go component
go get github.com/robotics-lab/gorai-component-advanced-imu

# Container service
# Users reference: ghcr.io/robotics-lab/advanced-slam:v1.0.0
```

### Private Proprietary

**Hosting**: Private git server, GitHub Enterprise, GitLab self-hosted

**Container Registry**: Private registry (Harbor, Artifactory, cloud provider)

**Authentication**:
```bash
# Git authentication
export GOPRIVATE=github.com/acme-corp/*
git config --global url."https://${TOKEN}@github.com/".insteadOf "https://github.com/"

# Container registry authentication
podman login registry.acme-corp.com
```

### Commercial Distribution

**Licensing**: Include license server checks in component

**Example**:
```go
func NewLicensedComponent(ctx context.Context, deps resource.Dependencies, conf resource.Config) (resource.Resource, error) {
    // Validate license key
    licenseKey := conf.Attributes["license_key"]
    if !validateLicense(licenseKey) {
        return nil, fmt.Errorf("invalid license key")
    }
    // ...
}
```

**Support**: Provide private support channels for paying customers

---

## User Consumption Workflow

### For Go Components

**Discovery**:
```bash
gorai component search lidar
# Shows: github.com/acme-corp/gorai-component-lidar v1.0.0
```

**Installation**:
```bash
gorai component add github.com/acme-corp/gorai-component-lidar@v1.0.0
# Adds to go.mod and prints import instructions
```

**Import** (in robot binary):
```go
import _ "github.com/acme-corp/gorai-component-lidar/custom-lidar"
```

**Configure** (in robot.json):
```json
{
  "components": [
    {
      "name": "front_lidar",
      "type": "sensor",
      "model": "custom-lidar",
      "attributes": {
        "port": "/dev/ttyUSB0",
        "baud_rate": 115200,
        "scan_range": 360.0
      }
    }
  ]
}
```

### For Container Services

**Discovery**:
```bash
gorai service search detector
# Shows available vision services
```

**Configuration** (in robot.json):
```json
{
  "services": [
    {
      "name": "detector",
      "rdl": "https://raw.githubusercontent.com/acme-corp/gorai-service-detector/v1.0.0/service.rdl.json",
      "attributes": {
        "confidence_threshold": 0.7,
        "model_variant": "medium"
      }
    }
  ]
}
```

**Deployment**:
```bash
gorai deploy robot.json
# Automatically pulls container and starts service
```

---

## Best Practices

### Naming Conventions

**Go Components**:
- Repository: `gorai-component-<category>` (e.g., `gorai-component-motor`)
- Package: lowercase, no underscores (e.g., `package customlidar`)
- Model name: kebab-case (e.g., `"custom-lidar"`)

**Container Services**:
- Repository: `gorai-service-<name>` (e.g., `gorai-service-vision`)
- Container image: lowercase, dash-separated (e.g., `custom-detector`)

### Versioning

Use semantic versioning:
- **v1.0.0**: Initial stable release
- **v1.1.0**: Add new features (backward compatible)
- **v1.0.1**: Bug fixes
- **v2.0.0**: Breaking changes (incompatible with v1.x)

### Configuration

**Use typed configuration structs**:
```go
type Config struct {
    Port     string  `json:"port"`
    BaudRate int     `json:"baud_rate"`
    Timeout  int     `json:"timeout,omitempty"`  // Optional
}
```

**Provide sensible defaults**:
```go
if config.Timeout == 0 {
    config.Timeout = 5000  // 5 seconds
}
```

**Validate configuration**:
```go
if config.Port == "" {
    return nil, fmt.Errorf("port is required")
}
```

### Error Handling

**Return descriptive errors**:
```go
return nil, fmt.Errorf("failed to connect to %s: %w", config.Port, err)
```

**Use error wrapping**:
```go
if err != nil {
    return fmt.Errorf("hardware initialization failed: %w", err)
}
```

### Testing

**Write unit tests**:
```go
func TestComponentCreation(t *testing.T) {
    // Test successful creation
    // Test error cases
    // Test configuration validation
}
```

**Add integration tests** (optional):
```go
//go:build integration
func TestHardwareIntegration(t *testing.T) {
    // Test with real hardware
}
```

### Documentation

**README.md must include**:
- What the component does
- Hardware requirements
- Installation instructions
- Configuration examples
- Troubleshooting tips

**Example**:
```markdown
# Custom LiDAR Driver

High-precision 360° LiDAR driver for ACME LiDAR v3.

## Installation

```bash
gorai component add github.com/acme-corp/gorai-component-lidar
```

## Configuration

```json
{
  "name": "lidar",
  "type": "sensor",
  "model": "custom-lidar",
  "attributes": {
    "port": "/dev/ttyUSB0"
  }
}
```

## Hardware Requirements

- UART/Serial port
- Baud rate: 115200
- Power: 5V, 500mA
```

---

## Examples

### Example 1: Third-Party Motor Driver

Repository: `github.com/robotics-supply/gorai-component-motor`

Provides:
- `motor/rs485` - RS485 motor controller
- `motor/ethercat` - EtherCAT motor controller

Users import:
```go
import _ "github.com/robotics-supply/gorai-component-motor/rs485"
```

Configure:
```json
{
  "components": [
    {
      "name": "arm_motor",
      "type": "motor",
      "model": "rs485",
      "attributes": {
        "port": "/dev/ttyUSB0",
        "device_id": 1
      }
    }
  ]
}
```

### Example 2: Custom Vision Service

Repository: `github.com/ai-lab/gorai-service-crop-detection`

Container: `ghcr.io/ai-lab/crop-detection:v2.1.0`

Service RDL: Defines configuration for crop types, confidence thresholds

Users configure:
```json
{
  "services": [
    {
      "name": "crop_detector",
      "rdl": "https://raw.githubusercontent.com/ai-lab/gorai-service-crop-detection/v2.1.0/service.rdl.json",
      "attributes": {
        "crop_types": ["wheat", "corn", "soybeans"],
        "confidence": 0.85
      }
    }
  ]
}
```

### Example 3: Private Enterprise Component

Repository: `github.com/megacorp-internal/gorai-warehouse-automation` (private)

Authentication:
```bash
export GOPRIVATE=github.com/megacorp-internal/*
```

Users (with access) import:
```go
import _ "github.com/megacorp-internal/gorai-warehouse-automation/forklift"
```

---

## FAQ

### Q: Can I charge for my components?

**A**: Yes! You control the license. Options:
- Sell license keys that your component validates
- Distribute via private repository with paid access
- Offer free basic version, paid premium features

### Q: How do I handle breaking changes?

**A**: Increment major version:
- `v1.x.x` → `v2.0.0`
- Update `gorai-component.yaml` compatibility
- Document migration guide in release notes

### Q: Can I use proprietary dependencies?

**A**: Yes for Go components (Go modules can be private). For container services, bundle dependencies in the container image.

### Q: How do I update my component?

**A**:
1. Make changes
2. Update version in metadata
3. Tag release: `git tag v1.1.0 && git push origin v1.1.0`
4. Users update: `go get -u github.com/you/your-component@v1.1.0`

### Q: What if my component needs custom hardware?

**A**: Document in `hardware_requirements` and README. Users are responsible for hardware setup.

### Q: Can I bundle firmware with my component?

**A**: Yes, use `//go:embed` to embed firmware files:
```go
//go:embed firmware/device.bin
var firmwareBytes []byte
```

---

## Conclusion

Third-party components are essential for a thriving Gorai ecosystem. By following these guidelines, you can:

- Create high-quality components that users trust
- Distribute publicly or privately as needed
- Maintain components independently from Gorai core
- Monetize your work if desired
- Contribute to the robotics community

**Next Steps**:
1. Use `gorai new component` or `gorai new service` to scaffold
2. Implement your component following best practices
3. Validate with `gorai component validate`
4. Publish to git repository
5. Optional: Submit to registry.gorai.dev for discovery

Questions? Open an issue at https://github.com/gorai/gorai/issues
