# Configuration Reference

GoRAI uses YAML configuration files for node and component settings.

## Configuration File

Default location: `config.yaml` in the current directory.

Override with `GORAI_CONFIG` environment variable or `--config` flag.

## Node Configuration

```yaml
node:
  name: my-robot
  namespace: gorai

nats:
  url: nats://localhost:4222
  timeout: 5s
  retry_interval: 1s
  max_reconnects: 10

logging:
  level: info
  format: json
```

## Component Configuration

```yaml
components:
  - type: sensor
    name: cpu-temp
    driver: temperature
    config:
      publish_rate: 1s
      topic: gorai.sensor.temperature.cpu

  - type: actuator
    name: drive
    driver: differential
    config:
      left_motor: motor-left
      right_motor: motor-right
```

## Service Configuration

```yaml
services:
  - type: vision
    name: object-detector
    config:
      model: yolov8n
      accelerator: npu
      input_topic: gorai.sensor.camera.front.image
      output_topic: gorai.service.vision.detections
```

## Configuration Priority

1. Command-line flags (highest)
2. Environment variables
3. Configuration file
4. Default values (lowest)

## Environment Variable Mapping

Configuration keys can be set via environment variables:

- Replace `.` with `_`
- Convert to uppercase
- Prefix with `GORAI_`

Example: `nats.url` → `GORAI_NATS_URL`
