package behavior

import (
	"context"
	"time"
)

// AIBehavior extends Service with AI/ML-specific capabilities.
// Use this interface for behaviors powered by ML models.
type AIBehavior interface {
	Service

	// GetModel returns the ML model identifier used by this behavior.
	GetModel(ctx context.Context) (string, error)

	// GetConfidence returns confidence in the current decision (0-1).
	GetConfidence(ctx context.Context) (float64, error)

	// GetExplanation returns a human-readable explanation of the current action.
	GetExplanation(ctx context.Context) (string, error)

	// Learn updates the model based on feedback (for online learning).
	Learn(ctx context.Context, feedback *Feedback) error

	// GetModelMetadata returns metadata about the AI model.
	GetModelMetadata(ctx context.Context) (*ModelMetadata, error)
}

// Feedback provides feedback to an AI behavior for learning.
type Feedback struct {
	// GoalID is the goal this feedback is for.
	GoalID string

	// Outcome indicates how well the goal was achieved.
	Outcome Outcome

	// Reward is the reward signal (for reinforcement learning).
	Reward float64

	// Timestamp is when the feedback was given.
	Timestamp time.Time

	// Metadata holds additional feedback data.
	Metadata map[string]any
}

// Outcome represents the outcome of a goal attempt.
type Outcome int

const (
	// OutcomeSuccess indicates the goal was achieved.
	OutcomeSuccess Outcome = iota
	// OutcomePartial indicates partial success.
	OutcomePartial
	// OutcomeFailure indicates complete failure.
	OutcomeFailure
	// OutcomeTimeout indicates the goal timed out.
	OutcomeTimeout
	// OutcomeCanceled indicates the goal was canceled.
	OutcomeCanceled
)

// String returns the string representation of an Outcome.
func (o Outcome) String() string {
	switch o {
	case OutcomeSuccess:
		return "success"
	case OutcomePartial:
		return "partial"
	case OutcomeFailure:
		return "failure"
	case OutcomeTimeout:
		return "timeout"
	case OutcomeCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}

// ModelMetadata describes an AI model.
type ModelMetadata struct {
	// Name is the model name.
	Name string

	// Version is the model version.
	Version string

	// Framework is the ML framework (tensorflow, pytorch, onnx, etc.).
	Framework string

	// Accelerator is the hardware accelerator being used.
	Accelerator string

	// InputShape is the expected input shape.
	InputShape []int

	// OutputShape is the expected output shape.
	OutputShape []int

	// LearningEnabled indicates if online learning is enabled.
	LearningEnabled bool

	// LastTrainedAt is when the model was last updated.
	LastTrainedAt time.Time
}

// LLMBehavior extends Service with LLM-specific capabilities.
// Use this interface for behaviors powered by Large Language Models.
type LLMBehavior interface {
	Service

	// GetLLMProvider returns the LLM provider (anthropic, openai, etc.).
	GetLLMProvider(ctx context.Context) (string, error)

	// GetLLMModel returns the specific model being used.
	GetLLMModel(ctx context.Context) (string, error)

	// SendPrompt sends a prompt and gets a response (for debugging/interaction).
	SendPrompt(ctx context.Context, prompt string) (string, error)

	// GetReasoningTrace returns the LLM's reasoning for the current action.
	GetReasoningTrace(ctx context.Context) ([]ReasoningStep, error)

	// SetSystemPrompt updates the system prompt/persona.
	SetSystemPrompt(ctx context.Context, prompt string) error

	// GetSystemPrompt returns the current system prompt.
	GetSystemPrompt(ctx context.Context) (string, error)

	// GetConversationHistory returns recent conversation history.
	GetConversationHistory(ctx context.Context, limit int) ([]ConversationTurn, error)

	// ClearConversationHistory clears the conversation history.
	ClearConversationHistory(ctx context.Context) error
}

// ReasoningStep represents one step in an LLM's reasoning process.
type ReasoningStep struct {
	// Step is the step number.
	Step int

	// Thought is what the LLM is thinking.
	Thought string

	// Action is the action the LLM decided to take.
	Action string

	// ActionInput is the input to the action.
	ActionInput map[string]any

	// Observation is what was observed after the action.
	Observation string

	// Timestamp is when this step occurred.
	Timestamp time.Time

	// Duration is how long this step took.
	Duration time.Duration

	// TokensUsed is the number of tokens used.
	TokensUsed int
}

// ConversationTurn represents a turn in the conversation.
type ConversationTurn struct {
	// Role is the speaker (user, assistant, system).
	Role string

	// Content is the message content.
	Content string

	// Timestamp is when this turn occurred.
	Timestamp time.Time

	// TokenCount is the number of tokens in this turn.
	TokenCount int
}

// LLMConfig configures an LLM-powered behavior.
type LLMConfig struct {
	// Provider is the LLM provider (anthropic, openai, local).
	Provider string

	// Model is the specific model to use.
	Model string

	// SystemPrompt is the system prompt/persona.
	SystemPrompt string

	// Temperature controls randomness (0-1).
	Temperature float64

	// MaxTokens is the maximum response tokens.
	MaxTokens int

	// AvailableTools lists tools the LLM can use.
	AvailableTools []Tool

	// MaxReasoningSteps limits reasoning iterations.
	MaxReasoningSteps int

	// TimeoutPerStep is the timeout for each reasoning step.
	TimeoutPerStep time.Duration

	// SafetyConstraints lists safety rules to follow.
	SafetyConstraints []string
}

// Tool represents a tool that an LLM can use.
type Tool struct {
	// Name is the tool name.
	Name string

	// Description describes what the tool does.
	Description string

	// Parameters describes the tool's parameters.
	Parameters map[string]ToolParameter

	// Required lists required parameters.
	Required []string
}

// ToolParameter describes a tool parameter.
type ToolParameter struct {
	// Type is the parameter type (string, number, boolean, object, array).
	Type string

	// Description describes the parameter.
	Description string

	// Enum lists allowed values (if applicable).
	Enum []string

	// Default is the default value (if applicable).
	Default any
}

// AICoordinator extends the coordinator concept with AI capabilities.
// Note: The full Coordinator interface is in the coordinator package.
// This interface is for behaviors that need AI coordinator capabilities.
type AICoordinator interface {
	// GenerateMission creates a mission from a natural language description.
	GenerateMission(ctx context.Context, description string) (any, error)

	// AdaptMission modifies the current mission based on new information.
	AdaptMission(ctx context.Context, situation string) error

	// ExplainPlan returns a human-readable explanation of the mission plan.
	ExplainPlan(ctx context.Context) (string, error)
}
