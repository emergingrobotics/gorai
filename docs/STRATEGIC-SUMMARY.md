# Gorai Strategic Summary

**Quick reference for developers and contributors**

This document summarizes the key strategic decisions that guide Gorai's development. For comprehensive analysis, see [vision-analysis.md](vision-analysis.md).

---

## Core Positioning

### What We Are
- **Prosumer robotics framework** — real autonomy between educational toys ($100-300) and enterprise platforms ($4,000+)
- **Simple binary deployment** — single Go binary, NATS messaging, no containers required
- **Go-first, pragmatically polyglot** — Go core, Python/C++ services via NATS when appropriate (future)

### What We're NOT
- **Not a ROS 2 replacement** — we target different markets (prosumer vs. enterprise/research)
- **Not language-purist** — we use the best tool for each job
- **Not anti-ROS 2** — bridge planned (future) for ecosystem compatibility
- **Not enterprise-focused** — we optimize for accessibility, not feature completeness

---

## Phased Architecture

Gorai follows a progressive architecture that starts simple and adds complexity only when needed:

### Phase 1: Simple Binary (Current)

```
┌─────────────────────────────────────────────────────────────────┐
│                    User Experience                               │
│                                                                  │
│   robot.json (RDL)  →  gorai run  →  Robot running              │
│                                                                  │
│   Users work with: Robot Definition Language (JSON)             │
│   Output: Single binary (~10-20MB)                              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    What Actually Runs                            │
│                                                                  │
│   Single Go Binary                                               │
│   ├── Components (GPS, GPIO, sensors, motors)                   │
│   ├── Message Router (NATS client)                              │
│   └── Behaviors (state machines, scripts)                       │
│                                                                  │
│   NATS Server (systemd service)                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Why Simple Binary First?**
- **Fastest time to working robot** — under 1 hour
- **Minimal dependencies** — just NATS server
- **Low resource usage** — ~20-50MB RAM vs 512MB+ for containers
- **Easy debugging** — single process, standard Go tooling
- **Progressive path** — graduate to containers/K3s when needed

### Phase 2: Optional Containers (Future)

For ML/vision services that require Python or C++:
- Containerized services via Podman
- Same NATS messaging backbone
- Mix native Go components with containerized services

### Phase 3: K3s Fleet Management (Future)

For production fleets and multi-robot coordination:
- K3s orchestration for fleet management
- Rolling updates, health checks, resource limits
- Same RDL configuration, expanded deployment

See [Future Roadmap](FUTURE-ROADMAP.md) and [archived K3s designs](archive/future-state/) for details.

---

## Key Strategic Decisions

### 1. Language Strategy: Pragmatic Polyglot

**Decision**: Go core + polyglot services via NATS clients

| Component Type | Language | Rationale |
|----------------|----------|-----------|
| Framework core | Pure Go | Concurrency, deployment, AI-assisted coding |
| Simple sensors | Pure Go | GPIO, I2C, GPS, IMU — protocol parsing |
| Vision preprocessing | Python | OpenCV, scikit-image ecosystem (future) |
| ML inference | Python or ONNX Runtime (C++) | PyTorch/TensorFlow training; ONNX deployment (future) |
| SLAM | C++ with Go wrapper | Cartographer, ORB-SLAM too valuable to rewrite (future) |
| Camera drivers | cgo wrappers | V4L2, RealSense SDKs in C/C++ |
| Motor controllers | Go or cgo | If SDK exists (Dynamixel), wrap; else pure Go |
| Web UI | Go templates + HTMX | Avoid separate JS frontend complexity |

**Why**: NATS has clients for 40+ languages. Any language can be a Gorai service.

### 2. ROS 2 Positioning: Coopetition, Not Competition

**Decision**: Build compatibility bridge, don't compete directly

```
DO:
- Build ROS 2 bridge (future phase)
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

### 3. Deployment: Progressive Complexity

**Decision**: Start simple, add orchestration only when needed

| Phase | Model | Overhead | Use Case |
|-------|-------|----------|----------|
| **Phase 1 (Current)** | Single binary | ~20-50MB | Learning, prototyping, single robots |
| **Phase 2 (Future)** | Binary + containers | ~100-200MB | ML/vision services |
| **Phase 3 (Future)** | K3s | ~512MB | Fleet management, production |

**Why Progressive?**
- Lower barrier to entry for beginners
- Faster iteration during development
- Add complexity only when value justifies it
- Same RDL format scales across all phases

### 4. Cloud Patterns vs. ROS 2

**Decision**: Embrace cloud-native patterns as competitive advantage

| Cloud Pattern | ROS 2 Approach | Gorai Approach | Advantage |
|---------------|----------------|----------------|-----------|
| Message Broker | DDS peer-to-peer | NATS server | Decouples components, easy monitoring |
| Service Mesh | Custom discovery | NATS queue groups | Automatic load balancing |
| Event Sourcing | rosbag (manual) | JetStream (built-in) | Replay streams, time-travel debug |
| Config Management | Per-node params | NATS KV store | Global config, hot reload |
| Observability | Custom diagnostics | Prometheus /metrics | Industry-standard tools |
| Orchestration | Manual | systemd / K3s (future) | Health checks, rolling updates |

**Why**: ROS 2's DDS is pre-cloud (2004 LAN design). We use infrastructure proven at global scale.

---

## Development Priorities

### Phase 1: Simple Binary Framework (Current)
1. NATS messaging, Resource model, Configuration
2. Basic sensors (GPS, IMU, compass) in pure Go
3. Motor control (I2C/PWM) in pure Go
4. CLI: validate, run, build, components
5. Simple examples (GPS tracker, blinky LED)

### Phase 2: Hardware Expansion
1. Camera (v4l2) integration
2. Marine-specific sensors
3. Waypoint navigation
4. Mission planner
5. Web dashboard

### Phase 3: Containerized Services (Future)
1. Vision services (Python/OpenCV)
2. ML inference services
3. Container orchestration (Podman)
4. Mixed native/container deployment

### Phase 4: Fleet & Ecosystem (Future)
1. K3s deployment infrastructure
2. ROS 2 bridge MVP (ROS2 <-> NATS)
3. Gazebo simulation integration
4. Fleet management dashboard
5. SLAM integration (Cartographer via C++)

---

## Target Market Definition

### Primary Users
- **Makers** — building capable autonomous robots (not toys)
- **Citizen scientists** — monitoring watersheds, marine environments
- **Students/educators** — teaching robotics without PhD toolchains
- **Small organizations** — needing autonomy without enterprise budgets
- **Hobbyists** — want more than Arduino, less complexity than ROS 2

### NOT Target Users
- Enterprise warehouse robotics -> use ROS 2 + AMR vendors
- Autonomous vehicle research -> use ROS 2 + Autoware
- PhD robotics research -> use ROS 2, YARP
- Defense/aerospace -> use certified systems
- Medical robotics -> use certified systems

---

## Messaging Guidelines

### Positioning Statements

**DO say**:
- "Professional robotics for prosumers — without the PhD"
- "Gorai makes professional robotics accessible"
- "Build a robot in under an hour"
- "Single binary, no containers required"
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
- Plan ROS 2 bridge for ecosystem compatibility
- Collaborate with ROS 2 community
- Credit ROS 2 for design inspiration
- Show pragmatism over purity

### Risk 3: Scaling Beyond Simple Binary
**Mitigation**:
- Progressive architecture design allows growth
- Container support planned for Phase 2
- K3s orchestration preserved for Phase 3
- Same RDL format works across all phases

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
- [Future Roadmap](FUTURE-ROADMAP.md) — Container/K3s expansion plans
- [Archived K3s Designs](archive/future-state/) — Preserved future state documentation

---

**Last Updated**: 2025-01-24
**Status**: Active strategic guidance
