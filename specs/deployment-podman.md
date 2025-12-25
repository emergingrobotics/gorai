# Deployment Specification: Podman-Everywhere Architecture

**Version:** 1.0
**Status:** Active
**Last Updated:** 2024-12-25

## 1. Overview

Gorai uses a **Podman-everywhere architecture** where all robots run as Podman pods managed by systemd. This provides container isolation and reproducibility without the overhead and compatibility issues of Kubernetes.

### 1.1 Design Philosophy

**"Containers without complexity."**

Podman pods provide the benefits of containerization (isolation, reproducibility, versioned artifacts) while leveraging systemd for lifecycle management. This approach:

1. **Works everywhere** — No kernel-specific requirements (unlike K3s which needs eBPF)
2. **Low overhead** — ~150 MB vs ~1.8 GB for K3s
3. **Simple debugging** — `podman logs`, `journalctl`, familiar Linux tools
4. **SD card viable** — No control plane database hammering storage
5. **Fleet-ready** — NATS-based coordination scales to multi-robot deployments

### 1.2 What Users See vs What Runs

```
┌─────────────────────────────────────────────────────────────────┐
│                    User Experience                               │
│                                                                  │
│   robot.yaml (RDL)  →  gorai deploy  →  Robot running           │
│                                                                  │
│   Users work with:                                              │
│   • Robot Definition Language (YAML/JSON)                       │
│   • gorai CLI commands                                          │
│   • Web dashboard                                               │
│                                                                  │
│   Users never need to know:                                     │
│   • Podman pod syntax                                           │
│   • systemd unit files                                          │
│   • Container networking details                                │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ gorai translates
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    What Actually Runs                            │
│                                                                  │
│   systemd                                                        │
│   └── gorai-{robot-name}.service                                │
│       └── Podman Pod: {robot-name}                              │
│           ├── nats (container)                                  │
│           ├── gorai-core (container)                            │
│           ├── detector (container, optional)                    │
│           └── ... additional service containers                 │
└─────────────────────────────────────────────────────────────────┘
```

### 1.3 Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│  Host System (Pi 5, Jetson, Orange Pi, x86)                     │
│                                                                  │
│  systemd                                                         │
│  └── gorai-rover1.service                                       │
│      └── Podman Pod: rover1                                     │
│          ┌──────────────────────────────────────────────┐       │
│          │  Shared network namespace (localhost)         │       │
│          │                                               │       │
│          │  ┌────────┐  ┌────────────┐  ┌──────────┐   │       │
│          │  │  nats  │  │ gorai-core │  │ detector │   │       │
│          │  │ :4222  │  │   :8080    │  │ (Python) │   │       │
│          │  └────────┘  └────────────┘  └──────────┘   │       │
│          │       ▲            │              │          │       │
│          │       └────────────┴──────────────┘          │       │
│          │              NATS messaging                  │       │
│          └──────────────────────────────────────────────┘       │
│                                                                  │
│  Hardware: /dev/video0, /dev/i2c-1, /dev/hailo0, GPIO           │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Hardware Requirements

### 2.1 Minimum Requirements

| Component | Requirement | Notes |
|-----------|-------------|-------|
| **Processor** | ARM64 (Cortex-A72+) or x86_64 | 4+ cores recommended |
| **RAM** | 2 GB minimum, 4 GB recommended | Podman overhead ~150 MB |
| **Storage** | 32 GB minimum | SD card acceptable, SSD recommended |
| **Network** | Ethernet or WiFi | Ethernet recommended for reliability |

### 2.2 Recommended Configurations

**Primary Platform: Raspberry Pi 5 (8GB)**

| Component | Specification | Est. Cost |
|-----------|---------------|-----------|
| Raspberry Pi 5 (8GB) | 4-core Cortex-A76, 8GB LPDDR4X | $80 |
| NVMe SSD (256GB) | PCIe Gen 2 x1, via HAT | $30 |
| NVMe HAT/Base | Pimoroni, Geekworm, etc. | $20 |
| Power Supply | 5V/5A USB-C PD | $15 |
| Heatsink/Case | Active cooling recommended | $15 |
| **Total** | | **~$160** |

**Performance Platform: Jetson Orin Nano Super**

| Component | Specification | Est. Cost |
|-----------|---------------|-----------|
| Jetson Orin Nano Super | 6-core A78AE, 67 TOPS, 8GB | $249 |
| NVMe SSD (256GB) | M.2 2280 | $30 |
| WiFi Module | Intel AC8265 (M.2 Key E) | $20 |
| Power Supply | 5V/5A USB-C | $15 |
| Active Cooling | Fan + heatsink | $20 |
| **Total** | | **~$335** |

**Budget AI Platform: Orange Pi 5B (8GB)**

| Component | Specification | Est. Cost |
|-----------|---------------|-----------|
| Orange Pi 5B (8GB/64GB) | 8-core RK3588S, 6 TOPS NPU, 64GB eMMC | $115 |
| Power Supply | 5V/4A USB-C | $15 |
| Heatsink/Fan | Required | $15 |
| **Total** | | **~$145** |

### 2.3 Supported Platforms

| Platform | RAM | AI Acceleration | Podman Support |
|----------|-----|-----------------|----------------|
| **Raspberry Pi 5** | 4-8 GB | External (Hailo, Coral) | ✅ Full |
| **Raspberry Pi 4** | 4-8 GB | External (Coral) | ✅ Full |
| **Jetson Orin Nano Super** | 8 GB | 67 TOPS (CUDA) | ✅ Full |
| **Orange Pi 5B** | 8-16 GB | 6 TOPS (RK3588 NPU) | ✅ Full |
| **Radxa Rock 5B** | 4-16 GB | 6 TOPS (RK3588 NPU) | ✅ Full |
| **x86_64 (any)** | 4+ GB | NVIDIA GPU (optional) | ✅ Full |

**Not Supported:**
- Raspberry Pi 3 and earlier (insufficient RAM)
- Raspberry Pi Zero/Zero 2 (insufficient RAM for containers)
- Systems with < 2GB RAM

### 2.4 Storage Considerations

Unlike K3s, Podman does not require high-IOPS storage for a control plane database.

| Storage Type | Podman Support | Notes |
|--------------|----------------|-------|
| SD Card (Class 10) | ✅ Acceptable | Slower image pulls, adequate for runtime |
| SD Card (A2) | ✅ Good | Better random I/O |
| eMMC | ✅ Good | Orange Pi 5B built-in |
| USB 3.0 SSD | ✅ Excellent | Recommended for performance |
| NVMe SSD | ✅ Excellent | Best performance |

**Recommendation:** SSD for development/frequent updates, SD card acceptable for deployed robots with stable images.

---

## 3. Podman Pod Architecture

### 3.1 What is a Pod?

A Podman pod is a group of containers that:
- Share a network namespace (communicate via `localhost`)
- Share IPC namespace (can use shared memory)
- Are managed as a single unit (start/stop together)
- Mirror Kubernetes pod semantics (same YAML format)

### 3.2 Standard Pod Structure

Every Gorai robot runs as a pod with these containers:

```
Pod: {robot-name}
├── nats          # Message broker (always present)
├── gorai-core    # Go orchestration + components
└── {services}    # External services (Python, C++, etc.)
```

**Container Communication:**

```
┌─────────────────────────────────────────────────────────────────┐
│  Pod Network Namespace                                           │
│                                                                  │
│  gorai-core ──► nats://localhost:4222 ──► NATS                  │
│  detector   ──► nats://localhost:4222 ──► NATS                  │
│  slam       ──► nats://localhost:4222 ──► NATS                  │
│                                                                  │
│  All containers see each other as localhost                     │
└─────────────────────────────────────────────────────────────────┘

External access via published ports:
  Host:4222  → Pod:4222 (NATS)
  Host:8080  → Pod:8080 (Dashboard)
  Host:9091  → Pod:9091 (Metrics)
```

### 3.3 Resource Limits

Podman uses cgroups v2 for resource isolation:

```yaml
# In RDL
services:
  - name: detector
    container:
      resources:
        memory: 2G      # Hard limit
        cpus: 2         # CPU cores
```

**Translated to Podman:**
```bash
podman run --memory=2g --cpus=2 ...
```

**systemd-level limits** (entire pod):
```ini
[Service]
MemoryMax=6G
CPUQuota=300%
```

---

## 4. RDL to Podman Translation

### 4.1 Translation Overview

| RDL Concept | Podman Artifact |
|-------------|-----------------|
| `robot.name` | Pod name, systemd unit name |
| Component | Part of gorai-core container |
| Service (internal) | Part of gorai-core container |
| Service (external) | Separate container in pod |
| Device access | `--device` flag |
| Resource limits | `--memory`, `--cpus` flags |
| Environment vars | `-e` flags or env file |

### 4.2 Example Translation

**Input: rover1.yaml**
```yaml
version: "3"

robot:
  name: rover1
  description: Autonomous rover with vision

platform:
  board: pi5-8gb
  accelerator: hailo-8l

nats:
  url: nats://localhost:4222

components:
  - name: camera
    type: camera
    model: v4l2
    attributes:
      device: /dev/video0
      width: 640
      height: 480

services:
  - name: detector
    type: vision
    model: yolox
    container:
      image: ghcr.io/gorai/yolox-detector:latest
      devices:
        - /dev/hailo0
      resources:
        memory: 2G
        cpus: 2

dashboard:
  enabled: true
  port: 8080
```

**Output: ~/.config/gorai/rover1/pod.yaml**
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: rover1
  labels:
    app: gorai
    robot: rover1
spec:
  restartPolicy: Always

  containers:
    # NATS message broker
    - name: nats
      image: docker.io/nats:2.10-alpine
      args: ["-js", "-sd", "/data", "-m", "8222"]
      ports:
        - containerPort: 4222
          hostPort: 4222
        - containerPort: 8222
          hostPort: 8222
      volumeMounts:
        - name: nats-data
          mountPath: /data
      resources:
        limits:
          memory: 512Mi
          cpu: "0.5"

    # Gorai core (Go orchestration + components)
    - name: gorai-core
      image: ghcr.io/gorai/gorai:latest
      args: ["run", "--config", "/etc/gorai/robot.yaml"]
      env:
        - name: NATS_URL
          value: "nats://localhost:4222"
        - name: GORAI_ROBOT_NAME
          value: "rover1"
      ports:
        - containerPort: 8080
          hostPort: 8080
        - containerPort: 9091
          hostPort: 9091
      volumeMounts:
        - name: config
          mountPath: /etc/gorai
        - name: video0
          mountPath: /dev/video0
      resources:
        limits:
          memory: 1Gi
          cpu: "2"
      securityContext:
        capabilities:
          add: ["SYS_RAWIO"]

    # Vision detector (Python + ONNX)
    - name: detector
      image: ghcr.io/gorai/yolox-detector:latest
      env:
        - name: NATS_URL
          value: "nats://localhost:4222"
        - name: GORAI_SERVICE_NAME
          value: "detector"
        - name: CONFIDENCE_THRESHOLD
          value: "0.5"
      volumeMounts:
        - name: hailo0
          mountPath: /dev/hailo0
      resources:
        limits:
          memory: 2Gi
          cpu: "2"
      securityContext:
        privileged: true

  volumes:
    - name: nats-data
      hostPath:
        path: /var/lib/gorai/rover1/nats
        type: DirectoryOrCreate
    - name: config
      hostPath:
        path: /etc/gorai/rover1
        type: Directory
    - name: video0
      hostPath:
        path: /dev/video0
        type: CharDevice
    - name: hailo0
      hostPath:
        path: /dev/hailo0
        type: CharDevice
```

**Output: /etc/systemd/system/gorai-rover1.service**
```ini
[Unit]
Description=Gorai Robot: rover1
Documentation=https://gorai.dev/docs
After=network-online.target
Wants=network-online.target
RequiresMountsFor=/var/lib/gorai

[Service]
Type=forking
Restart=on-failure
RestartSec=10
TimeoutStartSec=300
TimeoutStopSec=60

# Environment
Environment=PODMAN_SYSTEMD_UNIT=%n
WorkingDirectory=/etc/gorai/rover1

# Pre-start: ensure pod exists
ExecStartPre=-/usr/bin/podman pod exists rover1
ExecStartPre=-/usr/bin/podman pod rm -f rover1
ExecStartPre=/usr/bin/podman play kube --replace pod.yaml

# Start the pod
ExecStart=/usr/bin/podman pod start rover1

# Stop gracefully
ExecStop=/usr/bin/podman pod stop -t 30 rover1

# Cleanup
ExecStopPost=-/usr/bin/podman pod rm -f rover1

# Resource limits for entire pod
MemoryMax=6G
CPUQuota=400%

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=gorai-rover1

[Install]
WantedBy=multi-user.target
```

### 4.3 Device Passthrough

| Device Type | RDL Syntax | Podman Flag |
|-------------|------------|-------------|
| Camera | `/dev/video0` | `--device=/dev/video0` |
| I2C | `/dev/i2c-1` | `--device=/dev/i2c-1` + capabilities |
| SPI | `/dev/spidev0.0` | `--device=/dev/spidev0.0` + privileged |
| GPIO | `/dev/gpiomem` | `--device=/dev/gpiomem` |
| Hailo NPU | `/dev/hailo0` | `--device=/dev/hailo0` + privileged |
| Coral TPU | `/dev/apex_0` | `--device=/dev/apex_0` |
| NVIDIA GPU | `/dev/nvidia*` | `--device=nvidia.com/gpu=all` |
| RK3588 NPU | `/dev/dri/*` | `--device=/dev/dri` |

**Jetson GPU Access:**
```yaml
services:
  - name: detector
    container:
      image: ghcr.io/gorai/detector-cuda:latest
      devices:
        - nvidia.com/gpu=all
      runtime: nvidia
```

Translated to:
```bash
podman run --runtime=nvidia --device=nvidia.com/gpu=all ...
```

---

## 5. CLI Commands

### 5.1 Deployment Commands

```bash
# Validate configuration
gorai validate rover1.yaml

# Deploy robot (generates pod.yaml, systemd unit, starts service)
gorai deploy rover1.yaml

# Deploy without starting
gorai deploy rover1.yaml --no-start

# View generated artifacts without deploying
gorai deploy rover1.yaml --dry-run

# Undeploy (stop service, remove pod, cleanup)
gorai undeploy rover1
```

### 5.2 Runtime Commands

```bash
# Check robot status
gorai status rover1

# Example output:
#   Robot: rover1
#   Status: running
#   Uptime: 2h 34m 12s
#
#   Containers:
#     NAME        STATUS    MEMORY      CPU
#     nats        running   45 MB       0.2%
#     gorai-core  running   312 MB      2.1%
#     detector    running   1.8 GB      45.3%
#
#   Pod Totals: 2.2 GB / 6.0 GB memory, 47.6% CPU

# View logs (all containers, interleaved)
gorai logs rover1

# View logs (specific container)
gorai logs rover1 detector

# Follow logs
gorai logs rover1 -f

# Follow logs for specific container
gorai logs rover1 detector -f

# Tail last N lines
gorai logs rover1 --tail 100

# Restart entire robot
gorai restart rover1

# Restart specific container
gorai restart rover1 detector

# Stop robot
gorai stop rover1

# Start stopped robot
gorai start rover1

# Shell into container
gorai exec rover1 detector -- /bin/bash

# Run command in container
gorai exec rover1 gorai-core -- gorai component list

# Open dashboard in browser
gorai dashboard rover1
```

### 5.3 Image Management

```bash
# Pull latest images for robot
gorai pull rover1

# Update robot (pull + restart)
gorai update rover1

# List images used by robot
gorai images rover1

# Prune unused images
gorai prune
```

### 5.4 Debugging Commands

```bash
# Detailed status with events
gorai status rover1 --verbose

# Show pod definition
gorai inspect rover1

# Show systemd unit status
gorai service rover1

# Show resource usage over time
gorai top rover1

# Export diagnostics bundle
gorai diagnostics rover1 --output rover1-diag.tar.gz
```

---

## 6. Installation & Setup

### 6.1 Prerequisites

**Install Podman:**

```bash
# Raspberry Pi OS / Debian / Ubuntu
sudo apt update
sudo apt install -y podman

# Fedora
sudo dnf install -y podman

# Arch Linux
sudo pacman -S podman

# Verify installation
podman --version
```

**Enable cgroups v2 (if needed):**

```bash
# Check current cgroup version
cat /sys/fs/cgroup/cgroup.controllers

# If empty, enable cgroups v2 (reboot required)
sudo sed -i 's/^GRUB_CMDLINE_LINUX="/GRUB_CMDLINE_LINUX="systemd.unified_cgroup_hierarchy=1 /' /etc/default/grub
sudo update-grub
sudo reboot
```

### 6.2 Install Gorai

```bash
# Install gorai CLI
curl -sfL https://get.gorai.dev | sh

# Verify installation
gorai version

# Initialize gorai (creates directories, pulls base images)
gorai init
```

### 6.3 Deploy First Robot

```bash
# Create robot definition
cat > my-robot.yaml << 'EOF'
version: "3"
robot:
  name: my-robot
  description: My first Gorai robot

components:
  - name: heartbeat
    type: ticker
    attributes:
      interval_ms: 1000

dashboard:
  enabled: true
EOF

# Deploy
gorai deploy my-robot.yaml

# Check status
gorai status my-robot

# View dashboard
gorai dashboard my-robot
```

### 6.4 Jetson-Specific Setup

```bash
# Install NVIDIA Container Toolkit
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -s -L https://nvidia.github.io/libnvidia-container/gpgkey | sudo apt-key add -
curl -s -L https://nvidia.github.io/libnvidia-container/$distribution/libnvidia-container.list | \
  sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list

sudo apt update
sudo apt install -y nvidia-container-toolkit

# Configure Podman to use NVIDIA runtime
sudo nvidia-ctk runtime configure --runtime=podman

# Verify GPU access
podman run --rm --runtime=nvidia --device=nvidia.com/gpu=all \
  nvcr.io/nvidia/l4t-base:r36.2.0 nvidia-smi
```

---

## 7. Fleet Orchestration

For multi-robot deployments, Gorai uses NATS as the coordination layer instead of Kubernetes.

### 7.1 Fleet Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Control Node                                 │
│  (Edge server, cloud VM, or designated robot)                   │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │  NATS Hub    │  │  Fleet Mgr   │  │  Prometheus  │          │
│  │  :4222       │  │  (Go binary) │  │  :9090       │          │
│  │  :7422 leaf  │  │              │  │              │          │
│  └──────┬───────┘  └──────────────┘  └──────────────┘          │
└─────────┼───────────────────────────────────────────────────────┘
          │
          │ NATS Leaf Node Connections
          │
    ┌─────┴─────┬───────────────┬───────────────┐
    ▼           ▼               ▼               ▼
┌───────┐  ┌───────┐       ┌───────┐       ┌───────┐
│Robot 1│  │Robot 2│       │Robot 3│       │Robot N│
│ rover │  │ drone │       │ arm   │       │ ...   │
│       │  │       │       │       │       │       │
│ NATS  │  │ NATS  │       │ NATS  │       │ NATS  │
│ Leaf  │  │ Leaf  │       │ Leaf  │       │ Leaf  │
└───────┘  └───────┘       └───────┘       └───────┘
```

### 7.2 Fleet Agent

Each robot runs a lightweight fleet agent that:
1. Connects to the NATS hub as a leaf node
2. Subscribes to fleet commands on `gorai.fleet.{robot-name}.>`
3. Executes commands locally (restart, update, logs, etc.)
4. Publishes status to `gorai.fleet.{robot-name}.status`

**Enable fleet mode in RDL:**
```yaml
version: "3"
robot:
  name: rover1

fleet:
  enabled: true
  hub: nats://control-node.local:7422
  token: ${FLEET_TOKEN}
```

### 7.3 Fleet Commands

```bash
# Initialize fleet hub (on control node)
gorai fleet init

# Get join token
gorai fleet token

# Join robot to fleet (on robot)
gorai fleet join nats://control-node.local:7422 --token <token>

# List all robots in fleet
gorai fleet list

# Example output:
#   NAME      STATUS    UPTIME     VERSION    LAST SEEN
#   rover1    running   2h 34m     0.5.0      2s ago
#   rover2    running   1h 12m     0.5.0      1s ago
#   drone1    stopped   -          0.4.9      5m ago

# Deploy to specific robots
gorai fleet deploy rover1.yaml --robots rover1,rover2

# Deploy to all robots matching pattern
gorai fleet deploy *.yaml --robots 'rover*'

# Update all robots
gorai fleet update --all

# Restart service on multiple robots
gorai fleet restart detector --robots rover1,rover2

# Get logs from remote robot
gorai fleet logs rover1 detector --tail 100

# Execute command on remote robot
gorai fleet exec rover1 -- gorai status

# Get fleet-wide status
gorai fleet status
```

### 7.4 Fleet Configuration

**Control node: fleet-hub.yaml**
```yaml
version: "3"
robot:
  name: fleet-hub

services:
  - name: nats-hub
    container:
      image: docker.io/nats:2.10-alpine
      args:
        - "-js"
        - "-sd"
        - "/data"
        - "-p"
        - "4222"
        - "--cluster_name"
        - "gorai-fleet"
        - "--leafnodes"
        - "nats://0.0.0.0:7422"
      ports:
        - 4222:4222
        - 7422:7422
        - 8222:8222

  - name: fleet-manager
    container:
      image: ghcr.io/gorai/fleet-manager:latest
      env:
        - NATS_URL=nats://localhost:4222

  - name: prometheus
    container:
      image: docker.io/prom/prometheus:latest
      ports:
        - 9090:9090
      volumeMounts:
        - /etc/gorai/prometheus:/etc/prometheus
```

---

## 8. Observability

### 8.1 Prometheus Metrics

All Gorai containers expose Prometheus metrics:

| Container | Endpoint | Metrics |
|-----------|----------|---------|
| gorai-core | `:9091/metrics` | Component status, message rates, latency |
| nats | `:8222/varz` | Connection count, message throughput |
| detector | `:9091/metrics` | Inference time, FPS, model stats |

**Prometheus scrape config:**
```yaml
scrape_configs:
  - job_name: 'gorai-rover1'
    static_configs:
      - targets: ['rover1.local:9091']
    relabel_configs:
      - source_labels: [__address__]
        target_label: robot
        replacement: rover1
```

### 8.2 Logging

All container logs go to journald via systemd:

```bash
# View logs via gorai CLI
gorai logs rover1

# View logs via journalctl
journalctl -u gorai-rover1 -f

# View logs for specific container
journalctl -u gorai-rover1 CONTAINER_NAME=detector

# Export logs
journalctl -u gorai-rover1 --since "1 hour ago" > rover1.log
```

### 8.3 Dashboard

The embedded dashboard provides:
- Component status and health
- Sensor readings (current and historical)
- Camera feeds
- Inference results
- Alert status
- Resource usage graphs

Access at: `http://{robot-ip}:8080`

---

## 9. Networking

### 9.1 Pod Internal Networking

All containers in a pod share the same network namespace:

```
Container A → localhost:4222 → NATS (Container B)
Container C → localhost:8080 → Dashboard (Container A)
```

No special DNS or service discovery needed within a pod.

### 9.2 Host Port Publishing

Ports are published to the host for external access:

```yaml
# In pod.yaml
ports:
  - containerPort: 4222
    hostPort: 4222    # NATS client
  - containerPort: 8080
    hostPort: 8080    # Dashboard
  - containerPort: 9091
    hostPort: 9091    # Metrics
```

### 9.3 Multi-Robot Communication

Robots communicate via NATS leaf nodes:

```yaml
# Robot RDL
nats:
  url: nats://localhost:4222
  leaf:
    remotes:
      - url: nats://control-node.local:7422
```

**Message routing:**
```
rover1 publishes: gorai.rover1.sensor.camera
  → Local NATS → Leaf connection → Hub NATS
  → Subscribed robots receive message
```

---

## 10. Security

### 10.1 Container Security

- Containers run as non-root where possible
- Privileged mode only for hardware access (minimized)
- Capabilities added individually (`SYS_RAWIO`, `NET_ADMIN`) instead of full privileged
- Read-only root filesystem where applicable

### 10.2 Device Access

```yaml
# Prefer capabilities over privileged
securityContext:
  privileged: false
  capabilities:
    add:
      - SYS_RAWIO    # For I2C/SPI
      - NET_RAW      # For raw sockets
```

### 10.3 Secrets Management

```yaml
# In RDL - reference environment variables
nats:
  credentials: ${NATS_CREDENTIALS}
  tls:
    ca_file: /etc/gorai/tls/ca.crt
    cert_file: /etc/gorai/tls/client.crt
    key_file: /etc/gorai/tls/client.key
```

Secrets are stored in:
- Environment files (`/etc/gorai/{robot}/.env`)
- systemd credentials
- External secret managers (future)

### 10.4 Network Security

- NATS connections use TLS in production
- Fleet communication authenticated via tokens/credentials
- No ports published by default except explicitly configured

---

## 11. Troubleshooting

### 11.1 Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Pod won't start | Image pull failed | Check network, `gorai pull rover1` |
| Container crash loop | Application error | `gorai logs rover1 {container}` |
| Device not accessible | Permissions | Add device to RDL, check udev rules |
| Out of memory | Resource limits | Increase limits in RDL |
| GPU not available | Runtime not configured | Install nvidia-container-toolkit |
| Slow startup | Large images | Use smaller base images, pre-pull |

### 11.2 Debug Commands

```bash
# Check systemd service status
systemctl status gorai-rover1

# View systemd journal
journalctl -u gorai-rover1 -e

# List running pods
podman pod ps

# List containers in pod
podman pod inspect rover1

# Check container logs directly
podman logs rover1-detector

# Inspect container
podman inspect rover1-detector

# Check resource usage
podman stats

# Enter container for debugging
podman exec -it rover1-detector /bin/bash

# Check cgroup limits
cat /sys/fs/cgroup/system.slice/gorai-rover1.service/memory.max
```

### 11.3 Recovery

```bash
# Restart pod
gorai restart rover1

# Force recreate pod
gorai undeploy rover1
gorai deploy rover1.yaml

# Reset to clean state
gorai undeploy rover1
podman pod rm -f rover1
podman volume prune
gorai deploy rover1.yaml

# Check for orphaned containers
podman ps -a --filter "label=robot=rover1"
```

---

## 12. Migration

### 12.1 From Native Binaries

If running Gorai as native systemd services:

1. Create RDL from existing configuration
2. Build container images (or use prebuilt)
3. Deploy with Podman: `gorai deploy rover1.yaml`
4. Disable old services: `sudo systemctl disable gorai-*`

### 12.2 From K3s

If migrating from K3s deployment:

1. Export robot configuration: `gorai export rover1 > rover1.yaml`
2. Stop K3s deployment: `kubectl delete namespace gorai-rover1`
3. Deploy with Podman: `gorai deploy rover1.yaml`

The RDL format is compatible; only deployment mechanism changes.

### 12.3 To K3s (Future)

If scaling to fleet requires Kubernetes:

1. Same RDL works
2. Deploy with K3s flag: `gorai deploy rover1.yaml --runtime=k3s`
3. Generates Kubernetes manifests instead of Podman pod.yaml

---

## Appendix A: Podman vs Docker

Gorai uses Podman because:

| Feature | Podman | Docker |
|---------|--------|--------|
| Daemonless | ✅ No daemon | ❌ Requires dockerd |
| Rootless | ✅ Native support | ⚠️ Experimental |
| systemd integration | ✅ Excellent | ⚠️ Requires wrapper |
| Pod support | ✅ Native (like K8s) | ❌ Compose only |
| Drop-in replacement | ✅ `alias docker=podman` | - |
| Image compatibility | ✅ OCI standard | ✅ OCI standard |

Docker is supported via compatibility mode:
```bash
gorai deploy rover1.yaml --runtime=docker
```

---

## Appendix B: Resource Overhead Comparison

| Component | K3s | Podman + systemd |
|-----------|-----|------------------|
| Control plane | ~1.5 GB | 0 (systemd is control plane) |
| Container runtime | ~100 MB (containerd) | ~100 MB (Podman) |
| Networking | ~200 MB (Flannel/Calico) | ~10 MB (native) |
| **Total overhead** | **~1.8 GB** | **~110 MB** |

**Available RAM for workloads (8GB system):**
- K3s: ~5.5 GB
- Podman: ~7.5 GB

---

## Appendix C: Platform-Specific Notes

### C.1 Raspberry Pi

```bash
# Enable cgroups memory controller
echo ' cgroup_memory=1 cgroup_enable=memory' | sudo tee -a /boot/cmdline.txt
sudo reboot

# Install Podman
sudo apt install -y podman
```

### C.2 Jetson Orin

```bash
# Install NVIDIA Container Toolkit
sudo apt install -y nvidia-container-toolkit

# Configure for Podman
sudo nvidia-ctk runtime configure --runtime=podman

# Test GPU access
podman run --rm --runtime=nvidia --device=nvidia.com/gpu=all \
  nvcr.io/nvidia/l4t-base:r36.2.0 nvidia-smi
```

### C.3 Orange Pi 5B (RK3588)

```bash
# Install Podman on Armbian
sudo apt install -y podman

# NPU access requires privileged container
# or specific device mappings
devices:
  - /dev/dri/renderD128
  - /dev/dri/card0
```

### C.4 x86_64 with NVIDIA GPU

```bash
# Install NVIDIA Container Toolkit
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
curl -s -L https://nvidia.github.io/libnvidia-container/$distribution/libnvidia-container.list | \
  sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
  sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list

sudo apt update
sudo apt install -y nvidia-container-toolkit
sudo nvidia-ctk runtime configure --runtime=podman
```
