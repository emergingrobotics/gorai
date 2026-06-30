// Package proxy provides remote component proxies for discovered devices.
//
// Proxy components wrap NATS communication to make remote devices appear
// as local Gorai components. They implement the same interfaces as local
// components (Motor, Sensor, etc.) but forward calls over NATS.
//
// # Usage
//
//	factory := proxy.NewFactory(meshClient, natsConn)
//	motor, err := factory.Create(ctx, descriptor, "motor", "")
//	// motor implements the Motor interface
//	motor.(*proxy.RemoteMotor).SetPower(ctx, 0.5)
package proxy

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"

	"github.com/emergingrobotics/gorai/pkg/mesh"
)

// Factory creates proxy components for discovered services.
type Factory struct {
	meshClient *mesh.Client
	natsConn   *nats.Conn
	logger     *slog.Logger
	creators   map[string]CreatorFunc
}

// CreatorFunc creates a proxy component.
type CreatorFunc func(ctx context.Context, f *Factory, desc mesh.ServiceDescriptor) (any, error)

// FactoryOption configures a Factory.
type FactoryOption func(*Factory)

// WithFactoryLogger sets the logger.
func WithFactoryLogger(logger *slog.Logger) FactoryOption {
	return func(f *Factory) {
		f.logger = logger
	}
}

// NewFactory creates a new proxy factory.
func NewFactory(meshClient *mesh.Client, natsConn *nats.Conn, opts ...FactoryOption) *Factory {
	f := &Factory{
		meshClient: meshClient,
		natsConn:   natsConn,
		logger:     slog.Default(),
		creators:   make(map[string]CreatorFunc),
	}

	for _, opt := range opts {
		opt(f)
	}

	// Register default creators
	f.RegisterCreator("motor", createRemoteMotor)
	f.RegisterCreator("sensor", createRemoteSensor)
	f.RegisterCreator("sensor/imu", createRemoteSensor)
	f.RegisterCreator("sensor/gps", createRemoteSensor)
	f.RegisterCreator("camera", createRemoteCamera)
	f.RegisterCreator("pwm-controller", createRemotePWMController)
	f.RegisterCreator("servo", createRemoteServo)

	return f
}

// RegisterCreator registers a creator function for a component type.
func (f *Factory) RegisterCreator(componentType string, creator CreatorFunc) {
	f.creators[componentType] = creator
}

// Create creates a proxy component for the given service descriptor.
func (f *Factory) Create(ctx context.Context, desc mesh.ServiceDescriptor, componentType, subtype string) (any, error) {
	// Try type/subtype first
	if subtype != "" {
		key := componentType + "/" + subtype
		if creator, ok := f.creators[key]; ok {
			return creator(ctx, f, desc)
		}
	}

	// Try type only
	if creator, ok := f.creators[componentType]; ok {
		return creator(ctx, f, desc)
	}

	// Try descriptor's subtype
	if creator, ok := f.creators[desc.Subtype]; ok {
		return creator(ctx, f, desc)
	}

	// Fall back to generic remote component
	return createGenericRemote(ctx, f, desc)
}

// MeshClient returns the mesh client.
func (f *Factory) MeshClient() *mesh.Client {
	return f.meshClient
}

// NATSConn returns the NATS connection.
func (f *Factory) NATSConn() *nats.Conn {
	return f.natsConn
}

// Logger returns the logger.
func (f *Factory) Logger() *slog.Logger {
	return f.logger
}

// Default creator functions

func createRemoteMotor(ctx context.Context, f *Factory, desc mesh.ServiceDescriptor) (any, error) {
	return NewRemoteMotor(f.meshClient, f.natsConn, desc, WithRemoteMotorLogger(f.logger))
}

func createRemoteSensor(ctx context.Context, f *Factory, desc mesh.ServiceDescriptor) (any, error) {
	return NewRemoteSensor(f.meshClient, f.natsConn, desc, WithRemoteSensorLogger(f.logger))
}

func createRemoteCamera(ctx context.Context, f *Factory, desc mesh.ServiceDescriptor) (any, error) {
	return NewRemoteCamera(f.meshClient, f.natsConn, desc, WithRemoteCameraLogger(f.logger))
}

func createRemotePWMController(ctx context.Context, f *Factory, desc mesh.ServiceDescriptor) (any, error) {
	return NewRemotePWMController(f.meshClient, f.natsConn, desc, WithRemotePWMLogger(f.logger))
}

func createRemoteServo(ctx context.Context, f *Factory, desc mesh.ServiceDescriptor) (any, error) {
	return NewRemoteServo(f.meshClient, f.natsConn, desc, WithRemoteServoLogger(f.logger))
}

func createGenericRemote(ctx context.Context, f *Factory, desc mesh.ServiceDescriptor) (any, error) {
	return NewGenericRemote(f.meshClient, f.natsConn, desc, WithGenericRemoteLogger(f.logger))
}

// GetCommandSubject extracts the command subject from a service descriptor.
func GetCommandSubject(desc mesh.ServiceDescriptor) (string, error) {
	for _, sub := range desc.Subscribes {
		// Look for command or tx subjects
		if containsAny(sub, "command", "tx.command", ".command") {
			return sub, nil
		}
	}

	// Construct default command subject
	if len(desc.Subscribes) > 0 {
		return desc.Subscribes[0], nil
	}

	// Fall back to constructed subject
	if desc.RobotID != "" {
		return fmt.Sprintf("gsp.%s.tx.command", desc.Name), nil
	}

	return "", fmt.Errorf("no command subject found for %s", desc.Name)
}

// GetDataSubject extracts the data/sensor subject from a service descriptor.
func GetDataSubject(desc mesh.ServiceDescriptor) (string, error) {
	for _, pub := range desc.Publishes {
		// Look for sensor or data subjects
		if containsAny(pub, "sensor", "data", "rx.sensor") {
			return pub, nil
		}
	}

	// Use first publish subject
	if len(desc.Publishes) > 0 {
		return desc.Publishes[0], nil
	}

	// Fall back to constructed subject
	if desc.RobotID != "" {
		return fmt.Sprintf("gsp.%s.rx.sensor", desc.Name), nil
	}

	return "", fmt.Errorf("no data subject found for %s", desc.Name)
}

// containsAny checks if s contains any of the substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if contains(s, sub) {
			return true
		}
	}
	return false
}

// contains checks if s contains sub.
func contains(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
