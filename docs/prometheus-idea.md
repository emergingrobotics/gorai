# Prometheus Integration for Gorai Monitoring

**Status**: Proposal / Analysis
**Date**: 2025-12-11

## Executive Summary

This document evaluates running Prometheus **locally on the robot** as a dependency alongside NATS and Gorai. In this model:

- **Prometheus** runs on the robot as the time-series data store
- **Gorai** pushes metrics to local Prometheus (or Prometheus scrapes localhost)
- **Dashboard** queries local Prometheus API for both Gauges and History
- **Alert Manager** (optional) runs locally for robot-level alerting

This is **viable and has significant advantages** over an in-memory buffer approach:

1. **Persistent storage** — data survives Gorai restarts
2. **Efficient compression** — 1-2 bytes per sample vs raw storage
3. **Powerful queries** — PromQL for aggregations, rates, histograms
4. **Mature alerting** — Alert Manager for sophisticated robot alerts
5. **Fleet-ready** — same metrics can be federated to central Prometheus later

**Recommendation**: Adopt Prometheus as a local dependency, similar to NATS.

---

## Revised Architecture

```
┌──────────────────────────────────────────────────────────────────────────┐
│ Robot (Single Machine)                                                    │
│                                                                           │
│  ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐    │
│  │ Gorai Binary    │     │ NATS Server     │     │ Prometheus      │    │
│  │                 │     │ (nats-server)   │     │ (prometheus)    │    │
│  │ Components:     │     │                 │     │                 │    │
│  │  - Sensors      │────▶│ Message Bus     │     │ Time-Series DB  │    │
│  │  - Actuators    │     │ Real-time       │     │ /metrics scrape │    │
│  │  - Services     │     │ Pub/Sub         │     │ or push         │    │
│  │                 │     │                 │     │                 │    │
│  │ Dashboard:      │     └─────────────────┘     │ Retention:      │    │
│  │  - HTTP :8080   │                             │ 15d default     │    │
│  │  - WebSocket    │◀────────────────────────────│ (configurable)  │    │
│  │                 │     Query PromQL API        │                 │    │
│  │ Metrics:        │                             │                 │    │
│  │  - /metrics     │────────────────────────────▶│ Scrape :9090    │    │
│  │    :9090        │     (or push via remote_write)                │    │
│  └─────────────────┘                             └────────┬────────┘    │
│                                                           │             │
│  ┌────────────────────────────────────────────────────────▼──────────┐ │
│  │ Alert Manager (optional)                                           │ │
│  │  - Evaluate alert rules                                           │ │
│  │  - Local notifications (webhook, file, log)                       │ │
│  │  - Or forward to central Alert Manager                            │ │
│  └───────────────────────────────────────────────────────────────────┘ │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
                              │
                              │ HTTP :8080
                              ▼
┌──────────────────────────────────────────────────────────────────────────┐
│ Browser                                                                   │
│                                                                           │
│  Dashboard queries Prometheus via Gorai's API proxy:                     │
│   - Gauges: GET /api/sensors/latest → Prometheus instant query           │
│   - History: GET /api/sensors/history?range=5m → Prometheus range query  │
│                                                                           │
│  Or dashboard queries Prometheus directly:                               │
│   - http://robot:9090/api/v1/query                                       │
│   - http://robot:9090/api/v1/query_range                                 │
│                                                                           │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## Dependencies Model

Just as NATS is a required dependency for Gorai's messaging, Prometheus becomes a required (or strongly recommended) dependency for monitoring:

| Dependency | Purpose | Deployment |
|------------|---------|------------|
| **NATS** | Real-time messaging, pub/sub | Required, runs on robot |
| **Prometheus** | Time-series storage, queries | Required, runs on robot |
| **Alert Manager** | Alerting | Optional, runs on robot |

### Resource Requirements (On Robot)

| Component | RAM | CPU | Disk |
|-----------|-----|-----|------|
| Gorai | 50-200 MB | Variable | Minimal |
| NATS | 10-50 MB | Low | Minimal (unless JetStream) |
| Prometheus | 200-500 MB | Low | ~2 GB/month at 50 sensors × 1Hz |
| Alert Manager | 50 MB | Minimal | Minimal |
| **Total** | **~500 MB** | Low | ~2 GB/month |

This is reasonable for typical robot SBCs:
- Raspberry Pi 4 (4GB): Comfortable
- Jetson Nano (4GB): Comfortable
- Rock 5B (8-16GB): Plenty of headroom
- x86 mini-PC: No problem

---

## Data Flow

### Option A: Prometheus Scrapes Gorai

```
Gorai exposes /metrics endpoint
           │
           │ Prometheus scrapes every 5-15s
           ▼
┌─────────────────────────────────────┐
│ Prometheus                          │
│  - Stores time-series              │
│  - Compresses efficiently          │
│  - Handles retention/compaction    │
└─────────────────────────────────────┘
           │
           │ Dashboard queries via PromQL
           ▼
┌─────────────────────────────────────┐
│ Gorai Dashboard                     │
│  - Gauges: instant query           │
│  - History: range query            │
└─────────────────────────────────────┘
```

**prometheus.yml**:
```yaml
global:
  scrape_interval: 5s      # Fast for robotics
  evaluation_interval: 5s

scrape_configs:
  - job_name: 'gorai'
    static_configs:
      - targets: ['localhost:9090']
```

### Option B: Gorai Pushes to Prometheus (Remote Write)

```
Gorai pushes metrics via remote_write
           │
           │ HTTP POST every 5-10s
           ▼
┌─────────────────────────────────────┐
│ Prometheus (with remote write      │
│ receiver enabled)                  │
└─────────────────────────────────────┘
```

This avoids the pull model complexity but requires Prometheus `--web.enable-remote-write-receiver` flag.

### Recommended: Option A (Scrape)

Simpler, standard Prometheus deployment. Since everything is localhost, scrape latency is negligible.

---

## Latency Analysis

With Prometheus on localhost:

| Operation | Latency | Notes |
|-----------|---------|-------|
| Scrape (localhost) | <1ms | Network overhead eliminated |
| Instant query | 1-5ms | Single value lookup |
| Range query (5min) | 5-20ms | Depends on data volume |
| Range query (1hr) | 20-100ms | Still fast locally |

**For Gauges (current values)**:
```
Dashboard → Gorai API → Prometheus instant query → Response
Total: ~5-10ms
```

This is fast enough for "real-time" display. No need for separate WebSocket path.

**For History (time-series)**:
```
Dashboard → Gorai API → Prometheus range query → Response
Total: ~10-50ms
```

Also very fast. uPlot can handle this easily.

### Comparison with In-Memory Buffer

| Aspect | In-Memory Buffer | Local Prometheus |
|--------|------------------|------------------|
| Gauge latency | <1ms | 5-10ms |
| History latency | <1ms | 10-50ms |
| Persistence | No (lost on restart) | Yes |
| Retention | RAM-limited (5min typical) | Disk-based (days/weeks) |
| Query capability | Simple (latest, range) | Full PromQL |
| Compression | None | ~1-2 bytes/sample |
| Aggregations | Manual code | PromQL built-in |

**The 5-10ms latency increase is imperceptible to humans** and the benefits far outweigh it.

---

## Simplified Dashboard Implementation

With Prometheus as the data store, the dashboard becomes simpler:

```go
package dashboard

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/prometheus/client_golang/api"
    promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type Server struct {
    config     Config
    promClient promv1.API
    router     *chi.Mux
}

func New(config Config) (*Server, error) {
    // Connect to local Prometheus
    client, err := api.NewClient(api.Config{
        Address: config.PrometheusURL, // "http://localhost:9090"
    })
    if err != nil {
        return nil, err
    }

    s := &Server{
        config:     config,
        promClient: promv1.NewAPI(client),
        router:     chi.NewRouter(),
    }

    s.setupRoutes()
    return s, nil
}

func (s *Server) setupRoutes() {
    s.router.Get("/", s.handleHome)
    s.router.Get("/topology", s.handleTopology)
    s.router.Get("/sensors", s.handleSensors)
    s.router.Get("/cameras", s.handleCameras)
    s.router.Get("/inference", s.handleInference)

    // API endpoints that query Prometheus
    s.router.Get("/api/gauges", s.handleGauges)
    s.router.Get("/api/history", s.handleHistory)
}

// handleGauges returns current values for all sensors
func (s *Server) handleGauges(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Instant query for current values
    result, _, err := s.promClient.Query(ctx, `gorai_sensor_value`, time.Now())
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    json.NewEncoder(w).Encode(result)
}

// handleHistory returns time-series data for a sensor
func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    sensor := r.URL.Query().Get("sensor")
    rangeStr := r.URL.Query().Get("range")

    duration, _ := time.ParseDuration(rangeStr)
    if duration == 0 {
        duration = 5 * time.Minute
    }

    // Range query for history
    query := fmt.Sprintf(`gorai_sensor_value{sensor="%s"}`, sensor)
    result, _, err := s.promClient.QueryRange(ctx, query, promv1.Range{
        Start: time.Now().Add(-duration),
        End:   time.Now(),
        Step:  time.Second, // 1 second resolution
    })
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    json.NewEncoder(w).Encode(result)
}
```

**No circular buffer code needed** — Prometheus handles storage, retention, and compression.

---

## PromQL Query Examples

The dashboard can leverage PromQL for sophisticated queries:

### Gauges (Current Values)

```promql
# All sensor values
gorai_sensor_value

# Specific sensor
gorai_sensor_value{sensor="imu", field="accel_x"}

# All component states
gorai_component_state

# Components in error state
gorai_component_state == -1
```

### History (Time-Series)

```promql
# IMU acceleration over last 5 minutes
gorai_sensor_value{sensor="imu"}[5m]

# Rate of change (derivative)
deriv(gorai_sensor_value{sensor="battery_level"}[5m])

# Average over time window
avg_over_time(gorai_sensor_value{sensor="temperature"}[1m])

# 99th percentile inference latency
histogram_quantile(0.99, rate(gorai_inference_duration_seconds_bucket[5m]))
```

### Aggregations

```promql
# Average across all temperature sensors
avg(gorai_sensor_value{field=~"temperature.*"})

# Max CPU usage over last hour
max_over_time(gorai_cpu_usage[1h])

# Detection rate (detections per second)
rate(gorai_detections_total[1m])
```

---

## Alert Manager Integration

With Prometheus local, Alert Manager becomes straightforward:

### alertmanager.yml

```yaml
global:
  resolve_timeout: 5m

route:
  receiver: 'local'
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 1h

receivers:
  - name: 'local'
    webhook_configs:
      # Notify Gorai's own webhook for local handling
      - url: 'http://localhost:8080/api/alerts'
        send_resolved: true
```

### prometheus/alerts.yml

```yaml
groups:
  - name: robot_alerts
    rules:
      - alert: BatteryLow
        expr: gorai_sensor_value{sensor="battery", field="level"} < 0.2
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Battery below 20%"

      - alert: BatteryCritical
        expr: gorai_sensor_value{sensor="battery", field="level"} < 0.1
        for: 30s
        labels:
          severity: critical
        annotations:
          summary: "Battery below 10% - return to base"

      - alert: MotorOverheat
        expr: gorai_sensor_value{sensor=~"motor.*", field="temperature"} > 80
        for: 30s
        labels:
          severity: critical
        annotations:
          summary: "Motor {{ $labels.sensor }} overheating"

      - alert: InferenceSlow
        expr: |
          histogram_quantile(0.99,
            rate(gorai_inference_duration_seconds_bucket[5m])
          ) > 0.2
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Inference latency > 200ms"

      - alert: SensorOffline
        expr: |
          time() - gorai_sensor_last_reading_timestamp > 30
        for: 30s
        labels:
          severity: warning
        annotations:
          summary: "Sensor {{ $labels.sensor }} not reporting"

      - alert: HighErrorRate
        expr: rate(gorai_errors_total[5m]) > 0.1
        for: 1m
        labels:
          severity: warning
```

### Gorai Alert Handler

```go
// Handle alerts from Alert Manager
func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
    var alerts []Alert
    json.NewDecoder(r.Body).Decode(&alerts)

    for _, alert := range alerts {
        switch alert.Labels["severity"] {
        case "critical":
            // Take immediate action
            s.publishToNATS("gorai.alerts.critical", alert)
            // Maybe trigger emergency stop
            if alert.Labels["alertname"] == "BatteryCritical" {
                s.triggerReturnToBase()
            }
        case "warning":
            s.publishToNATS("gorai.alerts.warning", alert)
        }

        // Log to dashboard
        s.broadcastAlert(alert)
    }

    w.WriteHeader(http.StatusOK)
}
```

---

## Configuration

### RDL Schema Update

```json
{
  "robot": {
    "name": "sentinel"
  },

  "nats": {
    "url": "nats://localhost:4222"
  },

  "prometheus": {
    "url": "http://localhost:9090",
    "metrics_path": "/metrics",
    "metrics_port": 9091
  },

  "alerting": {
    "enabled": true,
    "alertmanager_url": "http://localhost:9093",
    "rules_file": "/etc/gorai/alerts.yml"
  },

  "dashboard": {
    "enabled": true,
    "listen": ":8080",
    "retention": "15d"
  }
}
```

Note: `retention` now configures Prometheus retention, not in-memory buffer.

### Prometheus Configuration

**/etc/prometheus/prometheus.yml**:
```yaml
global:
  scrape_interval: 5s
  evaluation_interval: 5s

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['localhost:9093']

rule_files:
  - /etc/gorai/alerts.yml

scrape_configs:
  - job_name: 'gorai'
    static_configs:
      - targets: ['localhost:9091']
    relabel_configs:
      - source_labels: [__address__]
        target_label: robot
        replacement: 'sentinel'
```

---

## Deployment

### Systemd Services

**/etc/systemd/system/gorai.service**:
```ini
[Unit]
Description=Gorai Robot Framework
After=network.target nats.service prometheus.service
Requires=nats.service prometheus.service

[Service]
Type=simple
ExecStart=/usr/local/bin/gorai --config /etc/gorai/robot.json
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

**/etc/systemd/system/prometheus.service**:
```ini
[Unit]
Description=Prometheus
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/prometheus \
  --config.file=/etc/prometheus/prometheus.yml \
  --storage.tsdb.path=/var/lib/prometheus \
  --storage.tsdb.retention.time=15d \
  --web.listen-address=:9090
Restart=always

[Install]
WantedBy=multi-user.target
```

### Container Deployment

**docker-compose.yml** (or Podman equivalent):
```yaml
version: '3.8'

services:
  nats:
    image: nats:latest
    ports:
      - "4222:4222"
    command: ["--jetstream"]

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - ./alerts.yml:/etc/prometheus/alerts.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.retention.time=15d'

  alertmanager:
    image: prom/alertmanager:latest
    ports:
      - "9093:9093"
    volumes:
      - ./alertmanager.yml:/etc/alertmanager/alertmanager.yml

  gorai:
    image: gorai:latest
    ports:
      - "8080:8080"
      - "9091:9091"
    depends_on:
      - nats
      - prometheus
    volumes:
      - ./robot.json:/etc/gorai/robot.json

volumes:
  prometheus_data:
```

---

## Fleet Monitoring (Future)

With Prometheus on each robot, federation to a central Prometheus is straightforward:

```
┌─────────┐ ┌─────────┐ ┌─────────┐
│ Robot 1 │ │ Robot 2 │ │ Robot 3 │
│ Prom    │ │ Prom    │ │ Prom    │
│ :9090   │ │ :9090   │ │ :9090   │
└────┬────┘ └────┬────┘ └────┬────┘
     │           │           │
     └───────────┼───────────┘
                 │ Federation scrape
                 ▼
┌─────────────────────────────────────┐
│ Central Prometheus                  │
│  - Aggregates fleet metrics         │
│  - Long-term storage (Thanos/Mimir) │
│  - Fleet-wide dashboards            │
└─────────────────────────────────────┘
```

Or use remote_write from each robot:
```yaml
# On each robot's prometheus.yml
remote_write:
  - url: "https://central-prometheus.example.com/api/v1/write"
```

---

## Advantages Summary

| Aspect | In-Memory Buffer | Local Prometheus |
|--------|------------------|------------------|
| **Persistence** | Lost on restart | Survives restarts |
| **Retention** | 5 minutes (RAM limited) | 15+ days (disk) |
| **Storage efficiency** | ~8 bytes/sample | ~1-2 bytes/sample |
| **Query language** | Custom code | PromQL (powerful) |
| **Aggregations** | Manual implementation | Built-in (avg, rate, quantile) |
| **Alerting** | Build from scratch | Alert Manager (mature) |
| **Fleet-ready** | No | Federation/remote_write |
| **Ecosystem** | None | Grafana, exporters, tools |
| **Code complexity** | More (buffer management) | Less (query API) |

---

## Disadvantages / Considerations

### 1. Additional Dependency

Prometheus is another binary to install and manage. However:
- Single static binary, easy to install
- Well-documented, stable
- Same operational model as NATS

### 2. Resource Overhead

~200-500 MB RAM for Prometheus. Acceptable for most robot SBCs, but might be tight on very constrained devices (e.g., Raspberry Pi Zero).

**Mitigation**: Could make Prometheus optional for extremely constrained deployments, falling back to in-memory buffer.

### 3. Scrape Interval

5-15s scrape interval means data is not truly "instant". For most robotics monitoring, this is fine:
- Human-readable dashboards don't need <1s updates
- Alerts can still trigger within seconds

For truly real-time data (control loops), continue using NATS directly — Prometheus is for monitoring, not control.

### 4. High-Frequency Data

100Hz IMU data shouldn't go directly to Prometheus. Pre-aggregate:

```go
// Don't: Export every IMU sample
// Do: Export 1Hz statistics
var imuStats struct {
    AccelXMean, AccelXMax, AccelXMin float64
    // computed over 1 second window
}

sensorValue.WithLabelValues("imu", "accel_x_mean").Set(imuStats.AccelXMean)
sensorValue.WithLabelValues("imu", "accel_x_max").Set(imuStats.AccelXMax)
```

---

## Recommendation

**Adopt Prometheus as a local dependency** for the following reasons:

1. **Simpler code**: No custom circular buffer, retention logic, or query implementation
2. **Persistence**: Data survives Gorai restarts — critical for debugging
3. **Powerful queries**: PromQL enables aggregations, rates, and percentiles
4. **Mature alerting**: Alert Manager is production-ready
5. **Fleet-ready**: Same architecture scales from one robot to many
6. **Ecosystem**: Grafana dashboards, exporters, documentation
7. **Go-native**: Excellent client libraries, same language as Gorai

### Migration Path

1. **Phase 1**: Add Prometheus as required dependency, implement metrics exporter
2. **Phase 2**: Update dashboard to query Prometheus instead of in-memory buffer
3. **Phase 3**: Add Alert Manager integration
4. **Phase 4**: Document federation for fleet monitoring

### Configuration Default

```json
{
  "prometheus": {
    "url": "http://localhost:9090"
  },
  "dashboard": {
    "enabled": true,
    "listen": ":8080"
  }
}
```

Prometheus becomes a peer dependency like NATS — required for full functionality.

---

## References

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Prometheus Go Client](https://github.com/prometheus/client_golang)
- [Alert Manager](https://prometheus.io/docs/alerting/latest/alertmanager/)
- [Prometheus Federation](https://prometheus.io/docs/prometheus/latest/federation/)
- [Remote Write](https://prometheus.io/docs/concepts/remote_write_spec/)
