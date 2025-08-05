package sorter

import (
	"strings"

	"github.com/dave/dst"
)

type MethodInfo struct {
	Name         string
	ReceiverName string
	ReceiverType string
	IsExported   bool
	FuncDecl     *dst.FuncDecl
	Position     int
	InDegree     int
	MaxDepth     int
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
		IsExported: isExported(decl.Name.Name),
		FuncDecl:   decl,
		Position:   position,
	}

	recv := decl.Recv.List[0]

	switch recvType := recv.Type.(type) {
	case *dst.Ident:
		method.ReceiverType = recvType.Name
		method.ReceiverName = recvType.Name
	case *dst.StarExpr:
		if ident, ok := recvType.X.(*dst.Ident); ok {
			method.ReceiverType = "*" + ident.Name
			method.ReceiverName = ident.Name
		}
	}

	return method
}

// Helper function since DST doesn't have ast.IsExported
func isExported(name string) bool {
	return len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z'
}

func sortMethods(methods []*MethodInfo) []*MethodInfo {
	sorted := make([]*MethodInfo, len(methods))
	copy(sorted, methods)

	// Use bubble sort for consistency with existing implementation
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if shouldSwap(sorted[j], sorted[j+1]) {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted
}

func shouldSwap(a, b *MethodInfo) bool {
	keyA := a.SortKey()
	keyB := b.SortKey()

	if keyA.ReceiverName != keyB.ReceiverName {
		return strings.Compare(keyA.ReceiverName, keyB.ReceiverName) > 0
	}

	if keyA.IsExported != keyB.IsExported {
		return !keyA.IsExported
	}

	if keyA.MaxDepth != keyB.MaxDepth {
		return keyA.MaxDepth > keyB.MaxDepth
	}

	if keyA.InDegree != keyB.InDegree {
		return keyA.InDegree < keyB.InDegree
	}

	return keyA.OriginalPos > keyB.OriginalPos
}
