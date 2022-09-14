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

func (checker *Checker) VisitWhileStatement(statement *ast.WhileStatement) (_ struct{}) {

	checker.VisitExpression(statement.Test, BoolType)

	// The body of the loop will maybe be evaluated.
	// That means that resource invalidations and
	// returns are not definite, but only potential.

	_ = checker.checkPotentiallyUnevaluated(func() Type {
		checker.functionActivations.WithLoop(func() {
			checker.checkBlock(statement.Block)
		})

		// ignored
		return nil
	})

	return
}

func (checker *Checker) VisitBreakStatement(statement *ast.BreakStatement) (_ struct{}) {

	// Ensure that the `break` statement is inside a loop or switch statement

	if !(checker.inLoop() || checker.inSwitch()) {
		checker.report(
			&ControlStatementError{
				ControlStatement: common.ControlStatementBreak,
				Range:            ast.NewRangeFromPositioned(checker.memoryGauge, statement),
			},
		)
		return
	}

	functionActivation := checker.functionActivations.Current()
	functionActivation.ReturnInfo.MaybeJumped = true
	functionActivation.ReturnInfo.DefinitelyJumped = true

	return
}

func (checker *Checker) VisitContinueStatement(statement *ast.ContinueStatement) (_ struct{}) {

	// Ensure that the `continue` statement is inside a loop statement

	if !checker.inLoop() {
		checker.report(
			&ControlStatementError{
				ControlStatement: common.ControlStatementContinue,
				Range:            ast.NewRangeFromPositioned(checker.memoryGauge, statement),
			},
		)
		return
	}

	functionActivation := checker.functionActivations.Current()
	functionActivation.ReturnInfo.MaybeJumped = true
	functionActivation.ReturnInfo.DefinitelyJumped = true

	return
}
