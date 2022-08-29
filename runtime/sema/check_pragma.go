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

package sema

import "github.com/onflow/cadence/runtime/ast"

func (checker *Checker) VisitPragmaDeclaration(p *ast.PragmaDeclaration) Type {

	invocPragma, isInvocPragma := p.Expression.(*ast.InvocationExpression)
	var isIdentPragma bool
	if !isInvocPragma {
		_, isIdentPragma = p.Expression.(*ast.IdentifierExpression)
	}

	// Pragma can be either an invocation expression or an identfier expression
	if !(isInvocPragma || isIdentPragma) {
		checker.report(&InvalidPragmaError{
			Message: "must be identifier or invocation expression",
			Range:   ast.NewRangeFromPositioned(checker.memoryGauge, p.Expression),
		})
	}

	if isInvocPragma {
		// Type arguments are not supported for pragmas
		if len(invocPragma.TypeArguments) > 0 {
			checker.report(&InvalidPragmaError{
				Message: "type arguments not supported",
				Range:   ast.NewRangeFromPositioned(checker.memoryGauge, invocPragma),
			})
		}
		// Ensure arguments are string expressions
		for _, arg := range invocPragma.Arguments {
			_, ok := arg.Expression.(*ast.StringExpression)
			if !ok {
				checker.report(&InvalidPragmaError{
					Message: "invalid argument",
					Range:   ast.NewRangeFromPositioned(checker.memoryGauge, invocPragma),
				})
			}
		}
	}

	return nil
}
