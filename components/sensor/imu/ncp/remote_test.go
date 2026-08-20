package ncp

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/emergingrobotics/gorai/components/sensor"
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

// fakeAHRS is a minimal server-side orientation sensor for e2e testing.
type fakeAHRS struct {
	name         resource.Name
	mu           sync.Mutex
	yaw          float64 // radians
	mounting     sensor.Mounting
	offsetDeg    [3]float64
	zeroed       bool
	calibrations int
}

func (f *fakeAHRS) Name() resource.Name { return f.name }
func (f *fakeAHRS) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}
func (f *fakeAHRS) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	return nil, nil
}
func (f *fakeAHRS) Close(ctx context.Context) error { return nil }

func (f *fakeAHRS) LinearAcceleration(ctx context.Context) (x, y, z float64, err error) {
	return 0, 0, 0, nil
}
func (f *fakeAHRS) AngularVelocity(ctx context.Context) (x, y, z float64, err error) {
	return 0, 0, 0, nil
}
func (f *fakeAHRS) Orientation(ctx context.Context) (x, y, z, w float64, err error) {
	return 0, 0, 0, 1, nil
}
func (f *fakeAHRS) GetMagneticField(ctx context.Context) (x, y, z float64, err error) {
	return 0, 0, 0, nil
}
func (f *fakeAHRS) GetEulerAngles(ctx context.Context) (roll, pitch, yaw float64, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return 0, 0, f.yaw, nil
}
func (f *fakeAHRS) GetQuaternion(ctx context.Context) (x, y, z, w float64, err error) {
	return 0, 0, 0, 1, nil
}
func (f *fakeAHRS) GetLinearAccelerationWithoutGravity(ctx context.Context) (x, y, z float64, err error) {
	return 0, 0, 0, nil
}
func (f *fakeAHRS) GetGravityVector(ctx context.Context) (x, y, z float64, err error) {
	return 0, 0, 0, nil
}
func (f *fakeAHRS) GetCalibrationStatus(ctx context.Context) (sys, gyro, accel, mag uint8, err error) {
	return 3, 3, 2, 1, nil
}
func (f *fakeAHRS) Calibrate(ctx context.Context, dur time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calibrations++
	f.zeroed = true
	return nil
}
func (f *fakeAHRS) ClearZero(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.zeroed = false
	return nil
}
func (f *fakeAHRS) SetMounting(ctx context.Context, m sensor.Mounting) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.mounting = m
	return nil
}
func (f *fakeAHRS) SetOffset(ctx context.Context, rollDeg, pitchDeg, yawDeg float64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.offsetDeg = [3]float64{rollDeg, pitchDeg, yawDeg}
	return nil
}
func (f *fakeAHRS) OrientationConfig(ctx context.Context) (sensor.OrientationState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return sensor.OrientationState{Mounting: f.mounting, OffsetDeg: f.offsetDeg, Zeroed: f.zeroed}, nil
}

var (
	_ sensor.AHRS                    = (*fakeAHRS)(nil)
	_ sensor.OrientationConfigurable = (*fakeAHRS)(nil)
)

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

func newClient(t *testing.T, nc *nats.Conn) *RemoteAHRS {
	t.Helper()
	compAny, err := New(context.Background(), testDeps{nc: nc}, registry.Config{
		"name": "imu", "robot": "surf", "component": "imu",
	})
	if err != nil {
		t.Fatalf("New client: %v", err)
	}
	client := compAny.(*RemoteAHRS)
	if err := client.Start(context.Background()); err != nil {
		t.Fatalf("Start client: %v", err)
	}
	return client
}

func exposeFake(t *testing.T, nc *nats.Conn, fake *fakeAHRS) {
	t.Helper()
	srv := ncp.NewServer(nc, subjects.NewBuilder("surf"), nil)
	t.Cleanup(func() { _ = srv.Close() })
	if _, err := srv.Expose("imu", fake); err != nil {
		t.Fatalf("expose: %v", err)
	}
}

func TestRemoteAHRSCalibrateAndClear(t *testing.T) {
	nc := startNATS(t)
	ctx := context.Background()
	fake := &fakeAHRS{name: resource.NewComponentName("gorai", "ahrs", "imu"), mounting: sensor.Mounting{X: "+x", Y: "+y", Z: "+z"}}
	exposeFake(t, nc, fake)
	client := newClient(t, nc)

	if err := client.Calibrate(ctx, 50*time.Millisecond); err != nil {
		t.Fatalf("Calibrate: %v", err)
	}
	if fake.calibrations != 1 || !fake.zeroed {
		t.Fatalf("server calibrations=%d zeroed=%v", fake.calibrations, fake.zeroed)
	}
	// Client cache reflects zeroed via response.
	if oc, _ := client.OrientationConfig(ctx); !oc.Zeroed {
		t.Fatalf("client cache not zeroed after calibrate")
	}

	if err := client.ClearZero(ctx); err != nil {
		t.Fatalf("ClearZero: %v", err)
	}
	if fake.zeroed {
		t.Fatalf("server still zeroed after clear")
	}
}

func TestRemoteAHRSSetMountingAndOffset(t *testing.T) {
	nc := startNATS(t)
	ctx := context.Background()
	fake := &fakeAHRS{name: resource.NewComponentName("gorai", "ahrs", "imu"), yaw: math.Pi / 2}
	exposeFake(t, nc, fake)
	client := newClient(t, nc)

	if err := client.SetMounting(ctx, sensor.Mounting{X: "+y", Y: "-x", Z: "+z"}); err != nil {
		t.Fatalf("SetMounting: %v", err)
	}
	if fake.mounting.X != "+y" || fake.mounting.Y != "-x" {
		t.Fatalf("server mounting = %+v", fake.mounting)
	}
	if oc, _ := client.OrientationConfig(ctx); oc.Mounting.X != "+y" {
		t.Fatalf("client mounting cache = %+v", oc.Mounting)
	}

	if err := client.SetOffset(ctx, 1, 2, 3); err != nil {
		t.Fatalf("SetOffset: %v", err)
	}
	if fake.offsetDeg != [3]float64{1, 2, 3} {
		t.Fatalf("server offset = %v", fake.offsetDeg)
	}

	// Cached euler comes back in radians (state publishes degrees).
	roll, pitch, yaw, _ := client.GetEulerAngles(ctx)
	if math.Abs(yaw-math.Pi/2) > 1e-6 || roll != 0 || pitch != 0 {
		t.Fatalf("client euler = (%v,%v,%v), want yaw=pi/2", roll, pitch, yaw)
	}
}

func TestRemoteAHRSRequiresTarget(t *testing.T) {
	nc := startNATS(t)
	if _, err := New(context.Background(), testDeps{nc: nc}, registry.Config{"name": "x"}); err == nil {
		t.Errorf("expected error when robot/component missing")
	}
}
