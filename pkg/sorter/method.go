package sorter

import (
	"go/token"
	"sort"

	"github.com/dave/dst"
)

type MethodInfo struct {
	Name         string
	ReceiverName string // the receiver's type name, without * or type parameters
	ReceiverType string // the receiver's type as written, e.g. *Server
	ReceiverVar  string // the receiver's variable name, or "" when unnamed
	IsExported   bool
	FuncDecl     *dst.FuncDecl
	Position     int
	Callees      []*MethodInfo // methods of the same type it uses, in order of first use
	Callers      int           // methods of the same type that use it
	UsedOutside  bool          // used by a function or another type's method in the file
}

// IsEntryPoint reports whether the method starts a group of its own rather
// than following a method that uses it: it's exported, used from outside its
// type, or not used by its type's other methods in this file.
func (m *MethodInfo) IsEntryPoint() bool {
	return m.IsExported || m.UsedOutside || m.Callers == 0
}

func extractMethodInfo(decl *dst.FuncDecl, position int) *MethodInfo {
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
		if _, pointer := recv.Type.(*dst.StarExpr); pointer {
			method.ReceiverType = "*" + name
		}
	}

	return method
}

// sortMethods orders each type's methods top-down, the way Clean Code's
// stepdown rule reads: entry points, exported first and otherwise in their
// current order, each followed by the helpers it uses, in the order it first
// uses them, and theirs in turn. A helper several methods use follows the
// first of them. Types come in the order of their first method.
func sortMethods(methods []*MethodInfo) []*MethodInfo {
	byPosition := make([]*MethodInfo, len(methods))
	copy(byPosition, methods)
	sort.SliceStable(byPosition, func(i, j int) bool { return byPosition[i].Position < byPosition[j].Position })

	var receivers []string
	byReceiver := make(map[string][]*MethodInfo)
	for _, m := range byPosition {
		if _, seen := byReceiver[m.ReceiverName]; !seen {
			receivers = append(receivers, m.ReceiverName)
		}
		byReceiver[m.ReceiverName] = append(byReceiver[m.ReceiverName], m)
	}

	sorted := make([]*MethodInfo, 0, len(methods))
	placed := make(map[*MethodInfo]bool, len(methods))
	var place func(m *MethodInfo)
	place = func(m *MethodInfo) {
		placed[m] = true
		sorted = append(sorted, m)
		for _, callee := range m.Callees {
			if !placed[callee] && !callee.IsEntryPoint() {
				place(callee)
			}
		}
	}
	for _, receiver := range receivers {
		group := byReceiver[receiver]
		for _, exported := range []bool{true, false} {
			for _, m := range group {
				if m.IsExported == exported && m.IsEntryPoint() && !placed[m] {
					place(m)
				}
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
