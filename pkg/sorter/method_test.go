package sorter

import (
	"strings"
	"testing"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

func TestExtractMethodInfo(t *testing.T) {
	source := `
package test

type Server struct{}

func (s *Server) PublicMethod() {}
func (s *Server) privateMethod() {}
func (s Server) ValueReceiver() {}
func NotAMethod() {}
`

	file, err := decorator.Parse(source)
	if err != nil {
		t.Fatal(err)
	}

	var methods []*MethodInfo
	position := 0
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*dst.FuncDecl); ok {
			if method := extractMethodInfo(funcDecl, position); method != nil {
				methods = append(methods, method)
				position++
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

func TestSortMethodsStepsDown(t *testing.T) {
	// Exported entry points first, then unexported ones; each followed by the
	// helpers it uses in the order it first uses them; a shared helper
	// follows its first user; methods used from outside their type are entry
	// points. Positions are the current order.
	run := &MethodInfo{Name: "run", ReceiverName: "S", Position: 0}
	helper := &MethodInfo{Name: "helper", ReceiverName: "S", Position: 1, Callers: 2}
	Start := &MethodInfo{Name: "Start", ReceiverName: "S", IsExported: true, Position: 2}
	parse := &MethodInfo{Name: "parse", ReceiverName: "S", Position: 3, Callers: 1}
	add := &MethodInfo{Name: "add", ReceiverName: "S", Position: 4, Callers: 1, UsedOutside: true}
	Start.Callees = []*MethodInfo{parse, helper}
	parse.Callees = []*MethodInfo{add}
	run.Callees = []*MethodInfo{helper}

	var got []string
	for _, m := range sortMethods([]*MethodInfo{run, helper, Start, parse, add}) {
		got = append(got, m.Name)
	}
	want := []string{"Start", "parse", "helper", "run", "add"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("got %v, want %v", got, want)
	}
}
