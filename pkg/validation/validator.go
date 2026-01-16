// Package validation provides component and service validation functionality.
package validation

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gorai/gorai/pkg/component/metadata"
	"github.com/gorai/gorai/pkg/service/rdl"
)

// ComponentValidator validates component repositories
type ComponentValidator struct {
	strict bool
	path   string
}

// NewComponentValidator creates a new component validator
func NewComponentValidator(path string, strict bool) *ComponentValidator {
	return &ComponentValidator{
		path:   path,
		strict: strict,
	}
}

// Validate performs comprehensive validation of a component
func (cv *ComponentValidator) Validate() (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:    true,
		Warnings: make([]string, 0),
		Errors:   make([]string, 0),
	}

	// 1. Check gorai-component.yaml exists
	metadataPath, err := metadata.FindInDirectory(cv.path)
	if err != nil {
		result.AddError("gorai-component.yaml not found")
		result.Valid = false
		return result, nil
	}
	result.AddCheck("gorai-component.yaml exists", true)

	// 2. Parse and validate metadata
	md, err := metadata.ParseFile(metadataPath)
	if err != nil {
		result.AddError(fmt.Sprintf("Failed to parse metadata: %v", err))
		result.Valid = false
		return result, nil
	}
	result.AddCheck("Metadata parses correctly", true)

	// 3. Validate metadata schema
	if errs := md.Validate(); len(errs) > 0 {
		for _, err := range errs {
			result.AddError(err.Error())
		}
		result.Valid = false
	} else {
		result.AddCheck("Metadata schema is valid", true)
	}

	// 4. Check go.mod exists
	goModPath := filepath.Join(cv.path, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		result.AddCheck("go.mod exists", true)

		// Verify go.mod has correct module path
		if cv.validateGoMod(goModPath, md.Component.Repository) {
			result.AddCheck("go.mod module matches repository", true)
		} else {
			result.AddWarning("go.mod module path doesn't match repository")
		}
	} else {
		result.AddError("go.mod not found")
		result.Valid = false
	}

	// 5. Check LICENSE file
	if cv.fileExists("LICENSE") || cv.fileExists("LICENSE.txt") || cv.fileExists("LICENSE.md") {
		result.AddCheck("LICENSE file exists", true)
	} else {
		if cv.strict {
			result.AddError("LICENSE file not found")
			result.Valid = false
		} else {
			result.AddWarning("LICENSE file not found")
		}
	}

	// 6. Check README
	if cv.fileExists("README.md") || cv.fileExists("README") {
		result.AddCheck("README exists", true)
	} else {
		if cv.strict {
			result.AddError("README.md not found")
			result.Valid = false
		} else {
			result.AddWarning("README.md not found")
		}
	}

	// 7. Run tests
	if cv.hasTests() {
		if testOutput, testsPassed := cv.runTests(); testsPassed {
			result.AddCheck("Tests pass", true)
			result.TestOutput = testOutput
		} else {
			result.AddError("Tests failed")
			result.TestOutput = testOutput
			result.Valid = false
		}
	} else {
		result.AddWarning("No tests found")
	}

	// 8. Check for examples
	if cv.fileExists("examples") {
		result.AddCheck("Examples directory exists", true)
		result.QualityScore += 10
	} else {
		result.AddWarning("No examples/ directory found")
	}

	// 9. Validate component registration (if possible)
	// This would require actually loading the Go code, which is complex
	// For now, just check that model files exist
	for _, p := range md.Component.Provides {
		modelDir := filepath.Join(cv.path, p.Model)
		if _, err := os.Stat(modelDir); err == nil {
			result.AddCheck(fmt.Sprintf("Model directory exists: %s", p.Model), true)
		} else {
			result.AddWarning(fmt.Sprintf("Model directory not found: %s", p.Model))
		}
	}

	// Calculate quality score
	result.QualityScore = cv.calculateQualityScore(md, result)

	return result, nil
}

// ServiceValidator validates service repositories
type ServiceValidator struct {
	strict bool
	path   string
}

// NewServiceValidator creates a new service validator
func NewServiceValidator(path string, strict bool) *ServiceValidator {
	return &ServiceValidator{
		path:   path,
		strict: strict,
	}
}

// Validate performs comprehensive validation of a service
func (sv *ServiceValidator) Validate() (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:    true,
		Warnings: make([]string, 0),
		Errors:   make([]string, 0),
	}

	// 1. Check service.rdl.json exists
	rdlPath, err := rdl.FindInDirectory(sv.path)
	if err != nil {
		result.AddError("service.rdl.json not found")
		result.Valid = false
		return result, nil
	}
	result.AddCheck("service.rdl.json exists", true)

	// 2. Parse and validate RDL
	serviceRDL, err := rdl.ParseFile(rdlPath)
	if err != nil {
		result.AddError(fmt.Sprintf("Failed to parse RDL: %v", err))
		result.Valid = false
		return result, nil
	}
	result.AddCheck("RDL parses correctly", true)

	// 3. Validate RDL schema
	if errs := serviceRDL.Validate(); len(errs) > 0 {
		for _, err := range errs {
			result.AddError(err.Error())
		}
		result.Valid = false
	} else {
		result.AddCheck("RDL schema is valid", true)
	}

	// 4. Check Containerfile exists
	if sv.fileExists("Containerfile") || sv.fileExists("Dockerfile") {
		result.AddCheck("Containerfile exists", true)
	} else {
		result.AddError("Containerfile/Dockerfile not found")
		result.Valid = false
	}

	// 5. Check README
	if sv.fileExists("README.md") {
		result.AddCheck("README.md exists", true)
	} else {
		result.AddWarning("README.md not found")
	}

	// 6. Validate container image accessibility (if possible)
	// This is optional and can be slow
	if serviceRDL.Service.Container != nil {
		result.AddCheck(fmt.Sprintf("Default image specified: %s", serviceRDL.Service.Container.DefaultImage), true)

		// Check image variants
		if len(serviceRDL.Service.Container.ImageVariants) > 0 {
			result.AddCheck(fmt.Sprintf("Image variants: %d", len(serviceRDL.Service.Container.ImageVariants)), true)
			result.QualityScore += 10
		}
	}

	// 7. Check NATS topic documentation
	if serviceRDL.Service.NATSTopics != nil {
		subsCount := len(serviceRDL.Service.NATSTopics.Subscribes)
		pubsCount := len(serviceRDL.Service.NATSTopics.Publishes)
		result.AddCheck(fmt.Sprintf("NATS topics documented (%d subscribes, %d publishes)", subsCount, pubsCount), true)
		result.QualityScore += 10
	} else {
		result.AddWarning("NATS topics not documented")
	}

	// Calculate quality score
	result.QualityScore = sv.calculateQualityScore(serviceRDL, result)

	return result, nil
}

// ValidationResult contains validation results
type ValidationResult struct {
	Valid        bool
	Errors       []string
	Warnings     []string
	Checks       []Check
	TestOutput   string
	QualityScore int
}

// Check represents a validation check
type Check struct {
	Name   string
	Passed bool
}

// AddError adds an error to the result
func (vr *ValidationResult) AddError(msg string) {
	vr.Errors = append(vr.Errors, msg)
}

// AddWarning adds a warning to the result
func (vr *ValidationResult) AddWarning(msg string) {
	vr.Warnings = append(vr.Warnings, msg)
}

// AddCheck adds a check result
func (vr *ValidationResult) AddCheck(name string, passed bool) {
	vr.Checks = append(vr.Checks, Check{Name: name, Passed: passed})
}

// Helper methods

func (cv *ComponentValidator) fileExists(name string) bool {
	path := filepath.Join(cv.path, name)
	_, err := os.Stat(path)
	return err == nil
}

func (sv *ServiceValidator) fileExists(name string) bool {
	path := filepath.Join(sv.path, name)
	_, err := os.Stat(path)
	return err == nil
}

func (cv *ComponentValidator) hasTests() bool {
	// Look for _test.go files
	matches, err := filepath.Glob(filepath.Join(cv.path, "**/*_test.go"))
	if err != nil {
		return false
	}
	return len(matches) > 0
}

func (cv *ComponentValidator) runTests() (string, bool) {
	cmd := exec.Command("go", "test", "./...", "-v")
	cmd.Dir = cv.path
	output, err := cmd.CombinedOutput()
	return string(output), err == nil
}

func (cv *ComponentValidator) validateGoMod(goModPath, expectedRepo string) bool {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return false
	}

	// Check if module line matches repository
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "module ") {
			modulePath := strings.TrimSpace(strings.TrimPrefix(line, "module"))
			return modulePath == expectedRepo
		}
	}

	return false
}

func (cv *ComponentValidator) calculateQualityScore(md *metadata.ComponentMetadata, result *ValidationResult) int {
	score := 50 // Base score

	// Add points for quality indicators
	if md.Component.Quality != nil {
		if md.Component.Quality.TestCoverage > 0 {
			score += md.Component.Quality.TestCoverage / 5 // Up to 20 points
		}
		if md.Component.Quality.HasExamples {
			score += 10
		}
		if md.Component.Quality.HasIntegrationTests {
			score += 10
		}
	}

	// Add points for documentation
	if md.Component.Documentation != "" {
		score += 5
	}

	// Subtract points for warnings/errors
	score -= len(result.Warnings) * 2
	score -= len(result.Errors) * 5

	// Clamp between 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

func (sv *ServiceValidator) calculateQualityScore(serviceRDL *rdl.ServiceRDL, result *ValidationResult) int {
	score := 50 // Base score

	// Add points for quality indicators
	if serviceRDL.Service.Quality != nil {
		if serviceRDL.Service.Quality.TestCoverage > 0 {
			score += serviceRDL.Service.Quality.TestCoverage / 5
		}
		if serviceRDL.Service.Quality.HasBenchmarks {
			score += 10
		}
	}

	// Add points for good documentation
	if serviceRDL.Service.Documentation != "" {
		score += 5
	}
	if len(serviceRDL.Service.Examples) > 0 {
		score += 10
	}

	// Add points for performance documentation
	if serviceRDL.Service.Performance != nil {
		score += 5
	}

	// Subtract points for warnings/errors
	score -= len(result.Warnings) * 2
	score -= len(result.Errors) * 5

	// Clamp between 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}
