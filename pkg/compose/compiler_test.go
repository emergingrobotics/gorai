package compose

import (
	"strings"
	"testing"

	"github.com/gorai/gorai/pkg/config"
	"gopkg.in/yaml.v3"
)

func minimalRDL() *config.RDL {
	cfg := &config.RDL{
		Version: "2",
		Robot: config.RobotConfig{
			Name: "testbot",
		},
		NATS: &config.NATSConfig{
			URL: "nats://localhost:4222",
		},
		Log: &config.LogConfig{Level: "info"},
	}
	return cfg
}

func TestCompileMinimalRDL(t *testing.T) {
	cfg := minimalRDL()
	compiler := New(cfg, "robot.json")
	pc, err := compiler.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pc.Version != "0.5" {
		t.Errorf("expected version 0.5, got %s", pc.Version)
	}
	if !pc.OrderedShutdown {
		t.Error("expected ordered_shutdown to be true")
	}

	// Single gorai process, no nats-server
	if len(pc.Processes) != 1 {
		t.Errorf("expected 1 process, got %d", len(pc.Processes))
	}
	gorai, exists := pc.Processes["gorai"]
	if !exists {
		t.Fatal("expected gorai process")
	}
	if gorai.Command != "gorai run robot.json" {
		t.Errorf("unexpected command: %s", gorai.Command)
	}
	if gorai.Namespace != "core" {
		t.Errorf("expected namespace core, got %s", gorai.Namespace)
	}
	if gorai.DependsOn != nil {
		t.Error("expected no depends_on for embedded NATS")
	}
	if _, exists := pc.Processes["nats-server"]; exists {
		t.Error("nats-server should not be emitted for embedded NATS")
	}
}

func TestCompileExternalNATS(t *testing.T) {
	cfg := minimalRDL()
	cfg.NATS.External = true

	compiler := New(cfg, "robot.json")
	pc, err := compiler.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nats, exists := pc.Processes["nats-server"]
	if !exists {
		t.Fatal("expected nats-server process")
	}
	if nats.Namespace != "infra" {
		t.Errorf("expected namespace infra, got %s", nats.Namespace)
	}
	if nats.Availability == nil || nats.Availability.Restart != "always" {
		t.Error("expected restart always for nats-server")
	}
	if nats.ReadinessProbe == nil || nats.ReadinessProbe.Exec == nil {
		t.Fatal("expected exec readiness probe for nats-server")
	}

	gorai := pc.Processes["gorai"]
	if gorai.DependsOn == nil {
		t.Fatal("expected gorai to depend on nats-server")
	}
	dep, exists := gorai.DependsOn["nats-server"]
	if !exists || dep.Condition != "process_healthy" {
		t.Error("expected gorai depends_on nats-server with condition process_healthy")
	}
}

func TestCompileNativeBinaryService(t *testing.T) {
	cfg := minimalRDL()
	cfg.Services = []config.ServiceConfig{
		{
			Name:  "navigator",
			Type:  "navigation",
			Model: "waypoint",
			External: &config.ExternalConfig{
				Enabled: true,
				Command: "/usr/local/bin/gorai-nav-waypoint",
				Args:    []string{"--flag", "value"},
				Restart: "on-failure",
				Env:     map[string]string{"CUSTOM_VAR": "custom_value"},
			},
		},
	}

	compiler := New(cfg, "robot.json")
	pc, err := compiler.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nav, exists := pc.Processes["navigator"]
	if !exists {
		t.Fatal("expected navigator process")
	}
	if nav.Command != "/usr/local/bin/gorai-nav-waypoint --flag value" {
		t.Errorf("unexpected command: %s", nav.Command)
	}
	if nav.Namespace != "services" {
		t.Errorf("expected namespace services, got %s", nav.Namespace)
	}

	// Check environment variables
	envMap := envListToMap(nav.Environment)
	if envMap["GORAI_SERVICE_NAME"] != "navigator" {
		t.Errorf("expected GORAI_SERVICE_NAME=navigator, got %s", envMap["GORAI_SERVICE_NAME"])
	}
	if envMap["CUSTOM_VAR"] != "custom_value" {
		t.Errorf("expected CUSTOM_VAR=custom_value, got %s", envMap["CUSTOM_VAR"])
	}

	// Check availability
	if nav.Availability == nil || nav.Availability.Restart != "on_failure" {
		t.Error("expected restart on_failure")
	}

	// Check shutdown
	if nav.Shutdown == nil || nav.Shutdown.Signal != 15 || nav.Shutdown.TimeoutSeconds != 10 {
		t.Error("expected shutdown signal 15 timeout 10")
	}

	// Check log rotation
	if nav.LogLocation != "./logs/navigator.log" {
		t.Errorf("expected log location ./logs/navigator.log, got %s", nav.LogLocation)
	}
}

func TestCompileContainerService(t *testing.T) {
	cfg := minimalRDL()
	cfg.Services = []config.ServiceConfig{
		{
			Name:  "detector",
			Type:  "vision",
			Model: "yolo",
			External: &config.ExternalConfig{
				Enabled: true,
				Restart: "on-failure",
				Container: &config.ContainerServiceConfig{
					Image:   "localhost/gorai-vision-yolo:latest",
					Devices: []string{"/dev/video0"},
					Volumes: []string{"./models:/models:ro"},
				},
			},
		},
	}

	compiler := New(cfg, "robot.json")
	pc, err := compiler.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	detector, exists := pc.Processes["detector"]
	if !exists {
		t.Fatal("expected detector process")
	}

	if !strings.Contains(detector.Command, "podman run --rm --name detector") {
		t.Errorf("expected podman run command, got: %s", detector.Command)
	}
	if !strings.Contains(detector.Command, "--network host") {
		t.Errorf("expected --network host in command: %s", detector.Command)
	}
	if !strings.Contains(detector.Command, "--device /dev/video0") {
		t.Errorf("expected --device /dev/video0 in command: %s", detector.Command)
	}
	if !strings.Contains(detector.Command, "-v ./models:/models:ro") {
		t.Errorf("expected -v volume in command: %s", detector.Command)
	}
	if !strings.Contains(detector.Command, "-e GORAI_ROBOT_NAME=testbot") {
		t.Errorf("expected -e GORAI_ROBOT_NAME in command: %s", detector.Command)
	}
	if !strings.Contains(detector.Command, "localhost/gorai-vision-yolo:latest") {
		t.Errorf("expected image at end of command: %s", detector.Command)
	}

	// Shutdown must use podman stop
	if detector.Shutdown == nil || detector.Shutdown.Command != "podman stop -t 10 detector" {
		t.Error("expected shutdown command: podman stop -t 10 detector")
	}
	if detector.Shutdown.TimeoutSeconds != 15 {
		t.Errorf("expected shutdown timeout 15, got %d", detector.Shutdown.TimeoutSeconds)
	}
	if detector.Namespace != "services" {
		t.Errorf("expected namespace services, got %s", detector.Namespace)
	}
}

func TestCompileDeviceReset(t *testing.T) {
	cfg := minimalRDL()
	cfg.Devices = []config.DeviceConfig{
		{ID: "pico-1", NATSPrefix: "gsp", ResetOnStartup: true},
	}

	compiler := New(cfg, "robot.json")
	pc, err := compiler.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reset, exists := pc.Processes["device-reset-pico-1"]
	if !exists {
		t.Fatal("expected device-reset-pico-1 process")
	}
	if reset.Command != "gorai device reset --nats-prefix gsp --device-id pico-1" {
		t.Errorf("unexpected command: %s", reset.Command)
	}
	if reset.Namespace != "core" {
		t.Errorf("expected namespace core, got %s", reset.Namespace)
	}
	if reset.Availability == nil || reset.Availability.Restart != "no" {
		t.Error("expected restart no for transient process")
	}

	// Should depend on gorai (embedded NATS)
	dep, exists := reset.DependsOn["gorai"]
	if !exists || dep.Condition != "process_healthy" {
		t.Error("expected depends_on gorai with process_healthy")
	}

	// Transient process must not have log_location
	if reset.LogLocation != "" {
		t.Errorf("expected no log_location for transient process, got %s", reset.LogLocation)
	}
}

func TestCompileDeviceDependency(t *testing.T) {
	cfg := minimalRDL()
	cfg.Devices = []config.DeviceConfig{
		{ID: "pico-1", NATSPrefix: "gsp", ResetOnStartup: true},
	}
	cfg.Components = []config.ComponentConfig{
		{
			Name:  "left_motor",
			Type:  "motor",
			Model: "remote",
			Attributes: map[string]any{
				"device_id": "pico-1",
			},
		},
	}
	cfg.Services = []config.ServiceConfig{
		{
			Name:  "navigator",
			Type:  "navigation",
			Model: "waypoint",
			External: &config.ExternalConfig{
				Enabled: true,
				Command: "/usr/local/bin/gorai-nav",
			},
			Attributes: map[string]any{
				"device_id": "pico-1",
			},
		},
	}

	compiler := New(cfg, "robot.json")
	pc, err := compiler.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nav := pc.Processes["navigator"]
	if nav == nil {
		t.Fatal("expected navigator process")
	}

	resetDep, exists := nav.DependsOn["device-reset-pico-1"]
	if !exists {
		t.Fatal("expected navigator to depend on device-reset-pico-1")
	}
	if resetDep.Condition != "process_completed" {
		t.Errorf("expected condition process_completed, got %s", resetDep.Condition)
	}
}

func TestCompileMetricsEnabled(t *testing.T) {
	cfg := minimalRDL()
	cfg.Metrics = &config.MetricsConfig{
		Enabled:   true,
		Retention: "14d",
	}

	compiler := New(cfg, "robot.json")
	pc, err := compiler.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	vm, exists := pc.Processes["victoria-metrics"]
	if !exists {
		t.Fatal("expected victoria-metrics process")
	}
	if vm.Namespace != "infra" {
		t.Errorf("expected namespace infra, got %s", vm.Namespace)
	}
	if !strings.Contains(vm.Command, "-retentionPeriod=14d") {
		t.Errorf("expected retention flag in command: %s", vm.Command)
	}
	if vm.Availability == nil || vm.Availability.Restart != "always" {
		t.Error("expected restart always")
	}
	if vm.ReadinessProbe == nil || vm.ReadinessProbe.HTTPGet == nil {
		t.Fatal("expected HTTP readiness probe")
	}
	if vm.ReadinessProbe.HTTPGet.Path != "/health" {
		t.Errorf("expected probe path /health, got %s", vm.ReadinessProbe.HTTPGet.Path)
	}

	// Check global env var
	found := false
	for _, envVar := range pc.Environment {
		if strings.HasPrefix(envVar, "VICTORIA_METRICS_URL=") {
			found = true
			if envVar != "VICTORIA_METRICS_URL=http://127.0.0.1:8428" {
				t.Errorf("unexpected VICTORIA_METRICS_URL: %s", envVar)
			}
		}
	}
	if !found {
		t.Error("expected VICTORIA_METRICS_URL in global environment")
	}
}

func TestCompileLoggingEnabled(t *testing.T) {
	cfg := minimalRDL()
	cfg.Logging = &config.LoggingConfig{
		Enabled:   true,
		Retention: "7d",
	}

	compiler := New(cfg, "robot.json")
	pc, err := compiler.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	vl, exists := pc.Processes["victoria-logs"]
	if !exists {
		t.Fatal("expected victoria-logs process")
	}
	if vl.Namespace != "infra" {
		t.Errorf("expected namespace infra, got %s", vl.Namespace)
	}
	if !strings.Contains(vl.Command, "-retentionPeriod=7d") {
		t.Errorf("expected retention flag in command: %s", vl.Command)
	}
	if vl.Availability == nil || vl.Availability.Restart != "always" {
		t.Error("expected restart always")
	}

	found := false
	for _, envVar := range pc.Environment {
		if strings.HasPrefix(envVar, "VICTORIA_LOGS_URL=") {
			found = true
			if envVar != "VICTORIA_LOGS_URL=http://127.0.0.1:9428" {
				t.Errorf("unexpected VICTORIA_LOGS_URL: %s", envVar)
			}
		}
	}
	if !found {
		t.Error("expected VICTORIA_LOGS_URL in global environment")
	}
}

func TestCompileDependencyTranslation(t *testing.T) {
	cfg := minimalRDL()
	cfg.Components = []config.ComponentConfig{
		{Name: "left_motor", Type: "motor", Model: "remote"},
		{Name: "right_motor", Type: "motor", Model: "remote"},
	}
	cfg.Services = []config.ServiceConfig{
		{
			Name:  "navigator",
			Type:  "navigation",
			Model: "waypoint",
			External: &config.ExternalConfig{
				Enabled: true,
				Command: "/usr/local/bin/gorai-nav",
			},
			DependsOn: []string{"left_motor", "right_motor"},
		},
	}

	compiler := New(cfg, "robot.json")
	pc, err := compiler.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nav := pc.Processes["navigator"]
	if nav == nil {
		t.Fatal("expected navigator process")
	}

	// Both component dependencies should translate to a single gorai dependency
	goraiDep, exists := nav.DependsOn["gorai"]
	if !exists {
		t.Fatal("expected depends_on gorai")
	}
	if goraiDep.Condition != "process_healthy" {
		t.Errorf("expected condition process_healthy, got %s", goraiDep.Condition)
	}

	// Should not have individual component dependencies
	if _, exists := nav.DependsOn["left_motor"]; exists {
		t.Error("should not have direct dependency on component left_motor")
	}
}

func TestCompileDependencyOnExternalService(t *testing.T) {
	cfg := minimalRDL()
	cfg.Services = []config.ServiceConfig{
		{
			Name:  "data-processor",
			Type:  "formatter",
			Model: "json",
			External: &config.ExternalConfig{
				Enabled: true,
				Command: "/usr/local/bin/data-proc",
			},
		},
		{
			Name:  "navigator",
			Type:  "navigation",
			Model: "waypoint",
			External: &config.ExternalConfig{
				Enabled: true,
				Command: "/usr/local/bin/gorai-nav",
			},
			DependsOn: []string{"data-processor"},
		},
	}

	compiler := New(cfg, "robot.json")
	pc, err := compiler.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nav := pc.Processes["navigator"]
	if nav == nil {
		t.Fatal("expected navigator process")
	}

	dep, exists := nav.DependsOn["data-processor"]
	if !exists {
		t.Fatal("expected depends_on data-processor")
	}
	if dep.Condition != "process_started" {
		t.Errorf("expected condition process_started, got %s", dep.Condition)
	}
}

func TestMapRestartPolicy(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"always", "always"},
		{"on-failure", "on_failure"},
		{"never", "no"},
		{"", "on_failure"},
		{"unknown", "on_failure"},
	}

	for _, tt := range tests {
		result := mapRestartPolicy(tt.input)
		if result != tt.expected {
			t.Errorf("mapRestartPolicy(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestCompileGlobalEnvironment(t *testing.T) {
	cfg := minimalRDL()
	cfg.Robot.Name = "scout"
	cfg.Robot.Namespace = "outdoor"
	cfg.NATS.URL = "nats://localhost:4222"

	compiler := New(cfg, "robot.json")
	pc, err := compiler.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	envMap := envListToMap(pc.Environment)
	if envMap["GORAI_ROBOT_NAME"] != "scout" {
		t.Errorf("expected GORAI_ROBOT_NAME=scout, got %s", envMap["GORAI_ROBOT_NAME"])
	}
	if envMap["GORAI_NAMESPACE"] != "outdoor" {
		t.Errorf("expected GORAI_NAMESPACE=outdoor, got %s", envMap["GORAI_NAMESPACE"])
	}
	if envMap["NATS_URL"] != "nats://localhost:4222" {
		t.Errorf("expected NATS_URL=nats://localhost:4222, got %s", envMap["NATS_URL"])
	}
}

func TestCompileLogConfiguration(t *testing.T) {
	cfg := minimalRDL()
	compiler := New(cfg, "robot.json")
	pc, err := compiler.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pc.LogConfiguration == nil {
		t.Fatal("expected log_configuration")
	}
	if len(pc.LogConfiguration.FieldsOrder) != 3 {
		t.Errorf("expected 3 fields in fields_order, got %d", len(pc.LogConfiguration.FieldsOrder))
	}
	expected := []string{"time", "level", "message"}
	for i, field := range pc.LogConfiguration.FieldsOrder {
		if field != expected[i] {
			t.Errorf("expected fields_order[%d] = %s, got %s", i, expected[i], field)
		}
	}
}

func TestCompileContainerServicePrivileged(t *testing.T) {
	cfg := minimalRDL()
	cfg.Services = []config.ServiceConfig{
		{
			Name:  "privileged-svc",
			Type:  "vision",
			Model: "custom",
			External: &config.ExternalConfig{
				Enabled: true,
				Container: &config.ContainerServiceConfig{
					Image:      "localhost/test:latest",
					Privileged: true,
				},
			},
		},
	}

	compiler := New(cfg, "robot.json")
	pc, err := compiler.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc := pc.Processes["privileged-svc"]
	if svc == nil {
		t.Fatal("expected privileged-svc process")
	}
	if !strings.Contains(svc.Command, "--privileged") {
		t.Errorf("expected --privileged in command: %s", svc.Command)
	}
}

func TestCompileToYAML(t *testing.T) {
	cfg := minimalRDL()
	compiler := New(cfg, "robot.json")
	yamlBytes, err := compiler.CompileToYAML()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it is valid YAML by parsing it back
	var parsed ProcessCompose
	if err := yaml.Unmarshal(yamlBytes, &parsed); err != nil {
		t.Fatalf("output is not valid YAML: %v", err)
	}

	if parsed.Version != "0.5" {
		t.Errorf("expected version 0.5, got %s", parsed.Version)
	}
	if !parsed.OrderedShutdown {
		t.Error("expected ordered_shutdown true")
	}
	if _, exists := parsed.Processes["gorai"]; !exists {
		t.Error("expected gorai process in parsed output")
	}
}

func TestCompileDeviceResetWithExternalNATS(t *testing.T) {
	cfg := minimalRDL()
	cfg.NATS.External = true
	cfg.Devices = []config.DeviceConfig{
		{ID: "pico-1", NATSPrefix: "gsp", ResetOnStartup: true},
	}

	compiler := New(cfg, "robot.json")
	pc, err := compiler.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reset := pc.Processes["device-reset-pico-1"]
	if reset == nil {
		t.Fatal("expected device-reset-pico-1 process")
	}

	// With external NATS, device reset should depend on nats-server
	dep, exists := reset.DependsOn["nats-server"]
	if !exists {
		t.Fatal("expected depends_on nats-server for device reset with external NATS")
	}
	if dep.Condition != "process_healthy" {
		t.Errorf("expected condition process_healthy, got %s", dep.Condition)
	}
}

func TestCompileContainerCustomNetwork(t *testing.T) {
	cfg := minimalRDL()
	cfg.Services = []config.ServiceConfig{
		{
			Name:  "custom-net-svc",
			Type:  "bridge",
			Model: "custom",
			External: &config.ExternalConfig{
				Enabled: true,
				Container: &config.ContainerServiceConfig{
					Image:   "localhost/test:latest",
					Network: "my-network",
				},
			},
		},
	}

	compiler := New(cfg, "robot.json")
	pc, err := compiler.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc := pc.Processes["custom-net-svc"]
	if svc == nil {
		t.Fatal("expected custom-net-svc process")
	}
	if !strings.Contains(svc.Command, "--network my-network") {
		t.Errorf("expected --network my-network in command: %s", svc.Command)
	}
}

func TestCompileNoDeviceResetWhenNotEnabled(t *testing.T) {
	cfg := minimalRDL()
	cfg.Devices = []config.DeviceConfig{
		{ID: "pico-1", NATSPrefix: "gsp", ResetOnStartup: false},
	}

	compiler := New(cfg, "robot.json")
	pc, err := compiler.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, exists := pc.Processes["device-reset-pico-1"]; exists {
		t.Error("should not emit device reset when reset_on_startup is false")
	}
}

func TestCompileNamespaceDefaultsToName(t *testing.T) {
	cfg := minimalRDL()
	cfg.Robot.Name = "scout"
	cfg.Robot.Namespace = ""

	compiler := New(cfg, "robot.json")
	pc, err := compiler.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	envMap := envListToMap(pc.Environment)
	if envMap["GORAI_NAMESPACE"] != "scout" {
		t.Errorf("expected GORAI_NAMESPACE=scout when namespace is empty, got %s", envMap["GORAI_NAMESPACE"])
	}
}

// envListToMap converts a list of "KEY=VALUE" strings to a map.
func envListToMap(envList []string) map[string]string {
	result := make(map[string]string, len(envList))
	for _, entry := range envList {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}
