package sensorpub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/emergingrobotics/gorai/pkg/subjects"
	"github.com/nats-io/nats.go"
)

func init() {
	registry.RegisterService("telemetry", "sensor_poller", New)
}

// readable is the minimal interface a source must satisfy (component.Sensor).
type readable interface {
	Readings(ctx context.Context) (map[string]any, error)
}

// boundSource pairs a configured source with its resolved reader and subject.
type boundSource struct {
	cfg     Source
	reader  readable
	subject string
}

// Publisher polls sensors and publishes their readings to NATS.
type Publisher struct {
	name     resource.Name
	config   Config
	logger   *slog.Logger
	nc       *nats.Conn
	subjects *subjects.Builder

	sources []boundSource

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a sensor publisher service.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	namespace, _ := conf["namespace"].(string)
	if namespace == "" {
		namespace, _ = conf["robot_name"].(string)
	}

	cfg, err := ParseConfig(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	logger := slog.Default()
	if loggerRes, err := deps.Get("logger"); err == nil {
		if l, ok := loggerRes.(*slog.Logger); ok {
			logger = l
		}
	}

	p := &Publisher{
		name:     resource.NewServiceName("gorai", "telemetry", nameStr),
		config:   cfg,
		logger:   logger.With("service", "sensorpub", "name", nameStr),
		subjects: subjects.NewBuilder(namespace),
	}

	if natsRes, err := deps.Get("nats"); err == nil {
		if nc, ok := natsRes.(*nats.Conn); ok {
			p.nc = nc
		}
	}
	if p.nc == nil {
		return nil, fmt.Errorf("NATS connection not available")
	}

	for _, src := range cfg.Sources {
		res, err := deps.Get(src.Name)
		if err != nil {
			return nil, fmt.Errorf("sensor %q not available: %w", src.Name, err)
		}
		reader, ok := res.(readable)
		if !ok {
			return nil, fmt.Errorf("sensor %q does not provide readings", src.Name)
		}
		p.sources = append(p.sources, boundSource{
			cfg:     src,
			reader:  reader,
			subject: p.subjects.ComponentData(src.Name),
		})
	}

	return p, nil
}

// Start launches one polling goroutine per source.
func (p *Publisher) Start(ctx context.Context) error {
	loopCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	for _, src := range p.sources {
		p.wg.Add(1)
		go p.pollLoop(loopCtx, src)
	}

	p.logger.Info("Sensor publisher started", "sources", len(p.sources))
	return nil
}

// pollLoop reads and publishes one source at its configured rate.
func (p *Publisher) pollLoop(ctx context.Context, src boundSource) {
	defer p.wg.Done()

	period := time.Duration(float64(time.Second) / src.cfg.RateHz)
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.publish(ctx, src)
		}
	}
}

// publish reads one source and publishes its readings as JSON.
func (p *Publisher) publish(ctx context.Context, src boundSource) {
	readings, err := src.reader.Readings(ctx)
	if err != nil {
		p.logger.Debug("sensor read failed", "name", src.cfg.Name, "error", err)
		return
	}
	if src.cfg.Kind != "" {
		readings["kind"] = src.cfg.Kind
	}
	data, err := json.Marshal(readings)
	if err != nil {
		return
	}
	if err := p.nc.Publish(src.subject, data); err != nil {
		p.logger.Debug("sensor publish failed", "subject", src.subject, "error", err)
	}
}

// Name returns the resource name.
func (p *Publisher) Name() resource.Name {
	return p.name
}

// Reconfigure is not supported; restart the service instead.
func (p *Publisher) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand exposes basic status.
func (p *Publisher) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if name, _ := cmd["command"].(string); name == "get_state" {
		return map[string]any{"sources": len(p.sources)}, nil
	}
	return nil, fmt.Errorf("unknown command: %v", cmd["command"])
}

// Close stops all polling goroutines.
func (p *Publisher) Close(ctx context.Context) error {
	if p.cancel != nil {
		p.cancel()
	}
	done := make(chan struct{})
	go func() { p.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
	}
	p.logger.Info("Sensor publisher stopped")
	return nil
}
