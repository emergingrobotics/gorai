package metadata

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseFile parses a gorai-component.yaml file from disk
func ParseFile(path string) (*ComponentMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return Parse(data)
}

// Parse parses gorai-component.yaml content
func Parse(data []byte) (*ComponentMetadata, error) {
	var metadata ComponentMetadata

	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate required fields
	if metadata.SchemaVersion == "" {
		return nil, fmt.Errorf("schema_version is required")
	}
	if metadata.Component.Name == "" {
		return nil, fmt.Errorf("component.name is required")
	}
	if metadata.Component.Repository == "" {
		return nil, fmt.Errorf("component.repository is required")
	}
	if metadata.Component.Version == "" {
		return nil, fmt.Errorf("component.version is required")
	}
	if len(metadata.Component.Provides) == 0 {
		return nil, fmt.Errorf("component.provides is required and must have at least one entry")
	}
	if metadata.Component.Compatibility.GoraiVersion == "" {
		return nil, fmt.Errorf("component.compatibility.gorai_version is required")
	}

	return &metadata, nil
}

// FetchFromRepository fetches gorai-component.yaml from a git repository
func FetchFromRepository(repo, version string) (*ComponentMetadata, error) {
	// Try common hosting platforms
	urls := []string{
		fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/gorai-component.yaml",
			strings.TrimPrefix(repo, "github.com/"), version),
		fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/gorai-component.yml",
			strings.TrimPrefix(repo, "github.com/"), version),
		fmt.Sprintf("https://gitlab.com/%s/-/raw/%s/gorai-component.yaml",
			strings.TrimPrefix(repo, "gitlab.com/"), version),
	}

	var lastErr error
	for _, url := range urls {
		resp, err := http.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
			continue
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		return Parse(data)
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to fetch metadata from %s: %w", repo, lastErr)
	}
	return nil, fmt.Errorf("failed to fetch metadata from %s", repo)
}

// FindInDirectory searches for gorai-component.yaml in a directory
func FindInDirectory(dir string) (string, error) {
	candidates := []string{
		"gorai-component.yaml",
		"gorai-component.yml",
		".gorai-component.yaml",
	}

	for _, candidate := range candidates {
		path := filepath.Join(dir, candidate)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("gorai-component.yaml not found in %s", dir)
}

// Validate performs deep validation of component metadata
func (cm *ComponentMetadata) Validate() []error {
	var errors []error

	// Validate schema version
	if cm.SchemaVersion != "1.0" {
		errors = append(errors, fmt.Errorf("unsupported schema_version: %s (expected '1.0')", cm.SchemaVersion))
	}

	// Validate component name format (kebab-case)
	if !isKebabCase(cm.Component.Name) {
		errors = append(errors, fmt.Errorf("component.name must be in kebab-case: %s", cm.Component.Name))
	}

	// Validate version format (semver with v prefix)
	if !strings.HasPrefix(cm.Component.Version, "v") {
		errors = append(errors, fmt.Errorf("component.version must start with 'v': %s", cm.Component.Version))
	}

	// Validate provides entries
	for i, p := range cm.Component.Provides {
		if p.Type == "" {
			errors = append(errors, fmt.Errorf("component.provides[%d].type is required", i))
		}
		if p.Model == "" {
			errors = append(errors, fmt.Errorf("component.provides[%d].model is required", i))
		}
		if !isValidModelName(p.Model) {
			errors = append(errors, fmt.Errorf("component.provides[%d].model must be kebab-case or snake_case: %s", i, p.Model))
		}
	}

	// Validate configuration attributes
	for name, attr := range cm.Component.Configuration {
		if !isValidConfigName(name) {
			errors = append(errors, fmt.Errorf("configuration key must be snake_case: %s", name))
		}
		if attr.Type == "" {
			errors = append(errors, fmt.Errorf("configuration.%s.type is required", name))
		}
		validTypes := []string{"string", "int", "float", "bool", "array", "object"}
		if !contains(validTypes, attr.Type) {
			errors = append(errors, fmt.Errorf("configuration.%s.type must be one of %v: got %s", name, validTypes, attr.Type))
		}
	}

	// Validate maturity level if specified
	if cm.Component.Metadata != nil && cm.Component.Metadata.Maturity != "" {
		validMaturity := []string{"experimental", "alpha", "beta", "stable", "mature"}
		if !contains(validMaturity, cm.Component.Metadata.Maturity) {
			errors = append(errors, fmt.Errorf("metadata.maturity must be one of %v: got %s", validMaturity, cm.Component.Metadata.Maturity))
		}
	}

	return errors
}

// Helper functions

func isKebabCase(s string) bool {
	if s == "" {
		return false
	}
	// Must start and end with alphanumeric, can contain hyphens in between
	for i, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}
		if r == '-' && i > 0 && i < len(s)-1 {
			continue
		}
		return false
	}
	return true
}

func isValidModelName(s string) bool {
	// Allow kebab-case or snake_case
	if s == "" {
		return false
	}
	for i, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}
		if (r == '-' || r == '_') && i > 0 && i < len(s)-1 {
			continue
		}
		return false
	}
	return true
}

func isValidConfigName(s string) bool {
	// Must be snake_case
	if s == "" {
		return false
	}
	for i, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}
		if r == '_' && i > 0 && i < len(s)-1 {
			continue
		}
		return false
	}
	return true
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
