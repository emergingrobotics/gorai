# Deployment Specification

**Version:** 2.0
**Status:** Draft
**Last Updated:** 2024-12-24

## 1. Overview

This specification defines how Gorai robot binaries and configurations are deployed from a development machine to target robot hardware.

Gorai embraces **distributed systems thinking** from the ground up. Deployment strategies match robot complexity across three tiers:

- **Tier 1: Simple Robots** (this spec) — Native binaries or few containers managed by systemd
- **Tier 2: Complex Robots** — K3s single-node for orchestration features (health monitoring, rolling updates, resource limits)
- **Tier 3: Fleet Management** — K3s multi-node clusters for edge-cloud hybrid deployments

**This document focuses on Tier 1 deployment** — the recommended starting point for most robots. For complex single robots needing orchestration (multi-language services, sophisticated ML pipelines), see Tier 2 (K3s single-node). For containerized alternatives, see [systemd-container-orchestration.md](systemd-container-orchestration.md) (Podman pods).

### 1.1 Design Goals

1. **Simple**: `make deploy` should just work
2. **Secure**: No passwords in scripts, use SSH keys
3. **Atomic**: Deployments succeed or fail completely
4. **Reversible**: Easy rollback to previous version
5. **Minimal downtime**: Hot-reload config when possible
6. **Scalable**: Easy upgrade path to Tier 2/3 when needed

### 1.2 Deployment Model (Tier 1)

```
┌─────────────────────┐                    ┌─────────────────────┐
│   Dev Machine       │                    │   Robot (Target)    │
│                     │                    │                     │
│  my-robot/          │                    │  /opt/myrobot/      │
│  ├── main.go        │    make deploy     │  ├── myrobot        │
│  ├── robot.json     │  ────────────────► │  ├── robot.json     │
│  ├── Makefile       │    (ssh + rsync)   │  └── myrobot.service│
│  └── deploy/        │                    │                     │
│      └── *.service  │                    │  systemd            │
└─────────────────────┘                    │  └── manages service│
                                           └─────────────────────┘
```

---

## 2. Deployment Methods

### 2.1 Primary: SSH + rsync

The recommended deployment method uses SSH for authentication and rsync for efficient file transfer.

**Requirements:**
- SSH access to robot (key-based authentication)
- rsync installed on both machines
- sudo access on robot (for systemd)

**Advantages:**
- Simple, standard tools
- Efficient (only transfers changes)
- Secure (SSH encryption)
- Works over any network

### 2.2 Alternative: SCP

For environments without rsync:

```bash
scp build/bin/linux-arm64/myrobot pi@robot:/opt/myrobot/
scp robot.json pi@robot:/opt/myrobot/
```

**Disadvantages:**
- Transfers entire file every time
- No atomic updates

### 2.3 Alternative: USB/SD Card

For initial setup or air-gapped environments:

1. Build binary on dev machine
2. Copy to USB drive
3. Insert USB in robot
4. Copy files to /opt/myrobot/

### 2.4 Future: OTA Updates

Planned for future versions:
- Pull-based updates from server
- Signed binary verification
- Automatic rollback on failure

---

## 3. Target Directory Structure

### 3.1 Standard Layout

```
/opt/{robot_name}/
├── {robot_name}           # Main binary (executable)
├── robot.json             # Active configuration
├── robot.json.backup      # Previous configuration
├── models/                # ML models (optional)
│   └── yolox_s.onnx
├── logs/                  # Log files (if file logging enabled)
│   └── robot.log
└── data/                  # Runtime data (maps, recordings)
    └── maps/
```

### 3.2 Directory Permissions

| Path | Owner | Permissions | Notes |
|------|-------|-------------|-------|
| `/opt/{robot}` | pi:pi | 755 | Base directory |
| `{robot}` binary | pi:pi | 755 | Executable |
| `robot.json` | pi:pi | 644 | Configuration |
| `models/` | pi:pi | 755 | ML models |
| `logs/` | pi:pi | 755 | Log directory |
| `data/` | pi:pi | 755 | Runtime data |

### 3.3 Alternative Locations

| Location | Use Case |
|----------|----------|
| `/opt/{robot}/` | Recommended (standard for add-on software) |
| `/home/pi/{robot}/` | User-space deployment |
| `/usr/local/bin/` | System-wide binary only |
| `/srv/{robot}/` | Server-style deployment |

---

## 4. Deployment Configuration

### 4.1 Makefile Variables

```makefile
# ============================================
# Deployment Configuration
# ============================================

# Robot connection (override with environment or command line)
ROBOT_HOST ?= pi@myrobot.local
ROBOT_USER ?= pi
ROBOT_ADDR ?= myrobot.local

# Paths on robot
ROBOT_PATH ?= /opt/myrobot
ROBOT_BIN  ?= $(ROBOT_PATH)/myrobot
ROBOT_CONF ?= $(ROBOT_PATH)/robot.json

# Service configuration
SERVICE_NAME ?= myrobot
SERVICE_FILE ?= /etc/systemd/system/$(SERVICE_NAME).service

# Build configuration
BINARY_NAME ?= myrobot
BUILD_TARGET ?= linux-arm64
BUILD_DIR ?= build/bin/$(BUILD_TARGET)
```

### 4.2 Environment Variables

```bash
# Set deployment target
export ROBOT_HOST=pi@192.168.1.100
export ROBOT_PATH=/opt/myrobot

# Or use .env file
cat > .env << EOF
ROBOT_HOST=pi@myrobot.local
ROBOT_PATH=/opt/myrobot
EOF
```

### 4.3 Multiple Robots

For deploying to multiple robots:

```makefile
# Define robots
ROBOT1_HOST = pi@robot1.local
ROBOT2_HOST = pi@robot2.local
ROBOT3_HOST = pi@robot3.local

deploy-robot1:
	$(MAKE) deploy ROBOT_HOST=$(ROBOT1_HOST)

deploy-robot2:
	$(MAKE) deploy ROBOT_HOST=$(ROBOT2_HOST)

deploy-all:
	$(MAKE) deploy-robot1
	$(MAKE) deploy-robot2
	$(MAKE) deploy-robot3

# Or with different configs
deploy-fleet:
	$(MAKE) deploy ROBOT_HOST=$(ROBOT1_HOST) CONFIG=configs/robot1.json
	$(MAKE) deploy ROBOT_HOST=$(ROBOT2_HOST) CONFIG=configs/robot2.json
```

---

## 5. Deployment Commands

### 5.1 Full Deployment

```makefile
# Build and deploy everything
deploy: build-$(BUILD_TARGET)
	@echo "Deploying to $(ROBOT_HOST):$(ROBOT_PATH)"

	# Ensure target directory exists
	ssh $(ROBOT_HOST) "sudo mkdir -p $(ROBOT_PATH) && sudo chown $(ROBOT_USER):$(ROBOT_USER) $(ROBOT_PATH)"

	# Backup current config if exists
	ssh $(ROBOT_HOST) "[ -f $(ROBOT_CONF) ] && cp $(ROBOT_CONF) $(ROBOT_CONF).backup || true"

	# Stop service before updating binary
	ssh $(ROBOT_HOST) "sudo systemctl stop $(SERVICE_NAME) 2>/dev/null || true"

	# Transfer binary and config
	rsync -avz --progress \
		$(BUILD_DIR)/$(BINARY_NAME) \
		robot.json \
		$(ROBOT_HOST):$(ROBOT_PATH)/

	# Set executable permission
	ssh $(ROBOT_HOST) "chmod +x $(ROBOT_BIN)"

	# Start service
	ssh $(ROBOT_HOST) "sudo systemctl start $(SERVICE_NAME)"

	@echo "Deployment complete"
```

### 5.2 Binary Only

```makefile
# Deploy only the binary (keeps existing config)
deploy-binary: build-$(BUILD_TARGET)
	ssh $(ROBOT_HOST) "sudo systemctl stop $(SERVICE_NAME) 2>/dev/null || true"
	rsync -avz --progress $(BUILD_DIR)/$(BINARY_NAME) $(ROBOT_HOST):$(ROBOT_PATH)/
	ssh $(ROBOT_HOST) "chmod +x $(ROBOT_BIN) && sudo systemctl start $(SERVICE_NAME)"
```

### 5.3 Config Only

```makefile
# Deploy only configuration (triggers reload)
deploy-config:
	ssh $(ROBOT_HOST) "[ -f $(ROBOT_CONF) ] && cp $(ROBOT_CONF) $(ROBOT_CONF).backup || true"
	rsync -avz robot.json $(ROBOT_HOST):$(ROBOT_PATH)/
	ssh $(ROBOT_HOST) "sudo systemctl reload $(SERVICE_NAME) 2>/dev/null || sudo systemctl restart $(SERVICE_NAME)"
```

### 5.4 Service Installation

```makefile
# Install systemd service (first-time setup)
deploy-service:
	scp deploy/$(SERVICE_NAME).service $(ROBOT_HOST):/tmp/
	ssh $(ROBOT_HOST) "\
		sudo mv /tmp/$(SERVICE_NAME).service $(SERVICE_FILE) && \
		sudo systemctl daemon-reload && \
		sudo systemctl enable $(SERVICE_NAME)"
```

### 5.5 Initial Setup

```makefile
# Complete first-time deployment
deploy-init: build-$(BUILD_TARGET) deploy-service
	# Create directory structure
	ssh $(ROBOT_HOST) "\
		sudo mkdir -p $(ROBOT_PATH)/{models,logs,data} && \
		sudo chown -R $(ROBOT_USER):$(ROBOT_USER) $(ROBOT_PATH)"

	# Deploy files
	rsync -avz --progress \
		$(BUILD_DIR)/$(BINARY_NAME) \
		robot.json \
		$(ROBOT_HOST):$(ROBOT_PATH)/

	# Set permissions and start
	ssh $(ROBOT_HOST) "\
		chmod +x $(ROBOT_BIN) && \
		sudo systemctl start $(SERVICE_NAME)"

	@echo "Initial deployment complete"
	@echo "View logs: make logs"
	@echo "Check status: make status"
```

---

## 6. systemd Service

### 6.1 Service File Template

```ini
# deploy/myrobot.service
[Unit]
Description=MyRobot Gorai Robot
Documentation=https://github.com/myorg/myrobot
After=network-online.target nats.service
Wants=network-online.target nats.service

[Service]
Type=simple
User=pi
Group=pi
WorkingDirectory=/opt/myrobot
ExecStart=/opt/myrobot/myrobot --config /opt/myrobot/robot.json
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=myrobot

# Environment
Environment="NATS_URL=nats://localhost:4222"
EnvironmentFile=-/opt/myrobot/.env

# Hardware access groups
SupplementaryGroups=gpio i2c spi video dialout input

# Resource limits
MemoryMax=512M
CPUQuota=80%

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/myrobot /tmp
PrivateTmp=true

# Capabilities for hardware access
AmbientCapabilities=CAP_NET_RAW CAP_SYS_RAWIO

[Install]
WantedBy=multi-user.target
```

### 6.2 Service Commands

```makefile
# Start robot
start:
	ssh $(ROBOT_HOST) "sudo systemctl start $(SERVICE_NAME)"

# Stop robot
stop:
	ssh $(ROBOT_HOST) "sudo systemctl stop $(SERVICE_NAME)"

# Restart robot
restart:
	ssh $(ROBOT_HOST) "sudo systemctl restart $(SERVICE_NAME)"

# Reload configuration (SIGHUP)
reload:
	ssh $(ROBOT_HOST) "sudo systemctl reload $(SERVICE_NAME)"

# Check status
status:
	ssh $(ROBOT_HOST) "sudo systemctl status $(SERVICE_NAME)"

# View logs
logs:
	ssh $(ROBOT_HOST) "sudo journalctl -u $(SERVICE_NAME) -f"

# View recent logs
logs-recent:
	ssh $(ROBOT_HOST) "sudo journalctl -u $(SERVICE_NAME) -n 100"

# Enable auto-start on boot
enable:
	ssh $(ROBOT_HOST) "sudo systemctl enable $(SERVICE_NAME)"

# Disable auto-start
disable:
	ssh $(ROBOT_HOST) "sudo systemctl disable $(SERVICE_NAME)"
```

### 6.3 Service Signals

| Signal | systemd Command | Robot Behavior |
|--------|-----------------|----------------|
| SIGTERM | `systemctl stop` | Graceful shutdown |
| SIGHUP | `systemctl reload` | Reload configuration |
| SIGINT | (manual) | Graceful shutdown |
| SIGKILL | `systemctl kill` | Immediate termination |

---

## 7. NATS Deployment

### 7.1 NATS on Robot (Tier 1)

For Tier 1 deployments, NATS runs as a native systemd service:

```ini
# /etc/systemd/system/nats.service
[Unit]
Description=NATS Message Broker
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/nats-server -js -sd /var/lib/nats
Restart=always
User=nats
Group=nats

[Install]
WantedBy=multi-user.target
```

### 7.2 NATS Installation Script

```bash
#!/bin/bash
# scripts/install-nats.sh

NATS_VERSION="2.10.7"
ARCH=$(uname -m)

case $ARCH in
    aarch64) NATS_ARCH="arm64" ;;
    armv7l)  NATS_ARCH="arm7" ;;
    x86_64)  NATS_ARCH="amd64" ;;
    *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Download and install
wget "https://github.com/nats-io/nats-server/releases/download/v${NATS_VERSION}/nats-server-v${NATS_VERSION}-linux-${NATS_ARCH}.tar.gz"
tar xzf nats-server-*.tar.gz
sudo mv nats-server-*/nats-server /usr/local/bin/
rm -rf nats-server-*

# Create user and directories
sudo useradd -r -s /bin/false nats || true
sudo mkdir -p /var/lib/nats /etc/nats
sudo chown nats:nats /var/lib/nats

# Install service
sudo cp nats.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable nats
sudo systemctl start nats

echo "NATS installed and running"
```

### 7.3 NATS in Containers (Tier 2)

For robots requiring containerized deployment (multi-language services, complex dependencies), NATS runs in a Podman container managed by systemd. See [systemd-container-orchestration.md](systemd-container-orchestration.md) for container-based deployment.

### 7.4 NATS Cluster (Multi-Robot)

For multi-robot systems, configure NATS cluster:

```conf
# /etc/nats/nats.conf (on each node)
server_name: robot1

port: 4222
http_port: 8222

jetstream {
    store_dir: /var/lib/nats
}

cluster {
    name: robot_cluster
    port: 6222
    routes: [
        nats://robot1.local:6222
        nats://robot2.local:6222
        nats://robot3.local:6222
    ]
}
```

---

## 8. SSH Configuration

### 8.1 SSH Key Setup

```bash
# Generate key (if not exists)
ssh-keygen -t ed25519 -f ~/.ssh/robot_deploy -N ""

# Copy to robot
ssh-copy-id -i ~/.ssh/robot_deploy.pub pi@myrobot.local

# Or manually
cat ~/.ssh/robot_deploy.pub | ssh pi@myrobot.local "mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys"
```

### 8.2 SSH Config

```
# ~/.ssh/config
Host myrobot
    HostName myrobot.local
    User pi
    IdentityFile ~/.ssh/robot_deploy
    StrictHostKeyChecking accept-new

Host robot-*
    User pi
    IdentityFile ~/.ssh/robot_deploy
    StrictHostKeyChecking accept-new

Host robot-1
    HostName 192.168.1.101

Host robot-2
    HostName 192.168.1.102
```

### 8.3 Connection Testing

```makefile
# Test SSH connection
ssh-test:
	@ssh -o ConnectTimeout=5 $(ROBOT_HOST) "echo 'Connection OK'"

# Interactive SSH
ssh:
	ssh $(ROBOT_HOST)
```

---

## 9. Rollback

### 9.1 Configuration Rollback

```makefile
# Rollback to previous configuration
rollback-config:
	ssh $(ROBOT_HOST) "\
		[ -f $(ROBOT_CONF).backup ] && \
		cp $(ROBOT_CONF).backup $(ROBOT_CONF) && \
		sudo systemctl restart $(SERVICE_NAME)"
```

### 9.2 Binary Rollback

```makefile
# Keep previous binary
deploy-with-backup: build-$(BUILD_TARGET)
	ssh $(ROBOT_HOST) "[ -f $(ROBOT_BIN) ] && cp $(ROBOT_BIN) $(ROBOT_BIN).backup || true"
	$(MAKE) deploy

# Rollback to previous binary
rollback-binary:
	ssh $(ROBOT_HOST) "\
		[ -f $(ROBOT_BIN).backup ] && \
		sudo systemctl stop $(SERVICE_NAME) && \
		cp $(ROBOT_BIN).backup $(ROBOT_BIN) && \
		sudo systemctl start $(SERVICE_NAME)"
```

### 9.3 Full Rollback

```makefile
# Rollback both binary and config
rollback:
	ssh $(ROBOT_HOST) "\
		sudo systemctl stop $(SERVICE_NAME) && \
		[ -f $(ROBOT_BIN).backup ] && cp $(ROBOT_BIN).backup $(ROBOT_BIN) || true && \
		[ -f $(ROBOT_CONF).backup ] && cp $(ROBOT_CONF).backup $(ROBOT_CONF) || true && \
		sudo systemctl start $(SERVICE_NAME)"
```

---

## 10. Health Checks

### 10.1 Service Health

```makefile
# Check if service is running
health:
	@ssh $(ROBOT_HOST) "systemctl is-active $(SERVICE_NAME)" || \
		(echo "Service not running!"; exit 1)

# Detailed health check
health-full:
	@echo "=== Service Status ==="
	@ssh $(ROBOT_HOST) "systemctl status $(SERVICE_NAME) --no-pager"
	@echo ""
	@echo "=== Recent Logs ==="
	@ssh $(ROBOT_HOST) "journalctl -u $(SERVICE_NAME) -n 20 --no-pager"
	@echo ""
	@echo "=== Resource Usage ==="
	@ssh $(ROBOT_HOST) "ps aux | grep $(BINARY_NAME) | grep -v grep"
```

### 10.2 NATS Health

```makefile
# Check NATS connection
nats-health:
	@ssh $(ROBOT_HOST) "curl -s http://localhost:8222/varz | head -20"
```

### 10.3 Pre-Deploy Checks

```makefile
# Verify deployment prerequisites
pre-deploy-check:
	@echo "Checking prerequisites..."
	@ssh -o ConnectTimeout=5 $(ROBOT_HOST) "echo 'SSH: OK'" || \
		(echo "SSH connection failed"; exit 1)
	@ssh $(ROBOT_HOST) "which rsync" > /dev/null || \
		(echo "rsync not installed on robot"; exit 1)
	@ssh $(ROBOT_HOST) "which systemctl" > /dev/null || \
		(echo "systemd not available"; exit 1)
	@echo "All checks passed"
```

---

## 11. Troubleshooting

### 11.1 Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Connection refused | SSH not running | Enable SSH on robot |
| Permission denied | Wrong key/password | Check SSH key setup |
| rsync not found | Not installed | `sudo apt install rsync` |
| Service won't start | Missing dependencies | Check `journalctl -u service` |
| Config not found | Wrong path | Verify ROBOT_PATH |
| Binary won't execute | Wrong architecture | Check `file binary` |

### 11.2 Debug Commands

```makefile
# Check binary architecture on robot
debug-binary:
	ssh $(ROBOT_HOST) "file $(ROBOT_BIN)"

# Check service dependencies
debug-service:
	ssh $(ROBOT_HOST) "systemctl list-dependencies $(SERVICE_NAME)"

# Check config syntax
debug-config:
	ssh $(ROBOT_HOST) "cat $(ROBOT_CONF) | python3 -m json.tool"

# Check hardware groups
debug-groups:
	ssh $(ROBOT_HOST) "groups $(ROBOT_USER)"

# List open files
debug-files:
	ssh $(ROBOT_HOST) "sudo lsof -p \$$(pgrep $(BINARY_NAME))"
```

---

## 12. Security Considerations

### 12.1 SSH Security

- Use key-based authentication only
- Disable password authentication: `PasswordAuthentication no`
- Use strong keys (Ed25519 recommended)
- Consider SSH certificates for fleet management

### 12.2 Service Security

The systemd service file includes security hardening:
- `NoNewPrivileges=true` - Prevent privilege escalation
- `ProtectSystem=strict` - Read-only system directories
- `ProtectHome=true` - No access to /home
- `PrivateTmp=true` - Isolated /tmp

### 12.3 Secrets Management

Never put secrets in robot.json. Use:
- Environment variables
- `/opt/myrobot/.env` file (mode 600)
- systemd credentials (future)

```bash
# .env file on robot
NATS_USER=robot
NATS_PASS=secretpassword
API_KEY=xxxx
```

---

## Appendix A: Complete Makefile

```makefile
# ============================================
# Robot Deployment Makefile
# ============================================

# Configuration
BINARY_NAME ?= myrobot
BUILD_TARGET ?= linux-arm64
ROBOT_HOST ?= pi@myrobot.local
ROBOT_PATH ?= /opt/myrobot
SERVICE_NAME ?= myrobot

# Derived
BUILD_DIR = build/bin/$(BUILD_TARGET)
ROBOT_BIN = $(ROBOT_PATH)/$(BINARY_NAME)
ROBOT_CONF = $(ROBOT_PATH)/robot.json

# ============================================
# Build
# ============================================

.PHONY: build build-pi build-all

build:
	go build -o build/bin/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

build-pi: build-linux-arm64

build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

build-linux-armv7:
	GOOS=linux GOARCH=arm GOARM=7 go build -o build/bin/linux-armv7/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

build-all: build-linux-arm64 build-linux-armv7

# ============================================
# Deploy
# ============================================

.PHONY: deploy deploy-binary deploy-config deploy-service deploy-init

deploy: build-$(BUILD_TARGET) pre-deploy-check
	@echo "Deploying to $(ROBOT_HOST):$(ROBOT_PATH)"
	ssh $(ROBOT_HOST) "sudo mkdir -p $(ROBOT_PATH) && sudo chown pi:pi $(ROBOT_PATH)"
	ssh $(ROBOT_HOST) "[ -f $(ROBOT_CONF) ] && cp $(ROBOT_CONF) $(ROBOT_CONF).backup || true"
	ssh $(ROBOT_HOST) "sudo systemctl stop $(SERVICE_NAME) 2>/dev/null || true"
	rsync -avz --progress $(BUILD_DIR)/$(BINARY_NAME) robot.json $(ROBOT_HOST):$(ROBOT_PATH)/
	ssh $(ROBOT_HOST) "chmod +x $(ROBOT_BIN) && sudo systemctl start $(SERVICE_NAME)"
	@echo "Deployment complete"

deploy-binary: build-$(BUILD_TARGET)
	ssh $(ROBOT_HOST) "sudo systemctl stop $(SERVICE_NAME) 2>/dev/null || true"
	rsync -avz --progress $(BUILD_DIR)/$(BINARY_NAME) $(ROBOT_HOST):$(ROBOT_PATH)/
	ssh $(ROBOT_HOST) "chmod +x $(ROBOT_BIN) && sudo systemctl start $(SERVICE_NAME)"

deploy-config:
	ssh $(ROBOT_HOST) "[ -f $(ROBOT_CONF) ] && cp $(ROBOT_CONF) $(ROBOT_CONF).backup || true"
	rsync -avz robot.json $(ROBOT_HOST):$(ROBOT_PATH)/
	ssh $(ROBOT_HOST) "sudo systemctl reload $(SERVICE_NAME) 2>/dev/null || sudo systemctl restart $(SERVICE_NAME)"

deploy-service:
	scp deploy/$(SERVICE_NAME).service $(ROBOT_HOST):/tmp/
	ssh $(ROBOT_HOST) "sudo mv /tmp/$(SERVICE_NAME).service /etc/systemd/system/ && sudo systemctl daemon-reload && sudo systemctl enable $(SERVICE_NAME)"

deploy-init: build-$(BUILD_TARGET) deploy-service
	ssh $(ROBOT_HOST) "sudo mkdir -p $(ROBOT_PATH)/{models,logs,data} && sudo chown -R pi:pi $(ROBOT_PATH)"
	rsync -avz --progress $(BUILD_DIR)/$(BINARY_NAME) robot.json $(ROBOT_HOST):$(ROBOT_PATH)/
	ssh $(ROBOT_HOST) "chmod +x $(ROBOT_BIN) && sudo systemctl start $(SERVICE_NAME)"

# ============================================
# Service Management
# ============================================

.PHONY: start stop restart reload status logs ssh

start:
	ssh $(ROBOT_HOST) "sudo systemctl start $(SERVICE_NAME)"

stop:
	ssh $(ROBOT_HOST) "sudo systemctl stop $(SERVICE_NAME)"

restart:
	ssh $(ROBOT_HOST) "sudo systemctl restart $(SERVICE_NAME)"

reload:
	ssh $(ROBOT_HOST) "sudo systemctl reload $(SERVICE_NAME)"

status:
	ssh $(ROBOT_HOST) "sudo systemctl status $(SERVICE_NAME)"

logs:
	ssh $(ROBOT_HOST) "sudo journalctl -u $(SERVICE_NAME) -f"

ssh:
	ssh $(ROBOT_HOST)

# ============================================
# Health & Debug
# ============================================

.PHONY: health pre-deploy-check

health:
	@ssh $(ROBOT_HOST) "systemctl is-active $(SERVICE_NAME)" && echo "Service: OK" || echo "Service: FAILED"

pre-deploy-check:
	@ssh -o ConnectTimeout=5 $(ROBOT_HOST) "echo 'SSH: OK'" || (echo "SSH failed"; exit 1)
	@ssh $(ROBOT_HOST) "which rsync > /dev/null" || (echo "rsync not found"; exit 1)

# ============================================
# Rollback
# ============================================

.PHONY: rollback rollback-config rollback-binary

rollback:
	ssh $(ROBOT_HOST) "sudo systemctl stop $(SERVICE_NAME)"
	ssh $(ROBOT_HOST) "[ -f $(ROBOT_BIN).backup ] && cp $(ROBOT_BIN).backup $(ROBOT_BIN) || true"
	ssh $(ROBOT_HOST) "[ -f $(ROBOT_CONF).backup ] && cp $(ROBOT_CONF).backup $(ROBOT_CONF) || true"
	ssh $(ROBOT_HOST) "sudo systemctl start $(SERVICE_NAME)"

rollback-config:
	ssh $(ROBOT_HOST) "[ -f $(ROBOT_CONF).backup ] && cp $(ROBOT_CONF).backup $(ROBOT_CONF)"
	ssh $(ROBOT_HOST) "sudo systemctl restart $(SERVICE_NAME)"

rollback-binary:
	ssh $(ROBOT_HOST) "sudo systemctl stop $(SERVICE_NAME)"
	ssh $(ROBOT_HOST) "[ -f $(ROBOT_BIN).backup ] && cp $(ROBOT_BIN).backup $(ROBOT_BIN)"
	ssh $(ROBOT_HOST) "sudo systemctl start $(SERVICE_NAME)"
```

---

## Appendix B: First-Time Robot Setup

```bash
#!/bin/bash
# scripts/setup-robot.sh
# Run this on a fresh Raspberry Pi

set -e

echo "=== Gorai Robot Setup ==="

# Update system
sudo apt update && sudo apt upgrade -y

# Install dependencies
sudo apt install -y rsync git

# Add user to hardware groups
sudo usermod -aG gpio,i2c,spi,video,dialout pi

# Enable I2C and SPI
sudo raspi-config nonint do_i2c 0
sudo raspi-config nonint do_spi 0

# Install NATS (optional, for standalone)
if [ "$1" = "--with-nats" ]; then
    curl -sf https://raw.githubusercontent.com/gorai/gorai/main/scripts/install-nats.sh | bash
fi

# Create robot directory
sudo mkdir -p /opt/myrobot
sudo chown pi:pi /opt/myrobot

echo "=== Setup Complete ==="
echo "Reboot recommended: sudo reboot"
```
