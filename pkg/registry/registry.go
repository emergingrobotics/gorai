// Package registry provides component and service registration.
//
// The registry allows implementations to register themselves at init time,
// enabling configuration-driven instantiation without explicit imports.
package registry

import (
	"context"
	"fmt"
	"sync"
)

// Config holds configuration for a resource.
type Config map[string]any

// Dependencies provides access to dependent resources.
type Dependencies interface {
	Get(name string) (any, error)
	GetByType(subtype string) ([]any, error)
}

// Constructor creates a new resource instance.
type Constructor func(ctx context.Context, deps Dependencies, conf Config) (any, error)

var (
	mu         sync.RWMutex
	components = make(map[string]map[string]Constructor) // subtype -> model -> constructor
	services   = make(map[string]map[string]Constructor) // subtype -> model -> constructor
)

// RegisterComponent registers a component implementation.
func RegisterComponent(subtype, model string, ctor Constructor) {
	mu.Lock()
	defer mu.Unlock()

	if components[subtype] == nil {
		components[subtype] = make(map[string]Constructor)
	}
	components[subtype][model] = ctor
}

// RegisterService registers a service implementation.
func RegisterService(subtype, model string, ctor Constructor) {
	mu.Lock()
	defer mu.Unlock()

	if services[subtype] == nil {
		services[subtype] = make(map[string]Constructor)
	}
	services[subtype][model] = ctor
}

// IsRegistered returns true if a component with the given subtype and model
// has been registered.
func IsRegistered(subtype, model string) bool {
	mu.RLock()
	defer mu.RUnlock()
	if m, ok := components[subtype]; ok {
		_, ok := m[model]
		return ok
	}
	return false
}

// LookupComponent finds a registered component constructor.
func LookupComponent(subtype, model string) (Constructor, error) {
	mu.RLock()
	defer mu.RUnlock()

	if m, ok := components[subtype]; ok {
		if ctor, ok := m[model]; ok {
			return ctor, nil
		}
		return nil, fmt.Errorf("model %q not found for component %q", model, subtype)
	}
	return nil, fmt.Errorf("component subtype %q not found", subtype)
}

// LookupService finds a registered service constructor.
func LookupService(subtype, model string) (Constructor, error) {
	mu.RLock()
	defer mu.RUnlock()

	if m, ok := services[subtype]; ok {
		if ctor, ok := m[model]; ok {
			return ctor, nil
		}
		return nil, fmt.Errorf("model %q not found for service %q", model, subtype)
	}
	return nil, fmt.Errorf("service subtype %q not found", subtype)
}

// ListComponents returns all registered component subtypes and models.
func ListComponents() map[string][]string {
	mu.RLock()
	defer mu.RUnlock()

	result := make(map[string][]string)
	for subtype, models := range components {
		for model := range models {
			result[subtype] = append(result[subtype], model)
		}
	}
	return result
}

// ListServices returns all registered service subtypes and models.
func ListServices() map[string][]string {
	mu.RLock()
	defer mu.RUnlock()

	result := make(map[string][]string)
	for subtype, models := range services {
		for model := range models {
			result[subtype] = append(result[subtype], model)
		}
	}
	return result
}
