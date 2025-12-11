# Coordinators

Coordinators sit at the top of Gorai's architecture, orchestrating missions that span multiple behaviors, services, and even multiple robots. They handle the big picture: task sequencing, resource allocation, failure recovery, and inter-robot coordination.

This chapter covers mission orchestration for single robots and coordination patterns for multi-robot systems.

## The Coordinator Role

Coordinators answer the question: "What should this robot (or fleet) accomplish?"

While behaviors decide moment-to-moment actions and services provide capabilities, coordinators plan and sequence high-level goals:

```
┌─────────────────────────────────────────────────────────┐
│                      Coordinator                        │
│   "Patrol Area A, then inspect Equipment B"            │
└─────────────────────────┬───────────────────────────────┘
                          │ Tasks
┌─────────────────────────▼───────────────────────────────┐
│                      Behaviors                          │
│   "Navigate to waypoint", "Track detected objects"      │
└─────────────────────────┬───────────────────────────────┘
                          │ Actions
┌─────────────────────────▼───────────────────────────────┐
│                      Services                           │
│   Navigation, Vision, SLAM                              │
└─────────────────────────┬───────────────────────────────┘
                          │ Commands
┌─────────────────────────▼───────────────────────────────┐
│                     Components                          │
│   Motors, Cameras, Sensors                              │
└─────────────────────────────────────────────────────────┘
```

## Coordinator Interface

```go
type Coordinator interface {
    // Start begins mission execution
    Start(ctx context.Context, mission Mission) error

    // Status returns current mission state
    Status(ctx context.Context) (MissionStatus, error)

    // Cancel aborts the current mission
    Cancel(ctx context.Context) error

    // Pause temporarily halts execution
    Pause(ctx context.Context) error

    // Resume continues paused execution
    Resume(ctx context.Context) error
}

type Mission struct {
    Name        string
    Tasks       []Task
    Constraints []Constraint
    Priority    int
}

type MissionStatus struct {
    State         MissionState
    CurrentTask   int
    Progress      float64
    StartTime     time.Time
    Errors        []error
}

type MissionState int

const (
    MissionPending MissionState = iota
    MissionRunning
    MissionPaused
    MissionCompleted
    MissionFailed
    MissionCanceled
)
```

## Mission Planning

### Task Decomposition

Missions break down into tasks:

```go
type Task struct {
    ID           string
    Type         TaskType
    Parameters   map[string]any
    Dependencies []string      // Task IDs this depends on
    Timeout      time.Duration
    Retries      int
}

type TaskType string

const (
    TaskNavigate  TaskType = "navigate"
    TaskInspect   TaskType = "inspect"
    TaskPickup    TaskType = "pickup"
    TaskDeliver   TaskType = "deliver"
    TaskWait      TaskType = "wait"
    TaskBehavior  TaskType = "behavior"
)
```

### Building Missions

```go
func CreatePatrolMission(waypoints []*Pose) Mission {
    tasks := make([]Task, 0, len(waypoints)*2)

    for i, wp := range waypoints {
        // Navigate to waypoint
        tasks = append(tasks, Task{
            ID:         fmt.Sprintf("nav_%d", i),
            Type:       TaskNavigate,
            Parameters: map[string]any{"goal": wp},
            Timeout:    5 * time.Minute,
        })

        // Inspect at waypoint
        tasks = append(tasks, Task{
            ID:           fmt.Sprintf("inspect_%d", i),
            Type:         TaskInspect,
            Parameters:   map[string]any{"duration": 30 * time.Second},
            Dependencies: []string{fmt.Sprintf("nav_%d", i)},
            Timeout:      2 * time.Minute,
        })
    }

    return Mission{
        Name:  "Patrol",
        Tasks: tasks,
    }
}
```

## Task Execution

### Sequential Execution

Simple linear execution:

```go
type SequentialExecutor struct {
    tasks   []Task
    current int
    robot   Robot
}

func (e *SequentialExecutor) Run(ctx context.Context) error {
    for e.current < len(e.tasks) {
        task := e.tasks[e.current]

        err := e.executeTask(ctx, task)
        if err != nil {
            return fmt.Errorf("task %s failed: %w", task.ID, err)
        }

        e.current++
    }
    return nil
}

func (e *SequentialExecutor) executeTask(ctx context.Context, task Task) error {
    ctx, cancel := context.WithTimeout(ctx, task.Timeout)
    defer cancel()

    switch task.Type {
    case TaskNavigate:
        goal := task.Parameters["goal"].(*Pose)
        return e.robot.NavigateTo(ctx, goal)

    case TaskInspect:
        duration := task.Parameters["duration"].(time.Duration)
        return e.robot.Inspect(ctx, duration)

    // ... other task types
    }

    return fmt.Errorf("unknown task type: %s", task.Type)
}
```

### Task Graph Execution

Handle dependencies and parallelism:

```go
type TaskGraph struct {
    tasks map[string]*Task
    deps  map[string][]string  // task -> dependencies
}

func (g *TaskGraph) Execute(ctx context.Context, executor TaskExecutor) error {
    completed := make(map[string]bool)
    running := make(map[string]context.CancelFunc)
    results := make(chan taskResult)

    for {
        // Find ready tasks
        ready := g.findReady(completed, running)

        if len(ready) == 0 && len(running) == 0 {
            // All done
            return nil
        }

        // Start ready tasks
        for _, task := range ready {
            taskCtx, cancel := context.WithCancel(ctx)
            running[task.ID] = cancel

            go func(t *Task) {
                err := executor.Execute(taskCtx, t)
                results <- taskResult{ID: t.ID, Err: err}
            }(task)
        }

        // Wait for a task to complete
        select {
        case result := <-results:
            delete(running, result.ID)
            if result.Err != nil {
                // Cancel all running tasks
                for _, cancel := range running {
                    cancel()
                }
                return result.Err
            }
            completed[result.ID] = true

        case <-ctx.Done():
            return ctx.Err()
        }
    }
}

func (g *TaskGraph) findReady(completed, running map[string]bool) []*Task {
    var ready []*Task

    for id, task := range g.tasks {
        if completed[id] || running[id] {
            continue
        }

        // Check all dependencies completed
        allDepsComplete := true
        for _, dep := range g.deps[id] {
            if !completed[dep] {
                allDepsComplete = false
                break
            }
        }

        if allDepsComplete {
            ready = append(ready, task)
        }
    }

    return ready
}
```

## Failure Recovery

### Retry Logic

```go
func (e *Executor) executeWithRetry(ctx context.Context, task Task) error {
    var lastErr error

    for attempt := 0; attempt <= task.Retries; attempt++ {
        if attempt > 0 {
            log.Printf("Retrying task %s (attempt %d/%d)", task.ID, attempt, task.Retries)
            time.Sleep(time.Second * time.Duration(attempt))  // Exponential backoff
        }

        err := e.executeTask(ctx, task)
        if err == nil {
            return nil
        }

        lastErr = err

        if !isRetryable(err) {
            return err
        }
    }

    return fmt.Errorf("task %s failed after %d attempts: %w", task.ID, task.Retries+1, lastErr)
}

func isRetryable(err error) bool {
    // Navigation failures are often retryable
    // Hardware failures typically aren't
    return errors.Is(err, ErrNavigationFailed) ||
           errors.Is(err, ErrTimeout)
}
```

### Recovery Behaviors

```go
type RecoveryStrategy interface {
    CanRecover(task Task, err error) bool
    Recover(ctx context.Context, task Task, err error) error
}

type BackupNavigation struct {
    nav NavigationService
}

func (b *BackupNavigation) CanRecover(task Task, err error) bool {
    return task.Type == TaskNavigate && errors.Is(err, ErrPathBlocked)
}

func (b *BackupNavigation) Recover(ctx context.Context, task Task, err error) error {
    // Back up and try alternate route
    if err := b.nav.MoveBackward(ctx, 1.0); err != nil {
        return err
    }

    // Replan
    goal := task.Parameters["goal"].(*Pose)
    return b.nav.SetGoal(ctx, goal)
}
```

## Multi-Robot Coordination

### Fleet Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Fleet Coordinator                      │
│              (Central or Distributed)                   │
└────────────────────────┬────────────────────────────────┘
                         │ NATS
         ┌───────────────┼───────────────┐
         │               │               │
         ▼               ▼               ▼
    ┌─────────┐     ┌─────────┐     ┌─────────┐
    │ Robot 1 │     │ Robot 2 │     │ Robot 3 │
    │ Coord   │     │ Coord   │     │ Coord   │
    └─────────┘     └─────────┘     └─────────┘
```

### Task Allocation

Assign tasks to robots:

```go
type FleetCoordinator struct {
    robots  map[string]*RobotProxy
    tasks   *TaskQueue
    allocs  map[string]string  // task -> robot
}

func (f *FleetCoordinator) Allocate(task Task) (string, error) {
    // Find capable robots
    capable := f.findCapable(task)
    if len(capable) == 0 {
        return "", fmt.Errorf("no robot capable of task %s", task.ID)
    }

    // Choose best robot (various strategies)
    best := f.selectBest(capable, task)

    // Assign
    f.allocs[task.ID] = best.ID
    best.AssignTask(task)

    return best.ID, nil
}

func (f *FleetCoordinator) selectBest(robots []*RobotProxy, task Task) *RobotProxy {
    var best *RobotProxy
    bestScore := math.MaxFloat64

    for _, robot := range robots {
        score := f.computeScore(robot, task)
        if score < bestScore {
            best = robot
            bestScore = score
        }
    }

    return best
}

func (f *FleetCoordinator) computeScore(robot *RobotProxy, task Task) float64 {
    // Consider: distance to task, current workload, battery level, etc.
    distance := robot.Position.DistanceTo(task.Location())
    workload := float64(len(robot.PendingTasks()))
    battery := 1.0 / robot.BatteryLevel()

    return distance*0.5 + workload*0.3 + battery*0.2
}
```

### Coordination via NATS

Robots coordinate through shared topics:

```go
// Robot announces status
statusPub := pub.New[*RobotStatus](node, "fleet.robots."+robotID+".status")

ticker := time.NewTicker(time.Second)
for range ticker.C {
    statusPub.Publish(ctx, &RobotStatus{
        ID:       robotID,
        Position: currentPosition(),
        Battery:  batteryLevel(),
        State:    currentState(),
    })
}

// Fleet coordinator subscribes to all robots
sub.New[*RobotStatus](node, "fleet.robots.*.status", func(status *RobotStatus) {
    coordinator.UpdateRobot(status)
})

// Task assignment over NATS
taskPub := pub.New[*TaskAssignment](node, "fleet.tasks.assign")
taskPub.Publish(ctx, &TaskAssignment{
    RobotID: targetRobot,
    Task:    task,
})

// Robot receives assignments
sub.New[*TaskAssignment](node, "fleet.tasks.assign", func(assign *TaskAssignment) {
    if assign.RobotID == robotID {
        localCoord.AcceptTask(assign.Task)
    }
})
```

### Conflict Resolution

Prevent robots from interfering:

```go
type ResourceManager struct {
    locks map[string]string  // resource -> robot holding lock
    mu    sync.Mutex
}

func (r *ResourceManager) Acquire(robotID, resourceID string, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)

    for time.Now().Before(deadline) {
        r.mu.Lock()
        holder, locked := r.locks[resourceID]
        if !locked {
            r.locks[resourceID] = robotID
            r.mu.Unlock()
            return nil
        }
        r.mu.Unlock()

        if holder == robotID {
            return nil  // Already hold it
        }

        time.Sleep(100 * time.Millisecond)
    }

    return fmt.Errorf("timeout acquiring %s", resourceID)
}

func (r *ResourceManager) Release(robotID, resourceID string) {
    r.mu.Lock()
    defer r.mu.Unlock()

    if r.locks[resourceID] == robotID {
        delete(r.locks, resourceID)
    }
}
```

## Practical Example: Patrol Coordinator

```go
type PatrolCoordinator struct {
    robot     Robot
    waypoints []*Pose
    current   int
    status    MissionStatus
}

func (p *PatrolCoordinator) Start(ctx context.Context, mission Mission) error {
    p.status = MissionStatus{State: MissionRunning, StartTime: time.Now()}

    for p.current < len(p.waypoints) {
        select {
        case <-ctx.Done():
            p.status.State = MissionCanceled
            return ctx.Err()
        default:
        }

        wp := p.waypoints[p.current]
        p.status.CurrentTask = p.current
        p.status.Progress = float64(p.current) / float64(len(p.waypoints))

        // Navigate to waypoint
        if err := p.robot.NavigateTo(ctx, wp); err != nil {
            p.status.State = MissionFailed
            p.status.Errors = append(p.status.Errors, err)
            return err
        }

        // Perform inspection
        if err := p.robot.Inspect(ctx, 30*time.Second); err != nil {
            log.Printf("Inspection at waypoint %d failed: %v", p.current, err)
            // Continue patrol despite inspection failure
        }

        p.current++
    }

    p.status.State = MissionCompleted
    p.status.Progress = 1.0
    return nil
}

func (p *PatrolCoordinator) Status(ctx context.Context) (MissionStatus, error) {
    return p.status, nil
}

func (p *PatrolCoordinator) Cancel(ctx context.Context) error {
    p.robot.Stop(ctx)
    p.status.State = MissionCanceled
    return nil
}
```

---

With coordinators completing the architecture from sensors to missions, Part III moves to practical development: setting up your environment, building the hello-sensor example, creating custom components, and testing strategies.
