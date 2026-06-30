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

// RemotePWMController is a proxy that implements the PWMController interface via NATS.
type RemotePWMController struct {
	meshClient *mesh.Client
	natsConn   *nats.Conn
	descriptor mesh.ServiceDescriptor
	logger     *slog.Logger

	cmdSubject   string
	stateSubject string

	mu            sync.RWMutex
	channelStates map[int]PWMChannelState
	lastStateAt   time.Time

	stateSub *nats.Subscription
	cancel   context.CancelFunc
	done     chan struct{}
}

// PWMChannelState holds the state of a PWM channel.
type PWMChannelState struct {
	Channel   int     `json:"channel"`
	PulseUs   int     `json:"pulse_us"`
	DutyCycle float64 `json:"duty_cycle"`
	Enabled   bool    `json:"enabled"`
}

// RemotePWMOption configures a RemotePWMController.
type RemotePWMOption func(*RemotePWMController)

// WithRemotePWMLogger sets the logger.
func WithRemotePWMLogger(logger *slog.Logger) RemotePWMOption {
	return func(p *RemotePWMController) {
		p.logger = logger
	}
}

// NewRemotePWMController creates a new remote PWM controller proxy.
func NewRemotePWMController(mc *mesh.Client, nc *nats.Conn, desc mesh.ServiceDescriptor, opts ...RemotePWMOption) (*RemotePWMController, error) {
	p := &RemotePWMController{
		meshClient:    mc,
		natsConn:      nc,
		descriptor:    desc,
		logger:        slog.Default(),
		channelStates: make(map[int]PWMChannelState),
		done:          make(chan struct{}),
	}

	for _, opt := range opts {
		opt(p)
	}

	// Find command subject
	var err error
	p.cmdSubject, err = GetCommandSubject(desc)
	if err != nil {
		return nil, err
	}

	// Find state subject for feedback
	p.stateSubject = findPWMStateSubject(desc)

	// Subscribe to state updates if available
	if p.stateSubject != "" {
		p.stateSub, err = nc.Subscribe(p.stateSubject, p.handleState)
		if err != nil {
			p.logger.Warn("failed to subscribe to state", "subject", p.stateSubject, "error", err)
		}
	}

	p.logger.Debug("created remote PWM controller",
		"name", desc.Name,
		"cmd_subject", p.cmdSubject,
		"state_subject", p.stateSubject,
	)

	return p, nil
}

// findPWMStateSubject finds a state subject from the descriptor.
func findPWMStateSubject(desc mesh.ServiceDescriptor) string {
	for _, pub := range desc.Publishes {
		if containsAny(pub, "state", "pwm_state", "channel_state") {
			return pub
		}
	}
	return ""
}

// handleState processes incoming state messages.
func (p *RemotePWMController) handleState(msg *nats.Msg) {
	var state struct {
		Data struct {
			Channels []PWMChannelState `json:"channels"`
		} `json:"data"`
	}

	if err := json.Unmarshal(msg.Data, &state); err != nil {
		p.logger.Debug("failed to parse state", "error", err)
		return
	}

	p.mu.Lock()
	for _, ch := range state.Data.Channels {
		p.channelStates[ch.Channel] = ch
	}
	p.lastStateAt = time.Now()
	p.mu.Unlock()
}

// SetPulse sets the PWM pulse width in microseconds for a channel.
func (p *RemotePWMController) SetPulse(ctx context.Context, channel int, pulseUs int) error {
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

	if err := p.natsConn.Publish(p.cmdSubject, data); err != nil {
		return fmt.Errorf("failed to publish command: %w", err)
	}

	// Update local state optimistically
	p.mu.Lock()
	p.channelStates[channel] = PWMChannelState{
		Channel: channel,
		PulseUs: pulseUs,
		Enabled: true,
	}
	p.mu.Unlock()

	return nil
}

// SetMultiplePulses sets PWM pulses for multiple channels at once.
func (p *RemotePWMController) SetMultiplePulses(ctx context.Context, pulses map[int]int) error {
	channels := make([]map[string]any, 0, len(pulses))
	for ch, pulse := range pulses {
		channels = append(channels, map[string]any{
			"channel":  ch,
			"pulse_us": pulse,
		})
	}

	cmd := map[string]any{
		"type": "PWM_SET",
		"data": map[string]any{
			"channels": channels,
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	if err := p.natsConn.Publish(p.cmdSubject, data); err != nil {
		return fmt.Errorf("failed to publish command: %w", err)
	}

	// Update local state optimistically
	p.mu.Lock()
	for ch, pulse := range pulses {
		p.channelStates[ch] = PWMChannelState{
			Channel: ch,
			PulseUs: pulse,
			Enabled: true,
		}
	}
	p.mu.Unlock()

	return nil
}

// GetPulse returns the current pulse width for a channel.
func (p *RemotePWMController) GetPulse(ctx context.Context, channel int) (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if state, ok := p.channelStates[channel]; ok {
		return state.PulseUs, nil
	}
	return 0, fmt.Errorf("channel %d state unknown", channel)
}

// SetEnabled enables or disables a channel.
func (p *RemotePWMController) SetEnabled(ctx context.Context, channel int, enabled bool) error {
	cmd := map[string]any{
		"type": "PWM_ENABLE",
		"data": map[string]any{
			"channel": channel,
			"enabled": enabled,
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	return p.natsConn.Publish(p.cmdSubject, data)
}

// Close releases resources.
func (p *RemotePWMController) Close(ctx context.Context) error {
	if p.stateSub != nil {
		p.stateSub.Unsubscribe()
	}
	if p.cancel != nil {
		p.cancel()
	}
	close(p.done)
	return nil
}

// Name returns the controller name.
func (p *RemotePWMController) Name() string {
	return p.descriptor.Name
}

// Descriptor returns the service descriptor.
func (p *RemotePWMController) Descriptor() mesh.ServiceDescriptor {
	return p.descriptor
}

// DoCommand sends a generic command.
func (p *RemotePWMController) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	msg, err := p.natsConn.RequestWithContext(ctx, p.cmdSubject, data)
	if err != nil {
		// Fall back to publish
		if err := p.natsConn.Publish(p.cmdSubject, data); err != nil {
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
