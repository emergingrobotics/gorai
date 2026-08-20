package bmp180

import (
	"context"
	"encoding/binary"
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
	registry.RegisterComponent("sensor", "bmp180", New)
}

// Register addresses (BMP180 datasheet).
const (
	regCalibration = 0xAA // start of 22-byte calibration block
	regControl     = 0xF4
	regData        = 0xF6
	cmdReadTemp    = 0x2E
	cmdReadPressue = 0x34
)

// deviceOpener opens an I2C device. Overridable in tests.
type deviceOpener func(bus string, addr uint16) (i2c.Device, error)

// Sensor is a BMP180 pressure/temperature sensor driver.
type Sensor struct {
	name   resource.Name
	config Config
	logger *slog.Logger
	opener deviceOpener

	dev i2c.Device
	cal calibration

	mu sync.Mutex
}

// New creates a BMP180 sensor component.
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

	s := &Sensor{
		name:   resource.NewComponentName("gorai", "sensor", nameStr),
		config: cfg,
		logger: logger.With("component", "sensor/bmp180", "name", nameStr),
		opener: defaultOpener,
	}
	return s, nil
}

// Start opens the device and reads calibration coefficients.
func (s *Sensor) Start(ctx context.Context) error {
	dev, err := s.opener(s.config.Bus, s.config.Address)
	if err != nil {
		return fmt.Errorf("failed to open BMP180: %w", err)
	}
	s.dev = dev

	raw, err := dev.ReadReg(ctx, regCalibration, 22)
	if err != nil {
		return fmt.Errorf("failed to read BMP180 calibration: %w", err)
	}
	cal, err := parseCalibration(raw)
	if err != nil {
		return err
	}
	s.cal = cal

	s.logger.Info("BMP180 initialized", "bus", s.config.Bus, "address", fmt.Sprintf("0x%02X", s.config.Address), "oversampling", s.config.Oversampling)
	return nil
}

// measure performs a temperature + pressure conversion and compensation.
func (s *Sensor) measure(ctx context.Context) (tempC, pressurePa float64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dev == nil {
		return 0, 0, fmt.Errorf("device not initialized")
	}

	// Uncompensated temperature.
	if err := s.dev.WriteByteReg(ctx, regControl, cmdReadTemp); err != nil {
		return 0, 0, err
	}
	sleep(ctx, 5*time.Millisecond)
	tRaw, err := s.dev.ReadReg(ctx, regData, 2)
	if err != nil {
		return 0, 0, err
	}
	ut := int32(binary.BigEndian.Uint16(tRaw))

	// Uncompensated pressure.
	if err := s.dev.WriteByteReg(ctx, regControl, cmdReadPressue|byte(s.config.Oversampling<<6)); err != nil {
		return 0, 0, err
	}
	sleep(ctx, conversionDelay(s.config.Oversampling))
	pRaw, err := s.dev.ReadReg(ctx, regData, 3)
	if err != nil {
		return 0, 0, err
	}
	up := (int32(pRaw[0])<<16 | int32(pRaw[1])<<8 | int32(pRaw[2])) >> (8 - s.config.Oversampling)

	tempC, pressurePa = s.cal.compensate(ut, up, s.config.Oversampling)
	return tempC, pressurePa, nil
}

// conversionDelay returns the max pressure conversion time for an oversampling
// setting (datasheet table).
func conversionDelay(oss uint) time.Duration {
	switch oss {
	case 1:
		return 8 * time.Millisecond
	case 2:
		return 14 * time.Millisecond
	case 3:
		return 26 * time.Millisecond
	default:
		return 5 * time.Millisecond
	}
}

// sleep waits for d or until the context is cancelled.
func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
	case <-ctx.Done():
	}
}

// Name returns the resource name.
func (s *Sensor) Name() resource.Name {
	return s.name
}

// Reconfigure is not supported; restart the component instead.
func (s *Sensor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return fmt.Errorf("reconfiguration not supported, restart component instead")
}

// DoCommand exposes a state query.
func (s *Sensor) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if name, _ := cmd["command"].(string); name == "get_state" {
		return s.Readings(ctx)
	}
	return nil, fmt.Errorf("unknown command: %v", cmd["command"])
}

// Close releases the device.
func (s *Sensor) Close(ctx context.Context) error {
	return nil
}

// Readings returns pressure (Pa), temperature (°C) and estimated altitude (m).
func (s *Sensor) Readings(ctx context.Context) (map[string]any, error) {
	tempC, pressurePa, err := s.measure(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"pressure_pa":   pressurePa,
		"temperature_c": tempC,
		"altitude_m":    altitude(pressurePa),
	}, nil
}

// Verify interface compliance.
var _ sensor.Sensor = (*Sensor)(nil)
