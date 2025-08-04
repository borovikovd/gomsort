package sorter

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestMethodSortKey(t *testing.T) {
	tests := []struct {
		name     string
		method   *MethodInfo
		expected MethodSortKey
	}{
		{
			name: "exported method",
			method: &MethodInfo{
				Name:         "Connect",
				ReceiverName: "Database",
				IsExported:   true,
				InDegree:     0,
				MaxDepth:     1,
				Position:     token.Pos(100),
			},
			expected: MethodSortKey{
				ReceiverName: "Database",
				IsExported:   true,
				InDegree:     0,
				MaxDepth:     1,
				OriginalPos:  token.Pos(100),
			},
		},
		{
			name: "private helper",
			method: &MethodInfo{
				Name:         "validateConnection",
				ReceiverName: "Database",
				IsExported:   false,
				InDegree:     3,
				MaxDepth:     0,
				Position:     token.Pos(200),
			},
			expected: MethodSortKey{
				ReceiverName: "Database",
				IsExported:   false,
				InDegree:     3,
				MaxDepth:     0,
				OriginalPos:  token.Pos(200),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.method.SortKey()
			if result != tt.expected {
				t.Errorf("SortKey() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestExtractMethodInfo(t *testing.T) {
	source := `
package test

type Server struct{}

func (s *Server) PublicMethod() {}
func (s *Server) privateMethod() {}
func (s Server) ValueReceiver() {}
func NotAMethod() {}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}

	var methods []*MethodInfo
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if method := extractMethodInfo(funcDecl); method != nil {
				methods = append(methods, method)
			}
		}
	}

	if len(methods) != 3 {
		t.Errorf("Expected 3 methods, got %d", len(methods))
	}

	expectedMethods := []struct {
		name         string
		receiverName string
		receiverType string
		isExported   bool
	}{
		{"PublicMethod", "Server", "*Server", true},
		{"privateMethod", "Server", "*Server", false},
		{"ValueReceiver", "Server", "Server", true},
	}

	for i, expected := range expectedMethods {
		if i >= len(methods) {
			t.Errorf("Missing method %d", i)
			continue
		}

		method := methods[i]
		if method.Name != expected.name {
			t.Errorf("Method %d: expected name %s, got %s", i, expected.name, method.Name)
		}
		if method.ReceiverName != expected.receiverName {
			t.Errorf("Method %d: expected receiver name %s, got %s", i, expected.receiverName, method.ReceiverName)
		}
		if method.ReceiverType != expected.receiverType {
			t.Errorf("Method %d: expected receiver type %s, got %s", i, expected.receiverType, method.ReceiverType)
		}
		if method.IsExported != expected.isExported {
			t.Errorf("Method %d: expected exported %v, got %v", i, expected.isExported, method.IsExported)
		}
	}
}

func TestShouldSwap(t *testing.T) {
	tests := []struct {
		name     string
		a        *MethodInfo
		b        *MethodInfo
		expected bool
	}{
		{
			name:     "different receivers - alphabetical order",
			a:        &MethodInfo{ReceiverName: "Client", IsExported: true, MaxDepth: 1, InDegree: 0, Position: 100},
			b:        &MethodInfo{ReceiverName: "Server", IsExported: true, MaxDepth: 1, InDegree: 0, Position: 200},
			expected: false, // Client comes before Server
		},
		{
			name:     "same receiver - exported before private",
			a:        &MethodInfo{ReceiverName: "Server", IsExported: false, MaxDepth: 1, InDegree: 0, Position: 100},
			b:        &MethodInfo{ReceiverName: "Server", IsExported: true, MaxDepth: 1, InDegree: 0, Position: 200},
			expected: true, // private should come after exported
		},
		{
			name:     "same receiver and export - lower depth first",
			a:        &MethodInfo{ReceiverName: "Server", IsExported: true, MaxDepth: 2, InDegree: 0, Position: 100},
			b:        &MethodInfo{ReceiverName: "Server", IsExported: true, MaxDepth: 1, InDegree: 0, Position: 200},
			expected: true, // higher depth should come after lower depth
		},
		{
			name:     "same receiver, export, depth - higher in-degree last",
			a:        &MethodInfo{ReceiverName: "Server", IsExported: true, MaxDepth: 1, InDegree: 0, Position: 100},
			b:        &MethodInfo{ReceiverName: "Server", IsExported: true, MaxDepth: 1, InDegree: 3, Position: 200},
			expected: true, // lower in-degree should come before higher in-degree
		},
		{
			name:     "all same - position fallback",
			a:        &MethodInfo{ReceiverName: "Server", IsExported: true, MaxDepth: 1, InDegree: 0, Position: 200},
			b:        &MethodInfo{ReceiverName: "Server", IsExported: true, MaxDepth: 1, InDegree: 0, Position: 100},
			expected: true, // higher position should come after lower position
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSwap(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("shouldSwap() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSortMethods(t *testing.T) {
	methods := []*MethodInfo{
		{Name: "helper", ReceiverName: "Server", IsExported: false, MaxDepth: 0, InDegree: 2, Position: 300},
		{Name: "Start", ReceiverName: "Server", IsExported: true, MaxDepth: 1, InDegree: 0, Position: 100},
		{Name: "Connect", ReceiverName: "Client", IsExported: true, MaxDepth: 1, InDegree: 0, Position: 200},
		{Name: "internal", ReceiverName: "Client", IsExported: false, MaxDepth: 0, InDegree: 1, Position: 400},
	}

	sorted := sortMethods(methods)

	expectedOrder := []string{"Connect", "internal", "Start", "helper"}
	for i, expected := range expectedOrder {
		if sorted[i].Name != expected {
			t.Errorf("Position %d: expected %s, got %s", i, expected, sorted[i].Name)
		}
	}
}
