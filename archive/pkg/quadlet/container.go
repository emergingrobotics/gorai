package quadlet

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gorai/gorai/pkg/config"
)

// generateContainer generates a .container unit file content.
func (g *Generator) generateContainer(name string, container *config.ContainerConfig) (string, error) {
	var b strings.Builder

	containerName := fmt.Sprintf("%s-%s", g.cfg.Robot.Name, name)
	networkName := fmt.Sprintf("%s-network", g.cfg.Robot.Name)

	// [Unit] section
	b.WriteString("[Unit]\n")
	b.WriteString(fmt.Sprintf("Description=Gorai container %s\n", containerName))

	// Add dependencies
	if len(container.DependsOn) > 0 {
		var requires, after []string
		for depName, dep := range container.DependsOn {
			serviceName := fmt.Sprintf("%s-%s.service", g.cfg.Robot.Name, depName)

			// Map compose conditions to systemd dependencies
			switch dep.Condition {
			case "service_healthy":
				// For healthy, we need Requires + After
				requires = append(requires, serviceName)
				after = append(after, serviceName)
			case "service_completed_successfully":
				// For completed, use Requires + After
				requires = append(requires, serviceName)
				after = append(after, serviceName)
			default: // "service_started" or empty
				requires = append(requires, serviceName)
				after = append(after, serviceName)
			}
		}

		if len(requires) > 0 {
			b.WriteString(fmt.Sprintf("Requires=%s\n", strings.Join(requires, " ")))
		}
		if len(after) > 0 {
			b.WriteString(fmt.Sprintf("After=%s\n", strings.Join(after, " ")))
		}
	}

	b.WriteString("\n")

	// [Container] section
	b.WriteString("[Container]\n")
	b.WriteString(fmt.Sprintf("ContainerName=%s\n", containerName))

	// Image or Build
	if container.Build != nil {
		// Reference a .build file if we have build config
		buildName := fmt.Sprintf("%s-%s.build", g.cfg.Robot.Name, name)
		b.WriteString(fmt.Sprintf("Image=%s\n", buildName))
	} else if container.Image != "" {
		b.WriteString(fmt.Sprintf("Image=%s\n", container.Image))
	}

	// Network - use default network unless network_mode is set
	if container.NetworkMode != "" {
		if container.NetworkMode == "host" {
			b.WriteString("Network=host\n")
		} else if container.NetworkMode == "none" {
			b.WriteString("Network=none\n")
		}
	} else {
		// Use default network
		b.WriteString(fmt.Sprintf("Network=%s.network\n", networkName))
	}

	// Additional networks
	for _, net := range container.Networks {
		netFile := fmt.Sprintf("%s-%s.network", g.cfg.Robot.Name, net)
		b.WriteString(fmt.Sprintf("Network=%s\n", netFile))
	}

	// Ports
	for _, port := range container.Ports {
		b.WriteString(fmt.Sprintf("PublishPort=%s\n", port))
	}

	// Volumes
	for _, vol := range container.Volumes {
		formattedVol := g.formatVolume(vol)
		b.WriteString(fmt.Sprintf("Volume=%s\n", formattedVol))
	}

	// Devices
	for _, dev := range container.Devices {
		formattedDev := g.formatDevice(dev)
		b.WriteString(fmt.Sprintf("AddDevice=%s\n", formattedDev))
	}

	// Group memberships for device access
	for _, group := range container.GroupAdd {
		b.WriteString(fmt.Sprintf("GroupAdd=%s\n", group))
	}

	// Environment variables
	env := g.buildEnvironment(name, container)
	for key, value := range env {
		b.WriteString(fmt.Sprintf("Environment=%s=%s\n", key, value))
	}

	// Environment files
	for _, envFile := range container.EnvFile {
		b.WriteString(fmt.Sprintf("EnvironmentFile=%s\n", envFile))
	}

	// Command and Entrypoint
	if len(container.Entrypoint) > 0 {
		b.WriteString(fmt.Sprintf("Entrypoint=%s\n", strings.Join(container.Entrypoint, " ")))
	}
	if len(container.Command) > 0 {
		b.WriteString(fmt.Sprintf("Exec=%s\n", strings.Join(container.Command, " ")))
	}

	// Security options
	if container.Privileged {
		b.WriteString("SecurityLabelDisable=true\n")
	}
	for _, opt := range container.SecurityOpt {
		if opt == "label=disable" || opt == "label:disable" {
			b.WriteString("SecurityLabelDisable=true\n")
		} else {
			b.WriteString(fmt.Sprintf("SecurityLabelType=%s\n", opt))
		}
	}

	// Capabilities
	for _, cap := range container.CapAdd {
		b.WriteString(fmt.Sprintf("AddCapability=%s\n", cap))
	}
	for _, cap := range container.CapDrop {
		b.WriteString(fmt.Sprintf("DropCapability=%s\n", cap))
	}

	// Resource limits
	if container.Resources != nil && container.Resources.Limits != nil {
		if container.Resources.Limits.Memory != "" {
			b.WriteString(fmt.Sprintf("PodmanArgs=--memory=%s\n", container.Resources.Limits.Memory))
		}
		if container.Resources.Limits.CPUs != "" {
			b.WriteString(fmt.Sprintf("PodmanArgs=--cpus=%s\n", container.Resources.Limits.CPUs))
		}
	}

	// Health check
	if container.Healthcheck != nil && len(container.Healthcheck.Test) > 0 {
		// Convert healthcheck test to command
		test := container.Healthcheck.Test
		if len(test) > 0 {
			var healthCmd string
			if test[0] == "CMD" || test[0] == "CMD-SHELL" {
				healthCmd = strings.Join(test[1:], " ")
			} else {
				healthCmd = strings.Join(test, " ")
			}
			b.WriteString(fmt.Sprintf("HealthCmd=%s\n", healthCmd))
		}

		if container.Healthcheck.Interval != "" {
			b.WriteString(fmt.Sprintf("HealthInterval=%s\n", container.Healthcheck.Interval))
		}
		if container.Healthcheck.Timeout != "" {
			b.WriteString(fmt.Sprintf("HealthTimeout=%s\n", container.Healthcheck.Timeout))
		}
		if container.Healthcheck.Retries > 0 {
			b.WriteString(fmt.Sprintf("HealthRetries=%d\n", container.Healthcheck.Retries))
		}
		if container.Healthcheck.StartPeriod != "" {
			b.WriteString(fmt.Sprintf("HealthStartPeriod=%s\n", container.Healthcheck.StartPeriod))
		}
	}

	// AutoUpdate (Quadlet-specific feature)
	autoUpdate := container.AutoUpdate
	if autoUpdate == "" {
		autoUpdate = "registry" // Default to registry for production deployments
	}
	if autoUpdate != "none" && autoUpdate != "disabled" {
		b.WriteString(fmt.Sprintf("AutoUpdate=%s\n", autoUpdate))
	}

	// Notify option (wait for health check)
	if container.Notify != "" {
		b.WriteString(fmt.Sprintf("Notify=%s\n", container.Notify))
	}

	b.WriteString("\n")

	// [Service] section
	b.WriteString("[Service]\n")

	// Restart policy
	restart := g.mapRestartPolicy(container.Restart)
	b.WriteString(fmt.Sprintf("Restart=%s\n", restart))

	// Restart delay to prevent tight loops
	if restart != "no" {
		b.WriteString("RestartSec=10\n")
	}

	// Stop timeout
	if container.StopGracePeriod != "" {
		// Convert duration string to seconds for systemd
		b.WriteString(fmt.Sprintf("TimeoutStopSec=%s\n", container.StopGracePeriod))
	}

	// Startup timeout for large images
	timeoutStart := container.TimeoutStart
	if timeoutStart == "" {
		timeoutStart = "300" // Default 5 minutes
	}
	b.WriteString(fmt.Sprintf("TimeoutStartSec=%s\n", timeoutStart))

	b.WriteString("\n")

	// [Install] section
	b.WriteString("[Install]\n")
	b.WriteString(fmt.Sprintf("WantedBy=%s\n", g.wantedBy()))

	return b.String(), nil
}

// buildEnvironment constructs the environment variables for a container.
func (g *Generator) buildEnvironment(name string, container *config.ContainerConfig) map[string]string {
	env := make(map[string]string)

	// Copy user-defined environment with interpolation
	for k, v := range container.Environment {
		env[k] = g.interpolateVariable(v)
	}

	// Add standard Gorai environment variables
	env["GORAI_ROBOT_NAME"] = g.cfg.Robot.Name

	// Add NATS URL if configured
	if g.cfg.NATS != nil && g.cfg.NATS.URL != "" {
		if _, exists := env["NATS_URL"]; !exists {
			env["NATS_URL"] = g.cfg.NATS.URL
		}
	}

	// Add components/services
	if len(container.ComponentNames) > 0 {
		env["GORAI_COMPONENTS"] = strings.Join(container.ComponentNames, ",")
	}
	if len(container.ServiceNames) > 0 {
		env["GORAI_SERVICES"] = strings.Join(container.ServiceNames, ",")
	}

	return env
}

// formatVolume formats a volume specification for Quadlet.
func (g *Generator) formatVolume(vol string) string {
	// Check if it's a named volume reference (contains .volume)
	parts := strings.SplitN(vol, ":", 2)
	if len(parts) >= 1 {
		// Check if source is a named volume (no path separators, not starting with . or /)
		source := parts[0]
		if !strings.HasPrefix(source, "/") && !strings.HasPrefix(source, ".") && !strings.Contains(source, string(filepath.Separator)) {
			// Could be a named volume - check if it exists in our volumes
			if g.cfg.Volumes != nil {
				if _, exists := g.cfg.Volumes[source]; exists {
					// Convert to .volume reference
					volFile := fmt.Sprintf("%s-%s.volume", g.cfg.Robot.Name, source)
					if len(parts) > 1 {
						return fmt.Sprintf("%s:%s", volFile, parts[1])
					}
					return volFile
				}
			}
		}
	}

	// Add :Z for SELinux relabeling if not already present
	if !strings.HasSuffix(vol, ":Z") && !strings.HasSuffix(vol, ":z") && !strings.Contains(vol, ":Z,") && !strings.Contains(vol, ":z,") {
		if strings.Contains(vol, ":") {
			// Has options but no SELinux label
			if strings.HasSuffix(vol, ":ro") || strings.HasSuffix(vol, ":rw") {
				vol = vol + ",Z"
			} else {
				vol = vol + ":Z"
			}
		} else {
			vol = vol + ":Z"
		}
	}

	return vol
}

// formatDevice formats a device specification for Quadlet.
func (g *Generator) formatDevice(dev string) string {
	// If device doesn't have a mapping, add one
	if !strings.Contains(dev, ":") {
		return fmt.Sprintf("%s:%s", dev, dev)
	}
	return dev
}

// mapRestartPolicy maps compose restart policies to systemd.
func (g *Generator) mapRestartPolicy(policy string) string {
	switch policy {
	case "always":
		return "always"
	case "unless-stopped":
		return "always" // systemd doesn't distinguish
	case "on-failure":
		return "on-failure"
	case "no", "":
		return "no"
	default:
		return "on-failure"
	}
}

// generateBuildUnit generates a .build unit file for containers with build config.
func (g *Generator) generateBuildUnit(name string, build *config.BuildConfig) string {
	var b strings.Builder

	buildName := fmt.Sprintf("%s-%s", g.cfg.Robot.Name, name)

	b.WriteString("[Unit]\n")
	b.WriteString(fmt.Sprintf("Description=Build image for %s\n", buildName))
	b.WriteString("\n")

	b.WriteString("[Build]\n")

	// Dockerfile/Containerfile
	dockerfile := build.Dockerfile
	if dockerfile == "" {
		dockerfile = "Containerfile"
	}

	context := build.Context
	if g.workspaceDir != "" && context != "" && !filepath.IsAbs(context) {
		context = filepath.Join(g.workspaceDir, context)
	}

	if context != "" {
		b.WriteString(fmt.Sprintf("SetWorkingDirectory=%s\n", context))
	}
	b.WriteString(fmt.Sprintf("File=%s\n", dockerfile))
	b.WriteString(fmt.Sprintf("ImageTag=%s:latest\n", buildName))

	// Build args
	for key, value := range build.Args {
		b.WriteString(fmt.Sprintf("BuildArg=%s=%s\n", key, value))
	}

	// Target
	if build.Target != "" {
		b.WriteString(fmt.Sprintf("Target=%s\n", build.Target))
	}

	b.WriteString("\n")

	b.WriteString("[Install]\n")
	b.WriteString(fmt.Sprintf("WantedBy=%s\n", g.wantedBy()))

	return b.String()
}
