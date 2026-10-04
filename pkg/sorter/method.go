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
	InDegree     int // distinct methods of the same type that call this one
	MaxDepth     int // longest chain of calls from an entry point to this one
}

type MethodSortKey struct {
	ReceiverName string
	IsExported   bool
	InDegree     int
	MaxDepth     int
	OriginalPos  int
}

func (m *MethodInfo) SortKey() MethodSortKey {
	return MethodSortKey{
		ReceiverName: m.ReceiverName,
		IsExported:   m.IsExported,
		InDegree:     m.InDegree,
		MaxDepth:     m.MaxDepth,
		OriginalPos:  m.Position,
	}
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

	typ, pointer := recv.Type, false
	if star, ok := typ.(*dst.StarExpr); ok {
		typ, pointer = star.X, true
	}
	// A generic type's receiver lists its type parameters: Set[T], Map[K, V].
	switch generic := typ.(type) {
	case *dst.IndexExpr:
		typ = generic.X
	case *dst.IndexListExpr:
		typ = generic.X
	}
	if ident, ok := typ.(*dst.Ident); ok {
		method.ReceiverName = ident.Name
		method.ReceiverType = ident.Name
		if pointer {
			method.ReceiverType = "*" + ident.Name
		}
	}

	return method
}

// sortMethods orders methods by receiver type, then exported first, then
// entry points before the helpers they call, then shared helpers last, then
// by original position.
func sortMethods(methods []*MethodInfo) []*MethodInfo {
	sorted := make([]*MethodInfo, len(methods))
	copy(sorted, methods)
	sort.SliceStable(sorted, func(i, j int) bool { return less(sorted[i], sorted[j]) })
	return sorted
}

func less(a, b *MethodInfo) bool {
	if a.ReceiverName != b.ReceiverName {
		return a.ReceiverName < b.ReceiverName
	}
	if a.IsExported != b.IsExported {
		return a.IsExported
	}
	if a.MaxDepth != b.MaxDepth {
		return a.MaxDepth < b.MaxDepth
	}
	if a.InDegree != b.InDegree {
		return a.InDegree < b.InDegree
	}
	return a.Position < b.Position
}

// shouldSwap reports whether b belongs before a.
func shouldSwap(a, b *MethodInfo) bool {
	return less(b, a)
}
