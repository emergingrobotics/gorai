package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/emergingrobotics/gorai/pkg/mesh"
	"github.com/emergingrobotics/gorai/pkg/proxy"
)

// Manager coordinates discovery from multiple sources and auto-adoption.
type Manager struct {
	meshClient   *mesh.Client
	natsConn     *nats.Conn
	config       *Config
	logger       *slog.Logger
	proxyFactory *proxy.Factory

	mu       sync.RWMutex
	adopted  map[string]*AdoptedResource // keyed by service ID
	watchers []*mesh.Watcher
	events   chan Event

	cancel context.CancelFunc
	done   chan struct{}
}

// ManagerOption configures a Manager.
type ManagerOption func(*managerConfig)

type managerConfig struct {
	logger       *slog.Logger
	proxyFactory *proxy.Factory
	eventBuffer  int
}

// WithManagerLogger sets the logger.
func WithManagerLogger(logger *slog.Logger) ManagerOption {
	return func(c *managerConfig) {
		c.logger = logger
	}
}

// WithProxyFactory sets a custom proxy factory.
func WithProxyFactory(f *proxy.Factory) ManagerOption {
	return func(c *managerConfig) {
		c.proxyFactory = f
	}
}

// WithEventBuffer sets the event channel buffer size.
func WithEventBuffer(size int) ManagerOption {
	return func(c *managerConfig) {
		c.eventBuffer = size
	}
}

// NewManager creates a new discovery manager.
func NewManager(meshClient *mesh.Client, natsConn *nats.Conn, config *Config, opts ...ManagerOption) *Manager {
	if config == nil {
		config = DefaultConfig()
	}

	cfg := &managerConfig{
		logger:      slog.Default(),
		eventBuffer: 100,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.proxyFactory == nil {
		cfg.proxyFactory = proxy.NewFactory(meshClient, natsConn, proxy.WithFactoryLogger(cfg.logger))
	}

	return &Manager{
		meshClient:   meshClient,
		natsConn:     natsConn,
		config:       config,
		logger:       cfg.logger,
		proxyFactory: cfg.proxyFactory,
		adopted:      make(map[string]*AdoptedResource),
		events:       make(chan Event, cfg.eventBuffer),
		done:         make(chan struct{}),
	}
}

// Start begins discovery and watching.
func (m *Manager) Start(ctx context.Context) error {
	if !m.config.Enabled {
		m.logger.Info("discovery disabled")
		return nil
	}

	ctx, m.cancel = context.WithCancel(ctx)

	// Initial scan
	if err := m.scan(ctx); err != nil {
		m.logger.Warn("initial scan failed", "error", err)
	}

	// Start watchers for each source
	for _, source := range m.config.Sources {
		if err := m.startSourceWatcher(ctx, source); err != nil {
			m.logger.Warn("failed to start watcher", "source", source.Type, "error", err)
		}
	}

	// Start periodic scan
	go m.scanLoop(ctx)

	m.logger.Info("discovery manager started",
		"sources", len(m.config.Sources),
		"rules", len(m.config.Rules),
	)

	return nil
}

// Stop stops the discovery manager.
func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}

	// Stop all watchers
	for _, w := range m.watchers {
		w.Stop()
	}

	close(m.done)
	close(m.events)

	m.logger.Info("discovery manager stopped")
}

// Events returns the event channel.
func (m *Manager) Events() <-chan Event {
	return m.events
}

// GetAdopted returns all adopted resources.
func (m *Manager) GetAdopted() []*AdoptedResource {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*AdoptedResource, 0, len(m.adopted))
	for _, r := range m.adopted {
		result = append(result, r)
	}
	return result
}

// GetAdoptedByType returns adopted resources matching a type.
func (m *Manager) GetAdoptedByType(typ string) []*AdoptedResource {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*AdoptedResource
	for _, r := range m.adopted {
		if r.AdoptedAs.Type == typ {
			result = append(result, r)
		}
	}
	return result
}

// GetAdoptedByPattern returns adopted resources matching a dependency pattern.
func (m *Manager) GetAdoptedByPattern(pattern string) ([]*AdoptedResource, error) {
	dp, err := ParseDependencyPattern(pattern)
	if err != nil {
		return nil, err
	}
	if dp == nil {
		return nil, fmt.Errorf("invalid pattern: %s", pattern)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*AdoptedResource
	for _, r := range m.adopted {
		if dp.Matches(*r) {
			result = append(result, r)
		}
	}
	return result, nil
}

// GetProxy returns the proxy component for an adopted resource.
func (m *Manager) GetProxy(serviceID string) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if r, ok := m.adopted[serviceID]; ok {
		return r.Proxy, true
	}
	return nil, false
}

// scan queries all sources for devices.
func (m *Manager) scan(ctx context.Context) error {
	for _, source := range m.config.Sources {
		services, err := m.querySource(ctx, source)
		if err != nil {
			m.logger.Warn("source query failed", "source", source.Type, "error", err)
			continue
		}

		for _, svc := range services {
			m.handleDiscovered(ctx, source, svc)
		}
	}
	return nil
}

// scanLoop periodically scans for new devices.
func (m *Manager) scanLoop(ctx context.Context) {
	interval := m.config.ScanInterval
	if interval == 0 {
		interval = 5 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.scan(ctx); err != nil {
				m.logger.Warn("periodic scan failed", "error", err)
			}
		}
	}
}

// querySource queries a single source for services.
func (m *Manager) querySource(ctx context.Context, source Source) ([]mesh.ServiceDescriptor, error) {
	switch source.Type {
	case "mesh":
		query := mesh.Query{}
		if source.Query != nil {
			query = *source.Query
		}
		return m.meshClient.FindServices(ctx, query)

	case "gateway":
		// Query mesh for services published by this gateway
		// Gateway devices have metadata indicating their gateway
		services, err := m.meshClient.ListServices(ctx)
		if err != nil {
			return nil, err
		}

		var result []mesh.ServiceDescriptor
		for _, svc := range services {
			if svc.Metadata["gateway"] == source.Gateway {
				result = append(result, svc)
			}
			// Also match by model pattern for GSP devices
			if svc.Model == "gsp-device" && source.Gateway != "" {
				result = append(result, svc)
			}
		}
		return result, nil

	default:
		return nil, fmt.Errorf("unknown source type: %s", source.Type)
	}
}

// startSourceWatcher starts watching a source for changes.
func (m *Manager) startSourceWatcher(ctx context.Context, source Source) error {
	query := mesh.Query{}
	if source.Query != nil {
		query = *source.Query
	}

	watcher, err := m.meshClient.WatchServices(ctx, query)
	if err != nil {
		return err
	}

	m.watchers = append(m.watchers, watcher)

	go func() {
		for event := range watcher.Events() {
			switch event.Type {
			case mesh.EventServiceJoined:
				m.handleDiscovered(ctx, source, *event.Service)
			case mesh.EventServiceLeft:
				m.handleLost(event.Service.ID)
			case mesh.EventServiceUpdated:
				// Could handle updates if needed
			}
		}
	}()

	return nil
}

// handleDiscovered processes a discovered service.
func (m *Manager) handleDiscovered(ctx context.Context, source Source, svc mesh.ServiceDescriptor) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Already adopted?
	if _, exists := m.adopted[svc.ID]; exists {
		return
	}

	// Emit discovered event
	m.emitEvent(Event{
		Type:      EventDiscovered,
		Service:   &svc,
		Timestamp: time.Now(),
	})

	// Should we auto-adopt?
	if !m.config.AutoAdopt {
		return
	}

	// Find matching rule
	rule, found := m.findMatchingRule(svc)
	if !found {
		m.logger.Debug("no matching rule for service", "name", svc.Name, "subtype", svc.Subtype)
		return
	}

	// Adopt the service
	if err := m.adoptLocked(ctx, source, svc, rule); err != nil {
		m.logger.Warn("failed to adopt service", "name", svc.Name, "error", err)
		m.emitEvent(Event{
			Type:      EventError,
			Service:   &svc,
			Error:     err,
			Timestamp: time.Now(),
		})
	}
}

// handleLost processes a lost service.
func (m *Manager) handleLost(serviceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	resource, exists := m.adopted[serviceID]
	if !exists {
		return
	}

	// Clean up proxy
	if closer, ok := resource.Proxy.(interface{ Close(context.Context) error }); ok {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		closer.Close(ctx)
		cancel()
	}

	delete(m.adopted, serviceID)

	m.emitEvent(Event{
		Type:      EventRemoved,
		Resource:  resource,
		Timestamp: time.Now(),
	})

	m.logger.Info("removed adopted resource", "name", resource.ComponentName(), "type", resource.ComponentType())
}

// findMatchingRule finds the first rule that matches the service.
func (m *Manager) findMatchingRule(svc mesh.ServiceDescriptor) (*Rule, bool) {
	for i := range m.config.Rules {
		rule := &m.config.Rules[i]
		if !rule.IsEnabled() {
			continue
		}
		if rule.Match.Matches(svc) {
			return rule, true
		}
	}
	return nil, false
}

// adoptLocked adopts a service (must hold mu lock).
func (m *Manager) adoptLocked(ctx context.Context, source Source, svc mesh.ServiceDescriptor, rule *Rule) error {
	// Create proxy component
	proxyComp, err := m.proxyFactory.Create(ctx, svc, rule.AdoptAs.Type, rule.AdoptAs.Subtype)
	if err != nil {
		return fmt.Errorf("failed to create proxy: %w", err)
	}

	// Merge config
	config := make(map[string]any)
	for k, v := range rule.Config {
		config[k] = v
	}

	resource := &AdoptedResource{
		ID:         svc.ID,
		Source:     source.Gateway,
		SourceType: source.Type,
		Descriptor: svc,
		AdoptedAs:  rule.AdoptAs,
		Config:     config,
		Proxy:      proxyComp,
		AdoptedAt:  time.Now(),
	}

	m.adopted[svc.ID] = resource

	m.emitEvent(Event{
		Type:      EventAdopted,
		Resource:  resource,
		Timestamp: time.Now(),
	})

	m.logger.Info("adopted resource",
		"name", resource.ComponentName(),
		"type", resource.ComponentType(),
		"source", source.Type,
	)

	return nil
}

// emitEvent sends an event to the channel (non-blocking).
func (m *Manager) emitEvent(event Event) {
	select {
	case m.events <- event:
	default:
		// Channel full, drop event
	}
}

// Adopt manually adopts a service with the given adoption settings.
func (m *Manager) Adopt(ctx context.Context, serviceID string, adoptAs AdoptAs, config map[string]any) (*AdoptedResource, error) {
	svc, err := m.meshClient.GetServiceByID(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, exists := m.adopted[serviceID]; exists {
		return existing, nil
	}

	rule := &Rule{
		AdoptAs: adoptAs,
		Config:  config,
	}

	source := Source{Type: "manual"}
	if err := m.adoptLocked(ctx, source, *svc, rule); err != nil {
		return nil, err
	}

	return m.adopted[serviceID], nil
}

// Remove removes an adopted resource.
func (m *Manager) Remove(serviceID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	resource, exists := m.adopted[serviceID]
	if !exists {
		return false
	}

	// Clean up proxy
	if closer, ok := resource.Proxy.(interface{ Close(context.Context) error }); ok {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		closer.Close(ctx)
		cancel()
	}

	delete(m.adopted, serviceID)

	m.emitEvent(Event{
		Type:      EventRemoved,
		Resource:  resource,
		Timestamp: time.Now(),
	})

	return true
}

// ResolveDependencies resolves @discovered: dependencies.
func (m *Manager) ResolveDependencies(deps []string) (map[string]any, []string, error) {
	resolved := make(map[string]any)
	var unresolved []string

	for _, dep := range deps {
		dp, err := ParseDependencyPattern(dep)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid pattern %s: %w", dep, err)
		}
		if dp == nil {
			// Not a discovered dependency, skip
			continue
		}

		matches, err := m.GetAdoptedByPattern(dep)
		if err != nil {
			return nil, nil, err
		}

		if len(matches) == 0 {
			unresolved = append(unresolved, dep)
			continue
		}

		// Add all matches
		for _, match := range matches {
			resolved[match.ComponentName()] = match.Proxy
		}
	}

	return resolved, unresolved, nil
}

// WaitForDependencies waits for @discovered: dependencies to be resolved.
func (m *Manager) WaitForDependencies(ctx context.Context, deps []string) (map[string]any, error) {
	for {
		resolved, unresolved, err := m.ResolveDependencies(deps)
		if err != nil {
			return nil, err
		}

		if len(unresolved) == 0 {
			return resolved, nil
		}

		// Wait for new adoptions
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for dependencies: %v", unresolved)
		case event := <-m.events:
			if event.Type == EventAdopted {
				// Re-check dependencies
				continue
			}
		case <-time.After(100 * time.Millisecond):
			// Periodic re-check
		}
	}
}
