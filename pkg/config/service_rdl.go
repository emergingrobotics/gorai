// Package config provides configuration loading and management for Gorai robots.
// This file defines the Service RDL (Robot Definition Language) structures for
// external services that can be defined independently of any specific robot.

package config

// ServiceRDL represents a Service Definition Language configuration.
// Service RDL files define external services independently, allowing them
// to be reused across multiple robots.
type ServiceRDL struct {
	Schema   string               `json:"$schema,omitempty"`
	Version  string               `json:"version"`
	Kind     string               `json:"kind"`
	Service  ServiceRDLMeta       `json:"service"`
	Subjects ServiceRDLSubjects   `json:"subjects"`
	Attrs    ServiceRDLAttributes `json:"attributes,omitempty"`
	Runtime  *ServiceRDLRuntime   `json:"runtime,omitempty"`
}

// ServiceRDLMeta defines the service metadata.
type ServiceRDLMeta struct {
	Type        string `json:"type"`
	Model       string `json:"model"`
	Description string `json:"description,omitempty"`
}

// ServiceRDLSubjects defines the input and output subjects for the service.
type ServiceRDLSubjects struct {
	Subscribe []ServiceRDLSubjectEntry `json:"subscribe,omitempty"`
	Publish   []ServiceRDLSubjectEntry `json:"publish,omitempty"`
}

// ServiceRDLSubjectEntry defines a single subject subscription or publication.
type ServiceRDLSubjectEntry struct {
	Name        string `json:"name"`
	Pattern     string `json:"pattern"`
	Description string `json:"description,omitempty"`
	Format      string `json:"format,omitempty"`
}

// ServiceRDLAttributes is a map of attribute name to attribute definition.
type ServiceRDLAttributes map[string]ServiceRDLAttrDef

// ServiceRDLAttrDef defines a configurable attribute for the service.
type ServiceRDLAttrDef struct {
	Type        string   `json:"type"`
	Default     any      `json:"default,omitempty"`
	Required    bool     `json:"required,omitempty"`
	Description string   `json:"description,omitempty"`
	Min         *float64 `json:"min,omitempty"`
	Max         *float64 `json:"max,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

// ServiceRDLRuntime defines default runtime configuration for the service.
type ServiceRDLRuntime struct {
	Container *ServiceRDLContainer `json:"container,omitempty"`
	Command   string               `json:"command,omitempty"`
	Env       map[string]string    `json:"env,omitempty"`
}

// ServiceRDLContainer defines container configuration in Service RDL.
type ServiceRDLContainer struct {
	Image       string            `json:"image,omitempty"`
	Build       *ServiceRDLBuild  `json:"build,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Network     string            `json:"network,omitempty"`
}

// ServiceRDLBuild defines build configuration in Service RDL.
type ServiceRDLBuild struct {
	Context       string            `json:"context,omitempty"`
	Containerfile string            `json:"containerfile,omitempty"`
	Args          map[string]string `json:"args,omitempty"`
}

// ResolvedSubjects holds the resolved subject names after pattern substitution.
type ResolvedSubjects struct {
	Subscribe map[string]string // name -> resolved subject
	Publish   map[string]string // name -> resolved subject
}

// AllSubjects returns all resolved subject names as a flat map.
func (rs *ResolvedSubjects) AllSubjects() map[string]string {
	result := make(map[string]string)
	for k, v := range rs.Subscribe {
		result[k] = v
	}
	for k, v := range rs.Publish {
		result[k] = v
	}
	return result
}

// InputSubjectsJSON returns the subscribe subjects as JSON-encodable map.
func (rs *ResolvedSubjects) InputSubjectsJSON() map[string]string {
	return rs.Subscribe
}

// OutputSubjectsJSON returns the publish subjects as JSON-encodable map.
func (rs *ResolvedSubjects) OutputSubjectsJSON() map[string]string {
	return rs.Publish
}

// ValidAttrTypes lists the valid attribute type names.
var ValidAttrTypes = []string{"string", "int", "float", "bool", "array", "object"}

// IsValidAttrType checks if the given type name is valid.
func IsValidAttrType(t string) bool {
	for _, valid := range ValidAttrTypes {
		if t == valid {
			return true
		}
	}
	return false
}
