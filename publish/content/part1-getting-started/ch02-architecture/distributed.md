## 2.3 Distributed Architecture

GoRAI is distributed by default. Even a single-board robot runs multiple nodes communicating through NATS. This section explains why and how.

### Why Distributed Matters for Robotics

Robots are inherently parallel systems:
- Sensors produce data continuously
- Actuators execute commands asynchronously
- Processing happens at different rates (vision at 30Hz, IMU at 1000Hz)
- Failures in one subsystem shouldn't cascade

Traditional monolithic architectures fight this reality. A single-threaded main loop serializes inherently parallel work. Shared memory creates coupling and race conditions. A crash anywhere stops everything.

Distributed architecture embraces the reality:
- Each node runs independently at its natural rate
- Message passing provides clean, typed interfaces between subsystems
- Failure isolation protects the system
- Horizontal scaling is natural—add nodes, add capability

### Primary Compute Responsibilities

The primary compute board (typically the most powerful SBC) usually handles:

**NATS Server**: The message broker runs here, accessible to all nodes:
```bash
# Start NATS server (often via scripts/start.sh)
nats-server -js  # -js enables JetStream
```

**High-Level Logic**: Navigation planning, behavior trees, mission management:
```go
// Brain node coordinates behavior
brain, _ := node.New("brain", node.WithNATS(natsURL))

// Subscribe to sensor fusion output
sub.New(brain, "gorai.perception.state", func(state *State) {
    decision := planner.Decide(state)
    cmdPub.Publish(ctx, decision.Commands)
})
```

**User Interfaces**: Web dashboards, API endpoints, remote control:
```go
// HTTP server for monitoring
http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
    status := collectStatus()
    json.NewEncoder(w).Encode(status)
})
```

**Data Logging**: Recording messages for debugging and replay:
```bash
# NATS CLI can record all traffic
nats sub "gorai.>" --dump messages.log
```

### Secondary Node Use Cases

Secondary nodes handle specialized, isolated tasks:

**Vision Processing**: Cameras generate significant data. Processing on a dedicated node (especially one with GPU/NPU) keeps the main compute responsive:

```go
// Vision node on Jetson/RK3588
vision, _ := node.New("vision", node.WithNATS(natsURL))
camera := setupCamera()
detector := loadModel()

for frame := range camera.Frames() {
    detections := detector.Detect(frame)
    detPub.Publish(ctx, detections)  // Only send results, not full images
}
```

**Sensor Fusion**: Combine IMU, encoders, and GPS into coherent state estimates:

```go
// Runs at high frequency, isolated from other processing
fusion, _ := node.New("fusion", node.WithNATS(natsURL))

// Subscribe to raw sensors
sub.New(fusion, "gorai.sensors.imu.data", imuHandler)
sub.New(fusion, "gorai.sensors.encoders.data", encoderHandler)
sub.New(fusion, "gorai.sensors.gps.data", gpsHandler)

// Publish fused state at consistent rate
ticker := time.NewTicker(10 * time.Millisecond) // 100Hz
for range ticker.C {
    state := filter.Update()
    statePub.Publish(ctx, state)
}
```

**Isolated Control Loops**: Arm manipulation, precise motor control:

```go
// Arm controller with dedicated timing
arm, _ := node.New("arm_controller", node.WithNATS(natsURL))

// High-frequency servo loop
for {
    joints := readJointStates()
    commands := controller.Compute(target, joints)
    applyCommands(commands)
    time.Sleep(time.Millisecond) // 1kHz loop
}
```

### Serial Gateway Pattern for Microcontrollers

TinyGo runs on microcontrollers (RP2040, ESP32) but can't connect directly to NATS. The serial gateway pattern bridges this gap:

```
┌──────────────────────────┐          ┌──────────────────────────┐
│     Linux Board          │          │     Microcontroller      │
│                          │          │     (TinyGo)             │
│  ┌────────────────────┐  │  Serial  │  ┌────────────────────┐  │
│  │   Serial Gateway   │◄─┼──────────┼──│   Motor Driver     │  │
│  │   (Go process)     │  │   UART   │  │   PWM/Encoder      │  │
│  └─────────┬──────────┘  │          │  └────────────────────┘  │
│            │             │          │                          │
│            ▼ NATS        │          └──────────────────────────┘
│  ┌────────────────────┐  │
│  │    Other Nodes     │  │
│  └────────────────────┘  │
└──────────────────────────┘
```

The gateway translates between NATS messages and a compact serial protocol:

```go
// Gateway code (runs on Linux)
gateway, _ := node.New("motor_gateway", node.WithNATS(natsURL))
serial, _ := openSerial("/dev/ttyUSB0", 115200)

// NATS to Serial
sub.New(gateway, "gorai.motors.+.command", func(cmd *MotorCommand) {
    packet := encodeCommand(cmd)
    serial.Write(packet)
})

// Serial to NATS
go func() {
    for {
        packet := readPacket(serial)
        feedback := decodeFeedback(packet)
        fbPub.Publish(ctx, feedback)
    }
}()
```

The microcontroller side handles real-time control:

```go
// TinyGo code on microcontroller
func main() {
    uart := machine.UART0
    uart.Configure(machine.UARTConfig{BaudRate: 115200})

    motor := setupMotor()
    encoder := setupEncoder()

    for {
        if uart.Buffered() > 0 {
            cmd := readCommand(uart)
            motor.SetPower(cmd.Power)
        }

        // Send encoder feedback
        position := encoder.Read()
        sendFeedback(uart, position)

        time.Sleep(time.Millisecond)
    }
}
```

This pattern gives you:
- Real-time control on dedicated hardware
- NATS integration without microcontroller networking complexity
- Clean separation between real-time and non-real-time code
