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

// VisitPragmaDeclaration checks that the pragma declaration is valid.
// It is valid if the root expression is an identifier or invocation.
// Invocations must
func (checker *Checker) VisitPragmaDeclaration(declaration *ast.PragmaDeclaration) (_ struct{}) {

	switch expression := declaration.Expression.(type) {
	case *ast.IdentifierExpression:
		// valid, NO-OP

	case *ast.InvocationExpression:
		checker.checkPragmaInvocationExpression(expression)

	default:
		checker.report(&InvalidPragmaError{
			Message: "expression must be literal, identifier, or invocation",
			Range: ast.NewRangeFromPositioned(
				checker.memoryGauge,
				expression,
			),
		})
	}

	return
}

func (checker *Checker) checkPragmaInvocationExpression(expression *ast.InvocationExpression) {
	// Invoked expression must be an identifier
	if _, ok := expression.InvokedExpression.(*ast.IdentifierExpression); !ok {
		checker.report(&InvalidPragmaError{
			Message: "invoked expression must be an identifier",
			Range: ast.NewRangeFromPositioned(
				checker.memoryGauge,
				expression.InvokedExpression,
			),
		})
	}

	// Type arguments are not supported for pragmas
	if len(expression.TypeArguments) > 0 {
		checker.report(&InvalidPragmaError{
			Message: "type arguments are not supported",
			Range: ast.NewRangeFromPositioned(
				checker.memoryGauge,
				expression.TypeArguments[0],
			),
		})
	}

	// Ensure arguments are valid
	for _, arg := range expression.Arguments {
		checker.checkPragmaArgumentExpression(arg.Expression)
	}
}

func (checker *Checker) checkPragmaArgumentExpression(expression ast.Expression) {
	switch expression := expression.(type) {
	case *ast.InvocationExpression:
		checker.checkPragmaInvocationExpression(expression)
		return

	case *ast.StringExpression,
		*ast.IntegerExpression,
		*ast.FixedPointExpression,
		*ast.ArrayExpression,
		*ast.DictionaryExpression,
		*ast.NilExpression,
		*ast.BoolExpression,
		*ast.PathExpression:

		return

	case *ast.UnaryExpression:
		if expression.Operation == ast.OperationMinus {
			return
		}
	}

	checker.report(&InvalidPragmaError{
		Message: "expression in invocation must be literal or invocation",
		Range: ast.NewRangeFromPositioned(
			checker.memoryGauge,
			expression,
		),
	})
}
