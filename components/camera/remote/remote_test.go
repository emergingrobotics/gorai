package remote

import (
	"context"
	"image"
	"image/color"
	"log/slog"
	"testing"
	"time"

	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				Subject:            "gorai.test.camera.data",
				Width:            640,
				Height:           480,
				BufferSize:       10,
				StaleThresholdMs: 5000,
			},
			wantErr: false,
		},
		{
			name: "missing subject",
			config: &Config{
				Subject:            "",
				Width:            640,
				Height:           480,
				BufferSize:       10,
				StaleThresholdMs: 5000,
			},
			wantErr: true,
			errMsg:  "subject is required",
		},
		{
			name: "invalid width",
			config: &Config{
				Subject:            "gorai.test.camera.data",
				Width:            0,
				Height:           480,
				BufferSize:       10,
				StaleThresholdMs: 5000,
			},
			wantErr: true,
			errMsg:  "width must be at least 1",
		},
		{
			name: "invalid height",
			config: &Config{
				Subject:            "gorai.test.camera.data",
				Width:            640,
				Height:           0,
				BufferSize:       10,
				StaleThresholdMs: 5000,
			},
			wantErr: true,
			errMsg:  "height must be at least 1",
		},
		{
			name: "invalid buffer_size",
			config: &Config{
				Subject:            "gorai.test.camera.data",
				Width:            640,
				Height:           480,
				BufferSize:       0,
				StaleThresholdMs: 5000,
			},
			wantErr: true,
			errMsg:  "buffer_size must be at least 1",
		},
		{
			name: "invalid stale_threshold",
			config: &Config{
				Subject:            "gorai.test.camera.data",
				Width:            640,
				Height:           480,
				BufferSize:       10,
				StaleThresholdMs: 50,
			},
			wantErr: true,
			errMsg:  "stale_threshold_ms must be at least 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfigParsing(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"subject":              "gorai.main-robot.camera.data",
			"width":              float64(1280),
			"height":             float64(720),
			"buffer_size":        float64(20),
			"stale_threshold_ms": float64(3000),
		},
	}

	cfg, err := NewConfigFromResource(conf)
	require.NoError(t, err)

	assert.Equal(t, "gorai.main-robot.camera.data", cfg.Subject)
	assert.Equal(t, 1280, cfg.Width)
	assert.Equal(t, 720, cfg.Height)
	assert.Equal(t, 20, cfg.BufferSize)
	assert.Equal(t, int64(3000), cfg.StaleThresholdMs)
}

func TestConfigDefaults(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"subject": "gorai.test.camera.data",
		},
	}

	cfg, err := NewConfigFromResource(conf)
	require.NoError(t, err)

	assert.Equal(t, "gorai.test.camera.data", cfg.Subject)
	assert.Equal(t, 640, cfg.Width)
	assert.Equal(t, 480, cfg.Height)
	assert.Equal(t, 10, cfg.BufferSize)
	assert.Equal(t, int64(5000), cfg.StaleThresholdMs)
}

func TestStaleThreshold(t *testing.T) {
	cfg := &Config{
		StaleThresholdMs: 3000,
	}

	threshold := cfg.StaleThreshold()
	assert.Equal(t, "3s", threshold.String())
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateClosed, "closed"},
		{StateStarting, "starting"},
		{StateConnected, "connected"},
		{StateStale, "stale"},
		{StateError, "error"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.state.String())
		})
	}
}

func TestRingBufferBehavior(t *testing.T) {
	// Create a camera with a small buffer for testing
	r := &RemoteCamera{
		frames:     make([]Frame, 3),
		frameHead:  0,
		frameCount: 0,
	}

	// Add frames
	for i := 0; i < 5; i++ {
		r.frameMu.Lock()
		frame := Frame{
			Data: []byte{byte(i)},
		}
		r.frames[r.frameHead] = frame
		r.frameHead = (r.frameHead + 1) % len(r.frames)
		if r.frameCount < len(r.frames) {
			r.frameCount++
		}
		r.frameMu.Unlock()
	}

	// Should have 3 frames (buffer size)
	assert.Equal(t, 3, r.frameCount)

	// Latest frame should be 4 (0-indexed)
	frame, err := r.getLatestFrame()
	require.NoError(t, err)
	assert.Equal(t, byte(4), frame.Data[0])
}

func TestGetLatestFrameEmpty(t *testing.T) {
	r := &RemoteCamera{
		frames:     make([]Frame, 3),
		frameHead:  0,
		frameCount: 0,
	}

	_, err := r.getLatestFrame()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no frames available")
}

func TestStreamFanOutMultipleConsumers(t *testing.T) {
	r := &RemoteCamera{
		config: &Config{
			Subject:            "gorai.test.camera.data",
			Width:            640,
			Height:           480,
			BufferSize:       10,
			StaleThresholdMs: 5000,
		},
		logger:       slog.Default(),
		state:        StateConnected,
		frames:       make([]Frame, 10),
		receiveTimes: make([]time.Time, 0, 100),
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
	}

	go func() {
		<-r.stopCh
		close(r.doneCh)
	}()

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	ch1, err := r.Stream(ctx1)
	require.NoError(t, err)

	ch2, err := r.Stream(ctx2)
	require.NoError(t, err)

	assert.NotEqual(t, ch1, ch2)

	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	r.streamSubsMu.Lock()
	for _, sub := range r.streamSubs {
		sub.ch <- img
	}
	r.streamSubsMu.Unlock()

	select {
	case got := <-ch1:
		assert.NotNil(t, got)
	case <-time.After(time.Second):
		t.Fatal("consumer 1 did not receive frame")
	}

	select {
	case got := <-ch2:
		assert.NotNil(t, got)
	case <-time.After(time.Second):
		t.Fatal("consumer 2 did not receive frame")
	}

	cancel1()
	cancel2()
	time.Sleep(50 * time.Millisecond)
	close(r.stopCh)
	<-r.doneCh
}
