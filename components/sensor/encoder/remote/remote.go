package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/emergingrobotics/gorai/components/sensor"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/nats-io/nats.go"
)

func init() {
	registry.RegisterComponent("encoder", "remote", New)
}

type encoderDataPayload struct {
	Encoders []encoderEntry `json:"encoders"`
}

type encoderEntry struct {
	Encoder  uint8 `json:"encoder"`
	Position int32 `json:"position"`
	Speed    int16 `json:"speed"`
	Flags    uint8 `json:"flags"`
	Time     uint16 `json:"time"`
}

type Config struct {
	NATSSubjectPrefix  string `json:"nats_subject_prefix"`
	DeviceID           string `json:"device_id"`
	EncoderIndex       int    `json:"encoder_index"`
	TicksPerRevolution int    `json:"ticks_per_revolution"`
}

func NewConfigFromResource(conf resource.Config) (*Config, error) {
	cfg := &Config{
		TicksPerRevolution: 360,
	}

	if v, ok := conf.GetString("nats_subject_prefix"); ok {
		cfg.NATSSubjectPrefix = v
	}
	if v, ok := conf.GetString("device_id"); ok {
		cfg.DeviceID = v
	}
	if v, ok := conf.GetInt("encoder_index"); ok {
		cfg.EncoderIndex = v
	}
	if v, ok := conf.GetInt("ticks_per_revolution"); ok {
		cfg.TicksPerRevolution = v
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.NATSSubjectPrefix == "" {
		return fmt.Errorf("nats_subject_prefix is required")
	}
	if c.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}
	if c.EncoderIndex < 0 || c.EncoderIndex > 3 {
		return fmt.Errorf("encoder_index must be 0-3")
	}
	if c.TicksPerRevolution <= 0 {
		return fmt.Errorf("ticks_per_revolution must be positive")
	}
	return nil
}

type RemoteEncoder struct {
	name   resource.Name
	config *Config
	logger *slog.Logger
	nc     *nats.Conn

	mu        sync.RWMutex
	position  float64
	velocity  float64
	direction bool
	state_sub *nats.Subscription
}

func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	res_conf := resource.NewConfig(conf)

	cfg, err := NewConfigFromResource(res_conf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	name := "remote_encoder"
	if n, ok := conf["name"].(string); ok {
		name = n
	}

	logger := slog.Default()
	if logger_res, err := deps.Get("logger"); err == nil {
		if l, ok := logger_res.(*slog.Logger); ok {
			logger = l
		}
	}

	r := &RemoteEncoder{
		name:   resource.NewComponentName("gorai", "encoder", name),
		config: cfg,
		logger: logger.With("component", "remote_encoder", "name", name),
	}

	if nats_res, err := deps.Get("nats"); err == nil {
		if nc, ok := nats_res.(*nats.Conn); ok {
			r.nc = nc
		}
	}
	if r.nc == nil {
		return nil, fmt.Errorf("NATS connection not available")
	}

	r.logger.Info("remote encoder component created",
		"device_id", cfg.DeviceID,
		"encoder_index", cfg.EncoderIndex,
	)

	return r, nil
}

func (r *RemoteEncoder) Name() resource.Name {
	return r.name
}

func (r *RemoteEncoder) Start(ctx context.Context) error {
	r.subscribeState()
	return nil
}

func (r *RemoteEncoder) subscribeState() {
	cfg := r.config
	subject := fmt.Sprintf("%s.%s.rx.sensor.encoder_data",
		cfg.NATSSubjectPrefix, cfg.DeviceID)
	r.logger.Info("subscribing to ENCODER_DATA", "subject", subject)

	sub, err := r.nc.Subscribe(subject, func(msg *nats.Msg) {
		var env struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(msg.Data, &env); err != nil {
			r.logger.Warn("failed to unmarshal ENCODER_DATA envelope", "error", err)
			return
		}
		var data encoderDataPayload
		if err := json.Unmarshal(env.Data, &data); err != nil {
			r.logger.Warn("failed to unmarshal ENCODER_DATA payload", "error", err)
			return
		}
		r.mu.Lock()
		defer r.mu.Unlock()
		for _, e := range data.Encoders {
			if int(e.Encoder) == cfg.EncoderIndex {
				r.position = float64(e.Position)
				r.velocity = float64(e.Speed)
				r.direction = e.Flags&0x01 != 0
			}
		}
	})
	if err != nil {
		r.logger.Warn("failed to subscribe to ENCODER_DATA", "error", err)
		return
	}
	r.mu.Lock()
	r.state_sub = sub
	r.mu.Unlock()
}

func (r *RemoteEncoder) Position(ctx context.Context) (float64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.position, nil
}

func (r *RemoteEncoder) ResetPosition(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.position = 0
	return nil
}

func (r *RemoteEncoder) Properties(ctx context.Context) (sensor.EncoderProperties, error) {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	return sensor.EncoderProperties{
		TicksPerRevolution:    cfg.TicksPerRevolution,
		AngleDegreesSupported: false,
		IsAbsolute:            false,
	}, nil
}

func (r *RemoteEncoder) GetVelocity(ctx context.Context) (float64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.velocity, nil
}

func (r *RemoteEncoder) GetResolution(ctx context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config.TicksPerRevolution, nil
}

func (r *RemoteEncoder) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	r.mu.Lock()
	r.config = cfg
	r.mu.Unlock()
	return nil
}

func (r *RemoteEncoder) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmd_name, _ := cmd["command"].(string)
	switch cmd_name {
	case "get_state":
		r.mu.RLock()
		defer r.mu.RUnlock()
		return map[string]any{
			"encoder_index": r.config.EncoderIndex,
			"device_id":     r.config.DeviceID,
			"position":      r.position,
			"velocity":      r.velocity,
			"direction":     r.direction,
		}, nil
	default:
		return nil, fmt.Errorf("unknown command: %s", cmd_name)
	}
}

func (r *RemoteEncoder) Close(ctx context.Context) error {
	r.mu.Lock()
	sub := r.state_sub
	r.state_sub = nil
	r.mu.Unlock()

	if sub != nil {
		_ = sub.Unsubscribe()
	}
	r.logger.Info("remote encoder component closed")
	return nil
}

var _ sensor.Encoder = (*RemoteEncoder)(nil)
