# Integrated Container Build Plan

**Date:** 2025-12-14
**Status:** Planning
**Goal:** Make `gorai build` automatically build all container services defined in RDL

## Problem Statement

Currently, container services like the Hailo detector must be built separately:
```bash
cd services/hailo-detector && podman build -t localhost/hailo-detector:latest .
```

This is error-prone and disconnected from the RDL configuration. Users must:
1. Know which services need containers
2. Build each one manually
3. Ensure image names match the RDL config
4. Rebuild when dependencies change

## Proposed Solution

Extend the RDL schema and `gorai build` command to automatically discover and build container services.

### RDL Schema Extension

Add a `build` section to `ContainerServiceConfig`:

```json
{
  "services": [
    {
      "name": "person_detector",
      "type": "object_detection",
      "model": "hailo_yolox",
      "external": {
        "enabled": true,
        "container": {
          "image": "localhost/hailo-detector:latest",
          "build": {
            "context": "./services/hailo-detector",
            "containerfile": "Containerfile",
            "args": {
              "MODEL_VERSION": "v2.10.0"
            }
          },
          "devices": ["/dev/hailo0"],
          "network": "host"
        }
      }
    }
  ]
}
```

#### Build Object Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `context` | string | Yes | - | Build context directory (relative to config) |
| `containerfile` | string | No | `Containerfile` | Containerfile name |
| `args` | object | No | {} | Build arguments |
| `target` | string | No | - | Multi-stage build target |
| `no_cache` | bool | No | false | Disable build cache |

### Convention-Based Discovery

If `build` is not specified but `image` starts with `localhost/`, look for:

1. `./services/<service-name>/Containerfile`
2. `./services/<service-name>/Dockerfile`
3. `./<service-name>/Containerfile`

This allows zero-config for standard layouts:

```json
{
  "external": {
    "container": {
      "image": "localhost/hailo-detector:latest"
    }
  }
}
```

With directory structure:
```
project/
├── robot.json
└── services/
    └── hailo-detector/
        ├── Containerfile
        └── detector.py
```

## Implementation Plan

### Phase 1: Schema Updates

**File:** `pkg/config/config.go`

```go
// ContainerServiceConfig configures an external service to run as a container.
type ContainerServiceConfig struct {
    Image       string               `json:"image"`
    Build       *ContainerBuildConfig `json:"build,omitempty"`
    Devices     []string             `json:"devices,omitempty"`
    Environment map[string]string    `json:"environment,omitempty"`
    Volumes     []string             `json:"volumes,omitempty"`
    Network     string               `json:"network,omitempty"`
    Privileged  bool                 `json:"privileged,omitempty"`
}

// ContainerBuildConfig configures how to build a container image.
type ContainerBuildConfig struct {
    Context       string            `json:"context"`
    Containerfile string            `json:"containerfile,omitempty"`
    Args          map[string]string `json:"args,omitempty"`
    Target        string            `json:"target,omitempty"`
    NoCache       bool              `json:"no_cache,omitempty"`
}
```

### Phase 2: Build Command Enhancement

**File:** `cmd/gorai/commands/build.go`

```go
func cmdBuild() error {
    // Parse flags
    cfg, err := config.Load(configPath)

    // Get buildable services
    buildables := getBuildableServices(cfg, configDir)

    // Build each container
    for _, svc := range buildables {
        if err := buildContainer(svc); err != nil {
            return fmt.Errorf("failed to build %s: %w", svc.Name, err)
        }
    }

    return nil
}

func getBuildableServices(cfg *config.RDL, configDir string) []BuildableService {
    var result []BuildableService

    for _, svc := range cfg.Services {
        if !svc.IsExternal() || svc.External.Container == nil {
            continue
        }

        container := svc.External.Container

        // Skip non-local images (they're pulled, not built)
        if !strings.HasPrefix(container.Image, "localhost/") {
            continue
        }

        bs := BuildableService{
            Name:  svc.Name,
            Image: container.Image,
        }

        // Use explicit build config if provided
        if container.Build != nil {
            bs.Context = filepath.Join(configDir, container.Build.Context)
            bs.Containerfile = container.Build.Containerfile
            bs.Args = container.Build.Args
            bs.Target = container.Build.Target
            bs.NoCache = container.Build.NoCache
        } else {
            // Convention-based discovery
            bs.Context = discoverBuildContext(configDir, svc.Name)
            bs.Containerfile = discoverContainerfile(bs.Context)
        }

        if bs.Context != "" {
            result = append(result, bs)
        }
    }

    return result
}

func buildContainer(svc BuildableService) error {
    args := []string{"build", "-t", svc.Image}

    if svc.Containerfile != "" {
        args = append(args, "-f", filepath.Join(svc.Context, svc.Containerfile))
    }

    for k, v := range svc.Args {
        args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, v))
    }

    if svc.Target != "" {
        args = append(args, "--target", svc.Target)
    }

    if svc.NoCache {
        args = append(args, "--no-cache")
    }

    args = append(args, svc.Context)

    cmd := exec.Command("podman", args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    return cmd.Run()
}
```

### Phase 3: CLI Integration

Update `gorai build` command:

```
gorai build - Build robot and container services

Usage:
  gorai build [--config robot.json] [flags]

Flags:
  -c, --config <file>     Path to robot configuration file
  --services              Build container services only
  --no-cache              Disable build cache for containers
  --parallel              Build containers in parallel
  -h, --help              Show this help message

Examples:
  gorai build --config robot.json           # Build all containers
  gorai build --config robot.json --services  # Only build service containers
  gorai build --config robot.json --no-cache  # Force rebuild
```

### Phase 4: Dependency Resolution

Handle build dependencies between services:

```json
{
  "services": [
    {
      "name": "base_ml",
      "external": {
        "container": {
          "image": "localhost/base-ml:latest",
          "build": {"context": "./services/base-ml"}
        }
      }
    },
    {
      "name": "person_detector",
      "depends_on": ["base_ml"],
      "external": {
        "container": {
          "image": "localhost/hailo-detector:latest",
          "build": {
            "context": "./services/hailo-detector",
            "args": {
              "BASE_IMAGE": "localhost/base-ml:latest"
            }
          }
        }
      }
    }
  ]
}
```

Build order respects `depends_on`.

## File Changes Required

### New Files
- None (all changes to existing files)

### Modified Files

1. **`pkg/config/config.go`**
   - Add `ContainerBuildConfig` struct
   - Add `Build` field to `ContainerServiceConfig`

2. **`cmd/gorai/commands/build.go`**
   - Add container build logic
   - Add convention-based discovery
   - Add parallel build support

3. **`cmd/gorai/commands/root.go`**
   - Update help text for build command

4. **`specs/robot-definition-language.md`**
   - Document `build` configuration

5. **`examples/hello-camera/hello-camera.json`**
   - Add build config to person_detector

## Example: Updated hello-camera.json

```json
{
  "$schema": "https://gorai.dev/schemas/rdl-v2.json",
  "version": "2",

  "robot": {
    "name": "hello-camera",
    "description": "Camera robot with Hailo NPU person detection"
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
        "height": 480
      }
    }
  ],

  "services": [
    {
      "name": "dashboard",
      "type": "dashboard",
      "model": "web"
    },
    {
      "name": "person_detector",
      "type": "object_detection",
      "model": "hailo_yolox",
      "external": {
        "enabled": true,
        "container": {
          "image": "localhost/hailo-detector:latest",
          "build": {
            "context": "../../services/hailo-detector"
          },
          "devices": ["/dev/hailo0"],
          "network": "host",
          "volumes": ["/opt/gorai/models:/models:ro"],
          "environment": {
            "MODEL_PATH": "/models/yolox_s_leaky.hef",
            "CONFIDENCE_THRESHOLD": "0.5"
          }
        },
        "managed": false
      }
    }
  ]
}
```

## Workflow

### Developer Workflow

```bash
# Clone project
git clone https://github.com/example/my-robot
cd my-robot

# Build everything (robot binary + all container services)
gorai build --config robot.json

# Start robot (containers started separately or via systemd)
gorai start --config robot.json
```

### CI/CD Workflow

```yaml
# .github/workflows/build.yml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Build robot and services
        run: gorai build --config robot.json

      - name: Push container images
        run: |
          podman push localhost/hailo-detector:latest \
            ghcr.io/${{ github.repository }}/hailo-detector:${{ github.sha }}
```

## Testing Plan

### Unit Tests
- Config parsing with build section
- Convention-based discovery logic
- Build command generation

### Integration Tests
- Build container from explicit config
- Build container from convention
- Handle missing Containerfile gracefully
- Parallel builds

### E2E Tests
- `gorai build` builds all containers
- Built images run correctly
- `gorai start` works with built images

## Migration

Existing projects without `build` config continue to work:
- If image is `localhost/*` and no build config, look for convention paths
- If no Containerfile found, skip with warning
- Remote images (not `localhost/`) are pulled, not built

## Timeline

| Phase | Description |
|-------|-------------|
| 1 | Schema updates to config.go |
| 2 | Build command implementation |
| 3 | CLI integration and help |
| 4 | Dependency resolution |
| 5 | Testing and documentation |

## Open Questions

1. **Should `gorai start` auto-build if needed?**
   - Option A: Yes, convenience for development
   - Option B: No, explicit build step required
   - Recommendation: Add `--build` flag to `gorai start`

2. **Where should services directory live?**
   - Option A: `./services/<name>/` (proposed)
   - Option B: `./containers/<name>/`
   - Option C: Configurable in RDL
   - Recommendation: Convention with override

3. **How to handle build failures?**
   - Option A: Stop on first failure
   - Option B: Continue and report all failures
   - Recommendation: Stop on first (default), `--continue-on-error` flag

4. **Multi-architecture builds?**
   - Could add `platforms` to build config
   - Use `podman build --platform linux/arm64,linux/amd64`
   - Defer to Phase 2
