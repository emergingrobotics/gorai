package rdl

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ParseFile parses a service.rdl.json file from disk
func ParseFile(path string) (*ServiceRDL, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return Parse(data)
}

// Parse parses service RDL JSON content
func Parse(data []byte) (*ServiceRDL, error) {
	var rdl ServiceRDL

	if err := json.Unmarshal(data, &rdl); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate required fields
	if rdl.SchemaVersion == "" {
		return nil, fmt.Errorf("schema_version is required")
	}
	if rdl.Service.Name == "" {
		return nil, fmt.Errorf("service.name is required")
	}
	if rdl.Service.Type == "" {
		return nil, fmt.Errorf("service.type is required")
	}
	if rdl.Service.Model == "" {
		return nil, fmt.Errorf("service.model is required")
	}
	if rdl.Service.Version == "" {
		return nil, fmt.Errorf("service.version is required")
	}

	return &rdl, nil
}

// FetchFromURL fetches service RDL from a URL
func FetchFromURL(url string) (*ServiceRDL, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RDL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return Parse(data)
}

// FetchFromRepository fetches service.rdl.json from a git repository
func FetchFromRepository(repo, version, subpath string) (*ServiceRDL, error) {
	if subpath == "" {
		subpath = "service.rdl.json"
	}

	// Try common hosting platforms
	urls := []string{
		fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s",
			strings.TrimPrefix(repo, "github.com/"), version, subpath),
		fmt.Sprintf("https://gitlab.com/%s/-/raw/%s/%s",
			strings.TrimPrefix(repo, "gitlab.com/"), version, subpath),
	}

	var lastErr error
	for _, url := range urls {
		rdl, err := FetchFromURL(url)
		if err != nil {
			lastErr = err
			continue
		}
		return rdl, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to fetch RDL from %s: %w", repo, lastErr)
	}
	return nil, fmt.Errorf("failed to fetch RDL from %s", repo)
}

// FindInDirectory searches for service.rdl.json in a directory
func FindInDirectory(dir string) (string, error) {
	candidates := []string{
		"service.rdl.json",
		"service-rdl.json",
		".service.rdl.json",
	}

	for _, candidate := range candidates {
		path := filepath.Join(dir, candidate)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("service.rdl.json not found in %s", dir)
}

// Validate performs deep validation of service RDL
func (rdl *ServiceRDL) Validate() []error {
	var errors []error

	// Validate schema version
	if rdl.SchemaVersion != "1.0" {
		errors = append(errors, fmt.Errorf("unsupported schema_version: %s (expected '1.0')", rdl.SchemaVersion))
	}

	// Validate service name format
	if !isValidServiceName(rdl.Service.Name) {
		errors = append(errors, fmt.Errorf("service.name must be kebab-case or snake_case: %s", rdl.Service.Name))
	}

	// Validate service type
	validTypes := []string{"vision", "slam", "navigation", "motion", "data_manager", "mlmodel", "generic"}
	if !contains(validTypes, rdl.Service.Type) {
		errors = append(errors, fmt.Errorf("service.type must be one of %v: got %s", validTypes, rdl.Service.Type))
	}

	// Validate version format
	if !strings.HasPrefix(rdl.Service.Version, "v") {
		errors = append(errors, fmt.Errorf("service.version must start with 'v': %s", rdl.Service.Version))
	}

	// Validate container configuration
	if rdl.Service.Container != nil {
		if rdl.Service.Container.DefaultImage == "" {
			errors = append(errors, fmt.Errorf("container.default_image is required when container is specified"))
		}

		// Validate pull policy
		if rdl.Service.Container.PullPolicy != "" {
			validPolicies := []string{"Always", "IfNotPresent", "Never"}
			if !contains(validPolicies, rdl.Service.Container.PullPolicy) {
				errors = append(errors, fmt.Errorf("container.pull_policy must be one of %v: got %s", validPolicies, rdl.Service.Container.PullPolicy))
			}
		}

		// Validate health check type
		if rdl.Service.Container.HealthCheck != nil {
			validHealthCheckTypes := []string{"http", "tcp", "nats", "exec"}
			if !contains(validHealthCheckTypes, rdl.Service.Container.HealthCheck.Type) {
				errors = append(errors, fmt.Errorf("container.health_check.type must be one of %v: got %s", validHealthCheckTypes, rdl.Service.Container.HealthCheck.Type))
			}
		}
	}

	// Validate NATS topics
	if rdl.Service.NATSTopics != nil {
		for i, topic := range rdl.Service.NATSTopics.Subscribes {
			if topic.Pattern == "" {
				errors = append(errors, fmt.Errorf("nats_topics.subscribes[%d].pattern is required", i))
			}
		}
		for i, topic := range rdl.Service.NATSTopics.Publishes {
			if topic.Pattern == "" {
				errors = append(errors, fmt.Errorf("nats_topics.publishes[%d].pattern is required", i))
			}
		}
	}

	// Validate configuration attributes
	for name, attr := range rdl.Service.Configuration {
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
	if rdl.Service.Metadata != nil && rdl.Service.Metadata.Maturity != "" {
		validMaturity := []string{"experimental", "alpha", "beta", "stable", "mature"}
		if !contains(validMaturity, rdl.Service.Metadata.Maturity) {
			errors = append(errors, fmt.Errorf("metadata.maturity must be one of %v: got %s", validMaturity, rdl.Service.Metadata.Maturity))
		}
	}

	return errors
}

// GetImageForVariant returns the image for a specific variant, or default if not found
func (rdl *ServiceRDL) GetImageForVariant(variant string) string {
	if rdl.Service.Container == nil {
		return ""
	}

	if variant != "" && rdl.Service.Container.ImageVariants != nil {
		if v, ok := rdl.Service.Container.ImageVariants[variant]; ok {
			return v.Image
		}
	}

	return rdl.Service.Container.DefaultImage
}

// Helper functions

func isValidServiceName(s string) bool {
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
