# Gorai Container Specification

**Date**: 2025-12-14
**Status**: Implemented
**Version**: 2.0

## Overview

The `gorai` container provides the CLI tool for building robot container images. Robots are deployed as sets of containers managed by systemd via Quadlet—there are no binaries to install on the robot itself.

## Deployment Model

A Gorai robot deployment consists of:

1. **Quadlet unit files** - `.container`, `.network`, `.volume` files in `~/.config/containers/systemd/`
2. **Container images** - OCI images pulled from registries or built locally
3. **systemd services** - Generated automatically from Quadlet files

```
Robot Deployment
┌─────────────────────────────────────────────────────────────────┐
│  Host System (Raspberry Pi, etc.)                               │
│                                                                  │
│  ~/.config/containers/systemd/                                  │
│  ├── robot-network.network     (Quadlet unit files)            │
│  ├── robot-nats.container                                       │
│  └── robot-core.container                                       │
│                                                                  │
│  Container Images                                                │
│  ├── nats:2.10-alpine          (pulled from registry)          │
│  └── localhost/robot-core      (built locally)                  │
│                                                                  │
│  systemd                                                         │
│  ├── robot-nats.service        (auto-generated from .container)│
│  └── robot-core.service                                         │
└─────────────────────────────────────────────────────────────────┘
```

## Gorai CLI Container

The `gorai` CLI container is used for **building** robot images, not running robots. Robots run via Quadlet/systemd on the host.

### Contents

| Component | Location | Description |
|-----------|----------|-------------|
| `gorai` CLI | `/usr/local/bin/gorai` | Robot management CLI |
| `podman` client | System path | For building images |

### NOT Included

- **Robot runtime** - Each robot container has its own application
- **Python** - No longer needed (podman-compose removed)
- **Component drivers** - Go in robot-specific containers
- **NATS server** - Runs in its own container

## Usage Patterns

### Pattern 1: Build Images (in container)

Use the gorai container to build robot images when Go isn't installed on the host:

```bash
podman run --rm -it \
  --security-opt label=disable \
  -v /run/podman/podman.sock:/run/podman/podman.sock:rw \
  -v $(pwd):/workspace:rw \
  -w /workspace \
  ghcr.io/gorai/gorai:latest \
  build --config robot.json
```

### Pattern 2: Direct CLI (Recommended)

Install the gorai CLI directly on the host for full Quadlet integration:

```bash
# Install CLI
go install github.com/gorai/gorai/cmd/gorai@latest

# Build and deploy
gorai build --config robot.json
gorai start --config robot.json
```

### Pattern 3: Robot Container Template

Each robot component runs in its own container. Example structure:

```dockerfile
# Containerfile.core - Robot core container
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /robot ./cmd/robot

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /robot /usr/local/bin/robot
ENTRYPOINT ["/usr/local/bin/robot"]
```

## Build Specification

### Base Image

| Stage | Image | Purpose |
|-------|-------|---------|
| builder | `golang:1.22-alpine` | Compile Go CLI |
| runtime | `alpine:3.19` | Minimal CLI container |

### Build Process

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o gorai ./cmd/gorai

FROM alpine:3.19
RUN apk add --no-cache podman
COPY --from=builder /build/gorai /usr/local/bin/
```

### Build Commands

```bash
# Build gorai CLI container
make container

# Or directly with podman
podman build -t ghcr.io/gorai/gorai:latest -f Containerfile.gorai .
```

## systemd Integration

The gorai CLI generates systemd service files and manages them via systemctl:

### Generated Files

```
.gorai/                                    # Local generated files
├── robot-nats.service
├── robot-gorai-core.service
└── robot-gorai-hailo.service

~/.config/systemd/user/                    # Installed for systemd
├── robot-nats.service
├── robot-gorai-core.service
└── robot-gorai-hailo.service
```

### CLI Commands

| Command | Action | systemd Equivalent |
|---------|--------|-------------------|
| `gorai build` | Generate service files | N/A |
| `gorai start` | Install + start services | `systemctl --user start` |
| `gorai stop` | Stop services | `systemctl --user stop` |
| `gorai status` | Show service status | `systemctl --user status` |
| `gorai logs` | Stream logs | `journalctl --user -u` |

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GORAI_ROBOT_NAME` | Robot identifier | From RDL config |
| `NATS_URL` | NATS server URL | `nats://nats:4222` |
| `LOG_LEVEL` | Logging verbosity | `info` |
| `CONTAINER_HOST` | Podman socket path | `unix:///run/podman/podman.sock` |

## Registry

| Registry | Image | Description |
|----------|-------|-------------|
| GitHub | `ghcr.io/gorai/gorai:latest` | Official CLI releases |
| Local | `localhost/gorai:latest` | Development builds |

## Version Tags

| Tag | Description |
|-----|-------------|
| `latest` | Most recent stable release |
| `v1.0.0` | Specific version |
| `main` | Latest from main branch |

## Related Containers

These are **examples** of robot-specific containers, not provided by Gorai:

| Container | Purpose | Contents |
|-----------|---------|----------|
| `robot-core` | Main robot application | Go application + drivers |
| `robot-hailo` | ML inference | Python + HailoRT |
| `nats` | Messaging | NATS server |
| `prometheus` | Monitoring | Prometheus server |

## Security Considerations

1. **Rootless by default**: Quadlet supports rootless containers. Use user mode (`~/.config/containers/systemd/`).

2. **User lingering**: Enable for services to run without login:
   ```bash
   loginctl enable-linger $USER
   ```

3. **Device access**: Use `group_add` in RDL for hardware access:
   ```json
   "group_add": ["video", "i2c"]
   ```

4. **SELinux**: Quadlet handles SELinux labels automatically via `:Z` volume suffix.

## Example Robot Project Structure

```
my-robot/
├── robot.json                    # RDL configuration
├── Containerfile.core            # Core container build
├── Containerfile.hailo           # ML inference container
├── cmd/
│   └── robot/
│       └── main.go               # Robot application
├── internal/
│   └── ...                       # Application code
└── .gorai/                       # Generated Quadlet files
    ├── my-robot-network.network
    ├── my-robot-nats.container
    └── my-robot-core.container
```

## Workflow

```bash
# 1. Create robot configuration
vim robot.json

# 2. Write robot application
# ... code in cmd/robot/main.go

# 3. Build images and generate Quadlet files
gorai build --config robot.json

# 4. Deploy
gorai start --config robot.json --enable

# 5. Monitor
gorai logs --config robot.json -f
gorai status --config robot.json

# 6. Update (auto-update or manual)
podman auto-update
# or
gorai stop --config robot.json
gorai build --config robot.json
gorai start --config robot.json
```
