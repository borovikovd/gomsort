package sorter

import (
	"go/ast"
	"sort"
)

// CallGraph records, for the methods in one file, which methods of the same
// run each one uses, and which are used from outside their run: by a
// function, another type's methods, or the same type's methods elsewhere in
// the file. A run is a type's methods one after another, with no other
// declaration between them; gomsort sorts each run on its own, so nothing
// moves past another declaration.
type CallGraph struct {
	methods map[string]*MethodInfo
}

func NewCallGraph() *CallGraph {
	return &CallGraph{methods: make(map[string]*MethodInfo)}
}

func buildCallGraph(file *ast.File) *CallGraph {
	cg := NewCallGraph()

	position, run, previous := 0, 0, ""
	for _, decl := range file.Decls {
		var method *MethodInfo
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			method = extractMethodInfo(funcDecl, position)
		}
		if method == nil {
			run++
			previous = ""
			continue
		}
		if method.ReceiverName != previous {
			run++
			previous = method.ReceiverName
		}
		method.Run = run
		cg.AddMethod(method)
		position++
	}

	usedOutside := make(map[string]bool)
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Body != nil {
			cg.addUses(funcDecl, usedOutside)
		}
	}
	for _, method := range cg.methods {
		if usedOutside[method.Name] {
			method.UsedOutside = true
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
	if from == nil {
		return
	}
	if from.Run != to.Run {
		to.UsedOutside = true
		return
	}
	if from.uses[to] {
		return
	}
	if from.uses == nil {
		from.uses = make(map[*MethodInfo]bool)
	}
	from.uses[to] = true
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

// addUses records the methods decl uses through its receiver, in the order
// it first uses them: calls such as s.connect(), and method values passed on
// such as run(s.serve). Any other x.name it marks in usedOutside, as a use of
// every method called name from outside its type. Matching by name alone may
// take a different type's method, or a field, for one of ours; that only
// keeps the method where it is.
func (cg *CallGraph) addUses(decl *ast.FuncDecl, usedOutside map[string]bool) {
	method := extractMethodInfo(decl, 0)
	ast.Inspect(decl.Body, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if ident, ok := sel.X.(*ast.Ident); ok && method != nil && method.ReceiverVar != "" && ident.Name == method.ReceiverVar {
			cg.AddCall(method.ReceiverName, method.Name, method.ReceiverName, sel.Sel.Name)
		} else {
			usedOutside[sel.Sel.Name] = true
		}
		return true
	})
}

// baseName returns the name of the named type typ refers to: T for T, *T,
// T[A] and *T[A, B].
func baseName(typ ast.Expr) string {
	switch typ := typ.(type) {
	case *ast.Ident:
		return typ.Name
	case *ast.StarExpr:
		return baseName(typ.X)
	case *ast.IndexExpr:
		return baseName(typ.X)
	case *ast.IndexListExpr:
		return baseName(typ.X)
	case *ast.ParenExpr:
		return baseName(typ.X)
	}
	return ""
}
