# Hello People Detector Example

A camera robot with AI-based person detection using an external service. This example demonstrates the **Service RDL** pattern for modular, reusable external services.

## Architecture

```
Robot Deployment
+-------------------------------------------------------------+
|  Host System (Raspberry Pi 5, etc.)                         |
|                                                             |
|  systemd                                                    |
|  +-- nats-server.service           (installed natively)     |
|  +-- hello-people-detector.service (gorai robot binary)     |
|                                                             |
|  Container Runtime (Podman)                                 |
|  +-- person-detector container     (external service)       |
|      - Subscribes to camera frames via NATS                 |
|      - Runs YOLOX inference on Hailo NPU                    |
|      - Publishes annotated images + detections              |
|                                                             |
|  Hardware                                                   |
|  +-- /dev/video0  -> camera component                       |
|  +-- /dev/hailo0  -> person detector container              |
+-------------------------------------------------------------+
```

## Key Concepts

### Service RDL

This example uses **Service RDL**, a separate RDL file that defines external service behavior independently of any robot. The Service RDL (`services/person-detector/person-detector.rdl.json`) specifies:

- **Topics**: What the service subscribes to and publishes
- **Attributes**: Configurable parameters with types, defaults, and validation
- **Runtime**: Default container/process configuration

The robot RDL references the Service RDL and provides:
- Service name and identity
- Attribute values (overriding defaults)
- Deployment-specific runtime configuration

### Benefits of Service RDL

1. **Modularity**: Services are self-contained packages
2. **Reusability**: Same service can be used across multiple robots
3. **Separation of concerns**: Service authors define behavior, robot integrators configure deployment
4. **Documentation**: Service RDL documents the service interface
5. **Validation**: Attributes are type-checked at load time

## Prerequisites

- Linux (Raspberry Pi OS, Ubuntu, Fedora)
- Go 1.22+ (for building gorai)
- NATS server (`sudo apt install nats-server`)
- Podman (for running external service container)
- Camera at `/dev/video0`
- Hailo-8L NPU at `/dev/hailo0` (optional, falls back to ONNX)

### Install Dependencies

```bash
# NATS server
sudo apt install nats-server
sudo systemctl enable --now nats-server

# Podman
sudo apt install podman

# Enable podman socket for managed containers
systemctl --user enable --now podman.socket
```

## Quick Start

### 1. Build Everything

```bash
# Build the robot binary and external service container
gorai build --config hello-people-detector.json
```

This will:
- Build the main robot binary
- Build the person-detector container image

### 2. Run the Robot

```bash
# Development: Run in foreground
gorai run --config hello-people-detector.json

# Production: Deploy as systemd service
gorai start --config hello-people-detector.json --enable
```

### 3. Access the Dashboard

Open http://localhost:8080 to view:
- Raw camera feed from `main_camera`
- Annotated feed with bounding boxes from `person_detector`
- Detection results and statistics

### 4. Check Status

```bash
gorai status --config hello-people-detector.json
```

Expected output:
```
Robot: hello-people-detector

COMPONENTS
NAME          TYPE    MODEL   STATUS
main_camera   camera  v4l2    running

SERVICES
NAME             TYPE              MODEL   STATUS    MODE
dashboard        dashboard         web     running   internal
person_detector  object_detection  yolox   running   external (container)
```

### 5. View Logs

```bash
# All logs
gorai logs --config hello-people-detector.json -f

# Just the external service
podman logs -f person_detector
```

## Configuration

### Robot RDL (`hello-people-detector.json`)

The robot RDL references the Service RDL:

```json
{
  "services": [
    {
      "name": "person_detector",
      "rdl": "./services/person-detector/person-detector.rdl.json",
      "attributes": {
        "input_component": "main_camera",
        "model_path": "/models/yolox_s_leaky.hef",
        "confidence_threshold": 0.5
      },
      "external": {
        "enabled": true,
        "container": {
          "devices": ["/dev/hailo0"],
          "volumes": ["/opt/gorai/models:/models:ro"]
        }
      }
    }
  ]
}
```

### Service RDL (`services/person-detector/person-detector.rdl.json`)

Defines the service interface:

```json
{
  "kind": "service",
  "service": {
    "type": "object_detection",
    "model": "yolox"
  },
  "topics": {
    "subscribe": [
      {
        "name": "input",
        "pattern": "gorai.{namespace}.{input_component}.data"
      }
    ],
    "publish": [
      {
        "name": "annotated",
        "pattern": "gorai.{namespace}.{service}.annotated"
      },
      {
        "name": "detections",
        "pattern": "gorai.{namespace}.{service}.detections"
      }
    ]
  },
  "attributes": {
    "confidence_threshold": {
      "type": "float",
      "default": 0.5
    }
  }
}
```

## NATS Topics

### Input (from camera)
```
gorai.hello-people-detector.main_camera.data
```

### Output (from person detector)
```
gorai.hello-people-detector.person_detector.annotated   # JPEG with bounding boxes
gorai.hello-people-detector.person_detector.detections  # JSON detection results
```

### Subscribe to Detection Results

```bash
# View raw detections
nats sub "gorai.hello-people-detector.person_detector.detections"
```

Example output:
```json
{
  "timestamp": "2024-12-15T10:30:00Z",
  "frame_id": 1234,
  "detections": [
    {
      "class": "person",
      "confidence": 0.87,
      "bbox": {"x1": 0.1, "y1": 0.2, "x2": 0.4, "y2": 0.9}
    }
  ]
}
```

## Running Without Hailo NPU

The person detector service can fall back to ONNX Runtime if no Hailo NPU is available:

```bash
# Download an ONNX model
wget -O /opt/gorai/models/yolox_s.onnx \
  https://github.com/Megvii-BaseDetection/YOLOX/releases/download/0.1.0/yolox_s.onnx

# Update config to use ONNX model
# Edit hello-people-detector.json:
#   "model_path": "/models/yolox_s.onnx"

# Rebuild and run
gorai build --config hello-people-detector.json
gorai run --config hello-people-detector.json
```

## Troubleshooting

### Container Won't Start

```bash
# Check container status
podman ps -a

# View container logs
podman logs person_detector

# Check if device is available
ls -la /dev/hailo0
```

### No Detections

1. Check confidence threshold (default 0.5 might be too high)
2. Verify model file exists at MODEL_PATH
3. Check NATS connectivity: `nats sub "gorai.hello-people-detector.>"`

### Permission Denied

```bash
# Add user to required groups
sudo usermod -aG video,hailo $USER

# Log out and back in
```

## Files

```
hello-people-detector/
+-- hello-people-detector.json      # Robot RDL
+-- README.md                        # This file
+-- services/
    +-- person-detector/
        +-- person-detector.rdl.json  # Service RDL
        +-- main.py                   # Python service entry point
        +-- Containerfile             # Container build definition
        +-- requirements.txt          # Python dependencies
        +-- config/                   # Configuration module
        +-- inference/                # Hailo/ONNX backend
        +-- processing/               # Post-processing (NMS, etc.)
        +-- annotate/                 # Bounding box drawing
```

## See Also

- [hello-camera](../hello-camera/) - Simple camera example without AI
- [Robot Definition Language Spec](../../specs/robot-definition-language.md) - Full RDL specification
- [Hailo Integration](../../plans/hailo.md) - Hailo NPU setup guide
