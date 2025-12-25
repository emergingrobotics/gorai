# Archived Specifications

This directory contains specifications from previous Gorai architecture versions that are no longer active.

## Archived Documents

### deployment-k3s-v3.md
**Archived:** 2024-12-25
**Reason:** Replaced by Podman-everywhere architecture

The K3s-everywhere architecture was replaced by Podman pods + systemd due to:
- K3s kernel incompatibility with NVIDIA Jetson (JetPack 6.x lacks eBPF features)
- High memory overhead (~1.8 GB vs ~150 MB for Podman)
- SSD requirement (K3s control plane database needs high IOPS)
- Complexity overkill for single-robot deployments

The new Podman-everywhere architecture provides:
- Universal hardware compatibility (including Jetson)
- Lower resource overhead
- SD card acceptable for deployed robots
- Same container isolation benefits
- Fleet coordination via NATS (not Kubernetes API)

### deployment-tiered-v2.md
**Archived:** 2024-12-25
**Reason:** Replaced by K3s-everywhere architecture (now also archived)

Previously, Gorai supported a tiered deployment model:
- Tier 1: Native binaries + systemd
- Tier 2: Podman pods + systemd
- Tier 3: K3s

### systemd-container-orchestration-v2.md
**Archived:** 2024-12-25
**Reason:** Replaced by K3s-everywhere architecture (now also archived)

Previously documented how to use Podman containers managed by systemd. This approach has been revived and expanded in the current Podman-everywhere architecture.

## Current Architecture

See:
- [../deployment-podman.md](../deployment-podman.md) - Current deployment specification
- [../hardware-requirements.md](../hardware-requirements.md) - Hardware requirements
- [../robot-definition-language.md](../robot-definition-language.md) - RDL v3 specification
