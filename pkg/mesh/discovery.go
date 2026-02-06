package mesh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// ErrNotFound is returned when a resource is not found.
var ErrNotFound = errors.New("not found")

// FindServices queries services matching the given filters.
func (c *Client) FindServices(ctx context.Context, q Query) ([]ServiceDescriptor, error) {
	var services []ServiceDescriptor

	// List all keys in the services bucket
	keys, err := c.kv.Services().Keys(ctx)
	if err != nil {
		if errors.Is(err, jetstream.ErrNoKeysFound) {
			return services, nil
		}
		return nil, fmt.Errorf("failed to list service keys: %w", err)
	}

	for _, key := range keys {
		entry, err := c.kv.Services().Get(ctx, key)
		if err != nil {
			if errors.Is(err, jetstream.ErrKeyNotFound) {
				continue
			}
			c.logger.Warn("failed to get service", "key", key, "error", err)
			continue
		}

		var desc ServiceDescriptor
		if err := json.Unmarshal(entry.Value(), &desc); err != nil {
			c.logger.Warn("failed to unmarshal service", "key", key, "error", err)
			continue
		}

		// Update status based on last seen time
		desc.Status = c.computeStatus(desc)

		if q.Matches(desc) {
			services = append(services, desc)
		}
	}

	return services, nil
}

// computeStatus determines the status based on last seen time.
func (c *Client) computeStatus(desc ServiceDescriptor) ServiceStatus {
	age := time.Since(desc.LastSeen)

	if age < DefaultStaleThreshold {
		return StatusHealthy
	}
	if age < DefaultServiceTTL {
		return StatusStale
	}
	return StatusUnknown
}

// GetService retrieves a specific service by robot ID and name.
func (c *Client) GetService(ctx context.Context, robotID, name string) (*ServiceDescriptor, error) {
	// Search for the service with any instance ID
	pattern := ServiceKeyShort(robotID, name)

	keys, err := c.kv.Services().Keys(ctx)
	if err != nil {
		if errors.Is(err, jetstream.ErrNoKeysFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to list service keys: %w", err)
	}

	for _, key := range keys {
		if strings.HasPrefix(key, pattern+"/") {
			entry, err := c.kv.Services().Get(ctx, key)
			if err != nil {
				continue
			}

			var desc ServiceDescriptor
			if err := json.Unmarshal(entry.Value(), &desc); err != nil {
				continue
			}

			desc.Status = c.computeStatus(desc)
			return &desc, nil
		}
	}

	return nil, ErrNotFound
}

// GetServiceByID retrieves a service by its instance ID.
func (c *Client) GetServiceByID(ctx context.Context, id string) (*ServiceDescriptor, error) {
	keys, err := c.kv.Services().Keys(ctx)
	if err != nil {
		if errors.Is(err, jetstream.ErrNoKeysFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to list service keys: %w", err)
	}

	for _, key := range keys {
		if strings.HasSuffix(key, "/"+id) {
			entry, err := c.kv.Services().Get(ctx, key)
			if err != nil {
				continue
			}

			var desc ServiceDescriptor
			if err := json.Unmarshal(entry.Value(), &desc); err != nil {
				continue
			}

			desc.Status = c.computeStatus(desc)
			return &desc, nil
		}
	}

	return nil, ErrNotFound
}

// ListServices lists all registered services.
func (c *Client) ListServices(ctx context.Context) ([]ServiceDescriptor, error) {
	return c.FindServices(ctx, Query{})
}

// ListServicesByRobot lists all services for a specific robot.
func (c *Client) ListServicesByRobot(ctx context.Context, robotID string) ([]ServiceDescriptor, error) {
	return c.FindServices(ctx, Query{RobotID: robotID})
}

// FindChannels queries channels matching the given filters.
func (c *Client) FindChannels(ctx context.Context, q ChannelQuery) ([]ChannelDescriptor, error) {
	var channels []ChannelDescriptor

	keys, err := c.kv.Channels().Keys(ctx)
	if err != nil {
		if errors.Is(err, jetstream.ErrNoKeysFound) {
			return channels, nil
		}
		return nil, fmt.Errorf("failed to list channel keys: %w", err)
	}

	for _, key := range keys {
		entry, err := c.kv.Channels().Get(ctx, key)
		if err != nil {
			if errors.Is(err, jetstream.ErrKeyNotFound) {
				continue
			}
			c.logger.Warn("failed to get channel", "key", key, "error", err)
			continue
		}

		var ch ChannelDescriptor
		if err := json.Unmarshal(entry.Value(), &ch); err != nil {
			c.logger.Warn("failed to unmarshal channel", "key", key, "error", err)
			continue
		}

		if c.matchesChannelQuery(ch, q) {
			channels = append(channels, ch)
		}
	}

	return channels, nil
}

// matchesChannelQuery checks if a channel matches the query filters.
func (c *Client) matchesChannelQuery(ch ChannelDescriptor, q ChannelQuery) bool {
	if q.RobotID != "" && ch.RobotID != q.RobotID {
		return false
	}
	if q.Publisher != "" && ch.Publisher != q.Publisher {
		return false
	}
	if q.QoS != "" && ch.QoS != q.QoS {
		return false
	}
	if q.Direction != "" && ch.Direction != q.Direction {
		return false
	}
	if q.SubjectPattern != "" && !matchSubjectPattern(ch.Subject, q.SubjectPattern) {
		return false
	}
	return true
}

// matchSubjectPattern matches a subject against a NATS-style pattern.
func matchSubjectPattern(subject, pattern string) bool {
	// Simple implementation supporting * and >
	subjectParts := strings.Split(subject, ".")
	patternParts := strings.Split(pattern, ".")

	for i, pp := range patternParts {
		if pp == ">" {
			return true // Match rest
		}
		if i >= len(subjectParts) {
			return false
		}
		if pp != "*" && pp != subjectParts[i] {
			return false
		}
	}

	return len(subjectParts) == len(patternParts)
}

// GetChannel retrieves a specific channel by subject.
func (c *Client) GetChannel(ctx context.Context, subject string) (*ChannelDescriptor, error) {
	key := ChannelKey(subject)
	entry, err := c.kv.Channels().Get(ctx, key)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	var ch ChannelDescriptor
	if err := json.Unmarshal(entry.Value(), &ch); err != nil {
		return nil, fmt.Errorf("failed to unmarshal channel: %w", err)
	}

	return &ch, nil
}

// ListChannels lists all registered channels.
func (c *Client) ListChannels(ctx context.Context) ([]ChannelDescriptor, error) {
	return c.FindChannels(ctx, ChannelQuery{})
}

// ListChannelsByRobot lists all channels for a specific robot.
func (c *Client) ListChannelsByRobot(ctx context.Context, robotID string) ([]ChannelDescriptor, error) {
	return c.FindChannels(ctx, ChannelQuery{RobotID: robotID})
}

// GetSchema retrieves a schema by name and version.
func (c *Client) GetSchema(ctx context.Context, name, version string) (*SchemaDescriptor, error) {
	key := SchemaKey(name, version)
	entry, err := c.kv.Schemas().Get(ctx, key)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	var schema SchemaDescriptor
	if err := json.Unmarshal(entry.Value(), &schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	return &schema, nil
}

// GetSchemaByKey retrieves a schema by its full key (name/version).
func (c *Client) GetSchemaByKey(ctx context.Context, key string) (*SchemaDescriptor, error) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid schema key format: %s", key)
	}
	return c.GetSchema(ctx, parts[0], parts[1])
}

// GetChannelSchema retrieves the schema for a channel.
func (c *Client) GetChannelSchema(ctx context.Context, subject string) (*SchemaDescriptor, error) {
	ch, err := c.GetChannel(ctx, subject)
	if err != nil {
		return nil, err
	}

	if ch.Schema == "" {
		return nil, fmt.Errorf("channel %s has no schema defined", subject)
	}

	return c.GetSchemaByKey(ctx, ch.Schema)
}

// ListSchemas lists all registered schemas.
func (c *Client) ListSchemas(ctx context.Context) ([]SchemaDescriptor, error) {
	var schemas []SchemaDescriptor

	keys, err := c.kv.Schemas().Keys(ctx)
	if err != nil {
		if errors.Is(err, jetstream.ErrNoKeysFound) {
			return schemas, nil
		}
		return nil, fmt.Errorf("failed to list schema keys: %w", err)
	}

	for _, key := range keys {
		entry, err := c.kv.Schemas().Get(ctx, key)
		if err != nil {
			continue
		}

		var schema SchemaDescriptor
		if err := json.Unmarshal(entry.Value(), &schema); err != nil {
			continue
		}

		schemas = append(schemas, schema)
	}

	return schemas, nil
}

// DiscoverEndpoints finds all RPC endpoints for a given service type.
func (c *Client) DiscoverEndpoints(ctx context.Context, subtype string) ([]Endpoint, error) {
	services, err := c.FindServices(ctx, Query{Subtype: subtype})
	if err != nil {
		return nil, err
	}

	var endpoints []Endpoint
	for _, svc := range services {
		endpoints = append(endpoints, svc.Endpoints...)
	}

	return endpoints, nil
}

// GetRobots returns a list of all unique robot IDs with registered services.
func (c *Client) GetRobots(ctx context.Context) ([]string, error) {
	services, err := c.ListServices(ctx)
	if err != nil {
		return nil, err
	}

	robotSet := make(map[string]struct{})
	for _, svc := range services {
		robotSet[svc.RobotID] = struct{}{}
	}

	robots := make([]string, 0, len(robotSet))
	for r := range robotSet {
		robots = append(robots, r)
	}

	return robots, nil
}

// Summary returns a summary of the mesh state.
type Summary struct {
	Robots        int                        `json:"robots"`
	Services      int                        `json:"services"`
	Channels      int                        `json:"channels"`
	Schemas       int                        `json:"schemas"`
	HealthyCount  int                        `json:"healthy_count"`
	StaleCount    int                        `json:"stale_count"`
	ServicesByType map[ServiceType]int       `json:"services_by_type"`
}

// GetSummary returns a summary of the mesh state.
func (c *Client) GetSummary(ctx context.Context) (*Summary, error) {
	services, err := c.ListServices(ctx)
	if err != nil {
		return nil, err
	}

	channels, err := c.ListChannels(ctx)
	if err != nil {
		return nil, err
	}

	schemas, err := c.ListSchemas(ctx)
	if err != nil {
		return nil, err
	}

	robots := make(map[string]struct{})
	byType := make(map[ServiceType]int)
	var healthy, stale int

	for _, svc := range services {
		robots[svc.RobotID] = struct{}{}
		byType[svc.Type]++
		switch svc.Status {
		case StatusHealthy:
			healthy++
		case StatusStale:
			stale++
		}
	}

	return &Summary{
		Robots:         len(robots),
		Services:       len(services),
		Channels:       len(channels),
		Schemas:        len(schemas),
		HealthyCount:   healthy,
		StaleCount:     stale,
		ServicesByType: byType,
	}, nil
}
