package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadServiceRDL loads a Service RDL file from the given path.
func LoadServiceRDL(path string) (*ServiceRDL, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read service RDL file: %w", err)
	}
	return ParseServiceRDL(data, path)
}

// LoadServiceRDLRelative loads a Service RDL file relative to a base directory.
func LoadServiceRDLRelative(rdlPath, baseDir string) (*ServiceRDL, error) {
	var fullPath string
	if filepath.IsAbs(rdlPath) {
		fullPath = rdlPath
	} else {
		fullPath = filepath.Join(baseDir, rdlPath)
	}
	return LoadServiceRDL(fullPath)
}

// ParseServiceRDL parses Service RDL from JSON bytes.
func ParseServiceRDL(data []byte, sourcePath string) (*ServiceRDL, error) {
	var rdl ServiceRDL
	if err := json.Unmarshal(data, &rdl); err != nil {
		return nil, fmt.Errorf("failed to parse service RDL JSON: %w", err)
	}

	// Validate the parsed RDL
	if err := rdl.Validate(sourcePath); err != nil {
		return nil, err
	}

	return &rdl, nil
}

// Validate validates the Service RDL configuration.
func (rdl *ServiceRDL) Validate(sourcePath string) error {
	var errs []string
	prefix := ""
	if sourcePath != "" {
		prefix = sourcePath + ": "
	}

	// Check version
	if rdl.Version == "" {
		errs = append(errs, prefix+"version is required")
	} else if rdl.Version != "1" {
		errs = append(errs, fmt.Sprintf("%sunsupported version %q, expected \"1\"", prefix, rdl.Version))
	}

	// Check kind
	if rdl.Kind == "" {
		errs = append(errs, prefix+"kind is required")
	} else if rdl.Kind != "service" {
		errs = append(errs, fmt.Sprintf("%skind must be \"service\", got %q", prefix, rdl.Kind))
	}

	// Check service metadata
	if rdl.Service.Type == "" {
		errs = append(errs, prefix+"service.type is required")
	}
	if rdl.Service.Model == "" {
		errs = append(errs, prefix+"service.model is required")
	}

	// Check subjects
	if len(rdl.Subjects.Subscribe) == 0 && len(rdl.Subjects.Publish) == 0 {
		errs = append(errs, prefix+"at least one subject (subscribe or publish) is required")
	}

	// Validate subscribe subjects
	subjectNames := make(map[string]bool)
	for i, subject := range rdl.Subjects.Subscribe {
		if subject.Name == "" {
			errs = append(errs, fmt.Sprintf("%ssubjects.subscribe[%d].name is required", prefix, i))
		} else if subjectNames[subject.Name] {
			errs = append(errs, fmt.Sprintf("%ssubjects.subscribe[%d].name: duplicate subject name %q", prefix, i, subject.Name))
		} else {
			subjectNames[subject.Name] = true
		}

		if subject.Pattern == "" {
			errs = append(errs, fmt.Sprintf("%ssubjects.subscribe[%d].pattern is required", prefix, i))
		}
	}

	// Validate publish subjects
	for i, subject := range rdl.Subjects.Publish {
		if subject.Name == "" {
			errs = append(errs, fmt.Sprintf("%ssubjects.publish[%d].name is required", prefix, i))
		} else if subjectNames[subject.Name] {
			errs = append(errs, fmt.Sprintf("%ssubjects.publish[%d].name: duplicate subject name %q", prefix, i, subject.Name))
		} else {
			subjectNames[subject.Name] = true
		}

		if subject.Pattern == "" {
			errs = append(errs, fmt.Sprintf("%ssubjects.publish[%d].pattern is required", prefix, i))
		}
	}

	// Validate attributes
	for name, attr := range rdl.Attrs {
		if attr.Type == "" {
			errs = append(errs, fmt.Sprintf("%sattributes.%s.type is required", prefix, name))
		} else if !IsValidAttrType(attr.Type) {
			errs = append(errs, fmt.Sprintf("%sattributes.%s.type: invalid type %q, valid types: %s",
				prefix, name, attr.Type, strings.Join(ValidAttrTypes, ", ")))
		}

		// Required attributes should not have defaults (debatable, but good practice)
		// Actually, we'll allow defaults on required attrs for documentation purposes
		// But warn if required has a default

		// Validate numeric ranges
		if attr.Min != nil && attr.Max != nil && *attr.Min > *attr.Max {
			errs = append(errs, fmt.Sprintf("%sattributes.%s: min (%v) cannot be greater than max (%v)",
				prefix, name, *attr.Min, *attr.Max))
		}

		// Validate enum only applies to string type
		if len(attr.Enum) > 0 && attr.Type != "string" {
			errs = append(errs, fmt.Sprintf("%sattributes.%s: enum can only be used with string type",
				prefix, name))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("service RDL validation errors:\n  %s", strings.Join(errs, "\n  "))
	}

	return nil
}

// GetDefaultAttributes returns a map of attribute names to their default values.
func (rdl *ServiceRDL) GetDefaultAttributes() map[string]any {
	defaults := make(map[string]any)
	for name, attr := range rdl.Attrs {
		if attr.Default != nil {
			defaults[name] = attr.Default
		}
	}
	return defaults
}

// GetRequiredAttributes returns a list of required attribute names.
func (rdl *ServiceRDL) GetRequiredAttributes() []string {
	var required []string
	for name, attr := range rdl.Attrs {
		if attr.Required {
			required = append(required, name)
		}
	}
	return required
}

// ValidateAttributes validates provided attributes against the Service RDL definitions.
func (rdl *ServiceRDL) ValidateAttributes(attrs map[string]any) error {
	var errs []string

	// Check required attributes
	for name, def := range rdl.Attrs {
		if def.Required {
			if _, ok := attrs[name]; !ok {
				errs = append(errs, fmt.Sprintf("required attribute %q is missing", name))
			}
		}
	}

	// Validate provided attributes
	for name, value := range attrs {
		def, ok := rdl.Attrs[name]
		if !ok {
			// Unknown attribute - could be a warning but we'll allow it for flexibility
			continue
		}

		// Type validation
		if err := validateAttrType(name, value, def); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("attribute validation errors:\n  %s", strings.Join(errs, "\n  "))
	}

	return nil
}

// validateAttrType validates a single attribute value against its definition.
func validateAttrType(name string, value any, def ServiceRDLAttrDef) error {
	switch def.Type {
	case "string":
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("attribute %q must be a string", name)
		}
		if len(def.Enum) > 0 {
			found := false
			for _, e := range def.Enum {
				if s == e {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("attribute %q must be one of: %s", name, strings.Join(def.Enum, ", "))
			}
		}

	case "int":
		var n float64
		switch v := value.(type) {
		case float64:
			n = v
		case int:
			n = float64(v)
		case int64:
			n = float64(v)
		default:
			return fmt.Errorf("attribute %q must be an integer", name)
		}
		if def.Min != nil && n < *def.Min {
			return fmt.Errorf("attribute %q must be >= %v", name, *def.Min)
		}
		if def.Max != nil && n > *def.Max {
			return fmt.Errorf("attribute %q must be <= %v", name, *def.Max)
		}

	case "float":
		var n float64
		switch v := value.(type) {
		case float64:
			n = v
		case int:
			n = float64(v)
		case int64:
			n = float64(v)
		default:
			return fmt.Errorf("attribute %q must be a number", name)
		}
		if def.Min != nil && n < *def.Min {
			return fmt.Errorf("attribute %q must be >= %v", name, *def.Min)
		}
		if def.Max != nil && n > *def.Max {
			return fmt.Errorf("attribute %q must be <= %v", name, *def.Max)
		}

	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("attribute %q must be a boolean", name)
		}

	case "array":
		if _, ok := value.([]any); !ok {
			// Also accept []string, []int, etc.
			// JSON unmarshaling creates []interface{}, so this is the common case
			return fmt.Errorf("attribute %q must be an array", name)
		}

	case "object":
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf("attribute %q must be an object", name)
		}
	}

	return nil
}

// MergeAttributes merges Service RDL defaults with provided attributes.
// Provided attributes override defaults.
func (rdl *ServiceRDL) MergeAttributes(provided map[string]any) map[string]any {
	result := make(map[string]any)

	// Start with defaults
	for name, def := range rdl.Attrs {
		if def.Default != nil {
			result[name] = def.Default
		}
	}

	// Override with provided values
	for name, value := range provided {
		result[name] = value
	}

	return result
}
