# Hello Camera Example

A camera robot with person detection using Hailo NPU, demonstrating multi-container deployment with Quadlet and systemd.

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
- [Podman](https://podman.io/) 4.4+ (for Quadlet support)
- systemd with user session support
- Camera at `/dev/video0`
- Hailo NPU at `/dev/hailo0` (for ML inference)

### Verify Podman Version

```bash
podman --version
# Must be 4.4.0 or higher
```

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

This generates Quadlet files in `.gorai/` and builds the container images.

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
● hello-camera-nats.service - Gorai container hello-camera-nats
     Loaded: loaded
     Active: active (running)

● hello-camera-gorai-core.service - Gorai container hello-camera-gorai-core
     Loaded: loaded
     Active: active (running)

● hello-camera-gorai-hailo.service - Gorai container hello-camera-gorai-hailo
     Loaded: loaded
     Active: active (running)
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

## Quadlet Files

The `.gorai/` directory contains the generated Quadlet unit files:

| File | Purpose |
|------|---------|
| `hello-camera-network.network` | Bridge network for container communication |
| `hello-camera-nats.container` | NATS messaging server |
| `hello-camera-gorai-core.container` | Camera capture and web dashboard |
| `hello-camera-gorai-hailo.container` | Hailo NPU person detection |

### Installation Location

When you run `gorai start`, these files are copied to:
```
~/.config/containers/systemd/
```

systemd automatically generates `.service` files from the Quadlet unit files.

## Manual Deployment

If you prefer to manage services directly with systemctl:

### Install Quadlet Files

```bash
# Copy Quadlet files to user systemd directory
mkdir -p ~/.config/containers/systemd
cp .gorai/*.container ~/.config/containers/systemd/
cp .gorai/*.network ~/.config/containers/systemd/

# Reload systemd to pick up new units
systemctl --user daemon-reload
```

### Start Services

```bash
# Start all robot services
systemctl --user start hello-camera-nats.service
systemctl --user start hello-camera-gorai-core.service
systemctl --user start hello-camera-gorai-hailo.service

# Or start just the top-level service (dependencies start automatically)
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
- Health check: HTTP on port 8222

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

### Auto-Update (Registry Images)

For images with `AutoUpdate=registry`, use:

```bash
podman auto-update
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

### Quadlet Files Not Loading

Verify Podman version supports Quadlet:

```bash
podman --version  # Must be 4.4+
```

Check for syntax errors:

```bash
/usr/libexec/podman/quadlet --dryrun --user
```
