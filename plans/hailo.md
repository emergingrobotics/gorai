# Hailo NPU Person Detection Service

**Date:** 2025-12-14
**Status:** Planning
**Author:** Claude + Human

## Overview

This plan describes how to implement a containerized AI service that uses the Hailo NPU to perform person detection using YOLOX, draws bounding boxes on frames, and publishes annotated images back to NATS for display in the dashboard.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Raspberry Pi 5                                  │
│                                                                              │
│  ┌────────────────────────────────────┐   ┌─────────────────────────────┐   │
│  │     Gorai Robot (Monolith)         │   │   Hailo Detector (Container) │   │
│  │                                    │   │                             │   │
│  │  ┌──────────┐                      │   │  ┌─────────────────────┐   │   │
│  │  │  Camera  │──publish──┐          │   │  │    YOLOX Model      │   │   │
│  │  │  (v4l2)  │           │          │   │  │    (HEF format)     │   │   │
│  │  └──────────┘           │          │   │  └──────────┬──────────┘   │   │
│  │                         │          │   │             │              │   │
│  │  ┌──────────┐           │          │   │  ┌──────────▼──────────┐   │   │
│  │  │Dashboard │           │          │   │  │   HailoRT Runtime   │   │   │
│  │  │  (web)   │◄──────────┼──────────┼───┼──│   (NPU inference)   │   │   │
│  │  └──────────┘           │          │   │  └──────────┬──────────┘   │   │
│  │                         │          │   │             │              │   │
│  └─────────────────────────┼──────────┘   │  ┌──────────▼──────────┐   │   │
│                            │              │  │  Bounding Box Draw  │   │   │
│                            ▼              │  │     (OpenCV)        │   │   │
│  ┌─────────────────────────────────────┐  │  └──────────┬──────────┘   │   │
│  │              NATS Server            │  │             │              │   │
│  │                                     │  │             │publish       │   │
│  │  gorai.robot.main_camera.data ──────┼──┼─subscribe───┘              │   │
│  │                                     │  │                            │   │
│  │  gorai.robot.person_detector.───────┼──┼─◄───────────────────────────   │
│  │       annotated                     │  │                            │   │
│  │  gorai.robot.person_detector.───────┼──┼─◄───────────────────────────   │
│  │       detections                    │  │                            │   │
│  └─────────────────────────────────────┘  └─────────────────────────────┘   │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                         Hailo M.2 NPU                                 │   │
│  │                    (accessed via /dev/hailo0)                         │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Key Design Principles

### Components vs Services

| Aspect | Components | Services |
|--------|------------|----------|
| **Runtime** | Native Go, compiled into monolith | Can be containers, external processes, or native |
| **Location** | Must run on same host as robot | Can run on same host or remote hosts |
| **Communication** | Direct function calls + NATS | NATS only |
| **Examples** | Camera, Motor, IMU, GPIO | AI inference, SLAM, Navigation |
| **Hardware Access** | Direct device access | Via NATS messages or own device access |

### Why Container for Hailo?

1. **Isolation** - HailoRT dependencies don't pollute host
2. **Python** - HailoRT has mature Python bindings
3. **Reproducibility** - Same container works across deployments
4. **Updates** - Can update AI service independently of robot
5. **Resource limits** - Can limit memory/CPU usage

## NATS Topics

### Input Topic
```
gorai.<robot>.main_camera.data
```
- Format: Raw JPEG bytes
- Rate: Up to 30 fps
- Size: ~50-150KB per frame at 640x480

### Output Topics

**Annotated Frames:**
```
gorai.<robot>.person_detector.annotated
```
- Format: JPEG with bounding boxes drawn
- Rate: Same as input (or subsampled)
- Size: Similar to input

**Detection Metadata:**
```
gorai.<robot>.person_detector.detections
```
- Format: JSON
- Content:
```json
{
  "timestamp": "2025-12-14T12:00:00.000Z",
  "frame_id": 12345,
  "detections": [
    {
      "class": "person",
      "confidence": 0.92,
      "bbox": {
        "x": 100,
        "y": 50,
        "width": 120,
        "height": 280
      }
    }
  ],
  "inference_ms": 15.3
}
```

## Implementation Plan

### Phase 1: Container Setup

#### 1.1 Base Container Image

Create `services/hailo-detector/Containerfile`:

```dockerfile
# Base image with HailoRT
FROM hailo-ai/hailo-runtime:4.17.0

# Install Python dependencies
RUN apt-get update && apt-get install -y \
    python3-pip \
    python3-opencv \
    && rm -rf /var/lib/apt/lists/*

# Install Python packages
RUN pip3 install \
    nats-py \
    numpy \
    pillow

# Copy application
WORKDIR /app
COPY detector.py .
COPY models/ /models/

# Environment
ENV NATS_URL=nats://localhost:4222
ENV MODEL_PATH=/models/yolox_s_leaky.hef
ENV CONFIDENCE_THRESHOLD=0.5
ENV INPUT_TOPIC=gorai.*.*.data
ENV OUTPUT_TOPIC_ANNOTATED=gorai.{robot}.person_detector.annotated
ENV OUTPUT_TOPIC_DETECTIONS=gorai.{robot}.person_detector.detections

ENTRYPOINT ["python3", "detector.py"]
```

#### 1.2 Detector Application

Create `services/hailo-detector/detector.py`:

```python
#!/usr/bin/env python3
"""
Hailo NPU Person Detector Service

Subscribes to camera frames, runs YOLOX inference on Hailo NPU,
draws bounding boxes, and publishes annotated frames.
"""

import asyncio
import json
import os
import time
from datetime import datetime

import cv2
import numpy as np
from hailo_platform import HEF, Device, VDevice, ConfigureParams
from hailo_platform import InputVStreamParams, OutputVStreamParams
from hailo_platform import InferVStreams, InputVStreams, OutputVStreams
import nats

# Configuration from environment
NATS_URL = os.getenv("NATS_URL", "nats://localhost:4222")
MODEL_PATH = os.getenv("MODEL_PATH", "/models/yolox_s_leaky.hef")
CONFIDENCE_THRESHOLD = float(os.getenv("CONFIDENCE_THRESHOLD", "0.5"))
ROBOT_NAME = os.getenv("GORAI_ROBOT_NAME", "robot")
SERVICE_NAME = os.getenv("GORAI_SERVICE_NAME", "person_detector")

INPUT_TOPIC = f"gorai.{ROBOT_NAME}.main_camera.data"
OUTPUT_TOPIC_ANNOTATED = f"gorai.{ROBOT_NAME}.{SERVICE_NAME}.annotated"
OUTPUT_TOPIC_DETECTIONS = f"gorai.{ROBOT_NAME}.{SERVICE_NAME}.detections"

# YOLOX class names (COCO)
CLASSES = ["person", "bicycle", "car", ...]  # Full COCO classes

class HailoDetector:
    def __init__(self, model_path: str):
        self.hef = HEF(model_path)
        self.device = VDevice()
        self.network_group = self._configure_network()
        self.input_vstream_info = self.hef.get_input_vstream_infos()[0]
        self.output_vstream_info = self.hef.get_output_vstream_infos()[0]

    def _configure_network(self):
        configure_params = ConfigureParams.create_from_hef(
            self.hef, interface=HailoStreamInterface.PCIe
        )
        network_group = self.device.configure(self.hef, configure_params)[0]
        return network_group

    def preprocess(self, frame: np.ndarray) -> np.ndarray:
        """Preprocess frame for YOLOX input."""
        # Resize to model input size (typically 640x640)
        input_shape = self.input_vstream_info.shape
        resized = cv2.resize(frame, (input_shape[2], input_shape[1]))
        # Normalize and convert to model format
        normalized = resized.astype(np.float32) / 255.0
        return normalized

    def postprocess(self, output: np.ndarray, original_shape: tuple) -> list:
        """Postprocess YOLOX output to detections."""
        detections = []
        # YOLOX output parsing logic
        # ... decode boxes, scores, classes
        return detections

    def infer(self, frame: np.ndarray) -> list:
        """Run inference on a frame."""
        preprocessed = self.preprocess(frame)

        with InferVStreams(self.network_group,
                          self.input_vstream_info,
                          self.output_vstream_info) as pipeline:
            pipeline.send(preprocessed)
            output = pipeline.recv()

        return self.postprocess(output, frame.shape)

def draw_bounding_boxes(frame: np.ndarray, detections: list) -> np.ndarray:
    """Draw bounding boxes on frame."""
    annotated = frame.copy()

    for det in detections:
        if det["class"] != "person":
            continue
        if det["confidence"] < CONFIDENCE_THRESHOLD:
            continue

        bbox = det["bbox"]
        x, y, w, h = bbox["x"], bbox["y"], bbox["width"], bbox["height"]

        # Draw rectangle
        cv2.rectangle(annotated, (x, y), (x + w, y + h), (0, 255, 0), 2)

        # Draw label
        label = f"person {det['confidence']:.2f}"
        cv2.putText(annotated, label, (x, y - 10),
                   cv2.FONT_HERSHEY_SIMPLEX, 0.5, (0, 255, 0), 2)

    return annotated

async def main():
    print(f"Hailo Person Detector Service")
    print(f"  NATS: {NATS_URL}")
    print(f"  Model: {MODEL_PATH}")
    print(f"  Input topic: {INPUT_TOPIC}")
    print(f"  Output topics: {OUTPUT_TOPIC_ANNOTATED}, {OUTPUT_TOPIC_DETECTIONS}")

    # Initialize Hailo
    detector = HailoDetector(MODEL_PATH)
    print("Hailo NPU initialized")

    # Connect to NATS
    nc = await nats.connect(NATS_URL)
    print("Connected to NATS")

    frame_count = 0

    async def message_handler(msg):
        nonlocal frame_count
        start_time = time.time()

        # Decode JPEG
        jpeg_data = msg.data
        nparr = np.frombuffer(jpeg_data, np.uint8)
        frame = cv2.imdecode(nparr, cv2.IMREAD_COLOR)

        if frame is None:
            return

        # Run inference
        detections = detector.infer(frame)
        inference_time = (time.time() - start_time) * 1000

        # Filter to persons only
        person_detections = [d for d in detections
                           if d["class"] == "person"
                           and d["confidence"] >= CONFIDENCE_THRESHOLD]

        # Draw bounding boxes
        annotated = draw_bounding_boxes(frame, person_detections)

        # Encode to JPEG
        _, jpeg_annotated = cv2.imencode('.jpg', annotated,
                                         [cv2.IMWRITE_JPEG_QUALITY, 80])

        # Publish annotated frame
        await nc.publish(OUTPUT_TOPIC_ANNOTATED, jpeg_annotated.tobytes())

        # Publish detections metadata
        detection_msg = {
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "frame_id": frame_count,
            "detections": person_detections,
            "inference_ms": inference_time
        }
        await nc.publish(OUTPUT_TOPIC_DETECTIONS,
                        json.dumps(detection_msg).encode())

        frame_count += 1
        if frame_count % 100 == 0:
            print(f"Processed {frame_count} frames, "
                  f"last inference: {inference_time:.1f}ms")

    # Subscribe to camera frames
    await nc.subscribe(INPUT_TOPIC, cb=message_handler)
    print(f"Subscribed to {INPUT_TOPIC}")

    # Keep running
    while True:
        await asyncio.sleep(1)

if __name__ == "__main__":
    asyncio.run(main())
```

### Phase 2: Model Preparation

#### 2.1 Obtain YOLOX HEF Model

```bash
# Option 1: Download pre-compiled from Hailo Model Zoo
wget https://hailo-model-zoo.s3.amazonaws.com/ModelZoo/Compiled/v2.10.0/yolox_s_leaky.hef

# Option 2: Compile from ONNX using Hailo Dataflow Compiler
hailo compiler yolox_s.onnx --hw-arch hailo8l
```

#### 2.2 Model Directory Structure

```
services/hailo-detector/
├── Containerfile
├── detector.py
├── requirements.txt
└── models/
    └── yolox_s_leaky.hef
```

### Phase 3: RDL Configuration

Update `hello-camera.json`:

```json
{
  "$schema": "https://gorai.dev/schemas/rdl-v2.json",
  "version": "2",

  "robot": {
    "name": "hello-camera",
    "description": "Camera robot with Hailo NPU person detection"
  },

  "nats": {
    "url": "nats://localhost:4222"
  },

  "components": [
    {
      "name": "main_camera",
      "type": "camera",
      "model": "v4l2",
      "attributes": {
        "device": "/dev/video0",
        "width": 640,
        "height": 480,
        "frame_rate": 30
      }
    }
  ],

  "services": [
    {
      "name": "dashboard",
      "type": "dashboard",
      "model": "web",
      "attributes": {
        "listen": ":8080"
      }
    },
    {
      "name": "person_detector",
      "type": "object_detection",
      "model": "hailo_yolox",
      "attributes": {
        "model_path": "/models/yolox_s_leaky.hef",
        "confidence_threshold": 0.5,
        "classes": ["person"]
      },
      "external": {
        "enabled": true,
        "container": {
          "image": "localhost/hailo-detector:latest",
          "devices": ["/dev/hailo0"],
          "environment": {
            "MODEL_PATH": "/models/yolox_s_leaky.hef",
            "CONFIDENCE_THRESHOLD": "0.5"
          }
        }
      }
    }
  ],

  "dashboard": {
    "enabled": true,
    "listen": ":8080"
  }
}
```

### Phase 4: Dashboard Integration

#### 4.1 AI/Models Tab

The dashboard AI/Models tab should:

1. **List AI Services** - Show configured AI services from RDL
2. **Subscribe to Output Topics** - Automatically subscribe to annotated frame topics
3. **Display Annotated Frames** - Show frames with bounding boxes like camera feed
4. **Show Detection Stats** - Display inference time, detection count, etc.

#### 4.2 Topic Discovery

The dashboard can discover AI output topics by pattern:
```
gorai.<robot>.*.annotated   → Annotated frame streams
gorai.<robot>.*.detections  → Detection metadata
```

### Phase 5: Container Management

#### 5.1 Build Container

```bash
cd services/hailo-detector
podman build -t localhost/hailo-detector:latest .
```

#### 5.2 Run Container Manually (Testing)

```bash
podman run --rm -it \
  --device /dev/hailo0 \
  -e NATS_URL=nats://host.containers.internal:4222 \
  -e GORAI_ROBOT_NAME=hello-camera \
  localhost/hailo-detector:latest
```

#### 5.3 Systemd Service for Container

Create `/etc/systemd/system/hailo-detector.service`:

```ini
[Unit]
Description=Hailo NPU Person Detector
After=network-online.target nats-server.service hello-camera.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/bin/podman run --rm \
    --name hailo-detector \
    --device /dev/hailo0 \
    -e NATS_URL=nats://localhost:4222 \
    -e GORAI_ROBOT_NAME=hello-camera \
    localhost/hailo-detector:latest
ExecStop=/usr/bin/podman stop hailo-detector
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## File Changes Required

### New Files

1. `services/hailo-detector/Containerfile`
2. `services/hailo-detector/detector.py`
3. `services/hailo-detector/requirements.txt`
4. `services/hailo-detector/models/.gitkeep`

### Modified Files

1. `specs/robot-definition-language.md` - Add container service config
2. `specs/gorai-container.md` - Clarify services can be containers
3. `pkg/config/config.go` - Add ContainerConfig to ExternalConfig
4. `examples/hello-camera/hello-camera.json` - Add person_detector service
5. `README.md` - Document component vs service distinction

## Testing Plan

### Unit Tests
- Container builds successfully
- Detector initializes with mock Hailo
- Bounding box drawing works correctly

### Integration Tests
- Container receives frames from NATS
- Hailo inference runs on real NPU
- Annotated frames published correctly
- Dashboard displays annotated stream

### Performance Tests
- Measure end-to-end latency (target: <100ms)
- Measure sustained throughput (target: 15+ fps)
- Memory usage under load

## Future Enhancements

1. **Multiple Models** - Support switching between detection models
2. **Tracking** - Add object tracking (DeepSORT, ByteTrack)
3. **Actions** - Trigger actions based on detections
4. **Recording** - Record clips when persons detected
5. **Remote Inference** - Run on separate host with better GPU/NPU

## Dependencies

### Host Requirements
- Raspberry Pi 5 with Hailo M.2 HAT
- HailoRT kernel driver installed
- `/dev/hailo0` device accessible
- Podman installed

### Container Requirements
- HailoRT 4.17.0+
- Python 3.10+
- OpenCV 4.x
- nats-py

## Timeline

| Phase | Tasks | Estimate |
|-------|-------|----------|
| 1 | Container setup, base detector | - |
| 2 | Model integration, inference | - |
| 3 | RDL config, external service | - |
| 4 | Dashboard integration | - |
| 5 | Testing and optimization | - |

## References

- [Hailo Developer Zone](https://hailo.ai/developer-zone/)
- [Hailo Model Zoo](https://github.com/hailo-ai/hailo_model_zoo)
- [YOLOX Paper](https://arxiv.org/abs/2107.08430)
- [HailoRT Documentation](https://hailo.ai/developer-zone/documentation/)
