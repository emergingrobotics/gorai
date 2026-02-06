package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/gorai/gorai/pkg/mesh"
)

// RemoteMotor is a proxy that implements the Motor interface via NATS.
type RemoteMotor struct {
	meshClient *mesh.Client
	natsConn   *nats.Conn
	descriptor mesh.ServiceDescriptor
	logger     *slog.Logger

	cmdSubject   string
	stateSubject string

	mu          sync.RWMutex
	power       float64
	isMoving    bool
	lastStateAt time.Time

	stateSub *nats.Subscription
	cancel   context.CancelFunc
	done     chan struct{}
}

// RemoteMotorOption configures a RemoteMotor.
type RemoteMotorOption func(*RemoteMotor)

// WithRemoteMotorLogger sets the logger.
func WithRemoteMotorLogger(logger *slog.Logger) RemoteMotorOption {
	return func(m *RemoteMotor) {
		m.logger = logger
	}
}

// NewRemoteMotor creates a new remote motor proxy.
func NewRemoteMotor(mc *mesh.Client, nc *nats.Conn, desc mesh.ServiceDescriptor, opts ...RemoteMotorOption) (*RemoteMotor, error) {
	m := &RemoteMotor{
		meshClient: mc,
		natsConn:   nc,
		descriptor: desc,
		logger:     slog.Default(),
		done:       make(chan struct{}),
	}

	for _, opt := range opts {
		opt(m)
	}

	// Find command subject
	var err error
	m.cmdSubject, err = GetCommandSubject(desc)
	if err != nil {
		return nil, err
	}

	// Find state subject for feedback
	m.stateSubject = findStateSubject(desc)

	// Subscribe to state updates if available
	if m.stateSubject != "" {
		m.stateSub, err = nc.Subscribe(m.stateSubject, m.handleState)
		if err != nil {
			m.logger.Warn("failed to subscribe to state", "subject", m.stateSubject, "error", err)
		}
	}

	m.logger.Debug("created remote motor",
		"name", desc.Name,
		"cmd_subject", m.cmdSubject,
		"state_subject", m.stateSubject,
	)

	return m, nil
}

// findStateSubject finds a state subject from the descriptor.
func findStateSubject(desc mesh.ServiceDescriptor) string {
	for _, pub := range desc.Publishes {
		if containsAny(pub, "state", "motor_state", "feedback") {
			return pub
		}
	}
	return ""
}

// handleState processes incoming state messages.
func (m *RemoteMotor) handleState(msg *nats.Msg) {
	var state struct {
		Data struct {
			Power    float64 `json:"power"`
			IsMoving bool    `json:"is_moving"`
		} `json:"data"`
	}

	if err := json.Unmarshal(msg.Data, &state); err != nil {
		m.logger.Debug("failed to parse state", "error", err)
		return
	}

	m.mu.Lock()
	m.power = state.Data.Power
	m.isMoving = state.Data.IsMoving
	m.lastStateAt = time.Now()
	m.mu.Unlock()
}

// SetPower sets the motor power from -1.0 to 1.0.
func (m *RemoteMotor) SetPower(ctx context.Context, power float64) error {
	if power < -1.0 {
		power = -1.0
	}
	if power > 1.0 {
		power = 1.0
	}

	cmd := map[string]any{
		"type": "MOTOR_SET",
		"data": map[string]any{
			"power": power,
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	if err := m.natsConn.Publish(m.cmdSubject, data); err != nil {
		return fmt.Errorf("failed to publish command: %w", err)
	}

	// Update local state optimistically
	m.mu.Lock()
	m.power = power
	m.isMoving = power != 0
	m.mu.Unlock()

	return nil
}

// SetPowerPWM sets motor power using PWM pulse width in microseconds.
func (m *RemoteMotor) SetPowerPWM(ctx context.Context, channel int, pulseUs int) error {
	cmd := map[string]any{
		"type": "PWM_SET",
		"data": map[string]any{
			"channels": []map[string]any{
				{"channel": channel, "pulse_us": pulseUs},
			},
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	return m.natsConn.Publish(m.cmdSubject, data)
}

// GetPower returns the current power level.
func (m *RemoteMotor) GetPower(ctx context.Context) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.power, nil
}

// Stop stops the motor.
func (m *RemoteMotor) Stop(ctx context.Context) error {
	return m.SetPower(ctx, 0)
}

// IsMoving returns true if the motor is moving.
func (m *RemoteMotor) IsMoving(ctx context.Context) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isMoving, nil
}

// Close releases resources.
func (m *RemoteMotor) Close(ctx context.Context) error {
	if m.stateSub != nil {
		m.stateSub.Unsubscribe()
	}
	if m.cancel != nil {
		m.cancel()
	}
	close(m.done)
	return nil
}

// Name returns the motor name.
func (m *RemoteMotor) Name() string {
	return m.descriptor.Name
}

// Descriptor returns the service descriptor.
func (m *RemoteMotor) Descriptor() mesh.ServiceDescriptor {
	return m.descriptor
}

// DoCommand sends a generic command.
func (m *RemoteMotor) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	// Use request-reply if possible
	msg, err := m.natsConn.RequestWithContext(ctx, m.cmdSubject, data)
	if err != nil {
		// Fall back to publish
		if err := m.natsConn.Publish(m.cmdSubject, data); err != nil {
			return nil, err
		}
		return nil, nil
	}

	var resp map[string]any
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}
