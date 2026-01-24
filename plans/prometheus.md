# Prometheus Integration Implementation Plan

This document outlines the implementation steps for integrating Prometheus as a mandatory dependency in Gorai.

## Overview

Prometheus will run locally on each robot alongside NATS and the Gorai binary, providing:

- Time-series storage for all sensor data
- Metrics exposition via `/metrics` endpoint
- Query interface for the web dashboard
- Alerting via Alert Manager (optional)

## Architecture

```
Robot (Single Machine)
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  ┌─────────────────┐    ┌─────────────────┐                │
│  │  Gorai Binary   │    │   NATS Server   │                │
│  │                 │    │    (:4222)      │                │
│  │  :8080 dashboard│    └─────────────────┘                │
│  │  :9091 /metrics │                                       │
│  └────────┬────────┘                                       │
│           │ scrape every 5s                                │
│           ▼                                                │
│  ┌─────────────────┐    ┌─────────────────┐                │
│  │   Prometheus    │───▶│  Alert Manager  │                │
│  │    (:9090)      │    │    (:9093)      │                │
│  │                 │    │   (optional)    │                │
│  └─────────────────┘    └─────────────────┘                │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Implementation Phases

### Phase 1: Metrics Exposition

**Goal**: Expose Gorai metrics in Prometheus format

#### 1.1 Add Prometheus Client Library

```go
// go.mod
require (
    github.com/prometheus/client_golang v1.17.0
)
```

#### 1.2 Create Metrics Package

Location: `pkg/metrics/metrics.go`

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // Sensor metrics
    SensorValue = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Namespace: "gorai",
            Name:      "sensor_value",
            Help:      "Current sensor reading",
        },
        []string{"robot", "sensor", "unit"},
    )

    SensorTimestamp = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Namespace: "gorai",
            Name:      "sensor_reading_timestamp",
            Help:      "Unix timestamp of last sensor reading",
        },
        []string{"robot", "sensor"},
    )

    // Component metrics
    ComponentState = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Namespace: "gorai",
            Name:      "component_state",
            Help:      "Component state (1=active, 0=inactive)",
        },
        []string{"robot", "component", "state"},
    )

    ComponentErrors = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "gorai",
            Name:      "component_errors_total",
            Help:      "Total component errors",
        },
        []string{"robot", "component"},
    )

    // System metrics
    MessagesTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "gorai",
            Name:      "messages_total",
            Help:      "Total messages sent/received",
        },
        []string{"robot", "direction"},
    )

    InferenceDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Namespace: "gorai",
            Name:      "inference_duration_seconds",
            Help:      "ML inference latency",
            Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0},
        },
        []string{"robot", "model"},
    )
)
```

#### 1.3 Add Metrics HTTP Handler

Location: `pkg/metrics/server.go`

```go
package metrics

import (
    "net/http"

    "github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
    addr string
    srv  *http.Server
}

func NewServer(addr string) *Server {
    mux := http.NewServeMux()
    mux.Handle("/metrics", promhttp.Handler())

    return &Server{
        addr: addr,
        srv: &http.Server{
            Addr:    addr,
            Handler: mux,
        },
    }
}

func (s *Server) Start() error {
    return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
    return s.srv.Shutdown(ctx)
}
```

#### 1.4 Integrate with Node

Location: `pkg/node/node.go`

```go
type Node struct {
    // existing fields...
    metricsServer *metrics.Server
}

func (n *Node) startMetrics() error {
    port := n.config.Prometheus.MetricsPort
    if port == 0 {
        port = 9091
    }
    n.metricsServer = metrics.NewServer(fmt.Sprintf(":%d", port))
    go n.metricsServer.Start()
    return nil
}
```

#### Tasks:
- [ ] Create `pkg/metrics/metrics.go` with metric definitions
- [ ] Create `pkg/metrics/server.go` with HTTP handler
- [ ] Add `prometheus` config section to node configuration
- [ ] Integrate metrics server startup in node initialization
- [ ] Update sensor base to record metrics on publish
- [ ] Update component base to record state changes
- [ ] Add message counters to NATS wrapper

---

### Phase 2: Configuration

**Goal**: Add Prometheus configuration to RDL

#### 2.1 Config Structures

Location: `pkg/config/prometheus.go`

```go
package config

type PrometheusConfig struct {
    URL            string            `json:"url"`
    MetricsPort    int               `json:"metrics_port"`
    MetricsPath    string            `json:"metrics_path"`
    ScrapeInterval string            `json:"scrape_interval"`
    Retention      string            `json:"retention"`
    Labels         map[string]string `json:"labels"`
}

type AlertingConfig struct {
    Enabled            bool   `json:"enabled"`
    AlertManagerURL    string `json:"alertmanager_url"`
    RulesFile          string `json:"rules_file"`
    EvaluationInterval string `json:"evaluation_interval"`
}

func DefaultPrometheusConfig() PrometheusConfig {
    return PrometheusConfig{
        URL:            "http://localhost:9090",
        MetricsPort:    9091,
        MetricsPath:    "/metrics",
        ScrapeInterval: "5s",
        Retention:      "15d",
        Labels:         make(map[string]string),
    }
}
```

#### 2.2 Update Robot Config

```go
type RobotConfig struct {
    // existing fields...
    Prometheus PrometheusConfig `json:"prometheus"`
    Alerting   AlertingConfig   `json:"alerting"`
}
```

#### Tasks:
- [ ] Create `pkg/config/prometheus.go`
- [ ] Add Prometheus section to RobotConfig
- [ ] Add Alerting section to RobotConfig
- [ ] Update config validation
- [ ] Add default values

---

### Phase 3: Dashboard Integration

**Goal**: Update dashboard to query Prometheus

#### 3.1 Prometheus Client

Location: `pkg/dashboard/prometheus.go`

```go
package dashboard

import (
    "context"
    "time"

    "github.com/prometheus/client_golang/api"
    promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
    "github.com/prometheus/common/model"
)

type PrometheusClient struct {
    api promv1.API
}

func NewPrometheusClient(url string) (*PrometheusClient, error) {
    client, err := api.NewClient(api.Config{Address: url})
    if err != nil {
        return nil, err
    }
    return &PrometheusClient{api: promv1.NewAPI(client)}, nil
}

// QueryGauges returns current values for all sensors
func (c *PrometheusClient) QueryGauges(ctx context.Context) (model.Value, error) {
    return c.api.Query(ctx, `gorai_sensor_value`, time.Now())
}

// QueryHistory returns time-series data for a sensor
func (c *PrometheusClient) QueryHistory(ctx context.Context, sensor string, duration time.Duration) (model.Value, error) {
    query := fmt.Sprintf(`gorai_sensor_value{sensor="%s"}`, sensor)
    return c.api.QueryRange(ctx, query, promv1.Range{
        Start: time.Now().Add(-duration),
        End:   time.Now(),
        Step:  time.Second,
    })
}

// QueryAlerts returns active alerts
func (c *PrometheusClient) QueryAlerts(ctx context.Context) (promv1.AlertsResult, error) {
    return c.api.Alerts(ctx)
}
```

#### 3.2 Update Dashboard Server

Location: `pkg/dashboard/server.go`

```go
type Server struct {
    config     Config
    promClient *PrometheusClient
    router     *chi.Mux
}

func New(config Config, prometheusURL string) (*Server, error) {
    if !config.Enabled {
        return nil, nil
    }

    promClient, err := NewPrometheusClient(prometheusURL)
    if err != nil {
        return nil, fmt.Errorf("connecting to prometheus: %w", err)
    }

    s := &Server{
        config:     config,
        promClient: promClient,
        router:     chi.NewRouter(),
    }
    s.setupRoutes()
    return s, nil
}
```

#### 3.3 API Handlers

```go
func (s *Server) handleGauges(w http.ResponseWriter, r *http.Request) {
    result, err := s.promClient.QueryGauges(r.Context())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(result)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
    sensor := r.URL.Query().Get("sensor")
    rangeStr := r.URL.Query().Get("range")

    duration, _ := time.ParseDuration(rangeStr)
    if duration == 0 {
        duration = 5 * time.Minute
    }

    result, err := s.promClient.QueryHistory(r.Context(), sensor, duration)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(result)
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
    result, err := s.promClient.QueryAlerts(r.Context())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(result)
}
```

#### Tasks:
- [ ] Create `pkg/dashboard/prometheus.go` with Prometheus client wrapper
- [ ] Update dashboard server to use Prometheus client
- [ ] Create `/api/gauges` endpoint
- [ ] Create `/api/history` endpoint
- [ ] Create `/api/alerts` endpoint
- [ ] Remove in-memory buffer code (no longer needed)

---

### Phase 4: Templates & UI

**Goal**: Create Templ templates and static assets

#### 4.1 Template Structure

```
pkg/dashboard/
├── templates/
│   ├── layout.templ       # Base layout
│   ├── home.templ         # Home page
│   ├── gauges.templ       # Gauges partial
│   ├── history.templ      # History partial
│   ├── alerts.templ       # Alerts page
│   └── cameras.templ      # Camera streams
└── static/
    ├── htmx.min.js        # HTMX (14KB)
    ├── uplot.min.js       # uPlot (50KB)
    ├── uplot.min.css
    └── style.css          # Custom styles
```

#### 4.2 Home Template

Location: `pkg/dashboard/templates/home.templ`

```go
package templates

templ Home(robotName string) {
    @Layout("Dashboard - " + robotName) {
        <div class="dashboard">
            <h1>{ robotName }</h1>

            <section class="gauges">
                <h2>Current Values</h2>
                <div hx-get="/partials/gauges"
                     hx-trigger="load, every 2s"
                     hx-swap="innerHTML">
                    Loading...
                </div>
            </section>

            <section class="history">
                <h2>History</h2>
                <select id="sensor-select"
                        hx-get="/partials/history"
                        hx-trigger="change"
                        hx-target="#history-chart">
                    <option value="">Select sensor...</option>
                </select>
                <div id="history-chart"></div>
            </section>

            <section class="alerts">
                <h2>Alerts</h2>
                <div hx-get="/partials/alerts"
                     hx-trigger="load, every 10s"
                     hx-swap="innerHTML">
                    Loading...
                </div>
            </section>
        </div>
    }
}
```

#### Tasks:
- [ ] Install Templ: `go install github.com/a-h/templ/cmd/templ@latest`
- [ ] Create `pkg/dashboard/templates/layout.templ`
- [ ] Create `pkg/dashboard/templates/home.templ`
- [ ] Create `pkg/dashboard/templates/gauges.templ`
- [ ] Create `pkg/dashboard/templates/history.templ`
- [ ] Create `pkg/dashboard/templates/alerts.templ`
- [ ] Create `pkg/dashboard/templates/cameras.templ`
- [ ] Download and embed HTMX
- [ ] Download and embed uPlot
- [ ] Create custom CSS
- [ ] Add `//go:generate templ generate` directive
- [ ] Update Makefile with templ generate step

---

### Phase 5: Alert Integration

**Goal**: Integrate with Prometheus Alert Manager

#### 5.1 Alert Webhook Handler

Location: `pkg/dashboard/alerts.go`

```go
package dashboard

import (
    "encoding/json"
    "net/http"
)

type AlertWebhook struct {
    Alerts []Alert `json:"alerts"`
}

type Alert struct {
    Status      string            `json:"status"`
    Labels      map[string]string `json:"labels"`
    Annotations map[string]string `json:"annotations"`
    StartsAt    string            `json:"startsAt"`
    EndsAt      string            `json:"endsAt"`
}

func (s *Server) handleAlertWebhook(w http.ResponseWriter, r *http.Request) {
    var webhook AlertWebhook
    if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Process alerts (log, notify, update UI)
    for _, alert := range webhook.Alerts {
        s.processAlert(alert)
    }

    w.WriteHeader(http.StatusOK)
}

func (s *Server) processAlert(alert Alert) {
    // Log alert
    // Could also: send to NATS, trigger LED, play sound, etc.
}
```

#### Tasks:
- [ ] Create `pkg/dashboard/alerts.go` with webhook handler
- [ ] Add `/api/alerts/webhook` endpoint
- [ ] Create alert processing logic
- [ ] Create default alert rules file template
- [ ] Document Alert Manager configuration

---

### Phase 6: Documentation & Examples

**Goal**: Complete documentation and examples

#### 6.1 Documentation Updates

- [ ] Update installation guide with Prometheus
- [ ] Create Prometheus configuration reference
- [ ] Create alerting cookbook with common rules
- [ ] Update API reference with metrics endpoint
- [ ] Add troubleshooting section

#### 6.2 Example Alert Rules

Location: `examples/alerts/gorai-alerts.yml`

```yaml
groups:
  - name: gorai-critical
    rules:
      - alert: RobotOffline
        expr: up{job="gorai"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Robot {{ $labels.instance }} is offline"

      - alert: LowBattery
        expr: gorai_sensor_value{sensor="battery"} < 20
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Battery low on {{ $labels.robot }}"
          description: "Battery at {{ $value }}%"

      - alert: MotorStalled
        expr: gorai_component_state{component=~"motor.*",state="stalled"} == 1
        for: 10s
        labels:
          severity: critical
        annotations:
          summary: "Motor stalled on {{ $labels.robot }}"

  - name: gorai-performance
    rules:
      - alert: HighInferenceLatency
        expr: histogram_quantile(0.95, gorai_inference_duration_seconds_bucket) > 0.5
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High inference latency on {{ $labels.robot }}"
```

#### Tasks:
- [ ] Create `examples/alerts/gorai-alerts.yml`
- [ ] Create `examples/prometheus/prometheus.yml`
- [ ] Create `examples/alertmanager/alertmanager.yml`
- [ ] Update hello-sensor example with metrics
- [ ] Add dashboard screenshots to docs

---

### Phase 7: Testing

**Goal**: Comprehensive test coverage

#### 7.1 Unit Tests

- [ ] Test metric registration and updates
- [ ] Test Prometheus client wrapper
- [ ] Test dashboard API handlers
- [ ] Test alert webhook processing
- [ ] Test config parsing and defaults

#### 7.2 Integration Tests

- [ ] Test metrics endpoint returns valid Prometheus format
- [ ] Test dashboard queries against real Prometheus
- [ ] Test alert rule evaluation
- [ ] Test end-to-end sensor → metric → query flow

#### 7.3 Test Helpers

Location: `pkg/testutil/prometheus.go`

```go
package testutil

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/testutil"
)

// AssertMetricValue checks a gauge value
func AssertMetricValue(t *testing.T, collector prometheus.Collector, expected float64) {
    if err := testutil.CollectAndCompare(collector, expected); err != nil {
        t.Errorf("metric value mismatch: %v", err)
    }
}
```

---

## Dependencies

### Go Modules

```go
require (
    github.com/prometheus/client_golang v1.17.0
    github.com/prometheus/common v0.45.0
    github.com/go-chi/chi/v5 v5.0.10
    github.com/a-h/templ v0.2.501
)
```

### External

| Dependency | Version | Purpose |
|------------|---------|---------|
| Prometheus | 2.48+ | Time-series database |
| Alert Manager | 0.26+ | Alerting (optional) |
| HTMX | 1.9+ | UI updates |
| uPlot | 1.6+ | Charts |

---

## File Checklist

### New Files

- [ ] `pkg/metrics/metrics.go`
- [ ] `pkg/metrics/server.go`
- [ ] `pkg/config/prometheus.go`
- [ ] `pkg/dashboard/prometheus.go`
- [ ] `pkg/dashboard/alerts.go`
- [ ] `pkg/dashboard/templates/layout.templ`
- [ ] `pkg/dashboard/templates/home.templ`
- [ ] `pkg/dashboard/templates/gauges.templ`
- [ ] `pkg/dashboard/templates/history.templ`
- [ ] `pkg/dashboard/templates/alerts.templ`
- [ ] `pkg/dashboard/templates/cameras.templ`
- [ ] `pkg/dashboard/static/htmx.min.js`
- [ ] `pkg/dashboard/static/uplot.min.js`
- [ ] `pkg/dashboard/static/uplot.min.css`
- [ ] `pkg/dashboard/static/style.css`
- [ ] `pkg/testutil/prometheus.go`
- [ ] `examples/alerts/gorai-alerts.yml`
- [ ] `examples/prometheus/prometheus.yml`
- [ ] `examples/alertmanager/alertmanager.yml`

### Modified Files

- [ ] `go.mod` - add dependencies
- [ ] `pkg/node/node.go` - integrate metrics server
- [ ] `pkg/config/robot.go` - add Prometheus/Alerting configs
- [ ] `pkg/sensor/base.go` - record metrics on publish
- [ ] `pkg/componentss/base.go` - record state changes
- [ ] `Makefile` - add templ generate

---

## Success Criteria

1. **Metrics Exposed**: `/metrics` endpoint returns valid Prometheus format
2. **Dashboard Queries**: Gauges and history display data from Prometheus
3. **Alerts Work**: Low battery alert fires when battery < 20%
4. **Resource Usage**: < 500MB total RAM on Pi 4
5. **Documentation**: Complete guides for setup and configuration
6. **Tests Pass**: All unit and integration tests green

---

## Estimated Effort

| Phase | Description | Complexity |
|-------|-------------|------------|
| 1 | Metrics Exposition | Medium |
| 2 | Configuration | Low |
| 3 | Dashboard Integration | Medium |
| 4 | Templates & UI | Medium |
| 5 | Alert Integration | Low |
| 6 | Documentation | Low |
| 7 | Testing | Medium |
