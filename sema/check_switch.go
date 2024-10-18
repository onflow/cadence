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
)

func (checker *Checker) VisitSwitchStatement(statement *ast.SwitchStatement) (_ struct{}) {

	testType := checker.VisitExpression(statement.Expression, statement, nil)

	testTypeIsValid := !testType.IsInvalidType()

	// The test expression must be equatable

	if testTypeIsValid && !testType.IsEquatable() {
		checker.report(
			&NotEquatableTypeError{
				Type:  testType,
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, statement.Expression),
			},
		)
	}

	// Check all cases

	checker.functionActivations.Current().WithSwitch(func() {
		checker.checkSwitchCasesStatements(
			statement,
			statement.Cases,
			testType,
			testTypeIsValid,
		)
	})

	return
}

func (checker *Checker) checkSwitchCaseExpression(
	statement *ast.SwitchStatement,
	caseExpression ast.Expression,
	testType Type,
	testTypeIsValid bool,
) {

	var caseExprExpectedType Type
	if testTypeIsValid {
		caseExprExpectedType = testType
	}

	caseType := checker.VisitExpression(caseExpression, statement, caseExprExpectedType)

	if caseType.IsInvalidType() {
		return
	}

	// The type of each case expression must be the same
	// as the type of the test expression

	if !testTypeIsValid {
		// If the test type is invalid,
		// at least the case type can be checked to be equatable

		if !caseType.IsEquatable() {
			checker.report(
				&NotEquatableTypeError{
					Type:  caseType,
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, caseExpression),
				},
			)
		}
	}
}

func (checker *Checker) checkSwitchCasesStatements(
	statement *ast.SwitchStatement,
	remainingCases []*ast.SwitchCase,
	testType Type,
	testTypeIsValid bool,
) {
	remainingCaseCount := len(remainingCases)
	if remainingCaseCount == 0 {
		return
	}

	currentFunctionActivation := checker.functionActivations.Current()

	// NOTE: always check blocks as if they're only *potentially* evaluated.
	// However, the default case's block must be checked directly as the "else",
	// because if a default case exists, the whole switch statement
	// will definitely have one case which will be taken.

	switchCase := remainingCases[0]

	caseExpression := switchCase.Expression

	// If the case has no expression, it is a default case
	if caseExpression == nil {

		// Only one default case is allowed, as the last case
		defaultAllowed := remainingCaseCount == 1
		if !defaultAllowed {
			checker.report(
				&SwitchDefaultPositionError{
					Range: switchCase.Range,
				},
			)
		}

		currentFunctionActivation.ReturnInfo.WithNewJumpTarget(func() {
			checker.checkSwitchCaseStatements(switchCase)
		})
		return
	}

	checker.checkSwitchCaseExpression(
		statement,
		caseExpression,
		testType,
		testTypeIsValid,
	)

	_, _ = checker.checkConditionalBranches(
		func() Type {

			currentFunctionActivation.ReturnInfo.WithNewJumpTarget(func() {
				checker.checkSwitchCaseStatements(switchCase)
			})

			// ignored
			return nil
		},
		func() Type {
			checker.checkSwitchCasesStatements(
				statement,
				remainingCases[1:],
				testType,
				testTypeIsValid,
			)

			// ignored
			return nil
		},
	)
}

func (checker *Checker) checkSwitchCaseStatements(switchCase *ast.SwitchCase) {

	// Switch-cases must have at least one statement.
	// This avoids cases that look like implicit fallthrough is assumed.

	if len(switchCase.Statements) == 0 {
		checker.report(
			&MissingSwitchCaseStatementsError{
				Pos: switchCase.EndPosition(checker.memoryGauge).Shifted(checker.memoryGauge, 1),
			},
		)
		return
	}

	// NOTE: the block ensures that the statements are checked in a new scope

	block := ast.NewBlock(
		checker.memoryGauge,
		switchCase.Statements,
		ast.NewRange(
			checker.memoryGauge,
			switchCase.Statements[0].StartPosition(),
			switchCase.EndPos,
		),
	)
	checker.checkBlock(block)
}
