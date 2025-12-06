// Package testutil provides testing utilities for Gorai.
package testutil

import (
	"testing"
	"time"
)

// Eventually retries a condition until it succeeds or times out.
func Eventually(t *testing.T, condition func() bool, timeout time.Duration, interval time.Duration) bool {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(interval)
	}
	return false
}

// RequireEventually fails the test if the condition doesn't succeed.
func RequireEventually(t *testing.T, condition func() bool, timeout time.Duration, interval time.Duration, msg string) {
	t.Helper()

	if !Eventually(t, condition, timeout, interval) {
		t.Fatalf("condition not met within %v: %s", timeout, msg)
	}
}

// MockDeps implements registry.Dependencies for testing.
type MockDeps struct {
	resources map[string]any
}

// NewMockDeps creates a new MockDeps.
func NewMockDeps() *MockDeps {
	return &MockDeps{
		resources: make(map[string]any),
	}
}

// Add adds a resource to the mock.
func (d *MockDeps) Add(name string, resource any) {
	d.resources[name] = resource
}

// Get retrieves a resource by name.
func (d *MockDeps) Get(name string) (any, error) {
	if r, ok := d.resources[name]; ok {
		return r, nil
	}
	return nil, nil
}

// GetByType retrieves resources by type.
func (d *MockDeps) GetByType(subtype string) ([]any, error) {
	// Simplified - real implementation would filter by type
	var results []any
	for _, r := range d.resources {
		results = append(results, r)
	}
	return results, nil
}
