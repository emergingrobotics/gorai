# Gorai Future Roadmap

This document describes the planned evolution of Gorai from simple binary deployment to full distributed fleet management. The architecture is designed to be progressive — start simple, add complexity only when needed.

---

## Current State: Phase 1 - Simple Binary

**Status: Active Development**

The current architecture focuses on simplicity and accessibility:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Single Go Binary                              │
│                                                                  │
│   ├── Components (GPS, GPIO, sensors, motors)                   │
│   ├── Behaviors (state machines, scripts)                       │
│   └── Message Router (NATS client)                              │
│                                                                  │
│   External: NATS Server (systemd service)                       │
└─────────────────────────────────────────────────────────────────┘
```

### Characteristics
- Single binary (~10-20MB)
- NATS as only external dependency
- ~20-50MB RAM usage
- No containers, no K8s
- Direct systemd integration

### Target Users
- Makers building their first robot
- Students learning robotics
- Rapid prototyping
- Single-device deployments

### When to Stay in Phase 1
- All components can be implemented in Go
- No ML/vision processing required
- Single robot deployment
- Resource-constrained devices (< 2GB RAM)

---

## Phase 2: Optional Containers (Future)

**Status: Planned**

Add containerized services for capabilities that benefit from Python or C++:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Robot System                                  │
│                                                                  │
│   ┌─────────────────┐     ┌─────────────────┐                   │
│   │  Go Binary      │     │  Container      │                   │
│   │  (core robot)   │────▶│  (vision/ML)    │                   │
│   └─────────────────┘     └─────────────────┘                   │
│            │                       │                             │
│            └───────────┬───────────┘                             │
│                        ▼                                         │
│                   NATS Server                                    │
└─────────────────────────────────────────────────────────────────┘
```

### When to Move to Phase 2
- Need vision processing (OpenCV, Python)
- Need ML inference (PyTorch, TensorFlow)
- Need SLAM (Cartographer, C++)
- Have sufficient resources (4GB+ RAM)

### Architecture Changes
- Add Podman for container management
- Container services communicate via NATS
- Go binary remains the orchestrator
- RDL extended with `services` section

### Preserved Documentation
- [Container Orchestration Design](archive/future-state/systemd-container-orchestration-v2.md)
- [Container Build Specification](archive/future-state/gorai-container-v2.md)

---

## Phase 3: K3s Fleet Management (Future)

**Status: Planned**

Full Kubernetes-based orchestration for production fleets:

```
┌─────────────────────────────────────────────────────────────────┐
│                    K3s Cluster                                   │
│                                                                  │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│   │  Robot 1    │  │  Robot 2    │  │  Robot N    │             │
│   │  (namespace)│  │  (namespace)│  │  (namespace)│             │
│   └─────────────┘  └─────────────┘  └─────────────┘             │
│          │               │               │                       │
│          └───────────────┴───────────────┘                       │
│                          │                                       │
│                   ┌──────▼──────┐                                │
│                   │ NATS Cluster│                                │
│                   │ (JetStream) │                                │
│                   └─────────────┘                                │
└─────────────────────────────────────────────────────────────────┘
```

### When to Move to Phase 3
- Managing multiple robots (fleet)
- Need rolling updates without downtime
- Need resource isolation and limits
- Need health checks and auto-restart
- Production deployment requirements

### Architecture Changes
- K3s cluster (single-node or multi-node)
- Each robot in separate namespace
- NATS deployed as K3s service
- Full orchestration via `gorai deploy`
- Users still work with RDL (same format)

### Preserved Documentation
- [K3s Installation Guide](archive/future-state/k3s-installation.md)
- [K3s Deployment Design](archive/future-state/deployment-k3s-v3.md)

---

## Phase 4: Advanced Ecosystem (Future)

**Status: Planned**

Extended ecosystem capabilities:

### ROS 2 Bridge
- Bidirectional NATS <-> ROS 2 communication
- Use ROS 2 drivers when no Go alternative exists
- Gazebo simulation integration
- Migration path for ROS 2 users

### Fleet Management Dashboard
- Web-based fleet monitoring
- Mission planning interface
- Real-time telemetry visualization
- Multi-robot coordination

### Advanced AI/ML
- SLAM integration (Cartographer via C++)
- Object tracking and recognition
- VLM (Vision Language Models) integration
- Edge-cloud hybrid inference

---

## Migration Path

### Phase 1 → Phase 2

When you need containerized services:

1. Install Podman on your robot
2. Add `services` section to RDL
3. Build container images for vision/ML services
4. Services communicate via existing NATS topics

**RDL changes (additive):**
```json
{
  "name": "my-robot",
  "nats": {"url": "nats://localhost:4222"},
  "components": [...],
  "services": [
    {
      "name": "vision",
      "image": "gorai/vision-opencv:latest",
      "topics": {
        "subscribe": ["camera.frames"],
        "publish": ["vision.detections"]
      }
    }
  ]
}
```

### Phase 2 → Phase 3

When you need fleet management:

1. Install K3s on your robot(s)
2. Change deployment command: `gorai run` → `gorai deploy`
3. Add optional resource limits to RDL
4. Multi-robot coordination via NATS namespaces

**No RDL changes required** — the same configuration works, just deployed differently.

---

## Archived Documentation

The following documentation is preserved for future implementation:

### Container/Podman (Phase 2)
- [systemd-container-orchestration-v2.md](archive/future-state/systemd-container-orchestration-v2.md) — Container lifecycle management
- [gorai-container-v2.md](archive/future-state/gorai-container-v2.md) — Container build specification

### K3s/Kubernetes (Phase 3)
- [k3s-installation.md](archive/future-state/k3s-installation.md) — Platform-specific K3s setup
- [deployment-k3s-v3.md](archive/future-state/deployment-k3s-v3.md) — K3s deployment architecture

### Archived Code

Container and K3s-related code is preserved in `/archive/`:
- `archive/pkg/quadlet/` — Quadlet systemd integration
- `archive/pkg/runtime/container_runner.go` — Container orchestration
- `archive/examples/` — K3s-based examples

---

## Decision Framework

Use this framework to decide which phase is appropriate:

| Question | Phase 1 | Phase 2 | Phase 3 |
|----------|---------|---------|---------|
| How many robots? | 1 | 1 | 2+ |
| Need ML/vision? | No | Yes | Yes |
| Available RAM | < 2GB | 4GB+ | 4GB+ |
| Need rolling updates? | No | No | Yes |
| Production deployment? | No | Maybe | Yes |
| Container experience? | No | Some | Yes |

---

## Contributing to Future Phases

If you're interested in contributing to Phase 2 or Phase 3 development:

1. Review the archived documentation
2. Check existing designs for consistency
3. Propose changes via GitHub issues
4. Maintain backward compatibility with Phase 1 RDL

The core principle: **Users should be able to start with Phase 1 and progressively add complexity without rewriting their robot configurations.**

---

**Last Updated**: 2025-01-24
