package config

import (
	"testing"
)

func TestTopicResolver_Resolve(t *testing.T) {
	tests := []struct {
		name    string
		vars    map[string]string
		pattern string
		want    string
		wantErr bool
	}{
		{
			name: "simple substitution",
			vars: map[string]string{
				"namespace": "myrobot",
				"service":   "detector",
			},
			pattern: "{namespace}.{service}.output",
			want:    "myrobot.detector.output",
			wantErr: false,
		},
		{
			name: "multiple occurrences",
			vars: map[string]string{
				"namespace": "test",
			},
			pattern: "{namespace}.camera.{namespace}.frame",
			want:    "test.camera.test.frame",
			wantErr: false,
		},
		{
			name:    "no variables",
			vars:    map[string]string{},
			pattern: "static.topic.name",
			want:    "static.topic.name",
			wantErr: false,
		},
		{
			name: "missing variable",
			vars: map[string]string{
				"namespace": "test",
			},
			pattern: "{namespace}.{missing}.topic",
			want:    "",
			wantErr: true,
		},
		{
			name: "input_component variable",
			vars: map[string]string{
				"namespace":       "myrobot",
				"input_component": "main_camera",
			},
			pattern: "{namespace}.camera.{input_component}.frame",
			want:    "myrobot.camera.main_camera.frame",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewTopicResolver(tt.vars)
			got, err := r.Resolve(tt.pattern)

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

			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTopicResolver_ResolveAll(t *testing.T) {
	vars := map[string]string{
		"namespace":       "myrobot",
		"service":         "detector",
		"input_component": "camera1",
	}

	topics := &ServiceRDLTopics{
		Subscribe: []ServiceRDLTopicEntry{
			{Name: "input", Pattern: "{namespace}.camera.{input_component}.frame"},
		},
		Publish: []ServiceRDLTopicEntry{
			{Name: "output", Pattern: "{namespace}.detection.{service}.objects"},
			{Name: "annotated", Pattern: "{namespace}.detection.{service}.annotated"},
		},
	}

	resolver := NewTopicResolver(vars)
	resolved, err := resolver.ResolveAll(topics)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check subscribe topics
	if len(resolved.Subscribe) != 1 {
		t.Errorf("expected 1 subscribe topic, got %d", len(resolved.Subscribe))
	}
	if resolved.Subscribe["input"] != "myrobot.camera.camera1.frame" {
		t.Errorf("unexpected input topic: %q", resolved.Subscribe["input"])
	}

	// Check publish topics
	if len(resolved.Publish) != 2 {
		t.Errorf("expected 2 publish topics, got %d", len(resolved.Publish))
	}
	if resolved.Publish["output"] != "myrobot.detection.detector.objects" {
		t.Errorf("unexpected output topic: %q", resolved.Publish["output"])
	}
	if resolved.Publish["annotated"] != "myrobot.detection.detector.annotated" {
		t.Errorf("unexpected annotated topic: %q", resolved.Publish["annotated"])
	}
}

func TestNewTopicResolverFromConfig(t *testing.T) {
	cfg := &RDL{
		Robot: RobotConfig{
			Name:      "test-robot",
			Namespace: "custom-ns",
		},
	}

	attrs := map[string]any{
		"input_component": "main_camera",
	}

	resolver := NewTopicResolverFromConfig(cfg, "detector", attrs)

	// Test that default variables are set
	resolved, err := resolver.Resolve("{namespace}.{service}.{input_component}.test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "custom-ns.detector.main_camera.test"
	if resolved != expected {
		t.Errorf("got %q, want %q", resolved, expected)
	}
}

func TestExtractVariables(t *testing.T) {
	tests := []struct {
		pattern string
		want    []string
	}{
		{
			pattern: "{namespace}.{service}.output",
			want:    []string{"namespace", "service"},
		},
		{
			pattern: "static.topic",
			want:    []string{},
		},
		{
			pattern: "{a}.{b}.{a}",
			want:    []string{"a", "b"}, // duplicates removed
		},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got := ExtractVariables(tt.pattern)
			if len(got) != len(tt.want) {
				t.Errorf("ExtractVariables() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ExtractVariables()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestAddVariable(t *testing.T) {
	resolver := NewTopicResolver(map[string]string{})

	// Initially should fail
	_, err := resolver.Resolve("{test}.topic")
	if err == nil {
		t.Error("expected error for undefined variable")
	}

	// Add variable
	resolver.AddVariable("test", "myvalue")

	// Now should succeed
	result, err := resolver.Resolve("{test}.topic")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "myvalue.topic" {
		t.Errorf("got %q, want %q", result, "myvalue.topic")
	}
}

func TestGetContext(t *testing.T) {
	original := map[string]string{
		"a": "1",
		"b": "2",
	}
	resolver := NewTopicResolver(original)

	// Get context
	ctx := resolver.GetContext()

	// Verify it's a copy
	ctx["c"] = "3"
	if len(resolver.GetContext()) != 2 {
		t.Error("GetContext should return a copy, not modify original")
	}
}
