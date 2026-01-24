# Gorai Core Implementation Plan

**Version 0.1.0**
**Generated: 2024-12-07**

This document provides a comprehensive implementation plan for the Gorai framework core, based on the specifications in `specs/code-organization.md` and `specs/gorai-framework-specification.md`. The final phases include implementing `specs/hello-sensor-design.md` as a validation example.

---

## Table of Contents

1. [Current State Analysis](#current-state-analysis)
2. [Gap Analysis: Spec vs Implementation](#gap-analysis)
3. [Implementation Phases](#implementation-phases)
   - [Phase 1: Protocol Buffer Foundation](#phase-1-protocol-buffer-foundation)
   - [Phase 2: Core Package Completion](#phase-2-core-package-completion)
   - [Phase 3: Resource Model Implementation](#phase-3-resource-model-implementation)
   - [Phase 4: Component Interface Refinement](#phase-4-component-interface-refinement)
   - [Phase 5: Network Transparency (NWS/NWC)](#phase-5-network-transparency)
   - [Phase 6: Hello Sensor Implementation](#phase-6-hello-sensor-implementation)
4. [Testing Strategy](#testing-strategy)
5. [Verification Checklist](#verification-checklist)

---

## Current State Analysis

### What Exists (Implemented)

| Component | Status | Notes |
|-----------|--------|-------|
| `pkg/node` | **Partial** | Node struct exists, basic lifecycle, missing some spec methods |
| `pkg/pub` | **Complete** | Generic `Publisher[T]` with protobuf support |
| `pkg/sub` | **Complete** | Generic `Subscriber[T]` with protobuf support |
| `pkg/services` | **Complete** | Generic `Server[Req,Resp]` and `Client[Req,Resp]` |
| `pkg/action` | **Skeleton** | Types defined, methods not implemented |
| `pkg/param` | **Complete** | NATS KV-backed parameter store |
| `pkg/registry` | **Complete** | Component/service registration pattern |
| `pkg/config` | **Partial** | Basic config loading, needs hot-reload |
| `pkg/tf` | **Complete** | Transform buffer implementation |
| `pkg/log` | **Complete** | Structured logging wrapper |
| `pkg/nats` | **Complete** | NATS connection management |
| `components/*` | **Partial** | Interfaces defined, fake implementations exist |
| `services/*` | **Partial** | Interfaces defined, no implementations |
| `driver/*` | **Interfaces only** | No actual driver implementations |
| `accel/*` | **Skeleton** | CPU stub only, no real accelerators |
| `api/proto/*` | **Partial** | std, geometry, sensor, vision defined; action, control, nav, ml empty |
| `nws/` | **Skeleton** | NWS/NWC structure, methods not implemented |
| `cmd/gorai` | **Partial** | Basic CLI structure, stubs for most commands |
| Tests | **None** | 0 test files in the codebase |

### What's Missing (Required by Spec)

1. **Protocol Buffers**:
   - `api/proto/gorai/action/action.proto` - empty
   - `api/proto/gorai/control/control.proto` - empty
   - `api/proto/gorai/nav/nav.proto` - empty
   - `api/proto/gorai/ml/ml.proto` - empty
   - Message envelope with metadata
   - Temperature sensor proto (for hello-sensor)

2. **Core Packages**:
   - `pkg/resource` - Resource interface as defined in spec (currently inlined in component)
   - Full `pkg/node` API (Spin, SpinOnce, FullName, JetStream access)
   - Action server/client implementation
   - QoS options for pub/sub (BestEffort, Reliable, Retained, History)

3. **Configuration**:
   - Hot reconfiguration without restart
   - Configuration change detection and diff

4. **Network Transparency**:
   - NWS `Wrap()` function implementation
   - NWC `Connect[T]()` function implementation
   - Remote resource access via NATS RPC

5. **CLI**:
   - `gorai topic list/echo/pub/hz/info`
   - `gorai service list/call/info`
   - `gorai action list/send/cancel`
   - `gorai param list/get/set/delete`
   - `gorai node list/info/kill`
   - `gorai launch <config.json>`

6. **Testing Infrastructure**:
   - `internal/testutil` needs expansion
   - Embedded NATS for testing
   - Mock/fake component patterns

---

## Gap Analysis

### Priority 1: Foundation (Required for Hello Sensor)

| Gap | Spec Requirement | Current State | Impact |
|-----|------------------|---------------|--------|
| Proto files | Complete proto definitions | 4 of 8 protos empty | Blocks typed messaging |
| Resource interface | Unified Resource interface | Scattered across packages | Blocks proper component model |
| Publisher QoS | QoS options | Basic pub only | Limits reliability control |
| Tests | All packages tested | 0 tests | No verification |

### Priority 2: Core Functionality

| Gap | Spec Requirement | Current State | Impact |
|-----|------------------|---------------|--------|
| Action pattern | Full Server/Client | Skeleton only | No long-running tasks |
| Hot config | Reconfigure without restart | Not implemented | Requires restart for changes |
| NWS/NWC | Network transparency | Skeleton only | No remote access |
| CLI | Full tooling | Stubs | Poor developer experience |

### Priority 3: Nice to Have

| Gap | Spec Requirement | Current State | Impact |
|-----|------------------|---------------|--------|
| Bag recording | Record/playback | Not implemented | No data recording |
| TF echoing | CLI TF commands | Not implemented | Limited debugging |

---

## Implementation Phases

### Phase 1: Protocol Buffer Foundation

**Goal**: Complete all Protocol Buffer definitions required by the spec.

#### Step 1.1: Complete action.proto

**File**: `api/proto/gorai/action/action.proto`

**Implementation**:
```protobuf
syntax = "proto3";
package gorai.action;

option go_package = "github.com/gorai/gorai/api/gen/gorai/action";

import "gorai/std/std.proto";
import "google/protobuf/any.proto";

message GoalID {
    string id = 1;
    gorai.std.Timestamp stamp = 2;
}

message GoalStatus {
    GoalID goal_id = 1;

    enum Status {
        STATUS_UNKNOWN = 0;
        STATUS_ACCEPTED = 1;
        STATUS_EXECUTING = 2;
        STATUS_CANCELING = 3;
        STATUS_SUCCEEDED = 4;
        STATUS_CANCELED = 5;
        STATUS_ABORTED = 6;
    }
    Status status = 2;
    string text = 3;
}

message GoalStatusArray {
    gorai.std.Header header = 1;
    repeated GoalStatus status_list = 2;
}

message CancelGoal {
    GoalID goal_id = 1;
    gorai.std.Timestamp stamp = 2;
}

message CancelGoalResponse {
    enum Code {
        ERROR_NONE = 0;
        ERROR_REJECTED = 1;
        ERROR_UNKNOWN_GOAL = 2;
        ERROR_GOAL_TERMINATED = 3;
    }
    Code return_code = 1;
    repeated GoalID goals_canceling = 2;
}
```

**Unit Test**: `api/gen/gorai/action/action_test.go`
```go
package action_test

import (
    "testing"

    "github.com/gorai/gorai/api/gen/gorai/action"
    "google.golang.org/protobuf/proto"
)

func TestGoalStatus_Marshal(t *testing.T) {
    gs := &action.GoalStatus{
        GoalId: &action.GoalID{Id: "test-goal"},
        Status: action.GoalStatus_STATUS_EXECUTING,
        Text:   "Running",
    }

    data, err := proto.Marshal(gs)
    if err != nil {
        t.Fatalf("failed to marshal: %v", err)
    }

    var decoded action.GoalStatus
    if err := proto.Unmarshal(data, &decoded); err != nil {
        t.Fatalf("failed to unmarshal: %v", err)
    }

    if decoded.GoalId.Id != "test-goal" {
        t.Errorf("expected goal_id 'test-goal', got %q", decoded.GoalId.Id)
    }
    if decoded.Status != action.GoalStatus_STATUS_EXECUTING {
        t.Errorf("expected STATUS_EXECUTING, got %v", decoded.Status)
    }
}
```

**Verification**:
```bash
buf lint api/proto
buf generate
go test ./api/gen/gorai/action/...
```

---

#### Step 1.2: Complete control.proto

**File**: `api/proto/gorai/control/control.proto`

**Implementation**: (as defined in spec, includes JointCommand, JointTrajectory, MotorCommand, etc.)

**Unit Test**: `api/gen/gorai/control/control_test.go`
- Test JointCommand marshaling
- Test MotorCommand with different modes
- Test GripperCommand position range

**Verification**:
```bash
buf lint api/proto
buf generate
go test ./api/gen/gorai/control/...
```

---

#### Step 1.3: Complete nav.proto

**File**: `api/proto/gorai/nav/nav.proto`

**Implementation**: (Path, Odometry, OccupancyGrid, Waypoint, NavigationGoal, etc.)

**Unit Test**: `api/gen/gorai/nav/nav_test.go`
- Test Path with multiple poses
- Test OccupancyGrid data serialization
- Test Waypoint properties

**Verification**:
```bash
buf lint api/proto
buf generate
go test ./api/gen/gorai/nav/...
```

---

#### Step 1.4: Complete ml.proto

**File**: `api/proto/gorai/ml/ml.proto`

**Implementation**: (Tensor, TensorList, ModelMetadata, InferenceRequest/Response)

**Unit Test**: `api/gen/gorai/ml/ml_test.go`
- Test Tensor with different data types
- Test shape serialization
- Test InferenceRequest/Response roundtrip

**Verification**:
```bash
buf lint api/proto
buf generate
go test ./api/gen/gorai/ml/...
```

---

#### Step 1.5: Add temperature.proto for Hello Sensor

**File**: `api/proto/gorai/sensor/temperature.proto`

**Implementation**: (as defined in hello-sensor-design.md)
- TemperatureReading
- TemperatureSensorInfo
- TemperatureDiagnostics

**Unit Test**: `api/gen/gorai/sensor/temperature_test.go`
- Test TemperatureReading with header
- Test celsius/fahrenheit values
- Test zone and source fields

**Verification**:
```bash
buf lint api/proto
buf generate
go test ./api/gen/gorai/sensor/...
```

---

### Phase 2: Core Package Completion

**Goal**: Complete core packages to spec with full test coverage.

#### Step 2.1: Create pkg/resource Package

**Files**:
- `pkg/resource/resource.go`
- `pkg/resource/name.go`
- `pkg/resource/resource_test.go`

**Implementation**:

```go
// pkg/resource/resource.go
package resource

import "context"

// Resource is the base interface for all Gorai components and services.
type Resource interface {
    // Name returns the unique resource identifier.
    Name() Name

    // Reconfigure updates the resource with new configuration.
    Reconfigure(ctx context.Context, deps Dependencies, conf Config) error

    // DoCommand executes arbitrary commands for extensibility.
    DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error)

    // Close releases all resources.
    Close(ctx context.Context) error
}

// Sensor represents resources that provide readings.
type Sensor interface {
    Resource
    Readings(ctx context.Context) (map[string]any, error)
}

// Actuator represents resources that can move.
type Actuator interface {
    Resource
    IsMoving(ctx context.Context) (bool, error)
    Stop(ctx context.Context) error
}

// Dependencies provides access to dependent resources.
type Dependencies interface {
    Get(name Name) (Resource, error)
    GetByType(subtype string) ([]Resource, error)
}

// Config holds resource configuration.
type Config struct {
    Attributes map[string]any
    Raw        []byte
}

// Unmarshal decodes config into a struct.
func (c Config) Unmarshal(v any) error {
    // JSON unmarshal from Raw
}
```

```go
// pkg/resource/name.go
package resource

import "fmt"

// Name identifies a resource with hierarchical naming.
type Name struct {
    Namespace string // e.g., "gorai", "mycompany"
    Type      string // "component" or "service"
    Subtype   string // e.g., "motor", "camera", "vision"
    Name      string // Instance name e.g., "left_motor"
}

// String returns the full resource name.
func (n Name) String() string {
    return fmt.Sprintf("%s:%s:%s/%s", n.Namespace, n.Type, n.Subtype, n.Name)
}

// Short returns just subtype/name.
func (n Name) Short() string {
    return fmt.Sprintf("%s/%s", n.Subtype, n.Name)
}

// Validate checks if name is valid.
func (n Name) Validate() error {
    if n.Namespace == "" || n.Type == "" || n.Subtype == "" || n.Name == "" {
        return fmt.Errorf("incomplete resource name: %v", n)
    }
    return nil
}
```

**Unit Tests**: `pkg/resource/resource_test.go`
```go
package resource_test

import (
    "testing"

    "github.com/gorai/gorai/pkg/resource"
)

func TestName_String(t *testing.T) {
    tests := []struct {
        name     resource.Name
        expected string
    }{
        {
            name: resource.Name{
                Namespace: "gorai",
                Type:      "component",
                Subtype:   "motor",
                Name:      "left_wheel",
            },
            expected: "gorai:component:motor/left_wheel",
        },
        {
            name: resource.Name{
                Namespace: "myco",
                Type:      "service",
                Subtype:   "vision",
                Name:      "detector",
            },
            expected: "myco:service:vision/detector",
        },
    }

    for _, tt := range tests {
        got := tt.name.String()
        if got != tt.expected {
            t.Errorf("Name.String() = %q, want %q", got, tt.expected)
        }
    }
}

func TestName_Validate(t *testing.T) {
    valid := resource.Name{
        Namespace: "gorai",
        Type:      "component",
        Subtype:   "motor",
        Name:      "test",
    }
    if err := valid.Validate(); err != nil {
        t.Errorf("valid name returned error: %v", err)
    }

    invalid := resource.Name{Type: "component"}
    if err := invalid.Validate(); err == nil {
        t.Error("invalid name should return error")
    }
}

func TestConfig_Unmarshal(t *testing.T) {
    cfg := resource.Config{
        Raw: []byte(`{"interval_ms": 1000, "zone": "cpu"}`),
    }

    var v struct {
        IntervalMS int    `json:"interval_ms"`
        Zone       string `json:"zone"`
    }

    if err := cfg.Unmarshal(&v); err != nil {
        t.Fatalf("Unmarshal failed: %v", err)
    }

    if v.IntervalMS != 1000 {
        t.Errorf("IntervalMS = %d, want 1000", v.IntervalMS)
    }
    if v.Zone != "cpu" {
        t.Errorf("Zone = %q, want 'cpu'", v.Zone)
    }
}
```

**Verification**:
```bash
go test -v ./pkg/resource/...
go test -cover ./pkg/resource/...
```

---

#### Step 2.2: Enhance pkg/node

**Files**:
- `pkg/node/node.go` (modify)
- `pkg/node/node_test.go` (new)

**Enhancements needed**:
1. Add `FullName() string` method
2. Add `JetStream() nats.JetStreamContext` method
3. Add `SpinOnce() error` method
4. Add namespace support
5. Improve `Spin()` to handle context cancellation properly

**Implementation additions**:
```go
// Add to Node struct
type Node struct {
    name      string
    namespace string
    nc        *nats.Conn
    js        nats.JetStreamContext
    logger    *slog.Logger
    ctx       context.Context
    cancel    context.CancelFunc
    // ... existing fields
}

// Add Option
func WithNamespace(ns string) Option {
    return func(n *Node) error {
        n.namespace = ns
        return nil
    }
}

// Add methods
func (n *Node) Namespace() string {
    return n.namespace
}

func (n *Node) FullName() string {
    if n.namespace != "" {
        return n.namespace + "." + n.name
    }
    return n.name
}

func (n *Node) JetStream() nats.JetStreamContext {
    return n.js
}

func (n *Node) SpinOnce() error {
    // Process one message from all subscriptions
    // For testing and single-threaded scenarios
}
```

**Unit Tests**: `pkg/node/node_test.go`
```go
package node_test

import (
    "context"
    "testing"
    "time"

    "github.com/gorai/gorai/pkg/node"
    "github.com/gorai/gorai/internal/testutil"
)

func TestNode_New(t *testing.T) {
    // Test node creation without NATS (should still work)
    n, err := node.New("test-node")
    if err != nil {
        t.Fatalf("New failed: %v", err)
    }
    defer n.Close()

    if n.Name() != "test-node" {
        t.Errorf("Name() = %q, want 'test-node'", n.Name())
    }
}

func TestNode_WithNamespace(t *testing.T) {
    n, err := node.New("sensor", node.WithNamespace("robot1"))
    if err != nil {
        t.Fatalf("New failed: %v", err)
    }
    defer n.Close()

    if n.Namespace() != "robot1" {
        t.Errorf("Namespace() = %q, want 'robot1'", n.Namespace())
    }
    if n.FullName() != "robot1.sensor" {
        t.Errorf("FullName() = %q, want 'robot1.sensor'", n.FullName())
    }
}

func TestNode_WithNATS(t *testing.T) {
    // Skip if no test NATS available
    // In integration tests, use embedded NATS
    t.Skip("requires NATS server")
}

func TestNode_Spin(t *testing.T) {
    n, err := node.New("test")
    if err != nil {
        t.Fatalf("New failed: %v", err)
    }
    defer n.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()

    err = n.Spin(ctx)
    if err != nil && err != context.DeadlineExceeded {
        t.Errorf("Spin failed: %v", err)
    }
}
```

**Verification**:
```bash
go test -v ./pkg/node/...
go test -cover ./pkg/node/...
```

---

#### Step 2.3: Add QoS to pkg/pub and pkg/sub

**Files**:
- `pkg/pub/qos.go` (new)
- `pkg/pub/publisher.go` (modify)
- `pkg/sub/subscriber.go` (modify)
- `pkg/pub/publisher_test.go` (new)
- `pkg/sub/subscriber_test.go` (new)

**Implementation**:
```go
// pkg/pub/qos.go
package pub

// QoS defines quality of service levels.
type QoS int

const (
    // BestEffort uses core NATS - no persistence
    BestEffort QoS = iota

    // Reliable uses JetStream with acknowledgment
    Reliable

    // Retained uses JetStream with last-value retention
    Retained

    // History(n) uses JetStream with n message limit
    History

    // Persistent uses JetStream with durable storage
    Persistent
)

// Option configures a publisher.
type Option func(*options)

type options struct {
    qos       QoS
    history   int
    streamName string
}

// WithQoS sets the quality of service.
func WithQoS(qos QoS) Option {
    return func(o *options) {
        o.qos = qos
    }
}

// WithHistory sets history depth (requires QoS >= Reliable).
func WithHistory(n int) Option {
    return func(o *options) {
        o.history = n
    }
}

// WithRetain enables last-value retention.
func WithRetain() Option {
    return func(o *options) {
        o.qos = Retained
    }
}
```

**Unit Tests**: `pkg/pub/publisher_test.go`
```go
package pub_test

import (
    "context"
    "testing"

    "github.com/gorai/gorai/api/gen/gorai/std"
    "github.com/gorai/gorai/pkg/pub"
)

func TestPublisher_Topic(t *testing.T) {
    // Test topic name is correctly set
}

func TestPublisher_WithQoS(t *testing.T) {
    // Test QoS options are applied
}

// Integration tests require NATS
func TestPublisher_Publish_Integration(t *testing.T) {
    t.Skip("requires NATS server")
}
```

**Verification**:
```bash
go test -v ./pkg/pub/...
go test -v ./pkg/sub/...
```

---

#### Step 2.4: Implement pkg/action Server and Client

**Files**:
- `pkg/action/server.go` (new, implement methods)
- `pkg/action/client.go` (new, implement methods)
- `pkg/action/action_test.go` (new)

**Implementation**:
```go
// pkg/action/server.go
package action

import (
    "context"
    "sync"

    "github.com/gorai/gorai/pkg/node"
    actionpb "github.com/gorai/gorai/api/gen/gorai/action"
    "google.golang.org/protobuf/proto"
)

// FeedbackSender allows sending feedback during action execution.
type FeedbackSender[Feedback proto.Message] interface {
    Send(fb Feedback) error
}

// Handler processes action goals.
type Handler[Goal, Feedback, Result proto.Message] func(
    ctx context.Context,
    goal Goal,
    feedback FeedbackSender[Feedback],
) (Result, error)

// Server handles action goals from clients.
type Server[Goal, Feedback, Result proto.Message] struct {
    node    *node.Node
    name    string
    handler Handler[Goal, Feedback, Result]

    mu      sync.RWMutex
    goals   map[string]*goalState[Feedback, Result]
}

type goalState[Feedback, Result proto.Message] struct {
    id       string
    status   actionpb.GoalStatus_Status
    cancel   context.CancelFunc
    feedback chan Feedback
    result   chan Result
    err      error
}

// NewServer creates an action server.
func NewServer[Goal, Feedback, Result proto.Message](
    n *node.Node,
    name string,
    handler Handler[Goal, Feedback, Result],
) (*Server[Goal, Feedback, Result], error) {
    s := &Server[Goal, Feedback, Result]{
        node:    n,
        name:    name,
        handler: handler,
        goals:   make(map[string]*goalState[Feedback, Result]),
    }

    // Subscribe to goal, cancel topics
    // ...

    return s, nil
}

// Close shuts down the server.
func (s *Server[G, F, R]) Close() error {
    // Cancel all active goals
    // Unsubscribe from topics
    return nil
}
```

```go
// pkg/action/client.go
package action

import (
    "context"

    "github.com/gorai/gorai/pkg/node"
    "google.golang.org/protobuf/proto"
)

// GoalHandle represents an active goal.
type GoalHandle[Feedback, Result proto.Message] interface {
    ID() string
    Status() Status
    Feedback() <-chan Feedback
    Result() (Result, error)
    Cancel() error
}

// Client sends goals to an action server.
type Client[Goal, Feedback, Result proto.Message] struct {
    node *node.Node
    name string
}

// NewClient creates an action client.
func NewClient[Goal, Feedback, Result proto.Message](
    n *node.Node,
    name string,
) (*Client[Goal, Feedback, Result], error) {
    return &Client[Goal, Feedback, Result]{
        node: n,
        name: name,
    }, nil
}

// SendGoal sends a goal and returns a handle for tracking.
func (c *Client[G, F, R]) SendGoal(ctx context.Context, goal G) (GoalHandle[F, R], error) {
    // Publish to goal topic
    // Subscribe to feedback and result topics
    // Return handle
    return nil, nil
}

// CancelAll cancels all active goals.
func (c *Client[G, F, R]) CancelAll(ctx context.Context) error {
    return nil
}
```

**Unit Tests**: `pkg/action/action_test.go`
```go
package action_test

import (
    "context"
    "testing"
    "time"
)

func TestActionServer_HandleGoal(t *testing.T) {
    t.Skip("requires NATS server")
}

func TestActionClient_SendGoal(t *testing.T) {
    t.Skip("requires NATS server")
}

func TestActionClient_Cancel(t *testing.T) {
    t.Skip("requires NATS server")
}
```

**Verification**:
```bash
go test -v ./pkg/action/...
```

---

### Phase 3: Resource Model Implementation

**Goal**: Align component and service packages with the Resource model.

#### Step 3.1: Update Component Base

**Files**:
- `components/component.go` (modify to use resource.Resource)
- `components/component_test.go` (new)

**Implementation**:
```go
// components/component.go
package component

import (
    "github.com/gorai/gorai/pkg/resource"
)

// Component is a hardware component.
// All components implement the resource.Resource interface.
type Component interface {
    resource.Resource
}

// Sensor is a component that provides readings.
type Sensor interface {
    Component
    resource.Sensor
}

// Actuator is a component that can move.
type Actuator interface {
    Component
    resource.Actuator
}
```

**Unit Tests**:
```go
package component_test

import (
    "testing"

    "github.com/gorai/gorai/components"
    "github.com/gorai/gorai/pkg/resource"
)

func TestComponent_IsResource(t *testing.T) {
    // Verify that Component embeds resource.Resource
    var _ resource.Resource = (component.Component)(nil)
}
```

---

#### Step 3.2: Update Service Base

Similar to components, align with resource.Resource.

---

#### Step 3.3: Update Fake Implementations

Update `components/motor/fake/fake.go` and `components/camera/fake/fake.go` to implement the full resource.Resource interface.

---

### Phase 4: Component Interface Refinement

**Goal**: Ensure all component interfaces match the spec exactly.

#### Step 4.1: Motor Interface

**Verify/Update**: `components/motor/motor.go`

**Required Methods** (from spec):
- SetPower(ctx, power float64) error
- SetVelocity(ctx, velocity float64) error
- GoTo(ctx, position, velocity float64) error
- GetPosition(ctx) (float64, error)
- GetVelocity(ctx) (float64, error)
- ResetZeroPosition(ctx) error
- Stop(ctx) error
- IsMoving(ctx) (bool, error)
- IsPowered(ctx) (bool, error)
- Properties(ctx) (Properties, error)

**Unit Tests**: `components/motor/motor_test.go`
```go
func TestFakeMotor_SetPower(t *testing.T) {
    ctx := context.Background()
    m := fake.New("test", nil, resource.Config{})

    if err := m.SetPower(ctx, 0.5); err != nil {
        t.Fatalf("SetPower failed: %v", err)
    }

    powered, err := m.IsPowered(ctx)
    if err != nil {
        t.Fatalf("IsPowered failed: %v", err)
    }
    if !powered {
        t.Error("expected motor to be powered")
    }
}

func TestFakeMotor_GoTo(t *testing.T) {
    ctx := context.Background()
    m := fake.New("test", nil, resource.Config{})

    if err := m.GoTo(ctx, 100.0, 10.0); err != nil {
        t.Fatalf("GoTo failed: %v", err)
    }

    pos, err := m.GetPosition(ctx)
    if err != nil {
        t.Fatalf("GetPosition failed: %v", err)
    }
    if pos != 100.0 {
        t.Errorf("position = %v, want 100.0", pos)
    }
}
```

---

#### Step 4.2: Camera Interface

Similar verification and testing for camera.

---

#### Step 4.3: Sensor Interface

Ensure sensor interfaces (IMU, GPS, Encoder, RangeFinder) match spec.

---

### Phase 5: Network Transparency

**Goal**: Implement NWS/NWC pattern for remote resource access.

#### Step 5.1: Implement nws.Wrap()

**File**: `nws/server.go`

**Implementation**:
```go
// nws/server.go
package nws

import (
    "context"
    "encoding/json"

    "github.com/gorai/gorai/pkg/node"
    "github.com/gorai/gorai/pkg/resource"
    "github.com/gorai/gorai/pkg/services"
)

// Server exposes a local resource over NATS.
type Server struct {
    node     *node.Node
    resource resource.Resource
    svc      *service.Server[*Request, *Response]
}

// Request is a remote method call.
type Request struct {
    Method string         `json:"method"`
    Args   map[string]any `json:"args"`
}

// Response is a remote method response.
type Response struct {
    Result any    `json:"result,omitempty"`
    Error  string `json:"error,omitempty"`
}

// Wrap exposes a resource over the network.
func Wrap(n *node.Node, res resource.Resource) (*Server, error) {
    s := &Server{
        node:     n,
        resource: res,
    }

    // Create service endpoint for RPC
    topic := res.Name().String() + ".rpc"

    var err error
    s.svc, err = service.NewServer(n, topic, s.handleRPC)
    if err != nil {
        return nil, err
    }

    return s, nil
}

func (s *Server) handleRPC(ctx context.Context, req *Request) (*Response, error) {
    // Use reflection to call methods on resource
    // Return serialized results
    return nil, nil
}

func (s *Server) Close() error {
    return s.svc.Close()
}
```

**Unit Tests**: `nws/server_test.go`

---

#### Step 5.2: Implement nws.Connect()

**File**: `nws/client.go`

**Implementation**:
```go
// nws/client.go
package nws

import (
    "context"

    "github.com/gorai/gorai/pkg/node"
    "github.com/gorai/gorai/pkg/resource"
    "github.com/gorai/gorai/pkg/services"
)

// Connect creates a client for a remote resource.
func Connect[T resource.Resource](n *node.Node, name resource.Name) (T, error) {
    // Create service client for RPC
    // Return proxy that implements T
    var zero T
    return zero, nil
}
```

**Unit Tests**: `nws/client_test.go`

---

### Phase 6: Hello Sensor Implementation

**Goal**: Implement the complete hello-sensor example as defined in the design doc.

#### Step 6.1: Create temperature.proto

**File**: `api/proto/gorai/sensor/temperature.proto`

**Implementation**: As defined in hello-sensor-design.md

**Verification**:
```bash
buf lint api/proto
buf generate
go build ./api/gen/gorai/sensor/...
```

---

#### Step 6.2: Create Reader Package

**Files**:
- `examples/hello-sensor/reader/reader.go`
- `examples/hello-sensor/reader/linux.go`
- `examples/hello-sensor/reader/darwin.go`
- `examples/hello-sensor/reader/unsupported.go`

**Implementation**: As defined in hello-sensor-design.md

**Unit Tests**: `examples/hello-sensor/reader/reader_test.go`
```go
package reader_test

import (
    "context"
    "runtime"
    "testing"

    "github.com/gorai/gorai/examples/hello-sensor/reader"
)

func TestNew(t *testing.T) {
    r, err := reader.New()
    if err != nil {
        if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
            t.Skip("unsupported platform")
        }
        t.Fatalf("New failed: %v", err)
    }
    defer r.Close()

    if r.Platform() != runtime.GOOS {
        t.Errorf("Platform() = %q, want %q", r.Platform(), runtime.GOOS)
    }
}

func TestLinuxReader_Read(t *testing.T) {
    if runtime.GOOS != "linux" {
        t.Skip("linux only")
    }

    r, err := reader.New()
    if err != nil {
        t.Fatalf("New failed: %v", err)
    }
    defer r.Close()

    reading, err := r.Read(context.Background(), "")
    if err != nil {
        t.Fatalf("Read failed: %v", err)
    }

    // Temperature should be reasonable (0-120°C)
    if reading.TemperatureC < 0 || reading.TemperatureC > 120 {
        t.Errorf("temperature %v out of reasonable range", reading.TemperatureC)
    }
}
```

**Verification**:
```bash
go test -v ./examples/hello-sensor/reader/...
GOOS=linux go build ./examples/hello-sensor/reader/...
GOOS=darwin go build ./examples/hello-sensor/reader/...
GOOS=windows go build ./examples/hello-sensor/reader/...
```

---

#### Step 6.3: Create Fake Reader

**File**: `examples/hello-sensor/sensor/fake/fake.go`

**Implementation**: As defined in hello-sensor-design.md

**Unit Tests**: `examples/hello-sensor/sensor/fake/fake_test.go`
```go
package fake_test

import (
    "context"
    "errors"
    "testing"

    "github.com/gorai/gorai/examples/hello-sensor/sensor/fake"
)

func TestFakeReader_Default(t *testing.T) {
    r := fake.New()

    if r.Platform() != "fake" {
        t.Errorf("Platform() = %q, want 'fake'", r.Platform())
    }

    reading, err := r.Read(context.Background(), "")
    if err != nil {
        t.Fatalf("Read failed: %v", err)
    }

    if reading.TemperatureC != 42.0 {
        t.Errorf("default temperature = %v, want 42.0", reading.TemperatureC)
    }
}

func TestFakeReader_SetTemperature(t *testing.T) {
    r := fake.New()
    r.SetTemperature(55.5)

    reading, _ := r.Read(context.Background(), "")
    if reading.TemperatureC != 55.5 {
        t.Errorf("temperature = %v, want 55.5", reading.TemperatureC)
    }
}

func TestFakeReader_SetError(t *testing.T) {
    r := fake.New()
    testErr := errors.New("sensor failure")
    r.SetError(testErr)

    _, err := r.Read(context.Background(), "")
    if err != testErr {
        t.Errorf("error = %v, want %v", err, testErr)
    }
}
```

---

#### Step 6.4: Create Temperature Sensor Component

**File**: `examples/hello-sensor/sensor/temperature.go`

**Implementation**: As defined in hello-sensor-design.md

**Unit Tests**: `examples/hello-sensor/sensor/temperature_test.go`
```go
package sensor_test

import (
    "testing"

    "github.com/gorai/gorai/examples/hello-sensor/sensor"
)

func TestCelsiusToFahrenheit(t *testing.T) {
    tests := []struct {
        celsius    float64
        fahrenheit float64
    }{
        {0, 32},
        {100, 212},
        {-40, -40},
        {37, 98.6},
    }

    for _, tt := range tests {
        got := sensor.CelsiusToFahrenheit(tt.celsius)
        // Allow small floating point difference
        if got < tt.fahrenheit-0.01 || got > tt.fahrenheit+0.01 {
            t.Errorf("CelsiusToFahrenheit(%v) = %v, want %v",
                tt.celsius, got, tt.fahrenheit)
        }
    }
}

func TestDefaultConfig(t *testing.T) {
    cfg := sensor.DefaultConfig()

    if cfg.Name != "cpu_temp" {
        t.Errorf("Name = %q, want 'cpu_temp'", cfg.Name)
    }
    if cfg.IntervalMS != 1000 {
        t.Errorf("IntervalMS = %d, want 1000", cfg.IntervalMS)
    }
}
```

---

#### Step 6.5: Component Tests with Fake Reader

**File**: `examples/hello-sensor/sensor/temperature_component_test.go`

**Build tag**: `//go:build component`

**Implementation**: As defined in hello-sensor-design.md

**Verification**:
```bash
go test -v -tags=component ./examples/hello-sensor/sensor/...
```

---

#### Step 6.6: Create Main Entry Point

**File**: `examples/hello-sensor/main.go`

**Implementation**: As defined in hello-sensor-design.md

**Verification**:
```bash
go build -o hello-sensor ./examples/hello-sensor
./hello-sensor --help
```

---

#### Step 6.7: Create Configuration Files

**Files**:
- `examples/hello-sensor/config/default.json`
- `examples/hello-sensor/config/raspberry_pi.json`

**Implementation**: As defined in hello-sensor-design.md

**Verification**:
```bash
python3 -m json.tool examples/hello-sensor/config/default.json > /dev/null
python3 -m json.tool examples/hello-sensor/config/raspberry_pi.json > /dev/null
```

---

#### Step 6.8: Create README

**File**: `examples/hello-sensor/README.md`

Document build, run, and usage instructions.

---

## Testing Strategy

### Unit Testing

Every package should have `*_test.go` files with:
- Table-driven tests for functions with multiple cases
- Error path testing
- Edge case coverage
- No external dependencies (mock/fake everything)

**Run**:
```bash
go test ./...
```

### Component Testing

Tests that require infrastructure (NATS) but use fakes for hardware:
- Use build tag `//go:build component`
- Start embedded NATS for testing
- Use fake readers/components

**Run**:
```bash
go test -tags=component ./...
```

### Integration Testing

Tests that run on actual hardware or require external services:
- Use build tag `//go:build integration`
- Test on real Linux systems for temperature sensor
- Test with real NATS server

**Run** (manual, on target hardware):
```bash
go test -tags=integration ./...
```

### Test Coverage Goals

| Package | Target Coverage |
|---------|-----------------|
| `pkg/resource` | 90% |
| `pkg/node` | 80% |
| `pkg/pub` | 80% |
| `pkg/sub` | 80% |
| `pkg/services` | 80% |
| `pkg/action` | 70% |
| `components/*` | 70% |
| `examples/hello-sensor` | 80% |

**Measure**:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## Verification Checklist

### Phase 1: Protocol Buffers
- [ ] `buf lint api/proto` passes
- [ ] `buf generate` succeeds
- [ ] All proto tests pass
- [ ] Generated files compile

### Phase 2: Core Packages
- [ ] `pkg/resource` tests pass
- [ ] `pkg/node` tests pass (>80% coverage)
- [ ] `pkg/pub` tests pass with QoS
- [ ] `pkg/sub` tests pass
- [ ] `pkg/action` tests pass

### Phase 3: Resource Model
- [ ] Components implement resource.Resource
- [ ] Services implement resource.Resource
- [ ] Fake implementations updated
- [ ] All component tests pass

### Phase 4: Component Interfaces
- [ ] Motor interface matches spec
- [ ] Camera interface matches spec
- [ ] Sensor interfaces match spec
- [ ] All interface tests pass

### Phase 5: Network Transparency
- [ ] `nws.Wrap()` implemented
- [ ] `nws.Connect()` implemented
- [ ] Remote access tested

### Phase 6: Hello Sensor
- [ ] temperature.proto compiles
- [ ] Reader package builds on linux/darwin/windows
- [ ] Fake reader tests pass
- [ ] Temperature sensor unit tests pass
- [ ] Component tests pass (with embedded NATS)
- [ ] Binary builds successfully
- [ ] CLI flags work as documented
- [ ] Publishes to correct NATS topics
- [ ] Graceful shutdown works

### Final Integration
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `go test -tags=component ./...` passes
- [ ] Hello sensor publishes data when run with NATS

---

## Summary

| Phase | Estimated Effort | Dependencies |
|-------|------------------|--------------|
| Phase 1: Protos | Small | buf tool |
| Phase 2: Core Packages | Medium | Phase 1 |
| Phase 3: Resource Model | Small | Phase 2 |
| Phase 4: Components | Small | Phase 3 |
| Phase 5: NWS/NWC | Medium | Phase 2 |
| Phase 6: Hello Sensor | Medium | Phase 1-4 |

**Critical Path**: Phase 1 → Phase 2 → Phase 6

The hello-sensor example serves as the validation that the core framework is functional and follows all specifications correctly.
