// Package compose generates podman-compose.json files from RDL configuration.
package compose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gorai/gorai/pkg/config"
)

// ComposeFile represents a podman-compose.json file structure.
type ComposeFile struct {
	Version  string                     `json:"version,omitempty"`
	Services map[string]*ComposeService `json:"services,omitempty"`
	Networks map[string]*ComposeNetwork `json:"networks,omitempty"`
	Volumes  map[string]*ComposeVolume  `json:"volumes,omitempty"`
}

// ComposeService represents a service in podman-compose.json.
type ComposeService struct {
	Image           string                       `json:"image,omitempty"`
	Build           *ComposeBuild                `json:"build,omitempty"`
	ContainerName   string                       `json:"container_name,omitempty"`
	DependsOn       map[string]*ComposeDependsOn `json:"depends_on,omitempty"`
	Environment     map[string]string            `json:"environment,omitempty"`
	EnvFile         []string                     `json:"env_file,omitempty"`
	Command         []string                     `json:"command,omitempty"`
	Entrypoint      []string                     `json:"entrypoint,omitempty"`
	Volumes         []string                     `json:"volumes,omitempty"`
	Devices         []string                     `json:"devices,omitempty"`
	Ports           []string                     `json:"ports,omitempty"`
	NetworkMode     string                       `json:"network_mode,omitempty"`
	Networks        []string                     `json:"networks,omitempty"`
	Privileged      bool                         `json:"privileged,omitempty"`
	SecurityOpt     []string                     `json:"security_opt,omitempty"`
	CapAdd          []string                     `json:"cap_add,omitempty"`
	CapDrop         []string                     `json:"cap_drop,omitempty"`
	GroupAdd        []string                     `json:"group_add,omitempty"`
	Deploy          *ComposeDeploy               `json:"deploy,omitempty"`
	Restart         string                       `json:"restart,omitempty"`
	StopGracePeriod string                       `json:"stop_grace_period,omitempty"`
	Healthcheck     *ComposeHealthcheck          `json:"healthcheck,omitempty"`
}

// ComposeBuild represents build configuration.
type ComposeBuild struct {
	Context    string            `json:"context,omitempty"`
	Dockerfile string            `json:"dockerfile,omitempty"`
	Args       map[string]string `json:"args,omitempty"`
	Target     string            `json:"target,omitempty"`
}

// ComposeDependsOn represents a dependency with condition.
type ComposeDependsOn struct {
	Condition string `json:"condition,omitempty"`
}

// ComposeHealthcheck represents healthcheck configuration.
type ComposeHealthcheck struct {
	Test        []string `json:"test,omitempty"`
	Interval    string   `json:"interval,omitempty"`
	Timeout     string   `json:"timeout,omitempty"`
	Retries     int      `json:"retries,omitempty"`
	StartPeriod string   `json:"start_period,omitempty"`
}

// ComposeDeploy represents deploy configuration for resource limits.
type ComposeDeploy struct {
	Resources *ComposeResources `json:"resources,omitempty"`
}

// ComposeResources represents resource limits.
type ComposeResources struct {
	Limits       *ComposeLimits `json:"limits,omitempty"`
	Reservations *ComposeLimits `json:"reservations,omitempty"`
}

// ComposeLimits represents CPU/memory limits.
type ComposeLimits struct {
	CPUs   string `json:"cpus,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// ComposeNetwork represents a network definition.
type ComposeNetwork struct {
	Name       string            `json:"name,omitempty"`
	Driver     string            `json:"driver,omitempty"`
	Internal   bool              `json:"internal,omitempty"`
	DriverOpts map[string]string `json:"driver_opts,omitempty"`
}

// ComposeVolume represents a volume definition.
type ComposeVolume struct {
	Driver     string            `json:"driver,omitempty"`
	DriverOpts map[string]string `json:"driver_opts,omitempty"`
}

// Generator creates podman-compose.yaml from RDL configuration.
type Generator struct {
	cfg         *config.RDL
	workspaceDir string // absolute path to workspace root (where RDL file is)
}

// NewGenerator creates a new compose file generator.
func NewGenerator(cfg *config.RDL) *Generator {
	return &Generator{cfg: cfg}
}

// SetWorkspaceDir sets the workspace directory for resolving relative paths.
func (g *Generator) SetWorkspaceDir(dir string) {
	g.workspaceDir = dir
}

// Generate creates a ComposeFile from the RDL configuration.
func (g *Generator) Generate() (*ComposeFile, error) {
	if g.cfg.Containers == nil || len(g.cfg.Containers) == 0 {
		return nil, fmt.Errorf("no containers defined in RDL configuration")
	}

	compose := &ComposeFile{
		Version:  "3.8",
		Services: make(map[string]*ComposeService),
		Networks: make(map[string]*ComposeNetwork),
		Volumes:  make(map[string]*ComposeVolume),
	}

	// Add default network
	networkName := fmt.Sprintf("%s-network", g.cfg.Robot.Name)
	compose.Networks["default"] = &ComposeNetwork{
		Name: networkName,
	}

	// Convert containers to services
	for name, container := range g.cfg.Containers {
		service, err := g.convertContainer(name, container)
		if err != nil {
			return nil, fmt.Errorf("failed to convert container %q: %w", name, err)
		}
		compose.Services[name] = service
	}

	// Add custom networks
	for name, network := range g.cfg.Networks {
		compose.Networks[name] = &ComposeNetwork{
			Name:       name,
			Driver:     network.Driver,
			Internal:   network.Internal,
			DriverOpts: network.Options,
		}
	}

	// Add named volumes
	for name, volume := range g.cfg.Volumes {
		compose.Volumes[name] = &ComposeVolume{
			Driver:     volume.Driver,
			DriverOpts: volume.Options,
		}
	}

	return compose, nil
}

// convertContainer converts an RDL ContainerConfig to a ComposeService.
func (g *Generator) convertContainer(name string, container *config.ContainerConfig) (*ComposeService, error) {
	service := &ComposeService{
		ContainerName: fmt.Sprintf("%s-%s", g.cfg.Robot.Name, name),
		Image:         container.Image,
		Restart:       container.Restart,
		Volumes:       container.Volumes,
		Devices:       container.Devices,
		Ports:         container.Ports,
		NetworkMode:   container.NetworkMode,
		Networks:      container.Networks,
		Privileged:    container.Privileged,
		SecurityOpt:   container.SecurityOpt,
		CapAdd:        container.CapAdd,
		CapDrop:       container.CapDrop,
		GroupAdd:      container.GroupAdd,
		EnvFile:       container.EnvFile,
		Command:       container.Command,
		Entrypoint:    container.Entrypoint,
		StopGracePeriod: container.StopGracePeriod,
	}

	// Handle build configuration
	if container.Build != nil {
		context := container.Build.Context
		dockerfile := container.Build.Dockerfile

		// Convert relative paths to absolute paths if workspace is set
		if g.workspaceDir != "" && context != "" && !filepath.IsAbs(context) {
			context = filepath.Join(g.workspaceDir, context)
		}

		// Default dockerfile name
		if dockerfile == "" {
			dockerfile = "Containerfile"
		}

		service.Build = &ComposeBuild{
			Context:    context,
			Dockerfile: dockerfile,
			Args:       container.Build.Args,
			Target:     container.Build.Target,
		}
	}

	// Handle dependencies
	if len(container.DependsOn) > 0 {
		service.DependsOn = make(map[string]*ComposeDependsOn)
		for depName, dep := range container.DependsOn {
			condition := dep.Condition
			if condition == "" {
				condition = "service_started"
			}
			service.DependsOn[depName] = &ComposeDependsOn{
				Condition: condition,
			}
		}
	}

	// Initialize environment map
	if service.Environment == nil {
		service.Environment = make(map[string]string)
	}

	// Handle environment with variable interpolation
	for k, v := range container.Environment {
		service.Environment[k] = g.interpolateVariable(v)
	}

	// Add standard Gorai environment variables (these override any set values)
	service.Environment["GORAI_ROBOT_NAME"] = g.cfg.Robot.Name
	if g.cfg.NATS != nil && g.cfg.NATS.URL != "" {
		// Only set if not already set
		if _, exists := service.Environment["NATS_URL"]; !exists {
			service.Environment["NATS_URL"] = g.cfg.NATS.URL
		}
	}

	// Add components/services as environment variables
	if len(container.ComponentNames) > 0 {
		service.Environment["GORAI_COMPONENTS"] = strings.Join(container.ComponentNames, ",")
	}
	if len(container.ServiceNames) > 0 {
		service.Environment["GORAI_SERVICES"] = strings.Join(container.ServiceNames, ",")
	}

	// Handle healthcheck
	if container.Healthcheck != nil {
		service.Healthcheck = &ComposeHealthcheck{
			Test:        container.Healthcheck.Test,
			Interval:    container.Healthcheck.Interval,
			Timeout:     container.Healthcheck.Timeout,
			Retries:     container.Healthcheck.Retries,
			StartPeriod: container.Healthcheck.StartPeriod,
		}
	}

	// Handle resource limits
	if container.Resources != nil {
		service.Deploy = &ComposeDeploy{
			Resources: &ComposeResources{},
		}
		if container.Resources.Limits != nil {
			service.Deploy.Resources.Limits = &ComposeLimits{
				CPUs:   container.Resources.Limits.CPUs,
				Memory: container.Resources.Limits.Memory,
			}
		}
		if container.Resources.Reservations != nil {
			service.Deploy.Resources.Reservations = &ComposeLimits{
				CPUs:   container.Resources.Reservations.CPUs,
				Memory: container.Resources.Reservations.Memory,
			}
		}
	}

	// Format devices for podman
	if len(service.Devices) > 0 {
		formattedDevices := make([]string, 0, len(service.Devices))
		for _, dev := range service.Devices {
			// If device doesn't have a mapping, add one (e.g., /dev/video0 -> /dev/video0:/dev/video0)
			if !strings.Contains(dev, ":") {
				dev = fmt.Sprintf("%s:%s", dev, dev)
			}
			formattedDevices = append(formattedDevices, dev)
		}
		service.Devices = formattedDevices
	}

	return service, nil
}

// interpolateVariable handles ${robot.name} style variable interpolation.
func (g *Generator) interpolateVariable(value string) string {
	// Replace ${robot.name}
	value = strings.ReplaceAll(value, "${robot.name}", g.cfg.Robot.Name)

	// Replace ${nats.url}
	if g.cfg.NATS != nil && g.cfg.NATS.URL != "" {
		value = strings.ReplaceAll(value, "${nats.url}", g.cfg.NATS.URL)
	}

	return value
}

// WriteJSON writes the compose file to the specified path as JSON.
func (g *Generator) WriteJSON(path string) error {
	compose, err := g.Generate()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(compose, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal compose file: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	return nil
}

// WriteYAML is an alias for WriteJSON for backward compatibility.
// Podman-compose accepts JSON files with .yaml extension.
func (g *Generator) WriteYAML(path string) error {
	return g.WriteJSON(path)
}

// GetServiceNames returns the sorted list of service names.
func (g *Generator) GetServiceNames() []string {
	if g.cfg.Containers == nil {
		return nil
	}

	names := make([]string, 0, len(g.cfg.Containers))
	for name := range g.cfg.Containers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
