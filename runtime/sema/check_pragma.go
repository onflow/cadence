/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package sema

import "github.com/onflow/cadence/runtime/ast"

func (checker *Checker) VisitPragmaDeclaration(p *ast.PragmaDeclaration) (_ struct{}) {

	switch e := p.Expression.(type) {
	case *ast.InvocationExpression:
		// Type arguments are not supported for pragmas
		if len(e.TypeArguments) > 0 {
			checker.report(&InvalidPragmaError{
				Message: "type arguments not supported",
				Range:   ast.NewRangeFromPositioned(checker.memoryGauge, e),
			})
		}

		// Ensure arguments are string expressions
		for _, arg := range e.Arguments {
			_, ok := arg.Expression.(*ast.StringExpression)
			if !ok {
				checker.report(&InvalidPragmaError{
					Message: "invalid non-string argument",
					Range:   ast.NewRangeFromPositioned(checker.memoryGauge, e),
				})
			}
		}

	case *ast.IdentifierExpression:
		// valid, NO-OP

	default:
		checker.report(&InvalidPragmaError{
			Message: "pragma must be identifier or invocation expression",
			Range:   ast.NewRangeFromPositioned(checker.memoryGauge, p.Expression),
		})
	}

	return
}
