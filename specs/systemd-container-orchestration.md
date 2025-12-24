# systemd Container Orchestration Specification

**Version:** 2.0
**Status:** Active (Tier 2 Deployment)
**Last Updated:** 2024-12-24

## Overview

This specification describes **containerized deployment using Podman pods** managed by systemd. This is an alternative approach for complex robots that prefer explicit container management over K3s orchestration.

**Recommended deployment tiers:**
- **Tier 1**: Simple robots — systemd with native binaries (see [deployment.md](deployment.md))
- **Tier 2**: Complex robots — **K3s single-node** (preferred for orchestration features)
- **Tier 2 Alternative**: Complex robots — Podman pods + systemd (this document)
- **Tier 3**: Fleet management — K3s multi-node clusters

**When to use Podman pods instead of K3s:**
- You want explicit control over container commands
- You prefer systemd service files over Kubernetes manifests
- You don't need K3s orchestration features (health checks, rolling updates, etc.)
- Your team is more familiar with Podman/Docker than Kubernetes

**When to use K3s (Tier 2 preferred):**
- You need health monitoring, rolling updates, resource limits
- You want orchestration features on a single robot
- You might scale to multi-node later
- K3s is designed for edge/IoT (70MB, 512MB RAM, works on Raspberry Pi)

This document describes **systemd service units** with `podman run` for container orchestration. This provides production-grade container management using native systemd capabilities.

### Why Podman for Tier 2

| Feature | Podman | Docker | Why for Robotics |
|---------|--------|--------|------------------|
| **Daemonless** | Yes | No | No root daemon, better security |
| **Rootless** | Native | Limited | Run as regular user |
| **systemd integration** | Excellent | Manual | Native service management |
| **Compatibility** | OCI standard | Proprietary | Works with Docker images |
| **Resource usage** | Lower | Higher | Better for edge devices |

### Why Traditional systemd Units

| Feature | Traditional systemd | Quadlet | Podman Compose |
|---------|---------------------|---------|----------------|
| Compatibility | Any Podman + systemd | Podman 4.4+ | Separate tool |
| Transparency | Explicit podman commands | Generated | YAML abstraction |
| Debugging | See exact commands | Abstraction layer | Docker Compose syntax |
| Boot integration | Native | Native | Manual setup |
| Dependency management | systemd units | systemd units | restart policies |

**Gorai uses traditional systemd units** for maximum transparency and control. You see exactly what runs.

## Distributed Systems Architecture

### File Flow

```
robot.json (RDL)
    │
    ▼
gorai build/start
    │
    ▼
.gorai/*.service         ─────► systemd
    │
    ▼
~/.config/systemd/user/  (user mode)
/etc/systemd/system/     (system mode)
    │
    ▼
systemctl daemon-reload
    │
    ▼
systemctl start {robot}-*.service
```

### Generated Files

For a robot named `hello-camera` with containers `nats`, `gorai-core`, and `gorai-hailo`:

```
.gorai/
├── hello-camera-nats.service
├── hello-camera-gorai-core.service
└── hello-camera-gorai-hailo.service
```

### Distributed Systems Thinking

**Components and services are logical concepts**, not deployment constraints. A "service" in Gorai is any process that communicates via NATS. In Tier 2:

- **NATS** runs in a container (message broker for all services)
- **Go core** runs in a container (orchestration, simple sensors)
- **Python vision** runs in a separate container (OpenCV, image preprocessing)
- **C++ SLAM** runs in another container (mapping, localization)

Each container:
- Is a separate process with its own lifecycle
- Communicates via NATS (not direct function calls)
- Can fail and restart independently
- Can be on the same machine or different machines (NATS handles routing)

This is **distributed systems** even when containers are on one robot. The architecture scales from single-board computers to multi-robot fleets without code changes.

## Service Unit Format

### Example Service Unit

```ini
[Unit]
Description=Gorai container hello-camera-nats
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
Restart=always
RestartSec=10
TimeoutStartSec=300

# Cleanup any existing container
ExecStartPre=-/usr/bin/podman stop -t 10 hello-camera-nats
ExecStartPre=-/usr/bin/podman rm -f hello-camera-nats
ExecStartPre=-/usr/bin/podman network create hello-camera-network

# Start container
ExecStart=/usr/bin/podman run --rm \
    --name hello-camera-nats \
    --network hello-camera-network \
    --network-alias nats \
    -p 4222:4222 \
    -p 8222:8222 \
    -e GORAI_ROBOT_NAME=hello-camera \
    -e NATS_URL=nats://nats:4222 \
    docker.io/nats:2.10-alpine

ExecStop=/usr/bin/podman stop -t 10 hello-camera-nats

[Install]
WantedBy=default.target
```

### Service with Dependencies

```ini
[Unit]
Description=Gorai container hello-camera-gorai-core
Requires=hello-camera-nats.service
After=hello-camera-nats.service

[Service]
Type=simple
Restart=always
RestartSec=10
TimeoutStartSec=300

# Cleanup any existing container
ExecStartPre=-/usr/bin/podman stop -t 10 hello-camera-gorai-core
ExecStartPre=-/usr/bin/podman rm -f hello-camera-gorai-core
ExecStartPre=-/usr/bin/podman network create hello-camera-network

# Start container
ExecStart=/usr/bin/podman run --rm \
    --name hello-camera-gorai-core \
    --network hello-camera-network \
    --network-alias gorai-core \
    -p 8080:8080 \
    -e GORAI_ROBOT_NAME=hello-camera \
    -e NATS_URL=nats://nats:4222 \
    -e LOG_LEVEL=info \
    -v /path/to/hello-camera.json:/etc/gorai/robot.json:ro \
    --device /dev/video0:/dev/video0 \
    --group-add video \
    localhost/hello-camera-core:latest

ExecStop=/usr/bin/podman stop -t 10 hello-camera-gorai-core

[Install]
WantedBy=default.target
```

## RDL to systemd Mapping

### Container Configuration

| RDL Field | systemd/podman |
|-----------|----------------|
| `image` | `podman run ... {image}` |
| `environment` | `-e KEY=VALUE` |
| `volumes` | `-v host:container:opts` |
| `devices` | `--device /dev/xxx` |
| `ports` | `-p host:container` |
| `group_add` | `--group-add group` |
| `security_opt` | `--security-opt label=disable` |
| `resources.limits.memory` | `--memory=1G` |
| `resources.limits.cpus` | `--cpus=2` |
| `restart` | `Restart=always/on-failure/no` |
| `depends_on` | `Requires=` + `After=` |

### Restart Policy Mapping

| RDL | systemd |
|-----|---------|
| `always` | `Restart=always` |
| `unless-stopped` | `Restart=always` |
| `on-failure` | `Restart=on-failure` |
| `no` | `Restart=no` |

### Dependency Mapping

RDL `depends_on` conditions map to systemd unit dependencies:

```json
"depends_on": {
  "nats": {
    "condition": "service_healthy"
  }
}
```

Generates:
```ini
[Unit]
Requires=hello-camera-nats.service
After=hello-camera-nats.service
```

## CLI Commands

### gorai build

Generates systemd service files and optionally builds container images.

```bash
gorai build --config robot.json           # Generate service files
gorai build --config robot.json --install # Also install to systemd
gorai build --config robot.json --no-cache # Rebuild images
```

### gorai start

Generates, installs service files, and starts services.

```bash
gorai start --config robot.json           # Start all containers
gorai start --config robot.json --build   # Build first, then start
gorai start --config robot.json --enable  # Enable auto-start at boot
```

### gorai stop

Stops container services.

```bash
gorai stop --config robot.json              # Stop all containers
gorai stop --config robot.json --disable    # Also disable auto-start
gorai stop --config robot.json --uninstall  # Remove service files
```

### gorai status

Shows service status using systemctl.

```bash
gorai status --config robot.json
```

### gorai logs

Shows logs using journalctl.

```bash
gorai logs --config robot.json -f           # Follow all logs
gorai logs --config robot.json -f nats      # Follow specific container
gorai logs --config robot.json --tail 100   # Last 100 lines
```

## Deployment Modes

### User Mode (Rootless) - Default

Service files installed to `~/.config/systemd/user/`:

```bash
# Enable user lingering for boot startup without login
loginctl enable-linger $USER

# Start services
gorai start --config robot.json --enable
```

### System Mode (Root)

For system-wide deployment (requires root):

```bash
# Files go to /etc/systemd/system/
sudo gorai start --config robot.json --enable
```

## Network Management

The service creates a podman network in `ExecStartPre`:

```ini
ExecStartPre=-/usr/bin/podman network create hello-camera-network
```

The `-` prefix means "don't fail if network already exists".

All containers join the same network and can reach each other by name using `--network-alias`:

```ini
--network hello-camera-network \
--network-alias nats \
```

This allows `nats://nats:4222` to work for service discovery.

## Device Access

For hardware devices (cameras, sensors, etc.):

```json
{
  "containers": {
    "gorai-core": {
      "devices": ["/dev/video0", "/dev/i2c-1"],
      "group_add": ["video", "i2c"],
      "security_opt": ["label=disable"]
    }
  }
}
```

Generates:

```ini
--device /dev/video0:/dev/video0 \
--device /dev/i2c-1:/dev/i2c-1 \
--group-add video \
--group-add i2c \
--security-opt label=disable \
```

## Environment Variables

Standard Gorai environment variables are automatically injected:

| Variable | Description |
|----------|-------------|
| `GORAI_ROBOT_NAME` | Robot name from RDL |
| `NATS_URL` | NATS connection URL |

## Resource Limits

```json
{
  "resources": {
    "limits": {
      "memory": "1G",
      "cpus": "2"
    }
  }
}
```

Generates:

```ini
--memory=1G \
--cpus=2 \
```

## Debugging

### Check Service Status

```bash
# Check service status
systemctl --user status hello-camera-nats.service

# View service file
systemctl --user cat hello-camera-nats.service

# View logs
journalctl --user -u hello-camera-nats.service -f

# Restart service
systemctl --user restart hello-camera-nats.service
```

### Manual Container Management

```bash
# List running containers
podman ps

# Exec into container
podman exec -it hello-camera-nats sh

# View container logs
podman logs -f hello-camera-nats
```

## Example: Complete Robot Configuration

```json
{
  "version": "1",
  "robot": {
    "name": "rover1",
    "description": "Example rover robot"
  },
  "nats": {
    "url": "nats://nats:4222",
    "container": "nats"
  },
  "containers": {
    "nats": {
      "image": "docker.io/nats:2.10-alpine",
      "ports": ["4222:4222", "8222:8222"],
      "restart": "always"
    },
    "gorai-core": {
      "image": "localhost/rover1-core:latest",
      "depends_on": {
        "nats": {"condition": "service_healthy"}
      },
      "devices": ["/dev/video0"],
      "group_add": ["video"],
      "environment": {
        "LOG_LEVEL": "info"
      },
      "volumes": [
        "./robot.json:/etc/gorai/robot.json:ro"
      ]
    }
  }
}
```

This generates two service files:
- `rover1-nats.service`
- `rover1-gorai-core.service`

And provides full systemd lifecycle management via:
- `gorai start/stop/status/logs`
- Or directly: `systemctl --user start/stop/status rover1-*.service`
