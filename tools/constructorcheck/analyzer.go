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
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "constructorcheck",
	Doc:      "reports value composite literals that should be replaced with constructor calls",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	in, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("failed to get result of inspect analysis")
	}

	in.Nodes(
		[]ast.Node{
			(*ast.GenDecl)(nil),
			(*ast.CompositeLit)(nil),
		},
		func(node ast.Node, push bool) bool {
			// Only consider traversal down
			if !push {
				return true
			}

			// Ignore declarations like `var _ Value = XValue{}`,
			// which are used to assert that the concrete type on the RHS (here `XValue`)
			// implements the Value interface

			genDecl, ok := node.(*ast.GenDecl)
			if ok && genDecl.Tok == token.VAR && len(genDecl.Specs) == 1 {
				spec := genDecl.Specs[0].(*ast.ValueSpec)
				if len(spec.Names) == 1 && spec.Names[0].Name == "_" {
					return false
				}
			}

			// Consider composite literals which have an identifier ending in `Value`,
			// e.g. `XValue{ /* ... */ }`

			compositeLit, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}

			if compositeLit.Type == nil || !isValueType(compositeLit.Type) {
				return true
			}

			ty := pass.TypesInfo.TypeOf(compositeLit).Underlying()
			if ty == nil {
				return true
			}

			_, ok = ty.(*types.Struct)
			if !ok {
				return true
			}

			pass.Reportf(
				compositeLit.Pos(),
				"value composite literal should be constructor function call",
			)

			return true
		},
	)

	return nil, nil
}

func isValueType(t ast.Expr) bool {
	switch t := t.(type) {
	case *ast.Ident:
		return strings.HasSuffix(t.Name, "Value")
	case *ast.SelectorExpr:
		return isValueType(t.Sel)
	default:
		return false
	}
}
