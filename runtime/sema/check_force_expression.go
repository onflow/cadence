/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

import (
	"github.com/onflow/cadence/runtime/ast"
)

func (checker *Checker) VisitForceExpression(expression *ast.ForceExpression) ast.Repr {

	// Expected type of the `expression.Expression` is the optional of expected type of current context.
	// i.e: if `x!` is `string, then `x` is expected to be `string?`.
	expectedType := checker.expectedType
	if expectedType != nil {
		expectedType = &OptionalType{Type: expectedType}
	}

	valueType := checker.VisitExpression(expression.Expression, expectedType)

	if valueType.IsInvalidType() {
		return valueType
	}

	checker.recordResourceInvalidation(
		expression.Expression,
		valueType,
		ResourceInvalidationKindMoveDefinite,
	)

	optionalType, ok := valueType.(*OptionalType)
	if !ok {

		// A non-optional type is forced. Suggest removing it

		checker.hint(
			&RemovalHint{
				Description: "unnecessary force operator",
				Range: ast.Range{
					StartPos: expression.EndPos,
					EndPos:   expression.EndPos,
				},
			},
		)

		return valueType
	}

	return optionalType.Type
}
