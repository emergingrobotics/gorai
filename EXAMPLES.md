# Gorai Examples

This document describes the example robots included with Gorai and how to build and run them.

## Prerequisites

Before running any example, ensure you have:

1. **Go 1.22+** installed
2. **NATS server** installed and running
3. **Gorai CLI** built

### Install NATS Server

```bash
# macOS
brew install nats-server

# Linux (Debian/Ubuntu)
sudo apt install nats-server

# Or download from https://nats.io/download/
```

### Build Gorai CLI

```bash
cd /path/to/gorai
make build
# Creates ./bin/gorai
```

### Start NATS Server

In a separate terminal:

```bash
nats-server
```

---

## Examples Overview

| Example | Description | Hardware Required | Status |
|---------|-------------|-------------------|--------|
| [blinky](#blinky) | LED blink via GPIO | Optional (has simulation mode) | Working |
| [gps-tracker](#gps-tracker) | GPS data publisher | Optional (has simulator) | Working |
| [hello-camera](#hello-camera) | V4L2 camera capture | Camera at /dev/video0 | Work in Progress |

---

## Blinky

The classic "hello world" of hardware - blink an LED!

### What It Does

- Controls a GPIO output pin
- Can be controlled via NATS messages to turn LED on/off
- Runs in simulation mode if no GPIO hardware is available

### Configuration

**File:** `examples/blinky/robot.json`

```json
{
  "robot": {
    "name": "blinky",
    "description": "Blink an LED on GPIO pin 17"
  },
  "components": [
    {
      "name": "led",
      "type": "gpio",
      "model": "output",
      "attributes": {
        "pin": 17
      }
    }
  ]
}
```

### Hardware Setup (Optional)

If running on a Raspberry Pi with GPIO:

```
GPIO 17 ─── 220Ω ─── LED(+) ─── LED(-) ─── GND
```

1. Connect an LED to GPIO pin 17
2. Use a 220-330 ohm resistor in series
3. Connect the other LED leg to ground

### Build and Run

```bash
# Build gorai CLI (if not already built)
make build

# Validate the configuration
./bin/gorai validate examples/blinky/robot.json

# Run the robot
./bin/gorai run examples/blinky/robot.json
```

### Test with NATS Commands

In another terminal, control the LED:

```bash
# Install NATS CLI (if needed)
# macOS: brew install nats-io/nats-tools/nats
# Linux: go install github.com/nats-io/natscli/nats@latest

# Turn LED on
nats pub "gorai.blinky.led.cmd" '{"action":"set","value":true}'

# Turn LED off
nats pub "gorai.blinky.led.cmd" '{"action":"set","value":false}'
```

### Simulation Mode

If you're not running on a Raspberry Pi with GPIO, the component will run in simulation mode and log state changes instead of controlling real hardware.

### Customization

To change the GPIO pin, edit `robot.json`:

```json
{
  "name": "led",
  "type": "gpio",
  "model": "output",
  "attributes": {
    "pin": 18
  }
}
```

---

## GPS Tracker

A simple GPS tracker that reads GPS data and publishes it to NATS.

### What It Does

- Reads GPS NMEA sentences from a serial port
- Publishes GPS data to NATS
- Uses built-in GPS simulator by default (no hardware required!)

### Configuration

**File:** `examples/gps-tracker/robot.json`

```json
{
  "robot": {
    "name": "gps-tracker",
    "description": "A simple GPS tracker that publishes location data"
  },
  "components": [
    {
      "name": "gps",
      "type": "serial",
      "model": "gps",
      "attributes": {
        "device": "/dev/gps-sim",
        "baud_rate": 9600
      }
    }
  ]
}
```

### Build and Run

```bash
# Build gorai CLI (if not already built)
make build

# Validate the configuration
./bin/gorai validate examples/gps-tracker/robot.json

# Run the robot (uses built-in simulator)
./bin/gorai run examples/gps-tracker/robot.json
```

### Watch GPS Data

In another terminal, subscribe to the GPS data:

```bash
# Install NATS CLI (if needed)
# macOS: brew install nats-io/nats-tools/nats
# Linux: go install github.com/nats-io/natscli/nats@latest

# Subscribe to all GPS topics
nats sub "gorai.gps-tracker.>"
```

You should see NMEA sentences like:

```
[gorai.gps-tracker.gps.nmea] {"sentence":"$GPGGA,120000.00,3749.19400,N,12225.69600,W,...","timestamp":"2026-01-01T00:00:00Z"}
```

### Using Real GPS Hardware

To use a real GPS receiver instead of the simulator:

1. Connect your GPS receiver via USB or UART

2. Find the device path:
   ```bash
   ls /dev/ttyUSB*   # USB GPS
   ls /dev/ttyAMA*   # GPIO UART (Raspberry Pi)
   ```

3. Edit `robot.json`:
   ```json
   {
     "attributes": {
       "device": "/dev/ttyUSB0",
       "baud_rate": 9600
     }
   }
   ```

4. Ensure you have permission to access the serial port:
   ```bash
   sudo usermod -a -G dialout $USER
   # Log out and back in for changes to take effect
   ```

### GPS Simulator Details

The built-in simulator (device: `/dev/gps-sim`) generates realistic GPS data:

- **Start position:** Golden Gate Bridge, San Francisco (37.8199° N, 122.4783° W)
- **Movement:** 60 mph due east (heading 90°)
- **Sentences:** GPGSV, GPRMC, GPGGA, GPGSA with proper checksums

---

## Hello Camera

A camera robot demonstrating V4L2 capture and web dashboard.

### Status

> **Work in Progress**: This example is under development and may not be fully functional. The camera component implementation is incomplete.

### What It Does

- Captures video from a V4L2 camera (USB or CSI)
- Publishes JPEG frames to NATS
- Provides a web dashboard on port 8080

### Configuration

**File:** `examples/hello-camera/hello-camera.json`

```json
{
  "robot": {
    "name": "hello-camera",
    "description": "A simple camera robot demonstrating V4L2 capture and web dashboard"
  },
  "components": [
    {
      "name": "main_camera",
      "type": "camera",
      "model": "v4l2",
      "attributes": {
        "device": "/dev/video0",
        "width": 640,
        "height": 480,
        "frame_rate": 30,
        "jpeg_quality": 80
      }
    }
  ],
  "services": [
    {
      "name": "dashboard",
      "type": "dashboard",
      "model": "web",
      "attributes": {
        "listen": ":8080"
      }
    }
  ]
}
```

### Hardware Requirements

- Camera connected at `/dev/video0`
- USB webcam or Raspberry Pi Camera Module

### Build and Run

```bash
# Build gorai CLI (if not already built)
make build

# Check if camera is available
v4l2-ctl --list-devices

# Validate the configuration
./bin/gorai validate examples/hello-camera/hello-camera.json

# Run the robot
./bin/gorai run examples/hello-camera/hello-camera.json
```

### Access the Dashboard

Open http://localhost:8080 in your browser to view the camera feed.

### Watch Camera Data via NATS

```bash
# Subscribe to camera frames
nats sub "gorai.hello-camera.main_camera.data"
```

### Troubleshooting

**Camera Not Found:**
```bash
# Check if camera is detected
v4l2-ctl --list-devices

# Add user to video group
sudo usermod -aG video $USER
# Log out and back in
```

**Permission Denied:**
```bash
# Check device permissions
ls -la /dev/video0

# Fix permissions (temporary)
sudo chmod 666 /dev/video0
```

---

## Running All Examples

### Quick Test Script

```bash
#!/bin/bash
# test-examples.sh

set -e

echo "Building gorai..."
make build

echo ""
echo "Validating all examples..."
./bin/gorai validate examples/blinky/robot.json
./bin/gorai validate examples/gps-tracker/robot.json
./bin/gorai validate examples/hello-camera/hello-camera.json

echo ""
echo "All examples validated successfully!"
```

### Common Commands

```bash
# Validate an example
./bin/gorai validate examples/<name>/robot.json

# Run an example
./bin/gorai run examples/<name>/robot.json

# List available components
./bin/gorai components

# Show version
./bin/gorai version
```

---

## Creating Your Own Robot

Use the examples as templates:

1. Copy an example directory:
   ```bash
   cp -r examples/blinky examples/my-robot
   ```

2. Edit `robot.json` with your configuration

3. Validate:
   ```bash
   ./bin/gorai validate examples/my-robot/robot.json
   ```

4. Run:
   ```bash
   ./bin/gorai run examples/my-robot/robot.json
   ```

See [specs/robot-definition-language.md](specs/robot-definition-language.md) for full RDL documentation.
