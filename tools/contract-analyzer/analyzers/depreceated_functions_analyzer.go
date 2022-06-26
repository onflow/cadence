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

var DeprecatedKeyFunctionsAnalyzer = (func() *analysis.Analyzer {

	elementFilter := []ast.Element{
		(*ast.InvocationExpression)(nil),
	}

	return &analysis.Analyzer{
		Description: "Detects usages of the deprecated key management API. It will be removed in a future release.",
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

					var replacement string
					functionName := member.Identifier.Identifier
					switch functionName {
					case sema.AuthAccountAddPublicKeyField:
						replacement = "keys.add"
					case sema.AuthAccountRemovePublicKeyField:
						replacement = "keys.revoke"
					default:
						return
					}

					report(
						analysis.Diagnostic{
							Location: location,
							Range:    ast.NewRangeFromPositioned(nil, memberExpression.Identifier),
							Category: "update recommended",
							Message: fmt.Sprintf(
								"deprecated function '%s' will get removed",
								functionName,
							),
							SecondaryMessage: fmt.Sprintf(
								"replace with '%s'",
								replacement,
							),
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
