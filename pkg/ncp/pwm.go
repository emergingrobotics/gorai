package ncp

import (
	"context"
	"fmt"

	"github.com/emergingrobotics/gorai/components/drive"
	"github.com/emergingrobotics/gorai/components/pwm"
)

// adapterFor returns an NCP capability adapter for a component, if its type is
// a supported capability. New capability types (motor, sensor, ...) are added
// here as additional cases.
func adapterFor(component any) (*capability, bool) {
	switch c := component.(type) {
	case drive.Drive:
		return driveCapability(c), true
	case pwm.PWM:
		return pwmCapability(c), true
	case orientationSensor:
		return ahrsCapability(c), true
	default:
		return nil, false
	}
}

// pwmCapability adapts a pwm.PWM to the NCP method/state model.
func pwmCapability(p pwm.PWM) *capability {
	return &capability{
		dispatch: func(ctx context.Context, req Request) (map[string]any, error) {
			switch req.Method {
			case "set_pulse":
				v, err := argFloat(req.Args, "pulse_us")
				if err != nil {
					return nil, err
				}
				if err := p.SetPulse(ctx, v); err != nil {
					return nil, err
				}
			case "set_normalized":
				v, err := argFloat(req.Args, "value")
				if err != nil {
					return nil, err
				}
				if err := p.SetNormalized(ctx, v); err != nil {
					return nil, err
				}
			case "set_duty":
				v, err := argFloat(req.Args, "duty")
				if err != nil {
					return nil, err
				}
				if err := p.SetDuty(ctx, v); err != nil {
					return nil, err
				}
			case "enable":
				if err := p.Enable(ctx); err != nil {
					return nil, err
				}
			case "disable":
				if err := p.Disable(ctx); err != nil {
					return nil, err
				}
			case "arm":
				if err := p.Arm(ctx); err != nil {
					return nil, err
				}
			case "get_state":
				// no-op; state is returned below
			default:
				return nil, fmt.Errorf("unknown pwm method %q", req.Method)
			}
			return pwmState(ctx, p)
		},
		state: func(ctx context.Context) (map[string]any, error) {
			return pwmState(ctx, p)
		},
	}
}

// pwmState returns the current PWM snapshot.
func pwmState(ctx context.Context, p pwm.PWM) (map[string]any, error) {
	pulse, err := p.GetPulse(ctx)
	if err != nil {
		return nil, err
	}
	enabled, err := p.IsEnabled(ctx)
	if err != nil {
		return nil, err
	}
	props, err := p.Properties(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"pulse_us":     pulse,
		"enabled":      enabled,
		"frequency_hz": props.FrequencyHz,
		"min_pulse_us": props.MinPulseUs,
		"max_pulse_us": props.MaxPulseUs,
	}, nil
}

// argFloat extracts a float64 argument by key (JSON numbers decode to float64).
func argFloat(args map[string]any, key string) (float64, error) {
	v, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("missing argument %q", key)
	}
	switch n := v.(type) {
	case float64:
		return n, nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	default:
		return 0, fmt.Errorf("argument %q must be a number, got %T", key, v)
	}
}
