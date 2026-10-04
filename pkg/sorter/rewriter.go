package sorter

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"slices"
	"sort"
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

// newOrder lays out each type's methods the way the Uber Go style guide
// orders a file, and returns the declarations' indexes in their new order,
// or nil when nothing moves. A type's methods gather where the first of them
// after the type's declaration is, or where its first method is when the
// type is declared in another file or only has methods above its
// declaration. There go, in order: the const, var and type declarations that
// sat between that point and its last method, the type's constructors (newT
// or NewT returning T) from anywhere after that point, and the methods in
// sorted order. Functions that sat between the methods follow the block.
// Nothing else moves.
func (s *Sorter) newOrder(sorted []*MethodInfo) []int {
	decls := s.file.Decls
	index := make(map[ast.Decl]int, len(decls))
	for i, decl := range decls {
		index[decl] = i
	}
	receiver := make(map[int]string, len(sorted))
	methods := make(map[string][]int)
	for _, method := range sorted {
		i := index[method.FuncDecl]
		receiver[i] = method.ReceiverName
		methods[method.ReceiverName] = append(methods[method.ReceiverName], i)
	}
	l := newLayout(decls, receiver)

	order := make([]int, 0, len(decls))
	for i := range decls {
		name, isMethod := receiver[i]
		switch {
		case l.moved[i]:
		case !isMethod:
			order = append(order, i)
		case i == l.anchor[name]:
			order = append(order, l.leading(i, name)...)
			order = append(order, methods[name]...)
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

// layout indexes a file's declarations once, so placing each type's block
// costs only what goes into it: the whole layout is linear in the file.
type layout struct {
	anchor, last map[string]int   // per type: the method its block replaces, and its last method
	gens         []int            // const, var and type declarations that may move, in order
	constructors map[string][]int // per type, in order
	moved        map[int]bool
}

func newLayout(decls []ast.Decl, receiver map[int]string) *layout {
	l := &layout{
		anchor:       make(map[string]int),
		last:         make(map[string]int),
		constructors: make(map[string][]int),
		moved:        make(map[int]bool),
	}
	hasMethods := make(map[string]bool)
	for _, name := range receiver {
		hasMethods[name] = true
	}
	declared := make(map[string]int)
	for i, decl := range decls {
		if gen, ok := decl.(*ast.GenDecl); ok {
			// A type with methods here anchors its own block, so it stays
			// where it is rather than moving away from them.
			anchorsBlock := false
			if gen.Tok == token.TYPE {
				for _, spec := range gen.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						declared[typeSpec.Name.Name] = i
						anchorsBlock = anchorsBlock || hasMethods[typeSpec.Name.Name]
					}
				}
			}
			if !anchorsBlock {
				l.gens = append(l.gens, i)
			}
		}
		if typeName := constructorOf(decl); typeName != "" {
			l.constructors[typeName] = append(l.constructors[typeName], i)
		}
		name, ok := receiver[i]
		if !ok {
			continue
		}
		at, seen := l.anchor[name]
		typeAt, isDeclared := declared[name]
		if !seen || (isDeclared && i > typeAt && at < typeAt) {
			l.anchor[name] = i
		}
		l.last[name] = i
	}
	return l
}

// leading returns what goes before the block of a type's methods placed at
// index at: the const, var and type declarations between there and its last
// method, then its constructors from anywhere after there. It marks them
// moved.
func (l *layout) leading(at int, typeName string) []int {
	var out []int
	take := func(i int) {
		if !l.moved[i] {
			out = append(out, i)
			l.moved[i] = true
		}
	}
	for k := sort.SearchInts(l.gens, at+1); k < len(l.gens) && l.gens[k] < l.last[typeName]; k++ {
		take(l.gens[k])
	}
	for _, i := range l.constructors[typeName] {
		if i > at {
			take(i)
		}
	}
	return out
}

// constructorOf returns T when decl is a function named newT or NewT,
// whatever follows, whose first result is T or a pointer to it, and ""
// otherwise.
func constructorOf(decl ast.Decl) string {
	fn, ok := decl.(*ast.FuncDecl)
	if !ok || fn.Recv != nil || fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return ""
	}
	if !strings.HasPrefix(fn.Name.Name, "new") && !strings.HasPrefix(fn.Name.Name, "New") {
		return ""
	}
	return baseName(fn.Type.Results.List[0].Type)
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
