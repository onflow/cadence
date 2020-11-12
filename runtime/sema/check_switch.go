/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
)

func (checker *Checker) VisitSwitchStatement(statement *ast.SwitchStatement) ast.Repr {

	testType := statement.Expression.Accept(checker).(Type)

	testTypeIsValid := !testType.IsInvalidType()

	// The test expression must be equatable

	if testTypeIsValid && !testType.IsEquatable() {
		checker.report(
			&NotEquatableTypeError{
				Type:  testType,
				Range: ast.NewRangeFromPositioned(statement.Expression),
			},
		)
	}

	// Check all cases

	caseCount := len(statement.Cases)

	for i, switchCase := range statement.Cases {
		// Only one default case is allowed, as the last case
		defaultAllowed := i == caseCount-1
		checker.visitSwitchCase(switchCase, defaultAllowed, testType, testTypeIsValid)
	}

	checker.functionActivations.WithSwitch(func() {
		checker.checkSwitchCasesStatements(statement.Cases)
	})

	return nil
}

func (checker *Checker) visitSwitchCase(
	switchCase *ast.SwitchCase,
	defaultAllowed bool,
	testType Type,
	testTypeIsValid bool,
) {
	caseExpression := switchCase.Expression

	// If the case has no expression, it is a default case

	if caseExpression == nil {

		// Only one default case is allowed, as the last case
		if !defaultAllowed {
			checker.report(
				&SwitchDefaultPositionError{
					Range: switchCase.Range,
				},
			)
		}
	} else {
		checker.checkSwitchCaseExpression(caseExpression, testType, testTypeIsValid)
	}
}

func (checker *Checker) checkSwitchCaseExpression(
	caseExpression ast.Expression,
	testType Type,
	testTypeIsValid bool,
) {

	caseType := caseExpression.Accept(checker).(Type)

	if caseType.IsInvalidType() {
		return
	}

	// The type of each case expression must be the same
	// as the type of the test expression

	if testTypeIsValid {
		// If the test type is valid,
		// the case type can be checked to be equatable and compatible in one go

		if !AreCompatibleEquatableTypes(testType, caseType) {
			checker.report(
				&InvalidBinaryOperandsError{
					Operation: ast.OperationEqual,
					LeftType:  testType,
					RightType: caseType,
					Range:     ast.NewRangeFromPositioned(caseExpression),
				},
			)
		}
	} else {
		// If the text type is invalid,
		// at least the case type can be checked to be equatable

		if !caseType.IsEquatable() {
			checker.report(
				&NotEquatableTypeError{
					Type:  caseType,
					Range: ast.NewRangeFromPositioned(caseExpression),
				},
			)
		}
	}
}

func (checker *Checker) checkSwitchCasesStatements(cases []*ast.SwitchCase) {
	caseCount := len(cases)
	if caseCount == 0 {
		return
	}

	// NOTE: always check blocks as if they're only *potentially* evaluated.
	// However, the default case's block must be checked directly as the "else",
	// because if a default case exists, the whole switch statement
	// will definitely have one case which will be taken.

	switchCase := cases[0]

	if caseCount == 1 {
		if switchCase.Expression == nil {
			checker.checkSwitchCaseStatements(switchCase)
			return
		}
	}

	_, _ = checker.checkConditionalBranches(
		func() Type {
			checker.checkSwitchCaseStatements(switchCase)
			return nil
		},
		func() Type {
			checker.checkSwitchCasesStatements(cases[1:])
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
				Pos: switchCase.EndPosition().Shifted(1),
			},
		)
		return
	}

	// NOTE: the block ensures that the statements are checked in a new scope

	block := &ast.Block{
		Statements: switchCase.Statements,
		Range: ast.Range{
			StartPos: switchCase.Statements[0].StartPosition(),
			EndPos:   switchCase.EndPos,
		},
	}
	block.Accept(checker)
}
