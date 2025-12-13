# Object Detection Service - Implementation Plan

**Date**: 2025-12-13
**Status**: Design
**Author**: AI-assisted design

## Executive Summary

This plan describes implementing a person detection service using:
- Hailo-8L NPU on Raspberry Pi 5
- YOLOx model for real-time person detection
- NATS for image transport and detection results
- Dashboard tab for visualizing detections

**Key Architectural Decision**: Components and services run in **OCI containers orchestrated by Podman**. The Python-based Hailo inference runs in its own container, isolated from the Go core.

> **See also**: [specs/podman-compose-functionality.md](/gorai/specs/podman-compose-functionality.md) for full container orchestration specification.

---

## Part 1: Architecture Decision - Separate Binaries

### Why Separate Processes (Not Monolithic)

After analyzing ROS2's distributed node architecture, Gorai's existing patterns, and robotics best practices, **separate processes is the correct approach**:

| Factor | Monolithic | Separate Processes |
|--------|------------|-------------------|
| **Distributed hardware** | Cannot span machines | Components on different Pis |
| **NPU runtime** | Hailo runtime may conflict | Isolated Python environment |
| **Fault isolation** | One crash kills everything | Services restart independently |
| **Language flexibility** | Go only | Go + Python for ML |
| **Resource control** | Shared memory | Per-process cgroups/limits |
| **Hot updates** | Full restart required | Update single service |
| **Scaling** | N/A | Multiple detection workers |

### Real-World Robotics Validation

- **ROS2**: Moved away from single Master node to DDS-based distributed nodes
- **Viam**: Components run as separate processes or remote connections
- **YARP**: Modules are separate processes communicating via ports

### The Critical Use Case: Heterogeneous Hardware

A typical Gorai robot may have:
```
┌─────────────────────────────────────────────────────────────┐
│ Raspberry Pi 5 (Main Brain)                                 │
│   - gorai-supervisor (orchestrator)                         │
│   - camera component (V4L2)                                 │
│   - motor component (GPIO/I2C)                              │
│   - dashboard service                                       │
└─────────────────────────────────────────────────────────────┘
         │ NATS                    │ NATS
         ▼                         ▼
┌─────────────────────┐  ┌─────────────────────────────────────┐
│ Hailo-8L AI HAT     │  │ Remote Sensor Node (Pi Zero)        │
│   - object_detection│  │   - ultrasonic components           │
│     service (Python)│  │   - gorai-node (Go)                 │
└─────────────────────┘  └─────────────────────────────────────┘
```

This topology is **impossible with a monolithic binary**.

---

## Part 2: Container Orchestration with Podman

### Overview

Gorai uses **Podman** to orchestrate containers:
1. Go core components run in `gorai-core` container
2. Python/Hailo inference runs in `gorai-hailo` container
3. NATS runs in its own container
4. `gorai start/stop` commands wrap `podman-compose`

### Container Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Host System (Raspberry Pi 5)                                │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐│
│  │ nats container                                         ││
│  │  - NATS server (port 4222, 8222)                       ││
│  │  - Health check on /healthz                            ││
│  └────────────────────────────────────────────────────────┘│
│                         │                                   │
│  ┌────────────────────────────────────────────────────────┐│
│  │ gorai-core container (Go)                              ││
│  │  - Camera component (V4L2)                             ││
│  │  - Dashboard service                                   ││
│  │  - /dev/video0 passthrough                             ││
│  └────────────────────────────────────────────────────────┘│
│                         │                                   │
│  ┌────────────────────────────────────────────────────────┐│
│  │ gorai-hailo container (Python)                         ││
│  │  - Object detection service                            ││
│  │  - Hailo runtime + YOLOx model                         ││
│  │  - /dev/hailo0 passthrough                             ││
│  └────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

### RDL Container Definition

```json
{
  "version": "1",
  "robot": {
    "name": "watchbot"
  },
  "containers": {
    "nats": {
      "image": "docker.io/nats:2.10-alpine",
      "ports": ["4222:4222", "8222:8222"],
      "restart": "always",
      "healthcheck": {
        "test": ["CMD", "wget", "-q", "--spider", "http://localhost:8222/healthz"],
        "interval": "10s"
      }
    },
    "gorai-core": {
      "build": {"context": ".", "dockerfile": "Containerfile"},
      "depends_on": {"nats": {"condition": "service_healthy"}},
      "devices": ["/dev/video0"],
      "ports": ["8080:8080"],
      "components": ["main_camera"],
      "services": ["dashboard"]
    },
    "gorai-hailo": {
      "build": {"context": "./services/object-detection", "dockerfile": "Containerfile.hailo"},
      "depends_on": {
        "nats": {"condition": "service_healthy"},
        "gorai-core": {"condition": "service_started"}
      },
      "devices": ["/dev/hailo0"],
      "volumes": ["./models:/models:ro"],
      "services": ["person_detector"]
    }
  },
  "services": [
    {
      "name": "person_detector",
      "type": "object_detection",
      "model": "hailo_yolox",
      "container": "gorai-hailo",
      "attributes": {
        "model_path": "/models/yolox_s_leaky.hef",
        "input_topic": "gorai.watchbot.main_camera.data",
        "confidence_threshold": 0.5,
        "classes": ["person"]
      }
    }
  ]
}
```

### Gorai CLI Commands

```bash
# Build and start all containers
gorai start --config robot.json --build --detach

# Check status
gorai status --config robot.json

# View logs
gorai logs --config robot.json --follow

# Stop all containers
gorai stop --config robot.json
```

### Dependency Conditions

| Condition | Description |
|-----------|-------------|
| `service_started` | Container has started (default) |
| `service_healthy` | Container passes health check |
| `service_completed_successfully` | Container exited with code 0 |

### Heartbeat Protocol (Optional)

Containers can still publish heartbeats for dashboard visibility:

**Topic**: `gorai.<robot>._system.heartbeat`

```json
{
  "name": "person_detector",
  "container": "gorai-hailo",
  "status": "running",
  "metrics": {
    "fps": 25.3,
    "inference_ms": 38
  },
  "timestamp": "2025-12-13T12:00:00Z"
}
```

---

## Part 3: Object Detection Service Design

### Data Flow

```
Camera                   Object Detection              Dashboard
  │                           Service                      │
  │                             │                          │
  │ JPEG frame                  │                          │
  ├────────────────────────────►│                          │
  │ gorai.robot.camera.data     │                          │
  │                             │                          │
  │                      ┌──────┴──────┐                   │
  │                      │ Hailo NPU   │                   │
  │                      │ YOLOx       │                   │
  │                      │ Inference   │                   │
  │                      └──────┬──────┘                   │
  │                             │                          │
  │                      ┌──────┴──────┐                   │
  │                      │ Draw boxes  │                   │
  │                      │ on image    │                   │
  │                      └──────┬──────┘                   │
  │                             │                          │
  │                             │ annotated JPEG           │
  │                             ├─────────────────────────►│
  │                             │ gorai.robot.detector.    │
  │                             │        annotated         │
  │                             │                          │
  │                             │ detection metadata       │
  │                             ├─────────────────────────►│
  │                             │ gorai.robot.detector.    │
  │                             │        detections        │
```

### NATS Topics

| Topic | Content | Format |
|-------|---------|--------|
| `gorai.<robot>.<camera>.data` | Raw camera frames | JPEG bytes |
| `gorai.<robot>.<detector>.annotated` | Frames with bounding boxes | JPEG bytes |
| `gorai.<robot>.<detector>.detections` | Detection metadata | JSON |

### Detection Message Format

```json
{
  "timestamp": "2025-12-13T12:00:00.123Z",
  "frame_id": 12345,
  "inference_time_ms": 38,
  "image_width": 640,
  "image_height": 480,
  "detections": [
    {
      "class_id": 0,
      "class_name": "person",
      "confidence": 0.92,
      "bbox": {
        "x": 120,
        "y": 80,
        "width": 150,
        "height": 320
      }
    }
  ]
}
```

### Service Interface

```go
// service/object_detection/object_detection.go

type ObjectDetection interface {
    service.Service

    // Detect runs detection on an image and returns results
    Detect(ctx context.Context, image []byte) (*DetectionResult, error)

    // DetectAsync subscribes to input topic and publishes to output topic
    DetectAsync(ctx context.Context) error

    // GetMetrics returns current performance metrics
    GetMetrics(ctx context.Context) (*Metrics, error)
}

type DetectionResult struct {
    Timestamp     time.Time
    FrameID       uint64
    InferenceMs   int64
    ImageWidth    int
    ImageHeight   int
    Detections    []Detection
    AnnotatedJPEG []byte  // Optional: image with boxes drawn
}

type Detection struct {
    ClassID    int
    ClassName  string
    Confidence float32
    BBox       BoundingBox
}

type BoundingBox struct {
    X      int
    Y      int
    Width  int
    Height int
}
```

---

## Part 4: Hailo NPU Integration

### Why Python for Inference

Hailo provides excellent Python support via:
- **HailoRT Python bindings** - direct hardware access
- **TAPPAS** - GStreamer-based pipelines
- **hailo-rpi5-examples** - ready-to-use detection examples

There are **no official Go bindings** for HailoRT. Options:

| Approach | Pros | Cons |
|----------|------|------|
| **Go + CGO** | Single binary | Complex, no official support |
| **Go subprocess** | Simple | Slow startup per frame |
| **Python service + NATS** | Clean separation, best Hailo support | Two processes |

**Recommendation**: Python inference service communicating via NATS.

### Python Service Architecture

```
gorai-object-detection (Python)
├── main.py                 # Entry point, NATS connection
├── inference/
│   ├── hailo_backend.py    # HailoRT inference
│   └── model_loader.py     # HEF file loading
├── processing/
│   ├── preprocess.py       # Image preprocessing
│   └── postprocess.py      # NMS, box decoding
├── annotate/
│   └── draw_boxes.py       # PIL/OpenCV drawing
└── config/
    └── settings.py         # Configuration from RDL
```

### HailoRT Integration Code

```python
# inference/hailo_backend.py

from hailo_platform import HEF, VDevice, ConfigureParams
import numpy as np

class HailoInference:
    def __init__(self, hef_path: str, batch_size: int = 1):
        self.hef = HEF(hef_path)
        self.vdevice = VDevice()

        # Configure network
        configure_params = ConfigureParams.create_from_hef(
            self.hef,
            interface=HailoStreamInterface.PCIe
        )
        self.network_group = self.vdevice.configure(
            self.hef,
            configure_params
        )[0]

        # Get input/output info
        self.input_vstream_info = self.network_group.get_input_vstream_infos()[0]
        self.output_vstream_info = self.network_group.get_output_vstream_infos()

        self.input_shape = self.input_vstream_info.shape

    def infer(self, image: np.ndarray) -> dict:
        """Run inference on preprocessed image."""
        with self.network_group.activate():
            input_data = {self.input_vstream_info.name: image}

            with InferVStreams(self.network_group, input_data) as infer_pipeline:
                results = infer_pipeline.infer(input_data)

        return results
```

### NATS Integration (Python)

```python
# main.py

import asyncio
import nats
from inference.hailo_backend import HailoInference
from processing.postprocess import decode_yolox_output
from annotate.draw_boxes import draw_detections
import json
import io
from PIL import Image

class ObjectDetectionService:
    def __init__(self, config):
        self.config = config
        self.nc = None
        self.inference = HailoInference(config['model_path'])

    async def connect(self):
        self.nc = await nats.connect(self.config['nats_url'])

    async def run(self):
        input_topic = self.config['input_topic']
        output_annotated = self.config['output_topic_annotated']
        output_detections = self.config['output_topic_detections']

        async def message_handler(msg):
            # Decode JPEG
            image = Image.open(io.BytesIO(msg.data))

            # Preprocess for model
            input_tensor = self.preprocess(image)

            # Run inference
            start = time.time()
            outputs = self.inference.infer(input_tensor)
            inference_ms = (time.time() - start) * 1000

            # Decode detections
            detections = decode_yolox_output(
                outputs,
                confidence_threshold=self.config['confidence_threshold'],
                classes=self.config['classes']
            )

            # Draw bounding boxes
            annotated = draw_detections(image, detections)

            # Encode annotated image
            buffer = io.BytesIO()
            annotated.save(buffer, format='JPEG', quality=80)
            annotated_bytes = buffer.getvalue()

            # Publish annotated image
            await self.nc.publish(output_annotated, annotated_bytes)

            # Publish detection metadata
            detection_msg = {
                'timestamp': datetime.utcnow().isoformat(),
                'inference_time_ms': inference_ms,
                'detections': [d.to_dict() for d in detections]
            }
            await self.nc.publish(output_detections, json.dumps(detection_msg).encode())

            # Publish heartbeat periodically
            # ...

        # Subscribe to camera frames
        await self.nc.subscribe(input_topic, cb=message_handler)

        # Keep running
        while True:
            await asyncio.sleep(1)
```

### Performance Targets

Based on [benchmarks](https://forums.raspberrypi.com/viewtopic.php?t=373867):
- **Hailo-8L throughput**: 13 TOPS
- **YOLOx-s @ 640x480**: ~30-40 FPS on Hailo-8L
- **Target latency**: <50ms per frame
- **Batch processing**: Optional batch=8 for higher throughput

---

## Part 5: Image Annotation

### Bounding Box Drawing

Using PIL (Pillow) for lightweight annotation:

```python
# annotate/draw_boxes.py

from PIL import Image, ImageDraw, ImageFont

def draw_detections(image: Image.Image, detections: list) -> Image.Image:
    """Draw bounding boxes and labels on image."""
    draw = ImageDraw.Draw(image)

    # Try to load a font, fall back to default
    try:
        font = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 16)
    except:
        font = ImageFont.load_default()

    for det in detections:
        bbox = det.bbox
        x1, y1 = bbox.x, bbox.y
        x2, y2 = x1 + bbox.width, y1 + bbox.height

        # Box color based on class
        color = get_class_color(det.class_id)

        # Draw rectangle
        draw.rectangle([x1, y1, x2, y2], outline=color, width=2)

        # Draw label background
        label = f"{det.class_name} {det.confidence:.0%}"
        text_bbox = draw.textbbox((x1, y1), label, font=font)
        draw.rectangle(
            [text_bbox[0]-2, text_bbox[1]-2, text_bbox[2]+2, text_bbox[3]+2],
            fill=color
        )

        # Draw label text
        draw.text((x1, y1), label, fill='white', font=font)

    return image

def get_class_color(class_id: int) -> str:
    """Return consistent color for each class."""
    colors = ['#FF6B6B', '#4ECDC4', '#45B7D1', '#96CEB4', '#FFEAA7']
    return colors[class_id % len(colors)]
```

---

## Part 6: Dashboard Model Outputs Tab

### New Tab: "AI / Models"

```
┌─────────────────────────────────────────────────────────────┐
│ Gorai Dashboard                                             │
├─────────────────────────────────────────────────────────────┤
│  [Status]  [Cameras]  [AI / Models]  [Logs]                │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Model: person_detector              Status: ● Running      │
│  ├── Input: main_camera                                     │
│  ├── Model: yolox_s_leaky.hef                              │
│  ├── FPS: 28.4                                             │
│  └── Inference: 35ms avg                                   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                                                      │   │
│  │     ┌──────────┐                                    │   │
│  │     │ person   │                                    │   │
│  │     │  92%     │                                    │   │
│  │     │          │     Annotated video stream         │   │
│  │     │          │     from detector.annotated        │   │
│  │     └──────────┘                                    │   │
│  │                                                      │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  Detection Log (last 10):                                   │
│  ├── 12:00:01 - 2 persons detected (confidence: 92%, 87%)  │
│  ├── 12:00:00 - 1 person detected (confidence: 95%)        │
│  └── 11:59:59 - 0 detections                               │
│                                                             │
│  Statistics (last 60s):                                     │
│  ├── Total detections: 847                                 │
│  ├── Avg per frame: 1.4                                    │
│  └── Max confidence: 98%                                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Dashboard Implementation

**New Files**:
- `/gorai/pkg/dashboard/models/handler.go` - HTTP handlers for models tab
- `/gorai/pkg/dashboard/models/monitor.go` - Track model service status
- `/gorai/pkg/dashboard/static/css/models.css` - Styles
- `/gorai/pkg/dashboard/static/js/models.js` - WebSocket for detections

**Endpoints**:
| URL | Description |
|-----|-------------|
| `/models` | Models overview page |
| `/models/{name}/stream` | MJPEG stream of annotated frames |
| `/ws/models` | WebSocket for detection events |
| `/api/models` | JSON API for model status |

**Model Status via NATS**:
Subscribe to `gorai.<robot>._system.heartbeat` and filter for `type: "service"` and `subtype: "object_detection"`.

---

## Part 7: File Structure

### New Directories

```
/gorai/
├── cmd/
│   ├── gorai-supervisor/       # NEW: Process supervisor
│   │   └── main.go
│   └── gorai-object-detection/ # NEW: Python detection service
│       ├── main.py
│       ├── inference/
│       ├── processing/
│       ├── annotate/
│       └── requirements.txt
├── pkg/
│   ├── supervisor/             # NEW: Supervisor library
│   │   ├── supervisor.go
│   │   ├── process.go
│   │   ├── health.go
│   │   └── config.go
│   └── dashboard/
│       └── models/             # NEW: Models dashboard tab
│           ├── handler.go
│           └── monitor.go
├── service/
│   └── object_detection/       # NEW: Service interface
│       └── object_detection.go
└── api/
    └── proto/
        └── gorai/
            └── detection/      # NEW: Detection messages
                └── detection.proto
```

### Python Service Structure

```
/gorai/cmd/gorai-object-detection/
├── main.py                     # Entry point
├── requirements.txt            # hailo-platform, nats-py, pillow
├── config/
│   └── settings.py             # Load from env/args
├── inference/
│   ├── __init__.py
│   ├── hailo_backend.py        # HailoRT wrapper
│   └── model_loader.py         # HEF loading
├── processing/
│   ├── __init__.py
│   ├── preprocess.py           # Image preprocessing
│   └── postprocess.py          # YOLO output decoding
├── annotate/
│   ├── __init__.py
│   └── draw_boxes.py           # Bounding box drawing
└── tests/
    └── test_inference.py
```

---

## Part 8: Implementation Order

### Phase 1: Supervisor Foundation (Week 1)

1. **Create gorai-supervisor skeleton**
   - CLI argument parsing
   - RDL loading with process config
   - Basic process start/stop

2. **Implement process management**
   - Start processes with stdout/stderr capture
   - Environment variable injection
   - PID tracking

3. **Add health monitoring**
   - NATS heartbeat subscription
   - Timeout detection
   - Restart logic

4. **Dashboard integration**
   - Supervisor status endpoint
   - Process control API

### Phase 2: Object Detection Service (Week 2)

5. **Python service skeleton**
   - NATS connection
   - Configuration loading
   - Heartbeat publishing

6. **Hailo integration**
   - HailoRT initialization
   - Model loading (HEF)
   - Basic inference

7. **YOLO postprocessing**
   - Output tensor decoding
   - Non-maximum suppression
   - Confidence filtering

8. **Image annotation**
   - Bounding box drawing
   - Label rendering
   - JPEG encoding

### Phase 3: Dashboard Models Tab (Week 3)

9. **Models tab backend**
   - Detection topic subscription
   - Model status aggregation
   - MJPEG stream for annotated frames

10. **Models tab frontend**
    - Video display with detections
    - Real-time statistics
    - Detection log

11. **Testing and optimization**
    - End-to-end testing
    - Latency optimization
    - Memory profiling

---

## Part 9: Configuration Example

### Complete RDL for Person Detection Robot

```json
{
  "version": "1",
  "robot": {
    "name": "watchbot",
    "description": "Person detection robot"
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
        "frame_rate": 30,
        "jpeg_quality": 80
      }
    }
  ],
  "services": [
    {
      "name": "person_detector",
      "type": "object_detection",
      "model": "hailo_yolox",
      "process": {
        "binary": "python3",
        "args": ["/opt/gorai/gorai-object-detection/main.py"],
        "working_dir": "/opt/gorai/gorai-object-detection",
        "env": {
          "PYTHONPATH": "/opt/gorai/gorai-object-detection"
        },
        "restart_policy": "always",
        "health_timeout_seconds": 30
      },
      "attributes": {
        "model_path": "/opt/models/yolox_s_leaky.hef",
        "input_topic": "gorai.watchbot.main_camera.data",
        "output_topic_annotated": "gorai.watchbot.person_detector.annotated",
        "output_topic_detections": "gorai.watchbot.person_detector.detections",
        "confidence_threshold": 0.5,
        "classes": ["person"],
        "batch_size": 1
      }
    }
  ],
  "dashboard": {
    "enabled": true,
    "listen": ":8080"
  }
}
```

---

## Part 10: Deployment

### Install Hailo Runtime

```bash
# On Raspberry Pi 5 with AI HAT
sudo apt update
sudo apt install hailo-all

# Verify installation
hailortcli fw-control identify
```

### Install Python Dependencies

```bash
cd /opt/gorai/gorai-object-detection
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

### requirements.txt

```
hailo-platform>=4.18.0
nats-py>=2.7.0
pillow>=10.0.0
numpy>=1.24.0
```

### Systemd Units

**gorai-supervisor.service**:
```ini
[Unit]
Description=Gorai Supervisor
After=network.target nats.service
Wants=nats.service

[Service]
Type=simple
ExecStart=/usr/local/bin/gorai-supervisor --config /opt/watchbot/watchbot.json
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

The supervisor then manages all component and service processes defined in the RDL.

---

## Summary

### Key Decisions

1. **Separate processes**: Components and services run as independent processes
2. **gorai-supervisor**: New orchestrator that reads RDL and manages processes
3. **Python for inference**: Best Hailo support, cleanly separated via NATS
4. **NATS as the bus**: All communication via NATS topics
5. **Heartbeat protocol**: Health monitoring via NATS heartbeats

### Benefits

- **Distributed robots**: Components can run on different machines
- **Fault isolation**: One service crash doesn't kill the robot
- **Language flexibility**: Go for system code, Python for ML
- **Hot updates**: Update single service without full restart
- **Resource control**: Per-process limits via cgroups

### Trade-offs

- **Complexity**: More moving parts to manage
- **Latency**: NATS serialization adds ~1-2ms per hop
- **Debugging**: Distributed logs (mitigated by log aggregation)

This architecture aligns with proven patterns from ROS2 and other robotics middleware while leveraging Gorai's NATS-based messaging backbone.

---

## References

- [Deploying YOLOX on Raspberry Pi AI Kit](https://christianjmills.com/posts/pytorch-train-object-detector-yolox-tutorial/rpi5-ai-kit-object-tracking/)
- [Hailo Application Code Examples](https://github.com/hailo-ai/Hailo-Application-Code-Examples)
- [TAPPAS Repository](https://github.com/hailo-ai/tappas)
- [ROS2 Managed Nodes](https://design.ros2.org/articles/node_lifecycle.html)
- [Hailo Community - Python Inference](https://community.hailo.ai/t/extracting-inference-results-from-a-tappas-pipeline-in-python/22)
- [Raspberry Pi Hailo Benchmarks](https://forums.raspberrypi.com/viewtopic.php?t=373867)
