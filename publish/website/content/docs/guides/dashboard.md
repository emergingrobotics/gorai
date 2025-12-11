---
title: "Web Dashboard"
description: "Built-in web dashboard for monitoring and visualization"
weight: 40
---

# Web Dashboard Guide

Gorai includes a built-in web dashboard for real-time monitoring. The dashboard is compiled into the Gorai binary using Go's `embed.FS` — no external files or dependencies required.

## Overview

The dashboard provides:

- **Gauges** — Real-time current values for all sensors
- **History** — Time-series graphs with configurable retention
- **Alerts** — Active alerts from Prometheus Alert Manager
- **Cameras** — Live video streaming (MJPEG/WebRTC)
- **Topology** — Node and component visualization

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Gorai Binary                         │
│  ┌──────────────────────────────────────────────────┐  │
│  │              Dashboard Server (:8080)            │  │
│  │                                                  │  │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────────────┐  │  │
│  │  │ Templ   │  │  HTMX   │  │     uPlot       │  │  │
│  │  │Templates│  │ Updates │  │    Charts       │  │  │
│  │  └────┬────┘  └────┬────┘  └────────┬────────┘  │  │
│  │       │            │                │           │  │
│  │       └────────────┴────────────────┘           │  │
│  │                     │                           │  │
│  │               ┌─────┴─────┐                     │  │
│  │               │Prometheus │                     │  │
│  │               │  Client   │                     │  │
│  │               └───────────┘                     │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼ PromQL queries
              ┌───────────────────────┐
              │  Prometheus (:9090)   │
              └───────────────────────┘
```

## Configuration

### Enable/Disable

The dashboard is enabled by default. To disable:

```json
{
  "dashboard": {
    "enabled": false
  }
}
```

### Full Configuration

```json
{
  "dashboard": {
    "enabled": true,
    "listen": ":8080",
    "video": {
      "enabled": true,
      "format": "mjpeg",
      "max_fps": 15,
      "quality": 80
    }
  }
}
```

### Video Formats

| Format | Latency | Browser Support | Notes |
|--------|---------|-----------------|-------|
| MJPEG | 1-2s | All browsers | Simple, universal |
| WebRTC | <0.5s | Modern browsers | Requires go2rtc or Pion |
| HLS | 5-10s | All browsers | Adaptive bitrate |

## REST API

The dashboard exposes REST endpoints that query Prometheus:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/gauges` | GET | Current sensor values |
| `/api/history?sensor=X&range=5m` | GET | Time-series data |
| `/api/alerts` | GET | Active alerts |
| `/api/cameras/{name}/stream` | GET | MJPEG video stream |

### Example: Get Gauges

```bash
curl http://localhost:8080/api/gauges
```

Response:
```json
{
  "resultType": "vector",
  "result": [
    {
      "metric": {"sensor": "temperature", "unit": "celsius"},
      "value": [1702300000, "42.5"]
    },
    {
      "metric": {"sensor": "battery", "unit": "percent"},
      "value": [1702300000, "87.3"]
    }
  ]
}
```

### Example: Get History

```bash
curl "http://localhost:8080/api/history?sensor=temperature&range=5m"
```

## Dashboard Sections

### Gauges

Real-time current values updated every 2 seconds via HTMX:

```html
<div hx-get="/api/gauges" hx-trigger="every 2s" hx-swap="innerHTML">
    Loading...
</div>
```

### History

Time-series charts rendered with uPlot. Default retention is 5 minutes:

- Select sensor from dropdown
- Choose time range (1m, 5m, 15m, 1h)
- Charts auto-refresh every 5 seconds

### Alerts

Active alerts from Prometheus Alert Manager:

- Severity levels: critical, warning, info
- Annotations with context
- Links to silence/acknowledge

### Cameras

MJPEG streaming from configured cameras:

```html
<img src="/api/cameras/front/stream" />
```

## Technology Stack

| Component | Size | Purpose |
|-----------|------|---------|
| Chi | ~20KB | HTTP router |
| Templ | 0KB runtime | Type-safe templates |
| HTMX | 14KB | Dynamic updates |
| uPlot | 50KB | Time-series charts |

**Total bundle**: ~100KB (gzipped) — all embedded in binary.

## Security

The dashboard is designed for **trusted networks**. For production:

1. **Bind to localhost**: `"listen": "127.0.0.1:8080"`
2. **Use SSH tunneling** for remote access
3. **Or disable**: `"enabled": false`

### SSH Tunnel Example

```bash
# From your workstation
ssh -L 8080:localhost:8080 robot@192.168.1.100

# Then open http://localhost:8080 in your browser
```

## Implementation

```go
package dashboard

import (
    "embed"
    "net/http"

    "github.com/go-chi/chi/v5"
    promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

//go:embed static/* templates/*
var assets embed.FS

type Server struct {
    config     Config
    promClient promv1.API
    router     *chi.Mux
}

func New(config Config, prometheusURL string) (*Server, error) {
    if !config.Enabled {
        return nil, nil
    }
    // Setup Prometheus client and routes
    // ...
}
```

## Next Steps

- [Prometheus Guide](../prometheus/) — Metrics and alerting configuration
- [Configuration Guide](../configuration/) — Full configuration reference
