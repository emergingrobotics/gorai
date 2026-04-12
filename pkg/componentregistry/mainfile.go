package componentregistry

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strconv"
)

// AddBlankImport adds a blank import to a Go source file if not already present.
// The file is reformatted with gofmt after editing.
func AddBlankImport(filePath, modulePath string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse %s: %w", filePath, err)
	}

	quotedPath := strconv.Quote(modulePath)

	// Check if already imported
	for _, imp := range node.Imports {
		if imp.Path.Value == quotedPath {
			return nil
		}
	}

	// Find existing grouped import declaration
	var targetDecl *ast.GenDecl
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if ok && genDecl.Tok == token.IMPORT && genDecl.Lparen.IsValid() {
			targetDecl = genDecl
			break
		}
	}

	newSpec := &ast.ImportSpec{
		Name: &ast.Ident{Name: "_"},
		Path: &ast.BasicLit{Kind: token.STRING, Value: quotedPath},
	}

	if targetDecl != nil {
		targetDecl.Specs = append(targetDecl.Specs, newSpec)
	} else {
		newDecl := &ast.GenDecl{
			Tok:    token.IMPORT,
			Lparen: 1,
			Specs:  []ast.Spec{newSpec},
		}
		node.Decls = append([]ast.Decl{newDecl}, node.Decls...)
	}

	// Write to temp file first for atomic replacement
	tmpPath := filePath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp file %s: %w", tmpPath, err)
	}

	if err := format.Node(f, fset, node); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("format %s: %w", filePath, err)
	}

	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename %s to %s: %w", tmpPath, filePath, err)
	}

	return nil
}
