package cameras

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/gorai/gorai/pkg/config"
)

// Handler provides HTTP handlers for camera endpoints.
type Handler struct {
	monitor  *Monitor
	robotCfg *config.RDL
	logger   *slog.Logger

	// WebSocket clients for camera status
	wsClients map[*websocket.Conn]bool
	wsMu      sync.RWMutex
}

// NewHandler creates a new camera handler.
func NewHandler(monitor *Monitor, robotCfg *config.RDL, logger *slog.Logger) *Handler {
	h := &Handler{
		monitor:   monitor,
		robotCfg:  robotCfg,
		logger:    logger,
		wsClients: make(map[*websocket.Conn]bool),
	}

	// Register for status updates
	if monitor != nil {
		monitor.OnStatusChange(func(status CameraStatus) {
			h.broadcastStatus(status)
		})
	}

	return h
}

// HandleList handles the camera list endpoint.
func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	cameras := h.monitor.GetCameras()

	// Check if JSON is requested
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cameras)
		return
	}

	// Return HTML page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.writeCamerasHTML(w, cameras)
}

// writeCamerasHTML writes the cameras page HTML.
func (h *Handler) writeCamerasHTML(w http.ResponseWriter, cameras []CameraInfo) {
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Cameras - Gorai Dashboard</title>
    <link rel="stylesheet" href="/static/css/main.css">
</head>
<body>
    <nav class="nav">
        <div class="nav-brand">Gorai</div>
        <ul class="nav-tabs">
            <li><a href="/">Status</a></li>
            <li><a href="/cameras" class="active">Cameras</a></li>
            <li><a href="/models">AI / Models</a></li>
        </ul>
    </nav>
    <main>
        <div class="cameras-toolbar">
            <h2>Camera Monitor</h2>
            <select id="layout-select" onchange="setLayout(this.value)">
                <option value="1x1">1 Camera</option>
                <option value="2x2" selected>2x2 Grid</option>
                <option value="3x3">3x3 Grid</option>
            </select>
        </div>
`))

	if len(cameras) == 0 {
		w.Write([]byte(`
        <div class="no-cameras">
            <p>No cameras configured</p>
            <small>Add a camera component to your robot configuration</small>
        </div>
`))
	} else {
		w.Write([]byte(`        <div id="camera-grid" class="camera-grid grid-2x2">
`))
		for _, cam := range cameras {
			h.writeCameraCard(w, cam)
		}
		w.Write([]byte(`        </div>
`))
	}

	w.Write([]byte(`    </main>
    <script src="/static/js/cameras.js"></script>
</body>
</html>
`))
}

// writeCameraCard writes a single camera card HTML.
func (h *Handler) writeCameraCard(w http.ResponseWriter, cam CameraInfo) {
	statusClass := "offline"
	statusText := "offline"
	if cam.Online {
		statusClass = "online"
		statusText = formatFPS(cam.FPS)
	}

	w.Write([]byte(`            <div class="camera-card" data-camera="`))
	w.Write([]byte(cam.Name))
	w.Write([]byte(`">
                <div class="camera-header">
                    <span class="camera-name">`))
	w.Write([]byte(cam.Name))
	w.Write([]byte(`</span>
                    <span class="camera-status `))
	w.Write([]byte(statusClass))
	w.Write([]byte(`">`))
	w.Write([]byte(statusText))
	w.Write([]byte(`</span>
                </div>
`))

	if cam.Online {
		w.Write([]byte(`                <img src="/cameras/`))
		w.Write([]byte(cam.Name))
		w.Write([]byte(`/stream" alt="`))
		w.Write([]byte(cam.Name))
		w.Write([]byte(`" class="camera-feed" loading="lazy">
`))
	} else {
		w.Write([]byte(`                <div class="camera-offline">
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="48" height="48">
                        <path d="M23 19a2 2 0 01-2 2H3a2 2 0 01-2-2V8a2 2 0 012-2h4l2-3h6l2 3h4a2 2 0 012 2z"/>
                        <circle cx="12" cy="13" r="4"/>
                        <line x1="1" y1="1" x2="23" y2="23"/>
                    </svg>
                    <p>Camera Offline</p>
                </div>
`))
	}

	w.Write([]byte(`                <div class="camera-controls">
                    <button class="btn" onclick="toggleStream('`))
	w.Write([]byte(cam.Name))
	w.Write([]byte(`')">Pause</button>
                    <a class="btn" href="/cameras/`))
	w.Write([]byte(cam.Name))
	w.Write([]byte(`/snapshot" download="`))
	w.Write([]byte(cam.Name))
	w.Write([]byte(`.jpg">Snapshot</a>
                </div>
            </div>
`))
}

// formatFPS formats a FPS value for display.
func formatFPS(fps float64) string {
	if fps >= 10 {
		return formatFloat(fps, 0) + " fps"
	}
	return formatFloat(fps, 1) + " fps"
}

// formatFloat formats a float with the given precision.
func formatFloat(f float64, precision int) string {
	if precision == 0 {
		return string(rune('0' + int(f)%10))
	}
	// Simple formatting for low precision
	intPart := int(f)
	fracPart := int((f - float64(intPart)) * 10)
	return string(rune('0'+intPart%100/10)) + string(rune('0'+intPart%10)) + "." + string(rune('0'+fracPart))
}

// HandleWebSocket handles WebSocket connections for camera status updates.
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow connections from any origin
	})
	if err != nil {
		h.logger.Warn("WebSocket accept error", "error", err)
		return
	}

	h.wsMu.Lock()
	h.wsClients[conn] = true
	h.wsMu.Unlock()

	h.logger.Debug("Camera WebSocket client connected", "client", r.RemoteAddr)

	// Send initial status for all cameras
	for _, cam := range h.monitor.GetCameras() {
		status := CameraStatus{
			Type:   "camera_status",
			Camera: cam.Name,
			Online: cam.Online,
			FPS:    cam.FPS,
		}
		data, _ := json.Marshal(status)
		conn.Write(r.Context(), websocket.MessageText, data)
	}

	// Keep connection open and handle messages
	ctx := r.Context()
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			break
		}
	}

	h.wsMu.Lock()
	delete(h.wsClients, conn)
	h.wsMu.Unlock()

	conn.Close(websocket.StatusNormalClosure, "")
	h.logger.Debug("Camera WebSocket client disconnected", "client", r.RemoteAddr)
}

// broadcastStatus sends a status update to all connected WebSocket clients.
func (h *Handler) broadcastStatus(status CameraStatus) {
	data, err := json.Marshal(status)
	if err != nil {
		return
	}

	h.wsMu.RLock()
	clients := make([]*websocket.Conn, 0, len(h.wsClients))
	for conn := range h.wsClients {
		clients = append(clients, conn)
	}
	h.wsMu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	for _, conn := range clients {
		if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
			h.wsMu.Lock()
			delete(h.wsClients, conn)
			h.wsMu.Unlock()
			conn.Close(websocket.StatusGoingAway, "write error")
		}
	}
}
