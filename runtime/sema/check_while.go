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

func (checker *Checker) VisitWhileStatement(statement *ast.WhileStatement) ast.Repr {

	checker.VisitExpression(statement.Test, BoolType)

	// The body of the loop will maybe be evaluated.
	// That means that resource invalidations and
	// returns are not definite, but only potential.

	_ = checker.checkPotentiallyUnevaluated(func() Type {
		checker.functionActivations.WithLoop(func() {
			statement.Block.Accept(checker)
		})

		// ignored
		return nil
	})

	checker.reportResourceUsesInLoop(statement.StartPos, statement.EndPosition(checker.memoryGauge))

	return nil
}

func (checker *Checker) reportResourceUsesInLoop(startPos, endPos ast.Position) {

	checker.resources.ForEach(func(resource any, info ResourceInfo) {

		// If the resource is a variable,
		// only report an error if the variable was declared outside the loop

		if variable, isVariable := resource.(*Variable); isVariable &&
			variable.Pos != nil &&
			variable.Pos.Compare(startPos) > 0 &&
			variable.Pos.Compare(endPos) < 0 {

			return
		}

		// Only report an error if the resource was invalidated

		if info.Invalidations.IsEmpty() {
			return
		}

		invalidations := info.Invalidations.All()

		_ = info.UsePositions.ForEach(func(usePosition ast.Position, _ ResourceUse) error {

			// Only report an error if the use is inside the loop

			if usePosition.Compare(startPos) < 0 ||
				usePosition.Compare(endPos) > 0 {

				return nil
			}

			if checker.resources.IsUseAfterInvalidationReported(resource, usePosition) {
				return nil
			}

			checker.resources.MarkUseAfterInvalidationReported(resource, usePosition)

			checker.report(
				&ResourceUseAfterInvalidationError{
					// TODO: improve position information
					StartPos:      usePosition,
					EndPos:        usePosition,
					Invalidations: invalidations,
					InLoop:        true,
				},
			)

			return nil
		})
	})
}

func (checker *Checker) VisitBreakStatement(statement *ast.BreakStatement) ast.Repr {

	// Ensure that the `break` statement is inside a loop or switch statement

	if !(checker.inLoop() || checker.inSwitch()) {
		checker.report(
			&ControlStatementError{
				ControlStatement: common.ControlStatementBreak,
				Range:            ast.NewRangeFromPositioned(checker.memoryGauge, statement),
			},
		)
		return nil
	}

	functionActivation := checker.functionActivations.Current()
	checker.resources.JumpsOrReturns = true
	functionActivation.ReturnInfo.DefinitelyJumped = true

	return nil
}

func (checker *Checker) VisitContinueStatement(statement *ast.ContinueStatement) ast.Repr {

	// Ensure that the `continue` statement is inside a loop statement

	if !checker.inLoop() {
		checker.report(
			&ControlStatementError{
				ControlStatement: common.ControlStatementContinue,
				Range:            ast.NewRangeFromPositioned(checker.memoryGauge, statement),
			},
		)
		return nil
	}

	functionActivation := checker.functionActivations.Current()
	checker.resources.JumpsOrReturns = true
	functionActivation.ReturnInfo.DefinitelyJumped = true

	return nil
}
