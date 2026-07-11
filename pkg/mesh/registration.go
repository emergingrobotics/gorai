package mesh

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// Registration represents an active service registration with automatic heartbeat.
type Registration struct {
	client   *Client
	desc     ServiceDescriptor
	cancel   context.CancelFunc
	done     chan struct{}
	mu       sync.RWMutex
	status   ServiceStatus
	logger   *slog.Logger
	channels []ChannelDescriptor
}

// RegistrationOption configures a registration.
type RegistrationOption func(*registrationConfig)

type registrationConfig struct {
	heartbeatInterval time.Duration
	logger            *slog.Logger
	channels          []ChannelDescriptor
	announceJoin      bool
	announceLeave     bool
}

// WithHeartbeatInterval sets the heartbeat interval.
func WithHeartbeatInterval(d time.Duration) RegistrationOption {
	return func(c *registrationConfig) {
		c.heartbeatInterval = d
	}
}

// WithRegistrationLogger sets the logger for the registration.
func WithRegistrationLogger(logger *slog.Logger) RegistrationOption {
	return func(c *registrationConfig) {
		c.logger = logger
	}
}

// WithChannels registers channels along with the service.
func WithChannels(channels ...ChannelDescriptor) RegistrationOption {
	return func(c *registrationConfig) {
		c.channels = append(c.channels, channels...)
	}
}

// WithAnnouncements enables join/leave announcements.
func WithAnnouncements(join, leave bool) RegistrationOption {
	return func(c *registrationConfig) {
		c.announceJoin = join
		c.announceLeave = leave
	}
}

// Register registers a service and starts automatic heartbeat.
func (c *Client) Register(ctx context.Context, desc ServiceDescriptor, opts ...RegistrationOption) (*Registration, error) {
	cfg := &registrationConfig{
		heartbeatInterval: DefaultHeartbeatInterval,
		logger:            c.logger,
		announceJoin:      true,
		announceLeave:     true,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Generate instance ID if not provided
	if desc.ID == "" {
		desc.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	desc.StartedAt = now
	desc.LastSeen = now
	desc.Status = StatusHealthy

	// Set host and PID if not provided
	if desc.Host == "" {
		desc.Host, _ = os.Hostname()
	}
	if desc.PID == 0 {
		desc.PID = os.Getpid()
	}

	// Store in KV
	key := ServiceKey(desc.RobotID, desc.Name, desc.ID)
	data, err := json.Marshal(desc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal service descriptor: %w", err)
	}

	if _, err := c.kv.Services().Put(ctx, key, data); err != nil {
		return nil, fmt.Errorf("failed to store service registration: %w", err)
	}

	// Create registration
	regCtx, cancel := context.WithCancel(context.Background())
	reg := &Registration{
		client:   c,
		desc:     desc,
		cancel:   cancel,
		done:     make(chan struct{}),
		status:   StatusHealthy,
		logger:   cfg.logger,
		channels: cfg.channels,
	}

	// Register channels
	for i := range cfg.channels {
		ch := &cfg.channels[i]
		ch.Publisher = desc.ID
		ch.RobotID = desc.RobotID
		if ch.CreatedAt.IsZero() {
			ch.CreatedAt = now
		}
		ch.UpdatedAt = now

		if err := c.RegisterChannel(ctx, *ch); err != nil {
			cfg.logger.Warn("failed to register channel", "subject", ch.Subject, "error", err)
		}
	}

	// Announce join
	if cfg.announceJoin {
		if err := c.announce(ctx, "join", desc); err != nil {
			cfg.logger.Warn("failed to announce join", "error", err)
		}
	}

	// Start heartbeat goroutine
	go reg.heartbeatLoop(regCtx, cfg.heartbeatInterval, cfg.announceLeave)

	cfg.logger.Info("service registered",
		"id", desc.ID,
		"name", desc.Name,
		"robot", desc.RobotID,
		"type", desc.Type,
		"subtype", desc.Subtype,
	)

	return reg, nil
}

// heartbeatLoop sends periodic heartbeats and refreshes the KV entry.
func (r *Registration) heartbeatLoop(ctx context.Context, interval time.Duration, announceLeave bool) {
	defer close(r.done)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Deregistering
			r.deregister(announceLeave)
			return
		case <-ticker.C:
			if err := r.sendHeartbeat(ctx); err != nil {
				r.logger.Warn("heartbeat failed", "error", err)
			}
		}
	}
}

// sendHeartbeat refreshes the service registration and publishes heartbeat.
func (r *Registration) sendHeartbeat(ctx context.Context) error {
	r.mu.Lock()
	r.desc.LastSeen = time.Now()
	r.desc.Status = r.status
	descCopy := r.desc
	r.mu.Unlock()

	// Refresh KV entry (resets TTL)
	key := ServiceKey(descCopy.RobotID, descCopy.Name, descCopy.ID)
	data, err := json.Marshal(descCopy)
	if err != nil {
		return fmt.Errorf("failed to marshal descriptor: %w", err)
	}

	if _, err := r.client.kv.Services().Put(ctx, key, data); err != nil {
		return fmt.Errorf("failed to refresh registration: %w", err)
	}

	// Publish heartbeat message
	hb := HeartbeatMessage{
		ServiceID: descCopy.ID,
		Status:    descCopy.Status,
		Timestamp: descCopy.LastSeen,
	}

	hbData, err := json.Marshal(hb)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat: %w", err)
	}
	subject := HeartbeatSubject(descCopy.ID)
	if err := r.client.nc.Publish(subject, hbData); err != nil {
		return fmt.Errorf("failed to publish heartbeat: %w", err)
	}

	return nil
}

// deregister removes the service from the registry.
func (r *Registration) deregister(announceLeave bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	r.mu.RLock()
	descCopy := r.desc
	r.mu.RUnlock()

	key := ServiceKey(descCopy.RobotID, descCopy.Name, descCopy.ID)

	// Delete from KV
	if err := r.client.kv.Services().Delete(ctx, key); err != nil {
		r.logger.Warn("failed to delete registration", "error", err)
	}

	// Remove channels owned by this service
	for _, ch := range r.channels {
		if err := r.client.DeregisterChannel(ctx, ch.Subject); err != nil {
			r.logger.Warn("failed to deregister channel", "subject", ch.Subject, "error", err)
		}
	}

	// Announce leave
	if announceLeave {
		if err := r.client.announce(ctx, "leave", descCopy); err != nil {
			r.logger.Warn("failed to announce leave", "error", err)
		}
	}

	r.logger.Info("service deregistered", "id", descCopy.ID, "name", descCopy.Name)
}

// Deregister stops the heartbeat and removes the service from the registry.
func (r *Registration) Deregister() {
	r.cancel()
	<-r.done
}

// SetStatus updates the service status.
func (r *Registration) SetStatus(status ServiceStatus) {
	r.mu.Lock()
	r.status = status
	r.mu.Unlock()
}

// Status returns the current service status.
func (r *Registration) Status() ServiceStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.status
}

// Descriptor returns a copy of the service descriptor.
func (r *Registration) Descriptor() ServiceDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.desc
}

// ID returns the service instance ID.
func (r *Registration) ID() string {
	return r.desc.ID
}

// announce publishes a join or leave announcement.
func (c *Client) announce(ctx context.Context, typ string, desc ServiceDescriptor) error {
	ann := Announcement{
		Type:      typ,
		Service:   desc,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(ann)
	if err != nil {
		return err
	}

	return c.nc.Publish(SubjectAnnounce, data)
}

// RegisterChannel registers a channel in the registry.
func (c *Client) RegisterChannel(ctx context.Context, ch ChannelDescriptor) error {
	if ch.CreatedAt.IsZero() {
		ch.CreatedAt = time.Now()
	}
	ch.UpdatedAt = time.Now()

	data, err := json.Marshal(ch)
	if err != nil {
		return fmt.Errorf("failed to marshal channel: %w", err)
	}

	key := ChannelKey(ch.Subject)
	if _, err := c.kv.Channels().Put(ctx, key, data); err != nil {
		return fmt.Errorf("failed to store channel: %w", err)
	}

	c.logger.Debug("channel registered", "subject", ch.Subject)
	return nil
}

// DeregisterChannel removes a channel from the registry.
func (c *Client) DeregisterChannel(ctx context.Context, subject string) error {
	key := ChannelKey(subject)
	if err := c.kv.Channels().Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete channel: %w", err)
	}
	return nil
}

// RegisterSchema registers a message schema.
func (c *Client) RegisterSchema(ctx context.Context, schema SchemaDescriptor) error {
	if schema.CreatedAt.IsZero() {
		schema.CreatedAt = time.Now()
	}
	schema.UpdatedAt = time.Now()

	data, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	key := SchemaKey(schema.Name, schema.Version)
	if _, err := c.kv.Schemas().Put(ctx, key, data); err != nil {
		return fmt.Errorf("failed to store schema: %w", err)
	}

	c.logger.Debug("schema registered", "name", schema.Name, "version", schema.Version)

	// Auto-register a discovery channel when the schema name is a capability
	// subject (gorai.<robot>.<cap>.<type>). Mesh + MCP discovery are channel-based,
	// so a schema alone leaves the capability invisible -- the "channels-vs-schemas
	// gap". Deriving the channel here closes it for every robot that follows the
	// "one schema per capability subject" convention, with no robot-side code.
	if ch, ok := capabilityChannelFromSchema(schema); ok {
		if err := c.RegisterChannel(ctx, ch); err != nil {
			c.logger.Warn("auto channel registration from schema failed", "subject", ch.Subject, "err", err)
		}
	}
	return nil
}

// capabilityChannelFromSchema derives a discovery channel from a schema whose
// Name is a capability subject: gorai.<robot>.<cap>.<type> with type in
// {data, command, state, event}. Returns ok=false for any other schema name
// (e.g. shared type schemas like "gorai.sensor.IMUReading"), so those are left
// as schema-only.
func capabilityChannelFromSchema(s SchemaDescriptor) (ChannelDescriptor, bool) {
	parts := strings.Split(s.Name, ".")
	if len(parts) < 4 || parts[0] != "gorai" {
		return ChannelDescriptor{}, false
	}
	typ := parts[len(parts)-1]
	switch typ {
	case "data", "command", "state", "event":
	default:
		return ChannelDescriptor{}, false
	}
	robot := parts[1]
	capName := strings.Join(parts[2:len(parts)-1], ".")
	if robot == "" || capName == "" {
		return ChannelDescriptor{}, false
	}
	dir := DirectionPub // robot publishes data/state/events
	if typ == "command" {
		dir = DirectionSub // robot receives commands
	}
	return ChannelDescriptor{
		Subject:     s.Name,
		Schema:      SchemaKey(s.Name, s.Version),
		RobotID:     robot,
		Direction:   dir,
		QoS:         QoSBestEffort,
		Description: s.Description,
	}, true
}

// BatchRegisterChannels registers multiple channels at once.
func (c *Client) BatchRegisterChannels(ctx context.Context, channels []ChannelDescriptor) error {
	for _, ch := range channels {
		if err := c.RegisterChannel(ctx, ch); err != nil {
			return err
		}
	}
	return nil
}

// QuickRegister is a convenience function for simple service registration.
func (c *Client) QuickRegister(ctx context.Context, robotID, name string, typ ServiceType, subtype, model string) (*Registration, error) {
	return c.Register(ctx, ServiceDescriptor{
		Name:    name,
		Type:    typ,
		Subtype: subtype,
		Model:   model,
		RobotID: robotID,
	})
}

// MustRegister registers a service and panics on error.
func (c *Client) MustRegister(ctx context.Context, desc ServiceDescriptor, opts ...RegistrationOption) *Registration {
	reg, err := c.Register(ctx, desc, opts...)
	if err != nil {
		panic(fmt.Sprintf("mesh: failed to register service: %v", err))
	}
	return reg
}

// ConnectAndRegister creates a mesh client and registers a service in one call.
func ConnectAndRegister(ctx context.Context, nc *nats.Conn, desc ServiceDescriptor, opts ...RegistrationOption) (*Client, *Registration, error) {
	client, err := NewClient(nc)
	if err != nil {
		return nil, nil, err
	}

	reg, err := client.Register(ctx, desc, opts...)
	if err != nil {
		return nil, nil, err
	}

	return client, reg, nil
}
