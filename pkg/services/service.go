// Package service provides request/reply service patterns for Gorai.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

// Server handles service requests.
type Server[Req, Resp proto.Message] struct {
	nc      *nats.Conn
	subject string
	sub     *nats.Subscription
}

// Client makes service requests.
type Client[Req, Resp proto.Message] struct {
	nc      *nats.Conn
	subject string
	timeout time.Duration
}

// NATSGetter is an interface for types that provide a NATS connection.
type NATSGetter interface {
	NATS() *nats.Conn
}

// Handler processes a service request and returns a response.
type Handler[Req, Resp proto.Message] func(ctx context.Context, req Req) (Resp, error)

// NewServer creates a new service server.
func NewServer[Req, Resp proto.Message](n NATSGetter, subject string, handler Handler[Req, Resp]) (*Server[Req, Resp], error) {
	s := &Server[Req, Resp]{
		nc:      n.NATS(),
		subject: subject,
	}

	sub, err := s.nc.Subscribe(subject, func(m *nats.Msg) {
		var req Req
		req = req.ProtoReflect().New().Interface().(Req)

		if err := proto.Unmarshal(m.Data, req); err != nil {
			return
		}

		resp, err := handler(context.Background(), req)
		if err != nil {
			return
		}

		data, err := proto.Marshal(resp)
		if err != nil {
			return
		}

		m.Respond(data)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	s.sub = sub
	return s, nil
}

// Close stops the server.
func (s *Server[Req, Resp]) Close() error {
	if s.sub != nil {
		return s.sub.Unsubscribe()
	}
	return nil
}

// NewClient creates a new service client.
func NewClient[Req, Resp proto.Message](n NATSGetter, subject string) *Client[Req, Resp] {
	return &Client[Req, Resp]{
		nc:      n.NATS(),
		subject: subject,
		timeout: 5 * time.Second,
	}
}

// WithTimeout sets the request timeout.
func (c *Client[Req, Resp]) WithTimeout(d time.Duration) *Client[Req, Resp] {
	c.timeout = d
	return c
}

// Call makes a service request.
func (c *Client[Req, Resp]) Call(ctx context.Context, req Req) (Resp, error) {
	var resp Resp

	data, err := proto.Marshal(req)
	if err != nil {
		return resp, fmt.Errorf("failed to marshal request: %w", err)
	}

	msg, err := c.nc.Request(c.subject, data, c.timeout)
	if err != nil {
		return resp, fmt.Errorf("request failed: %w", err)
	}

	resp = resp.ProtoReflect().New().Interface().(Resp)
	if err := proto.Unmarshal(msg.Data, resp); err != nil {
		return resp, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return resp, nil
}
