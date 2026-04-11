package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const minimalRDL = `{
	"version": "2",
	"robot": {
		"name": "test-robot"
	},
	"components": [],
	"services": []
}`

func TestCompileToStdout(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "robot.json")
	if err := os.WriteFile(configPath, []byte(minimalRDL), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	os.Stdout = w

	// Save and restore os.Args
	oldArgs := os.Args
	os.Args = []string{"gorai", "compile", configPath, "--stdout"}
	defer func() {
		os.Args = oldArgs
		os.Stdout = oldStdout
	}()

	compileErr := cmdCompile()
	w.Close()

	buf := make([]byte, 64*1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])
	r.Close()

	os.Stdout = oldStdout

	if compileErr != nil {
		t.Fatalf("cmdCompile returned error: %v", compileErr)
	}

	if !strings.Contains(output, "version:") {
		t.Errorf("expected YAML output with 'version:', got:\n%s", output)
	}
	if !strings.Contains(output, "gorai") {
		t.Errorf("expected YAML output to contain 'gorai' process, got:\n%s", output)
	}
	if !strings.Contains(output, "GORAI_ROBOT_NAME=test-robot") {
		t.Errorf("expected YAML output to contain robot name environment variable, got:\n%s", output)
	}
}

func TestCompileToFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "robot.json")
	if err := os.WriteFile(configPath, []byte(minimalRDL), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	outputPath := filepath.Join(tmpDir, "output.yaml")

	oldArgs := os.Args
	os.Args = []string{"gorai", "compile", configPath, "-o", outputPath}
	defer func() { os.Args = oldArgs }()

	if err := cmdCompile(); err != nil {
		t.Fatalf("cmdCompile returned error: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	output := string(data)
	if !strings.Contains(output, "version:") {
		t.Errorf("expected YAML output with 'version:', got:\n%s", output)
	}
	if !strings.Contains(output, "gorai") {
		t.Errorf("expected YAML output to contain 'gorai' process, got:\n%s", output)
	}
}

func TestCompileInvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bad.json")
	if err := os.WriteFile(configPath, []byte(`{invalid json`), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	oldArgs := os.Args
	os.Args = []string{"gorai", "compile", configPath, "--stdout"}
	defer func() { os.Args = oldArgs }()

	err := cmdCompile()
	if err == nil {
		t.Fatal("expected error for invalid config, got nil")
	}
	if !strings.Contains(err.Error(), "failed to load config") {
		t.Errorf("expected 'failed to load config' error, got: %v", err)
	}
}

func TestCompileNoConfigFile(t *testing.T) {
	oldArgs := os.Args
	// Use a working dir with no config files
	oldWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	os.Args = []string{"gorai", "compile"}
	defer func() {
		os.Args = oldArgs
		os.Chdir(oldWd)
	}()

	err := cmdCompile()
	if err == nil {
		t.Fatal("expected error when no config file found, got nil")
	}
	if !strings.Contains(err.Error(), "no config file specified") {
		t.Errorf("expected 'no config file specified' error, got: %v", err)
	}
}
