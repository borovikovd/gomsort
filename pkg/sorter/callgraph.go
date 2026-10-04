package sorter

import (
	"go/token"
	"sort"

	"github.com/dave/dst"
)

// CallGraph records, for the methods in one file, which methods of the same
// type each one uses, and which are used from outside their type: by a
// function or by another type's methods.
type CallGraph struct {
	methods map[string]*MethodInfo
}

func NewCallGraph() *CallGraph {
	return &CallGraph{methods: make(map[string]*MethodInfo)}
}

func buildCallGraph(file *dst.File) *CallGraph {
	cg := NewCallGraph()

	position := 0
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*dst.FuncDecl); ok {
			if method := extractMethodInfo(funcDecl, position); method != nil {
				cg.AddMethod(method)
				position++
			}
		}
	}

	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*dst.FuncDecl); ok && funcDecl.Body != nil {
			cg.addUses(funcDecl)
		}
	}
	return cg
}

func methodKey(receiver, method string) string {
	return receiver + "." + method
}

func (cg *CallGraph) AddMethod(method *MethodInfo) {
	cg.methods[methodKey(method.ReceiverName, method.Name)] = method
}

// AddCall records that a function or method uses a method. fromReceiver is
// "" for a function. A method using itself says nothing about where it
// belongs, so recursion is left out.
func (cg *CallGraph) AddCall(fromReceiver, fromMethod, toReceiver, toMethod string) {
	to, ok := cg.methods[methodKey(toReceiver, toMethod)]
	if !ok || (fromReceiver == toReceiver && fromMethod == toMethod) {
		return
	}
	if fromReceiver != toReceiver {
		to.UsedOutside = true
		return
	}
	from := cg.methods[methodKey(fromReceiver, fromMethod)]
	if from == nil || containsMethod(from.Callees, to) {
		return
	}
	from.Callees = append(from.Callees, to)
	to.Callers++
}

func (cg *CallGraph) GetMethods() []*MethodInfo {
	methods := make([]*MethodInfo, 0, len(cg.methods))
	for _, method := range cg.methods {
		methods = append(methods, method)
	}
	sort.Slice(methods, func(i, j int) bool {
		return methods[i].Position < methods[j].Position
	})
	return methods
}

// addUses records every method decl uses, in the order it first uses them:
// calls such as g.add(c), and method values passed on such as run(s.serve).
func (cg *CallGraph) addUses(decl *dst.FuncDecl) {
	fromReceiver, fromName := "", decl.Name.Name
	vars := make(varTypes)
	if method := extractMethodInfo(decl, 0); method != nil {
		fromReceiver = method.ReceiverName
	}
	if decl.Recv != nil {
		vars.declareFields(decl.Recv)
	}
	vars.declareFields(decl.Type.Params)
	vars.declareFields(decl.Type.Results)

	dst.Inspect(decl.Body, func(n dst.Node) bool {
		vars.learn(n)
		if sel, ok := n.(*dst.SelectorExpr); ok {
			if typeName := baseName(vars.typeOf(sel.X)); typeName != "" {
				cg.AddCall(fromReceiver, fromName, typeName, sel.Sel.Name)
			}
		}
		return true
	})
}

func containsMethod(list []*MethodInfo, m *MethodInfo) bool {
	for _, x := range list {
		if x == m {
			return true
		}
	}
	return false
}

// varTypes holds the type each variable in a function was declared with, as
// far as the file writes it out. Scopes are flattened: a later declaration
// of the same name replaces an earlier one.
type varTypes map[string]dst.Expr

// learn records the variables n declares: x := T{...}, var x T, the value
// in for _, x := range xs, and a function literal's parameters.
func (v varTypes) learn(n dst.Node) {
	switch n := n.(type) {
	case *dst.AssignStmt:
		if n.Tok == token.DEFINE && len(n.Lhs) == len(n.Rhs) {
			for i, lhs := range n.Lhs {
				if ident, ok := lhs.(*dst.Ident); ok {
					v.declare(ident.Name, v.typeOf(n.Rhs[i]))
				}
			}
		}
	case *dst.ValueSpec:
		for i, name := range n.Names {
			switch {
			case n.Type != nil:
				v.declare(name.Name, n.Type)
			case len(n.Values) == len(n.Names):
				v.declare(name.Name, v.typeOf(n.Values[i]))
			}
		}
	case *dst.RangeStmt:
		if value, ok := n.Value.(*dst.Ident); ok && n.Tok == token.DEFINE {
			v.declare(value.Name, elementType(v.typeOf(n.X)))
		}
	case *dst.FuncLit:
		v.declareFields(n.Type.Params)
	}
}

func (v varTypes) declare(name string, typ dst.Expr) {
	if typ == nil {
		delete(v, name)
		return
	}
	v[name] = typ
}

// typeOf returns the type expression of x's value when the file shows it: a
// variable's declared type, T{...}, &T{...}, new(T) or *p.
func (v varTypes) typeOf(x dst.Expr) dst.Expr {
	switch x := x.(type) {
	case *dst.Ident:
		return v[x.Name]
	case *dst.ParenExpr:
		return v.typeOf(x.X)
	case *dst.CompositeLit:
		return x.Type
	case *dst.UnaryExpr:
		if x.Op == token.AND {
			if typ := v.typeOf(x.X); typ != nil {
				return &dst.StarExpr{X: typ}
			}
		}
	case *dst.StarExpr:
		if star, ok := v.typeOf(x.X).(*dst.StarExpr); ok {
			return star.X
		}
	case *dst.CallExpr:
		if ident, ok := x.Fun.(*dst.Ident); ok && ident.Name == "new" && len(x.Args) == 1 {
			return &dst.StarExpr{X: x.Args[0]}
		}
	}
	return nil
}

func (v varTypes) declareFields(fields *dst.FieldList) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		typ := field.Type
		if ellipsis, ok := typ.(*dst.Ellipsis); ok {
			typ = &dst.ArrayType{Elt: ellipsis.Elt}
		}
		for _, name := range field.Names {
			v.declare(name.Name, typ)
		}
	}
}

// elementType returns the element type of a slice, array or map type.
func elementType(typ dst.Expr) dst.Expr {
	switch typ := typ.(type) {
	case *dst.ArrayType:
		return typ.Elt
	case *dst.MapType:
		return typ.Value
	}
	return nil
}

// baseName returns the name of the named type typ refers to: T for T, *T,
// T[A] and *T[A, B].
func baseName(typ dst.Expr) string {
	switch typ := typ.(type) {
	case *dst.Ident:
		return typ.Name
	case *dst.StarExpr:
		return baseName(typ.X)
	case *dst.IndexExpr:
		return baseName(typ.X)
	case *dst.IndexListExpr:
		return baseName(typ.X)
	case *dst.ParenExpr:
		return baseName(typ.X)
	}
	return ""
}
