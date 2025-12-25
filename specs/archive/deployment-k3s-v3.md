# Deployment Specification: K3s-Everywhere Architecture

**Version:** 3.0
**Status:** Active
**Last Updated:** 2024-12-25

## 1. Overview

Gorai uses a **K3s-everywhere architecture** where all robots, from simple sensor platforms to complex autonomous systems, deploy on Kubernetes (K3s). This provides a consistent deployment model that scales from single robots to multi-robot fleets without architectural changes.

### 1.1 Design Philosophy

**"AI at the edge requires capable hardware. Capable hardware can run K3s."**

Rather than supporting multiple deployment tiers (native binaries, Podman, K3s), Gorai standardizes on K3s for all deployments. This decision is driven by:

1. **Edge AI requirements** - ML inference, computer vision, and SLAM require 4GB+ RAM regardless of orchestration
2. **Consistency** - One deployment model to learn, debug, and maintain
3. **Fleet-ready** - Every robot can join a fleet with zero architecture changes
4. **Container benefits** - Reproducible builds, versioned artifacts, isolated dependencies
5. **Ecosystem access** - Helm charts, GitOps, observability tools work out of the box

### 1.2 What Users See vs What Runs

```
┌─────────────────────────────────────────────────────────────────┐
│                    User Experience                               │
│                                                                  │
│   robot.json (RDL)  →  gorai deploy  →  Robot running           │
│                                                                  │
│   Users work with:                                              │
│   • Robot Definition Language (JSON)                            │
│   • gorai CLI commands                                          │
│   • Web dashboard                                               │
│                                                                  │
│   Users never need to know:                                     │
│   • Kubernetes, kubectl, manifests                              │
│   • Container runtimes, containerd                              │
│   • Pods, Deployments, Services, ConfigMaps                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ gorai translates
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    What Actually Runs                            │
│                                                                  │
│   K3s Cluster (single-node or multi-node)                       │
│   ├── Namespace: gorai-{robot-name}                             │
│   ├── Pod: nats (message broker)                                │
│   ├── Pod: gorai-core (Go orchestration + components)           │
│   ├── Pod: {service-name} (vision, SLAM, etc.)                  │
│   ├── ConfigMap: robot-config                                   │
│   ├── Service: nats, dashboard                                  │
│   └── PersistentVolumeClaim: data, models                       │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Hardware Requirements

### 2.1 Minimum Requirements

| Component | Requirement | Notes |
|-----------|-------------|-------|
| **Compute** | Raspberry Pi 4 (4GB) or equivalent | ARM64 or x86_64 |
| **Storage** | External SSD (128GB+) via USB 3.0 | SD cards not supported for production |
| **Memory** | 4GB minimum, 8GB recommended | K3s uses ~1.5GB, rest for workloads |
| **Network** | Ethernet recommended | WiFi acceptable for development |
| **Power** | Quality 5V/3A supply (Pi 4) or 5V/5A (Pi 5) | Undervoltage causes instability |

### 2.2 Recommended Configurations

**Primary Platform: Raspberry Pi 5**
| Component | Specification | Estimated Cost |
|-----------|---------------|----------------|
| Raspberry Pi 5 (8GB) | Primary compute | $80 |
| NVMe SSD (256GB) + HAT | Fast storage | $50 |
| Quality power supply | 5V/5A USB-C | $15 |
| Ethernet connection | Gigabit | - |
| **Total** | | **~$145** |

**Alternative Platform: Orange Pi 5B (Budget AI)**
| Component | Specification | Estimated Cost |
|-----------|---------------|----------------|
| Orange Pi 5B (8GB/64GB) | 8-core RK3588S, 6 TOPS NPU, 64GB eMMC | $115 |
| Quality power supply | 5V/4A USB-C | $15 |
| Active cooling | Heatsink + fan (required) | $15 |
| **Total** | | **~$145** |

Built-in NPU and eMMC eliminate need for external accelerator and SSD.

### 2.3 Supported Platforms

**Primary (Reference):**
- Raspberry Pi 5 (8GB) — Best ecosystem, largest community
- Raspberry Pi 5 (4GB) — Tight but workable

**Alternative (Built-in NPU):**
- Orange Pi 5B (8GB) — Built-in 6 TOPS NPU, eMMC, budget AI
- Orange Pi 5B (16GB) — Extra RAM for complex workloads

**Minimum Baseline:**
- Raspberry Pi 4 Model B (4GB, 8GB) — USB SSD required

**Other Supported:**
- NVIDIA Jetson Orin Nano — CUDA acceleration
- Radxa Rock 5B — RK3588 NPU, NVMe support
- Orange Pi 5 Plus — RK3588, dual 2.5GbE
- Any x86_64 system with 4GB+ RAM

**Not Supported:**
- Raspberry Pi 3 and earlier (insufficient RAM)
- Raspberry Pi Zero/Zero 2 (insufficient RAM)
- Raspberry Pi 4 (2GB) (insufficient RAM for K3s + workloads)
- Orange Pi 5B (4GB) (tight RAM for K3s + AI)
- SD card-only deployments (storage I/O insufficient)

### 2.4 Storage Requirements

**Why SSD/eMMC is mandatory:**

| Storage Type | Random IOPS | K3s Stability |
|--------------|-------------|---------------|
| SD Card (Class 10) | 10-30 | ❌ Database corruption within months |
| SD Card (A2) | 30-50 | ❌ Marginal, not recommended |
| eMMC (built-in) | 100-500 | ✅ Acceptable (Orange Pi 5B) |
| USB 3.0 SSD | 500-5000 | ✅ Stable production use |
| NVMe SSD | 5000-50000 | ✅ Excellent |

K3s uses SQLite (or etcd for HA) which requires sustained random write performance. SD cards cannot provide this reliably; eMMC and SSDs can.

---

## 3. Architecture

### 3.1 Single Robot Deployment

```
┌─────────────────────────────────────────────────────────────────┐
│                    Raspberry Pi 5 (8GB)                          │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    K3s (containerd)                      │   │
│  │                                                          │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐              │   │
│  │  │   NATS   │  │  gorai   │  │ detector │              │   │
│  │  │          │  │   core   │  │ (vision) │              │   │
│  │  │ :4222    │  │  :8080   │  │          │              │   │
│  │  └──────────┘  └──────────┘  └──────────┘              │   │
│  │       │              │              │                    │   │
│  │       └──────────────┴──────────────┘                   │   │
│  │                      │                                   │   │
│  │              NATS messaging                              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│  Hardware: /dev/video0, /dev/i2c-1, /dev/hailo0, GPIO           │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 Multi-Robot Fleet

```
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│     Robot 1      │  │     Robot 2      │  │     Robot 3      │
│  K3s (worker)    │  │  K3s (worker)    │  │  K3s (worker)    │
│                  │  │                  │  │                  │
│  ┌────────────┐  │  │  ┌────────────┐  │  │  ┌────────────┐  │
│  │ NATS Leaf  │  │  │  │ NATS Leaf  │  │  │  │ NATS Leaf  │  │
│  └─────┬──────┘  │  │  └─────┬──────┘  │  │  └─────┬──────┘  │
└────────┼─────────┘  └────────┼─────────┘  └────────┼─────────┘
         │                     │                     │
         └─────────────────────┼─────────────────────┘
                               │
                    ┌──────────┴──────────┐
                    │   Control Plane      │
                    │   (Edge Server or    │
                    │    Cloud VM)         │
                    │                      │
                    │  K3s Server          │
                    │  NATS Hub            │
                    │  Prometheus          │
                    │  ArgoCD (GitOps)     │
                    └─────────────────────┘
```

### 3.3 Resource Budget

**Raspberry Pi 4 (4GB):**
```
Total RAM:     4,096 MB
├── OS:          300 MB
├── K3s:       1,500 MB
├── NATS:        256 MB (configured limit)
└── Available: 2,040 MB for robot workloads

Total CPU:     4 cores
├── K3s:       ~30% (~1.2 cores)
└── Available: ~70% (~2.8 cores) for robot workloads
```

**Raspberry Pi 5 (8GB):**
```
Total RAM:     8,192 MB
├── OS:          300 MB
├── K3s:       1,500 MB
├── NATS:        512 MB (configured limit)
└── Available: 5,880 MB for robot workloads

Total CPU:     4 cores (faster than Pi 4)
├── K3s:       ~25% (~1 core)
└── Available: ~75% (~3 cores) for robot workloads
```

---

## 4. RDL to Kubernetes Translation

### 4.1 Translation Overview

The `gorai` CLI translates Robot Definition Language (RDL) to Kubernetes manifests:

| RDL Concept | Kubernetes Resource |
|-------------|---------------------|
| `robot.name` | Namespace |
| `robot.namespace` | Label on all resources |
| Component | Part of gorai-core Deployment |
| Service (internal) | Part of gorai-core Deployment |
| Service (external) | Separate Deployment |
| `nats` config | NATS Deployment + Service |
| `dashboard` | Service (NodePort or LoadBalancer) |
| Device access | securityContext + hostPath volumes |
| Resource limits | resources.limits/requests |

### 4.2 Example Translation

**Input: robot.json**
```json
{
  "version": "3",
  "robot": {
    "name": "rover1",
    "description": "Autonomous rover with vision"
  },
  "nats": {
    "url": "nats://nats:4222"
  },
  "components": [
    {
      "name": "camera",
      "type": "camera",
      "model": "v4l2",
      "attributes": {
        "device": "/dev/video0",
        "width": 640,
        "height": 480
      }
    }
  ],
  "services": [
    {
      "name": "detector",
      "type": "vision",
      "model": "yolox",
      "external": {
        "enabled": true,
        "container": {
          "image": "ghcr.io/gorai/yolox-detector:latest",
          "devices": ["/dev/hailo0"]
        }
      },
      "attributes": {
        "confidence_threshold": 0.5
      }
    }
  ],
  "dashboard": {
    "enabled": true
  }
}
```

**Output: .gorai/manifests/namespace.yaml**
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: gorai-rover1
  labels:
    gorai.dev/robot: rover1
    gorai.dev/managed-by: gorai
```

**Output: .gorai/manifests/nats.yaml**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nats
  namespace: gorai-rover1
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nats
  template:
    metadata:
      labels:
        app: nats
    spec:
      containers:
      - name: nats
        image: nats:2.10-alpine
        args: ["-js", "-sd", "/data"]
        ports:
        - containerPort: 4222
        - containerPort: 8222
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        volumeMounts:
        - name: data
          mountPath: /data
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: nats-data
---
apiVersion: v1
kind: Service
metadata:
  name: nats
  namespace: gorai-rover1
spec:
  selector:
    app: nats
  ports:
  - name: client
    port: 4222
  - name: monitor
    port: 8222
```

**Output: .gorai/manifests/gorai-core.yaml**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gorai-core
  namespace: gorai-rover1
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gorai-core
  template:
    metadata:
      labels:
        app: gorai-core
    spec:
      containers:
      - name: gorai-core
        image: ghcr.io/gorai/gorai:latest
        args: ["run", "--config", "/etc/gorai/robot.json"]
        env:
        - name: NATS_URL
          value: "nats://nats:4222"
        - name: GORAI_ROBOT_NAME
          value: "rover1"
        ports:
        - containerPort: 8080
          name: dashboard
        - containerPort: 9091
          name: metrics
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "2"
        securityContext:
          privileged: false
        volumeMounts:
        - name: config
          mountPath: /etc/gorai
        - name: video0
          mountPath: /dev/video0
      volumes:
      - name: config
        configMap:
          name: robot-config
      - name: video0
        hostPath:
          path: /dev/video0
          type: CharDevice
---
apiVersion: v1
kind: Service
metadata:
  name: dashboard
  namespace: gorai-rover1
spec:
  type: NodePort
  selector:
    app: gorai-core
  ports:
  - name: http
    port: 8080
    nodePort: 30080
```

**Output: .gorai/manifests/detector.yaml**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: detector
  namespace: gorai-rover1
spec:
  replicas: 1
  selector:
    matchLabels:
      app: detector
  template:
    metadata:
      labels:
        app: detector
    spec:
      containers:
      - name: detector
        image: ghcr.io/gorai/yolox-detector:latest
        env:
        - name: NATS_URL
          value: "nats://nats:4222"
        - name: GORAI_ROBOT_NAME
          value: "rover1"
        - name: GORAI_SERVICE_NAME
          value: "detector"
        - name: CONFIDENCE_THRESHOLD
          value: "0.5"
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2"
        securityContext:
          privileged: true  # Required for Hailo device
        volumeMounts:
        - name: hailo
          mountPath: /dev/hailo0
      volumes:
      - name: hailo
        hostPath:
          path: /dev/hailo0
          type: CharDevice
```

### 4.3 Device Passthrough

Hardware devices are passed through using hostPath volumes with appropriate security contexts:

| Device Type | Volume Type | Security Context |
|-------------|-------------|------------------|
| Camera (/dev/video*) | hostPath CharDevice | runAsGroup: video (44) |
| I2C (/dev/i2c-*) | hostPath CharDevice | privileged: true or capabilities |
| SPI (/dev/spidev*) | hostPath CharDevice | privileged: true |
| GPIO (/dev/gpiomem) | hostPath CharDevice | runAsGroup: gpio |
| Hailo NPU (/dev/hailo*) | hostPath CharDevice | privileged: true |
| RK3588 NPU (/dev/dri/*) | hostPath CharDevice | runAsGroup: render, video |
| USB devices | hostPath CharDevice | depends on device |

**Note:** Some devices require `privileged: true`. Gorai minimizes this by using specific capabilities where possible.

---

## 5. CLI Commands

### 5.1 Deployment Commands

```bash
# Validate configuration
gorai validate robot.json

# Build container images (if using local images)
gorai build robot.json

# Deploy to local K3s cluster
gorai deploy robot.json

# Deploy and watch logs
gorai deploy robot.json --follow

# Deploy to remote cluster
gorai deploy robot.json --kubeconfig ~/.kube/robot-cluster

# Undeploy (remove from cluster)
gorai undeploy robot.json

# View generated manifests without deploying
gorai deploy robot.json --dry-run
```

### 5.2 Runtime Commands

```bash
# Check robot status
gorai status rover1

# View logs (all pods)
gorai logs rover1

# View logs (specific service)
gorai logs rover1 detector

# Follow logs
gorai logs rover1 -f

# Open dashboard in browser
gorai dashboard rover1

# Execute command in pod
gorai exec rover1 gorai-core -- /bin/sh

# Restart a service
gorai restart rover1 detector
```

### 5.3 Cluster Management

```bash
# Install K3s on current machine
gorai cluster init

# Join this machine to existing cluster
gorai cluster join <server-url> --token <token>

# Show cluster status
gorai cluster status

# List all robots in cluster
gorai list
```

---

## 6. Initial Setup

### 6.1 Single Robot Setup

```bash
# 1. Flash Raspberry Pi OS (64-bit, Lite) to SSD
# 2. Boot Pi with SSD connected via USB 3.0
# 3. SSH into Pi

# Install gorai CLI
curl -sfL https://get.gorai.dev | sh

# Initialize K3s cluster (single node)
gorai cluster init

# Wait for cluster to be ready (~2-3 minutes)
gorai cluster status

# Deploy your robot
gorai deploy robot.json

# Access dashboard
gorai dashboard my-robot
```

### 6.2 Fleet Setup

```bash
# On control plane node (edge server or first robot):
gorai cluster init --cluster-mode

# Note the join token and server URL
gorai cluster token

# On each robot:
gorai cluster join https://control-plane:6443 --token <token>

# Deploy robots (from control plane or CI/CD):
gorai deploy robots/robot1.json
gorai deploy robots/robot2.json
gorai deploy robots/robot3.json
```

### 6.3 GitOps Setup (Advanced)

For production fleets, use ArgoCD for GitOps deployment:

```bash
# Install ArgoCD on control plane
gorai addons install argocd

# Configure repository
gorai addons argocd add-repo https://github.com/myorg/robot-configs

# Robots auto-deploy when config is pushed to git
```

---

## 7. Networking

### 7.1 Pod Networking

All pods within a robot's namespace communicate via K3s internal networking:
- NATS is accessible at `nats:4222` within the namespace
- Services discover each other via DNS

### 7.2 External Access

| Service | Access Method | Default Port |
|---------|---------------|--------------|
| Dashboard | NodePort | 30080 |
| NATS (external) | NodePort (optional) | 30422 |
| Metrics | ClusterIP (internal) | 9091 |

### 7.3 Multi-Robot Communication

Robots communicate via NATS leaf nodes connecting to a hub:

```
Robot 1 (NATS Leaf) ──┐
Robot 2 (NATS Leaf) ──┼──► NATS Hub (Control Plane)
Robot 3 (NATS Leaf) ──┘
```

Configure in RDL:
```json
{
  "nats": {
    "url": "nats://nats:4222",
    "leaf": {
      "remotes": [{
        "url": "nats://control-plane.local:7422"
      }]
    }
  }
}
```

---

## 8. Observability

### 8.1 Prometheus Integration

Gorai automatically configures Prometheus scraping:

```yaml
# Annotations added to pods
prometheus.io/scrape: "true"
prometheus.io/port: "9091"
prometheus.io/path: "/metrics"
```

### 8.2 Logging

All container logs go to stdout/stderr and are collected by K3s:

```bash
# View via gorai CLI
gorai logs rover1

# Or via kubectl
kubectl logs -n gorai-rover1 -l app=gorai-core -f
```

### 8.3 Dashboard

The embedded dashboard provides:
- Component status
- Sensor readings (current and historical)
- Camera feeds
- Inference results
- Alert status

Access at: `http://<robot-ip>:30080`

---

## 9. Upgrades and Rollbacks

### 9.1 Rolling Updates

```bash
# Update robot configuration
gorai deploy robot.json

# K3s performs rolling update automatically
```

### 9.2 Rollback

```bash
# View deployment history
gorai history rover1

# Rollback to previous version
gorai rollback rover1

# Rollback to specific revision
gorai rollback rover1 --revision 3
```

### 9.3 K3s Upgrades

```bash
# Upgrade K3s (handled by system-upgrade-controller)
gorai cluster upgrade

# Or manual upgrade
curl -sfL https://get.k3s.io | INSTALL_K3S_CHANNEL=stable sh -
```

---

## 10. Troubleshooting

### 10.1 Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Pods stuck in Pending | Insufficient resources | Check `gorai status`, reduce resource requests |
| CrashLoopBackOff | Application error | Check `gorai logs <robot> <service>` |
| Device not accessible | Permission denied | Verify device exists, check securityContext |
| NATS connection refused | NATS pod not ready | Wait for NATS pod, check `gorai status` |
| Slow kubectl responses | SD card storage | Migrate to SSD (mandatory) |
| Database corruption | Power loss + SD card | Use SSD, add UPS |

### 10.2 Debug Commands

```bash
# Detailed status
gorai status rover1 --verbose

# Check events
gorai events rover1

# Describe a failing pod
gorai describe rover1 detector

# Interactive shell in pod
gorai exec rover1 gorai-core -- /bin/sh

# Check K3s system pods
kubectl get pods -n kube-system

# Check node resources
kubectl top nodes
kubectl top pods -n gorai-rover1
```

### 10.3 Recovery

```bash
# Restart all pods for a robot
gorai restart rover1

# Force delete stuck pods
gorai delete rover1 detector --force

# Reinstall K3s (last resort)
gorai cluster reset
gorai cluster init
```

---

## 11. Security

### 11.1 Pod Security

- Pods run as non-root where possible
- Privileged mode only for hardware access (minimized)
- Network policies isolate robot namespaces
- Secrets stored in K3s secrets (not ConfigMaps)

### 11.2 Cluster Security

- K3s API secured with TLS
- Service account tokens for pod authentication
- RBAC limits namespace access
- No external access by default (NodePort explicit)

### 11.3 Secrets Management

```json
{
  "nats": {
    "credentials_file": "${NATS_CREDS}",
    "tls": {
      "ca_file": "/etc/gorai/tls/ca.crt",
      "cert_file": "/etc/gorai/tls/client.crt",
      "key_file": "/etc/gorai/tls/client.key"
    }
  }
}
```

Secrets referenced via environment variables are stored in Kubernetes Secrets and mounted into pods.

---

## Appendix A: K3s Configuration

### A.1 Default K3s Options

Gorai installs K3s with these options:

```bash
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="server \
  --disable traefik \
  --disable servicelb \
  --write-kubeconfig-mode 644 \
  --kube-apiserver-arg=enable-admission-plugins=NodeRestriction \
  " sh -
```

**Disabled components:**
- `traefik`: Gorai uses NodePort for dashboard (simpler for single-robot)
- `servicelb`: Not needed for edge deployment

### A.2 High Availability

For production fleets, use embedded etcd HA:

```bash
# First server
gorai cluster init --cluster-mode --ha

# Additional servers
gorai cluster join https://server1:6443 --token <token> --server
```

Requires 3+ server nodes for etcd quorum.

---

## Appendix B: Migration from Tiered Deployment

If migrating from previous Gorai versions with systemd/Podman deployment:

1. **Backup configuration**: `cp robot.json robot.json.backup`
2. **Install K3s**: `gorai cluster init`
3. **Update RDL version**: Change `"version": "2"` to `"version": "3"`
4. **Deploy**: `gorai deploy robot.json`
5. **Verify**: `gorai status <robot-name>`
6. **Remove old services**: `sudo systemctl disable gorai-*`

The RDL format is largely compatible; main changes are in deployment semantics.
