# Surface Vehicle Example

The GoRAI-Skimmer is an autonomous surface vehicle for water monitoring and bathymetry.

## What You'll Build

A boat-form robot capable of autonomous navigation, data collection, and remote operation.

## Hardware Required

- Single-board computer (Orange Pi 5 recommended)
- Marine-grade hull
- Brushless motors (2x) with ESCs
- GPS module
- IMU (MPU6050 or similar)
- Depth sounder (optional)
- Waterproof enclosure

**Estimated cost**: ~$530

## What You'll Learn

- Autonomous navigation
- GPS integration
- Motor control with ESCs
- Waterproofing considerations
- Mission planning
- Telemetry and logging

## Prerequisites

- Completed Hello Sensor and Pan-Tilt examples
- Hardware listed above
- Understanding of coordinate systems
- Soldering skills for motor connections

## Quick Start

```bash
# Clone the repository
git clone https://github.com/gorai/gorai.git
cd gorai/examples/skimmer

# Configure for your hardware
cp config.example.yaml config.yaml
# Edit config.yaml

# Run in simulation first
go run . --simulate

# Run on hardware
go run .
```

## Project Structure

```
skimmer/
├── main.go
├── config.yaml
├── sensors/
│   ├── gps.go
│   ├── imu.go
│   └── depth.go
├── actuators/
│   ├── thruster.go
│   └── rudder.go
├── services/
│   └── navigation.go
├── behaviors/
│   ├── waypoint.go
│   ├── return_home.go
│   └── survey.go
└── coordinator/
    └── mission.go
```

## Architecture

```
┌─────────────────────────────────────────┐
│            Mission Coordinator          │
└───────────────────┬─────────────────────┘
                    │
    ┌───────────────┼───────────────┐
    ▼               ▼               ▼
┌────────┐    ┌──────────┐    ┌──────────┐
│Waypoint│    │  Survey  │    │Return    │
│Behavior│    │ Behavior │    │Home      │
└───┬────┘    └────┬─────┘    └────┬─────┘
    │              │               │
    └──────────────┼───────────────┘
                   │
           ┌───────▼───────┐
           │  Navigation   │
           │   Service     │
           └───────┬───────┘
                   │
    ┌──────────────┼──────────────┐
    ▼              ▼              ▼
┌───────┐    ┌─────────┐    ┌─────────┐
│  GPS  │    │   IMU   │    │ Depth   │
└───────┘    └─────────┘    └─────────┘
```

## Safety Considerations

1. **Always test in simulation first**
2. Implement geofencing
3. Include emergency stop functionality
4. Monitor battery levels
5. Have a recovery plan

## Next Steps

1. Add water quality sensors
2. Implement obstacle avoidance
3. Create a ground station UI
4. Add multi-vehicle coordination
