# Hardware Compatibility

This appendix lists hardware tested and supported with GoRAI.

## Supported SBCs

| Board | CPU | RAM | NPU | GPIO | Notes |
|-------|-----|-----|-----|------|-------|
| Raspberry Pi 5 | Cortex-A76 | 8GB | No | Yes | Best starter |
| Orange Pi 5 | RK3588S | 8-16GB | 6 TOPS | Yes | Best for AI |
| Jetson Orin Nano | Cortex-A78 | 8GB | GPU | Yes | CUDA support |
| BeagleBone AI-64 | TDA4VM | 4GB | 8 TOPS | Yes | Real-time PRUs |
| Rock 5B | RK3588 | 8-16GB | 6 TOPS | Yes | PCIe support |

### Recommended Configuration

**For learning/development**:
- Raspberry Pi 5 (8GB)
- USB webcam
- Basic sensors (temperature, distance)

**For AI/ML workloads**:
- Orange Pi 5 or Jetson Orin Nano
- USB3 camera or CSI camera
- NPU-accelerated inference

## Supported Microcontrollers (TinyGo)

| Board | CPU | RAM | Flash | Notes |
|-------|-----|-----|-------|-------|
| RP2040 (Pico) | Cortex-M0+ | 264KB | 2MB | Dual core |
| ESP32-C3 | RISC-V | 400KB | 4MB | WiFi/BLE |
| STM32F4 | Cortex-M4 | 128KB+ | 512KB+ | Industrial |

### TinyGo Notes

- Use latest TinyGo release
- Some packages not available on MCUs
- Serial communication for NATS bridging

## Common Sensors

| Sensor | Interface | GoRAI Support |
|--------|-----------|---------------|
| MPU6050 | I2C | Example available |
| BME280 | I2C/SPI | Example available |
| GPS (NMEA) | UART | Example available |
| HC-SR04 | GPIO | Example available |
| Encoders | GPIO | Included |
| VL53L0X | I2C | Example available |
| Camera (USB) | V4L2 | Included |
| Camera (CSI) | MMAL/V4L2 | Platform-specific |

## Common Actuators

| Actuator | Interface | GoRAI Support |
|----------|-----------|---------------|
| DC Motors | PWM+GPIO | Included |
| Steppers | GPIO | Example available |
| RC Servos | PWM | Included |
| DRV8833 | PWM+GPIO | Example available |
| PCA9685 | I2C | Example available |

## Motor Controllers

| Controller | Motors | Interface | Notes |
|------------|--------|-----------|-------|
| DRV8833 | 2 DC | PWM+GPIO | Low cost |
| TB6612FNG | 2 DC | PWM+GPIO | Higher current |
| Roboclaw | 2 DC | Serial | Encoder support |
| ODrive | 2 BLDC | USB/CAN | High performance |

## Camera Options

| Camera | Resolution | Interface | FPS | Notes |
|--------|------------|-----------|-----|-------|
| Generic USB | 1080p | USB | 30 | Easy setup |
| Pi Camera v3 | 12MP | CSI | 60 | RPi only |
| OAK-D Lite | 4K | USB | 30 | Stereo + AI |
| RealSense D435 | 1080p | USB | 90 | Depth sensing |
