# Archived Specifications

This directory contains specifications from previous Gorai architecture versions that are no longer active.

## Current Architecture

**Gorai uses K3s-everywhere architecture.** See:
- [../k3s-installation.md](../k3s-installation.md) — K3s installation by platform
- [../hardware-requirements.md](../hardware-requirements.md) — Hardware requirements
- [../robot-definition-language.md](../robot-definition-language.md) — RDL v3 specification

## Archived Documents

### hardware-v1.md
**Archived:** 2025-01-12
**Reason:** Superseded by `hardware-requirements.md`

Old hardware list that incorrectly included unsupported platforms (Pi 3, Pi Zero).

### gorai-container-v2.md
**Archived:** 2025-01-12
**Reason:** Described obsolete tiered deployment model

Previously documented a tiered deployment approach (Tier 1: systemd, Tier 2: K3s single-node, Tier 3: K3s multi-node). Now replaced by K3s-everywhere where all robots use K3s.

### deployment-k3s-v3.md
**Archived:** 2024-12-25
**Reason:** Was temporarily archived during Podman evaluation

This was archived when evaluating Podman as an alternative. K3s-everywhere has been restored as the deployment model.

### deployment-tiered-v2.md
**Archived:** 2024-12-25
**Reason:** Replaced by K3s-everywhere architecture

Previously, Gorai supported a tiered deployment model:
- Tier 1: Native binaries + systemd
- Tier 2: Podman pods + systemd
- Tier 3: K3s

This has been simplified to K3s-everywhere.

### systemd-container-orchestration-v2.md
**Archived:** 2024-12-25
**Reason:** Replaced by K3s-everywhere architecture

Previously documented how to use Podman containers managed by systemd. No longer the recommended approach.

### deployment-podman-v1.md
**Archived:** 2025-01-12
**Reason:** Podman-everywhere was evaluated but K3s-everywhere was chosen

Podman-everywhere was considered as an alternative to K3s due to:
- Lower memory overhead
- SD card compatibility
- Jetson kernel compatibility concerns

However, K3s-everywhere was chosen because:
- Consistent deployment model from 1 to 100+ robots
- Production-grade orchestration (health checks, rolling updates)
- Every robot is fleet-ready from day one
- Jetson Orin Nano Super is fully compatible with K3s
- Modern hardware (4GB+ RAM, SSD) makes K3s overhead acceptable
