package nws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorai/gorai/pkg/resource"
	"github.com/nats-io/nats.go"
)

// ResourceClient provides remote access to a resource over NATS.
type ResourceClient struct {
	nc      *nats.Conn
	name    resource.Name
	subject string
	timeout time.Duration
}

// ClientOption configures a ResourceClient.
type ClientOption func(*ResourceClient)

// WithTimeout sets the request timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *ResourceClient) {
		c.timeout = d
	}
}

// Connect creates a client for a remote resource.
func Connect(nc *nats.Conn, name resource.Name, opts ...ClientOption) (*ResourceClient, error) {
	if nc == nil {
		return nil, fmt.Errorf("NATS connection is required")
	}

	c := &ResourceClient{
		nc:      nc,
		name:    name,
		subject: name.Topic() + ".rpc",
		timeout: 5 * time.Second,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// ConnectWithNode creates a client using a node's NATS connection.
func ConnectWithNode(n NATSGetter, name resource.Name, opts ...ClientOption) (*ResourceClient, error) {
	nc := n.NATS()
	if nc == nil {
		return nil, fmt.Errorf("node has no NATS connection")
	}
	return Connect(nc, name, opts...)
}

// Call invokes a method on the remote resource.
func (c *ResourceClient) Call(ctx context.Context, method string, args map[string]any) (any, error) {
	req := Request{
		Method: method,
		Args:   args,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use request timeout from context or default
	timeout := c.timeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}

	msg, err := c.nc.Request(c.subject, data, timeout)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	var resp Response
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("remote error: %s", resp.Error)
	}

	return resp.Result, nil
}

// Name returns the resource name.
func (c *ResourceClient) Name() resource.Name {
	return c.name
}

// Subject returns the NATS subject for this client.
func (c *ResourceClient) Subject() string {
	return c.subject
}

// Reconfigure calls the Reconfigure method on the remote resource.
func (c *ResourceClient) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	_, err := c.Call(ctx, "Reconfigure", map[string]any{
		"arg1": conf.Attributes,
	})
	return err
}

// DoCommand calls the DoCommand method on the remote resource.
func (c *ResourceClient) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	result, err := c.Call(ctx, "DoCommand", map[string]any{
		"arg1": cmd,
	})
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	if m, ok := result.(map[string]any); ok {
		return m, nil
	}

	return map[string]any{"result": result}, nil
}

// Close releases client resources (no-op for client).
func (c *ResourceClient) Close(ctx context.Context) error {
	// Nothing to close on client side
	return nil
}

// Ping checks if the remote resource is available.
func (c *ResourceClient) Ping(ctx context.Context) error {
	_, err := c.Call(ctx, "Name", nil)
	return err
}

// WatchHealth subscribes to health updates from the remote resource.
func (c *ResourceClient) WatchHealth(ctx context.Context) (<-chan map[string]any, error) {
	healthSubject := c.subject + ".health"

	ch := make(chan map[string]any, 10)

	sub, err := c.nc.Subscribe(healthSubject, func(msg *nats.Msg) {
		var status map[string]any
		if err := json.Unmarshal(msg.Data, &status); err != nil {
			return
		}

		select {
		case ch <- status:
		default:
			// Channel full, drop message
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to health: %w", err)
	}

	// Unsubscribe when context is done
	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
		close(ch)
	}()

	return ch, nil
}

// Verify that ResourceClient implements resource.Resource
var _ resource.Resource = (*ResourceClient)(nil)
