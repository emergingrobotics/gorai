# Pan-Tilt Platform Example

The Pan-Tilt Platform (GoRAI-Sentinel) demonstrates sensor fusion and servo control.

## What You'll Build

A camera mount that can pan and tilt, with optional sensor fusion from depth sensors.

## Hardware Required

- Raspberry Pi 4/5 or similar SBC
- 2x servo motors (SG90 or similar)
- Pan-tilt bracket kit
- USB webcam or Pi Camera
- Optional: VL53L0X ToF sensor

**Estimated cost**: $150-350

## What You'll Learn

- Real hardware integration
- Servo control with PWM
- Camera streaming
- Sensor fusion concepts
- Behavior implementation

## Prerequisites

- Completed Hello Sensor example
- Hardware listed above
- Basic electronics knowledge

## Quick Start

```bash
# Clone the repository
git clone https://github.com/gorai/gorai.git
cd gorai/examples/pan-tilt

# Configure for your hardware
cp config.example.yaml config.yaml
# Edit config.yaml with your pin assignments

# Run
go run .
```

## Project Structure

```
pan-tilt/
├── main.go           # Entry point
├── config.yaml       # Hardware configuration
├── servo/
│   └── servo.go      # Servo component
├── camera/
│   └── camera.go     # Camera component
└── behaviors/
    └── track.go      # Object tracking behavior
```

## Architecture

```
┌─────────────┐     ┌─────────────┐
│   Camera    │────▶│   Vision    │
│  (sensor)   │     │  (service)  │
└─────────────┘     └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │   Tracker   │
                    │ (behavior)  │
                    └──────┬──────┘
                           │
        ┌──────────────────┼──────────────────┐
        ▼                  ▼                  ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  Pan Servo  │    │ Tilt Servo  │    │  ToF Depth  │
│ (actuator)  │    │ (actuator)  │    │  (sensor)   │
└─────────────┘    └─────────────┘    └─────────────┘
```

## Next Steps

1. Add object detection with AI/ML
2. Implement smooth tracking behaviors
3. Try the [Skimmer example](../skimmer/_index.md) for a mobile platform
