package coordinator_test

import (
	"context"
	"testing"

	"github.com/gorai/gorai/pkg/resource"
	"github.com/gorai/gorai/services"
	"github.com/gorai/gorai/services/behavior"
	"github.com/gorai/gorai/services/coordinator"
	"github.com/gorai/gorai/services/coordinator/fake"
)

func TestCoordinator_IsService(t *testing.T) {
	// Coordinator must implement service.Service
	var _ service.Service = (coordinator.Service)(nil)
}

func TestFakeCoordinator_StartStop(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "coordinator", "test")
	c := fake.NewWithName(name)

	// Initially not running
	running, err := c.IsRunning(ctx)
	if err != nil {
		t.Fatalf("IsRunning failed: %v", err)
	}
	if running {
		t.Error("expected not running initially")
	}

	// Start
	if err := c.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	running, err = c.IsRunning(ctx)
	if err != nil {
		t.Fatalf("IsRunning failed: %v", err)
	}
	if !running {
		t.Error("expected running after Start")
	}

	// Start again should fail
	if err := c.Start(ctx); err == nil {
		t.Error("expected error when starting already running coordinator")
	}

	// Stop
	if err := c.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	running, err = c.IsRunning(ctx)
	if err != nil {
		t.Fatalf("IsRunning failed: %v", err)
	}
	if running {
		t.Error("expected not running after Stop")
	}
}

func TestFakeCoordinator_GetState(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "coordinator", "test")
	c := fake.NewWithName(name)

	// Initial state
	state, err := c.GetState(ctx)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if state.Status != behavior.StatusIdle {
		t.Errorf("initial status = %v, want StatusIdle", state.Status)
	}

	// Start and check state
	c.Start(ctx)
	state, err = c.GetState(ctx)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if state.Status != behavior.StatusRunning {
		t.Errorf("status after start = %v, want StatusRunning", state.Status)
	}
}

func TestFakeCoordinator_Mission(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "coordinator", "test")
	c := fake.NewWithName(name)

	// Initially no mission
	mission, err := c.GetMission(ctx)
	if err != nil {
		t.Fatalf("GetMission failed: %v", err)
	}
	if mission != nil {
		t.Error("expected nil mission initially")
	}

	// Set mission
	testMission := &coordinator.Mission{
		ID:          "mission_1",
		Name:        "Test Mission",
		Description: "A test mission",
		Phases: []coordinator.Phase{
			{
				Name:        "phase_1",
				Description: "First phase",
				Behaviors:   []coordinator.BehaviorRef{{Name: "patrol", Required: true}},
			},
			{
				Name:        "phase_2",
				Description: "Second phase",
				Behaviors:   []coordinator.BehaviorRef{{Name: "dock", Required: true}},
			},
		},
		Priority:  1,
		OnFailure: coordinator.FailureAbort,
	}

	if err := c.SetMission(ctx, testMission); err != nil {
		t.Fatalf("SetMission failed: %v", err)
	}

	mission, err = c.GetMission(ctx)
	if err != nil {
		t.Fatalf("GetMission failed: %v", err)
	}
	if mission.ID != "mission_1" {
		t.Errorf("mission ID = %q, want 'mission_1'", mission.ID)
	}
	if len(mission.Phases) != 2 {
		t.Errorf("phase count = %d, want 2", len(mission.Phases))
	}

	// Check current phase was set
	state, _ := c.GetState(ctx)
	if state.CurrentPhase != "phase_1" {
		t.Errorf("current phase = %q, want 'phase_1'", state.CurrentPhase)
	}
}

func TestFakeCoordinator_GetProgress(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "coordinator", "test")
	c := fake.NewWithName(name)

	// Set mission
	testMission := &coordinator.Mission{
		ID:   "mission_1",
		Name: "Test Mission",
		Phases: []coordinator.Phase{
			{Name: "phase_1"},
			{Name: "phase_2"},
			{Name: "phase_3"},
			{Name: "phase_4"},
		},
	}
	c.SetMission(ctx, testMission)

	// Get progress
	progress, err := c.GetProgress(ctx)
	if err != nil {
		t.Fatalf("GetProgress failed: %v", err)
	}
	if progress.MissionID != "mission_1" {
		t.Errorf("mission ID = %q, want 'mission_1'", progress.MissionID)
	}
	if progress.TotalPhases != 4 {
		t.Errorf("total phases = %d, want 4", progress.TotalPhases)
	}
	if progress.CurrentPhase != 0 {
		t.Errorf("current phase = %d, want 0", progress.CurrentPhase)
	}
}

func TestFakeCoordinator_ManagedBehaviors(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "coordinator", "test")
	c := fake.NewWithName(name)

	// Initially no managed behaviors
	behaviors, err := c.GetManagedBehaviors(ctx)
	if err != nil {
		t.Fatalf("GetManagedBehaviors failed: %v", err)
	}
	if len(behaviors) != 0 {
		t.Errorf("initial behaviors count = %d, want 0", len(behaviors))
	}

	// Add behaviors
	c.AddManagedBehavior("patrol")
	c.AddManagedBehavior("inspect")
	c.AddManagedBehavior("dock")

	behaviors, err = c.GetManagedBehaviors(ctx)
	if err != nil {
		t.Fatalf("GetManagedBehaviors failed: %v", err)
	}
	if len(behaviors) != 3 {
		t.Errorf("behaviors count = %d, want 3", len(behaviors))
	}
}

func TestFakeCoordinator_PauseResume(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "coordinator", "test")
	c := fake.NewWithName(name)

	// Can't pause when not running
	if err := c.Pause(ctx); err == nil {
		t.Error("expected error when pausing non-running coordinator")
	}

	// Start and pause
	c.Start(ctx)
	if err := c.Pause(ctx); err != nil {
		t.Fatalf("Pause failed: %v", err)
	}

	paused, err := c.IsPaused(ctx)
	if err != nil {
		t.Fatalf("IsPaused failed: %v", err)
	}
	if !paused {
		t.Error("expected paused after Pause")
	}

	// Resume
	if err := c.Resume(ctx); err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	paused, err = c.IsPaused(ctx)
	if err != nil {
		t.Fatalf("IsPaused failed: %v", err)
	}
	if paused {
		t.Error("expected not paused after Resume")
	}
}

func TestFakeCoordinator_SkipPhase(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "coordinator", "test")
	c := fake.NewWithName(name)

	// Set mission
	testMission := &coordinator.Mission{
		ID: "mission_1",
		Phases: []coordinator.Phase{
			{Name: "phase_1"},
			{Name: "phase_2"},
			{Name: "phase_3"},
		},
	}
	c.SetMission(ctx, testMission)

	// Skip to next phase
	if err := c.SkipPhase(ctx); err != nil {
		t.Fatalf("SkipPhase failed: %v", err)
	}

	state, _ := c.GetState(ctx)
	if state.CurrentPhase != "phase_2" {
		t.Errorf("current phase = %q, want 'phase_2'", state.CurrentPhase)
	}
	if state.CurrentPhaseIndex != 1 {
		t.Errorf("current phase index = %d, want 1", state.CurrentPhaseIndex)
	}
}

func TestFakeCoordinator_CancelMission(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "coordinator", "test")
	c := fake.NewWithName(name)

	// Set mission
	testMission := &coordinator.Mission{ID: "mission_1", Phases: []coordinator.Phase{{Name: "phase_1"}}}
	c.SetMission(ctx, testMission)

	// Cancel
	if err := c.CancelMission(ctx); err != nil {
		t.Fatalf("CancelMission failed: %v", err)
	}

	mission, _ := c.GetMission(ctx)
	if mission != nil {
		t.Error("expected nil mission after cancel")
	}

	state, _ := c.GetState(ctx)
	if state.Status != behavior.StatusCanceled {
		t.Errorf("status = %v, want StatusCanceled", state.Status)
	}
}

func TestFakeCoordinator_DerivedSensors(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "coordinator", "test")
	c := fake.NewWithName(name)

	// Initially no derived sensors
	sensors, err := c.GetDerivedSensors(ctx)
	if err != nil {
		t.Fatalf("GetDerivedSensors failed: %v", err)
	}
	if len(sensors) != 0 {
		t.Errorf("initial sensors count = %d, want 0", len(sensors))
	}

	// Add derived sensor
	sensorName := resource.NewComponentName("gorai", "sensor", "mission_status")
	c.AddDerivedSensor(sensorName)

	sensors, err = c.GetDerivedSensors(ctx)
	if err != nil {
		t.Fatalf("GetDerivedSensors failed: %v", err)
	}
	if len(sensors) != 1 {
		t.Errorf("sensors count = %d, want 1", len(sensors))
	}
}

func TestFakeCoordinator_GetProperties(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "coordinator", "mission_coordinator")
	c := fake.NewWithName(name)

	props, err := c.GetProperties(ctx)
	if err != nil {
		t.Fatalf("GetProperties failed: %v", err)
	}

	if props.Name != "mission_coordinator" {
		t.Errorf("name = %q, want 'mission_coordinator'", props.Name)
	}
	if !props.SupportsParallelPhases {
		t.Error("expected SupportsParallelPhases to be true")
	}
	if !props.SupportsAI {
		t.Error("expected SupportsAI to be true")
	}
}

func TestFakeCoordinator_GetPhaseHistory(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "coordinator", "test")
	c := fake.NewWithName(name)

	// Add phase results
	c.AddPhaseResult(coordinator.PhaseResult{
		PhaseName: "phase_1",
		Status:    behavior.StatusSuccess,
	})
	c.AddPhaseResult(coordinator.PhaseResult{
		PhaseName: "phase_2",
		Status:    behavior.StatusSuccess,
	})
	c.AddPhaseResult(coordinator.PhaseResult{
		PhaseName: "phase_3",
		Status:    behavior.StatusFailure,
		Error:     "Test error",
	})

	// Get all history
	history, err := c.GetPhaseHistory(ctx, 0)
	if err != nil {
		t.Fatalf("GetPhaseHistory failed: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("history length = %d, want 3", len(history))
	}

	// Get limited history
	history, err = c.GetPhaseHistory(ctx, 2)
	if err != nil {
		t.Fatalf("GetPhaseHistory failed: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("limited history length = %d, want 2", len(history))
	}
}

func TestFakeCoordinator_AICoordinator(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "coordinator", "test")
	c := fake.NewWithName(name)

	// Test GenerateMission
	mission, err := c.GenerateMission(ctx, "Patrol the warehouse and check for anomalies")
	if err != nil {
		t.Fatalf("GenerateMission failed: %v", err)
	}
	if mission == nil {
		t.Fatal("expected non-nil mission")
	}
	if len(mission.Phases) != 2 {
		t.Errorf("generated phases = %d, want 2", len(mission.Phases))
	}

	// Set the generated mission
	c.SetMission(ctx, mission)

	// Test ExplainPlan
	explanation, err := c.ExplainPlan(ctx)
	if err != nil {
		t.Fatalf("ExplainPlan failed: %v", err)
	}
	if explanation == "" {
		t.Error("expected non-empty explanation")
	}

	// Test AdaptMission
	if err := c.AdaptMission(ctx, "Battery is low"); err != nil {
		t.Fatalf("AdaptMission failed: %v", err)
	}

	// Test GetPlanningMetadata
	metadata, err := c.GetPlanningMetadata(ctx)
	if err != nil {
		t.Fatalf("GetPlanningMetadata failed: %v", err)
	}
	if metadata.Type != "llm" {
		t.Errorf("planning type = %q, want 'llm'", metadata.Type)
	}
}

func TestFakeCoordinator_LLMCoordinator(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "coordinator", "test")
	c := fake.NewWithName(name)

	// Test GetLLMProvider
	provider, err := c.GetLLMProvider(ctx)
	if err != nil {
		t.Fatalf("GetLLMProvider failed: %v", err)
	}
	if provider != "fake" {
		t.Errorf("provider = %q, want 'fake'", provider)
	}

	// Test GetLLMModel
	model, err := c.GetLLMModel(ctx)
	if err != nil {
		t.Fatalf("GetLLMModel failed: %v", err)
	}
	if model != "fake-planner-1.0" {
		t.Errorf("model = %q, want 'fake-planner-1.0'", model)
	}

	// Generate mission to create planning history
	c.GenerateMission(ctx, "Test mission")

	// Test GetPlanningHistory
	history, err := c.GetPlanningHistory(ctx, 10)
	if err != nil {
		t.Fatalf("GetPlanningHistory failed: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("history length = %d, want 1", len(history))
	}
	if history[0].DecisionType != "plan" {
		t.Errorf("decision type = %q, want 'plan'", history[0].DecisionType)
	}

	// Test planning constraints
	constraints := []string{"no_restricted_zones", "prefer_safe_routes"}
	if err := c.SetPlanningConstraints(ctx, constraints); err != nil {
		t.Fatalf("SetPlanningConstraints failed: %v", err)
	}

	gotConstraints, err := c.GetPlanningConstraints(ctx)
	if err != nil {
		t.Fatalf("GetPlanningConstraints failed: %v", err)
	}
	if len(gotConstraints) != 2 {
		t.Errorf("constraints count = %d, want 2", len(gotConstraints))
	}
}

func TestFakeCoordinator_Name(t *testing.T) {
	name := resource.NewServiceName("gorai", "coordinator", "mission_coordinator")
	c := fake.NewWithName(name)

	if c.Name().String() != "gorai:service:coordinator/mission_coordinator" {
		t.Errorf("Name() = %q, want 'gorai:service:coordinator/mission_coordinator'", c.Name().String())
	}
}

func TestFakeCoordinator_DoCommand(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "coordinator", "test")
	c := fake.NewWithName(name)

	// Test get_state command
	result, err := c.DoCommand(ctx, map[string]any{
		"command": "get_state",
	})
	if err != nil {
		t.Fatalf("DoCommand get_state failed: %v", err)
	}
	if result["status"].(string) != "idle" {
		t.Errorf("status = %v, want 'idle'", result["status"])
	}

	// Test add_behavior command
	_, err = c.DoCommand(ctx, map[string]any{
		"command": "add_behavior",
		"name":    "test_behavior",
	})
	if err != nil {
		t.Fatalf("DoCommand add_behavior failed: %v", err)
	}

	behaviors, _ := c.GetManagedBehaviors(ctx)
	if len(behaviors) != 1 {
		t.Errorf("behaviors count = %d, want 1", len(behaviors))
	}

	// Test unknown command
	_, err = c.DoCommand(ctx, map[string]any{
		"command": "unknown",
	})
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestFakeCoordinator_Close(t *testing.T) {
	ctx := context.Background()
	name := resource.NewServiceName("gorai", "coordinator", "test")
	c := fake.NewWithName(name)

	c.Start(ctx)

	if err := c.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	running, _ := c.IsRunning(ctx)
	if running {
		t.Error("expected not running after Close")
	}
}

func TestFailurePolicy_String(t *testing.T) {
	tests := []struct {
		fp   coordinator.FailurePolicy
		want string
	}{
		{coordinator.FailureAbort, "abort"},
		{coordinator.FailureRetry, "retry"},
		{coordinator.FailureSkip, "skip"},
		{coordinator.FailureFallback, "fallback"},
		{coordinator.FailurePolicy(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.fp.String()
		if got != tt.want {
			t.Errorf("FailurePolicy(%d).String() = %q, want %q", tt.fp, got, tt.want)
		}
	}
}
