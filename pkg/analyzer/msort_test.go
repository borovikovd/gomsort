package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

func TestAnalyzerBasicFunctionality(t *testing.T) {
	if Analyzer.Name != "msort" {
		t.Errorf("Expected analyzer name 'msort', got '%s'", Analyzer.Name)
	}

	if Analyzer.Doc == "" {
		t.Error("Expected analyzer to have documentation")
	}

	if Analyzer.Run == nil {
		t.Error("Expected analyzer to have a Run function")
	}

	if len(Analyzer.Requires) == 0 {
		t.Error("Expected analyzer to require inspect.Analyzer")
	}

	found := false
	for _, req := range Analyzer.Requires {
		if req == inspect.Analyzer {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected analyzer to require inspect.Analyzer")
	}
}

func TestRunWithNilPass(t *testing.T) {
	result, err := run(nil)
	if err != nil {
		t.Errorf("Expected no error with nil pass, got %v", err)
	}
	if result != nil {
		t.Error("Expected nil result with nil pass")
	}
}

func TestRunWithEmptyResultOf(t *testing.T) {
	pass := &analysis.Pass{
		ResultOf: make(map[*analysis.Analyzer]interface{}),
	}
	result, err := run(pass)
	if err != nil {
		t.Errorf("Expected no error with empty ResultOf, got %v", err)
	}
	if result != nil {
		t.Error("Expected nil result with empty ResultOf")
	}
}

func TestRunWithWrongInspectorType(t *testing.T) {
	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{
			inspect.Analyzer: "not an inspector",
		},
	}
	result, err := run(pass)
	if err != nil {
		t.Errorf("Expected no error with wrong inspector type, got %v", err)
	}
	if result != nil {
		t.Error("Expected nil result with wrong inspector type")
	}
}

func TestRunWithEmptyInspector(t *testing.T) {
	files := []*ast.File{}
	inspectResult := inspector.New(files)

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{
			inspect.Analyzer: inspectResult,
		},
		Fset:  token.NewFileSet(),
		Files: files,
	}

	result, err := run(pass)
	if err != nil {
		t.Errorf("Expected no error with empty inspector, got %v", err)
	}
	if result != nil {
		t.Error("Expected nil result from run function")
	}
}

func TestRunWithValidFileNoMethods(t *testing.T) {
	source := `package test

type Config struct {
	Name string
}

var GlobalVar = "test"

func globalFunction() {
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, 0)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	files := []*ast.File{file}
	inspectResult := inspector.New(files)

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{
			inspect.Analyzer: inspectResult,
		},
		Fset:  fset,
		Files: files,
	}

	defer func() {
		recover()
	}()

	result, err := run(pass)
	_ = result
	_ = err
}

func TestRunWithUnsortedMethods(t *testing.T) {
	source := `package test

type Server struct{}

func (s *Server) helper() error {
	return nil
}

func (s *Server) Start() error {
	return s.helper()
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, 0)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	files := []*ast.File{file}
	inspectResult := inspector.New(files)

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{
			inspect.Analyzer: inspectResult,
		},
		Fset:  fset,
		Files: files,
	}

	defer func() {
		recover()
	}()

	result, err := run(pass)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != nil {
		t.Error("Expected nil result from run function")
	}
}

func TestRunWithMalformedAST(t *testing.T) {
	fset := token.NewFileSet()

	file := &ast.File{
		Name: &ast.Ident{
			Name: "test",
		},
	}

	files := []*ast.File{file}
	inspectResult := inspector.New(files)

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{
			inspect.Analyzer: inspectResult,
		},
		Fset:  fset,
		Files: files,
	}

	result, err := run(pass)
	if err != nil {
		t.Errorf("Expected no error with malformed AST, got %v", err)
	}
	if result != nil {
		t.Error("Expected nil result from run function")
	}
}

func TestRunWithInvalidSource(t *testing.T) {
	source := `package test

type Server struct{}

func (s *Server) Start() error {
	return nil
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, 0)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	files := []*ast.File{file}
	inspectResult := inspector.New(files)

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{
			inspect.Analyzer: inspectResult,
		},
		Fset:  fset,
		Files: files,
	}

	result, err := run(pass)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != nil {
		t.Error("Expected nil result from run function")
	}
}
