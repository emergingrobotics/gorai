# Glossary

Terms and definitions used throughout GoRAI documentation.

| Term | Definition |
|------|------------|
| **Actuator** | Component that performs physical actions (motors, servos, relays) |
| **Base** | Actuator representing a mobile platform (differential drive, holonomic) |
| **Behavior** | High-level robot action composed of sensor reads and actuator commands |
| **Component** | Hardware abstraction in GoRAI - sensors, actuators, cameras |
| **Consumer** | NATS entity that receives messages from a stream |
| **Coordinator** | Module that orchestrates behaviors and manages robot state |
| **DDS** | Data Distribution Service - middleware used by ROS 2 |
| **Fake** | Test implementation that simulates real hardware behavior |
| **Frame** | Coordinate reference for spatial data |
| **GPIO** | General Purpose Input/Output pins on SBCs |
| **I2C** | Inter-Integrated Circuit - serial communication protocol |
| **JetStream** | NATS persistence and streaming layer |
| **Node** | GoRAI process managing resources and NATS connection |
| **NPU** | Neural Processing Unit - accelerator for ML inference |
| **NWC** | Network Wrapper Client - consumes remote resources over NATS |
| **NWS** | Network Wrapper Server - exposes local resources over NATS |
| **Odometry** | Estimation of robot position from sensor data |
| **Pose** | Position and orientation in 3D space |
| **Proto / Protobuf** | Protocol Buffers - binary serialization format |
| **PWM** | Pulse Width Modulation - technique for analog-like signals |
| **QoS** | Quality of Service - delivery guarantees for messages |
| **Resource** | Base interface for all GoRAI components and services |
| **SBC** | Single Board Computer (Raspberry Pi, etc.) |
| **Sensor** | Component that provides readings from the physical world |
| **Service** | Software capability exposing request/reply functionality |
| **SPI** | Serial Peripheral Interface - high-speed serial protocol |
| **Stream** | JetStream persistent message storage |
| **Subject** | NATS term for topic - the address for message routing |
| **TinyGo** | Go compiler for microcontrollers |
| **Topic** | NATS subject for pub/sub messaging |
| **Transform** | Translation and rotation between coordinate frames |
| **Twist** | Linear and angular velocity |
| **UART** | Universal Asynchronous Receiver-Transmitter - serial protocol |
| **V4L2** | Video4Linux2 - Linux camera API |

## Abbreviations

| Abbreviation | Meaning |
|--------------|---------|
| AI | Artificial Intelligence |
| API | Application Programming Interface |
| BLDC | Brushless DC (motor) |
| CLI | Command Line Interface |
| CSI | Camera Serial Interface |
| DC | Direct Current |
| FPS | Frames Per Second |
| IMU | Inertial Measurement Unit |
| LiDAR | Light Detection and Ranging |
| MCU | Microcontroller Unit |
| ML | Machine Learning |
| NMEA | National Marine Electronics Association (GPS protocol) |
| RGB | Red Green Blue (color encoding) |
| SDK | Software Development Kit |
| ToF | Time of Flight (distance sensor) |
| USB | Universal Serial Bus |
