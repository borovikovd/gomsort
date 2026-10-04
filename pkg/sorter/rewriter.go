package sorter

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"slices"
	"strings"
)

type Sorter struct {
	source string
	fset   *token.FileSet
	file   *ast.File // nil when there's nothing to sort, and source isn't parsed
}

// NewFromSource parses source, unless there's nothing to sort: no line
// starts a method declaration, or the file is generated, which is left as
// it is since the next generation would undo the order. Parsing dominates
// the time, and generated files can be large enough to dominate memory.
func NewFromSource(source string) (*Sorter, error) {
	if !hasMethods(source) || isGenerated(source) {
		return &Sorter{source: source}, nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", source, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	return &Sorter{
		source: source,
		fset:   fset,
		file:   file,
	}, nil
}

func WriteFile(filename string, content []byte) error {
	return os.WriteFile(filename, content, 0644)
}

// fileText splits a file's source by top-level declaration: the header up
// to the end of the package clause's line, each declaration's text from the
// end of the previous one's last line to the end of its own, which takes in
// its doc comment and a comment after it on its last line, and what follows
// the last declaration.
type fileText struct {
	header string
	decls  []string
	tail   string
}

// Sort reorders each type's methods and reports whether anything moved. It
// moves each declaration's text as it is, with the comments and blank lines
// before it, and formats the result with gofmt. An unchanged file comes back
// as it was given.
func (s *Sorter) Sort() ([]byte, bool, error) {
	if s.file == nil {
		return []byte(s.source), false, nil
	}
	order := s.newOrder(sortMethods(buildCallGraph(s.file).GetMethods()))
	text, ok := s.declText()
	if order == nil || !ok {
		return []byte(s.source), false, nil
	}

	var out strings.Builder
	out.WriteString(text.header)
	for k, i := range order {
		chunk := text.decls[i]
		// A declaration that has a new one above it gets one blank line, as
		// gofmt'd code has; declarations that stay together keep their spacing.
		if k == 0 && i != 0 || k > 0 && order[k-1] != i-1 {
			chunk = "\n" + strings.TrimLeft(chunk, "\n")
		}
		out.WriteString(chunk)
	}
	out.WriteString(text.tail)

	formatted, err := format.Source([]byte(out.String()))
	if err != nil {
		return nil, true, err
	}
	return formatted, true, nil
}

// newOrder returns the declarations' indexes with each run of a type's
// methods in sorted order, or nil when nothing moves. A run's methods take
// the places the run already holds, so nothing moves past another
// declaration.
func (s *Sorter) newOrder(sorted []*MethodInfo) []int {
	index := make(map[ast.Decl]int, len(s.file.Decls))
	for i, decl := range s.file.Decls {
		index[decl] = i
	}
	order := make([]int, len(s.file.Decls))
	for i := range order {
		order[i] = i
	}
	type runKey struct {
		receiver string
		run      int
	}
	runs := make(map[runKey][]int)
	var keys []runKey
	for _, method := range sorted {
		key := runKey{method.ReceiverName, method.Run}
		if _, seen := runs[key]; !seen {
			keys = append(keys, key)
		}
		runs[key] = append(runs[key], index[method.FuncDecl])
	}
	for _, key := range keys {
		places := slices.Sorted(slices.Values(runs[key]))
		for k, i := range runs[key] {
			order[places[k]] = i
		}
	}
	if slices.IsSorted(order) {
		return nil
	}
	return order
}

// declText splits the source, or reports false when a declaration shares a
// line with the one before it, which gofmt never writes.
func (s *Sorter) declText() (fileText, bool) {
	tf := s.fset.File(s.file.Pos())
	lineEnd := func(pos token.Pos) int {
		offset := tf.Offset(pos)
		if i := strings.IndexByte(s.source[offset:], '\n'); i >= 0 {
			return offset + i + 1
		}
		return len(s.source)
	}

	start := lineEnd(s.file.Name.End())
	text := fileText{header: s.source[:start]}
	for _, decl := range s.file.Decls {
		begin := decl.Pos()
		if doc := docOf(decl); doc != nil {
			begin = doc.Pos()
		}
		if tf.Offset(begin) < start {
			return fileText{}, false
		}
		end := lineEnd(decl.End())
		text.decls = append(text.decls, s.source[start:end])
		start = end
	}
	text.tail = s.source[start:]
	return text, true
}

func docOf(decl ast.Decl) *ast.CommentGroup {
	switch decl := decl.(type) {
	case *ast.FuncDecl:
		return decl.Doc
	case *ast.GenDecl:
		return decl.Doc
	}
	return nil
}

// hasMethods reports whether a line of source starts a method declaration,
// as gofmt writes one: "func (". A file written otherwise isn't sorted.
func hasMethods(source string) bool {
	return strings.HasPrefix(source, "func (") || strings.Contains(source, "\nfunc (")
}

// generatedMarker is Go's marker for generated files:
// https://go.dev/s/generatedcode.
var generatedMarker = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

// isGenerated reports whether source has the generated-code marker before
// its package clause. It reads only that far.
func isGenerated(source string) bool {
	for rest := source; rest != ""; {
		var line string
		line, rest, _ = strings.Cut(rest, "\n")
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
