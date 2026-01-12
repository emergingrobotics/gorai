# Gorai

**Professional robotics for prosumers — without the PhD**

Gorai is a Go-based robotics framework designed for makers, citizen scientists, students, and small organizations who need real autonomy without ROS 2's complexity. Built on battle-tested cloud infrastructure (NATS, Prometheus, K3s), Gorai applies distributed systems patterns to robotics.

## Strategic Positioning

**Target Market**: Prosumers who find Arduino too limiting and ROS 2 too complex.

**Not competing with ROS 2** in enterprise/research. We're "ROS 2 for prosumers," not "ROS 2 killer."

See [docs/vision-analysis.md](docs/vision-analysis.md) for comprehensive strategic analysis.

## Core Architecture

- **K3s-everywhere** — All robots deploy on Kubernetes (K3s); RDL abstracts complexity
- **Consistent deployment model** — ~512 MB overhead; single-node to multi-robot fleets
- **Go core + pragmatic polyglot services** — Go for orchestration; Python/C++ via containers
- **Cloud-native patterns** — NATS (message broker), Prometheus (observability), K3s (orchestration)
- **RDL abstracts deployment** — Users write JSON/YAML configs; `gorai deploy` handles K3s manifests
- **Edge AI focus** — Hailo (26 TOPS), Jetson (67 TOPS), RK3588 NPU (6 TOPS)

## Deployment Model

```
User: robot.json → gorai deploy → Robot running on K3s

What runs:
├── K3s cluster (single-node or multi-node)
├── NATS pod (message broker)
├── gorai-core pod (Go orchestration + components)
└── Service pods (vision, SLAM, etc.)
```

Users never need kubectl or K8s knowledge — the CLI abstracts it.

## Hardware Platforms

| Platform | Role | AI Performance | Storage |
|----------|------|----------------|---------|
| **Raspberry Pi 5 (8GB)** | Primary | External (Hailo 13-26 TOPS) | NVMe SSD |
| **Jetson Orin Nano Super** | Performance | 67 TOPS (CUDA) | NVMe SSD |
| **Orange Pi 5B (8GB)** | Budget AI | 6 TOPS (built-in) | 64GB eMMC |

**Not supported:** Pi 3, Pi Zero, Pi 4 (2GB), SD card-only deployments

See [specs/hardware-requirements.md](specs/hardware-requirements.md) for full requirements.

## Key Documentation

- [specs/k3s-installation.md](specs/k3s-installation.md) — K3s setup by platform
- [specs/robot-definition-language.md](specs/robot-definition-language.md) — RDL configuration
- [specs/hardware-requirements.md](specs/hardware-requirements.md) — Hardware specs
- [specs/gorai-framework-specification.md](specs/gorai-framework-specification.md) — Full technical spec
