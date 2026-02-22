package config

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeValidConfig(t *testing.T, path string, robotName string) {
	t.Helper()
	cfg := map[string]any{
		"version": "2",
		"robot":   map[string]any{"name": robotName},
		"components": []any{
			map[string]any{"name": "c1", "type": "switch", "model": "tasmota", "attributes": map[string]any{"address": "1.2.3.4"}},
		},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))
}

func TestWatcher_FileModified(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "robot.json")
	writeValidConfig(t, configPath, "original")

	received := make(chan *RDL, 1)
	w, err := NewWatcher(configPath, func(cfg *RDL) {
		received <- cfg
	}, slog.Default())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, w.Start(ctx))

	// Wait for watcher to be ready
	time.Sleep(100 * time.Millisecond)

	// Modify the config
	writeValidConfig(t, configPath, "modified")

	select {
	case cfg := <-received:
		assert.Equal(t, "modified", cfg.Robot.Name)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for watcher callback")
	}
}

func TestWatcher_DebounceMultipleRapidWrites(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "robot.json")
	writeValidConfig(t, configPath, "initial")

	var mu sync.Mutex
	callCount := 0
	received := make(chan *RDL, 10)
	w, err := NewWatcher(configPath, func(cfg *RDL) {
		mu.Lock()
		callCount++
		mu.Unlock()
		received <- cfg
	}, slog.Default())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, w.Start(ctx))

	time.Sleep(100 * time.Millisecond)

	// Write 5 times rapidly
	for i := 0; i < 5; i++ {
		writeValidConfig(t, configPath, "rapid")
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for debounce to settle
	time.Sleep(2 * time.Second)

	// Should have received exactly 1 callback (debounce collapsed the writes)
	mu.Lock()
	count := callCount
	mu.Unlock()
	assert.Equal(t, 1, count, "expected exactly 1 callback after rapid writes")
}

func TestWatcher_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "robot.json")
	writeValidConfig(t, configPath, "valid")

	received := make(chan *RDL, 1)
	w, err := NewWatcher(configPath, func(cfg *RDL) {
		received <- cfg
	}, slog.Default())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, w.Start(ctx))

	time.Sleep(100 * time.Millisecond)

	// Write invalid JSON
	require.NoError(t, os.WriteFile(configPath, []byte("{{{invalid"), 0644))

	// Should NOT receive a callback
	select {
	case <-received:
		t.Fatal("should not have received callback for invalid JSON")
	case <-time.After(2 * time.Second):
		// Expected: no callback for invalid JSON
	}
}

func TestWatcher_ContextCancelled(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "robot.json")
	writeValidConfig(t, configPath, "test")

	received := make(chan *RDL, 1)
	w, err := NewWatcher(configPath, func(cfg *RDL) {
		received <- cfg
	}, slog.Default())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	require.NoError(t, w.Start(ctx))

	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()
	time.Sleep(200 * time.Millisecond)

	// Write to file after cancellation
	writeValidConfig(t, configPath, "after-cancel")

	// Should NOT receive callback
	select {
	case <-received:
		t.Fatal("should not have received callback after context cancellation")
	case <-time.After(2 * time.Second):
		// Expected: no callback after cancellation
	}
}

func TestWatcher_EditorRename(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "robot.json")
	writeValidConfig(t, configPath, "original")

	received := make(chan *RDL, 1)
	w, err := NewWatcher(configPath, func(cfg *RDL) {
		received <- cfg
	}, slog.Default())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, w.Start(ctx))

	time.Sleep(100 * time.Millisecond)

	// Simulate editor write-to-temp-then-rename
	tmpPath := filepath.Join(dir, "robot.json.tmp")
	writeValidConfig(t, tmpPath, "renamed")
	require.NoError(t, os.Rename(tmpPath, configPath))

	select {
	case cfg := <-received:
		assert.Equal(t, "renamed", cfg.Robot.Name)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for watcher callback after rename")
	}
}
