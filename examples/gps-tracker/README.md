# GPS Tracker Example

A simple GPS tracker that reads GPS data and publishes it to NATS.

## What it does

- Reads GPS NMEA sentences from a serial port (or simulator)
- Publishes GPS data to NATS
- By default uses the GPS simulator (no hardware required!)

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
./bin/gorai run examples/gps-tracker/robot.json
```

### 3. Watch GPS Data

In another terminal, subscribe to the GPS data:

```bash
# Install NATS CLI
# macOS:
brew install nats-io/nats-tools/nats

# Linux:
go install github.com/nats-io/natscli/nats@latest

# Subscribe to GPS data
nats sub "gorai.gps-tracker.>"
```

## Using Real GPS Hardware

To use a real GPS receiver:

1. Connect your GPS receiver via USB or UART
2. Find the device path:
   ```bash
   ls /dev/ttyUSB*   # USB GPS
   ls /dev/ttyAMA*   # GPIO UART (Raspberry Pi)
   ```

3. Edit `robot.json` and change the device:
   ```json
   {
     "device": "/dev/ttyUSB0",
     "baud_rate": 9600
   }
   ```

4. Make sure you have permission to access the serial port:
   ```bash
   sudo usermod -a -G dialout $USER
   # Log out and back in for changes to take effect
   ```
