package componentregistry

// Registry is the top-level registry JSON structure.
type Registry struct {
	Version    string               `json:"version"`
	Components map[string]Component `json:"components"`
}

// Component is a single entry in the registry.
type Component struct {
	Module      string   `json:"module"`
	Type        string   `json:"type"`
	Model       string   `json:"model"`
	Description string   `json:"description"`
	Hardware    string   `json:"hardware,omitempty"`
	Version     string   `json:"version"`
	Tags        []string `json:"tags,omitempty"`
}
