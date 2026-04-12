package componentregistry

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// LoadFromFile loads a registry from a local JSON file.
func LoadFromFile(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read registry file: %w", err)
	}
	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse registry JSON: %w", err)
	}
	return &reg, nil
}

// Search finds components matching a query string. Matches against name,
// type, model, description, and tags. Case-insensitive.
func (r *Registry) Search(query string) []Component {
	q := strings.ToLower(query)
	var results []Component
	for name, c := range r.Components {
		if matches(q, name, c) {
			results = append(results, c)
		}
	}
	return results
}

// Lookup finds a component by its registry key (e.g., "sensor/hc-sr04").
func (r *Registry) Lookup(key string) (Component, bool) {
	c, ok := r.Components[key]
	return c, ok
}

func matches(query, name string, c Component) bool {
	fields := []string{
		strings.ToLower(name),
		strings.ToLower(c.Type),
		strings.ToLower(c.Model),
		strings.ToLower(c.Description),
	}
	for _, tag := range c.Tags {
		fields = append(fields, strings.ToLower(tag))
	}
	for _, field := range fields {
		if strings.Contains(field, query) {
			return true
		}
	}
	return false
}
