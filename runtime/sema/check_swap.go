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

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

func (checker *Checker) VisitSwapStatement(swap *ast.SwapStatement) (_ struct{}) {

	// First visit the two expressions as if they were the target of the assignment.
	leftTargetType := checker.checkSwapStatementExpression(swap.Left, common.OperandSideLeft)
	rightTargetType := checker.checkSwapStatementExpression(swap.Right, common.OperandSideRight)

	// Then re-visit the same expressions, this time treat them as the value-expr of the assignment.
	// The 'expected type' of the two expression would be the types obtained from the previous visit, swapped.
	leftValueType := checker.VisitExpression(swap.Left, rightTargetType)
	rightValueType := checker.VisitExpression(swap.Right, leftTargetType)

	checker.Elaboration.SetSwapStatementTypes(
		swap,
		SwapStatementTypes{
			LeftType:  leftValueType,
			RightType: rightValueType,
		},
	)

	if leftValueType.IsResourceType() {
		checker.elaborateNestedResourceMoveExpression(swap.Left)
	}

	if rightValueType.IsResourceType() {
		checker.elaborateNestedResourceMoveExpression(swap.Right)
	}

	return
}

func (checker *Checker) checkSwapStatementExpression(
	expression ast.Expression,
	opSide common.OperandSide,
) Type {

	// Expression in either side of the swap statement must be a target expression.
	// (e.g. identifier expression, indexing expression, or member access expression)
	if !IsValidAssignmentTargetExpression(expression) {
		checker.report(
			&InvalidSwapExpressionError{
				Side:  opSide,
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, expression),
			},
		)
		return InvalidType
	}

	return checker.visitAssignmentValueType(expression)
}
