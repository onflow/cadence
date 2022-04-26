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
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/tools/analysis"
)

// Reference to optional analyzer

var ReferenceToOptionalAnalyzer = (func() *analysis.Analyzer {

	elementFilter := []ast.Element{
		(*ast.ReferenceExpression)(nil),
	}

	return &analysis.Analyzer{
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
					referenceExpression, ok := element.(*ast.ReferenceExpression)
					if !ok {
						return
					}

					indexExpression, ok := referenceExpression.Expression.(*ast.IndexExpression)
					if !ok {
						return
					}

					indexedType := elaboration.IndexExpressionIndexedTypes[indexExpression]
					resultType := indexedType.ElementType(false)
					_, ok = resultType.(*sema.OptionalType)
					if !ok {
						return
					}

					report(
						analysis.Diagnostic{
							Location: location,
							Range:    ast.NewRangeFromPositioned(indexExpression),
							Message:  "reference to optional",
						},
					)
				},
			)

			return nil
		},
	}
})()

func init() {

	registerAnalyzer(
		"reference-to-optional",
		ReferenceToOptionalAnalyzer,
	)
}

// Deprecated key functions analyzer

var DeprecatedKeyFunctionsAnalyzer = (func() *analysis.Analyzer {

	elementFilter := []ast.Element{
		(*ast.InvocationExpression)(nil),
	}

	return &analysis.Analyzer{
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
					invocationExpression, ok := element.(*ast.InvocationExpression)
					if !ok {
						return
					}

					memberExpression, ok := invocationExpression.InvokedExpression.(*ast.MemberExpression)
					if !ok {
						return
					}

					memberInfo := elaboration.MemberExpressionMemberInfos[memberExpression]
					member := memberInfo.Member
					if member == nil {
						return
					}

					if member.ContainerType != sema.AuthAccountType {
						return
					}

					var details string
					switch member.Identifier.Identifier {
					case sema.AuthAccountAddPublicKeyField:
						details = fmt.Sprintf(
							"replace '%s' with '%s'",
							sema.AuthAccountAddPublicKeyField,
							"keys.add",
						)
					case sema.AuthAccountRemovePublicKeyField:
						details = fmt.Sprintf(
							"replace '%s' with '%s'",
							sema.AuthAccountRemovePublicKeyField,
							"keys.revoke",
						)
					default:
						return
					}

					report(
						analysis.Diagnostic{
							Location: location,
							Range:    ast.NewRangeFromPositioned(element),
							Message:  fmt.Sprintf("use of deprecated key management API: %s", details),
						},
					)
				},
			)

			return nil
		},
	}
})()

func init() {
	registerAnalyzer(
		"deprecated-key-functions",
		DeprecatedKeyFunctionsAnalyzer,
	)
}

// Number supertype arithmetic operations analyzer

var NumberSupertypeBinaryOperationsAnalyzer = (func() *analysis.Analyzer {

	elementFilter := []ast.Element{
		(*ast.BinaryExpression)(nil),
	}

	numberSupertypes := map[sema.Type]struct{}{
		sema.NumberType:           {},
		sema.SignedNumberType:     {},
		sema.IntegerType:          {},
		sema.SignedIntegerType:    {},
		sema.FixedPointType:       {},
		sema.SignedFixedPointType: {},
	}

	return &analysis.Analyzer{
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
					binaryExpression, ok := element.(*ast.BinaryExpression)
					if !ok {
						return
					}

					leftType := elaboration.BinaryExpressionLeftTypes[binaryExpression]

					if _, ok := numberSupertypes[leftType]; !ok {
						return
					}

					report(
						analysis.Diagnostic{
							Location: location,
							Range:    ast.NewRangeFromPositioned(element),
							Message:  "arithmetic operation on number supertype",
						},
					)
				},
			)

			return nil
		},
	}
})()

func init() {
	registerAnalyzer(
		"number-supertype-binary-operations",
		NumberSupertypeBinaryOperationsAnalyzer,
	)
}
