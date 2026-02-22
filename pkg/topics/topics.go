// Package topics defines NATS topic naming conventions for Gorai.
//
// Topic Structure:
//
//	gorai.<robot_id>.<component>.<message_type>
//	gorai.<robot_id>.system.<category>
//
// Examples:
//
//	gorai.my-robot.main_camera.data         - Camera image data
//	gorai.my-robot.left_motor.command       - Motor command
//	gorai.my-robot.imu.state                - IMU state
//	gorai.my-robot.system.startup          - System startup events
//	gorai.my-robot.system.logs             - System logs
//	gorai.my-robot.system.diagnostics      - System diagnostics
package topics

import "fmt"

// Message types
const (
	// Data is for sensor readings and continuous data streams
	Data = "data"
	// Command is for control commands to actuators
	Command = "command"
	// State is for component state updates
	State = "state"
	// Status is for component status (health, errors)
	Status = "status"
)

// System categories (component name is system)
const (
	// SystemComponent is the special component name for system-level messages
	SystemComponent = "system"

	// Startup is for system startup events (component detection, initialization)
	Startup = "startup"
	// Logs is for system log messages
	Logs = "logs"
	// Diagnostics is for system diagnostic information
	Diagnostics = "diagnostics"
	// Heartbeat is for system heartbeat/keepalive messages
	Heartbeat = "heartbeat"
	// Shutdown is for system shutdown events
	Shutdown = "shutdown"
	// EmergencyStop is for emergency stop commands that halt all actuators
	EmergencyStop = "estop"
	// ConfigReloaded is for config hot-reload events (success or rejection)
	ConfigReloaded = "config_reloaded"
)

// Builder helps construct topic strings.
type Builder struct {
	robotID string
}

// NewBuilder creates a new topic builder for the given robot ID.
func NewBuilder(robotID string) *Builder {
	return &Builder{robotID: robotID}
}

// Component returns a topic for a specific component and message type.
// Example: Component("main_camera", "data") -> "gorai.my-robot.main_camera.data"
func (b *Builder) Component(component, messageType string) string {
	return fmt.Sprintf("gorai.%s.%s.%s", b.robotID, component, messageType)
}

// ComponentData returns a data topic for a component.
func (b *Builder) ComponentData(component string) string {
	return b.Component(component, Data)
}

// ComponentCommand returns a command topic for a component.
func (b *Builder) ComponentCommand(component string) string {
	return b.Component(component, Command)
}

// ComponentState returns a state topic for a component.
func (b *Builder) ComponentState(component string) string {
	return b.Component(component, State)
}

// ComponentStatus returns a status topic for a component.
func (b *Builder) ComponentStatus(component string) string {
	return b.Component(component, Status)
}

// System returns a system topic for the given category.
// Example: System("startup") -> "gorai.my-robot.system.startup"
func (b *Builder) System(category string) string {
	return fmt.Sprintf("gorai.%s.%s.%s", b.robotID, SystemComponent, category)
}

// SystemStartup returns the system startup topic.
func (b *Builder) SystemStartup() string {
	return b.System(Startup)
}

// SystemLogs returns the system logs topic.
func (b *Builder) SystemLogs() string {
	return b.System(Logs)
}

// SystemDiagnostics returns the system diagnostics topic.
func (b *Builder) SystemDiagnostics() string {
	return b.System(Diagnostics)
}

// SystemHeartbeat returns the system heartbeat topic.
func (b *Builder) SystemHeartbeat() string {
	return b.System(Heartbeat)
}

// SystemShutdown returns the system shutdown topic.
func (b *Builder) SystemShutdown() string {
	return b.System(Shutdown)
}

// SystemEmergencyStop returns the emergency stop topic.
// All actuators should subscribe to this and immediately cease motion.
func (b *Builder) SystemEmergencyStop() string {
	return b.System(EmergencyStop)
}

// SystemConfigReloaded returns the config reload event topic.
func (b *Builder) SystemConfigReloaded() string {
	return b.System(ConfigReloaded)
}

// GlobalEmergencyStop returns the global emergency stop topic (all robots).
func GlobalEmergencyStop() string {
	return "gorai.estop"
}

// All returns a wildcard topic for all messages from this robot.
// Example: All() -> "gorai.my-robot.>"
func (b *Builder) All() string {
	return fmt.Sprintf("gorai.%s.>", b.robotID)
}

// AllComponents returns a wildcard for all component messages of a given type.
// Example: AllComponents("data") -> "gorai.my-robot.*.data"
func (b *Builder) AllComponents(messageType string) string {
	return fmt.Sprintf("gorai.%s.*.%s", b.robotID, messageType)
}

// StartupEvent represents a startup event message.
type StartupEvent struct {
	// EventType: "component_detected", "component_missing", "component_error", "robot_started", "robot_ready"
	EventType string `json:"event_type"`
	// Component name (empty for robot-level events)
	Component string `json:"component,omitempty"`
	// ComponentType (e.g., "camera", "motor")
	ComponentType string `json:"component_type,omitempty"`
	// Message is a human-readable description
	Message string `json:"message"`
	// Details contains additional information
	Details map[string]any `json:"details,omitempty"`
	// Timestamp in RFC3339 format
	Timestamp string `json:"timestamp"`
	// Success indicates if the event represents success or failure
	Success bool `json:"success"`
}

// ConfigReloadEvent represents a config hot-reload event published to NATS.
type ConfigReloadEvent struct {
	Timestamp         string   `json:"timestamp"`
	Rejected          bool     `json:"rejected"`
	Reason            string   `json:"reason,omitempty"`
	UpdatedComponents []string `json:"updated_components,omitempty"`
	UpdatedServices   []string `json:"updated_services,omitempty"`
	FailedComponents  []string `json:"failed_components,omitempty"`
	FailedServices    []string `json:"failed_services,omitempty"`
}

// Event types for startup events
const (
	EventComponentDetected = "component_detected"
	EventComponentMissing  = "component_missing"
	EventComponentError    = "component_error"
	EventRobotStarted      = "robot_started"
	EventRobotReady        = "robot_ready"
	EventRobotShutdown     = "robot_shutdown"
)
