---
title: "Prometheus Metrics"
description: "Configure and use Prometheus for metrics and alerting"
weight: 35
---

# Prometheus Metrics Guide

Gorai uses [Prometheus](https://prometheus.io/) as a mandatory dependency for metrics collection, time-series storage, and alerting. Prometheus runs locally on the robot alongside NATS and the Gorai binary.

## Overview

```
Robot Architecture
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

## Why Prometheus?

| Benefit | Description |
|---------|-------------|
| **Battle-tested** | Industry standard used by Kubernetes, GitLab, countless production systems |
| **Local-first** | Runs entirely on-robot with no cloud dependency |
| **Efficient storage** | Custom TSDB optimized for time-series data |
| **Powerful queries** | PromQL enables complex analysis |
| **Alerting built-in** | Alert Manager handles notifications |
| **Ecosystem** | Grafana dashboards, exporters, recording rules |

## Installation

### Debian/Ubuntu

```bash
sudo apt install prometheus prometheus-alertmanager
prometheus --version
```

### Fedora/RHEL

```bash
sudo dnf install prometheus prometheus-alertmanager
```

### From Binary

```bash
# Prometheus
wget https://github.com/prometheus/prometheus/releases/download/v2.48.0/prometheus-2.48.0.linux-amd64.tar.gz
tar xvfz prometheus-*.tar.gz
sudo mv prometheus-*/prometheus /usr/local/bin/
sudo mv prometheus-*/promtool /usr/local/bin/

# Alert Manager (optional)
wget https://github.com/prometheus/alertmanager/releases/download/v0.26.0/alertmanager-0.26.0.linux-amd64.tar.gz
tar xvfz alertmanager-*.tar.gz
sudo mv alertmanager-*/alertmanager /usr/local/bin/
```

## Configuration

### Robot Configuration (RDL)

```json
{
  "prometheus": {
    "url": "http://localhost:9090",
    "metrics_port": 9091,
    "metrics_path": "/metrics",
    "scrape_interval": "5s",
    "retention": "15d",
    "labels": {
      "environment": "production",
      "location": "warehouse-a"
    }
  }
}
```

### Prometheus Configuration

```yaml
# /etc/prometheus/prometheus.yml
global:
  scrape_interval: 5s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'gorai'
    static_configs:
      - targets: ['localhost:9091']

rule_files:
  - '/etc/gorai/alerts.yml'

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['localhost:9093']
```

## Metrics

Gorai exposes metrics at `:9091/metrics` in Prometheus format.

### Sensor Metrics

```
gorai_sensor_value{robot="sentinel",sensor="temperature",unit="celsius"} 42.5
gorai_sensor_value{robot="sentinel",sensor="battery",unit="percent"} 87.3
gorai_sensor_reading_timestamp{robot="sentinel",sensor="gps"} 1702300000
```

### Component Metrics

```
gorai_component_state{robot="sentinel",component="motor_left",state="running"} 1
gorai_component_errors_total{robot="sentinel",component="camera"} 3
gorai_component_restarts_total{robot="sentinel",component="lidar"} 1
```

### System Metrics

```
gorai_messages_total{robot="sentinel",direction="sent"} 15420
gorai_messages_total{robot="sentinel",direction="received"} 12891
gorai_inference_duration_seconds{robot="sentinel",model="yolox"} 0.045
gorai_uptime_seconds{robot="sentinel"} 86400
```

## PromQL Examples

### Current Value

```promql
gorai_sensor_value{sensor="temperature"}
```

### Rate of Change

```promql
rate(gorai_messages_total[5m])
```

### Average Over Time

```promql
avg_over_time(gorai_sensor_value{sensor="battery"}[1h])
```

### Prediction (Linear)

```promql
predict_linear(gorai_sensor_value{sensor="battery"}[1h], 3600)
```

## Alerting

### Alert Rules

Create `/etc/gorai/alerts.yml`:

```yaml
groups:
  - name: gorai
    rules:
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

      - alert: HighInferenceLatency
        expr: gorai_inference_duration_seconds > 0.5
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Slow inference on {{ $labels.robot }}"
```

### Alert Manager Configuration

```yaml
# /etc/alertmanager/alertmanager.yml
route:
  receiver: 'default'
  group_wait: 30s

receivers:
  - name: 'default'
    webhook_configs:
      - url: 'http://localhost:8080/api/alerts/webhook'
```

## Resource Usage

| Component | RAM | Disk | Notes |
|-----------|-----|------|-------|
| Prometheus | 200-400MB | ~1GB/day | Depends on metric count |
| Alert Manager | 30-50MB | Minimal | Optional |
| Gorai | ~50MB | Minimal | Varies with modules |

**Total**: ~300-500MB RAM — compatible with Raspberry Pi 4, Jetson Nano, and similar platforms.

## Next Steps

- [Dashboard Guide](../dashboard/) — Web dashboard backed by Prometheus
- [Configuration Guide](../configuration/) — Full configuration reference
