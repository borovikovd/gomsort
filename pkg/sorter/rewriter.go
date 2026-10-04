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

// reorderMethods gathers each type's methods, in sorted order, where its
// first method is. Declarations that sat between them, such as helper
// functions, follow the block in their order; nothing before a type's first
// method moves. It reports whether anything moved.
func (s *Sorter) reorderMethods(sorted []*MethodInfo) bool {
	receiver := make(map[dst.Decl]string, len(sorted))
	blocks := make(map[string][]dst.Decl)
	for _, method := range sorted {
		receiver[method.FuncDecl] = method.ReceiverName
		blocks[method.ReceiverName] = append(blocks[method.ReceiverName], method.FuncDecl)
	}

	decls := make([]dst.Decl, 0, len(s.file.Decls))
	for _, decl := range s.file.Decls {
		name, isMethod := receiver[decl]
		if !isMethod {
			decls = append(decls, decl)
			continue
		}
		if block, first := blocks[name]; first {
			decls = append(decls, block...)
			delete(blocks, name)
		}
	}

	changed := !slices.Equal(decls, s.file.Decls)
	s.file.Decls = decls
	return changed
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
