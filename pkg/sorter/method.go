package sorter

import (
	"go/ast"
	"go/token"
	"sort"
)

type MethodInfo struct {
	Name         string
	ReceiverName string // the receiver's type name, without * or type parameters
	ReceiverType string // the receiver's type as written, e.g. *Server
	ReceiverVar  string // the receiver's variable name, or "" when unnamed
	IsExported   bool
	FuncDecl     *ast.FuncDecl
	Position     int
	Run          int           // which run of consecutive methods of its type, in file order, it's in
	Callees      []*MethodInfo // methods of its run it uses, in order of first use
	Callers      int           // methods of its run that use it
	UsedOutside  bool          // used from outside its run: by a function, another type, or another run

	uses map[*MethodInfo]bool // Callees, for looking up
}

// IsEntryPoint reports whether the method starts a group of its own rather
// than following a method that uses it: it's exported, used from outside its
// type, or not used by its type's other methods in this file.
func (m *MethodInfo) IsEntryPoint() bool {
	return m.IsExported || m.UsedOutside || m.Callers == 0
}

func extractMethodInfo(decl *ast.FuncDecl, position int) *MethodInfo {
	if decl.Recv == nil || len(decl.Recv.List) == 0 {
		return nil
	}

	method := &MethodInfo{
		Name:       decl.Name.Name,
		IsExported: token.IsExported(decl.Name.Name),
		FuncDecl:   decl,
		Position:   position,
	}

	recv := decl.Recv.List[0]
	if len(recv.Names) > 0 && recv.Names[0].Name != "_" {
		method.ReceiverVar = recv.Names[0].Name
	}
	if name := baseName(recv.Type); name != "" {
		method.ReceiverName = name
		method.ReceiverType = name
		if _, pointer := recv.Type.(*ast.StarExpr); pointer {
			method.ReceiverType = "*" + name
		}
	}

	return method
}

// sortMethods orders each run of a type's methods the way Go code usually
// reads: exported methods first, in their current order, then the rest in
// call order. That is, the helpers the exported methods use, in the order
// they first use them, then the other unexported entry points in their
// current order, each followed by its helpers, depth first. A helper several
// methods use follows the first of them. Runs come in file order.
func sortMethods(methods []*MethodInfo) []*MethodInfo {
	byPosition := make([]*MethodInfo, len(methods))
	copy(byPosition, methods)
	sort.SliceStable(byPosition, func(i, j int) bool { return byPosition[i].Position < byPosition[j].Position })

	type runKey struct {
		receiver string
		run      int
	}
	var runs []runKey
	byRun := make(map[runKey][]*MethodInfo)
	for _, m := range byPosition {
		key := runKey{m.ReceiverName, m.Run}
		if _, seen := byRun[key]; !seen {
			runs = append(runs, key)
		}
		byRun[key] = append(byRun[key], m)
	}

	p := &placer{placed: make(map[*MethodInfo]bool, len(methods))}
	for _, key := range runs {
		p.placeRun(byRun[key])
	}
	return p.sorted
}

// placer appends methods in their sorted order.
type placer struct {
	sorted []*MethodInfo
	placed map[*MethodInfo]bool
}

func (p *placer) placeRun(group []*MethodInfo) {
	var exported []*MethodInfo
	for _, m := range group {
		if m.IsExported {
			p.placed[m] = true
			p.sorted = append(p.sorted, m)
			exported = append(exported, m)
		}
	}
	for _, m := range exported {
		p.placeHelpers(m)
	}
	for _, m := range group {
		if m.IsEntryPoint() && !p.placed[m] {
			p.place(m)
		}
	}
	// What's left is used only within the run, recursively: its entry is in
	// another run. Start from the recursive groups nothing else left uses,
	// so their helpers still follow them.
	roots := p.leftoverRoots(group)
	for _, m := range group {
		if !p.placed[m] && roots[m] {
			p.place(m)
		}
	}
	// Every method left is reachable from a root; this keeps any that isn't
	// rather than drop it.
	for _, m := range group {
		if !p.placed[m] {
			p.place(m)
		}
	}
}

func (p *placer) placeHelpers(m *MethodInfo) {
	for _, callee := range m.Callees {
		if !p.placed[callee] && !callee.IsEntryPoint() {
			p.place(callee)
		}
	}
}

func (p *placer) place(m *MethodInfo) {
	p.placed[m] = true
	p.sorted = append(p.sorted, m)
	p.placeHelpers(m)
}

// leftoverRoots returns the methods of group not yet placed that are in a
// recursive group no other unplaced method uses.
func (p *placer) leftoverRoots(group []*MethodInfo) map[*MethodInfo]bool {
	component := p.recursiveGroups(group)
	used := make(map[int]bool)
	for _, v := range group {
		for _, w := range v.Callees {
			if !p.placed[v] && !p.placed[w] && component[w] != component[v] {
				used[component[w]] = true
			}
		}
	}
	roots := make(map[*MethodInfo]bool)
	for _, m := range group {
		if !p.placed[m] && !used[component[m]] {
			roots[m] = true
		}
	}
	return roots
}

// recursiveGroups numbers the strongly connected components among the
// methods of group not yet placed, with Tarjan's algorithm, in time linear
// in the run: methods that reach each other share a number.
func (p *placer) recursiveGroups(group []*MethodInfo) map[*MethodInfo]int {
	var (
		next      int
		index     = make(map[*MethodInfo]int)
		low       = make(map[*MethodInfo]int)
		onStack   = make(map[*MethodInfo]bool)
		component = make(map[*MethodInfo]int)
		stack     []*MethodInfo
		visit     func(v *MethodInfo)
	)
	visit = func(v *MethodInfo) {
		index[v], low[v] = next, next
		next++
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range v.Callees {
			if p.placed[w] {
				continue
			}
			if _, seen := index[w]; !seen {
				visit(w)
				low[v] = min(low[v], low[w])
			} else if onStack[w] {
				low[v] = min(low[v], index[w])
			}
		}
		if low[v] != index[v] {
			return
		}
		for {
			w := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[w] = false
			component[w] = index[v]
			if w == v {
				return
			}
		}
	}
	for _, m := range group {
		if _, seen := index[m]; !seen && !p.placed[m] {
			visit(m)
		}
	}
	return component
}
