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

func (checker *Checker) VisitSwitchStatement(statement *ast.SwitchStatement) (_ struct{}) {
	checker.checkSwitchStatement(statement, false)
	return
}

func (checker *Checker) checkSwitchStatement(statement *ast.SwitchStatement, isExhaustive bool) {

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

	// If the #exhaustive pragma was set, verify exhaustiveness.

	if isExhaustive {
		isExhaustive = checker.checkSwitchExhaustiveOverEnum(statement, testType)
	}

	// Check all cases

	checker.functionActivations.Current().WithSwitch(func() {
		checker.checkSwitchCasesStatements(
			statement,
			statement.Cases,
			testType,
			testTypeIsValid,
			isExhaustive,
		)
	})
}

// checkSwitchExhaustiveOverEnum checks whether a switch statement on an enum type
// covers all enum cases. Reports errors if the test type is not an enum
// or if not all enum cases are covered. Returns true if the switch is exhaustive.
func (checker *Checker) checkSwitchExhaustiveOverEnum(
	statement *ast.SwitchStatement,
	testType Type,
) bool {
	compositeType, ok := testType.(*CompositeType)
	if !ok || compositeType.Kind != common.CompositeKindEnum {
		checker.report(
			&InvalidPragmaError{
				Message: "the #exhaustive pragma can only be used with enum types",
				Range:   ast.NewRangeFromPositioned(checker.memoryGauge, statement.Expression),
			},
		)
		return false
	}

	enumCases := compositeType.EnumCases
	if len(enumCases) == 0 {
		return true
	}

	// Build a set of enum case names for quick lookup
	enumCaseSet := make(map[string]struct{}, len(enumCases))
	for _, name := range enumCases {
		enumCaseSet[name] = struct{}{}
	}

	// Track which enum cases are covered by switch cases
	coveredCases := make(map[string]struct{})

	for _, switchCase := range statement.Cases {
		if switchCase.Expression == nil {
			// Default case — skip
			continue
		}

		memberExpr, ok := switchCase.Expression.(*ast.MemberExpression)
		if !ok {
			continue
		}

		identExpr, ok := memberExpr.Expression.(*ast.IdentifierExpression)
		if !ok {
			continue
		}

		// Look up the identifier in scope to verify it refers to the enum type
		variable := checker.valueActivations.Find(identExpr.Identifier.Identifier)
		if variable == nil {
			continue
		}

		funcType, ok := variable.Type.(*FunctionType)
		if !ok {
			continue
		}

		if funcType.TypeFunctionType != compositeType {
			continue
		}

		// The member name is an enum case reference
		memberName := memberExpr.Identifier.Identifier
		if _, isEnumCase := enumCaseSet[memberName]; isEnumCase {
			coveredCases[memberName] = struct{}{}
		}
	}

	if len(coveredCases) == len(enumCases) {
		return true
	}

	// Report which enum cases are missing
	missingCases := make([]string, 0, len(enumCases)-len(coveredCases))
	for _, name := range enumCases {
		if _, covered := coveredCases[name]; !covered {
			missingCases = append(missingCases, name)
		}
	}

	checker.report(
		&MissingSwitchCasesError{
			MissingCases: missingCases,
			Range:        ast.NewRangeFromPositioned(checker.memoryGauge, statement),
		},
	)

	return false
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
	isExhaustive bool,
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
	//
	// Similarly, if the switch is exhaustive over an enum type
	// (via the #exhaustive pragma), the last case is treated like a default case.

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

	// If this is the last case and the switch is exhaustive over an enum,
	// treat this case like a default: it is guaranteed to be taken
	// if none of the previous cases matched.
	if remainingCaseCount == 1 && isExhaustive {
		currentFunctionActivation.ReturnInfo.WithNewJumpTarget(func() {
			checker.checkSwitchCaseStatements(switchCase)
		})
		return
	}

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
				isExhaustive,
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
