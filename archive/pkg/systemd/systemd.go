// Package systemd generates traditional systemd unit files from RDL configuration.
// This approach works on any system with Podman and systemd, without requiring
// the Quadlet generator.
package systemd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gorai/gorai/pkg/config"
)

// Generator creates systemd unit files from RDL configuration.
type Generator struct {
	cfg          *config.RDL
	workspaceDir string // absolute path to workspace root (where RDL file is)
	userMode     bool   // true for user mode (~/.config/systemd/user)
}

// UnitFiles holds all generated systemd unit file contents.
type UnitFiles struct {
	Services map[string]string // filename -> content
}

// Option configures the Generator.
type Option func(*Generator)

// WithWorkspaceDir sets the workspace directory for resolving relative paths.
func WithWorkspaceDir(dir string) Option {
	return func(g *Generator) {
		g.workspaceDir = dir
	}
}

// WithUserMode sets whether to generate for user mode or system mode.
func WithUserMode(userMode bool) Option {
	return func(g *Generator) {
		g.userMode = userMode
	}
}

// NewGenerator creates a new systemd unit generator.
func NewGenerator(cfg *config.RDL, opts ...Option) *Generator {
	g := &Generator{
		cfg:      cfg,
		userMode: true, // Default to user mode
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Generate creates all systemd unit files from the RDL configuration.
func (g *Generator) Generate() (*UnitFiles, error) {
	if g.cfg.Containers == nil || len(g.cfg.Containers) == 0 {
		return nil, fmt.Errorf("no containers defined in RDL configuration")
	}

	files := &UnitFiles{
		Services: make(map[string]string),
	}

	// Generate service for each container
	for name, container := range g.cfg.Containers {
		filename := fmt.Sprintf("%s-%s.service", g.cfg.Robot.Name, name)
		content, err := g.generateService(name, *container)
		if err != nil {
			return nil, fmt.Errorf("failed to generate service %q: %w", name, err)
		}
		files.Services[filename] = content
	}

	return files, nil
}

// GetTargetDir returns the target directory for systemd unit files.
func (g *Generator) GetTargetDir() string {
	if g.userMode {
		// User mode: ~/.config/systemd/user/
		home, err := os.UserHomeDir()
		if err != nil {
			return "/tmp/systemd/user"
		}
		return filepath.Join(home, ".config", "systemd", "user")
	}
	// System mode: /etc/systemd/system/
	return "/etc/systemd/system"
}

// GetLocalDir returns the local .gorai directory for generated files.
func (g *Generator) GetLocalDir() string {
	if g.workspaceDir != "" {
		return filepath.Join(g.workspaceDir, ".gorai")
	}
	return ".gorai"
}

// WriteFiles writes all systemd files to the local .gorai directory.
func (g *Generator) WriteFiles() error {
	files, err := g.Generate()
	if err != nil {
		return err
	}

	targetDir := g.GetLocalDir()

	// Ensure directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
	}

	// Write all service files
	for filename, content := range files.Services {
		path := filepath.Join(targetDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
	}

	return nil
}

// InstallFiles copies systemd files to the systemd directory.
func (g *Generator) InstallFiles() error {
	files, err := g.Generate()
	if err != nil {
		return err
	}

	targetDir := g.GetTargetDir()

	// Ensure directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
	}

	// Write all service files
	for filename, content := range files.Services {
		path := filepath.Join(targetDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
	}

	return nil
}

// GetServiceNames returns the sorted list of service names.
func (g *Generator) GetServiceNames() []string {
	if g.cfg.Containers == nil {
		return nil
	}

	names := make([]string, 0, len(g.cfg.Containers))
	for name := range g.cfg.Containers {
		names = append(names, fmt.Sprintf("%s-%s.service", g.cfg.Robot.Name, name))
	}
	sort.Strings(names)
	return names
}

// GetNetworkName returns the network name for this robot.
func (g *Generator) GetNetworkName() string {
	return fmt.Sprintf("%s-network", g.cfg.Robot.Name)
}

// generateService creates a systemd service unit file for a container.
func (g *Generator) generateService(name string, container config.ContainerConfig) (string, error) {
	var sb strings.Builder

	serviceName := fmt.Sprintf("%s-%s", g.cfg.Robot.Name, name)
	networkName := g.GetNetworkName()

	// [Unit] section
	sb.WriteString("[Unit]\n")
	sb.WriteString(fmt.Sprintf("Description=Gorai container %s\n", serviceName))

	// Dependencies
	deps := g.getDependencies(name, container)
	if len(deps) > 0 {
		sb.WriteString(fmt.Sprintf("Requires=%s\n", strings.Join(deps, " ")))
		sb.WriteString(fmt.Sprintf("After=%s\n", strings.Join(deps, " ")))
	} else {
		sb.WriteString("After=network-online.target\n")
		sb.WriteString("Wants=network-online.target\n")
	}
	sb.WriteString("\n")

	// [Service] section
	sb.WriteString("[Service]\n")
	sb.WriteString("Type=simple\n")

	// Restart policy
	restart := container.Restart
	if restart == "" || restart == "unless-stopped" {
		restart = "always"
	}
	sb.WriteString(fmt.Sprintf("Restart=%s\n", restart))
	sb.WriteString("RestartSec=10\n")

	// Timeout
	timeout := container.TimeoutStart
	if timeout == "" {
		timeout = "300"
	}
	sb.WriteString(fmt.Sprintf("TimeoutStartSec=%s\n", timeout))
	sb.WriteString("\n")

	// ExecStartPre - cleanup and network creation
	sb.WriteString("# Cleanup any existing container\n")
	sb.WriteString(fmt.Sprintf("ExecStartPre=-/usr/bin/podman stop -t 10 %s\n", serviceName))
	sb.WriteString(fmt.Sprintf("ExecStartPre=-/usr/bin/podman rm -f %s\n", serviceName))
	sb.WriteString(fmt.Sprintf("ExecStartPre=-/usr/bin/podman network create %s\n", networkName))
	sb.WriteString("\n")

	// ExecStart - podman run command
	sb.WriteString("# Start container\n")
	sb.WriteString("ExecStart=/usr/bin/podman run --rm \\\n")
	sb.WriteString(fmt.Sprintf("    --name %s \\\n", serviceName))
	sb.WriteString(fmt.Sprintf("    --network %s \\\n", networkName))

	// Add network alias for service discovery
	sb.WriteString(fmt.Sprintf("    --network-alias %s \\\n", name))

	// Published ports
	for _, port := range container.Ports {
		sb.WriteString(fmt.Sprintf("    -p %s \\\n", port))
	}

	// Environment variables - add defaults
	sb.WriteString(fmt.Sprintf("    -e GORAI_ROBOT_NAME=%s \\\n", g.cfg.Robot.Name))
	if g.cfg.NATS != nil && g.cfg.NATS.URL != "" {
		sb.WriteString(fmt.Sprintf("    -e NATS_URL=%s \\\n", g.cfg.NATS.URL))
	}

	// Custom environment variables
	for key, value := range container.Environment {
		sb.WriteString(fmt.Sprintf("    -e %s=%s \\\n", key, g.interpolateVariable(value)))
	}

	// Volumes
	for _, vol := range container.Volumes {
		// Handle relative paths
		volPath := g.resolveVolumePath(vol)
		sb.WriteString(fmt.Sprintf("    -v %s \\\n", volPath))
	}

	// Devices
	for _, device := range container.Devices {
		sb.WriteString(fmt.Sprintf("    --device %s \\\n", device))
	}

	// Group memberships
	for _, group := range container.GroupAdd {
		sb.WriteString(fmt.Sprintf("    --group-add %s \\\n", group))
	}

	// Security options
	for _, opt := range container.SecurityOpt {
		if opt == "label=disable" {
			sb.WriteString("    --security-opt label=disable \\\n")
		}
	}

	// Resource limits
	if container.Resources != nil && container.Resources.Limits != nil {
		if container.Resources.Limits.Memory != "" {
			sb.WriteString(fmt.Sprintf("    --memory=%s \\\n", container.Resources.Limits.Memory))
		}
		if container.Resources.Limits.CPUs != "" {
			sb.WriteString(fmt.Sprintf("    --cpus=%s \\\n", container.Resources.Limits.CPUs))
		}
	}

	// Image
	sb.WriteString(fmt.Sprintf("    %s\n", container.Image))
	sb.WriteString("\n")

	// ExecStop
	sb.WriteString(fmt.Sprintf("ExecStop=/usr/bin/podman stop -t 10 %s\n", serviceName))
	sb.WriteString("\n")

	// Security hardening
	sb.WriteString("# Security hardening\n")
	sb.WriteString("NoNewPrivileges=yes\n")
	sb.WriteString("ProtectSystem=strict\n")
	sb.WriteString("ProtectHome=true\n")
	sb.WriteString("PrivateTmp=true\n")
	sb.WriteString("\n")

	// [Install] section
	sb.WriteString("[Install]\n")
	sb.WriteString(fmt.Sprintf("WantedBy=%s\n", g.wantedBy()))

	return sb.String(), nil
}

// getDependencies returns the systemd service dependencies for a container.
func (g *Generator) getDependencies(name string, container config.ContainerConfig) []string {
	if container.DependsOn == nil {
		return nil
	}

	var deps []string
	for depName := range container.DependsOn {
		serviceName := fmt.Sprintf("%s-%s.service", g.cfg.Robot.Name, depName)
		deps = append(deps, serviceName)
	}
	sort.Strings(deps)
	return deps
}

// resolveVolumePath resolves relative volume paths to absolute paths.
func (g *Generator) resolveVolumePath(vol string) string {
	// Split volume spec: src:dest[:options]
	parts := strings.SplitN(vol, ":", 3)
	if len(parts) < 2 {
		return vol
	}

	src := parts[0]
	dest := parts[1]
	options := ""
	if len(parts) == 3 {
		options = ":" + parts[2]
	}

	// If source is relative, make it absolute
	if !strings.HasPrefix(src, "/") && g.workspaceDir != "" {
		src = filepath.Join(g.workspaceDir, src)
	}

	return src + ":" + dest + options
}

// interpolateVariable handles ${robot.name} style variable interpolation.
func (g *Generator) interpolateVariable(value string) string {
	value = strings.ReplaceAll(value, "${robot.name}", g.cfg.Robot.Name)

	if g.cfg.NATS != nil && g.cfg.NATS.URL != "" {
		value = strings.ReplaceAll(value, "${nats.url}", g.cfg.NATS.URL)
	}

	return value
}

// wantedBy returns the appropriate systemd target.
func (g *Generator) wantedBy() string {
	if g.userMode {
		return "default.target"
	}
	return "multi-user.target"
}
