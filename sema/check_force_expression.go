/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	"github.com/onflow/cadence/ast"
)

func (checker *Checker) VisitForceExpression(expression *ast.ForceExpression) Type {

	// Expected type of the `expression.Expression` is the optional of expected type of current context.
	// i.e: if `x!` is `String`, then `x` is expected to be `String?`.
	expectedType := wrapWithOptionalIfNotNil(checker.expectedType)

	valueType := checker.VisitExpression(expression.Expression, expression, expectedType)

	if valueType.IsInvalidType() {
		return valueType
	}

	checker.recordResourceInvalidation(
		expression.Expression,
		valueType,
		ResourceInvalidationKindMoveDefinite,
	)

	checker.Elaboration.SetForceExpressionType(expression, valueType)

	optionalType, ok := valueType.(*OptionalType)
	if !ok {
		return valueType
	}

	return optionalType.Type
}
