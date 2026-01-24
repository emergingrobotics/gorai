package behavior_test

import (
	"context"
	"testing"
	"time"

	"github.com/gorai/gorai/pkg/resource"
	"github.com/gorai/gorai/services"
	"github.com/gorai/gorai/services/behavior"
	"github.com/gorai/gorai/services/behavior/fake"
)

func TestBehavior_IsService(t *testing.T) {
	// Behavior must implement service.Service
	var _ service.Service = (behavior.Service)(nil)
}

func TestFakeBehavior_StartStop(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "behavior", "test")
	b := fake.NewWithName(name)

	// Initially not running
	running, err := b.IsRunning(ctx)
	if err != nil {
		t.Fatalf("IsRunning failed: %v", err)
	}
	if running {
		t.Error("expected not running initially")
	}

	// Start
	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	running, err = b.IsRunning(ctx)
	if err != nil {
		t.Fatalf("IsRunning failed: %v", err)
	}
	if !running {
		t.Error("expected running after Start")
	}

	// Start again should fail
	if err := b.Start(ctx); err == nil {
		t.Error("expected error when starting already running behavior")
	}

	// Stop
	if err := b.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	running, err = b.IsRunning(ctx)
	if err != nil {
		t.Fatalf("IsRunning failed: %v", err)
	}
	if running {
		t.Error("expected not running after Stop")
	}
}

func TestFakeBehavior_GetState(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "behavior", "test")
	b := fake.NewWithName(name)

	// Initial state
	state, err := b.GetState(ctx)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if state.Status != behavior.StatusIdle {
		t.Errorf("initial status = %v, want StatusIdle", state.Status)
	}

	// Start and check state
	b.Start(ctx)
	state, err = b.GetState(ctx)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if state.Status != behavior.StatusRunning {
		t.Errorf("status after start = %v, want StatusRunning", state.Status)
	}

	// Set variable and check
	b.SetVariable("test_key", "test_value")
	state, err = b.GetState(ctx)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if state.Variables["test_key"] != "test_value" {
		t.Errorf("variable test_key = %v, want 'test_value'", state.Variables["test_key"])
	}
}

func TestFakeBehavior_Goals(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "behavior", "test")
	b := fake.NewWithName(name)

	// Initially no goal
	goal, err := b.GetGoal(ctx)
	if err != nil {
		t.Fatalf("GetGoal failed: %v", err)
	}
	if goal != nil {
		t.Error("expected nil goal initially")
	}

	// Set goal
	testGoal := &behavior.Goal{
		ID:       "goal_1",
		Type:     "navigate",
		Target:   map[string]float64{"x": 1.0, "y": 2.0},
		Priority: 1,
		Timeout:  30 * time.Second,
	}
	if err := b.SetGoal(ctx, testGoal); err != nil {
		t.Fatalf("SetGoal failed: %v", err)
	}

	goal, err = b.GetGoal(ctx)
	if err != nil {
		t.Fatalf("GetGoal failed: %v", err)
	}
	if goal.ID != "goal_1" {
		t.Errorf("goal ID = %q, want 'goal_1'", goal.ID)
	}
	if goal.Type != "navigate" {
		t.Errorf("goal type = %q, want 'navigate'", goal.Type)
	}

	// Clear goal
	if err := b.ClearGoal(ctx); err != nil {
		t.Fatalf("ClearGoal failed: %v", err)
	}

	goal, err = b.GetGoal(ctx)
	if err != nil {
		t.Fatalf("GetGoal failed: %v", err)
	}
	if goal != nil {
		t.Error("expected nil goal after ClearGoal")
	}
}

func TestFakeBehavior_Tick(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "behavior", "test")
	b := fake.NewWithName(name)

	// Start behavior
	b.Start(ctx)

	// Tick
	result, err := b.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick failed: %v", err)
	}
	if result.Status != behavior.StatusRunning {
		t.Errorf("tick status = %v, want StatusRunning", result.Status)
	}
	if result.Action != "tick_1" {
		t.Errorf("tick action = %q, want 'tick_1'", result.Action)
	}

	// Tick again
	result, err = b.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick failed: %v", err)
	}
	if result.Action != "tick_2" {
		t.Errorf("tick action = %q, want 'tick_2'", result.Action)
	}

	// Check state tick count
	state, _ := b.GetState(ctx)
	if state.TickCount != 2 {
		t.Errorf("tick count = %d, want 2", state.TickCount)
	}
}

func TestFakeBehavior_PauseResume(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "behavior", "test")
	b := fake.NewWithName(name)

	// Can't pause when not running
	if err := b.Pause(ctx); err == nil {
		t.Error("expected error when pausing non-running behavior")
	}

	// Start and pause
	b.Start(ctx)
	if err := b.Pause(ctx); err != nil {
		t.Fatalf("Pause failed: %v", err)
	}

	paused, err := b.IsPaused(ctx)
	if err != nil {
		t.Fatalf("IsPaused failed: %v", err)
	}
	if !paused {
		t.Error("expected paused after Pause")
	}

	// Tick when paused should do nothing
	result, err := b.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick failed: %v", err)
	}
	if result.Action != "none" {
		t.Errorf("tick action when paused = %q, want 'none'", result.Action)
	}

	// Resume
	if err := b.Resume(ctx); err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	paused, err = b.IsPaused(ctx)
	if err != nil {
		t.Fatalf("IsPaused failed: %v", err)
	}
	if paused {
		t.Error("expected not paused after Resume")
	}

	// Can't resume when not paused
	if err := b.Resume(ctx); err == nil {
		t.Error("expected error when resuming non-paused behavior")
	}
}

func TestFakeBehavior_DerivedSensors(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "behavior", "test")
	b := fake.NewWithName(name)

	// Initially no derived sensors
	sensors, err := b.GetDerivedSensors(ctx)
	if err != nil {
		t.Fatalf("GetDerivedSensors failed: %v", err)
	}
	if len(sensors) != 0 {
		t.Errorf("initial derived sensors count = %d, want 0", len(sensors))
	}

	// Add derived sensor
	sensorName := resource.NewComponentName("gorai", "sensor", "estimated_pose")
	b.AddDerivedSensor(sensorName)

	sensors, err = b.GetDerivedSensors(ctx)
	if err != nil {
		t.Fatalf("GetDerivedSensors failed: %v", err)
	}
	if len(sensors) != 1 {
		t.Errorf("derived sensors count = %d, want 1", len(sensors))
	}
	if sensors[0].String() != sensorName.String() {
		t.Errorf("derived sensor = %q, want %q", sensors[0].String(), sensorName.String())
	}
}

func TestFakeBehavior_GetProperties(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "behavior", "patrol")
	b := fake.NewWithName(name)

	props, err := b.GetProperties(ctx)
	if err != nil {
		t.Fatalf("GetProperties failed: %v", err)
	}

	if props.Name != "patrol" {
		t.Errorf("name = %q, want 'patrol'", props.Name)
	}
	if !props.SupportsGoals {
		t.Error("expected SupportsGoals to be true")
	}
	if !props.SupportsTicking {
		t.Error("expected SupportsTicking to be true")
	}
}

func TestFakeBehavior_GetHistory(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "behavior", "test")
	b := fake.NewWithName(name)

	b.Start(ctx)

	// Generate some ticks
	for i := 0; i < 5; i++ {
		b.Tick(ctx)
	}

	// Get all history
	history, err := b.GetHistory(ctx, 0)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(history) != 5 {
		t.Errorf("history length = %d, want 5", len(history))
	}

	// Get limited history
	history, err = b.GetHistory(ctx, 3)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("limited history length = %d, want 3", len(history))
	}
}

func TestFakeBehavior_AIBehavior(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "behavior", "test")
	b := fake.NewWithName(name)

	// Test GetModel
	model, err := b.GetModel(ctx)
	if err != nil {
		t.Fatalf("GetModel failed: %v", err)
	}
	if model != "fake_model" {
		t.Errorf("model = %q, want 'fake_model'", model)
	}

	// Test GetConfidence
	conf, err := b.GetConfidence(ctx)
	if err != nil {
		t.Fatalf("GetConfidence failed: %v", err)
	}
	if conf != 0.95 {
		t.Errorf("confidence = %v, want 0.95", conf)
	}

	b.SetConfidence(0.8)
	conf, _ = b.GetConfidence(ctx)
	if conf != 0.8 {
		t.Errorf("confidence = %v, want 0.8", conf)
	}

	// Test GetExplanation
	explanation, err := b.GetExplanation(ctx)
	if err != nil {
		t.Fatalf("GetExplanation failed: %v", err)
	}
	if explanation != "This is a fake behavior" {
		t.Errorf("explanation = %q, want 'This is a fake behavior'", explanation)
	}

	b.SetExplanation("New explanation")
	explanation, _ = b.GetExplanation(ctx)
	if explanation != "New explanation" {
		t.Errorf("explanation = %q, want 'New explanation'", explanation)
	}

	// Test Learn
	feedback := &behavior.Feedback{
		GoalID:  "goal_1",
		Outcome: behavior.OutcomeSuccess,
		Reward:  1.0,
	}
	if err := b.Learn(ctx, feedback); err != nil {
		t.Fatalf("Learn failed: %v", err)
	}

	// Test GetModelMetadata
	metadata, err := b.GetModelMetadata(ctx)
	if err != nil {
		t.Fatalf("GetModelMetadata failed: %v", err)
	}
	if metadata.Name != "fake_model" {
		t.Errorf("metadata name = %q, want 'fake_model'", metadata.Name)
	}
}

func TestFakeBehavior_LLMBehavior(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "behavior", "test")
	b := fake.NewWithName(name)

	// Test GetLLMProvider
	provider, err := b.GetLLMProvider(ctx)
	if err != nil {
		t.Fatalf("GetLLMProvider failed: %v", err)
	}
	if provider != "fake" {
		t.Errorf("provider = %q, want 'fake'", provider)
	}

	// Test GetLLMModel
	model, err := b.GetLLMModel(ctx)
	if err != nil {
		t.Fatalf("GetLLMModel failed: %v", err)
	}
	if model != "fake-llm-1.0" {
		t.Errorf("model = %q, want 'fake-llm-1.0'", model)
	}

	// Test SendPrompt
	response, err := b.SendPrompt(ctx, "Hello")
	if err != nil {
		t.Fatalf("SendPrompt failed: %v", err)
	}
	if response != "This is a fake response to: Hello" {
		t.Errorf("response = %q, want 'This is a fake response to: Hello'", response)
	}

	// Test system prompt
	if err := b.SetSystemPrompt(ctx, "New prompt"); err != nil {
		t.Fatalf("SetSystemPrompt failed: %v", err)
	}
	prompt, err := b.GetSystemPrompt(ctx)
	if err != nil {
		t.Fatalf("GetSystemPrompt failed: %v", err)
	}
	if prompt != "New prompt" {
		t.Errorf("prompt = %q, want 'New prompt'", prompt)
	}

	// Test reasoning trace
	b.AddReasoningStep(behavior.ReasoningStep{
		Step:    1,
		Thought: "I need to navigate",
		Action:  "navigate_to",
	})
	trace, err := b.GetReasoningTrace(ctx)
	if err != nil {
		t.Fatalf("GetReasoningTrace failed: %v", err)
	}
	if len(trace) != 1 {
		t.Errorf("trace length = %d, want 1", len(trace))
	}
	if trace[0].Thought != "I need to navigate" {
		t.Errorf("thought = %q, want 'I need to navigate'", trace[0].Thought)
	}

	// Test conversation history
	b.AddConversationTurn(behavior.ConversationTurn{
		Role:    "user",
		Content: "Go to the kitchen",
	})
	history, err := b.GetConversationHistory(ctx, 10)
	if err != nil {
		t.Fatalf("GetConversationHistory failed: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("history length = %d, want 1", len(history))
	}

	// Clear conversation history
	if err := b.ClearConversationHistory(ctx); err != nil {
		t.Fatalf("ClearConversationHistory failed: %v", err)
	}
	history, _ = b.GetConversationHistory(ctx, 10)
	if len(history) != 0 {
		t.Errorf("history length after clear = %d, want 0", len(history))
	}
}

func TestFakeBehavior_Name(t *testing.T) {
	name := resource.NewServiceName("gorai", "behavior", "patrol_behavior")
	b := fake.NewWithName(name)

	if b.Name().String() != "gorai:service:behavior/patrol_behavior" {
		t.Errorf("Name() = %q, want 'gorai:service:behavior/patrol_behavior'", b.Name().String())
	}
}

func TestFakeBehavior_DoCommand(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "behavior", "test")
	b := fake.NewWithName(name)

	// Test get_state command
	result, err := b.DoCommand(ctx, map[string]any{
		"command": "get_state",
	})
	if err != nil {
		t.Fatalf("DoCommand get_state failed: %v", err)
	}
	if result["status"].(string) != "idle" {
		t.Errorf("status = %v, want 'idle'", result["status"])
	}

	// Test set_variable command
	_, err = b.DoCommand(ctx, map[string]any{
		"command": "set_variable",
		"key":     "test_key",
		"value":   "test_value",
	})
	if err != nil {
		t.Fatalf("DoCommand set_variable failed: %v", err)
	}

	state, _ := b.GetState(ctx)
	if state.Variables["test_key"] != "test_value" {
		t.Errorf("variable = %v, want 'test_value'", state.Variables["test_key"])
	}

	// Test unknown command
	_, err = b.DoCommand(ctx, map[string]any{
		"command": "unknown",
	})
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestFakeBehavior_Close(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "behavior", "test")
	b := fake.NewWithName(name)

	b.Start(ctx)

	if err := b.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	running, _ := b.IsRunning(ctx)
	if running {
		t.Error("expected not running after Close")
	}
}

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status behavior.Status
		want   string
	}{
		{behavior.StatusIdle, "idle"},
		{behavior.StatusRunning, "running"},
		{behavior.StatusSuccess, "success"},
		{behavior.StatusFailure, "failure"},
		{behavior.StatusCanceled, "canceled"},
		{behavior.Status(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.status.String()
		if got != tt.want {
			t.Errorf("Status(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestBehaviorType_String(t *testing.T) {
	tests := []struct {
		bt   behavior.BehaviorType
		want string
	}{
		{behavior.BehaviorTypeBehaviorTree, "behavior_tree"},
		{behavior.BehaviorTypeStateMachine, "state_machine"},
		{behavior.BehaviorTypeSubsumption, "subsumption"},
		{behavior.BehaviorTypeUtility, "utility"},
		{behavior.BehaviorTypeAIAgent, "ai_agent"},
		{behavior.BehaviorTypeLLMAgent, "llm_agent"},
		{behavior.BehaviorTypeCustom, "custom"},
		{behavior.BehaviorType(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.bt.String()
		if got != tt.want {
			t.Errorf("BehaviorType(%d).String() = %q, want %q", tt.bt, got, tt.want)
		}
	}
}

func TestOutcome_String(t *testing.T) {
	tests := []struct {
		outcome behavior.Outcome
		want    string
	}{
		{behavior.OutcomeSuccess, "success"},
		{behavior.OutcomePartial, "partial"},
		{behavior.OutcomeFailure, "failure"},
		{behavior.OutcomeTimeout, "timeout"},
		{behavior.OutcomeCanceled, "canceled"},
		{behavior.Outcome(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.outcome.String()
		if got != tt.want {
			t.Errorf("Outcome(%d).String() = %q, want %q", tt.outcome, got, tt.want)
		}
	}
}
