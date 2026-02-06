package mesh

import (
	"context"
	"encoding/json"
)

// Common schema names.
const (
	SchemaIMUReading    = "gorai.sensor.IMUReading"
	SchemaGPSReading    = "gorai.sensor.GPSReading"
	SchemaMotorCommand  = "gorai.actuator.MotorCommand"
	SchemaMotorState    = "gorai.actuator.MotorState"
	SchemaCameraFrame   = "gorai.camera.Frame"
	SchemaHeartbeat     = "gorai.system.Heartbeat"
	SchemaAnnouncement  = "gorai.system.Announcement"
)

// PredefinedSchemas contains commonly used message schemas.
var PredefinedSchemas = []SchemaDescriptor{
	{
		Name:        SchemaIMUReading,
		Version:     "v1",
		Format:      SchemaFormatJSON,
		Description: "Inertial measurement unit reading with linear acceleration and angular velocity",
		Definition: json.RawMessage(`{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type": "object",
			"required": ["timestamp"],
			"properties": {
				"timestamp": {
					"type": "string",
					"format": "date-time",
					"description": "Reading timestamp in RFC3339 format"
				},
				"linear_acceleration": {
					"$ref": "#/definitions/Vector3",
					"description": "Linear acceleration in m/s^2"
				},
				"angular_velocity": {
					"$ref": "#/definitions/Vector3",
					"description": "Angular velocity in rad/s"
				},
				"orientation": {
					"$ref": "#/definitions/Quaternion",
					"description": "Orientation as quaternion"
				},
				"temperature": {
					"type": "number",
					"description": "Sensor temperature in Celsius"
				}
			},
			"definitions": {
				"Vector3": {
					"type": "object",
					"properties": {
						"x": {"type": "number"},
						"y": {"type": "number"},
						"z": {"type": "number"}
					},
					"required": ["x", "y", "z"]
				},
				"Quaternion": {
					"type": "object",
					"properties": {
						"w": {"type": "number"},
						"x": {"type": "number"},
						"y": {"type": "number"},
						"z": {"type": "number"}
					},
					"required": ["w", "x", "y", "z"]
				}
			}
		}`),
	},
	{
		Name:        SchemaGPSReading,
		Version:     "v1",
		Format:      SchemaFormatJSON,
		Description: "GPS position reading with coordinates and fix quality",
		Definition: json.RawMessage(`{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type": "object",
			"required": ["timestamp", "latitude", "longitude"],
			"properties": {
				"timestamp": {
					"type": "string",
					"format": "date-time"
				},
				"latitude": {
					"type": "number",
					"minimum": -90,
					"maximum": 90,
					"description": "Latitude in decimal degrees"
				},
				"longitude": {
					"type": "number",
					"minimum": -180,
					"maximum": 180,
					"description": "Longitude in decimal degrees"
				},
				"altitude": {
					"type": "number",
					"description": "Altitude in meters above sea level"
				},
				"speed": {
					"type": "number",
					"minimum": 0,
					"description": "Ground speed in m/s"
				},
				"heading": {
					"type": "number",
					"minimum": 0,
					"maximum": 360,
					"description": "Heading in degrees from true north"
				},
				"fix_quality": {
					"type": "string",
					"enum": ["none", "gps", "dgps", "rtk_fixed", "rtk_float"],
					"description": "GPS fix quality"
				},
				"satellites": {
					"type": "integer",
					"minimum": 0,
					"description": "Number of satellites in view"
				},
				"hdop": {
					"type": "number",
					"minimum": 0,
					"description": "Horizontal dilution of precision"
				}
			}
		}`),
	},
	{
		Name:        SchemaMotorCommand,
		Version:     "v1",
		Format:      SchemaFormatJSON,
		Description: "Command to control a motor's power or velocity",
		Definition: json.RawMessage(`{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type": "object",
			"properties": {
				"power": {
					"type": "number",
					"minimum": -1,
					"maximum": 1,
					"description": "Power level from -1 (full reverse) to 1 (full forward)"
				},
				"velocity": {
					"type": "number",
					"description": "Target velocity in RPM (requires velocity control)"
				},
				"position": {
					"type": "number",
					"description": "Target position in revolutions (requires position control)"
				},
				"stop": {
					"type": "boolean",
					"description": "Emergency stop flag"
				}
			}
		}`),
	},
	{
		Name:        SchemaMotorState,
		Version:     "v1",
		Format:      SchemaFormatJSON,
		Description: "Current state of a motor",
		Definition: json.RawMessage(`{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type": "object",
			"required": ["timestamp"],
			"properties": {
				"timestamp": {
					"type": "string",
					"format": "date-time"
				},
				"power": {
					"type": "number",
					"description": "Current power level"
				},
				"velocity": {
					"type": "number",
					"description": "Current velocity in RPM"
				},
				"position": {
					"type": "number",
					"description": "Current position in revolutions"
				},
				"is_moving": {
					"type": "boolean",
					"description": "Whether the motor is currently moving"
				},
				"temperature": {
					"type": "number",
					"description": "Motor temperature in Celsius"
				},
				"current": {
					"type": "number",
					"description": "Motor current draw in Amps"
				}
			}
		}`),
	},
	{
		Name:        SchemaCameraFrame,
		Version:     "v1",
		Format:      SchemaFormatJSON,
		Description: "Camera frame metadata (actual image data sent separately)",
		Definition: json.RawMessage(`{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type": "object",
			"required": ["timestamp", "width", "height", "format"],
			"properties": {
				"timestamp": {
					"type": "string",
					"format": "date-time"
				},
				"sequence": {
					"type": "integer",
					"description": "Frame sequence number"
				},
				"width": {
					"type": "integer",
					"minimum": 1,
					"description": "Frame width in pixels"
				},
				"height": {
					"type": "integer",
					"minimum": 1,
					"description": "Frame height in pixels"
				},
				"format": {
					"type": "string",
					"enum": ["rgb8", "bgr8", "rgba8", "yuv420", "jpeg", "h264"],
					"description": "Pixel format"
				},
				"encoding": {
					"type": "string",
					"enum": ["raw", "jpeg", "png", "h264"],
					"description": "Image encoding"
				},
				"data_subject": {
					"type": "string",
					"description": "NATS subject where image data is published"
				}
			}
		}`),
	},
	{
		Name:        SchemaHeartbeat,
		Version:     "v1",
		Format:      SchemaFormatJSON,
		Description: "Service heartbeat message",
		Definition: json.RawMessage(`{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type": "object",
			"required": ["service_id", "status", "timestamp"],
			"properties": {
				"service_id": {
					"type": "string",
					"description": "Service instance UUID"
				},
				"status": {
					"type": "string",
					"enum": ["healthy", "degraded", "stale", "unknown"],
					"description": "Service health status"
				},
				"timestamp": {
					"type": "string",
					"format": "date-time"
				},
				"metrics": {
					"type": "object",
					"additionalProperties": {"type": "number"},
					"description": "Optional health metrics"
				}
			}
		}`),
	},
	{
		Name:        SchemaAnnouncement,
		Version:     "v1",
		Format:      SchemaFormatJSON,
		Description: "Service join/leave announcement",
		Definition: json.RawMessage(`{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type": "object",
			"required": ["type", "service", "timestamp"],
			"properties": {
				"type": {
					"type": "string",
					"enum": ["join", "leave"],
					"description": "Announcement type"
				},
				"service": {
					"$ref": "#/definitions/ServiceDescriptor"
				},
				"timestamp": {
					"type": "string",
					"format": "date-time"
				}
			},
			"definitions": {
				"ServiceDescriptor": {
					"type": "object",
					"required": ["id", "name", "type", "robot_id"],
					"properties": {
						"id": {"type": "string"},
						"name": {"type": "string"},
						"type": {"type": "string", "enum": ["component", "service"]},
						"subtype": {"type": "string"},
						"model": {"type": "string"},
						"robot_id": {"type": "string"},
						"version": {"type": "string"},
						"started_at": {"type": "string", "format": "date-time"},
						"last_seen": {"type": "string", "format": "date-time"},
						"status": {"type": "string"}
					}
				}
			}
		}`),
	},
}

// RegisterPredefinedSchemas registers all predefined schemas.
func (c *Client) RegisterPredefinedSchemas(ctx context.Context) error {
	for _, schema := range PredefinedSchemas {
		if err := c.RegisterSchema(ctx, schema); err != nil {
			return err
		}
	}
	return nil
}

// NewJSONSchema creates a new JSON schema descriptor.
func NewJSONSchema(name, version, description string, definition interface{}) (SchemaDescriptor, error) {
	defBytes, err := json.Marshal(definition)
	if err != nil {
		return SchemaDescriptor{}, err
	}

	return SchemaDescriptor{
		Name:        name,
		Version:     version,
		Format:      SchemaFormatJSON,
		Description: description,
		Definition:  defBytes,
	}, nil
}

// ValidateAgainstSchema validates data against a schema.
// Note: This is a placeholder - actual validation would require a JSON Schema library.
func (c *Client) ValidateAgainstSchema(ctx context.Context, schemaKey string, data []byte) error {
	// TODO: Implement actual validation using a JSON Schema library
	// For now, just verify the schema exists
	_, err := c.GetSchemaByKey(ctx, schemaKey)
	return err
}

// ChannelBuilder helps construct channel descriptors with schemas.
type ChannelBuilder struct {
	ch ChannelDescriptor
}

// NewChannelBuilder creates a new channel builder.
func NewChannelBuilder(subject string) *ChannelBuilder {
	return &ChannelBuilder{
		ch: ChannelDescriptor{
			Subject: subject,
			QoS:     QoSBestEffort,
		},
	}
}

// WithSchema sets the schema reference.
func (b *ChannelBuilder) WithSchema(schemaKey string) *ChannelBuilder {
	b.ch.Schema = schemaKey
	return b
}

// WithQoS sets the quality of service.
func (b *ChannelBuilder) WithQoS(qos QoS) *ChannelBuilder {
	b.ch.QoS = qos
	return b
}

// WithDirection sets the direction.
func (b *ChannelBuilder) WithDirection(dir Direction) *ChannelBuilder {
	b.ch.Direction = dir
	return b
}

// WithDescription sets the description.
func (b *ChannelBuilder) WithDescription(desc string) *ChannelBuilder {
	b.ch.Description = desc
	return b
}

// WithSampleRate sets the sample rate.
func (b *ChannelBuilder) WithSampleRate(rate string) *ChannelBuilder {
	b.ch.SampleRate = rate
	return b
}

// WithRobotID sets the robot ID.
func (b *ChannelBuilder) WithRobotID(robotID string) *ChannelBuilder {
	b.ch.RobotID = robotID
	return b
}

// Build returns the constructed channel descriptor.
func (b *ChannelBuilder) Build() ChannelDescriptor {
	return b.ch
}

// CommonChannels returns common channel descriptors for a robot.
func CommonChannels(robotID string) []ChannelDescriptor {
	prefix := "gorai." + robotID

	return []ChannelDescriptor{
		{
			Subject:     prefix + ".system.startup",
			QoS:         QoSReliable,
			Direction:   DirectionPub,
			RobotID:     robotID,
			Description: "System startup events",
		},
		{
			Subject:     prefix + ".system.shutdown",
			QoS:         QoSReliable,
			Direction:   DirectionPub,
			RobotID:     robotID,
			Description: "System shutdown events",
		},
		{
			Subject:     prefix + ".system.logs",
			QoS:         QoSBestEffort,
			Direction:   DirectionPub,
			RobotID:     robotID,
			Description: "System log messages",
		},
		{
			Subject:     prefix + ".system.diagnostics",
			QoS:         QoSRetained,
			Direction:   DirectionPub,
			RobotID:     robotID,
			Description: "System diagnostic information",
		},
	}
}
