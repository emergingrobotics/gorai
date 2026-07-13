package ncp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/emergingrobotics/gorai/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// commandTimeout bounds how long a single capability invocation may run.
const commandTimeout = 5 * time.Second

// capability adapts a concrete component to the NCP method/state model.
type capability struct {
	// dispatch invokes a method and returns a result map.
	dispatch func(ctx context.Context, req Request) (map[string]any, error)

	// state returns the current capability snapshot for `.state` publishing.
	state func(ctx context.Context) (map[string]any, error)
}

// Server exposes local components over NCP subjects on a NATS connection.
type Server struct {
	nc       *nats.Conn
	subjects *subjects.Builder
	logger   *slog.Logger

	mu   sync.Mutex
	subs []*nats.Subscription
}

// NewServer creates an NCP server bound to the given NATS connection and
// subject builder.
func NewServer(nc *nats.Conn, subj *subjects.Builder, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		nc:       nc,
		subjects: subj,
		logger:   logger.With("subsystem", "ncp"),
	}
}

// Expose inspects a component and, if it implements a known capability,
// subscribes to its command subject. It returns true when the component was
// exposed.
func (s *Server) Expose(name string, component any) (bool, error) {
	if s.nc == nil {
		return false, fmt.Errorf("ncp server has no NATS connection")
	}

	cap, ok := adapterFor(component)
	if !ok {
		return false, nil
	}

	subject := s.subjects.ComponentCommand(name)
	stateSubject := s.subjects.ComponentState(name)

	sub, err := s.nc.Subscribe(subject, s.makeHandler(name, stateSubject, cap))
	if err != nil {
		return false, fmt.Errorf("failed to subscribe %s: %w", subject, err)
	}

	s.mu.Lock()
	s.subs = append(s.subs, sub)
	s.mu.Unlock()

	// Publish an initial state snapshot so late subscribers see current values.
	s.publishState(stateSubject, cap)
	return true, nil
}

// makeHandler builds the NATS message handler for a capability.
func (s *Server) makeHandler(name, stateSubject string, cap *capability) nats.MsgHandler {
	return func(msg *nats.Msg) {
		ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
		defer cancel()

		var resp Response
		var req Request
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			resp = Response{Error: fmt.Sprintf("invalid request: %v", err)}
		} else if result, err := cap.dispatch(ctx, req); err != nil {
			s.logger.Warn("NCP command failed", "component", name, "method", req.Method, "error", err)
			resp = Response{Error: err.Error()}
		} else {
			resp = Response{OK: true, Result: result}
		}

		if msg.Reply != "" {
			data, err := json.Marshal(resp)
			if err != nil {
				s.logger.Warn("failed to marshal NCP response", "error", err)
				return
			}
			if err := msg.Respond(data); err != nil {
				s.logger.Warn("failed to respond to NCP command", "error", err)
			}
		}

		if resp.OK {
			s.publishState(stateSubject, cap)
		}
	}
}

// publishState publishes the current capability snapshot to its state subject.
func (s *Server) publishState(subject string, cap *capability) {
	if cap.state == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	state, err := cap.state(ctx)
	if err != nil {
		return
	}
	data, err := json.Marshal(state)
	if err != nil {
		return
	}
	if err := s.nc.Publish(subject, data); err != nil {
		s.logger.Debug("failed to publish NCP state", "subject", subject, "error", err)
	}
}

// Close unsubscribes all exposed capabilities.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sub := range s.subs {
		_ = sub.Unsubscribe()
	}
	s.subs = nil
	return nil
}
