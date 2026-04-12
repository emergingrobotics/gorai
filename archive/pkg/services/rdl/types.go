// Package rdl provides types and functions for parsing and validating
// service RDL (Robot Definition Language) files.
package rdl

import "encoding/json"

// ServiceRDL represents the complete service.rdl.json structure
type ServiceRDL struct {
	SchemaVersion string  `json:"schema_version" yaml:"schema_version"`
	Service       Service `json:"service" yaml:"service"`
}

// Service represents the main service metadata
type Service struct {
	Name          string          `json:"name" yaml:"name"`
	Type          string          `json:"type" yaml:"type"`
	Model         string          `json:"model" yaml:"model"`
	Version       string          `json:"version" yaml:"version"`
	Repository    string          `json:"repository,omitempty" yaml:"repository,omitempty"`
	Description   string          `json:"description,omitempty" yaml:"description,omitempty"`
	Author        string          `json:"author,omitempty" yaml:"author,omitempty"`
	License       string          `json:"license,omitempty" yaml:"license,omitempty"`
	Homepage      string          `json:"homepage,omitempty" yaml:"homepage,omitempty"`
	Documentation string          `json:"documentation,omitempty" yaml:"documentation,omitempty"`
	Compatibility *Compatibility  `json:"compatibility,omitempty" yaml:"compatibility,omitempty"`
	Container     *Container      `json:"container,omitempty" yaml:"container,omitempty"`
	NATSTopics    *NATSTopics     `json:"nats_topics,omitempty" yaml:"nats_topics,omitempty"`
	Configuration map[string]*ConfigAttr `json:"configuration,omitempty" yaml:"configuration,omitempty"`
	Performance   *Performance    `json:"performance,omitempty" yaml:"performance,omitempty"`
	Keywords      []string        `json:"keywords,omitempty" yaml:"keywords,omitempty"`
	Examples      []Example       `json:"examples,omitempty" yaml:"examples,omitempty"`
	Quality       *Quality        `json:"quality,omitempty" yaml:"quality,omitempty"`
	Metadata      *Metadata       `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// Compatibility describes version and platform constraints
type Compatibility struct {
	GoraiVersion string   `json:"gorai_version,omitempty" yaml:"gorai_version,omitempty"`
	Platforms    []string `json:"platforms,omitempty" yaml:"platforms,omitempty"`
	Accelerators []string `json:"accelerators,omitempty" yaml:"accelerators,omitempty"`
}

// Container describes container deployment configuration
type Container struct {
	DefaultImage        string                    `json:"default_image" yaml:"default_image"`
	ImageVariants       map[string]*ImageVariant  `json:"image_variants,omitempty" yaml:"image_variants,omitempty"`
	PullPolicy          string                    `json:"pull_policy,omitempty" yaml:"pull_policy,omitempty"`
	ResourceRequirements *ResourceRequirements    `json:"resource_requirements,omitempty" yaml:"resource_requirements,omitempty"`
	ResourceLimits      *ResourceRequirements     `json:"resource_limits,omitempty" yaml:"resource_limits,omitempty"`
	HealthCheck         *HealthCheck              `json:"health_check,omitempty" yaml:"health_check,omitempty"`
	Volumes             []Volume                  `json:"volumes,omitempty" yaml:"volumes,omitempty"`
	Environment         map[string]string         `json:"environment,omitempty" yaml:"environment,omitempty"`
}

// ImageVariant describes a platform or accelerator-specific container image
type ImageVariant struct {
	Image       string `json:"image" yaml:"image"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Accelerator string `json:"accelerator,omitempty" yaml:"accelerator,omitempty"`
	Platform    string `json:"platform,omitempty" yaml:"platform,omitempty"`
}

// ResourceRequirements describes resource requirements/limits
type ResourceRequirements struct {
	Memory string `json:"memory,omitempty" yaml:"memory,omitempty"`
	CPU    string `json:"cpu,omitempty" yaml:"cpu,omitempty"`
	GPU    string `json:"gpu,omitempty" yaml:"gpu,omitempty"`
}

// HealthCheck describes how to check service health
type HealthCheck struct {
	Type     string `json:"type" yaml:"type"`
	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Interval string `json:"interval,omitempty" yaml:"interval,omitempty"`
	Timeout  string `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Retries  int    `json:"retries,omitempty" yaml:"retries,omitempty"`
}

// Volume describes a volume mount
type Volume struct {
	Name        string `json:"name" yaml:"name"`
	MountPath   string `json:"mount_path" yaml:"mount_path"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
}

// NATSTopics describes NATS message bus topics
type NATSTopics struct {
	Subscribes []Topic `json:"subscribes" yaml:"subscribes"`
	Publishes  []Topic `json:"publishes" yaml:"publishes"`
}

// Topic represents a NATS topic subscription or publication
type Topic struct {
	Pattern     string `json:"pattern" yaml:"pattern"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	MessageType string `json:"message_type,omitempty" yaml:"message_type,omitempty"`
	QoS         string `json:"qos,omitempty" yaml:"qos,omitempty"`
	Frequency   string `json:"frequency,omitempty" yaml:"frequency,omitempty"`
}

// UnmarshalJSON handles both string and object forms of Topic
func (t *Topic) UnmarshalJSON(data []byte) error {
	// Try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		t.Pattern = s
		return nil
	}

	// Try object form
	type topicAlias Topic
	var ta topicAlias
	if err := json.Unmarshal(data, &ta); err != nil {
		return err
	}
	*t = Topic(ta)
	return nil
}

// ConfigAttr describes a configuration attribute
type ConfigAttr struct {
	Type        string        `json:"type" yaml:"type"`
	Required    bool          `json:"required,omitempty" yaml:"required,omitempty"`
	Default     interface{}   `json:"default,omitempty" yaml:"default,omitempty"`
	Description string        `json:"description,omitempty" yaml:"description,omitempty"`
	Example     interface{}   `json:"example,omitempty" yaml:"example,omitempty"`
	Enum        []interface{} `json:"enum,omitempty" yaml:"enum,omitempty"`
	Range       []float64     `json:"range,omitempty" yaml:"range,omitempty"`
	EnvVar      string        `json:"env_var,omitempty" yaml:"env_var,omitempty"`
}

// Performance describes performance characteristics
type Performance struct {
	Latency     *Latency `json:"latency,omitempty" yaml:"latency,omitempty"`
	Throughput  string   `json:"throughput,omitempty" yaml:"throughput,omitempty"`
	StartupTime string   `json:"startup_time,omitempty" yaml:"startup_time,omitempty"`
}

// Latency describes latency characteristics
type Latency struct {
	Typical string `json:"typical,omitempty" yaml:"typical,omitempty"`
	Max     string `json:"max,omitempty" yaml:"max,omitempty"`
}

// Example provides a usage example
type Example struct {
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	Config      map[string]interface{} `json:"config" yaml:"config"`
}

// Quality contains quality metrics
type Quality struct {
	CIStatus      string `json:"ci_status,omitempty" yaml:"ci_status,omitempty"`
	TestCoverage  int    `json:"test_coverage,omitempty" yaml:"test_coverage,omitempty"`
	HasBenchmarks bool   `json:"has_benchmarks,omitempty" yaml:"has_benchmarks,omitempty"`
}

// Metadata contains additional metadata
type Metadata struct {
	FirstReleased string `json:"first_released,omitempty" yaml:"first_released,omitempty"`
	LastUpdated   string `json:"last_updated,omitempty" yaml:"last_updated,omitempty"`
	Maturity      string `json:"maturity,omitempty" yaml:"maturity,omitempty"`
}
