# Gorai

**Professional robotics for prosumers — without the PhD**

Gorai is a Go-based robotics framework designed for makers, citizen scientists, students, and small organizations who need real autonomy without ROS 2's complexity. Built on battle-tested cloud infrastructure (NATS, Prometheus), Gorai applies distributed systems patterns to robotics.

## Strategic Positioning

**Target Market**: Prosumers who find Arduino too limiting and ROS 2 too complex.

**Not competing with ROS 2** in enterprise/research. We're "ROS 2 for prosumers," not "ROS 2 killer."

See [docs/vision-analysis.md](docs/vision-analysis.md) for comprehensive strategic analysis.

## Core Attributes

- **Podman-everywhere architecture** — All robots run as Podman pods managed by systemd; RDL abstracts complexity
- **Containers without complexity** — ~150 MB overhead (vs ~1.8 GB for K3s); works on all platforms including Jetson
- **Edge AI focus** — From 6 TOPS (Orange Pi 5B) to 67 TOPS (Jetson Orin Super); choose your performance tier
- **Go core + pragmatic polyglot services** — Go for orchestration; Python/C++ via containers for vision/SLAM
- **Cloud-native patterns** — NATS (message broker), Prometheus (observability), Podman (containers)
- **RDL abstracts deployment** — Users write YAML configs; `gorai deploy` generates pod definitions + systemd units
- **AI-assisted development** — designed for modern AI-powered coding
- **Low barrier to entry** — productive in days, not months
- **Modular architecture** — unified Resource interface for all components/services
- **Linux-first** — Raspberry Pi 5 (8GB) primary; Jetson Orin Super for maximum AI
- **ROS 2 bridge planned** — Future phase for ecosystem compatibility
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
- **RDL (Robot Definition Language)**: YAML config translates to Podman pods + systemd units automatically
- **AI co-design**: Built with AI assistance; optimized for AI-assisted development
- **Auto-generated plumbing**: RDL generates container definitions, pod specs, systemd services
- **Edge AI from day one**: NVIDIA CUDA (67 TOPS), Hailo (26 TOPS), RK3588 NPU (6 TOPS), Coral TPU
- **Linux-centric**: Raspberry Pi 5 primary; Jetson Orin Super for performance; Orange Pi 5B for budget AI
- **Battle-tested infrastructure**: NATS.io, Prometheus, Podman proven at scale
- **Commercially-friendly**: Apache 2.0 open source
- **Democratize prosumer robotics**: Real autonomy accessible to makers, not just enterprises

## Hardware Requirements

| Platform | Role | AI Performance | Storage | Cost |
|----------|------|----------------|---------|------|
| **Raspberry Pi 5 (8GB)** | Primary | External (Hailo 13-26 TOPS) | NVMe SSD | ~$160 |
| **Jetson Orin Nano Super** | Performance | 67 TOPS (CUDA) | NVMe SSD | ~$335 |
| **Orange Pi 5B (8GB)** | Budget AI | 6 TOPS (built-in) | 64GB eMMC | ~$145 |
| **Raspberry Pi 4 (4GB)** | Minimum | External (Coral) | USB SSD or SD | ~$105 |

**Choose Jetson Orin Super for maximum AI** — 67 TOPS enables multi-model inference, VLMs, on-robot LLMs.
**Choose Pi 5 for best ecosystem** — largest community, most documentation, beginner-friendly.
**Choose Orange Pi 5B for budget AI** — built-in NPU at lowest cost.

**Not supported:** Pi 3, Pi Zero, Pi 4 (2GB)

See [specs/hardware-requirements.md](specs/hardware-requirements.md) for full requirements.
