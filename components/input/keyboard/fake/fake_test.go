package fake

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFanOutMultipleConsumers(t *testing.T) {
	ctx := context.Background()
	f, err := New(ctx, nil, map[string]any{"name": "test_kb"})
	require.NoError(t, err)

	kb := f.(*FakeKeyboard)

	ctx1, cancel1 := context.WithCancel(ctx)
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(ctx)
	defer cancel2()

	ch1, err := kb.Events(ctx1)
	require.NoError(t, err)

	ch2, err := kb.Events(ctx2)
	require.NoError(t, err)

	assert.NotEqual(t, ch1, ch2)

	kb.SimulateKeyPress("W")

	select {
	case ev := <-ch1:
		assert.Equal(t, "W", ev.Key)
		assert.True(t, ev.Pressed)
	case <-time.After(time.Second):
		t.Fatal("consumer 1 did not receive press event")
	}

	select {
	case ev := <-ch2:
		assert.Equal(t, "W", ev.Key)
		assert.True(t, ev.Pressed)
	case <-time.After(time.Second):
		t.Fatal("consumer 2 did not receive press event")
	}

	kb.SimulateKeyRelease("W")

	select {
	case ev := <-ch1:
		assert.Equal(t, "W", ev.Key)
		assert.False(t, ev.Pressed)
	case <-time.After(time.Second):
		t.Fatal("consumer 1 did not receive release event")
	}

	select {
	case ev := <-ch2:
		assert.Equal(t, "W", ev.Key)
		assert.False(t, ev.Pressed)
	case <-time.After(time.Second):
		t.Fatal("consumer 2 did not receive release event")
	}
}

func TestFanOutRepeatBroadcast(t *testing.T) {
	ctx := context.Background()
	f, err := New(ctx, nil, map[string]any{"name": "test_kb"})
	require.NoError(t, err)

	kb := f.(*FakeKeyboard)

	ctx1, cancel1 := context.WithCancel(ctx)
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(ctx)
	defer cancel2()

	ch1, err := kb.Events(ctx1)
	require.NoError(t, err)

	ch2, err := kb.Events(ctx2)
	require.NoError(t, err)

	kb.SimulateKeyRepeat("W")

	select {
	case ev := <-ch1:
		assert.True(t, ev.Repeat)
		assert.True(t, ev.Pressed)
	case <-time.After(time.Second):
		t.Fatal("consumer 1 did not receive repeat event")
	}

	select {
	case ev := <-ch2:
		assert.True(t, ev.Repeat)
		assert.True(t, ev.Pressed)
	case <-time.After(time.Second):
		t.Fatal("consumer 2 did not receive repeat event")
	}
}

func TestFanOutContextCleanup(t *testing.T) {
	ctx := context.Background()
	f, err := New(ctx, nil, map[string]any{"name": "test_kb"})
	require.NoError(t, err)

	kb := f.(*FakeKeyboard)

	ctx1, cancel1 := context.WithCancel(ctx)
	ctx2, cancel2 := context.WithCancel(ctx)
	defer cancel2()

	ch1, err := kb.Events(ctx1)
	require.NoError(t, err)

	ch2, err := kb.Events(ctx2)
	require.NoError(t, err)

	kb.subscribersMu.Lock()
	assert.Len(t, kb.subscribers, 2)
	kb.subscribersMu.Unlock()

	// Cancel first consumer
	cancel1()
	time.Sleep(50 * time.Millisecond)

	kb.subscribersMu.Lock()
	assert.Len(t, kb.subscribers, 1)
	kb.subscribersMu.Unlock()

	// Remaining consumer should still receive events
	kb.SimulateKeyPress("A")

	select {
	case ev := <-ch2:
		assert.Equal(t, "A", ev.Key)
	case <-time.After(time.Second):
		t.Fatal("remaining consumer did not receive event")
	}

	// ch1 should be closed
	_, ok := <-ch1
	assert.False(t, ok, "cancelled consumer's channel should be closed")
}

func TestCloseClosesAllSubscribers(t *testing.T) {
	ctx := context.Background()
	f, err := New(ctx, nil, map[string]any{"name": "test_kb"})
	require.NoError(t, err)

	kb := f.(*FakeKeyboard)

	ctx1, cancel1 := context.WithCancel(ctx)
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(ctx)
	defer cancel2()

	ch1, err := kb.Events(ctx1)
	require.NoError(t, err)

	ch2, err := kb.Events(ctx2)
	require.NoError(t, err)

	err = kb.Close(ctx)
	require.NoError(t, err)

	_, ok1 := <-ch1
	assert.False(t, ok1)

	_, ok2 := <-ch2
	assert.False(t, ok2)
}
