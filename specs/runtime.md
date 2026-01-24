# Runtime Specification

**Version:** 1.0
**Status:** Draft
**Last Updated:** 2024

## 1. Overview

This specification defines the behavior of the Gorai robot runtime—how a robot binary starts up, initializes components and services, runs, handles signals, and shuts down.

### 1.1 Scope

This specification covers:
- Robot lifecycle (startup, running, shutdown)
- Configuration loading and validation
- Component and service instantiation
- Dependency resolution
- Signal handling
- Hot reload behavior
- Error handling and recovery

### 1.2 Design Goals

1. **Predictable**: Same config produces same behavior
2. **Observable**: Clear logging of lifecycle events
3. **Resilient**: Graceful handling of failures
4. **Responsive**: Quick startup, clean shutdown
5. **Extensible**: Hooks for custom behavior

---

## 2. Runtime Architecture

### 2.1 Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                      Robot Runtime                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   Config    │  │    NATS     │  │      Registry       │  │
│  │   Loader    │  │  Connection │  │  (Components/Svcs)  │  │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘  │
│         │                │                     │            │
│         ▼                ▼                     ▼            │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                   Robot Manager                       │   │
│  │  - Lifecycle orchestration                           │   │
│  │  - Dependency resolution                             │   │
│  │  - Resource management                               │   │
│  └──────────────────────────────────────────────────────┘   │
│         │                                                   │
│         ▼                                                   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                   Resource Pool                       │   │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐     │   │
│  │  │Component│ │Component│ │ Service │ │ Service │     │   │
│  │  │  (IMU)  │ │ (Motor) │ │ (Vision)│ │  (Nav)  │     │   │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘     │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Key Interfaces

```go
// Robot is the main runtime interface
type Robot interface {
    // Start initializes all components and services
    Start(ctx context.Context) error

    // Run blocks until context is cancelled or fatal error
    Run(ctx context.Context) error

    // Stop gracefully shuts down all resources
    Stop(ctx context.Context) error

    // Reconfigure reloads configuration
    Reconfigure(ctx context.Context, cfg *config.Robot) error

    // ResourceByName returns a resource by name
    ResourceByName(name resource.Name) (resource.Resource, error)

    // Components returns all components
    Components() []resource.Resource

    // Services returns all services
    Services() []resource.Resource
}
```

---

## 3. Startup Sequence

### 3.1 Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Startup Sequence                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. Parse command line flags                                │
│  2. Load configuration file                                 │
│  3. Validate configuration                                  │
│  4. Initialize logging                                      │
│  5. Connect to NATS                                         │
│  6. Build dependency graph                                  │
│  7. Instantiate components (in dependency order)            │
│  8. Instantiate services (in dependency order)              │
│  9. Start background processes                              │
│  10. Signal ready                                           │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 Phase 1: Initialization

```go
func main() {
    // 1. Parse flags
    configPath := flag.String("config", "robot.json", "Configuration file")
    flag.Parse()

    // 2. Load configuration
    cfg, err := config.Load(*configPath)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // 3. Validate configuration
    if err := config.Validate(cfg); err != nil {
        log.Fatalf("Invalid config: %v", err)
    }

    // 4. Initialize logging
    logger := log.New(cfg.Log)
}
```

### 3.3 Phase 2: Connection

```go
func (r *robot) connect(ctx context.Context) error {
    // 5. Connect to NATS
    opts := []nats.Option{
        nats.Name(r.cfg.Robot.Name),
        nats.ReconnectWait(r.cfg.NATS.ReconnectWait),
        nats.MaxReconnects(r.cfg.NATS.MaxReconnects),
    }

    nc, err := nats.Connect(r.cfg.NATS.URL, opts...)
    if err != nil {
        return fmt.Errorf("NATS connection failed: %w", err)
    }

    r.nc = nc
    r.logger.Info("Connected to NATS", "url", r.cfg.NATS.URL)
    return nil
}
```

### 3.4 Phase 3: Resource Instantiation

```go
func (r *robot) instantiateResources(ctx context.Context) error {
    // 6. Build dependency graph
    graph := buildDependencyGraph(r.cfg.Components, r.cfg.Services)

    // Check for cycles
    if cycle := graph.FindCycle(); cycle != nil {
        return fmt.Errorf("circular dependency: %v", cycle)
    }

    // Get topological order
    order := graph.TopologicalSort()

    // 7. Instantiate components
    for _, name := range order {
        if compCfg := findComponent(r.cfg.Components, name); compCfg != nil {
            if compCfg.Disabled {
                r.logger.Info("Skipping disabled component", "name", name)
                continue
            }

            if err := r.instantiateComponent(ctx, compCfg); err != nil {
                return fmt.Errorf("failed to create component %s: %w", name, err)
            }
        }
    }

    // 8. Instantiate services
    for _, name := range order {
        if svcCfg := findService(r.cfg.Services, name); svcCfg != nil {
            if svcCfg.Disabled {
                r.logger.Info("Skipping disabled service", "name", name)
                continue
            }

            if err := r.instantiateService(ctx, svcCfg); err != nil {
                return fmt.Errorf("failed to create service %s: %w", name, err)
            }
        }
    }

    return nil
}
```

### 3.5 Phase 4: Ready

```go
func (r *robot) Start(ctx context.Context) error {
    // Connect
    if err := r.connect(ctx); err != nil {
        return err
    }

    // Instantiate
    if err := r.instantiateResources(ctx); err != nil {
        return err
    }

    // 9. Start background processes
    r.startBackgroundTasks(ctx)

    // 10. Signal ready
    r.logger.Info("Robot started",
        "name", r.cfg.Robot.Name,
        "components", len(r.components),
        "services", len(r.services))

    return nil
}
```

---

## 4. Dependency Resolution

### 4.1 Dependency Graph

Resources declare dependencies via `depends_on`:

```json
{
    "components": [
        { "name": "left_motor", "type": "motor", "model": "gpio" },
        { "name": "right_motor", "type": "motor", "model": "gpio" },
        { "name": "base", "type": "base", "model": "differential",
          "depends_on": ["left_motor", "right_motor"] }
    ],
    "services": [
        { "name": "navigator", "type": "navigation", "model": "default",
          "depends_on": ["base", "mapper"] },
        { "name": "mapper", "type": "slam", "model": "cartographer",
          "depends_on": ["lidar", "imu"] }
    ]
}
```

### 4.2 Resolution Algorithm

```go
type DependencyGraph struct {
    nodes map[string]*Node
    edges map[string][]string  // node -> dependencies
}

func (g *DependencyGraph) TopologicalSort() ([]string, error) {
    visited := make(map[string]bool)
    visiting := make(map[string]bool)
    order := make([]string, 0, len(g.nodes))

    var visit func(name string) error
    visit = func(name string) error {
        if visiting[name] {
            return fmt.Errorf("circular dependency at %s", name)
        }
        if visited[name] {
            return nil
        }

        visiting[name] = true

        for _, dep := range g.edges[name] {
            if err := visit(dep); err != nil {
                return err
            }
        }

        visiting[name] = false
        visited[name] = true
        order = append(order, name)
        return nil
    }

    for name := range g.nodes {
        if err := visit(name); err != nil {
            return nil, err
        }
    }

    return order, nil
}
```

### 4.3 Dependency Injection

When instantiating a resource, its dependencies are passed via Dependencies:

```go
func (r *robot) instantiateComponent(ctx context.Context, cfg *config.ComponentConfig) error {
    // Build dependencies map
    deps := make(resource.Dependencies)
    for _, depName := range cfg.DependsOn {
        dep, ok := r.resources[depName]
        if !ok {
            return fmt.Errorf("dependency %s not found", depName)
        }
        deps[depName] = dep
    }

    // Look up constructor
    ctor, err := registry.LookupComponent(cfg.Type, cfg.Model)
    if err != nil {
        return err
    }

    // Create resource
    res, err := ctor(ctx, deps, registry.Config(cfg.Attributes))
    if err != nil {
        return err
    }

    r.resources[cfg.Name] = res
    r.components = append(r.components, res)
    return nil
}
```

---

## 5. Running State

### 5.1 Main Loop

```go
func (r *robot) Run(ctx context.Context) error {
    // Wait for context cancellation or fatal error
    select {
    case <-ctx.Done():
        return ctx.Err()
    case err := <-r.fatalErrors:
        return fmt.Errorf("fatal error: %w", err)
    }
}
```

### 5.2 Background Tasks

The runtime manages background tasks:

```go
func (r *robot) startBackgroundTasks(ctx context.Context) {
    // Health monitoring
    go r.monitorHealth(ctx)

    // Stats collection
    go r.collectStats(ctx)

    // NATS reconnection handling
    r.nc.SetReconnectHandler(func(nc *nats.Conn) {
        r.logger.Info("Reconnected to NATS")
    })

    r.nc.SetDisconnectErrHandler(func(nc *nats.Conn, err error) {
        r.logger.Warn("Disconnected from NATS", "error", err)
    })
}
```

### 5.3 Resource Access

Resources are accessed by name:

```go
// Get a specific component
motor, err := r.ResourceByName(resource.NewName("gorai", "component", "motor", "left_motor"))
if err != nil {
    return err
}

// Type assert
m, ok := motor.(motor.Motor)
if !ok {
    return fmt.Errorf("resource is not a motor")
}

// Use it
m.SetPower(ctx, 0.5)
```

---

## 6. Signal Handling

### 6.1 Supported Signals

| Signal | Action | Description |
|--------|--------|-------------|
| SIGTERM | Shutdown | Graceful shutdown |
| SIGINT | Shutdown | Graceful shutdown (Ctrl+C) |
| SIGHUP | Reload | Reload configuration |
| SIGUSR1 | Custom | Dump status (optional) |
| SIGUSR2 | Custom | Toggle debug (optional) |

### 6.2 Signal Setup

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Setup signal handling
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

    // Create and start robot
    r, err := robot.New(ctx, cfg)
    if err != nil {
        log.Fatal(err)
    }

    if err := r.Start(ctx); err != nil {
        log.Fatal(err)
    }

    // Handle signals
    go func() {
        for sig := range sigCh {
            switch sig {
            case syscall.SIGTERM, syscall.SIGINT:
                log.Info("Shutdown signal received")
                cancel()
            case syscall.SIGHUP:
                log.Info("Reload signal received")
                if err := r.Reload(ctx); err != nil {
                    log.Error("Reload failed", "error", err)
                }
            }
        }
    }()

    // Run until done
    if err := r.Run(ctx); err != nil && err != context.Canceled {
        log.Fatal(err)
    }

    // Cleanup
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer shutdownCancel()

    if err := r.Stop(shutdownCtx); err != nil {
        log.Error("Shutdown error", "error", err)
    }
}
```

---

## 7. Shutdown Sequence

### 7.1 Overview

```
┌─────────────────────────────────────────────────────────────┐
│                   Shutdown Sequence                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. Receive shutdown signal (SIGTERM/SIGINT)                │
│  2. Cancel main context                                     │
│  3. Stop accepting new requests                             │
│  4. Stop services (reverse dependency order)                │
│  5. Stop actuators (motors, servos, etc.)                   │
│  6. Stop sensors                                            │
│  7. Close NATS connection                                   │
│  8. Flush logs                                              │
│  9. Exit                                                    │
│                                                             │
│  Timeout: 30 seconds (configurable)                         │
│  Force kill if timeout exceeded                             │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 7.2 Graceful Stop

```go
func (r *robot) Stop(ctx context.Context) error {
    r.logger.Info("Stopping robot")

    // 3. Stop accepting new requests
    r.stopping = true

    // 4. Stop services (reverse order)
    for i := len(r.services) - 1; i >= 0; i-- {
        svc := r.services[i]
        r.logger.Debug("Stopping service", "name", svc.Name())
        if err := svc.Close(ctx); err != nil {
            r.logger.Error("Service stop error", "name", svc.Name(), "error", err)
        }
    }

    // 5. Stop actuators first (safety)
    for _, comp := range r.components {
        if actuator, ok := comp.(resource.Actuator); ok {
            r.logger.Debug("Stopping actuator", "name", comp.Name())
            if err := actuator.Stop(ctx); err != nil {
                r.logger.Error("Actuator stop error", "name", comp.Name(), "error", err)
            }
        }
    }

    // 6. Stop all components
    for _, comp := range r.components {
        r.logger.Debug("Closing component", "name", comp.Name())
        if err := comp.Close(ctx); err != nil {
            r.logger.Error("Component close error", "name", comp.Name(), "error", err)
        }
    }

    // 7. Close NATS
    if r.nc != nil {
        r.nc.Drain()
        r.nc.Close()
    }

    // 8. Flush logs
    r.logger.Info("Robot stopped")

    return nil
}
```

### 7.3 Timeout Handling

```go
func (r *robot) StopWithTimeout(timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    done := make(chan error, 1)
    go func() {
        done <- r.Stop(ctx)
    }()

    select {
    case err := <-done:
        return err
    case <-ctx.Done():
        r.logger.Warn("Shutdown timeout, forcing exit")
        return ctx.Err()
    }
}
```

---

## 8. Hot Reload

### 8.1 Reload Trigger

Hot reload is triggered by:
- SIGHUP signal
- API call (future)
- File watcher (optional)

### 8.2 Reload Scope

| Aspect | Hot Reloadable | Notes |
|--------|----------------|-------|
| Component attributes | Yes | Via Reconfigure() |
| Service attributes | Yes | Via Reconfigure() |
| Log level | Yes | Immediate |
| Add new component | No | Requires restart |
| Remove component | No | Requires restart |
| Change component type | No | Requires restart |
| NATS URL | No | Requires restart |

### 8.3 Reload Process

```go
func (r *robot) Reload(ctx context.Context) error {
    r.logger.Info("Reloading configuration")

    // Load new config
    newCfg, err := config.Load(r.configPath)
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }

    // Validate
    if err := config.Validate(newCfg); err != nil {
        return fmt.Errorf("invalid config: %w", err)
    }

    // Check structural compatibility
    if err := r.checkReloadCompatibility(newCfg); err != nil {
        return fmt.Errorf("incompatible config change: %w", err)
    }

    // Update logging
    r.logger.SetLevel(newCfg.Log.Level)

    // Reconfigure components
    for _, compCfg := range newCfg.Components {
        comp, ok := r.resources[compCfg.Name]
        if !ok {
            continue // New component, can't hot reload
        }

        deps := r.buildDeps(compCfg.DependsOn)
        if err := comp.Reconfigure(ctx, deps, registry.Config(compCfg.Attributes)); err != nil {
            r.logger.Error("Component reconfigure failed",
                "name", compCfg.Name,
                "error", err)
            // Continue with other components
        } else {
            r.logger.Info("Component reconfigured", "name", compCfg.Name)
        }
    }

    // Reconfigure services
    for _, svcCfg := range newCfg.Services {
        svc, ok := r.resources[svcCfg.Name]
        if !ok {
            continue
        }

        deps := r.buildDeps(svcCfg.DependsOn)
        if err := svc.Reconfigure(ctx, deps, registry.Config(svcCfg.Attributes)); err != nil {
            r.logger.Error("Service reconfigure failed",
                "name", svcCfg.Name,
                "error", err)
        } else {
            r.logger.Info("Service reconfigured", "name", svcCfg.Name)
        }
    }

    r.cfg = newCfg
    r.logger.Info("Configuration reloaded")
    return nil
}
```

---

## 9. Error Handling

### 9.1 Error Categories

| Category | Severity | Action |
|----------|----------|--------|
| Validation error | Fatal | Exit immediately |
| NATS connection failed | Fatal | Exit (or retry) |
| Component creation failed | Fatal | Exit (no partial start) |
| Component runtime error | Recoverable | Log, continue |
| Service creation failed | Configurable | Skip or exit |
| Resource timeout | Recoverable | Log, retry |

### 9.2 Error Recovery

```go
// Component with automatic recovery
func (r *robot) monitorComponent(ctx context.Context, comp resource.Resource) {
    for {
        select {
        case <-ctx.Done():
            return
        case err := <-comp.Errors():
            r.logger.Error("Component error",
                "name", comp.Name(),
                "error", err)

            // Attempt recovery
            if recoverable, ok := comp.(Recoverable); ok {
                if err := recoverable.Recover(ctx); err != nil {
                    r.logger.Error("Recovery failed",
                        "name", comp.Name(),
                        "error", err)
                } else {
                    r.logger.Info("Component recovered", "name", comp.Name())
                }
            }
        }
    }
}
```

### 9.3 Fatal Errors

```go
func (r *robot) reportFatalError(err error) {
    select {
    case r.fatalErrors <- err:
    default:
        // Channel full, already shutting down
    }
}
```

---

## 10. Logging

### 10.1 Lifecycle Events

The runtime logs these events:

| Event | Level | Message |
|-------|-------|---------|
| Startup begin | Info | "Starting robot" |
| Config loaded | Debug | "Configuration loaded" |
| NATS connected | Info | "Connected to NATS" |
| Component created | Debug | "Component created" |
| Service created | Debug | "Service created" |
| Robot ready | Info | "Robot started" |
| Reload triggered | Info | "Reloading configuration" |
| Shutdown begin | Info | "Stopping robot" |
| Component stopped | Debug | "Component stopped" |
| Robot stopped | Info | "Robot stopped" |

### 10.2 Log Format

```go
// Structured logging example
r.logger.Info("Robot started",
    "name", r.cfg.Robot.Name,
    "namespace", r.cfg.Robot.Namespace,
    "components", len(r.components),
    "services", len(r.services),
    "nats_url", r.cfg.NATS.URL)
```

JSON output:
```json
{
    "time": "2024-01-15T10:30:00Z",
    "level": "INFO",
    "msg": "Robot started",
    "name": "my-robot",
    "namespace": "mybot",
    "components": 5,
    "services": 2,
    "nats_url": "nats://localhost:4222"
}
```

---

## 11. Metrics and Observability

### 11.1 Built-in Metrics

The runtime exposes metrics via NATS:

| Metric | Type | Description |
|--------|------|-------------|
| `gorai.{ns}.robot.uptime` | Gauge | Seconds since start |
| `gorai.{ns}.robot.components` | Gauge | Number of components |
| `gorai.{ns}.robot.services` | Gauge | Number of services |
| `gorai.{ns}.robot.errors` | Counter | Error count |
| `gorai.{ns}.robot.restarts` | Counter | Reload count |

### 11.2 Health Endpoint

```go
func (r *robot) publishHealth(ctx context.Context) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            health := &Health{
                Status:     "running",
                Uptime:     time.Since(r.startTime).Seconds(),
                Components: len(r.components),
                Services:   len(r.services),
                Timestamp:  time.Now(),
            }

            data, _ := json.Marshal(health)
            r.nc.Publish(fmt.Sprintf("gorai.%s.robot.health", r.cfg.Robot.Namespace), data)
        }
    }
}
```

---

## 12. API Reference

### 12.1 robot.New()

```go
// New creates a new robot from configuration
func New(ctx context.Context, cfg *config.Robot, opts ...Option) (Robot, error)

// Options
type Option func(*robotOptions)

func WithLogger(logger *slog.Logger) Option
func WithNATSConnection(nc *nats.Conn) Option
func WithRegistry(reg *registry.Registry) Option
```

### 12.2 robot.Start()

```go
// Start initializes and starts all resources
// Returns error if any component fails to start
func (r *Robot) Start(ctx context.Context) error
```

### 12.3 robot.Run()

```go
// Run blocks until context is cancelled or fatal error
// Normal shutdown returns nil or context.Canceled
func (r *Robot) Run(ctx context.Context) error
```

### 12.4 robot.Stop()

```go
// Stop gracefully shuts down all resources
// Respects context timeout
func (r *Robot) Stop(ctx context.Context) error
```

### 12.5 robot.Reconfigure()

```go
// Reconfigure updates configuration without restart
// Only attribute changes are supported
func (r *Robot) Reconfigure(ctx context.Context, cfg *config.Robot) error
```

---

## 13. Example Implementation

### 13.1 Minimal main.go

```go
package main

import (
    "context"
    "flag"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "github.com/gorai/gorai/pkg/config"
    "github.com/gorai/gorai/pkg/robot"

    // Import components to register them
    _ "github.com/gorai/gorai/components/motor/gpio"
    _ "github.com/gorai/gorai/components/sensor/mpu6050"
)

func main() {
    configPath := flag.String("config", "robot.json", "Path to configuration")
    flag.Parse()

    // Load config
    cfg, err := config.Load(*configPath)
    if err != nil {
        slog.Error("Failed to load config", "error", err)
        os.Exit(1)
    }

    // Setup context with signal handling
    ctx, cancel := context.WithCancel(context.Background())
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

    // Create robot
    r, err := robot.New(ctx, cfg)
    if err != nil {
        slog.Error("Failed to create robot", "error", err)
        os.Exit(1)
    }

    // Start robot
    if err := r.Start(ctx); err != nil {
        slog.Error("Failed to start robot", "error", err)
        os.Exit(1)
    }

    // Signal handler
    go func() {
        for sig := range sigCh {
            switch sig {
            case syscall.SIGTERM, syscall.SIGINT:
                slog.Info("Shutdown signal received")
                cancel()
            case syscall.SIGHUP:
                slog.Info("Reload signal received")
                newCfg, err := config.Load(*configPath)
                if err != nil {
                    slog.Error("Reload failed", "error", err)
                    continue
                }
                if err := r.Reconfigure(ctx, newCfg); err != nil {
                    slog.Error("Reconfigure failed", "error", err)
                }
            }
        }
    }()

    // Run
    if err := r.Run(ctx); err != nil && err != context.Canceled {
        slog.Error("Robot error", "error", err)
    }

    // Shutdown
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer shutdownCancel()

    if err := r.Stop(shutdownCtx); err != nil {
        slog.Error("Shutdown error", "error", err)
        os.Exit(1)
    }

    slog.Info("Robot stopped cleanly")
}
```

---

## Appendix A: State Machine

```
                    ┌─────────────┐
                    │   Created   │
                    └──────┬──────┘
                           │ Start()
                           ▼
                    ┌─────────────┐
                    │  Starting   │
                    └──────┬──────┘
                           │ (all resources ready)
                           ▼
         ┌────────────────────────────────────┐
         │                                    │
    SIGHUP│            ┌─────────────┐        │ SIGTERM/SIGINT
         │            │   Running   │◄───────┤
         │            └──────┬──────┘        │
         │                   │               │
         ▼                   │               │
    ┌─────────────┐         │               │
    │  Reloading  │─────────┘               │
    └─────────────┘                         │
                                            │
                                            ▼
                                     ┌─────────────┐
                                     │  Stopping   │
                                     └──────┬──────┘
                                            │
                                            ▼
                                     ┌─────────────┐
                                     │   Stopped   │
                                     └─────────────┘
```

---

## Appendix B: Timeout Defaults

| Operation | Default Timeout | Configurable |
|-----------|-----------------|--------------|
| NATS connect | 5s | Yes |
| Component start | 30s | Yes |
| Service start | 60s | Yes |
| Graceful shutdown | 30s | Yes |
| Reconfigure | 10s | Yes |
