package compose

import (
	"fmt"
	"net"
	"regexp"
	"sort"
	"strings"

	"github.com/gorai/gorai/pkg/config"
	"gopkg.in/yaml.v3"
)

// shellSafePattern matches strings that contain only safe characters and need no quoting.
var shellSafePattern = regexp.MustCompile(`^[a-zA-Z0-9_./:@=,\-]+$`)

// shellQuote returns a shell-safe representation of a string value.
// Values that contain only safe characters are returned as-is.
// All other values are single-quoted with internal single quotes escaped.
func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	if shellSafePattern.MatchString(value) {
		return value
	}
	// Single-quote the value, escaping any internal single quotes
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

// Compiler transforms an RDL configuration into a ProcessCompose structure.
type Compiler struct {
	config     *config.RDL
	configPath string
}

// New creates a new Compiler for the given RDL configuration and config file path.
func New(cfg *config.RDL, configPath string) *Compiler {
	return &Compiler{
		config:     cfg,
		configPath: configPath,
	}
}

// Compile transforms the RDL configuration into a ProcessCompose structure.
func (c *Compiler) Compile() (*ProcessCompose, error) {
	pc := &ProcessCompose{
		Version:         "0.5",
		OrderedShutdown: true,
		Processes:       make(map[string]*Process),
	}

	c.addGlobalEnvironment(pc)
	c.addLogConfiguration(pc)

	externalNATS := c.shouldEmitExternalNATS()
	if externalNATS {
		c.addNATSServerProcess(pc)
	}

	c.addGoraiProcess(pc, externalNATS)
	c.addDeviceResetProcesses(pc, externalNATS)
	c.addServiceProcesses(pc, externalNATS)

	if c.config.IsMetricsEnabled() {
		c.addVictoriaMetricsProcess(pc)
	}

	if c.config.IsLoggingEnabled() {
		c.addVictoriaLogsProcess(pc)
	}

	return pc, nil
}

// CompileToYAML compiles the RDL and returns the result as YAML bytes.
func (c *Compiler) CompileToYAML() ([]byte, error) {
	pc, err := c.Compile()
	if err != nil {
		return nil, err
	}
	return marshalDeterministic(pc)
}

// marshalDeterministic produces YAML with processes in a deterministic order.
// yaml.v3 preserves insertion order of maps, so we build an ordered yaml.Node tree.
func marshalDeterministic(pc *ProcessCompose) ([]byte, error) {
	// Marshal then re-parse so we get a node tree we can reorder.
	raw, err := yaml.Marshal(pc)
	if err != nil {
		return nil, fmt.Errorf("marshal process-compose: %w", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal for reorder: %w", err)
	}

	// The document node wraps a mapping node.
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		sortProcessesNode(doc.Content[0])
	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return nil, fmt.Errorf("marshal sorted: %w", err)
	}
	return out, nil
}

// sortProcessesNode finds the "processes" key in a mapping node and sorts its children.
func sortProcessesNode(mapping *yaml.Node) {
	if mapping.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		keyNode := mapping.Content[i]
		valueNode := mapping.Content[i+1]
		if keyNode.Value == "processes" && valueNode.Kind == yaml.MappingNode {
			sortMappingNode(valueNode)
			return
		}
	}
}

// sortMappingNode sorts a YAML mapping node's key-value pairs by key.
func sortMappingNode(node *yaml.Node) {
	if node.Kind != yaml.MappingNode || len(node.Content) < 4 {
		return
	}
	type pair struct {
		key   *yaml.Node
		value *yaml.Node
	}
	pairs := make([]pair, 0, len(node.Content)/2)
	for i := 0; i < len(node.Content)-1; i += 2 {
		pairs = append(pairs, pair{key: node.Content[i], value: node.Content[i+1]})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].key.Value < pairs[j].key.Value
	})
	for i, p := range pairs {
		node.Content[i*2] = p.key
		node.Content[i*2+1] = p.value
	}
}

func (c *Compiler) addGlobalEnvironment(pc *ProcessCompose) {
	natsURL := c.effectiveNATSURL()
	pc.Environment = []string{
		fmt.Sprintf("GORAI_ROBOT_NAME=%s", c.config.Robot.Name),
		fmt.Sprintf("GORAI_NAMESPACE=%s", c.config.GetEffectiveNamespace()),
		fmt.Sprintf("NATS_URL=%s", natsURL),
	}

	if c.config.IsMetricsEnabled() {
		pc.Environment = append(pc.Environment,
			fmt.Sprintf("VICTORIA_METRICS_URL=http://%s", c.config.Metrics.GetListen()))
	}

	if c.config.IsLoggingEnabled() {
		pc.Environment = append(pc.Environment,
			fmt.Sprintf("VICTORIA_LOGS_URL=http://%s", c.config.Logging.GetListen()))
	}
}

func (c *Compiler) addLogConfiguration(pc *ProcessCompose) {
	pc.LogConfiguration = &LogConfiguration{
		FieldsOrder: []string{"time", "level", "message"},
		DisableJSON: false,
		NoMetadata:  false,
	}
}

func (c *Compiler) addGoraiProcess(pc *ProcessCompose, externalNATS bool) {
	process := &Process{
		Command:   fmt.Sprintf("gorai run %s", c.configPath),
		Namespace: "core",
		ReadinessProbe: &Probe{
			HTTPGet: &HTTPProbe{
				Host: "127.0.0.1",
				Port: 4180,
				Path: "/healthz",
			},
			InitialDelaySeconds: 1,
			PeriodSeconds:       5,
			FailureThreshold:    3,
		},
		Availability: &Availability{
			Restart: "on_failure",
		},
		Shutdown: &Shutdown{
			Signal:         15,
			TimeoutSeconds: 30,
		},
		LogLocation: "./logs/gorai.log",
		LogRotation: defaultLogRotation(),
	}

	if externalNATS {
		process.DependsOn = map[string]*DependsOn{
			"nats-server": {Condition: "process_healthy"},
		}
	}

	pc.Processes["gorai"] = process
}

func (c *Compiler) addNATSServerProcess(pc *ProcessCompose) {
	natsURL := c.effectiveNATSURL()
	command := "nats-server"
	if c.config.NATS != nil && c.config.NATS.JetStream {
		command += " --jetstream --store_dir ./data/jetstream/"
	}

	process := &Process{
		Command:   command,
		Namespace: "infra",
		ReadinessProbe: &Probe{
			Exec: &ExecProbe{
				Command: fmt.Sprintf("nats-server --signal check=%s", natsURL),
			},
			InitialDelaySeconds: 1,
			PeriodSeconds:       2,
			FailureThreshold:    5,
		},
		Availability: &Availability{
			Restart: "always",
		},
		Shutdown: &Shutdown{
			Signal:         15,
			TimeoutSeconds: 10,
		},
		LogLocation: "./logs/nats-server.log",
		LogRotation: defaultLogRotation(),
	}

	pc.Processes["nats-server"] = process
}

func (c *Compiler) addDeviceResetProcesses(pc *ProcessCompose, externalNATS bool) {
	for _, device := range c.config.Devices {
		if !device.ResetOnStartup {
			continue
		}

		processName := fmt.Sprintf("device-reset-%s", device.ID)
		depTarget := "gorai"
		if externalNATS {
			depTarget = "nats-server"
		}

		process := &Process{
			Command:   fmt.Sprintf("gorai device reset --nats-prefix %s --device-id %s", shellQuote(device.NATSPrefix), shellQuote(device.ID)),
			Namespace: "core",
			DependsOn: map[string]*DependsOn{
				depTarget: {Condition: "process_healthy"},
			},
			Availability: &Availability{
				Restart: "no",
			},
		}

		pc.Processes[processName] = process
	}
}

func (c *Compiler) addServiceProcesses(pc *ProcessCompose, externalNATS bool) {
	componentNames := c.buildComponentNameSet()
	externalServiceNames := c.buildExternalServiceNameSet()
	deviceResetProcesses := c.buildDeviceResetMap()

	for i := range c.config.Services {
		svc := &c.config.Services[i]
		if !svc.IsExternal() {
			continue
		}

		if svc.External.IsContainer() {
			c.addContainerServiceProcess(pc, svc, externalNATS, componentNames, externalServiceNames, deviceResetProcesses)
		} else if svc.External.Command != "" {
			c.addNativeBinaryServiceProcess(pc, svc, externalNATS, componentNames, externalServiceNames, deviceResetProcesses)
		}
	}
}

func (c *Compiler) addNativeBinaryServiceProcess(
	pc *ProcessCompose,
	svc *config.ServiceConfig,
	externalNATS bool,
	componentNames map[string]bool,
	externalServiceNames map[string]bool,
	deviceResetProcesses map[string]string,
) {
	command := shellQuote(svc.External.Command)
	for _, arg := range svc.External.Args {
		command += " " + shellQuote(arg)
	}

	envMap := config.GetResolvedEnvironment(c.config, svc)
	envList := mapToSortedEnvList(envMap)

	process := &Process{
		Command:     command,
		Namespace:   "services",
		Environment: envList,
		DependsOn:   c.buildServiceDependencies(svc, externalNATS, componentNames, externalServiceNames, deviceResetProcesses),
		Availability: &Availability{
			Restart:        mapRestartPolicy(svc.External.Restart),
			BackoffSeconds: 2,
		},
		Shutdown: &Shutdown{
			Signal:         15,
			TimeoutSeconds: 10,
		},
		LogLocation: fmt.Sprintf("./logs/%s.log", svc.Name),
		LogRotation: defaultLogRotation(),
	}

	pc.Processes[svc.Name] = process
}

func (c *Compiler) addContainerServiceProcess(
	pc *ProcessCompose,
	svc *config.ServiceConfig,
	externalNATS bool,
	componentNames map[string]bool,
	externalServiceNames map[string]bool,
	deviceResetProcesses map[string]string,
) {
	container := svc.External.Container

	// Build the podman run command with shell-safe quoting
	parts := []string{"podman run --rm --name " + shellQuote(svc.Name)}

	// Network
	network := "host"
	if container.Network != "" {
		network = container.Network
	}
	parts = append(parts, "--network "+shellQuote(network))

	// Devices
	for _, device := range container.Devices {
		parts = append(parts, "--device "+shellQuote(device))
	}

	// Volumes
	for _, volume := range container.Volumes {
		parts = append(parts, "-v "+shellQuote(volume))
	}

	// Privileged
	if container.Privileged {
		parts = append(parts, "--privileged")
	}

	// Environment: global + service-specific
	allEnv := c.buildContainerEnvironment(svc)
	envKeys := sortedKeys(allEnv)
	for _, key := range envKeys {
		parts = append(parts, fmt.Sprintf("-e %s=%s", shellQuote(key), shellQuote(allEnv[key])))
	}

	// Image
	parts = append(parts, shellQuote(container.Image))

	command := strings.Join(parts, " ")

	process := &Process{
		Command:   command,
		Namespace: "services",
		DependsOn: c.buildServiceDependencies(svc, externalNATS, componentNames, externalServiceNames, deviceResetProcesses),
		Availability: &Availability{
			Restart:        mapRestartPolicy(svc.External.Restart),
			BackoffSeconds: 2,
		},
		Shutdown: &Shutdown{
			Command:        fmt.Sprintf("podman stop -t 10 %s", shellQuote(svc.Name)),
			TimeoutSeconds: 15,
		},
		LogLocation: fmt.Sprintf("./logs/%s.log", svc.Name),
		LogRotation: defaultLogRotation(),
	}

	pc.Processes[svc.Name] = process
}

func (c *Compiler) buildContainerEnvironment(svc *config.ServiceConfig) map[string]string {
	env := make(map[string]string)

	// Global environment
	env["GORAI_ROBOT_NAME"] = c.config.Robot.Name
	env["GORAI_NAMESPACE"] = c.config.GetEffectiveNamespace()
	env["NATS_URL"] = c.effectiveNATSURL()

	if c.config.IsMetricsEnabled() {
		env["VICTORIA_METRICS_URL"] = fmt.Sprintf("http://%s", c.config.Metrics.GetListen())
	}
	if c.config.IsLoggingEnabled() {
		env["VICTORIA_LOGS_URL"] = fmt.Sprintf("http://%s", c.config.Logging.GetListen())
	}

	// Service-specific from GetResolvedEnvironment
	resolved := config.GetResolvedEnvironment(c.config, svc)
	for key, value := range resolved {
		env[key] = value
	}

	return env
}

func (c *Compiler) buildServiceDependencies(
	svc *config.ServiceConfig,
	externalNATS bool,
	componentNames map[string]bool,
	externalServiceNames map[string]bool,
	deviceResetProcesses map[string]string,
) map[string]*DependsOn {
	deps := make(map[string]*DependsOn)

	for _, depName := range svc.DependsOn {
		if componentNames[depName] {
			// Components run inside gorai controller
			deps["gorai"] = &DependsOn{Condition: "process_healthy"}
		} else if externalServiceNames[depName] {
			// External service dependency
			deps[depName] = &DependsOn{Condition: "process_started"}
		}
	}

	// Check if service references a device (via attributes.device_id)
	if deviceID := c.getServiceDeviceID(svc); deviceID != "" {
		if resetProcess, exists := deviceResetProcesses[deviceID]; exists {
			deps[resetProcess] = &DependsOn{Condition: "process_completed"}
		}
	}

	if len(deps) == 0 {
		return nil
	}
	return deps
}

func (c *Compiler) addVictoriaMetricsProcess(pc *ProcessCompose) {
	metrics := c.config.Metrics
	listen := metrics.GetListen()
	host, portStr := splitHostPort(listen)
	port := parsePort(portStr)

	command := fmt.Sprintf("victoria-metrics -retentionPeriod=%s -httpListenAddr=%s -storageDataPath=./data/victoria-metrics/",
		metrics.GetRetention(), listen)

	process := &Process{
		Command:   command,
		Namespace: "infra",
		ReadinessProbe: &Probe{
			HTTPGet: &HTTPProbe{
				Host: host,
				Port: port,
				Path: "/health",
			},
			InitialDelaySeconds: 2,
			PeriodSeconds:       5,
		},
		Availability: &Availability{
			Restart: "always",
		},
		Shutdown: &Shutdown{
			Signal:         15,
			TimeoutSeconds: 10,
		},
		LogLocation: "./logs/victoria-metrics.log",
		LogRotation: defaultLogRotation(),
	}

	pc.Processes["victoria-metrics"] = process
}

func (c *Compiler) addVictoriaLogsProcess(pc *ProcessCompose) {
	logging := c.config.Logging
	listen := logging.GetListen()
	host, portStr := splitHostPort(listen)
	port := parsePort(portStr)

	command := fmt.Sprintf("victoria-logs -retentionPeriod=%s -httpListenAddr=%s -storageDataPath=./data/victoria-logs/",
		logging.GetRetention(), listen)

	process := &Process{
		Command:   command,
		Namespace: "infra",
		ReadinessProbe: &Probe{
			HTTPGet: &HTTPProbe{
				Host: host,
				Port: port,
				Path: "/health",
			},
			InitialDelaySeconds: 2,
			PeriodSeconds:       5,
		},
		Availability: &Availability{
			Restart: "always",
		},
		Shutdown: &Shutdown{
			Signal:         15,
			TimeoutSeconds: 10,
		},
		LogLocation: "./logs/victoria-logs.log",
		LogRotation: defaultLogRotation(),
	}

	pc.Processes["victoria-logs"] = process
}

// shouldEmitExternalNATS returns true when a separate nats-server process should be emitted.
func (c *Compiler) shouldEmitExternalNATS() bool {
	if c.config.NATS == nil {
		return false
	}
	return c.config.NATS.External && c.config.NATS.IsLocalURL()
}

func (c *Compiler) effectiveNATSURL() string {
	if c.config.NATS == nil || c.config.NATS.URL == "" {
		return "nats://localhost:4222"
	}
	return c.config.NATS.URL
}

func (c *Compiler) buildComponentNameSet() map[string]bool {
	names := make(map[string]bool, len(c.config.Components))
	for _, comp := range c.config.Components {
		names[comp.Name] = true
	}
	return names
}

func (c *Compiler) buildExternalServiceNameSet() map[string]bool {
	names := make(map[string]bool)
	for _, svc := range c.config.Services {
		if svc.IsExternal() {
			names[svc.Name] = true
		}
	}
	return names
}

// buildDeviceResetMap returns device ID -> reset process name for devices with reset_on_startup.
func (c *Compiler) buildDeviceResetMap() map[string]string {
	result := make(map[string]string)
	for _, device := range c.config.Devices {
		if device.ResetOnStartup {
			result[device.ID] = fmt.Sprintf("device-reset-%s", device.ID)
		}
	}
	return result
}

// getServiceDeviceID returns the device_id from a service's attributes, if present.
func (c *Compiler) getServiceDeviceID(svc *config.ServiceConfig) string {
	if svc.Attributes == nil {
		return ""
	}
	if deviceID, ok := svc.Attributes["device_id"]; ok {
		if id, ok := deviceID.(string); ok {
			return id
		}
	}
	return ""
}

// mapRestartPolicy converts an RDL restart value to the process-compose equivalent.
func mapRestartPolicy(rdlRestart string) string {
	switch rdlRestart {
	case "always":
		return "always"
	case "on-failure":
		return "on_failure"
	case "never":
		return "no"
	default:
		return "on_failure"
	}
}

func defaultLogRotation() *LogRotation {
	return &LogRotation{
		MaxSizeMB:  10,
		MaxBackups: 3,
		MaxAgeDays: 7,
		Compress:   true,
	}
}

func mapToSortedEnvList(env map[string]string) []string {
	keys := sortedKeys(env)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, fmt.Sprintf("%s=%s", key, env[key]))
	}
	return result
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func splitHostPort(address string) (string, string) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return address, ""
	}
	return host, port
}

func parsePort(portStr string) int {
	port := 0
	for _, ch := range portStr {
		if ch >= '0' && ch <= '9' {
			port = port*10 + int(ch-'0')
		}
	}
	return port
}

