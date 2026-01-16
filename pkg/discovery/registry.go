// Package discovery provides functionality for discovering third-party
// components and services from registries and source repositories.
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gorai/gorai/pkg/component/metadata"
	"gopkg.in/yaml.v3"
)

const (
	// DefaultRegistryURL is the default component registry
	DefaultRegistryURL = "https://registry.gorai.dev"

	// Cache TTL for registry queries
	CacheTTL = 24 * time.Hour
)

// RegistryClient interacts with component/service registries
type RegistryClient struct {
	baseURL    string
	httpClient *http.Client
	cacheDir   string
}

// NewRegistryClient creates a new registry client
func NewRegistryClient() *RegistryClient {
	baseURL := os.Getenv("GORAI_REGISTRY_URL")
	if baseURL == "" {
		baseURL = DefaultRegistryURL
	}

	cacheDir := os.Getenv("GORAI_CACHE_DIR")
	if cacheDir == "" {
		home, _ := os.UserHomeDir()
		cacheDir = filepath.Join(home, ".cache", "gorai", "registry")
	}

	return &RegistryClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cacheDir: cacheDir,
	}
}

// SearchComponents searches for components matching the query
func (rc *RegistryClient) SearchComponents(ctx context.Context, query string, filters SearchFilters) ([]metadata.SearchResult, error) {
	// Build query parameters
	params := url.Values{}
	params.Set("q", query)
	if filters.Type != "" {
		params.Set("type", filters.Type)
	}
	if filters.Platform != "" {
		params.Set("platform", filters.Platform)
	}
	if filters.License != "" {
		params.Set("license", filters.License)
	}
	params.Set("limit", fmt.Sprintf("%d", filters.Limit))
	if filters.Limit == 0 {
		params.Set("limit", "20")
	}

	// Try registry API first
	endpoint := fmt.Sprintf("%s/api/v1/components/search?%s", rc.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := rc.httpClient.Do(req)
	if err != nil {
		// Fallback to GitHub search if registry unavailable
		return rc.searchGitHub(ctx, query, filters)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Fallback to GitHub search
		return rc.searchGitHub(ctx, query, filters)
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Results, nil
}

// SearchServices searches for services matching the query
func (rc *RegistryClient) SearchServices(ctx context.Context, query string, filters ServiceSearchFilters) ([]ServiceSearchResult, error) {
	params := url.Values{}
	params.Set("q", query)
	if filters.Type != "" {
		params.Set("type", filters.Type)
	}
	if filters.Accelerator != "" {
		params.Set("accelerator", filters.Accelerator)
	}
	params.Set("limit", fmt.Sprintf("%d", filters.Limit))
	if filters.Limit == 0 {
		params.Set("limit", "20")
	}

	endpoint := fmt.Sprintf("%s/api/v1/services/search?%s", rc.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := rc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("registry unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry error: HTTP %d", resp.StatusCode)
	}

	var result ServiceSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Results, nil
}

// GetComponentMetadata fetches metadata for a specific component
func (rc *RegistryClient) GetComponentMetadata(ctx context.Context, repository, version string) (*metadata.ComponentMetadata, error) {
	// Try cache first
	if md, err := rc.getCachedMetadata(repository, version); err == nil {
		return md, nil
	}

	// Fetch from repository
	md, err := metadata.FetchFromRepository(repository, version)
	if err != nil {
		return nil, err
	}

	// Cache the result
	_ = rc.cacheMetadata(repository, version, md)

	return md, nil
}

// searchGitHub falls back to GitHub search API
func (rc *RegistryClient) searchGitHub(ctx context.Context, query string, filters SearchFilters) ([]metadata.SearchResult, error) {
	// Build GitHub search query
	q := fmt.Sprintf("gorai-component %s", query)
	if filters.Type != "" {
		q += fmt.Sprintf(" %s", filters.Type)
	}

	endpoint := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&per_page=%d",
		url.QueryEscape(q), filters.Limit)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add GitHub token if available
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	}

	resp, err := rc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github search failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API error: HTTP %d", resp.StatusCode)
	}

	var ghResult GitHubSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&ghResult); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert GitHub results to our format
	results := make([]metadata.SearchResult, 0, len(ghResult.Items))
	for _, item := range ghResult.Items {
		results = append(results, metadata.SearchResult{
			Repository:  item.FullName,
			Name:        item.Name,
			Description: item.Description,
			License:     item.License.SPDXID,
			LastUpdated: item.UpdatedAt,
			Stars:       item.StargazersCount,
		})
	}

	return results, nil
}

// Cache management

func (rc *RegistryClient) getCachedMetadata(repository, version string) (*metadata.ComponentMetadata, error) {
	cachePath := rc.getCachePath(repository, version)

	// Check if cache exists and is fresh
	info, err := os.Stat(cachePath)
	if err != nil {
		return nil, err
	}

	if time.Since(info.ModTime()) > CacheTTL {
		return nil, fmt.Errorf("cache expired")
	}

	return metadata.ParseFile(cachePath)
}

func (rc *RegistryClient) cacheMetadata(repository, version string, md *metadata.ComponentMetadata) error {
	cachePath := rc.getCachePath(repository, version)

	// Ensure cache directory exists
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return err
	}

	// Marshal to YAML
	data, err := yaml.Marshal(md)
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0644)
}

func (rc *RegistryClient) getCachePath(repository, version string) string {
	// Create safe filename from repository and version
	safe := url.PathEscape(repository + "@" + version)
	return filepath.Join(rc.cacheDir, safe+".yaml")
}

// Types

// SearchFilters contains filters for component search
type SearchFilters struct {
	Type       string
	Capability string
	Platform   string
	License    string
	Limit      int
	Sort       string
}

// ServiceSearchFilters contains filters for service search
type ServiceSearchFilters struct {
	Type        string
	Accelerator string
	Platform    string
	Limit       int
}

// SearchResponse is the response from the registry search API
type SearchResponse struct {
	Results []metadata.SearchResult `json:"results"`
	Total   int                     `json:"total"`
	Page    int                     `json:"page"`
}

// ServiceSearchResponse is the response for service search
type ServiceSearchResponse struct {
	Results []ServiceSearchResult `json:"results"`
	Total   int                   `json:"total"`
	Page    int                   `json:"page"`
}

// ServiceSearchResult represents a service search result
type ServiceSearchResult struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Model       string    `json:"model"`
	Version     string    `json:"version"`
	Image       string    `json:"image"`
	Description string    `json:"description"`
	License     string    `json:"license"`
	Maturity    string    `json:"maturity"`
	Performance string    `json:"performance,omitempty"`
	LastUpdated time.Time `json:"last_updated"`
}

// GitHubSearchResult represents GitHub API search results
type GitHubSearchResult struct {
	TotalCount int          `json:"total_count"`
	Items      []GitHubRepo `json:"items"`
}

// GitHubRepo represents a GitHub repository in search results
type GitHubRepo struct {
	FullName        string    `json:"full_name"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	StargazersCount int       `json:"stargazers_count"`
	UpdatedAt       time.Time `json:"updated_at"`
	License         struct {
		SPDXID string `json:"spdx_id"`
	} `json:"license"`
}
