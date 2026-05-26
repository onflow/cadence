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
		functionActivation.ReturnInfo.MaybeJumpedLoop = true

	case ControlKindSwitch:
		functionActivation.ReturnInfo.MaybeJumpedSwitch = true
	}

	// `break` is a kind of definite exit (see `DefinitelyExited`).
	// Setting it means a subsequent statement is correctly reported as
	// unreachable, and an if-else where one branch breaks and the other
	// returns/halts/jumps still propagates as "every path terminated".
	functionActivation.ReturnInfo.DefinitelyExited = true

	// NOTE: unlike `VisitReturnStatement`, no explicit `checkResourceLoss`
	// call here.
	//
	// Break can occur in multiple sibling branches (e.g.
	// `if cond { break } else { break }`) whose resource scopes are
	// cloned independently. If each break reported leaks at its site,
	// the same resource would be reported twice — once per branch — and
	// the clones share no "already-reported" state to deduplicate.
	//
	// Instead, the leak is reported by the surrounding scope's
	// `leaveValueScope` → `checkResourceLoss`. After the if-else merge,
	// the merged resource state reflects both branches, and the scope-
	// leave runs the check exactly once because the
	// `MaybeJumpedLoop`/`MaybeJumpedSwitch` flag tells the skip
	// condition NOT to skip (see `checkResourceLoss`).

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
	functionActivation.ReturnInfo.MaybeJumpedLoop = true
	// `continue` is a kind of definite exit (see `DefinitelyExited`).
	// Setting it means a subsequent statement is correctly reported as
	// unreachable, and an if-else where one branch continues and the
	// other returns/halts/jumps still propagates as "every path
	// terminated".
	functionActivation.ReturnInfo.DefinitelyExited = true

	// NOTE: like `break` and unlike `VisitReturnStatement`, no explicit
	// `checkResourceLoss` here — the surrounding scope's
	// `leaveValueScope` reports the leak. See the matching note in
	// `VisitBreakStatement`.

	return
}
