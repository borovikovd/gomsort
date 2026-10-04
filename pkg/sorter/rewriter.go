package sorter

import (
	"bytes"
	"os"
	"regexp"
	"slices"
	"strings"

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

// Sort reorders each type's methods and reports whether any moved. Generated
// files are left as they are: the next generation would undo the order.
func (s *Sorter) Sort() ([]byte, bool, error) {
	changed := false
	if !isGenerated(s.source) {
		changed = s.reorderMethods(sortMethods(buildCallGraph(s.file).GetMethods()))
	}

	var buf bytes.Buffer
	if err := decorator.Fprint(&buf, s.file); err != nil {
		return nil, changed, err
	}
	return buf.Bytes(), changed, nil
}

// reorderMethods lays out each type's methods the way the Uber Go style
// guide orders a file. Where the type's first method is go, in order: the
// const, var and type declarations that sat between its methods, the type's
// constructors (newT or NewT returning T) from anywhere after that point,
// and the methods in sorted order. Functions that sat between them follow
// the block in their order. Nothing before a type's first method moves. It
// reports whether anything moved.
func (s *Sorter) reorderMethods(sorted []*MethodInfo) bool {
	receiver := make(map[dst.Decl]string, len(sorted))
	methods := make(map[string][]dst.Decl)
	last := make(map[string]int)
	for _, method := range sorted {
		receiver[method.FuncDecl] = method.ReceiverName
		methods[method.ReceiverName] = append(methods[method.ReceiverName], method.FuncDecl)
	}
	for i, decl := range s.file.Decls {
		if name, ok := receiver[decl]; ok {
			last[name] = i
		}
	}

	moved := make(map[dst.Decl]bool)
	decls := make([]dst.Decl, 0, len(s.file.Decls))
	for i, decl := range s.file.Decls {
		if moved[decl] {
			continue
		}
		name, isMethod := receiver[decl]
		if !isMethod {
			decls = append(decls, decl)
			continue
		}
		block, first := methods[name]
		if !first {
			continue
		}
		// A type with one method has nothing between its methods.
		for _, later := range s.file.Decls[i+1 : max(i+1, last[name])] {
			if _, isGen := later.(*dst.GenDecl); isGen && !moved[later] {
				decls = append(decls, later)
				moved[later] = true
			}
		}
		for _, later := range s.file.Decls[i+1:] {
			if isConstructor(later, name) && !moved[later] {
				decls = append(decls, later)
				moved[later] = true
			}
		}
		decls = append(decls, block...)
		delete(methods, name)
	}

	changed := !slices.Equal(decls, s.file.Decls)
	s.file.Decls = decls
	return changed
}

// isConstructor reports whether decl is a function named newT or NewT,
// whatever follows, whose first result is typeName or a pointer to it.
func isConstructor(decl dst.Decl, typeName string) bool {
	fn, ok := decl.(*dst.FuncDecl)
	if !ok || fn.Recv != nil || fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return false
	}
	if !strings.HasPrefix(fn.Name.Name, "new") && !strings.HasPrefix(fn.Name.Name, "New") {
		return false
	}
	return baseName(fn.Type.Results.List[0].Type) == typeName
}

// generatedMarker is Go's marker for generated files:
// https://go.dev/s/generatedcode.
var generatedMarker = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

// isGenerated reports whether source has the generated-code marker before
// its package clause.
func isGenerated(source string) bool {
	for _, line := range strings.Split(source, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if strings.HasPrefix(line, "package ") {
			return false
		}
		if generatedMarker.MatchString(line) {
			return true
		}
	}
	return false
}
