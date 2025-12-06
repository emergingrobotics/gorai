# Gorai-Sentinel: Distributed Architecture Options

This document evaluates distributed hardware architectures for Gorai-Sentinel, with a focus on separation of concerns, modularity, and the addition of ML inference capabilities.

---

## The Problem with Centralized Architecture

The baseline Gorai-Sentinel design runs everything on a single Raspberry Pi 5:

```
┌────────────────────────────────────────────────┐
│              Raspberry Pi 5                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐     │
│  │  Camera  │  │   ToF    │  │ Pan-Tilt │     │
│  │  (USB)   │  │  (I2C)   │  │  (I2C)   │     │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘     │
│       │             │             │            │
│  ┌────┴─────────────┴─────────────┴───────┐   │
│  │              NATS Server               │   │
│  └────────────────────────────────────────┘   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │  Fusion  │  │   ML     │  │   GUI    │    │
│  │   Node   │  │Inference │  │   Node   │    │
│  └──────────┘  └──────────┘  └──────────┘    │
└────────────────────────────────────────────────┘
```

**Problems:**

| Issue | Impact |
|-------|--------|
| **Single point of failure** | One board dying kills everything |
| **Limited modularity** | Can't swap camera without reconfiguring everything |
| **Thermal constraints** | RPi5 + AI HAT + servos = heat management nightmare |
| **I/O congestion** | USB, I2C, and NATS all competing for CPU cycles |
| **Upgrade path** | Replacing one sensor means touching the whole system |
| **Testing** | Can't develop/test camera node without full hardware stack |

---

## Option A: RPi5 + Hailo-8L AI Kit (Enhanced Centralized)

Add the [Raspberry Pi AI Kit](https://www.raspberrypi.com/products/ai-kit/) to enable on-device ML inference.

### Hardware

| Component | Part | Cost |
|-----------|------|------|
| Brain | Raspberry Pi 5 (4GB) | $60 |
| AI Accelerator | [Hailo-8L AI Kit](https://www.raspberrypi.com/documentation/accessories/ai-kit.html) | $70 |
| Camera | RPi Camera v3 (CSI) | $35 |
| ToF Sensor | VL53L5CX | $25 |
| Servos | 2x MG996R | $20 |
| Servo Controller | PCA9685 | $10 |
| **Total** | | **~$220** |

### Hailo-8L Specifications

- **Performance**: 13 TOPS (tera-operations per second)
- **Interface**: M.2 HAT+ via PCIe 2.0
- **Frameworks**: TensorFlow, TensorFlow Lite, ONNX, Keras, PyTorch
- **Power**: Low power consumption, leaves CPU free for other tasks
- **Upgrade path**: Hailo-8 (26 TOPS) available for more demanding workloads

### ML Capabilities

The Hailo-8L can run these models entirely on the NPU:
- Object detection (YOLO, SSD)
- Semantic/instance segmentation
- Pose estimation
- Face detection and landmarking

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Raspberry Pi 5                        │
│  ┌─────────────────────────────────────────────────┐    │
│  │              Hailo-8L (13 TOPS NPU)             │    │
│  │   Object Detection │ Segmentation │ Pose Est.  │    │
│  └─────────────────────────────────────────────────┘    │
│                         ▲                                │
│                         │ PCIe 2.0                       │
│  ┌──────────┐  ┌───────┴────┐  ┌──────────┐            │
│  │  Camera  │  │   Fusion   │  │ Pan-Tilt │            │
│  │  (CSI)   │──│    Node    │──│  (I2C)   │            │
│  └──────────┘  └────────────┘  └──────────┘            │
│       │              │               │                  │
│  ┌────┴──────────────┴───────────────┴─────────┐       │
│  │                 NATS Server                  │       │
│  └──────────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────────┘
```

### Pros/Cons

| Pros | Cons |
|------|------|
| Simplest to set up | Still centralized—single point of failure |
| Native rpicam-apps integration | Thermal management with HAT + servos |
| Strong ecosystem (hailo-rpi5-examples) | Limited to CSI camera (can't easily swap) |
| $70 for 13 TOPS is excellent value | USB peripherals still compete for bandwidth |

### When to Choose This

- Prototyping and development
- Cost-sensitive deployments
- When ML inference is the primary goal
- Single-unit, non-production systems

---

## Option B: Distributed Architecture with RV1106 Sensor Nodes

Separate concerns by using dedicated vision processors for sensor fusion and motor control, with the RPi5 serving purely as the NATS brain and high-level coordinator.

### The Rockchip RV1106

The [RV1106](https://wiki.luckfox.com/Luckfox-Pico-RV1106/) is a vision-focused SoC designed for IPC (IP camera) applications:

| Spec | Value |
|------|-------|
| CPU | ARM Cortex-A7 @ 1.2GHz (single core) |
| NPU | 0.5-1.0 TOPS (int4/int8/int16) |
| ISP | 5MP @ 30fps, HDR, WDR, noise reduction |
| Video | H.265/H.264 encoding, 5M @ 30fps |
| Memory | 128-256MB DDR2/DDR3L |
| Interfaces | MIPI CSI, 100M Ethernet, I2C, SPI, GPIO |
| Power | ~1W typical |

The [Luckfox Pico Ultra](https://www.luckfox.com/EN-Luckfox-Pico-Ultra) variant adds:
- **PoE support** (power + data over single Ethernet cable)
- WiFi 6 option
- 8GB eMMC storage
- **Price: $18-30** depending on options

### Hardware Bill of Materials

| Component | Part | Cost | Notes |
|-----------|------|------|-------|
| **Brain** | Raspberry Pi 5 (4GB) | $60 | NATS server, high-level control |
| **AI Accelerator** | Hailo-8L AI Kit | $70 | Optional, for heavy ML workloads |
| **Sensor Node** | Luckfox Pico Ultra (PoE) | $30 | Camera + ToF + local processing |
| **Motor Node** | Luckfox Pico Pro | $20 | Pan-tilt control, real-time loop |
| **Camera** | SC3336 3MP (MIPI CSI) | $15 | Direct to RV1106 ISP |
| **ToF Sensor** | VL53L5CX | $25 | I2C to sensor node |
| **Servos** | 2x MG996R | $20 | Controlled by motor node |
| **PoE Switch** | 5-port Gigabit PoE | $50-80 | Powers sensor nodes |
| **Total** | | **~$290-350** |

### PoE Switch Options

| Switch | Ports | PoE Budget | Price | Notes |
|--------|-------|------------|-------|-------|
| [BotBlox SwitchBlox](https://botblox.io/products/small-ethernet-switch) | 5 | - | $75 | Tiny (44.5mm²), robotics-focused |
| [TRENDnet TI-PG50](https://www.trendnet.com/products/industrial-unmanaged-poe-switch/5-port-industrial-fast-ethernet-PoEplus-din-rail-switch-TI-PE50) | 5 | 120W | $130 | Industrial, DIN-rail, -40°C to 75°C |
| Generic 5-port PoE+ | 5 | 60W | $40-60 | Consumer grade |

### Distributed Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          PoE Ethernet Switch                         │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐        │
│  │     Port 1      │ │     Port 2      │ │     Port 3      │        │
│  │   (Data only)   │ │   (PoE + Data)  │ │   (PoE + Data)  │        │
│  └────────┬────────┘ └────────┬────────┘ └────────┬────────┘        │
└───────────┼───────────────────┼───────────────────┼─────────────────┘
            │                   │                   │
            ▼                   ▼                   ▼
┌───────────────────┐ ┌─────────────────────┐ ┌─────────────────────┐
│   Raspberry Pi 5  │ │  Luckfox Pico Ultra │ │  Luckfox Pico Pro   │
│                   │ │   (Sensor Node)     │ │   (Motor Node)      │
│  ┌─────────────┐  │ │                     │ │                     │
│  │ NATS Server │  │ │  ┌─────────────┐    │ │  ┌─────────────┐    │
│  └─────────────┘  │ │  │ Camera Node │    │ │  │ PanTilt Node│    │
│  ┌─────────────┐  │ │  │ (RV1106 ISP)│    │ │  │ (Real-time) │    │
│  │ Hailo-8L    │  │ │  └─────────────┘    │ │  └─────────────┘    │
│  │ (optional)  │  │ │  ┌─────────────┐    │ │  ┌─────────────┐    │
│  └─────────────┘  │ │  │  ToF Node   │    │ │  │   PCA9685   │    │
│  ┌─────────────┐  │ │  │  (I2C)      │    │ │  │   Servos    │    │
│  │ Coordinator │  │ │  └─────────────┘    │ │  └─────────────┘    │
│  │    Node     │  │ │  ┌─────────────┐    │ │                     │
│  └─────────────┘  │ │  │ Fusion Node │    │ │  Powered by PoE     │
│  ┌─────────────┐  │ │  │ (RV1106 NPU)│    │ │                     │
│  │    GUI      │  │ │  └─────────────┘    │ └─────────────────────┘
│  └─────────────┘  │ │                     │
│                   │ │  Powered by PoE     │
└───────────────────┘ └─────────────────────┘
```

### NATS Topic Flow

```
Sensor Node publishes:
  gorai.sentinel.camera.image          → 30 Hz, H.264 compressed via RV1106
  gorai.sentinel.camera.image/raw      → On-demand raw frames
  gorai.sentinel.tof.depth_grid        → 15 Hz
  gorai.sentinel.fusion.detections     → Object detection results (RV1106 NPU)

Motor Node subscribes:
  gorai.sentinel.pantilt.command       → Position commands
  gorai.sentinel.pantilt.track         → Track detection (from coordinator)

Motor Node publishes:
  gorai.sentinel.pantilt.state         → 50 Hz position feedback

Coordinator (RPi5) subscribes to all, publishes:
  gorai.sentinel.planning.target       → High-level goals
  gorai.sentinel.ml.inference          → Heavy ML results (via Hailo-8L)
```

### Node Distribution Strategy

| Node | Runs On | Rationale |
|------|---------|-----------|
| **Camera Node** | RV1106 Sensor Node | Native ISP, H.264 encode in hardware |
| **ToF Node** | RV1106 Sensor Node | Colocated with camera for fusion |
| **Fusion Node** | RV1106 Sensor Node | 0.5-1 TOPS NPU handles lightweight fusion |
| **PanTilt Node** | RV1106 Motor Node | Dedicated real-time control loop |
| **NATS Server** | RPi5 | Central messaging hub |
| **Heavy ML** | RPi5 + Hailo-8L | 13 TOPS for complex models |
| **Coordinator** | RPi5 | High-level planning, state machines |
| **GUI/Viz** | RPi5 | Display output |

### Separation of Concerns

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Concern Boundaries                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  PERCEPTION (Sensor Node)          ACTUATION (Motor Node)           │
│  ┌─────────────────────────┐      ┌─────────────────────────┐       │
│  │ • Image acquisition     │      │ • Servo control         │       │
│  │ • Depth sensing         │      │ • Position feedback     │       │
│  │ • Hardware encoding     │      │ • Motion profiles       │       │
│  │ • Lightweight detection │      │ • Limit enforcement     │       │
│  │ • Sensor fusion         │      │ • Real-time loop (50Hz) │       │
│  └─────────────────────────┘      └─────────────────────────┘       │
│           │                                  ▲                       │
│           │ Ethernet (NATS)                  │ Ethernet (NATS)       │
│           ▼                                  │                       │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    COORDINATION (RPi5)                       │    │
│  │  • NATS message routing                                     │    │
│  │  • Heavy ML inference (Hailo-8L)                            │    │
│  │  • State machine / behavior logic                           │    │
│  │  • User interface                                           │    │
│  │  • Logging / recording                                      │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Pros/Cons

| Pros | Cons |
|------|------|
| True modularity—swap nodes independently | More complex setup |
| Failure isolation—sensor node crash doesn't kill motor control | Network latency (typically <1ms on LAN) |
| Thermal distribution—no heat concentration | More hardware to manage |
| Parallel development—teams can work on nodes independently | Higher BOM cost (~$70-130 more) |
| Natural scaling—add more sensor nodes easily | PoE switch required |
| Clean separation of real-time (motor) vs. soft real-time (vision) | |

### When to Choose This

- Production systems where reliability matters
- Multi-sensor configurations (multiple camera angles)
- When developing with a team
- Systems that need to scale
- When thermal management is critical

---

## Option C: Hybrid (Recommended Starting Point)

Start with Option A for initial development, then migrate to Option B as the system matures.

### Phase 1: Centralized Development

```
Week 1-8: Everything on RPi5 + Hailo-8L
- Prove out NATS messaging patterns
- Develop node interfaces
- Validate ML models on Hailo-8L
- Get fusion working
```

### Phase 2: Extract Sensor Node

```
Week 9-12: Add Luckfox Pico Ultra for camera/ToF
- Port camera node to RV1106
- Port ToF node to RV1106
- Keep fusion on RPi5 initially
- Validate distributed messaging
```

### Phase 3: Extract Motor Node

```
Week 13-16: Add Luckfox Pico Pro for pan-tilt
- Port PanTilt node to dedicated board
- Optimize real-time control loop
- Move fusion to sensor node
- RPi5 becomes pure coordinator
```

### Migration Path

The Gorai node abstraction enables this migration:

```go
// Node doesn't care where NATS server is
n, _ := node.New("camera_driver",
    node.WithNATS("nats://192.168.1.100:4222"),  // RPi5 IP
    node.WithNamespace("sentinel"),
)

// Same code runs on RPi5 or RV1106
// Only difference is the NATS server address
```

---

## Option D: Minimal RV1106 Head Unit + Ainstein Radar

A streamlined architecture where a single RV1106 board serves as the "head unit" handling video streaming, pan-tilt control, and radar data acquisition. All compute-intensive work (fusion, ML) happens on the RPi5.

### Design Philosophy

This option prioritizes:
- **Simplicity**: One embedded board for all head-mounted hardware
- **Compute centralization**: RPi5 handles all "thinking"
- **Radar over ToF**: Weather-resistant ranging with longer range
- **Minimal edge processing**: RV1106 just streams data, doesn't process it

### Ainstein US-D1 Radar Specifications

The [Ainstein US-D1](https://ainstein.ai/us-d1/) is a 24 GHz radar altimeter designed for UAVs:

| Spec | Value |
|------|-------|
| **Frequency** | 24 GHz |
| **Range** | 0.5m - 50m |
| **Precision** | 4-6 cm |
| **Field of View** | 43° × 30° |
| **Update Rate** | 100 Hz |
| **Interface** | UART or CAN |
| **Power** | 2W @ 5V |
| **Size** | 108 × 79 × 20 mm |
| **Weight** | 110g |
| **Environmental** | IP67 (with sealant), -20°C to 65°C |

**Advantages over VL53L5CX ToF:**

| Aspect | US-D1 Radar | VL53L5CX ToF |
|--------|-------------|--------------|
| Range | 0.5-50m | 0.2-4m |
| Weather | All-weather, works in rain/fog/dust | Degrades in adverse conditions |
| Sunlight | Immune | Can be affected by bright IR |
| Resolution | Single point | 8×8 grid |
| Update rate | 100 Hz | 15-60 Hz |
| FoV | 43° × 30° | 45° × 45° |
| Price | ~$200-300 | ~$25 |
| Power | 2W | 0.2W |

The radar trades spatial resolution for range, weather immunity, and reliability.

### Hardware Bill of Materials

| Component | Part | Cost | Notes |
|-----------|------|------|-------|
| **Brain** | Raspberry Pi 5 (4GB) | $60 | NATS, fusion, ML |
| **AI Accelerator** | Hailo-8L AI Kit | $70 | Object detection, segmentation |
| **Head Unit** | Luckfox Pico Ultra (PoE) | $30 | Camera + radar + pan-tilt control |
| **Camera** | SC3336 3MP MIPI | $15 | Direct to RV1106 ISP |
| **Radar** | Ainstein US-D1 | $250 | UART to RV1106 |
| **Servos** | 2× MG996R | $20 | Controlled by RV1106 |
| **Servo Driver** | PCA9685 | $10 | I2C from RV1106 |
| **PoE Injector** | Single-port PoE+ | $15 | Powers head unit |
| **Total** | | **~$470** |

### Architecture

```
                                    ┌──────────────────────────┐
                                    │     Raspberry Pi 5       │
                                    │                          │
                                    │  ┌────────────────────┐  │
                                    │  │    NATS Server     │  │
                                    │  └────────────────────┘  │
                                    │  ┌────────────────────┐  │
                                    │  │    Hailo-8L NPU    │  │
                                    │  │  (13 TOPS for ML)  │  │
                                    │  └────────────────────┘  │
                                    │  ┌────────────────────┐  │
                                    │  │   Fusion Node      │  │
                                    │  │ (RGB + Radar)      │  │
                                    │  └────────────────────┘  │
                                    │  ┌────────────────────┐  │
                                    │  │   Coordinator      │  │
                                    │  │ (Tracking, State)  │  │
                                    │  └────────────────────┘  │
                                    └────────────┬─────────────┘
                                                 │
                                                 │ Ethernet (PoE)
                                                 │
┌────────────────────────────────────────────────┴────────────────────────────┐
│                        RV1106 Head Unit (Luckfox Pico Ultra)                 │
│                                                                              │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐  │
│   │   Camera    │    │   Radar     │    │  Pan-Tilt   │    │    NATS     │  │
│   │    Node     │    │    Node     │    │    Node     │    │   Client    │  │
│   └──────┬──────┘    └──────┬──────┘    └──────┬──────┘    └─────────────┘  │
│          │                  │                  │                             │
│   ┌──────┴──────┐    ┌──────┴──────┐    ┌──────┴──────┐                     │
│   │  SC3336     │    │  US-D1      │    │  PCA9685    │                     │
│   │  (MIPI CSI) │    │  (UART)     │    │  (I2C)      │                     │
│   └─────────────┘    └─────────────┘    └─────────────┘                     │
│                                                                              │
│   Powered by PoE                                                            │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Data Flow

```
RV1106 Head Unit                          RPi5 Brain
────────────────                          ──────────

Camera Node
  │
  ├─► H.264 encode (RV1106 VPU)
  │     720p @ 30fps, ~3 Mbps
  │
  └─────────────────────────────────────► gorai.sentinel.camera.h264
                                              │
                                              ▼
                                          Decode (CPU/VPU)
                                              │
                                              ▼
                                          Hailo-8L NPU
                                          ├─► Object Detection
                                          ├─► Pose Estimation
                                          └─► Segmentation
                                              │
                                              ▼
                                          gorai.sentinel.ml.detections

Radar Node
  │
  ├─► Parse UART @ 100 Hz
  │     Range + velocity data
  │
  └─────────────────────────────────────► gorai.sentinel.radar.range
                                              │
                                              ▼
                                          Fusion Node
                                          ├─► Associate detections with range
                                          ├─► Track objects in 3D
                                          └─► Estimate velocities
                                              │
                                              ▼
                                          gorai.sentinel.fusion.tracks

PanTilt Node ◄───────────────────────────── gorai.sentinel.pantilt.command
  │
  ├─► Servo control @ 50 Hz
  │
  └─────────────────────────────────────► gorai.sentinel.pantilt.state
```

### Message Types

```protobuf
// radar.proto
syntax = "proto3";
package gorai.sensor;

import "std.proto";

message RadarRange {
    gorai.std.Header header = 1;
    float range = 2;           // meters
    float velocity = 3;        // m/s (if supported)
    float snr = 4;             // signal-to-noise ratio
    uint32 status = 5;         // sensor status flags
}

message RadarTrack {
    gorai.std.Header header = 1;
    uint32 track_id = 2;
    float range = 3;
    float azimuth = 4;         // radians
    float elevation = 5;       // radians
    float velocity = 6;
    float confidence = 7;
}
```

### RV1106 Node Implementation

The head unit runs three lightweight nodes:

```go
// cmd/sentinel/head/main.go
package main

func main() {
    // Connect to NATS on RPi5
    n, _ := node.New("head_unit",
        node.WithNATS("nats://192.168.1.100:4222"),
        node.WithNamespace("sentinel"),
    )
    defer n.Close()

    // Start all head unit nodes concurrently
    g, ctx := errgroup.WithContext(context.Background())

    g.Go(func() error { return runCameraNode(ctx, n) })
    g.Go(func() error { return runRadarNode(ctx, n) })
    g.Go(func() error { return runPanTiltNode(ctx, n) })

    if err := g.Wait(); err != nil {
        log.Fatal(err)
    }
}
```

**Camera Node** (streams H.264 over NATS):

```go
func runCameraNode(ctx context.Context, n *node.Node) error {
    // RV1106 ISP + VPU handles capture and encode
    cam, _ := rv1106.OpenCamera("/dev/video0", rv1106.Config{
        Width:   1280,
        Height:  720,
        FPS:     30,
        Codec:   rv1106.H264,
        Bitrate: 3_000_000, // 3 Mbps
    })
    defer cam.Close()

    pub := pub.New[sensor.CompressedImage](n, "camera.h264")

    for frame := range cam.Frames(ctx) {
        pub.Publish(ctx, &sensor.CompressedImage{
            Header: std.NewHeaderAt("camera_optical", frame.Timestamp),
            Format: "h264",
            Data:   frame.Data,
        })
    }
    return nil
}
```

**Radar Node** (parses US-D1 UART):

```go
func runRadarNode(ctx context.Context, n *node.Node) error {
    radar, _ := usd1.Open("/dev/ttyS1", usd1.Config{
        BaudRate: 115200,
    })
    defer radar.Close()

    pub := pub.New[sensor.RadarRange](n, "radar.range")

    for {
        select {
        case <-ctx.Done():
            return nil
        default:
            reading, _ := radar.Read()
            pub.Publish(ctx, &sensor.RadarRange{
                Header:   std.NewHeader("radar_link"),
                Range:    reading.Range,
                Velocity: reading.Velocity,
                SNR:      reading.SNR,
                Status:   reading.Status,
            })
        }
    }
}
```

**PanTilt Node** (same as before, runs locally):

```go
func runPanTiltNode(ctx context.Context, n *node.Node) error {
    pca, _ := pca9685.Open(1, 0x40) // I2C bus 1, addr 0x40
    defer pca.Close()

    statePub := pub.New[control.PanTiltState](n, "pantilt.state")

    sub.New[control.PanTiltCommand](n, "pantilt.command", func(cmd *control.PanTiltCommand) {
        pca.SetServo(0, cmd.Pan)
        pca.SetServo(1, cmd.Tilt)
    })

    // Publish state at 50 Hz
    ticker := time.NewTicker(20 * time.Millisecond)
    for {
        select {
        case <-ctx.Done():
            return nil
        case <-ticker.C:
            statePub.Publish(ctx, &control.PanTiltState{
                Header: std.NewHeader("tilt_link"),
                Pan:    pca.GetPosition(0),
                Tilt:   pca.GetPosition(1),
            })
        }
    }
}
```

### Fusion Strategy

With radar providing only a single range value (not a depth grid), fusion works differently:

```go
// Fusion on RPi5 combines ML detections with radar range
type FusionNode struct {
    detections chan *ml.Detections   // from Hailo-8L
    radar      chan *sensor.RadarRange
    pantilt    chan *control.PanTiltState
}

func (f *FusionNode) fuse(ctx context.Context) {
    for {
        select {
        case det := <-f.detections:
            // Find detection closest to radar beam center
            // Radar FoV is 43°×30°, centered on optical axis
            centerDet := findCenterDetection(det.Objects, radarFoV)
            if centerDet != nil && f.lastRadar != nil {
                // Assign radar range to center detection
                centerDet.Distance = f.lastRadar.Range
                centerDet.Velocity = f.lastRadar.Velocity
            }
            f.publishTracks(det)

        case r := <-f.radar:
            f.lastRadar = r
        }
    }
}
```

**Fusion approach:**
1. ML detects objects in camera frame (bounding boxes)
2. Radar provides range to whatever is at beam center
3. Associate radar range with detection nearest beam center
4. Use pan-tilt position to project into world coordinates
5. Track objects over time with Kalman filter

### Network Bandwidth

| Stream | Rate | Size | Bandwidth |
|--------|------|------|-----------|
| H.264 video | 30 fps | ~12 KB/frame | ~3 Mbps |
| Radar range | 100 Hz | 24 bytes | ~19 Kbps |
| PanTilt state | 50 Hz | 48 bytes | ~19 Kbps |
| PanTilt command | 50 Hz | 32 bytes | ~13 Kbps |
| ML detections | 30 Hz | ~1 KB | ~240 Kbps |
| **Total** | | | **~3.5 Mbps** |

100M Ethernet has massive headroom.

### Pros/Cons

| Pros | Cons |
|------|------|
| Simplest distributed setup—one Ethernet cable | Higher cost (~$250 for radar vs $25 for ToF) |
| All-weather operation (rain, fog, dust) | Single-point ranging (no depth grid) |
| 50m range vs 4m for ToF | Larger/heavier sensor |
| 100 Hz update rate | Wider beam = less precise angular resolution |
| RV1106 does minimal work—just stream | Requires pan-tilt scanning to build depth map |
| Fusion/ML centralized = easier debugging | |
| PoE = single cable for power + data | |

### When to Choose This

- Outdoor deployments where weather matters
- Applications needing >4m range
- When simplicity of edge processing is valued
- Security/surveillance applications
- When budget allows for radar ($250 premium)

### Scanning Mode for Depth Mapping

Since radar gives single-point range, use pan-tilt to build a depth map:

```go
// Scan pattern to build sparse depth map
type ScanMapper struct {
    panRange   [2]float64  // min, max radians
    tiltRange  [2]float64
    step       float64     // angular step
    dwellTime  time.Duration
    depthMap   map[GridCell]float64
}

func (s *ScanMapper) runScan(ctx context.Context, pantilt *action.Client, radar <-chan *sensor.RadarRange) {
    for pan := s.panRange[0]; pan <= s.panRange[1]; pan += s.step {
        for tilt := s.tiltRange[0]; tilt <= s.tiltRange[1]; tilt += s.step {
            // Move to position
            pantilt.SendGoal(ctx, &MoveToGoal{Pan: pan, Tilt: tilt})

            // Dwell and collect radar samples
            samples := collectSamples(ctx, radar, s.dwellTime)
            avgRange := average(samples)

            // Store in depth map
            s.depthMap[GridCell{pan, tilt}] = avgRange
        }
    }
}
```

A 90°×45° scan at 5° steps = 18×9 = 162 points. At 100ms dwell time = ~16 seconds for full scan.

---

## Hardware Considerations

### RV1106 vs RV1108

| Spec | RV1106 | RV1108 |
|------|--------|--------|
| CPU | Cortex-A7 @ 1.2GHz | Cortex-A7 @ 1.2GHz |
| NPU | 0.5-1.0 TOPS | None |
| ISP | 5MP, HDR, WDR | 8MP |
| Video Codec | H.265/H.264 | H.264 only |
| Availability | Current, well-supported | Legacy |
| **Recommendation** | **Use RV1106** | Avoid |

### Luckfox Pico Variants

| Model | Memory | NPU | Ethernet | PoE | Price |
|-------|--------|-----|----------|-----|-------|
| Pico | 64MB | 0.5T | No | No | $9 |
| Pico Pro | 128MB | 0.5T | 100M | No | $15 |
| Pico Max | 256MB | 0.5T | 100M | No | $20 |
| Pico Ultra | 256MB | 0.5-1.0T | 100M | Optional | $18-30 |
| **Pico Ultra W PoE** | 256MB | 1.0T | 100M | **Yes** | **$30** |

**Recommendation**: Luckfox Pico Ultra with PoE module for sensor node, Pico Pro for motor node (doesn't need PoE if colocated with power).

### Network Bandwidth

Verify that 100M Ethernet is sufficient:

| Stream | Bandwidth |
|--------|-----------|
| H.264 1080p @ 30fps | ~4-8 Mbps |
| H.264 720p @ 30fps | ~2-4 Mbps |
| ToF depth grid (8x8 float32 @ 15Hz) | ~8 Kbps |
| PanTilt state (2 float64 @ 50Hz) | ~6 Kbps |
| **Total** | **<15 Mbps** |

100M Ethernet provides ~80 Mbps usable throughput—plenty of headroom.

---

## Software Architecture for Distribution

### Node Discovery

Use NATS service discovery instead of hardcoded addresses:

```go
// Sensor node registers itself
n.RegisterService("sentinel.sensor", ServiceInfo{
    Node:     "camera_driver",
    Hostname: hostname,
    Topics:   []string{"camera.image", "tof.depth_grid"},
})

// Coordinator discovers available sensors
services := n.DiscoverServices("sentinel.sensor")
for _, svc := range services {
    log.Printf("Found sensor: %s at %s", svc.Node, svc.Hostname)
}
```

### Health Monitoring

Each node publishes heartbeats:

```go
// Every node publishes health
ticker := time.NewTicker(time.Second)
for range ticker.C {
    n.Publish("health", &Health{
        Node:      n.Name(),
        Status:    "ok",
        CPU:       getCPUUsage(),
        Memory:    getMemUsage(),
        Uptime:    time.Since(startTime),
    })
}
```

Coordinator monitors and alerts on missing heartbeats.

### Graceful Degradation

When sensor node fails, coordinator should:
1. Detect missing heartbeat
2. Stop sending tracking commands to motor node
3. Move pan-tilt to safe position
4. Alert operator
5. Attempt reconnection

```go
func (c *Coordinator) monitorSensors(ctx context.Context) {
    timeout := 3 * time.Second
    for {
        select {
        case <-ctx.Done():
            return
        case <-time.After(timeout):
            if time.Since(c.lastSensorHeartbeat) > timeout {
                c.handleSensorFailure()
            }
        }
    }
}
```

---

## Recommendations

### For Development/Prototyping
**Option A: RPi5 + Hailo-8L**
- Fastest path to working system
- Easiest debugging (everything local)
- ~$220 total cost

### For Production/Scaling
**Option B: Distributed with RV1106**
- True separation of concerns
- Failure isolation
- ~$290-350 total cost

### For Long-Term
**Option C: Hybrid Migration**
- Start simple, add complexity as needed
- Proves architecture before investing in hardware
- Final cost same as Option B

### For Outdoor/All-Weather
**Option D: RV1106 Head Unit + Radar**
- Weather-immune ranging (rain, fog, dust)
- 50m range for outdoor applications
- Simplest distributed architecture (single Ethernet cable)
- ~$470 total cost (radar adds ~$250)

---

## Next Steps

1. **Validate RV1106 toolchain**: Ensure Go cross-compiles cleanly for ARM Cortex-A7
2. **Test NATS on RV1106**: Verify NATS client runs on 256MB RAM
3. **Benchmark network latency**: Measure actual pub/sub latency over Ethernet
4. **Prototype camera streaming**: Test H.264 encoding performance on RV1106
5. **Evaluate PoE switch options**: Determine if consumer-grade is sufficient

---

## References

- [Raspberry Pi AI Kit Documentation](https://www.raspberrypi.com/documentation/accessories/ai-kit.html)
- [Hailo RPi5 Examples](https://github.com/hailo-ai/hailo-rpi5-examples)
- [Luckfox Pico Wiki](https://wiki.luckfox.com/Luckfox-Pico-RV1106/)
- [Luckfox Pico Ultra](https://www.luckfox.com/EN-Luckfox-Pico-Ultra)
- [Rockchip RV1106 Datasheet](https://rockchips.net/product/rv1106/)
- [BotBlox SwitchBlox](https://botblox.io/products/small-ethernet-switch)
- [TRENDnet Industrial PoE Switches](https://www.trendnet.com/products/industrial-unmanaged-poe-switch/5-port-industrial-fast-ethernet-PoEplus-din-rail-switch-TI-PE50)
- [Ainstein US-D1 Radar Altimeter](https://ainstein.ai/us-d1/)
