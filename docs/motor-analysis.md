# Prosumer Robotics Motors: A Comprehensive Analysis

## Overview

This document analyzes the most common motor types used in prosumer robotics, including land-based and waterborne applications. For each motor type, we examine the control parameters, controller interfaces, and communication protocols relevant to Gorai integration.

## Motor Categories

### 1. Brushless DC Motors (BLDC)

**Use Cases:** Drone propellers, wheeled robots, robotic arms, gimbals

BLDC motors are the workhorses of modern robotics due to their high efficiency, long lifespan, and excellent power-to-weight ratio.

#### Control Methods

| Method | Description | Precision | Complexity |
|--------|-------------|-----------|------------|
| **Trapezoidal (6-step)** | Simple commutation using Hall sensors | Low | Low |
| **Sinusoidal** | Smoother commutation, less torque ripple | Medium | Medium |
| **FOC (Field Oriented Control)** | Vector control for maximum efficiency | High | High |

#### Key Parameters

```
Motor Parameters:
├── KV Rating (RPM per volt)
├── Pole Pairs (typically 7-14)
├── Phase Resistance (mΩ)
├── Phase Inductance (µH)
├── Maximum Current (A)
└── Torque Constant (Nm/A)

Control Parameters:
├── PWM Frequency: 8-32 kHz (typical)
├── Control Loop Rate: 10-40 kHz (FOC)
├── Position: encoder counts or electrical angle
├── Velocity: RPM or rad/s
├── Current/Torque: Amps (Iq for FOC)
└── Bus Voltage: typically 12-48V
```

#### Common Controllers

**ODrive**
- Protocol: CAN bus, USB, UART, ASCII/binary
- Features: Dual motor, FOC, encoder support, trajectory planning
- Interface: Position, velocity, or torque mode
- CAN Protocol Example:
  ```
  Axis 0 Set Position: ID 0x00C (node_id << 5 | 0x0C)
  Data: [position (float32), velocity_ff (int16), torque_ff (int16)]
  ```

**VESC (Vedder ESC)**
- Protocol: UART, CAN, PPM, ADC
- Features: Open source, configurable, data logging
- Commands: Duty cycle, current, RPM, position

**SimpleFOC**
- Protocol: Serial commands, I2C, SPI
- Features: Arduino-compatible, educational
- Open source library for custom controllers

---

### 2. Servo Motors

**Use Cases:** Robotic arms, pan-tilt mechanisms, legged robots, RC applications

Servo motors integrate motor, gearbox, and feedback in a single package.

#### Categories

| Type | Feedback | Protocol | Typical Use |
|------|----------|----------|-------------|
| **RC/Hobby Servo** | Potentiometer | PWM | Simple positioning |
| **Digital Servo** | Potentiometer | PWM | Faster response |
| **Smart Servo** | Magnetic encoder | Serial bus | Robotics arms |
| **Industrial Servo** | Absolute encoder | EtherCAT/CANopen | High precision |

#### RC Servo PWM Protocol

```
Standard PWM Servo:
├── Signal Frequency: 50 Hz (20ms period)
├── Pulse Width: 1000-2000 µs
│   ├── 1000 µs = -90° (or full CCW)
│   ├── 1500 µs = center (0°)
│   └── 2000 µs = +90° (or full CW)
├── Neutral: 1500 µs
└── Signal Voltage: 3.3V or 5V logic

Extended Range (some servos):
├── 500-2500 µs for 180° travel
└── Continuous rotation: speed control instead of position
```

#### Smart Servo Protocols

**Dynamixel (Robotis)**
- Protocol: TTL half-duplex serial (1Mbps)
- Daisy-chainable with unique IDs
- Control table registers for all parameters
- Read/write position, velocity, torque, temperature
- Packet format:
  ```
  [0xFF][0xFF][ID][Length][Instruction][Params...][Checksum]
  ```

**LX-16A / LewanSoul**
- Protocol: Serial bus (115200 baud default)
- Features: Position feedback, ID configuration
- Commands: Move, read position, set ID, set limits

**Feetech SCS/STS Series**
- Protocol: Half-duplex serial
- Compatible pinout with Dynamixel
- Lower cost alternative

#### Key Parameters

```
Servo Parameters:
├── Torque Rating (kg·cm or Nm)
├── Speed (sec/60° at no load)
├── Operating Voltage (4.8V-8.4V typical)
├── Rotation Range (180°, 270°, or continuous)
├── Dead Band (µs, for PWM servos)
└── Gear Type (plastic, metal, titanium)

Control Parameters:
├── Target Position (degrees or raw units)
├── Moving Speed (for smart servos)
├── Torque Limit (for smart servos)
├── Compliance/PID gains (for smart servos)
└── Temperature/Voltage limits
```

---

### 3. Stepper Motors

**Use Cases:** 3D printers, CNC, camera sliders, precision positioning

Stepper motors provide open-loop position control through discrete steps.

#### Types

| Type | Steps/Rev | Torque | Speed | Cost |
|------|-----------|--------|-------|------|
| **NEMA 17** | 200 (1.8°) | 0.3-0.7 Nm | Medium | Low |
| **NEMA 23** | 200 (1.8°) | 0.9-3.0 Nm | Medium | Medium |
| **NEMA 34** | 200 (1.8°) | 3.0-12 Nm | Lower | Higher |

#### Control Methods

**Step/Direction Interface**
```
Signals:
├── STEP: Rising edge = one step
├── DIR: High/Low = direction
├── ENABLE: Active low (usually)
└── FAULT: Driver error output

Microstepping:
├── Full step: 200 steps/rev
├── 1/2 step: 400 steps/rev
├── 1/4 step: 800 steps/rev
├── 1/8 step: 1600 steps/rev
├── 1/16 step: 3200 steps/rev
└── 1/256 step: 51200 steps/rev (TMC drivers)
```

#### Common Drivers

**A4988**
- Interface: Step/Direction
- Microstepping: Up to 1/16
- Current: Up to 2A
- Simple, low cost

**DRV8825**
- Interface: Step/Direction
- Microstepping: Up to 1/32
- Current: Up to 2.5A
- Drop-in A4988 replacement

**TMC2209**
- Interface: Step/Direction + UART
- Microstepping: Up to 1/256
- Features: StealthChop (silent), StallGuard (sensorless homing)
- UART commands for tuning and diagnostics

**TMC5160**
- Interface: SPI or Step/Direction
- Features: Motion controller built-in, higher current
- Can run autonomous motion profiles

#### Key Parameters

```
Motor Parameters:
├── Steps per Revolution (200 typical)
├── Holding Torque (Nm)
├── Rated Current (A per phase)
├── Phase Resistance (Ω)
├── Phase Inductance (mH)
└── Rotor Inertia (g·cm²)

Control Parameters:
├── Step Pulse Rate (Hz, determines speed)
├── Acceleration Ramp (steps/s²)
├── Microstepping Resolution
├── Run Current (% of rated)
├── Hold Current (% of run)
└── Stall Detection Threshold (TMC)
```

---

### 4. Underwater Thrusters

**Use Cases:** ROVs, AUVs, underwater drones, boat propulsion

Underwater thrusters are specialized BLDC motors designed for marine environments.

#### Common Products

**Blue Robotics T100/T200**
- Type: Brushless DC, waterproof
- Control: Standard RC PWM via ESC
- Thrust: T100 (2.36 kgf), T200 (5.1 kgf forward)
- Bidirectional with deadband at center

**M200 Motor**
- Higher thrust variant
- Same control interface

#### Control Interface

```
Blue Robotics Basic ESC:
├── PWM Frequency: 50-400 Hz
├── Signal Range: 1100-1900 µs
│   ├── 1100 µs = Full reverse
│   ├── 1500 µs = Stopped (deadband ±25 µs)
│   └── 1900 µs = Full forward
├── Deadband: 1475-1525 µs
└── Initialization: Send 1500 µs for 7 seconds

Advanced Control:
├── I2C interface (some ESCs)
├── CAN bus (BlueESC Rev1)
└── Serial protocol (custom firmware)
```

#### Key Parameters

```
Thruster Parameters:
├── Maximum Thrust (kgf, forward/reverse)
├── Operating Voltage (6-20V typical)
├── Maximum Current (varies with load)
├── Depth Rating (meters)
├── Prop Diameter (mm)
└── Bollard Pull Curve (thrust vs PWM)

Control Parameters:
├── PWM Pulse Width (µs)
├── Deadband Range (µs)
├── Ramp Rate (for smooth transitions)
├── Temperature Limits
└── Current Monitoring
```

#### ESC Calibration

Most RC ESCs require calibration:
```
1. Send maximum signal (2000 µs)
2. Power on ESC (wait for tones)
3. Send minimum signal (1000 µs)
4. Wait for confirmation tones
5. ESC now calibrated to your PWM range
```

For bidirectional ESCs:
```
1. Many require programming via programming card
2. Set to "forward/reverse" mode
3. Center point becomes stop
4. May need to disable braking for underwater use
```

---

### 5. Linear Actuators

**Use Cases:** Adjustable mechanisms, hatches, deployable systems

#### Types

| Type | Speed | Precision | Force |
|------|-------|-----------|-------|
| **Lead Screw** | Slow | High | High |
| **Ball Screw** | Medium | Very High | High |
| **Belt Drive** | Fast | Medium | Medium |
| **Pneumatic** | Very Fast | Low | Variable |

#### Control

Most linear actuators use one of the motor types above internally:
- DC motor with encoder for position feedback
- Stepper motor for open-loop positioning
- Servo motor for closed-loop control

---

## Controller Communication Protocols

### PWM (Pulse Width Modulation)

```
Applications: RC servos, ESCs, simple motor drivers
Signal: 3.3V or 5V digital
Frequency: 50-400 Hz (servo), 1-32 kHz (motor driver)

Pros:
├── Universal support
├── Simple interface
├── One wire per device
└── No addressing needed

Cons:
├── No feedback channel
├── One signal per device
├── Limited precision
└── Analog interpretation varies
```

### Step/Direction

```
Applications: Stepper motors
Signals: STEP, DIR, ENABLE (3 wires minimum)

Pros:
├── Simple pulse counting
├── Open-loop position control
├── Standard interface
└── Easy acceleration control

Cons:
├── No feedback (open loop)
├── Multiple wires per motor
├── Can lose steps under load
└── Limited speed range
```

### Serial (UART)

```
Applications: Smart servos, motor controllers
Baud Rates: 9600 to 1Mbps
Modes: Full duplex, half duplex (bus)

Protocols:
├── Dynamixel: Half-duplex, packet-based
├── RoboClaw: Packet serial or simple serial
├── ODrive: ASCII commands or binary
└── Custom: Many variations

Pros:
├── Bidirectional communication
├── Rich command set
├── Daisy-chaining possible
└── Feedback included

Cons:
├── Wiring complexity
├── Protocol variations
├── Half-duplex timing sensitive
└── Limited bus length
```

### CAN Bus

```
Applications: Industrial, automotive, advanced robotics
Speed: 125 kbps to 1 Mbps
Standard: CAN 2.0A/B, CANopen, etc.

Example (ODrive):
├── Message ID = (node_id << 5) | command_id
├── Heartbeat: ID 0x001, 4 bytes
├── Set Position: ID 0x00C, 8 bytes
└── Get Encoder: ID 0x009, request/response

Pros:
├── Robust differential signaling
├── Multi-master bus
├── Error detection built-in
├── Long cable runs (100m+)
└── Industry standard

Cons:
├── Requires transceiver chip
├── More complex setup
├── Protocol overhead
└── Limited bandwidth per node
```

### I2C

```
Applications: Sensors, small actuators, PWM drivers
Speed: 100 kHz to 3.4 MHz
Addressing: 7-bit (128 devices max)

Common Uses:
├── PCA9685 PWM driver (16 channels)
├── Sensor hubs
└── Small servo controllers

Pros:
├── Simple two-wire bus
├── Multiple devices on bus
├── Built into most MCUs
└── Address-based

Cons:
├── Short cable length (<1m)
├── No error correction
├── Clock stretching issues
├── Limited bandwidth
```

### SPI

```
Applications: High-speed sensor/driver communication
Speed: Up to 50+ MHz
Signals: MOSI, MISO, SCK, CS (per device)

Common Uses:
├── TMC5160 stepper driver
├── High-speed encoders
└── ADC/DAC chips

Pros:
├── Very fast
├── Full duplex
├── Simple protocol
└── No addressing overhead

Cons:
├── Many wires (4+ per device)
├── Short cable length
├── No built-in error detection
└── CS pin per device
```

---

## Gorai Integration Considerations

### Actuator Interface Hierarchy

Based on this analysis, the Gorai motor/actuator interface should support:

```
Actuator (base interface)
├── SetPower(percent float64)      // Universal: -100% to +100%
├── Stop()                         // Emergency stop
├── GetPosition() (float64, error) // If feedback available
└── GetVelocity() (float64, error) // If feedback available

PositionActuator
├── SetPosition(pos float64)       // Target position
├── SetVelocity(vel float64)       // Max velocity
└── IsMoving() bool

VelocityActuator
├── SetVelocity(vel float64)       // Target velocity
└── SetAcceleration(accel float64)

ServoActuator
├── SetAngle(degrees float64)
├── GetAngle() float64
├── SetSpeed(degPerSec float64)
└── SetTorqueLimit(percent float64)

StepperActuator
├── Step(steps int, direction bool)
├── SetMicrostepping(divisor int)
├── SetCurrent(runMA, holdMA int)
└── Home(direction bool) error

ThrusterActuator
├── SetThrust(percent float64)     // -100% to +100%
├── GetRPM() (int, error)          // If telemetry available
└── GetTemperature() (float64, error)
```

### Link Types for Motor Controllers

Motor controllers connect via various Link types:

| Controller | Primary Link | Alternative |
|------------|-------------|-------------|
| RC Servo | GPIO (PWM) | PCA9685 (I2C) |
| ODrive | CAN | UART, USB |
| RoboClaw | Serial | USB |
| Dynamixel | Serial (half-duplex) | - |
| TMC2209 | GPIO (Step/Dir) | UART |
| ESC | GPIO (PWM) | I2C, CAN |

### Configuration Parameters

Motor configurations should include:

```json
{
  "actuators": [
    {
      "name": "left_wheel",
      "type": "bldc",
      "controller": "odrive",
      "link": "can_bus_0",
      "config": {
        "node_id": 0,
        "control_mode": "velocity",
        "encoder_cpr": 8192,
        "velocity_limit": 50.0,
        "current_limit": 20.0
      }
    },
    {
      "name": "pan_servo",
      "type": "servo",
      "controller": "pwm",
      "link": "gpio",
      "config": {
        "pin": 18,
        "min_pulse_us": 500,
        "max_pulse_us": 2500,
        "min_angle": -90,
        "max_angle": 90
      }
    },
    {
      "name": "thruster_port",
      "type": "thruster",
      "controller": "basic_esc",
      "link": "gpio",
      "config": {
        "pin": 12,
        "deadband_us": 25,
        "reverse_enabled": true
      }
    }
  ]
}
```

---

## Summary Table

| Motor Type | Control Method | Feedback | Best For |
|------------|---------------|----------|----------|
| **BLDC** | PWM/FOC | Encoder/Hall | High-speed, high-power |
| **RC Servo** | PWM | None/Pot | Simple positioning |
| **Smart Servo** | Serial | Encoder | Robotic arms |
| **Stepper** | Step/Dir | None/Encoder | Precision, CNC |
| **Thruster** | PWM (ESC) | None/Telem | Underwater |

---

## Sources

- [ODrive Documentation](https://docs.odriverobotics.com/)
- [Blue Robotics Thruster Documentation](https://bluerobotics.com/store/thrusters/)
- [Dynamixel Protocol](https://emanual.robotis.com/docs/en/dxl/protocol2/)
- [TMC2209 Datasheet](https://www.trinamic.com/products/integrated-circuits/details/tmc2209-la/)
- [RoboClaw User Manual](https://www.basicmicro.com/downloads)
- [SimpleFOC Library](https://simplefoc.com/)
- [ros2_control Hardware Interfaces](https://control.ros.org/rolling/doc/ros2_control/hardware_interface/doc/hardware_interface_types_userdoc.html)
