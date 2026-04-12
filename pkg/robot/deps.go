package robot

import (
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
)

// componentDeps provides access to already-created components, the NATS
// connection, and the logger. Passed to component constructors.
// It replaces the old robotDeps and satisfies registry.Dependencies.
type componentDeps struct {
	components map[string]any
	natsConn   *nats.Conn
	logger     *slog.Logger
}

func newComponentDeps(nc *nats.Conn, logger *slog.Logger) *componentDeps {
	return &componentDeps{
		components: make(map[string]any),
		natsConn:   nc,
		logger:     logger,
	}
}

// Get returns a named dependency. "nats" and "logger" are always available.
// Components created earlier in the topological order are also available.
func (d *componentDeps) Get(name string) (any, error) {
	switch name {
	case "nats":
		return d.natsConn, nil
	case "logger":
		return d.logger, nil
	}
	if comp, ok := d.components[name]; ok {
		return comp, nil
	}
	return nil, fmt.Errorf("dependency %q not found", name)
}

// GetByType returns all components. Not filtered by type yet.
func (d *componentDeps) GetByType(subtype string) ([]any, error) {
	return nil, nil
}

// Add registers a created component so downstream components can access it.
func (d *componentDeps) Add(name string, component any) {
	d.components[name] = component
}
