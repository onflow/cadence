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

package analyzers

import (
	"fmt"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/tools/analysis"
)

var CastAnalyzer = (func() *analysis.Analyzer {

	elementFilter := []ast.Element{
		(*ast.CastingExpression)(nil),
	}

	return &analysis.Analyzer{
		Description: "Detects unnecessary cast expressions",
		Requires: []*analysis.Analyzer{
			analysis.InspectorAnalyzer,
		},
		Run: func(pass *analysis.Pass) interface{} {
			inspector := pass.ResultOf[analysis.InspectorAnalyzer].(*ast.Inspector)

			location := pass.Program.Location
			elaboration := pass.Program.Elaboration
			report := pass.Report

			inspector.Preorder(
				elementFilter,
				func(element ast.Element) {

					castingExpression, ok := element.(*ast.CastingExpression)
					if !ok {
						return
					}

					redundantType, ok := elaboration.RedundantCastTypes[castingExpression]
					if ok {
						report(
							analysis.Diagnostic{
								Location: location,
								Range:    ast.NewRangeFromPositioned(nil, castingExpression.TypeAnnotation),
								Category: "lint",
								Message:  fmt.Sprintf("cast to `%s` is redundant", redundantType),
							},
						)
						return
					}

					alwaysSucceedingTypes, ok := elaboration.AlwaysSucceedingCastTypes[castingExpression]
					if ok {
						switch castingExpression.Operation {
						case ast.OperationFailableCast:
							report(
								analysis.Diagnostic{
									Location: location,
									Range:    ast.NewRangeFromPositioned(nil, castingExpression),
									Category: "lint",
									Message: fmt.Sprintf("failable cast ('%s') from `%s` to `%s` always succeeds",
										ast.OperationFailableCast.Symbol(),
										alwaysSucceedingTypes.Left,
										alwaysSucceedingTypes.Right),
								},
							)
						case ast.OperationForceCast:
							report(
								analysis.Diagnostic{
									Location: location,
									Range:    ast.NewRangeFromPositioned(nil, castingExpression),
									Category: "lint",
									Message: fmt.Sprintf("force cast ('%s') from `%s` to `%s` always succeeds",
										ast.OperationForceCast.Symbol(),
										alwaysSucceedingTypes.Left,
										alwaysSucceedingTypes.Right),
								},
							)
						default:
							panic(errors.NewUnreachableError())
						}
					}
				},
			)

			return nil
		},
	}
})()

func init() {
	registerAnalyzer(
		"cast-analysis",
		CastAnalyzer,
	)
}
