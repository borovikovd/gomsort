package sorter

import (
	"bytes"
	"os"
	"regexp"
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

// reorderMethods puts each type's methods, in sorted order, into the places
// that type's methods already hold, so types, functions and other
// declarations stay where they are. It reports whether any method moved.
func (s *Sorter) reorderMethods(sorted []*MethodInfo) bool {
	receiver := make(map[*dst.FuncDecl]string, len(sorted))
	byReceiver := make(map[string][]*dst.FuncDecl)
	for _, method := range sorted {
		receiver[method.FuncDecl] = method.ReceiverName
		byReceiver[method.ReceiverName] = append(byReceiver[method.ReceiverName], method.FuncDecl)
	}

	changed := false
	next := make(map[string]int)
	for i, decl := range s.file.Decls {
		funcDecl, ok := decl.(*dst.FuncDecl)
		if !ok {
			continue
		}
		name, ok := receiver[funcDecl]
		if !ok {
			continue
		}
		replacement := byReceiver[name][next[name]]
		next[name]++
		if replacement != funcDecl {
			s.file.Decls[i] = replacement
			changed = true
		}
	}
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
