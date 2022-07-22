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

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

func (checker *Checker) VisitUnaryExpression(expression *ast.UnaryExpression) ast.Repr {

	var expectedType Type
	if expression.Operation == ast.OperationMove {
		expectedType = checker.expectedType
	}

	valueType := checker.VisitExpressionWithForceType(expression.Expression, expectedType, false)

	reportInvalidUnaryOperator := func(expectedType Type) {
		checker.report(
			&InvalidUnaryOperandError{
				Operation:    expression.Operation,
				ExpectedType: expectedType,
				ActualType:   valueType,
				Range:        ast.NewRangeFromPositioned(checker.memoryGauge, expression.Expression),
			},
		)
	}

	checkExpectedType := func(valueType, expectedType Type) Type {
		if !valueType.IsInvalidType() &&
			!IsSameTypeKind(valueType, expectedType) {

			reportInvalidUnaryOperator(expectedType)
			return InvalidType
		}

		return valueType
	}

	switch expression.Operation {
	case ast.OperationNegate:
		return checkExpectedType(valueType, BoolType)

	case ast.OperationMinus:
		return checkExpectedType(valueType, SignedNumberType)

	case ast.OperationMove:
		if !valueType.IsInvalidType() &&
			!valueType.IsResourceType() {

			checker.report(
				&InvalidMoveOperationError{
					Range: ast.NewRange(
						checker.memoryGauge,
						expression.StartPos,
						expression.Expression.StartPosition(),
					),
				},
			)
		}

		checker.recordResourceInvalidation(
			expression.Expression,
			valueType,
			ResourceInvalidationKindMoveDefinite,
		)

		return valueType
	}

	panic(&unsupportedOperation{
		kind:      common.OperationKindUnary,
		operation: expression.Operation,
		Range:     ast.NewRangeFromPositioned(checker.memoryGauge, expression),
	})
}
