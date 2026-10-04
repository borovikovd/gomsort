package sorter

import (
	"slices"
	"sort"

	"github.com/dave/dst"
)

type CallGraph struct {
	methods map[string]*MethodInfo
	calls   map[string]map[string]bool // caller → the methods it uses, without itself
}

func NewCallGraph() *CallGraph {
	return &CallGraph{
		methods: make(map[string]*MethodInfo),
		calls:   make(map[string]map[string]bool),
	}
}

func buildCallGraph(file *dst.File) *CallGraph {
	cg := NewCallGraph()

	var methods []*MethodInfo
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*dst.FuncDecl); ok {
			if method := extractMethodInfo(funcDecl, len(methods)); method != nil {
				cg.AddMethod(method)
				methods = append(methods, method)
			}
		}
	}

	// A method uses another when it calls it or passes it on as a value
	// through its receiver: s.connect() or http.HandleFunc("/", s.serve).
	// Fields can't share a method's name, so AddCall ignores them.
	for _, method := range methods {
		if method.ReceiverVar == "" || method.FuncDecl.Body == nil {
			continue
		}
		dst.Inspect(method.FuncDecl.Body, func(n dst.Node) bool {
			if sel, ok := n.(*dst.SelectorExpr); ok {
				if ident, ok := sel.X.(*dst.Ident); ok && ident.Name == method.ReceiverVar {
					cg.AddCall(method.ReceiverName, method.Name, method.ReceiverName, sel.Sel.Name)
				}
			}
			return true
		})
	}

	cg.CalculateMetrics()
	return cg
}

func methodKey(receiver, method string) string {
	return receiver + "." + method
}

func (cg *CallGraph) AddMethod(method *MethodInfo) {
	cg.methods[methodKey(method.ReceiverName, method.Name)] = method
}

// AddCall records that one method uses another. A method calling itself says
// nothing about where it belongs, so recursion is left out.
func (cg *CallGraph) AddCall(fromReceiver, fromMethod, toReceiver, toMethod string) {
	from, to := methodKey(fromReceiver, fromMethod), methodKey(toReceiver, toMethod)
	if from == to {
		return
	}
	if _, exists := cg.methods[to]; !exists {
		return
	}
	if cg.calls[from] == nil {
		cg.calls[from] = make(map[string]bool)
	}
	cg.calls[from][to] = true
}

// CalculateMetrics sets each method's InDegree, the distinct methods that use
// it, and MaxDepth, the longest chain of calls reaching it from a method
// nothing calls. Methods that call each other share a depth.
func (cg *CallGraph) CalculateMetrics() {
	keys := cg.sortedKeys()
	for _, key := range keys {
		cg.methods[key].InDegree = 0
		cg.methods[key].MaxDepth = 0
	}
	for _, caller := range keys {
		for callee := range cg.calls[caller] {
			cg.methods[callee].InDegree++
		}
	}

	components := cg.components(keys)
	component := make(map[string]int, len(keys))
	for i, members := range components {
		for _, key := range members {
			component[key] = i
		}
	}
	// Components come callers first, so each one's depth is final by the
	// time it passes it on.
	depth := make([]int, len(components))
	for i, members := range components {
		for _, key := range members {
			for callee := range cg.calls[key] {
				if j := component[callee]; j != i {
					depth[j] = max(depth[j], depth[i]+1)
				}
			}
		}
	}
	for i, members := range components {
		for _, key := range members {
			cg.methods[key].MaxDepth = depth[i]
		}
	}
}

func (cg *CallGraph) GetMethods() []*MethodInfo {
	methods := make([]*MethodInfo, 0, len(cg.methods))
	for _, key := range cg.sortedKeys() {
		methods = append(methods, cg.methods[key])
	}
	sort.Slice(methods, func(i, j int) bool {
		return methods[i].Position < methods[j].Position
	})
	return methods
}

// components returns the call graph's strongly connected components (Tarjan's
// algorithm) in topological order: every component comes before the ones it
// calls into.
func (cg *CallGraph) components(keys []string) [][]string {
	var (
		next    int
		index   = make(map[string]int, len(keys))
		low     = make(map[string]int, len(keys))
		onStack = make(map[string]bool, len(keys))
		stack   []string
		out     [][]string
		visit   func(string)
	)
	visit = func(v string) {
		index[v], low[v] = next, next
		next++
		stack = append(stack, v)
		onStack[v] = true
		callees := make([]string, 0, len(cg.calls[v]))
		for w := range cg.calls[v] {
			callees = append(callees, w)
		}
		sort.Strings(callees)
		for _, w := range callees {
			if _, seen := index[w]; !seen {
				visit(w)
				low[v] = min(low[v], low[w])
			} else if onStack[w] {
				low[v] = min(low[v], index[w])
			}
		}
		if low[v] == index[v] {
			var members []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				members = append(members, w)
				if w == v {
					break
				}
			}
			out = append(out, members)
		}
	}
	for _, key := range keys {
		if _, seen := index[key]; !seen {
			visit(key)
		}
	}
	// Tarjan's algorithm finishes a component after everything it calls.
	slices.Reverse(out)
	return out
}

func (cg *CallGraph) sortedKeys() []string {
	keys := make([]string, 0, len(cg.methods))
	for key := range cg.methods {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
