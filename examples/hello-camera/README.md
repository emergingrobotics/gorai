# Hello Camera Example

> **🚧 WORK IN PROGRESS - NOT YET FUNCTIONAL 🚧**
>
> This example is currently under development and does not work yet. The implementation is incomplete.
>
> **Working examples:** See [hello-robot](../hello-robot/) or [hello-robot-production](../hello-robot-production/) for fully functional examples.

A simple camera robot demonstrating V4L2 capture and web dashboard.

## Architecture

```
K3s Cluster (single-node)
┌─────────────────────────────────────────────────────────────┐
│  Namespace: gorai-hello-camera                              │
│                                                             │
│  ┌─────────────┐   ┌──────────────────────────────────────┐ │
│  │ nats pod    │   │ gorai-core pod                       │ │
│  │             │◄──│  ├── camera component (V4L2)         │ │
│  │ NATS server │   │  └── dashboard service (:8080)       │ │
│  └─────────────┘   └──────────────────────────────────────┘ │
│                                                             │
│  Hardware passthrough: /dev/video0                          │
└─────────────────────────────────────────────────────────────┘
```

## Prerequisites

- Raspberry Pi 5 (8GB) or equivalent with K3s installed
- NVMe SSD or USB 3.0 SSD (SD cards not supported)
- Camera at `/dev/video0`
- Go 1.22+ (for building gorai CLI)

See [K3s Installation Guide](../../specs/k3s-installation.md) for setup instructions.

## Quick Start

### 1. Verify K3s is Running

```bash
sudo k3s kubectl get nodes
# Should show: Ready
```

### 2. Deploy

```bash
# Deploy to K3s
gorai deploy hello-camera.json

# Watch deployment
gorai status hello-camera
```

### 3. Access the Dashboard

Open http://localhost:8080 in your browser to view the camera feed.

### 4. View Logs

```bash
gorai logs hello-camera -f
```

### 5. Undeploy

```bash
gorai undeploy hello-camera
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

Subscribe to view frames (from within the cluster):
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

### Pod Not Starting

```bash
# Check pod status
sudo k3s kubectl get pods -n gorai-hello-camera

# View pod logs
sudo k3s kubectl logs -n gorai-hello-camera -l app=gorai-core
```

### Device Passthrough Issues

```bash
# Verify device exists
ls -la /dev/video0

# Check permissions
sudo chmod 666 /dev/video0
```

## Next Steps

See [hello-people-detector](../hello-people-detector/) for an example that adds AI-based person detection using an external service.
