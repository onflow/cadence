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
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/tools/analysis"
)

func ReplacementHint(
	expr ast.Expression,
	location common.Location,
	r ast.Range,
) *analysis.Diagnostic {
	return &analysis.Diagnostic{
		Location: location,
		Range:    r,
		Category: "lint",
		Message:  fmt.Sprintf("consider replacing with: `%s`", expr),
	}
}

func suggestIntegerLiteralConversionReplacement(
	argument *ast.IntegerExpression,
	location common.Location,
	targetType sema.Type,
	invocationRange ast.Range,
) *analysis.Diagnostic {
	negative := argument.Value.Sign() < 0

	if sema.IsSameTypeKind(targetType, sema.FixedPointType) {

		// If the integer literal is converted to a fixed-point type,
		// suggest replacing it with a fixed-point literal

		signed := sema.IsSameTypeKind(targetType, sema.SignedFixedPointType)

		var hintExpression ast.Expression = ast.NewFixedPointExpression(
			nil,
			"",
			negative,
			common.NewBigIntFromAbsoluteValue(nil, argument.Value),
			common.NewBigInt(nil),
			1,
			argument.Range,
		)

		// If the fixed-point literal is positive
		// and the target fixed-point type is signed,
		// then a static cast is required

		if !negative && signed {
			hintExpression = ast.NewCastingExpression(
				nil,
				hintExpression,
				ast.OperationCast,
				ast.NewTypeAnnotation(
					nil,
					false,
					ast.NewNominalType(
						nil,
						ast.NewIdentifier(
							nil,
							targetType.String(),
							ast.EmptyPosition,
						),
						nil,
					),
					ast.EmptyPosition,
				),
				nil,
			)
		}

		return ReplacementHint(hintExpression, location, invocationRange)

	} else if sema.IsSameTypeKind(targetType, sema.IntegerType) {

		// If the integer literal is converted to an integer type,
		// suggest replacing it with a fixed-point literal

		var hintExpression ast.Expression = argument

		// If the target type is not `Int`,
		// then a static cast is required,
		// as all integer literals (positive and negative)
		// are inferred to be of type `Int`

		if !sema.IsSameTypeKind(targetType, sema.IntType) {
			hintExpression = ast.NewCastingExpression(
				nil,
				hintExpression,
				ast.OperationCast,
				ast.NewTypeAnnotation(
					nil,
					false,
					ast.NewNominalType(
						nil,
						ast.NewIdentifier(
							nil,
							targetType.String(),
							ast.EmptyPosition,
						),
						nil,
					),
					ast.EmptyPosition,
				),
				nil,
			)
		}

		return ReplacementHint(hintExpression, location, invocationRange)
	}

	return nil
}

func suggestFixedPointLiteralConversionReplacement(
	argument *ast.FixedPointExpression,
	location common.Location,
	targetType sema.Type,
	invocationRange ast.Range,
) *analysis.Diagnostic {
	// If the fixed-point literal is converted to a fixed-point type,
	// suggest replacing it with a fixed-point literal

	if !sema.IsSameTypeKind(targetType, sema.FixedPointType) {
		return nil
	}

	negative := argument.Negative
	signed := sema.IsSameTypeKind(targetType, sema.SignedFixedPointType)

	if (!negative && !signed) || (negative && signed) {
		return ReplacementHint(argument, location, invocationRange)
	}

	return nil
}

var NumberFunctionArgumentAnalyzer = (func() *analysis.Analyzer {

	elementFilter := []ast.Element{
		(*ast.IntegerExpression)(nil),
		(*ast.FixedPointExpression)(nil),
	}

	return &analysis.Analyzer{
		Description: "Detects redundant uses of number conversion functions.",
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

					var diagnostic *analysis.Diagnostic

					switch expr := element.(type) {
					case *ast.IntegerExpression:
						argumentData, ok := elaboration.NumberConversionArgumentTypes[expr]
						if !ok {
							return
						}
						diagnostic = suggestIntegerLiteralConversionReplacement(expr, location, argumentData.Type, argumentData.Range)
					case *ast.FixedPointExpression:
						argumentData, ok := elaboration.NumberConversionArgumentTypes[expr]
						if !ok {
							return
						}
						diagnostic = suggestFixedPointLiteralConversionReplacement(expr, location, argumentData.Type, argumentData.Range)
					default:
						return
					}

					if diagnostic != nil {
						report(*diagnostic)
					}
				},
			)

			return nil
		},
	}
})()

func init() {
	registerAnalyzer(
		"number-function-argument-analyzer",
		NumberFunctionArgumentAnalyzer,
	)
}
