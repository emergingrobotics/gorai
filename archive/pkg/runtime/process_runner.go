package runtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/emergingrobotics/gorai/pkg/config"
)

// ProcessRunner manages native process-based services.
type ProcessRunner struct {
	mu sync.RWMutex
}

// NewProcessRunner creates a new ProcessRunner.
func NewProcessRunner() *ProcessRunner {
	return &ProcessRunner{}
}

// ProcessService represents a native process-based service instance.
type ProcessService struct {
	config    *config.ServiceConfig
	env       map[string]string
	configDir string
	cmd       *exec.Cmd
	pid       int
	isRunning bool
	logFile   *os.File
	mu        sync.RWMutex
}

// NewService creates a new ProcessService for the given config.
func (r *ProcessRunner) NewService(svc *config.ServiceConfig, env map[string]string, configDir string) *ProcessService {
	return &ProcessService{
		config:    svc,
		env:       env,
		configDir: configDir,
	}
}

// Start starts the process service.
func (s *ProcessService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return nil
	}

	external := s.config.External
	if external == nil || external.Command == "" {
		return fmt.Errorf("no command configured")
	}

	// Build command arguments
	args := append([]string{}, external.Args...)

	// Add config path if not already specified
	hasConfig := false
	for _, arg := range args {
		if arg == "--config" || arg == "-c" {
			hasConfig = true
			break
		}
	}
	if !hasConfig {
		// We'll pass config via environment instead
	}

	// Create the command
	s.cmd = exec.CommandContext(ctx, external.Command, args...)
	s.cmd.Dir = s.configDir

	// Set environment
	s.cmd.Env = os.Environ()
	for k, v := range s.env {
		s.cmd.Env = append(s.cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Create log file for output
	logPath := fmt.Sprintf("/tmp/gorai-%s.log", s.config.Name)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	s.logFile = logFile

	s.cmd.Stdout = logFile
	s.cmd.Stderr = logFile

	// Start the process
	if err := s.cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("failed to start process: %w", err)
	}

	s.pid = s.cmd.Process.Pid
	s.isRunning = true

	// Monitor the process in background
	go s.monitor()

	return nil
}

// monitor watches the process and handles restarts.
func (s *ProcessService) monitor() {
	if s.cmd == nil || s.cmd.Process == nil {
		return
	}

	// Wait for process to exit
	err := s.cmd.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.isRunning = false

	// Check restart policy
	restart := s.config.External.Restart
	if restart == "" {
		restart = "always"
	}

	shouldRestart := false
	switch restart {
	case "always":
		shouldRestart = true
	case "on-failure":
		if err != nil {
			shouldRestart = true
		}
	case "never":
		shouldRestart = false
	}

	if shouldRestart {
		// Wait a bit before restarting
		time.Sleep(time.Second)

		// Restart (unlock first to avoid deadlock)
		s.mu.Unlock()
		_ = s.Start(context.Background())
		s.mu.Lock()
	}
}

// Stop stops the process service.
func (s *ProcessService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning || s.cmd == nil || s.cmd.Process == nil {
		return nil
	}

	// Send SIGTERM first
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// Process might already be dead
		s.isRunning = false
		return nil
	}

	// Wait for graceful shutdown with timeout
	done := make(chan struct{})
	go func() {
		s.cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Process exited gracefully
	case <-time.After(10 * time.Second):
		// Force kill
		s.cmd.Process.Kill()
	case <-ctx.Done():
		// Context cancelled, force kill
		s.cmd.Process.Kill()
	}

	s.isRunning = false

	// Close log file
	if s.logFile != nil {
		s.logFile.Close()
		s.logFile = nil
	}

	return nil
}

// Status returns the status of the process service.
func (s *ProcessService) Status() (ServiceStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := ServiceStatus{
		Name:    s.config.Name,
		PID:     s.pid,
		Running: s.isRunning,
	}

	// Check if process is actually running
	if s.isRunning && s.cmd != nil && s.cmd.Process != nil {
		// Check if process exists
		if err := s.cmd.Process.Signal(syscall.Signal(0)); err != nil {
			status.Running = false
			status.Error = "process not found"
		} else {
			status.Running = true
			status.Healthy = true // Assume healthy if running
		}
	}

	return status, nil
}

// Logs returns a reader for the process logs.
func (s *ProcessService) Logs(follow bool) (io.ReadCloser, error) {
	logPath := fmt.Sprintf("/tmp/gorai-%s.log", s.config.Name)

	if follow {
		// Use tail -f for following logs
		cmd := exec.Command("tail", "-f", logPath)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
		}

		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("failed to start tail: %w", err)
		}

		return &tailReader{
			pipe: stdout,
			cmd:  cmd,
		}, nil
	}

	// Just open the file for non-following read
	return os.Open(logPath)
}

// IsRunning checks if the process is running.
func (s *ProcessService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isRunning || s.cmd == nil || s.cmd.Process == nil {
		return false
	}

	// Check if process actually exists
	if err := s.cmd.Process.Signal(syscall.Signal(0)); err != nil {
		return false
	}

	return true
}

// tailReader wraps tail -f output.
type tailReader struct {
	pipe io.ReadCloser
	cmd  *exec.Cmd
}

func (r *tailReader) Read(p []byte) (n int, err error) {
	return r.pipe.Read(p)
}

func (r *tailReader) Close() error {
	r.cmd.Process.Kill()
	r.pipe.Close()
	return r.cmd.Wait()
}
