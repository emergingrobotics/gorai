package robot

import (
	"testing"

	"github.com/gorai/gorai/pkg/config"
)

func TestTopoSort_NoDeps(t *testing.T) {
	components := []config.ComponentConfig{
		{Name: "a", Type: "sensor", Model: "fake"},
		{Name: "b", Type: "motor", Model: "fake"},
	}
	sorted, err := topoSortComponents(components)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sorted) != 2 {
		t.Fatalf("expected 2 components, got %d", len(sorted))
	}
}

func TestTopoSort_LinearDeps(t *testing.T) {
	components := []config.ComponentConfig{
		{Name: "servo", Type: "servo", Model: "fake", DependsOn: []string{"mcu"}},
		{Name: "mcu", Type: "i2c_device", Model: "fake"},
	}
	sorted, err := topoSortComponents(components)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sorted[0].Name != "mcu" {
		t.Fatalf("expected mcu first, got %s", sorted[0].Name)
	}
	if sorted[1].Name != "servo" {
		t.Fatalf("expected servo second, got %s", sorted[1].Name)
	}
}

func TestTopoSort_DiamondDeps(t *testing.T) {
	components := []config.ComponentConfig{
		{Name: "base", Type: "base", Model: "fake", DependsOn: []string{"left", "right"}},
		{Name: "left", Type: "motor", Model: "fake", DependsOn: []string{"mcu"}},
		{Name: "right", Type: "motor", Model: "fake", DependsOn: []string{"mcu"}},
		{Name: "mcu", Type: "i2c_device", Model: "fake"},
	}
	sorted, err := topoSortComponents(components)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	indexOf := func(name string) int {
		for i, c := range sorted {
			if c.Name == name {
				return i
			}
		}
		return -1
	}
	if indexOf("mcu") >= indexOf("left") || indexOf("mcu") >= indexOf("right") {
		t.Fatal("mcu must come before left and right")
	}
	if indexOf("left") >= indexOf("base") || indexOf("right") >= indexOf("base") {
		t.Fatal("left and right must come before base")
	}
}

func TestTopoSort_CyclicDeps(t *testing.T) {
	components := []config.ComponentConfig{
		{Name: "a", Type: "x", Model: "fake", DependsOn: []string{"b"}},
		{Name: "b", Type: "x", Model: "fake", DependsOn: []string{"a"}},
	}
	_, err := topoSortComponents(components)
	if err == nil {
		t.Fatal("expected error for cyclic dependency")
	}
}

func TestTopoSort_MissingDep(t *testing.T) {
	components := []config.ComponentConfig{
		{Name: "a", Type: "x", Model: "fake", DependsOn: []string{"nonexistent"}},
	}
	_, err := topoSortComponents(components)
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
}
