// Package robot provides the main robot runtime for Gorai.
package robot

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorai/gorai/driver/camera/v4l2"
	"github.com/gorai/gorai/pkg/config"
	"github.com/gorai/gorai/pkg/dashboard"
	hwv4l2 "github.com/gorai/gorai/pkg/hardware/v4l2"
	gorainats "github.com/gorai/gorai/pkg/nats"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
	"github.com/gorai/gorai/pkg/topics"
)

// robotDeps implements the registry.Dependencies interface.
type robotDeps struct {
	deps map[string]any
}

// Get returns a dependency by name.
func (d *robotDeps) Get(name string) (any, error) {
	if v, ok := d.deps[name]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("dependency %q not found", name)
}

// GetByType returns all dependencies of a given type.
func (d *robotDeps) GetByType(subtype string) ([]any, error) {
	// For now, just return empty. In future, track component types.
	return nil, nil
}

// Robot represents a running robot instance.
type Robot struct {
	cfg        *config.RDL
	configPath string
	logger     *slog.Logger
	ctx        context.Context
	cancel     context.CancelFunc

	// NATS client for messaging
	nats   *gorainats.Client
	topics *topics.Builder

	// Web dashboard
	dashboard *dashboard.Dashboard

	// Active cameras
	cameras   map[string]*v4l2.Camera
	camerasMu sync.RWMutex

	// Active components (from registry)
	components   map[string]any
	componentsMu sync.RWMutex

	// Active internal services (from registry)
	services   map[string]any
	servicesMu sync.RWMutex

	// External services (managed child processes)
	externalServices   map[string]*ExternalService
	externalServicesMu sync.RWMutex

	// Frame counters for logging
	frameCounters   map[string]uint64
	frameCountersMu sync.Mutex
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
		topics:           topics.NewBuilder(cfg.Robot.Name),
		cameras:          make(map[string]*v4l2.Camera),
		components:       make(map[string]any),
		services:         make(map[string]any),
		externalServices: make(map[string]*ExternalService),
		frameCounters:    make(map[string]uint64),
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

	// Connect to NATS
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
	r.publishStartupEvent(topics.EventRobotStarted, "", "", "Robot starting initialization", true, nil)

	// Detect and validate hardware components
	if err := r.detectHardware(ctx); err != nil {
		r.publishStartupEvent(topics.EventComponentError, "", "", fmt.Sprintf("Hardware detection failed: %v", err), false, nil)
		return err
	}

	// Initialize and start components
	for _, comp := range r.cfg.Components {
		if comp.Disabled {
			r.logger.Info("Skipping disabled component", "name", comp.Name)
			continue
		}

		switch comp.Type {
		case "camera":
			// Remote cameras use registry, local cameras use built-in V4L2 driver
			if comp.Model == "remote" {
				if err := r.startRegistryComponent(ctx, comp); err != nil {
					return fmt.Errorf("failed to start component %s: %w", comp.Name, err)
				}
			} else {
				if err := r.startCamera(ctx, comp); err != nil {
					return fmt.Errorf("failed to start camera %s: %w", comp.Name, err)
				}
			}
		default:
			if err := r.startRegistryComponent(ctx, comp); err != nil {
				return fmt.Errorf("failed to start component %s: %w", comp.Name, err)
			}
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
	r.publishStartupEvent(topics.EventRobotReady, "", "", "Robot initialization complete", true, map[string]any{
		"components": len(r.cfg.Components),
		"services":   len(r.cfg.Services),
	})

	r.logger.Info("Robot started", "components", len(r.cfg.Components), "services", len(r.cfg.Services))
	return nil
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
		dashboard.WithTopics(r.topics),
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
		listen = ":8080"
	}
	r.logger.Info("Dashboard started", "listen", listen)
	return nil
}

// detectHardware detects and validates all hardware components.
func (r *Robot) detectHardware(ctx context.Context) error {
	for _, comp := range r.cfg.Components {
		if comp.Disabled {
			continue
		}

		switch comp.Type {
		case "camera":
			if err := r.detectCamera(comp); err != nil {
				return err
			}
		default:
			r.logger.Debug("No hardware detection for component type", "type", comp.Type, "name", comp.Name)
		}
	}
	return nil
}

// detectCamera checks if a camera device is present.
func (r *Robot) detectCamera(comp config.ComponentConfig) error {
	// Skip hardware detection for remote cameras (they subscribe to NATS, no local device)
	if comp.Model == "remote" {
		r.logger.Debug("Skipping hardware detection for remote camera", "name", comp.Name)
		return nil
	}

	devicePath := "/dev/video0"
	if comp.Attributes != nil {
		if dev, ok := comp.Attributes["device"].(string); ok {
			devicePath = dev
		}
	}

	r.logger.Info("Detecting camera", "name", comp.Name, "device", devicePath)

	result := hwv4l2.DetectDevice(devicePath)

	if !result.Found {
		errMsg := fmt.Sprintf("Camera %q not found at %s: %s", comp.Name, devicePath, result.Error)
		r.logger.Error("Camera detection failed", "name", comp.Name, "device", devicePath, "error", result.Error)

		r.publishStartupEvent(topics.EventComponentMissing, comp.Name, comp.Type, errMsg, false, map[string]any{
			"device": devicePath,
			"error":  result.Error,
		})

		return fmt.Errorf("%s", errMsg)
	}

	deviceInfo := result.Device
	summary := hwv4l2.GetDeviceSummary(deviceInfo)

	r.logger.Info("Camera detected",
		"name", comp.Name,
		"device", devicePath,
		"device_name", deviceInfo.Name,
		"driver", deviceInfo.Driver,
	)

	r.publishStartupEvent(topics.EventComponentDetected, comp.Name, comp.Type,
		fmt.Sprintf("Camera %q detected: %s", comp.Name, summary), true, map[string]any{
			"device":      devicePath,
			"device_name": deviceInfo.Name,
			"driver":      deviceInfo.Driver,
			"index":       deviceInfo.Index,
		})

	return nil
}

// startCamera creates and starts a camera component.
func (r *Robot) startCamera(ctx context.Context, comp config.ComponentConfig) error {
	// Extract configuration from attributes
	devicePath := "/dev/video0"
	width := uint32(640)
	height := uint32(480)
	frameRate := float64(30)
	jpegQuality := 80

	if comp.Attributes != nil {
		if dev, ok := comp.Attributes["device"].(string); ok {
			devicePath = dev
		}
		if w, ok := comp.Attributes["width"].(float64); ok {
			width = uint32(w)
		}
		if h, ok := comp.Attributes["height"].(float64); ok {
			height = uint32(h)
		}
		if fr, ok := comp.Attributes["frame_rate"].(float64); ok {
			frameRate = fr
		}
		if q, ok := comp.Attributes["jpeg_quality"].(float64); ok {
			jpegQuality = int(q)
		}
	}

	cfg := &v4l2.Config{
		Device:      devicePath,
		Width:       width,
		Height:      height,
		FrameRate:   frameRate,
		JPEGQuality: jpegQuality,
	}

	// Create frame topic for this camera
	frameTopic := r.topics.ComponentData(comp.Name)

	// Create camera with frame callback
	cam, err := v4l2.New(cfg,
		v4l2.WithLogger(r.logger),
		v4l2.WithOnFrame(func(jpeg []byte, timestamp time.Time) {
			r.publishFrame(comp.Name, frameTopic, jpeg, timestamp)
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to create camera: %w", err)
	}

	// Open the camera
	if err := cam.Open(); err != nil {
		return fmt.Errorf("failed to open camera: %w", err)
	}

	// Start streaming
	if err := cam.Start(r.ctx); err != nil {
		cam.Close()
		return fmt.Errorf("failed to start camera: %w", err)
	}

	// Store camera reference
	r.camerasMu.Lock()
	r.cameras[comp.Name] = cam
	r.camerasMu.Unlock()

	r.logger.Info("Camera started",
		"name", comp.Name,
		"device", devicePath,
		"resolution", fmt.Sprintf("%dx%d", width, height),
		"fps", frameRate,
		"topic", frameTopic,
	)

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
		r.logger.Warn("Component not found in registry", "name", comp.Name, "type", comp.Type, "model", comp.Model)
		r.logger.Info("Skipping unregistered component", "name", comp.Name, "type", comp.Type, "model", comp.Model)
		return nil
	}

	// Build config for constructor
	conf := registry.Config{
		"name":  comp.Name,
		"type":  comp.Type,
		"model": comp.Model,
		// Pass NATS configuration for components that need it
		"nats_url":   r.getNATSURL(),
		"namespace":  "gorai",
		"robot_name": r.cfg.Robot.Name,
	}
	// Merge component attributes into conf
	for k, v := range comp.Attributes {
		conf[k] = v
	}

	// Build dependencies with NATS and logger
	deps := &robotDeps{deps: make(map[string]any)}
	if r.nats != nil {
		deps.deps["nats"] = r.nats.Conn()
	}
	if r.logger != nil {
		deps.deps["logger"] = r.logger
	}

	// Create the component
	component, err := ctor(ctx, deps, conf)
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

	r.publishStartupEvent(topics.EventComponentDetected, comp.Name, comp.Type,
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
		"namespace":  "gorai",
		"robot_name": r.cfg.Robot.Name,
	}
	// Merge service attributes into conf
	for k, v := range svc.Attributes {
		conf[k] = v
	}

	// Build dependencies with NATS and logger for initial construction
	deps := &robotDeps{deps: make(map[string]any)}
	if r.nats != nil {
		deps.deps["nats"] = r.nats.Conn()
	}
	if r.logger != nil {
		deps.deps["logger"] = r.logger
	}

	// Create the service
	service, err := ctor(ctx, deps, conf)
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

// publishFrame publishes a camera frame to NATS.
func (r *Robot) publishFrame(cameraName, topic string, jpeg []byte, timestamp time.Time) {
	if r.nats == nil {
		return
	}

	// Publish raw JPEG data to the topic
	if err := r.nats.Publish(topic, jpeg); err != nil {
		r.logger.Warn("Failed to publish frame", "camera", cameraName, "error", err)
		return
	}

	// Update frame counter and log periodically
	r.frameCountersMu.Lock()
	r.frameCounters[cameraName]++
	count := r.frameCounters[cameraName]
	r.frameCountersMu.Unlock()

	// Log every 100 frames
	if count%100 == 0 {
		r.logger.Debug("Camera frames published",
			"camera", cameraName,
			"frames", count,
			"topic", topic,
			"size_kb", len(jpeg)/1024,
		)
	}
}

// publishStartupEvent publishes a startup event to the NATS system topic.
func (r *Robot) publishStartupEvent(eventType, component, componentType, message string, success bool, details map[string]any) {
	if r.nats == nil {
		return
	}

	event := topics.StartupEvent{
		EventType:     eventType,
		Component:     component,
		ComponentType: componentType,
		Message:       message,
		Details:       details,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Success:       success,
	}

	subject := r.topics.SystemStartup()
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
	r.publishStartupEvent(topics.EventRobotShutdown, "", "", "Robot shutting down", true, nil)

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

	// Stop all registry components
	r.componentsMu.Lock()
	for name, comp := range r.components {
		r.logger.Info("Stopping component", "name", name)
		if closeable, ok := comp.(Closeable); ok {
			if err := closeable.Close(ctx); err != nil {
				r.logger.Warn("Error closing component", "name", name, "error", err)
			}
		}
	}
	r.components = make(map[string]any)
	r.componentsMu.Unlock()

	// Stop all cameras
	r.camerasMu.Lock()
	for name, cam := range r.cameras {
		r.logger.Info("Stopping camera", "name", name)
		if err := cam.Close(); err != nil {
			r.logger.Warn("Error closing camera", "name", name, "error", err)
		}
	}
	r.cameras = make(map[string]*v4l2.Camera)
	r.camerasMu.Unlock()

	// Cancel internal context
	r.cancel()

	// Close NATS connection
	if r.nats != nil {
		r.nats.Close()
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

// Topics returns the topic builder for this robot.
func (r *Robot) Topics() *topics.Builder {
	return r.topics
}
