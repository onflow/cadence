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

import "github.com/onflow/cadence/ast"

func (checker *Checker) VisitReturnStatement(statement *ast.ReturnStatement) (_ struct{}) {
	functionActivation := checker.functionActivations.Current()

	defer func() {
		// Check for resource loss at the return site, BEFORE marking
		// the function as having definitely exited.
		//
		// This is the only termination kind that calls `checkResourceLoss`
		// explicitly. The reason is twofold:
		//
		// 1. Reporting at the return statement (rather than at the
		//    end-of-function scope-leave) makes the error point at the
		//    return — the source location the programmer expects.
		//
		// 2. Return is a single-site termination: the value scopes
		//    being abandoned are unambiguous, and there are no sibling
		//    branches that might each try to report the same leak.
		//    By contrast, break/continue can occur in multiple sibling
		//    branches (e.g. `if cond { break } else { break }`) whose
		//    resource scopes are cloned independently — having them
		//    report at their site would double-report. Halt
		//    intentionally suppresses leak reports (see the skip in
		//    `checkResourceLoss`).
		//
		// `checkResourceLoss` checks all variables declared inside the
		// function. The function activation's `ValueActivationDepth` is
		// where the *function* is declared (its parent scope); two
		// value-activation scopes are defined for the function itself
		// (parameters, then body), so `+ 1` covers both.
		//
		// The check runs BEFORE `DefinitelyExited` is set, so the
		// scope-leave skip in `checkResourceLoss` does not yet
		// suppress it. After this point, `DefinitelyExited = true`
		// causes subsequent scope-leaves to skip and avoid
		// double-reporting.
		checker.checkResourceLoss(functionActivation.ValueActivationDepth + 1)
		functionActivation.ReturnInfo.MaybeReturned = true
		functionActivation.ReturnInfo.DefinitelyReturned = true
		functionActivation.ReturnInfo.DefinitelyExited = true
	}()

	returnType := functionActivation.ReturnType

	if statement.Expression == nil {

		// If the return statement has no expression,
		// and the enclosing function's return type is non-Void,
		// then the return statement is missing an expression

		if returnType != VoidType {
			checker.report(
				&MissingReturnValueError{
					ExpectedValueType: returnType,
					Range:             ast.NewRangeFromPositioned(checker.memoryGauge, statement),
				},
			)
		}

		return
	}

	// If the return statement has a return value,
	// check that the value's type matches the enclosing function's return type

	valueType := checker.VisitExpression(statement.Expression, statement, returnType)

	checker.Elaboration.SetReturnStatementTypes(
		statement,
		ReturnStatementTypes{
			ValueType:  valueType,
			ReturnType: returnType,
		},
	)

	if returnType == VoidType {
		return
	}

	checker.checkResourceMoveOperation(statement.Expression, valueType)

	return
}
