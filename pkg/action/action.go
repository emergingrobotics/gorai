// Package action provides long-running action patterns for Gorai.
//
// Actions are like services but support feedback during execution
// and can be canceled.
package action

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// Goal represents an action goal.
type Goal[T proto.Message] struct {
	ID   string
	Data T
}

// Feedback represents action feedback.
type Feedback[T proto.Message] struct {
	GoalID string
	Data   T
}

// Result represents an action result.
type Result[T proto.Message] struct {
	GoalID string
	Data   T
	Error  error
}

// Server handles action requests.
type Server[GoalT, FeedbackT, ResultT proto.Message] struct {
	// TODO: Implement action server
}

// Client sends action goals.
type Client[GoalT, FeedbackT, ResultT proto.Message] struct {
	// TODO: Implement action client
}

// Handler processes an action goal.
type Handler[GoalT, FeedbackT, ResultT proto.Message] func(
	ctx context.Context,
	goal Goal[GoalT],
	feedback chan<- Feedback[FeedbackT],
) (ResultT, error)

// NewServer creates a new action server.
func NewServer[GoalT, FeedbackT, ResultT proto.Message]() *Server[GoalT, FeedbackT, ResultT] {
	return &Server[GoalT, FeedbackT, ResultT]{}
}

// NewClient creates a new action client.
func NewClient[GoalT, FeedbackT, ResultT proto.Message]() *Client[GoalT, FeedbackT, ResultT] {
	return &Client[GoalT, FeedbackT, ResultT]{}
}
