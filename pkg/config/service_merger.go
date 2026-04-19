package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// ServiceMerger handles merging Service RDL with robot service configuration.
type ServiceMerger struct {
	robotConfig *RDL
	configDir   string
}

// NewServiceMerger creates a new ServiceMerger.
func NewServiceMerger(robotConfig *RDL, configDir string) *ServiceMerger {
	return &ServiceMerger{
		robotConfig: robotConfig,
		configDir:   configDir,
	}
}

// LoadAndMergeServices loads Service RDL files and merges them with service configs.
// This modifies the services in place.
func (m *ServiceMerger) LoadAndMergeServices() error {
	var errs []string

	for i := range m.robotConfig.Services {
		svc := &m.robotConfig.Services[i]

		if !svc.HasServiceRDL() {
			continue
		}

		// Load Service RDL
		rdl, err := LoadServiceRDLRelative(svc.RDL, m.configDir)
		if err != nil {
			errs = append(errs, fmt.Sprintf("service %q: failed to load RDL %q: %v", svc.Name, svc.RDL, err))
			continue
		}

		// Merge configuration
		if err := m.mergeService(svc, rdl); err != nil {
			errs = append(errs, fmt.Sprintf("service %q: %v", svc.Name, err))
			continue
		}

		// Store the loaded RDL
		svc.SetServiceRDL(rdl)
	}

	if len(errs) > 0 {
		return fmt.Errorf("service RDL loading errors:\n  %s", strings.Join(errs, "\n  "))
	}

	return nil
}

// mergeService merges a Service RDL into a ServiceConfig.
func (m *ServiceMerger) mergeService(svc *ServiceConfig, rdl *ServiceRDL) error {
	// Merge type and model (Service RDL provides defaults, robot RDL overrides)
	if svc.Type == "" {
		svc.Type = rdl.Service.Type
	}
	if svc.Model == "" {
		svc.Model = rdl.Service.Model
	}

	// Merge attributes (Service RDL provides defaults, robot RDL overrides)
	mergedAttrs := rdl.MergeAttributes(svc.Attributes)
	svc.Attributes = mergedAttrs

	// Validate attributes against Service RDL definitions
	if err := rdl.ValidateAttributes(svc.Attributes); err != nil {
		return fmt.Errorf("attribute validation: %w", err)
	}

	// Resolve subject patterns
	resolver := NewSubjectResolverFromConfig(m.robotConfig, svc.Name, svc.Attributes)
	resolved, err := resolver.ResolveAll(&rdl.Subjects)
	if err != nil {
		return fmt.Errorf("subject resolution: %w", err)
	}
	svc.SetResolvedSubjects(resolved)

	// Merge external/runtime configuration
	if err := m.mergeRuntimeConfig(svc, rdl); err != nil {
		return fmt.Errorf("runtime config: %w", err)
	}

	return nil
}

// mergeRuntimeConfig merges Service RDL runtime config with external config.
func (m *ServiceMerger) mergeRuntimeConfig(svc *ServiceConfig, rdl *ServiceRDL) error {
	if rdl.Runtime == nil {
		return nil
	}

	// Initialize external config if not present
	if svc.External == nil {
		svc.External = &ExternalConfig{}
	}

	// If Service RDL has container config and service doesn't have explicit container
	if rdl.Runtime.Container != nil && svc.External.Container == nil {
		svc.External.Container = &ContainerServiceConfig{}
	}

	// Merge container configuration
	if rdl.Runtime.Container != nil && svc.External.Container != nil {
		rdlContainer := rdl.Runtime.Container
		svcContainer := svc.External.Container

		// Image (with variable substitution)
		if svcContainer.Image == "" && rdlContainer.Image != "" {
			// Resolve variables in image name
			resolver := NewSubjectResolverFromConfig(m.robotConfig, svc.Name, svc.Attributes)
			resolved, err := resolver.Resolve(rdlContainer.Image)
			if err != nil {
				return fmt.Errorf("resolving image name: %w", err)
			}
			svcContainer.Image = resolved
		}

		// Build configuration
		if svcContainer.Build == nil && rdlContainer.Build != nil {
			// Get the directory containing the Service RDL file
			rdlDir := filepath.Dir(filepath.Join(m.configDir, svc.RDL))

			context := rdlContainer.Build.Context
			if context != "" && !filepath.IsAbs(context) {
				context = filepath.Join(rdlDir, context)
			}

			svcContainer.Build = &ContainerBuildConfig{
				Context:       context,
				Containerfile: rdlContainer.Build.Containerfile,
				Args:          rdlContainer.Build.Args,
			}
		}

		// Network (Service RDL provides default, robot RDL overrides)
		if svcContainer.Network == "" && rdlContainer.Network != "" {
			svcContainer.Network = rdlContainer.Network
		}

		// Environment variables (merge, robot RDL overrides)
		if rdlContainer.Environment != nil {
			if svcContainer.Environment == nil {
				svcContainer.Environment = make(map[string]string)
			}
			for k, v := range rdlContainer.Environment {
				if _, exists := svcContainer.Environment[k]; !exists {
					svcContainer.Environment[k] = v
				}
			}
		}
	}

	// Merge command (Service RDL provides default)
	if svc.External.Command == "" && rdl.Runtime.Command != "" {
		svc.External.Command = rdl.Runtime.Command
	}

	// Merge environment variables for non-container services
	if rdl.Runtime.Env != nil {
		if svc.External.Env == nil {
			svc.External.Env = make(map[string]string)
		}
		for k, v := range rdl.Runtime.Env {
			if _, exists := svc.External.Env[k]; !exists {
				svc.External.Env[k] = v
			}
		}
	}

	return nil
}

// GetResolvedEnvironment returns the environment variables for an external service,
// including resolved topic information.
func GetResolvedEnvironment(cfg *RDL, svc *ServiceConfig) map[string]string {
	env := make(map[string]string)

	// Add base Gorai environment
	env["GORAI_ROBOT_NAME"] = cfg.Robot.Name
	env["GORAI_SERVICE_NAME"] = svc.Name
	env["GORAI_NAMESPACE"] = cfg.GetEffectiveNamespace()

	// Add per-service log level (defaults to "error" if not specified)
	env["LOG_LEVEL"] = strings.ToUpper(svc.GetLogLevel())

	// Add NATS URL
	if cfg.NATS != nil {
		if cfg.NATS.URL != "" {
			env["NATS_URL"] = cfg.NATS.URL
		} else if len(cfg.NATS.URLs) > 0 {
			env["NATS_URL"] = cfg.NATS.URLs[0]
		}
	}

	// Add resolved subjects if available
	if subjects := svc.GetResolvedSubjects(); subjects != nil {
		// Input subjects as JSON
		if len(subjects.Subscribe) > 0 {
			if data, err := json.Marshal(subjects.Subscribe); err == nil {
				env["GORAI_INPUT_SUBJECTS"] = string(data)
			}
		}

		// Output subjects as JSON
		if len(subjects.Publish) > 0 {
			if data, err := json.Marshal(subjects.Publish); err == nil {
				env["GORAI_OUTPUT_SUBJECTS"] = string(data)
			}
		}

		// Also add individual subjects as environment variables for convenience
		for name, subject := range subjects.Subscribe {
			envKey := fmt.Sprintf("INPUT_SUBJECT_%s", strings.ToUpper(name))
			env[envKey] = subject
		}
		for name, subject := range subjects.Publish {
			envKey := fmt.Sprintf("OUTPUT_SUBJECT_%s", strings.ToUpper(name))
			env[envKey] = subject
		}
	}

	// Add service attributes as environment variables (string values only)
	for key, value := range svc.Attributes {
		if s, ok := value.(string); ok {
			envKey := strings.ToUpper(key)
			env[envKey] = s
		}
	}

	// Merge explicit environment from external config
	if svc.External != nil && svc.External.Env != nil {
		for k, v := range svc.External.Env {
			env[k] = v
		}
	}

	// Merge container environment if applicable
	if svc.External != nil && svc.External.Container != nil && svc.External.Container.Environment != nil {
		for k, v := range svc.External.Container.Environment {
			env[k] = v
		}
	}

	return env
}

// LoadWithServiceRDL loads a robot configuration and all referenced Service RDL files.
func LoadWithServiceRDL(path string) (*RDL, error) {
	// Load the base config
	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}

	// Get the config directory for relative path resolution
	configDir := filepath.Dir(path)
	if !filepath.IsAbs(configDir) {
		if absPath, err := filepath.Abs(path); err == nil {
			configDir = filepath.Dir(absPath)
		}
	}

	// Load and merge Service RDL files
	merger := NewServiceMerger(cfg, configDir)
	if err := merger.LoadAndMergeServices(); err != nil {
		return nil, err
	}

	return cfg, nil
}
