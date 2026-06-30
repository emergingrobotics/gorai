package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/emergingrobotics/gorai/pkg/mesh"
)

// RemoteServo is a proxy that implements the Servo interface via NATS.
type RemoteServo struct {
	meshClient *mesh.Client
	natsConn   *nats.Conn
	descriptor mesh.ServiceDescriptor
	logger     *slog.Logger

	cmdSubject   string
	stateSubject string

	// Servo configuration
	channel  int
	minPulse int // Minimum pulse width in microseconds
	maxPulse int // Maximum pulse width in microseconds
	minAngle float64
	maxAngle float64

	mu          sync.RWMutex
	position    float64 // Current angle in degrees
	lastStateAt time.Time

	stateSub *nats.Subscription
	cancel   context.CancelFunc
	done     chan struct{}
}

// RemoteServoOption configures a RemoteServo.
type RemoteServoOption func(*RemoteServo)

// WithRemoteServoLogger sets the logger.
func WithRemoteServoLogger(logger *slog.Logger) RemoteServoOption {
	return func(s *RemoteServo) {
		s.logger = logger
	}
}

// WithServoChannel sets the PWM channel for the servo.
func WithServoChannel(channel int) RemoteServoOption {
	return func(s *RemoteServo) {
		s.channel = channel
	}
}

// WithServoPulseRange sets the pulse width range in microseconds.
func WithServoPulseRange(minPulse, maxPulse int) RemoteServoOption {
	return func(s *RemoteServo) {
		s.minPulse = minPulse
		s.maxPulse = maxPulse
	}
}

// WithServoAngleRange sets the angle range in degrees.
func WithServoAngleRange(minAngle, maxAngle float64) RemoteServoOption {
	return func(s *RemoteServo) {
		s.minAngle = minAngle
		s.maxAngle = maxAngle
	}
}

// NewRemoteServo creates a new remote servo proxy.
func NewRemoteServo(mc *mesh.Client, nc *nats.Conn, desc mesh.ServiceDescriptor, opts ...RemoteServoOption) (*RemoteServo, error) {
	s := &RemoteServo{
		meshClient: mc,
		natsConn:   nc,
		descriptor: desc,
		logger:     slog.Default(),
		// Default servo configuration
		channel:  0,
		minPulse: 500,  // 0.5ms
		maxPulse: 2500, // 2.5ms
		minAngle: 0,
		maxAngle: 180,
		done:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(s)
	}

	// Extract config from metadata if available
	if ch, ok := desc.Metadata["channel"]; ok {
		var chInt int
		fmt.Sscanf(ch, "%d", &chInt)
		if chInt > 0 {
			s.channel = chInt
		}
	}
	if minP, ok := desc.Metadata["min_pulse"]; ok {
		var minPInt int
		fmt.Sscanf(minP, "%d", &minPInt)
		if minPInt > 0 {
			s.minPulse = minPInt
		}
	}
	if maxP, ok := desc.Metadata["max_pulse"]; ok {
		var maxPInt int
		fmt.Sscanf(maxP, "%d", &maxPInt)
		if maxPInt > 0 {
			s.maxPulse = maxPInt
		}
	}

	// Find command subject
	var err error
	s.cmdSubject, err = GetCommandSubject(desc)
	if err != nil {
		return nil, err
	}

	// Find state subject for feedback
	s.stateSubject = findServoStateSubject(desc)

	// Subscribe to state updates if available
	if s.stateSubject != "" {
		s.stateSub, err = nc.Subscribe(s.stateSubject, s.handleState)
		if err != nil {
			s.logger.Warn("failed to subscribe to state", "subject", s.stateSubject, "error", err)
		}
	}

	s.logger.Debug("created remote servo",
		"name", desc.Name,
		"channel", s.channel,
		"cmd_subject", s.cmdSubject,
	)

	return s, nil
}

// findServoStateSubject finds a state subject from the descriptor.
func findServoStateSubject(desc mesh.ServiceDescriptor) string {
	for _, pub := range desc.Publishes {
		if containsAny(pub, "state", "servo_state", "position") {
			return pub
		}
	}
	return ""
}

// handleState processes incoming state messages.
func (s *RemoteServo) handleState(msg *nats.Msg) {
	var state struct {
		Data struct {
			Position float64 `json:"position"`
			Angle    float64 `json:"angle"`
		} `json:"data"`
	}

	if err := json.Unmarshal(msg.Data, &state); err != nil {
		s.logger.Debug("failed to parse state", "error", err)
		return
	}

	s.mu.Lock()
	if state.Data.Angle != 0 {
		s.position = state.Data.Angle
	} else {
		s.position = state.Data.Position
	}
	s.lastStateAt = time.Now()
	s.mu.Unlock()
}

// SetAngle sets the servo angle in degrees.
func (s *RemoteServo) SetAngle(ctx context.Context, angle float64) error {
	// Clamp angle to valid range
	if angle < s.minAngle {
		angle = s.minAngle
	}
	if angle > s.maxAngle {
		angle = s.maxAngle
	}

	// Convert angle to pulse width
	pulseUs := s.angleToPulse(angle)

	cmd := map[string]any{
		"type": "PWM_SET",
		"data": map[string]any{
			"channels": []map[string]any{
				{"channel": s.channel, "pulse_us": pulseUs},
			},
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	if err := s.natsConn.Publish(s.cmdSubject, data); err != nil {
		return fmt.Errorf("failed to publish command: %w", err)
	}

	// Update local state optimistically
	s.mu.Lock()
	s.position = angle
	s.mu.Unlock()

	return nil
}

// GetAngle returns the current servo angle.
func (s *RemoteServo) GetAngle(ctx context.Context) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.position, nil
}

// SetPulse sets the servo pulse width directly in microseconds.
func (s *RemoteServo) SetPulse(ctx context.Context, pulseUs int) error {
	cmd := map[string]any{
		"type": "PWM_SET",
		"data": map[string]any{
			"channels": []map[string]any{
				{"channel": s.channel, "pulse_us": pulseUs},
			},
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	if err := s.natsConn.Publish(s.cmdSubject, data); err != nil {
		return fmt.Errorf("failed to publish command: %w", err)
	}

	// Update local state
	s.mu.Lock()
	s.position = s.pulseToAngle(pulseUs)
	s.mu.Unlock()

	return nil
}

// angleToPulse converts an angle to pulse width.
func (s *RemoteServo) angleToPulse(angle float64) int {
	// Linear interpolation from angle to pulse
	ratio := (angle - s.minAngle) / (s.maxAngle - s.minAngle)
	pulse := float64(s.minPulse) + ratio*float64(s.maxPulse-s.minPulse)
	return int(pulse)
}

// pulseToAngle converts a pulse width to angle.
func (s *RemoteServo) pulseToAngle(pulseUs int) float64 {
	// Linear interpolation from pulse to angle
	ratio := float64(pulseUs-s.minPulse) / float64(s.maxPulse-s.minPulse)
	return s.minAngle + ratio*(s.maxAngle-s.minAngle)
}

// GetPosition returns the current position as a normalized value (0.0 to 1.0).
func (s *RemoteServo) GetPosition(ctx context.Context) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return (s.position - s.minAngle) / (s.maxAngle - s.minAngle), nil
}

// SetPosition sets the position as a normalized value (0.0 to 1.0).
func (s *RemoteServo) SetPosition(ctx context.Context, position float64) error {
	// Clamp to valid range
	if position < 0 {
		position = 0
	}
	if position > 1 {
		position = 1
	}
	angle := s.minAngle + position*(s.maxAngle-s.minAngle)
	return s.SetAngle(ctx, angle)
}

// Close releases resources.
func (s *RemoteServo) Close(ctx context.Context) error {
	if s.stateSub != nil {
		s.stateSub.Unsubscribe()
	}
	if s.cancel != nil {
		s.cancel()
	}
	close(s.done)
	return nil
}

// Name returns the servo name.
func (s *RemoteServo) Name() string {
	return s.descriptor.Name
}

// Channel returns the PWM channel.
func (s *RemoteServo) Channel() int {
	return s.channel
}

// Descriptor returns the service descriptor.
func (s *RemoteServo) Descriptor() mesh.ServiceDescriptor {
	return s.descriptor
}

// DoCommand sends a generic command.
func (s *RemoteServo) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	msg, err := s.natsConn.RequestWithContext(ctx, s.cmdSubject, data)
	if err != nil {
		// Fall back to publish
		if err := s.natsConn.Publish(s.cmdSubject, data); err != nil {
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
