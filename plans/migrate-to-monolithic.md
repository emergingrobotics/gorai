# Migration Plan: Containerized Multi-Process to Monolithic Architecture

**Date**: 2025-12-14
**Status**: ❌ **SUPERSEDED** - This plan is obsolete
**Author**: AI-assisted

> **SUPERSEDED (2024-12-24)**: This plan has been superseded by the distributed systems tiered approach.
>
> **New Strategy**: Gorai now uses a tiered deployment model:
> - **Tier 1**: Native binaries or few containers managed by systemd (simple robots)
> - **Tier 2**: Podman pods for multi-language/multi-container systems (research platforms)
> - **Tier 3**: K3s clusters for fleet management (10+ robots)
>
> See [STRATEGIC-SUMMARY.md](../docs/STRATEGIC-SUMMARY.md) and [deployment.md](../specs/deployment.md) for current approach.

---

## Original Executive Summary (Obsolete)

This plan described the migration from the current containerized multi-process architecture back to a simpler monolithic single-binary approach. The primary motivations were:

1. **Complexity**: Rootless Podman + systemd + device access has proven more complex than the value it provides
2. **Device Access**: Hardware device permissions (cameras, GPIO, etc.) are problematic in containerized environments
3. **Debugging**: Container abstraction layers make debugging harder
4. **Overhead**: Container orchestration adds operational complexity without proportional benefit for single-host robots

### What Changes

| Aspect | Current (Containerized) | New (Monolithic) |
|--------|------------------------|------------------|
| Deployment | Multiple containers per robot | Single binary + NATS |
| Components | Run in assigned containers | Run in main process |
| Services | Run in assigned containers | Run in main process (or external) |
| CLI | `gorai start/stop` manages containers | `gorai run` starts binary directly |
| RDL | Has `containers` section | No `containers` section |
| Device access | Via container device mapping | Direct hardware access |
| Process isolation | Container-level | None (single process) |

### What Stays

1. **NATS as message bus** - Still the communication backbone
2. **Build tooling container** - For cross-compilation when Go isn't installed
3. **External process support** - Some services (e.g., Hailo ML) can optionally run as separate processes
4. **RDL for configuration** - Simplified but still declarative
5. **Topic-based pub/sub** - Same NATS topic hierarchy

---

## Phase 1: RDL Specification Changes

### 1.1 Remove `containers` Section

**File**: `/gorai/specs/robot-definition-language.md`

Remove the entire containers-related specification. The `containers` field and all container mapping will be deprecated.

**Before (current)**:
```json
{
  "version": "1",
  "robot": { "name": "hello-camera" },
  "nats": { "url": "nats://nats:4222", "container": "nats" },
  "containers": {
    "nats": { "image": "nats:2.10-alpine", ... },
    "gorai-core": { "build": {...}, "components": ["camera"], ... }
  },
  "components": [
    { "name": "camera", "type": "camera", "container": "gorai-core", ... }
  ]
}
```

**After (new)**:
```json
{
  "version": "2",
  "robot": { "name": "hello-camera" },
  "nats": { "url": "nats://localhost:4222" },
  "components": [
    { "name": "camera", "type": "camera", "model": "v4l2", ... }
  ],
  "services": [
    { "name": "dashboard", "type": "dashboard", ... }
  ]
}
```

### 1.2 Add `external` Flag for Services

For services that should run as separate processes (e.g., ML inference on Hailo), add an `external` section:

```json
{
  "services": [
    {
      "name": "person_detector",
      "type": "object_detection",
      "model": "hailo_yolox",
      "external": {
        "enabled": true,
        "command": "/opt/gorai/services/hailo-detector",
        "managed": true
      },
      "attributes": {
        "input_topic": "gorai.hello-camera.camera.data",
        "output_topic": "gorai.hello-camera.person_detector.detections"
      }
    }
  ]
}
```

| Field | Description |
|-------|-------------|
| `external.enabled` | If true, service runs as separate process |
| `external.command` | Path to executable (optional, for managed services) |
| `external.managed` | If true, main robot process spawns/manages this service |

Services without `external` or with `external.enabled: false` run in the main robot process.

### 1.3 Update Config Structs

**File**: `/gorai/pkg/config/config.go`

```go
// Remove or deprecate:
// - ContainerConfig
// - BuildConfig
// - DependsOnCondition
// - HealthcheckConfig
// - ResourceConfig
// - ResourceLimits
// - NetworkConfig
// - VolumeConfig

// Modify RDL struct:
type RDL struct {
    Schema     string            `json:"$schema,omitempty"`
    Version    string            `json:"version"`
    Robot      RobotConfig       `json:"robot"`
    NATS       *NATSConfig       `json:"nats,omitempty"`
    Components []ComponentConfig `json:"components,omitempty"`
    Services   []ServiceConfig   `json:"services,omitempty"`
    Remotes    []RemoteConfig    `json:"remotes,omitempty"`
    Log        *LogConfig        `json:"log,omitempty"`
    Dashboard  *DashboardConfig  `json:"dashboard,omitempty"`

    // Deprecated - kept for migration warnings
    Containers map[string]*ContainerConfig `json:"containers,omitempty"`
}

// Remove from NATSConfig:
// - Container string (container name for NATS service)

// Remove from ComponentConfig:
// - Container string (container this component runs in)

// Modify ServiceConfig:
type ServiceConfig struct {
    Name       string           `json:"name"`
    Type       string           `json:"type"`
    Model      string           `json:"model"`
    Disabled   bool             `json:"disabled,omitempty"`
    External   *ExternalConfig  `json:"external,omitempty"`  // NEW
    Attributes map[string]any   `json:"attributes,omitempty"`
    DependsOn  []string         `json:"depends_on,omitempty"`
}

// NEW
type ExternalConfig struct {
    Enabled bool   `json:"enabled,omitempty"`
    Command string `json:"command,omitempty"`
    Managed bool   `json:"managed,omitempty"`
}
```

---

## Phase 2: CLI Command Changes

### 2.1 Deprecate Container Commands

**Files to modify**:
- `/gorai/cmd/gorai/commands/start.go` - Deprecate or repurpose
- `/gorai/cmd/gorai/commands/stop.go` - Deprecate or repurpose
- `/gorai/cmd/gorai/commands/build.go` - Keep for image building only
- `/gorai/cmd/gorai/commands/status.go` - Show robot status, not containers
- `/gorai/cmd/gorai/commands/logs.go` - Show robot logs directly

### 2.2 New/Modified Commands

| Command | Purpose |
|---------|---------|
| `gorai run` | Run robot directly (foreground) |
| `gorai start` | Start robot as systemd service (native, not container) |
| `gorai stop` | Stop robot systemd service |
| `gorai status` | Show robot process status |
| `gorai logs` | Show robot logs (journalctl or file) |
| `gorai build` | Build robot binary (keep for cross-compilation) |
| `gorai deploy` | Deploy binary to robot via SSH/rsync |

### 2.3 `gorai run` Implementation

```go
// cmd/gorai/commands/run.go
func cmdRun() error {
    // 1. Load configuration
    cfg, err := config.Load(configPath)

    // 2. Warn if containers section exists
    if cfg.Containers != nil && len(cfg.Containers) > 0 {
        log.Warn("'containers' section is deprecated and will be ignored")
    }

    // 3. Connect to NATS (external NATS server required)
    // 4. Create robot instance
    robot, err := robot.New(ctx, cfg)

    // 5. Start all components and services
    robot.Start(ctx)

    // 6. Start external services if managed
    for _, svc := range cfg.Services {
        if svc.External != nil && svc.External.Enabled && svc.External.Managed {
            startExternalService(ctx, svc)
        }
    }

    // 7. Run until signal
    robot.Run(ctx)
}
```

---

## Phase 3: Remove Containerization Code

### 3.1 Files to Remove or Deprecate

| File | Action | Reason |
|------|--------|--------|
| `/gorai/pkg/systemd/systemd.go` | Deprecate | Generates container service files |
| `/gorai/pkg/systemd/runner.go` | Modify | Keep for native systemd, remove container logic |
| `/gorai/pkg/quadlet/*` | Remove | Quadlet container generation |
| `/gorai/Containerfile.gorai` | Keep | Build tooling container only |

### 3.2 Files to Modify

| File | Changes |
|------|---------|
| `/gorai/pkg/config/config.go` | Remove container-related structs |
| `/gorai/pkg/robot/robot.go` | Ensure works without containers |
| `/gorai/cmd/gorai/commands/start.go` | Native systemd service, not containers |
| `/gorai/cmd/gorai/commands/status.go` | Process status, not container status |

### 3.3 New systemd Service Generation

Generate simple native systemd service files (no containers):

```ini
# /etc/systemd/system/myrobot.service
[Unit]
Description=MyRobot Gorai Robot
After=network-online.target nats.service
Wants=network-online.target nats.service

[Service]
Type=simple
User=pi
Group=pi
WorkingDirectory=/opt/myrobot
ExecStart=/opt/myrobot/myrobot run --config /opt/myrobot/robot.json
Restart=always
RestartSec=5

# Hardware access
SupplementaryGroups=gpio i2c spi video dialout plugdev input

[Install]
WantedBy=multi-user.target
```

---

## Phase 4: Robot Runtime Changes

### 4.1 Component Loading

**File**: `/gorai/pkg/robot/robot.go`

Components continue to be loaded from the registry and initialized in the main process. No changes needed to component loading logic.

### 4.2 Service Loading

Services can be:
1. **Internal** (default) - Loaded and run in main process
2. **External managed** - Spawned as child process by robot
3. **External unmanaged** - Expected to be running independently

```go
func (r *Robot) startServices(ctx context.Context) error {
    for _, svc := range r.cfg.Services {
        if svc.Disabled {
            continue
        }

        if svc.External != nil && svc.External.Enabled {
            if svc.External.Managed {
                // Spawn as child process
                r.spawnExternalService(ctx, svc)
            } else {
                // Just verify it's reachable via NATS
                r.verifyExternalService(ctx, svc)
            }
        } else {
            // Load from registry and start in-process
            r.startInternalService(ctx, svc)
        }
    }
    return nil
}
```

### 4.3 External Service Management

For managed external services:

```go
type ExternalService struct {
    name    string
    cmd     *exec.Cmd
    cancel  context.CancelFunc
}

func (r *Robot) spawnExternalService(ctx context.Context, svc config.ServiceConfig) error {
    svcCtx, cancel := context.WithCancel(ctx)

    cmd := exec.CommandContext(svcCtx, svc.External.Command,
        "--config", r.configPath,
        "--service", svc.Name,
    )
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Start(); err != nil {
        cancel()
        return err
    }

    r.externalServices[svc.Name] = &ExternalService{
        name:   svc.Name,
        cmd:    cmd,
        cancel: cancel,
    }

    return nil
}
```

---

## Phase 5: Documentation Updates

### 5.1 Specs to Update

| File | Changes |
|------|---------|
| `/gorai/specs/robot-definition-language.md` | Remove containers, add external services |
| `/gorai/specs/systemd-container-orchestration.md` | Archive/deprecate |
| `/gorai/specs/gorai-container.md` | Update - build only, not runtime |
| `/gorai/specs/deployment.md` | Update for binary deployment |
| `/gorai/specs/runtime.md` | Update architecture description |

### 5.2 New Documentation

Create:
- `/gorai/specs/external-services.md` - How external services work
- `/gorai/specs/nats-setup.md` - Running NATS natively (not in container)

### 5.3 README Updates

Update `/gorai/README.md` and example READMEs to reflect new architecture.

### 5.4 Example Updates

**File**: `/gorai/examples/hello-camera/hello-camera.json`

Update to new format without containers:

```json
{
  "$schema": "https://gorai.dev/schemas/rdl-v2.json",
  "version": "2",

  "robot": {
    "name": "hello-camera",
    "namespace": "hello-camera",
    "description": "A camera robot with person detection using Hailo NPU"
  },

  "nats": {
    "url": "nats://localhost:4222"
  },

  "components": [
    {
      "name": "main_camera",
      "type": "camera",
      "model": "v4l2",
      "attributes": {
        "device": "/dev/video0",
        "width": 640,
        "height": 480,
        "frame_rate": 30
      }
    }
  ],

  "services": [
    {
      "name": "dashboard",
      "type": "dashboard",
      "model": "web",
      "attributes": {
        "listen": ":8080"
      }
    },
    {
      "name": "person_detector",
      "type": "object_detection",
      "model": "hailo_yolox",
      "external": {
        "enabled": true,
        "command": "/opt/gorai/services/hailo-detector",
        "managed": true
      },
      "attributes": {
        "model_path": "/opt/models/yolox_s_leaky.hef",
        "input_topic": "gorai.hello-camera.main_camera.data",
        "output_topic": "gorai.hello-camera.person_detector.detections"
      }
    }
  ],

  "log": {
    "level": "info",
    "format": "json"
  }
}
```

---

## Phase 6: Migration Path

### 6.1 Version Bump

- RDL version changes from `"1"` to `"2"`
- Old configs with `containers` section show deprecation warning
- Migration tool to convert v1 to v2 configs

### 6.2 Backward Compatibility

For transition period:
1. If `containers` section present, warn and ignore
2. If components/service has `container` field, warn and ignore
3. Provide `gorai migrate --config old.json --output new.json` command

### 6.3 Migration Script

```go
// cmd/gorai/commands/migrate.go
func cmdMigrate() error {
    oldCfg, err := config.Load(inputPath)

    // Convert v1 to v2
    newCfg := &config.RDL{
        Version: "2",
        Robot:   oldCfg.Robot,
        NATS: &config.NATSConfig{
            URL: "nats://localhost:4222",  // Update from container URL
        },
        Components: oldCfg.Components,
        Services:   oldCfg.Services,
        Log:        oldCfg.Log,
        Dashboard:  oldCfg.Dashboard,
    }

    // Remove container references
    for i := range newCfg.Components {
        newCfg.Components[i].Container = ""
    }
    for i := range newCfg.Services {
        newCfg.Services[i].Container = ""
    }

    // Write new config
    return config.Write(outputPath, newCfg)
}
```

---

## Phase 7: NATS Deployment

### 7.1 Native NATS Installation

NATS should run natively on the robot, not in a container:

```bash
# Install NATS server
curl -sf https://get.nats.io | sh

# Or via package manager
sudo apt install nats-server

# Create systemd service
sudo systemctl enable nats-server
sudo systemctl start nats-server
```

### 7.2 NATS Service File

```ini
# /etc/systemd/system/nats.service
[Unit]
Description=NATS Message Broker
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/nats-server -js -sd /var/lib/nats
Restart=always
User=nats
Group=nats

[Install]
WantedBy=multi-user.target
```

---

## Implementation Order

### Step 1: RDL and Config (1-2 days)
1. Update `/gorai/specs/robot-definition-language.md`
2. Update `/gorai/pkg/config/config.go`
3. Add migration warnings for deprecated fields

### Step 2: CLI Commands (2-3 days)
1. Add `gorai run` command
2. Modify `gorai start` for native systemd
3. Update `gorai status` and `gorai logs`
4. Add `gorai migrate` command

### Step 3: Runtime Changes (2-3 days)
1. Update `/gorai/pkg/robot/robot.go`
2. Add external service management
3. Ensure all components work without containers

### Step 4: Remove Container Code (1 day)
1. Remove/deprecate `/gorai/pkg/systemd/systemd.go` container logic
2. Remove `/gorai/pkg/quadlet/`
3. Update service file generation for native binary

### Step 5: Documentation (1-2 days)
1. Update all affected spec files
2. Update README files
3. Update examples

### Step 6: Testing (2-3 days)
1. Test on Raspberry Pi with hello-camera example
2. Test external service (Hailo detector)
3. Test migration path from v1 to v2 config

---

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Breaking existing deployments | Version bump + migration tool + deprecation warnings |
| ML services need isolation | External service support with separate processes |
| Loss of container benefits | Document when containers are still useful |
| NATS dependency | Clear docs on native NATS installation |

---

## Success Criteria

1. `gorai run --config hello-camera.json` works on Raspberry Pi
2. Camera streams frames via NATS
3. Dashboard accessible at localhost:8080
4. External Hailo detector works as managed service
5. All specs updated
6. All examples updated
7. Migration path tested

---

## Open Questions

1. **Should we keep `gorai build` for container images?**
   Recommendation: Yes, for edge cases where containers are still useful (CI, cross-platform)

2. **How to handle resource limits without containers?**
   Options: systemd resource limits, cgroups directly, or document as user responsibility

3. **What about multi-host robot deployments?**
   Use `remotes` section + NATS clustering (existing feature)

4. **Should external services have their own topic namespace?**
   Recommendation: No, use standard robot namespace for consistency

---

## Appendix: Files Changed Summary

### Remove
- `/gorai/pkg/quadlet/` (entire directory)

### Deprecate
- `/gorai/pkg/systemd/systemd.go` (container-specific generation)

### Major Changes
- `/gorai/specs/robot-definition-language.md`
- `/gorai/pkg/config/config.go`
- `/gorai/cmd/gorai/commands/start.go`
- `/gorai/cmd/gorai/commands/run.go` (new)
- `/gorai/pkg/robot/robot.go`

### Minor Changes
- `/gorai/specs/deployment.md`
- `/gorai/specs/gorai-container.md`
- `/gorai/specs/runtime.md`
- `/gorai/README.md`
- `/gorai/examples/hello-camera/hello-camera.json`
- `/gorai/examples/hello-camera/README.md`

### Archive
- `/gorai/specs/systemd-container-orchestration.md` (move to `archive/`)
