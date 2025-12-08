## 3.6 The NATS CLI

The `nats` command-line tool is indispensable for GoRAI development. It lets you observe, debug, and interact with the message bus directly.

### Installation

```bash
# macOS
brew install nats-io/nats-tools/nats

# Linux (via go install)
go install github.com/nats-io/natscli/nats@latest

# Or download from releases
# https://github.com/nats-io/natscli/releases
```

Verify installation:
```bash
nats --version
```

### Basic Commands

**Subscribe to messages**:
```bash
# All messages
nats sub ">"

# All sensor messages
nats sub "gorai.sensors.>"

# Specific topic
nats sub "gorai.sensors.temp.data"

# With timestamps
nats sub "gorai.>" --raw
```

**Publish messages**:
```bash
# Simple text
nats pub "test.topic" "hello world"

# JSON
nats pub "gorai.test" '{"value": 42}'

# From file
nats pub "gorai.config" --file config.json
```

**Request/Reply**:
```bash
# Send request, wait for reply
nats request "gorai.services.echo" "ping" --timeout 5s
```

### Monitoring with `nats server`

Check server status:
```bash
# Server info
nats server info

# Connection list
nats server connections

# Request counts
nats server report connections
```

Real-time monitoring:
```bash
# Watch message rates
nats server report accounts --top

# Stream activity
nats stream report
```

### JetStream Commands

**Stream management**:
```bash
# List streams
nats stream list

# Stream info
nats stream info SENSOR_DATA

# View messages in stream
nats stream view SENSOR_DATA

# Purge (delete all messages)
nats stream purge SENSOR_DATA
```

**Consumer management**:
```bash
# List consumers
nats consumer list SENSOR_DATA

# Consumer info (shows lag, pending, etc.)
nats consumer info SENSOR_DATA my_consumer

# Get next message
nats consumer next SENSOR_DATA my_consumer
```

### Debugging Robot Communication

**Watch all traffic during development**:
```bash
# Terminal 1: Watch everything
nats sub ">" --raw

# Terminal 2: Run your robot
go run ./examples/hello-sensor
```

**Filter for specific patterns**:
```bash
# Only motor commands
nats sub "gorai.motors.*.command"

# Only errors/warnings (if you publish them)
nats sub "gorai.*.error" "gorai.*.warn"
```

**Interactive testing**:
```bash
# Test a motor service manually
nats request "gorai.motors.left.set_power" \
    '{"power": 0.5}' \
    --timeout 1s

# Simulate sensor data
nats pub "gorai.sensors.fake.data" \
    '{"temperature": 42.5}'
```

**Measure latency**:
```bash
# Round-trip time to server
nats rtt

# Latency distribution
nats bench "test.latency" --pub 1000 --sub 1 --size 256
```

### Useful One-Liners

```bash
# Count messages per second on a topic
nats sub "gorai.sensors.imu.data" --count 1000 2>&1 | tail -1

# Dump last N messages from a stream
nats stream view SENSOR_DATA --last 10

# Create a quick test stream
nats stream add TEST --subjects "test.>" --storage memory

# Follow logs with pretty JSON
nats sub "gorai.logs.>" | jq .

# Export stream contents
nats stream view SENSOR_DATA --json > data.jsonl
```

*Cross-reference: Chapter 9 uses these commands to observe the hello-sensor example.*

---

With a solid understanding of NATS—its patterns, QoS levels, JetStream features, and debugging tools—you're ready to explore how GoRAI uses these capabilities for specific component types. Chapter 4 begins with sensors.
