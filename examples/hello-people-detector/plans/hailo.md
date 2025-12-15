# Hailo-8 NPU Integration Plan

Goal: Achieve real-time 30fps person detection using Hailo-8 NPU in a containerized service.

## Current State

- Hailo-8 NPU connected via PCIe (M.2 HAT+)
- Device available at `/dev/hailo0` on host
- HailoRT installed on host (verified via `hailortcli fw-control identify`)
- Person detector runs in Podman container using ONNX on CPU (~2 fps)
- Container passes `--device /dev/hailo0` but lacks HailoRT Python bindings

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Raspberry Pi 5 Host                       │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                 person-detector container                ││
│  │  ┌─────────────┐    ┌──────────────┐    ┌────────────┐ ││
│  │  │ Python App  │───▶│ HailoRT      │───▶│ /dev/hailo0│ ││
│  │  │ (main.py)   │    │ Python Bindings   │ (passthrough)│ ││
│  │  └─────────────┘    └──────────────┘    └─────┬──────┘ ││
│  └───────────────────────────────────────────────┼─────────┘│
│                                                  │          │
│  ┌───────────────────────────────────────────────┼─────────┐│
│  │              HailoRT Driver (kernel)          │         ││
│  └───────────────────────────────────────────────┼─────────┘│
│                                                  ▼          │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                    Hailo-8 NPU (26 TOPS)                ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

## Requirements

### 1. Host Requirements (Already Met)

```bash
# Verify Hailo device exists
ls -la /dev/hailo0

# Verify HailoRT is working
hailortcli fw-control identify

# Check HailoRT version (important for matching container bindings)
hailortcli --version
# or
dpkg -l | grep hailort
```

Expected output from `hailortcli fw-control identify`:
```
Executing on device: 0000:01:00.0
Identifying board
Control Protocol Version: 2
Firmware Version: 4.18.0 (or similar)
Board Name: Hailo-8
Device Architecture: HAILO8
Serial Number: HLDDLBB...
```

### 2. Container Requirements

#### A. Device Passthrough
The container must have access to `/dev/hailo0`:

```bash
podman run --device /dev/hailo0 ...
```

For rootless Podman, the user running the container must have access to the device:
```bash
# On host, add user to hailo group (if group exists)
sudo usermod -a -G hailo $USER

# Or set device permissions
sudo chmod 666 /dev/hailo0

# Or use udev rule for persistent permissions
# /etc/udev/rules.d/99-hailo.rules
SUBSYSTEM=="hailo", MODE="0666"
```

#### B. HailoRT Python Bindings

The container needs `hailort` Python package matching the host HailoRT version.

**Option 1: Copy wheel from host installation**
```bash
# On host, find the installed hailort package location
pip show hailort
# Or find the wheel
find /opt/hailo -name "*.whl" 2>/dev/null
ls /usr/share/hailo/*.whl 2>/dev/null
```

**Option 2: Download from Hailo Developer Zone**
- URL: https://hailo.ai/developer-zone/software-downloads/
- Download: `hailort-X.Y.Z-cpXX-cpXX-linux_aarch64.whl`
- Requires Hailo developer account (free)

**Option 3: Install from Hailo apt repository**
```bash
# If Hailo apt repo is configured on host
apt-cache show python3-hailort
```

### 3. Model Requirements

Hailo NPU requires models in HEF (Hailo Executable Format), not ONNX.

**Option A: Download pre-compiled from Hailo Model Zoo**
```bash
# Hailo provides pre-compiled models
# https://github.com/hailo-ai/hailo_model_zoo

# Example YOLOv8 models available:
# - yolov8n.hef (nano - fastest, ~100fps on Hailo-8)
# - yolov8s.hef (small - ~50fps on Hailo-8)
# - yolov8m.hef (medium - ~25fps on Hailo-8)

# Download location (check Hailo's releases):
# https://hailo.ai/developer-zone/model-zoo/
```

**Option B: Compile ONNX to HEF**
Requires Hailo Dataflow Compiler (DFC) which runs on x86_64 Linux only:
```bash
# On x86 machine with DFC installed:
hailo compiler yolov8s.onnx --hw-arch hailo8
# Outputs: yolov8s.hef
```

## Implementation Steps

### Step 1: Verify Host Setup

Run these commands on the Pi host (rpi1):

```bash
# 1. Check device exists and permissions
ls -la /dev/hailo0

# 2. Check HailoRT version
hailortcli --version

# 3. Test NPU is working
hailortcli fw-control identify

# 4. Find HailoRT Python wheel location
find /usr -name "hailort*.whl" 2>/dev/null
find /opt -name "hailort*.whl" 2>/dev/null
pip3 show hailort 2>/dev/null

# 5. Check what HEF models are available
ls -la /opt/gorai/models/*.hef 2>/dev/null
ls -la /usr/share/hailo-rpi5-examples/resources/*.hef 2>/dev/null
```

### Step 2: Obtain HailoRT Python Wheel

**If wheel exists on host:**
```bash
# Copy to service directory
cp /path/to/hailort-*.whl /gorai/examples/hello-people-detector/services/person-detector/
```

**If not available, download from Hailo:**
1. Go to https://hailo.ai/developer-zone/software-downloads/
2. Log in (create free account if needed)
3. Download HailoRT wheel for:
   - Architecture: `linux_aarch64` (ARM64)
   - Python version: `cp311` (Python 3.11, matching container)
   - Version: Must match host HailoRT version

### Step 3: Obtain HEF Model

**Option A: Check if pre-installed on Pi**
```bash
# Raspberry Pi Hailo examples often include models
ls /usr/share/hailo-rpi5-examples/resources/*.hef

# Copy to models directory
sudo cp /usr/share/hailo-rpi5-examples/resources/yolov8s_h8l.hef /opt/gorai/models/
```

**Option B: Download from Hailo Model Zoo**
```bash
# Check Hailo Model Zoo releases
# https://github.com/hailo-ai/hailo_model_zoo/releases

# Download appropriate model for Hailo-8 (not Hailo-8L)
wget https://hailo-model-zoo.s3.eu-west-2.amazonaws.com/ModelZoo/Compiled/v2.11.0/hailo8/yolov8s.hef
sudo mv yolov8s.hef /opt/gorai/models/
```

### Step 4: Update Containerfile

```dockerfile
# Containerfile for Gorai Person Detector Service with Hailo NPU
FROM docker.io/python:3.11-slim-bookworm

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgl1 \
    libglib2.0-0 \
    libsm6 \
    libxext6 \
    libxrender1 \
    fonts-dejavu-core \
    wget \
    curl \
    # HailoRT dependencies
    libhailort1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy HailoRT wheel (must be obtained from host or Hailo)
COPY hailort-*.whl /tmp/

# Copy requirements
COPY requirements.txt .

# Install Python dependencies including HailoRT
RUN pip install --no-cache-dir /tmp/hailort-*.whl && \
    pip install --no-cache-dir -r requirements.txt && \
    rm /tmp/hailort-*.whl

# Copy application code
COPY . .

# Create non-root user with video group access
RUN useradd -m -u 1000 gorai && \
    usermod -a -G video gorai && \
    chown -R gorai:gorai /app

USER gorai

# Environment defaults - using Hailo
ENV PYTHONUNBUFFERED=1
ENV LOG_LEVEL=info
ENV NATS_URL=nats://localhost:4222
ENV MODEL_PATH=/models/yolov8s.hef
ENV CONFIDENCE_THRESHOLD=0.5
ENV INPUT_SIZE=640

ENTRYPOINT ["python", "main.py"]
```

### Step 5: Update requirements.txt

```
# Gorai Person Detector Service Dependencies

# NATS client
nats-py>=2.6.0

# Image processing
opencv-python-headless>=4.8.0
numpy>=1.24.0

# Note: hailort is installed separately from wheel
# onnxruntime kept as fallback
onnxruntime>=1.16.0
```

### Step 6: Update Hailo Backend Code

The `inference/hailo_backend.py` needs updates for proper Hailo usage:

```python
# Key changes needed in hailo_backend.py:

# 1. Import Hailo modules correctly
from hailo_platform import (
    HEF,
    VDevice,
    HailoStreamInterface,
    ConfigureParams,
    InputVStreamParams,
    OutputVStreamParams,
    FormatType,
)

# 2. Initialize with proper streaming setup
def _initialize_hailo(self) -> None:
    """Initialize Hailo NPU backend."""
    logger.info(f"Initializing Hailo backend with {self.model_path}")

    # Load HEF model
    self._hef = HEF(self.model_path)

    # Create virtual device (auto-detects Hailo-8/8L)
    params = VDevice.create_params()
    self._vdevice = VDevice(params)

    # Configure network group
    configure_params = ConfigureParams.create_from_hef(
        self._hef,
        interface=HailoStreamInterface.PCIe
    )
    self._network_groups = self._vdevice.configure(self._hef, configure_params)
    self._network_group = self._network_groups[0]

    # Get stream info
    self._input_vstreams_info = self._hef.get_input_vstream_infos()
    self._output_vstreams_info = self._hef.get_output_vstream_infos()

    # Log model info
    for info in self._input_vstreams_info:
        logger.info(f"Input: {info.name}, shape: {info.shape}, format: {info.format}")
    for info in self._output_vstreams_info:
        logger.info(f"Output: {info.name}, shape: {info.shape}, format: {info.format}")

    self._use_hailo = True
    logger.info("Hailo backend initialized successfully")

# 3. Inference with proper vstreams
def _infer_hailo(self, input_tensor: np.ndarray) -> np.ndarray:
    """Run inference on Hailo NPU."""
    # Create vstream params
    input_params = InputVStreamParams.make_from_network_group(
        self._network_group,
        quantized=False,
        format_type=FormatType.FLOAT32
    )
    output_params = OutputVStreamParams.make_from_network_group(
        self._network_group,
        quantized=False,
        format_type=FormatType.FLOAT32
    )

    # Run inference
    with self._network_group.activate():
        # Create input/output vstreams
        input_vstreams = self._network_group.create_input_vstreams(input_params)
        output_vstreams = self._network_group.create_output_vstreams(output_params)

        # Send input
        input_name = self._input_vstreams_info[0].name
        input_vstreams[input_name].send(input_tensor)

        # Get output
        output_name = self._output_vstreams_info[0].name
        output = output_vstreams[output_name].recv()

    return output
```

### Step 7: Update Podman Run Command

The container must be run with device access. Update the RDL or Makefile:

```bash
# Direct podman command
podman run -d \
    --name person_detector \
    --device /dev/hailo0 \
    --network host \
    -v /opt/gorai/models:/models:ro \
    -e NATS_URL=nats://localhost:4222 \
    -e GORAI_ROBOT_NAME=hello-people-detector \
    -e GORAI_SERVICE_NAME=person_detector \
    -e MODEL_PATH=/models/yolov8s.hef \
    localhost/person-detector:latest
```

**For rootless Podman**, ensure device permissions:
```bash
# Either run as root
sudo podman run --device /dev/hailo0 ...

# Or fix device permissions on host
sudo chmod 666 /dev/hailo0
```

## Troubleshooting

### "Device not found" Error
```bash
# Check device exists
ls -la /dev/hailo*

# Check kernel module loaded
lsmod | grep hailo

# Reload if needed
sudo modprobe hailo_pci
```

### "Permission denied" on /dev/hailo0
```bash
# Check current permissions
ls -la /dev/hailo0

# Add udev rule for persistent permissions
echo 'SUBSYSTEM=="hailo", MODE="0666"' | sudo tee /etc/udev/rules.d/99-hailo.rules
sudo udevadm control --reload-rules
sudo udevadm trigger
```

### HailoRT Version Mismatch
```bash
# Host version
hailortcli --version

# Container version (run inside container)
python -c "import hailo_platform; print(hailo_platform.__version__)"

# These MUST match!
```

### "Failed to create vdevice" Error
```bash
# Check if another process is using Hailo
lsof /dev/hailo0

# Kill conflicting process or stop other containers
podman stop $(podman ps -q)
```

## Expected Performance

| Model | Input Size | Hailo-8 FPS | CPU (ONNX) FPS |
|-------|------------|-------------|----------------|
| YOLOv8n | 640x640 | ~100 fps | ~1 fps |
| YOLOv8s | 640x640 | ~50 fps | ~0.5 fps |
| YOLOv8m | 640x640 | ~25 fps | ~0.2 fps |

With YOLOv8s on Hailo-8 at 640x640, expect **~50 fps** which is well above the 30 fps target.

## Quick Start Checklist

- [ ] Verify `/dev/hailo0` exists and is accessible
- [ ] Get HailoRT version: `hailortcli --version`
- [ ] Obtain matching `hailort-*.whl` for Python 3.11 ARM64
- [ ] Obtain `yolov8s.hef` (or yolov8n.hef for even faster)
- [ ] Copy wheel to `services/person-detector/`
- [ ] Update Containerfile to install wheel
- [ ] Copy HEF model to `/opt/gorai/models/`
- [ ] Rebuild container: `make build`
- [ ] Test: `make run`

## Files to Modify

1. `services/person-detector/Containerfile` - Add HailoRT wheel installation
2. `services/person-detector/inference/hailo_backend.py` - Fix Hailo initialization
3. `services/person-detector/requirements.txt` - Remove Pillow (unused)
4. `/opt/gorai/models/` - Add yolov8s.hef model file
