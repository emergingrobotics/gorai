## 2.4 Configuration & Hot Reload

Robots need configuration: motor directions, sensor calibrations, network addresses, behavioral parameters. GoRAI provides a configuration system that works at runtime, not just at startup.

### JSON-Based Configuration

Configuration files are JSON, readable and editable without special tools:

```json
{
  "components": [
    {
      "name": "left_motor",
      "type": "motor",
      "model": "gpio",
      "attributes": {
        "pin_forward": 17,
        "pin_reverse": 18,
        "pin_pwm": 12,
        "max_rpm": 200,
        "encoder_pin": 23,
        "ticks_per_revolution": 1200
      }
    },
    {
      "name": "front_camera",
      "type": "camera",
      "model": "v4l2",
      "attributes": {
        "device": "/dev/video0",
        "width": 640,
        "height": 480,
        "fps": 30
      }
    }
  ],
  "services": [
    {
      "name": "detector",
      "type": "vision",
      "model": "yolox",
      "attributes": {
        "model_path": "/models/yolox_s.onnx",
        "confidence_threshold": 0.5,
        "nms_threshold": 0.4
      },
      "depends_on": ["front_camera"]
    }
  ]
}
```

The structure is intentional:
- **name**: Unique identifier within type
- **type**: Component/service category (motor, camera, vision)
- **model**: Specific implementation (gpio motor, v4l2 camera, yolox detector)
- **attributes**: Implementation-specific settings
- **depends_on**: Resources this one needs (for initialization order)

### Loading Configuration

The `config` package parses configuration files:

```go
import "github.com/gorai/gorai/pkg/config"

cfg, err := config.LoadFile("robot.json")
if err != nil {
    log.Fatal(err)
}

for _, comp := range cfg.Components {
    fmt.Printf("Component: %s (type=%s, model=%s)\n",
        comp.Name, comp.Type, comp.Model)
}
```

Configuration flows into resources through the `Config` type:

```go
type Config struct {
    Attributes map[string]any
    Raw        []byte
}

// Resources receive Config during creation and reconfiguration
func (m *Motor) Reconfigure(ctx context.Context, deps Dependencies, conf Config) error {
    // Parse typed configuration
    var cfg MotorConfig
    if err := conf.Unmarshal(&cfg); err != nil {
        return err
    }

    m.maxRPM = cfg.MaxRPM
    m.ticksPerRev = cfg.TicksPerRevolution
    return nil
}
```

### Runtime Reconfiguration Without Restart

The `Reconfigure()` method on every resource enables runtime updates:

```go
// Change motor parameters without restarting
newConf := resource.NewConfig(map[string]any{
    "max_rpm": 250,  // Increase from 200
})
motor.Reconfigure(ctx, deps, newConf)
```

This matters for:
- **Tuning**: Adjust PID gains while watching behavior
- **Adaptation**: Change parameters based on conditions (indoor vs outdoor)
- **Debugging**: Temporarily lower speeds, increase logging
- **Fleet management**: Push configuration updates to deployed robots

### Dependency Injection

Resources often depend on other resources. A vision service needs a camera. A navigation service needs motors and sensors. GoRAI manages these dependencies explicitly:

```go
type Dependencies interface {
    Get(name Name) (Resource, error)
    GetByType(subtype string) ([]Resource, error)
    All() []Resource
}

// Vision service uses dependency injection
func NewVisionService(deps Dependencies, conf Config) (*VisionService, error) {
    cameraName := conf.GetString("camera")
    camera, err := deps.Get(resource.MustParseName(cameraName))
    if err != nil {
        return nil, fmt.Errorf("camera not found: %w", err)
    }

    return &VisionService{
        camera: camera.(Camera),
        // ...
    }, nil
}
```

The configuration's `depends_on` field ensures proper initialization order. Resources start only after their dependencies are ready.

*Cross-reference: See `pkg/config/` for configuration loading implementation.*

### Configuration Best Practices

**Keep hardware-specific values in config, not code**:
```go
// Good: values from config
maxSpeed := cfg.GetFloat("max_speed")

// Bad: hardcoded values
maxSpeed := 1.5
```

**Validate early**:
```go
func (m *Motor) Reconfigure(ctx, deps, conf) error {
    var cfg MotorConfig
    if err := conf.Unmarshal(&cfg); err != nil {
        return fmt.Errorf("invalid config: %w", err)
    }

    if cfg.MaxRPM <= 0 {
        return fmt.Errorf("max_rpm must be positive, got %f", cfg.MaxRPM)
    }

    // ... apply valid configuration
}
```

**Support sensible defaults**:
```go
type MotorConfig struct {
    MaxRPM          float64 `json:"max_rpm"`
    TicksPerRev     int     `json:"ticks_per_revolution"`
    ControlLoopHz   int     `json:"control_loop_hz"`
}

func DefaultMotorConfig() MotorConfig {
    return MotorConfig{
        MaxRPM:        100,
        TicksPerRev:   1000,
        ControlLoopHz: 100,
    }
}
```
