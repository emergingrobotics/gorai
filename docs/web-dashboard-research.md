# Web Dashboard Research for Gorai

**Date**: 2025-12-11
**Purpose**: Research modern Go-based web UI frameworks and libraries for building an embedded robot control dashboard

## Executive Summary

For Gorai's embedded web dashboard, the recommended approach is:

1. **Backend**: Go standard library `net/http` or **Chi** router with **embed.FS** for asset bundling
2. **Real-time Communication**: **coder/websocket** (maintained fork of nhooyr.io/websocket)
3. **Frontend Approach**: **HTMX + Templ** for server-side rendering OR lightweight SPA with **uPlot** for charts
4. **Video Streaming**: **go2rtc** for multi-protocol camera streaming (WebRTC, MJPEG, HLS)
5. **Charts**: **uPlot** for high-frequency sensor data visualization

This stack provides:
- Single binary deployment with embedded assets
- Sub-100KB JavaScript payload
- 60fps real-time chart updates
- WebRTC with <500ms latency for video
- Apache 2.0 compatible licensing

---

## 1. Go Web Frameworks

### Recommended: Chi Router

**GitHub**: https://github.com/go-chi/chi
**License**: MIT
**Bundle Size**: Minimal (library, not bundled)

**Pros**:
- Idiomatic Go, follows `net/http` patterns
- Lightweight and composable
- Easy middleware chaining
- Works seamlessly with `embed.FS`

**Cons**:
- Less "batteries included" than Gin/Echo

### Alternative: Gin

**GitHub**: https://github.com/gin-gonic/gin
**License**: MIT
**Bundle Size**: Minimal (library)

**Pros**:
- Most popular Go web framework (70k+ stars)
- Fast performance benchmarks
- Rich middleware ecosystem
- JSON validation built-in

**Cons**:
- More opinionated than Chi
- Larger dependency tree

### Alternative: Fiber

**GitHub**: https://github.com/gofiber/fiber
**License**: MIT
**Bundle Size**: Minimal (library)

**Pros**:
- Fastest performance (built on fasthttp)
- Express.js-like API (familiar for Node.js developers)
- Built-in WebSocket support

**Cons**:
- Uses fasthttp instead of `net/http` (different patterns)
- Less idiomatic Go

### Alternative: Echo

**GitHub**: https://github.com/labstack/echo
**License**: MIT
**Bundle Size**: Minimal (library)

**Pros**:
- High performance
- Extensive middleware
- Good documentation

**Cons**:
- More complex than Chi for simple use cases

### Standard Library (net/http)

**License**: BSD-3-Clause (Go)
**Bundle Size**: Minimal (stdlib)

**Pros**:
- Zero dependencies
- Perfect for embedded systems
- Works great with `embed.FS`

**Cons**:
- More verbose routing
- Manual middleware composition

---

## 2. Embedding Static Assets (embed.FS)

**Available**: Go 1.16+
**License**: BSD-3-Clause (Go standard library)
**Documentation**: https://pkg.go.dev/embed

### Key Features

```go
//go:embed static/*
var staticFiles embed.FS

// Serve embedded files
http.Handle("/", http.FileServer(http.FS(staticFiles)))
```

- Compile-time embedding of files into binary
- Perfect for single-binary robot deployments
- Works with all Go web frameworks
- Can embed entire frontend builds (React/Vue/Svelte/HTMX)

### Best Practices for Robotics

```go
//go:embed dist/*
var dashboardUI embed.FS

//go:embed static/js/* static/css/*
var assets embed.FS

// Strip "dist" prefix for serving
distFS, _ := fs.Sub(dashboardUI, "dist")
http.Handle("/", http.FileServer(http.FS(distFS)))
```

---

## 3. WebSocket Libraries for Real-Time Updates

### Recommended: coder/websocket

**GitHub**: https://github.com/coder/websocket
**License**: ISC (permissive, compatible with Apache 2.0)
**Formerly**: nhooyr.io/websocket

**Pros**:
- Idiomatic Go with context support
- Faster than gorilla/websocket
- Memory efficient
- Uses standard `net/http`
- Actively maintained by Coder

**Cons**:
- Less mature than gorilla (but rapidly improving)

### Alternative: gorilla/websocket

**GitHub**: https://github.com/gorilla/websocket
**License**: BSD-2-Clause
**Stars**: 22k+

**Pros**:
- Battle-tested and widely used
- Extensive documentation
- Large community

**Cons**:
- Writes directly to net.Conn (duplicates features)
- Uses `unsafe` internally
- Slower than coder/websocket

### Integration with NATS

Since Gorai uses NATS extensively, the WebSocket layer should bridge browser clients to NATS topics:

```
Browser (WebSocket) <--> Go Server <--> NATS <--> Robot Components
```

This allows the dashboard to subscribe to sensor streams, command topics, and telemetry without exposing NATS directly to browsers.

---

## 4. Frontend Approaches

### Option A: HTMX + Templ (Recommended for Simplicity)

**HTMX**: https://htmx.org
**License**: BSD-2-Clause
**Bundle Size**: ~14KB minified + gzipped

**Templ**: https://github.com/a-h/templ
**License**: MIT
**Type**: Go templating language that compiles to Go code

**Why This Stack**:
- Minimal JavaScript (just HTMX)
- Server-side rendering = easier debugging
- Type-safe templates with Go
- Single binary deployment
- Works with JavaScript disabled (graceful degradation)

**Example Resources**:
- https://github.com/Guillembonet/go-templ-htmx
- https://github.com/emarifer/go-echo-templ-htmx

**Pros**:
- ~14KB total JS (just HTMX)
- Server does the heavy lifting
- Easy to reason about
- Great for forms, controls, status updates

**Cons**:
- Real-time charts still need additional JS
- Not ideal for complex client-side state

### Option B: Lightweight SPA with uPlot

Build a minimal single-page app with:
- Vanilla JS or Preact (~3KB React alternative)
- uPlot for charts
- WebSocket connection for real-time data

**Total Bundle**: ~50-100KB minified

---

## 5. JavaScript Charting Libraries

### Recommended: uPlot

**GitHub**: https://github.com/leeoniya/uPlot
**License**: MIT
**Bundle Size**: ~50KB minified

**Performance** (from benchmarks):
- Creates 166,650 point chart in 25ms (cold start)
- ~100,000 pts/ms scaling
- 60fps live streaming up to 100k in-view points
- 10% CPU, 12MB RAM when updating 3,600 points at 60fps

**Comparison** (updating 3,600 points at 60fps):
- **uPlot**: 10% CPU, 12.3MB RAM
- **Chart.js**: 40% CPU, 77MB RAM
- **ECharts**: 70% CPU, 85MB RAM

**Why uPlot for Robotics**:
- Built specifically for time-series data
- Handles high-frequency sensor updates
- Minimal memory footprint
- Canvas-based (faster than SVG)

**Limitations**:
- Time-series focused (not for pie/bar charts)
- Less feature-rich than Plotly/ECharts

### Alternative: Chart.js

**GitHub**: https://github.com/chartjs/Chart.js
**License**: MIT
**Bundle Size**: ~254KB minified

**Pros**:
- Most popular charting library (2M+ weekly downloads)
- Beginner-friendly API
- Many chart types
- Large plugin ecosystem

**Cons**:
- 5x larger than uPlot
- 4x slower real-time performance
- Higher memory usage

**Good For**: Dashboards with mixed chart types (bar, pie, doughnut)

### Alternative: Plotly.js

**GitHub**: https://github.com/plotly/plotly.js
**License**: MIT
**Bundle Size**: ~3.6MB minified (full), ~1MB (basic)

**Pros**:
- 40+ chart types including 3D
- Scientific visualization
- WebGL rendering for 2D/3D performance

**Cons**:
- Massive bundle size
- Slow real-time updates
- Overkill for most robotics dashboards

**Good For**: Scientific data analysis, not real-time control

### Alternative: Apache ECharts

**GitHub**: https://github.com/apache/echarts
**License**: Apache 2.0 (perfect match!)
**Bundle Size**: ~900KB minified (full)

**Pros**:
- Apache 2.0 license (same as Gorai)
- Many chart types
- Good performance with data sampling
- Progressive rendering

**Cons**:
- Large bundle size
- Slower than uPlot for real-time data

**Good For**: Complex visualizations with lower frequency updates

### Alternative: Dygraphs

**GitHub**: https://github.com/danvk/dygraphs
**License**: MIT
**Bundle Size**: ~150KB minified

**Pros**:
- Built for time-series
- Handles millions of points
- Lightweight compared to Plotly/ECharts

**Cons**:
- Less active development
- Older API patterns
- Smaller than uPlot but slower

---

## 6. Video/Camera Streaming

### Recommended: go2rtc

**GitHub**: https://github.com/AlexxIT/go2rtc
**License**: MIT
**Website**: https://go2rtc.com

**The Ultimate Camera Streaming Solution**

**Features**:
- Multi-protocol input: RTSP, RTMP, HTTP, MJPEG, USB cameras
- Multi-protocol output: WebRTC, HLS, MJPEG, MPEG-TS
- Zero-dependency, zero-config deployment
- Cross-platform (Windows, macOS, Linux, ARM)

**Latency Comparison**:
- **WebRTC**: 0.5s (best)
- **MJPEG**: 1-2s
- **HLS**: 5-10s (worst)

**Why Perfect for Robotics**:
- Single camera → multiple output formats simultaneously
- Browser-compatible (WebRTC for Chrome/Firefox, MJPEG fallback)
- Acts as stream proxy (reduces camera load)
- Docker/Podman ready

**Example Setup**:
```yaml
# go2rtc.yaml
streams:
  front_camera:
    - rtsp://camera_ip/stream
    - exec:ffmpeg -i /dev/video0 ...

api:
  listen: ":1984"
```

**Browser Endpoints**:
- `http://localhost:1984/stream.html` - Interactive player
- `http://localhost:1984/api/stream.mjpeg` - MJPEG stream
- WebRTC via JavaScript API

### Alternative: Pion WebRTC (Go Library)

**GitHub**: https://github.com/pion/webrtc
**License**: MIT

**Pros**:
- Pure Go WebRTC implementation
- Full control over streaming pipeline
- Can integrate directly into Gorai

**Cons**:
- Requires more implementation work
- Need to handle signaling yourself

**Good For**: Custom streaming requirements, direct integration

### Alternative: MJPEG Streaming (Simple)

**Implementation**: Built-in with Go standard library

```go
http.HandleFunc("/stream.mjpeg", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
    for {
        frame := captureFrame()
        fmt.Fprintf(w, "--frame\r\n")
        fmt.Fprintf(w, "Content-Type: image/jpeg\r\n\r\n")
        w.Write(frame)
        fmt.Fprintf(w, "\r\n")
        time.Sleep(33 * time.Millisecond) // ~30fps
    }
})
```

**Pros**:
- Simple to implement
- Works in all browsers
- No JavaScript needed

**Cons**:
- Higher latency (1-2s)
- Less efficient bandwidth usage
- No adaptive bitrate

---

## 7. Complete Stack Recommendations

### Option 1: Minimal HTMX Dashboard (Recommended for MVP)

**Backend**:
- Chi router + embed.FS
- coder/websocket for real-time updates

**Frontend**:
- HTMX (14KB) for interactivity
- Templ for type-safe HTML templates
- uPlot (50KB) for sensor charts
- go2rtc for camera streaming

**Total JS**: ~65KB
**Build**: Single binary with embedded assets
**Real-time**: WebSocket bridge to NATS
**License**: All MIT/BSD/ISC (Apache 2.0 compatible)

**Example Structure**:
```
cmd/dashboard/
  main.go           # Serves dashboard
  templates/        # Templ files
    dashboard.templ
    sensors.templ
  static/
    htmx.min.js
    uplot.min.js
    uplot.min.css
    dashboard.js    # WebSocket + uPlot integration
```

### Option 2: Lightweight SPA Dashboard

**Backend**:
- Chi router + embed.FS
- coder/websocket

**Frontend**:
- Preact (3KB) or vanilla JS
- uPlot for charts
- Custom WebSocket client

**Total JS**: ~80-120KB
**Build**: `npm run build` → embed dist/
**Real-time**: WebSocket → NATS bridge

**Good For**: More complex UI state, multiple views

### Option 3: Full-Featured Dashboard

**Backend**:
- Gin + embed.FS
- gorilla/websocket (battle-tested)

**Frontend**:
- React + TypeScript
- Apache ECharts (license match!)
- go2rtc integration

**Total JS**: ~800KB-1.5MB
**Build**: Production React build → embed
**Real-time**: WebSocket + HTTP API

**Good For**: Production deployments with complex visualizations

---

## 8. Real-Time Performance Considerations

### High-Frequency Sensor Data (100Hz+)

**Challenges**:
- Browser can't render 100+ chart updates/sec
- WebSocket backpressure

**Solutions**:
1. **Server-side downsampling**: Aggregate to 10-30Hz before sending
2. **Client-side buffering**: Use `requestAnimationFrame` for rendering
3. **Selective updates**: Only send changed values

**uPlot Strategy**:
```javascript
let buffer = [];
const maxPoints = 3600; // 2 minutes at 30Hz

ws.onmessage = (msg) => {
    buffer.push(JSON.parse(msg.data));
    if (buffer.length > maxPoints) buffer.shift();
};

function render() {
    if (buffer.length > 0) {
        chart.setData([timestamps, values]);
    }
    requestAnimationFrame(render);
}
render();
```

### Multiple Camera Streams

**Use go2rtc**:
- Single source → multiple format outputs
- Browser chooses best format (WebRTC > MJPEG)
- Reduces camera load

**Layout**:
- 1 stream: Full screen
- 2-4 streams: Grid layout
- 5+ streams: Thumbnails + click to expand

---

## 9. Examples from Robotics/IoT Projects

### Similar Projects Using Go

1. **Home Assistant go2rtc Integration**
   - https://github.com/AlexxIT/go2rtc
   - Used for camera streaming in home automation
   - Handles dozens of simultaneous camera feeds

2. **Portainer** (Docker/Podman UI)
   - https://github.com/portainer/portainer
   - Go backend, Angular frontend
   - Embedded assets in binary
   - Real-time container logs via WebSocket

3. **Gobot Dashboard Example**
   - https://gobot.io
   - Uses Gin + embedded HTML
   - Real-time sensor visualization

4. **Grafana** (Observability Platform)
   - https://github.com/grafana/grafana
   - Go backend, React frontend
   - Time-series visualization
   - WebSocket for live updates

### Key Lessons

1. **Embed everything**: Single binary deployment is critical for robotics
2. **WebSocket + downsampling**: Don't send raw sensor data to browsers
3. **Choose charts wisely**: uPlot for time-series, Chart.js for mixed types
4. **Video complexity**: Use go2rtc unless you have specific needs
5. **Keep JS minimal**: Every KB matters on embedded systems

---

## 10. Recommended Architecture for Gorai

```
┌─────────────────────────────────────────────────────────┐
│ Gorai Robot Binary                                      │
│                                                          │
│  ┌──────────────────────────────────────────────┐      │
│  │ Dashboard Service (Optional)                 │      │
│  │                                               │      │
│  │  Chi Router + embed.FS                       │      │
│  │  ├─ /            → Dashboard HTML (Templ)    │      │
│  │  ├─ /ws          → WebSocket Handler         │      │
│  │  ├─ /api/*       → REST endpoints            │      │
│  │  └─ /video/*     → Proxy to go2rtc           │      │
│  │                                               │      │
│  │  WebSocket Bridge                             │      │
│  │  ├─ Subscribe to NATS topics                 │      │
│  │  ├─ Downsample sensor data                   │      │
│  │  └─ Publish commands to NATS                 │      │
│  └──────────────────────────────────────────────┘      │
│                         │                               │
│                         │                               │
│  ┌──────────────────────▼───────────────────────┐      │
│  │ NATS (Core Message Bus)                      │      │
│  │                                               │      │
│  │  Topics:                                     │      │
│  │  - gorai.robot.camera.data                   │      │
│  │  - gorai.robot.sensors.imu                   │      │
│  │  - gorai.robot.cmd_vel                       │      │
│  └──────────────────────────────────────────────┘      │
│                         │                               │
│  ┌──────────────────────▼───────────────────────┐      │
│  │ Robot Components & Services                  │      │
│  │  - Cameras, Motors, Sensors                  │      │
│  │  - Vision, SLAM, Navigation                  │      │
│  └──────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────┘
                         │
                         │ HTTP/WebSocket
                         ▼
┌─────────────────────────────────────────────────────────┐
│ Browser Dashboard                                       │
│                                                          │
│  HTML/CSS (from Templ templates)                        │
│  HTMX (14KB) - Interactivity                           │
│  uPlot (50KB) - Sensor charts                          │
│  dashboard.js - WebSocket client                        │
│                                                          │
│  Views:                                                 │
│  ├─ System Status & Topology                           │
│  ├─ Camera Feeds (go2rtc)                              │
│  ├─ Sensor Graphs (real-time)                          │
│  ├─ ML Model Inference Results                         │
│  └─ Command Interface                                   │
└─────────────────────────────────────────────────────────┘
```

### Configuration Example

```json
{
  "dashboard": {
    "enabled": true,
    "listen": ":8080",
    "websocket_buffer": 100,
    "sensor_downsample_hz": 10,
    "video": {
      "go2rtc_url": "http://localhost:1984",
      "streams": ["front_camera", "depth_camera"]
    }
  }
}
```

### Code Example: WebSocket Bridge

```go
type DashboardServer struct {
    nc     *nats.Conn
    router *chi.Mux

    // WebSocket clients
    clients map[*websocket.Conn]bool
    mu      sync.RWMutex
}

func (s *DashboardServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    ws, err := websocket.Accept(w, r, nil)
    if err != nil {
        return
    }
    defer ws.Close(websocket.StatusNormalClosure, "")

    s.mu.Lock()
    s.clients[ws] = true
    s.mu.Unlock()

    // Subscribe to NATS topics
    sub, _ := s.nc.Subscribe("gorai.*.sensors.>", func(msg *nats.Msg) {
        // Downsample and forward to WebSocket
        s.broadcastToClients(msg.Data)
    })
    defer sub.Unsubscribe()

    // Read commands from WebSocket, publish to NATS
    for {
        _, data, err := ws.Read(context.Background())
        if err != nil {
            break
        }
        s.nc.Publish("gorai.cmd", data)
    }
}
```

---

## 11. License Compatibility Summary

All recommended libraries are compatible with Apache 2.0:

| Library | License | Compatible | Notes |
|---------|---------|------------|-------|
| Chi | MIT | ✅ | Permissive |
| Gin | MIT | ✅ | Permissive |
| Fiber | MIT | ✅ | Permissive |
| Echo | MIT | ✅ | Permissive |
| embed.FS | BSD-3 | ✅ | Go stdlib |
| coder/websocket | ISC | ✅ | Permissive |
| gorilla/websocket | BSD-2 | ✅ | Permissive |
| HTMX | BSD-2 | ✅ | Permissive |
| Templ | MIT | ✅ | Permissive |
| uPlot | MIT | ✅ | Permissive |
| Chart.js | MIT | ✅ | Permissive |
| Apache ECharts | Apache 2.0 | ✅ | Perfect match! |
| Plotly.js | MIT | ✅ | Permissive |
| go2rtc | MIT | ✅ | Permissive |
| Pion WebRTC | MIT | ✅ | Permissive |

---

## 12. Implementation Roadmap

### Phase 1: MVP Dashboard (1-2 weeks)

**Goal**: Basic monitoring dashboard

1. Set up Chi router with embed.FS
2. Create Templ templates for:
   - System status page
   - Sensor list view
3. Implement WebSocket bridge to NATS
4. Add uPlot for 1-2 sensor graphs
5. Integrate go2rtc for single camera view

**Output**: Working dashboard showing live sensors + camera

### Phase 2: Enhanced Visualization (1-2 weeks)

**Goal**: Rich data visualization

1. Add multi-sensor charting
2. Implement topology view (which components are running)
3. Add command interface (send commands to robot)
4. Performance optimization (downsampling, buffering)
5. Add multiple camera views

**Output**: Production-ready monitoring dashboard

### Phase 3: ML Integration (1-2 weeks)

**Goal**: Show inference results

1. Subscribe to ML model output topics
2. Visualize bounding boxes on camera feed
3. Show classification results
4. Display inference timing/performance

**Output**: AI-aware dashboard

### Phase 4: Control Interface (2-3 weeks)

**Goal**: Interactive robot control

1. Add joystick/gamepad support
2. Implement waypoint navigation UI
3. Add configuration editor
4. Create action progress indicators

**Output**: Full control dashboard

---

## 13. Testing Strategy

### Performance Benchmarks

1. **Chart rendering**: Measure FPS with 10/30/60Hz data
2. **WebSocket throughput**: Max messages/sec before backpressure
3. **Memory usage**: Monitor over 1hr runtime
4. **Bundle size**: Measure embedded asset size

### Browser Compatibility

- Chrome/Edge (WebRTC primary)
- Firefox (WebRTC)
- Safari (HLS fallback)
- Mobile browsers

### Load Testing

- 10 simultaneous WebSocket connections
- 4 concurrent camera streams
- 20 sensor topics at 10Hz each

---

## 14. References

### Documentation
- Go embed package: https://pkg.go.dev/embed
- HTMX documentation: https://htmx.org/docs/
- Templ guide: https://templ.guide/
- uPlot documentation: https://github.com/leeoniya/uPlot/tree/master/docs
- go2rtc documentation: https://go2rtc.com/

### Tutorials
- [Go + HTMX + Templ Setup](https://medium.com/ostinato-rigore/go-htmx-templ-tailwind-complete-project-setup-hot-reloading-2ca1ba6c28be)
- [Building Reactive UIs with Go, Templ, and HTMX](https://medium.com/@iamsiddharths/building-reactive-uis-with-go-templ-and-htmx-a-simpler-path-beyond-spas-17e7dad2c7a2)
- [HTMX with Go templ](https://callistaenterprise.se/blogg/teknik/2024/01/08/htmx-with-go-templ/)

### Benchmark Sources
- [JavaScript Charting Libraries Comparison](https://blog.logrocket.com/comparing-most-popular-javascript-charting-libraries/)
- [uPlot Performance Benchmarks](https://github.com/leeoniya/uPlot#performance)
- [Best JavaScript Chart Libraries 2025](https://www.scichart.com/blog/best-javascript-chart-libraries/)

### Example Projects
- https://github.com/Guillembonet/go-templ-htmx
- https://github.com/emarifer/go-echo-templ-htmx
- https://github.com/stackus/todos (HTMX + Templ example)

---

## 15. Conclusion

For Gorai's embedded robot dashboard, the **HTMX + Templ + uPlot** stack provides the best balance of:

- **Simplicity**: Server-side rendering, minimal JavaScript
- **Performance**: <100ms chart updates, <500ms video latency
- **Size**: ~65KB total JavaScript
- **Deployment**: Single binary with embedded assets
- **License**: All Apache 2.0 compatible

The modular architecture allows starting simple (Phase 1 MVP) and growing into a full-featured control interface (Phase 4) without architectural rewrites.

**Next Steps**:
1. Create `pkg/dashboard` package
2. Implement basic Chi router + embed.FS setup
3. Build WebSocket bridge to NATS
4. Create Templ templates for system status
5. Integrate uPlot for sensor visualization

This research provides the foundation for a production-quality embedded dashboard that aligns with Gorai's goals of simplicity, modularity, and single-binary deployment.
