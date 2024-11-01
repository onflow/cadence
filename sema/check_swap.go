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
	"github.com/onflow/cadence/common"
)

func (checker *Checker) VisitSwapStatement(swap *ast.SwapStatement) (_ struct{}) {

	// First visit the two expressions as if they were the target of the assignment.
	leftTargetType := checker.checkSwapStatementExpression(swap.Left, common.OperandSideLeft)
	rightTargetType := checker.checkSwapStatementExpression(swap.Right, common.OperandSideRight)

	// Then re-visit the same expressions, this time treat them as the value-expr of the assignment.
	// The 'expected type' of the two expression would be the types obtained from the previous visit, swapped.
	leftValueType := checker.VisitExpression(swap.Left, swap, rightTargetType)
	rightValueType := checker.VisitExpression(swap.Right, swap, leftTargetType)

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

	// If the left or right side is an index expression,
	// and the indexed type (type of the target expression) is a resource type,
	// then the target expression must be considered as a nested resource move expression.
	//
	// This is because the evaluation of the index expression
	// should not be able to access/move the target resource.
	//
	// For example, if a side is `a.b[c()]`, then `a.b` is the target expression.
	// If `a.b` is a resource, then `c()` should not be able to access/move it.

	for _, side := range []ast.Expression{swap.Left, swap.Right} {
		if indexExpression, ok := side.(*ast.IndexExpression); ok {
			indexExpressionTypes, ok := checker.Elaboration.IndexExpressionTypes(indexExpression)

			// If the indexed type is a resource type,
			// then the target expression must be considered as a nested resource move expression.
			//
			// The index expression might have been invalid,
			// so the indexed type might be unavailable.
			if ok && indexExpressionTypes.IndexedType.IsResourceType() {
				targetExpression := indexExpression.TargetExpression
				checker.elaborateNestedResourceMoveExpression(targetExpression)
			}
		}
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
