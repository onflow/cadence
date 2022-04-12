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

func (checker *Checker) VisitSwapStatement(swap *ast.SwapStatement) ast.Repr {

	leftType := checker.VisitExpression(swap.Left, nil)
	rightType := checker.VisitExpression(swap.Right, nil)

	checker.Elaboration.SwapStatementLeftTypes[swap] = leftType
	checker.Elaboration.SwapStatementRightTypes[swap] = rightType

	lhsValid := checker.checkSwapStatementExpression(swap.Left, leftType, common.OperandSideLeft)
	rhsValid := checker.checkSwapStatementExpression(swap.Right, rightType, common.OperandSideRight)

	// The types of both sides must be subtypes of each other,
	// so that assignment can be performed in both directions.
	// i.e: The two types have to be equal.
	if lhsValid && rhsValid && !leftType.Equal(rightType) {
		checker.report(
			&TypeMismatchError{
				ExpectedType: leftType,
				ActualType:   rightType,
				Range:        ast.NewRangeFromPositioned(checker.memoryGauge, swap.Right),
			},
		)
	}

	if leftType.IsResourceType() {
		checker.elaborateNestedResourceMoveExpression(swap.Left)
	}

	if rightType.IsResourceType() {
		checker.elaborateNestedResourceMoveExpression(swap.Right)
	}

	return nil
}

func (checker *Checker) checkSwapStatementExpression(
	expression ast.Expression,
	exprType Type,
	opSide common.OperandSide,
) bool {

	// Expression in either side of the swap statement must be a target expression.
	// (e.g. identifier expression, indexing expression, or member access expression)
	if !IsValidAssignmentTargetExpression(expression) {
		checker.report(
			&InvalidSwapExpressionError{
				Side:  opSide,
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, expression),
			},
		)
		return false
	}

	if exprType.IsInvalidType() {
		return false
	}

	checker.visitAssignmentValueType(expression)
	return true
}
