package v4l2

import (
	"testing"
	"time"

	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid config with defaults",
			config: &Config{
				Device:        "/dev/video0",
				Width:         640,
				Height:        480,
				FrameRate:     30,
				JPEGQuality:   80,
				PublishToBus:  true,
				PublishRateHz: 15,
				FrameID:       "camera_link",
			},
			expectErr: false,
		},
		{
			name: "empty device path",
			config: &Config{
				Device:        "",
				Width:         640,
				Height:        480,
				FrameRate:     30,
				JPEGQuality:   80,
				PublishRateHz: 15,
			},
			expectErr: true,
			errMsg:    "device path cannot be empty",
		},
		{
			name: "invalid width",
			config: &Config{
				Device:        "/dev/video0",
				Width:         0,
				Height:        480,
				FrameRate:     30,
				JPEGQuality:   80,
				PublishRateHz: 15,
			},
			expectErr: true,
			errMsg:    "width must be positive",
		},
		{
			name: "negative height",
			config: &Config{
				Device:        "/dev/video0",
				Width:         640,
				Height:        -1,
				FrameRate:     30,
				JPEGQuality:   80,
				PublishRateHz: 15,
			},
			expectErr: true,
			errMsg:    "height must be positive",
		},
		{
			name: "zero frame rate",
			config: &Config{
				Device:        "/dev/video0",
				Width:         640,
				Height:        480,
				FrameRate:     0,
				JPEGQuality:   80,
				PublishRateHz: 15,
			},
			expectErr: true,
			errMsg:    "frame_rate must be positive",
		},
		{
			name: "frame rate too high",
			config: &Config{
				Device:        "/dev/video0",
				Width:         640,
				Height:        480,
				FrameRate:     200,
				JPEGQuality:   80,
				PublishRateHz: 15,
			},
			expectErr: true,
			errMsg:    "exceeds maximum",
		},
		{
			name: "jpeg quality too low",
			config: &Config{
				Device:        "/dev/video0",
				Width:         640,
				Height:        480,
				FrameRate:     30,
				JPEGQuality:   0,
				PublishRateHz: 15,
			},
			expectErr: true,
			errMsg:    "jpeg_quality must be 1-100",
		},
		{
			name: "jpeg quality too high",
			config: &Config{
				Device:        "/dev/video0",
				Width:         640,
				Height:        480,
				FrameRate:     30,
				JPEGQuality:   101,
				PublishRateHz: 15,
			},
			expectErr: true,
			errMsg:    "jpeg_quality must be 1-100",
		},
		{
			name: "zero publish rate",
			config: &Config{
				Device:        "/dev/video0",
				Width:         640,
				Height:        480,
				FrameRate:     30,
				JPEGQuality:   80,
				PublishRateHz: 0,
			},
			expectErr: true,
			errMsg:    "publish_rate_hz must be positive",
		},
		{
			name: "publish rate exceeds frame rate",
			config: &Config{
				Device:        "/dev/video0",
				Width:         640,
				Height:        480,
				FrameRate:     30,
				JPEGQuality:   80,
				PublishRateHz: 60,
			},
			expectErr: true,
			errMsg:    "cannot exceed frame_rate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigParsing(t *testing.T) {
	resConf := resource.Config{
		Attributes: map[string]any{
			"device":          "/dev/video1",
			"width":           1280.0,
			"height":          720.0,
			"frame_rate":      60.0,
			"jpeg_quality":    85.0,
			"publish_to_bus":  false,
			"publish_rate_hz": 30.0,
			"frame_id":        "front_camera",
		},
	}

	cfg, err := NewConfigFromResource(resConf)
	require.NoError(t, err)

	assert.Equal(t, "/dev/video1", cfg.Device)
	assert.Equal(t, 1280, cfg.Width)
	assert.Equal(t, 720, cfg.Height)
	assert.Equal(t, 60.0, cfg.FrameRate)
	assert.Equal(t, 85, cfg.JPEGQuality)
	assert.False(t, cfg.PublishToBus)
	assert.Equal(t, 30.0, cfg.PublishRateHz)
	assert.Equal(t, "front_camera", cfg.FrameID)
}

func TestConfigDefaults(t *testing.T) {
	resConf := resource.Config{
		Attributes: map[string]any{},
	}

	cfg, err := NewConfigFromResource(resConf)
	require.NoError(t, err)

	assert.Equal(t, "/dev/video0", cfg.Device)
	assert.Equal(t, 640, cfg.Width)
	assert.Equal(t, 480, cfg.Height)
	assert.Equal(t, 30.0, cfg.FrameRate)
	assert.Equal(t, 80, cfg.JPEGQuality)
	assert.True(t, cfg.PublishToBus)
	assert.Equal(t, 15.0, cfg.PublishRateHz)
	assert.Equal(t, "camera_link", cfg.FrameID)
}

func TestConfigParsingWithIntegers(t *testing.T) {
	resConf := resource.Config{
		Attributes: map[string]any{
			"width":           1920,
			"height":          1080,
			"frame_rate":      30,
			"jpeg_quality":    75,
			"publish_rate_hz": 15,
		},
	}

	cfg, err := NewConfigFromResource(resConf)
	require.NoError(t, err)

	assert.Equal(t, 1920, cfg.Width)
	assert.Equal(t, 1080, cfg.Height)
	assert.Equal(t, 30.0, cfg.FrameRate)
	assert.Equal(t, 75, cfg.JPEGQuality)
	assert.Equal(t, 15.0, cfg.PublishRateHz)
}

func TestPublishInterval(t *testing.T) {
	tests := []struct {
		name             string
		frameRate        float64
		publishRateHz    float64
		expectedInterval time.Duration
	}{
		{
			name:             "publish equals frame rate",
			frameRate:        30,
			publishRateHz:    30,
			expectedInterval: 0, // Publish every frame
		},
		{
			name:             "publish half of frame rate",
			frameRate:        30,
			publishRateHz:    15,
			expectedInterval: time.Second / 15, // ~66ms
		},
		{
			name:             "publish at 1 Hz",
			frameRate:        30,
			publishRateHz:    1,
			expectedInterval: time.Second, // 1s
		},
		{
			name:             "high frame rate, low publish rate",
			frameRate:        60,
			publishRateHz:    10,
			expectedInterval: time.Second / 10, // 100ms
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				FrameRate:     tt.frameRate,
				PublishRateHz: tt.publishRateHz,
			}
			interval := cfg.PublishInterval()
			assert.Equal(t, tt.expectedInterval, interval)
		})
	}
}

func TestCaptureInterval(t *testing.T) {
	tests := []struct {
		name             string
		frameRate        float64
		expectedInterval time.Duration
	}{
		{
			name:             "30 FPS",
			frameRate:        30,
			expectedInterval: time.Second / 30, // ~33ms
		},
		{
			name:             "60 FPS",
			frameRate:        60,
			expectedInterval: time.Second / 60, // ~16ms
		},
		{
			name:             "1 FPS",
			frameRate:        1,
			expectedInterval: time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				FrameRate: tt.frameRate,
			}
			interval := cfg.CaptureInterval()
			assert.Equal(t, tt.expectedInterval, interval)
		})
	}
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpening, "opening"},
		{StateStreaming, "streaming"},
		{StateError, "error"},
		{State(100), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestConfigValidationEdgeCases(t *testing.T) {
	// Test minimum valid values
	cfg := &Config{
		Device:        "/dev/video0",
		Width:         1,
		Height:        1,
		FrameRate:     0.1,
		JPEGQuality:   1,
		PublishRateHz: 0.1,
	}
	err := cfg.Validate()
	assert.NoError(t, err)

	// Test maximum valid values
	cfg = &Config{
		Device:        "/dev/video0",
		Width:         4096,
		Height:        2160,
		FrameRate:     120,
		JPEGQuality:   100,
		PublishRateHz: 120,
	}
	err = cfg.Validate()
	assert.NoError(t, err)
}

func TestConfigParsingPartialOverride(t *testing.T) {
	// Only override some values, check defaults for others
	resConf := resource.Config{
		Attributes: map[string]any{
			"device": "/dev/video2",
			"width":  800.0,
		},
	}

	cfg, err := NewConfigFromResource(resConf)
	require.NoError(t, err)

	// Overridden values
	assert.Equal(t, "/dev/video2", cfg.Device)
	assert.Equal(t, 800, cfg.Width)

	// Default values
	assert.Equal(t, 480, cfg.Height)
	assert.Equal(t, 30.0, cfg.FrameRate)
	assert.Equal(t, 80, cfg.JPEGQuality)
	assert.True(t, cfg.PublishToBus)
	assert.Equal(t, 15.0, cfg.PublishRateHz)
	assert.Equal(t, "camera_link", cfg.FrameID)
}

func TestFrameRateRelationship(t *testing.T) {
	// Test that publish rate at boundary conditions
	cfg := &Config{
		Device:        "/dev/video0",
		Width:         640,
		Height:        480,
		FrameRate:     30,
		JPEGQuality:   80,
		PublishRateHz: 30, // Exactly equal to frame rate
	}
	err := cfg.Validate()
	assert.NoError(t, err)

	// Publish rate just below frame rate
	cfg.PublishRateHz = 29.9
	err = cfg.Validate()
	assert.NoError(t, err)

	// Publish rate just above frame rate (should fail)
	cfg.PublishRateHz = 30.1
	err = cfg.Validate()
	assert.Error(t, err)
}

func TestConfigResolutions(t *testing.T) {
	// Test common resolutions
	resolutions := []struct {
		width  int
		height int
	}{
		{320, 240},   // QVGA
		{640, 480},   // VGA
		{800, 600},   // SVGA
		{1280, 720},  // 720p
		{1920, 1080}, // 1080p
		{3840, 2160}, // 4K
	}

	for _, res := range resolutions {
		cfg := &Config{
			Device:        "/dev/video0",
			Width:         res.width,
			Height:        res.height,
			FrameRate:     30,
			JPEGQuality:   80,
			PublishRateHz: 15,
		}
		err := cfg.Validate()
		assert.NoError(t, err, "Resolution %dx%d should be valid", res.width, res.height)
	}
}

func TestNegativeFrameRate(t *testing.T) {
	cfg := &Config{
		Device:        "/dev/video0",
		Width:         640,
		Height:        480,
		FrameRate:     -1,
		JPEGQuality:   80,
		PublishRateHz: 15,
	}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "frame_rate must be positive")
}

func TestNegativePublishRate(t *testing.T) {
	cfg := &Config{
		Device:        "/dev/video0",
		Width:         640,
		Height:        480,
		FrameRate:     30,
		JPEGQuality:   80,
		PublishRateHz: -5,
	}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publish_rate_hz must be positive")
}

