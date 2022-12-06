/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
	Name:     "maprange",
	Doc:      "reports range statements over maps",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
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
