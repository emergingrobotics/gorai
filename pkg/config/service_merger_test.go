package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestServiceMerger_LoadAndMergeServices(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "merger-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a service RDL file
	serviceRDL := `{
		"version": "1",
		"kind": "service",
		"service": {
			"type": "vision",
			"model": "yolo"
		},
		"subjects": {
			"subscribe": [
				{
					"name": "input",
					"pattern": "{namespace}.camera.{input_component}.frame"
				}
			],
			"publish": [
				{
					"name": "output",
					"pattern": "{namespace}.detection.{service}.objects"
				}
			]
		},
		"attributes": {
			"input_component": {"type": "string", "required": true},
			"threshold": {"type": "float", "default": 0.5}
		}
	}`

	rdlPath := filepath.Join(tmpDir, "detector.rdl.json")
	if err := os.WriteFile(rdlPath, []byte(serviceRDL), 0644); err != nil {
		t.Fatalf("failed to write rdl: %v", err)
	}

	// Create robot config that references the service RDL
	cfg := &RDL{
		Robot: RobotConfig{
			Name:      "test-robot",
			Namespace: "test-ns",
		},
		Services: []ServiceConfig{
			{
				Name: "person_detector",
				RDL:  "detector.rdl.json",
				Attributes: map[string]any{
					"input_component": "main_camera",
				},
			},
		},
	}

	// Create merger and run
	merger := NewServiceMerger(cfg, tmpDir)
	err = merger.LoadAndMergeServices()
	if err != nil {
		t.Fatalf("LoadAndMergeServices failed: %v", err)
	}

	// Verify service was merged correctly
	svc := &cfg.Services[0]

	// Check type and model were set from RDL
	if svc.Type != "vision" {
		t.Errorf("expected type 'vision', got %q", svc.Type)
	}
	if svc.Model != "yolo" {
		t.Errorf("expected model 'yolo', got %q", svc.Model)
	}

	// Check attributes were merged
	if svc.Attributes["threshold"] != 0.5 {
		t.Errorf("expected threshold 0.5, got %v", svc.Attributes["threshold"])
	}
}

func TestServiceMerger_MissingRequiredAttribute(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "merger-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a service RDL with required attribute
	serviceRDL := `{
		"version": "1",
		"kind": "service",
		"service": {"type": "test", "model": "v1"},
		"subjects": {"subscribe": [{"name": "t", "pattern": "p"}]},
		"attributes": {
			"must_have": {"type": "string", "required": true}
		}
	}`

	rdlPath := filepath.Join(tmpDir, "test.rdl.json")
	if err := os.WriteFile(rdlPath, []byte(serviceRDL), 0644); err != nil {
		t.Fatalf("failed to write rdl: %v", err)
	}

	cfg := &RDL{
		Robot: RobotConfig{Name: "robot"},
		Services: []ServiceConfig{
			{
				Name: "test-svc",
				RDL:  "test.rdl.json",
				// Missing required attribute
			},
		},
	}

	merger := NewServiceMerger(cfg, tmpDir)
	err = merger.LoadAndMergeServices()
	if err == nil {
		t.Error("expected error for missing required attribute")
	}
}

func TestServiceMerger_NonExistentRDL(t *testing.T) {
	cfg := &RDL{
		Robot: RobotConfig{Name: "robot"},
		Services: []ServiceConfig{
			{
				Name: "test-svc",
				RDL:  "nonexistent.rdl.json",
			},
		},
	}

	merger := NewServiceMerger(cfg, "/tmp")
	err := merger.LoadAndMergeServices()
	if err == nil {
		t.Error("expected error for non-existent RDL file")
	}
}

func TestGetResolvedEnvironment(t *testing.T) {
	cfg := &RDL{
		Robot: RobotConfig{
			Name:      "test-robot",
			Namespace: "test-ns",
		},
	}

	svc := &ServiceConfig{
		Name: "detector",
		Attributes: map[string]any{
			"input_component": "camera1",
			"threshold":       0.7,
		},
	}

	// Set resolved subjects
	svc.SetResolvedSubjects(&ResolvedSubjects{
		Subscribe: map[string]string{
			"input": "test-ns.camera.camera1.frame",
		},
		Publish: map[string]string{
			"output":    "test-ns.detection.detector.objects",
			"annotated": "test-ns.detection.detector.annotated",
		},
	})

	env := GetResolvedEnvironment(cfg, svc)

	// Check standard variables
	if env["GORAI_ROBOT_NAME"] != "test-robot" {
		t.Errorf("expected GORAI_ROBOT_NAME=test-robot, got %s", env["GORAI_ROBOT_NAME"])
	}
	if env["GORAI_NAMESPACE"] != "test-ns" {
		t.Errorf("expected GORAI_NAMESPACE=test-ns, got %s", env["GORAI_NAMESPACE"])
	}
	if env["GORAI_SERVICE_NAME"] != "detector" {
		t.Errorf("expected GORAI_SERVICE_NAME=detector, got %s", env["GORAI_SERVICE_NAME"])
	}

	// Check attributes (only string values are exported)
	if env["INPUT_COMPONENT"] != "camera1" {
		t.Errorf("expected INPUT_COMPONENT=camera1, got %s", env["INPUT_COMPONENT"])
	}

	// Check resolved subjects
	if env["INPUT_SUBJECT_INPUT"] != "test-ns.camera.camera1.frame" {
		t.Errorf("unexpected input subject: %s", env["INPUT_SUBJECT_INPUT"])
	}
	if env["OUTPUT_SUBJECT_OUTPUT"] != "test-ns.detection.detector.objects" {
		t.Errorf("unexpected output subject: %s", env["OUTPUT_SUBJECT_OUTPUT"])
	}
}

func TestLoadWithServiceRDL(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "load-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create service RDL
	serviceRDL := `{
		"version": "1",
		"kind": "service",
		"service": {"type": "test", "model": "v1"},
		"subjects": {"subscribe": [{"name": "in", "pattern": "test.in"}], "publish": []},
		"attributes": {}
	}`
	rdlPath := filepath.Join(tmpDir, "svc.rdl.json")
	if err := os.WriteFile(rdlPath, []byte(serviceRDL), 0644); err != nil {
		t.Fatalf("failed to write rdl: %v", err)
	}

	// Create robot config
	robotConfig := `{
		"version": "2.0",
		"robot": {"name": "test-robot"},
		"components": [],
		"services": [
			{
				"name": "my-service",
				"rdl": "svc.rdl.json"
			}
		]
	}`
	configPath := filepath.Join(tmpDir, "robot.json")
	if err := os.WriteFile(configPath, []byte(robotConfig), 0644); err != nil {
		t.Fatalf("failed to write robot config: %v", err)
	}

	// Load with Service RDL support
	cfg, err := LoadWithServiceRDL(configPath)
	if err != nil {
		t.Fatalf("LoadWithServiceRDL failed: %v", err)
	}

	// Verify service was populated from RDL
	if len(cfg.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(cfg.Services))
	}
	svc := cfg.Services[0]
	if svc.Type != "test" {
		t.Errorf("expected type 'test', got %q", svc.Type)
	}
	if svc.Model != "v1" {
		t.Errorf("expected model 'v1', got %q", svc.Model)
	}
}

func TestServiceConfigHasServiceRDL(t *testing.T) {
	svc := &ServiceConfig{}
	if svc.HasServiceRDL() {
		t.Error("expected HasServiceRDL=false for empty RDL")
	}

	svc.RDL = "some.rdl.json"
	if !svc.HasServiceRDL() {
		t.Error("expected HasServiceRDL=true when RDL is set")
	}
}

func TestServiceConfigRDLAccessors(t *testing.T) {
	svc := &ServiceConfig{}

	// Initially nil
	if svc.GetServiceRDL() != nil {
		t.Error("expected nil ServiceRDL initially")
	}
	if svc.GetResolvedSubjects() != nil {
		t.Error("expected nil ResolvedSubjects initially")
	}

	// Set and get
	rdl := &ServiceRDL{Version: "1"}
	svc.SetServiceRDL(rdl)
	if svc.GetServiceRDL() != rdl {
		t.Error("SetServiceRDL/GetServiceRDL mismatch")
	}

	subjects := &ResolvedSubjects{Subscribe: map[string]string{"a": "b"}}
	svc.SetResolvedSubjects(subjects)
	if svc.GetResolvedSubjects() != subjects {
		t.Error("SetResolvedSubjects/GetResolvedSubjects mismatch")
	}
}
