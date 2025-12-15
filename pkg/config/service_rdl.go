// Package config provides configuration loading and management for Gorai robots.
// This file defines the Service RDL (Robot Definition Language) structures for
// external services that can be defined independently of any specific robot.

package config

// ServiceRDL represents a Service Definition Language configuration.
// Service RDL files define external services independently, allowing them
// to be reused across multiple robots.
type ServiceRDL struct {
	Schema  string               `json:"$schema,omitempty"`
	Version string               `json:"version"`
	Kind    string               `json:"kind"`
	Service ServiceRDLMeta       `json:"service"`
	Topics  ServiceRDLTopics     `json:"topics"`
	Attrs   ServiceRDLAttributes `json:"attributes,omitempty"`
	Runtime *ServiceRDLRuntime   `json:"runtime,omitempty"`
}

// ServiceRDLMeta defines the service metadata.
type ServiceRDLMeta struct {
	Type        string `json:"type"`
	Model       string `json:"model"`
	Description string `json:"description,omitempty"`
}

// ServiceRDLTopics defines the input and output topics for the service.
type ServiceRDLTopics struct {
	Subscribe []ServiceRDLTopicEntry `json:"subscribe,omitempty"`
	Publish   []ServiceRDLTopicEntry `json:"publish,omitempty"`
}

// ServiceRDLTopicEntry defines a single topic subscription or publication.
type ServiceRDLTopicEntry struct {
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

// ResolvedTopics holds the resolved topic names after pattern substitution.
type ResolvedTopics struct {
	Subscribe map[string]string // name -> resolved topic
	Publish   map[string]string // name -> resolved topic
}

// AllTopics returns all resolved topic names as a flat map.
func (rt *ResolvedTopics) AllTopics() map[string]string {
	result := make(map[string]string)
	for k, v := range rt.Subscribe {
		result[k] = v
	}
	for k, v := range rt.Publish {
		result[k] = v
	}
	return result
}

// InputTopicsJSON returns the subscribe topics as JSON-encodable map.
func (rt *ResolvedTopics) InputTopicsJSON() map[string]string {
	return rt.Subscribe
}

// OutputTopicsJSON returns the publish topics as JSON-encodable map.
func (rt *ResolvedTopics) OutputTopicsJSON() map[string]string {
	return rt.Publish
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
