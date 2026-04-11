package config

import (
	"strings"
	"testing"
)

func TestNATSExternalField(t *testing.T) {
	t.Run("default is false", func(t *testing.T) {
		cfg, err := LoadFromBytes([]byte(`{
			"version": "2",
			"robot": {"name": "test-bot"},
			"nats": {"url": "nats://localhost:4222"}
		}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.NATS.External {
			t.Error("NATS.External should default to false")
		}
	})

	t.Run("parses true", func(t *testing.T) {
		cfg, err := LoadFromBytes([]byte(`{
			"version": "2",
			"robot": {"name": "test-bot"},
			"nats": {"url": "nats://localhost:4222", "external": true}
		}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !cfg.NATS.External {
			t.Error("NATS.External should be true")
		}
	})

	t.Run("absent nats section defaults external false", func(t *testing.T) {
		cfg, err := LoadFromBytes([]byte(`{
			"version": "2",
			"robot": {"name": "test-bot"}
		}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.NATS.External {
			t.Error("NATS.External should default to false when nats section is absent")
		}
	})
}

func TestIsLocalNATSURL(t *testing.T) {
	tests := []struct {
		url   string
		local bool
	}{
		{"nats://localhost:4222", true},
		{"nats://127.0.0.1:4222", true},
		{"nats://localhost", true},
		{"nats://127.0.0.1", true},
		{"nats://192.168.1.50:4222", false},
		{"nats://nats.example.com:4222", false},
		{"", true}, // empty defaults to localhost
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			cfg := &NATSConfig{URL: tt.url}
			if got := cfg.IsLocalURL(); got != tt.local {
				t.Errorf("IsLocalURL(%q) = %v, want %v", tt.url, got, tt.local)
			}
		})
	}
}

func TestShouldEmbedNATS(t *testing.T) {
	tests := []struct {
		name     string
		nats     *NATSConfig
		embedded bool
	}{
		{
			name:     "nil config embeds",
			nats:     nil,
			embedded: true,
		},
		{
			name:     "localhost without external embeds",
			nats:     &NATSConfig{URL: "nats://localhost:4222"},
			embedded: true,
		},
		{
			name:     "localhost with external does not embed",
			nats:     &NATSConfig{URL: "nats://localhost:4222", External: true},
			embedded: false,
		},
		{
			name:     "remote url does not embed",
			nats:     &NATSConfig{URL: "nats://192.168.1.50:4222"},
			embedded: false,
		},
		{
			name:     "remote url with external does not embed",
			nats:     &NATSConfig{URL: "nats://192.168.1.50:4222", External: true},
			embedded: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &RDL{
				Version: "2",
				Robot:   RobotConfig{Name: "test"},
				NATS:    tt.nats,
			}
			cfg.applyDefaults()
			if got := cfg.ShouldEmbedNATS(); got != tt.embedded {
				t.Errorf("ShouldEmbedNATS() = %v, want %v", got, tt.embedded)
			}
		})
	}
}

func TestMetricsConfig(t *testing.T) {
	t.Run("absent means disabled", func(t *testing.T) {
		cfg, err := LoadFromBytes([]byte(`{
			"version": "2",
			"robot": {"name": "test-bot"}
		}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Metrics != nil {
			t.Error("Metrics should be nil when not specified")
		}
		if cfg.IsMetricsEnabled() {
			t.Error("IsMetricsEnabled should return false when Metrics is nil")
		}
	})

	t.Run("explicit enabled", func(t *testing.T) {
		cfg, err := LoadFromBytes([]byte(`{
			"version": "2",
			"robot": {"name": "test-bot"},
			"metrics": {"enabled": true, "retention": "14d"}
		}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Metrics == nil {
			t.Fatal("Metrics should not be nil")
		}
		if !cfg.Metrics.Enabled {
			t.Error("Metrics.Enabled should be true")
		}
		if cfg.Metrics.Retention != "14d" {
			t.Errorf("Metrics.Retention = %q, want %q", cfg.Metrics.Retention, "14d")
		}
	})

	t.Run("defaults", func(t *testing.T) {
		cfg, err := LoadFromBytes([]byte(`{
			"version": "2",
			"robot": {"name": "test-bot"},
			"metrics": {"enabled": true}
		}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Metrics.GetRetention() != "7d" {
			t.Errorf("default retention = %q, want %q", cfg.Metrics.GetRetention(), "7d")
		}
		if cfg.Metrics.GetListen() != "127.0.0.1:8428" {
			t.Errorf("default listen = %q, want %q", cfg.Metrics.GetListen(), "127.0.0.1:8428")
		}
	})
}

func TestLoggingConfig(t *testing.T) {
	t.Run("absent means disabled", func(t *testing.T) {
		cfg, err := LoadFromBytes([]byte(`{
			"version": "2",
			"robot": {"name": "test-bot"}
		}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Logging != nil {
			t.Error("Logging should be nil when not specified")
		}
		if cfg.IsLoggingEnabled() {
			t.Error("IsLoggingEnabled should return false when Logging is nil")
		}
	})

	t.Run("explicit enabled", func(t *testing.T) {
		cfg, err := LoadFromBytes([]byte(`{
			"version": "2",
			"robot": {"name": "test-bot"},
			"logging": {"enabled": true, "retention": "14d", "listen": "0.0.0.0:9428"}
		}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Logging == nil {
			t.Fatal("Logging should not be nil")
		}
		if !cfg.Logging.Enabled {
			t.Error("Logging.Enabled should be true")
		}
		if cfg.Logging.Retention != "14d" {
			t.Errorf("Logging.Retention = %q, want %q", cfg.Logging.Retention, "14d")
		}
		if cfg.Logging.Listen != "0.0.0.0:9428" {
			t.Errorf("Logging.Listen = %q, want %q", cfg.Logging.Listen, "0.0.0.0:9428")
		}
	})

	t.Run("defaults", func(t *testing.T) {
		cfg, err := LoadFromBytes([]byte(`{
			"version": "2",
			"robot": {"name": "test-bot"},
			"logging": {"enabled": true}
		}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Logging.GetRetention() != "3d" {
			t.Errorf("default retention = %q, want %q", cfg.Logging.GetRetention(), "3d")
		}
		if cfg.Logging.GetListen() != "127.0.0.1:9428" {
			t.Errorf("default listen = %q, want %q", cfg.Logging.GetListen(), "127.0.0.1:9428")
		}
	})
}

func TestNATSExternalWarning(t *testing.T) {
	cfg := &RDL{
		Version: "2",
		Robot:   RobotConfig{Name: "test"},
		NATS: &NATSConfig{
			URL:      "nats://192.168.1.50:4222",
			External: true,
		},
	}
	cfg.applyDefaults()
	warnings := cfg.DeprecationWarnings()
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "external") && strings.Contains(w, "redundant") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about redundant external flag with non-local URL")
	}
}
