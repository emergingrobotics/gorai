# Hello Sensor Example

The Hello Sensor is your introduction to GoRAI development. It demonstrates core concepts with minimal complexity.

## What You'll Build

A sensor that reads CPU temperature and publishes it over NATS.

## What You'll Learn

- Creating a GoRAI node
- Implementing the Sensor interface
- Platform-specific hardware access
- Publishing structured messages
- Fake implementations for testing

## Prerequisites

- Go 1.21+
- NATS server running
- Basic Go knowledge

## Quick Start

```bash
# Clone the repository
git clone https://github.com/gorai/gorai.git
cd gorai/examples/hello-sensor

# Run the sensor
go run .
```

## Project Structure

```
hello-sensor/
├── main.go           # Entry point
├── reader/
│   ├── reader.go     # Interface definition
│   ├── linux.go      # Linux implementation
│   ├── darwin.go     # macOS implementation
│   └── unsupported.go
└── sensor/
    ├── temperature.go # Sensor component
    └── fake/
        └── fake.go    # Test double
```

## Next Steps

After completing this example:

1. **Read the deep dive**: [Chapter 11: Hello Sensor Deep Dive](../../part3-development/ch11-hello-sensor/_index.md)
2. **Try Pan-Tilt**: Add real hardware with the [Pan-Tilt example](../pan-tilt/_index.md)
3. **Build custom components**: Learn to create your own in [Chapter 12](../../part3-development/ch12-custom/_index.md)
