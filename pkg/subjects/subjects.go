// Package subjects defines NATS subject naming conventions for Gorai.
//
// These suffixes are the wire form of NCP (the NATS Capability Protocol):
// a resource (sensor) is read on .state and streamed on .data; a tool
// (actuator) is invoked on .command; a capability pushes asynchronous
// notifications on .event. See ../../VISION.md.
//
// Subject Structure:
//
//	gorai.<robot_id>.<component>.<message_type>
//	gorai.<robot_id>.system.<category>
//
// Examples:
//
//	gorai.my-robot.main_camera.data         - Camera image data (resource stream)
//	gorai.my-robot.left_motor.command       - Motor command (tool invocation)
//	gorai.my-robot.imu.state                - IMU state (resource snapshot)
//	gorai.my-robot.estop_button.event       - Asynchronous event (notification)
//	gorai.my-robot.system.startup          - System startup events
//	gorai.my-robot.system.logs             - System logs
//	gorai.my-robot.system.diagnostics      - System diagnostics
package subjects

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
	// Event is for asynchronous notifications pushed by a capability
	// (faults, limit switches, threshold crossings, button presses) — the
	// NCP equivalent of an MCP server-to-client notification.
	Event = "event"
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
)

// Builder helps construct subject strings.
type Builder struct {
	robotID string
}

// NewBuilder creates a new subject builder for the given robot ID.
func NewBuilder(robotID string) *Builder {
	return &Builder{robotID: robotID}
}

// Component returns a subject for a specific component and message type.
// Example: Component("main_camera", "data") -> "gorai.my-robot.main_camera.data"
func (b *Builder) Component(component, messageType string) string {
	return fmt.Sprintf("gorai.%s.%s.%s", b.robotID, component, messageType)
}

// ComponentData returns a data subject for a component.
func (b *Builder) ComponentData(component string) string {
	return b.Component(component, Data)
}

// ComponentCommand returns a command subject for a component.
func (b *Builder) ComponentCommand(component string) string {
	return b.Component(component, Command)
}

// ComponentState returns a state subject for a component.
func (b *Builder) ComponentState(component string) string {
	return b.Component(component, State)
}

// ComponentStatus returns a status subject for a component.
func (b *Builder) ComponentStatus(component string) string {
	return b.Component(component, Status)
}

// ComponentEvent returns an event subject for a component.
// Capabilities publish asynchronous notifications here (faults, limits,
// button presses); observers subscribe without polling.
func (b *Builder) ComponentEvent(component string) string {
	return b.Component(component, Event)
}

// System returns a system subject for the given category.
// Example: System("startup") -> "gorai.my-robot.system.startup"
func (b *Builder) System(category string) string {
	return fmt.Sprintf("gorai.%s.%s.%s", b.robotID, SystemComponent, category)
}

// SystemStartup returns the system startup subject.
func (b *Builder) SystemStartup() string {
	return b.System(Startup)
}

// SystemLogs returns the system logs subject.
func (b *Builder) SystemLogs() string {
	return b.System(Logs)
}

// SystemDiagnostics returns the system diagnostics subject.
func (b *Builder) SystemDiagnostics() string {
	return b.System(Diagnostics)
}

// SystemHeartbeat returns the system heartbeat subject.
func (b *Builder) SystemHeartbeat() string {
	return b.System(Heartbeat)
}

// SystemShutdown returns the system shutdown subject.
func (b *Builder) SystemShutdown() string {
	return b.System(Shutdown)
}

// SystemEmergencyStop returns the emergency stop subject.
// All actuators should subscribe to this and immediately cease motion.
func (b *Builder) SystemEmergencyStop() string {
	return b.System(EmergencyStop)
}

// GlobalEmergencyStop returns the global emergency stop subject (all robots).
func GlobalEmergencyStop() string {
	return "gorai.estop"
}

// All returns a wildcard subject for all messages from this robot.
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

// Event types for startup events
const (
	EventComponentDetected = "component_detected"
	EventComponentMissing  = "component_missing"
	EventComponentError    = "component_error"
	EventRobotStarted      = "robot_started"
	EventRobotReady        = "robot_ready"
	EventRobotShutdown     = "robot_shutdown"
)
