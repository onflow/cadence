package main

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "maprangecheck",
	Doc:      "reports range statements over maps",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	in, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("failed to get result of inspect analysis")
	}

	in.Preorder(
		[]ast.Node{
			(*ast.RangeStmt)(nil),
		},
		func(node ast.Node) {
			rangeStmt, ok := node.(*ast.RangeStmt)
			if !ok {
				return
			}

			ty := pass.TypesInfo.TypeOf(rangeStmt.X).Underlying()
			if ty == nil {
				return
			}

			mapTy, ok := ty.(*types.Map)
			if !ok {
				return
			}

			pass.Reportf(
				rangeStmt.X.Pos(),
				"range statement over map: %v",
				mapTy,
			)
		},
	)

	return nil, nil
}
