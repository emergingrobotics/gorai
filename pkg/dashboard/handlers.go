package dashboard

import (
	"encoding/json"
	"html"
	"net/http"
	"strconv"
	"time"
)

// handleIndex serves the main dashboard page.
func (d *Dashboard) handleIndex(w http.ResponseWriter, r *http.Request) {
	cameras := d.cameraMonitor.GetCameras()

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

        <div style="margin-top: 2rem;">
            <div class="status-card">
                <h3>Components</h3>
                <div class="component-list">
`))

	for _, comp := range d.robotCfg.Components {
		status := "offline"
		statusLabel := "offline"
		if !comp.Disabled {
			if comp.Type == "camera" {
				for _, cam := range cameras {
					if cam.Name == comp.Name && cam.Online {
						status = "online"
						statusLabel = "online"
						break
					}
				}
			} else {
				statusLabel = d.componentStatusLabel(comp.Name)
				if statusLabel == "" {
					status = "active"
					statusLabel = "active"
				} else {
					status = d.componentStatusClass(comp.Name)
				}
			}
		} else {
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

	// Services section
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
		}

		w.Write([]byte(`                </div>
            </div>
        </div>
`))
	}

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
	cameras := d.cameraMonitor.GetCameras()

	onlineCount := 0
	for _, cam := range cameras {
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
	status.Cameras.Total = len(cameras)
	status.Cameras.Online = onlineCount

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleCamerasAPI serves the cameras API endpoint.
func (d *Dashboard) handleCamerasAPI(w http.ResponseWriter, r *http.Request) {
	cameras := d.cameraMonitor.GetCameras()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cameras)
}

// componentStatusLabel returns a formatted display string for a component's
// status value, or empty string if no status value is cached.
func (d *Dashboard) componentStatusLabel(componentName string) string {
	return d.componentMonitor.FormatStatusValue(componentName)
}

// componentStatusClass returns the CSS class for a component's status value.
func (d *Dashboard) componentStatusClass(componentName string) string {
	_, valueType, _, _, ok := d.componentMonitor.GetStatusValue(componentName)
	if !ok {
		return "active"
	}
	switch valueType {
	case "binary":
		value, _, _, _, _ := d.componentMonitor.GetStatusValue(componentName)
		if s, ok := value.(string); ok && s == "on" {
			return "online"
		}
		return "offline"
	default:
		return "online"
	}
}

