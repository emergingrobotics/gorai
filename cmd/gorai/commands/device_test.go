package commands

import (
	"os"
	"strings"
	"testing"
)

func TestDeviceResetMissingNatsPrefix(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"gorai", "device", "reset", "--device-id", "pwm-board"}
	defer func() { os.Args = oldArgs }()

	err := cmdDevice()
	if err == nil {
		t.Fatal("expected error for missing --nats-prefix, got nil")
	}
	if !strings.Contains(err.Error(), "--nats-prefix is required") {
		t.Errorf("expected '--nats-prefix is required' error, got: %v", err)
	}
}

func TestDeviceResetMissingDeviceID(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"gorai", "device", "reset", "--nats-prefix", "robot1"}
	defer func() { os.Args = oldArgs }()

	err := cmdDevice()
	if err == nil {
		t.Fatal("expected error for missing --device-id, got nil")
	}
	if !strings.Contains(err.Error(), "--device-id is required") {
		t.Errorf("expected '--device-id is required' error, got: %v", err)
	}
}

func TestDeviceResetNatsURLFromEnv(t *testing.T) {
	// This test verifies flag parsing only; actual NATS connection will fail
	// because no server is running, but the error message reveals the URL used.
	oldArgs := os.Args
	os.Args = []string{"gorai", "device", "reset", "--nats-prefix", "robot1", "--device-id", "board1"}
	defer func() { os.Args = oldArgs }()

	t.Setenv("NATS_URL", "nats://custom-host:9999")

	err := cmdDevice()
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
	if !strings.Contains(err.Error(), "nats://custom-host:9999") {
		t.Errorf("expected error to mention custom NATS URL, got: %v", err)
	}
}

func TestDeviceResetNatsURLDefault(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"gorai", "device", "reset", "--nats-prefix", "robot1", "--device-id", "board1"}
	defer func() { os.Args = oldArgs }()

	// Clear NATS_URL to test default
	t.Setenv("NATS_URL", "")

	err := cmdDevice()
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
	if !strings.Contains(err.Error(), "nats://localhost:4222") {
		t.Errorf("expected error to mention default NATS URL, got: %v", err)
	}
}

func TestDeviceUnknownSubcommand(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"gorai", "device", "foobar"}
	defer func() { os.Args = oldArgs }()

	err := cmdDevice()
	if err == nil {
		t.Fatal("expected error for unknown subcommand, got nil")
	}
	if !strings.Contains(err.Error(), "unknown device subcommand") {
		t.Errorf("expected 'unknown device subcommand' error, got: %v", err)
	}
}

func TestDeviceResetExplicitNatsURL(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"gorai", "device", "reset", "--nats-prefix", "robot1", "--device-id", "board1", "--nats-url", "nats://explicit:1234"}
	defer func() { os.Args = oldArgs }()

	err := cmdDevice()
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
	if !strings.Contains(err.Error(), "nats://explicit:1234") {
		t.Errorf("expected error to mention explicit NATS URL, got: %v", err)
	}
}
