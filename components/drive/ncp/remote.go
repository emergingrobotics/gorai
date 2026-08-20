package ncp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/emergingrobotics/gorai/components/drive"
	"github.com/emergingrobotics/gorai/pkg/ncp"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/emergingrobotics/gorai/pkg/subjects"
	"github.com/nats-io/nats.go"
)

func init() {
	registry.RegisterComponent("drive", "ncp", New)
}

// RemoteDrive implements drive.Drive by invoking a remote drive capability
// over NCP.
type RemoteDrive struct {
	name         resource.Name
	config       Config
	logger       *slog.Logger
	nc           *nats.Conn
	cmdSubject   string
	stateSubject string
	timeout      time.Duration
	armTimeout   time.Duration

	mu       sync.RWMutex
	state    drive.State
	stateSub *nats.Subscription
}

// New creates a new NCP remote drive client.
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

	r := &RemoteDrive{
		name:       resource.NewComponentName("gorai", "drive", nameStr),
		config:     cfg,
		logger:     logger.With("component", "drive/ncp", "name", nameStr, "target", cfg.Robot+"/"+cfg.Component),
		timeout:    time.Duration(cfg.TimeoutMs) * time.Millisecond,
		armTimeout: time.Duration(cfg.ArmTimeoutMs) * time.Millisecond,
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
func (r *RemoteDrive) Start(ctx context.Context) error {
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
	r.logger.Info("NCP remote drive ready", "command", r.cmdSubject)
	return nil
}

// call issues an NCP request using the default request timeout.
func (r *RemoteDrive) call(ctx context.Context, method string, args map[string]any) (*ncp.Response, error) {
	return r.callTimeout(ctx, method, args, r.timeout)
}

// callTimeout issues an NCP request bounded by the given timeout.
func (r *RemoteDrive) callTimeout(ctx context.Context, method string, args map[string]any, timeout time.Duration) (*ncp.Response, error) {
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

// updateFromResult refreshes the local cache from a response/state map.
func (r *RemoteDrive) updateFromResult(result map[string]any) {
	if result == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if v, ok := result["surge"].(float64); ok {
		r.state.Surge = v
	}
	if v, ok := result["yaw"].(float64); ok {
		r.state.Yaw = v
	}
	if v, ok := result["left"].(float64); ok {
		r.state.Left = v
	}
	if v, ok := result["right"].(float64); ok {
		r.state.Right = v
	}
	if v, ok := result["armed"].(bool); ok {
		r.state.Armed = v
	}
	if v, ok := result["active"].(bool); ok {
		r.state.Active = v
	}
}

// Name returns the resource name.
func (r *RemoteDrive) Name() resource.Name {
	return r.name
}

// Reconfigure is not supported; restart the component instead.
func (r *RemoteDrive) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return fmt.Errorf("reconfiguration not supported, restart component instead")
}

// DoCommand forwards generic commands to the remote capability.
func (r *RemoteDrive) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
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
func (r *RemoteDrive) Close(ctx context.Context) error {
	if r.stateSub != nil {
		_ = r.stateSub.Unsubscribe()
	}
	return nil
}

// SetIntent forwards a normalized surge/yaw intent to the remote controller.
func (r *RemoteDrive) SetIntent(ctx context.Context, surge, yaw float64) error {
	_, err := r.call(ctx, "set_intent", map[string]any{"surge": surge, "yaw": yaw})
	return err
}

// Stop commands the remote controller to neutral.
func (r *RemoteDrive) Stop(ctx context.Context) error {
	_, err := r.call(ctx, "stop", nil)
	return err
}

// Arm triggers the remote arming sequence with the longer arm timeout.
func (r *RemoteDrive) Arm(ctx context.Context) error {
	_, err := r.callTimeout(ctx, "arm", nil, r.armTimeout)
	return err
}

// State returns the cached drive state.
func (r *RemoteDrive) State(ctx context.Context) (drive.State, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state, nil
}

// Verify interface compliance at compile time.
var _ drive.Drive = (*RemoteDrive)(nil)
