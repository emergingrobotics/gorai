package ncp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/emergingrobotics/gorai/components/pwm"
	"github.com/emergingrobotics/gorai/pkg/ncp"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/emergingrobotics/gorai/pkg/subjects"
	"github.com/nats-io/nats.go"
)

func init() {
	registry.RegisterComponent("pwm", "ncp", New)
}

// RemotePWM implements pwm.PWM by invoking a remote PWM capability over NCP.
type RemotePWM struct {
	name         resource.Name
	config       Config
	logger       *slog.Logger
	nc           *nats.Conn
	cmdSubject   string
	stateSubject string
	timeout      time.Duration
	armTimeout   time.Duration

	mu       sync.RWMutex
	pulseUs  float64
	enabled  bool
	stateSub *nats.Subscription
}

// New creates a new NCP remote PWM client.
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

	r := &RemotePWM{
		name:    resource.NewComponentName("gorai", "pwm", nameStr),
		config:  cfg,
		logger:  logger.With("component", "pwm/ncp", "name", nameStr, "target", cfg.Robot+"/"+cfg.Component),
		pulseUs:    cfg.InitialPulseUs,
		timeout:    time.Duration(cfg.TimeoutMs) * time.Millisecond,
		armTimeout: time.Duration(cfg.ArmTimeoutMs) * time.Millisecond,
	}

	// Build the target robot's command/state subjects.
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
func (r *RemotePWM) Start(ctx context.Context) error {
	sub, err := r.nc.Subscribe(r.stateSubject, func(msg *nats.Msg) {
		var state map[string]any
		if err := json.Unmarshal(msg.Data, &state); err != nil {
			return
		}
		r.mu.Lock()
		defer r.mu.Unlock()
		if v, ok := state["pulse_us"].(float64); ok {
			r.pulseUs = v
		}
		if v, ok := state["enabled"].(bool); ok {
			r.enabled = v
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to remote state: %w", err)
	}
	r.stateSub = sub
	r.logger.Info("NCP remote PWM ready", "command", r.cmdSubject)
	return nil
}

// call issues an NCP request using the default request timeout.
func (r *RemotePWM) call(ctx context.Context, method string, args map[string]any) (*ncp.Response, error) {
	return r.callTimeout(ctx, method, args, r.timeout)
}

// callTimeout issues an NCP request bounded by the given timeout and returns
// the decoded response.
func (r *RemotePWM) callTimeout(ctx context.Context, method string, args map[string]any, timeout time.Duration) (*ncp.Response, error) {
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

// updateFromResult refreshes the local cache from a response result map.
func (r *RemotePWM) updateFromResult(result map[string]any) {
	if result == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if v, ok := result["pulse_us"].(float64); ok {
		r.pulseUs = v
	}
	if v, ok := result["enabled"].(bool); ok {
		r.enabled = v
	}
}

// Name returns the resource name.
func (r *RemotePWM) Name() resource.Name {
	return r.name
}

// Reconfigure is not supported; restart the component instead.
func (r *RemotePWM) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return fmt.Errorf("reconfiguration not supported, restart component instead")
}

// DoCommand forwards generic commands to the remote capability.
func (r *RemotePWM) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
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
	resp, err := r.call(ctx, method, args)
	if err != nil {
		return nil, err
	}
	return resp.Result, nil
}

// Close unsubscribes from remote state.
func (r *RemotePWM) Close(ctx context.Context) error {
	if r.stateSub != nil {
		_ = r.stateSub.Unsubscribe()
	}
	return nil
}

// SetPulse sets the remote pulse width in microseconds (clamped locally).
func (r *RemotePWM) SetPulse(ctx context.Context, pulseUs float64) error {
	if pulseUs < r.config.MinPulseUs {
		pulseUs = r.config.MinPulseUs
	}
	if pulseUs > r.config.MaxPulseUs {
		pulseUs = r.config.MaxPulseUs
	}
	_, err := r.call(ctx, "set_pulse", map[string]any{"pulse_us": pulseUs})
	return err
}

// SetNormalized maps -1.0..1.0 to the configured range and sets the pulse.
func (r *RemotePWM) SetNormalized(ctx context.Context, value float64) error {
	if value < -1.0 {
		value = -1.0
	}
	if value > 1.0 {
		value = 1.0
	}
	pulseUs := r.config.CenterPulseUs() + value*r.config.PulseRangeUs()
	return r.SetPulse(ctx, pulseUs)
}

// SetDuty forwards a duty-cycle request; the remote applies its own period.
func (r *RemotePWM) SetDuty(ctx context.Context, duty float64) error {
	if duty < 0.0 {
		duty = 0.0
	}
	if duty > 1.0 {
		duty = 1.0
	}
	_, err := r.call(ctx, "set_duty", map[string]any{"duty": duty})
	return err
}

// Enable starts remote signal generation.
func (r *RemotePWM) Enable(ctx context.Context) error {
	_, err := r.call(ctx, "enable", nil)
	return err
}

// Disable stops remote signal generation.
func (r *RemotePWM) Disable(ctx context.Context) error {
	_, err := r.call(ctx, "disable", nil)
	return err
}

// Arm triggers the remote arming sequence, using the longer arm timeout since
// the remote blocks for the full sequence duration.
func (r *RemotePWM) Arm(ctx context.Context) error {
	_, err := r.callTimeout(ctx, "arm", nil, r.armTimeout)
	return err
}

// IsEnabled returns the cached enabled state.
func (r *RemotePWM) IsEnabled(ctx context.Context) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.enabled, nil
}

// GetPulse returns the cached pulse width in microseconds.
func (r *RemotePWM) GetPulse(ctx context.Context) (float64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.pulseUs, nil
}

// GetNormalized returns the cached pulse as a -1.0..1.0 value.
func (r *RemotePWM) GetNormalized(ctx context.Context) (float64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rangeUs := r.config.PulseRangeUs()
	if rangeUs == 0 {
		return 0, nil
	}
	return (r.pulseUs - r.config.CenterPulseUs()) / rangeUs, nil
}

// Properties returns the client-side PWM configuration.
func (r *RemotePWM) Properties(ctx context.Context) (pwm.Properties, error) {
	return pwm.Properties{
		MinPulseUs: r.config.MinPulseUs,
		MaxPulseUs: r.config.MaxPulseUs,
		Mode:       "remote",
	}, nil
}

// Verify interface compliance at compile time.
var _ pwm.PWM = (*RemotePWM)(nil)
