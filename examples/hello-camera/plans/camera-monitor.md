# Camera Monitor Design for Gorai Dashboard

**Date**: 2025-12-13
**Status**: Design
**Author**: AI-assisted design

## Overview

The camera monitor is a built-in dashboard tab that displays live video feeds from cameras connected to the robot. It subscribes to camera frame topics on NATS and renders them in the browser with minimal latency.

## Goals

1. Display live camera feeds from all configured cameras
2. Support multiple cameras in a responsive grid layout
3. Provide frame rate and quality controls
4. Minimize latency (<500ms target)
5. Handle camera disconnect/reconnect gracefully
6. Keep JavaScript footprint minimal (aligned with HTMX + Templ stack)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Browser - Camera Monitor Tab                                │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Camera Grid (HTMX-driven)                            │  │
│  │                                                       │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │  │
│  │  │  Camera 1   │  │  Camera 2   │  │  Camera 3   │  │  │
│  │  │  <img src=  │  │  <img src=  │  │  <img src=  │  │  │
│  │  │  /stream/>  │  │  /stream/>  │  │  /stream/>  │  │  │
│  │  │  30 fps     │  │  15 fps     │  │  offline    │  │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  │  │
│  │                                                       │  │
│  │  Controls: [Pause All] [Grid 2x2 ▼] [Quality 80% ▼]  │  │
│  └──────────────────────────────────────────────────────┘  │
│                         │                                   │
└─────────────────────────┼───────────────────────────────────┘
                          │ HTTP (MJPEG stream)
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ Gorai Dashboard Server                                      │
│                                                              │
│  /cameras                    → Camera list (HTMX partial)   │
│  /cameras/{name}/stream      → MJPEG stream endpoint        │
│  /cameras/{name}/snapshot    → Single JPEG frame            │
│  /ws/cameras                 → WebSocket for status updates │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Camera Stream Handler                                 │  │
│  │                                                        │  │
│  │  - Subscribes to gorai.<robot>.<camera>.data          │  │
│  │  - Converts NATS messages to MJPEG boundary stream    │  │
│  │  - Handles client disconnect/reconnect                 │  │
│  │  - Rate limiting per client                           │  │
│  └──────────────────────────────────────────────────────┘  │
│                         │                                   │
└─────────────────────────┼───────────────────────────────────┘
                          │ NATS Subscribe
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ NATS                                                        │
│                                                              │
│  Topic: gorai.hello-camera.front_camera.data                │
│  Payload: Raw JPEG bytes (per frame)                        │
│                                                              │
└─────────────────────────────────────────────────────────────┘
                          ▲
                          │ Publish
┌─────────────────────────┼───────────────────────────────────┐
│ V4L2 Camera Driver                                          │
│                                                              │
│  - Captures frames from /dev/video0                         │
│  - Encodes to JPEG (if YUYV) or passes through (if MJPEG)  │
│  - Publishes to NATS at configured frame rate              │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Streaming Strategy: NATS-to-MJPEG Bridge

### Why MJPEG (not WebRTC)

1. **Simplicity**: MJPEG works with a single `<img>` tag, no JavaScript required
2. **Universal support**: Works in all browsers including mobile Safari
3. **Lower complexity**: No ICE/STUN/TURN negotiation needed
4. **Sufficient for monitoring**: 1-2s latency is acceptable for dashboard viewing
5. **Frame-by-frame**: Individual frames already available from camera driver

### Why not go2rtc

While go2rtc offers WebRTC with lower latency, for the initial implementation:
- Frames are already JPEG on NATS topics
- go2rtc would require re-encoding or a separate capture pipeline
- MJPEG is simpler to implement and debug
- go2rtc can be added later as an optional high-performance mode

## API Endpoints

### GET /cameras

Returns HTML partial listing all configured cameras with their status.

**Response** (HTMX partial):
```html
<div id="camera-grid" class="camera-grid grid-2x2">
  <div class="camera-card" data-camera="front_camera">
    <div class="camera-header">
      <span class="camera-name">front_camera</span>
      <span class="camera-status online">30 fps</span>
    </div>
    <img src="/cameras/front_camera/stream"
         alt="front_camera"
         loading="lazy"
         onerror="this.src='/static/camera-offline.svg'">
    <div class="camera-controls">
      <button hx-post="/cameras/front_camera/pause"
              hx-swap="none">Pause</button>
      <a href="/cameras/front_camera/snapshot"
         download="front_camera.jpg">Snapshot</a>
    </div>
  </div>
  <!-- More camera cards... -->
</div>
```

### GET /cameras/{name}/stream

MJPEG stream endpoint that bridges NATS frames to HTTP.

**Content-Type**: `multipart/x-mixed-replace; boundary=frame`

**Implementation**:
```go
func (s *DashboardServer) handleCameraStream(w http.ResponseWriter, r *http.Request) {
    cameraName := chi.URLParam(r, "name")
    topic := s.topics.ComponentData(cameraName)

    w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
    w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
    w.Header().Set("Connection", "close")

    // Subscribe to camera frames
    sub, err := s.nats.Subscribe(topic, func(msg *nats.Msg) {
        fmt.Fprintf(w, "--frame\r\n")
        fmt.Fprintf(w, "Content-Type: image/jpeg\r\n")
        fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(msg.Data))
        w.Write(msg.Data)
        fmt.Fprintf(w, "\r\n")

        if f, ok := w.(http.Flusher); ok {
            f.Flush()
        }
    })
    if err != nil {
        http.Error(w, "Failed to subscribe", http.StatusInternalServerError)
        return
    }
    defer sub.Unsubscribe()

    // Wait for client disconnect
    <-r.Context().Done()
}
```

### GET /cameras/{name}/snapshot

Returns a single JPEG frame (latest available).

**Content-Type**: `image/jpeg`

### GET /ws/cameras

WebSocket endpoint for camera status updates (online/offline, fps, errors).

**Message Format** (JSON):
```json
{
  "type": "camera_status",
  "camera": "front_camera",
  "online": true,
  "fps": 28.5,
  "resolution": "640x480",
  "last_frame": "2025-12-13T12:30:00Z"
}
```

## Frontend Components

### Templ Template: cameras.templ

```go
package templates

templ CamerasPage(cameras []CameraInfo) {
    <div id="cameras-tab" class="tab-content">
        <div class="cameras-toolbar">
            <button hx-post="/cameras/pause-all"
                    hx-swap="none"
                    class="btn">
                Pause All
            </button>
            <select hx-get="/cameras"
                    hx-trigger="change"
                    hx-target="#camera-grid"
                    name="layout">
                <option value="1x1">1x1</option>
                <option value="2x2" selected>2x2</option>
                <option value="3x3">3x3</option>
                <option value="auto">Auto</option>
            </select>
            <select hx-post="/cameras/quality"
                    hx-swap="none"
                    name="quality">
                <option value="50">Low (50%)</option>
                <option value="80" selected>Medium (80%)</option>
                <option value="95">High (95%)</option>
            </select>
        </div>

        <div id="camera-grid"
             class="camera-grid"
             hx-get="/cameras"
             hx-trigger="every 10s"
             hx-swap="outerHTML">
            for _, cam := range cameras {
                @CameraCard(cam)
            }
        </div>
    </div>
}

templ CameraCard(cam CameraInfo) {
    <div class="camera-card" data-camera={ cam.Name }>
        <div class="camera-header">
            <span class="camera-name">{ cam.Name }</span>
            if cam.Online {
                <span class="camera-status online">{ cam.FPS } fps</span>
            } else {
                <span class="camera-status offline">offline</span>
            }
        </div>
        if cam.Online {
            <img src={ "/cameras/" + cam.Name + "/stream" }
                 alt={ cam.Name }
                 loading="lazy"
                 class="camera-feed"/>
        } else {
            <div class="camera-offline">
                <svg><!-- offline icon --></svg>
                <p>Camera offline</p>
            </div>
        }
        <div class="camera-controls">
            <button onclick={ "toggleStream('" + cam.Name + "')" }>
                Pause
            </button>
            <a href={ "/cameras/" + cam.Name + "/snapshot" }
               download={ cam.Name + ".jpg" }>
                Snapshot
            </a>
        </div>
    </div>
}
```

### CSS: cameras.css

```css
.camera-grid {
    display: grid;
    gap: 1rem;
    padding: 1rem;
}

.camera-grid.grid-1x1 { grid-template-columns: 1fr; }
.camera-grid.grid-2x2 { grid-template-columns: repeat(2, 1fr); }
.camera-grid.grid-3x3 { grid-template-columns: repeat(3, 1fr); }

@media (max-width: 768px) {
    .camera-grid { grid-template-columns: 1fr !important; }
}

.camera-card {
    background: var(--card-bg, #1e1e1e);
    border-radius: 8px;
    overflow: hidden;
    box-shadow: 0 2px 8px rgba(0,0,0,0.3);
}

.camera-header {
    display: flex;
    justify-content: space-between;
    padding: 0.5rem 1rem;
    background: var(--header-bg, #2d2d2d);
}

.camera-name {
    font-weight: 600;
    color: var(--text-primary, #fff);
}

.camera-status {
    font-size: 0.875rem;
    padding: 0.125rem 0.5rem;
    border-radius: 4px;
}

.camera-status.online {
    background: var(--success-bg, #1b4332);
    color: var(--success-text, #95d5b2);
}

.camera-status.offline {
    background: var(--error-bg, #4a1919);
    color: var(--error-text, #f5c6cb);
}

.camera-feed {
    width: 100%;
    height: auto;
    display: block;
    aspect-ratio: 4/3;
    object-fit: contain;
    background: #000;
}

.camera-offline {
    aspect-ratio: 4/3;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    background: #111;
    color: #666;
}

.camera-controls {
    display: flex;
    gap: 0.5rem;
    padding: 0.5rem;
    background: var(--controls-bg, #252525);
}

.cameras-toolbar {
    display: flex;
    gap: 1rem;
    padding: 1rem;
    background: var(--toolbar-bg, #1a1a1a);
    border-bottom: 1px solid var(--border, #333);
}
```

### JavaScript: cameras.js (minimal)

```javascript
// Camera stream control - minimal JS for pause/resume
const cameraStreams = {};

function toggleStream(cameraName) {
    const card = document.querySelector(`[data-camera="${cameraName}"]`);
    const img = card.querySelector('.camera-feed');
    const btn = card.querySelector('button');

    if (cameraStreams[cameraName]?.paused) {
        // Resume: restore src
        img.src = `/cameras/${cameraName}/stream`;
        btn.textContent = 'Pause';
        cameraStreams[cameraName].paused = false;
    } else {
        // Pause: clear src to stop stream
        img.src = '/static/paused.svg';
        btn.textContent = 'Resume';
        cameraStreams[cameraName] = { paused: true };
    }
}

// WebSocket for status updates
function connectCameraStatus() {
    const ws = new WebSocket(`ws://${location.host}/ws/cameras`);

    ws.onmessage = (event) => {
        const status = JSON.parse(event.data);
        updateCameraStatus(status);
    };

    ws.onclose = () => {
        // Reconnect after 3 seconds
        setTimeout(connectCameraStatus, 3000);
    };
}

function updateCameraStatus(status) {
    const card = document.querySelector(`[data-camera="${status.camera}"]`);
    if (!card) return;

    const statusEl = card.querySelector('.camera-status');
    if (status.online) {
        statusEl.className = 'camera-status online';
        statusEl.textContent = `${status.fps.toFixed(1)} fps`;
    } else {
        statusEl.className = 'camera-status offline';
        statusEl.textContent = 'offline';
    }
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', connectCameraStatus);
```

## Server-Side Implementation

### CameraInfo struct

```go
type CameraInfo struct {
    Name       string
    Device     string
    Online     bool
    FPS        float64
    Resolution string
    LastFrame  time.Time
}
```

### Camera Status Tracking

```go
type CameraMonitor struct {
    mu         sync.RWMutex
    cameras    map[string]*CameraStatus
    nats       *gorainats.Client
    topics     *topics.Builder
}

type CameraStatus struct {
    Name       string
    Online     bool
    FPS        float64
    Resolution string
    LastFrame  time.Time

    // For FPS calculation
    frameCount int
    fpsWindow  time.Time
}

func (m *CameraMonitor) Start(ctx context.Context) error {
    // Subscribe to all camera data topics
    pattern := m.topics.ComponentData("*")

    _, err := m.nats.Subscribe(pattern, func(msg *nats.Msg) {
        // Extract camera name from topic
        cameraName := extractCameraName(msg.Subject)

        m.mu.Lock()
        defer m.mu.Unlock()

        status, ok := m.cameras[cameraName]
        if !ok {
            status = &CameraStatus{Name: cameraName}
            m.cameras[cameraName] = status
        }

        status.Online = true
        status.LastFrame = time.Now()
        status.frameCount++

        // Calculate FPS every second
        if time.Since(status.fpsWindow) >= time.Second {
            status.FPS = float64(status.frameCount) / time.Since(status.fpsWindow).Seconds()
            status.frameCount = 0
            status.fpsWindow = time.Now()
        }
    })

    return err
}

// Mark cameras as offline if no frames for 5 seconds
func (m *CameraMonitor) checkOffline() {
    m.mu.Lock()
    defer m.mu.Unlock()

    threshold := 5 * time.Second
    for _, status := range m.cameras {
        if time.Since(status.LastFrame) > threshold {
            status.Online = false
            status.FPS = 0
        }
    }
}
```

## Configuration

### RDL Dashboard Configuration

```json
{
  "dashboard": {
    "enabled": true,
    "listen": ":8080",
    "cameras": {
      "enabled": true,
      "default_layout": "2x2",
      "default_quality": 80,
      "max_fps": 30,
      "offline_threshold_seconds": 5
    }
  }
}
```

## Performance Considerations

### Memory Management

1. **MJPEG boundary writes**: Use `sync.Pool` for frame buffers
2. **Client cleanup**: Ensure subscriptions are unsubscribed on disconnect
3. **Max concurrent streams**: Limit to prevent memory exhaustion

### Bandwidth Optimization

1. **Quality control**: Allow users to reduce JPEG quality
2. **Frame rate limiting**: Server-side throttling per client
3. **Lazy loading**: Only start streams for visible cameras

### Example Rate Limiter

```go
type StreamRateLimiter struct {
    maxFPS   float64
    lastSent time.Time
}

func (r *StreamRateLimiter) ShouldSend() bool {
    minInterval := time.Duration(float64(time.Second) / r.maxFPS)
    if time.Since(r.lastSent) >= minInterval {
        r.lastSent = time.Now()
        return true
    }
    return false
}
```

## Error Handling

### Camera Offline States

1. **Camera not in config**: Show "Not configured" message
2. **Camera in config but no device**: Show "Device missing" with last known status
3. **Camera device exists but stream failed**: Show "Stream error" with retry button
4. **NATS disconnected**: Show "Connection lost" banner for all cameras

### Graceful Degradation

```html
<img src="/cameras/front_camera/stream"
     onerror="this.onerror=null; this.src='/static/camera-error.svg';
              this.parentElement.classList.add('error')"
     alt="front_camera">
```

## Testing Plan

### Unit Tests

1. `TestCameraMonitor_FPSCalculation` - Verify FPS tracking accuracy
2. `TestMJPEGStream_Boundary` - Verify correct multipart format
3. `TestCameraStatus_Offline` - Verify offline detection

### Integration Tests

1. Start robot with camera, verify stream endpoint works
2. Disconnect camera, verify offline status propagates
3. Reconnect camera, verify stream resumes

### Manual Testing

1. Open dashboard in multiple browsers simultaneously
2. Test with 1, 2, 4 cameras
3. Test pause/resume functionality
4. Test snapshot download
5. Test mobile responsiveness

## Future Enhancements

### Phase 2: Advanced Features

1. **WebRTC mode**: Optional low-latency mode using go2rtc
2. **Recording**: Save stream to file from dashboard
3. **PTZ controls**: If camera supports pan/tilt/zoom
4. **ROI overlay**: Draw regions of interest on stream
5. **ML overlay**: Show bounding boxes from vision service

### Phase 3: Multi-Robot

1. **Camera discovery**: Auto-detect cameras from multiple robots
2. **Aggregated view**: View cameras from all robots in fleet
3. **Camera search**: Filter by robot, type, or status

## File Structure

```
pkg/dashboard/
  cameras/
    handler.go       # HTTP handlers for camera endpoints
    monitor.go       # Camera status tracking
    stream.go        # MJPEG streaming implementation

  templates/
    cameras.templ    # Templ template for camera tab

  static/
    css/
      cameras.css    # Camera-specific styles
    js/
      cameras.js     # Minimal camera control JS
    img/
      camera-offline.svg
      camera-error.svg
      paused.svg
```

## Summary

The camera monitor provides a simple, efficient way to view live camera feeds from the robot dashboard. By bridging NATS frame topics directly to MJPEG streams, we avoid the complexity of WebRTC while still achieving acceptable latency for monitoring purposes.

Key design decisions:
1. **NATS-to-MJPEG bridge** - Leverages existing frame publishing
2. **HTMX-driven UI** - Minimal JavaScript, server-rendered
3. **WebSocket status** - Separate channel for status updates (fps, online/offline)
4. **Lazy loading** - Only active cameras stream data
5. **Quality controls** - User can trade bandwidth for quality

This design aligns with Gorai's philosophy of simplicity, modularity, and Go-first implementation.
