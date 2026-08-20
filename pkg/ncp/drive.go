package ncp

import (
	"context"
	"fmt"

	"github.com/emergingrobotics/gorai/components/drive"
)

// driveCapability adapts a drive.Drive to the NCP method/state model.
func driveCapability(d drive.Drive) *capability {
	return &capability{
		dispatch: func(ctx context.Context, req Request) (map[string]any, error) {
			switch req.Method {
			case "set_intent":
				surge, err := argFloat(req.Args, "surge")
				if err != nil {
					return nil, err
				}
				yaw, err := argFloat(req.Args, "yaw")
				if err != nil {
					return nil, err
				}
				if err := d.SetIntent(ctx, surge, yaw); err != nil {
					return nil, err
				}
			case "stop":
				if err := d.Stop(ctx); err != nil {
					return nil, err
				}
			case "arm":
				if err := d.Arm(ctx); err != nil {
					return nil, err
				}
			case "get_state":
				// no-op; state is returned below
			default:
				return nil, fmt.Errorf("unknown drive method %q", req.Method)
			}
			return driveState(ctx, d)
		},
		state: func(ctx context.Context) (map[string]any, error) {
			return driveState(ctx, d)
		},
	}
}

// driveState returns the current drive snapshot.
func driveState(ctx context.Context, d drive.Drive) (map[string]any, error) {
	st, err := d.State(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"surge":  st.Surge,
		"yaw":    st.Yaw,
		"left":   st.Left,
		"right":  st.Right,
		"armed":  st.Armed,
		"active": st.Active,
	}, nil
}
