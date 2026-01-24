# Hello People Detector Example

> **🚧 WORK IN PROGRESS - NOT YET FUNCTIONAL 🚧**
>
> This example is currently under development and does not work yet. The implementation is incomplete.
>
> **Working examples:** See [hello-robot](../hello-robot/) or [hello-robot-production](../hello-robot-production/) for fully functional examples.

A camera robot with AI-based person detection using an external service. This example demonstrates the **Service RDL** pattern for modular, reusable external services.

## Architecture

```
K3s Cluster (single-node)
┌─────────────────────────────────────────────────────────────────┐
│  Namespace: gorai-hello-people-detector                         │
│                                                                 │
│  ┌─────────────┐   ┌──────────────────────────────────────────┐ │
│  │ nats pod    │   │ gorai-core pod                           │ │
│  │             │◄──│  ├── camera component (V4L2)             │ │
│  │ NATS server │   │  └── dashboard service (:8080)           │ │
│  └─────────────┘   └──────────────────────────────────────────┘ │
│        ▲                                                        │
│        │ NATS messaging                                         │
│        ▼                                                        │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ person-detector pod (external service)                      ││
│  │  ├── Subscribes to camera frames via NATS                   ││
│  │  ├── Runs YOLOX inference on Hailo NPU                      ││
│  │  └── Publishes annotated images + detections                ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                 │
│  Hardware passthrough: /dev/video0, /dev/hailo0                 │
└─────────────────────────────────────────────────────────────────┘
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

- Raspberry Pi 5 (8GB) with K3s installed
- NVMe SSD or USB 3.0 SSD (SD cards not supported)
- Camera at `/dev/video0`
- Hailo-8 NPU at `/dev/hailo0` (for real-time inference)
- Go 1.22+ (for building gorai CLI)

See [K3s Installation Guide](../../specs/k3s-installation.md) for K3s setup.

## Host Setup (Raspberry Pi 5)

This section covers Hailo NPU setup before deploying the robot.

### 1. Verify K3s is Running

```bash
sudo k3s kubectl get nodes
# Should show: Ready
```

### 2. Hailo-8 NPU Setup

The Hailo-8 AI accelerator provides 26 TOPS for real-time inference (~50 fps for YOLOv8s).

#### Install Hailo Software Stack

Raspberry Pi OS includes Hailo packages in the official repository:

```bash
# Install Hailo meta-package (includes everything)
sudo apt install -y hailo-all

# This installs:
# - hailort          : HailoRT runtime library
# - hailofw          : Hailo firmware
# - hailo-dkms       : PCIe kernel driver
# - python3-hailort  : Python bindings
# - hailo-tappas-core: Pre-compiled models and tools
```

#### Verify Installation

```bash
# Check device is detected
ls -la /dev/hailo0
# Expected: crw-rw-rw- 1 root plugdev 238, 0 ... /dev/hailo0

# Check HailoRT version
hailortcli --version
# Expected: HailoRT-CLI version 4.20.0 (or similar)

# Verify NPU communication
hailortcli fw-control identify
# Expected output:
# Executing on device: 0001:04:00.0
# Identifying board
# Control Protocol Version: 2
# Firmware Version: 4.20.0
# Board Name: Hailo-8
# Device Architecture: HAILO8
# Serial Number: HLLWM2B...
```

#### Device Permissions

The device should be accessible to all users by default. If not:

```bash
# Check current permissions
ls -la /dev/hailo0

# If permissions are restrictive, add udev rule
echo 'SUBSYSTEM=="hailo", MODE="0666"' | sudo tee /etc/udev/rules.d/99-hailo.rules
sudo udevadm control --reload-rules
sudo udevadm trigger

# Or add user to plugdev group
sudo usermod -aG plugdev $USER
# Log out and back in
```

### 3. Prepare Container Build Dependencies

The container needs Hailo Python bindings and the native library. These must be copied from the host into the container build context.

#### Locate Required Files

```bash
# Python bindings location
ls /usr/lib/python3/dist-packages/hailo_platform/
# Key file: pyhailort/_pyhailort.cpython-311-aarch64-linux-gnu.so

# Native library location
ls /usr/lib/libhailort*
# Files: libhailort.so -> libhailort.so.4.20.0

# Pre-compiled HEF models
ls /usr/share/hailo-models/*.hef
# Models available:
# - yolov8s_h8.hef      : YOLOv8 small for Hailo-8 (~50 fps)
# - yolov8s_h8l.hef     : YOLOv8 small for Hailo-8L
# - yolov6n_h8.hef      : YOLOv6 nano for Hailo-8
# - yolox_s_leaky_h8l_rpi.hef : YOLOX small for Hailo-8L
```

#### Copy Files to Build Context

Run this script to copy all required files for container build:

```bash
cd /path/to/gorai/examples/hello-people-detector/services/person-detector

# Create directories for Hailo runtime files
mkdir -p hailo_runtime/lib
mkdir -p hailo_runtime/python

# Copy Python bindings (entire hailo_platform package)
cp -r /usr/lib/python3/dist-packages/hailo_platform hailo_runtime/python/
cp -r /usr/lib/python3/dist-packages/hailort-*.egg-info hailo_runtime/python/

# Copy native library
cp /usr/lib/libhailort.so.4.20.0 hailo_runtime/lib/
ln -sf libhailort.so.4.20.0 hailo_runtime/lib/libhailort.so

# Verify
ls -la hailo_runtime/lib/
ls -la hailo_runtime/python/hailo_platform/
```

#### Copy Model File

```bash
# Create models directory if needed
sudo mkdir -p /opt/gorai/models

# Copy the YOLOv8s model for Hailo-8
sudo cp /usr/share/hailo-models/yolov8s_h8.hef /opt/gorai/models/

# Verify
ls -la /opt/gorai/models/
```

### 4. Verify Host Python Bindings Work

Before building the container, verify the host installation works:

```bash
# Test Python can import hailo_platform
python3 -c "
from hailo_platform import HEF, VDevice
print('HailoRT Python bindings loaded successfully')
print(f'Creating VDevice...')
vd = VDevice()
print('VDevice created - Hailo NPU is accessible')
"
```

### 5. Version Compatibility Matrix

| Component | Version | Location |
|-----------|---------|----------|
| Raspberry Pi OS | Bookworm (Debian 12) | - |
| Python | 3.11.x | System |
| HailoRT | 4.20.0 | `/usr/lib/libhailort.so.4.20.0` |
| Python bindings | 4.20.0 | `/usr/lib/python3/dist-packages/hailo_platform/` |
| Container Python | 3.11.x | Must match host Python version |

**Important**: The container's Python version must match the host's Python version (3.11) because the native extension (`_pyhailort.cpython-311-aarch64-linux-gnu.so`) is compiled for a specific Python version.

### Summary Checklist

Before deploying, ensure:

- [ ] K3s is running (`sudo k3s kubectl get nodes`)
- [ ] `hailortcli fw-control identify` shows Hailo-8 NPU
- [ ] `/dev/hailo0` exists with appropriate permissions
- [ ] `python3-hailort` package installed (`dpkg -l | grep python3-hailort`)
- [ ] Python can import hailo_platform
- [ ] Files copied to `services/person-detector/hailo_runtime/`:
  - [ ] `lib/libhailort.so.4.20.0`
  - [ ] `lib/libhailort.so` (symlink)
  - [ ] `python/hailo_platform/` (directory)
- [ ] Model file at `/opt/gorai/models/yolov8s_h8.hef`

## Quick Start

### 1. Build Container Images

```bash
# Build the external service container
gorai build --config hello-people-detector.json
```

This will:
- Build the person-detector container image
- Import it into the K3s containerd registry

### 2. Deploy to K3s

```bash
# Deploy to K3s
gorai deploy hello-people-detector.json

# Watch deployment
gorai status hello-people-detector
```

### 3. Access the Dashboard

Open http://localhost:8080 to view:
- Raw camera feed from `main_camera`
- Annotated feed with bounding boxes from `person_detector`
- Detection results and statistics

### 4. Check Status

```bash
gorai status hello-people-detector
```

Expected output:
```
Robot: hello-people-detector

PODS
NAME                         READY   STATUS
nats-0                       1/1     Running
gorai-core-xxxxx             1/1     Running
person-detector-xxxxx        1/1     Running

COMPONENTS
NAME          TYPE    MODEL   STATUS
main_camera   camera  v4l2    running

SERVICES
NAME             TYPE              MODEL   STATUS
dashboard        dashboard         web     running
person_detector  object_detection  yolox   running
```

### 5. View Logs

```bash
# All logs via gorai CLI
gorai logs hello-people-detector -f

# Person detector pod logs directly
sudo k3s kubectl logs -n gorai-hello-people-detector -l app=person-detector -f
```

### 6. Undeploy

```bash
gorai undeploy hello-people-detector
```

## Understanding the Log Output

The person detector logs timing information every 10 frames:

```
Frame 100: total=523.4ms (infer=498.2ms, post=0.3ms, draw=18.1ms, pub=6.8ms) | 2 detections | input=45.2KB | fps=1.9 | skipped=127
```

| Field | Description |
|-------|-------------|
| `total` | Total frame processing time |
| `infer` | Inference time (decode + preprocess + model + parse) |
| `post` | Post-processing time (NMS, filtering) |
| `draw` | Bounding box drawing + JPEG encoding |
| `pub` | NATS publish time |
| `detections` | Number of persons detected |
| `input` | Input JPEG size |
| `fps` | Current processing FPS |
| `skipped` | Frames skipped (when processing can't keep up) |

### Performance Analysis

- **If `infer` is high (>100ms)**: Using ONNX CPU fallback, not Hailo NPU
- **If `draw` is high (>50ms)**: Image encoding bottleneck
- **If `skipped` is growing**: Camera FPS exceeds processing FPS
- **Expected with Hailo-8**: `infer` ~20ms, `total` ~40ms, fps ~25-50

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
# View raw detections (from within cluster or with port-forward)
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

# Rebuild and deploy
gorai build --config hello-people-detector.json
gorai deploy hello-people-detector.json
```

## Troubleshooting

### Pod Won't Start

```bash
# Check pod status
sudo k3s kubectl get pods -n gorai-hello-people-detector

# View pod events
sudo k3s kubectl describe pod -n gorai-hello-people-detector -l app=person-detector

# View pod logs
sudo k3s kubectl logs -n gorai-hello-people-detector -l app=person-detector
```

### Hailo NPU Not Detected in Container

```bash
# Check if device is passed through to pod
sudo k3s kubectl exec -n gorai-hello-people-detector -it deploy/person-detector -- ls -la /dev/hailo0

# Check if another process is using the NPU
lsof /dev/hailo0
```

### "Failed to create VDevice" Error

This usually means another process has the NPU open:

```bash
# Check what's using the device
sudo lsof /dev/hailo0

# If rpicam-apps is running, stop it
pkill -f rpicam
```

### No Detections / No Bounding Boxes

1. **Check model format**: Ensure using `.hef` file for Hailo, `.onnx` for CPU fallback
2. **Check model architecture**: `yolov8s_h8.hef` for Hailo-8, `yolov8s_h8l.hef` for Hailo-8L
3. **Lower confidence threshold**: Default 0.5 might be too high for your scene
4. **Check NATS connectivity**: `nats sub "gorai.hello-people-detector.>"`
5. **View pod logs**: `sudo k3s kubectl logs -n gorai-hello-people-detector -l app=person-detector -f`

### Slow Performance (< 10 fps)

If running slower than expected:

```bash
# Check which backend is being used
sudo k3s kubectl logs -n gorai-hello-people-detector -l app=person-detector | grep -i "backend\|hailo\|onnx"

# If "ONNX backend" appears, Hailo isn't being used
```

Expected performance:
| Backend | Model | FPS |
|---------|-------|-----|
| Hailo-8 NPU | yolov8s_h8.hef | ~50 fps |
| Hailo-8L NPU | yolov8s_h8l.hef | ~25 fps |
| ONNX CPU | yolov8s.onnx | ~2 fps |

### Permission Denied on /dev/hailo0

```bash
# Check current permissions
ls -la /dev/hailo0

# Add udev rule for persistent permissions
echo 'SUBSYSTEM=="hailo", MODE="0666"' | sudo tee /etc/udev/rules.d/99-hailo.rules
sudo udevadm control --reload-rules
sudo udevadm trigger

# Or add user to plugdev group
sudo usermod -aG video,plugdev $USER
# Log out and back in
```

### HailoRT Version Mismatch

The container's HailoRT version must match the host:

```bash
# Check host version
hailortcli --version

# If mismatch, rebuild container with updated hailo_runtime/ files
```

## Files

```
hello-people-detector/
├── hello-people-detector.json      # Robot RDL
├── README.md                        # This file
├── Makefile                         # Build and run targets
├── plans/
│   └── hailo.md                     # Hailo NPU integration plan
└── services/
    └── person-detector/
        ├── person-detector.rdl.json  # Service RDL
        ├── main.py                   # Python service entry point
        ├── Containerfile             # Container build definition
        ├── requirements.txt          # Python dependencies
        ├── config/                   # Configuration module
        │   └── settings.py
        ├── inference/                # Hailo/ONNX backend
        │   └── hailo_backend.py
        ├── processing/               # Post-processing (NMS, etc.)
        │   └── postprocess.py
        ├── annotate/                 # Bounding box drawing
        │   └── draw_boxes.py
        └── hailo_runtime/            # Copied from host (not in git)
            ├── lib/
            │   ├── libhailort.so.4.20.0
            │   └── libhailort.so -> libhailort.so.4.20.0
            └── python/
                └── hailo_platform/   # Python bindings
```

**Note**: The `hailo_runtime/` directory is not checked into git. It must be populated from the host system before building. See [Host Setup](#3-prepare-container-build-dependencies).

## See Also

- [hello-camera](../hello-camera/) - Simple camera example without AI
- [Robot Definition Language Spec](../../specs/robot-definition-language.md) - Full RDL specification
- [Hailo Integration](../../plans/hailo.md) - Hailo NPU setup guide
