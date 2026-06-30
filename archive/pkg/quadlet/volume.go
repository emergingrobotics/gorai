package quadlet

import (
	"fmt"
	"strings"

	"github.com/emergingrobotics/gorai/pkg/config"
)

// generateVolume generates a .volume unit file content.
func (g *Generator) generateVolume(name string, volume *config.VolumeConfig) string {
	var b strings.Builder

	volumeName := fmt.Sprintf("%s-%s", g.cfg.Robot.Name, name)

	b.WriteString("[Unit]\n")
	b.WriteString(fmt.Sprintf("Description=Gorai volume %s\n", volumeName))
	b.WriteString("\n")

	b.WriteString("[Volume]\n")
	b.WriteString(fmt.Sprintf("VolumeName=%s\n", volumeName))

	// Driver
	if volume.Driver != "" {
		b.WriteString(fmt.Sprintf("Driver=%s\n", volume.Driver))
	}

	// Driver options
	for key, value := range volume.Options {
		b.WriteString(fmt.Sprintf("Options=%s=%s\n", key, value))
	}

	b.WriteString("\n")

	b.WriteString("[Install]\n")
	b.WriteString(fmt.Sprintf("WantedBy=%s\n", g.wantedBy()))

	return b.String()
}
