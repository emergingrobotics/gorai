# PWM Controller Example

This example demonstrates using the gorai-gsp library to control PWM outputs on an RP2040-based controller running the rp2040-pwm firmware.

## Prerequisites

- RP2040 board (e.g., Raspberry Pi Pico) flashed with rp2040-pwm firmware
- USB connection to the RP2040 board
- NATS server (optional, for integration with gorai ecosystem)

## Hardware Setup

1. Flash your RP2040 board with the rp2040-pwm firmware:
   ```bash
   cd ../../../rp2040-pwm
   make flash
   ```

2. Connect a servo to PWM channel 0 (GPIO 0, Pin 1 on Pico)

3. Note your serial port (typically `/dev/ttyACM0` on Linux)

## Building

From the gorai directory:
```bash
make build-example-pwm-controller
```

Or directly:
```bash
cd examples/pwm-controller
go build -o ../../bin/pwm-controller .
```

## Usage

### Sweep Demo (default)
Sweeps the servo back and forth through its full range:
```bash
./bin/pwm-controller
```

### Center Demo
Centers the servo and holds it there:
```bash
./bin/pwm-controller -demo center
```

### Manual Position
Sets a specific pulse width:
```bash
./bin/pwm-controller -demo manual -pulse 1200
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `-port` | `/dev/ttyACM0` | Serial port for PWM controller |
| `-baud` | `230400` | Baud rate |
| `-channel` | `0` | PWM channel to control (0-15) |
| `-demo` | `sweep` | Demo mode: `sweep`, `center`, or `manual` |
| `-pulse` | `1500` | Pulse width in µs (for manual mode) |
| `-v`, `-verbose` | `false` | Enable verbose output |

### Examples

```bash
# Control channel 1 instead of channel 0
./bin/pwm-controller -channel 1

# Use a different serial port
./bin/pwm-controller -port /dev/ttyUSB0

# Set servo to minimum position
./bin/pwm-controller -demo manual -pulse 1000

# Set servo to maximum position
./bin/pwm-controller -demo manual -pulse 2000
```

## Protocol

This example uses the Gorai Serial Protocol v2 (GSP/2) to communicate with the rp2040-pwm firmware. The protocol provides:

- Binary framing with CRC-16 error detection
- PWM channel control (0-15 channels)
- Channel enable/disable
- Failsafe protection
- Heartbeat monitoring

## Integration with Gorai

This example can be extended to:

1. Subscribe to NATS topics for remote PWM control
2. Publish PWM state to NATS for monitoring
3. Integrate with other gorai components for full robot control

## Related

- [rp2040-pwm](../../../rp2040-pwm) - Firmware for the RP2040 PWM controller
- [gorai-gsp](../../../gorai-gsp) - Gorai Serial Protocol library
