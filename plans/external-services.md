# External Services Implementation Plan

**Status:** Design Complete, Implementation Pending
**Last Updated:** 2024-12-15
**Author:** Claude Code

## Overview

This document describes the implementation plan for the External Services feature in Gorai. External services allow compute-intensive workloads (ML inference, SLAM, etc.) to run as separate processes or containers while communicating with the main robot via NATS.

The key innovation is the **Service RDL** pattern: external services can have their own definition files that specify their interface (topics, attributes) independently of any specific robot. This enables modular, reusable, and shareable service packages.

## Design Summary

### Service RDL Architecture

```
Robot Deployment
+-------------------------------------------------------------+
|  Robot RDL (robot.json)                                     |
|  +-- references Service RDL files                           |
|  +-- provides deployment-specific configuration             |
|                                                             |
|  Service RDL (services/detector/detector.rdl.json)          |
|  +-- defines service type and model                         |
|  +-- specifies topic patterns with variables                |
|  +-- declares configurable attributes with types            |
|  +-- provides default runtime configuration                 |
+-------------------------------------------------------------+
```

### Topic Pattern Resolution

Service RDL uses pattern variables that are resolved at runtime:

| Variable | Source | Example |
|----------|--------|---------|
| `{namespace}` | Robot namespace | `hello-camera` |
| `{robot}` | Robot name | `hello-camera` |
| `{service}` | Service name from robot RDL | `person_detector` |
| `{custom_attr}` | Custom attribute from service | `main_camera` |

**Resolution Example:**
```
Pattern:  gorai.{namespace}.{input_component}.data
Context:  namespace=hello-camera, input_component=main_camera
Result:   gorai.hello-camera.main_camera.data
```

## Implementation Steps

### Phase 1: Service RDL Parser

**Goal:** Parse and validate Service RDL files.

**Files to modify/create:**
- `pkg/config/service_rdl.go` - Service RDL data structures
- `pkg/config/service_rdl_parser.go` - Parser implementation
- `pkg/config/service_rdl_test.go` - Unit tests

**Tasks:**

1. **Define Service RDL structs** (`service_rdl.go`)
   ```go
   type ServiceRDL struct {
       Schema   string         `json:"$schema,omitempty"`
       Version  string         `json:"version"`
       Kind     string         `json:"kind"`
       Service  ServiceMeta    `json:"service"`
       Topics   TopicsConfig   `json:"topics"`
       Attrs    AttrsConfig    `json:"attributes,omitempty"`
       Runtime  RuntimeConfig  `json:"runtime,omitempty"`
   }

   type ServiceMeta struct {
       Type        string `json:"type"`
       Model       string `json:"model"`
       Description string `json:"description,omitempty"`
   }

   type TopicsConfig struct {
       Subscribe []TopicEntry `json:"subscribe,omitempty"`
       Publish   []TopicEntry `json:"publish,omitempty"`
   }

   type TopicEntry struct {
       Name        string `json:"name"`
       Pattern     string `json:"pattern"`
       Description string `json:"description,omitempty"`
       Format      string `json:"format,omitempty"`
   }

   type AttrDef struct {
       Type        string      `json:"type"`
       Default     interface{} `json:"default,omitempty"`
       Required    bool        `json:"required,omitempty"`
       Description string      `json:"description,omitempty"`
       Min         *float64    `json:"min,omitempty"`
       Max         *float64    `json:"max,omitempty"`
       Enum        []string    `json:"enum,omitempty"`
   }
   ```

2. **Implement parser** (`service_rdl_parser.go`)
   - Load JSON from file path
   - Validate required fields (version, kind, service, topics)
   - Validate attribute definitions
   - Return parsed ServiceRDL struct

3. **Add validation**
   - `kind` must be "service"
   - `version` must be "1"
   - At least one topic must be defined
   - Attribute types must be valid (string, int, float, bool, array, object)
   - Required attributes must not have defaults

### Phase 2: Topic Pattern Resolution

**Goal:** Resolve topic patterns with variable substitution.

**Files to modify/create:**
- `pkg/config/topic_resolver.go` - Topic pattern resolver
- `pkg/config/topic_resolver_test.go` - Unit tests

**Tasks:**

1. **Create resolver struct**
   ```go
   type TopicResolver struct {
       context map[string]string
   }

   func NewTopicResolver(robotConfig *RobotConfig, serviceName string, attrs map[string]interface{}) *TopicResolver
   func (r *TopicResolver) Resolve(pattern string) (string, error)
   func (r *TopicResolver) ResolveAll(topics *TopicsConfig) (*ResolvedTopics, error)
   ```

2. **Variable extraction**
   - Parse `{variable}` patterns from string
   - Look up variables in context map
   - Error on undefined required variables

3. **Context building**
   - Add robot-level variables (namespace, robot, name)
   - Add service-level variables (service)
   - Add attribute values as variables
   - Support nested attribute access (future)

### Phase 3: Service RDL Merging

**Goal:** Merge Service RDL with robot service definition.

**Files to modify/create:**
- `pkg/config/config.go` - Update Service struct
- `pkg/config/service_merger.go` - Merger implementation

**Tasks:**

1. **Update Service struct** (`config.go`)
   ```go
   type Service struct {
       Name       string                 `json:"name"`
       RDL        string                 `json:"rdl,omitempty"`  // NEW
       Type       string                 `json:"type,omitempty"`
       Model      string                 `json:"model,omitempty"`
       Disabled   bool                   `json:"disabled,omitempty"`
       External   *ExternalConfig        `json:"external,omitempty"`
       Attributes map[string]interface{} `json:"attributes,omitempty"`
       DependsOn  []string               `json:"depends_on,omitempty"`
   }
   ```

2. **Implement merger**
   - If `rdl` is set, load Service RDL file
   - Merge type/model from Service RDL (robot RDL overrides)
   - Validate provided attributes against Service RDL definitions
   - Apply attribute defaults from Service RDL
   - Merge runtime config (robot RDL overrides)

3. **Attribute validation**
   - Check required attributes are provided
   - Validate types match definitions
   - Validate numeric ranges
   - Validate enum values

### Phase 4: External Service Manager

**Goal:** Manage external service lifecycle (spawn, monitor, restart).

**Files to modify/create:**
- `pkg/runtime/external_manager.go` - External service manager
- `pkg/runtime/container_runner.go` - Container execution
- `pkg/runtime/process_runner.go` - Native process execution

**Tasks:**

1. **External Manager interface**
   ```go
   type ExternalManager interface {
       Start(ctx context.Context, service *config.Service, resolved *ResolvedTopics) error
       Stop(ctx context.Context, serviceName string) error
       Status(serviceName string) (ServiceStatus, error)
       Logs(serviceName string, follow bool) (io.ReadCloser, error)
   }
   ```

2. **Container runner**
   - Use podman/docker for container management
   - Pass resolved topics via environment variables
   - Handle device mapping, volumes, network
   - Implement restart policies

3. **Process runner**
   - Fork native process with `--config` and `--service` args
   - Set environment variables for topics
   - Implement process monitoring and restart

4. **Environment variables**
   ```
   GORAI_ROBOT_NAME=hello-people-detector
   GORAI_SERVICE_NAME=person_detector
   GORAI_NAMESPACE=hello-people-detector
   NATS_URL=nats://localhost:4222
   GORAI_INPUT_TOPICS={"input":"gorai.hello-people-detector.main_camera.data"}
   GORAI_OUTPUT_TOPICS={"annotated":"gorai.hello-people-detector.person_detector.annotated","detections":"gorai.hello-people-detector.person_detector.detections"}
   ```

### Phase 5: Build Command Integration

**Goal:** Build external service containers during `gorai build`.

**Files to modify:**
- `cmd/gorai/commands/build.go` - Build command

**Tasks:**

1. **Service discovery**
   - Scan services with `external.container.build` config
   - Also check for Service RDL runtime defaults

2. **Build orchestration**
   - Resolve build context path relative to robot config
   - Execute podman/docker build
   - Tag with configured image name
   - Report build status

3. **Service RDL context**
   - If Service RDL specifies build context, use that as base
   - Robot RDL build config overrides Service RDL

### Phase 6: Validate Command Integration

**Goal:** Validate Service RDL files and service configurations.

**Files to modify:**
- `cmd/gorai/commands/validate.go` - Validate command

**Tasks:**

1. **Service RDL validation**
   - Check Service RDL file exists if `rdl` is specified
   - Parse and validate Service RDL schema
   - Check required attributes are provided in robot RDL
   - Validate attribute types and constraints

2. **Error messages**
   - Show file path and line number when possible
   - Suggest fixes for common errors
   - List valid options for enum/type errors

### Phase 7: Run/Start Command Integration

**Goal:** Start external services when robot starts.

**Files to modify:**
- `cmd/gorai/commands/run.go` - Run command
- `cmd/gorai/commands/start.go` - Start command

**Tasks:**

1. **Service startup ordering**
   - Start external services after NATS is available
   - Respect depends_on for service ordering
   - Wait for external service health before proceeding

2. **Managed vs unmanaged**
   - Managed: spawn and monitor external service
   - Unmanaged: verify connectivity only

3. **Health checking**
   - For containers: use container health check
   - For processes: check PID and optionally NATS heartbeat
   - Timeout and retry logic

### Phase 8: Status/Logs Command Integration

**Goal:** Show external service status and logs.

**Files to modify:**
- `cmd/gorai/commands/status.go` - Status command
- `cmd/gorai/commands/logs.go` - Logs command

**Tasks:**

1. **Status display**
   - Show external services separately
   - Include container/process state
   - Show resource usage if available

2. **Log aggregation**
   - Support `--container` flag for specific service
   - Support `--follow` for live logs
   - Aggregate all service logs by default

## Testing Strategy

### Unit Tests

- Service RDL parser: valid/invalid schemas
- Topic resolver: various pattern combinations
- Service merger: override logic, validation
- Attribute validation: types, ranges, enums

### Integration Tests

- Full robot config load with Service RDL
- Build command with Service RDL services
- Start/stop lifecycle with external services

### End-to-End Tests

- hello-people-detector example runs correctly
- External service receives messages via NATS
- Annotated images are published correctly

## Migration Notes

### Backwards Compatibility

- Existing inline external services (without `rdl`) continue to work
- Service RDL is purely additive
- No changes to existing robot RDL format

### Deprecations

- None

## File Summary

### New Files

| File | Purpose |
|------|---------|
| `pkg/config/service_rdl.go` | Service RDL data structures |
| `pkg/config/service_rdl_parser.go` | Service RDL parser |
| `pkg/config/topic_resolver.go` | Topic pattern resolver |
| `pkg/config/service_merger.go` | Service configuration merger |
| `pkg/runtime/external_manager.go` | External service lifecycle |
| `pkg/runtime/container_runner.go` | Container execution |
| `pkg/runtime/process_runner.go` | Process execution |

### Modified Files

| File | Changes |
|------|---------|
| `pkg/config/config.go` | Add `rdl` field to Service struct |
| `cmd/gorai/commands/build.go` | Build Service RDL containers |
| `cmd/gorai/commands/validate.go` | Validate Service RDL files |
| `cmd/gorai/commands/run.go` | Start external services |
| `cmd/gorai/commands/start.go` | Deploy external services |
| `cmd/gorai/commands/status.go` | Show external service status |
| `cmd/gorai/commands/logs.go` | Show external service logs |

## Example Artifacts

The following example files have been created to demonstrate the pattern:

### hello-people-detector Example

```
examples/hello-people-detector/
+-- hello-people-detector.json          # Robot RDL with Service RDL reference
+-- README.md                            # Documentation
+-- services/
    +-- person-detector/
        +-- person-detector.rdl.json     # Service RDL
        +-- main.py                       # Python service
        +-- Containerfile                 # Container build
        +-- requirements.txt              # Dependencies
        +-- config/                       # Configuration module
        +-- inference/                    # Hailo/ONNX backend
        +-- processing/                   # Post-processing
        +-- annotate/                     # Bounding box drawing
```

## Open Questions

1. **Service RDL versioning**: How to handle Service RDL schema updates?
   - Proposal: Follow semver, major versions may break compatibility

2. **Remote Service RDL**: Support fetching Service RDL from URL?
   - Proposal: Future feature, start with local files only

3. **Service RDL registry**: Should there be a central registry for sharing services?
   - Proposal: Out of scope for initial implementation

4. **Inter-service dependencies**: How do external services depend on each other?
   - Proposal: Use `depends_on` in robot RDL, resolved at startup
