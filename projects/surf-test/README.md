# Surf Test - Distributed Robot Control

A two-robot system demonstrating distributed control using NATS messaging.

## Architecture

```
┌─────────────────────────────────────────┐
│            Ground Station (PC)          │
│                                         │
│  ┌─────────────┐    ┌────────────────┐  │
│  │  Keyboard   │───►│   Keyboard     │  │
│  │  (evdev)    │    │   Publisher    │  │
│  └─────────────┘    └───────┬────────┘  │
│                             │           │
│  ┌─────────────────────────┐│           │
│  │ Dashboard (:8081)       ││           │
│  │ - View camera feed      ││           │
│  │ - Monitor status        ││           │
│  └─────────────────────────┘│           │
└─────────────────────────────┼───────────┘
                              │ NATS
                              │ gorai.ground-station.keyboard.events
                              ▼
┌─────────────────────────────────────────┐
│         Main Robot (Raspberry Pi 5)     │
│                                         │
│  ┌─────────────┐    ┌────────────────┐  │
│  │   Remote    │◄───│   NATS Server  │  │
│  │  Keyboard   │    │   (:4222)      │  │
│  └──────┬──────┘    └────────────────┘  │
│         │                               │
│         ▼                               │
│  ┌─────────────┐    ┌────────────────┐  │
│  │   Motor     │───►│  Pan Servo     │  │
│  │ Controller  │    │  (GPIO 18)     │  │
│  │             │───►│  Tilt Servo    │  │
│  │             │    │  (GPIO 19)     │  │
│  └─────────────┘    └────────────────┘  │
│                                         │
│  ┌─────────────┐    ┌────────────────┐  │
│  │   Camera    │───►│   NATS Pub     │  │
│  │  (V4L2)     │    │   (to GS)      │  │
│  └─────────────┘    └────────────────┘  │
│                                         │
│  ┌─────────────────────────────────┐    │
│  │ Dashboard (:8080)               │    │
│  │ - Local camera view             │    │
│  │ - PWM status                    │    │
│  └─────────────────────────────────┘    │
└─────────────────────────────────────────┘
```

## Hardware Setup

### Main Robot (Raspberry Pi 5)

| Component | Connection |
|-----------|------------|
| Pan Servo | GPIO 18 (PWM0) |
| Tilt Servo | GPIO 19 (PWM1) |
| USB Camera | USB port |

### Ground Station (PC)

| Component | Connection |
|-----------|------------|
| USB Keyboard | USB port |

## Control Keys

| Key | Action |
|-----|--------|
| W | Tilt Up (+5°) |
| S | Tilt Down (-5°) |
| A | Pan Left (-5°) |
| D | Pan Right (+5°) |

## Servo Limits

| Servo | Min Angle | Max Angle | Initial |
|-------|-----------|-----------|---------|
| Pan | -90° | +90° | 0° |
| Tilt | -45° | +45° | 0° |

## Configuration

### Main Robot (`main-robot.json`)

- NATS server listens on `0.0.0.0:4222`
- Camera publishes at 15 FPS (captures at 30 FPS)
- Remote keyboard monitors `gorai.ground-station.keyboard.events`
- Auto-releases keys after 3 seconds of no events (safety)

### Ground Station (`ground-station.json`)

- Connects to NATS at `RASPBERRY_PI_IP:4222`
- **Important**: Replace `RASPBERRY_PI_IP` with your Pi's actual IP address
- Dashboard on port 8081 (to avoid conflict if testing locally)

## Running

### 1. Start the Main Robot (on Raspberry Pi)

```bash
# Find your Pi's IP address
hostname -I

# Run with logging (recommended)
./projects/surf-test/run-main-robot.sh

# Or run directly without file logging
gorai run projects/surf-test/main-robot.json
```

### 2. Start the Ground Station (on PC)

```bash
# Edit ground-station.json and replace RASPBERRY_PI_IP with the actual IP
# e.g., "nats://192.168.1.100:4222"

# Find your keyboard device
ls -la /dev/input/by-id/
# Look for the keyboard device (e.g., usb-XXXX-event-kbd)
# Update the "device" field in ground-station.json if needed

# Run with logging (recommended)
./projects/surf-test/run-ground-station.sh

# Or run directly without file logging
gorai run projects/surf-test/ground-station.json
```

### 3. Access Dashboards

- **Main Robot Dashboard**: `http://RASPBERRY_PI_IP:8080`
- **Ground Station Dashboard**: `http://localhost:8081`

## Logging

### Log Levels

Both robots are configured with comprehensive logging:

| Component | Log Level | What's Logged |
|-----------|-----------|---------------|
| **Global** | `debug` | All debug-level messages |
| `remote_keyboard` | `trace` | Every key event received from NATS |
| `motor_controller` | `trace` | Every motor command and state change |
| `keyboard_publisher` | `trace` | Every key event published to NATS |
| `main_keyboard` | `debug` | Key events and state changes |
| `main_camera` | `debug` | Frame captures and publishes |
| `pan_servo` / `tilt_servo` | `debug` | PWM pulse changes |

### Log Files

When using the run scripts, logs are saved to `logs/` directory:

```
projects/surf-test/logs/
├── main-robot-20260126-143052.log
├── main-robot-20260126-151230.log
├── ground-station-20260126-143055.log
└── ...
```

### Log Management

Use the `view-logs.sh` utility:

```bash
# List all log files
./view-logs.sh list

# View latest main robot log
./view-logs.sh latest main-robot

# Tail ground station log in real-time
./view-logs.sh tail ground-station

# Search all logs for errors
./view-logs.sh search "error"

# Clean up logs older than 3 days
./view-logs.sh clean 3

# Check log directory size
./view-logs.sh size
```

### Log Format

Logs use human-readable text format with timestamps:

```
2026-01-26T14:30:52.123Z DEBUG [remote_keyboard] received key event key=W pressed=true
2026-01-26T14:30:52.124Z TRACE [motor_controller] processing key event key=W motor=tilt action=forward
2026-01-26T14:30:52.125Z DEBUG [tilt_servo] setting pulse pulse_us=1575 angle=15.0
```

### Reducing Log Verbosity

For production, edit the JSON configs:

1. Change global level to `info`:
   ```json
   "log": {
     "level": "info",
     ...
   }
   ```

2. Remove or change per-component `log_level` to `info` or `warn`

## Testing Locally

For local testing (both on same machine):

1. Start a local NATS server:
   ```bash
   nats-server
   ```

2. Update both configs to use `nats://localhost:4222`

3. Run main robot in one terminal:
   ```bash
   gorai run projects/surf-test/main-robot.json
   ```

4. Run ground station in another terminal:
   ```bash
   gorai run projects/surf-test/ground-station.json
   ```

## Troubleshooting

### Finding Keyboard Device

```bash
# List input devices
cat /proc/bus/input/devices

# Or use evtest
sudo apt install evtest
sudo evtest
```

### NATS Connection Issues

```bash
# Check if NATS is running
nats server ping nats://RASPBERRY_PI_IP:4222

# Monitor NATS traffic
nats sub "gorai.>" --server nats://RASPBERRY_PI_IP:4222
```

### Camera Not Working

```bash
# Check camera device
v4l2-ctl --list-devices

# Test camera capture
v4l2-ctl --device=/dev/video0 --all
```

### PWM/Servo Issues

```bash
# Check GPIO chip access
ls -la /dev/gpiochip*

# User may need to be in gpio group
sudo usermod -a -G gpio $USER
```

## Safety Features

1. **Auto-release on disconnect**: If the ground station loses connection for >3 seconds, all keys are automatically released, stopping any servo movement.

2. **Servo limits**: Both servos have defined min/max angles to prevent mechanical damage.

3. **No grab mode**: The keyboard is not exclusively grabbed, so you can still use it for other applications.

