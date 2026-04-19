package config

import (
	"fmt"
	"regexp"
	"strings"
)

// SubjectResolver resolves subject patterns with variable substitution.
type SubjectResolver struct {
	context map[string]string
}

// NewSubjectResolver creates a new SubjectResolver with the given context.
// The context contains variable name to value mappings.
func NewSubjectResolver(context map[string]string) *SubjectResolver {
	return &SubjectResolver{
		context: context,
	}
}

// NewSubjectResolverFromConfig creates a SubjectResolver from robot config and service info.
func NewSubjectResolverFromConfig(cfg *RDL, serviceName string, attrs map[string]any) *SubjectResolver {
	context := make(map[string]string)

	// Add robot-level variables
	context["namespace"] = cfg.GetEffectiveNamespace()
	context["robot"] = cfg.Robot.Name
	context["name"] = cfg.Robot.Name

	// Add service-level variables
	context["service"] = serviceName

	// Add attribute values as variables (only string values)
	for k, v := range attrs {
		if s, ok := v.(string); ok {
			context[k] = s
		}
	}

	return NewSubjectResolver(context)
}

// Resolve resolves a single subject pattern.
// Pattern variables are in the form {variable_name}.
func (r *SubjectResolver) Resolve(pattern string) (string, error) {
	// Find all variables in the pattern
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(pattern, -1)

	result := pattern
	var missingVars []string

	for _, match := range matches {
		fullMatch := match[0]  // e.g., "{namespace}"
		varName := match[1]    // e.g., "namespace"

		value, ok := r.context[varName]
		if !ok {
			missingVars = append(missingVars, varName)
			continue
		}

		result = strings.Replace(result, fullMatch, value, 1)
	}

	if len(missingVars) > 0 {
		return "", fmt.Errorf("undefined variables in pattern %q: %s", pattern, strings.Join(missingVars, ", "))
	}

	return result, nil
}

// ResolveAll resolves all subjects in a ServiceRDLSubjects configuration.
func (r *SubjectResolver) ResolveAll(subjects *ServiceRDLSubjects) (*ResolvedSubjects, error) {
	result := &ResolvedSubjects{
		Subscribe: make(map[string]string),
		Publish:   make(map[string]string),
	}

	var errs []string

	// Resolve subscribe subjects
	for _, subject := range subjects.Subscribe {
		resolved, err := r.Resolve(subject.Pattern)
		if err != nil {
			errs = append(errs, fmt.Sprintf("subscribe.%s: %v", subject.Name, err))
			continue
		}
		result.Subscribe[subject.Name] = resolved
	}

	// Resolve publish subjects
	for _, subject := range subjects.Publish {
		resolved, err := r.Resolve(subject.Pattern)
		if err != nil {
			errs = append(errs, fmt.Sprintf("publish.%s: %v", subject.Name, err))
			continue
		}
		result.Publish[subject.Name] = resolved
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("subject resolution errors:\n  %s", strings.Join(errs, "\n  "))
	}

	return result, nil
}

// AddVariable adds a variable to the context.
func (r *SubjectResolver) AddVariable(name, value string) {
	r.context[name] = value
}

// GetContext returns a copy of the current context.
func (r *SubjectResolver) GetContext() map[string]string {
	result := make(map[string]string)
	for k, v := range r.context {
		result[k] = v
	}
	return result
}

// ExtractVariables extracts all variable names from a pattern.
func ExtractVariables(pattern string) []string {
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(pattern, -1)

	vars := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		varName := match[1]
		if !seen[varName] {
			vars = append(vars, varName)
			seen[varName] = true
		}
	}

	return vars
}

// ValidatePatternVariables checks if all variables in patterns can be resolved.
func ValidatePatternVariables(subjects *ServiceRDLSubjects, attrs ServiceRDLAttributes) error {
	var errs []string

	// Built-in variables that are always available
	builtins := map[string]bool{
		"namespace": true,
		"robot":     true,
		"name":      true,
		"service":   true,
	}

	// Collect all variables from patterns
	allPatterns := make([]struct {
		name    string
		pattern string
	}, 0)

	for _, t := range subjects.Subscribe {
		allPatterns = append(allPatterns, struct {
			name    string
			pattern string
		}{"subscribe." + t.Name, t.Pattern})
	}
	for _, t := range subjects.Publish {
		allPatterns = append(allPatterns, struct {
			name    string
			pattern string
		}{"publish." + t.Name, t.Pattern})
	}

	// Check each pattern
	for _, p := range allPatterns {
		vars := ExtractVariables(p.pattern)
		for _, v := range vars {
			if !builtins[v] {
				// Must be an attribute
				if _, ok := attrs[v]; !ok {
					errs = append(errs, fmt.Sprintf("%s: variable {%s} must be a builtin (namespace, robot, service) or defined as an attribute",
						p.name, v))
				}
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("pattern variable errors:\n  %s", strings.Join(errs, "\n  "))
	}

	return nil
}
