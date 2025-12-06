// Package nws provides Network Wrapper Server/Client for remote resource access.
//
// The NWS/NWC pattern (borrowed from YARP) allows resources to be accessed
// transparently whether they are local or remote. An NWS wraps a local resource
// and exposes it over the network; an NWC provides the same interface but
// communicates with a remote NWS.
package nws

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
)

// Server wraps a local resource and exposes it over the network.
type Server struct {
	nc       *nats.Conn
	resource any
	subject  string
	sub      *nats.Subscription
}

// ServerOption configures a Server.
type ServerOption func(*Server)

// WithSubject sets the NATS subject for the server.
func WithSubject(subject string) ServerOption {
	return func(s *Server) {
		s.subject = subject
	}
}

// NewServer creates a new network wrapper server.
func NewServer(nc *nats.Conn, resource any, opts ...ServerOption) (*Server, error) {
	s := &Server{
		nc:       nc,
		resource: resource,
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.subject == "" {
		return nil, fmt.Errorf("subject is required")
	}

	// TODO: Implement reflection-based RPC handling
	// This would use NATS request/reply to handle method calls

	return s, nil
}

// Start begins serving requests.
func (s *Server) Start(ctx context.Context) error {
	// TODO: Subscribe to subject and handle RPC requests
	return nil
}

// Stop stops serving requests.
func (s *Server) Stop() error {
	if s.sub != nil {
		return s.sub.Unsubscribe()
	}
	return nil
}

// Client connects to a remote resource over the network.
type Client struct {
	nc      *nats.Conn
	subject string
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// NewClient creates a new network wrapper client.
func NewClient(nc *nats.Conn, subject string, opts ...ClientOption) *Client {
	c := &Client{
		nc:      nc,
		subject: subject,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Call invokes a remote method.
func (c *Client) Call(ctx context.Context, method string, args any) (any, error) {
	// TODO: Implement RPC call
	return nil, fmt.Errorf("not implemented")
}
