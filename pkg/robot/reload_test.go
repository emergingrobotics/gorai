package robot

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/gorai/gorai/driver/camera/v4l2"
	"github.com/gorai/gorai/pkg/config"
	"github.com/gorai/gorai/pkg/resource"
	"github.com/gorai/gorai/pkg/topics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockReconfigurable records calls to Reconfigure for test assertions.
type mockReconfigurable struct {
	name             resource.Name
	mu               sync.Mutex
	reconfigureCalls []resource.Config
	reconfigureError error
}

func (m *mockReconfigurable) Name() resource.Name { return m.name }
func (m *mockReconfigurable) Reconfigure(_ context.Context, _ resource.Dependencies, conf resource.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reconfigureCalls = append(m.reconfigureCalls, conf)
	return m.reconfigureError
}
func (m *mockReconfigurable) DoCommand(_ context.Context, _ map[string]any) (map[string]any, error) {
	return nil, nil
}
func (m *mockReconfigurable) Close(_ context.Context) error { return nil }

func (m *mockReconfigurable) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.reconfigureCalls)
}

func (m *mockReconfigurable) lastConfig() resource.Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reconfigureCalls[len(m.reconfigureCalls)-1]
}

func newTestRobot(cfg *config.RDL) *Robot {
	ctx, cancel := context.WithCancel(context.Background())
	r := &Robot{
		cfg:              cfg,
		ctx:              ctx,
		cancel:           cancel,
		topics:           topics.NewBuilder(cfg.Robot.Name),
		components:       make(map[string]any),
		services:         make(map[string]any),
		cameras:          make(map[string]*v4l2.Camera),
		externalServices: make(map[string]*ExternalService),
		frameCounters:    make(map[string]uint64),
	}
	r.logger = defaultTestLogger()
	return r
}

func defaultTestLogger() *slog.Logger {
	return slog.Default()
}

func makeTestConfig() *config.RDL {
	return &config.RDL{
		Version: "2",
		Robot:   config.RobotConfig{Name: "test-robot", Namespace: "gorai"},
		NATS:    &config.NATSConfig{URL: "nats://localhost:4222"},
		Components: []config.ComponentConfig{
			{Name: "plug_a", Type: "switch", Model: "tasmota", Attributes: map[string]any{"poll_interval": "30s"}},
			{Name: "bulb_a", Type: "switch", Model: "kauf", Attributes: map[string]any{"brightness": float64(240)}},
		},
		Services: []config.ServiceConfig{
			{Name: "light-controller", Type: "automation", Model: "light-controller", Attributes: map[string]any{"retry_attempts": float64(3)}},
		},
	}
}

func cloneTestConfig(c *config.RDL) *config.RDL {
	n := *c
	n.Components = make([]config.ComponentConfig, len(c.Components))
	for i, comp := range c.Components {
		n.Components[i] = comp
		n.Components[i].Attributes = make(map[string]any)
		for k, v := range comp.Attributes {
			n.Components[i].Attributes[k] = v
		}
	}
	n.Services = make([]config.ServiceConfig, len(c.Services))
	for i, svc := range c.Services {
		n.Services[i] = svc
		n.Services[i].Attributes = make(map[string]any)
		for k, v := range svc.Attributes {
			n.Services[i].Attributes[k] = v
		}
	}
	if c.NATS != nil {
		nats := *c.NATS
		n.NATS = &nats
	}
	return &n
}

func TestReload_AttributeChangeOnComponent(t *testing.T) {
	cfg := makeTestConfig()
	r := newTestRobot(cfg)
	defer r.cancel()

	mock := &mockReconfigurable{name: resource.Name{Name: "plug_a"}}
	r.components["plug_a"] = mock

	newCfg := cloneTestConfig(cfg)
	newCfg.Components[0].Attributes["poll_interval"] = "60s"

	r.handleConfigReload(newCfg)

	require.Equal(t, 1, mock.callCount())
	assert.Equal(t, "60s", mock.lastConfig().Attributes["poll_interval"])
}

func TestReload_AttributeChangeOnService(t *testing.T) {
	cfg := makeTestConfig()
	r := newTestRobot(cfg)
	defer r.cancel()

	mock := &mockReconfigurable{name: resource.Name{Name: "light-controller"}}
	r.services["light-controller"] = mock

	newCfg := cloneTestConfig(cfg)
	newCfg.Services[0].Attributes["retry_attempts"] = float64(5)

	r.handleConfigReload(newCfg)

	require.Equal(t, 1, mock.callCount())
	assert.Equal(t, float64(5), mock.lastConfig().Attributes["retry_attempts"])
}

func TestReload_StructuralChangeRejected(t *testing.T) {
	cfg := makeTestConfig()
	r := newTestRobot(cfg)
	defer r.cancel()

	mock := &mockReconfigurable{name: resource.Name{Name: "plug_a"}}
	r.components["plug_a"] = mock

	// Add a new component (structural change)
	newCfg := cloneTestConfig(cfg)
	newCfg.Components = append(newCfg.Components, config.ComponentConfig{Name: "plug_b", Type: "switch", Model: "tasmota"})

	r.handleConfigReload(newCfg)

	assert.Equal(t, 0, mock.callCount(), "no Reconfigure should be called on structural rejection")
	assert.Equal(t, cfg, r.Config(), "config should remain unchanged after structural rejection")
}

func TestReload_NonResourceComponentSkipped(t *testing.T) {
	cfg := makeTestConfig()
	r := newTestRobot(cfg)
	defer r.cancel()

	// Register a component that does NOT implement resource.Resource
	r.components["plug_a"] = "not-a-resource"

	newCfg := cloneTestConfig(cfg)
	newCfg.Components[0].Attributes["poll_interval"] = "60s"

	// Should not panic
	r.handleConfigReload(newCfg)

	// Config should be updated (attribute change was valid, component just doesn't support reconfigure)
	assert.Equal(t, newCfg, r.Config())
}

func TestReload_ReconfigureError(t *testing.T) {
	cfg := makeTestConfig()
	r := newTestRobot(cfg)
	defer r.cancel()

	failMock := &mockReconfigurable{
		name:             resource.Name{Name: "plug_a"},
		reconfigureError: assert.AnError,
	}
	successMock := &mockReconfigurable{name: resource.Name{Name: "bulb_a"}}
	r.components["plug_a"] = failMock
	r.components["bulb_a"] = successMock

	newCfg := cloneTestConfig(cfg)
	newCfg.Components[0].Attributes["poll_interval"] = "10s"
	newCfg.Components[1].Attributes["brightness"] = float64(100)

	r.handleConfigReload(newCfg)

	// Both should have been called despite the first failing
	assert.Equal(t, 1, failMock.callCount())
	assert.Equal(t, 1, successMock.callCount())

	// Config should still be updated (partial failure does not roll back)
	assert.Equal(t, newCfg, r.Config())
}

func TestReload_ConfigUpdatedAfterSuccess(t *testing.T) {
	cfg := makeTestConfig()
	r := newTestRobot(cfg)
	defer r.cancel()

	mock := &mockReconfigurable{name: resource.Name{Name: "plug_a"}}
	r.components["plug_a"] = mock

	newCfg := cloneTestConfig(cfg)
	newCfg.Components[0].Attributes["poll_interval"] = "120s"

	r.handleConfigReload(newCfg)

	assert.Equal(t, newCfg, r.Config())
}

func TestReload_ConfigNotUpdatedAfterRejection(t *testing.T) {
	cfg := makeTestConfig()
	r := newTestRobot(cfg)
	defer r.cancel()

	// Structural change: remove a component
	newCfg := cloneTestConfig(cfg)
	newCfg.Components = newCfg.Components[:1]

	r.handleConfigReload(newCfg)

	assert.Equal(t, cfg, r.Config())
}

func TestReload_NoChangeDetected(t *testing.T) {
	cfg := makeTestConfig()
	r := newTestRobot(cfg)
	defer r.cancel()

	mock := &mockReconfigurable{name: resource.Name{Name: "plug_a"}}
	r.components["plug_a"] = mock

	// Same config, no changes
	newCfg := cloneTestConfig(cfg)
	r.handleConfigReload(newCfg)

	assert.Equal(t, 0, mock.callCount(), "no Reconfigure should be called when nothing changed")
}

func TestReload_MultipleComponentsAndServices(t *testing.T) {
	cfg := makeTestConfig()
	r := newTestRobot(cfg)
	defer r.cancel()

	compMock1 := &mockReconfigurable{name: resource.Name{Name: "plug_a"}}
	compMock2 := &mockReconfigurable{name: resource.Name{Name: "bulb_a"}}
	svcMock := &mockReconfigurable{name: resource.Name{Name: "light-controller"}}
	r.components["plug_a"] = compMock1
	r.components["bulb_a"] = compMock2
	r.services["light-controller"] = svcMock

	newCfg := cloneTestConfig(cfg)
	newCfg.Components[0].Attributes["poll_interval"] = "10s"
	newCfg.Services[0].Attributes["retry_attempts"] = float64(1)

	r.handleConfigReload(newCfg)

	assert.Equal(t, 1, compMock1.callCount())
	assert.Equal(t, 0, compMock2.callCount(), "unchanged component should not be reconfigured")
	assert.Equal(t, 1, svcMock.callCount())
}
