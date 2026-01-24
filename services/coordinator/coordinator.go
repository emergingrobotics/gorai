// Package coordinator defines the coordinator service interface.
//
// A Coordinator is a special type of Behavior that ONLY works through other
// behaviors. It does not directly use components—instead, it orchestrates
// multiple behaviors to achieve complex, multi-phase goals.
//
// Think of Coordinators as "meta-behaviors" or "behavior managers" that:
//   - Sequence behaviors (do A, then B, then C)
//   - Run behaviors in parallel (do A and B simultaneously)
//   - Select behaviors based on conditions
//   - Manage behavior priorities and conflicts
package coordinator

import (
	"context"
	"time"

	"github.com/gorai/gorai/pkg/resource"
	"github.com/gorai/gorai/services"
	"github.com/gorai/gorai/services/behavior"
)

// Service orchestrates multiple behaviors without directly using components.
// Coordinators implement higher-level logic by delegating to other behaviors.
type Service interface {
	service.Service

	// Start begins coordinator execution.
	Start(ctx context.Context) error

	// Stop halts coordinator and all managed behaviors.
	Stop(ctx context.Context) error

	// IsRunning returns true if the coordinator is active.
	IsRunning(ctx context.Context) (bool, error)

	// GetState returns coordinator state including child behavior states.
	GetState(ctx context.Context) (*State, error)

	// GetManagedBehaviors returns behaviors this coordinator manages.
	GetManagedBehaviors(ctx context.Context) ([]string, error)

	// SetMission sets a high-level mission for the coordinator.
	SetMission(ctx context.Context, mission *Mission) error

	// GetMission returns the current mission.
	GetMission(ctx context.Context) (*Mission, error)

	// GetProgress returns mission progress.
	GetProgress(ctx context.Context) (*Progress, error)

	// GetDerivedSensors returns sensors exposed by this coordinator.
	GetDerivedSensors(ctx context.Context) ([]resource.Name, error)
}

// State represents coordinator state.
type State struct {
	// Status is the current execution status.
	Status behavior.Status

	// CurrentPhase is the name of the current phase.
	CurrentPhase string

	// CurrentPhaseIndex is the index of the current phase.
	CurrentPhaseIndex int

	// BehaviorStates maps behavior names to their states.
	BehaviorStates map[string]*behavior.State

	// MissionProgress is overall mission progress (0.0 - 1.0).
	MissionProgress float64

	// StartTime is when the coordinator started.
	StartTime time.Time

	// ElapsedTime is how long the coordinator has been running.
	ElapsedTime time.Duration

	// Error holds error message if status is Failure.
	Error string
}

// Mission represents a high-level objective composed of phases.
type Mission struct {
	// ID is a unique identifier for this mission.
	ID string

	// Name is a human-readable mission name.
	Name string

	// Description describes the mission objective.
	Description string

	// Phases are the steps in this mission.
	Phases []Phase

	// Priority determines mission importance (higher = more important).
	Priority int

	// Timeout is the maximum time for the entire mission.
	Timeout time.Duration

	// OnFailure specifies what to do if the mission fails.
	OnFailure FailurePolicy

	// FallbackMission is the mission to execute on failure (if policy is FailureFallback).
	FallbackMission string

	// Parameters holds mission-specific parameters.
	Parameters map[string]any
}

// Phase is a step in a mission.
type Phase struct {
	// Name is the phase name.
	Name string

	// Description describes what this phase does.
	Description string

	// Behaviors lists behaviors to run in this phase.
	Behaviors []BehaviorRef

	// Parallel indicates whether to run behaviors in parallel (true) or sequence (false).
	Parallel bool

	// Condition is an optional condition to check before starting.
	// If specified and evaluates to false, the phase is skipped.
	Condition string

	// OnComplete is the next phase on success (empty = next in sequence).
	OnComplete string

	// OnFailure is the phase to go to on failure (empty or "abort" = abort mission).
	OnFailure string

	// Timeout is the maximum time for this phase.
	Timeout time.Duration

	// RetryCount is how many times to retry on failure (0 = no retries).
	RetryCount int
}

// BehaviorRef references a behavior to be used in a phase.
type BehaviorRef struct {
	// Name is the name of the behavior service.
	Name string

	// Goal is the goal to set for this behavior.
	Goal *behavior.Goal

	// Required indicates if phase fails when this behavior fails.
	Required bool

	// Timeout is the timeout for this specific behavior (0 = use phase timeout).
	Timeout time.Duration
}

// FailurePolicy specifies what to do when a mission or phase fails.
type FailurePolicy int

const (
	// FailureAbort stops everything on failure.
	FailureAbort FailurePolicy = iota
	// FailureRetry retries the failed phase.
	FailureRetry
	// FailureSkip skips the failed phase and continues.
	FailureSkip
	// FailureFallback executes a fallback mission.
	FailureFallback
)

// String returns the string representation of a FailurePolicy.
func (fp FailurePolicy) String() string {
	switch fp {
	case FailureAbort:
		return "abort"
	case FailureRetry:
		return "retry"
	case FailureSkip:
		return "skip"
	case FailureFallback:
		return "fallback"
	default:
		return "unknown"
	}
}

// Progress tracks mission completion.
type Progress struct {
	// MissionID is the current mission ID.
	MissionID string

	// MissionName is the current mission name.
	MissionName string

	// CurrentPhase is the index of the current phase.
	CurrentPhase int

	// TotalPhases is the total number of phases.
	TotalPhases int

	// PhaseName is the name of the current phase.
	PhaseName string

	// PhaseProgress is progress within the current phase (0.0 - 1.0).
	PhaseProgress float64

	// OverallProgress is total mission progress (0.0 - 1.0).
	OverallProgress float64

	// CompletedPhases lists completed phase names.
	CompletedPhases []string

	// FailedPhases lists failed phase names.
	FailedPhases []string

	// EstimatedTimeRemaining is estimated time to completion.
	EstimatedTimeRemaining time.Duration

	// StartTime is when the mission started.
	StartTime time.Time

	// ElapsedTime is how long the mission has been running.
	ElapsedTime time.Duration
}

// Properties describes coordinator capabilities.
type Properties struct {
	// Name is a human-readable name.
	Name string

	// Description describes what the coordinator does.
	Description string

	// ManagedBehaviors lists behaviors this coordinator can use.
	ManagedBehaviors []string

	// SupportsParallelPhases indicates if parallel phases are supported.
	SupportsParallelPhases bool

	// SupportsConditions indicates if phase conditions are supported.
	SupportsConditions bool

	// SupportsAI indicates if AI-powered planning is available.
	SupportsAI bool

	// MaxConcurrentBehaviors is the maximum parallel behaviors (0 = unlimited).
	MaxConcurrentBehaviors int
}

// Extended is an optional extended interface for coordinators with
// additional capabilities.
type Extended interface {
	Service

	// GetProperties returns the coordinator properties.
	GetProperties(ctx context.Context) (Properties, error)

	// Pause pauses coordinator execution.
	Pause(ctx context.Context) error

	// Resume resumes a paused coordinator.
	Resume(ctx context.Context) error

	// IsPaused returns true if the coordinator is paused.
	IsPaused(ctx context.Context) (bool, error)

	// SkipPhase skips the current phase and moves to the next.
	SkipPhase(ctx context.Context) error

	// RetryPhase retries the current phase.
	RetryPhase(ctx context.Context) error

	// CancelMission cancels the current mission.
	CancelMission(ctx context.Context) error

	// GetPhaseHistory returns history of phase executions.
	GetPhaseHistory(ctx context.Context, limit int) ([]PhaseResult, error)
}

// PhaseResult represents the result of a phase execution.
type PhaseResult struct {
	// PhaseName is the phase name.
	PhaseName string

	// Status is the result status.
	Status behavior.Status

	// StartTime is when the phase started.
	StartTime time.Time

	// Duration is how long the phase took.
	Duration time.Duration

	// BehaviorResults maps behavior names to their results.
	BehaviorResults map[string]behavior.Status

	// Error holds the error message if the phase failed.
	Error string

	// RetryCount is how many times this phase was retried.
	RetryCount int
}
