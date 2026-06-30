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

// GoalState represents the current state of a goal from client perspective.
type GoalState int

const (
	GoalStatePending GoalState = iota
	GoalStateActive
	GoalStateSucceeded
	GoalStateAborted
	GoalStateCanceled
	GoalStateLost
)

// String returns the string representation of GoalState.
func (s GoalState) String() string {
	switch s {
	case GoalStatePending:
		return "Pending"
	case GoalStateActive:
		return "Active"
	case GoalStateSucceeded:
		return "Succeeded"
	case GoalStateAborted:
		return "Aborted"
	case GoalStateCanceled:
		return "Canceled"
	case GoalStateLost:
		return "Lost"
	default:
		return "Unknown"
	}
}

// ClientGoalHandle tracks a goal sent by the client.
type ClientGoalHandle[GoalT, FeedbackT, ResultT proto.Message] struct {
	ID         string
	Goal       GoalT
	State      GoalState
	feedbackCh chan FeedbackT
	resultCh   chan ResultT
	doneCh     chan struct{}
	result     ResultT
	err        error
	client     *Client[GoalT, FeedbackT, ResultT]
	mu         sync.Mutex
}

// Feedback returns a channel that receives feedback messages.
func (h *ClientGoalHandle[GoalT, FeedbackT, ResultT]) Feedback() <-chan FeedbackT {
	return h.feedbackCh
}

// Wait blocks until the goal completes and returns the result.
func (h *ClientGoalHandle[GoalT, FeedbackT, ResultT]) Wait(ctx context.Context) (ResultT, error) {
	select {
	case <-h.doneCh:
		return h.result, h.err
	case <-ctx.Done():
		var zero ResultT
		return zero, ctx.Err()
	}
}

// Cancel requests cancellation of this goal.
func (h *ClientGoalHandle[GoalT, FeedbackT, ResultT]) Cancel(ctx context.Context) error {
	return h.client.CancelGoal(ctx, h.ID)
}

// Done returns a channel that closes when the goal completes.
func (h *ClientGoalHandle[GoalT, FeedbackT, ResultT]) Done() <-chan struct{} {
	return h.doneCh
}

// clientOptions holds client configuration.
type clientOptions struct {
	timeout time.Duration
	logger  *slog.Logger
}

// ClientOption configures a client.
type ClientOption func(*clientOptions)

// WithClientTimeout sets the request timeout.
func WithClientTimeout(d time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.timeout = d
	}
}

// WithClientLogger sets the logger for the client.
func WithClientLogger(logger *slog.Logger) ClientOption {
	return func(o *clientOptions) {
		o.logger = logger
	}
}

// Client sends action goals.
type Client[GoalT, FeedbackT, ResultT proto.Message] struct {
	nc          *nats.Conn
	name        string
	timeout     time.Duration
	logger      *slog.Logger
	goals       map[string]*ClientGoalHandle[GoalT, FeedbackT, ResultT]
	feedbackSub *nats.Subscription
	resultSub   *nats.Subscription
	statusSub   *nats.Subscription
	mu          sync.RWMutex
	closed      bool
}

// NewClient creates a new action client.
func NewClient[GoalT, FeedbackT, ResultT proto.Message](
	n NATSGetter,
	name string,
	opts ...ClientOption,
) (*Client[GoalT, FeedbackT, ResultT], error) {
	nc := n.NATS()
	if nc == nil {
		return nil, fmt.Errorf("no NATS connection")
	}

	options := clientOptions{
		timeout: 5 * time.Second,
		logger:  slog.Default(),
	}
	for _, opt := range opts {
		opt(&options)
	}

	c := &Client[GoalT, FeedbackT, ResultT]{
		nc:      nc,
		name:    name,
		timeout: options.timeout,
		logger:  options.logger,
		goals:   make(map[string]*ClientGoalHandle[GoalT, FeedbackT, ResultT]),
	}

	// Subscribe to feedback
	feedbackTopic := fmt.Sprintf("action.%s.feedback", name)
	feedbackSub, err := nc.Subscribe(feedbackTopic, c.handleFeedback)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to feedback: %w", err)
	}
	c.feedbackSub = feedbackSub

	// Subscribe to results
	resultTopic := fmt.Sprintf("action.%s.result", name)
	resultSub, err := nc.Subscribe(resultTopic, c.handleResult)
	if err != nil {
		feedbackSub.Unsubscribe()
		return nil, fmt.Errorf("failed to subscribe to result: %w", err)
	}
	c.resultSub = resultSub

	// Subscribe to status
	statusTopic := fmt.Sprintf("action.%s.status", name)
	statusSub, err := nc.Subscribe(statusTopic, c.handleStatus)
	if err != nil {
		feedbackSub.Unsubscribe()
		resultSub.Unsubscribe()
		return nil, fmt.Errorf("failed to subscribe to status: %w", err)
	}
	c.statusSub = statusSub

	return c, nil
}

// SendGoal sends a goal to the action server.
func (c *Client[GoalT, FeedbackT, ResultT]) SendGoal(ctx context.Context, goal GoalT) (*ClientGoalHandle[GoalT, FeedbackT, ResultT], error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("client is closed")
	}
	c.mu.Unlock()

	data, err := proto.Marshal(goal)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal goal: %w", err)
	}

	// Send goal with request-reply to get goal ID
	topic := fmt.Sprintf("action.%s.goal", c.name)
	msg, err := c.nc.Request(topic, data, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to send goal: %w", err)
	}

	// Parse goal ID from response
	var goalID actionpb.GoalID
	if err := proto.Unmarshal(msg.Data, &goalID); err != nil {
		return nil, fmt.Errorf("failed to unmarshal goal ID: %w", err)
	}

	handle := &ClientGoalHandle[GoalT, FeedbackT, ResultT]{
		ID:         goalID.Id,
		Goal:       goal,
		State:      GoalStatePending,
		feedbackCh: make(chan FeedbackT, 10),
		resultCh:   make(chan ResultT, 1),
		doneCh:     make(chan struct{}),
		client:     c,
	}

	c.mu.Lock()
	c.goals[goalID.Id] = handle
	c.mu.Unlock()

	return handle, nil
}

// SendGoalAndWait sends a goal and waits for the result.
func (c *Client[GoalT, FeedbackT, ResultT]) SendGoalAndWait(ctx context.Context, goal GoalT) (ResultT, error) {
	handle, err := c.SendGoal(ctx, goal)
	if err != nil {
		var zero ResultT
		return zero, err
	}
	return handle.Wait(ctx)
}

// CancelGoal requests cancellation of a specific goal.
func (c *Client[GoalT, FeedbackT, ResultT]) CancelGoal(ctx context.Context, goalID string) error {
	cancelReq := &actionpb.CancelGoal{
		GoalId: &actionpb.GoalID{Id: goalID},
		Stamp: &std.Timestamp{
			Seconds: time.Now().Unix(),
			Nanos:   int32(time.Now().Nanosecond()),
		},
	}

	data, err := proto.Marshal(cancelReq)
	if err != nil {
		return fmt.Errorf("failed to marshal cancel request: %w", err)
	}

	topic := fmt.Sprintf("action.%s.cancel", c.name)
	msg, err := c.nc.Request(topic, data, c.timeout)
	if err != nil {
		return fmt.Errorf("failed to send cancel request: %w", err)
	}

	var response actionpb.CancelGoalResponse
	if err := proto.Unmarshal(msg.Data, &response); err != nil {
		return fmt.Errorf("failed to unmarshal cancel response: %w", err)
	}

	if response.ReturnCode == actionpb.CancelGoalResponse_CODE_UNKNOWN_GOAL {
		return fmt.Errorf("unknown goal: %s", goalID)
	}

	return nil
}

// CancelAllGoals requests cancellation of all active goals.
func (c *Client[GoalT, FeedbackT, ResultT]) CancelAllGoals(ctx context.Context) error {
	cancelReq := &actionpb.CancelGoal{
		Stamp: &std.Timestamp{
			Seconds: time.Now().Unix(),
			Nanos:   int32(time.Now().Nanosecond()),
		},
	}

	data, err := proto.Marshal(cancelReq)
	if err != nil {
		return fmt.Errorf("failed to marshal cancel request: %w", err)
	}

	topic := fmt.Sprintf("action.%s.cancel", c.name)
	_, err = c.nc.Request(topic, data, c.timeout)
	if err != nil {
		return fmt.Errorf("failed to send cancel request: %w", err)
	}

	return nil
}

// handleFeedback processes incoming feedback.
func (c *Client[GoalT, FeedbackT, ResultT]) handleFeedback(msg *nats.Msg) {
	var actionFb actionpb.ActionFeedback
	if err := proto.Unmarshal(msg.Data, &actionFb); err != nil {
		c.logger.Error("failed to unmarshal feedback", "error", err)
		return
	}

	if actionFb.Status == nil || actionFb.Status.GoalId == nil {
		return
	}

	goalID := actionFb.Status.GoalId.Id

	c.mu.RLock()
	handle, ok := c.goals[goalID]
	c.mu.RUnlock()

	if !ok {
		return
	}

	// Unmarshal the feedback data
	var fb FeedbackT
	fb = fb.ProtoReflect().New().Interface().(FeedbackT)
	if err := proto.Unmarshal(actionFb.Feedback, fb); err != nil {
		c.logger.Error("failed to unmarshal feedback data", "error", err)
		return
	}

	// Send feedback non-blocking
	select {
	case handle.feedbackCh <- fb:
	default:
		c.logger.Warn("feedback channel full, dropping feedback", "goal_id", goalID)
	}
}

// handleResult processes incoming results.
func (c *Client[GoalT, FeedbackT, ResultT]) handleResult(msg *nats.Msg) {
	var actionResult actionpb.ActionResult
	if err := proto.Unmarshal(msg.Data, &actionResult); err != nil {
		c.logger.Error("failed to unmarshal result", "error", err)
		return
	}

	if actionResult.Status == nil || actionResult.Status.GoalId == nil {
		return
	}

	goalID := actionResult.Status.GoalId.Id

	c.mu.Lock()
	handle, ok := c.goals[goalID]
	if ok {
		delete(c.goals, goalID)
	}
	c.mu.Unlock()

	if !ok {
		return
	}

	// Unmarshal the result data
	var result ResultT
	result = result.ProtoReflect().New().Interface().(ResultT)
	if err := proto.Unmarshal(actionResult.Result, result); err != nil {
		c.logger.Error("failed to unmarshal result data", "error", err)
		handle.err = err
	} else {
		handle.result = result
	}

	// Update state based on status
	handle.mu.Lock()
	switch actionResult.Status.Status {
	case actionpb.Status_STATUS_SUCCEEDED:
		handle.State = GoalStateSucceeded
	case actionpb.Status_STATUS_ABORTED:
		handle.State = GoalStateAborted
		handle.err = fmt.Errorf("goal aborted")
	case actionpb.Status_STATUS_PREEMPTED:
		handle.State = GoalStateCanceled
		handle.err = fmt.Errorf("goal canceled")
	default:
		handle.State = GoalStateLost
		handle.err = fmt.Errorf("goal lost: %s", actionResult.Status.Status)
	}
	handle.mu.Unlock()

	close(handle.feedbackCh)
	close(handle.doneCh)
}

// handleStatus processes incoming status updates.
func (c *Client[GoalT, FeedbackT, ResultT]) handleStatus(msg *nats.Msg) {
	var statusArray actionpb.GoalStatusArray
	if err := proto.Unmarshal(msg.Data, &statusArray); err != nil {
		c.logger.Error("failed to unmarshal status", "error", err)
		return
	}

	for _, status := range statusArray.StatusList {
		if status.GoalId == nil {
			continue
		}

		c.mu.RLock()
		handle, ok := c.goals[status.GoalId.Id]
		c.mu.RUnlock()

		if !ok {
			continue
		}

		handle.mu.Lock()
		switch status.Status {
		case actionpb.Status_STATUS_PENDING:
			handle.State = GoalStatePending
		case actionpb.Status_STATUS_ACTIVE:
			handle.State = GoalStateActive
		}
		handle.mu.Unlock()
	}
}

// Name returns the action name.
func (c *Client[GoalT, FeedbackT, ResultT]) Name() string {
	return c.name
}

// Close shuts down the client.
func (c *Client[GoalT, FeedbackT, ResultT]) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true

	// Close all pending goals
	for _, handle := range c.goals {
		handle.mu.Lock()
		handle.State = GoalStateLost
		handle.err = fmt.Errorf("client closed")
		handle.mu.Unlock()
		close(handle.feedbackCh)
		close(handle.doneCh)
	}
	c.goals = make(map[string]*ClientGoalHandle[GoalT, FeedbackT, ResultT])
	c.mu.Unlock()

	var errs []error
	if c.feedbackSub != nil {
		if err := c.feedbackSub.Unsubscribe(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.resultSub != nil {
		if err := c.resultSub.Unsubscribe(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.statusSub != nil {
		if err := c.statusSub.Unsubscribe(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
