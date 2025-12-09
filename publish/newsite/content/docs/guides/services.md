---
title: "Working with Services"
description: "Build and use services for vision, navigation, and more"
weight: 20
---

# Working with Services

Services are software capabilities that process data, make decisions, and coordinate actions. Unlike components that abstract hardware, services are pure software.

## Components vs Services

| Aspect | Components | Services |
|--------|------------|----------|
| Purpose | Hardware abstraction | Software capabilities |
| Examples | Motors, cameras, sensors | Vision, navigation, SLAM |
| Dependencies | Physical hardware | Other resources |
| Location | Close to hardware | Anywhere on network |

**Components** answer: "What does this hardware do?"
**Services** answer: "What can this robot accomplish?"

## The Service Interface

Services implement the base Resource interface:

```go
type Service interface {
    resource.Resource
}
```

Specific service types add domain-specific methods:

```go
type VisionService interface {
    Service
    DetectObjects(ctx context.Context, img image.Image) (*Detections, error)
    Classify(ctx context.Context, img image.Image) (*Classifications, error)
}
```

## Built-in Service Types

### Vision Service

Process images to extract semantic information:

```go
type VisionService interface {
    Service
    DetectObjects(ctx context.Context, img image.Image) (*Detections, error)
    Classify(ctx context.Context, img image.Image) (*Classifications, error)
    Segment(ctx context.Context, img image.Image) (*SegmentationMask, error)
}

// Usage
img, _ := camera.Image(ctx)
detections, _ := visionService.DetectObjects(ctx, img)

for _, det := range detections.Detections {
    if det.Label == "person" && det.Confidence > 0.7 {
        log.Printf("Person detected at %v", det.BoundingBox)
    }
}
```

### Navigation Service

Plan and execute paths:

```go
type NavigationService interface {
    Service
    SetGoal(ctx context.Context, goal *Pose) error
    GetPath(ctx context.Context) (*Path, error)
    GetPosition(ctx context.Context) (*Pose, error)
    Cancel(ctx context.Context) error
    IsNavigating(ctx context.Context) (bool, error)
}
```

### SLAM Service

Simultaneous Localization and Mapping:

```go
type SLAMService interface {
    Service
    GetMap(ctx context.Context) (*Map, error)
    GetPosition(ctx context.Context) (*Pose, error)
    SaveMap(ctx context.Context, path string) error
    LoadMap(ctx context.Context, path string) error
}
```

## Creating Custom Services

### When to Create a Service

Create a service when you have:

- Pure software functionality (no hardware)
- Processing that depends on multiple components
- Stateful logic that persists across calls
- Capability that should be reusable

### Service Implementation

```go
type ObjectTracker struct {
    name    resource.Name
    camera  Camera
    vision  VisionService
    tracked map[int]*TrackedObject
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

    // Match detections to existing tracks, update positions
    // ... tracking logic ...

    return t.getTrackedObjects(), nil
}
```

### Service Registration

```go
func init() {
    registry.RegisterService("vision", "yolox", NewYOLOXVision)
}

func NewYOLOXVision(ctx context.Context, deps resource.Dependencies, conf resource.Config) (Service, error) {
    camName := conf.GetString("camera")
    camera, err := deps.Get(resource.MustParseName(camName))
    if err != nil {
        return nil, fmt.Errorf("camera %s not found: %w", camName, err)
    }

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

## Service Lifecycle

Services follow the same lifecycle as components:

1. **Creation**: Factory function called with dependencies
2. **Configuration**: Initial config applied
3. **Running**: Service processes requests
4. **Reconfiguration**: Config updates applied live
5. **Shutdown**: Close() called for cleanup

## Next Steps

- [NATS Messaging Guide](../nats/)
- [Configuration Guide](../configuration/)
- [Testing Guide](../testing/)
