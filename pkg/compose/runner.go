package compose

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

// Runner executes podman-compose commands.
type Runner struct {
	composePath string    // Path to generated compose file
	projectName string    // Project name (robot name)
	workDir     string    // Working directory for compose commands
	stdout      io.Writer
	stderr      io.Writer
	podmanHost  string    // Optional: podman socket path for containerized execution
}

// NewRunner creates a new podman-compose runner.
func NewRunner(composePath, projectName, workDir string) *Runner {
	return &Runner{
		composePath: composePath,
		projectName: projectName,
		workDir:     workDir,
		stdout:      os.Stdout,
		stderr:      os.Stderr,
	}
}

// SetOutput sets custom stdout and stderr writers.
func (r *Runner) SetOutput(stdout, stderr io.Writer) {
	r.stdout = stdout
	r.stderr = stderr
}

// Up starts all containers.
func (r *Runner) Up(ctx context.Context, opts UpOptions) error {
	args := []string{"up"}

	if opts.Detach {
		args = append(args, "-d")
	}
	if opts.Build {
		args = append(args, "--build")
	}
	if opts.ForceRecreate {
		args = append(args, "--force-recreate")
	}
	if opts.RemoveOrphans {
		args = append(args, "--remove-orphans")
	}
	if len(opts.Services) > 0 {
		args = append(args, opts.Services...)
	}

	return r.run(ctx, args...)
}

// UpOptions configures the up command.
type UpOptions struct {
	Detach        bool
	Build         bool
	ForceRecreate bool
	RemoveOrphans bool
	Services      []string
}

// Down stops and removes containers.
func (r *Runner) Down(ctx context.Context, opts DownOptions) error {
	args := []string{"down"}

	if opts.Volumes {
		args = append(args, "--volumes")
	}
	if opts.RemoveOrphans {
		args = append(args, "--remove-orphans")
	}
	if opts.Timeout > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", opts.Timeout))
	}

	return r.run(ctx, args...)
}

// DownOptions configures the down command.
type DownOptions struct {
	Volumes       bool
	RemoveOrphans bool
	Timeout       int
}

// Build builds container images.
func (r *Runner) Build(ctx context.Context, opts BuildOptions) error {
	args := []string{"build"}

	if opts.NoCache {
		args = append(args, "--no-cache")
	}
	if opts.Pull {
		args = append(args, "--pull")
	}
	if len(opts.Services) > 0 {
		args = append(args, opts.Services...)
	}

	return r.run(ctx, args...)
}

// BuildOptions configures the build command.
type BuildOptions struct {
	NoCache  bool
	Pull     bool
	Services []string
}

// Logs streams logs from containers.
func (r *Runner) Logs(ctx context.Context, opts LogsOptions) error {
	args := []string{"logs"}

	if opts.Follow {
		args = append(args, "-f")
	}
	if opts.Tail != "" {
		args = append(args, "--tail", opts.Tail)
	}
	if opts.Timestamps {
		args = append(args, "-t")
	}
	if len(opts.Services) > 0 {
		args = append(args, opts.Services...)
	}

	return r.run(ctx, args...)
}

// LogsOptions configures the logs command.
type LogsOptions struct {
	Follow     bool
	Tail       string
	Timestamps bool
	Services   []string
}

// Ps lists containers.
func (r *Runner) Ps(ctx context.Context, all bool) (string, error) {
	args := []string{"ps"}
	if all {
		args = append(args, "-a")
	}

	var stdout bytes.Buffer
	origStdout := r.stdout
	r.stdout = &stdout
	defer func() { r.stdout = origStdout }()

	if err := r.run(ctx, args...); err != nil {
		return "", err
	}

	return stdout.String(), nil
}

// Exec runs a command in a running container.
func (r *Runner) Exec(ctx context.Context, service string, command []string, interactive bool) error {
	args := []string{"exec"}
	if interactive {
		args = append(args, "-it")
	}
	args = append(args, service)
	args = append(args, command...)

	return r.run(ctx, args...)
}

// Stop stops running containers.
func (r *Runner) Stop(ctx context.Context, timeout int, services ...string) error {
	args := []string{"stop"}
	if timeout > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", timeout))
	}
	if len(services) > 0 {
		args = append(args, services...)
	}

	return r.run(ctx, args...)
}

// Start starts existing containers.
func (r *Runner) Start(ctx context.Context, services ...string) error {
	args := []string{"start"}
	if len(services) > 0 {
		args = append(args, services...)
	}

	return r.run(ctx, args...)
}

// Restart restarts containers.
func (r *Runner) Restart(ctx context.Context, timeout int, services ...string) error {
	args := []string{"restart"}
	if timeout > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", timeout))
	}
	if len(services) > 0 {
		args = append(args, services...)
	}

	return r.run(ctx, args...)
}

// run executes a podman-compose command.
func (r *Runner) run(ctx context.Context, args ...string) error {
	// Check if podman-compose is available
	composeCmd, err := r.findComposeCommand()
	if err != nil {
		return err
	}

	// Build full argument list
	fullArgs := []string{"-f", r.composePath}
	if r.projectName != "" {
		fullArgs = append(fullArgs, "-p", r.projectName)
	}
	fullArgs = append(fullArgs, args...)

	cmd := exec.CommandContext(ctx, composeCmd, fullArgs...)
	cmd.Dir = r.workDir
	cmd.Stdout = r.stdout
	cmd.Stderr = r.stderr
	cmd.Stdin = os.Stdin

	// Set environment
	cmd.Env = os.Environ()

	// Set podman socket if running in container
	if r.podmanHost != "" {
		cmd.Env = append(cmd.Env, "CONTAINER_HOST=unix://"+r.podmanHost)
	} else {
		// Auto-detect podman socket when running in container
		socketPaths := []string{
			"/run/podman/podman.sock",
			fmt.Sprintf("/run/user/%d/podman/podman.sock", os.Getuid()),
		}
		for _, sock := range socketPaths {
			if _, err := os.Stat(sock); err == nil {
				cmd.Env = append(cmd.Env, "CONTAINER_HOST=unix://"+sock)
				break
			}
		}
	}

	return cmd.Run()
}

// findComposeCommand finds the podman-compose command.
func (r *Runner) findComposeCommand() (string, error) {
	// Try podman-compose first (installed in gorai container)
	if path, err := exec.LookPath("podman-compose"); err == nil {
		return path, nil
	}

	// Try podman compose (podman v4+ built-in)
	if path, err := exec.LookPath("podman"); err == nil {
		// Check if podman compose works
		cmd := exec.Command(path, "compose", "--version")
		if err := cmd.Run(); err == nil {
			return path + " compose", nil
		}
	}

	// Try docker-compose as fallback
	if path, err := exec.LookPath("docker-compose"); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("podman-compose not found. Run gorai in the gorai container or install with: pip install podman-compose")
}

// SetPodmanHost sets the CONTAINER_HOST environment variable for the runner.
// This is needed when running inside a container with podman socket mounted.
func (r *Runner) SetPodmanHost(socketPath string) {
	r.podmanHost = socketPath
}

// GetProjectDir returns a suitable project directory for the compose file.
func GetProjectDir(configPath string) string {
	if configPath == "" {
		return os.TempDir()
	}

	// Use the directory containing the config file
	dir := filepath.Dir(configPath)
	if filepath.IsAbs(dir) {
		return dir
	}

	// Convert to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return os.TempDir()
	}

	return absDir
}

// GetComposePath returns the path for the generated compose file.
func GetComposePath(configPath, robotName string) string {
	dir := GetProjectDir(configPath)
	return filepath.Join(dir, ".gorai", robotName+"-compose.json")
}

// ContainerStatus represents the status of a container.
type ContainerStatus struct {
	Name   string
	Image  string
	Status string
	Health string
	Ports  string
}

// ParsePsOutput parses the output of podman-compose ps.
func ParsePsOutput(output string) []ContainerStatus {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return nil
	}

	var statuses []ContainerStatus
	// Skip header line
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		status := ContainerStatus{
			Name: fields[0],
		}
		if len(fields) > 1 {
			status.Image = fields[1]
		}
		if len(fields) > 2 {
			status.Status = fields[2]
		}
		if len(fields) > 3 {
			status.Health = fields[3]
		}
		if len(fields) > 4 {
			status.Ports = strings.Join(fields[4:], " ")
		}

		statuses = append(statuses, status)
	}

	return statuses
}
