package dashboard

import (
	"encoding/json"
	"html"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorai/gorai/pkg/config"
	"github.com/gorai/gorai/pkg/dashboard/cameras"
)

// handleIndex serves the main dashboard page.
func (d *Dashboard) handleIndex(w http.ResponseWriter, r *http.Request) {
	cameraList := d.cameraMonitor.GetCameras()

	// Check if there are any external services (AI/ML models)
	hasModels := false
	for _, svc := range d.robotCfg.Services {
		if svc.IsExternal() && !svc.Disabled {
			hasModels = true
			break
		}
	}

	robotName := html.EscapeString(d.robotCfg.Robot.Name)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>`))
	w.Write([]byte(robotName))
	w.Write([]byte(` - Gorai Dashboard</title>
    <link rel="stylesheet" href="/static/css/main.css">
</head>
<body>
    <nav class="nav">
        <div class="nav-brand">Gorai - `))
	w.Write([]byte(robotName))
	w.Write([]byte(`</div>
        <ul class="nav-tabs">
            <li><a href="/" class="active">Status</a></li>
            <li><a href="/cameras">Cameras</a></li>`))
	if hasModels {
		w.Write([]byte(`
            <li><a href="/models">AI / Models</a></li>`))
	}
	w.Write([]byte(`
        </ul>
    </nav>
    <main>
        <div class="status-grid">
            <div class="status-card">
                <h3>Robot</h3>
                <div class="value">`))
	w.Write([]byte(html.EscapeString(d.robotCfg.Robot.Name)))
	w.Write([]byte(`</div>
                <div class="label">Name</div>
            </div>
            <div class="status-card">
                <h3>Components</h3>
                <div class="value">`))
	w.Write([]byte(strconv.Itoa(len(d.robotCfg.Components))))
	w.Write([]byte(`</div>
                <div class="label">Configured</div>
            </div>
            <div class="status-card">
                <h3>Services</h3>
                <div class="value">`))
	w.Write([]byte(strconv.Itoa(len(d.robotCfg.Services))))
	w.Write([]byte(`</div>
                <div class="label">Configured</div>
            </div>
        </div>

`))

	// Build set of components referenced by services (to exclude from standalone list)
	serviceComponents := make(map[string]bool)
	for _, svc := range d.robotCfg.Services {
		for _, name := range d.getServiceDevices(svc) {
			serviceComponents[name] = true
		}
	}

	// Services section (rendered first, above components)
	if len(d.robotCfg.Services) > 0 {
		w.Write([]byte(`
        <div style="margin-top: 2rem;">
            <div class="status-card">
                <h3>Services</h3>
                <div class="component-list">
`))
		for _, svc := range d.robotCfg.Services {
			status := "active"
			statusLabel := "active"
			if svc.Disabled {
				status = "disabled"
				statusLabel = "disabled"
			} else if sv := d.serviceMonitor.FormatStatusValue(svc.Name); sv != "" {
				statusLabel = sv
				status = "online"
			}

			w.Write([]byte(`                    <div class="component-item">
                        <div>
                            <div class="name">`))
			w.Write([]byte(html.EscapeString(svc.Name)))
			w.Write([]byte(`</div>
                            <div class="type">`))
			w.Write([]byte(html.EscapeString(svc.Type)))
			if svc.Model != "" {
				w.Write([]byte(` / `))
				w.Write([]byte(html.EscapeString(svc.Model)))
			}
			w.Write([]byte(`</div>
                        </div>
                        <span class="camera-status `))
			w.Write([]byte(status))
			w.Write([]byte(`" data-service="`))
			w.Write([]byte(html.EscapeString(svc.Name)))
			w.Write([]byte(`">`))
			w.Write([]byte(html.EscapeString(statusLabel)))
			w.Write([]byte(`</span>
                    </div>
`))

			// List components referenced by this service
			devices := d.getServiceDevices(svc)
			if len(devices) > 0 {
				for _, devName := range devices {
					devStatus, devLabel := d.resolveComponentStatus(devName, cameraList)

					w.Write([]byte(`                    <div class="component-item" style="padding-left: 2.5rem; border-left: 3px solid #e0e0e0;">
                        <div>
                            <div class="name">`))
					w.Write([]byte(html.EscapeString(devName)))
					w.Write([]byte(`</div>
                            <div class="type">component</div>
                        </div>
                        `))
					d.writeStatusBadge(w, devName, devStatus, devLabel)
					w.Write([]byte(`
                    </div>
`))
				}
			}

			// Service detail rows (suntimes sunrise/sunset, light controller schedules)
			d.writeServiceDetails(w, svc.Name)
		}

		w.Write([]byte(`                </div>
            </div>
        </div>
`))
	}

	// Components section (only those not already listed under a service)
	w.Write([]byte(`
        <div style="margin-top: 2rem;">
            <div class="status-card">
                <h3>Components</h3>
                <div class="component-list">
`))

	for _, comp := range d.robotCfg.Components {
		if serviceComponents[comp.Name] {
			continue
		}

		status, statusLabel := d.resolveComponentStatus(comp.Name, cameraList)
		if comp.Disabled {
			status = "disabled"
			statusLabel = "disabled"
		}

		w.Write([]byte(`                    <div class="component-item">
                        <div>
                            <div class="name">`))
		w.Write([]byte(html.EscapeString(comp.Name)))
		w.Write([]byte(`</div>
                            <div class="type">`))
		w.Write([]byte(html.EscapeString(comp.Type)))
		if comp.Model != "" {
			w.Write([]byte(` / `))
			w.Write([]byte(html.EscapeString(comp.Model)))
		}
		w.Write([]byte(`</div>
                        </div>
                        `))
		d.writeStatusBadge(w, comp.Name, status, statusLabel)
		w.Write([]byte(`
                    </div>
`))
	}

	w.Write([]byte(`                </div>
            </div>
        </div>
`))

	w.Write([]byte(`    </main>
    <script src="/static/js/dashboard.js"></script>
</body>
</html>
`))
}

// handleHealth serves the health check endpoint.
func (d *Dashboard) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := struct {
		Status    string `json:"status"`
		Robot     string `json:"robot"`
		Timestamp string `json:"timestamp"`
	}{
		Status:    "ok",
		Robot:     d.robotCfg.Robot.Name,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// handleStatus serves the robot status API endpoint.
func (d *Dashboard) handleStatus(w http.ResponseWriter, r *http.Request) {
	cameraList := d.cameraMonitor.GetCameras()

	onlineCount := 0
	for _, cam := range cameraList {
		if cam.Online {
			onlineCount++
		}
	}

	status := struct {
		Robot      string `json:"robot"`
		Components int    `json:"components"`
		Services   int    `json:"services"`
		Cameras    struct {
			Total  int `json:"total"`
			Online int `json:"online"`
		} `json:"cameras"`
		Timestamp string `json:"timestamp"`
	}{
		Robot:      d.robotCfg.Robot.Name,
		Components: len(d.robotCfg.Components),
		Services:   len(d.robotCfg.Services),
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	status.Cameras.Total = len(cameraList)
	status.Cameras.Online = onlineCount

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleCamerasAPI serves the cameras API endpoint.
func (d *Dashboard) handleCamerasAPI(w http.ResponseWriter, r *http.Request) {
	cameraList := d.cameraMonitor.GetCameras()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cameraList)
}

// maxCommandBody limits the request body for command endpoints.
const maxCommandBody = 1024

// handleComponentCommand handles POST /api/components/{name}/command.
// Publishes an on/off command to the component's NATS command topic.
func (d *Dashboard) handleComponentCommand(w http.ResponseWriter, r *http.Request) {
	componentName := chi.URLParam(r, "name")
	if componentName == "" {
		http.Error(w, `{"error":"component name required"}`, http.StatusBadRequest)
		return
	}

	// Validate component exists and is not disabled
	found := false
	disabled := false
	for _, comp := range d.robotCfg.Components {
		if comp.Name == componentName {
			found = true
			disabled = comp.Disabled
			break
		}
	}
	if !found {
		http.Error(w, `{"error":"component not found"}`, http.StatusNotFound)
		return
	}
	if disabled {
		http.Error(w, `{"error":"component is disabled"}`, http.StatusBadRequest)
		return
	}

	// Validate that this component has a binary status type
	_, valueType, _, _, hasStatus := d.componentMonitor.GetStatusValue(componentName)
	if !hasStatus || valueType != "binary" {
		http.Error(w, `{"error":"component is not a binary switch"}`, http.StatusBadRequest)
		return
	}

	// Parse request body
	body, err := io.ReadAll(io.LimitReader(r.Body, maxCommandBody))
	if err != nil {
		http.Error(w, `{"error":"failed to read body"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.Command != "on" && req.Command != "off" {
		http.Error(w, `{"error":"command must be on or off"}`, http.StatusBadRequest)
		return
	}

	// Check NATS connection
	if d.nats == nil {
		http.Error(w, `{"error":"NATS not connected"}`, http.StatusServiceUnavailable)
		return
	}

	// Publish command to NATS
	commandTopic := d.topics.ComponentCommand(componentName)
	commandMsg := map[string]any{
		"command":   req.Command,
		"source":    "dashboard",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(commandMsg)
	if err != nil {
		http.Error(w, `{"error":"failed to marshal command"}`, http.StatusInternalServerError)
		return
	}

	if err := d.nats.Publish(commandTopic, data); err != nil {
		d.logger.Error("failed to publish component command", "component", componentName, "error", err)
		http.Error(w, `{"error":"failed to publish command"}`, http.StatusInternalServerError)
		return
	}

	d.logger.Info("dashboard command sent", "component", componentName, "command", req.Command)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":    "sent",
		"component": componentName,
		"command":   req.Command,
	})
}

// resolveComponentStatus returns the CSS class and display label for a component.
func (d *Dashboard) resolveComponentStatus(componentName string, cameras []cameras.CameraInfo) (cssClass string, label string) {
	// Check if it's a camera
	for _, comp := range d.robotCfg.Components {
		if comp.Name == componentName && comp.Type == "camera" {
			for _, cam := range cameras {
				if cam.Name == componentName && cam.Online {
					return "online", "online"
				}
			}
			return "offline", "offline"
		}
	}

	// Check component monitor for status_value
	statusLabel := d.componentMonitor.FormatStatusValue(componentName)
	if statusLabel == "" {
		return "active", "active"
	}

	_, valueType, _, _, ok := d.componentMonitor.GetStatusValue(componentName)
	if !ok {
		return "active", statusLabel
	}
	switch valueType {
	case "binary":
		value, _, _, _, _ := d.componentMonitor.GetStatusValue(componentName)
		if s, ok := value.(string); ok && s == "on" {
			return "online", statusLabel
		}
		return "offline", statusLabel
	default:
		return "online", statusLabel
	}
}

// writeStatusBadge writes a status badge as either a clickable <button> (for binary
// components) or a <span> (for everything else).
func (d *Dashboard) writeStatusBadge(w http.ResponseWriter, componentName, cssClass, label string) {
	value, valueType, _, _, hasStatus := d.componentMonitor.GetStatusValue(componentName)
	if hasStatus && valueType == "binary" {
		// Use the raw status_value ("on"/"off") for data-state, not the display label
		rawState := "off"
		if s, ok := value.(string); ok {
			rawState = s
		}
		w.Write([]byte(`<button class="camera-status `))
		w.Write([]byte(cssClass))
		w.Write([]byte(` component-toggle" data-component="`))
		w.Write([]byte(html.EscapeString(componentName)))
		w.Write([]byte(`" data-state="`))
		w.Write([]byte(html.EscapeString(rawState)))
		w.Write([]byte(`">`))
		w.Write([]byte(html.EscapeString(label)))
		w.Write([]byte(`</button>`))
		return
	}
	w.Write([]byte(`<span class="camera-status `))
	w.Write([]byte(cssClass))
	w.Write([]byte(`">`))
	w.Write([]byte(html.EscapeString(label)))
	w.Write([]byte(`</span>`))
}

// writeServiceDetails renders detail rows for a service (suntimes sunrise/sunset,
// light controller schedule on/off times).
func (d *Dashboard) writeServiceDetails(w http.ResponseWriter, serviceName string) {
	rows := d.serviceMonitor.FormatServiceDetail(serviceName)
	if len(rows) == 0 {
		return
	}

	for _, row := range rows {
		statusClass := "active"
		if row.Status == "active" {
			statusClass = "online"
		}

		w.Write([]byte(`                    <div class="component-item" style="padding-left: 2.5rem; border-left: 3px solid #e0e0e0;">
                        <div>
                            <div class="name">`))
		w.Write([]byte(html.EscapeString(row.Label)))
		w.Write([]byte(`</div>
                            <div class="type">`))
		w.Write([]byte(html.EscapeString(row.Value)))
		w.Write([]byte(`</div>
                        </div>
                        <span class="camera-status `))
		w.Write([]byte(statusClass))
		w.Write([]byte(`">`))
		if row.Status != "" {
			w.Write([]byte(html.EscapeString(row.Status)))
		}
		w.Write([]byte(`</span>
                    </div>
`))
	}
}

// getServiceDevices extracts the unique list of device/component names
// referenced by a service's schedules configuration.
func (d *Dashboard) getServiceDevices(svc config.ServiceConfig) []string {
	seen := make(map[string]bool)
	var devices []string

	schedules, ok := svc.Attributes["schedules"].([]any)
	if !ok {
		return nil
	}

	for _, schedRaw := range schedules {
		sched, ok := schedRaw.(map[string]any)
		if !ok {
			continue
		}
		devList, ok := sched["devices"].([]any)
		if !ok {
			continue
		}
		for _, dev := range devList {
			name, ok := dev.(string)
			if !ok || name == "" {
				continue
			}
			if !seen[name] {
				seen[name] = true
				devices = append(devices, name)
			}
		}
	}

	return devices
}

