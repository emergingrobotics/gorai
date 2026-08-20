package ncp

import (
	"context"
	"fmt"
	"time"

	"github.com/emergingrobotics/gorai/components/sensor"
)

// orientationSensor is an AHRS whose reference frame can be reconfigured at
// runtime. Components implementing it are exposed with the ahrs capability.
type orientationSensor interface {
	sensor.AHRS
	sensor.OrientationConfigurable
}

// ahrsCapability adapts an orientation sensor to the NCP method/state model.
func ahrsCapability(a orientationSensor) *capability {
	return &capability{
		dispatch: func(ctx context.Context, req Request) (map[string]any, error) {
			switch req.Method {
			case "calibrate":
				dur := 2 * time.Second
				if v, err := argFloat(req.Args, "duration_ms"); err == nil && v > 0 {
					dur = time.Duration(v) * time.Millisecond
				}
				if err := a.Calibrate(ctx, dur); err != nil {
					return nil, err
				}
			case "clear_zero":
				if err := a.ClearZero(ctx); err != nil {
					return nil, err
				}
			case "set_mounting":
				m := sensor.Mounting{
					X: argString(req.Args, "x"),
					Y: argString(req.Args, "y"),
					Z: argString(req.Args, "z"),
				}
				if err := a.SetMounting(ctx, m); err != nil {
					return nil, err
				}
			case "set_offset":
				roll, _ := argFloat(req.Args, "roll_deg")
				pitch, _ := argFloat(req.Args, "pitch_deg")
				yaw, _ := argFloat(req.Args, "yaw_deg")
				if err := a.SetOffset(ctx, roll, pitch, yaw); err != nil {
					return nil, err
				}
			case "get_state":
				// no-op; state is returned below
			default:
				return nil, fmt.Errorf("unknown ahrs method %q", req.Method)
			}
			return ahrsState(ctx, a)
		},
		state: func(ctx context.Context) (map[string]any, error) {
			return ahrsState(ctx, a)
		},
	}
}

// ahrsState returns the current orientation snapshot plus frame configuration.
func ahrsState(ctx context.Context, a orientationSensor) (map[string]any, error) {
	const rad2deg = 180.0 / 3.141592653589793
	roll, pitch, yaw, err := a.GetEulerAngles(ctx)
	if err != nil {
		return nil, err
	}
	oc, err := a.OrientationConfig(ctx)
	if err != nil {
		return nil, err
	}
	_, calGyro, calAccel, calMag, _ := a.GetCalibrationStatus(ctx)

	return map[string]any{
		"roll":        roll * rad2deg,
		"pitch":       pitch * rad2deg,
		"yaw":         yaw * rad2deg,
		"zeroed":      oc.Zeroed,
		"mounting":    map[string]any{"x": oc.Mounting.X, "y": oc.Mounting.Y, "z": oc.Mounting.Z},
		"offset_deg":  map[string]any{"roll": oc.OffsetDeg[0], "pitch": oc.OffsetDeg[1], "yaw": oc.OffsetDeg[2]},
		"calib_gyro":  calGyro,
		"calib_accel": calAccel,
		"calib_mag":   calMag,
	}, nil
}

// argString extracts a string argument by key, returning "" when absent.
func argString(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	s, _ := args[key].(string)
	return s
}
