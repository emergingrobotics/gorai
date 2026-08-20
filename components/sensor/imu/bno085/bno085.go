package bno085

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/emergingrobotics/gorai/components/sensor"
	"github.com/emergingrobotics/gorai/driver/i2c"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("ahrs", "bno085", New)
}

// deviceOpener opens an I2C device. Overridable in tests.
type deviceOpener func(bus string, addr uint16) (i2c.Device, error)

// degToRad converts degrees to radians.
const degToRad = 3.141592653589793 / 180.0

// AHRS is a BNO085 driver exposing fused orientation over I2C/SHTP.
type AHRS struct {
	name   resource.Name
	config Config
	logger *slog.Logger
	opener deviceOpener

	dev i2c.Device
	seq byte // outgoing control-channel sequence number

	mu    sync.RWMutex
	state imuState

	// tmu guards the runtime frame configuration below.
	tmu         sync.RWMutex
	mounting    sensor.Mounting
	mountMatrix [3][3]float64
	mountQuat   quat
	offsetDeg   [3]float64
	offsetQuat  quat
	zeroQuat    *quat // set by calibration; overrides offsetQuat when non-nil

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a BNO085 AHRS component.
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

	// Validate already ensured the mounting is a valid right-handed remap.
	mat, _ := mountingMatrix(cfg.Mounting)

	a := &AHRS{
		name:        resource.NewComponentName("gorai", "ahrs", nameStr),
		config:      cfg,
		logger:      logger.With("component", "ahrs/bno085", "name", nameStr),
		opener:      defaultOpener,
		mounting:    cfg.Mounting,
		mountMatrix: mat,
		mountQuat:   matrixToQuat(mat),
		offsetDeg:   cfg.OffsetDeg,
		offsetQuat:  eulerToQuat(cfg.OffsetDeg[0]*degToRad, cfg.OffsetDeg[1]*degToRad, cfg.OffsetDeg[2]*degToRad),
	}
	return a, nil
}

// frame returns a consistent snapshot of the active mounting and zero reference.
func (a *AHRS) frame() (mat [3][3]float64, mount quat, ref quat) {
	a.tmu.RLock()
	defer a.tmu.RUnlock()
	mat = a.mountMatrix
	mount = a.mountQuat
	if a.zeroQuat != nil {
		ref = *a.zeroQuat
	} else {
		ref = a.offsetQuat
	}
	return
}

// orient applies the mounting and zero reference to a raw sensor quaternion,
// returning the body orientation relative to the reference frame.
func orient(raw, mount, ref quat) quat {
	body := raw.mul(mount.conjugate())
	return ref.conjugate().mul(body).normalize()
}

// Start opens the device, enables reports, and begins the read loop.
func (a *AHRS) Start(ctx context.Context) error {
	a.logger.Debug("BNO085 bringup: opening device", "bus", a.config.Bus, "address", fmt.Sprintf("0x%02X", a.config.Address))
	dev, err := a.opener(a.config.Bus, a.config.Address)
	if err != nil {
		return fmt.Errorf("failed to open BNO085: %w", err)
	}
	a.dev = dev

	// Drain whatever the device advertises immediately after power-on (SHTP
	// advertisement + reset-complete) so we can see it while debugging.
	a.logger.Debug("BNO085 bringup: draining startup packets")
	for i := 0; i < 4; i++ {
		res, err := a.readPacket(ctx)
		if err != nil {
			a.logger.Debug("BNO085 bringup: startup read", "attempt", i, "error", err)
			break
		}
		if res.channel < 0 {
			a.logger.Debug("BNO085 bringup: startup read empty/bad header", "attempt", i)
			continue
		}
		a.logger.Debug("BNO085 bringup: startup packet",
			"attempt", i, "channel", res.channel, "length", res.length,
			"cargo", fmt.Sprintf("% x", res.cargo))
	}

	// Enable the reports we consume.
	intervalUs := uint32(a.config.ReportIntervalMs * 1000)
	for _, id := range []byte{reportRotationVec, reportAccelerometer, reportGyroscope, reportMagnetometer} {
		cmd := setFeatureCommand(id, intervalUs)
		a.logger.Debug("BNO085 bringup: enabling feature report",
			"report_id", fmt.Sprintf("0x%02X", id),
			"interval_us", intervalUs,
			"seq", a.seq,
			"command", fmt.Sprintf("% x", cmd))
		if err := a.writePacket(ctx, chControl, cmd); err != nil {
			return fmt.Errorf("failed to enable report 0x%02X: %w", id, err)
		}
	}

	loopCtx, cancel := context.WithCancel(ctx)
	a.cancel = cancel
	a.wg.Add(1)
	go a.readLoop(loopCtx)

	a.logger.Info("BNO085 started", "bus", a.config.Bus, "address", fmt.Sprintf("0x%02X", a.config.Address), "interval_ms", a.config.ReportIntervalMs)
	return nil
}

// writePacket frames a cargo with a 4-byte SHTP header and writes it.
func (a *AHRS) writePacket(ctx context.Context, channel byte, cargo []byte) error {
	total := len(cargo) + 4
	pkt := make([]byte, total)
	pkt[0] = byte(total & 0xFF)
	pkt[1] = byte((total >> 8) & 0xFF)
	pkt[2] = channel
	pkt[3] = a.seq
	a.seq++
	copy(pkt[4:], cargo)
	return a.dev.Write(ctx, pkt)
}

// readResult summarizes a single SHTP packet read for diagnostics.
type readResult struct {
	channel int    // SHTP channel, or -1 when no valid packet was read
	length  int    // cargo length in bytes
	reports int    // number of input reports decoded from the cargo
	cargo   []byte // raw cargo bytes (nil when none)
}

// readLoop reads SHTP packets and decodes sensor reports. It also emits a
// once-per-second debug summary of the decoded reading and the read statistics
// so the hardware bring-up can be diagnosed without flooding the logs.
func (a *AHRS) readLoop(ctx context.Context) {
	defer a.wg.Done()

	readTicker := time.NewTicker(time.Duration(a.config.ReportIntervalMs) * time.Millisecond)
	defer readTicker.Stop()
	logTicker := time.NewTicker(time.Second)
	defer logTicker.Stop()

	var packets, reports, readErrs uint64
	chanCounts := map[int]uint64{}
	var lastErr error
	var lastCargo []byte
	lastCargoChan := -1

	for {
		select {
		case <-ctx.Done():
			return
		case <-readTicker.C:
			res, err := a.readPacket(ctx)
			if err != nil {
				readErrs++
				lastErr = err
				a.logger.Debug("BNO085 read error", "error", err)
				continue
			}
			if res.channel < 0 {
				continue
			}
			packets++
			chanCounts[res.channel]++
			reports += uint64(res.reports)
			if len(res.cargo) > 0 {
				lastCargo = res.cargo
				lastCargoChan = res.channel
			}
		case <-logTicker.C:
			s := a.snapshot()
			roll, pitch, yaw := quaternionToEuler(s.qi, s.qj, s.qk, s.qr)
			const rad2deg = 180.0 / 3.141592653589793
			a.logger.Debug("BNO085 reading (1s)",
				"roll_deg", roll*rad2deg,
				"pitch_deg", pitch*rad2deg,
				"yaw_deg", yaw*rad2deg,
				"quat", fmt.Sprintf("x=%.3f y=%.3f z=%.3f w=%.3f", s.qi, s.qj, s.qk, s.qr),
				"accel", fmt.Sprintf("x=%.3f y=%.3f z=%.3f", s.ax, s.ay, s.az),
				"gyro", fmt.Sprintf("x=%.3f y=%.3f z=%.3f", s.gx, s.gy, s.gz),
				"mag", fmt.Sprintf("x=%.3f y=%.3f z=%.3f", s.mx, s.my, s.mz),
				"calib_accel", s.calAccel, "calib_gyro", s.calGyro, "calib_mag", s.calMag)

			lastErrStr := ""
			if lastErr != nil {
				lastErrStr = lastErr.Error()
			}
			a.logger.Debug("BNO085 read stats (1s)",
				"packets", packets,
				"reports_decoded", reports,
				"read_errors", readErrs,
				"channels", fmt.Sprintf("%v", chanCounts),
				"last_error", lastErrStr,
				"last_cargo_channel", lastCargoChan,
				"last_cargo", fmt.Sprintf("% x", lastCargo))

			packets, reports, readErrs = 0, 0, 0
			chanCounts = map[int]uint64{}
			lastErr = nil
			lastCargo = nil
			lastCargoChan = -1
		}
	}
}

// maxPacketBytes caps a single SHTP packet read so a corrupt length field can
// never trigger an enormous allocation/read. Sensor-report packets are small.
const maxPacketBytes = 512

// readPacket reads one SHTP packet and decodes any input reports it carries. A
// channel of -1 indicates no valid packet (empty read or bad header).
//
// On the BNO08x each I2C read is a fresh transaction that restarts at the SHTP
// header, so reading the 4-byte header does NOT consume it. We first read the
// header to learn the packet length, then read the full packet (header + cargo)
// in a single transaction and slice the header off. Reading only the cargo in a
// second transaction re-reads the header and never advances the stream.
func (a *AHRS) readPacket(ctx context.Context) (readResult, error) {
	head, err := a.dev.Read(ctx, 4)
	if err != nil {
		return readResult{channel: -1}, err
	}
	h, ok := parseHeader(head)
	if !ok {
		a.logger.Debug("BNO085 bad header", "bytes", fmt.Sprintf("% x", head))
		return readResult{channel: -1}, nil
	}
	if h.length == 0 {
		return readResult{channel: int(h.channel)}, nil
	}

	total := h.length + 4 // full packet length including the 4-byte header
	if total > maxPacketBytes {
		a.logger.Debug("BNO085 oversized packet, skipping", "channel", h.channel, "total", total)
		return readResult{channel: int(h.channel), length: h.length}, nil
	}

	full, err := a.dev.Read(ctx, total)
	if err != nil {
		return readResult{channel: int(h.channel), length: h.length}, err
	}
	cargo := full[4:]

	reports := 0
	if h.channel == chInput || h.channel == chWake || h.channel == chGyro {
		a.mu.Lock()
		reports = parseInputReports(cargo, &a.state)
		a.mu.Unlock()
	}
	return readResult{channel: int(h.channel), length: h.length, reports: reports, cargo: cargo}, nil
}

// snapshot returns a copy of the current state.
func (a *AHRS) snapshot() imuState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state
}

// Name returns the resource name.
func (a *AHRS) Name() resource.Name {
	return a.name
}

// Reconfigure is not supported; restart the component instead.
func (a *AHRS) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return fmt.Errorf("reconfiguration not supported, restart component instead")
}

// DoCommand exposes state and orientation-configuration commands.
func (a *AHRS) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	name, _ := cmd["command"].(string)
	switch name {
	case "get_state":
		return a.Readings(ctx)
	case "calibrate":
		dur := 2 * time.Second
		if v, ok := toFloatCfg(cmd["duration_ms"]); ok && v > 0 {
			dur = time.Duration(v) * time.Millisecond
		}
		if err := a.Calibrate(ctx, dur); err != nil {
			return nil, err
		}
		return a.Readings(ctx)
	case "clear_zero":
		if err := a.ClearZero(ctx); err != nil {
			return nil, err
		}
		return a.Readings(ctx)
	case "set_mounting":
		m := sensor.Mounting{}
		m.X, _ = cmd["x"].(string)
		m.Y, _ = cmd["y"].(string)
		m.Z, _ = cmd["z"].(string)
		if err := a.SetMounting(ctx, m); err != nil {
			return nil, err
		}
		return a.Readings(ctx)
	case "set_offset":
		roll, _ := toFloatCfg(cmd["roll_deg"])
		pitch, _ := toFloatCfg(cmd["pitch_deg"])
		yaw, _ := toFloatCfg(cmd["yaw_deg"])
		if err := a.SetOffset(ctx, roll, pitch, yaw); err != nil {
			return nil, err
		}
		return a.Readings(ctx)
	default:
		return nil, fmt.Errorf("unknown command: %v", cmd["command"])
	}
}

// Calibrate samples the mounted body orientation over dur and stores the
// average as the zero reference, overriding any configured offset.
func (a *AHRS) Calibrate(ctx context.Context, dur time.Duration) error {
	if dur <= 0 {
		dur = 2 * time.Second
	}
	interval := time.Duration(a.config.ReportIntervalMs) * time.Millisecond
	if interval <= 0 {
		interval = 20 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	deadline := time.NewTimer(dur)
	defer deadline.Stop()

	var samples []quat
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			if len(samples) == 0 {
				return fmt.Errorf("no orientation samples captured (is the IMU reporting?)")
			}
			avg := averageQuats(samples)
			a.tmu.Lock()
			z := avg
			a.zeroQuat = &z
			a.tmu.Unlock()
			a.logger.Info("orientation calibrated", "samples", len(samples))
			return nil
		case <-ticker.C:
			s := a.snapshot()
			if !s.haveQuat {
				continue
			}
			_, mount, _ := a.frame()
			body := quat{s.qi, s.qj, s.qk, s.qr}.mul(mount.conjugate()).normalize()
			samples = append(samples, body)
		}
	}
}

// ClearZero removes a calibrated zero, reverting to the configured offset.
func (a *AHRS) ClearZero(ctx context.Context) error {
	a.tmu.Lock()
	a.zeroQuat = nil
	a.tmu.Unlock()
	a.logger.Info("orientation zero cleared")
	return nil
}

// SetMounting updates the axis-remap describing the sensor's mounting.
func (a *AHRS) SetMounting(ctx context.Context, m sensor.Mounting) error {
	mat, err := mountingMatrix(m)
	if err != nil {
		return err
	}
	a.tmu.Lock()
	a.mounting = m
	a.mountMatrix = mat
	a.mountQuat = matrixToQuat(mat)
	a.tmu.Unlock()
	a.logger.Info("mounting updated", "x", m.X, "y", m.Y, "z", m.Z)
	return nil
}

// SetOffset updates the hardcoded orientation offset (degrees). This does not
// clear a calibrated zero; the offset only applies when no zero is set.
func (a *AHRS) SetOffset(ctx context.Context, rollDeg, pitchDeg, yawDeg float64) error {
	a.tmu.Lock()
	a.offsetDeg = [3]float64{rollDeg, pitchDeg, yawDeg}
	a.offsetQuat = eulerToQuat(rollDeg*degToRad, pitchDeg*degToRad, yawDeg*degToRad)
	a.tmu.Unlock()
	a.logger.Info("orientation offset updated", "roll_deg", rollDeg, "pitch_deg", pitchDeg, "yaw_deg", yawDeg)
	return nil
}

// OrientationConfig returns the current frame configuration.
func (a *AHRS) OrientationConfig(ctx context.Context) (sensor.OrientationState, error) {
	a.tmu.RLock()
	defer a.tmu.RUnlock()
	return sensor.OrientationState{
		Mounting:  a.mounting,
		OffsetDeg: a.offsetDeg,
		Zeroed:    a.zeroQuat != nil,
	}, nil
}

// Close stops the read loop.
func (a *AHRS) Close(ctx context.Context) error {
	if a.cancel != nil {
		a.cancel()
	}
	done := make(chan struct{})
	go func() { a.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
	}
	return nil
}

// Readings returns the latest AHRS readings in the body frame, including Euler
// angles in degrees, with the mounting remap and zero reference applied.
func (a *AHRS) Readings(ctx context.Context) (map[string]any, error) {
	s := a.snapshot()
	mat, mount, ref := a.frame()

	out := orient(quat{s.qi, s.qj, s.qk, s.qr}, mount, ref)
	roll, pitch, yaw := quaternionToEuler(out.x, out.y, out.z, out.w)
	ax, ay, az := applyMatrix(mat, s.ax, s.ay, s.az)
	gx, gy, gz := applyMatrix(mat, s.gx, s.gy, s.gz)
	mx, my, mz := applyMatrix(mat, s.mx, s.my, s.mz)

	const rad2deg = 180.0 / 3.141592653589793
	a.tmu.RLock()
	zeroed := a.zeroQuat != nil
	a.tmu.RUnlock()
	return map[string]any{
		"quaternion":  map[string]float64{"x": out.x, "y": out.y, "z": out.z, "w": out.w},
		"roll":        roll * rad2deg,
		"pitch":       pitch * rad2deg,
		"yaw":         yaw * rad2deg,
		"accel":       map[string]float64{"x": ax, "y": ay, "z": az},
		"gyro":        map[string]float64{"x": gx, "y": gy, "z": gz},
		"mag":         map[string]float64{"x": mx, "y": my, "z": mz},
		"accel_x":     ax,
		"accel_y":     ay,
		"accel_z":     az,
		"gyro_x":      gx,
		"gyro_y":      gy,
		"gyro_z":      gz,
		"calib_accel": s.calAccel,
		"calib_gyro":  s.calGyro,
		"calib_mag":   s.calMag,
		"zeroed":      zeroed,
	}, nil
}

// LinearAcceleration returns body-frame acceleration in m/s² (x, y, z).
func (a *AHRS) LinearAcceleration(ctx context.Context) (x, y, z float64, err error) {
	s := a.snapshot()
	mat, _, _ := a.frame()
	x, y, z = applyMatrix(mat, s.ax, s.ay, s.az)
	return x, y, z, nil
}

// AngularVelocity returns body-frame rotation rate in rad/s (x, y, z).
func (a *AHRS) AngularVelocity(ctx context.Context) (x, y, z float64, err error) {
	s := a.snapshot()
	mat, _, _ := a.frame()
	x, y, z = applyMatrix(mat, s.gx, s.gy, s.gz)
	return x, y, z, nil
}

// Orientation returns the body orientation quaternion (x, y, z, w) with the
// mounting remap and zero reference applied.
func (a *AHRS) Orientation(ctx context.Context) (x, y, z, w float64, err error) {
	s := a.snapshot()
	_, mount, ref := a.frame()
	out := orient(quat{s.qi, s.qj, s.qk, s.qr}, mount, ref)
	return out.x, out.y, out.z, out.w, nil
}

// GetMagneticField returns body-frame magnetic field in µT (x, y, z).
func (a *AHRS) GetMagneticField(ctx context.Context) (x, y, z float64, err error) {
	s := a.snapshot()
	mat, _, _ := a.frame()
	x, y, z = applyMatrix(mat, s.mx, s.my, s.mz)
	return x, y, z, nil
}

// GetEulerAngles returns body-frame roll, pitch, yaw in radians.
func (a *AHRS) GetEulerAngles(ctx context.Context) (roll, pitch, yaw float64, err error) {
	s := a.snapshot()
	_, mount, ref := a.frame()
	out := orient(quat{s.qi, s.qj, s.qk, s.qr}, mount, ref)
	roll, pitch, yaw = quaternionToEuler(out.x, out.y, out.z, out.w)
	return roll, pitch, yaw, nil
}

// GetQuaternion returns the orientation quaternion (x, y, z, w).
func (a *AHRS) GetQuaternion(ctx context.Context) (x, y, z, w float64, err error) {
	return a.Orientation(ctx)
}

// GetLinearAccelerationWithoutGravity returns acceleration; gravity removal is
// not separately reported here, so it mirrors LinearAcceleration.
func (a *AHRS) GetLinearAccelerationWithoutGravity(ctx context.Context) (x, y, z float64, err error) {
	return a.LinearAcceleration(ctx)
}

// GetGravityVector is not separately reported; returns zeros.
func (a *AHRS) GetGravityVector(ctx context.Context) (x, y, z float64, err error) {
	return 0, 0, 0, nil
}

// GetCalibrationStatus returns calibration status (0-3) for each subsystem.
// System status is reported as the rotation-vector accuracy bucket.
func (a *AHRS) GetCalibrationStatus(ctx context.Context) (sys, gyro, accel, mag uint8, err error) {
	s := a.snapshot()
	return s.calGyro, s.calGyro, s.calAccel, s.calMag, nil
}

// Verify interface compliance.
var _ sensor.AHRS = (*AHRS)(nil)
var _ sensor.OrientationConfigurable = (*AHRS)(nil)
