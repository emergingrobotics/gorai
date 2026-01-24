package runtime

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/gorai/gorai/pkg/config"
)

// ContainerRunner manages container-based services using podman.
type ContainerRunner struct {
	mu sync.RWMutex
}

// NewContainerRunner creates a new ContainerRunner.
func NewContainerRunner() *ContainerRunner {
	return &ContainerRunner{}
}

// ContainerService represents a container-based service instance.
type ContainerService struct {
	config       *config.ServiceConfig
	env          map[string]string
	configDir    string
	containerID  string
	containerName string
	isRunning    bool
	mu           sync.RWMutex
}

// NewService creates a new ContainerService for the given config.
func (r *ContainerRunner) NewService(svc *config.ServiceConfig, env map[string]string, configDir string) *ContainerService {
	return &ContainerService{
		config:        svc,
		env:           env,
		configDir:     configDir,
		containerName: svc.Name,
	}
}

// Start starts the container service.
func (s *ContainerService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return nil
	}

	container := s.config.External.Container
	if container == nil {
		return fmt.Errorf("no container configuration")
	}

	// Remove existing container with same name (if any)
	_ = exec.CommandContext(ctx, "podman", "rm", "-f", s.containerName).Run()

	// Build podman run command
	args := []string{"run", "-d", "--name", s.containerName}

	// Add restart policy
	restart := s.config.External.Restart
	if restart == "" {
		restart = "always"
	}
	if restart != "never" {
		args = append(args, "--restart", restart)
	}

	// Add network mode
	network := container.Network
	if network == "" {
		network = "host"
	}
	args = append(args, "--network", network)

	// Add devices
	for _, device := range container.Devices {
		args = append(args, "--device", device)
	}

	// Add volumes
	for _, volume := range container.Volumes {
		args = append(args, "-v", volume)
	}

	// Add environment variables
	for k, v := range s.env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add privileged if needed
	if container.Privileged {
		args = append(args, "--privileged")
	}

	// Add image
	args = append(args, container.Image)

	// Run the container
	cmd := exec.CommandContext(ctx, "podman", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start container: %w, stderr: %s", err, stderr.String())
	}

	s.containerID = strings.TrimSpace(stdout.String())
	s.isRunning = true

	return nil
}

// Stop stops the container service.
func (s *ContainerService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return nil
	}

	// Stop the container
	cmd := exec.CommandContext(ctx, "podman", "stop", "-t", "10", s.containerName)
	if err := cmd.Run(); err != nil {
		// Try force remove
		_ = exec.CommandContext(ctx, "podman", "rm", "-f", s.containerName).Run()
	}

	s.isRunning = false
	return nil
}

// Status returns the status of the container service.
func (s *ContainerService) Status() (ServiceStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := ServiceStatus{
		Name:      s.config.Name,
		Container: s.containerName,
	}

	// Check if container is running
	cmd := exec.Command("podman", "inspect", "--format", "{{.State.Running}}", s.containerName)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		status.Running = false
		status.Error = "container not found"
		return status, nil
	}

	running := strings.TrimSpace(stdout.String())
	status.Running = running == "true"

	// Check health if available
	cmd = exec.Command("podman", "inspect", "--format", "{{.State.Health.Status}}", s.containerName)
	stdout.Reset()
	cmd.Stdout = &stdout
	if cmd.Run() == nil {
		health := strings.TrimSpace(stdout.String())
		status.Healthy = health == "healthy" || health == ""
	}

	return status, nil
}

// Logs returns a reader for the container logs.
func (s *ContainerService) Logs(follow bool) (io.ReadCloser, error) {
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, s.containerName)

	cmd := exec.Command("podman", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start logs command: %w", err)
	}

	// Combine stdout and stderr
	return &combinedReader{
		stdout: stdout,
		stderr: stderr,
		cmd:    cmd,
	}, nil
}

// IsRunning checks if the container is running.
func (s *ContainerService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isRunning {
		return false
	}

	// Verify with podman
	cmd := exec.Command("podman", "inspect", "--format", "{{.State.Running}}", s.containerName)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return false
	}

	return strings.TrimSpace(stdout.String()) == "true"
}

// combinedReader combines stdout and stderr into a single reader.
type combinedReader struct {
	stdout io.ReadCloser
	stderr io.ReadCloser
	cmd    *exec.Cmd
}

func (r *combinedReader) Read(p []byte) (n int, err error) {
	// Try to read from stdout first
	n, err = r.stdout.Read(p)
	if n > 0 {
		return n, nil
	}
	if err != io.EOF {
		return n, err
	}

	// Then try stderr
	return r.stderr.Read(p)
}

func (r *combinedReader) Close() error {
	r.stdout.Close()
	r.stderr.Close()
	return r.cmd.Wait()
}

// BuildContainer builds a container image for the service.
func BuildContainer(ctx context.Context, svc *config.ServiceConfig, configDir string, noCache bool) error {
	container := svc.External.Container
	if container == nil {
		return fmt.Errorf("no container configuration")
	}

	buildCtx := container.GetBuildContext(configDir)
	if buildCtx == "" {
		return fmt.Errorf("no build context configured")
	}

	// Check if build context exists
	if _, err := os.Stat(buildCtx); os.IsNotExist(err) {
		return fmt.Errorf("build context does not exist: %s", buildCtx)
	}

	// Build args
	args := []string{"build"}

	if noCache || (container.Build != nil && container.Build.NoCache) {
		args = append(args, "--no-cache")
	}

	// Tag
	args = append(args, "-t", container.Image)

	// Containerfile
	containerfile := container.GetContainerfile()
	args = append(args, "-f", containerfile)

	// Build args from config
	if container.Build != nil {
		for key, value := range container.Build.Args {
			args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
		}

		// Target for multi-stage builds
		if container.Build.Target != "" {
			args = append(args, "--target", container.Build.Target)
		}
	}

	// Context
	args = append(args, buildCtx)

	// Run build
	cmd := exec.CommandContext(ctx, "podman", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = buildCtx

	return cmd.Run()
}

// ContainerExists checks if a container with the given name exists.
func ContainerExists(name string) bool {
	cmd := exec.Command("podman", "container", "exists", name)
	return cmd.Run() == nil
}

// ImageExists checks if a container image exists locally.
func ImageExists(image string) bool {
	cmd := exec.Command("podman", "image", "exists", image)
	return cmd.Run() == nil
}
