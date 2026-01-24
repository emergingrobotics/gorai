# Hailo Person Detector Service

> **Work in Progress**: This service is archived as part of Phase 2 (optional containers for ML/vision). It is not yet complete and will be developed further when container-based polyglot services are implemented. See [FUTURE-ROADMAP.md](../../../FUTURE-ROADMAP.md) for the phased architecture plan.

A containerized service that runs YOLOX person detection on the Hailo NPU.

## Overview

This service:
1. Subscribes to camera frames from NATS
2. Runs YOLOX inference on the Hailo NPU
3. Draws bounding boxes around detected persons
4. Publishes annotated frames and detection metadata to NATS

## Prerequisites

- Raspberry Pi 5 with Hailo M.2 HAT
- HailoRT installed on the host
- NATS server running
- YOLOX model in HEF format

## Model Setup

Download or compile a YOLOX model for Hailo:

```bash
# Option 1: Download from Hailo Model Zoo
wget https://hailo-model-zoo.s3.amazonaws.com/ModelZoo/Compiled/v2.10.0/hailo8l/yolox_s_leaky.hef \
  -O models/yolox_s_leaky.hef

# Option 2: Compile from ONNX using Hailo Dataflow Compiler
hailo compiler yolox_s.onnx --hw-arch hailo8l -o models/yolox_s_leaky.hef
```

## Building

```bash
podman build -t localhost/hailo-detector:latest .
```

## Running

### With Podman (recommended)

```bash
podman run --rm -it \
  --device /dev/hailo0 \
  --network host \
  -e NATS_URL=nats://localhost:4222 \
  -e GORAI_ROBOT_NAME=hello-camera \
  -e CONFIDENCE_THRESHOLD=0.5 \
  -v $(pwd)/models:/models:ro \
  localhost/hailo-detector:latest
```

### As a systemd service

```bash
# Copy service file
sudo cp deploy/hailo-detector.service /etc/systemd/system/

# Edit configuration
sudo systemctl edit hailo-detector

# Start service
sudo systemctl enable --now hailo-detector
```

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `GORAI_ROBOT_NAME` | `robot` | Robot name for topic namespace |
| `GORAI_SERVICE_NAME` | `person_detector` | Service name |
| `MODEL_PATH` | `/models/yolox_s_leaky.hef` | Path to YOLOX HEF model |
| `CONFIDENCE_THRESHOLD` | `0.5` | Detection confidence threshold |
| `NMS_THRESHOLD` | `0.45` | Non-maximum suppression threshold |
| `INPUT_SIZE` | `640` | Model input size |
| `INPUT_TOPIC` | Auto | Override input topic |
| `OUTPUT_TOPIC_ANNOTATED` | Auto | Override annotated output topic |
| `OUTPUT_TOPIC_DETECTIONS` | Auto | Override detections output topic |

## NATS Topics

### Input
- `gorai.<robot>.main_camera.data` - Raw JPEG frames from camera

### Output
- `gorai.<robot>.person_detector.annotated` - JPEG frames with bounding boxes
- `gorai.<robot>.person_detector.detections` - JSON detection metadata

### Detection Message Format

```json
{
  "timestamp": "2025-12-14T12:00:00.000Z",
  "frame_id": 12345,
  "robot": "hello-camera",
  "service": "person_detector",
  "detections": [
    {
      "class": "person",
      "class_id": 0,
      "confidence": 0.92,
      "bbox": {
        "x": 100,
        "y": 50,
        "width": 120,
        "height": 280
      }
    }
  ],
  "inference_ms": 15.3,
  "input_shape": [480, 640]
}
```

## Development

### Running without Hailo (mock mode)

If the Hailo SDK is not available, the detector runs in mock mode with random detections:

```bash
python3 detector.py
```

### Testing NATS connectivity

```bash
# Subscribe to detections
nats sub "gorai.hello-camera.person_detector.>"

# Publish test frame (requires a JPEG file)
nats pub gorai.hello-camera.main_camera.data --file test.jpg
```

## Performance

On Raspberry Pi 5 with Hailo-8L M.2:

| Model | Input Size | Inference Time | FPS |
|-------|-----------|----------------|-----|
| YOLOX-S | 640x640 | ~15ms | ~66 |
| YOLOX-Nano | 416x416 | ~8ms | ~125 |

Note: End-to-end latency includes NATS messaging and image encoding/decoding.

## Troubleshooting

### "Hailo SDK not available"

Install HailoRT on the host system:
```bash
# Follow Hailo's installation guide for your platform
# https://hailo.ai/developer-zone/documentation/
```

### "Failed to connect to NATS"

Ensure NATS server is running:
```bash
sudo systemctl status nats-server
# Or start it:
sudo systemctl start nats-server
```

### No detections appearing

1. Check confidence threshold (try lowering to 0.3)
2. Verify model path is correct
3. Check input topic matches camera output
4. View logs: `podman logs hailo-detector`
