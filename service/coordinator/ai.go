package coordinator

import (
	"context"
	"time"

	"github.com/gorai/gorai/service/behavior"
)

// AICoordinator extends Service with AI/ML capabilities for
// high-level mission planning and orchestration.
type AICoordinator interface {
	Service

	// GenerateMission creates a mission from a natural language description.
	GenerateMission(ctx context.Context, description string) (*Mission, error)

	// AdaptMission modifies the current mission based on new information.
	AdaptMission(ctx context.Context, situation string) error

	// ExplainPlan returns a human-readable explanation of the mission plan.
	ExplainPlan(ctx context.Context) (string, error)

	// GetPlanningMetadata returns metadata about the AI planning system.
	GetPlanningMetadata(ctx context.Context) (*PlanningMetadata, error)
}

// LLMCoordinator extends AICoordinator with LLM-specific capabilities.
type LLMCoordinator interface {
	AICoordinator

	// GetLLMProvider returns the LLM provider.
	GetLLMProvider(ctx context.Context) (string, error)

	// GetLLMModel returns the specific LLM model.
	GetLLMModel(ctx context.Context) (string, error)

	// GetPlanningHistory returns the history of planning decisions.
	GetPlanningHistory(ctx context.Context, limit int) ([]PlanningDecision, error)

	// SetPlanningConstraints sets constraints for planning.
	SetPlanningConstraints(ctx context.Context, constraints []string) error

	// GetPlanningConstraints returns current planning constraints.
	GetPlanningConstraints(ctx context.Context) ([]string, error)
}

// PlanningMetadata describes the AI planning system.
type PlanningMetadata struct {
	// Type is the planning approach (e.g., "llm", "htn", "goap").
	Type string

	// Provider is the AI provider (for LLM-based planning).
	Provider string

	// Model is the specific model being used.
	Model string

	// AvailableBehaviors lists behaviors the planner knows about.
	AvailableBehaviors []BehaviorDescription

	// PlanningStrategy is the current planning strategy.
	PlanningStrategy string

	// ReplanningTriggers lists conditions that trigger replanning.
	ReplanningTriggers []string

	// LastPlanTime is when the last plan was generated.
	LastPlanTime time.Time
}

// BehaviorDescription describes a behavior for planning purposes.
type BehaviorDescription struct {
	// Name is the behavior name.
	Name string

	// Description describes what the behavior does.
	Description string

	// Capabilities lists what the behavior can do.
	Capabilities []string

	// Preconditions lists conditions required for the behavior.
	Preconditions []string

	// Effects lists the effects of running the behavior.
	Effects []string

	// AverageRuntime is the typical runtime.
	AverageRuntime time.Duration

	// SuccessRate is the historical success rate (0-1).
	SuccessRate float64
}

// PlanningDecision represents a decision made during planning.
type PlanningDecision struct {
	// Timestamp is when the decision was made.
	Timestamp time.Time

	// DecisionType is the type of decision (e.g., "plan", "replan", "adapt").
	DecisionType string

	// Input is what triggered the decision.
	Input string

	// Reasoning is the explanation for the decision.
	Reasoning string

	// OutputMission is the resulting mission (if applicable).
	OutputMission *Mission

	// Changes describes changes made (for adaptations).
	Changes []string

	// TokensUsed is the number of LLM tokens used (if applicable).
	TokensUsed int

	// Duration is how long the decision took.
	Duration time.Duration
}

// AICoordinatorConfig configures an AI-powered coordinator.
type AICoordinatorConfig struct {
	// LLMProvider is the LLM provider (anthropic, openai, local).
	LLMProvider string

	// LLMModel is the specific model to use.
	LLMModel string

	// SystemPrompt is the system prompt for planning.
	SystemPrompt string

	// AvailableBehaviors describes behaviors the coordinator can use.
	AvailableBehaviors []BehaviorDescription

	// PlanningStrategy is the planning approach.
	PlanningStrategy string

	// ReplanningTriggers lists conditions that trigger replanning.
	ReplanningTriggers []string

	// SafetyConstraints lists safety rules to follow.
	SafetyConstraints []string

	// MaxPlanLength is the maximum number of phases in a plan.
	MaxPlanLength int

	// PlanningTimeout is the timeout for planning operations.
	PlanningTimeout time.Duration
}

// Common planning strategies.
const (
	// PlanningStrategyMinimizeTime optimizes for fastest completion.
	PlanningStrategyMinimizeTime = "minimize_time"

	// PlanningStrategyMinimizeEnergy optimizes for lowest energy use.
	PlanningStrategyMinimizeEnergy = "minimize_energy"

	// PlanningStrategyMaximizeSafety prioritizes safe operations.
	PlanningStrategyMaximizeSafety = "maximize_safety"

	// PlanningStrategyBalanced balances multiple factors.
	PlanningStrategyBalanced = "balanced"
)

// Common replanning triggers.
const (
	// ReplanTriggerBehaviorFailure replans when a behavior fails.
	ReplanTriggerBehaviorFailure = "behavior_failure"

	// ReplanTriggerBatteryLow replans when battery is low.
	ReplanTriggerBatteryLow = "battery_low"

	// ReplanTriggerNewPriorityTask replans when a higher priority task arrives.
	ReplanTriggerNewPriorityTask = "new_priority_task"

	// ReplanTriggerObstacleDetected replans when an obstacle is detected.
	ReplanTriggerObstacleDetected = "obstacle_detected"

	// ReplanTriggerTimeoutRisk replans when timeout is at risk.
	ReplanTriggerTimeoutRisk = "timeout_risk"
)
