package action

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	actionpb "github.com/emergingrobotics/gorai/api/gen/gorai/action"
	"github.com/emergingrobotics/gorai/api/gen/gorai/std"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

// NATSGetter provides access to NATS connection.
type NATSGetter interface {
	NATS() *nats.Conn
}

// GoalHandle represents an active goal being executed.
type GoalHandle[GoalT, FeedbackT, ResultT proto.Message] struct {
	ID       string
	Goal     GoalT
	Status   actionpb.Status
	ctx      context.Context
	cancel   context.CancelFunc
	feedback chan FeedbackT
	server   *Server[GoalT, FeedbackT, ResultT]
	mu       sync.Mutex
}

// SendFeedback sends feedback for this goal.
func (h *GoalHandle[GoalT, FeedbackT, ResultT]) SendFeedback(fb FeedbackT) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.Status != actionpb.Status_STATUS_ACTIVE {
		return fmt.Errorf("goal %s is not active", h.ID)
	}

	return h.server.publishFeedback(h.ID, fb)
}

// SetSucceeded marks the goal as succeeded and publishes the result.
func (h *GoalHandle[GoalT, FeedbackT, ResultT]) SetSucceeded(result ResultT) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.Status = actionpb.Status_STATUS_SUCCEEDED
	return h.server.publishResult(h.ID, h.Status, result)
}

// SetAborted marks the goal as aborted and publishes the result.
func (h *GoalHandle[GoalT, FeedbackT, ResultT]) SetAborted(result ResultT) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.Status = actionpb.Status_STATUS_ABORTED
	h.cancel()
	return h.server.publishResult(h.ID, h.Status, result)
}

// SetCanceled marks the goal as preempted and publishes the result.
func (h *GoalHandle[GoalT, FeedbackT, ResultT]) SetCanceled(result ResultT) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.Status = actionpb.Status_STATUS_PREEMPTED
	return h.server.publishResult(h.ID, h.Status, result)
}

// IsCanceling returns true if cancellation was requested.
func (h *GoalHandle[GoalT, FeedbackT, ResultT]) IsCanceling() bool {
	select {
	case <-h.ctx.Done():
		return true
	default:
		return false
	}
}

// Context returns the goal's context.
func (h *GoalHandle[GoalT, FeedbackT, ResultT]) Context() context.Context {
	return h.ctx
}

// ServerHandler processes action goals.
type ServerHandler[GoalT, FeedbackT, ResultT proto.Message] func(
	ctx context.Context,
	handle *GoalHandle[GoalT, FeedbackT, ResultT],
)

// serverOptions holds server configuration.
type serverOptions struct {
	logger *slog.Logger
}

// ServerOption configures a server.
type ServerOption func(*serverOptions)

// WithServerLogger sets the logger for the server.
func WithServerLogger(logger *slog.Logger) ServerOption {
	return func(o *serverOptions) {
		o.logger = logger
	}
}

// Server handles action requests.
type Server[GoalT, FeedbackT, ResultT proto.Message] struct {
	nc           *nats.Conn
	name         string
	handler      ServerHandler[GoalT, FeedbackT, ResultT]
	goals        map[string]*GoalHandle[GoalT, FeedbackT, ResultT]
	goalSub      *nats.Subscription
	cancelSub    *nats.Subscription
	logger       *slog.Logger
	mu           sync.RWMutex
	closed       bool
}

// NewServer creates a new action server.
func NewServer[GoalT, FeedbackT, ResultT proto.Message](
	n NATSGetter,
	name string,
	handler ServerHandler[GoalT, FeedbackT, ResultT],
	opts ...ServerOption,
) (*Server[GoalT, FeedbackT, ResultT], error) {
	nc := n.NATS()
	if nc == nil {
		return nil, fmt.Errorf("no NATS connection")
	}

	options := serverOptions{
		logger: slog.Default(),
	}
	for _, opt := range opts {
		opt(&options)
	}

	s := &Server[GoalT, FeedbackT, ResultT]{
		nc:      nc,
		name:    name,
		handler: handler,
		goals:   make(map[string]*GoalHandle[GoalT, FeedbackT, ResultT]),
		logger:  options.logger,
	}

	// Subscribe to goals
	goalTopic := fmt.Sprintf("action.%s.goal", name)
	goalSub, err := nc.Subscribe(goalTopic, s.handleGoal)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to goals: %w", err)
	}
	s.goalSub = goalSub

	// Subscribe to cancel requests
	cancelTopic := fmt.Sprintf("action.%s.cancel", name)
	cancelSub, err := nc.Subscribe(cancelTopic, s.handleCancel)
	if err != nil {
		goalSub.Unsubscribe()
		return nil, fmt.Errorf("failed to subscribe to cancel: %w", err)
	}
	s.cancelSub = cancelSub

	s.logger.Info("action server started", "name", name)
	return s, nil
}

// handleGoal processes incoming goal requests.
func (s *Server[GoalT, FeedbackT, ResultT]) handleGoal(msg *nats.Msg) {
	// Unmarshal goal
	var goal GoalT
	goal = goal.ProtoReflect().New().Interface().(GoalT)

	if err := proto.Unmarshal(msg.Data, goal); err != nil {
		s.logger.Error("failed to unmarshal goal", "error", err)
		return
	}

	// Generate goal ID
	goalID := fmt.Sprintf("%s-%d", s.name, time.Now().UnixNano())

	// Create goal handle
	ctx, cancel := context.WithCancel(context.Background())
	handle := &GoalHandle[GoalT, FeedbackT, ResultT]{
		ID:       goalID,
		Goal:     goal,
		Status:   actionpb.Status_STATUS_PENDING,
		ctx:      ctx,
		cancel:   cancel,
		feedback: make(chan FeedbackT, 10),
		server:   s,
	}

	s.mu.Lock()
	s.goals[goalID] = handle
	s.mu.Unlock()

	// Reply with goal ID if reply subject is set
	if msg.Reply != "" {
		goalIDProto := &actionpb.GoalID{
			Id: goalID,
			Stamp: &std.Timestamp{
				Seconds: time.Now().Unix(),
				Nanos:   int32(time.Now().Nanosecond()),
			},
		}
		data, _ := proto.Marshal(goalIDProto)
		msg.Respond(data)
	}

	// Update status to active
	handle.mu.Lock()
	handle.Status = actionpb.Status_STATUS_ACTIVE
	handle.mu.Unlock()

	s.publishStatus(handle)

	// Execute handler in goroutine
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.goals, goalID)
			s.mu.Unlock()
			cancel()
		}()

		s.handler(ctx, handle)

		// If handler returned without setting a terminal status, mark as succeeded
		handle.mu.Lock()
		if handle.Status == actionpb.Status_STATUS_ACTIVE {
			handle.Status = actionpb.Status_STATUS_SUCCEEDED
		}
		handle.mu.Unlock()
	}()
}

// handleCancel processes cancel requests.
func (s *Server[GoalT, FeedbackT, ResultT]) handleCancel(msg *nats.Msg) {
	var cancelReq actionpb.CancelGoal
	if err := proto.Unmarshal(msg.Data, &cancelReq); err != nil {
		s.logger.Error("failed to unmarshal cancel request", "error", err)
		return
	}

	response := &actionpb.CancelGoalResponse{
		ReturnCode:     actionpb.CancelGoalResponse_CODE_NONE,
		GoalsCanceling: []*actionpb.GoalID{},
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if cancelReq.GoalId != nil && cancelReq.GoalId.Id != "" {
		// Cancel specific goal
		if handle, ok := s.goals[cancelReq.GoalId.Id]; ok {
			handle.mu.Lock()
			if handle.Status == actionpb.Status_STATUS_ACTIVE {
				handle.Status = actionpb.Status_STATUS_PREEMPTING
				handle.cancel()
				response.GoalsCanceling = append(response.GoalsCanceling, &actionpb.GoalID{Id: handle.ID})
			}
			handle.mu.Unlock()
		} else {
			response.ReturnCode = actionpb.CancelGoalResponse_CODE_UNKNOWN_GOAL
		}
	} else {
		// Cancel all goals
		for _, handle := range s.goals {
			handle.mu.Lock()
			if handle.Status == actionpb.Status_STATUS_ACTIVE {
				handle.Status = actionpb.Status_STATUS_PREEMPTING
				handle.cancel()
				response.GoalsCanceling = append(response.GoalsCanceling, &actionpb.GoalID{Id: handle.ID})
			}
			handle.mu.Unlock()
		}
	}

	if msg.Reply != "" {
		data, _ := proto.Marshal(response)
		msg.Respond(data)
	}
}

// publishStatus publishes a status update.
func (s *Server[GoalT, FeedbackT, ResultT]) publishStatus(handle *GoalHandle[GoalT, FeedbackT, ResultT]) error {
	status := &actionpb.GoalStatus{
		GoalId: &actionpb.GoalID{Id: handle.ID},
		Status: handle.Status,
	}

	statusArray := &actionpb.GoalStatusArray{
		Header: &std.Header{
			Stamp: &std.Timestamp{
				Seconds: time.Now().Unix(),
				Nanos:   int32(time.Now().Nanosecond()),
			},
		},
		StatusList: []*actionpb.GoalStatus{status},
	}

	data, err := proto.Marshal(statusArray)
	if err != nil {
		return err
	}

	topic := fmt.Sprintf("action.%s.status", s.name)
	return s.nc.Publish(topic, data)
}

// publishFeedback publishes feedback for a goal.
func (s *Server[GoalT, FeedbackT, ResultT]) publishFeedback(goalID string, feedback FeedbackT) error {
	fbData, err := proto.Marshal(feedback)
	if err != nil {
		return err
	}

	s.mu.RLock()
	handle, ok := s.goals[goalID]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("goal %s not found", goalID)
	}

	actionFb := &actionpb.ActionFeedback{
		Header: &std.Header{
			Stamp: &std.Timestamp{
				Seconds: time.Now().Unix(),
				Nanos:   int32(time.Now().Nanosecond()),
			},
		},
		Status: &actionpb.GoalStatus{
			GoalId: &actionpb.GoalID{Id: goalID},
			Status: handle.Status,
		},
		Feedback: fbData,
	}

	data, err := proto.Marshal(actionFb)
	if err != nil {
		return err
	}

	topic := fmt.Sprintf("action.%s.feedback", s.name)
	return s.nc.Publish(topic, data)
}

// publishResult publishes the final result for a goal.
func (s *Server[GoalT, FeedbackT, ResultT]) publishResult(goalID string, status actionpb.Status, result ResultT) error {
	resultData, err := proto.Marshal(result)
	if err != nil {
		return err
	}

	actionResult := &actionpb.ActionResult{
		Header: &std.Header{
			Stamp: &std.Timestamp{
				Seconds: time.Now().Unix(),
				Nanos:   int32(time.Now().Nanosecond()),
			},
		},
		Status: &actionpb.GoalStatus{
			GoalId: &actionpb.GoalID{Id: goalID},
			Status: status,
		},
		Result: resultData,
	}

	data, err := proto.Marshal(actionResult)
	if err != nil {
		return err
	}

	topic := fmt.Sprintf("action.%s.result", s.name)
	return s.nc.Publish(topic, data)
}

// Name returns the action name.
func (s *Server[GoalT, FeedbackT, ResultT]) Name() string {
	return s.name
}

// ActiveGoals returns the number of active goals.
func (s *Server[GoalT, FeedbackT, ResultT]) ActiveGoals() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.goals)
}

// Close shuts down the server.
func (s *Server[GoalT, FeedbackT, ResultT]) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true

	// Cancel all active goals
	for _, handle := range s.goals {
		handle.cancel()
	}
	s.mu.Unlock()

	var errs []error
	if s.goalSub != nil {
		if err := s.goalSub.Unsubscribe(); err != nil {
			errs = append(errs, err)
		}
	}
	if s.cancelSub != nil {
		if err := s.cancelSub.Unsubscribe(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
