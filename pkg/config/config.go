// Package config provides configuration loading and management for Gorai robots.
// It implements the Robot Definition Language (RDL) specification.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// RDL represents the complete Robot Definition Language configuration.
type RDL struct {
	Schema     string                       `json:"$schema,omitempty"`
	Version    string                       `json:"version"`
	Robot      RobotConfig                  `json:"robot"`
	NATS       *NATSConfig                  `json:"nats,omitempty"`
	Containers map[string]*ContainerConfig  `json:"containers,omitempty"`
	Networks   map[string]*NetworkConfig    `json:"networks,omitempty"`
	Volumes    map[string]*VolumeConfig     `json:"volumes,omitempty"`
	Components []ComponentConfig            `json:"components,omitempty"`
	Services   []ServiceConfig              `json:"services,omitempty"`
	Remotes    []RemoteConfig               `json:"remotes,omitempty"`
	Log        *LogConfig                   `json:"log,omitempty"`
	Dashboard  *DashboardConfig             `json:"dashboard,omitempty"`
}

// RobotConfig defines the robot's identity.
type RobotConfig struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace,omitempty"`
	Description string `json:"description,omitempty"`
}

// NATSConfig defines the NATS connection configuration.
type NATSConfig struct {
	URL             string     `json:"url,omitempty"`
	URLs            []string   `json:"urls,omitempty"`
	JetStream       bool       `json:"jetstream,omitempty"`
	CredentialsFile string     `json:"credentials_file,omitempty"`
	TLS             *TLSConfig `json:"tls,omitempty"`
	ConnectTimeout  string     `json:"connect_timeout,omitempty"`
	ReconnectWait   string     `json:"reconnect_wait,omitempty"`
	MaxReconnects   int        `json:"max_reconnects,omitempty"`

	// Deprecated: Container field is no longer used in RDL v2
	Container string `json:"container,omitempty"`
}

// TLSConfig defines TLS settings for NATS.
type TLSConfig struct {
	CAFile   string `json:"ca_file,omitempty"`
	CertFile string `json:"cert_file,omitempty"`
	KeyFile  string `json:"key_file,omitempty"`
}

// ComponentConfig represents a component configuration.
type ComponentConfig struct {
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Model      string         `json:"model"`
	Disabled   bool           `json:"disabled,omitempty"`
	Attributes map[string]any `json:"attributes,omitempty"`
	DependsOn  []string       `json:"depends_on,omitempty"`

	// Deprecated: Container field is no longer used in RDL v2
	Container string `json:"container,omitempty"`
}

// ServiceConfig represents a service configuration.
type ServiceConfig struct {
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Model      string          `json:"model"`
	Disabled   bool            `json:"disabled,omitempty"`
	External   *ExternalConfig `json:"external,omitempty"`
	Attributes map[string]any  `json:"attributes,omitempty"`
	DependsOn  []string        `json:"depends_on,omitempty"`

	// Deprecated: Container field is no longer used in RDL v2
	Container string `json:"container,omitempty"`
}

// ExternalConfig configures a service to run as an external process.
type ExternalConfig struct {
	Enabled bool              `json:"enabled,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Managed bool              `json:"managed,omitempty"`
	Restart string            `json:"restart,omitempty"` // "always", "on-failure", "never"
	Env     map[string]string `json:"env,omitempty"`
}

// IsExternal returns true if this service should run as an external process.
func (s *ServiceConfig) IsExternal() bool {
	return s.External != nil && s.External.Enabled
}

// IsManaged returns true if this external service should be managed by the robot.
func (s *ServiceConfig) IsManaged() bool {
	if !s.IsExternal() {
		return false
	}
	// Default to managed if not specified
	return s.External.Managed || s.External.Command != ""
}

// RemoteConfig defines a connection to a remote robot/node.
type RemoteConfig struct {
	Name       string   `json:"name"`
	Address    string   `json:"address"`
	Namespace  string   `json:"namespace,omitempty"`
	Components []string `json:"components,omitempty"`
	Services   []string `json:"services,omitempty"`
}

// LogConfig defines logging configuration.
type LogConfig struct {
	Level      string `json:"level,omitempty"`
	Format     string `json:"format,omitempty"`
	Output     string `json:"output,omitempty"`
	File       string `json:"file,omitempty"`
	MaxSizeMB  int    `json:"max_size_mb,omitempty"`
	MaxBackups int    `json:"max_backups,omitempty"`
	MaxAgeDays int    `json:"max_age_days,omitempty"`
}

// DashboardConfig defines the web dashboard configuration.
type DashboardConfig struct {
	Enabled   *bool            `json:"enabled,omitempty"`
	Listen    string           `json:"listen,omitempty"`
	WebSocket *WebSocketConfig `json:"websocket,omitempty"`
	Video     *VideoConfig     `json:"video,omitempty"`
}

// WebSocketConfig defines WebSocket settings for the dashboard.
type WebSocketConfig struct {
	BufferSize         int `json:"buffer_size,omitempty"`
	SensorDownsampleHz int `json:"sensor_downsample_hz,omitempty"`
}

// VideoConfig defines video streaming settings.
type VideoConfig struct {
	Enabled bool   `json:"enabled,omitempty"`
	Format  string `json:"format,omitempty"`
	MaxFPS  int    `json:"max_fps,omitempty"`
	Quality int    `json:"quality,omitempty"`
}

// ContainerConfig defines a container for Quadlet/systemd orchestration.
type ContainerConfig struct {
	// Image specification (one of: Image, Build)
	Image string       `json:"image,omitempty"`
	Build *BuildConfig `json:"build,omitempty"`

	// Dependencies
	DependsOn map[string]*DependsOnCondition `json:"depends_on,omitempty"`

	// Runtime configuration
	Environment map[string]string `json:"environment,omitempty"`
	EnvFile     []string          `json:"env_file,omitempty"`
	Command     []string          `json:"command,omitempty"`
	Entrypoint  []string          `json:"entrypoint,omitempty"`

	// Storage
	Volumes []string `json:"volumes,omitempty"`

	// Devices (for hardware access)
	Devices []string `json:"devices,omitempty"`

	// Networking
	Ports       []string `json:"ports,omitempty"`
	NetworkMode string   `json:"network_mode,omitempty"`
	Networks    []string `json:"networks,omitempty"`

	// Security
	Privileged  bool     `json:"privileged,omitempty"`
	SecurityOpt []string `json:"security_opt,omitempty"`
	CapAdd      []string `json:"cap_add,omitempty"`
	CapDrop     []string `json:"cap_drop,omitempty"`
	GroupAdd    []string `json:"group_add,omitempty"`

	// Resource limits
	Resources *ResourceConfig `json:"resources,omitempty"`

	// Lifecycle
	Restart         string `json:"restart,omitempty"`
	StopGracePeriod string `json:"stop_grace_period,omitempty"`

	// Health checking
	Healthcheck *HealthcheckConfig `json:"healthcheck,omitempty"`

	// Gorai-specific: which components/services run here
	ComponentNames []string `json:"components,omitempty"`
	ServiceNames   []string `json:"services,omitempty"`

	// Quadlet-specific options (optional)
	AutoUpdate   string `json:"auto_update,omitempty"`   // "registry" | "local" | "" (default: registry)
	Notify       string `json:"notify,omitempty"`        // "true" | "healthy" | "" (wait for health check)
	TimeoutStart string `json:"timeout_start,omitempty"` // Startup timeout (e.g., "300" for 5 min)
}

// BuildConfig defines container build settings.
type BuildConfig struct {
	Context    string            `json:"context,omitempty"`
	Dockerfile string            `json:"dockerfile,omitempty"`
	Args       map[string]string `json:"args,omitempty"`
	Target     string            `json:"target,omitempty"`
}

// DependsOnCondition defines a dependency condition.
type DependsOnCondition struct {
	Condition string `json:"condition,omitempty"` // service_started, service_healthy, service_completed_successfully
}

// HealthcheckConfig defines container health check settings.
type HealthcheckConfig struct {
	Test        []string `json:"test,omitempty"`
	Interval    string   `json:"interval,omitempty"`
	Timeout     string   `json:"timeout,omitempty"`
	Retries     int      `json:"retries,omitempty"`
	StartPeriod string   `json:"start_period,omitempty"`
}

// ResourceConfig defines container resource limits.
type ResourceConfig struct {
	Limits       *ResourceLimits `json:"limits,omitempty"`
	Reservations *ResourceLimits `json:"reservations,omitempty"`
}

// ResourceLimits defines CPU and memory limits.
type ResourceLimits struct {
	CPUs   string `json:"cpus,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// NetworkConfig defines a custom network.
type NetworkConfig struct {
	Driver   string            `json:"driver,omitempty"`
	Internal bool              `json:"internal,omitempty"`
	Options  map[string]string `json:"driver_opts,omitempty"`
}

// VolumeConfig defines a named volume.
type VolumeConfig struct {
	Driver  string            `json:"driver,omitempty"`
	Options map[string]string `json:"driver_opts,omitempty"`
}

// Load loads configuration from a JSON file.
func Load(path string) (*RDL, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	return LoadFromBytes(data)
}

// LoadFromBytes loads configuration from JSON bytes.
func LoadFromBytes(data []byte) (*RDL, error) {
	// Expand environment variables
	data = expandEnvVars(data)

	var cfg RDL
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply defaults
	cfg.applyDefaults()

	return &cfg, nil
}

// expandEnvVars expands environment variable references in JSON.
// Supports ${VAR} and ${VAR:-default} syntax.
func expandEnvVars(data []byte) []byte {
	// Pattern: ${VAR} or ${VAR:-default}
	re := regexp.MustCompile(`\$\{([^}:]+)(?::-([^}]*))?\}`)
	return re.ReplaceAllFunc(data, func(match []byte) []byte {
		parts := re.FindSubmatch(match)
		varName := string(parts[1])
		defaultVal := ""
		if len(parts) > 2 {
			defaultVal = string(parts[2])
		}

		if val := os.Getenv(varName); val != "" {
			return []byte(val)
		}
		return []byte(defaultVal)
	})
}

// applyDefaults applies default values to the configuration.
func (cfg *RDL) applyDefaults() {
	// Default namespace to robot name
	if cfg.Robot.Namespace == "" {
		cfg.Robot.Namespace = cfg.Robot.Name
	}

	// Default NATS URL
	if cfg.NATS == nil {
		cfg.NATS = &NATSConfig{
			URL: "nats://localhost:4222",
		}
	} else if cfg.NATS.URL == "" && len(cfg.NATS.URLs) == 0 {
		cfg.NATS.URL = "nats://localhost:4222"
	}

	// Default log settings
	if cfg.Log == nil {
		cfg.Log = &LogConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		}
	} else {
		if cfg.Log.Level == "" {
			cfg.Log.Level = "info"
		}
		if cfg.Log.Format == "" {
			cfg.Log.Format = "text"
		}
		if cfg.Log.Output == "" {
			cfg.Log.Output = "stdout"
		}
	}

	// Default dashboard settings (enabled by default)
	if cfg.Dashboard == nil {
		enabled := true
		cfg.Dashboard = &DashboardConfig{
			Enabled: &enabled,
			Listen:  ":8080",
		}
	} else {
		if cfg.Dashboard.Enabled == nil {
			enabled := true
			cfg.Dashboard.Enabled = &enabled
		}
		if cfg.Dashboard.Listen == "" {
			cfg.Dashboard.Listen = ":8080"
		}
	}
}

// Validate validates the configuration against RDL rules.
func (cfg *RDL) Validate() error {
	var errs []string

	// Validate version
	if cfg.Version == "" {
		errs = append(errs, "version is required")
	} else if cfg.Version != "1" && cfg.Version != "2" {
		errs = append(errs, fmt.Sprintf("unsupported version %q, expected \"1\" or \"2\"", cfg.Version))
	}

	// Validate robot name
	if err := validateName(cfg.Robot.Name); err != nil {
		errs = append(errs, fmt.Sprintf("robot.name: %v", err))
	}

	// Validate components
	names := make(map[string]bool)
	for i, comp := range cfg.Components {
		if err := validateName(comp.Name); err != nil {
			errs = append(errs, fmt.Sprintf("components[%d].name: %v", i, err))
		}
		if names[comp.Name] {
			errs = append(errs, fmt.Sprintf("components[%d].name: duplicate name %q", i, comp.Name))
		}
		names[comp.Name] = true

		if comp.Type == "" {
			errs = append(errs, fmt.Sprintf("components[%d].type: required", i))
		}
		if comp.Model == "" {
			errs = append(errs, fmt.Sprintf("components[%d].model: required", i))
		}
	}

	// Validate services
	for i, svc := range cfg.Services {
		if err := validateName(svc.Name); err != nil {
			errs = append(errs, fmt.Sprintf("services[%d].name: %v", i, err))
		}
		if names[svc.Name] {
			errs = append(errs, fmt.Sprintf("services[%d].name: duplicate name %q", i, svc.Name))
		}
		names[svc.Name] = true

		if svc.Type == "" {
			errs = append(errs, fmt.Sprintf("services[%d].type: required", i))
		}
		if svc.Model == "" {
			errs = append(errs, fmt.Sprintf("services[%d].model: required", i))
		}
	}

	// Validate dependencies exist
	for i, comp := range cfg.Components {
		for _, dep := range comp.DependsOn {
			if !names[dep] {
				errs = append(errs, fmt.Sprintf("components[%d].depends_on: unknown dependency %q", i, dep))
			}
		}
	}
	for i, svc := range cfg.Services {
		for _, dep := range svc.DependsOn {
			if !names[dep] {
				errs = append(errs, fmt.Sprintf("services[%d].depends_on: unknown dependency %q", i, dep))
			}
		}
	}

	// Check for circular dependencies
	if err := cfg.checkCircularDependencies(); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation errors:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

// validateName checks if a name is valid per RDL spec.
func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > 63 {
		return fmt.Errorf("name too long (max 63 characters)")
	}

	// Must start with letter
	if !regexp.MustCompile(`^[a-zA-Z]`).MatchString(name) {
		return fmt.Errorf("name must start with a letter")
	}

	// Only letters, numbers, hyphens, underscores
	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`).MatchString(name) {
		return fmt.Errorf("name may only contain letters, numbers, hyphens, and underscores")
	}

	return nil
}

// checkCircularDependencies detects circular dependencies using DFS.
func (cfg *RDL) checkCircularDependencies() error {
	// Build dependency graph
	deps := make(map[string][]string)
	for _, comp := range cfg.Components {
		deps[comp.Name] = comp.DependsOn
	}
	for _, svc := range cfg.Services {
		deps[svc.Name] = svc.DependsOn
	}

	// DFS state
	const (
		white = iota // unvisited
		gray         // visiting
		black        // visited
	)
	color := make(map[string]int)
	var path []string

	var visit func(node string) error
	visit = func(node string) error {
		color[node] = gray
		path = append(path, node)

		for _, dep := range deps[node] {
			switch color[dep] {
			case gray:
				// Found cycle - find where it starts
				cycleStart := 0
				for i, n := range path {
					if n == dep {
						cycleStart = i
						break
					}
				}
				cycle := append(path[cycleStart:], dep)
				return fmt.Errorf("circular dependency detected: %s", strings.Join(cycle, " → "))
			case white:
				if err := visit(dep); err != nil {
					return err
				}
			}
		}

		color[node] = black
		path = path[:len(path)-1]
		return nil
	}

	// Visit all nodes
	for name := range deps {
		if color[name] == white {
			if err := visit(name); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetEffectiveNamespace returns the effective namespace for the robot.
func (cfg *RDL) GetEffectiveNamespace() string {
	if cfg.Robot.Namespace != "" {
		return cfg.Robot.Namespace
	}
	return cfg.Robot.Name
}

// IsDashboardEnabled returns whether the dashboard should be enabled.
func (cfg *RDL) IsDashboardEnabled() bool {
	if cfg.Dashboard == nil || cfg.Dashboard.Enabled == nil {
		return true // Default to enabled
	}
	return *cfg.Dashboard.Enabled
}

// ToJSON serializes the configuration to JSON.
func (cfg *RDL) ToJSON(indent bool) ([]byte, error) {
	if indent {
		return json.MarshalIndent(cfg, "", "  ")
	}
	return json.Marshal(cfg)
}

// DeprecationWarnings returns a list of deprecation warnings for v1 features.
func (cfg *RDL) DeprecationWarnings() []string {
	var warnings []string

	// Check for containers section
	if cfg.Containers != nil && len(cfg.Containers) > 0 {
		warnings = append(warnings, "The 'containers' section is deprecated in RDL v2 and will be ignored. "+
			"Use 'external' on services for processes that need to run separately.")
	}

	// Check for container field in NATS config
	if cfg.NATS != nil && cfg.NATS.Container != "" {
		warnings = append(warnings, "nats.container is deprecated in RDL v2. "+
			"NATS should run as a native systemd service, not in a container.")
	}

	// Check for container field in components
	for _, comp := range cfg.Components {
		if comp.Container != "" {
			warnings = append(warnings, fmt.Sprintf("components[%s].container is deprecated in RDL v2. "+
				"All components run in the main robot process.", comp.Name))
		}
	}

	// Check for container field in services
	for _, svc := range cfg.Services {
		if svc.Container != "" {
			warnings = append(warnings, fmt.Sprintf("services[%s].container is deprecated in RDL v2. "+
				"Use 'external' instead for services that need to run separately.", svc.Name))
		}
	}

	return warnings
}

// HasDeprecatedFields returns true if the config uses any deprecated v1 fields.
func (cfg *RDL) HasDeprecatedFields() bool {
	return len(cfg.DeprecationWarnings()) > 0
}

// GetExternalServices returns all services configured to run as external processes.
func (cfg *RDL) GetExternalServices() []ServiceConfig {
	var external []ServiceConfig
	for _, svc := range cfg.Services {
		if svc.IsExternal() {
			external = append(external, svc)
		}
	}
	return external
}

// GetInternalServices returns all services configured to run in the main process.
func (cfg *RDL) GetInternalServices() []ServiceConfig {
	var internal []ServiceConfig
	for _, svc := range cfg.Services {
		if !svc.IsExternal() {
			internal = append(internal, svc)
		}
	}
	return internal
}
