# NATS Setup Guide for Gorai

This guide covers installing, configuring, and verifying NATS for use with Gorai. NATS is the messaging backbone of Gorai, providing pub/sub, request/reply, and persistent storage.

## Files in this Directory

| File | Purpose |
|------|---------|
| `nats-setup.md` | This guide |
| `nats.conf` | NATS server configuration file |
| `gorai-nats-setup.sh` | Creates Gorai streams and KV buckets |
| `gorai-nats-verify.sh` | Verifies NATS is configured correctly |
| `podman-compose.yml` | Podman Compose configuration |

## Table of Contents

1. [Overview](#overview)
2. [Linux Installation](#linux-installation)
3. [macOS Installation (Development Only)](#macos-installation-development-only)
4. [Podman Installation](#podman-installation)
5. [Configuration](#configuration)
6. [JetStream Setup](#jetstream-setup)
7. [Gorai Stream and KV Configuration](#gorai-stream-and-kv-configuration)
8. [Verification Scripts](#verification-scripts)
9. [Troubleshooting](#troubleshooting)

---

## Overview

Gorai requires NATS with JetStream enabled. JetStream provides:

- **Streams**: Persistent message storage for sensor data, logs
- **KV Store**: Parameter storage with watch capability
- **Object Store**: Large binary storage (maps, models)

### Minimum Requirements

| Component | Version | Notes |
|-----------|---------|-------|
| NATS Server | 2.10+ | JetStream support required |
| NATS CLI | 0.1.0+ | For administration and testing |

### Network Ports

| Port | Protocol | Purpose |
|------|----------|---------|
| 4222 | TCP | Client connections |
| 6222 | TCP | Cluster routing (optional) |
| 8222 | HTTP | Monitoring endpoint |

---

## Linux Installation

### Option 1: Binary Installation (Recommended for Production)

```bash
# Download latest release
curl -L https://github.com/nats-io/nats-server/releases/download/v2.10.22/nats-server-v2.10.22-linux-amd64.tar.gz -o nats-server.tar.gz

# For ARM64 (Raspberry Pi, Jetson, Rock5B):
# curl -L https://github.com/nats-io/nats-server/releases/download/v2.10.22/nats-server-v2.10.22-linux-arm64.tar.gz -o nats-server.tar.gz

# Extract and install
tar -xzf nats-server.tar.gz
sudo mv nats-server-v2.10.22-linux-amd64/nats-server /usr/local/bin/
rm -rf nats-server.tar.gz nats-server-v2.10.22-linux-amd64

# Verify installation
nats-server --version
```

### Option 2: Go Install

```bash
# Requires Go 1.21+
go install github.com/nats-io/nats-server/v2@latest

# Ensure $GOPATH/bin is in your PATH
export PATH=$PATH:$(go env GOPATH)/bin
```

### Option 3: Package Manager

```bash
# Ubuntu/Debian (may not be latest version)
sudo apt-get update
sudo apt-get install -y nats-server

# Fedora
sudo dnf install nats-server

# Arch Linux
sudo pacman -S nats-server
```

### Install NATS CLI

```bash
# Binary download
curl -L https://github.com/nats-io/natscli/releases/download/v0.1.5/nats-0.1.5-linux-amd64.zip -o nats-cli.zip
unzip nats-cli.zip
sudo mv nats-0.1.5-linux-amd64/nats /usr/local/bin/
rm -rf nats-cli.zip nats-0.1.5-linux-amd64

# Or via Go
go install github.com/nats-io/natscli/nats@latest
```

### Systemd Service Setup

Create `/etc/systemd/system/nats.service`:

```ini
[Unit]
Description=NATS Server
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/nats-server -c /etc/nats/nats.conf
ExecReload=/bin/kill -HUP $MAINPID
Restart=on-failure
RestartSec=5
User=nats
Group=nats
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

Setup:

```bash
# Create nats user
sudo useradd -r -s /bin/false nats

# Create directories
sudo mkdir -p /etc/nats /var/lib/nats/jetstream
sudo chown -R nats:nats /var/lib/nats

# Copy configuration (see Configuration section below)
sudo cp nats.conf /etc/nats/nats.conf

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable nats
sudo systemctl start nats

# Check status
sudo systemctl status nats
```

---

## macOS Installation (Development Only)

> **Note**: macOS installation is intended for development and testing only. Production Gorai deployments should use Linux.

### Homebrew (Recommended)

```bash
# Install NATS server
brew install nats-server

# Install NATS CLI
brew tap nats-io/nats-tools
brew install nats-io/nats-tools/nats

# Verify
nats-server --version
nats --version
```

### Go Install

```bash
go install github.com/nats-io/nats-server/v2@latest
go install github.com/nats-io/natscli/nats@latest
```

### Running on macOS

For development, run directly in terminal:

```bash
# Simple JetStream server
nats-server -js -sd /tmp/nats-data

# With configuration file
nats-server -c ~/gorai-nats.conf
```

For background service (launchd):

Create `~/Library/LaunchAgents/io.nats.server.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>io.nats.server</string>
    <key>ProgramArguments</key>
    <array>
        <string>/opt/homebrew/bin/nats-server</string>
        <string>-js</string>
        <string>-sd</string>
        <string>/tmp/nats-data</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

Load with:

```bash
launchctl load ~/Library/LaunchAgents/io.nats.server.plist
```

---

## Podman Installation

Gorai uses Podman as its reference container runtime. Podman is daemonless, rootless-capable, and produces OCI-compliant containers.

### Install Podman

```bash
# Fedora
sudo dnf install podman podman-compose

# Ubuntu/Debian
sudo apt-get install podman podman-compose

# macOS (via Homebrew)
brew install podman podman-compose
podman machine init
podman machine start
```

### Quick Start

```bash
# Run NATS with JetStream
podman run -d \
  --name gorai-nats \
  -p 4222:4222 \
  -p 8222:8222 \
  -v nats-data:/data:Z \
  docker.io/library/nats:latest \
  -js \
  -sd /data

# Verify
podman logs gorai-nats
```

### Podman Compose

Use the included `podman-compose.yml` file:

```bash
# From the nats/ directory
podman-compose up -d
podman-compose logs -f nats
```

### Rootless Operation

Podman can run containers without root privileges:

```bash
# Ensure user namespaces are configured
podman system migrate

# Run rootless
podman run -d \
  --name gorai-nats \
  -p 4222:4222 \
  -p 8222:8222 \
  nats:latest -js
```

> **Note**: For Docker compatibility, you can alias `docker` to `podman`:
> ```bash
> alias docker=podman
> alias docker-compose=podman-compose
> ```

---

## Configuration

### Basic Configuration (`nats.conf`)

A ready-to-use configuration file is included as `nats.conf` in this directory. Copy it to the appropriate location:

```bash
# Linux production
sudo cp nats.conf /etc/nats/nats.conf

# macOS development
cp nats.conf ~/gorai-nats.conf

# Run with config
nats-server -c /etc/nats/nats.conf      # Linux
nats-server -c ~/gorai-nats.conf        # macOS
```

The configuration includes:

```hcl
# Gorai NATS Configuration
# /etc/nats/nats.conf (Linux) or ~/gorai-nats.conf (macOS dev)

# Server identification
server_name: gorai-nats

# Client connections
port: 4222
host: 0.0.0.0

# HTTP monitoring
http_port: 8222

# Maximum payload (10MB for images/point clouds)
max_payload: 10485760

# Connection limits
max_connections: 1024
max_subscriptions: 0  # unlimited

# Logging
debug: false
trace: false
logtime: true
log_file: "/var/log/nats/nats.log"  # Linux
# log_file: "/tmp/nats.log"  # macOS dev

# JetStream configuration
jetstream {
    store_dir: "/var/lib/nats/jetstream"  # Linux
    # store_dir: "/tmp/nats-data"  # macOS dev

    # Memory and storage limits
    max_memory_store: 1GB
    max_file_store: 10GB
}

# Authorization (optional, for production)
# authorization {
#     users = [
#         { user: gorai, password: $GORAI_NATS_PASSWORD }
#     ]
# }
```

### Production Configuration Additions

For production deployments, add:

```hcl
# TLS configuration
tls {
    cert_file: "/etc/nats/server-cert.pem"
    key_file: "/etc/nats/server-key.pem"
    ca_file: "/etc/nats/ca.pem"
    verify: true
}

# Cluster configuration (for multi-node)
cluster {
    name: gorai-cluster
    port: 6222
    routes: [
        nats://nats-1:6222
        nats://nats-2:6222
    ]
}
```

---

## JetStream Setup

### Enable JetStream

JetStream is enabled via the `-js` flag or the `jetstream` block in configuration.

Verify JetStream is running:

```bash
# Check server info
nats server info

# Should show JetStream enabled
nats account info
```

---

## Gorai Stream and KV Configuration

Gorai uses specific JetStream streams and KV buckets. The `gorai-nats-setup.sh` script creates all required resources.

### Streams Created

| Stream | Purpose | Retention |
|--------|---------|-----------|
| `GORAI_SENSORS` | Sensor data (images, IMU, etc.) | 1 hour, 1GB max |
| `GORAI_COMMANDS` | Control commands | 24 hours, 100MB max |
| `GORAI_STATE` | State updates (last value per subject) | 100MB max |
| `GORAI_ACTIONS` | Action goals, feedback, results | 1 hour, 500MB max |
| `GORAI_LOGS` | System logs and diagnostics | 7 days, 5GB max |

### KV Buckets Created

| Bucket | Purpose | History |
|--------|---------|---------|
| `GORAI_PARAMS` | Robot parameters (gains, thresholds) | 5 revisions |
| `GORAI_CONFIG` | Robot configuration | 10 revisions |

### Object Stores Created

| Store | Purpose | Max Size |
|-------|---------|----------|
| `GORAI_MODELS` | ML model storage | 10GB |
| `GORAI_MAPS` | SLAM map storage | 5GB |

### Run the Setup Script

```bash
# From the nats/ directory
./gorai-nats-setup.sh

# Or specify a different NATS server
NATS_URL=nats://192.168.1.100:4222 ./gorai-nats-setup.sh
```

---

## Verification Scripts

### Run Verification

The `gorai-nats-verify.sh` script tests:

1. Server connectivity
2. JetStream enabled
3. Required streams exist
4. Required KV buckets exist
5. Pub/Sub round-trip
6. KV put/get/delete operations
7. JetStream publish

```bash
# Run verification
./gorai-nats-verify.sh

# With custom server and robot ID
NATS_URL=nats://192.168.1.100:4222 ROBOT=sentinel ./gorai-nats-verify.sh
```

Expected output:
```
Verifying NATS configuration for Gorai
=======================================
Server: nats://localhost:4222
Robot ID: test

Test 1: Server connectivity...
  ✓ NATS server is reachable

Test 2: JetStream enabled...
  ✓ JetStream is enabled

Test 3: Required streams...
  ✓ Stream GORAI_SENSORS exists
  ✓ Stream GORAI_COMMANDS exists
  ✓ Stream GORAI_STATE exists
  ✓ Stream GORAI_ACTIONS exists
  ✓ Stream GORAI_LOGS exists

Test 4: Required KV buckets...
  ✓ KV bucket GORAI_PARAMS exists
  ✓ KV bucket GORAI_CONFIG exists

Test 5: Pub/Sub round-trip...
  ✓ Pub/Sub round-trip successful

Test 6: KV operations...
  ✓ KV put/get/delete successful

Test 7: JetStream publish...
  ✓ JetStream publish successful

=======================================
All verification tests passed!
```

### Interactive Testing

Test pub/sub manually:

```bash
# Terminal 1: Subscribe to all Gorai messages
nats sub "gorai.>"

# Terminal 2: Publish test messages
nats pub gorai.test.camera.data '{"width":640,"height":480}'
nats pub gorai.test.motor.command '{"power":0.5}'
nats pub gorai.test.vision.detections '{"count":3}'
```

Test KV store:

```bash
# Set parameters
nats kv put GORAI_PARAMS camera.exposure 100
nats kv put GORAI_PARAMS camera.gain 1.5
nats kv put GORAI_PARAMS motor.max_rpm 3000

# Get parameters
nats kv get GORAI_PARAMS camera.exposure

# Watch for changes (in another terminal)
nats kv watch GORAI_PARAMS ">"

# Update a value
nats kv put GORAI_PARAMS camera.exposure 200
```

### Monitoring

View server statistics:

```bash
# Server info
nats server info

# Real-time stats
nats server report connections
nats server report jetstream

# Stream statistics
nats stream report

# Monitor HTTP endpoint
curl http://localhost:8222/varz | jq .
curl http://localhost:8222/jsz | jq .
```

---

## Troubleshooting

### Common Issues

#### "Connection refused" on port 4222

```bash
# Check if NATS is running
sudo systemctl status nats    # Linux
pgrep nats-server             # Any OS

# Check port binding
sudo netstat -tlnp | grep 4222
sudo lsof -i :4222

# Start NATS if not running
sudo systemctl start nats     # Linux systemd
nats-server -js               # Manual start
```

#### "JetStream not enabled"

Ensure your configuration includes:

```hcl
jetstream {
    store_dir: "/var/lib/nats/jetstream"
}
```

Or start with `-js` flag:

```bash
nats-server -js -sd /tmp/nats-data
```

#### "Permission denied" on JetStream directory

```bash
# Fix ownership
sudo chown -R nats:nats /var/lib/nats

# Or use a different directory
nats-server -js -sd /tmp/nats-data
```

#### Streams not persisting after restart

Ensure `store_dir` points to a persistent location, not `/tmp`:

```hcl
jetstream {
    store_dir: "/var/lib/nats/jetstream"
}
```

#### High memory usage

Adjust JetStream limits:

```hcl
jetstream {
    max_memory_store: 256MB   # Reduce from default
    max_file_store: 1GB       # Reduce from default
}
```

Or reduce stream retention:

```bash
nats stream edit GORAI_SENSORS --max-msgs 10000 --max-bytes 100MB
```

### Logs

View NATS logs:

```bash
# Systemd
sudo journalctl -u nats -f

# Podman
podman logs -f gorai-nats

# Log file
tail -f /var/log/nats/nats.log
```

Enable debug logging (temporarily):

```bash
nats-server -js -DV
```

### Reset Everything

To completely reset NATS (destroys all data):

```bash
# Stop NATS
sudo systemctl stop nats

# Remove data
sudo rm -rf /var/lib/nats/jetstream/*

# Start fresh
sudo systemctl start nats

# Re-run setup
./gorai-nats-setup.sh
```

---

## Next Steps

After NATS is configured:

1. Run the verification script to ensure everything is working
2. Try the [minimal example](../examples/minimal/main.go)
3. Read the [Framework Specification](../specs/gorai-framework-specification.md) for topic naming conventions
4. Set up your robot configuration in the GORAI_CONFIG KV bucket
