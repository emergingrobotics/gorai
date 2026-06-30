# Hello Camera Example

> **Work in Progress**: This example is under development and may not be fully functional. The camera component implementation is incomplete.

A camera robot demonstrating V4L2 capture and web dashboard.

## What it does

- Captures video from a V4L2 camera (USB or CSI)
- Publishes JPEG frames to NATS
- Provides a web dashboard on port 10101

## Prerequisites

- Camera connected at `/dev/video0`
- USB webcam or Raspberry Pi Camera Module
- NATS server running

## Running

### 1. Start NATS Server

```bash
# Install NATS server
# macOS:
brew install nats-server

# Linux:
sudo apt install nats-server

# Start the server
nats-server
```

### 2. Check Camera

```bash
# Verify camera is available
v4l2-ctl --list-devices

# Should show something like:
# USB Camera (usb-0000:00:14.0-1):
#     /dev/video0
```

### 3. Run the Robot

From the gorai root directory:

```bash
./bin/gorai run examples/hello-camera/hello-camera.json
```

### 4. Access the Dashboard

Open http://localhost:10101 in your browser to view the camera feed.

### 5. Watch Camera Data via NATS

```bash
nats sub "gorai.hello-camera.main_camera.data"
```

## Configuration

The `hello-camera.json` defines:

| Component | Type | Description |
|-----------|------|-------------|
| `main_camera` | camera (v4l2) | USB/CSI camera at /dev/video0 |
| `dashboard` | service | Web UI on port 10101 |

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
# Verify device exists
ls -la /dev/video0

# Check permissions (temporary fix)
sudo chmod 666 /dev/video0
```
