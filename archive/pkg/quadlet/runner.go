package quadlet

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Runner executes systemctl commands for Quadlet-managed containers.
type Runner struct {
	robotName   string    // Robot name (used for service naming)
	quadletDir  string    // Directory containing Quadlet files
	userMode    bool      // Use systemctl --user
	stdout      io.Writer
	stderr      io.Writer
}

// NewRunner creates a new Quadlet runner.
func NewRunner(robotName, quadletDir string, userMode bool) *Runner {
	return &Runner{
		robotName:  robotName,
		quadletDir: quadletDir,
		userMode:   userMode,
		stdout:     os.Stdout,
		stderr:     os.Stderr,
	}
}

// SetOutput sets custom stdout and stderr writers.
func (r *Runner) SetOutput(stdout, stderr io.Writer) {
	r.stdout = stdout
	r.stderr = stderr
}

// Install copies Quadlet files to the systemd directory and reloads systemd.
func (r *Runner) Install(ctx context.Context) error {
	targetDir := r.getSystemdDir()

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create systemd directory: %w", err)
	}

	// Find all Quadlet files for this robot
	files, err := r.findQuadletFiles()
	if err != nil {
		return fmt.Errorf("failed to find Quadlet files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no Quadlet files found in %s", r.quadletDir)
	}

	// Copy files to systemd directory
	for _, file := range files {
		src := filepath.Join(r.quadletDir, file)
		dst := filepath.Join(targetDir, file)

		content, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", src, err)
		}

		if err := os.WriteFile(dst, content, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", dst, err)
		}
	}

	// Reload systemd to pick up new units
	return r.daemonReload(ctx)
}

// Uninstall removes Quadlet files and reloads systemd.
func (r *Runner) Uninstall(ctx context.Context) error {
	targetDir := r.getSystemdDir()

	// Find all Quadlet files for this robot
	files, err := r.findQuadletFiles()
	if err != nil {
		return fmt.Errorf("failed to find Quadlet files: %w", err)
	}

	// Remove files from systemd directory
	for _, file := range files {
		path := filepath.Join(targetDir, file)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}

	// Reload systemd
	return r.daemonReload(ctx)
}

// Start starts containers.
func (r *Runner) Start(ctx context.Context, containers ...string) error {
	services := r.getServiceNames(containers)
	if len(services) == 0 {
		return fmt.Errorf("no services to start")
	}

	args := []string{"start"}
	args = append(args, services...)

	return r.systemctl(ctx, args...)
}

// Stop stops containers.
func (r *Runner) Stop(ctx context.Context, containers ...string) error {
	services := r.getServiceNames(containers)
	if len(services) == 0 {
		return fmt.Errorf("no services to stop")
	}

	args := []string{"stop"}
	args = append(args, services...)

	return r.systemctl(ctx, args...)
}

// Restart restarts containers.
func (r *Runner) Restart(ctx context.Context, containers ...string) error {
	services := r.getServiceNames(containers)
	if len(services) == 0 {
		return fmt.Errorf("no services to restart")
	}

	args := []string{"restart"}
	args = append(args, services...)

	return r.systemctl(ctx, args...)
}

// Status gets the status of containers.
func (r *Runner) Status(ctx context.Context, containers ...string) (string, error) {
	services := r.getServiceNames(containers)
	if len(services) == 0 {
		// Get all robot services
		services = r.getAllServiceNames()
	}

	args := []string{"status", "--no-pager"}
	args = append(args, services...)

	var stdout bytes.Buffer
	origStdout := r.stdout
	r.stdout = &stdout
	defer func() { r.stdout = origStdout }()

	// Don't fail on non-zero exit (inactive services return non-zero)
	_ = r.systemctl(ctx, args...)

	return stdout.String(), nil
}

// Logs streams logs from containers using journalctl.
func (r *Runner) Logs(ctx context.Context, opts LogsOptions) error {
	services := r.getServiceNames(opts.Containers)

	args := []string{}

	if r.userMode {
		args = append(args, "--user")
	}

	// Add unit filters
	for _, svc := range services {
		args = append(args, "-u", svc)
	}

	if opts.Follow {
		args = append(args, "-f")
	}
	if opts.Tail > 0 {
		args = append(args, "-n", fmt.Sprintf("%d", opts.Tail))
	}
	if opts.Timestamps {
		args = append(args, "-o", "short-iso")
	}

	args = append(args, "--no-pager")

	return r.journalctl(ctx, args...)
}

// LogsOptions configures the logs command.
type LogsOptions struct {
	Containers []string
	Follow     bool
	Tail       int
	Timestamps bool
}

// Enable enables services to start at boot.
func (r *Runner) Enable(ctx context.Context, containers ...string) error {
	services := r.getServiceNames(containers)
	if len(services) == 0 {
		services = r.getAllServiceNames()
	}

	args := []string{"enable"}
	args = append(args, services...)

	return r.systemctl(ctx, args...)
}

// Disable disables services from starting at boot.
func (r *Runner) Disable(ctx context.Context, containers ...string) error {
	services := r.getServiceNames(containers)
	if len(services) == 0 {
		services = r.getAllServiceNames()
	}

	args := []string{"disable"}
	args = append(args, services...)

	return r.systemctl(ctx, args...)
}

// IsActive checks if a service is active.
func (r *Runner) IsActive(ctx context.Context, container string) (bool, error) {
	service := fmt.Sprintf("%s-%s.service", r.robotName, container)

	args := []string{"is-active", "--quiet", service}

	err := r.systemctl(ctx, args...)
	return err == nil, nil
}

// DaemonReload reloads systemd to pick up changed unit files.
func (r *Runner) DaemonReload(ctx context.Context) error {
	return r.daemonReload(ctx)
}

// AutoUpdate runs podman auto-update for all containers.
func (r *Runner) AutoUpdate(ctx context.Context, dryRun bool) error {
	args := []string{"auto-update"}
	if dryRun {
		args = append(args, "--dry-run")
	}

	cmd := exec.CommandContext(ctx, "podman", args...)
	cmd.Stdout = r.stdout
	cmd.Stderr = r.stderr

	return cmd.Run()
}

// Build builds container images using podman build.
func (r *Runner) Build(ctx context.Context, opts BuildOptions) error {
	// Find containers with build configurations
	buildFiles, err := filepath.Glob(filepath.Join(r.quadletDir, r.robotName+"-*.build"))
	if err != nil {
		return fmt.Errorf("failed to find build files: %w", err)
	}

	if len(buildFiles) == 0 && len(opts.Containers) == 0 {
		return fmt.Errorf("no build configurations found")
	}

	// For now, we'll run podman build directly for each Containerfile
	// In a full implementation, we'd parse the .build files
	for _, container := range opts.Containers {
		args := []string{"build"}

		if opts.NoCache {
			args = append(args, "--no-cache")
		}
		if opts.Pull {
			args = append(args, "--pull=always")
		}

		// Add tag
		tag := fmt.Sprintf("%s-%s:latest", r.robotName, container)
		args = append(args, "-t", tag)

		// Context would come from the build config
		args = append(args, ".")

		cmd := exec.CommandContext(ctx, "podman", args...)
		cmd.Dir = r.quadletDir
		cmd.Stdout = r.stdout
		cmd.Stderr = r.stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to build %s: %w", container, err)
		}
	}

	return nil
}

// BuildOptions configures the build command.
type BuildOptions struct {
	Containers []string
	NoCache    bool
	Pull       bool
}

// ContainerStatus represents the status of a container.
type ContainerStatus struct {
	Name    string
	Service string
	Active  string
	Status  string
}

// GetStatus returns structured status information for all containers.
func (r *Runner) GetStatus(ctx context.Context) ([]ContainerStatus, error) {
	services := r.getAllServiceNames()
	var statuses []ContainerStatus

	for _, svc := range services {
		status := ContainerStatus{
			Service: svc,
			Name:    strings.TrimSuffix(strings.TrimPrefix(svc, r.robotName+"-"), ".service"),
		}

		// Check if active
		args := []string{"is-active", svc}
		var stdout bytes.Buffer
		origStdout := r.stdout
		r.stdout = &stdout
		_ = r.systemctl(ctx, args...)
		r.stdout = origStdout
		status.Active = strings.TrimSpace(stdout.String())

		// Get status
		args = []string{"show", svc, "--property=SubState", "--value"}
		stdout.Reset()
		r.stdout = &stdout
		_ = r.systemctl(ctx, args...)
		r.stdout = origStdout
		status.Status = strings.TrimSpace(stdout.String())

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// systemctl runs a systemctl command.
func (r *Runner) systemctl(ctx context.Context, args ...string) error {
	fullArgs := make([]string, 0, len(args)+1)
	if r.userMode {
		fullArgs = append(fullArgs, "--user")
	}
	fullArgs = append(fullArgs, args...)

	cmd := exec.CommandContext(ctx, "systemctl", fullArgs...)
	cmd.Stdout = r.stdout
	cmd.Stderr = r.stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// journalctl runs a journalctl command.
func (r *Runner) journalctl(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "journalctl", args...)
	cmd.Stdout = r.stdout
	cmd.Stderr = r.stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// daemonReload reloads the systemd daemon.
func (r *Runner) daemonReload(ctx context.Context) error {
	return r.systemctl(ctx, "daemon-reload")
}

// getSystemdDir returns the systemd directory for Quadlet files.
func (r *Runner) getSystemdDir() string {
	if r.userMode {
		home, err := os.UserHomeDir()
		if err != nil {
			return "/tmp/containers/systemd"
		}
		return filepath.Join(home, ".config", "containers", "systemd")
	}
	return "/etc/containers/systemd"
}

// findQuadletFiles finds all Quadlet files for this robot in the source directory.
func (r *Runner) findQuadletFiles() ([]string, error) {
	var files []string

	entries, err := os.ReadDir(r.quadletDir)
	if err != nil {
		return nil, err
	}

	prefix := r.robotName + "-"
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		// Check for Quadlet extensions
		if strings.HasSuffix(name, ".container") ||
			strings.HasSuffix(name, ".network") ||
			strings.HasSuffix(name, ".volume") ||
			strings.HasSuffix(name, ".build") ||
			strings.HasSuffix(name, ".pod") {
			files = append(files, name)
		}
	}

	return files, nil
}

// getServiceNames converts container names to systemd service names.
func (r *Runner) getServiceNames(containers []string) []string {
	if len(containers) == 0 {
		return r.getAllServiceNames()
	}

	services := make([]string, len(containers))
	for i, c := range containers {
		services[i] = fmt.Sprintf("%s-%s.service", r.robotName, c)
	}
	return services
}

// getAllServiceNames returns all service names for this robot.
func (r *Runner) getAllServiceNames() []string {
	files, err := r.findQuadletFiles()
	if err != nil {
		return nil
	}

	var services []string
	for _, file := range files {
		if strings.HasSuffix(file, ".container") {
			// Convert .container to .service
			svc := strings.TrimSuffix(file, ".container") + ".service"
			services = append(services, svc)
		}
	}

	return services
}
