// Package fake provides a fake behavior implementation for testing.
package fake

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
	"github.com/gorai/gorai/service/behavior"
)

func init() {
	registry.RegisterService("behavior", "fake", New)
}

// Behavior is a fake behavior service for testing.
type Behavior struct {
	name           resource.Name
	mu             sync.RWMutex
	running        bool
	paused         bool
	state          behavior.State
	goal           *behavior.Goal
	derivedSensors []resource.Name
	history        []*behavior.TickResult
	properties     behavior.Properties

	// AI-specific
	modelName   string
	confidence  float64
	explanation string

	// LLM-specific
	llmProvider     string
	llmModel        string
	systemPrompt    string
	reasoningTrace  []behavior.ReasoningStep
	conversationHx  []behavior.ConversationTurn
}

// New creates a new fake behavior.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewServiceName("gorai", "behavior", nameStr)

	b := &Behavior{
		name:           name,
		running:        false,
		paused:         false,
		state:          behavior.State{Status: behavior.StatusIdle, Variables: make(map[string]any)},
		derivedSensors: []resource.Name{},
		history:        []*behavior.TickResult{},
		properties: behavior.Properties{
			Type:               behavior.BehaviorTypeCustom,
			Name:               nameStr,
			Description:        "Fake behavior for testing",
			TickRateHz:         10,
			SupportsGoals:      true,
			SupportsTicking:    true,
			DerivedSensorTypes: []string{},
		},
		modelName:   "fake_model",
		confidence:  0.95,
		explanation: "This is a fake behavior",
		llmProvider: "fake",
		llmModel:    "fake-llm-1.0",
		systemPrompt: "You are a fake robot assistant.",
	}

	return b, nil
}

// NewWithName creates a fake behavior with a specific resource name.
func NewWithName(name resource.Name) *Behavior {
	return &Behavior{
		name:           name,
		running:        false,
		paused:         false,
		state:          behavior.State{Status: behavior.StatusIdle, Variables: make(map[string]any)},
		derivedSensors: []resource.Name{},
		history:        []*behavior.TickResult{},
		properties: behavior.Properties{
			Type:               behavior.BehaviorTypeCustom,
			Name:               name.Name,
			Description:        "Fake behavior for testing",
			TickRateHz:         10,
			SupportsGoals:      true,
			SupportsTicking:    true,
			DerivedSensorTypes: []string{},
		},
		modelName:   "fake_model",
		confidence:  0.95,
		explanation: "This is a fake behavior",
		llmProvider: "fake",
		llmModel:    "fake-llm-1.0",
		systemPrompt: "You are a fake robot assistant.",
	}
}

// Name returns the behavior's resource name.
func (b *Behavior) Name() resource.Name {
	return b.name
}

// Reconfigure updates the behavior configuration.
func (b *Behavior) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand executes arbitrary commands.
func (b *Behavior) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "get_state":
			b.mu.RLock()
			defer b.mu.RUnlock()
			return map[string]any{
				"status":    b.state.Status.String(),
				"running":   b.running,
				"paused":    b.paused,
				"tick_count": b.state.TickCount,
			}, nil
		case "set_variable":
			if key, ok := cmd["key"].(string); ok {
				b.mu.Lock()
				b.state.Variables[key] = cmd["value"]
				b.mu.Unlock()
				return map[string]any{"status": "ok"}, nil
			}
			return nil, fmt.Errorf("missing key")
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (b *Behavior) Close(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.running = false
	b.state.Status = behavior.StatusIdle
	return nil
}

// Start begins behavior execution.
func (b *Behavior) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.running {
		return fmt.Errorf("behavior already running")
	}
	b.running = true
	b.paused = false
	b.state.Status = behavior.StatusRunning
	b.state.StartTime = time.Now()
	return nil
}

// Stop halts behavior execution.
func (b *Behavior) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.running = false
	b.paused = false
	b.state.Status = behavior.StatusIdle
	return nil
}

// IsRunning returns true if the behavior is active.
func (b *Behavior) IsRunning(ctx context.Context) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.running, nil
}

// GetState returns current behavior state.
func (b *Behavior) GetState(ctx context.Context) (*behavior.State, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	state := b.state
	state.Duration = time.Since(b.state.StartTime)
	return &state, nil
}

// SetGoal sets a goal for the behavior.
func (b *Behavior) SetGoal(ctx context.Context, goal *behavior.Goal) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.goal = goal
	return nil
}

// GetGoal returns the current goal.
func (b *Behavior) GetGoal(ctx context.Context) (*behavior.Goal, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.goal, nil
}

// Tick executes one cycle of the behavior.
func (b *Behavior) Tick(ctx context.Context) (*behavior.TickResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running || b.paused {
		return &behavior.TickResult{
			Status:   b.state.Status,
			Action:   "none",
			Duration: 0,
		}, nil
	}

	start := time.Now()
	b.state.TickCount++
	b.state.LastTick = start

	result := &behavior.TickResult{
		Status:   behavior.StatusRunning,
		Action:   "tick_" + fmt.Sprint(b.state.TickCount),
		Duration: time.Since(start),
	}

	b.history = append(b.history, result)
	if len(b.history) > 100 {
		b.history = b.history[1:]
	}

	return result, nil
}

// GetDerivedSensors returns sensors exposed by this behavior.
func (b *Behavior) GetDerivedSensors(ctx context.Context) ([]resource.Name, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make([]resource.Name, len(b.derivedSensors))
	copy(result, b.derivedSensors)
	return result, nil
}

// Extended interface methods

// GetProperties returns the behavior properties.
func (b *Behavior) GetProperties(ctx context.Context) (behavior.Properties, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.properties, nil
}

// Pause pauses behavior execution.
func (b *Behavior) Pause(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.running {
		return fmt.Errorf("behavior not running")
	}
	b.paused = true
	return nil
}

// Resume resumes a paused behavior.
func (b *Behavior) Resume(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.paused {
		return fmt.Errorf("behavior not paused")
	}
	b.paused = false
	return nil
}

// IsPaused returns true if the behavior is paused.
func (b *Behavior) IsPaused(ctx context.Context) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.paused, nil
}

// GetHistory returns recent tick history.
func (b *Behavior) GetHistory(ctx context.Context, limit int) ([]*behavior.TickResult, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if limit <= 0 || limit > len(b.history) {
		limit = len(b.history)
	}

	start := len(b.history) - limit
	result := make([]*behavior.TickResult, limit)
	copy(result, b.history[start:])
	return result, nil
}

// ClearGoal removes the current goal.
func (b *Behavior) ClearGoal(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.goal = nil
	return nil
}

// AIBehavior interface methods

// GetModel returns the ML model identifier.
func (b *Behavior) GetModel(ctx context.Context) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.modelName, nil
}

// GetConfidence returns confidence in the current decision.
func (b *Behavior) GetConfidence(ctx context.Context) (float64, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.confidence, nil
}

// GetExplanation returns a human-readable explanation.
func (b *Behavior) GetExplanation(ctx context.Context) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.explanation, nil
}

// Learn updates the model based on feedback.
func (b *Behavior) Learn(ctx context.Context, feedback *behavior.Feedback) error {
	// Fake implementation - just acknowledge the feedback
	return nil
}

// GetModelMetadata returns metadata about the AI model.
func (b *Behavior) GetModelMetadata(ctx context.Context) (*behavior.ModelMetadata, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return &behavior.ModelMetadata{
		Name:            b.modelName,
		Version:         "1.0.0",
		Framework:       "fake",
		Accelerator:     "cpu",
		LearningEnabled: false,
	}, nil
}

// LLMBehavior interface methods

// GetLLMProvider returns the LLM provider.
func (b *Behavior) GetLLMProvider(ctx context.Context) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.llmProvider, nil
}

// GetLLMModel returns the specific LLM model.
func (b *Behavior) GetLLMModel(ctx context.Context) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.llmModel, nil
}

// SendPrompt sends a prompt and gets a response.
func (b *Behavior) SendPrompt(ctx context.Context, prompt string) (string, error) {
	return "This is a fake response to: " + prompt, nil
}

// GetReasoningTrace returns the LLM's reasoning.
func (b *Behavior) GetReasoningTrace(ctx context.Context) ([]behavior.ReasoningStep, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make([]behavior.ReasoningStep, len(b.reasoningTrace))
	copy(result, b.reasoningTrace)
	return result, nil
}

// SetSystemPrompt updates the system prompt.
func (b *Behavior) SetSystemPrompt(ctx context.Context, prompt string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.systemPrompt = prompt
	return nil
}

// GetSystemPrompt returns the current system prompt.
func (b *Behavior) GetSystemPrompt(ctx context.Context) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.systemPrompt, nil
}

// GetConversationHistory returns recent conversation history.
func (b *Behavior) GetConversationHistory(ctx context.Context, limit int) ([]behavior.ConversationTurn, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if limit <= 0 || limit > len(b.conversationHx) {
		limit = len(b.conversationHx)
	}

	start := len(b.conversationHx) - limit
	result := make([]behavior.ConversationTurn, limit)
	copy(result, b.conversationHx[start:])
	return result, nil
}

// ClearConversationHistory clears the conversation history.
func (b *Behavior) ClearConversationHistory(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.conversationHx = []behavior.ConversationTurn{}
	return nil
}

// Test helper methods

// SetRunning sets the running state (for testing).
func (b *Behavior) SetRunning(running bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.running = running
	if running {
		b.state.Status = behavior.StatusRunning
	} else {
		b.state.Status = behavior.StatusIdle
	}
}

// SetStatus sets the behavior status (for testing).
func (b *Behavior) SetStatus(status behavior.Status) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.state.Status = status
}

// AddDerivedSensor adds a derived sensor (for testing).
func (b *Behavior) AddDerivedSensor(name resource.Name) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.derivedSensors = append(b.derivedSensors, name)
}

// SetConfidence sets the confidence (for testing).
func (b *Behavior) SetConfidence(confidence float64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.confidence = confidence
}

// SetExplanation sets the explanation (for testing).
func (b *Behavior) SetExplanation(explanation string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.explanation = explanation
}

// AddReasoningStep adds a reasoning step (for testing).
func (b *Behavior) AddReasoningStep(step behavior.ReasoningStep) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.reasoningTrace = append(b.reasoningTrace, step)
}

// AddConversationTurn adds a conversation turn (for testing).
func (b *Behavior) AddConversationTurn(turn behavior.ConversationTurn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.conversationHx = append(b.conversationHx, turn)
}

// SetVariable sets a state variable (for testing).
func (b *Behavior) SetVariable(key string, value any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.state.Variables[key] = value
}

// Verify interface compliance.
var _ behavior.Service = (*Behavior)(nil)
var _ behavior.Extended = (*Behavior)(nil)
var _ behavior.AIBehavior = (*Behavior)(nil)
var _ behavior.LLMBehavior = (*Behavior)(nil)
