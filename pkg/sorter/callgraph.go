package sorter

import (
	"go/ast"
	"go/token"
	"strings"
)

type CallGraph struct {
	methods   map[string]*MethodInfo
	calls     map[string][]string
	positions map[string]token.Pos
}

func NewCallGraph() *CallGraph {
	return &CallGraph{
		methods:   make(map[string]*MethodInfo),
		calls:     make(map[string][]string),
		positions: make(map[string]token.Pos),
	}
}

func (cg *CallGraph) AddMethod(method *MethodInfo) {
	key := methodKey(method.ReceiverName, method.Name)
	cg.methods[key] = method
	cg.positions[key] = method.Position
	if cg.calls[key] == nil {
		cg.calls[key] = []string{}
	}
}

func (cg *CallGraph) AddCall(fromReceiver, fromMethod, toReceiver, toMethod string) {
	fromKey := methodKey(fromReceiver, fromMethod)
	toKey := methodKey(toReceiver, toMethod)

	if cg.calls[fromKey] == nil {
		cg.calls[fromKey] = []string{}
	}

	for _, existing := range cg.calls[fromKey] {
		if existing == toKey {
			return
		}
	}

	cg.calls[fromKey] = append(cg.calls[fromKey], toKey)
}

func (cg *CallGraph) CalculateMetrics() {
	inDegree := make(map[string]int)

	for _, calls := range cg.calls {
		for _, target := range calls {
			inDegree[target]++
		}
	}

	maxDepth := make(map[string]int)
	visited := make(map[string]bool)

	var calculateDepth func(string) int
	calculateDepth = func(method string) int {
		if visited[method] {
			return 0
		}

		if depth, exists := maxDepth[method]; exists {
			return depth
		}

		visited[method] = true
		defer func() { visited[method] = false }()

		depth := 0
		for _, target := range cg.calls[method] {
			targetDepth := calculateDepth(target)
			if targetDepth+1 > depth {
				depth = targetDepth + 1
			}
		}

		maxDepth[method] = depth
		return depth
	}

	for key := range cg.methods {
		calculateDepth(key)
	}

	for key, method := range cg.methods {
		method.InDegree = inDegree[key]
		method.MaxDepth = maxDepth[key]
	}
}

func (cg *CallGraph) GetMethods() []*MethodInfo {
	methods := make([]*MethodInfo, 0, len(cg.methods))
	for _, method := range cg.methods {
		methods = append(methods, method)
	}
	return methods
}

func buildCallGraph(file *ast.File) *CallGraph {
	cg := NewCallGraph()

	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if method := extractMethodInfo(funcDecl); method != nil {
				cg.AddMethod(method)
			}
		}
	}

	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if method := extractMethodInfo(funcDecl); method != nil {
				visitor := &callVisitor{
					callGraph:       cg,
					currentReceiver: method.ReceiverName,
					currentMethod:   method.Name,
				}
				ast.Walk(visitor, funcDecl.Body)
			}
		}
	}

	cg.CalculateMetrics()
	return cg
}

type callVisitor struct {
	callGraph       *CallGraph
	currentReceiver string
	currentMethod   string
}

func (v *callVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.CallExpr:
		if sel, ok := n.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				if ident.Name == "self" || ident.Name == v.currentReceiver ||
					(len(ident.Name) == 1 && strings.EqualFold(ident.Name[0:1], v.currentReceiver[0:1])) {
					v.callGraph.AddCall(v.currentReceiver, v.currentMethod, v.currentReceiver, sel.Sel.Name)
				}
			}
		}
	}
	return v
}

func methodKey(receiver, method string) string {
	return receiver + "." + method
}
