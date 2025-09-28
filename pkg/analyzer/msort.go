package analyzer

import (
	"bytes"
	"go/ast"
	"go/format"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/borovikovd/gomsort/pkg/sorter"
)

var Analyzer = &analysis.Analyzer{
	Name:     "msort",
	Doc:      "reports methods that are not optimally sorted for readability",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	if pass == nil {
		return nil, nil
	}

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

		// Convert AST to source code
		var buf bytes.Buffer
		if err := format.Node(&buf, pass.Fset, file); err != nil {
			return
		}

		// Use DST-based sorter
		methodSorter, err := sorter.NewFromSource(buf.String())
		if err != nil {
			return
		}

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
