// Package fake provides a fake space implementation for testing.
package fake

import (
	"context"
	"fmt"
	"sync"

	"github.com/gorai/gorai/component/space"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("space", "fake", New)
}

// Space is a fake space for testing.
type Space struct {
	name       resource.Name
	mu         sync.RWMutex
	spaceType  space.SpaceType
	bounds     resource.Bounds
	contents   []string
	volume     float64 // cubic meters
	usedVolume float64
	maxWeight  float64
	weight     float64
}

// New creates a new fake space.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "space", nameStr)

	// Default to a 1m x 1m x 1m container
	volume := 1.0
	if v, ok := conf["volume"].(float64); ok {
		volume = v
	}

	maxWeight := 100.0
	if w, ok := conf["max_weight"].(float64); ok {
		maxWeight = w
	}

	s := &Space{
		name:      name,
		spaceType: space.SpaceTypeContainer,
		bounds: resource.Bounds{
			MinX: 0, MinY: 0, MinZ: 0,
			MaxX: 1, MaxY: 1, MaxZ: 1,
		},
		contents:   []string{},
		volume:     volume,
		usedVolume: 0,
		maxWeight:  maxWeight,
		weight:     0,
	}

	return s, nil
}

// NewWithName creates a fake space with a specific resource name.
func NewWithName(name resource.Name) *Space {
	return &Space{
		name:      name,
		spaceType: space.SpaceTypeContainer,
		bounds: resource.Bounds{
			MinX: 0, MinY: 0, MinZ: 0,
			MaxX: 1, MaxY: 1, MaxZ: 1,
		},
		contents:   []string{},
		volume:     1.0,
		usedVolume: 0,
		maxWeight:  100.0,
		weight:     0,
	}
}

// Name returns the space's resource name.
func (s *Space) Name() resource.Name {
	return s.name
}

// Reconfigure updates the space configuration.
func (s *Space) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if vol, ok := conf.GetFloat("volume"); ok {
		s.volume = vol
	}
	if maxWeight, ok := conf.GetFloat("max_weight"); ok {
		s.maxWeight = maxWeight
	}
	return nil
}

// DoCommand executes arbitrary commands for extensibility.
func (s *Space) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "add_content":
			if id, ok := cmd["id"].(string); ok {
				return map[string]any{"status": "ok"}, s.AddContent(ctx, id)
			}
			return nil, fmt.Errorf("invalid id value")
		case "remove_content":
			if id, ok := cmd["id"].(string); ok {
				return map[string]any{"status": "ok"}, s.RemoveContent(ctx, id)
			}
			return nil, fmt.Errorf("invalid id value")
		case "clear_contents":
			return map[string]any{"status": "ok"}, s.ClearContents(ctx)
		case "get_state":
			s.mu.RLock()
			defer s.mu.RUnlock()
			return map[string]any{
				"type":        s.spaceType.String(),
				"volume":      s.volume,
				"used_volume": s.usedVolume,
				"weight":      s.weight,
				"contents":    s.contents,
			}, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (s *Space) Close(ctx context.Context) error {
	return nil
}

// GetVolume returns the volume in cubic meters.
func (s *Space) GetVolume(ctx context.Context) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.volume, nil
}

// GetBounds returns the bounding box geometry.
func (s *Space) GetBounds(ctx context.Context) (*resource.Bounds, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bounds := s.bounds // Copy
	return &bounds, nil
}

// GetContents returns identifiers of what's currently in this space.
func (s *Space) GetContents(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]string, len(s.contents))
	copy(result, s.contents)
	return result, nil
}

// IsEmpty returns true if space contains nothing.
func (s *Space) IsEmpty(ctx context.Context) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.contents) == 0, nil
}

// Extended interface methods

// GetProperties returns the space properties.
func (s *Space) GetProperties(ctx context.Context) (space.Properties, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return space.Properties{
		Type:             s.spaceType,
		Name:             s.name.Name,
		MaxVolume:        s.volume,
		MaxWeight:        s.maxWeight,
		CanTrackContents: true,
		CanMeasureVolume: true,
	}, nil
}

// GetUsedVolume returns the volume currently occupied.
func (s *Space) GetUsedVolume(ctx context.Context) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.usedVolume, nil
}

// GetWeight returns the current weight of contents.
func (s *Space) GetWeight(ctx context.Context) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.weight, nil
}

// AddContent adds an item to the space contents.
func (s *Space) AddContent(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already exists
	for _, c := range s.contents {
		if c == id {
			return fmt.Errorf("item %q already in space", id)
		}
	}

	s.contents = append(s.contents, id)
	return nil
}

// RemoveContent removes an item from the space contents.
func (s *Space) RemoveContent(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, c := range s.contents {
		if c == id {
			s.contents = append(s.contents[:i], s.contents[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("item %q not found in space", id)
}

// ClearContents removes all tracked contents.
func (s *Space) ClearContents(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.contents = []string{}
	s.usedVolume = 0
	s.weight = 0
	return nil
}

// IsFull returns true if the space is at capacity.
func (s *Space) IsFull(ctx context.Context) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.usedVolume >= s.volume || s.weight >= s.maxWeight, nil
}

// Test helper methods

// SetBounds sets the bounding box (for testing).
func (s *Space) SetBounds(bounds resource.Bounds) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bounds = bounds
}

// SetUsedVolume sets the used volume (for testing).
func (s *Space) SetUsedVolume(vol float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.usedVolume = vol
}

// SetWeight sets the weight (for testing).
func (s *Space) SetWeight(weight float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.weight = weight
}

// SetType sets the space type (for testing).
func (s *Space) SetType(spaceType space.SpaceType) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.spaceType = spaceType
}

// Verify interface compliance.
var _ space.Space = (*Space)(nil)
var _ space.Extended = (*Space)(nil)
