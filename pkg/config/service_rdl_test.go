package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseServiceRDL(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name: "valid service rdl",
			json: `{
				"$schema": "https://gorai.dev/schemas/service-rdl-v1.json",
				"version": "1",
				"kind": "service",
				"service": {
					"type": "vision",
					"model": "yolo",
					"description": "Test detection service"
				},
				"topics": {
					"subscribe": [
						{
							"name": "input_frame",
							"pattern": "{namespace}.camera.{input_component}.frame",
							"description": "Input video frames"
						}
					],
					"publish": [
						{
							"name": "detections",
							"pattern": "{namespace}.detection.{service}.objects",
							"description": "Detection results"
						}
					]
				},
				"attributes": {
					"confidence_threshold": {
						"type": "float",
						"required": true,
						"description": "Detection confidence threshold"
					}
				}
			}`,
			wantErr: false,
		},
		{
			name: "missing required fields",
			json: `{
				"version": "1"
			}`,
			wantErr: true,
		},
		{
			name: "invalid json",
			json: `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rdl, err := ParseServiceRDL([]byte(tt.json), "test.json")
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if rdl == nil {
				t.Error("expected rdl but got nil")
			}
		})
	}
}

func TestServiceRDLValidate(t *testing.T) {
	tests := []struct {
		name    string
		rdl     *ServiceRDL
		wantErr bool
	}{
		{
			name: "valid rdl",
			rdl: &ServiceRDL{
				Version: "1",
				Kind:    "service",
				Service: ServiceRDLMeta{
					Type:        "vision",
					Model:       "yolo",
					Description: "A test service",
				},
				Topics: ServiceRDLTopics{
					Subscribe: []ServiceRDLTopicEntry{
						{Name: "input", Pattern: "test.input"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing version",
			rdl: &ServiceRDL{
				Kind: "service",
				Service: ServiceRDLMeta{
					Type:  "test",
					Model: "v1",
				},
				Topics: ServiceRDLTopics{
					Subscribe: []ServiceRDLTopicEntry{{Name: "t", Pattern: "p"}},
				},
			},
			wantErr: true,
		},
		{
			name: "missing kind",
			rdl: &ServiceRDL{
				Version: "1",
				Service: ServiceRDLMeta{
					Type:  "test",
					Model: "v1",
				},
				Topics: ServiceRDLTopics{
					Subscribe: []ServiceRDLTopicEntry{{Name: "t", Pattern: "p"}},
				},
			},
			wantErr: true,
		},
		{
			name: "missing service type",
			rdl: &ServiceRDL{
				Version: "1",
				Kind:    "service",
				Service: ServiceRDLMeta{
					Model: "v1",
				},
				Topics: ServiceRDLTopics{
					Subscribe: []ServiceRDLTopicEntry{{Name: "t", Pattern: "p"}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rdl.Validate("")
			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoadServiceRDL(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "service-rdl-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a valid service RDL file
	validRDL := `{
		"version": "1",
		"kind": "service",
		"service": {
			"type": "test-type",
			"model": "test-model",
			"description": "Test"
		},
		"topics": {
			"subscribe": [{"name": "input", "pattern": "test.in"}],
			"publish": []
		}
	}`

	validPath := filepath.Join(tmpDir, "valid.rdl.json")
	if err := os.WriteFile(validPath, []byte(validRDL), 0644); err != nil {
		t.Fatalf("failed to write valid rdl: %v", err)
	}

	// Test loading valid file
	rdl, err := LoadServiceRDL(validPath)
	if err != nil {
		t.Errorf("failed to load valid rdl: %v", err)
	}
	if rdl == nil {
		t.Error("expected rdl but got nil")
	}
	if rdl != nil && rdl.Service.Type != "test-type" {
		t.Errorf("expected service type 'test-type', got %q", rdl.Service.Type)
	}

	// Test loading non-existent file
	_, err = LoadServiceRDL(filepath.Join(tmpDir, "nonexistent.json"))
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestMergeAttributes(t *testing.T) {
	rdl := &ServiceRDL{
		Version: "1",
		Kind:    "service",
		Service: ServiceRDLMeta{Type: "test", Model: "v1"},
		Topics:  ServiceRDLTopics{Subscribe: []ServiceRDLTopicEntry{{Name: "t", Pattern: "p"}}},
		Attrs: ServiceRDLAttributes{
			"threshold": {Type: "float", Default: 0.5},
			"debug":     {Type: "bool", Default: false},
		},
	}

	// Test with no overrides
	provided := map[string]any{}
	merged := rdl.MergeAttributes(provided)
	if merged["threshold"] != 0.5 {
		t.Errorf("expected threshold=0.5, got %v", merged["threshold"])
	}
	if merged["debug"] != false {
		t.Errorf("expected debug=false, got %v", merged["debug"])
	}

	// Test with override
	provided = map[string]any{"threshold": 0.8}
	merged = rdl.MergeAttributes(provided)
	if merged["threshold"] != 0.8 {
		t.Errorf("expected threshold=0.8, got %v", merged["threshold"])
	}
}

func TestValidateAttributes(t *testing.T) {
	rdl := &ServiceRDL{
		Version: "1",
		Kind:    "service",
		Service: ServiceRDLMeta{Type: "test", Model: "v1"},
		Topics:  ServiceRDLTopics{Subscribe: []ServiceRDLTopicEntry{{Name: "t", Pattern: "p"}}},
		Attrs: ServiceRDLAttributes{
			"threshold": {Type: "float", Required: true},
			"optional":  {Type: "string", Required: false},
		},
	}

	// Test missing required
	err := rdl.ValidateAttributes(map[string]any{})
	if err == nil {
		t.Error("expected error for missing required attribute")
	}

	// Test with required present
	err = rdl.ValidateAttributes(map[string]any{"threshold": 0.5})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test wrong type
	err = rdl.ValidateAttributes(map[string]any{"threshold": "not a number"})
	if err == nil {
		t.Error("expected error for wrong type")
	}
}

func TestGetRequiredAttributes(t *testing.T) {
	rdl := &ServiceRDL{
		Attrs: ServiceRDLAttributes{
			"required1": {Type: "string", Required: true},
			"required2": {Type: "int", Required: true},
			"optional1": {Type: "bool", Required: false},
		},
	}

	required := rdl.GetRequiredAttributes()
	if len(required) != 2 {
		t.Errorf("expected 2 required attributes, got %d", len(required))
	}

	// Check that both required attrs are in the list
	found1, found2 := false, false
	for _, name := range required {
		if name == "required1" {
			found1 = true
		}
		if name == "required2" {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Errorf("missing required attributes: found1=%v, found2=%v", found1, found2)
	}
}

func TestGetDefaultAttributes(t *testing.T) {
	rdl := &ServiceRDL{
		Attrs: ServiceRDLAttributes{
			"with_default":    {Type: "float", Default: 0.5},
			"without_default": {Type: "string"},
		},
	}

	defaults := rdl.GetDefaultAttributes()
	if len(defaults) != 1 {
		t.Errorf("expected 1 default attribute, got %d", len(defaults))
	}
	if defaults["with_default"] != 0.5 {
		t.Errorf("expected with_default=0.5, got %v", defaults["with_default"])
	}
}
