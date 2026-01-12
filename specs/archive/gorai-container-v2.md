# Gorai Container Specification

**Date**: 2024-12-24
**Status**: Active
**Version**: 2.5

## Overview

The `gorai` container provides the CLI tool for **development builds**. Production robots use tiered deployment based on complexity:

- **Tier 1**: Simple robots — systemd with native binaries or few containers
- **Tier 2**: Complex robots — K3s single-node (preferred) or Podman pods
- **Tier 3**: Fleet management — K3s multi-node clusters

See [deployment.md](deployment.md) for Tier 1, [systemd-container-orchestration.md](systemd-container-orchestration.md) for Podman pods alternative, and K3s documentation for Tier 2/3.

## Deployment Models

Gorai uses **distributed systems thinking** where components and services communicate via NATS. Deployment tier matches robot complexity:

```
Tier 1: Simple Robots (systemd)
┌─────────────────────────────────────────────────────────────────┐
│  Host System (Raspberry Pi, etc.)                               │
│                                                                  │
│  /opt/{robot}/                                                  │
│  ├── robot.json           (Configuration file)                  │
│  └── robot binary         (Native Go binary)                    │
│                                                                  │
│  systemd                                                        │
│  ├── nats.service         (NATS message broker)                 │
│  └── {robot}.service      (Robot process)                       │
│                                                                  │
│  Optional: Some services in containers                          │
│  └── podman run ...       (Managed by systemd)                  │
└─────────────────────────────────────────────────────────────────┘

Tier 2: Complex Robots (K3s single-node - preferred)
┌─────────────────────────────────────────────────────────────────┐
│  Host System (Raspberry Pi 4+, Jetson, RK3588)                  │
│                                                                  │
│  K3s (single-node Kubernetes)                                   │
│  ├── nats-server pod           (Message broker)                 │
│  ├── robot-core pod            (Go binary)                      │
│  ├── vision-yolo pod           (Python + PyTorch)               │
│  └── slam-cartographer pod     (C++ + libraries)                │
│                                                                  │
│  Features: health checks, rolling updates, resource limits      │
└─────────────────────────────────────────────────────────────────┘

Tier 2 Alternative: Complex Robots (Podman pods)
┌─────────────────────────────────────────────────────────────────┐
│  Host System                                                     │
│                                                                  │
│  systemd manages multiple containers                            │
│  ├── {robot}-nats.service       (NATS container)                │
│  ├── {robot}-core.service       (Go core container)             │
│  ├── {robot}-vision.service     (Python vision container)       │
│  └── {robot}-slam.service       (C++ SLAM container)            │
│                                                                  │
│  All communicate via NATS network                               │
└─────────────────────────────────────────────────────────────────┘

Tier 3: Fleet Management (K3s multi-node)
┌─────────────────────────────────────────────────────────────────┐
│  K3s Cluster (multi-node)                                       │
│                                                                  │
│  ├── Edge nodes (robots in field)                               │
│  │   └── Pods with robot services                               │
│  └── Cloud nodes (processing)                                   │
│      └── Heavy ML inference, data warehouse                     │
│                                                                  │
│  NATS spanning edge and cloud                                   │
└─────────────────────────────────────────────────────────────────┘
```

### Running Robots (v2)

```bash
# Run directly (foreground, development)
gorai run --config robot.json

# Deploy as systemd service (production)
gorai start --config robot.json --enable
gorai status --config robot.json
gorai logs --config robot.json -f
gorai stop --config robot.json
```

## Gorai CLI Container

The `gorai` container is used for **development builds only**, not for running robots.

### Use Cases

1. **Cross-compilation** - Build ARM binaries on x86 development machines
2. **CI/CD pipelines** - Reproducible build environment
3. **Development** - When Go isn't installed on the host

### Contents

| Component | Location | Description |
|-----------|----------|-------------|
| `gorai` CLI | `/usr/local/bin/gorai` | Robot management CLI |
| Go toolchain | System path | For building robot binaries |

### NOT Included

- **Robot runtime** - Robots run natively on hosts
- **NATS server** - Install natively: `sudo apt install nats-server`
- **Hardware drivers** - Compiled into robot binary

## Usage Patterns

### Pattern 1: Native CLI (Recommended)

Install the gorai CLI directly on the host:

```bash
# Install CLI
go install github.com/gorai/gorai/cmd/gorai@latest

# Run robot
gorai run --config robot.json

# Or deploy as service
gorai start --config robot.json --enable
```

### Pattern 2: Container for Builds Only

Use the gorai container for cross-compilation:

```bash
# Build robot binary for ARM64
podman run --rm -it \
  -v $(pwd):/workspace:rw \
  -w /workspace \
  ghcr.io/gorai/gorai:latest \
  build --target linux-arm64 --config robot.json
```

### Pattern 3: Development Environment

Use the container for development when Go isn't installed:

```bash
# Generate code from RDL
podman run --rm -it \
  -v $(pwd):/workspace:rw \
  -w /workspace \
  ghcr.io/gorai/gorai:latest \
  generate --config robot.json
```

## Build Specification

### Base Image

| Stage | Image | Purpose |
|-------|-------|---------|
| builder | `golang:1.22-alpine` | Compile Go CLI |
| runtime | `alpine:3.19` | Minimal CLI container |

### Containerfile

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o gorai ./cmd/gorai

FROM alpine:3.19
RUN apk add --no-cache ca-certificates git
COPY --from=builder /build/gorai /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/gorai"]
```

### Build Commands

```bash
# Build gorai CLI container
make container

# Or directly with podman
podman build -t ghcr.io/gorai/gorai:latest -f Containerfile.gorai .
```

## CLI Commands (v2)

| Command | Action | Description |
|---------|--------|-------------|
| `gorai run` | Run robot | Run robot directly in foreground |
| `gorai start` | Deploy service | Generate + install + start systemd service |
| `gorai stop` | Stop service | Stop systemd service |
| `gorai status` | Show status | Display service status |
| `gorai logs` | Stream logs | View journalctl logs |
| `gorai migrate` | Migrate config | Convert v1 config to v2 format |
| `gorai build` | Build binary | Build robot binary for target platform |
| `gorai validate` | Validate config | Check RDL configuration |

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GORAI_ROBOT_NAME` | Robot identifier | From RDL config |
| `NATS_URL` | NATS server URL | `nats://localhost:4222` |
| `LOG_LEVEL` | Logging verbosity | `info` |

## Registry

| Registry | Image | Description |
|----------|-------|-------------|
| GitHub | `ghcr.io/gorai/gorai:latest` | Official CLI releases |
| Local | `localhost/gorai:latest` | Development builds |

## Version Tags

| Tag | Description |
|-----|-------------|
| `latest` | Most recent stable release |
| `v2.0.0` | Specific version |
| `main` | Latest from main branch |

## Migration from v1

If you have a v1 configuration with containers:

```bash
# Migrate configuration
gorai migrate --config old-robot.json --output robot.json

# Install NATS natively
sudo apt install nats-server
sudo systemctl enable nats-server

# Start robot as native service
gorai start --config robot.json --enable
```

## External Services

For services that must run separately (e.g., ML inference with Hailo), use the `external` configuration in RDL:

```json
{
  "services": [
    {
      "name": "object_detector",
      "type": "object_detection",
      "external": {
        "enabled": true,
        "command": "/opt/gorai/services/hailo-detector",
        "managed": true,
        "restart": "always"
      }
    }
  ]
}
```

The main robot process will spawn and manage external services as child processes.

## Example Robot Project Structure (v2)

```
my-robot/
├── robot.json                    # RDL v2 configuration
├── cmd/
│   └── robot/
│       └── main.go               # Robot application (optional custom)
├── internal/
│   └── ...                       # Application code
├── models/                       # ML models (if using inference)
│   └── yolox.hef
└── deploy/
    └── my-robot.service          # Optional: custom service file
```

## Workflow (v2)

```bash
# 1. Create robot configuration
vim robot.json

# 2. Validate configuration
gorai validate robot.json

# 3. Run in development
gorai run --config robot.json

# 4. Deploy to production
gorai start --config robot.json --enable

# 5. Monitor
gorai logs --config robot.json -f
gorai status --config robot.json

# 6. Update
gorai stop --config robot.json
# ... update robot.json or rebuild binary ...
gorai start --config robot.json
```

## Security Considerations

1. **Hardware groups**: Ensure user is in required groups:
   ```bash
   sudo usermod -aG gpio,i2c,spi,video,dialout,plugdev $USER
   ```

2. **User lingering**: Enable for services to run without login:
   ```bash
   loginctl enable-linger $USER
   ```

3. **System mode**: For system-wide deployment (requires root):
   ```bash
   sudo gorai start --config robot.json --system --enable
   ```
