package systemd

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

// Runner executes systemctl commands for container services.
type Runner struct {
	robotName  string
	serviceDir string // Directory containing .service files
	userMode   bool   // Use systemctl --user
	stdout     io.Writer
	stderr     io.Writer
}

// NewRunner creates a new systemd runner.
func NewRunner(robotName, serviceDir string, userMode bool) *Runner {
	return &Runner{
		robotName:  robotName,
		serviceDir: serviceDir,
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

// DaemonReload reloads systemd to pick up changed unit files.
func (r *Runner) DaemonReload(ctx context.Context) error {
	return r.systemctl(ctx, "daemon-reload")
}

// Uninstall removes service files from the systemd directory and reloads systemd.
func (r *Runner) Uninstall(ctx context.Context) error {
	targetDir := GetSystemdDir(r.userMode)

	// Find all service files for this robot
	files, err := r.findServiceFiles()
	if err != nil {
		return fmt.Errorf("failed to find service files: %w", err)
	}

	// Remove files from systemd directory
	for _, file := range files {
		path := filepath.Join(targetDir, file)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}

	// Reload systemd
	return r.DaemonReload(ctx)
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

// ContainerStatus represents the status of a container.
type ContainerStatus struct {
	Name    string
	Service string
	Active  string
	Status  string
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
	files, err := r.findServiceFiles()
	if err != nil {
		return nil
	}

	var services []string
	for _, file := range files {
		if strings.HasSuffix(file, ".service") {
			services = append(services, file)
		}
	}

	return services
}

// findServiceFiles finds all service files for this robot in the source directory.
func (r *Runner) findServiceFiles() ([]string, error) {
	var files []string

	entries, err := os.ReadDir(r.serviceDir)
	if err != nil {
		return nil, err
	}

	prefix := r.robotName + "-"
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".service") {
			files = append(files, name)
		}
	}

	return files, nil
}

// GetSystemdDir returns the systemd directory for user or system mode.
func GetSystemdDir(userMode bool) string {
	if userMode {
		home, err := os.UserHomeDir()
		if err != nil {
			return "/tmp/systemd/user"
		}
		return filepath.Join(home, ".config", "systemd", "user")
	}
	return "/etc/systemd/system"
}
