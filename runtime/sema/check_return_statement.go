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

import "github.com/onflow/cadence/runtime/ast"

func (checker *Checker) VisitReturnStatement(statement *ast.ReturnStatement) Type {
	functionActivation := checker.functionActivations.Current()

	defer func() {
		checker.checkResourceLossForFunction()
		checker.resources.JumpsOrReturns = true
		functionActivation.ReturnInfo.MaybeReturned = true
		functionActivation.ReturnInfo.DefinitelyReturned = true
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

		return nil
	}

	// If the return statement has a return value,
	// check that the value's type matches the enclosing function's return type

	valueType := checker.VisitExpression(statement.Expression, returnType)

	checker.Elaboration.ReturnStatementTypes[statement] =
		ReturnStatementTypes{
			ValueType:  valueType,
			ReturnType: returnType,
		}

	if returnType == VoidType {
		return nil
	}

	checker.checkVariableMove(statement.Expression)
	checker.checkResourceMoveOperation(statement.Expression, valueType)

	return nil
}

func (checker *Checker) checkResourceLossForFunction() {
	functionValueActivationDepth :=
		checker.functionActivations.Current().ValueActivationDepth
	checker.checkResourceLoss(functionValueActivationDepth)
}
