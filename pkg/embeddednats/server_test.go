package embeddednats

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func TestServerStartAndAcceptConnections(t *testing.T) {
	server, err := New(Config{
		Host: "127.0.0.1",
		Port: portFreeEphemeral,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Shutdown()

	if !server.IsRunning() {
		t.Fatal("server should be running after Start")
	}

	// Connect a real NATS client
	nc, err := nats.Connect(server.ClientURL())
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}
	defer nc.Close()

	if !nc.IsConnected() {
		t.Fatal("client should be connected")
	}
}

func TestServerJetStreamEnabled(t *testing.T) {
	jetStreamDirectory := t.TempDir()

	server, err := New(Config{
		Host:         "127.0.0.1",
		Port:         portFreeEphemeral,
		JetStream:    true,
		JetStreamDir: jetStreamDirectory,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Shutdown()

	nc, err := nats.Connect(server.ClientURL())
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}
	defer nc.Close()

	jetStream, err := nc.JetStream()
	if err != nil {
		t.Fatalf("failed to get JetStream context: %v", err)
	}

	// Creating a stream should succeed when JetStream is enabled
	_, err = jetStream.AddStream(&nats.StreamConfig{
		Name:     "TEST",
		Subjects: []string{"test.>"},
	})
	if err != nil {
		t.Fatalf("JetStream should be enabled but AddStream failed: %v", err)
	}
}

func TestServerJetStreamDisabled(t *testing.T) {
	server, err := New(Config{
		Host:      "127.0.0.1",
		Port:      portFreeEphemeral,
		JetStream: false,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Shutdown()

	nc, err := nats.Connect(server.ClientURL())
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}
	defer nc.Close()

	jetStream, err := nc.JetStream()
	if err != nil {
		t.Fatalf("failed to get JetStream context: %v", err)
	}

	// Creating a stream should fail when JetStream is disabled
	_, err = jetStream.AddStream(&nats.StreamConfig{
		Name:     "TEST",
		Subjects: []string{"test.>"},
	})
	if err == nil {
		t.Fatal("JetStream should be disabled but AddStream succeeded")
	}
}

func TestServerShutdownCleanly(t *testing.T) {
	server, err := New(Config{
		Host: "127.0.0.1",
		Port: portFreeEphemeral,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	clientURL := server.ClientURL()

	nc, err := nats.Connect(clientURL)
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}

	// Drain the client first, then shut down the server (per REQ-NATS-EMBED-7)
	nc.Drain()
	nc.Close()

	server.Shutdown()

	if server.IsRunning() {
		t.Fatal("server should not be running after Shutdown")
	}

	// Verify that new connections are rejected
	_, err = nats.Connect(clientURL, nats.MaxReconnects(0), nats.Timeout(500*time.Millisecond))
	if err == nil {
		t.Fatal("should not be able to connect after shutdown")
	}
}

func TestClientURLFormat(t *testing.T) {
	server, err := New(Config{
		Host: "127.0.0.1",
		Port: portFreeEphemeral,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Shutdown()

	url := server.ClientURL()
	// The URL must start with nats:// and contain the host
	expectedPrefix := "nats://127.0.0.1:"
	if len(url) <= len(expectedPrefix) {
		t.Fatalf("ClientURL %q is too short", url)
	}
	if url[:len(expectedPrefix)] != expectedPrefix {
		t.Fatalf("ClientURL %q does not start with %q", url, expectedPrefix)
	}
}

func TestIsRunningReturnsCorrectState(t *testing.T) {
	server, err := New(Config{
		Host: "127.0.0.1",
		Port: portFreeEphemeral,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Before Start
	if server.IsRunning() {
		t.Fatal("server should not be running before Start")
	}

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// After Start
	if !server.IsRunning() {
		t.Fatal("server should be running after Start")
	}

	server.Shutdown()

	// After Shutdown
	if server.IsRunning() {
		t.Fatal("server should not be running after Shutdown")
	}
}

func TestWaitReadyWithCancelledContext(t *testing.T) {
	server, err := New(Config{
		Host: "127.0.0.1",
		Port: portFreeEphemeral,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Use an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = server.WaitReady(ctx)
	if err == nil {
		t.Fatal("WaitReady should return error with cancelled context")
	}
}

func TestWaitReadySucceedsAfterStart(t *testing.T) {
	server, err := New(Config{
		Host: "127.0.0.1",
		Port: portFreeEphemeral,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady should succeed on running server: %v", err)
	}
}

func TestDefaultConfigValues(t *testing.T) {
	server, err := New(Config{})
	if err != nil {
		t.Fatalf("failed to create server with empty config: %v", err)
	}

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Shutdown()

	expectedPrefix := fmt.Sprintf("nats://%s:", defaultHost)
	url := server.ClientURL()
	if len(url) <= len(expectedPrefix) || url[:len(expectedPrefix)] != expectedPrefix {
		t.Fatalf("default config should use host %s, got URL %q", defaultHost, url)
	}
}
