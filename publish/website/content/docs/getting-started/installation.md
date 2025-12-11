---
title: "Installation"
description: "Install Gorai and its dependencies"
weight: 10
---

# Installation

This guide covers installing Gorai and its dependencies on your development machine.

## Prerequisites

- **Go 1.21+** — [Download Go](https://go.dev/dl/)
- **NATS Server** — Local or remote NATS installation
- **Git** — For cloning repositories

## Install Go

### macOS

```bash
brew install go
```

### Ubuntu/Debian

```bash
sudo apt update
sudo apt install golang-go
```

### Verify Installation

```bash
go version
# go version go1.21.0 linux/amd64
```

## Install NATS

### Option 1: Binary

```bash
# macOS
brew install nats-server

# Linux
curl -L https://github.com/nats-io/nats-server/releases/download/v2.10.0/nats-server-v2.10.0-linux-amd64.tar.gz | tar xz
sudo mv nats-server-v2.10.0-linux-amd64/nats-server /usr/local/bin/
```

### Option 2: Docker

```bash
docker run -p 4222:4222 nats:latest
```

### Verify Installation

```bash
nats-server --version
# nats-server: v2.10.0
```

## Install Gorai CLI

```bash
go install github.com/gorai/gorai/cmd/gorai@latest
```

Verify:

```bash
gorai version
```

## Next Steps

With everything installed, proceed to the [Quick Start](../quickstart/) to build your first component.
