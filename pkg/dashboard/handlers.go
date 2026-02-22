package dashboard

import (
	"encoding/json"
	"html"
	"net/http"
	"strconv"
	"time"

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
			if svc.Disabled {
				status = "disabled"
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
			w.Write([]byte(`">`))
			w.Write([]byte(status))
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
                        <span class="camera-status `))
					w.Write([]byte(devStatus))
					w.Write([]byte(`">`))
					w.Write([]byte(html.EscapeString(devLabel)))
					w.Write([]byte(`</span>
                    </div>
`))
				}
			}
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
                        <span class="camera-status `))
		w.Write([]byte(status))
		w.Write([]byte(`">`))
		w.Write([]byte(html.EscapeString(statusLabel)))
		w.Write([]byte(`</span>
                    </div>
`))
	}

	w.Write([]byte(`                </div>
            </div>
        </div>
`))

	w.Write([]byte(`    </main>
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

