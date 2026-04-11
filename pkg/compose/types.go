// Package compose transforms RDL configuration into process-compose YAML.
package compose

// ProcessCompose represents the top-level process-compose configuration.
type ProcessCompose struct {
	Version         string               `yaml:"version"`
	OrderedShutdown bool                 `yaml:"ordered_shutdown,omitempty"`
	Environment     []string             `yaml:"environment,omitempty"`
	LogConfiguration *LogConfiguration   `yaml:"log_configuration,omitempty"`
	Processes       map[string]*Process  `yaml:"processes"`
}

// Process represents a single process in the process-compose configuration.
type Process struct {
	Command        string                `yaml:"command"`
	Namespace      string                `yaml:"namespace,omitempty"`
	Description    string                `yaml:"description,omitempty"`
	Environment    []string              `yaml:"environment,omitempty"`
	DependsOn      map[string]*DependsOn `yaml:"depends_on,omitempty"`
	ReadinessProbe *Probe                `yaml:"readiness_probe,omitempty"`
	Availability   *Availability         `yaml:"availability,omitempty"`
	Shutdown       *Shutdown             `yaml:"shutdown,omitempty"`
	LogLocation    string                `yaml:"log_location,omitempty"`
	LogRotation    *LogRotation          `yaml:"log_rotation,omitempty"`
}

// DependsOn specifies a dependency condition for a process.
type DependsOn struct {
	Condition string `yaml:"condition"`
}

// Probe defines a readiness or liveness probe for a process.
type Probe struct {
	Exec                *ExecProbe `yaml:"exec,omitempty"`
	HTTPGet             *HTTPProbe `yaml:"http_get,omitempty"`
	InitialDelaySeconds int        `yaml:"initial_delay_seconds,omitempty"`
	PeriodSeconds       int        `yaml:"period_seconds,omitempty"`
	FailureThreshold    int        `yaml:"failure_threshold,omitempty"`
}

// ExecProbe runs a command to check process health.
type ExecProbe struct {
	Command string `yaml:"command"`
}

// HTTPProbe checks process health via an HTTP GET request.
type HTTPProbe struct {
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	Path   string `yaml:"path"`
	Scheme string `yaml:"scheme,omitempty"`
}

// Availability controls process restart behavior.
type Availability struct {
	Restart        string `yaml:"restart"`
	BackoffSeconds int    `yaml:"backoff_seconds,omitempty"`
}

// Shutdown defines how a process should be stopped.
type Shutdown struct {
	Command        string `yaml:"command,omitempty"`
	Signal         int    `yaml:"signal,omitempty"`
	TimeoutSeconds int    `yaml:"timeout_seconds,omitempty"`
}

// LogConfiguration controls process-compose log output formatting.
type LogConfiguration struct {
	FieldsOrder []string `yaml:"fields_order,omitempty"`
	DisableJSON bool     `yaml:"disable_json"`
	NoMetadata  bool     `yaml:"no_metadata"`
}

// LogRotation controls per-process log file rotation.
type LogRotation struct {
	MaxSizeMB  int  `yaml:"max_size_mb,omitempty"`
	MaxBackups int  `yaml:"max_backups,omitempty"`
	MaxAgeDays int  `yaml:"max_age_days,omitempty"`
	Compress   bool `yaml:"compress,omitempty"`
}
