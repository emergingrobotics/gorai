package quadlet

import (
	"fmt"
	"strings"

	"github.com/gorai/gorai/pkg/config"
)

// generateDefaultNetwork generates the default network unit file.
func (g *Generator) generateDefaultNetwork(name string) string {
	var b strings.Builder

	b.WriteString("[Unit]\n")
	b.WriteString(fmt.Sprintf("Description=Gorai default network for %s\n", g.cfg.Robot.Name))
	b.WriteString("\n")

	b.WriteString("[Network]\n")
	b.WriteString(fmt.Sprintf("NetworkName=%s\n", name))
	b.WriteString("Driver=bridge\n")
	b.WriteString("\n")

	b.WriteString("[Install]\n")
	b.WriteString(fmt.Sprintf("WantedBy=%s\n", g.wantedBy()))

	return b.String()
}

// generateNetwork generates a .network unit file content.
func (g *Generator) generateNetwork(name string, network *config.NetworkConfig) string {
	var b strings.Builder

	networkName := fmt.Sprintf("%s-%s", g.cfg.Robot.Name, name)

	b.WriteString("[Unit]\n")
	b.WriteString(fmt.Sprintf("Description=Gorai network %s\n", networkName))
	b.WriteString("\n")

	b.WriteString("[Network]\n")
	b.WriteString(fmt.Sprintf("NetworkName=%s\n", networkName))

	// Driver
	driver := network.Driver
	if driver == "" {
		driver = "bridge"
	}
	b.WriteString(fmt.Sprintf("Driver=%s\n", driver))

	// Internal network (no external access)
	if network.Internal {
		b.WriteString("Internal=true\n")
	}

	// Driver options
	for key, value := range network.Options {
		b.WriteString(fmt.Sprintf("Options=%s=%s\n", key, value))
	}

	b.WriteString("\n")

	b.WriteString("[Install]\n")
	b.WriteString(fmt.Sprintf("WantedBy=%s\n", g.wantedBy()))

	return b.String()
}
