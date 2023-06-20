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
	"github.com/onflow/cadence/runtime/errors"
)

func (checker *Checker) visitConditions(conditions []ast.Condition) {
	// all condition blocks are `view`
	checker.InNewPurityScope(true, func() {
		// flag the checker to be inside a condition.
		// this flag is used to detect illegal expressions,
		// see e.g. VisitFunctionExpression

		wasInCondition := checker.inCondition
		checker.inCondition = true
		defer func() {
			checker.inCondition = wasInCondition
		}()

		// check all conditions: check the expression
		// and ensure the result is boolean

		for _, condition := range conditions {
			checker.checkCondition(condition)
		}
	})
}

func (checker *Checker) checkCondition(condition ast.Condition) Type {

	switch condition := condition.(type) {
	case *ast.TestCondition:

		// check test expression is boolean
		checker.VisitExpression(condition.Test, BoolType)

		// check message expression results in a string
		if condition.Message != nil {
			checker.VisitExpression(condition.Message, StringType)
		}

	case *ast.EmitCondition:
		// TODO:
		panic("TODO")

	default:
		panic(errors.NewUnreachableError())
	}

	return nil
}
