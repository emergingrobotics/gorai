# Gorai Container Specification

**Date**: 2025-12-13
**Status**: Implemented
**Version**: 1.0

## Overview

The `gorai` container is the primary deployment artifact for Gorai. It contains all Go binaries needed to run and orchestrate robot systems, plus tooling required for container-based deployment.

## Contents

### Go Binaries

| Binary | Description | Location |
|--------|-------------|----------|
| `gorai` | CLI for robot management (start, stop, status, logs, build) | `/usr/local/bin/gorai` |
| `gorai-robot` | Standalone robot runtime for containerized deployment | `/usr/local/bin/gorai-robot` |

### Tooling

| Tool | Description | Purpose |
|------|-------------|---------|
| `podman-compose` | Container orchestration | Generates and runs compose files from RDL |

### NOT Included

The following are explicitly **not** in this container:

- **Component drivers** (camera, motor, etc.) - These go in robot-specific containers
- **Python ML services** - These go in separate containers (e.g., `gorai-hailo`)
- **Hardware SDKs** - Each hardware type has its own container
- **NATS server** - Runs in its own container

## Container Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ gorai container (ghcr.io/gorai/gorai:latest)                │
│                                                              │
│   /usr/local/bin/                                           │
│   ├── gorai           # CLI tool                            │
│   └── gorai-robot     # Robot runtime                       │
│                                                              │
│   /usr/local/lib/python3.11/                                │
│   └── site-packages/                                        │
│       └── podman_compose/  # Orchestration                  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Usage Patterns

### Pattern 1: Orchestrator Mode (Primary)

The gorai container runs on the host, orchestrating other containers:

```bash
# Build gorai container (once)
make container

# Start robot system
podman run --rm -it \
  --security-opt label=disable \
  -v /run/podman/podman.sock:/run/podman/podman.sock:rw \
  -v $(pwd):/workspace:rw \
  -w /workspace \
  ghcr.io/gorai/gorai:latest \
  start --config robot.json --build --detach
```

This spawns:
- `nats` container (messaging)
- `gorai-core` container (robot components)
- `gorai-hailo` container (ML inference)
- etc.

### Pattern 2: Runtime Mode

The `gorai-robot` binary can be extracted and used in other containers:

```dockerfile
# In a robot-specific Containerfile
FROM ghcr.io/gorai/gorai:latest AS gorai
FROM custom-base:latest

COPY --from=gorai /usr/local/bin/gorai-robot /usr/local/bin/
ENTRYPOINT ["gorai-robot"]
```

## Build Specification

### Base Images

| Stage | Image | Purpose |
|-------|-------|---------|
| builder | `golang:1.22-alpine` | Compile Go binaries |
| runtime | `python:3.11-slim-bookworm` | Python for podman-compose |

### Build Process

```dockerfile
# Stage 1: Build all Go binaries
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o gorai ./cmd/gorai
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o gorai-robot ./cmd/gorai-robot

# Stage 2: Runtime with podman-compose
FROM python:3.11-slim-bookworm
RUN pip install --no-cache-dir podman-compose
COPY --from=builder /build/gorai /usr/local/bin/
COPY --from=builder /build/gorai-robot /usr/local/bin/
```

### Build Commands

```bash
# Top-level Makefile
make container           # Build ghcr.io/gorai/gorai:latest
make container-push      # Push to registry
make container-clean     # Remove local images
```

## Podman Socket Requirements

The container requires access to the host's podman socket to orchestrate sibling containers:

```bash
# User socket (rootless podman)
/run/user/$(id -u)/podman/podman.sock

# System socket (root podman)
/run/podman/podman.sock
```

Enable the user socket:
```bash
systemctl --user enable --now podman.socket
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GORAI_ROBOT_NAME` | Robot identifier | From RDL |
| `NATS_URL` | NATS server URL | `nats://nats:4222` |
| `LOG_LEVEL` | Logging verbosity | `info` |

## Labels

```dockerfile
LABEL org.opencontainers.image.title="Gorai"
LABEL org.opencontainers.image.description="Gorai robotics framework - CLI and runtime"
LABEL org.opencontainers.image.source="https://github.com/gorai/gorai"
LABEL org.opencontainers.image.version="1.0.0"
```

## Registry

| Registry | Image | Description |
|----------|-------|-------------|
| GitHub | `ghcr.io/gorai/gorai:latest` | Official releases |
| Local | `localhost/gorai:latest` | Development builds |

## Version Tags

| Tag | Description |
|-----|-------------|
| `latest` | Most recent stable release |
| `v1.0.0` | Specific version |
| `main` | Latest from main branch |
| `dev` | Development builds |

## Related Containers

| Container | Purpose | Contents |
|-----------|---------|----------|
| `gorai` (this) | CLI + orchestration | Go binaries + podman-compose |
| `gorai-core` | Robot runtime | gorai-robot + V4L2/GPIO drivers |
| `gorai-hailo` | ML inference | Python + Hailo runtime |
| `nats` | Messaging | NATS server |

## Security Considerations

1. **Podman socket access**: The container needs socket access to spawn siblings. Use `--security-opt label=disable` for SELinux compatibility.

2. **Non-root execution**: The gorai binaries can run as non-root. The container can be configured to run as UID 1000.

3. **Read-only root**: The container filesystem can be mounted read-only; only `/workspace` needs write access.

## Example Makefile Integration

```makefile
# In robot project Makefile
GORAI_IMAGE ?= ghcr.io/gorai/gorai:latest
PODMAN_SOCK := $(shell test -S /run/podman/podman.sock && echo /run/podman/podman.sock || echo /run/user/$$(id -u)/podman/podman.sock)

gorai = podman run --rm -it \
    --security-opt label=disable \
    -v $(PODMAN_SOCK):/run/podman/podman.sock:rw \
    -v $(PWD):/workspace:rw \
    -w /workspace \
    $(GORAI_IMAGE)

up:
    $(gorai) start --config robot.json --build --detach

down:
    $(gorai) stop --config robot.json

status:
    $(gorai) status --config robot.json
```
