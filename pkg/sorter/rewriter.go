package sorter

import (
	"bytes"
	"os"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

type Sorter struct {
	source string
	file   *dst.File
}

func NewFromSource(source string) (*Sorter, error) {
	file, err := decorator.Parse(source)
	if err != nil {
		return nil, err
	}

	return &Sorter{
		source: source,
		file:   file,
	}, nil
}

func WriteFile(filename string, content []byte) error {
	return os.WriteFile(filename, content, 0644)
}

func (s *Sorter) Sort() ([]byte, bool, error) {
	callGraph := buildCallGraph(s.file)
	methods := callGraph.GetMethods()

	if len(methods) == 0 {
		// No methods to sort, just return formatted source
		var buf bytes.Buffer
		if err := decorator.Fprint(&buf, s.file); err != nil {
			return nil, false, err
		}
		return buf.Bytes(), false, nil
	}

	sortedMethods := sortMethods(methods)

	changed := s.hasOrderChanged(methods, sortedMethods)
	if !changed {
		// No changes needed, return formatted source
		var buf bytes.Buffer
		if err := decorator.Fprint(&buf, s.file); err != nil {
			return nil, false, err
		}
		return buf.Bytes(), false, nil
	}

	// Reorder methods in DST - decorations will move automatically
	s.reorderMethods(sortedMethods)

	// Format with DST
	var buf bytes.Buffer
	if err := decorator.Fprint(&buf, s.file); err != nil {
		return nil, true, err
	}

	return buf.Bytes(), true, nil
}

func (s *Sorter) hasOrderChanged(original, sorted []*MethodInfo) bool {
	if len(original) != len(sorted) {
		return true
	}

	for i, method := range original {
		if method.Position != sorted[i].Position {
			return true
		}
	}

	return false
}

func (s *Sorter) reorderMethods(sortedMethods []*MethodInfo) {
	// Create method lookup map
	methodMap := make(map[*dst.FuncDecl]bool)
	for _, method := range sortedMethods {
		methodMap[method.FuncDecl] = true
	}

	// Collect non-method declarations first
	newDecls := make([]dst.Decl, 0, len(s.file.Decls))
	for _, decl := range s.file.Decls {
		if funcDecl, ok := decl.(*dst.FuncDecl); ok {
			// Skip methods - we'll add them in sorted order
			if methodMap[funcDecl] {
				continue
			}
		}
		newDecls = append(newDecls, decl)
	}

	// Add sorted methods - their decorations (comments) will move with them automatically
	for _, method := range sortedMethods {
		newDecls = append(newDecls, method.FuncDecl)
	}

	// Update the DST file with reordered declarations
	s.file.Decls = newDecls
}
