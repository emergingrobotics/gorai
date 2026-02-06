package mesh

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
)

// Client is the main mesh client for service discovery and registration.
type Client struct {
	nc     *nats.Conn
	kv     *KVManager
	logger *slog.Logger
}

// ClientOption configures a Client.
type ClientOption func(*clientConfig)

type clientConfig struct {
	logger   *slog.Logger
	kvConfig *KVConfig
}

// WithClientLogger sets the logger for the client.
func WithClientLogger(logger *slog.Logger) ClientOption {
	return func(c *clientConfig) {
		c.logger = logger
	}
}

// WithKVConfig sets the KV configuration.
func WithKVConfig(cfg *KVConfig) ClientOption {
	return func(c *clientConfig) {
		c.kvConfig = cfg
	}
}

// NewClient creates a new mesh client.
func NewClient(nc *nats.Conn, opts ...ClientOption) (*Client, error) {
	cfg := &clientConfig{
		logger:   slog.Default(),
		kvConfig: DefaultKVConfig(),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	kv, err := NewKVManager(context.Background(), nc, cfg.kvConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize KV manager: %w", err)
	}

	return &Client{
		nc:     nc,
		kv:     kv,
		logger: cfg.logger,
	}, nil
}

// NewClientWithContext creates a new mesh client with context.
func NewClientWithContext(ctx context.Context, nc *nats.Conn, opts ...ClientOption) (*Client, error) {
	cfg := &clientConfig{
		logger:   slog.Default(),
		kvConfig: DefaultKVConfig(),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	kv, err := NewKVManager(ctx, nc, cfg.kvConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize KV manager: %w", err)
	}

	return &Client{
		nc:     nc,
		kv:     kv,
		logger: cfg.logger,
	}, nil
}

// KV returns the underlying KV manager.
func (c *Client) KV() *KVManager {
	return c.kv
}

// Conn returns the underlying NATS connection.
func (c *Client) Conn() *nats.Conn {
	return c.nc
}

// Close cleans up client resources. Note: does not close the NATS connection.
func (c *Client) Close() error {
	// Nothing to clean up currently; NATS connection is managed externally
	return nil
}

// Ping checks connectivity to NATS.
func (c *Client) Ping() error {
	return c.nc.Flush()
}

// IsConnected returns true if connected to NATS.
func (c *Client) IsConnected() bool {
	return c.nc.IsConnected()
}

// Connect creates a NATS connection and mesh client in one call.
func Connect(ctx context.Context, url string, opts ...ClientOption) (*Client, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS at %s: %w", url, err)
	}

	client, err := NewClientWithContext(ctx, nc, opts...)
	if err != nil {
		nc.Close()
		return nil, err
	}

	return client, nil
}

// MustConnect creates a mesh client and panics on error.
func MustConnect(ctx context.Context, url string, opts ...ClientOption) *Client {
	client, err := Connect(ctx, url, opts...)
	if err != nil {
		panic(fmt.Sprintf("mesh: failed to connect: %v", err))
	}
	return client
}

// DefaultURL is the default NATS server URL.
const DefaultURL = "nats://localhost:4222"

// ConnectDefault connects to the default NATS server.
func ConnectDefault(ctx context.Context, opts ...ClientOption) (*Client, error) {
	return Connect(ctx, DefaultURL, opts...)
}

// Reset clears all mesh data. Use with caution - primarily for testing.
func (c *Client) Reset(ctx context.Context) error {
	return c.kv.DeleteBuckets(ctx)
}
