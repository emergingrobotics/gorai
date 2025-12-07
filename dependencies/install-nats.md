# NATS Installation Guide

This document summarizes the NATS components required for Gorai and provides quick installation instructions. For comprehensive setup, configuration, and troubleshooting, see the full guide at [`../nats/nats-setup.md`](../nats/nats-setup.md).

---

## Required Components

| Component | Minimum Version | Purpose |
|-----------|-----------------|---------|
| **NATS Server** | 2.10+ | Message broker with JetStream |
| **NATS CLI** | 0.1.0+ | Administration, testing, debugging |

### What is JetStream?

JetStream is NATS's persistence layer. Gorai requires JetStream for:

- **Streams**: Persistent message storage for sensor data, commands, logs
- **KV Store**: Parameter storage with watch capability (similar to etcd/Redis)
- **Object Store**: Large binary storage for ML models and SLAM maps

---

## Quick Install

### Linux (Production)

```bash
# NATS Server - Binary install (recommended)
curl -L https://github.com/nats-io/nats-server/releases/download/v2.10.22/nats-server-v2.10.22-linux-amd64.tar.gz -o nats-server.tar.gz
tar -xzf nats-server.tar.gz
sudo mv nats-server-v2.10.22-linux-amd64/nats-server /usr/local/bin/
rm -rf nats-server.tar.gz nats-server-v2.10.22-linux-amd64

# For ARM64 (Raspberry Pi, Jetson, Rock5B):
curl -L https://github.com/nats-io/nats-server/releases/download/v2.10.22/nats-server-v2.10.22-linux-arm64.tar.gz -o nats-server.tar.gz
# ... same extraction steps

# NATS CLI
curl -L https://github.com/nats-io/natscli/releases/download/v0.1.5/nats-0.1.5-linux-amd64.zip -o nats-cli.zip
unzip nats-cli.zip
sudo mv nats-0.1.5-linux-amd64/nats /usr/local/bin/
rm -rf nats-cli.zip nats-0.1.5-linux-amd64

# Verify
nats-server --version
nats --version
```

### Linux (Go Install - Alternative)

```bash
# Requires Go 1.21+
go install github.com/nats-io/nats-server/v2@latest
go install github.com/nats-io/natscli/nats@latest

# Ensure $GOPATH/bin is in PATH
export PATH=$PATH:$(go env GOPATH)/bin
```

### macOS (Development Only)

```bash
# Homebrew
brew install nats-server
brew tap nats-io/nats-tools
brew install nats-io/nats-tools/nats

# Verify
nats-server --version
nats --version
```

### Container (Podman/Docker)

```bash
# Quick start with JetStream
podman run -d \
  --name gorai-nats \
  -p 4222:4222 \
  -p 8222:8222 \
  -v nats-data:/data:Z \
  docker.io/library/nats:latest \
  -js -sd /data

# Or use the provided Podman Compose file
cd nats/
podman-compose up -d
```

---

## Running NATS

### Development (Quick Start)

```bash
# Start NATS with JetStream enabled
nats-server -js

# Or with data persistence
nats-server -js -sd /tmp/nats-data
```

### Production (Systemd)

See [`../nats/nats-setup.md`](../nats/nats-setup.md#systemd-service-setup) for full systemd configuration.

Quick summary:
```bash
# Copy config
sudo mkdir -p /etc/nats /var/lib/nats/jetstream
sudo cp ../nats/nats.conf /etc/nats/nats.conf

# Create service user
sudo useradd -r -s /bin/false nats
sudo chown -R nats:nats /var/lib/nats

# Enable service (after creating systemd unit file)
sudo systemctl enable nats
sudo systemctl start nats
```

---

## Gorai-Specific Setup

After NATS is running, create Gorai streams and KV buckets:

```bash
# From the repository root
./nats/gorai-nats-setup.sh
```

This creates:

| Resource | Type | Purpose |
|----------|------|---------|
| `GORAI_SENSORS` | Stream | Sensor data (images, IMU, etc.) |
| `GORAI_COMMANDS` | Stream | Control commands |
| `GORAI_STATE` | Stream | State updates (last value) |
| `GORAI_ACTIONS` | Stream | Action goals/feedback/results |
| `GORAI_LOGS` | Stream | System logs |
| `GORAI_PARAMS` | KV Bucket | Robot parameters |
| `GORAI_CONFIG` | KV Bucket | Robot configuration |
| `GORAI_MODELS` | Object Store | ML models |
| `GORAI_MAPS` | Object Store | SLAM maps |

---

## Verification

```bash
# Run the verification script
./nats/gorai-nats-verify.sh

# Or test manually
nats server info          # Check server
nats account info         # Check JetStream
nats stream ls            # List streams
nats kv ls                # List KV buckets
```

### Test Pub/Sub

```bash
# Terminal 1: Subscribe
nats sub "gorai.>"

# Terminal 2: Publish
nats pub gorai.test.sensor.data '{"temperature": 45.2}'
```

---

## Network Ports

| Port | Protocol | Purpose |
|------|----------|---------|
| 4222 | TCP | Client connections |
| 6222 | TCP | Cluster routing (optional) |
| 8222 | HTTP | Monitoring endpoint |

---

## Configuration Files

All configuration files are in the [`../nats/`](../nats/) directory:

| File | Purpose |
|------|---------|
| `nats.conf` | NATS server configuration |
| `gorai-nats-setup.sh` | Creates Gorai streams/KV buckets |
| `gorai-nats-verify.sh` | Verifies setup is correct |
| `podman-compose.yml` | Container deployment |
| `nats-setup.md` | Full documentation |

---

## Troubleshooting

### "Connection refused" on port 4222

```bash
# Check if NATS is running
pgrep nats-server
sudo systemctl status nats  # Linux

# Start NATS
nats-server -js
```

### "JetStream not enabled"

Ensure you start with `-js` flag or have `jetstream` block in config:

```hcl
jetstream {
    store_dir: "/var/lib/nats/jetstream"
}
```

### Full Troubleshooting Guide

See [`../nats/nats-setup.md#troubleshooting`](../nats/nats-setup.md#troubleshooting)

---

## Go Client Library

Gorai uses the official NATS Go client. This is already in `go.mod`:

```go
require github.com/nats-io/nats.go v1.37.0
```

No additional installation needed for Go development.

---

## Summary

**Minimum installation for development:**
1. Install NATS Server: `go install github.com/nats-io/nats-server/v2@latest`
2. Install NATS CLI: `go install github.com/nats-io/natscli/nats@latest`
3. Start server: `nats-server -js`
4. Setup Gorai resources: `./nats/gorai-nats-setup.sh`
5. Verify: `./nats/gorai-nats-verify.sh`

**For production deployment**, see the full guide: [`../nats/nats-setup.md`](../nats/nats-setup.md)
