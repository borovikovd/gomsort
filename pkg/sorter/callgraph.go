package sorter

import (
	"sort"
	"strings"

	"github.com/dave/dst"
)

type CallGraph struct {
	methods   map[string]*MethodInfo
	calls     map[string][]string
	positions map[string]int
}

func NewCallGraph() *CallGraph {
	return &CallGraph{
		methods:   make(map[string]*MethodInfo),
		calls:     make(map[string][]string),
		positions: make(map[string]int),
	}
}

func buildCallGraph(file *dst.File) *CallGraph {
	cg := NewCallGraph()

	// First pass: collect all methods
	position := 0
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*dst.FuncDecl); ok {
			if method := extractMethodInfo(funcDecl, position); method != nil {
				cg.AddMethod(method)
				position++
			}
		}
	}

	// Second pass: analyze method calls
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*dst.FuncDecl); ok {
			if method := extractMethodInfo(funcDecl, 0); method != nil && funcDecl.Body != nil {
				visitor := &callVisitor{
					callGraph:       cg,
					currentReceiver: method.ReceiverName,
					currentMethod:   method.Name,
				}
				dst.Walk(visitor, funcDecl.Body)
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

func methodKey(receiver, method string) string {
	return receiver + "." + method
}

func (cg *CallGraph) AddMethod(method *MethodInfo) {
	key := methodKey(method.ReceiverName, method.Name)
	cg.methods[key] = method
	cg.positions[key] = method.Position
}

func (cg *CallGraph) AddCall(fromReceiver, fromMethod, toReceiver, toMethod string) {
	fromKey := methodKey(fromReceiver, fromMethod)
	toKey := methodKey(toReceiver, toMethod)

	if _, exists := cg.methods[toKey]; exists {
		cg.calls[fromKey] = append(cg.calls[fromKey], toKey)
	}
}

func (cg *CallGraph) CalculateMetrics() {
	// Calculate in-degree for each method
	inDegree := make(map[string]int)

	// Get keys in deterministic order to avoid race conditions
	keys := make([]string, 0, len(cg.methods))
	for key := range cg.methods {
		keys = append(keys, key)
	}
	// Sort keys to ensure deterministic order
	sort.Strings(keys)

	for _, key := range keys {
		if calls, exists := cg.calls[key]; exists {
			for _, calledMethod := range calls {
				inDegree[calledMethod]++
			}
		}
	}

	// Calculate max depth for each method using DFS
	for _, key := range keys {
		visited := make(map[string]bool)
		cg.methods[key].MaxDepth = cg.calculateMaxDepth(key, visited)
		cg.methods[key].InDegree = inDegree[key]
	}
}

func (cg *CallGraph) GetMethods() []*MethodInfo {
	methods := make([]*MethodInfo, 0, len(cg.methods))

	// Get keys in deterministic order to avoid race conditions
	keys := make([]string, 0, len(cg.methods))
	for key := range cg.methods {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Sort by original position to ensure deterministic order
	for _, key := range keys {
		methods = append(methods, cg.methods[key])
	}

	// Sort by original position to ensure deterministic order
	sort.Slice(methods, func(i, j int) bool {
		return methods[i].Position < methods[j].Position
	})

	return methods
}

func (cg *CallGraph) calculateMaxDepth(methodKey string, visited map[string]bool) int {
	if visited[methodKey] {
		return 0
	}

	visited[methodKey] = true
	maxDepth := 0

	if calls, exists := cg.calls[methodKey]; exists {
		for _, calledMethod := range calls {
			depth := cg.calculateMaxDepth(calledMethod, visited)
			if depth+1 > maxDepth {
				maxDepth = depth + 1
			}
		}
	}

	delete(visited, methodKey)
	return maxDepth
}

func (v *callVisitor) Visit(node dst.Node) dst.Visitor {
	switch n := node.(type) {
	case *dst.CallExpr:
		if sel, ok := n.Fun.(*dst.SelectorExpr); ok {
			if ident, ok := sel.X.(*dst.Ident); ok {
				if ident.Name == "self" || ident.Name == v.currentReceiver ||
					(len(ident.Name) == 1 && len(v.currentReceiver) > 0 &&
						strings.EqualFold(ident.Name[0:1], v.currentReceiver[0:1])) {
					v.callGraph.AddCall(v.currentReceiver, v.currentMethod, v.currentReceiver, sel.Sel.Name)
				}
			}
		}
	}
	return v
}
