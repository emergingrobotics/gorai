package gateway

import (
	"context"
	"log/slog"
	"sync"

	"github.com/gorai/gorai/pkg/nats"
	"github.com/gorai/gorai/pkg/resource"
)

// Gateway bridges GSP serial devices to NATS messaging.
type Gateway struct {
	name   resource.Name
	nc     *nats.Client
	logger *slog.Logger
	config *Config

	mu      sync.RWMutex
	ports   map[string]*PortHandler
	running bool
	cancel  context.CancelFunc
}

// New creates a new GSP gateway.
func New(name string, nc *nats.Client, cfg *Config, opts ...Option) (*Gateway, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	g := &Gateway{
		name:   resource.NewServiceName("gorai", "gateway", name),
		nc:     nc,
		logger: slog.Default().With("service", "gateway", "name", name),
		config: cfg,
		ports:  make(map[string]*PortHandler),
	}

	for _, opt := range opts {
		opt(g)
	}

	return g, nil
}

// Option configures a Gateway.
type Option func(*Gateway)

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) Option {
	return func(g *Gateway) {
		g.logger = logger.With("service", "gateway", "name", g.name.Name)
	}
}

// Name returns the resource name.
func (g *Gateway) Name() resource.Name {
	return g.name
}

// Reconfigure updates the gateway configuration.
func (g *Gateway) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	// Parse new configuration
	cfg := DefaultConfig()
	if err := conf.Unmarshal(cfg); err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	wasRunning := g.running
	if wasRunning {
		g.stopLocked()
	}

	g.config = cfg

	if wasRunning {
		return g.startLocked(ctx)
	}
	return nil
}

// DoCommand handles custom commands.
func (g *Gateway) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if action, ok := cmd["action"].(string); ok {
		switch action {
		case "start":
			return nil, g.Start(ctx)
		case "stop":
			return nil, g.Stop()
		case "stats":
			return g.GetStats(), nil
		}
	}
	return nil, nil
}

// Close stops the gateway and releases resources.
func (g *Gateway) Close(ctx context.Context) error {
	return g.Stop()
}

// Start begins bridging serial ports to NATS.
func (g *Gateway) Start(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.startLocked(ctx)
}

func (g *Gateway) startLocked(ctx context.Context) error {
	if g.running {
		return ErrAlreadyRunning
	}

	ctx, cancel := context.WithCancel(ctx)
	g.cancel = cancel

	// Start a handler for each configured port
	for _, portCfg := range g.config.Ports {
		handler := NewPortHandler(portCfg, g.nc, g.logger, g.config)
		g.ports[portCfg.Device] = handler

		go func(h *PortHandler) {
			if err := h.Run(ctx); err != nil {
				g.logger.Error("port handler error", "device", h.config.Device, "error", err)
			}
		}(handler)
	}

	g.running = true
	g.logger.Info("gateway started", "ports", len(g.config.Ports))
	return nil
}

// Stop halts all port handlers.
func (g *Gateway) Stop() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.stopLocked()
}

func (g *Gateway) stopLocked() error {
	if !g.running {
		return ErrNotRunning
	}

	if g.cancel != nil {
		g.cancel()
		g.cancel = nil
	}

	// Stop all port handlers
	for _, handler := range g.ports {
		handler.Stop()
	}
	g.ports = make(map[string]*PortHandler)

	g.running = false
	g.logger.Info("gateway stopped")
	return nil
}

// IsRunning returns true if the gateway is active.
func (g *Gateway) IsRunning() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.running
}

// GetStats returns statistics for all ports.
func (g *Gateway) GetStats() map[string]any {
	g.mu.RLock()
	defer g.mu.RUnlock()

	stats := make(map[string]any)
	for device, handler := range g.ports {
		stats[device] = handler.GetStats()
	}
	return stats
}

// GetPortStats returns statistics for a specific port.
func (g *Gateway) GetPortStats(device string) (*PortStats, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	handler, ok := g.ports[device]
	if !ok {
		return nil, ErrPortNotFound
	}
	return handler.GetStats(), nil
}
