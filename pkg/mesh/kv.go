package mesh

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Bucket names for the mesh registry.
const (
	// BucketServices stores active service registrations.
	BucketServices = "gorai-services"
	// BucketChannels stores channel descriptors.
	BucketChannels = "gorai-channels"
	// BucketSchemas stores message schemas.
	BucketSchemas = "gorai-schemas"
)

// Well-known subjects for mesh communication.
const (
	// SubjectAnnounce is for service join/leave announcements.
	SubjectAnnounce = "gorai.mesh.announce"
	// SubjectHeartbeatPrefix is the prefix for heartbeat subjects.
	SubjectHeartbeatPrefix = "gorai.mesh.heartbeat"
)

// Default configuration values.
const (
	// DefaultServiceTTL is how long a service registration lives without heartbeat.
	DefaultServiceTTL = 30 * time.Second
	// DefaultHeartbeatInterval is how often services send heartbeats.
	DefaultHeartbeatInterval = 10 * time.Second
	// DefaultStaleThreshold is when a service is considered stale.
	DefaultStaleThreshold = 20 * time.Second
)

// KVManager manages the NATS KV buckets for the mesh registry.
type KVManager struct {
	js       jetstream.JetStream
	services jetstream.KeyValue
	channels jetstream.KeyValue
	schemas  jetstream.KeyValue
}

// KVConfig holds configuration for KV bucket creation.
type KVConfig struct {
	// ServiceTTL is the TTL for service entries (default 30s).
	ServiceTTL time.Duration
	// MaxServiceHistory is the max history per service key (default 1).
	MaxServiceHistory int
	// Replicas is the number of replicas for HA (default 1).
	Replicas int
}

// DefaultKVConfig returns the default KV configuration.
func DefaultKVConfig() *KVConfig {
	return &KVConfig{
		ServiceTTL:        DefaultServiceTTL,
		MaxServiceHistory: 1,
		Replicas:          1,
	}
}

// NewKVManager creates a new KV manager, initializing buckets if needed.
func NewKVManager(ctx context.Context, nc *nats.Conn, cfg *KVConfig) (*KVManager, error) {
	if cfg == nil {
		cfg = DefaultKVConfig()
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	mgr := &KVManager{js: js}

	// Initialize services bucket (with TTL for automatic expiry)
	mgr.services, err = mgr.getOrCreateBucket(ctx, BucketServices, &jetstream.KeyValueConfig{
		Bucket:      BucketServices,
		Description: "Active Gorai service registrations",
		TTL:         cfg.ServiceTTL,
		History:     uint8(cfg.MaxServiceHistory),
		Replicas:    cfg.Replicas,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize services bucket: %w", err)
	}

	// Initialize channels bucket (persistent, no TTL)
	mgr.channels, err = mgr.getOrCreateBucket(ctx, BucketChannels, &jetstream.KeyValueConfig{
		Bucket:      BucketChannels,
		Description: "Gorai channel/subject descriptors",
		History:     5, // Keep some history for channels
		Replicas:    cfg.Replicas,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize channels bucket: %w", err)
	}

	// Initialize schemas bucket (persistent, versioned)
	mgr.schemas, err = mgr.getOrCreateBucket(ctx, BucketSchemas, &jetstream.KeyValueConfig{
		Bucket:      BucketSchemas,
		Description: "Gorai message schemas",
		History:     10, // Keep schema history for versioning
		Replicas:    cfg.Replicas,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize schemas bucket: %w", err)
	}

	return mgr, nil
}

// getOrCreateBucket gets an existing bucket or creates it if it doesn't exist.
func (m *KVManager) getOrCreateBucket(ctx context.Context, name string, cfg *jetstream.KeyValueConfig) (jetstream.KeyValue, error) {
	kv, err := m.js.KeyValue(ctx, name)
	if err == nil {
		return kv, nil
	}

	if !errors.Is(err, jetstream.ErrBucketNotFound) {
		return nil, fmt.Errorf("failed to get bucket %s: %w", name, err)
	}

	// Bucket doesn't exist, create it
	kv, err = m.js.CreateKeyValue(ctx, *cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket %s: %w", name, err)
	}

	return kv, nil
}

// Services returns the services KV bucket.
func (m *KVManager) Services() jetstream.KeyValue {
	return m.services
}

// Channels returns the channels KV bucket.
func (m *KVManager) Channels() jetstream.KeyValue {
	return m.channels
}

// Schemas returns the schemas KV bucket.
func (m *KVManager) Schemas() jetstream.KeyValue {
	return m.schemas
}

// JetStream returns the underlying JetStream context.
func (m *KVManager) JetStream() jetstream.JetStream {
	return m.js
}

// ServiceKey generates the KV key for a service.
// Format: <robot_id>/<service_name>/<instance_id>
func ServiceKey(robotID, serviceName, instanceID string) string {
	return fmt.Sprintf("%s/%s/%s", robotID, serviceName, instanceID)
}

// ServiceKeyShort generates a shorter key without instance ID.
// Format: <robot_id>/<service_name>
func ServiceKeyShort(robotID, serviceName string) string {
	return fmt.Sprintf("%s/%s", robotID, serviceName)
}

// ChannelKey generates the KV key for a channel.
// Uses the subject directly, replacing . with / for KV compatibility.
func ChannelKey(subject string) string {
	// NATS KV keys can contain dots, so we use the subject directly
	return subject
}

// SchemaKey generates the KV key for a schema.
// Format: <name>/<version>
func SchemaKey(name, version string) string {
	return fmt.Sprintf("%s/%s", name, version)
}

// HeartbeatSubject returns the heartbeat subject for a service.
func HeartbeatSubject(serviceID string) string {
	return fmt.Sprintf("%s.%s", SubjectHeartbeatPrefix, serviceID)
}

// DeleteBuckets removes all mesh KV buckets. Use with caution.
func (m *KVManager) DeleteBuckets(ctx context.Context) error {
	var errs []error

	if err := m.js.DeleteKeyValue(ctx, BucketServices); err != nil && !errors.Is(err, jetstream.ErrBucketNotFound) {
		errs = append(errs, fmt.Errorf("failed to delete services bucket: %w", err))
	}

	if err := m.js.DeleteKeyValue(ctx, BucketChannels); err != nil && !errors.Is(err, jetstream.ErrBucketNotFound) {
		errs = append(errs, fmt.Errorf("failed to delete channels bucket: %w", err))
	}

	if err := m.js.DeleteKeyValue(ctx, BucketSchemas); err != nil && !errors.Is(err, jetstream.ErrBucketNotFound) {
		errs = append(errs, fmt.Errorf("failed to delete schemas bucket: %w", err))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
