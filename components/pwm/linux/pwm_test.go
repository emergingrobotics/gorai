package linux

import (
	"context"
	"testing"
	"time"

	driverpwm "github.com/emergingrobotics/gorai/driver/pwm"
	"github.com/emergingrobotics/gorai/pkg/registry"
)

// fakeChannel is an in-memory driverpwm.Channel for testing pulse/duty logic.
type fakeChannel struct {
	period       uint64
	duty         uint64
	dutyWrites   []uint64
	enabled      bool
	inverted     bool
	polarityCall int
}

func (f *fakeChannel) Enable(ctx context.Context) error  { f.enabled = true; return nil }
func (f *fakeChannel) Disable(ctx context.Context) error { f.enabled = false; return nil }
func (f *fakeChannel) Enabled() bool                     { return f.enabled }
func (f *fakeChannel) SetPeriod(ctx context.Context, ns uint64) error {
	f.period = ns
	return nil
}
func (f *fakeChannel) Period() uint64 { return f.period }
func (f *fakeChannel) SetDuty(ctx context.Context, ns uint64) error {
	f.duty = ns
	f.dutyWrites = append(f.dutyWrites, ns)
	return nil
}
func (f *fakeChannel) Duty() uint64 { return f.duty }
func (f *fakeChannel) SetDutyCycle(ctx context.Context, duty float64) error {
	f.duty = uint64(float64(f.period) * duty)
	return nil
}
func (f *fakeChannel) DutyCycle() float64 {
	if f.period == 0 {
		return 0
	}
	return float64(f.duty) / float64(f.period)
}
func (f *fakeChannel) SetFrequency(ctx context.Context, hz float64) error {
	f.period = uint64(1e9 / hz)
	return nil
}
func (f *fakeChannel) Frequency() float64 {
	if f.period == 0 {
		return 0
	}
	return 1e9 / float64(f.period)
}
func (f *fakeChannel) SetPolarity(ctx context.Context, inverted bool) error {
	f.inverted = inverted
	f.polarityCall++
	return nil
}

var _ driverpwm.Channel = (*fakeChannel)(nil)

func newTestPWM(t *testing.T, cfg Config) (*PWM, *fakeChannel) {
	t.Helper()
	fc := &fakeChannel{}
	opener := func(chip, channel int) (driverpwm.Channel, error) { return fc, nil }
	p, err := newPWM(context.Background(), "thruster", cfg, opener, nil)
	if err != nil {
		t.Fatalf("newPWM failed: %v", err)
	}
	return p, fc
}

func escConfig() Config {
	cfg := DefaultConfig()
	cfg.Chip = 2
	cfg.Channel = 1
	return cfg
}

func TestNeutralOnStart(t *testing.T) {
	_, fc := newTestPWM(t, escConfig())

	if fc.period != 20_000_000 {
		t.Errorf("expected 20ms period at 50Hz, got %d ns", fc.period)
	}
	if fc.duty != 1_500_000 {
		t.Errorf("expected 1500us neutral duty, got %d ns", fc.duty)
	}
	if !fc.enabled {
		t.Errorf("expected channel enabled on start")
	}
}

func TestEnableOnStartFalse(t *testing.T) {
	cfg := escConfig()
	cfg.EnableOnStart = false
	_, fc := newTestPWM(t, cfg)
	if fc.enabled {
		t.Errorf("expected channel disabled when enable_on_start is false")
	}
}

func TestSetPulseDutyMath(t *testing.T) {
	p, fc := newTestPWM(t, escConfig())
	ctx := context.Background()

	if err := p.SetPulse(ctx, 1750); err != nil {
		t.Fatalf("SetPulse: %v", err)
	}
	if fc.duty != 1_750_000 {
		t.Errorf("expected 1750us -> 1_750_000 ns, got %d", fc.duty)
	}
	got, _ := p.GetPulse(ctx)
	if got != 1750 {
		t.Errorf("expected GetPulse 1750, got %f", got)
	}
}

func TestSetPulseClamping(t *testing.T) {
	p, fc := newTestPWM(t, escConfig())
	ctx := context.Background()

	if err := p.SetPulse(ctx, 500); err != nil {
		t.Fatalf("SetPulse: %v", err)
	}
	if fc.duty != 1_000_000 {
		t.Errorf("expected clamp to min 1000us, got %d ns", fc.duty)
	}

	if err := p.SetPulse(ctx, 3000); err != nil {
		t.Fatalf("SetPulse: %v", err)
	}
	if fc.duty != 2_000_000 {
		t.Errorf("expected clamp to max 2000us, got %d ns", fc.duty)
	}
}

func TestSetNormalized(t *testing.T) {
	p, fc := newTestPWM(t, escConfig())
	ctx := context.Background()

	cases := []struct {
		value   float64
		wantDut uint64
	}{
		{-1.0, 1_000_000},
		{0.0, 1_500_000},
		{1.0, 2_000_000},
		{0.5, 1_750_000},
		{-2.0, 1_000_000}, // clamped
		{2.0, 2_000_000},  // clamped
	}
	for _, c := range cases {
		if err := p.SetNormalized(ctx, c.value); err != nil {
			t.Fatalf("SetNormalized(%f): %v", c.value, err)
		}
		if fc.duty != c.wantDut {
			t.Errorf("SetNormalized(%f): expected duty %d, got %d", c.value, c.wantDut, fc.duty)
		}
	}
}

func TestEnableDisable(t *testing.T) {
	p, fc := newTestPWM(t, escConfig())
	ctx := context.Background()

	if err := p.Disable(ctx); err != nil {
		t.Fatalf("Disable: %v", err)
	}
	if fc.enabled {
		t.Errorf("expected disabled")
	}
	if on, _ := p.IsEnabled(ctx); on {
		t.Errorf("IsEnabled should report false")
	}

	if err := p.Enable(ctx); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if !fc.enabled {
		t.Errorf("expected enabled")
	}
}

func TestInvertAppliesPolarity(t *testing.T) {
	cfg := escConfig()
	cfg.Invert = true
	_, fc := newTestPWM(t, cfg)
	if fc.polarityCall == 0 || !fc.inverted {
		t.Errorf("expected polarity to be set for inverted config")
	}
}

func TestArmSequence(t *testing.T) {
	cfg := escConfig()
	cfg.EnableOnStart = false
	cfg.ArmSequence = []ArmStep{
		{PulseUs: 2000, HoldMs: 1},
		{PulseUs: 1500, HoldMs: 1},
	}
	p, fc := newTestPWM(t, cfg)

	// Construction sets the initial neutral pulse once (enable_on_start=false).
	if len(fc.dutyWrites) != 1 || fc.dutyWrites[0] != 1_500_000 {
		t.Fatalf("unexpected duty writes before arm: %v", fc.dutyWrites)
	}

	if err := p.Arm(context.Background()); err != nil {
		t.Fatalf("Arm: %v", err)
	}

	// Arm ensures enabled, then writes each step pulse in order.
	if !fc.enabled {
		t.Errorf("expected channel enabled after arm")
	}
	gotSteps := fc.dutyWrites[1:]
	want := []uint64{2_000_000, 1_500_000}
	if len(gotSteps) != len(want) {
		t.Fatalf("expected %d arm duty writes, got %v", len(want), gotSteps)
	}
	for i, w := range want {
		if gotSteps[i] != w {
			t.Errorf("arm step %d: expected duty %d, got %d", i, w, gotSteps[i])
		}
	}
	// Final pulse should be neutral.
	if fc.duty != 1_500_000 {
		t.Errorf("expected final neutral duty 1_500_000, got %d", fc.duty)
	}
}

func TestArmRejectsConcurrent(t *testing.T) {
	cfg := escConfig()
	cfg.ArmSequence = []ArmStep{{PulseUs: 2000, HoldMs: 40}}
	p, _ := newTestPWM(t, cfg)

	errCh := make(chan error, 1)
	go func() { errCh <- p.Arm(context.Background()) }()

	// Give the first arm time to take the guard, then a second must be rejected.
	time.Sleep(10 * time.Millisecond)
	if err := p.Arm(context.Background()); err == nil {
		t.Errorf("expected concurrent arm to be rejected")
	}
	if err := <-errCh; err != nil {
		t.Errorf("first arm should succeed, got %v", err)
	}
}

func TestArmContextCancel(t *testing.T) {
	cfg := escConfig()
	cfg.ArmSequence = []ArmStep{{PulseUs: 2000, HoldMs: 5000}}
	p, _ := newTestPWM(t, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	if err := p.Arm(ctx); err == nil {
		t.Errorf("expected arm to return context error on cancel")
	}
}

func TestArmSequenceParseAndValidate(t *testing.T) {
	conf := registry.Config{
		"chip":    float64(0),
		"channel": float64(1),
		"arm_sequence": []any{
			map[string]any{"pulse_us": float64(2000), "hold_ms": float64(1000)},
			map[string]any{"pulse_us": float64(1500), "hold_ms": float64(1000)},
		},
	}
	cfg, err := ParseConfig(conf)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if len(cfg.ArmSequence) != 2 {
		t.Fatalf("expected 2 arm steps, got %d", len(cfg.ArmSequence))
	}
	if cfg.ArmSequence[0].PulseUs != 2000 || cfg.ArmSequence[0].HoldMs != 1000 {
		t.Errorf("unexpected first arm step: %+v", cfg.ArmSequence[0])
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("valid arm config rejected: %v", err)
	}

	badHold := cfg
	badHold.ArmSequence = []ArmStep{{PulseUs: 1500, HoldMs: 0}}
	if err := badHold.Validate(); err == nil {
		t.Errorf("expected error for non-positive hold_ms")
	}

	badPulse := cfg
	badPulse.ArmSequence = []ArmStep{{PulseUs: 3000, HoldMs: 100}}
	if err := badPulse.Validate(); err == nil {
		t.Errorf("expected error for out-of-range pulse_us")
	}
}

func TestDefaultArmSequenceApplied(t *testing.T) {
	conf := registry.Config{"chip": float64(0), "channel": float64(1)}
	cfg, err := ParseConfig(conf)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if len(cfg.ArmSequence) != 2 {
		t.Fatalf("expected default arm sequence of 2 steps, got %d", len(cfg.ArmSequence))
	}
	if cfg.ArmSequence[0].PulseUs != cfg.MaxPulseUs || cfg.ArmSequence[0].HoldMs != 1000 {
		t.Errorf("unexpected default arm step 0: %+v", cfg.ArmSequence[0])
	}
	if cfg.ArmSequence[1].PulseUs != cfg.InitialPulseUs || cfg.ArmSequence[1].HoldMs != 1000 {
		t.Errorf("unexpected default arm step 1: %+v", cfg.ArmSequence[1])
	}
}

func TestConfigValidate(t *testing.T) {
	valid := escConfig()
	if err := valid.Validate(); err != nil {
		t.Errorf("valid config rejected: %v", err)
	}

	bad := escConfig()
	bad.MinPulseUs = 2500
	bad.MaxPulseUs = 2000
	if err := bad.Validate(); err == nil {
		t.Errorf("expected error when min >= max")
	}

	overPeriod := escConfig()
	overPeriod.FrequencyHz = 1000 // period 1000us
	overPeriod.MaxPulseUs = 2000  // exceeds period
	if err := overPeriod.Validate(); err == nil {
		t.Errorf("expected error when max pulse exceeds period")
	}
}
