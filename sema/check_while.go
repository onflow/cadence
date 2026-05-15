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
	"github.com/onflow/cadence/errors"
)

func (checker *Checker) VisitWhileStatement(statement *ast.WhileStatement) (_ struct{}) {

	checker.VisitExpression(statement.Test, statement, BoolType)

	// The body of the loop will maybe be evaluated.
	// That means that resource invalidations and
	// returns are not definite, but only potential.

	_, _ = checker.checkPotentiallyUnevaluated(func() Type {
		checker.functionActivations.Current().WithLoop(func() {
			checker.checkBlock(statement.Block)
		})

		// ignored
		return nil
	})

	return
}

func (checker *Checker) VisitBreakStatement(statement *ast.BreakStatement) (_ struct{}) {

	// `break` targets the innermost enclosing loop or switch.

	functionActivation := checker.functionActivations.Current()
	if functionActivation == nil {
		panic(errors.NewUnreachableError())
	}

	innermost := functionActivation.InnermostControl()

	functionActivation.ReturnInfo.AddJumpOffset(statement.StartPos.Offset)

	switch innermost {
	case ControlKindNone:
		checker.report(
			&ControlStatementError{
				ControlStatement: common.ControlStatementBreak,
				Range:            ast.NewRangeFromPositioned(checker.memoryGauge, statement),
			},
		)
		return

	case ControlKindLoop:
		functionActivation.ReturnInfo.DefinitelyJumpedLoop = true
		functionActivation.ReturnInfo.MaybeJumpedLoop = true

	case ControlKindSwitch:
		functionActivation.ReturnInfo.DefinitelyJumpedSwitch = true
		functionActivation.ReturnInfo.MaybeJumpedSwitch = true
	}

	// `break` is a kind of definite exit (see DefinitelyExited).
	// Set it so that an if-else where one branch breaks and the other
	// returns/halts/jumps still propagates as "definitely terminated".
	functionActivation.ReturnInfo.DefinitelyExited = true

	return
}

func (checker *Checker) VisitContinueStatement(statement *ast.ContinueStatement) (_ struct{}) {

	// `continue` always targets the enclosing loop, even when nested inside switches.

	functionActivation := checker.functionActivations.Current()
	if functionActivation == nil {
		panic(errors.NewUnreachableError())
	}

	if !checker.inLoop() {
		checker.report(
			&ControlStatementError{
				ControlStatement: common.ControlStatementContinue,
				Range:            ast.NewRangeFromPositioned(checker.memoryGauge, statement),
			},
		)
		return
	}

	functionActivation.ReturnInfo.AddJumpOffset(statement.StartPos.Offset)
	functionActivation.ReturnInfo.DefinitelyJumpedLoop = true
	functionActivation.ReturnInfo.MaybeJumpedLoop = true
	// `continue` is a kind of definite exit (see DefinitelyExited).
	functionActivation.ReturnInfo.DefinitelyExited = true

	return
}
