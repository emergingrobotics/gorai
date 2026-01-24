# Gorai Web Dashboard Proposal

**Status**: Proposal
**Author**: AI-Assisted Design
**Date**: 2025-12-11

## Overview

This proposal describes an optional embedded web dashboard for Gorai robots. The dashboard provides real-time visualization of sensors, cameras, and ML inference results, plus a system topology view showing all components and services.

The dashboard runs as part of the main gorai binary and can be enabled or disabled via configuration.

## Goals

1. **Single binary** — No external web server or separate deployment
2. **Optional** — Disable via configuration for headless/production deployments
3. **Real-time** — Live sensor graphs, camera feeds, and inference visualization
4. **Inspectable** — View every component and service, drill into details
5. **Lightweight** — Minimal JavaScript (~65KB), fast load times
6. **No external dependencies** — No npm at build time, no CDN at runtime

## Non-Goals

- Full robot programming environment (use standard Go tooling)
- Long-term data storage (use external time-series DB if needed)
- Multi-robot fleet management (separate concern)
- User authentication (assume trusted network for MVP)

---

## Architecture

```
┌────────────────────────────────────────────────────────────────┐
│ Gorai Robot Binary                                              │
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ Dashboard Service (pkg/dashboard)                          │ │
│  │                                                             │ │
│  │  Chi Router + embed.FS (embedded static assets)           │ │
│  │  ├─ GET /              → Dashboard home (Templ)           │ │
│  │  ├─ GET /topology      → System topology view             │ │
│  │  ├─ GET /sensors       → Sensor inspection                │ │
│  │  ├─ GET /cameras       → Camera feeds                     │ │
│  │  ├─ GET /inference     → ML model outputs                 │ │
│  │  ├─ GET /ws            → WebSocket endpoint               │ │
│  │  └─ GET /api/*         → REST API                         │ │
│  │                                                             │ │
│  │  WebSocket Bridge                                          │ │
│  │  ├─ Subscribe to NATS sensor/state topics                 │ │
│  │  ├─ Downsample high-frequency data (100Hz → 10Hz)         │ │
│  │  ├─ Forward to connected browser clients                  │ │
│  │  └─ Accept commands, publish to NATS                      │ │
│  └───────────────────────────────────────────────────────────┘ │
│                              │                                   │
│                              ▼                                   │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ NATS Message Bus                                           │ │
│  │  gorai.{robot}.{component}.data                           │ │
│  │  gorai.{robot}.{component}.state                          │ │
│  │  gorai.{robot}.vision.detections                          │ │
│  │  gorai.{robot}.cmd.*                                       │ │
│  └───────────────────────────────────────────────────────────┘ │
│                              │                                   │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ Robot Components & Services                                │ │
│  │  Sensors, Actuators, Cameras, Vision, Navigation          │ │
│  └───────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────┘
                              │
                              │ HTTP/WebSocket (port 8080)
                              ▼
┌────────────────────────────────────────────────────────────────┐
│ Browser                                                         │
│                                                                  │
│  Embedded Assets (served from binary):                         │
│  ├─ HTML templates (Templ → Go → HTML)                        │
│  ├─ HTMX (14KB) — server-driven interactivity                 │
│  ├─ uPlot (50KB) — real-time sensor charts                    │
│  └─ Custom JS (~5KB) — WebSocket client, camera viewer        │
│                                                                  │
│  Total JavaScript: ~65KB (no npm, no CDN)                      │
└────────────────────────────────────────────────────────────────┘
```

---

## Technology Stack

### Backend (Go)

| Component | Library | License | Purpose |
|-----------|---------|---------|---------|
| Router | [Chi](https://github.com/go-chi/chi) | MIT | HTTP routing, middleware |
| WebSocket | [coder/websocket](https://github.com/coder/websocket) | ISC | Real-time browser connection |
| Templates | [Templ](https://github.com/a-h/templ) | MIT | Type-safe HTML generation |
| Assets | `embed.FS` (stdlib) | BSD-3 | Compile assets into binary |

### Frontend (Browser)

| Component | Library | Size | License | Purpose |
|-----------|---------|------|---------|---------|
| Interactivity | [HTMX](https://htmx.org) | 14KB | BSD-2 | Server-driven UI updates |
| Charts | [uPlot](https://github.com/leeoniya/uPlot) | 50KB | MIT | High-performance time-series |
| Styling | [PicoCSS](https://picocss.com) or custom | 10KB | MIT | Minimal CSS framework |

### Video Streaming

| Approach | Latency | Browser Support | Complexity |
|----------|---------|-----------------|------------|
| **MJPEG** (recommended for MVP) | 1-2s | All browsers | Simple |
| WebRTC via [go2rtc](https://github.com/AlexxIT/go2rtc) | <0.5s | Modern browsers | Medium |
| HLS | 5-10s | All browsers | Medium |

**Recommendation**: Start with MJPEG for simplicity, add go2rtc WebRTC option later.

---

## Configuration

```yaml
# gorai.yaml
dashboard:
  enabled: true                    # false to disable entirely
  listen: ":8080"                  # HTTP listen address

  # WebSocket settings
  websocket:
    buffer_size: 100               # Message buffer per client
    sensor_downsample_hz: 10       # Max sensor update rate to browser

  # Video streaming
  video:
    enabled: true
    format: "mjpeg"                # mjpeg, webrtc, or hls
    max_fps: 15
    quality: 80                    # JPEG quality 1-100

  # Access control (future)
  # auth:
  #   enabled: false
  #   token: ""
```

### Disabling the Dashboard

```yaml
dashboard:
  enabled: false
```

When disabled:
- No HTTP server starts
- No WebSocket connections accepted
- Zero CPU/memory overhead
- Binary size unchanged (assets still embedded, but unused)

---

## Dashboard Views

### 1. Home / System Status

**URL**: `/`

Shows:
- Robot name and uptime
- NATS connection status
- Quick stats: # components, # services, # active topics
- Recent events/errors
- Links to other views

```
┌──────────────────────────────────────────────────┐
│ 🤖 Gorai Dashboard — sentinel                    │
├──────────────────────────────────────────────────┤
│                                                   │
│  Status: ● Running        Uptime: 2h 34m        │
│  NATS: ● Connected        Topics: 23 active     │
│                                                   │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐│
│  │ Components  │ │ Services    │ │ Cameras     ││
│  │     8       │ │     3       │ │     2       ││
│  └─────────────┘ └─────────────┘ └─────────────┘│
│                                                   │
│  Recent Events                                   │
│  • 14:23:01 Vision: detected 3 objects          │
│  • 14:22:58 IMU: calibration complete           │
│  • 14:22:45 Motor: speed limit applied          │
│                                                   │
└──────────────────────────────────────────────────┘
```

### 2. System Topology

**URL**: `/topology`

Interactive tree/graph showing:
- All registered Resources (components + services)
- Parent-child relationships
- Current state (running, stopped, error)
- Click to drill into any resource

```
┌──────────────────────────────────────────────────┐
│ System Topology                      [Refresh]   │
├──────────────────────────────────────────────────┤
│                                                   │
│  sentinel (robot)                                │
│  ├─ Components                                   │
│  │  ├─ ● camera_front (sensor/camera)           │
│  │  ├─ ● camera_depth (sensor/depth)            │
│  │  ├─ ● imu (sensor/imu)                       │
│  │  ├─ ● gps (sensor/gps)                       │
│  │  ├─ ● motor_left (actuator/motor)            │
│  │  ├─ ● motor_right (actuator/motor)           │
│  │  └─ ○ gripper (actuator/gripper) [stopped]   │
│  │                                               │
│  └─ Services                                     │
│     ├─ ● vision (service/vision)                │
│     ├─ ● navigation (service/navigation)        │
│     └─ ● behavior (service/behavior)            │
│                                                   │
│  Click any item to inspect                       │
└──────────────────────────────────────────────────┘
```

### 3. Component Inspector

**URL**: `/components/{name}`

Detailed view of a single component:
- Configuration (read-only)
- Current state
- Live data stream (if sensor)
- Controls (if actuator)
- NATS topics it publishes/subscribes

```
┌──────────────────────────────────────────────────┐
│ ← Back    camera_front                           │
├──────────────────────────────────────────────────┤
│                                                   │
│  Type: sensor/camera      State: ● Running      │
│  Model: usb_camera        Uptime: 2h 34m        │
│                                                   │
│  Configuration                                   │
│  ┌────────────────────────────────────────────┐ │
│  │ device: /dev/video0                        │ │
│  │ resolution: 1280x720                       │ │
│  │ fps: 30                                    │ │
│  │ format: mjpeg                              │ │
│  └────────────────────────────────────────────┘ │
│                                                   │
│  NATS Topics                                     │
│  • Publishes: gorai.sentinel.camera_front.data  │
│  • Publishes: gorai.sentinel.camera_front.state │
│                                                   │
│  Live Preview                                    │
│  ┌────────────────────────────────────────────┐ │
│  │                                            │ │
│  │           [Camera Feed Here]               │ │
│  │                                            │ │
│  └────────────────────────────────────────────┘ │
│                                                   │
└──────────────────────────────────────────────────┘
```

### 4. Sensors View

**URL**: `/sensors`

Real-time graphs for all sensors:
- Select which sensors to display
- Configurable time window (10s, 1m, 5m)
- Auto-scaling Y axis
- Pause/resume updates

```
┌──────────────────────────────────────────────────┐
│ Sensors                    Window: [1 minute ▼] │
├──────────────────────────────────────────────────┤
│                                                   │
│  IMU Acceleration (m/s²)                        │
│  ┌────────────────────────────────────────────┐ │
│  │    ╭─╮    ╭──╮                             │ │
│  │ ───╯ ╰────╯  ╰───────────────────────────  │ │
│  │  X: 0.02   Y: -0.01   Z: 9.81              │ │
│  └────────────────────────────────────────────┘ │
│                                                   │
│  IMU Gyroscope (rad/s)                          │
│  ┌────────────────────────────────────────────┐ │
│  │ ────────────────────────────────────────── │ │
│  │  X: 0.00   Y: 0.00   Z: 0.01               │ │
│  └────────────────────────────────────────────┘ │
│                                                   │
│  GPS Position                                    │
│  ┌────────────────────────────────────────────┐ │
│  │  Lat: 37.7749°   Lon: -122.4194°          │ │
│  │  Alt: 15.2m      Sats: 8                   │ │
│  └────────────────────────────────────────────┘ │
│                                                   │
└──────────────────────────────────────────────────┘
```

### 5. Cameras View

**URL**: `/cameras`

Grid of all camera feeds:
- Live MJPEG or WebRTC streams
- Click to expand single camera
- Show frame rate and resolution
- Optional: overlay inference results

```
┌──────────────────────────────────────────────────┐
│ Cameras                                          │
├──────────────────────────────────────────────────┤
│                                                   │
│  ┌─────────────────────┐ ┌─────────────────────┐ │
│  │                     │ │                     │ │
│  │   camera_front      │ │   camera_depth      │ │
│  │   1280x720 @ 30fps  │ │   640x480 @ 15fps   │ │
│  │                     │ │                     │ │
│  │  [Live Feed]        │ │  [Depth Map]        │ │
│  │                     │ │                     │ │
│  └─────────────────────┘ └─────────────────────┘ │
│                                                   │
│  [x] Show inference overlays                    │
│                                                   │
└──────────────────────────────────────────────────┘
```

### 6. Inference View

**URL**: `/inference`

ML model outputs and performance:
- What the vision service sees
- Bounding boxes overlaid on camera feed
- Classification results with confidence
- Inference timing (ms per frame)
- Model information

```
┌──────────────────────────────────────────────────┐
│ ML Inference                                     │
├──────────────────────────────────────────────────┤
│                                                   │
│  Vision Service: ● Running                       │
│  Model: yolox-nano (ONNX)                       │
│  Device: CPU                                     │
│  Inference: 45ms avg (22 fps)                   │
│                                                   │
│  ┌────────────────────────────────────────────┐ │
│  │                                            │ │
│  │   [Camera Feed with Bounding Boxes]        │ │
│  │                                            │ │
│  │   ┌──────┐                                 │ │
│  │   │person│ 94%                             │ │
│  │   └──────┘                                 │ │
│  │              ┌────┐                        │ │
│  │              │ dog│ 87%                    │ │
│  │              └────┘                        │ │
│  │                                            │ │
│  └────────────────────────────────────────────┘ │
│                                                   │
│  Detections (last frame)                        │
│  ┌────────────────────────────────────────────┐ │
│  │ Class      Confidence   Box                │ │
│  │ person     0.94         [120,80,200,300]   │ │
│  │ dog        0.87         [400,200,150,120]  │ │
│  └────────────────────────────────────────────┘ │
│                                                   │
│  Performance History                            │
│  ┌────────────────────────────────────────────┐ │
│  │ [Graph: inference time over last minute]   │ │
│  └────────────────────────────────────────────┘ │
│                                                   │
└──────────────────────────────────────────────────┘
```

---

## Implementation Plan

### Phase 1: Foundation (MVP)

**Goal**: Basic working dashboard with system status

**Tasks**:
1. Create `pkg/dashboard` package
2. Set up Chi router with embed.FS
3. Implement WebSocket bridge to NATS
4. Create Templ templates for home page
5. Add configuration loading and enable/disable flag
6. Basic CSS styling

**Output**: Dashboard shows robot status and component list

### Phase 2: Sensor Visualization

**Goal**: Real-time sensor graphs

**Tasks**:
1. Integrate uPlot for time-series charts
2. Implement downsampling for high-frequency sensors
3. Create sensor selection UI
4. Add configurable time windows
5. WebSocket protocol for efficient data transfer

**Output**: Live updating sensor graphs

### Phase 3: Camera Integration

**Goal**: Live camera feeds in browser

**Tasks**:
1. Implement MJPEG streaming endpoint
2. Create camera grid view
3. Add single-camera expand mode
4. Optimize for bandwidth (quality settings)
5. Optional: integrate go2rtc for WebRTC

**Output**: Multiple camera feeds viewable simultaneously

### Phase 4: Inference Visualization

**Goal**: Show ML model outputs

**Tasks**:
1. Subscribe to vision service output topics
2. Overlay bounding boxes on camera feed (canvas)
3. Display classification results
4. Show inference performance metrics
5. Create inference history graph

**Output**: Visual understanding of what the robot "sees"

### Phase 5: Interactive Controls (Future)

**Goal**: Control robot from dashboard

**Tasks**:
1. Add command buttons for actuators
2. Implement joystick/gamepad input
3. Create waypoint setting UI
4. Add emergency stop button
5. Implement action progress indicators

**Output**: Full robot control interface

---

## Data Flow

### Sensor Data Path

```
Sensor Component
       │
       │ Publishes at native rate (e.g., 100Hz)
       ▼
NATS Topic: gorai.sentinel.imu.data
       │
       │ Dashboard subscribes
       ▼
WebSocket Bridge
       │
       │ Downsamples to 10Hz
       │ Buffers for batch send
       ▼
WebSocket Connection
       │
       │ JSON messages
       ▼
Browser JavaScript
       │
       │ requestAnimationFrame
       ▼
uPlot Chart Update
```

### Camera Data Path

```
Camera Component
       │
       │ Captures frame
       ▼
NATS Topic: gorai.sentinel.camera.data
       │
       │ Dashboard subscribes
       ▼
MJPEG Encoder
       │
       │ JPEG compression
       ▼
HTTP Response (multipart/x-mixed-replace)
       │
       │ Continuous stream
       ▼
Browser <img> tag (auto-updating)
```

### Command Path (Future)

```
Browser UI
       │
       │ Button click / joystick input
       ▼
WebSocket Message
       │
       │ JSON command
       ▼
WebSocket Bridge
       │
       │ Validates command
       ▼
NATS Publish: gorai.sentinel.cmd_vel
       │
       │ Routed to component
       ▼
Motor Controller
```

---

## WebSocket Protocol

### Client → Server Messages

```typescript
// Subscribe to topics
{ "type": "subscribe", "topics": ["sensors.imu", "sensors.gps"] }

// Unsubscribe
{ "type": "unsubscribe", "topics": ["sensors.imu"] }

// Send command (future)
{ "type": "command", "target": "motor_left", "action": "set_power", "value": 0.5 }
```

### Server → Client Messages

```typescript
// Sensor data
{ "type": "sensor", "name": "imu", "timestamp": 1702300000000, "data": {...} }

// State change
{ "type": "state", "component": "motor_left", "state": "running" }

// Inference result
{ "type": "inference", "model": "vision", "detections": [...] }

// Error
{ "type": "error", "message": "Component not found" }
```

---

## Performance Considerations

### Sensor Downsampling

Problem: Sensors may publish at 100Hz+, browsers can't render that fast.

Solution:
```go
type Downsampler struct {
    interval time.Duration  // e.g., 100ms for 10Hz
    last     time.Time
}

func (d *Downsampler) ShouldSend() bool {
    if time.Since(d.last) >= d.interval {
        d.last = time.Now()
        return true
    }
    return false
}
```

### Chart Rendering

uPlot performance characteristics:
- 166,650 points in 25ms initial render
- 60fps updates up to 100k visible points
- 10% CPU at 60fps with 3,600 points

Strategy:
- Keep 2-5 minutes of data in browser
- Use sliding window (drop old points)
- Batch updates with requestAnimationFrame

### Camera Bandwidth

At 720p MJPEG:
- ~50-100KB per frame at quality=80
- 30fps = 1.5-3 MB/s per camera

Mitigation:
- Reduce resolution for grid view
- Lower frame rate for non-focused cameras
- Quality slider in UI
- WebRTC for better compression (future)

### Memory Management

Browser-side:
```javascript
const MAX_POINTS = 3600;  // 2 minutes at 30Hz

function addDataPoint(data) {
    buffer.push(data);
    if (buffer.length > MAX_POINTS) {
        buffer.shift();  // Drop oldest
    }
}
```

Server-side:
- Bound WebSocket write buffer
- Drop messages if client can't keep up
- Per-client rate limiting

---

## Security Considerations

### MVP (Trusted Network)

For initial implementation, assume dashboard is only accessible on trusted network:
- No authentication
- No HTTPS (HTTP only)
- Bind to specific interface if needed

### Future Security (Optional)

If exposed to untrusted networks:

1. **Token Authentication**
   ```yaml
   dashboard:
     auth:
       enabled: true
       token: "secret-token-here"
   ```

2. **HTTPS**
   ```yaml
   dashboard:
     tls:
       cert: "/path/to/cert.pem"
       key: "/path/to/key.pem"
   ```

3. **Read-Only Mode**
   ```yaml
   dashboard:
     readonly: true  # Disable command interface
   ```

---

## File Structure

```
pkg/dashboard/
├── dashboard.go           # Main service, config loading
├── server.go              # HTTP server setup
├── websocket.go           # WebSocket handler and NATS bridge
├── handlers/
│   ├── home.go            # Home page handler
│   ├── topology.go        # Topology view handler
│   ├── sensors.go         # Sensors view handler
│   ├── cameras.go         # Camera streaming handler
│   └── inference.go       # Inference view handler
├── templates/
│   ├── layout.templ       # Base layout
│   ├── home.templ         # Home page
│   ├── topology.templ     # Topology view
│   ├── sensors.templ      # Sensors view
│   ├── cameras.templ      # Cameras view
│   └── inference.templ    # Inference view
├── static/
│   ├── js/
│   │   ├── htmx.min.js    # HTMX library
│   │   ├── uplot.min.js   # uPlot library
│   │   └── dashboard.js   # Custom WebSocket + chart code
│   └── css/
│       ├── uplot.min.css  # uPlot styles
│       └── dashboard.css  # Custom styles
└── embed.go               # embed.FS declarations
```

---

## Evaluation

### Will This Work?

**Yes**, with high confidence:

1. **Proven stack**: Chi, HTMX, uPlot are all battle-tested
2. **Successful examples**: go2rtc handles video for Home Assistant
3. **Performance validated**: uPlot benchmarks confirm real-time capability
4. **Go ecosystem mature**: embed.FS, WebSocket libraries are solid
5. **Aligned with Gorai goals**: Single binary, no npm, modular

### Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| uPlot limited chart types | Medium | Low | Use for time-series only, simple displays for other data |
| High-frequency data overload | Medium | Medium | Server-side downsampling, client-side buffering |
| Camera bandwidth on slow networks | Medium | Medium | Quality/resolution controls, WebRTC option |
| Browser compatibility | Low | Low | HTMX works everywhere, MJPEG universal |
| Template complexity | Low | Low | Templ is straightforward, good docs |

### Alternative Approaches Considered

1. **React/Vue SPA**: More powerful but requires npm, larger bundle, more complexity
2. **Grafana embedded**: Feature-rich but heavy, separate process, complex setup
3. **Terminal UI (TUI)**: Lightweight but limited visualization, no camera support
4. **gRPC-web**: More complex than WebSocket, overkill for this use case

### Conclusion

The HTMX + Templ + uPlot stack is the right choice for Gorai because:

- **Minimal JavaScript** — Aligns with Go-centric philosophy
- **Single binary** — No deployment complexity
- **Real-time capable** — uPlot handles high-frequency data
- **Extensible** — Can grow from MVP to full control interface
- **License compatible** — All MIT/BSD/ISC licenses

The phased implementation plan allows delivering value quickly (Phase 1 in days) while building toward comprehensive visualization and control.

---

## References

- [Chi Router](https://github.com/go-chi/chi)
- [Templ](https://github.com/a-h/templ)
- [HTMX](https://htmx.org)
- [uPlot](https://github.com/leeoniya/uPlot)
- [go2rtc](https://github.com/AlexxIT/go2rtc)
- [coder/websocket](https://github.com/coder/websocket)
- Detailed research: `/gorai/docs/web-dashboard-research.md`
