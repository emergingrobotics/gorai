package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/gorai/gorai/pkg/mesh"
)

// GenericRemote is a generic proxy for components that don't have specific implementations.
// It provides basic DoCommand functionality and data subscription.
type GenericRemote struct {
	meshClient *mesh.Client
	natsConn   *nats.Conn
	descriptor mesh.ServiceDescriptor
	logger     *slog.Logger

	cmdSubject  string
	dataSubject string

	mu         sync.RWMutex
	lastData   map[string]any
	lastDataAt time.Time

	dataSub *nats.Subscription
	cancel  context.CancelFunc
	done    chan struct{}
}

// GenericRemoteOption configures a GenericRemote.
type GenericRemoteOption func(*GenericRemote)

// WithGenericRemoteLogger sets the logger.
func WithGenericRemoteLogger(logger *slog.Logger) GenericRemoteOption {
	return func(g *GenericRemote) {
		g.logger = logger
	}
}

// NewGenericRemote creates a new generic remote proxy.
func NewGenericRemote(mc *mesh.Client, nc *nats.Conn, desc mesh.ServiceDescriptor, opts ...GenericRemoteOption) (*GenericRemote, error) {
	g := &GenericRemote{
		meshClient: mc,
		natsConn:   nc,
		descriptor: desc,
		logger:     slog.Default(),
		lastData:   make(map[string]any),
		done:       make(chan struct{}),
	}

	for _, opt := range opts {
		opt(g)
	}

	// Find command subject (may not exist)
	var err error
	g.cmdSubject, err = GetCommandSubject(desc)
	if err != nil {
		g.cmdSubject = "" // Not all components have commands
	}

	// Find data subject (may not exist)
	g.dataSubject, err = GetDataSubject(desc)
	if err != nil {
		g.dataSubject = ""
	}

	// Subscribe to data if available
	if g.dataSubject != "" {
		g.dataSub, err = nc.Subscribe(g.dataSubject, g.handleData)
		if err != nil {
			g.logger.Warn("failed to subscribe to data", "subject", g.dataSubject, "error", err)
		}
	}

	g.logger.Debug("created generic remote",
		"name", desc.Name,
		"type", desc.Type,
		"subtype", desc.Subtype,
		"cmd_subject", g.cmdSubject,
		"data_subject", g.dataSubject,
	)

	return g, nil
}

// handleData processes incoming data messages.
func (g *GenericRemote) handleData(msg *nats.Msg) {
	var envelope struct {
		DeviceID   string         `json:"device_id"`
		Type       string         `json:"type"`
		Timestamp  time.Time      `json:"ts"`
		ReceivedAt time.Time      `json:"received_at"`
		Data       map[string]any `json:"data"`
	}

	if err := json.Unmarshal(msg.Data, &envelope); err != nil {
		// Try parsing as just data
		var data map[string]any
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			g.logger.Debug("failed to parse data", "error", err)
			return
		}
		envelope.Data = data
	}

	g.mu.Lock()
	// Merge new data
	for k, v := range envelope.Data {
		g.lastData[k] = v
	}
	if envelope.Type != "" {
		g.lastData["_type"] = envelope.Type
	}
	if !envelope.Timestamp.IsZero() {
		g.lastData["_timestamp"] = envelope.Timestamp
	}
	g.lastDataAt = time.Now()
	g.mu.Unlock()
}

// GetData returns the latest data received from the component.
func (g *GenericRemote) GetData(ctx context.Context) (map[string]any, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Return a copy
	result := make(map[string]any)
	for k, v := range g.lastData {
		result[k] = v
	}
	return result, nil
}

// GetValue returns a specific value from the latest data.
func (g *GenericRemote) GetValue(ctx context.Context, key string) (any, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if v, ok := g.lastData[key]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("key %q not found", key)
}

// LastDataAt returns when data was last received.
func (g *GenericRemote) LastDataAt() time.Time {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.lastDataAt
}

// DoCommand sends a command to the component and optionally waits for a response.
func (g *GenericRemote) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if g.cmdSubject == "" {
		return nil, fmt.Errorf("component does not support commands")
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	// Try request-reply first
	msg, err := g.natsConn.RequestWithContext(ctx, g.cmdSubject, data)
	if err != nil {
		// Fall back to fire-and-forget
		if pubErr := g.natsConn.Publish(g.cmdSubject, data); pubErr != nil {
			return nil, fmt.Errorf("failed to publish command: %w", pubErr)
		}
		return nil, nil
	}

	var resp map[string]any
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return resp, nil
}

// Publish sends a message to a specific subject.
func (g *GenericRemote) Publish(ctx context.Context, subject string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	return g.natsConn.Publish(subject, payload)
}

// Request sends a request and waits for a response.
func (g *GenericRemote) Request(ctx context.Context, subject string, data any) (map[string]any, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	msg, err := g.natsConn.RequestWithContext(ctx, subject, payload)
	if err != nil {
		return nil, err
	}

	var resp map[string]any
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return resp, nil
}

// Subscribe subscribes to a subject and calls the handler for each message.
func (g *GenericRemote) Subscribe(subject string, handler func(data map[string]any)) (*nats.Subscription, error) {
	return g.natsConn.Subscribe(subject, func(msg *nats.Msg) {
		var data map[string]any
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			g.logger.Debug("failed to parse message", "subject", subject, "error", err)
			return
		}
		handler(data)
	})
}

// Close releases resources.
func (g *GenericRemote) Close(ctx context.Context) error {
	if g.dataSub != nil {
		g.dataSub.Unsubscribe()
	}
	if g.cancel != nil {
		g.cancel()
	}
	close(g.done)
	return nil
}

// Name returns the component name.
func (g *GenericRemote) Name() string {
	return g.descriptor.Name
}

// Type returns the component type.
func (g *GenericRemote) Type() string {
	return string(g.descriptor.Type)
}

// Subtype returns the component subtype.
func (g *GenericRemote) Subtype() string {
	return g.descriptor.Subtype
}

// Descriptor returns the service descriptor.
func (g *GenericRemote) Descriptor() mesh.ServiceDescriptor {
	return g.descriptor
}

// CommandSubject returns the command subject.
func (g *GenericRemote) CommandSubject() string {
	return g.cmdSubject
}

// DataSubject returns the data subject.
func (g *GenericRemote) DataSubject() string {
	return g.dataSubject
}

// PublishSubjects returns all subjects this component publishes to.
func (g *GenericRemote) PublishSubjects() []string {
	return g.descriptor.Publishes
}

// SubscribeSubjects returns all subjects this component subscribes to.
func (g *GenericRemote) SubscribeSubjects() []string {
	return g.descriptor.Subscribes
}
