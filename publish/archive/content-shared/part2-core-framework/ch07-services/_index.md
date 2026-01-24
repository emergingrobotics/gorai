# Chapter 7: Services

Services are the brains of a robot—software capabilities that process sensor data, make decisions, and coordinate actions. Unlike components that abstract hardware, services are pure software.

## 7.1 Components vs Services

The distinction matters for architecture:

| Aspect | Components | Services |
|--------|------------|----------|
| Purpose | Hardware abstraction | Software capabilities |
| Examples | Motors, cameras, sensors | Vision, navigation, SLAM |
| Dependencies | Physical hardware | Other resources (components/services) |
| Location | Close to hardware | Anywhere on network |
| State | Hardware state | Computed state |

**Components** answer: "What does this hardware do?"
**Services** answer: "What can this robot accomplish?"

A robot might have:
- Camera component: Captures images
- Vision service: Detects objects in those images
- Motor components: Spin wheels
- Navigation service: Plans paths and drives motors

## 7.2 The Service Interface

Services implement the same base Resource interface:

```go
// From services/service.go
type Service interface {
    resource.Resource
}
```

Specific service types add domain-specific methods:

```go
type VisionService interface {
    Service

    // Detect finds objects in an image.
    Detect(ctx context.Context, img image.Image) (*Detections, error)

    // Classify identifies what an image contains.
    Classify(ctx context.Context, img image.Image) (*Classifications, error)
}
```

### Service Registration

Services register themselves for discovery:

```go
func init() {
    registry.RegisterService("vision", "yolox", NewYOLOXVision)
}

func NewYOLOXVision(ctx context.Context, deps resource.Dependencies, conf resource.Config) (Service, error) {
    // Get camera dependency
    camName := conf.GetString("camera")
    camera, err := deps.Get(resource.MustParseName(camName))
    if err != nil {
        return nil, fmt.Errorf("camera %s not found: %w", camName, err)
    }

    // Load model
    modelPath := conf.GetString("model_path")
    model, err := loadModel(modelPath)
    if err != nil {
        return nil, err
    }

    return &YOLOXVision{
        camera: camera.(Camera),
        model:  model,
    }, nil
}
```

### Discovery Mechanisms

Find services by type:

```go
// Get all vision services
visionServices, _ := deps.GetByType("vision")

// Get specific service by name
detector, _ := deps.Get(resource.MustParseName("gorai:service:vision/detector"))
```


## 7.3 Built-in Service Types

### 7.3.1 Vision Service

Process images to extract semantic information:

```go
type VisionService interface {
    Service

    // Object detection
    DetectObjects(ctx context.Context, img image.Image) (*Detections, error)

    // Classification
    Classify(ctx context.Context, img image.Image) (*Classifications, error)

    // Segmentation
    Segment(ctx context.Context, img image.Image) (*SegmentationMask, error)
}

type Detection struct {
    Label      string
    Confidence float64
    BoundingBox Rectangle
}

type Detections struct {
    Detections []Detection
}
```

Usage:
```go
img, _ := camera.Image(ctx)
detections, _ := visionService.DetectObjects(ctx, img)

for _, det := range detections.Detections {
    if det.Label == "person" && det.Confidence > 0.7 {
        log.Printf("Person detected at %v", det.BoundingBox)
    }
}
```

*Cross-reference: See Chapter 12 for ML models powering vision services.*

### 7.3.2 Navigation Service

Plan and execute paths:

```go
type NavigationService interface {
    Service

    // SetGoal sets a navigation target.
    SetGoal(ctx context.Context, goal *Pose) error

    // GetPath returns the planned path to current goal.
    GetPath(ctx context.Context) (*Path, error)

    // GetPosition returns current estimated position.
    GetPosition(ctx context.Context) (*Pose, error)

    // Cancel stops current navigation.
    Cancel(ctx context.Context) error

    // IsNavigating returns true if actively navigating.
    IsNavigating(ctx context.Context) (bool, error)
}
```

Navigation services typically:
- Subscribe to sensor data (LiDAR, odometry)
- Maintain an internal map or use provided map
- Plan paths avoiding obstacles
- Publish velocity commands to base

### 7.3.3 SLAM Service

Simultaneous Localization and Mapping:

```go
type SLAMService interface {
    Service

    // GetMap returns the current map.
    GetMap(ctx context.Context) (*Map, error)

    // GetPosition returns position within the map.
    GetPosition(ctx context.Context) (*Pose, error)

    // SaveMap persists the current map.
    SaveMap(ctx context.Context, path string) error

    // LoadMap loads a previously saved map.
    LoadMap(ctx context.Context, path string) error
}
```

SLAM fuses:
- LiDAR scans
- Camera images
- Odometry
- IMU data

To produce:
- 2D or 3D map of environment
- Robot's position within that map

### 7.3.4 Motion Planning

Generate collision-free trajectories:

```go
type MotionService interface {
    Service

    // Plan generates a trajectory from current to goal pose.
    Plan(ctx context.Context, goal *Pose) (*Trajectory, error)

    // PlanWithConstraints plans respecting constraints.
    PlanWithConstraints(ctx context.Context, goal *Pose, constraints *Constraints) (*Trajectory, error)

    // Execute runs a trajectory on the arm/base.
    Execute(ctx context.Context, trajectory *Trajectory) error
}

type Constraints struct {
    MaxVelocity     float64
    MaxAcceleration float64
    Obstacles       []*Obstacle
    JointLimits     []JointLimit
}
```


## 7.4 Custom Services

### When to Create a Service

Create a service when you have:
- Pure software functionality (no hardware)
- Processing that depends on multiple components
- Stateful logic that persists across calls
- Capability that should be reusable

Examples:
- Object tracking: Maintains object identities across frames
- Battery monitor: Watches battery levels, triggers alerts
- Behavior coordinator: Implements state machine for robot behavior

### Service Implementation Example

```go
// Custom service: object tracking
type ObjectTracker struct {
    name    resource.Name
    camera  Camera
    vision  VisionService
    tracked map[int]*TrackedObject
    nextID  int
    mu      sync.RWMutex
}

func NewObjectTracker(deps resource.Dependencies, conf resource.Config) (*ObjectTracker, error) {
    camName := conf.GetString("camera")
    camera, _ := deps.Get(resource.MustParseName(camName))

    visName := conf.GetString("vision")
    vision, _ := deps.Get(resource.MustParseName(visName))

    return &ObjectTracker{
        name:    resource.NewServiceName("gorai", "tracking", "tracker"),
        camera:  camera.(Camera),
        vision:  vision.(VisionService),
        tracked: make(map[int]*TrackedObject),
    }, nil
}

func (t *ObjectTracker) Name() resource.Name {
    return t.name
}

func (t *ObjectTracker) Update(ctx context.Context) ([]TrackedObject, error) {
    img, err := t.camera.Image(ctx)
    if err != nil {
        return nil, err
    }

    detections, err := t.vision.DetectObjects(ctx, img)
    if err != nil {
        return nil, err
    }

    t.mu.Lock()
    defer t.mu.Unlock()

    // Match detections to existing tracks
    // Update positions, assign IDs to new objects
    // Remove stale tracks
    // ...

    return t.getTrackedObjects(), nil
}

func (t *ObjectTracker) GetObject(id int) (*TrackedObject, error) {
    t.mu.RLock()
    defer t.mu.RUnlock()

    obj, ok := t.tracked[id]
    if !ok {
        return nil, fmt.Errorf("object %d not found", id)
    }
    return obj, nil
}

// Resource interface methods
func (t *ObjectTracker) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
    return nil
}

func (t *ObjectTracker) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
    return nil, nil
}

func (t *ObjectTracker) Close(ctx context.Context) error {
    return nil
}
```

### Service Lifecycle

Services follow the same lifecycle as components:

1. **Creation**: Factory function called with dependencies
2. **Configuration**: Initial config applied
3. **Running**: Service processes requests
4. **Reconfiguration**: Config updates applied live
5. **Shutdown**: Close() called for cleanup

*Cross-reference: See Chapter 10 for complete custom service implementation guide.*

---

Chapter 8 covers setting up your development environment to build components and services.
