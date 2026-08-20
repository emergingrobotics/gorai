package bno085

import (
	"testing"

	"github.com/emergingrobotics/gorai/pkg/registry"
)

func TestParseConfigDefaults(t *testing.T) {
	cfg, err := ParseConfig(registry.Config{})
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	id := identityMounting()
	if cfg.Mounting != id {
		t.Fatalf("default mounting = %+v, want %+v", cfg.Mounting, id)
	}
	if cfg.OffsetDeg != [3]float64{} {
		t.Fatalf("default offset = %v, want zero", cfg.OffsetDeg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config invalid: %v", err)
	}
}

func TestParseConfigMountingAndOffset(t *testing.T) {
	cfg, err := ParseConfig(registry.Config{
		"mounting": map[string]any{"x": "+y", "y": "-x", "z": "+z"},
		"offset":   map[string]any{"roll_deg": 5.0, "pitch_deg": -3.0, "yaw_deg": 90.0},
	})
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if cfg.Mounting.X != "+y" || cfg.Mounting.Y != "-x" || cfg.Mounting.Z != "+z" {
		t.Fatalf("mounting = %+v", cfg.Mounting)
	}
	if cfg.OffsetDeg != [3]float64{5, -3, 90} {
		t.Fatalf("offset = %v", cfg.OffsetDeg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
}

func TestValidateRejectsBadMounting(t *testing.T) {
	cfg, err := ParseConfig(registry.Config{
		"mounting": map[string]any{"x": "+x", "y": "+y", "z": "-z"}, // left-handed
	})
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected left-handed mounting to fail validation")
	}
}

func TestPartialMountingDefaultsToIdentity(t *testing.T) {
	cfg, err := ParseConfig(registry.Config{
		"mounting": map[string]any{"z": "+z"},
	})
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if cfg.Mounting != identityMounting() {
		t.Fatalf("partial mounting = %+v, want identity", cfg.Mounting)
	}
}
