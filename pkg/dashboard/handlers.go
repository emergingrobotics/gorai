package dashboard

import (
	"encoding/json"
	"net/http"
	"time"
)

// handleIndex serves the main dashboard page.
func (d *Dashboard) handleIndex(w http.ResponseWriter, r *http.Request) {
	cameras := d.cameraMonitor.GetCameras()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Gorai Dashboard</title>
    <link rel="stylesheet" href="/static/css/main.css">
</head>
<body>
    <nav class="nav">
        <div class="nav-brand">Gorai</div>
        <ul class="nav-tabs">
            <li><a href="/" class="active">Status</a></li>
            <li><a href="/cameras">Cameras</a></li>
        </ul>
    </nav>
    <main>
        <div class="status-grid">
            <div class="status-card">
                <h3>Robot</h3>
                <div class="value">`))
	w.Write([]byte(d.robotCfg.Robot.Name))
	w.Write([]byte(`</div>
                <div class="label">Name</div>
            </div>
            <div class="status-card">
                <h3>Components</h3>
                <div class="value">`))
	w.Write([]byte(itoa(len(d.robotCfg.Components))))
	w.Write([]byte(`</div>
                <div class="label">Configured</div>
            </div>
            <div class="status-card">
                <h3>Cameras</h3>
                <div class="value">`))
	onlineCount := 0
	for _, cam := range cameras {
		if cam.Online {
			onlineCount++
		}
	}
	w.Write([]byte(itoa(onlineCount)))
	w.Write([]byte(` / `))
	w.Write([]byte(itoa(len(cameras))))
	w.Write([]byte(`</div>
                <div class="label">Online</div>
            </div>
        </div>

        <div style="margin-top: 2rem;">
            <div class="status-card">
                <h3>Components</h3>
                <div class="component-list">
`))

	for _, comp := range d.robotCfg.Components {
		status := "offline"
		if !comp.Disabled {
			// Check if it's a camera and if it's online
			if comp.Type == "camera" {
				for _, cam := range cameras {
					if cam.Name == comp.Name && cam.Online {
						status = "online"
						break
					}
				}
			} else {
				status = "active"
			}
		} else {
			status = "disabled"
		}

		w.Write([]byte(`                    <div class="component-item">
                        <div>
                            <div class="name">`))
		w.Write([]byte(comp.Name))
		w.Write([]byte(`</div>
                            <div class="type">`))
		w.Write([]byte(comp.Type))
		if comp.Model != "" {
			w.Write([]byte(` / `))
			w.Write([]byte(comp.Model))
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
    </main>
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

// itoa converts an int to a string (simple implementation to avoid strconv).
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	if i < 0 {
		return "-" + itoa(-i)
	}
	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}
