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

	sorted := make([]*MethodInfo, 0, len(methods))
	placed := make(map[*MethodInfo]bool, len(methods))
	var place, placeHelpers func(m *MethodInfo)
	place = func(m *MethodInfo) {
		placed[m] = true
		sorted = append(sorted, m)
		placeHelpers(m)
	}
	placeHelpers = func(m *MethodInfo) {
		for _, callee := range m.Callees {
			if !placed[callee] && !callee.IsEntryPoint() {
				place(callee)
			}
		}
	}
	for _, key := range runs {
		group := byRun[key]
		var exported []*MethodInfo
		for _, m := range group {
			if m.IsExported {
				placed[m] = true
				sorted = append(sorted, m)
				exported = append(exported, m)
			}
		}
		for _, m := range exported {
			placeHelpers(m)
		}
		for _, m := range group {
			if m.IsEntryPoint() && !placed[m] {
				place(m)
			}
		}
		// Helpers only other helpers use, in a cycle, keep their order.
		for _, m := range group {
			if !placed[m] {
				place(m)
			}
		}
	}
	return sorted
}
