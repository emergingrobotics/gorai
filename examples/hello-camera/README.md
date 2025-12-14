# Hello Camera Example

A camera robot with person detection using Hailo NPU, demonstrating multi-container deployment with systemd.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  systemd (user mode)                                            │
│  ├── hello-camera-nats.service                                  │
│  ├── hello-camera-gorai-core.service                            │
│  └── hello-camera-gorai-hailo.service                           │
│                                                                  │
│  Containers                                                      │
│  ├── hello-camera-nats       (NATS messaging)                   │
│  ├── hello-camera-gorai-core (camera + dashboard)               │
│  └── hello-camera-gorai-hailo (person detection)                │
│                                                                  │
│  Hardware                                                        │
│  ├── /dev/video0  → gorai-core container                        │
│  └── /dev/hailo0  → gorai-hailo container                       │
└─────────────────────────────────────────────────────────────────┘
```

## Prerequisites

- Linux (Raspberry Pi OS, Ubuntu, Fedora)
- [Podman](https://podman.io/) (any version)
- systemd with user session support
- Camera at `/dev/video0`
- Hailo NPU at `/dev/hailo0` (for ML inference)

### Enable User Lingering

For services to run without login:

```bash
loginctl enable-linger $USER
```

## Quick Start

### 1. Build Container Images

```bash
# Build the core and hailo container images
gorai build --config hello-camera.json
```

This generates systemd service files in `.gorai/` and builds the container images.

### 2. Start the Robot

```bash
# Deploy and start all services
gorai start --config hello-camera.json

# Or enable auto-start at boot
gorai start --config hello-camera.json --enable
```

### 3. Check Status

```bash
gorai status --config hello-camera.json
```

Expected output:
```
Robot: hello-camera

CONTAINERS
SERVICE                        ACTIVE       STATUS
───────────────────────────────────────────────────────────
hello-camera-nats.service      active       running
hello-camera-gorai-core.service active      running
hello-camera-gorai-hailo.service active     running
```

### 4. View Logs

```bash
# Follow all robot logs
gorai logs --config hello-camera.json -f

# View specific container
gorai logs --config hello-camera.json --container gorai-core -f
```

### 5. Access the Dashboard

Open http://localhost:8080 in your browser to view the camera feed and detection results.

### 6. Stop the Robot

```bash
gorai stop --config hello-camera.json
```

## Generated Service Files

The `.gorai/` directory contains the generated systemd service files:

| File | Purpose |
|------|---------|
| `hello-camera-nats.service` | NATS messaging server |
| `hello-camera-gorai-core.service` | Camera capture and web dashboard |
| `hello-camera-gorai-hailo.service` | Hailo NPU person detection |

### Installation Location

When you run `gorai start`, these files are copied to:
```
~/.config/systemd/user/
```

## Manual Deployment

If you prefer to manage services directly with systemctl:

### Install Service Files

```bash
# Copy service files to user systemd directory
mkdir -p ~/.config/systemd/user
cp .gorai/*.service ~/.config/systemd/user/

# Reload systemd to pick up new units
systemctl --user daemon-reload
```

### Start Services

```bash
# Start all robot services (dependencies start automatically)
systemctl --user start hello-camera-nats.service
systemctl --user start hello-camera-gorai-core.service
systemctl --user start hello-camera-gorai-hailo.service
```

### Enable Auto-Start

```bash
systemctl --user enable hello-camera-nats.service
systemctl --user enable hello-camera-gorai-core.service
systemctl --user enable hello-camera-gorai-hailo.service
```

### View Logs

```bash
# All robot services
journalctl --user -u "hello-camera-*" -f

# Specific service
journalctl --user -u hello-camera-gorai-core.service -f
```

### Stop Services

```bash
systemctl --user stop hello-camera-gorai-hailo.service
systemctl --user stop hello-camera-gorai-core.service
systemctl --user stop hello-camera-nats.service
```

## Container Details

### NATS Server (`hello-camera-nats`)

- Image: `docker.io/nats:2.10-alpine`
- Ports: 4222 (clients), 8222 (monitoring)
- Network alias: `nats`

### Core Container (`hello-camera-gorai-core`)

- Image: `localhost/hello-camera-core:latest`
- Built from: `Containerfile.core`
- Components: `main_camera` (V4L2 camera)
- Services: `dashboard` (web UI on port 8080)
- Devices: `/dev/video0`

### Hailo Container (`hello-camera-gorai-hailo`)

- Image: `localhost/hello-camera-hailo:latest`
- Built from: `services/object-detection/Containerfile`
- Services: `person_detector` (YOLOX model)
- Devices: `/dev/hailo0`
- Memory limit: 1GB

## Updating

### Rebuild and Restart

```bash
gorai stop --config hello-camera.json
gorai build --config hello-camera.json
gorai start --config hello-camera.json
```

## Troubleshooting

### Service Won't Start

Check the service status and logs:

```bash
systemctl --user status hello-camera-gorai-core.service
journalctl --user -u hello-camera-gorai-core.service --no-pager -n 50
```

### Container Not Found

Verify images are built:

```bash
podman images | grep hello-camera
```

### Permission Denied on Device

Add user to required groups:

```bash
# For camera
sudo usermod -aG video $USER

# For Hailo NPU
sudo usermod -aG hailo $USER
```

Log out and back in for changes to take effect.

### View Generated Service File

```bash
cat ~/.config/systemd/user/hello-camera-nats.service
```
