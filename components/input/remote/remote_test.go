package remote

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/gorai/gorai/components/input"
	"github.com/gorai/gorai/pkg/resource"
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
			name: "valid config",
			config: &Config{
				Topic:            "gorai.test.keyboard.events",
				BufferSize:       100,
				StaleThresholdMs: 3000,
			},
			expectErr: false,
		},
		{
			name: "missing topic",
			config: &Config{
				Topic:            "",
				BufferSize:       100,
				StaleThresholdMs: 3000,
			},
			expectErr: true,
			errMsg:    "topic is required",
		},
		{
			name: "buffer size too low",
			config: &Config{
				Topic:            "test.topic",
				BufferSize:       0,
				StaleThresholdMs: 3000,
			},
			expectErr: true,
			errMsg:    "buffer_size must be at least 1",
		},
		{
			name: "stale threshold too low",
			config: &Config{
				Topic:            "test.topic",
				BufferSize:       100,
				StaleThresholdMs: 50,
			},
			expectErr: true,
			errMsg:    "stale_threshold_ms must be at least 100",
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
			"topic":                     "gorai.gs.keyboard.events",
			"buffer_size":               200.0,
			"stale_threshold_ms":        3000.0,
			"auto_release_on_disconnect": false,
		},
	}

	cfg, err := NewConfigFromResource(resConf)
	require.NoError(t, err)

	assert.Equal(t, "gorai.gs.keyboard.events", cfg.Topic)
	assert.Equal(t, 200, cfg.BufferSize)
	assert.Equal(t, int64(3000), cfg.StaleThresholdMs)
	assert.False(t, cfg.AutoReleaseOnDisconnect)
}

func TestConfigDefaults(t *testing.T) {
	resConf := resource.Config{
		Attributes: map[string]any{
			"topic": "test.topic",
		},
	}

	cfg, err := NewConfigFromResource(resConf)
	require.NoError(t, err)

	assert.Equal(t, "test.topic", cfg.Topic)
	assert.Equal(t, 100, cfg.BufferSize)
	assert.Equal(t, int64(5000), cfg.StaleThresholdMs)
	assert.True(t, cfg.AutoReleaseOnDisconnect)
}

func TestStaleThreshold(t *testing.T) {
	cfg := &Config{StaleThresholdMs: 3000}
	assert.Equal(t, 3*time.Second, cfg.StaleThreshold())
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "closed"},
		{StateStarting, "starting"},
		{StateConnected, "connected"},
		{StateStale, "stale"},
		{StateError, "error"},
		{State(100), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestKeyEventMessageJSON(t *testing.T) {
	msg := KeyEventMessage{
		Timestamp: 1234567890,
		Seq:       42,
		Key:       "W",
		Code:      17,
		Pressed:   true,
		Repeat:    false,
		Modifiers: ModifiersData{
			Shift: true,
			Ctrl:  false,
			Alt:   true,
		},
	}

	// Encode
	data, err := json.Marshal(msg)
	require.NoError(t, err)

	// Decode
	var decoded KeyEventMessage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, msg.Timestamp, decoded.Timestamp)
	assert.Equal(t, msg.Seq, decoded.Seq)
	assert.Equal(t, msg.Key, decoded.Key)
	assert.Equal(t, msg.Code, decoded.Code)
	assert.Equal(t, msg.Pressed, decoded.Pressed)
	assert.Equal(t, msg.Repeat, decoded.Repeat)
	assert.Equal(t, msg.Modifiers.Shift, decoded.Modifiers.Shift)
	assert.Equal(t, msg.Modifiers.Ctrl, decoded.Modifiers.Ctrl)
	assert.Equal(t, msg.Modifiers.Alt, decoded.Modifiers.Alt)
}

func TestModifiersConversion(t *testing.T) {
	natsModifiers := ModifiersData{
		Shift:    true,
		Ctrl:     true,
		Alt:      false,
		Meta:     false,
		CapsLock: true,
		NumLock:  false,
	}

	inputModifiers := input.Modifiers{
		Shift:    natsModifiers.Shift,
		Ctrl:     natsModifiers.Ctrl,
		Alt:      natsModifiers.Alt,
		Meta:     natsModifiers.Meta,
		CapsLock: natsModifiers.CapsLock,
		NumLock:  natsModifiers.NumLock,
	}

	assert.True(t, inputModifiers.Shift)
	assert.True(t, inputModifiers.Ctrl)
	assert.False(t, inputModifiers.Alt)
	assert.False(t, inputModifiers.Meta)
	assert.True(t, inputModifiers.CapsLock)
	assert.False(t, inputModifiers.NumLock)
}

func TestKeyStateTracking(t *testing.T) {
	pressedKeys := make(map[uint16]bool)

	// Press W (code 17)
	pressedKeys[17] = true
	assert.True(t, pressedKeys[17])
	assert.False(t, pressedKeys[31]) // S not pressed

	// Press S (code 31)
	pressedKeys[31] = true
	assert.True(t, pressedKeys[31])

	// Release W
	delete(pressedKeys, 17)
	assert.False(t, pressedKeys[17])
	assert.True(t, pressedKeys[31])

	// Count pressed keys
	assert.Equal(t, 1, len(pressedKeys))
}

func TestEventBufferOverflow(t *testing.T) {
	bufferSize := 5
	eventCh := make(chan input.KeyEvent, bufferSize)

	// Fill the buffer
	for i := 0; i < bufferSize; i++ {
		eventCh <- input.KeyEvent{
			Key:     "W",
			Code:    17,
			Pressed: true,
		}
	}

	// Verify buffer is full
	assert.Equal(t, bufferSize, len(eventCh))

	// Simulate overflow handling: drop oldest
	select {
	case <-eventCh:
		// Dropped oldest
	default:
		t.Fatal("channel should not be empty")
	}

	// Now we can add one more
	eventCh <- input.KeyEvent{
		Key:     "S",
		Code:    31,
		Pressed: true,
	}

	assert.Equal(t, bufferSize, len(eventCh))
}

func TestStaleDetection(t *testing.T) {
	threshold := 100 * time.Millisecond
	lastEventTime := time.Now()

	// Not stale immediately
	assert.False(t, time.Since(lastEventTime) > threshold)

	// Wait for threshold
	time.Sleep(threshold + 10*time.Millisecond)

	// Now stale
	assert.True(t, time.Since(lastEventTime) > threshold)
}

func TestAutoReleaseOnDisconnect(t *testing.T) {
	pressedKeys := map[uint16]bool{
		17: true, // W
		31: true, // S
		30: true, // A
	}

	// Simulate release all
	releasedEvents := []input.KeyEvent{}
	for code := range pressedKeys {
		event := input.KeyEvent{
			Code:    code,
			Pressed: false,
			Repeat:  false,
		}
		releasedEvents = append(releasedEvents, event)
	}

	// Verify we generated release events for all keys
	assert.Equal(t, 3, len(releasedEvents))

	// Clear pressed keys
	pressedKeys = make(map[uint16]bool)
	assert.Equal(t, 0, len(pressedKeys))
}

func TestFanOutMultipleConsumers(t *testing.T) {
	cfg := &Config{
		Topic:            "gorai.test.keyboard.events",
		BufferSize:       100,
		StaleThresholdMs: 5000,
	}

	r := &RemoteKeyboard{
		name:         resource.NewComponentName("gorai", "input", "test_remote_keyboard"),
		config:       cfg,
		logger:       slog.Default(),
		state:        StateConnected,
		pressedKeys:  make(map[uint16]bool),
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

	ch1, err := r.Events(ctx1)
	require.NoError(t, err)

	ch2, err := r.Events(ctx2)
	require.NoError(t, err)

	assert.NotEqual(t, ch1, ch2, "each caller should get a unique channel")

	event := input.KeyEvent{
		Key:     "W",
		Code:    17,
		Pressed: true,
	}

	r.sendEvent(event)

	// Both consumers should receive the same event
	select {
	case got := <-ch1:
		assert.Equal(t, event, got)
	case <-time.After(time.Second):
		t.Fatal("consumer 1 did not receive event")
	}

	select {
	case got := <-ch2:
		assert.Equal(t, event, got)
	case <-time.After(time.Second):
		t.Fatal("consumer 2 did not receive event")
	}
}

func TestFanOutContextCleanup(t *testing.T) {
	cfg := &Config{
		Topic:            "gorai.test.keyboard.events",
		BufferSize:       100,
		StaleThresholdMs: 5000,
	}

	r := &RemoteKeyboard{
		name:         resource.NewComponentName("gorai", "input", "test_remote_keyboard"),
		config:       cfg,
		logger:       slog.Default(),
		state:        StateConnected,
		pressedKeys:  make(map[uint16]bool),
		receiveTimes: make([]time.Time, 0, 100),
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
	}

	go func() {
		<-r.stopCh
		close(r.doneCh)
	}()

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	_, err := r.Events(ctx1)
	require.NoError(t, err)

	_, err = r.Events(ctx2)
	require.NoError(t, err)

	r.subscribersMu.Lock()
	assert.Len(t, r.subscribers, 2)
	r.subscribersMu.Unlock()

	// Cancel first consumer's context
	cancel1()
	time.Sleep(50 * time.Millisecond)

	r.subscribersMu.Lock()
	assert.Len(t, r.subscribers, 1)
	r.subscribersMu.Unlock()

	// Cleanup
	cancel2()
	time.Sleep(50 * time.Millisecond)
	close(r.stopCh)
	<-r.doneCh
}

func TestRateTracking(t *testing.T) {
	times := make([]time.Time, 0, 100)
	now := time.Now()

	// Add 10 timestamps
	for i := 0; i < 10; i++ {
		times = append(times, now.Add(-time.Duration(i*100)*time.Millisecond))
	}

	// Remove timestamps older than 1 second
	cutoff := now.Add(-time.Second)
	newTimes := []time.Time{}
	for _, t := range times {
		if !t.Before(cutoff) {
			newTimes = append(newTimes, t)
		}
	}

	// All 10 should be within 1 second
	assert.Equal(t, 10, len(newTimes))

	// Calculate rate
	rate := float64(len(newTimes))
	assert.Equal(t, 10.0, rate)
}

