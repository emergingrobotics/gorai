package nws_test

import (
	"context"
	"testing"

	"github.com/emergingrobotics/gorai/nws"
	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/nats-io/nats.go"
)

// mockResource is a minimal resource for testing.
type mockResource struct {
	name resource.Name
}

func (m *mockResource) Name() resource.Name {
	return m.name
}

func (m *mockResource) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

func (m *mockResource) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	return map[string]any{"echo": cmd}, nil
}

func (m *mockResource) Close(ctx context.Context) error {
	return nil
}

func TestWrap_NilConnection(t *testing.T) {
	res := &mockResource{name: resource.NewComponentName("test", "mock", "test1")}
	_, err := nws.Wrap(nil, res)
	if err == nil {
		t.Error("expected error for nil connection")
	}
}

func TestConnect_NilConnection(t *testing.T) {
	name := resource.NewComponentName("test", "mock", "test1")
	_, err := nws.Connect(nil, name)
	if err == nil {
		t.Error("expected error for nil connection")
	}
}

func TestResourceClient_ImplementsResource(t *testing.T) {
	// Compile-time check that ResourceClient implements resource.Resource
	var _ resource.Resource = (*nws.ResourceClient)(nil)
}

// mockNATSGetter provides a mock NATS getter for testing.
type mockNATSGetter struct {
	nc *nats.Conn
}

func (m *mockNATSGetter) NATS() *nats.Conn {
	return m.nc
}

func TestWrapWithNode_NilNATS(t *testing.T) {
	mock := &mockNATSGetter{nc: nil}
	res := &mockResource{name: resource.NewComponentName("test", "mock", "test1")}

	_, err := nws.WrapWithNode(mock, res)
	if err == nil {
		t.Error("expected error for nil NATS connection")
	}
}

func TestConnectWithNode_NilNATS(t *testing.T) {
	mock := &mockNATSGetter{nc: nil}
	name := resource.NewComponentName("test", "mock", "test1")

	_, err := nws.ConnectWithNode(mock, name)
	if err == nil {
		t.Error("expected error for nil NATS connection")
	}
}

func TestResourceServer_Subject(t *testing.T) {
	// This test would require a real NATS connection
	// For now, we just verify the expected subject format
	name := resource.NewComponentName("gorai", "motor", "left")
	expectedSubject := "gorai.component.motor.left.rpc"

	// The subject is name.Subject() + ".rpc"
	if name.Subject()+".rpc" != expectedSubject {
		t.Errorf("subject = %q, want %q", name.Subject()+".rpc", expectedSubject)
	}
}

func TestResourceClient_Name(t *testing.T) {
	// Create mock client to test Name method
	name := resource.NewComponentName("gorai", "motor", "right")

	// We can't create a real client without NATS, but we can verify the interface
	if name.String() != "gorai:component:motor/right" {
		t.Errorf("name = %q, want 'gorai:component:motor/right'", name.String())
	}
}

// Note: Full integration tests would require a running NATS server.
// These tests verify the basic error handling and interface compliance.
