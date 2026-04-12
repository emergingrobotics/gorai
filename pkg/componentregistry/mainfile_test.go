package componentregistry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddBlankImport(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.go")
	os.WriteFile(mainPath, []byte(`package main

import (
	"fmt"
)

func main() {
	fmt.Println("hello")
}
`), 0644)

	err := AddBlankImport(mainPath, "github.com/example/gorai-driver-hcsr04")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(mainPath)
	if !strings.Contains(string(content), `_ "github.com/example/gorai-driver-hcsr04"`) {
		t.Fatalf("import not added. Content:\n%s", content)
	}
}

func TestAddBlankImport_AlreadyPresent(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.go")
	os.WriteFile(mainPath, []byte(`package main

import (
	"fmt"
	_ "github.com/example/gorai-driver-hcsr04"
)

func main() {
	fmt.Println("hello")
}
`), 0644)

	err := AddBlankImport(mainPath, "github.com/example/gorai-driver-hcsr04")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(mainPath)
	count := strings.Count(string(content), "gorai-driver-hcsr04")
	if count != 1 {
		t.Fatalf("expected 1 occurrence, got %d", count)
	}
}

func TestAddBlankImport_NoImportBlock(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.go")
	os.WriteFile(mainPath, []byte(`package main

func main() {}
`), 0644)

	err := AddBlankImport(mainPath, "github.com/example/gorai-driver-hcsr04")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(mainPath)
	if !strings.Contains(string(content), `_ "github.com/example/gorai-driver-hcsr04"`) {
		t.Fatalf("import not added. Content:\n%s", content)
	}
}
