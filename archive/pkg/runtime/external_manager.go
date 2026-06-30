// Package runtime provides runtime management for Gorai robots, including
// external service lifecycle management.
package runtime

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/emergingrobotics/gorai/pkg/config"
)

// ServiceStatus represents the status of an external service.
type ServiceStatus struct {
	Name      string
	Running   bool
	Healthy   bool
	PID       int
	Container string
	Error     string
}

// ExternalManager manages the lifecycle of external services.
type ExternalManager interface {
	// Start starts an external service.
	Start(ctx context.Context, svc *config.ServiceConfig) error

	// Stop stops an external service.
	Stop(ctx context.Context, serviceName string) error

	// StopAll stops all managed services.
	StopAll(ctx context.Context) error

	// Status returns the status of an external service.
	Status(serviceName string) (ServiceStatus, error)

	// Logs returns a reader for the service logs.
	Logs(serviceName string, follow bool) (io.ReadCloser, error)

	// IsRunning checks if a service is running.
	IsRunning(serviceName string) bool
}

// Manager implements ExternalManager.
type Manager struct {
	cfg            *config.RDL
	configDir      string
	containerRunner *ContainerRunner
	processRunner   *ProcessRunner

	mu       sync.RWMutex
	services map[string]*managedService
}

type managedService struct {
	config    *config.ServiceConfig
	runner    ServiceRunner
	isRunning bool
}

// ServiceRunner is the interface for running a service (container or process).
type ServiceRunner interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Status() (ServiceStatus, error)
	Logs(follow bool) (io.ReadCloser, error)
	IsRunning() bool
}

// NewManager creates a new external service manager.
func NewManager(cfg *config.RDL, configDir string) *Manager {
	return &Manager{
		cfg:            cfg,
		configDir:      configDir,
		containerRunner: NewContainerRunner(),
		processRunner:   NewProcessRunner(),
		services:       make(map[string]*managedService),
	}
}

// Start starts an external service.
func (m *Manager) Start(ctx context.Context, svc *config.ServiceConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already running
	if ms, exists := m.services[svc.Name]; exists && ms.isRunning {
		return fmt.Errorf("service %q is already running", svc.Name)
	}

	// Get resolved environment
	env := config.GetResolvedEnvironment(m.cfg, svc)

	// Determine which runner to use
	var runner ServiceRunner
	if svc.External.IsContainer() {
		runner = m.containerRunner.NewService(svc, env, m.configDir)
	} else if svc.External.Command != "" {
		runner = m.processRunner.NewService(svc, env, m.configDir)
	} else {
		return fmt.Errorf("service %q has no command or container configuration", svc.Name)
	}

	// Start the service
	if err := runner.Start(ctx); err != nil {
		return fmt.Errorf("failed to start service %q: %w", svc.Name, err)
	}

	// Track the service
	m.services[svc.Name] = &managedService{
		config:    svc,
		runner:    runner,
		isRunning: true,
	}

	return nil
}

// Stop stops an external service.
func (m *Manager) Stop(ctx context.Context, serviceName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ms, exists := m.services[serviceName]
	if !exists {
		return fmt.Errorf("service %q is not managed", serviceName)
	}

	if !ms.isRunning {
		return nil // Already stopped
	}

	if err := ms.runner.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop service %q: %w", serviceName, err)
	}

	ms.isRunning = false
	return nil
}

// StopAll stops all managed services.
func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []string
	for name, ms := range m.services {
		if !ms.isRunning {
			continue
		}

		if err := ms.runner.Stop(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
		} else {
			ms.isRunning = false
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors stopping services: %v", errs)
	}

	return nil
}

// Status returns the status of an external service.
func (m *Manager) Status(serviceName string) (ServiceStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ms, exists := m.services[serviceName]
	if !exists {
		return ServiceStatus{Name: serviceName, Running: false}, nil
	}

	return ms.runner.Status()
}

// Logs returns a reader for the service logs.
func (m *Manager) Logs(serviceName string, follow bool) (io.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ms, exists := m.services[serviceName]
	if !exists {
		return nil, fmt.Errorf("service %q is not managed", serviceName)
	}

	return ms.runner.Logs(follow)
}

// IsRunning checks if a service is running.
func (m *Manager) IsRunning(serviceName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ms, exists := m.services[serviceName]
	if !exists {
		return false
	}

	return ms.isRunning && ms.runner.IsRunning()
}

// GetManagedServices returns a list of all managed service names.
func (m *Manager) GetManagedServices() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.services))
	for name := range m.services {
		names = append(names, name)
	}
	return names
}

// StartAll starts all external services in the configuration.
func (m *Manager) StartAll(ctx context.Context) error {
	external := m.cfg.GetExternalServices()
	if len(external) == 0 {
		return nil
	}

	var errs []string
	for i := range external {
		svc := &external[i]
		if svc.Disabled {
			continue
		}
		if !svc.IsManaged() {
			continue
		}

		if err := m.Start(ctx, svc); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", svc.Name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors starting services:\n  %v", errs)
	}

	return nil
}

// CheckContainerStatus checks the status of a container by name (without needing a manager).
// This is useful for checking status of containers started by other processes.
func CheckContainerStatus(containerName string) ServiceStatus {
	status := ServiceStatus{
		Name:      containerName,
		Container: containerName,
	}

	// Check if container exists and is running
	cmd := exec.Command("podman", "inspect", "--format", "{{.State.Running}}", containerName)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		status.Running = false
		status.Error = "container not found"
		return status
	}

	running := strings.TrimSpace(stdout.String())
	status.Running = running == "true"

	// Get PID if running
	if status.Running {
		cmd = exec.Command("podman", "inspect", "--format", "{{.State.Pid}}", containerName)
		stdout.Reset()
		cmd.Stdout = &stdout
		if cmd.Run() == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(stdout.String())); err == nil {
				status.PID = pid
			}
		}
	}

	return status
}

// CheckProcessStatus checks the status of a process by service name.
// It looks for a process log file and checks if the process is running.
func CheckProcessStatus(serviceName string) ServiceStatus {
	status := ServiceStatus{
		Name: serviceName,
	}

	// Look for PID file or log file
	logPath := fmt.Sprintf("/tmp/gorai-%s.log", serviceName)

	// Check if there's a process writing to the log file
	cmd := exec.Command("fuser", logPath)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		status.Running = false
		status.Error = "process not found"
		return status
	}

	// Parse PID from fuser output
	pidStr := strings.TrimSpace(stdout.String())
	if pidStr != "" {
		// fuser output may include multiple PIDs, take the first
		parts := strings.Fields(pidStr)
		if len(parts) > 0 {
			if pid, err := strconv.Atoi(parts[0]); err == nil {
				status.PID = pid
				status.Running = true
				status.Healthy = true
			}
		}
	}

	return status
}

// GetContainerLogs gets logs from a container by name.
func GetContainerLogs(containerName string, follow bool) (io.ReadCloser, error) {
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, containerName)

	cmd := exec.Command("podman", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start logs command: %w", err)
	}

	return &cmdCloser{stdout: stdout, cmd: cmd}, nil
}

// GetProcessLogs gets logs from a process log file.
func GetProcessLogs(serviceName string, follow bool) (io.ReadCloser, error) {
	logPath := fmt.Sprintf("/tmp/gorai-%s.log", serviceName)

	if follow {
		cmd := exec.Command("tail", "-f", logPath)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
		}

		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("failed to start tail: %w", err)
		}

		return &cmdCloser{stdout: stdout, cmd: cmd}, nil
	}

	// Non-following mode - just cat the file
	cmd := exec.Command("cat", logPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	return &cmdCloser{stdout: stdout, cmd: cmd}, nil
}

// cmdCloser wraps a command and its stdout for proper cleanup.
type cmdCloser struct {
	stdout io.ReadCloser
	cmd    *exec.Cmd
}

func (c *cmdCloser) Read(p []byte) (int, error) {
	return c.stdout.Read(p)
}

func (c *cmdCloser) Close() error {
	c.stdout.Close()
	if c.cmd.Process != nil {
		c.cmd.Process.Kill()
	}
	return c.cmd.Wait()
}
