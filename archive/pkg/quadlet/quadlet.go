// Package quadlet generates systemd Quadlet unit files from RDL configuration.
// Quadlet is systemd's native container management that replaces podman-compose.
package quadlet

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gorai/gorai/pkg/config"
)

// Generator creates Quadlet unit files from RDL configuration.
type Generator struct {
	cfg          *config.RDL
	workspaceDir string // absolute path to workspace root (where RDL file is)
	userMode     bool   // true for rootless (~/.config), false for system (/etc)
}

// QuadletFiles holds all generated Quadlet unit file contents.
type QuadletFiles struct {
	Networks   map[string]string // filename -> content
	Volumes    map[string]string
	Containers map[string]string
}

// Option configures the Generator.
type Option func(*Generator)

// WithWorkspaceDir sets the workspace directory for resolving relative paths.
func WithWorkspaceDir(dir string) Option {
	return func(g *Generator) {
		g.workspaceDir = dir
	}
}

// WithUserMode sets whether to generate for user mode (rootless) or system mode.
func WithUserMode(userMode bool) Option {
	return func(g *Generator) {
		g.userMode = userMode
	}
}

// NewGenerator creates a new Quadlet generator.
func NewGenerator(cfg *config.RDL, opts ...Option) *Generator {
	g := &Generator{
		cfg:      cfg,
		userMode: true, // Default to rootless
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Generate creates all Quadlet unit files from the RDL configuration.
func (g *Generator) Generate() (*QuadletFiles, error) {
	if g.cfg.Containers == nil || len(g.cfg.Containers) == 0 {
		return nil, fmt.Errorf("no containers defined in RDL configuration")
	}

	files := &QuadletFiles{
		Networks:   make(map[string]string),
		Volumes:    make(map[string]string),
		Containers: make(map[string]string),
	}

	// Generate default network
	networkName := fmt.Sprintf("%s-network", g.cfg.Robot.Name)
	files.Networks[networkName+".network"] = g.generateDefaultNetwork(networkName)

	// Generate custom networks
	for name, network := range g.cfg.Networks {
		filename := fmt.Sprintf("%s-%s.network", g.cfg.Robot.Name, name)
		content := g.generateNetwork(name, network)
		files.Networks[filename] = content
	}

	// Generate volumes
	for name, volume := range g.cfg.Volumes {
		filename := fmt.Sprintf("%s-%s.volume", g.cfg.Robot.Name, name)
		content := g.generateVolume(name, volume)
		files.Volumes[filename] = content
	}

	// Generate containers
	for name, container := range g.cfg.Containers {
		filename := fmt.Sprintf("%s-%s.container", g.cfg.Robot.Name, name)
		content, err := g.generateContainer(name, container)
		if err != nil {
			return nil, fmt.Errorf("failed to generate container %q: %w", name, err)
		}
		files.Containers[filename] = content
	}

	return files, nil
}

// GetTargetDir returns the target directory for Quadlet files.
func (g *Generator) GetTargetDir() string {
	if g.userMode {
		// User mode: ~/.config/containers/systemd/
		home, err := os.UserHomeDir()
		if err != nil {
			return "/tmp/quadlet"
		}
		return filepath.Join(home, ".config", "containers", "systemd")
	}
	// System mode: /etc/containers/systemd/
	return "/etc/containers/systemd"
}

// GetLocalDir returns the local .gorai directory for generated files.
func (g *Generator) GetLocalDir() string {
	if g.workspaceDir != "" {
		return filepath.Join(g.workspaceDir, ".gorai")
	}
	return ".gorai"
}

// WriteFiles writes all Quadlet files to the local .gorai directory.
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

	// Write all files
	allFiles := make(map[string]string)
	for k, v := range files.Networks {
		allFiles[k] = v
	}
	for k, v := range files.Volumes {
		allFiles[k] = v
	}
	for k, v := range files.Containers {
		allFiles[k] = v
	}

	for filename, content := range allFiles {
		path := filepath.Join(targetDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
	}

	return nil
}

// InstallFiles copies Quadlet files to the systemd directory.
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

	// Write all files
	allFiles := make(map[string]string)
	for k, v := range files.Networks {
		allFiles[k] = v
	}
	for k, v := range files.Volumes {
		allFiles[k] = v
	}
	for k, v := range files.Containers {
		allFiles[k] = v
	}

	for filename, content := range allFiles {
		path := filepath.Join(targetDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
	}

	return nil
}

// GetServiceNames returns the sorted list of container service names.
func (g *Generator) GetServiceNames() []string {
	if g.cfg.Containers == nil {
		return nil
	}

	names := make([]string, 0, len(g.cfg.Containers))
	for name := range g.cfg.Containers {
		// Service names are the container filenames without .container
		names = append(names, fmt.Sprintf("%s-%s", g.cfg.Robot.Name, name))
	}
	sort.Strings(names)
	return names
}

// GetQuadletDir returns the path where Quadlet files are stored.
func GetQuadletDir(configPath string) string {
	if configPath == "" {
		return filepath.Join(os.TempDir(), ".gorai")
	}

	dir := filepath.Dir(configPath)
	if !filepath.IsAbs(dir) {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return filepath.Join(os.TempDir(), ".gorai")
		}
		dir = absDir
	}

	return filepath.Join(dir, ".gorai")
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
