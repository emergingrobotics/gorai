# Quadlet Container Orchestration Specification

## Overview

Gorai uses **Quadlet**, systemd's native container management, for container orchestration. Quadlet transforms declarative configuration files into systemd service units, providing production-grade container management without additional abstraction layers.

### Why Quadlet over podman-compose

| Feature | Quadlet | podman-compose |
|---------|---------|----------------|
| Integration | Native systemd | Translation layer |
| Production Use | Designed for servers/edge | Development focused |
| Dependencies | True systemd ordering | Compose-style depends_on |
| Logging | Native journald | Separate logging |
| Auto-updates | Built-in AutoUpdate | Requires watchtower |
| Debugging | systemctl, journalctl | docker-compose logs |

## Architecture

### File Flow

```
robot.json (RDL)
    │
    ▼
gorai build/start
    │
    ▼
.gorai/*.container    ─────┐
.gorai/*.network           ├─► systemd
.gorai/*.volume       ─────┘
    │
    ▼
~/.config/containers/systemd/  (user mode)
/etc/containers/systemd/       (system mode)
    │
    ▼
systemctl daemon-reload
    │
    ▼
systemctl start {robot}-*.service
```

### Generated Files

For a robot named `hello-camera` with containers `nats` and `gorai-core`:

```
.gorai/
├── hello-camera-network.network     # Default bridge network
├── hello-camera-nats.container      # NATS container unit
└── hello-camera-gorai-core.container # Core container unit
```

## Quadlet File Formats

### Container Units (.container)

```ini
[Unit]
Description=Gorai container hello-camera-nats
Requires=hello-camera-network-network.service
After=hello-camera-network-network.service

[Container]
ContainerName=hello-camera-nats
Image=nats:2.10-alpine
Network=hello-camera-network.network
PublishPort=4222:4222
PublishPort=8222:8222
Environment=GORAI_ROBOT_NAME=hello-camera
HealthCmd=wget -q --spider http://localhost:8222/healthz
AutoUpdate=registry

[Service]
Restart=always
RestartSec=10
TimeoutStartSec=300

[Install]
WantedBy=default.target
```

### Network Units (.network)

```ini
[Unit]
Description=Gorai default network for hello-camera

[Network]
NetworkName=hello-camera-network
Driver=bridge

[Install]
WantedBy=default.target
```

### Volume Units (.volume)

```ini
[Unit]
Description=Gorai volume hello-camera-data

[Volume]
VolumeName=hello-camera-data

[Install]
WantedBy=default.target
```

## RDL to Quadlet Mapping

### Container Configuration

| RDL Field | Quadlet Section | Quadlet Key |
|-----------|-----------------|-------------|
| `image` | [Container] | `Image=` |
| `command` | [Container] | `Exec=` |
| `entrypoint` | [Container] | `Entrypoint=` |
| `environment` | [Container] | `Environment=` (repeated) |
| `env_file` | [Container] | `EnvironmentFile=` |
| `volumes` | [Container] | `Volume=` (repeated) |
| `devices` | [Container] | `AddDevice=` (repeated) |
| `ports` | [Container] | `PublishPort=` (repeated) |
| `networks` | [Container] | `Network=` |
| `network_mode: host` | [Container] | `Network=host` |
| `privileged` | [Container] | `SecurityLabelDisable=true` |
| `security_opt` | [Container] | `SecurityOpt=` |
| `cap_add` | [Container] | `AddCapability=` |
| `cap_drop` | [Container] | `DropCapability=` |
| `group_add` | [Container] | `GroupAdd=` |
| `resources.limits.memory` | [Container] | `PodmanArgs=--memory=` |
| `resources.limits.cpus` | [Container] | `PodmanArgs=--cpus=` |
| `restart` | [Service] | `Restart=` |
| `stop_grace_period` | [Service] | `TimeoutStopSec=` |
| `healthcheck.test` | [Container] | `HealthCmd=` |
| `depends_on` | [Unit] | `Requires=` + `After=` |

### Quadlet-Specific RDL Fields

```json
{
  "containers": {
    "mycontainer": {
      "auto_update": "registry",    // "registry" | "local" | "none"
      "notify": "healthy",          // "true" | "healthy" | ""
      "timeout_start": "300"        // Startup timeout in seconds
    }
  }
}
```

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

Generates Quadlet files and optionally builds container images.

```bash
gorai build --config robot.json           # Generate Quadlet files
gorai build --config robot.json --install # Also install to systemd
gorai build --config robot.json --no-cache # Rebuild images
```

### gorai start

Generates, installs Quadlet files, and starts services.

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
gorai stop --config robot.json --uninstall  # Remove Quadlet files
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

Quadlet files installed to `~/.config/containers/systemd/`:

```bash
# Enable user lingering for boot startup without login
loginctl enable-linger $USER

# Start services
gorai start --config robot.json --enable
```

### System Mode (Root)

For system-wide deployment (requires root):

```bash
# Files go to /etc/containers/systemd/
sudo gorai start --config robot.json --enable
```

## Auto-Updates

Quadlet supports automatic container image updates:

```ini
[Container]
AutoUpdate=registry
```

To trigger updates:

```bash
# Check for updates (dry-run)
podman auto-update --dry-run

# Apply updates
podman auto-update
```

Enable the systemd timer for automatic checks:

```bash
systemctl --user enable podman-auto-update.timer
systemctl --user start podman-auto-update.timer
```

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
[Container]
AddDevice=/dev/video0:/dev/video0
AddDevice=/dev/i2c-1:/dev/i2c-1
GroupAdd=video
GroupAdd=i2c
SecurityLabelDisable=true
```

## Health Checks

RDL health checks translate to Quadlet health commands:

```json
{
  "healthcheck": {
    "test": ["CMD", "wget", "-q", "--spider", "http://localhost:8222/healthz"],
    "interval": "10s",
    "timeout": "5s",
    "retries": 3,
    "start_period": "5s"
  }
}
```

Generates:

```ini
[Container]
HealthCmd=wget -q --spider http://localhost:8222/healthz
HealthInterval=10s
HealthTimeout=5s
HealthRetries=3
HealthStartPeriod=5s
```

## Environment Variables

Standard Gorai environment variables are automatically injected:

| Variable | Description |
|----------|-------------|
| `GORAI_ROBOT_NAME` | Robot name from RDL |
| `NATS_URL` | NATS connection URL |
| `GORAI_COMPONENTS` | Comma-separated component names |
| `GORAI_SERVICES` | Comma-separated service names |

## Volume SELinux Labels

Volumes automatically get `:Z` suffix for SELinux relabeling:

```ini
[Container]
Volume=./config:/etc/gorai:ro,Z
Volume=data.volume:/data:Z
```

## Debugging

### Check Generated Services

```bash
# List generated service files (dry-run)
/usr/libexec/podman/quadlet --user --dryrun

# Validate service files
systemd-analyze --user verify hello-camera-nats.service
```

### Service Management

```bash
# Check service status
systemctl --user status hello-camera-nats.service

# View logs
journalctl --user -u hello-camera-nats.service -f

# Restart service
systemctl --user restart hello-camera-nats.service
```

## Migration from podman-compose

If migrating from an existing podman-compose setup:

1. Stop existing containers: `podman-compose down`
2. Generate Quadlet files: `gorai build --config robot.json`
3. Install and start: `gorai start --config robot.json`
4. Remove compose file: `rm .gorai/*-compose.json`

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
      "image": "nats:2.10-alpine",
      "ports": ["4222:4222", "8222:8222"],
      "restart": "always",
      "healthcheck": {
        "test": ["CMD", "wget", "-q", "--spider", "http://localhost:8222/healthz"],
        "interval": "10s",
        "timeout": "5s",
        "retries": 3
      }
    },
    "gorai-core": {
      "image": "gorai/core:latest",
      "depends_on": {
        "nats": {"condition": "service_healthy"}
      },
      "devices": ["/dev/video0"],
      "group_add": ["video"],
      "environment": {
        "LOG_LEVEL": "info"
      },
      "components": ["camera"],
      "services": ["navigation"]
    }
  },
  "components": [
    {
      "name": "camera",
      "type": "camera",
      "model": "v4l2",
      "container": "gorai-core"
    }
  ],
  "services": [
    {
      "name": "navigation",
      "type": "navigation",
      "model": "default",
      "container": "gorai-core"
    }
  ]
}
```

This generates three Quadlet files:
- `rover1-network.network`
- `rover1-nats.container`
- `rover1-gorai-core.container`

And provides full systemd lifecycle management via:
- `gorai start/stop/status/logs`
- Or directly: `systemctl --user start/stop/status rover1-*.service`
