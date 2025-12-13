// Package nats provides a NATS client wrapper for Gorai.
package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

// Client wraps a NATS connection with Gorai-specific functionality.
type Client struct {
	conn   *nats.Conn
	logger *slog.Logger
}

// Config holds NATS connection configuration.
type Config struct {
	URL            string
	Name           string
	ConnectTimeout time.Duration
	ReconnectWait  time.Duration
	MaxReconnects  int
}

// DefaultConfig returns a default NATS configuration.
func DefaultConfig() *Config {
	return &Config{
		URL:            "nats://localhost:4222",
		Name:           "gorai-client",
		ConnectTimeout: 5 * time.Second,
		ReconnectWait:  2 * time.Second,
		MaxReconnects:  -1, // unlimited
	}
}

// Option configures a Client.
type Option func(*Client)

// WithLogger sets the logger for the client.
func WithLogger(logger *slog.Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}

// Connect creates a new NATS client connection.
func Connect(ctx context.Context, cfg *Config, opts ...Option) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	c := &Client{
		logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(c)
	}

	// Build NATS options
	natsOpts := []nats.Option{
		nats.Name(cfg.Name),
		nats.Timeout(cfg.ConnectTimeout),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				c.logger.Warn("NATS disconnected", "error", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			c.logger.Info("NATS reconnected", "url", nc.ConnectedUrl())
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			c.logger.Error("NATS error", "error", err, "subject", sub.Subject)
		}),
	}

	// Connect with context timeout
	var conn *nats.Conn
	var err error

	done := make(chan struct{})
	go func() {
		conn, err = nats.Connect(cfg.URL, natsOpts...)
		close(done)
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
		if err != nil {
			return nil, fmt.Errorf("failed to connect to NATS at %s: %w", cfg.URL, err)
		}
	}

	c.conn = conn
	c.logger.Info("Connected to NATS", "url", cfg.URL, "name", cfg.Name)

	return c, nil
}

// Close closes the NATS connection.
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Drain()
		c.conn.Close()
		c.logger.Info("NATS connection closed")
	}
}

// Conn returns the underlying NATS connection.
func (c *Client) Conn() *nats.Conn {
	return c.conn
}

// Publish publishes a message to the given subject.
func (c *Client) Publish(subject string, data []byte) error {
	return c.conn.Publish(subject, data)
}

// PublishJSON publishes a JSON-encoded message to the given subject.
func (c *Client) PublishJSON(subject string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return c.conn.Publish(subject, data)
}

// Subscribe subscribes to messages on the given subject.
func (c *Client) Subscribe(subject string, handler func(msg *nats.Msg)) (*nats.Subscription, error) {
	return c.conn.Subscribe(subject, handler)
}

// SubscribeJSON subscribes to JSON messages on the given subject.
// The handler receives the decoded message.
func (c *Client) SubscribeJSON(subject string, handler func(v json.RawMessage)) (*nats.Subscription, error) {
	return c.conn.Subscribe(subject, func(msg *nats.Msg) {
		handler(json.RawMessage(msg.Data))
	})
}

// Request sends a request and waits for a response.
func (c *Client) Request(subject string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	return c.conn.Request(subject, data, timeout)
}

// RequestJSON sends a JSON request and waits for a response.
func (c *Client) RequestJSON(subject string, req any, timeout time.Duration) (*nats.Msg, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	return c.conn.Request(subject, data, timeout)
}

// IsConnected returns true if connected to NATS.
func (c *Client) IsConnected() bool {
	return c.conn != nil && c.conn.IsConnected()
}

// Status returns the connection status.
func (c *Client) Status() nats.Status {
	if c.conn == nil {
		return nats.CLOSED
	}
	return c.conn.Status()
}
