package ncp

import (
	"context"
	"sync"
	"testing"

	"github.com/emergingrobotics/gorai/components/drive"
	"github.com/emergingrobotics/gorai/pkg/embeddednats"
	"github.com/emergingrobotics/gorai/pkg/ncp"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/emergingrobotics/gorai/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// testDeps provides a nats dependency for component construction.
type testDeps struct{ nc *nats.Conn }

func (d testDeps) Get(name string) (any, error) {
	if name == "nats" {
		return d.nc, nil
	}
	return nil, nil
}
func (d testDeps) GetByType(subtype string) ([]any, error) { return nil, nil }

// fakeDrive is a minimal server-side drive.Drive for e2e testing.
type fakeDrive struct {
	name resource.Name
	mu   sync.Mutex
	st   drive.State
	arms int
	stops int
}

func (f *fakeDrive) Name() resource.Name { return f.name }
func (f *fakeDrive) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}
func (f *fakeDrive) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	return nil, nil
}
func (f *fakeDrive) Close(ctx context.Context) error { return nil }
func (f *fakeDrive) SetIntent(ctx context.Context, surge, yaw float64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.st.Surge, f.st.Yaw = surge, yaw
	f.st.Left, f.st.Right = surge+yaw, surge-yaw
	f.st.Active = true
	return nil
}
func (f *fakeDrive) Stop(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stops++
	f.st = drive.State{Armed: f.st.Armed}
	return nil
}
func (f *fakeDrive) Arm(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.arms++
	f.st.Armed = true
	return nil
}
func (f *fakeDrive) State(ctx context.Context) (drive.State, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.st, nil
}

var _ drive.Drive = (*fakeDrive)(nil)

func startNATS(t *testing.T) *nats.Conn {
	t.Helper()
	srv, err := embeddednats.New(embeddednats.Config{Host: "127.0.0.1", Port: -1})
	if err != nil {
		t.Fatalf("embedded nats: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("start nats: %v", err)
	}
	t.Cleanup(srv.Shutdown)
	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(nc.Close)
	return nc
}

func newClient(t *testing.T, nc *nats.Conn) *RemoteDrive {
	t.Helper()
	compAny, err := New(context.Background(), testDeps{nc: nc}, registry.Config{
		"name": "drive", "robot": "surf", "component": "drive",
	})
	if err != nil {
		t.Fatalf("New client: %v", err)
	}
	client := compAny.(*RemoteDrive)
	if err := client.Start(context.Background()); err != nil {
		t.Fatalf("Start client: %v", err)
	}
	return client
}

func TestRemoteDriveSetIntent(t *testing.T) {
	nc := startNATS(t)
	ctx := context.Background()

	fake := &fakeDrive{name: resource.NewComponentName("gorai", "drive", "drive")}
	srv := ncp.NewServer(nc, subjects.NewBuilder("surf"), nil)
	t.Cleanup(func() { _ = srv.Close() })
	if _, err := srv.Expose("drive", fake); err != nil {
		t.Fatalf("expose: %v", err)
	}

	client := newClient(t, nc)

	if err := client.SetIntent(ctx, 0.6, -0.2); err != nil {
		t.Fatalf("SetIntent: %v", err)
	}
	st, _ := fake.State(ctx)
	if st.Surge != 0.6 || st.Yaw != -0.2 {
		t.Errorf("server intent = %+v, want surge 0.6 yaw -0.2", st)
	}
	// Client cache updates from the response.
	cst, _ := client.State(ctx)
	if cst.Surge != 0.6 {
		t.Errorf("client cached surge = %v, want 0.6", cst.Surge)
	}
}

func TestRemoteDriveArmAndStop(t *testing.T) {
	nc := startNATS(t)
	ctx := context.Background()

	fake := &fakeDrive{name: resource.NewComponentName("gorai", "drive", "drive")}
	srv := ncp.NewServer(nc, subjects.NewBuilder("surf"), nil)
	t.Cleanup(func() { _ = srv.Close() })
	if _, err := srv.Expose("drive", fake); err != nil {
		t.Fatalf("expose: %v", err)
	}

	client := newClient(t, nc)

	if err := client.Arm(ctx); err != nil {
		t.Fatalf("Arm: %v", err)
	}
	if fake.arms != 1 {
		t.Errorf("expected 1 arm, got %d", fake.arms)
	}
	if err := client.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if fake.stops != 1 {
		t.Errorf("expected 1 stop, got %d", fake.stops)
	}
}

func TestRemoteDriveRequiresTarget(t *testing.T) {
	nc := startNATS(t)
	if _, err := New(context.Background(), testDeps{nc: nc}, registry.Config{"name": "x"}); err == nil {
		t.Errorf("expected error when robot/component missing")
	}
}
