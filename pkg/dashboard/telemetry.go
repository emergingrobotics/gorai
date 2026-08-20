package dashboard

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/emergingrobotics/gorai/pkg/config"
	gorainats "github.com/emergingrobotics/gorai/pkg/nats"
	"github.com/emergingrobotics/gorai/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// telemetryFrame is the envelope forwarded to browsers over the WebSocket. The
// "type" field lets the client demux telemetry from other /ws traffic.
type telemetryFrame struct {
	Type  string          `json:"type"`
	Topic string          `json:"topic"`
	Data  json.RawMessage `json:"data"`
}

// telemetryMonitor bridges a remote robot's NATS telemetry to the dashboard
// WebSocket hub, keeping the latest payload per component for late joiners.
type telemetryMonitor struct {
	nats    *gorainats.Client
	cfg     *config.TelemetryBridge
	logger  *slog.Logger
	hub     *WebSocketHub
	allowed map[string]bool

	mu     sync.RWMutex
	latest map[string]json.RawMessage
	subs   []*nats.Subscription
}

// newTelemetryMonitor creates a telemetry bridge. It returns nil when telemetry
// is not configured or NATS is unavailable.
func newTelemetryMonitor(cfg *config.TelemetryBridge, nc *gorainats.Client, hub *WebSocketHub, logger *slog.Logger) *telemetryMonitor {
	if cfg == nil || cfg.Robot == "" || nc == nil {
		return nil
	}
	allowed := make(map[string]bool, len(cfg.Components))
	for _, name := range cfg.Components {
		allowed[name] = true
	}
	return &telemetryMonitor{
		nats:    nc,
		cfg:     cfg,
		logger:  logger,
		hub:     hub,
		allowed: allowed,
		latest:  make(map[string]json.RawMessage),
	}
}

// Start subscribes to the target robot's data and state subjects.
func (m *telemetryMonitor) Start(ctx context.Context) error {
	builder := subjects.NewBuilder(m.cfg.Robot)
	for _, subject := range []string{
		builder.AllComponents(subjects.Data),
		builder.AllComponents(subjects.State),
	} {
		sub, err := m.nats.Subscribe(subject, m.handle)
		if err != nil {
			m.logger.Warn("failed to subscribe telemetry", "subject", subject, "error", err)
			continue
		}
		m.subs = append(m.subs, sub)
	}
	m.logger.Info("Telemetry bridge started", "robot", m.cfg.Robot, "components", m.cfg.Components)
	return nil
}

// handle relays a telemetry message to the WebSocket hub and cache.
func (m *telemetryMonitor) handle(msg *nats.Msg) {
	component := componentFromSubject(msg.Subject)
	if component == "" {
		return
	}
	if len(m.allowed) > 0 && !m.allowed[component] {
		return
	}
	if !json.Valid(msg.Data) {
		return
	}

	data := json.RawMessage(append([]byte(nil), msg.Data...))
	m.mu.Lock()
	m.latest[component] = data
	m.mu.Unlock()

	m.hub.BroadcastJSON(telemetryFrame{Type: "telemetry", Topic: component, Data: data})
}

// snapshot returns the latest payload per component.
func (m *telemetryMonitor) snapshot() map[string]json.RawMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]json.RawMessage, len(m.latest))
	for k, v := range m.latest {
		out[k] = v
	}
	return out
}

// Stop unsubscribes all telemetry subscriptions.
func (m *telemetryMonitor) Stop() {
	for _, sub := range m.subs {
		_ = sub.Unsubscribe()
	}
	m.subs = nil
}

// componentFromSubject extracts the component name from a
// gorai.<robot>.<component>.<type> subject.
func componentFromSubject(subject string) string {
	parts := strings.Split(subject, ".")
	if len(parts) < 4 {
		return ""
	}
	return parts[2]
}

// handleTelemetry returns the latest telemetry snapshot for initial paint.
func (d *Dashboard) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if d.telemetry == nil {
		w.Write([]byte("{}"))
		return
	}
	json.NewEncoder(w).Encode(d.telemetry.snapshot())
}

// writeTelemetryPanels renders the always-on telemetry panels. Values are
// populated client-side from the WebSocket stream.
func (d *Dashboard) writeTelemetryPanels(w http.ResponseWriter) {
	if d.cfg == nil || d.cfg.Telemetry == nil {
		return
	}
	w.Write([]byte(`
        <div class="telemetry-strip" id="telemetry">
            <div class="telemetry-card" data-panel="gps">
                <h4>GPS</h4>
                <div class="telemetry-row"><span>Fix</span><span data-field="fix">--</span></div>
                <div class="telemetry-row"><span>Lat</span><span data-field="latitude">--</span></div>
                <div class="telemetry-row"><span>Lon</span><span data-field="longitude">--</span></div>
                <div class="telemetry-row"><span>HDOP</span><span data-field="hdop">--</span></div>
                <div class="telemetry-row"><span>Sats</span><span data-field="satellites">--</span></div>
            </div>
            <div class="telemetry-card" data-panel="imu">
                <h4>IMU</h4>
                <div class="telemetry-row"><span>Yaw</span><span data-field="yaw">--</span></div>
                <div class="telemetry-row"><span>Roll</span><span data-field="roll">--</span></div>
                <div class="telemetry-row"><span>Pitch</span><span data-field="pitch">--</span></div>
            </div>
            <div class="telemetry-card" data-panel="bmp">
                <h4>Environment</h4>
                <div class="telemetry-row"><span>Pressure</span><span data-field="pressure_pa">--</span></div>
                <div class="telemetry-row"><span>Temp</span><span data-field="temperature_c">--</span></div>
                <div class="telemetry-row"><span>Depth/Alt</span><span data-field="altitude_m">--</span></div>
            </div>
        </div>`))
}
