# Gorai

<img src="./images/gorai.png" width="25%">

**Professional robotics for prosumers — without the PhD**

*Pronounced "go-ray" (like "sting-ray")*

Gorai is a Go-based robotics framework designed for makers, citizen scientists, students, and small organizations who need real autonomy without ROS 2's complexity. Built on battle-tested cloud infrastructure (NATS, Prometheus, K3s), Gorai applies distributed systems patterns to robotics.

---

## Who Is Gorai For?

**Use Gorai if you:**
- Want **real autonomy** (not educational toys, not simulations)
- Need **approachable software** (productive in days, not months)
- Value **modern tooling** (AI-assisted coding, simple deployment)
- Are building **prosumer robots** (marine monitoring, land vehicles, research platforms)

**Use ROS 2 if you:**
- Work in **enterprise/research** robotics (warehouse automation, autonomous vehicles)
- Need the **full ROS ecosystem** (thousands of packages, simulation, SLAM libraries)
- Are in **academia** where ROS 2 is the standard

We're not replacing ROS 2 — we're targeting a different market. Think "ROS 2 for prosumers."

---

## Architecture: K3s-Everywhere

Gorai uses **K3s** (Lightweight Kubernetes) for all deployments. Every robot runs the same way, from a single Raspberry Pi to a multi-robot fleet.

```
┌─────────────────────────────────────────────────────────────────┐
│                    User Experience                               │
│                                                                  │
│   robot.json (RDL)  →  gorai deploy  →  Robot running           │
│                                                                  │
│   Users work with: Robot Definition Language (JSON/YAML)        │
│   Users never need: kubectl, manifests, pods, deployments       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    What Actually Runs                            │
│                                                                  │
│   K3s Cluster (single-node or multi-node)                       │
│   ├── Namespace: gorai-{robot-name}                             │
│   ├── Pod: nats (message broker)                                │
│   ├── Pod: gorai-core (Go orchestration + components)           │
│   └── Pod: {services} (vision, SLAM, navigation)                │
└─────────────────────────────────────────────────────────────────┘
```

**Why K3s?** AI at the edge requires capable hardware. Capable hardware can run K3s (~512 MB overhead). One deployment model from 1 robot to 100+.

---

## Quick Start

### Prerequisites

- Raspberry Pi 5 (8GB) or equivalent — see [Hardware Requirements](specs/hardware-requirements.md)
- NVMe SSD or USB 3.0 SSD (SD cards not supported for K3s)
- Linux (Raspberry Pi OS 64-bit recommended)

### 1. Install K3s

```bash
curl -sfL https://get.k3s.io | sh -s - \
  --disable traefik \
  --write-kubeconfig-mode 644

# Verify
sudo k3s kubectl get nodes
```

See [K3s Installation Guide](specs/k3s-installation.md) for platform-specific instructions.

### 2. Install Gorai CLI

```bash
curl -sfL https://get.gorai.dev | sh
```

### 3. Create a Robot Configuration

```json
{
  "$schema": "https://gorai.dev/schemas/rdl-v3.json",
  "version": "3",
  "robot": {
    "name": "my-robot",
    "description": "Example robot with camera"
  },
  "components": [
    {
      "name": "main_camera",
      "type": "camera",
      "model": "v4l2",
      "attributes": {
        "device": "/dev/video0",
        "width": 640,
        "height": 480
      }
    }
  ],
  "dashboard": {
    "enabled": true
  }
}
```

### 4. Deploy

```bash
gorai deploy robot.json
gorai status my-robot
gorai logs my-robot -f
```

---

## Hardware Platforms

| Platform | AI Performance | Cost | Best For |
|----------|----------------|------|----------|
| **Raspberry Pi 5 (8GB)** | External (Hailo 13-26 TOPS) | ~$160 | Primary platform, best ecosystem |
| **Jetson Orin Nano Super** | 67 TOPS (CUDA) | ~$335 | Maximum AI, VLMs |
| **Orange Pi 5B (8GB)** | 6 TOPS (built-in NPU) | ~$145 | Budget AI builds |

**Not supported:** Pi 3, Pi Zero, Pi 4 (2GB), SD card-only deployments

See [Hardware Requirements](specs/hardware-requirements.md) for details.

---

## Examples

| Example | Description | Status |
|---------|-------------|--------|
| [hello-robot](examples/hello-robot/) | Basic NATS pub/sub messaging | ✅ Working |
| [hello-robot-production](examples/hello-robot-production/) | Production-ready with health checks | ✅ Working |
| [hello-camera](examples/hello-camera/) | Camera capture + web dashboard | 🚧 In Progress |
| [hello-people-detector](examples/hello-people-detector/) | AI person detection with Hailo NPU | 🚧 In Progress |

---

## CLI Commands

| Command | Description |
|---------|-------------|
| `gorai deploy <config>` | Deploy robot to K3s cluster |
| `gorai undeploy <name>` | Remove robot from cluster |
| `gorai status <name>` | Show robot status |
| `gorai logs <name> -f` | Stream robot logs |
| `gorai dashboard <name>` | Open web dashboard |
| `gorai validate <config>` | Validate configuration |
| `gorai build <config>` | Build container images |

---

## Documentation

### Getting Started
- [K3s Installation Guide](specs/k3s-installation.md) — Platform-specific K3s setup
- [Hardware Requirements](specs/hardware-requirements.md) — Supported platforms and specs
- [Robot Definition Language](specs/robot-definition-language.md) — RDL configuration format

### Architecture & Design
- [Framework Specification](specs/gorai-framework-specification.md) — Complete technical spec
- [Vision Analysis](docs/vision-analysis.md) — Strategic architecture assessment
- [Design Comparison](docs/general-designs.md) — Analysis of ROS 2, Viam, YARP

### Reference
- [Code Organization](specs/code-organization.md) — Module structure
- [Runtime Specification](specs/runtime.md) — Robot lifecycle
- [Component Reference](docs/component-reference.md) — Component APIs

---

## Why Gorai?

### Cloud-Native Patterns

| Pattern | ROS 2 | Gorai | Benefit |
|---------|-------|-------|---------|
| **Message Broker** | DDS peer-to-peer | NATS server | Decoupled, easy monitoring |
| **Event Sourcing** | rosbag (manual) | JetStream (built-in) | Replay, time-travel debug |
| **Observability** | Custom diagnostics | Prometheus /metrics | Industry-standard tools |
| **Orchestration** | Manual | K3s | Health checks, rolling updates |

### Language Strategy

- **Go core** — NATS orchestration, configuration, web dashboard
- **Python services** — Vision (OpenCV), ML inference (PyTorch)
- **C++ services** — SLAM (Cartographer), point cloud processing

NATS has clients for 40+ languages. Use the best tool for each job.

---

## Project Status

**Version:** 0.1.0 (Early Development)

### Roadmap

**Phase 1: Core Framework** — ✅ Complete
- NATS messaging, Resource model, Configuration, Web dashboard

**Phase 2: First Product** — 🔄 In Progress
- Marine sensors, Waypoint navigation, Mission planner

**Phase 3: Ecosystem** — ⏳ Planned
- ROS 2 bridge, Gazebo simulation, Community drivers

---

## Contributing

Gorai is built with [Claude Code](https://claude.ai/claude-code). We believe AI-assisted development is the future — the project is organized for both humans and AI to reason about effectively.

See [CLAUDE.md](CLAUDE.md) for contributor guidelines.

---

## License

Apache 2.0
