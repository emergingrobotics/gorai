package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuthConfigParse_NKey(t *testing.T) {
	json := `{
		"robot": {"name": "test"},
		"nats": {
			"url": "nats://server:4222",
			"auth": {
				"method": "nkey",
				"nkey_file": "/etc/gorai/robot.nkey"
			}
		}
	}`

	dir := t.TempDir()
	path := filepath.Join(dir, "robot.json")
	if err := os.WriteFile(path, []byte(json), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.NATS == nil {
		t.Fatal("NATS config is nil")
	}
	if cfg.NATS.Auth == nil {
		t.Fatal("NATS auth config is nil")
	}
	if cfg.NATS.Auth.Method != "nkey" {
		t.Errorf("auth method = %q, want %q", cfg.NATS.Auth.Method, "nkey")
	}
	if cfg.NATS.Auth.NKeyFile != "/etc/gorai/robot.nkey" {
		t.Errorf("nkey_file = %q, want %q", cfg.NATS.Auth.NKeyFile, "/etc/gorai/robot.nkey")
	}
}

func TestAuthConfigParse_Token(t *testing.T) {
	json := `{
		"robot": {"name": "test"},
		"nats": {
			"url": "nats://localhost:4222",
			"auth": {
				"method": "token",
				"token": "dev-secret-token"
			}
		}
	}`

	dir := t.TempDir()
	path := filepath.Join(dir, "robot.json")
	if err := os.WriteFile(path, []byte(json), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.NATS.Auth == nil {
		t.Fatal("NATS auth config is nil")
	}
	if cfg.NATS.Auth.Method != "token" {
		t.Errorf("auth method = %q, want %q", cfg.NATS.Auth.Method, "token")
	}
	if cfg.NATS.Auth.Token != "dev-secret-token" {
		t.Errorf("token = %q, want %q", cfg.NATS.Auth.Token, "dev-secret-token")
	}
}

func TestAuthConfigParse_None(t *testing.T) {
	json := `{
		"robot": {"name": "test"},
		"nats": {
			"url": "nats://localhost:4222"
		}
	}`

	dir := t.TempDir()
	path := filepath.Join(dir, "robot.json")
	if err := os.WriteFile(path, []byte(json), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.NATS.Auth != nil {
		t.Errorf("auth config should be nil when not specified, got %+v", cfg.NATS.Auth)
	}
}

func TestDashboardConfigParse_Auth(t *testing.T) {
	json := `{
		"robot": {"name": "test"},
		"nats": {"url": "nats://localhost:4222"},
		"dashboard": {
			"listen": "0.0.0.0:8080",
			"username": "admin",
			"password": "secret123"
		}
	}`

	dir := t.TempDir()
	path := filepath.Join(dir, "robot.json")
	if err := os.WriteFile(path, []byte(json), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Dashboard == nil {
		t.Fatal("dashboard config is nil")
	}
	if cfg.Dashboard.Username != "admin" {
		t.Errorf("username = %q, want %q", cfg.Dashboard.Username, "admin")
	}
	if cfg.Dashboard.Password != "secret123" {
		t.Errorf("password = %q, want %q", cfg.Dashboard.Password, "secret123")
	}
}

func TestAuthValidation_TokenRemoteRejected(t *testing.T) {
	cfg := &RDL{
		Version: "1",
		Robot:   RobotConfig{Name: "test"},
		NATS: &NATSConfig{
			URL: "nats://remote-server:4222",
			Auth: &AuthConfig{
				Method: "token",
				Token:  "my-token",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for token auth with remote URL")
	}
	if !strings.Contains(err.Error(), "token is only allowed for localhost") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAuthValidation_TokenLocalhostAllowed(t *testing.T) {
	cfg := &RDL{
		Version: "1",
		Robot:   RobotConfig{Name: "test"},
		NATS: &NATSConfig{
			URL: "nats://localhost:4222",
			Auth: &AuthConfig{
				Method: "token",
				Token:  "my-token",
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestAuthValidation_NKeyMissingFile(t *testing.T) {
	cfg := &RDL{
		Version: "1",
		Robot:   RobotConfig{Name: "test"},
		NATS: &NATSConfig{
			URL: "nats://server:4222",
			Auth: &AuthConfig{
				Method: "nkey",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for nkey without nkey_file")
	}
	if !strings.Contains(err.Error(), "nkey_file required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAuthValidation_InvalidMethod(t *testing.T) {
	cfg := &RDL{
		Version: "1",
		Robot:   RobotConfig{Name: "test"},
		NATS: &NATSConfig{
			URL: "nats://localhost:4222",
			Auth: &AuthConfig{
				Method: "invalid",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for invalid auth method")
	}
	if !strings.Contains(err.Error(), "unsupported value") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDashboardConfigParse_NoAuth(t *testing.T) {
	json := `{
		"robot": {"name": "test"},
		"nats": {"url": "nats://localhost:4222"},
		"dashboard": {
			"listen": "127.0.0.1:8080"
		}
	}`

	dir := t.TempDir()
	path := filepath.Join(dir, "robot.json")
	if err := os.WriteFile(path, []byte(json), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Dashboard == nil {
		t.Fatal("dashboard config is nil")
	}
	if cfg.Dashboard.Username != "" {
		t.Errorf("username should be empty, got %q", cfg.Dashboard.Username)
	}
	if cfg.Dashboard.Password != "" {
		t.Errorf("password should be empty, got %q", cfg.Dashboard.Password)
	}
}
