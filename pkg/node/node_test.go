package node_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/emergingrobotics/gorai/pkg/node"
)

func TestNode_New(t *testing.T) {
	n, err := node.New("test-node")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer n.Close()

	if n.Name() != "test-node" {
		t.Errorf("Name() = %q, want 'test-node'", n.Name())
	}
}

func TestNode_WithNamespace(t *testing.T) {
	n, err := node.New("sensor", node.WithNamespace("robot1"))
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer n.Close()

	if n.Namespace() != "robot1" {
		t.Errorf("Namespace() = %q, want 'robot1'", n.Namespace())
	}
	if n.FullName() != "robot1.sensor" {
		t.Errorf("FullName() = %q, want 'robot1.sensor'", n.FullName())
	}
}

func TestNode_FullName_NoNamespace(t *testing.T) {
	n, err := node.New("standalone")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer n.Close()

	if n.FullName() != "standalone" {
		t.Errorf("FullName() = %q, want 'standalone'", n.FullName())
	}
}

func TestNode_WithLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	n, err := node.New("test", node.WithLogger(logger))
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer n.Close()

	if n.Logger() != logger {
		t.Error("Logger() returned wrong logger")
	}
}

func TestNode_NoNATS(t *testing.T) {
	n, err := node.New("test")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer n.Close()

	// Should work without NATS
	if n.NATS() != nil {
		t.Error("NATS() should be nil when no connection configured")
	}
	if n.IsConnected() {
		t.Error("IsConnected() should be false")
	}
	if n.HasJetStream() {
		t.Error("HasJetStream() should be false")
	}
}

func TestNode_Spin(t *testing.T) {
	n, err := node.New("test")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer n.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if n.IsRunning() {
		t.Error("IsRunning() should be false before Spin")
	}

	err = n.Spin(ctx)
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Spin failed: %v", err)
	}

	if n.IsRunning() {
		t.Error("IsRunning() should be false after Spin completes")
	}
}

func TestNode_Spin_AlreadyRunning(t *testing.T) {
	n, err := node.New("test")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer n.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start Spin in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- n.Spin(ctx)
	}()

	// Wait a bit for Spin to start
	time.Sleep(50 * time.Millisecond)

	// Try to Spin again - should fail
	err = n.Spin(ctx)
	if err == nil {
		t.Error("expected error when calling Spin twice")
	}

	// Clean up
	cancel()
	<-errCh
}

func TestNode_Shutdown(t *testing.T) {
	n, err := node.New("test")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer n.Close()

	ctx := context.Background()

	// Start Spin in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- n.Spin(ctx)
	}()

	// Wait a bit for Spin to start
	time.Sleep(50 * time.Millisecond)

	// Shutdown should stop Spin
	n.Shutdown()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Spin returned error after Shutdown: %v", err)
		}
	case <-time.After(time.Second):
		t.Error("Spin did not return after Shutdown")
	}
}

func TestNode_SpinOnce_NoNATS(t *testing.T) {
	n, err := node.New("test")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer n.Close()

	ctx := context.Background()
	err = n.SpinOnce(ctx, 100*time.Millisecond)
	if err != nil {
		t.Errorf("SpinOnce failed: %v", err)
	}
}

func TestNode_Close_Multiple(t *testing.T) {
	n, err := node.New("test")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Close multiple times should not panic
	n.Close()
	n.Close()
	n.Close()
}
