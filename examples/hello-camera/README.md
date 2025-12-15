# Hello Camera Example

A simple camera robot demonstrating V4L2 capture and web dashboard.

## Architecture

```
Robot Deployment
+-------------------------------------------------------------+
|  Host System (Raspberry Pi, etc.)                           |
|                                                             |
|  systemd                                                    |
|  +-- nats-server.service           (installed natively)     |
|  +-- hello-camera.service          (gorai robot binary)     |
|                                                             |
|  Hardware                                                   |
|  +-- /dev/video0  -> camera component                       |
+-------------------------------------------------------------+
```

## Prerequisites

- Linux (Raspberry Pi OS, Ubuntu, Fedora)
- Go 1.22+ (for building gorai)
- NATS server (`sudo apt install nats-server`)
- Camera at `/dev/video0`

## Quick Start

### 1. Install NATS

```bash
sudo apt install nats-server
sudo systemctl enable --now nats-server
```

### 2. Run

```bash
# Run in foreground (development)
make run

# Or run as systemd service (production)
make run-background
```

### 3. Access the Dashboard

Open http://localhost:8080 in your browser to view the camera feed.

### 4. Check Status

```bash
make status
```

### 5. View Logs

```bash
make logs
```

### 6. Stop

```bash
make stop
```

## Configuration

The `hello-camera.json` defines:

| Component | Type | Description |
|-----------|------|-------------|
| `main_camera` | camera (v4l2) | USB/CSI camera at /dev/video0 |
| `dashboard` | service | Web UI on port 8080 |

### Camera Attributes

| Attribute | Default | Description |
|-----------|---------|-------------|
| `device` | /dev/video0 | V4L2 device path |
| `width` | 640 | Capture width |
| `height` | 480 | Capture height |
| `frame_rate` | 30 | Target FPS |
| `jpeg_quality` | 80 | JPEG compression (1-100) |

## NATS Topics

The camera publishes JPEG frames to:
```
gorai.hello-camera.main_camera.data
```

Subscribe to view frames:
```bash
nats sub "gorai.hello-camera.main_camera.data"
```

## Troubleshooting

### Camera Not Found

```bash
# Check if camera is detected
v4l2-ctl --list-devices

# Add user to video group
sudo usermod -aG video $USER
# Log out and back in
```

### Permission Denied

```bash
# Check device permissions
ls -la /dev/video0

# Fix permissions (temporary)
sudo chmod 666 /dev/video0
```

## Next Steps

See [hello-people-detector](../hello-people-detector/) for an example that adds AI-based person detection using an external service.
