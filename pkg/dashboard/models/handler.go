package models

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/gorai/gorai/pkg/config"
)

// Handler provides HTTP handlers for model service endpoints.
type Handler struct {
	monitor  *Monitor
	robotCfg *config.RDL
	logger   *slog.Logger

	// WebSocket clients for model status
	wsClients      map[*websocket.Conn]bool
	wsMu           sync.RWMutex
	OriginPatterns []string
}

// NewHandler creates a new model handler.
func NewHandler(monitor *Monitor, robotCfg *config.RDL, logger *slog.Logger) *Handler {
	return &Handler{
		monitor:        monitor,
		robotCfg:       robotCfg,
		logger:         logger,
		wsClients:      make(map[*websocket.Conn]bool),
		OriginPatterns: []string{"http://127.0.0.1:*", "http://localhost:*"},
	}
}

// HandleList handles the model list endpoint.
func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	models := h.monitor.GetModels()

	// Check if JSON is requested
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models)
		return
	}

	// Return HTML page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.writeModelsHTML(w, models)
}

// writeModelsHTML writes the models page HTML.
func (h *Handler) writeModelsHTML(w http.ResponseWriter, models []ModelStatus) {
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AI Models - Gorai Dashboard</title>
    <link rel="stylesheet" href="/static/css/main.css">
</head>
<body>
    <nav class="nav">
        <div class="nav-brand">Gorai</div>
        <ul class="nav-tabs">
            <li><a href="/">Status</a></li>
            <li><a href="/cameras">Cameras</a></li>
            <li><a href="/models" class="active">AI / Models</a></li>
        </ul>
    </nav>
    <main>
        <div class="cameras-toolbar">
            <h2>AI / Model Services</h2>
        </div>
`))

	if len(models) == 0 {
		w.Write([]byte(`
        <div class="no-cameras">
            <p>No model services detected</p>
            <small>Model services publish heartbeats to gorai.&lt;robot&gt;.system.heartbeat</small>
        </div>
`))
	} else {
		// Show annotated video feeds for each model
		w.Write([]byte(`        <div class="camera-grid grid-2x2">
`))
		for _, model := range models {
			h.writeModelFeedCard(w, model)
		}
		w.Write([]byte(`        </div>
`))
	}

	w.Write([]byte(`    </main>
    <script src="/static/js/models.js"></script>
</body>
</html>
`))
}

// writeModelCard writes a single model service card HTML.
func (h *Handler) writeModelCard(w http.ResponseWriter, model ModelStatus) {
	statusClass := "offline"
	switch model.Status {
	case "running":
		statusClass = "online"
	case "error":
		statusClass = "error"
	}

	w.Write([]byte(`            <div class="model-card" data-model="`))
	w.Write([]byte(model.Name))
	w.Write([]byte(`">
                <div class="model-header">
                    <span class="model-name">`))
	w.Write([]byte(model.Name))
	w.Write([]byte(`</span>
                    <span class="camera-status `))
	w.Write([]byte(statusClass))
	w.Write([]byte(`">`))
	w.Write([]byte(model.Status))
	w.Write([]byte(`</span>
                </div>
                <div class="model-metrics">
                    <div class="metric">
                        <span class="metric-value">`))
	w.Write([]byte(fmt.Sprintf("%.1f", model.FPS)))
	w.Write([]byte(`</span>
                        <span class="metric-label">FPS</span>
                    </div>
                    <div class="metric">
                        <span class="metric-value">`))
	w.Write([]byte(fmt.Sprintf("%.1f", model.InferenceMs)))
	w.Write([]byte(`</span>
                        <span class="metric-label">Inference (ms)</span>
                    </div>
                    <div class="metric">
                        <span class="metric-value">`))
	w.Write([]byte(fmt.Sprintf("%d", model.FramesProcessed)))
	w.Write([]byte(`</span>
                        <span class="metric-label">Frames</span>
                    </div>
                    <div class="metric">
                        <span class="metric-value">`))
	w.Write([]byte(fmt.Sprintf("%d", model.TotalDetections)))
	w.Write([]byte(`</span>
                        <span class="metric-label">Detections</span>
                    </div>
                </div>
                <div class="model-uptime">
                    Uptime: `))
	w.Write([]byte(formatUptime(model.UptimeSeconds)))
	w.Write([]byte(`
                </div>
            </div>
`))
}

// writeModelFeedCard writes a card with the annotated video feed.
func (h *Handler) writeModelFeedCard(w http.ResponseWriter, model ModelStatus) {
	statusClass := "offline"
	statusText := "offline"
	if model.Status == "running" {
		statusClass = "online"
		statusText = fmt.Sprintf("%.1f fps", model.FPS)
	}

	w.Write([]byte(`            <div class="camera-card" data-model="`))
	w.Write([]byte(model.Name))
	w.Write([]byte(`">
                <div class="camera-header">
                    <span class="camera-name">`))
	w.Write([]byte(model.Name))
	w.Write([]byte(`</span>
                    <span class="camera-status `))
	w.Write([]byte(statusClass))
	w.Write([]byte(`">`))
	w.Write([]byte(statusText))
	w.Write([]byte(`</span>
                </div>
`))

	if model.Status == "running" {
		w.Write([]byte(`                <img src="/models/`))
		w.Write([]byte(model.Name))
		w.Write([]byte(`/stream" alt="`))
		w.Write([]byte(model.Name))
		w.Write([]byte(`" class="camera-feed" loading="lazy">
`))
	} else {
		w.Write([]byte(`                <div class="camera-offline">
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="48" height="48">
                        <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
                        <path d="M8 21h8"/>
                        <path d="M12 17v4"/>
                        <line x1="2" y1="2" x2="22" y2="22"/>
                    </svg>
                    <p>Model Offline</p>
                </div>
`))
	}

	w.Write([]byte(`                <div class="camera-controls">
                    <a class="btn" href="/models/`))
	w.Write([]byte(model.Name))
	w.Write([]byte(`/snapshot" download="`))
	w.Write([]byte(model.Name))
	w.Write([]byte(`-annotated.jpg">Snapshot</a>
                </div>
            </div>
`))
}

// formatUptime formats seconds into a human-readable duration.
func formatUptime(seconds float64) string {
	d := time.Duration(seconds) * time.Second
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	secs := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, mins, secs)
	}
	if mins > 0 {
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	return fmt.Sprintf("%ds", secs)
}

// HandleDetections returns recent detection events as JSON.
func (h *Handler) HandleDetections(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	detections := h.monitor.GetRecentDetections(limit)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detections)
}

// Maximum WebSocket clients for model status updates.
const maxModelWSClients = 100

// HandleWebSocket handles WebSocket connections for model status updates.
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	h.wsMu.RLock()
	clientCount := len(h.wsClients)
	h.wsMu.RUnlock()
	if clientCount >= maxModelWSClients {
		http.Error(w, "too many WebSocket connections", http.StatusServiceUnavailable)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: h.OriginPatterns,
	})
	if err != nil {
		h.logger.Warn("WebSocket accept error", "error", err)
		return
	}

	h.wsMu.Lock()
	h.wsClients[conn] = true
	h.wsMu.Unlock()

	h.logger.Debug("Model WebSocket client connected", "client", r.RemoteAddr)

	// Send initial status for all models
	for _, model := range h.monitor.GetModels() {
		data, _ := json.Marshal(map[string]interface{}{
			"type":  "model_status",
			"model": model,
		})
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
	h.logger.Debug("Model WebSocket client disconnected", "client", r.RemoteAddr)
}

// BroadcastModelStatus sends a model status update to all connected WebSocket clients.
func (h *Handler) BroadcastModelStatus(model ModelStatus) {
	data, err := json.Marshal(map[string]interface{}{
		"type":  "model_status",
		"model": model,
	})
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

// BroadcastDetection sends a detection event to all connected WebSocket clients.
func (h *Handler) BroadcastDetection(event DetectionEvent) {
	data, err := json.Marshal(map[string]interface{}{
		"type":      "detection",
		"detection": event,
	})
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
