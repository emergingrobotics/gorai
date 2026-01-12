# Gorai Strategic Summary

**Quick reference for developers and contributors**

This document summarizes the key strategic decisions that guide Gorai's development. For comprehensive analysis, see [vision-analysis.md](vision-analysis.md).

---

## Core Positioning

### What We Are
- **Prosumer robotics framework** — real autonomy between educational toys ($100-300) and enterprise platforms ($4,000+)
- **Cloud-native architecture** — NATS, Prometheus, K3s (battle-tested distributed systems)
- **Go-first, pragmatically polyglot** — Go core, Python/C++ services via NATS when appropriate

### What We're NOT
- **Not a ROS 2 replacement** — we target different markets (prosumer vs. enterprise/research)
- **Not language-purist** — we use the best tool for each job
- **Not anti-ROS 2** — bridge planned (Phase 3) for ecosystem compatibility
- **Not enterprise-focused** — we optimize for accessibility, not feature completeness

---

## Key Strategic Decisions

### 1. Language Strategy: Pragmatic Polyglot

**Decision**: Go core + polyglot services via NATS clients

| Component Type | Language | Rationale |
|----------------|----------|-----------|
| Framework core | Pure Go | Concurrency, deployment, AI-assisted coding |
| Simple sensors | Pure Go | GPIO, I2C, GPS, IMU — protocol parsing |
| Vision preprocessing | Python | OpenCV, scikit-image ecosystem |
| ML inference | Python or ONNX Runtime (C++) | PyTorch/TensorFlow training; ONNX deployment |
| SLAM | C++ with Go wrapper | Cartographer, ORB-SLAM too valuable to rewrite |
| Camera drivers | cgo wrappers | V4L2, RealSense SDKs in C/C++ |
| Motor controllers | Go or cgo | If SDK exists (Dynamixel), wrap; else pure Go |
| Web UI | Go templates + HTMX | Avoid separate JS frontend complexity |

**Why**: NATS has clients for 40+ languages. Any language can be a Gorai service.

### 2. ROS 2 Positioning: Coopetition, Not Competition

**Decision**: Build compatibility bridge, don't compete directly

```
DO:
- Build ROS 2 bridge (Phase 3)
- Position as "ROS 2 for prosumers"
- Reuse ROS 2 message type designs
- Document ROS 2 migration paths
- Collaborate with ROS 2 community

DON'T:
- Try to replace ROS 2 in research/enterprise
- Ignore ROS 2 ecosystem
- Start language wars
- Lock users into Gorai-only world
```

**Why ROS 2 Bridge Matters**:
1. **Component reuse** — use ROS 2 drivers when no Go alternative exists
2. **Simulation** — Gazebo integration via bridge
3. **Migration path** — students can keep ROS 2 packages they know
4. **Ecosystem perception** — avoid "vendor lock-in" perception

### 3. Deployment: K3s-Everywhere

**Decision**: All robots deploy on K3s, from single Pi to multi-robot fleets

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

**Why K3s-Everywhere?**
- **One deployment model** — same approach from 1 robot to 100+
- **Production-grade orchestration** — health checks, rolling updates, resource limits
- **Fleet-ready from day one** — no migration when scaling
- **Modern hardware handles it** — ~512 MB overhead acceptable on 4GB+ systems
- **RDL abstracts complexity** — users never need kubectl knowledge

**What is K3s?**
- Lightweight Kubernetes (~70MB binary vs. ~1GB)
- 512MB RAM vs. 1GB for full Kubernetes
- Single binary install: `curl -sfL https://get.k3s.io | sh -`
- Certified Kubernetes (full API compatibility)
- Designed for edge computing, IoT, and resource-constrained devices

### 4. Cloud Patterns vs. ROS 2

**Decision**: Embrace cloud-native patterns as competitive advantage

| Cloud Pattern | ROS 2 Approach | Gorai Approach | Advantage |
|---------------|----------------|----------------|-----------|
| Message Broker | DDS peer-to-peer | NATS server | Decouples components, easy monitoring |
| Service Mesh | Custom discovery | NATS queue groups | Automatic load balancing |
| Event Sourcing | rosbag (manual) | JetStream (built-in) | Replay streams, time-travel debug |
| Config Management | Per-node params | NATS KV store | Global config, hot reload |
| Observability | Custom diagnostics | Prometheus /metrics | Industry-standard tools |
| Orchestration | Manual | K3s | Health checks, rolling updates |

**Why**: ROS 2's DDS is pre-cloud (2004 LAN design). We use infrastructure proven at global scale.

---

## Development Priorities

### Phase 1: Core Framework (Current)
1. NATS messaging, Resource model, Configuration
2. Basic sensors (GPS, IMU, compass) in pure Go
3. Motor control (I2C/PWM) in pure Go
4. Web dashboard in Go
5. K3s deployment infrastructure

### Phase 2: First Product
1. Marine-specific sensors
2. Waypoint navigation
3. Mission planner
4. LoRa telemetry
5. Hardware design finalized

### Phase 3: Ecosystem
1. ROS 2 bridge MVP (ROS2 → NATS)
2. Gazebo simulation integration
3. Drive (wheeled robot) hardware
4. Community drivers

### Phase 4: Advanced
1. SLAM integration (Cartographer via C++)
2. Fleet management dashboard
3. Advanced ML (object tracking, SLAM)

---

## Target Market Definition

### Primary Users
- **Makers** — building capable autonomous robots (not toys)
- **Citizen scientists** — monitoring watersheds, marine environments
- **Students/educators** — teaching robotics without PhD toolchains
- **Small organizations** — needing autonomy without enterprise budgets
- **Hobbyists** — want more than Arduino, less complexity than ROS 2

### NOT Target Users
- Enterprise warehouse robotics → use ROS 2 + AMR vendors
- Autonomous vehicle research → use ROS 2 + Autoware
- PhD robotics research → use ROS 2, YARP
- Defense/aerospace → use certified systems
- Medical robotics → use certified systems

---

## Messaging Guidelines

### Positioning Statements

**DO say**:
- "Professional robotics for prosumers — without the PhD"
- "Gorai makes professional robotics accessible"
- "We learned from ROS 2's decades of experience"
- "Go core, best tool for each job"
- "Cloud-native patterns proven at scale"

**DON'T say**:
- "Gorai is better than ROS 2" (different markets)
- "Pure Go or nothing" (we're pragmatic)
- "We reinvented everything" (we borrowed proven patterns)
- "ROS 2 is bad" (it's excellent for its target market)

### Competitive Positioning

Not "vs. ROS 2" — different markets:
- **Gorai**: Prosumer, days to productivity, accessible
- **ROS 2**: Enterprise/research, months to mastery, comprehensive

Think: "ROS 2 for prosumers," not "ROS 2 killer"

---

## Risk Mitigation

### Risk 1: Go Ecosystem Gaps
**Mitigation**:
- Support polyglot services (Python/C++ via NATS)
- Provide cgo wrappers for critical libraries
- Build community driver ecosystem
- Document escape hatches

### Risk 2: "Not Invented Here" Perception
**Mitigation**:
- Build ROS 2 bridge early (Phase 3)
- Collaborate with ROS 2 community
- Credit ROS 2 for design inspiration
- Show pragmatism over purity

### Risk 3: K3s Overhead Concerns
**Mitigation**:
- Target hardware with 4GB+ RAM (Pi 5, Jetson, Orange Pi 5B)
- K3s overhead (~512MB) acceptable on modern hardware
- Benefits (orchestration, fleet-ready) outweigh costs
- Document hardware requirements clearly

### Risk 4: Limited Adoption
**Mitigation**:
- Free tutorials on accessible hardware
- Clear examples with working code
- Open source core (Apache 2.0)
- No vendor lock-in (NATS = open)

---

## Contributing Guidelines

When contributing to Gorai:

1. **Read vision-analysis.md** — understand strategic context
2. **Follow language strategy** — Go core, polyglot when appropriate
3. **Don't fight ROS 2** — we're complementary, not competitive
4. **Pragmatism over purity** — best tool for job
5. **Keep it simple** — complexity only when needed
6. **Document design decisions** — AI-assisted dev requires clarity

---

## Further Reading

- [Vision Analysis](vision-analysis.md) — Comprehensive strategic analysis
- [Design Comparison](general-designs.md) — ROS 2, Viam, YARP analysis
- [Framework Specification](../specs/gorai-framework-specification.md) — Technical spec
- [Code Organization](../specs/code-organization.md) — Module structure
- [K3s Installation](../specs/k3s-installation.md) — Platform-specific K3s setup

---

**Last Updated**: 2025-01-12
**Status**: Active strategic guidance
