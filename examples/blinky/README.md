# Blinky LED Example

The classic "hello world" of hardware - blink an LED!

## What it does

- Controls a GPIO output pin
- Can be controlled via NATS messages to turn LED on/off

## Hardware Setup

1. Connect an LED to GPIO pin 17 (or any GPIO pin)
2. Use a 220-330 ohm resistor in series
3. Connect the other LED leg to ground

```
GPIO 17 ─── 220Ω ─── LED ─── GND
```

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

### 2. Run the Robot

From the gorai root directory:

```bash
./bin/gorai run examples/blinky/robot.json
```

### 3. Control the LED

In another terminal, send commands via NATS:

```bash
# Turn LED on
nats pub "gorai.blinky.led.cmd" '{"action":"set","value":true}'

# Turn LED off
nats pub "gorai.blinky.led.cmd" '{"action":"set","value":false}'
```

## Simulation Mode

If you're not running on a Raspberry Pi with GPIO, the component will run in simulation mode and log state changes instead of controlling real hardware.

## Changing the Pin

Edit `robot.json` and change the `pin` attribute:

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
