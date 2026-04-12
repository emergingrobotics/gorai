package robot

import (
	"fmt"
	"strings"

	"github.com/gorai/gorai/pkg/config"
)

// topoSortComponents returns components ordered so that dependencies come
// before dependents. Returns an error for cycles or missing dependencies.
func topoSortComponents(components []config.ComponentConfig) ([]config.ComponentConfig, error) {
	byName := make(map[string]config.ComponentConfig, len(components))
	for _, c := range components {
		byName[c.Name] = c
	}

	for _, c := range components {
		for _, dep := range c.DependsOn {
			if _, ok := byName[dep]; !ok {
				return nil, fmt.Errorf("component %q depends on %q which is not defined", c.Name, dep)
			}
		}
	}

	inDegree := make(map[string]int, len(components))
	dependents := make(map[string][]string)
	for _, c := range components {
		if _, ok := inDegree[c.Name]; !ok {
			inDegree[c.Name] = 0
		}
		for _, dep := range c.DependsOn {
			inDegree[c.Name]++
			dependents[dep] = append(dependents[dep], c.Name)
		}
	}

	var queue []string
	for _, c := range components {
		if inDegree[c.Name] == 0 {
			queue = append(queue, c.Name)
		}
	}

	var sorted []config.ComponentConfig
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		sorted = append(sorted, byName[name])

		for _, dependent := range dependents[name] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(sorted) != len(components) {
		var stuck []string
		for name, degree := range inDegree {
			if degree > 0 {
				stuck = append(stuck, name)
			}
		}
		return nil, fmt.Errorf("circular dependency involving: %s", strings.Join(stuck, ", "))
	}

	return sorted, nil
}
