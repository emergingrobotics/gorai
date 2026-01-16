// Package metadata provides types and functions for parsing and validating
// gorai-component.yaml metadata files.
package metadata

import (
	"time"
)

// ComponentMetadata represents the complete gorai-component.yaml structure
type ComponentMetadata struct {
	SchemaVersion string    `yaml:"schema_version" json:"schema_version"`
	Component     Component `yaml:"component" json:"component"`
}

// Component represents the main component metadata
type Component struct {
	Name                 string                  `yaml:"name" json:"name"`
	Repository           string                  `yaml:"repository" json:"repository"`
	Version              string                  `yaml:"version" json:"version"`
	Provides             []Provides              `yaml:"provides" json:"provides"`
	Compatibility        Compatibility           `yaml:"compatibility" json:"compatibility"`
	Author               string                  `yaml:"author,omitempty" json:"author,omitempty"`
	License              string                  `yaml:"license,omitempty" json:"license,omitempty"`
	Description          string                  `yaml:"description,omitempty" json:"description,omitempty"`
	Homepage             string                  `yaml:"homepage,omitempty" json:"homepage,omitempty"`
	Documentation        string                  `yaml:"documentation,omitempty" json:"documentation,omitempty"`
	Support              *Support                `yaml:"support,omitempty" json:"support,omitempty"`
	Keywords             []string                `yaml:"keywords,omitempty" json:"keywords,omitempty"`
	HardwareRequirements *HardwareRequirements   `yaml:"hardware_requirements,omitempty" json:"hardware_requirements,omitempty"`
	Configuration        map[string]*ConfigAttr  `yaml:"configuration,omitempty" json:"configuration,omitempty"`
	Dependencies         *Dependencies           `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Quality              *Quality                `yaml:"quality,omitempty" json:"quality,omitempty"`
	Metadata             *Metadata               `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// Provides describes a component type+model pair
type Provides struct {
	Type        string `yaml:"type" json:"type"`
	Model       string `yaml:"model" json:"model"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// Compatibility describes version and platform constraints
type Compatibility struct {
	GoraiVersion string   `yaml:"gorai_version" json:"gorai_version"`
	Platforms    []string `yaml:"platforms,omitempty" json:"platforms,omitempty"`
	GoVersion    string   `yaml:"go_version,omitempty" json:"go_version,omitempty"`
}

// Support contains support contact information
type Support struct {
	Email string `yaml:"email,omitempty" json:"email,omitempty"`
	URL   string `yaml:"url,omitempty" json:"url,omitempty"`
	Chat  string `yaml:"chat,omitempty" json:"chat,omitempty"`
}

// HardwareRequirements describes required hardware interfaces
type HardwareRequirements struct {
	GPIO     bool     `yaml:"gpio,omitempty" json:"gpio,omitempty"`
	I2C      bool     `yaml:"i2c,omitempty" json:"i2c,omitempty"`
	SPI      bool     `yaml:"spi,omitempty" json:"spi,omitempty"`
	UART     bool     `yaml:"uart,omitempty" json:"uart,omitempty"`
	CAN      bool     `yaml:"can,omitempty" json:"can,omitempty"`
	USB      bool     `yaml:"usb,omitempty" json:"usb,omitempty"`
	Ethernet bool     `yaml:"ethernet,omitempty" json:"ethernet,omitempty"`
	PWM      bool     `yaml:"pwm,omitempty" json:"pwm,omitempty"`
	Custom   []string `yaml:"custom,omitempty" json:"custom,omitempty"`
}

// ConfigAttr describes a configuration attribute
type ConfigAttr struct {
	Type                string        `yaml:"type" json:"type"`
	Required            bool          `yaml:"required,omitempty" json:"required,omitempty"`
	Default             interface{}   `yaml:"default,omitempty" json:"default,omitempty"`
	Description         string        `yaml:"description,omitempty" json:"description,omitempty"`
	Example             interface{}   `yaml:"example,omitempty" json:"example,omitempty"`
	Enum                []interface{} `yaml:"enum,omitempty" json:"enum,omitempty"`
	Range               []float64     `yaml:"range,omitempty" json:"range,omitempty"`
	Pattern             string        `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	Deprecated          bool          `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`
	DeprecationMessage  string        `yaml:"deprecation_message,omitempty" json:"deprecation_message,omitempty"`
}

// Dependencies describes external dependencies
type Dependencies struct {
	SystemPackages []string     `yaml:"system_packages,omitempty" json:"system_packages,omitempty"`
	GoModules      []GoModule   `yaml:"go_modules,omitempty" json:"go_modules,omitempty"`
	Other          []string     `yaml:"other,omitempty" json:"other,omitempty"`
}

// GoModule describes a Go module dependency
type GoModule struct {
	Module  string `yaml:"module" json:"module"`
	Version string `yaml:"version" json:"version"`
}

// Quality contains quality metrics
type Quality struct {
	CIStatus              string `yaml:"ci_status,omitempty" json:"ci_status,omitempty"`
	TestCoverage          int    `yaml:"test_coverage,omitempty" json:"test_coverage,omitempty"`
	HasExamples           bool   `yaml:"has_examples,omitempty" json:"has_examples,omitempty"`
	HasIntegrationTests   bool   `yaml:"has_integration_tests,omitempty" json:"has_integration_tests,omitempty"`
}

// Metadata contains additional metadata
type Metadata struct {
	FirstReleased string `yaml:"first_released,omitempty" json:"first_released,omitempty"`
	LastUpdated   string `yaml:"last_updated,omitempty" json:"last_updated,omitempty"`
	Maturity      string `yaml:"maturity,omitempty" json:"maturity,omitempty"`
	Maintained    bool   `yaml:"maintained,omitempty" json:"maintained,omitempty"`
}

// SearchResult represents a component search result
type SearchResult struct {
	Repository  string    `json:"repository"`
	Version     string    `json:"version"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Provides    []Provides `json:"provides"`
	License     string    `json:"license"`
	Maturity    string    `json:"maturity"`
	LastUpdated time.Time `json:"last_updated"`
	Downloads   int       `json:"downloads,omitempty"`
	Stars       int       `json:"stars,omitempty"`
}
