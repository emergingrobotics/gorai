# Installation

## Prerequisites

- Go 1.21 or later
- NATS server (local or remote)
- Linux (primary) or macOS (development)

## Install GoRAI

```bash
go get github.com/gorai/gorai
```

## Start NATS

Using Podman:

```bash
podman run -d --name nats -p 4222:4222 nats:latest
```

Or Docker:

```bash
docker run -d --name nats -p 4222:4222 nats:latest
```

## Verify Installation

```bash
go run github.com/gorai/gorai/cmd/gorai version
```

## Next Steps

- [Quick Start](quickstart.md) - Build your first GoRAI node
- [Concepts](concepts.md) - Understand the architecture
