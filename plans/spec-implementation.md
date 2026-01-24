# Specification Implementation Plan

## Overview

This plan implements the v0.2.0 specification updates to the Gorai codebase. The changes include:
- New component types: Power, Space, Link
- New service types: Behavior, Coordinator
- AI/LLM interfaces for behaviors and coordinators
- Derived sensors capability

## Current State Analysis

### Already Implemented:
- `pkg/resource/resource.go`: Resource, Sensor, Actuator interfaces
- `components/`: motor, camera, sensor, arm, base, gripper
- `services/`: vision, motion, navigation, slam, mlmodel
- `driver/`: gpio, i2c, serial, spi

### Needs Implementation:
1. **Power Component** - Battery, power supply, power distribution
2. **Space Component** - Container, workspace, zone
3. **Link Component** - Communication links (serial, IP, NATS, etc.)
4. **Behavior Service** - Decision-making with derived sensors, AI/LLM support
5. **Coordinator Service** - Mission orchestration, AI support

---

## Phase 1: Core Resource Extensions

Update `pkg/resource/resource.go` with new capability interfaces.

### 1.1 Add Power interface
```go
type Power interface {
    Resource
    GetCapacity(ctx context.Context) (float64, error)
    GetLevel(ctx context.Context) (float64, error)
    GetVoltage(ctx context.Context) (float64, error)
    GetCurrent(ctx context.Context) (float64, error)
    IsCharging(ctx context.Context) (bool, error)
}
```

### 1.2 Add Space interface
```go
type Space interface {
    Resource
    GetVolume(ctx context.Context) (float64, error)
    GetBounds(ctx context.Context) (*Bounds, error)
    GetContents(ctx context.Context) ([]string, error)
    IsEmpty(ctx context.Context) (bool, error)
}
```

### 1.3 Add Link interface and types
```go
type LinkType int
const (
    LinkTypeSerial LinkType = iota
    LinkTypeIP
    LinkTypeNATS
    LinkTypeCAN
    LinkTypeI2C
    LinkTypeSPI
)

type LinkDirection int
const (
    LinkBidirectional LinkDirection = iota
    LinkBroadcast
)

type LinkStats struct {
    BytesSent     uint64
    BytesReceived uint64
    MessagesSent  uint64
    MessagesRecv  uint64
    ErrorCount    uint64
    Latency       time.Duration
}

type Link interface {
    Resource
    Type() LinkType
    Direction() LinkDirection
    IsConnected(ctx context.Context) (bool, error)
    GetStats(ctx context.Context) (*LinkStats, error)
}
```

---

## Phase 2: Power Component

Create `/gorai/components/power/` package.

### Files:
- `power.go` - Interface definition
- `power_test.go` - Unit tests
- `fake/fake.go` - Fake implementation

---

## Phase 3: Space Component

Create `/gorai/components/space/` package.

### Files:
- `space.go` - Interface definition
- `space_test.go` - Unit tests
- `fake/fake.go` - Fake implementation

---

## Phase 4: Link Component

Create `/gorai/components/link/` package.

### Files:
- `link.go` - Interface definition with NATSLink
- `link_test.go` - Unit tests
- `fake/fake.go` - Fake implementation

---

## Phase 5: Behavior Service

Create `/gorai/services/behavior/` package.

### Files:
- `behavior.go` - Core interface, State, Goal, Status types
- `derived.go` - DerivedSensor interface
- `ai.go` - AIBehavior, LLMBehavior interfaces
- `behavior_test.go` - Unit tests
- `fake/fake.go` - Fake implementation

---

## Phase 6: Coordinator Service

Create `/gorai/services/coordinator/` package.

### Files:
- `coordinator.go` - Core interface, Mission, Phase types
- `ai.go` - AICoordinator interface
- `coordinator_test.go` - Unit tests
- `fake/fake.go` - Fake implementation

---

## Phase 7: Integration

1. Update `components/component.go` with Power, Space, Link types
2. Update `services/service.go` with Behavior, Coordinator types
3. Run full test suite

---

## Testing Strategy

Each phase runs tests before proceeding:
```bash
go test ./pkg/resource/...
go test ./components/power/...
go test ./components/space/...
go test ./components/link/...
go test ./services/behavior/...
go test ./services/coordinator/...
go test ./...
```
