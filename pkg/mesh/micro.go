package mesh

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
)

// RegistryService provides NATS micro service endpoints for mesh queries.
// This enables service discovery via NATS request-reply.
type RegistryService struct {
	client  *Client
	service micro.Service
	logger  *slog.Logger
}

// RegistryServiceConfig configures the registry micro service.
type RegistryServiceConfig struct {
	// Name is the service name (default: "gorai-registry").
	Name string
	// Version is the service version (default: "1.0.0").
	Version string
	// Description is the service description.
	Description string
	// Logger is the logger to use.
	Logger *slog.Logger
}

// DefaultRegistryServiceConfig returns the default configuration.
func DefaultRegistryServiceConfig() *RegistryServiceConfig {
	return &RegistryServiceConfig{
		Name:        "gorai-registry",
		Version:     "1.0.0",
		Description: "Gorai mesh service and channel registry",
		Logger:      slog.Default(),
	}
}

// NewRegistryService creates and starts a new registry micro service.
func NewRegistryService(client *Client, cfg *RegistryServiceConfig) (*RegistryService, error) {
	if cfg == nil {
		cfg = DefaultRegistryServiceConfig()
	}

	rs := &RegistryService{
		client: client,
		logger: cfg.Logger,
	}

	// Create the micro service
	svc, err := micro.AddService(client.nc, micro.Config{
		Name:        cfg.Name,
		Version:     cfg.Version,
		Description: cfg.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create micro service: %w", err)
	}

	rs.service = svc

	// Add endpoints
	if err := rs.addEndpoints(); err != nil {
		svc.Stop()
		return nil, err
	}

	cfg.Logger.Info("registry service started", "name", cfg.Name, "version", cfg.Version)

	return rs, nil
}

// addEndpoints registers all the query endpoints.
func (rs *RegistryService) addEndpoints() error {
	grp := rs.service.AddGroup("registry")

	// List services
	if err := grp.AddEndpoint("list-services", micro.HandlerFunc(rs.handleListServices),
		micro.WithEndpointMetadata(map[string]string{
			"description": "List all registered services",
		})); err != nil {
		return err
	}

	// Find services
	if err := grp.AddEndpoint("find-services", micro.HandlerFunc(rs.handleFindServices),
		micro.WithEndpointMetadata(map[string]string{
			"description": "Find services matching query filters",
		})); err != nil {
		return err
	}

	// Get service
	if err := grp.AddEndpoint("get-service", micro.HandlerFunc(rs.handleGetService),
		micro.WithEndpointMetadata(map[string]string{
			"description": "Get a specific service by robot ID and name",
		})); err != nil {
		return err
	}

	// List channels
	if err := grp.AddEndpoint("list-channels", micro.HandlerFunc(rs.handleListChannels),
		micro.WithEndpointMetadata(map[string]string{
			"description": "List all registered channels",
		})); err != nil {
		return err
	}

	// Find channels
	if err := grp.AddEndpoint("find-channels", micro.HandlerFunc(rs.handleFindChannels),
		micro.WithEndpointMetadata(map[string]string{
			"description": "Find channels matching query filters",
		})); err != nil {
		return err
	}

	// Get channel
	if err := grp.AddEndpoint("get-channel", micro.HandlerFunc(rs.handleGetChannel),
		micro.WithEndpointMetadata(map[string]string{
			"description": "Get a specific channel by subject",
		})); err != nil {
		return err
	}

	// List schemas
	if err := grp.AddEndpoint("list-schemas", micro.HandlerFunc(rs.handleListSchemas),
		micro.WithEndpointMetadata(map[string]string{
			"description": "List all registered schemas",
		})); err != nil {
		return err
	}

	// Get schema
	if err := grp.AddEndpoint("get-schema", micro.HandlerFunc(rs.handleGetSchema),
		micro.WithEndpointMetadata(map[string]string{
			"description": "Get a specific schema by name and version",
		})); err != nil {
		return err
	}

	// Get summary
	if err := grp.AddEndpoint("summary", micro.HandlerFunc(rs.handleSummary),
		micro.WithEndpointMetadata(map[string]string{
			"description": "Get a summary of the mesh state",
		})); err != nil {
		return err
	}

	// List robots
	if err := grp.AddEndpoint("list-robots", micro.HandlerFunc(rs.handleListRobots),
		micro.WithEndpointMetadata(map[string]string{
			"description": "List all robots with registered services",
		})); err != nil {
		return err
	}

	return nil
}

// Response wrapper for consistent API responses.
type apiResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func (rs *RegistryService) respond(req micro.Request, data interface{}) {
	resp := apiResponse{Success: true, Data: data}
	respData, err := json.Marshal(resp)
	if err != nil {
		respData = []byte(`{"success":false,"error":"failed to marshal response"}`)
	}
	req.Respond(respData)
}

func (rs *RegistryService) respondError(req micro.Request, err error) {
	resp := apiResponse{Success: false, Error: err.Error()}
	respData, marshalErr := json.Marshal(resp)
	if marshalErr != nil {
		respData = []byte(`{"success":false,"error":"failed to marshal error response"}`)
	}
	req.Respond(respData)
}

func (rs *RegistryService) handleListServices(req micro.Request) {
	ctx := context.Background()
	services, err := rs.client.ListServices(ctx)
	if err != nil {
		rs.respondError(req, err)
		return
	}
	rs.respond(req, services)
}

// FindServicesRequest is the request body for find-services.
type FindServicesRequest struct {
	RobotID  string            `json:"robot_id,omitempty"`
	Type     ServiceType       `json:"type,omitempty"`
	Subtype  string            `json:"subtype,omitempty"`
	Model    string            `json:"model,omitempty"`
	Name     string            `json:"name,omitempty"`
	Status   ServiceStatus     `json:"status,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func (rs *RegistryService) handleFindServices(req micro.Request) {
	ctx := context.Background()

	var qr FindServicesRequest
	if len(req.Data()) > 0 {
		if err := json.Unmarshal(req.Data(), &qr); err != nil {
			rs.respondError(req, fmt.Errorf("invalid request: %w", err))
			return
		}
	}

	q := Query{
		RobotID:  qr.RobotID,
		Type:     qr.Type,
		Subtype:  qr.Subtype,
		Model:    qr.Model,
		Name:     qr.Name,
		Status:   qr.Status,
		Metadata: qr.Metadata,
	}

	services, err := rs.client.FindServices(ctx, q)
	if err != nil {
		rs.respondError(req, err)
		return
	}
	rs.respond(req, services)
}

// GetServiceRequest is the request body for get-service.
type GetServiceRequest struct {
	RobotID string `json:"robot_id"`
	Name    string `json:"name"`
}

func (rs *RegistryService) handleGetService(req micro.Request) {
	ctx := context.Background()

	var gr GetServiceRequest
	if err := json.Unmarshal(req.Data(), &gr); err != nil {
		rs.respondError(req, fmt.Errorf("invalid request: %w", err))
		return
	}

	service, err := rs.client.GetService(ctx, gr.RobotID, gr.Name)
	if err != nil {
		rs.respondError(req, err)
		return
	}
	rs.respond(req, service)
}

func (rs *RegistryService) handleListChannels(req micro.Request) {
	ctx := context.Background()
	channels, err := rs.client.ListChannels(ctx)
	if err != nil {
		rs.respondError(req, err)
		return
	}
	rs.respond(req, channels)
}

// FindChannelsRequest is the request body for find-channels.
type FindChannelsRequest struct {
	RobotID        string    `json:"robot_id,omitempty"`
	Publisher      string    `json:"publisher,omitempty"`
	QoS            QoS       `json:"qos,omitempty"`
	Direction      Direction `json:"direction,omitempty"`
	SubjectPattern string    `json:"subject_pattern,omitempty"`
}

func (rs *RegistryService) handleFindChannels(req micro.Request) {
	ctx := context.Background()

	var qr FindChannelsRequest
	if len(req.Data()) > 0 {
		if err := json.Unmarshal(req.Data(), &qr); err != nil {
			rs.respondError(req, fmt.Errorf("invalid request: %w", err))
			return
		}
	}

	q := ChannelQuery{
		RobotID:        qr.RobotID,
		Publisher:      qr.Publisher,
		QoS:            qr.QoS,
		Direction:      qr.Direction,
		SubjectPattern: qr.SubjectPattern,
	}

	channels, err := rs.client.FindChannels(ctx, q)
	if err != nil {
		rs.respondError(req, err)
		return
	}
	rs.respond(req, channels)
}

// GetChannelRequest is the request body for get-channel.
type GetChannelRequest struct {
	Subject string `json:"subject"`
}

func (rs *RegistryService) handleGetChannel(req micro.Request) {
	ctx := context.Background()

	var gr GetChannelRequest
	if err := json.Unmarshal(req.Data(), &gr); err != nil {
		rs.respondError(req, fmt.Errorf("invalid request: %w", err))
		return
	}

	channel, err := rs.client.GetChannel(ctx, gr.Subject)
	if err != nil {
		rs.respondError(req, err)
		return
	}
	rs.respond(req, channel)
}

func (rs *RegistryService) handleListSchemas(req micro.Request) {
	ctx := context.Background()
	schemas, err := rs.client.ListSchemas(ctx)
	if err != nil {
		rs.respondError(req, err)
		return
	}
	rs.respond(req, schemas)
}

// GetSchemaRequest is the request body for get-schema.
type GetSchemaRequest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (rs *RegistryService) handleGetSchema(req micro.Request) {
	ctx := context.Background()

	var gr GetSchemaRequest
	if err := json.Unmarshal(req.Data(), &gr); err != nil {
		rs.respondError(req, fmt.Errorf("invalid request: %w", err))
		return
	}

	schema, err := rs.client.GetSchema(ctx, gr.Name, gr.Version)
	if err != nil {
		rs.respondError(req, err)
		return
	}
	rs.respond(req, schema)
}

func (rs *RegistryService) handleSummary(req micro.Request) {
	ctx := context.Background()
	summary, err := rs.client.GetSummary(ctx)
	if err != nil {
		rs.respondError(req, err)
		return
	}
	rs.respond(req, summary)
}

func (rs *RegistryService) handleListRobots(req micro.Request) {
	ctx := context.Background()
	robots, err := rs.client.GetRobots(ctx)
	if err != nil {
		rs.respondError(req, err)
		return
	}
	rs.respond(req, robots)
}

// Stop stops the registry service.
func (rs *RegistryService) Stop() error {
	return rs.service.Stop()
}

// Info returns information about the service.
func (rs *RegistryService) Info() micro.Info {
	return rs.service.Info()
}

// Stats returns statistics about the service.
func (rs *RegistryService) Stats() micro.Stats {
	return rs.service.Stats()
}

// QueryRegistry sends a query to a remote registry service.
// This is useful when you want to use the registry without creating a full client.
func QueryRegistry(nc *nats.Conn, endpoint string, request, response interface{}) error {
	subject := "gorai-registry.registry." + endpoint

	var reqData []byte
	if request != nil {
		var err error
		reqData, err = json.Marshal(request)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	msg, err := nc.Request(subject, reqData, 5*time.Second)
	if err != nil {
		return fmt.Errorf("registry query failed: %w", err)
	}

	var resp apiResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("registry error: %s", resp.Error)
	}

	if response != nil {
		dataBytes, _ := json.Marshal(resp.Data)
		if err := json.Unmarshal(dataBytes, response); err != nil {
			return fmt.Errorf("failed to unmarshal data: %w", err)
		}
	}

	return nil
}
