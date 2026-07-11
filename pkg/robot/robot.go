// Package robot provides the main robot runtime for Gorai.
package robot

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/emergingrobotics/gorai/pkg/config"
	"github.com/emergingrobotics/gorai/pkg/dashboard"
	"github.com/emergingrobotics/gorai/pkg/embeddednats"
	gorainats "github.com/emergingrobotics/gorai/pkg/nats"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/emergingrobotics/gorai/pkg/subjects"
)

// Robot represents a running robot instance.
type Robot struct {
	cfg        *config.RDL
	configPath string
	logger     *slog.Logger
	ctx        context.Context
	cancel     context.CancelFunc

	// Embedded NATS server (nil when using external NATS)
	embeddedNATS *embeddednats.Server

	// NATS client for messaging
	nats     *gorainats.Client
	subjects *subjects.Builder

	// Web dashboard
	dashboard *dashboard.Dashboard

	// Active components (from registry)
	components   map[string]any
	componentsMu sync.RWMutex

	// Component names in creation order (for reverse shutdown)
	componentOrder []string

	// Active internal services (from registry)
	services   map[string]any
	servicesMu sync.RWMutex

	// External services (managed child processes)
	externalServices   map[string]*ExternalService
	externalServicesMu sync.RWMutex

	// Shared dependency bag for component constructors
	sharedDeps *componentDeps
}

// ExternalService represents a managed external service process.
type ExternalService struct {
	Name    string
	Config  config.ServiceConfig
	Cmd     *exec.Cmd
	Cancel  context.CancelFunc
	Running bool
}

// Option configures a Robot.
type Option func(*Robot)

// WithLogger sets the logger for the robot.
func WithLogger(logger *slog.Logger) Option {
	return func(r *Robot) {
		r.logger = logger
	}
}

// WithConfigPath sets the config file path (used for external services).
func WithConfigPath(path string) Option {
	return func(r *Robot) {
		r.configPath = path
	}
}

// New creates a new Robot from the given configuration.
func New(ctx context.Context, cfg *config.RDL, opts ...Option) (*Robot, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	rCtx, cancel := context.WithCancel(ctx)
	r := &Robot{
		cfg:              cfg,
		logger:           slog.Default(),
		ctx:              rCtx,
		cancel:           cancel,
		subjects:         subjects.NewBuilder(cfg.GetEffectiveNamespace()),
		components:       make(map[string]any),
		services:         make(map[string]any),
		externalServices: make(map[string]*ExternalService),
	}

	for _, opt := range opts {
		opt(r)
	}

	r.logger.Info("Robot created", "name", cfg.Robot.Name)

	return r, nil
}

// Start initializes and starts all robot components and services.
func (r *Robot) Start(ctx context.Context) error {
	r.logger.Info("Starting robot", "name", r.cfg.Robot.Name)

	// Start embedded NATS if configured
	if r.cfg.ShouldEmbedNATS() {
		if err := r.startEmbeddedNATS(); err != nil {
			return fmt.Errorf("failed to start embedded NATS: %w", err)
		}
	}

	// Connect to NATS (works whether embedded or external)
	if err := r.connectNATS(ctx); err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Reset configured devices before provisioning
	r.resetDevices()

	// Start dashboard if enabled
	if err := r.startDashboard(ctx); err != nil {
		r.logger.Warn("Failed to start dashboard", "error", err)
	}

	// Publish robot started event
	r.publishStartupEvent(subjects.EventRobotStarted, "", "", "Robot starting initialization", true, nil)

	// Sort components by dependency order
	sortedComponents, err := topoSortComponents(r.cfg.Components)
	if err != nil {
		return fmt.Errorf("component dependency error: %w", err)
	}

	// Create shared deps that accumulates created components
	var natsConn *nats.Conn
	if r.nats != nil {
		natsConn = r.nats.Conn()
	}
	r.sharedDeps = newComponentDeps(natsConn, r.logger)

	// Initialize and start components in dependency order
	for _, comp := range sortedComponents {
		if comp.Disabled {
			r.logger.Info("Skipping disabled component", "name", comp.Name)
			continue
		}

		if err := r.startRegistryComponent(ctx, comp); err != nil {
			return fmt.Errorf("failed to start component %s: %w", comp.Name, err)
		}
	}

	// Initialize services
	for _, svc := range r.cfg.Services {
		if svc.Disabled {
			r.logger.Info("Skipping disabled service", "name", svc.Name)
			continue
		}

		if svc.IsExternal() {
			// Start external service
			if svc.IsManaged() {
				if err := r.startExternalService(ctx, svc); err != nil {
					r.logger.Error("Failed to start external service", "name", svc.Name, "error", err)
					// Don't fail robot startup for external service failures
				}
			} else {
				r.logger.Info("External service (unmanaged)", "name", svc.Name, "type", svc.Type)
			}
		} else {
			r.logger.Info("Initializing internal service", "name", svc.Name, "type", svc.Type)
			if err := r.startInternalService(ctx, svc); err != nil {
				r.logger.Error("Failed to start internal service", "name", svc.Name, "error", err)
				// Don't fail robot startup for service failures
			}
		}
	}

	// Publish robot ready event
	r.publishStartupEvent(subjects.EventRobotReady, "", "", "Robot initialization complete", true, map[string]any{
		"components": len(r.cfg.Components),
		"services":   len(r.cfg.Services),
	})

	r.logger.Info("Robot started", "components", len(r.cfg.Components), "services", len(r.cfg.Services))
	return nil
}

// startEmbeddedNATS creates and starts the embedded NATS server.
func (r *Robot) startEmbeddedNATS() error {
	host, port := parseNATSURL(r.getNATSURL())

	// An explicit listen address lets the embedded server bind a LAN interface
	// (e.g. "0.0.0.0:4222") while the robot's own client keeps dialing nats.url
	// (localhost). This is how you expose the embedded bus on the network.
	if r.cfg.NATS != nil && r.cfg.NATS.Listen != "" {
		if lh, lp, ok := parseHostPort(r.cfg.NATS.Listen); ok {
			host, port = lh, lp
			r.logger.Info("embedded NATS binding to explicit listen address", "listen", r.cfg.NATS.Listen)
		} else {
			r.logger.Warn("invalid nats.listen; using url-derived bind", "listen", r.cfg.NATS.Listen)
		}
	}

	natsConfig := embeddednats.Config{
		Host:      host,
		Port:      port,
		JetStream: r.cfg.NATS.IsJetStreamEnabled(),
		Logger:    r.logger,
	}

	if r.cfg.NATS != nil && r.cfg.NATS.TLS != nil {
		natsConfig.TLS = &embeddednats.TLSConfig{
			CAFile:   r.cfg.NATS.TLS.CAFile,
			CertFile: r.cfg.NATS.TLS.CertFile,
			KeyFile:  r.cfg.NATS.TLS.KeyFile,
		}
	}

	server, err := embeddednats.New(natsConfig)
	if err != nil {
		return fmt.Errorf("failed to create embedded NATS server: %w", err)
	}

	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start embedded NATS server: %w", err)
	}

	r.embeddedNATS = server
	return nil
}

// parseNATSURL extracts host and port from a NATS URL.
// Returns defaults of "127.0.0.1" and 4222 on parse failure.
func parseNATSURL(natsURL string) (string, int) {
	parsed, err := url.Parse(natsURL)
	if err != nil {
		return "127.0.0.1", 4222
	}

	host := parsed.Hostname()
	if host == "" {
		host = "127.0.0.1"
	}

	port := 4222
	if portString := parsed.Port(); portString != "" {
		if parsedPort, err := strconv.Atoi(portString); err == nil {
			port = parsedPort
		}
	}

	return host, port
}

// parseHostPort splits a "host:port" (host may be empty -> 0.0.0.0) into host
// and port, returning ok=false if the port is missing/invalid.
func parseHostPort(addr string) (string, int, bool) {
	i := strings.LastIndex(addr, ":")
	if i < 0 {
		return "", 0, false
	}
	host := addr[:i]
	if host == "" {
		host = "0.0.0.0"
	}
	port, err := strconv.Atoi(addr[i+1:])
	if err != nil {
		return "", 0, false
	}
	return host, port, true
}

// connectNATS establishes connection to the NATS server.
func (r *Robot) connectNATS(ctx context.Context) error {
	natsURL := "nats://localhost:4222"
	if r.cfg.NATS != nil && r.cfg.NATS.URL != "" {
		natsURL = r.cfg.NATS.URL
	}

	natsCfg := &gorainats.Config{
		URL:            natsURL,
		Name:           fmt.Sprintf("gorai-%s", r.cfg.Robot.Name),
		ConnectTimeout: 10 * time.Second,
		ReconnectWait:  2 * time.Second,
		MaxReconnects:  -1,
	}

	client, err := gorainats.Connect(ctx, natsCfg, gorainats.WithLogger(r.logger))
	if err != nil {
		return err
	}

	r.nats = client
	r.logger.Info("Connected to NATS", "url", natsURL)
	return nil
}

// resetDevices sends a GSP/2 RESET command to each configured device
// that has reset_on_startup enabled. Best-effort: failures are logged
// but do not block startup.
func (r *Robot) resetDevices() {
	if r.nats == nil || len(r.cfg.Devices) == 0 {
		return
	}

	reset_count := 0
	for _, dev := range r.cfg.Devices {
		if !dev.ResetOnStartup {
			continue
		}

		subject := fmt.Sprintf("%s.%s.tx.system.reset", dev.NATSPrefix, dev.ID)
		payload := []byte(`{"subsystem":0}`)

		if err := r.nats.Publish(subject, payload); err != nil {
			r.logger.Warn("Failed to send device reset", "device", dev.ID, "subject", subject, "error", err)
			continue
		}

		r.logger.Info("Sent device reset", "device", dev.ID, "subject", subject)
		reset_count++
	}

	if reset_count > 0 {
		if err := r.nats.Conn().Flush(); err != nil {
			r.logger.Warn("Failed to flush NATS after device reset", "error", err)
		}
		time.Sleep(500 * time.Millisecond)
		r.logger.Info("Device reset complete, waiting for devices to enter listening state", "devices_reset", reset_count)
	}
}

// startDashboard creates and starts the web dashboard if enabled.
func (r *Robot) startDashboard(ctx context.Context) error {
	if !r.cfg.IsDashboardEnabled() {
		r.logger.Debug("Dashboard disabled")
		return nil
	}

	dashCfg := r.cfg.Dashboard
	if dashCfg == nil {
		dashCfg = &config.DashboardConfig{}
	}

	d, err := dashboard.New(dashCfg, r.cfg,
		dashboard.WithNATS(r.nats),
		dashboard.WithSubjects(r.subjects),
		dashboard.WithLogger(r.logger),
	)
	if err != nil {
		return fmt.Errorf("failed to create dashboard: %w", err)
	}

	if err := d.Start(ctx); err != nil {
		return fmt.Errorf("failed to start dashboard: %w", err)
	}

	r.dashboard = d

	listen := dashCfg.Listen
	if listen == "" {
		listen = ":10101"
	}
	r.logger.Info("Dashboard started", "listen", listen)
	return nil
}


// Startable is an interface for components that can be started.
type Startable interface {
	Start(ctx context.Context) error
}

// Closeable is an interface for components that can be closed.
type Closeable interface {
	Close(ctx context.Context) error
}

// startRegistryComponent starts a component from the registry.
func (r *Robot) startRegistryComponent(ctx context.Context, comp config.ComponentConfig) error {
	// Look up the constructor
	ctor, err := registry.LookupComponent(comp.Type, comp.Model)
	if err != nil {
		return fmt.Errorf("component %q: type %q model %q not found in registry; "+
			"this component may need to be installed -- try: gorai component search %s",
			comp.Name, comp.Type, comp.Model, comp.Type)
	}

	// Build config for constructor
	conf := registry.Config{
		"name":  comp.Name,
		"type":  comp.Type,
		"model": comp.Model,
		// Pass NATS configuration for components that need it
		"nats_url":   r.getNATSURL(),
		"namespace":  r.cfg.GetEffectiveNamespace(),
		"robot_name": r.cfg.Robot.Name,
	}
	// Merge component attributes into conf
	for k, v := range comp.Attributes {
		conf[k] = v
	}

	// Use shared deps (includes already-created components, NATS, logger)

	// Create the component
	component, err := ctor(ctx, r.sharedDeps, conf)
	if err != nil {
		return fmt.Errorf("failed to create component %s: %w", comp.Name, err)
	}

	// If component is startable, start it
	if startable, ok := component.(Startable); ok {
		if err := startable.Start(r.ctx); err != nil {
			return fmt.Errorf("failed to start component %s: %w", comp.Name, err)
		}
		r.logger.Info("Component started", "name", comp.Name, "type", comp.Type, "model", comp.Model)
	} else {
		r.logger.Info("Component initialized", "name", comp.Name, "type", comp.Type, "model", comp.Model)
	}

	// Track the component
	r.componentsMu.Lock()
	r.components[comp.Name] = component
	r.componentsMu.Unlock()

	// Register so downstream components can access it via deps.Get()
	r.sharedDeps.Add(comp.Name, component)
	r.componentOrder = append(r.componentOrder, comp.Name)

	r.publishStartupEvent(subjects.EventComponentDetected, comp.Name, comp.Type,
		fmt.Sprintf("Component %q started", comp.Name), true, nil)

	return nil
}

// getNATSURL returns the NATS URL from configuration.
func (r *Robot) getNATSURL() string {
	if r.cfg.NATS != nil && r.cfg.NATS.URL != "" {
		return r.cfg.NATS.URL
	}
	return "nats://localhost:4222"
}

// serviceDeps implements resource.Dependencies for services.
// It provides access to robot components by name.
type serviceDeps struct {
	robot *Robot
}

func (d *serviceDeps) Get(name resource.Name) (resource.Resource, error) {
	d.robot.componentsMu.RLock()
	defer d.robot.componentsMu.RUnlock()

	// Look up component by short name
	comp, ok := d.robot.components[name.Name]
	if !ok {
		return nil, fmt.Errorf("component %q not found", name.Name)
	}

	// Check if it implements resource.Resource
	if res, ok := comp.(resource.Resource); ok {
		return res, nil
	}

	// For components that don't implement resource.Resource directly,
	// wrap them in a simple adapter that returns the component
	return &componentAdapter{name: name, component: comp}, nil
}

func (d *serviceDeps) GetByType(subtype string) ([]resource.Resource, error) {
	d.robot.componentsMu.RLock()
	defer d.robot.componentsMu.RUnlock()

	var result []resource.Resource
	for _, comp := range d.robot.components {
		if res, ok := comp.(resource.Resource); ok {
			if res.Name().Subtype == subtype {
				result = append(result, res)
			}
		}
	}
	return result, nil
}

func (d *serviceDeps) All() []resource.Resource {
	d.robot.componentsMu.RLock()
	defer d.robot.componentsMu.RUnlock()

	result := make([]resource.Resource, 0, len(d.robot.components))
	for _, comp := range d.robot.components {
		if res, ok := comp.(resource.Resource); ok {
			result = append(result, res)
		}
	}
	return result
}

// componentAdapter wraps a component that doesn't implement resource.Resource.
type componentAdapter struct {
	name      resource.Name
	component any
}

func (a *componentAdapter) Name() resource.Name { return a.name }
func (a *componentAdapter) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}
func (a *componentAdapter) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	return nil, fmt.Errorf("DoCommand not supported")
}
func (a *componentAdapter) Close(ctx context.Context) error { return nil }

// startInternalService creates and starts an internal service from the registry.
func (r *Robot) startInternalService(ctx context.Context, svc config.ServiceConfig) error {
	// Look up the constructor
	ctor, err := registry.LookupService(svc.Type, svc.Model)
	if err != nil {
		r.logger.Warn("Service not found in registry", "name", svc.Name, "type", svc.Type, "model", svc.Model)
		return nil
	}

	// Build config for constructor
	conf := registry.Config{
		"name":       svc.Name,
		"type":       svc.Type,
		"model":      svc.Model,
		"nats_url":   r.getNATSURL(),
		"namespace":  r.cfg.GetEffectiveNamespace(),
		"robot_name": r.cfg.Robot.Name,
	}
	// Merge service attributes into conf
	for k, v := range svc.Attributes {
		conf[k] = v
	}

	// Use shared deps (includes already-created components, NATS, logger)

	// Create the service
	service, err := ctor(ctx, r.sharedDeps, conf)
	if err != nil {
		return fmt.Errorf("failed to create service %s: %w", svc.Name, err)
	}

	// If service implements resource.Resource, call Reconfigure to wire up dependencies
	if res, ok := service.(resource.Resource); ok {
		svcDeps := &serviceDeps{robot: r}
		resConf := resource.NewConfig(conf)
		if err := res.Reconfigure(ctx, svcDeps, resConf); err != nil {
			return fmt.Errorf("failed to configure service %s: %w", svc.Name, err)
		}
	}

	// If service is startable, start it
	if startable, ok := service.(Startable); ok {
		if err := startable.Start(ctx); err != nil {
			return fmt.Errorf("failed to start service %s: %w", svc.Name, err)
		}
		r.logger.Info("Service started", "name", svc.Name, "type", svc.Type, "model", svc.Model)
	} else {
		r.logger.Info("Service initialized", "name", svc.Name, "type", svc.Type, "model", svc.Model)
	}

	// Track the service
	r.servicesMu.Lock()
	r.services[svc.Name] = service
	r.servicesMu.Unlock()

	return nil
}


// publishStartupEvent publishes a startup event to the NATS system topic.
func (r *Robot) publishStartupEvent(eventType, component, componentType, message string, success bool, details map[string]any) {
	if r.nats == nil {
		return
	}

	event := subjects.StartupEvent{
		EventType:     eventType,
		Component:     component,
		ComponentType: componentType,
		Message:       message,
		Details:       details,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Success:       success,
	}

	subject := r.subjects.SystemStartup()
	if err := r.nats.PublishJSON(subject, event); err != nil {
		r.logger.Warn("Failed to publish startup event", "error", err, "subject", subject)
	} else {
		r.logger.Debug("Published startup event", "subject", subject, "event_type", eventType)
	}
}

// validateExternalCommand checks that an external service command is safe to execute.
func validateExternalCommand(command string) error {
	if command == "" {
		return fmt.Errorf("command is empty")
	}

	// Require absolute path to prevent PATH-based attacks
	if !filepath.IsAbs(command) {
		return fmt.Errorf("external service command must be an absolute path, got: %s", command)
	}

	// Reject shell metacharacters in the command path
	shellMetachars := "`$|;&(){}[]!#~"
	for _, ch := range shellMetachars {
		if strings.ContainsRune(command, ch) {
			return fmt.Errorf("external service command contains forbidden character %q: %s", ch, command)
		}
	}

	// Verify the command exists and is executable
	info, err := os.Stat(command)
	if err != nil {
		return fmt.Errorf("external service command not found: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("external service command is a directory: %s", command)
	}
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("external service command is not executable: %s", command)
	}

	return nil
}

// startExternalService spawns a managed external service process.
func (r *Robot) startExternalService(ctx context.Context, svc config.ServiceConfig) error {
	if svc.External == nil || svc.External.Command == "" {
		return fmt.Errorf("external service %s has no command configured", svc.Name)
	}

	if err := validateExternalCommand(svc.External.Command); err != nil {
		return fmt.Errorf("external service %s: %w", svc.Name, err)
	}

	r.logger.Info("Starting external service",
		"name", svc.Name,
		"command", svc.External.Command,
		"managed", svc.External.Managed,
	)

	// Create cancellable context for this service
	svcCtx, cancel := context.WithCancel(r.ctx)

	// Build command arguments
	args := svc.External.Args
	if r.configPath != "" {
		args = append(args, "--config", r.configPath)
	}
	args = append(args, "--service", svc.Name)

	cmd := exec.CommandContext(svcCtx, svc.External.Command, args...)

	// Set environment variables
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("GORAI_ROBOT_NAME=%s", r.cfg.Robot.Name))
	cmd.Env = append(cmd.Env, fmt.Sprintf("GORAI_SERVICE_NAME=%s", svc.Name))
	if r.cfg.NATS != nil && r.cfg.NATS.URL != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("NATS_URL=%s", r.cfg.NATS.URL))
	}
	for k, v := range svc.External.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Inherit stdout/stderr for logging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the process
	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("failed to start external service %s: %w", svc.Name, err)
	}

	// Track the service
	extSvc := &ExternalService{
		Name:    svc.Name,
		Config:  svc,
		Cmd:     cmd,
		Cancel:  cancel,
		Running: true,
	}

	r.externalServicesMu.Lock()
	r.externalServices[svc.Name] = extSvc
	r.externalServicesMu.Unlock()

	// Monitor the process in a goroutine
	go r.monitorExternalService(extSvc)

	r.logger.Info("External service started", "name", svc.Name, "pid", cmd.Process.Pid)
	return nil
}

// monitorExternalService monitors an external service and restarts if needed.
func (r *Robot) monitorExternalService(svc *ExternalService) {
	for {
		// Wait for process to exit
		err := svc.Cmd.Wait()

		r.externalServicesMu.Lock()
		svc.Running = false
		r.externalServicesMu.Unlock()

		// Check if we should restart
		select {
		case <-r.ctx.Done():
			// Robot is shutting down, don't restart
			r.logger.Debug("External service stopped (robot shutting down)", "name", svc.Name)
			return
		default:
		}

		// Determine restart policy
		restart := svc.Config.External.Restart
		if restart == "" {
			restart = "always"
		}

		shouldRestart := false
		switch restart {
		case "always":
			shouldRestart = true
		case "on-failure":
			shouldRestart = err != nil
		case "never":
			shouldRestart = false
		}

		if !shouldRestart {
			if err != nil {
				r.logger.Error("External service exited with error", "name", svc.Name, "error", err)
			} else {
				r.logger.Info("External service exited", "name", svc.Name)
			}
			return
		}

		r.logger.Warn("External service exited, restarting",
			"name", svc.Name,
			"error", err,
			"restart_policy", restart,
		)

		// Wait a bit before restarting
		time.Sleep(2 * time.Second)

		// Check again if robot is still running
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		// Restart the service
		if err := r.startExternalService(r.ctx, svc.Config); err != nil {
			r.logger.Error("Failed to restart external service", "name", svc.Name, "error", err)
			return
		}
		return // The new instance will be monitored by its own goroutine
	}
}

// stopExternalServices stops all managed external service processes.
func (r *Robot) stopExternalServices(ctx context.Context) {
	r.externalServicesMu.Lock()
	defer r.externalServicesMu.Unlock()

	for name, svc := range r.externalServices {
		r.logger.Info("Stopping external service", "name", name)

		// Cancel the context to signal shutdown
		if svc.Cancel != nil {
			svc.Cancel()
		}

		// Wait for process to exit or kill it
		if svc.Cmd != nil && svc.Cmd.Process != nil && svc.Running {
			// Give it 5 seconds to exit gracefully
			done := make(chan error, 1)
			go func() {
				done <- svc.Cmd.Wait()
			}()

			select {
			case <-done:
				r.logger.Debug("External service stopped gracefully", "name", name)
			case <-time.After(5 * time.Second):
				r.logger.Warn("External service did not stop gracefully, killing", "name", name)
				svc.Cmd.Process.Kill()
			}
		}
	}

	r.externalServices = make(map[string]*ExternalService)
}

// Run runs the robot until the context is cancelled.
func (r *Robot) Run(ctx context.Context) error {
	r.logger.Info("Robot running", "name", r.cfg.Robot.Name)

	// Wait for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.ctx.Done():
		return r.ctx.Err()
	}
}

// Stop gracefully stops the robot.
func (r *Robot) Stop(ctx context.Context) error {
	r.logger.Info("Stopping robot", "name", r.cfg.Robot.Name)

	// Publish shutdown event
	r.publishStartupEvent(subjects.EventRobotShutdown, "", "", "Robot shutting down", true, nil)

	// Stop external services first (they may depend on NATS)
	r.stopExternalServices(ctx)

	// Stop dashboard
	if r.dashboard != nil {
		if err := r.dashboard.Stop(ctx); err != nil {
			r.logger.Warn("Error stopping dashboard", "error", err)
		}
	}

	// Stop all internal services first (they may depend on components)
	r.servicesMu.Lock()
	for name, svc := range r.services {
		r.logger.Info("Stopping service", "name", name)
		if closeable, ok := svc.(Closeable); ok {
			if err := closeable.Close(ctx); err != nil {
				r.logger.Warn("Error closing service", "name", name, "error", err)
			}
		}
	}
	r.services = make(map[string]any)
	r.servicesMu.Unlock()

	// Stop all registry components in reverse dependency order
	for i := len(r.componentOrder) - 1; i >= 0; i-- {
		name := r.componentOrder[i]
		r.componentsMu.RLock()
		comp, ok := r.components[name]
		r.componentsMu.RUnlock()
		if !ok {
			continue
		}
		r.logger.Info("Stopping component", "name", name)
		if closeable, ok := comp.(Closeable); ok {
			if err := closeable.Close(ctx); err != nil {
				r.logger.Warn("Error closing component", "name", name, "error", err)
			}
		}
	}
	r.componentsMu.Lock()
	r.components = make(map[string]any)
	r.componentsMu.Unlock()

	// Cancel internal context
	r.cancel()

	// Close NATS connection
	if r.nats != nil {
		r.nats.Close()
	}

	// Shut down embedded NATS server (after client is closed)
	if r.embeddedNATS != nil {
		r.embeddedNATS.Shutdown()
	}

	r.logger.Info("Robot stopped", "name", r.cfg.Robot.Name)
	return nil
}

// Reconfigure updates the robot configuration at runtime.
func (r *Robot) Reconfigure(ctx context.Context, cfg *config.RDL) error {
	r.logger.Info("Reconfiguring robot", "name", cfg.Robot.Name)

	// TODO: Implement hot reconfiguration
	r.cfg = cfg

	return nil
}

// Config returns the current robot configuration.
func (r *Robot) Config() *config.RDL {
	return r.cfg
}

// NATS returns the NATS client.
func (r *Robot) NATS() *gorainats.Client {
	return r.nats
}

// Subjects returns the subject builder for this robot.
func (r *Robot) Subjects() *subjects.Builder {
	return r.subjects
}
