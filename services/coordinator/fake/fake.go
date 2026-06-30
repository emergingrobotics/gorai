// Package fake provides a fake coordinator implementation for testing.
package fake

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/emergingrobotics/gorai/services/behavior"
	"github.com/emergingrobotics/gorai/services/coordinator"
)

func init() {
	registry.RegisterService("coordinator", "fake", New)
}

// Coordinator is a fake coordinator service for testing.
type Coordinator struct {
	name             resource.Name
	mu               sync.RWMutex
	running          bool
	paused           bool
	mission          *coordinator.Mission
	state            coordinator.State
	managedBehaviors []string
	derivedSensors   []resource.Name
	phaseHistory     []coordinator.PhaseResult
	properties       coordinator.Properties

	// AI-specific
	llmProvider          string
	llmModel             string
	planningMetadata     coordinator.PlanningMetadata
	planningHistory      []coordinator.PlanningDecision
	planningConstraints  []string
}

// New creates a new fake coordinator.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewServiceName("gorai", "coordinator", nameStr)

	c := &Coordinator{
		name:             name,
		running:          false,
		paused:           false,
		state:            coordinator.State{Status: behavior.StatusIdle, BehaviorStates: make(map[string]*behavior.State)},
		managedBehaviors: []string{},
		derivedSensors:   []resource.Name{},
		phaseHistory:     []coordinator.PhaseResult{},
		properties: coordinator.Properties{
			Name:                   nameStr,
			Description:            "Fake coordinator for testing",
			ManagedBehaviors:       []string{},
			SupportsParallelPhases: true,
			SupportsConditions:     true,
			SupportsAI:             true,
			MaxConcurrentBehaviors: 5,
		},
		llmProvider: "fake",
		llmModel:    "fake-planner-1.0",
		planningMetadata: coordinator.PlanningMetadata{
			Type:               "llm",
			Provider:           "fake",
			Model:              "fake-planner-1.0",
			PlanningStrategy:   coordinator.PlanningStrategyBalanced,
			ReplanningTriggers: []string{coordinator.ReplanTriggerBehaviorFailure},
		},
		planningConstraints: []string{},
	}

	return c, nil
}

// NewWithName creates a fake coordinator with a specific resource name.
func NewWithName(name resource.Name) *Coordinator {
	return &Coordinator{
		name:             name,
		running:          false,
		paused:           false,
		state:            coordinator.State{Status: behavior.StatusIdle, BehaviorStates: make(map[string]*behavior.State)},
		managedBehaviors: []string{},
		derivedSensors:   []resource.Name{},
		phaseHistory:     []coordinator.PhaseResult{},
		properties: coordinator.Properties{
			Name:                   name.Name,
			Description:            "Fake coordinator for testing",
			ManagedBehaviors:       []string{},
			SupportsParallelPhases: true,
			SupportsConditions:     true,
			SupportsAI:             true,
			MaxConcurrentBehaviors: 5,
		},
		llmProvider: "fake",
		llmModel:    "fake-planner-1.0",
		planningMetadata: coordinator.PlanningMetadata{
			Type:               "llm",
			Provider:           "fake",
			Model:              "fake-planner-1.0",
			PlanningStrategy:   coordinator.PlanningStrategyBalanced,
			ReplanningTriggers: []string{coordinator.ReplanTriggerBehaviorFailure},
		},
		planningConstraints: []string{},
	}
}

// Name returns the coordinator's resource name.
func (c *Coordinator) Name() resource.Name {
	return c.name
}

// Reconfigure updates the coordinator configuration.
func (c *Coordinator) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand executes arbitrary commands.
func (c *Coordinator) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "get_state":
			c.mu.RLock()
			defer c.mu.RUnlock()
			return map[string]any{
				"status":         c.state.Status.String(),
				"running":        c.running,
				"paused":         c.paused,
				"current_phase":  c.state.CurrentPhase,
				"mission_progress": c.state.MissionProgress,
			}, nil
		case "add_behavior":
			if name, ok := cmd["name"].(string); ok {
				c.mu.Lock()
				c.managedBehaviors = append(c.managedBehaviors, name)
				c.mu.Unlock()
				return map[string]any{"status": "ok"}, nil
			}
			return nil, fmt.Errorf("missing name")
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (c *Coordinator) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = false
	c.state.Status = behavior.StatusIdle
	return nil
}

// Start begins coordinator execution.
func (c *Coordinator) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		return fmt.Errorf("coordinator already running")
	}
	c.running = true
	c.paused = false
	c.state.Status = behavior.StatusRunning
	c.state.StartTime = time.Now()
	return nil
}

// Stop halts coordinator and all managed behaviors.
func (c *Coordinator) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = false
	c.paused = false
	c.state.Status = behavior.StatusIdle
	return nil
}

// IsRunning returns true if the coordinator is active.
func (c *Coordinator) IsRunning(ctx context.Context) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running, nil
}

// GetState returns coordinator state.
func (c *Coordinator) GetState(ctx context.Context) (*coordinator.State, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	state := c.state
	state.ElapsedTime = time.Since(c.state.StartTime)
	return &state, nil
}

// GetManagedBehaviors returns behaviors this coordinator manages.
func (c *Coordinator) GetManagedBehaviors(ctx context.Context) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]string, len(c.managedBehaviors))
	copy(result, c.managedBehaviors)
	return result, nil
}

// SetMission sets a mission for the coordinator.
func (c *Coordinator) SetMission(ctx context.Context, mission *coordinator.Mission) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mission = mission
	if mission != nil && len(mission.Phases) > 0 {
		c.state.CurrentPhase = mission.Phases[0].Name
		c.state.CurrentPhaseIndex = 0
	}
	return nil
}

// GetMission returns the current mission.
func (c *Coordinator) GetMission(ctx context.Context) (*coordinator.Mission, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.mission, nil
}

// GetProgress returns mission progress.
func (c *Coordinator) GetProgress(ctx context.Context) (*coordinator.Progress, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	progress := &coordinator.Progress{
		StartTime:   c.state.StartTime,
		ElapsedTime: time.Since(c.state.StartTime),
	}

	if c.mission != nil {
		progress.MissionID = c.mission.ID
		progress.MissionName = c.mission.Name
		progress.TotalPhases = len(c.mission.Phases)
		progress.CurrentPhase = c.state.CurrentPhaseIndex
		progress.PhaseName = c.state.CurrentPhase

		if progress.TotalPhases > 0 {
			progress.OverallProgress = float64(c.state.CurrentPhaseIndex) / float64(progress.TotalPhases)
		}
	}

	return progress, nil
}

// GetDerivedSensors returns sensors exposed by this coordinator.
func (c *Coordinator) GetDerivedSensors(ctx context.Context) ([]resource.Name, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]resource.Name, len(c.derivedSensors))
	copy(result, c.derivedSensors)
	return result, nil
}

// Extended interface methods

// GetProperties returns the coordinator properties.
func (c *Coordinator) GetProperties(ctx context.Context) (coordinator.Properties, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.properties, nil
}

// Pause pauses coordinator execution.
func (c *Coordinator) Pause(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.running {
		return fmt.Errorf("coordinator not running")
	}
	c.paused = true
	return nil
}

// Resume resumes a paused coordinator.
func (c *Coordinator) Resume(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.paused {
		return fmt.Errorf("coordinator not paused")
	}
	c.paused = false
	return nil
}

// IsPaused returns true if the coordinator is paused.
func (c *Coordinator) IsPaused(ctx context.Context) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.paused, nil
}

// SkipPhase skips the current phase.
func (c *Coordinator) SkipPhase(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.mission == nil || c.state.CurrentPhaseIndex >= len(c.mission.Phases)-1 {
		return fmt.Errorf("no phase to skip to")
	}

	c.state.CurrentPhaseIndex++
	c.state.CurrentPhase = c.mission.Phases[c.state.CurrentPhaseIndex].Name
	return nil
}

// RetryPhase retries the current phase.
func (c *Coordinator) RetryPhase(ctx context.Context) error {
	// Fake implementation - just acknowledge
	return nil
}

// CancelMission cancels the current mission.
func (c *Coordinator) CancelMission(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mission = nil
	c.state.Status = behavior.StatusCanceled
	c.state.CurrentPhase = ""
	c.state.CurrentPhaseIndex = 0
	return nil
}

// GetPhaseHistory returns history of phase executions.
func (c *Coordinator) GetPhaseHistory(ctx context.Context, limit int) ([]coordinator.PhaseResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if limit <= 0 || limit > len(c.phaseHistory) {
		limit = len(c.phaseHistory)
	}

	start := len(c.phaseHistory) - limit
	result := make([]coordinator.PhaseResult, limit)
	copy(result, c.phaseHistory[start:])
	return result, nil
}

// AICoordinator interface methods

// GenerateMission creates a mission from a natural language description.
func (c *Coordinator) GenerateMission(ctx context.Context, description string) (*coordinator.Mission, error) {
	// Generate a fake mission based on the description
	mission := &coordinator.Mission{
		ID:          "generated_" + fmt.Sprint(time.Now().UnixNano()),
		Name:        "Generated Mission",
		Description: description,
		Phases: []coordinator.Phase{
			{
				Name:        "phase_1",
				Description: "First phase",
				Behaviors:   []coordinator.BehaviorRef{},
			},
			{
				Name:        "phase_2",
				Description: "Second phase",
				Behaviors:   []coordinator.BehaviorRef{},
			},
		},
		Priority:  1,
		OnFailure: coordinator.FailureAbort,
	}

	c.mu.Lock()
	c.planningHistory = append(c.planningHistory, coordinator.PlanningDecision{
		Timestamp:     time.Now(),
		DecisionType:  "plan",
		Input:         description,
		Reasoning:     "Generated fake mission from description",
		OutputMission: mission,
	})
	c.mu.Unlock()

	return mission, nil
}

// AdaptMission modifies the current mission.
func (c *Coordinator) AdaptMission(ctx context.Context, situation string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.mission == nil {
		return fmt.Errorf("no mission to adapt")
	}

	c.planningHistory = append(c.planningHistory, coordinator.PlanningDecision{
		Timestamp:    time.Now(),
		DecisionType: "adapt",
		Input:        situation,
		Reasoning:    "Adapted mission based on situation",
		Changes:      []string{"adaptation_applied"},
	})

	return nil
}

// ExplainPlan returns a human-readable explanation.
func (c *Coordinator) ExplainPlan(ctx context.Context) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.mission == nil {
		return "No mission currently set.", nil
	}

	explanation := fmt.Sprintf("Mission: %s\n", c.mission.Name)
	explanation += fmt.Sprintf("Description: %s\n", c.mission.Description)
	explanation += fmt.Sprintf("Phases: %d\n", len(c.mission.Phases))

	for i, phase := range c.mission.Phases {
		explanation += fmt.Sprintf("  %d. %s: %s\n", i+1, phase.Name, phase.Description)
	}

	return explanation, nil
}

// GetPlanningMetadata returns metadata about the AI planning system.
func (c *Coordinator) GetPlanningMetadata(ctx context.Context) (*coordinator.PlanningMetadata, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	metadata := c.planningMetadata
	return &metadata, nil
}

// LLMCoordinator interface methods

// GetLLMProvider returns the LLM provider.
func (c *Coordinator) GetLLMProvider(ctx context.Context) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.llmProvider, nil
}

// GetLLMModel returns the specific LLM model.
func (c *Coordinator) GetLLMModel(ctx context.Context) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.llmModel, nil
}

// GetPlanningHistory returns the history of planning decisions.
func (c *Coordinator) GetPlanningHistory(ctx context.Context, limit int) ([]coordinator.PlanningDecision, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if limit <= 0 || limit > len(c.planningHistory) {
		limit = len(c.planningHistory)
	}

	start := len(c.planningHistory) - limit
	result := make([]coordinator.PlanningDecision, limit)
	copy(result, c.planningHistory[start:])
	return result, nil
}

// SetPlanningConstraints sets constraints for planning.
func (c *Coordinator) SetPlanningConstraints(ctx context.Context, constraints []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.planningConstraints = make([]string, len(constraints))
	copy(c.planningConstraints, constraints)
	return nil
}

// GetPlanningConstraints returns current planning constraints.
func (c *Coordinator) GetPlanningConstraints(ctx context.Context) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]string, len(c.planningConstraints))
	copy(result, c.planningConstraints)
	return result, nil
}

// Test helper methods

// SetRunning sets the running state (for testing).
func (c *Coordinator) SetRunning(running bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = running
	if running {
		c.state.Status = behavior.StatusRunning
	} else {
		c.state.Status = behavior.StatusIdle
	}
}

// SetStatus sets the coordinator status (for testing).
func (c *Coordinator) SetStatus(status behavior.Status) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Status = status
}

// AddManagedBehavior adds a managed behavior (for testing).
func (c *Coordinator) AddManagedBehavior(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.managedBehaviors = append(c.managedBehaviors, name)
}

// AddDerivedSensor adds a derived sensor (for testing).
func (c *Coordinator) AddDerivedSensor(name resource.Name) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.derivedSensors = append(c.derivedSensors, name)
}

// AddPhaseResult adds a phase result to history (for testing).
func (c *Coordinator) AddPhaseResult(result coordinator.PhaseResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.phaseHistory = append(c.phaseHistory, result)
}

// SetCurrentPhase sets the current phase (for testing).
func (c *Coordinator) SetCurrentPhase(phaseName string, index int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.CurrentPhase = phaseName
	c.state.CurrentPhaseIndex = index
}

// SetMissionProgress sets the mission progress (for testing).
func (c *Coordinator) SetMissionProgress(progress float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.MissionProgress = progress
}

// Verify interface compliance.
var _ coordinator.Service = (*Coordinator)(nil)
var _ coordinator.Extended = (*Coordinator)(nil)
var _ coordinator.AICoordinator = (*Coordinator)(nil)
var _ coordinator.LLMCoordinator = (*Coordinator)(nil)
