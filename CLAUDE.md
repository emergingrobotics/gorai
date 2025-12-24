# Gorai

**Professional robotics for prosumers — without the PhD**

Gorai is a Go-based robotics framework designed for makers, citizen scientists, students, and small organizations who need real autonomy without ROS 2's complexity. Built on battle-tested cloud infrastructure (NATS, Prometheus), Gorai applies distributed systems patterns to robotics.

## Strategic Positioning

**Target Market**: Prosumers who find Arduino too limiting and ROS 2 too complex.

**Not competing with ROS 2** in enterprise/research. We're "ROS 2 for prosumers," not "ROS 2 killer."

See [docs/vision-analysis.md](docs/vision-analysis.md) for comprehensive strategic analysis.

## Core Attributes

- **Distributed systems thinking** — Multiple processes coordinated via NATS; scalable from single process to multi-node clusters
- **Go core + pragmatic polyglot services** — Go for orchestration; Python/C++ via NATS for vision/SLAM
- **Cloud-native patterns** — NATS (message broker), Prometheus (observability), JetStream (event sourcing)
- **Tiered deployment** — systemd (simple) → Podman pods (research) → K3s (fleet); complexity matches needs
- **AI-assisted development** — designed for modern AI-powered coding
- **Low barrier to entry** — productive in days, not months
- **Edge AI focus** — TPU/NPU acceleration built-in
- **ROS 2 bridge planned** — Phase 3 for ecosystem compatibility
- **Modular architecture** — unified Resource interface for all components/services
- **Linux-first** — Raspberry Pi 5 reference platform
- **HAVE FUN!**

## Target Audience

We aim to become the robotics platform of choice for:

- **Makers** building capable autonomous robots (not toys)
- **Citizen scientists** monitoring watersheds, marine environments
- **Students/educators** teaching robotics without PhD toolchains
- **Small organizations** needing real autonomy without enterprise budgets
- **Hobbyists** who want more than Arduino, less complexity than ROS 2
- **Developers** who value modern tooling (AI assist, simple deployment)
- **Anyone** building prosumer robotics on Linux platforms

## Why Gorai

### Cloud-Native Advantages

Gorai applies **proven distributed systems patterns** that ROS 2 lacks:

- **Message Broker**: NATS server (vs. ROS 2 direct DDS peer-to-peer)
- **Service Mesh**: NATS queue groups for automatic load balancing (vs. ROS 2 custom discovery)
- **Event Sourcing**: JetStream built-in replay (vs. ROS 2 rosbag manual recording)
- **Config Management**: NATS KV store with hot reload (vs. ROS 2 per-node parameters)
- **Observability**: Prometheus /metrics (vs. ROS 2 custom diagnostics)
- **Security**: NATS auth, TLS, JWT (vs. ROS 2 complex DDS Security)

**ROS 2's DDS is pre-cloud** (designed for LANs in 2004). Gorai uses infrastructure powering Coinbase, Mastercard, Siemens.

### Key Differentiators

- **Learns from cloud + robotics middleware**: Three generations of robotics frameworks + modern distributed systems
- **RDL (Robot Definition Language)**: Defines software interfaces, not physical attributes
- **AI co-design**: Built with AI assistance; optimized for AI-assisted development
- **Auto-generated plumbing**: RDL + AI generate boilerplate
- **Edge AI from day one**: TPU/NPU acceleration, ONNX Runtime, TensorFlow Lite
- **Linux-centric**: Raspberry Pi 5 reference platform; TinyGo for microcontrollers via serial bridge
- **Battle-tested infrastructure**: NATS.io and Prometheus proven at scale
- **Commercially-friendly**: Apache 2.0 open source
- **Democratize prosumer robotics**: Real autonomy accessible to makers, not just enterprises
