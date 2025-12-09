# Behaviors

How does a robot decide what to do next? Behaviors are the decision-making layer that sits above services and components. They observe the robot's state and environment, then select appropriate actions.

This chapter introduces behavior patterns used in robotics: finite state machines for simple sequential behaviors, behavior trees for complex hierarchical decisions, and reactive architectures for real-time response.

## The Behavior Problem

Robots face continuous decisions:

- Should I stop for this obstacle?
- Should I continue navigating or respond to a new command?
- Should I switch from searching to tracking when I find an object?
- How do I handle conflicting goals?

Without structure, decision logic becomes a tangled mess of if-else chains. Behavior patterns provide that structure.

## Behavior Interface

A simple behavior interface:

```go
// Behavior makes decisions based on robot state
type Behavior interface {
    // Update is called each tick
    Update(ctx context.Context, state *RobotState) (Action, error)

    // Name identifies this behavior
    Name() string
}

type RobotState struct {
    Position    *Pose
    Velocity    *Twist
    Sensors     map[string]SensorReading
    Detections  []Detection
    BatteryPct  float64
    IsEmergency bool
}

type Action struct {
    Type       ActionType
    Velocity   *Twist
    Goal       *Pose
    // ...
}
```

## Finite State Machines

FSMs are the simplest behavior structure: discrete states with transitions.

### State Machine Basics

```go
type State string

const (
    StateIdle     State = "idle"
    StateSearching State = "searching"
    StateTracking  State = "tracking"
    StateReturning State = "returning"
)

type FSM struct {
    current State
    states  map[State]StateHandler
}

type StateHandler interface {
    Enter(ctx context.Context, state *RobotState)
    Update(ctx context.Context, state *RobotState) (Action, State)
    Exit(ctx context.Context, state *RobotState)
}
```

### Implementing States

```go
type SearchingState struct {
    searchPattern *Pattern
    startTime     time.Time
}

func (s *SearchingState) Enter(ctx context.Context, state *RobotState) {
    s.startTime = time.Now()
    s.searchPattern.Reset()
}

func (s *SearchingState) Update(ctx context.Context, state *RobotState) (Action, State) {
    // Check for emergency
    if state.IsEmergency {
        return StopAction(), StateIdle
    }

    // Check for detection
    if len(state.Detections) > 0 {
        return Action{}, StateTracking
    }

    // Check for timeout
    if time.Since(s.startTime) > 5*time.Minute {
        return Action{}, StateReturning
    }

    // Continue search pattern
    nextVelocity := s.searchPattern.NextStep()
    return Action{Type: ActionMove, Velocity: nextVelocity}, StateSearching
}

func (s *SearchingState) Exit(ctx context.Context, state *RobotState) {
    log.Printf("Search completed after %v", time.Since(s.startTime))
}
```

### Running the FSM

```go
func (fsm *FSM) Run(ctx context.Context, state *RobotState) Action {
    handler := fsm.states[fsm.current]

    action, nextState := handler.Update(ctx, state)

    if nextState != fsm.current {
        handler.Exit(ctx, state)
        fsm.current = nextState
        fsm.states[nextState].Enter(ctx, state)
    }

    return action
}
```

### When FSMs Work Well

- Simple, linear sequences
- Clear state boundaries
- Few states (< 10)
- Transitions are obvious

### When FSMs Struggle

- Many states with similar transitions
- Hierarchical behaviors (states within states)
- Priority overrides (emergency stop from any state)
- Complex conditions

## Behavior Trees

Behavior trees scale better than FSMs for complex decisions.

### Core Concepts

Behavior trees are made of nodes that return one of three statuses:

```go
type Status int

const (
    Running Status = iota  // Still executing
    Success               // Completed successfully
    Failure               // Failed
)

type BTNode interface {
    Tick(ctx context.Context, bb *Blackboard) Status
}
```

The **Blackboard** is shared state:

```go
type Blackboard struct {
    data map[string]any
    mu   sync.RWMutex
}

func (b *Blackboard) Get(key string) any {
    b.mu.RLock()
    defer b.mu.RUnlock()
    return b.data[key]
}

func (b *Blackboard) Set(key string, value any) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.data[key] = value
}
```

### Control Nodes

**Sequence**: Run children in order, fail if any fails

```go
type Sequence struct {
    children []BTNode
    current  int
}

func (s *Sequence) Tick(ctx context.Context, bb *Blackboard) Status {
    for s.current < len(s.children) {
        status := s.children[s.current].Tick(ctx, bb)

        switch status {
        case Running:
            return Running
        case Failure:
            s.current = 0  // Reset for next tick
            return Failure
        case Success:
            s.current++
        }
    }

    s.current = 0
    return Success
}
```

**Selector**: Try children in order, succeed if any succeeds

```go
type Selector struct {
    children []BTNode
    current  int
}

func (s *Selector) Tick(ctx context.Context, bb *Blackboard) Status {
    for s.current < len(s.children) {
        status := s.children[s.current].Tick(ctx, bb)

        switch status {
        case Running:
            return Running
        case Success:
            s.current = 0
            return Success
        case Failure:
            s.current++
        }
    }

    s.current = 0
    return Failure
}
```

### Action Nodes

Leaf nodes that do actual work:

```go
type MoveToGoal struct {
    nav NavigationService
}

func (m *MoveToGoal) Tick(ctx context.Context, bb *Blackboard) Status {
    goal := bb.Get("goal").(*Pose)

    navigating, _ := m.nav.IsNavigating(ctx)
    if !navigating {
        m.nav.SetGoal(ctx, goal)
        return Running
    }

    position, _ := m.nav.GetPosition(ctx)
    if position.DistanceTo(goal) < 0.1 {
        return Success
    }

    return Running
}
```

### Condition Nodes

Check state without changing it:

```go
type HasTarget struct{}

func (h *HasTarget) Tick(ctx context.Context, bb *Blackboard) Status {
    if bb.Get("target") != nil {
        return Success
    }
    return Failure
}
```

### Decorator Nodes

Modify child behavior:

```go
// Inverter flips success/failure
type Inverter struct {
    child BTNode
}

func (i *Inverter) Tick(ctx context.Context, bb *Blackboard) Status {
    status := i.child.Tick(ctx, bb)
    switch status {
    case Success:
        return Failure
    case Failure:
        return Success
    default:
        return Running
    }
}

// Repeat runs child N times
type Repeat struct {
    child BTNode
    times int
    count int
}

func (r *Repeat) Tick(ctx context.Context, bb *Blackboard) Status {
    status := r.child.Tick(ctx, bb)

    if status == Success {
        r.count++
        if r.count >= r.times {
            r.count = 0
            return Success
        }
        return Running
    }

    if status == Failure {
        r.count = 0
        return Failure
    }

    return Running
}
```

### Building a Behavior Tree

```go
// Search and track behavior
tree := &Selector{
    children: []BTNode{
        // Priority 1: Emergency stop
        &Sequence{
            children: []BTNode{
                &IsEmergency{},
                &EmergencyStop{},
            },
        },
        // Priority 2: Low battery
        &Sequence{
            children: []BTNode{
                &BatteryLow{threshold: 20},
                &ReturnToBase{},
            },
        },
        // Priority 3: Track if have target
        &Sequence{
            children: []BTNode{
                &HasTarget{},
                &TrackTarget{},
            },
        },
        // Priority 4: Search
        &Search{pattern: spiralPattern},
    },
}
```

## Reactive Architectures

Reactive systems prioritize immediate response over planning.

### Subsumption Architecture

Lower layers can override higher layers:

```go
type SubsumptionLayer struct {
    name     string
    priority int
    behavior Behavior
}

type SubsumptionArchitecture struct {
    layers []SubsumptionLayer  // Sorted by priority (high first)
}

func (s *SubsumptionArchitecture) Update(ctx context.Context, state *RobotState) Action {
    for _, layer := range s.layers {
        action, activated := layer.behavior.TryActivate(ctx, state)
        if activated {
            return action
        }
    }
    return NoAction()
}
```

Example layers:

1. **Avoid collision** (highest priority)
2. **Navigate to goal**
3. **Wander randomly** (lowest priority)

### Priority-Based Selection

```go
type PriorityBehavior struct {
    behaviors []PrioritizedBehavior
}

type PrioritizedBehavior struct {
    behavior Behavior
    priority int
    active   func(*RobotState) bool
}

func (p *PriorityBehavior) Update(ctx context.Context, state *RobotState) Action {
    // Sort by priority
    sort.Slice(p.behaviors, func(i, j int) bool {
        return p.behaviors[i].priority > p.behaviors[j].priority
    })

    // Run highest priority active behavior
    for _, pb := range p.behaviors {
        if pb.active(state) {
            return pb.behavior.Update(ctx, state)
        }
    }

    return NoAction()
}
```

## Testing Behaviors

Behaviors need thorough testing.

### State-Based Testing

```go
func TestSearchToTrackTransition(t *testing.T) {
    fsm := NewSearchTrackFSM()

    // Start in idle
    assert.Equal(t, StateIdle, fsm.Current())

    // Trigger search
    state := &RobotState{Detections: nil}
    fsm.Update(ctx, state)
    fsm.Transition(StateSearching)

    // Find target
    state.Detections = []Detection{{Label: "target"}}
    action, nextState := fsm.Update(ctx, state)

    assert.Equal(t, StateTracking, nextState)
}
```

### Behavior Tree Testing

```go
func TestEmergencyStopPriority(t *testing.T) {
    bb := NewBlackboard()
    tree := buildSearchTree()

    // Normal operation
    bb.Set("emergency", false)
    bb.Set("target", &Target{})
    status := tree.Tick(ctx, bb)
    // Should be tracking

    // Emergency overrides
    bb.Set("emergency", true)
    status = tree.Tick(ctx, bb)
    // Should trigger emergency stop
}
```

### Simulation Testing

Test behaviors in simulated environments:

```go
func TestNavigationBehavior(t *testing.T) {
    sim := NewSimulator()
    robot := sim.SpawnRobot()

    behavior := NewNavigationBehavior(robot)
    goal := &Pose{X: 10, Y: 5}

    for !robot.AtGoal(goal) {
        state := robot.GetState()
        action := behavior.Update(ctx, state)
        sim.Execute(robot, action)
        sim.Step(100 * time.Millisecond)
    }

    assert.True(t, robot.AtGoal(goal))
}
```

## Choosing Behavior Patterns

| Situation | Recommended Pattern |
|-----------|-------------------|
| Simple sequences (< 5 states) | FSM |
| Hierarchical decisions | Behavior Tree |
| Real-time priorities | Subsumption |
| Complex missions | BT + Coordinator |
| Learning/adaptive | BT with dynamic weights |

Start with FSMs. Add behavior trees when FSMs become unwieldy. Always ensure safety behaviors have priority.

---

With behaviors covering decision-making, Chapter 10 explores coordinators—the top-level orchestration for complex missions and multi-robot systems.
