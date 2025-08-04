package analyzer

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/borovikovd/go-msort/pkg/sorter"
)

var Analyzer = &analysis.Analyzer{
	Name:     "msort",
	Doc:      "reports methods that are not optimally sorted for readability",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, nil
	}

	nodeFilter := []ast.Node{
		(*ast.File)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		file, ok := n.(*ast.File)
		if !ok {
			return
		}

		fset := pass.Fset
		methodSorter := sorter.New(fset, file)

		_, changed, err := methodSorter.Sort()
		if err != nil {
			return
		}

		if changed {
			pass.Reportf(file.Pos(), "methods in this file could be better sorted for readability")
		}
	})

	return nil, nil
}
