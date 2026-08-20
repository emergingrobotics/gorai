package ncp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/emergingrobotics/gorai/components/sensor"
	"github.com/emergingrobotics/gorai/pkg/ncp"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/emergingrobotics/gorai/pkg/subjects"
	"github.com/nats-io/nats.go"
)

func init() {
	registry.RegisterComponent("ahrs", "ncp", New)
}

const degToRad = 3.141592653589793 / 180.0

// RemoteAHRS implements sensor.AHRS and sensor.OrientationConfigurable by
// invoking a remote AHRS capability over NCP.
type RemoteAHRS struct {
	name             resource.Name
	config           Config
	logger           *slog.Logger
	nc               *nats.Conn
	cmdSubject       string
	stateSubject     string
	timeout          time.Duration
	calibrateTimeout time.Duration

	mu       sync.RWMutex
	state    orientationCache
	stateSub *nats.Subscription
}

// orientationCache holds the last snapshot from the remote state subject.
type orientationCache struct {
	roll, pitch, yaw float64 // radians
	mounting         sensor.Mounting
	offsetDeg        [3]float64
	zeroed           bool
	calGyro          uint8
	calAccel         uint8
	calMag           uint8
}

// New creates a new NCP remote AHRS client.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)

	cfg, err := ParseConfig(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	logger := slog.Default()
	if loggerRes, err := deps.Get("logger"); err == nil {
		if l, ok := loggerRes.(*slog.Logger); ok {
			logger = l
		}
	}

	r := &RemoteAHRS{
		name:             resource.NewComponentName("gorai", "ahrs", nameStr),
		config:           cfg,
		logger:           logger.With("component", "ahrs/ncp", "name", nameStr, "target", cfg.Robot+"/"+cfg.Component),
		timeout:          time.Duration(cfg.TimeoutMs) * time.Millisecond,
		calibrateTimeout: time.Duration(cfg.CalibrateTimeoutMs) * time.Millisecond,
		state:            orientationCache{mounting: sensor.Mounting{X: "+x", Y: "+y", Z: "+z"}},
	}

	builder := subjects.NewBuilder(cfg.Robot)
	r.cmdSubject = builder.ComponentCommand(cfg.Component)
	r.stateSubject = builder.ComponentState(cfg.Component)

	if natsRes, err := deps.Get("nats"); err == nil {
		if nc, ok := natsRes.(*nats.Conn); ok {
			r.nc = nc
		}
	}
	if r.nc == nil {
		return nil, fmt.Errorf("NATS connection not available")
	}

	return r, nil
}

// Start subscribes to the remote state subject to keep the local cache fresh.
func (r *RemoteAHRS) Start(ctx context.Context) error {
	sub, err := r.nc.Subscribe(r.stateSubject, func(msg *nats.Msg) {
		var st map[string]any
		if err := json.Unmarshal(msg.Data, &st); err != nil {
			return
		}
		r.updateFromResult(st)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to remote state: %w", err)
	}
	r.stateSub = sub
	r.logger.Info("NCP remote AHRS ready", "command", r.cmdSubject)
	return nil
}

// callTimeout issues an NCP request bounded by the given timeout.
func (r *RemoteAHRS) callTimeout(ctx context.Context, method string, args map[string]any, timeout time.Duration) (*ncp.Response, error) {
	req := ncp.Request{Method: method, Args: args}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	msg, err := r.nc.Request(r.cmdSubject, data, timeout)
	if err != nil {
		return nil, fmt.Errorf("remote %q failed: %w", method, err)
	}

	var resp ncp.Response
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}
	if !resp.OK {
		return &resp, fmt.Errorf("remote %q error: %s", method, resp.Error)
	}

	r.updateFromResult(resp.Result)
	return &resp, nil
}

// call issues an NCP request using the default request timeout.
func (r *RemoteAHRS) call(ctx context.Context, method string, args map[string]any) (*ncp.Response, error) {
	return r.callTimeout(ctx, method, args, r.timeout)
}

// updateFromResult refreshes the local cache from a response/state map.
func (r *RemoteAHRS) updateFromResult(result map[string]any) {
	if result == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if v, ok := result["roll"].(float64); ok {
		r.state.roll = v * degToRad
	}
	if v, ok := result["pitch"].(float64); ok {
		r.state.pitch = v * degToRad
	}
	if v, ok := result["yaw"].(float64); ok {
		r.state.yaw = v * degToRad
	}
	if v, ok := result["zeroed"].(bool); ok {
		r.state.zeroed = v
	}
	if m, ok := result["mounting"].(map[string]any); ok {
		if s, ok := m["x"].(string); ok {
			r.state.mounting.X = s
		}
		if s, ok := m["y"].(string); ok {
			r.state.mounting.Y = s
		}
		if s, ok := m["z"].(string); ok {
			r.state.mounting.Z = s
		}
	}
	if o, ok := result["offset_deg"].(map[string]any); ok {
		if v, ok := o["roll"].(float64); ok {
			r.state.offsetDeg[0] = v
		}
		if v, ok := o["pitch"].(float64); ok {
			r.state.offsetDeg[1] = v
		}
		if v, ok := o["yaw"].(float64); ok {
			r.state.offsetDeg[2] = v
		}
	}
	if v, ok := result["calib_gyro"].(float64); ok {
		r.state.calGyro = uint8(v)
	}
	if v, ok := result["calib_accel"].(float64); ok {
		r.state.calAccel = uint8(v)
	}
	if v, ok := result["calib_mag"].(float64); ok {
		r.state.calMag = uint8(v)
	}
}

// Name returns the resource name.
func (r *RemoteAHRS) Name() resource.Name {
	return r.name
}

// Reconfigure is not supported; restart the component instead.
func (r *RemoteAHRS) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return fmt.Errorf("reconfiguration not supported, restart component instead")
}

// DoCommand forwards generic commands to the remote capability.
func (r *RemoteAHRS) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	method, _ := cmd["command"].(string)
	if method == "" {
		return nil, fmt.Errorf("command is required")
	}
	args := map[string]any{}
	for k, v := range cmd {
		if k != "command" {
			args[k] = v
		}
	}
	timeout := r.timeout
	if method == "calibrate" {
		timeout = r.calibrateTimeout
	}
	resp, err := r.callTimeout(ctx, method, args, timeout)
	if err != nil {
		return nil, err
	}
	return resp.Result, nil
}

// Close unsubscribes from remote state.
func (r *RemoteAHRS) Close(ctx context.Context) error {
	if r.stateSub != nil {
		_ = r.stateSub.Unsubscribe()
	}
	return nil
}

// --- sensor.OrientationConfigurable ---

// Calibrate triggers the remote calibration with the longer calibrate timeout.
func (r *RemoteAHRS) Calibrate(ctx context.Context, dur time.Duration) error {
	args := map[string]any{}
	if dur > 0 {
		args["duration_ms"] = float64(dur.Milliseconds())
	}
	_, err := r.callTimeout(ctx, "calibrate", args, r.calibrateTimeout)
	return err
}

// ClearZero clears the remote calibrated zero.
func (r *RemoteAHRS) ClearZero(ctx context.Context) error {
	_, err := r.call(ctx, "clear_zero", nil)
	return err
}

// SetMounting updates the remote mounting axis-remap.
func (r *RemoteAHRS) SetMounting(ctx context.Context, m sensor.Mounting) error {
	_, err := r.call(ctx, "set_mounting", map[string]any{"x": m.X, "y": m.Y, "z": m.Z})
	return err
}

// SetOffset updates the remote hardcoded orientation offset (degrees).
func (r *RemoteAHRS) SetOffset(ctx context.Context, rollDeg, pitchDeg, yawDeg float64) error {
	_, err := r.call(ctx, "set_offset", map[string]any{
		"roll_deg": rollDeg, "pitch_deg": pitchDeg, "yaw_deg": yawDeg,
	})
	return err
}

// OrientationConfig returns the cached frame configuration.
func (r *RemoteAHRS) OrientationConfig(ctx context.Context) (sensor.OrientationState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return sensor.OrientationState{
		Mounting:  r.state.mounting,
		OffsetDeg: r.state.offsetDeg,
		Zeroed:    r.state.zeroed,
	}, nil
}

// --- sensor.AHRS (values sourced from the cached state snapshot) ---

// GetEulerAngles returns the cached roll, pitch, yaw in radians.
func (r *RemoteAHRS) GetEulerAngles(ctx context.Context) (roll, pitch, yaw float64, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state.roll, r.state.pitch, r.state.yaw, nil
}

// GetCalibrationStatus returns cached calibration status for each subsystem.
func (r *RemoteAHRS) GetCalibrationStatus(ctx context.Context) (sys, gyro, accel, mag uint8, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state.calGyro, r.state.calGyro, r.state.calAccel, r.state.calMag, nil
}

// Orientation returns the orientation quaternion (x, y, z, w) derived from the
// cached Euler angles.
func (r *RemoteAHRS) Orientation(ctx context.Context) (x, y, z, w float64, err error) {
	roll, pitch, yaw, _ := r.GetEulerAngles(ctx)
	return eulerToQuat(roll, pitch, yaw)
}

// GetQuaternion returns the orientation quaternion (x, y, z, w).
func (r *RemoteAHRS) GetQuaternion(ctx context.Context) (x, y, z, w float64, err error) {
	return r.Orientation(ctx)
}

// The remote client does not relay raw inertial vectors; live accel/gyro/mag
// are available via the vehicle's telemetry (.data) stream instead.

// LinearAcceleration returns zero (not relayed over the config channel).
func (r *RemoteAHRS) LinearAcceleration(ctx context.Context) (x, y, z float64, err error) {
	return 0, 0, 0, nil
}

// AngularVelocity returns zero (not relayed over the config channel).
func (r *RemoteAHRS) AngularVelocity(ctx context.Context) (x, y, z float64, err error) {
	return 0, 0, 0, nil
}

// GetMagneticField returns zero (not relayed over the config channel).
func (r *RemoteAHRS) GetMagneticField(ctx context.Context) (x, y, z float64, err error) {
	return 0, 0, 0, nil
}

// GetLinearAccelerationWithoutGravity returns zero (not relayed).
func (r *RemoteAHRS) GetLinearAccelerationWithoutGravity(ctx context.Context) (x, y, z float64, err error) {
	return 0, 0, 0, nil
}

// GetGravityVector returns zero (not relayed).
func (r *RemoteAHRS) GetGravityVector(ctx context.Context) (x, y, z float64, err error) {
	return 0, 0, 0, nil
}

// eulerToQuat builds a quaternion (x, y, z, w) from aerospace ZYX Euler angles.
func eulerToQuat(roll, pitch, yaw float64) (x, y, z, w float64, err error) {
	cr, sr := math.Cos(roll*0.5), math.Sin(roll*0.5)
	cp, sp := math.Cos(pitch*0.5), math.Sin(pitch*0.5)
	cy, sy := math.Cos(yaw*0.5), math.Sin(yaw*0.5)
	x = sr*cp*cy - cr*sp*sy
	y = cr*sp*cy + sr*cp*sy
	z = cr*cp*sy - sr*sp*cy
	w = cr*cp*cy + sr*sp*sy
	return x, y, z, w, nil
}

// Verify interface compliance at compile time.
var (
	_ sensor.AHRS                    = (*RemoteAHRS)(nil)
	_ sensor.OrientationConfigurable = (*RemoteAHRS)(nil)
)
