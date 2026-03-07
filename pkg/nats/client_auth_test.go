package nats

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfig_NKeyFile(t *testing.T) {
	cfg := &Config{
		URL:      "nats://localhost:4222",
		NKeyFile: "/etc/gorai/robot.nkey",
	}

	if cfg.NKeyFile != "/etc/gorai/robot.nkey" {
		t.Errorf("NKeyFile = %q, want %q", cfg.NKeyFile, "/etc/gorai/robot.nkey")
	}
}

func TestConfig_Token(t *testing.T) {
	cfg := &Config{
		URL:   "nats://localhost:4222",
		Token: "dev-token",
	}

	if cfg.Token != "dev-token" {
		t.Errorf("Token = %q, want %q", cfg.Token, "dev-token")
	}
}

func TestConnect_InvalidNKeyFile(t *testing.T) {
	cfg := &Config{
		URL:            "nats://localhost:4222",
		Name:           "test",
		NKeyFile:       "/nonexistent/path/robot.nkey",
		ConnectTimeout: 1,
		ReconnectWait:  1,
		MaxReconnects:  0,
	}

	ctx := t.Context()
	_, err := Connect(ctx, cfg)
	if err == nil {
		t.Fatal("expected error for nonexistent NKey file")
	}
}

func TestValidateSeedFilePermissions_Secure(t *testing.T) {
	dir := t.TempDir()
	seedFile := filepath.Join(dir, "secure.nkey")
	if err := os.WriteFile(seedFile, []byte("seed-content"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := ValidateSeedFilePermissions(seedFile); err != nil {
		t.Errorf("expected no error for 0600 permissions, got: %v", err)
	}
}

func TestValidateSeedFilePermissions_Insecure(t *testing.T) {
	dir := t.TempDir()
	seedFile := filepath.Join(dir, "insecure.nkey")
	if err := os.WriteFile(seedFile, []byte("seed-content"), 0644); err != nil {
		t.Fatal(err)
	}

	err := ValidateSeedFilePermissions(seedFile)
	if err == nil {
		t.Fatal("expected error for 0644 permissions")
	}
	if !strings.Contains(err.Error(), "insecure permissions") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidateSeedFilePermissions_WorldReadable(t *testing.T) {
	dir := t.TempDir()
	seedFile := filepath.Join(dir, "world.nkey")
	if err := os.WriteFile(seedFile, []byte("seed-content"), 0666); err != nil {
		t.Fatal(err)
	}

	err := ValidateSeedFilePermissions(seedFile)
	if err == nil {
		t.Fatal("expected error for 0666 permissions")
	}
}

func TestConnect_InsecureNKeyPermissions(t *testing.T) {
	dir := t.TempDir()
	nkeyFile := filepath.Join(dir, "insecure.nkey")
	if err := os.WriteFile(nkeyFile, []byte("SUACSSL3UAHUDXKFSNVUZRF5UHPMWZ6BFDTJ7M6USDXIEDNPPQYYYCU3VY"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		URL:            "nats://localhost:4222",
		Name:           "test",
		NKeyFile:       nkeyFile,
		ConnectTimeout: 1,
		ReconnectWait:  1,
		MaxReconnects:  0,
	}

	ctx := t.Context()
	_, err := Connect(ctx, cfg)
	if err == nil {
		t.Fatal("expected error for insecure NKey file permissions")
	}
	if !strings.Contains(err.Error(), "insecure permissions") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConnect_InvalidNKeyContent(t *testing.T) {
	dir := t.TempDir()
	nkeyFile := filepath.Join(dir, "bad.nkey")
	if err := os.WriteFile(nkeyFile, []byte("not-a-valid-nkey-seed"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		URL:            "nats://localhost:4222",
		Name:           "test",
		NKeyFile:       nkeyFile,
		ConnectTimeout: 1,
		ReconnectWait:  1,
		MaxReconnects:  0,
	}

	ctx := t.Context()
	_, err := Connect(ctx, cfg)
	if err == nil {
		t.Fatal("expected error for invalid NKey content")
	}
}
