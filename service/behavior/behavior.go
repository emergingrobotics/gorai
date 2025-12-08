// Package behavior defines the behavior service interface.
//
// Behaviors are the "brain" of the robot. They implement high-level
// decision-making logic that determines what the robot should do based
// on sensor inputs, goals, and the current state.
//
// Behaviors can use:
//   - Components (sensors, actuators) for direct hardware interaction
//   - Other services (vision, navigation, motion) for higher-level capabilities
//   - Other behaviors for hierarchical decision-making
//
// Behaviors can also expose derived sensors that provide computed or
// inferred data as a byproduct of the behavior's operation.
package behavior

import (
	"context"
	"time"

	"github.com/gorai/gorai/pkg/resource"
	"github.com/gorai/gorai/service"
)

// Service is the interface for behavior services.
// Behaviors implement decision-making logic that coordinates
// components and other services to achieve goals.
type Service interface {
	service.Service

	// Start begins behavior execution.
	Start(ctx context.Context) error

	// Stop halts behavior execution.
	Stop(ctx context.Context) error

	// IsRunning returns true if the behavior is active.
	IsRunning(ctx context.Context) (bool, error)

	// GetState returns current behavior state.
	GetState(ctx context.Context) (*State, error)

	// SetGoal sets a goal for the behavior to achieve.
	SetGoal(ctx context.Context, goal *Goal) error

	// GetGoal returns the current goal.
	GetGoal(ctx context.Context) (*Goal, error)

	// Tick executes one cycle of the behavior (for external schedulers).
	Tick(ctx context.Context) (*TickResult, error)

	// GetDerivedSensors returns sensors exposed by this behavior.
	// These are virtual sensors providing computed/inferred data.
	GetDerivedSensors(ctx context.Context) ([]resource.Name, error)
}

// Status represents the execution status of a behavior.
type Status int

const (
	// StatusIdle indicates the behavior is not running.
	StatusIdle Status = iota
	// StatusRunning indicates the behavior is actively executing.
	StatusRunning
	// StatusSuccess indicates the behavior completed successfully.
	StatusSuccess
	// StatusFailure indicates the behavior failed.
	StatusFailure
	// StatusCanceled indicates the behavior was canceled.
	StatusCanceled
)

// String returns the string representation of a Status.
func (s Status) String() string {
	switch s {
	case StatusIdle:
		return "idle"
	case StatusRunning:
		return "running"
	case StatusSuccess:
		return "success"
	case StatusFailure:
		return "failure"
	case StatusCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}

// State represents the current state of a behavior.
type State struct {
	// Status is the current execution status.
	Status Status

	// CurrentNode is the current node name (for behavior trees).
	CurrentNode string

	// Variables holds blackboard or state variables.
	Variables map[string]any

	// LastTick is the time of the last tick.
	LastTick time.Time

	// TickCount is the number of ticks executed.
	TickCount uint64

	// Error holds the error message if Status is Failure.
	Error string

	// StartTime is when the behavior started running.
	StartTime time.Time

	// Duration is how long the behavior has been running.
	Duration time.Duration
}

// Goal represents an objective for the behavior to achieve.
type Goal struct {
	// ID is a unique identifier for this goal.
	ID string

	// Type identifies the kind of goal (e.g., "navigate", "pick", "patrol").
	Type string

	// Target is the goal-specific target (pose, object, etc.).
	Target any

	// Priority determines goal importance (higher = more important).
	Priority int

	// Timeout is the maximum time to achieve the goal.
	Timeout time.Duration

	// Parameters holds goal-specific parameters.
	Parameters map[string]any
}

// TickResult is the result of one behavior tick.
type TickResult struct {
	// Status is the result status of this tick.
	Status Status

	// Action describes what action was taken.
	Action string

	// Duration is how long the tick took.
	Duration time.Duration

	// Data holds optional result data.
	Data map[string]any
}

// Properties describes behavior capabilities.
type Properties struct {
	// Type indicates the behavior implementation type.
	Type BehaviorType

	// Name is a human-readable name.
	Name string

	// Description describes what the behavior does.
	Description string

	// TickRateHz is the preferred tick rate (0 = event-driven).
	TickRateHz float64

	// SupportsGoals indicates whether the behavior accepts goals.
	SupportsGoals bool

	// SupportsTicking indicates whether the behavior can be externally ticked.
	SupportsTicking bool

	// DerivedSensorTypes lists types of sensors this behavior can expose.
	DerivedSensorTypes []string
}

// BehaviorType identifies the behavior implementation approach.
type BehaviorType int

const (
	// BehaviorTypeBehaviorTree uses a hierarchical tree of behaviors.
	BehaviorTypeBehaviorTree BehaviorType = iota
	// BehaviorTypeStateMachine uses a finite state machine.
	BehaviorTypeStateMachine
	// BehaviorTypeSubsumption uses priority-based layers.
	BehaviorTypeSubsumption
	// BehaviorTypeUtility uses utility-based selection.
	BehaviorTypeUtility
	// BehaviorTypeAIAgent uses AI/ML-powered decision making.
	BehaviorTypeAIAgent
	// BehaviorTypeLLMAgent uses LLM-powered reasoning.
	BehaviorTypeLLMAgent
	// BehaviorTypeCustom is a custom implementation.
	BehaviorTypeCustom
)

// String returns the string representation of a BehaviorType.
func (bt BehaviorType) String() string {
	switch bt {
	case BehaviorTypeBehaviorTree:
		return "behavior_tree"
	case BehaviorTypeStateMachine:
		return "state_machine"
	case BehaviorTypeSubsumption:
		return "subsumption"
	case BehaviorTypeUtility:
		return "utility"
	case BehaviorTypeAIAgent:
		return "ai_agent"
	case BehaviorTypeLLMAgent:
		return "llm_agent"
	case BehaviorTypeCustom:
		return "custom"
	default:
		return "unknown"
	}
}

// Extended is an optional extended interface for behaviors with
// additional capabilities.
type Extended interface {
	Service

	// GetProperties returns the behavior properties.
	GetProperties(ctx context.Context) (Properties, error)

	// Pause pauses behavior execution without stopping.
	Pause(ctx context.Context) error

	// Resume resumes a paused behavior.
	Resume(ctx context.Context) error

	// IsPaused returns true if the behavior is paused.
	IsPaused(ctx context.Context) (bool, error)

	// GetHistory returns recent tick history.
	GetHistory(ctx context.Context, limit int) ([]*TickResult, error)

	// ClearGoal removes the current goal.
	ClearGoal(ctx context.Context) error
}
