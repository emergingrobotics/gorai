// Package mesh provides NATS-native service discovery for Gorai.
//
// The mesh package enables runtime service registration and discovery across
// independent processes using NATS KV as the persistent registry. This allows
// modules that are not part of a single compiled binary to automatically find
// available channels and services.
//
// # Architecture
//
// The mesh uses three NATS KV buckets:
//   - gorai-services: Active service registrations (TTL-based, requires heartbeat)
//   - gorai-channels: Channel/subject descriptors (persistent)
//   - gorai-schemas: Message schemas for channels (persistent, versioned)
//
// # Well-Known Subjects
//
//   - gorai.mesh.announce: Service join/leave announcements
//   - gorai.mesh.heartbeat.<service-id>: Per-service heartbeats
//   - $SRV.gorai.registry.*: NATS micro service query API
//
// # Usage
//
//	client := mesh.NewClient(natsConn)
//	reg, err := client.Register(ctx, mesh.ServiceDescriptor{...})
//	defer reg.Deregister()
//
// See the mesh subcommands (gorai mesh services, gorai mesh channels) for
// developer-time discovery.
package mesh

import (
	"encoding/json"
	"time"
)

// ServiceType identifies whether a resource is a component or service.
type ServiceType string

const (
	// TypeComponent is a hardware abstraction (motor, camera, sensor).
	TypeComponent ServiceType = "component"
	// TypeService is a software service (behavior, telemetry, vision).
	TypeService ServiceType = "service"
)

// ServiceStatus represents the health status of a service.
type ServiceStatus string

const (
	// StatusHealthy indicates the service is running and responsive.
	StatusHealthy ServiceStatus = "healthy"
	// StatusDegraded indicates the service is running but with issues.
	StatusDegraded ServiceStatus = "degraded"
	// StatusStale indicates the service missed heartbeats.
	StatusStale ServiceStatus = "stale"
	// StatusUnknown indicates the status cannot be determined.
	StatusUnknown ServiceStatus = "unknown"
)

// QoS defines the quality of service for a channel.
type QoS string

const (
	// QoSBestEffort uses core NATS, fire-and-forget delivery.
	QoSBestEffort QoS = "best_effort"
	// QoSReliable uses JetStream with acknowledgment.
	QoSReliable QoS = "reliable"
	// QoSRetained uses JetStream, keeps only last message per subject.
	QoSRetained QoS = "retained"
	// QoSHistory uses JetStream, keeps last N messages.
	QoSHistory QoS = "history"
)

// Direction indicates the message flow direction for a channel.
type Direction string

const (
	// DirectionPub indicates publish-only (sensor data).
	DirectionPub Direction = "pub"
	// DirectionSub indicates subscribe-only (commands).
	DirectionSub Direction = "sub"
	// DirectionReqRep indicates request-reply pattern.
	DirectionReqRep Direction = "req-rep"
	// DirectionBidirectional indicates both publish and subscribe.
	DirectionBidirectional Direction = "bidirectional"
)

// SchemaFormat identifies the schema definition format.
type SchemaFormat string

const (
	// SchemaFormatJSON uses JSON Schema.
	SchemaFormatJSON SchemaFormat = "json-schema"
	// SchemaFormatProtobuf uses Protocol Buffer descriptors.
	SchemaFormatProtobuf SchemaFormat = "protobuf"
	// SchemaFormatAvro uses Apache Avro schema.
	SchemaFormatAvro SchemaFormat = "avro"
)

// ServiceDescriptor describes a running service instance.
type ServiceDescriptor struct {
	// ID is a unique instance identifier (UUID).
	ID string `json:"id"`

	// Name is a human-readable service name (e.g., "motor-controller").
	Name string `json:"name"`

	// Type indicates component or service.
	Type ServiceType `json:"type"`

	// Subtype is the resource subtype (e.g., "motor", "camera", "behavior").
	Subtype string `json:"subtype"`

	// Model is the implementation model (e.g., "fake", "v4l2", "pwm").
	Model string `json:"model"`

	// RobotID identifies which robot this service belongs to.
	RobotID string `json:"robot_id"`

	// Version is the semantic version of this service.
	Version string `json:"version,omitempty"`

	// Endpoints lists RPC subjects this service exposes.
	Endpoints []Endpoint `json:"endpoints,omitempty"`

	// Publishes lists channel subjects this service writes to.
	Publishes []string `json:"publishes,omitempty"`

	// Subscribes lists channel subjects this service reads from.
	Subscribes []string `json:"subscribes,omitempty"`

	// Metadata contains custom key-value tags.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Host is the hostname or IP where this service runs.
	Host string `json:"host,omitempty"`

	// PID is the process ID (for debugging).
	PID int `json:"pid,omitempty"`

	// StartedAt is when the service started.
	StartedAt time.Time `json:"started_at"`

	// LastSeen is updated by heartbeat.
	LastSeen time.Time `json:"last_seen"`

	// Status is the current health status.
	Status ServiceStatus `json:"status"`
}

// Endpoint describes an RPC endpoint exposed by a service.
type Endpoint struct {
	// Name is the endpoint name (e.g., "SetPower", "GetReadings").
	Name string `json:"name"`

	// Subject is the NATS subject for this endpoint.
	Subject string `json:"subject"`

	// Description explains what this endpoint does.
	Description string `json:"description,omitempty"`

	// RequestSchema is the schema key for request messages.
	RequestSchema string `json:"request_schema,omitempty"`

	// ResponseSchema is the schema key for response messages.
	ResponseSchema string `json:"response_schema,omitempty"`
}

// ChannelDescriptor describes a NATS subject/channel.
type ChannelDescriptor struct {
	// Subject is the full NATS subject pattern.
	Subject string `json:"subject"`

	// Schema is a reference to the schema key (e.g., "gorai.sensor.IMUReading/v1").
	Schema string `json:"schema,omitempty"`

	// QoS is the quality of service level.
	QoS QoS `json:"qos"`

	// Direction indicates message flow direction.
	Direction Direction `json:"direction"`

	// Publisher is the service ID that owns/publishes to this channel.
	Publisher string `json:"publisher,omitempty"`

	// Description is a human-readable explanation.
	Description string `json:"description,omitempty"`

	// SampleRate describes the expected publish rate (e.g., "100Hz", "on-demand").
	SampleRate string `json:"sample_rate,omitempty"`

	// RobotID identifies which robot this channel belongs to.
	RobotID string `json:"robot_id,omitempty"`

	// CreatedAt is when this channel was registered.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when this channel was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

// SchemaDescriptor describes a message schema.
type SchemaDescriptor struct {
	// Name is the schema name (e.g., "gorai.sensor.IMUReading").
	Name string `json:"name"`

	// Version is the schema version (e.g., "v1", "v2").
	Version string `json:"version"`

	// Format indicates the schema format (json-schema, protobuf, avro).
	Format SchemaFormat `json:"format"`

	// Definition is the actual schema content.
	Definition json.RawMessage `json:"definition"`

	// Description explains the message type.
	Description string `json:"description,omitempty"`

	// Examples provides sample messages.
	Examples []json.RawMessage `json:"examples,omitempty"`

	// CreatedAt is when this schema was registered.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when this schema was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

// SchemaKey returns the full schema key (name/version).
func (s *SchemaDescriptor) SchemaKey() string {
	return s.Name + "/" + s.Version
}

// Query defines filters for service discovery.
type Query struct {
	// RobotID filters by robot.
	RobotID string

	// Type filters by component or service.
	Type ServiceType

	// Subtype filters by resource subtype.
	Subtype string

	// Model filters by implementation model.
	Model string

	// Name filters by service name (exact match).
	Name string

	// Status filters by health status.
	Status ServiceStatus

	// Metadata filters by metadata key-value pairs (all must match).
	Metadata map[string]string
}

// Matches returns true if the service matches the query filters.
func (q Query) Matches(s ServiceDescriptor) bool {
	if q.RobotID != "" && s.RobotID != q.RobotID {
		return false
	}
	if q.Type != "" && s.Type != q.Type {
		return false
	}
	if q.Subtype != "" && s.Subtype != q.Subtype {
		return false
	}
	if q.Model != "" && s.Model != q.Model {
		return false
	}
	if q.Name != "" && s.Name != q.Name {
		return false
	}
	if q.Status != "" && s.Status != q.Status {
		return false
	}
	for k, v := range q.Metadata {
		if s.Metadata[k] != v {
			return false
		}
	}
	return true
}

// ChannelQuery defines filters for channel discovery.
type ChannelQuery struct {
	// RobotID filters by robot.
	RobotID string

	// Publisher filters by publishing service ID.
	Publisher string

	// QoS filters by quality of service.
	QoS QoS

	// Direction filters by message direction.
	Direction Direction

	// SubjectPattern matches subjects (supports * and > wildcards).
	SubjectPattern string
}

// EventType identifies the type of mesh event.
type EventType string

const (
	// EventServiceJoined indicates a new service registered.
	EventServiceJoined EventType = "service_joined"
	// EventServiceLeft indicates a service deregistered or timed out.
	EventServiceLeft EventType = "service_left"
	// EventServiceUpdated indicates a service descriptor changed.
	EventServiceUpdated EventType = "service_updated"
	// EventChannelAdded indicates a new channel was registered.
	EventChannelAdded EventType = "channel_added"
	// EventChannelRemoved indicates a channel was removed.
	EventChannelRemoved EventType = "channel_removed"
	// EventChannelUpdated indicates a channel descriptor changed.
	EventChannelUpdated EventType = "channel_updated"
)

// Event represents a mesh change notification.
type Event struct {
	// Type is the event type.
	Type EventType `json:"type"`

	// Service is set for service events.
	Service *ServiceDescriptor `json:"service,omitempty"`

	// Channel is set for channel events.
	Channel *ChannelDescriptor `json:"channel,omitempty"`

	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`
}

// Announcement is published to gorai.mesh.announce on service join/leave.
type Announcement struct {
	// Type is "join" or "leave".
	Type string `json:"type"`

	// Service is the service descriptor.
	Service ServiceDescriptor `json:"service"`

	// Timestamp is when this announcement was made.
	Timestamp time.Time `json:"timestamp"`
}

// HeartbeatMessage is sent periodically to gorai.mesh.heartbeat.<service-id>.
type HeartbeatMessage struct {
	// ServiceID identifies the service.
	ServiceID string `json:"service_id"`

	// Status is the current health status.
	Status ServiceStatus `json:"status"`

	// Metrics contains optional health metrics.
	Metrics map[string]float64 `json:"metrics,omitempty"`

	// Timestamp is when this heartbeat was sent.
	Timestamp time.Time `json:"timestamp"`
}
