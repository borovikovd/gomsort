package sorter

import (
	"bytes"
	"go/token"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

type Sorter struct {
	source string
	file   *dst.File // nil for a generated file, which isn't parsed
}

// NewFromSource parses source, unless it's generated: generated files are
// left as they are, since the next generation would undo the order, and they
// can be large enough that parsing them would dominate memory.
func NewFromSource(source string) (*Sorter, error) {
	if isGenerated(source) {
		return &Sorter{source: source}, nil
	}
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

// Sort reorders each type's methods and reports whether anything moved. An
// unchanged file comes back as it was given.
func (s *Sorter) Sort() ([]byte, bool, error) {
	if s.file == nil || !s.reorderMethods(sortMethods(buildCallGraph(s.file).GetMethods())) {
		return []byte(s.source), false, nil
	}

	var buf bytes.Buffer
	if err := decorator.Fprint(&buf, s.file); err != nil {
		return nil, true, err
	}
	return buf.Bytes(), true, nil
}

// reorderMethods lays out each type's methods the way the Uber Go style
// guide orders a file. A type's methods gather where the first of them after
// the type's declaration is, or where its first method is when the type is
// declared in another file or only has methods above its declaration. There
// go, in order: the const, var and type declarations that sat between that
// point and its last method, the type's constructors (newT or NewT returning
// T) from anywhere after that point, and the methods in sorted order.
// Functions that sat between the methods follow the block. Nothing else
// moves. It reports whether anything moved.
func (s *Sorter) reorderMethods(sorted []*MethodInfo) bool {
	receiver := make(map[dst.Decl]string, len(sorted))
	methods := make(map[string][]dst.Decl)
	for _, method := range sorted {
		receiver[method.FuncDecl] = method.ReceiverName
		methods[method.ReceiverName] = append(methods[method.ReceiverName], method.FuncDecl)
	}
	l := newLayout(s.file.Decls, receiver)

	decls := make([]dst.Decl, 0, len(s.file.Decls))
	for i, decl := range s.file.Decls {
		name, isMethod := receiver[decl]
		switch {
		case l.moved[decl]:
		case !isMethod:
			decls = append(decls, decl)
		case i == l.anchor[name]:
			decls = append(decls, l.leading(i, name)...)
			decls = append(decls, methods[name]...)
		}
	}

	changed := !slices.Equal(decls, s.file.Decls)
	s.file.Decls = decls
	return changed
}

// layout indexes a file's declarations once, so placing each type's block
// costs only what goes into it: the whole layout is linear in the file.
type layout struct {
	decls        []dst.Decl
	anchor, last map[string]int   // per type: the method its block replaces, and its last method
	gens         []int            // const, var and type declarations, in order
	constructors map[string][]int // per type, in order
	moved        map[dst.Decl]bool
}

func newLayout(decls []dst.Decl, receiver map[dst.Decl]string) *layout {
	l := &layout{
		decls:        decls,
		anchor:       make(map[string]int),
		last:         make(map[string]int),
		constructors: make(map[string][]int),
		moved:        make(map[dst.Decl]bool),
	}
	declared := make(map[string]int)
	for i, decl := range decls {
		if gen, ok := decl.(*dst.GenDecl); ok {
			l.gens = append(l.gens, i)
			if gen.Tok == token.TYPE {
				for _, spec := range gen.Specs {
					if typeSpec, ok := spec.(*dst.TypeSpec); ok {
						declared[typeSpec.Name.Name] = i
					}
				}
			}
		}
		if typeName := constructorOf(decl); typeName != "" {
			l.constructors[typeName] = append(l.constructors[typeName], i)
		}
		name, ok := receiver[decl]
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
func (l *layout) leading(at int, typeName string) []dst.Decl {
	var out []dst.Decl
	take := func(i int) {
		if decl := l.decls[i]; !l.moved[decl] {
			out = append(out, decl)
			l.moved[decl] = true
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
func constructorOf(decl dst.Decl) string {
	fn, ok := decl.(*dst.FuncDecl)
	if !ok || fn.Recv != nil || fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return ""
	}
	if !strings.HasPrefix(fn.Name.Name, "new") && !strings.HasPrefix(fn.Name.Name, "New") {
		return ""
	}
	return baseName(fn.Type.Results.List[0].Type)
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
