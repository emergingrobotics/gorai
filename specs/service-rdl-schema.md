# Gorai Service RDL Schema

**Version**: 1.0
**Date**: 2026-01-16
**Status**: Proposed

---

## Overview

This document defines the enhanced schema for `service.rdl.json`, the Robot Definition Language file that third-party container service developers include in their repositories. This extends the base RDL with metadata for:

- Self-description of service capabilities
- Discovery and installation
- Container deployment configuration
- NATS topic documentation
- Configuration schema
- Performance characteristics

**File Name**: `service.rdl.json` (typically at repository root)

**Format**: JSON (validated against JSON Schema at [service-rdl-schema.json](service-rdl-schema.json))

---

## Purpose

The service RDL serves dual purposes:

1. **Distribution Metadata**: Describes the service for discovery, installation, and validation
2. **Deployment Template**: Can be referenced in user's `robot.json` to include this service

Users can reference the RDL file directly:

```json
{
  "services": [
    {
      "name": "my_detector",
      "rdl": "https://raw.githubusercontent.com/gorai/gorai-service-vision/v1.0.0/yolox/service.rdl.json",
      "attributes": {
        "confidence_threshold": 0.7
      }
    }
  ]
}
```

---

## Complete Example

```json
{
  "schema_version": "1.0",
  "service": {
    "name": "yolox-detector",
    "type": "vision",
    "model": "yolox",
    "version": "v1.2.0",
    "repository": "github.com/gorai/gorai-service-vision",

    "description": "Real-time object detection using YOLOX neural network with multiple model size variants",
    "author": "Gorai Project",
    "license": "Apache-2.0",
    "homepage": "https://github.com/gorai/gorai-service-vision",
    "documentation": "https://docs.gorai.dev/services/yolox",

    "compatibility": {
      "gorai_version": ">=0.3.0",
      "platforms": ["linux/arm64", "linux/amd64"],
      "accelerators": ["cpu", "hailo", "cuda"]
    },

    "container": {
      "default_image": "ghcr.io/gorai/yolox:v1.2.0",
      "image_variants": {
        "cpu": {
          "image": "ghcr.io/gorai/yolox:v1.2.0-cpu",
          "description": "CPU-only inference (slower but universal)",
          "accelerator": "cpu"
        },
        "hailo": {
          "image": "ghcr.io/gorai/yolox:v1.2.0-hailo",
          "description": "Optimized for Hailo AI accelerator (26 TOPS)",
          "accelerator": "hailo"
        },
        "cuda": {
          "image": "ghcr.io/gorai/yolox:v1.2.0-cuda",
          "description": "NVIDIA CUDA acceleration for Jetson",
          "accelerator": "cuda"
        }
      },
      "pull_policy": "IfNotPresent",
      "resource_requirements": {
        "memory": "2Gi",
        "cpu": "1",
        "gpu": "optional"
      },
      "resource_limits": {
        "memory": "4Gi",
        "cpu": "2"
      },
      "health_check": {
        "type": "http",
        "endpoint": "http://localhost:8080/health",
        "interval": "30s",
        "timeout": "5s",
        "retries": 3
      },
      "volumes": [
        {
          "name": "model-cache",
          "mount_path": "/models",
          "description": "Model weights cache (optional, speeds up startup)",
          "required": false
        }
      ],
      "environment": {
        "LOG_LEVEL": "INFO",
        "MODEL_PATH": "/models/yolox_s.onnx"
      }
    },

    "nats_topics": {
      "subscribes": [
        {
          "pattern": "camera.*.frame",
          "description": "Camera frames for object detection (JPEG or PNG encoded)",
          "message_type": "image/jpeg",
          "qos": "at-most-once"
        }
      ],
      "publishes": [
        {
          "pattern": "vision.{name}.detections",
          "description": "Detection results with bounding boxes and confidence scores",
          "message_type": "application/json",
          "frequency": "30 Hz"
        }
      ]
    },

    "configuration": {
      "confidence_threshold": {
        "type": "float",
        "default": 0.6,
        "range": [0.0, 1.0],
        "description": "Minimum confidence score for detections (0.0-1.0)",
        "env_var": "CONFIDENCE_THRESHOLD"
      },
      "model_size": {
        "type": "string",
        "enum": ["tiny", "small", "medium", "large"],
        "default": "small",
        "description": "YOLOX model size (tiny=fastest, large=most accurate)",
        "env_var": "MODEL_SIZE"
      },
      "nms_threshold": {
        "type": "float",
        "default": 0.45,
        "range": [0.0, 1.0],
        "description": "Non-maximum suppression threshold",
        "env_var": "NMS_THRESHOLD"
      },
      "class_filter": {
        "type": "array",
        "required": false,
        "description": "Only detect these COCO classes (empty = all classes)",
        "example": ["person", "car", "bicycle"],
        "env_var": "CLASS_FILTER"
      }
    },

    "performance": {
      "latency": {
        "typical": "50ms",
        "max": "100ms"
      },
      "throughput": "30 fps",
      "startup_time": "5s"
    },

    "keywords": [
      "vision",
      "object-detection",
      "yolox",
      "deep-learning",
      "real-time"
    ],

    "examples": [
      {
        "name": "basic-detection",
        "description": "Basic object detection with default settings",
        "config": {
          "name": "detector",
          "rdl": "https://raw.githubusercontent.com/gorai/gorai-service-vision/v1.2.0/yolox/service.rdl.json",
          "attributes": {
            "confidence_threshold": 0.6,
            "model_size": "small"
          }
        }
      },
      {
        "name": "person-only",
        "description": "Detect only people with high confidence",
        "config": {
          "name": "person_detector",
          "rdl": "https://raw.githubusercontent.com/gorai/gorai-service-vision/v1.2.0/yolox/service.rdl.json",
          "attributes": {
            "confidence_threshold": 0.8,
            "model_size": "medium",
            "class_filter": ["person"]
          }
        }
      }
    ],

    "quality": {
      "ci_status": "https://github.com/gorai/gorai-service-vision/actions",
      "test_coverage": 82,
      "has_benchmarks": true
    },

    "metadata": {
      "first_released": "2025-08-15",
      "last_updated": "2026-01-10",
      "maturity": "stable"
    }
  }
}
```

---

## Key Sections Explained

### Container Configuration

#### Image Variants

Provide different container images for different platforms/accelerators:

```json
"image_variants": {
  "cpu": {
    "image": "ghcr.io/gorai/yolox:v1.2.0-cpu",
    "description": "CPU-only (universal)",
    "accelerator": "cpu"
  },
  "hailo": {
    "image": "ghcr.io/gorai/yolox:v1.2.0-hailo",
    "description": "Hailo accelerator optimized",
    "accelerator": "hailo"
  }
}
```

Users can select a variant in their `robot.json`:

```json
{
  "services": [
    {
      "name": "detector",
      "rdl": "https://.../service.rdl.json",
      "external": {
        "container": {
          "image_variant": "hailo"
        }
      }
    }
  ]
}
```

#### Resource Requirements

Specify minimum and maximum resources:

```json
"resource_requirements": {
  "memory": "2Gi",    // Minimum 2GB RAM
  "cpu": "1",         // 1 CPU core
  "gpu": "optional"   // GPU nice-to-have but not required
},
"resource_limits": {
  "memory": "4Gi",    // Don't use more than 4GB
  "cpu": "2"          // Max 2 cores
}
```

#### Health Checks

Define how Gorai should check if the service is healthy:

```json
"health_check": {
  "type": "http",                          // HTTP, TCP, NATS, or exec
  "endpoint": "http://localhost:8080/health",
  "interval": "30s",                       // Check every 30 seconds
  "timeout": "5s",                         // 5 second timeout
  "retries": 3                             // 3 failures = unhealthy
}
```

### NATS Topics

Document what topics the service uses:

```json
"nats_topics": {
  "subscribes": [
    {
      "pattern": "camera.*.frame",
      "description": "Camera frames for detection",
      "message_type": "image/jpeg",
      "qos": "at-most-once"
    }
  ],
  "publishes": [
    {
      "pattern": "vision.{name}.detections",
      "description": "Detection results",
      "message_type": "application/json",
      "frequency": "30 Hz"
    }
  ]
}
```

The `{name}` placeholder is replaced with the service instance name from `robot.json`.

### Configuration Schema

Define all configuration attributes:

```json
"configuration": {
  "confidence_threshold": {
    "type": "float",
    "default": 0.6,
    "range": [0.0, 1.0],
    "description": "Minimum confidence for detections",
    "env_var": "CONFIDENCE_THRESHOLD"  // Passed as env var to container
  }
}
```

**Important**: The `env_var` field tells Gorai what environment variable name to use when passing this value to the container.

### Performance Characteristics

Help users understand expected performance:

```json
"performance": {
  "latency": {
    "typical": "50ms",  // Typical processing time
    "max": "100ms"      // Worst-case latency
  },
  "throughput": "30 fps",      // How many inputs/second
  "startup_time": "5s"         // How long until ready
}
```

### Examples

Provide ready-to-use configuration examples:

```json
"examples": [
  {
    "name": "basic-detection",
    "description": "Simple object detection",
    "config": {
      "name": "detector",
      "rdl": "https://.../service.rdl.json",
      "attributes": {
        "confidence_threshold": 0.6
      }
    }
  }
]
```

Users can copy these directly into their `robot.json`.

---

## Usage in robot.json

Users reference the service RDL in two ways:

### Method 1: RDL Reference (Recommended)

```json
{
  "services": [
    {
      "name": "my_detector",
      "rdl": "https://raw.githubusercontent.com/gorai/gorai-service-vision/v1.2.0/yolox/service.rdl.json",
      "attributes": {
        "confidence_threshold": 0.7,
        "model_size": "medium"
      }
    }
  ]
}
```

Gorai will:
1. Download and cache the RDL
2. Extract container image and default configuration
3. Merge user's `attributes` with defaults
4. Deploy the container

### Method 2: Inline Configuration

```json
{
  "services": [
    {
      "name": "my_detector",
      "type": "vision",
      "model": "yolox",
      "attributes": {
        "confidence_threshold": 0.7
      },
      "external": {
        "enabled": true,
        "container": {
          "image": "ghcr.io/gorai/yolox:v1.2.0",
          "pull_policy": "IfNotPresent"
        }
      }
    }
  ]
}
```

This gives more control but requires knowing the container image and configuration.

---

## Validation

Validate your service RDL:

```bash
gorai service validate service.rdl.json
```

Checks:
- JSON is valid and conforms to schema
- Required fields are present
- Version format is correct
- Container image is accessible
- NATS topic patterns are valid
- Configuration schema is consistent

---

## CLI Integration

### Service Search

```bash
$ gorai service search vision

ghcr.io/gorai/yolox:v1.2.0
  Real-time object detection using YOLOX
  Type: vision | Model: yolox
  Accelerators: cpu, hailo, cuda
  Latency: 50ms | Throughput: 30 fps
  ★ Stable
```

### Service Info

```bash
$ gorai service info ghcr.io/gorai/yolox:v1.2.0

Name:    yolox-detector
Type:    vision
Model:   yolox
Version: v1.2.0

Description:
  Real-time object detection using YOLOX neural network

Container Images:
  • default: ghcr.io/gorai/yolox:v1.2.0
  • cpu:     ghcr.io/gorai/yolox:v1.2.0-cpu (universal)
  • hailo:   ghcr.io/gorai/yolox:v1.2.0-hailo (26 TOPS)
  • cuda:    ghcr.io/gorai/yolox:v1.2.0-cuda (Jetson)

NATS Topics:
  Subscribes: camera.*.frame (image/jpeg)
  Publishes:  vision.{name}.detections (30 Hz)

Configuration:
  • confidence_threshold (float): Min confidence (default: 0.6)
  • model_size (string): tiny|small|medium|large (default: small)
  • nms_threshold (float): NMS threshold (default: 0.45)
  • class_filter (array): Filter by classes (optional)

Performance:
  Latency:    50ms typical, 100ms max
  Throughput: 30 fps
  Startup:    5s

Resources:
  Memory: 2Gi required, 4Gi limit
  CPU:    1 core required, 2 cores limit
  GPU:    Optional

Documentation: https://docs.gorai.dev/services/yolox
```

---

## Best Practices

### 1. Provide Multiple Image Variants

Build optimized images for different platforms:

```bash
# Build multi-platform images
podman build --platform linux/amd64,linux/arm64 -t myservice:latest .

# Build accelerator-specific variants
podman build -f Containerfile.hailo -t myservice:latest-hailo .
podman build -f Containerfile.cuda -t myservice:latest-cuda .
```

### 2. Document NATS Message Formats

Include message schemas in documentation:

```markdown
## NATS Messages

### Input: camera.*.frame

```json
{
  "timestamp": 1234567890,
  "camera_name": "front_camera",
  "image": "base64-encoded-jpeg..."
}
```

### Output: vision.{name}.detections

```json
{
  "timestamp": 1234567890,
  "detections": [
    {
      "class": "person",
      "confidence": 0.92,
      "bbox": [100, 100, 200, 300]
    }
  ]
}
```
```

### 3. Set Realistic Resource Requirements

Test your service and measure actual resource usage:

```bash
# Monitor container resources
podman stats myservice

# Set requirements based on observed usage
"resource_requirements": {
  "memory": "1.5Gi",  # Observed: 1.2Gi used
  "cpu": "0.5"        # Observed: 50% CPU
}
```

### 4. Implement Health Checks

Add a simple health endpoint:

```python
# Python example
from flask import Flask, jsonify

app = Flask(__name__)

@app.route('/health')
def health():
    # Check if model loaded, NATS connected, etc.
    if model_loaded and nats_connected:
        return jsonify({"status": "healthy"}), 200
    else:
        return jsonify({"status": "unhealthy"}), 503

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8080)
```

### 5. Version Compatibility

Use semantic versioning in compatibility constraints:

```json
"compatibility": {
  "gorai_version": ">=0.3.0, <1.0.0"  // Compatible with 0.3.x
}
```

Update when breaking changes occur:

```json
"compatibility": {
  "gorai_version": ">=0.4.0"  // Requires features from 0.4.0+
}
```

---

## Comparison: service.rdl.json vs robot.json Service Config

### service.rdl.json (Distribution)

Lives in service repository. Defines:
- What the service does
- What container images are available
- What configuration options exist
- Default values
- Performance characteristics

**Purpose**: Self-description for discovery and documentation

### robot.json Service Config (Deployment)

Lives in user's robot configuration. Defines:
- Service instance name
- Which service to use (via RDL reference)
- User-specific configuration values
- Overrides (e.g., custom container image)

**Purpose**: Deployment configuration for specific robot

**Example**:

```json
// service.rdl.json (in service repo)
{
  "service": {
    "configuration": {
      "confidence_threshold": {
        "type": "float",
        "default": 0.6
      }
    },
    "container": {
      "default_image": "ghcr.io/gorai/yolox:v1.0.0"
    }
  }
}

// robot.json (user's robot)
{
  "services": [
    {
      "name": "detector",
      "rdl": "https://.../service.rdl.json",
      "attributes": {
        "confidence_threshold": 0.8  // Override default
      }
    }
  ]
}
```

---

## Future Extensions

Planned for schema v2.0:

- **Cost information**: Compute costs, licensing fees
- **Privacy/security**: Data handling policies, certifications
- **Model cards**: ML model provenance, biases, datasets
- **Multi-service coordination**: Dependencies on other services
- **Automated testing**: Integration test specifications

---

## See Also

- [specs/gorai-component-schema.yaml](gorai-component-schema.yaml) - Go component metadata
- [docs/third-party-component-ecosystem.md](../docs/third-party-component-ecosystem.md) - Developer guide
- [specs/robot-definition-language.md](robot-definition-language.md) - Full RDL specification
- [docs/modules-approach.md](../docs/modules-approach.md) - Architecture overview
