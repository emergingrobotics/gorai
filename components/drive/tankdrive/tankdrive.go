package tankdrive

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/emergingrobotics/gorai/components/drive"
	"github.com/emergingrobotics/gorai/components/pwm"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("drive", "tank-drive", New)
}

// Controller mixes a surge/yaw intent into two PWM actuators on a fixed-rate
// control loop.
type Controller struct {
	name   resource.Name
	config Config
	logger *slog.Logger

	left  pwm.PWM
	right pwm.PWM

	mu       sync.RWMutex
	surge    float64
	yaw      float64
	curLeft  float64
	curRight float64
	armed    bool
	active   bool
	lastCmd  time.Time

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a tank-drive controller. It resolves its two PWM actuators from
// dependencies (declare them in the RDL depends_on for correct ordering).
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

	left, err := resolvePWM(deps, cfg.Left)
	if err != nil {
		return nil, err
	}
	right, err := resolvePWM(deps, cfg.Right)
	if err != nil {
		return nil, err
	}

	c := &Controller{
		name:   resource.NewComponentName("gorai", "drive", nameStr),
		config: cfg,
		logger: logger.With("component", "drive/tank-drive", "name", nameStr),
		left:   left,
		right:  right,
	}
	return c, nil
}

// resolvePWM resolves a dependency by name and asserts it to pwm.PWM.
func resolvePWM(deps registry.Dependencies, name string) (pwm.PWM, error) {
	res, err := deps.Get(name)
	if err != nil {
		return nil, fmt.Errorf("PWM dependency %q not available: %w", name, err)
	}
	p, ok := res.(pwm.PWM)
	if !ok {
		return nil, fmt.Errorf("dependency %q is not a PWM component", name)
	}
	return p, nil
}

// Start launches the control loop.
func (c *Controller) Start(ctx context.Context) error {
	loopCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	c.mu.Lock()
	c.lastCmd = time.Now()
	c.mu.Unlock()

	c.wg.Add(1)
	go c.loop(loopCtx)

	c.logger.Info("tank-drive controller started",
		"left", c.config.Left, "right", c.config.Right, "rate_hz", c.config.RateHz)
	return nil
}

// loop runs the fixed-rate control loop until the context is cancelled.
func (c *Controller) loop(ctx context.Context) {
	defer c.wg.Done()

	period := time.Duration(float64(time.Second) / c.config.RateHz)
	dt := period.Seconds()
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.neutralize(context.Background())
			return
		case <-ticker.C:
			c.tick(ctx, dt)
		}
	}
}

// tick computes and applies one control step.
func (c *Controller) tick(ctx context.Context, dt float64) {
	timeout := time.Duration(c.config.CommandTimeoutMs) * time.Millisecond

	c.mu.Lock()
	surge, yaw := c.surge, c.yaw
	active := true
	if timeout > 0 && time.Since(c.lastCmd) > timeout {
		surge, yaw = 0, 0
		active = false
	}

	surge = applyDeadband(surge, c.config.Deadband)
	yaw = applyDeadband(yaw, c.config.Deadband)

	targetLeft := clamp(surge+yaw*c.config.YawGain, -1, 1)
	targetRight := clamp(surge-yaw*c.config.YawGain, -1, 1)

	maxDelta := c.config.SlewPerS * dt
	c.curLeft = slew(c.curLeft, targetLeft, maxDelta)
	c.curRight = slew(c.curRight, targetRight, maxDelta)
	c.active = active

	outLeft := c.curLeft
	outRight := c.curRight
	if c.config.InvertLeft {
		outLeft = -outLeft
	}
	if c.config.InvertRight {
		outRight = -outRight
	}
	c.mu.Unlock()

	if err := c.left.SetNormalized(ctx, outLeft); err != nil {
		c.logger.Warn("failed to set left thruster", "error", err)
	}
	if err := c.right.SetNormalized(ctx, outRight); err != nil {
		c.logger.Warn("failed to set right thruster", "error", err)
	}
}

// neutralize commands both actuators to zero (center pulse).
func (c *Controller) neutralize(ctx context.Context) {
	c.mu.Lock()
	c.surge, c.yaw = 0, 0
	c.curLeft, c.curRight = 0, 0
	c.active = false
	c.mu.Unlock()
	_ = c.left.SetNormalized(ctx, 0)
	_ = c.right.SetNormalized(ctx, 0)
}

// SetIntent sets the normalized motion intent.
func (c *Controller) SetIntent(ctx context.Context, surge, yaw float64) error {
	c.mu.Lock()
	c.surge = clamp(surge, -1, 1)
	c.yaw = clamp(yaw, -1, 1)
	c.lastCmd = time.Now()
	c.mu.Unlock()
	return nil
}

// Stop commands zero motion.
func (c *Controller) Stop(ctx context.Context) error {
	return c.SetIntent(ctx, 0, 0)
}

// Arm arms both underlying actuators.
func (c *Controller) Arm(ctx context.Context) error {
	if err := c.left.Arm(ctx); err != nil {
		return fmt.Errorf("failed to arm left: %w", err)
	}
	if err := c.right.Arm(ctx); err != nil {
		return fmt.Errorf("failed to arm right: %w", err)
	}
	c.mu.Lock()
	c.armed = true
	c.lastCmd = time.Now()
	c.mu.Unlock()
	c.logger.Info("drive armed")
	return nil
}

// State returns the latest drive state.
func (c *Controller) State(ctx context.Context) (drive.State, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return drive.State{
		Surge:  c.surge,
		Yaw:    c.yaw,
		Left:   c.curLeft,
		Right:  c.curRight,
		Armed:  c.armed,
		Active: c.active,
	}, nil
}

// Name returns the resource name.
func (c *Controller) Name() resource.Name {
	return c.name
}

// Reconfigure is not supported; restart the component instead.
func (c *Controller) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return fmt.Errorf("reconfiguration not supported, restart component instead")
}

// DoCommand exposes drive methods through the generic command interface.
func (c *Controller) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmdName, _ := cmd["command"].(string)
	switch cmdName {
	case "set_intent":
		surge, _ := toFloat(cmd["surge"])
		yaw, _ := toFloat(cmd["yaw"])
		if err := c.SetIntent(ctx, surge, yaw); err != nil {
			return nil, err
		}
	case "stop":
		if err := c.Stop(ctx); err != nil {
			return nil, err
		}
	case "arm":
		if err := c.Arm(ctx); err != nil {
			return nil, err
		}
	case "get_state":
		// handled below
	default:
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}
	st, _ := c.State(ctx)
	return map[string]any{
		"surge": st.Surge, "yaw": st.Yaw,
		"left": st.Left, "right": st.Right,
		"armed": st.Armed, "active": st.Active,
	}, nil
}

// Close stops the control loop and neutralizes output.
func (c *Controller) Close(ctx context.Context) error {
	if c.cancel != nil {
		c.cancel()
	}
	done := make(chan struct{})
	go func() { c.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
	}
	c.neutralize(context.Background())
	return nil
}

// toFloat coerces a JSON-decoded numeric value to float64.
func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

// clamp constrains v to [lo, hi].
func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// applyDeadband zeroes v when its magnitude is below band.
func applyDeadband(v, band float64) float64 {
	if v < band && v > -band {
		return 0
	}
	return v
}

// slew moves cur toward target by at most maxDelta (0 = no limit).
func slew(cur, target, maxDelta float64) float64 {
	if maxDelta <= 0 {
		return target
	}
	d := target - cur
	if d > maxDelta {
		d = maxDelta
	}
	if d < -maxDelta {
		d = -maxDelta
	}
	return cur + d
}

// Verify interface compliance at compile time.
var _ drive.Drive = (*Controller)(nil)
