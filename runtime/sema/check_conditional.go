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
	"github.com/onflow/cadence/runtime/errors"
)

func (checker *Checker) VisitIfStatement(statement *ast.IfStatement) ast.Repr {

	thenElement := statement.Then

	var elseElement ast.Element = ast.NotAnElement{}
	if statement.Else != nil {
		elseElement = statement.Else
	}

	switch test := statement.Test.(type) {
	case ast.Expression:
		checker.visitConditional(test, thenElement, elseElement)

	case *ast.VariableDeclaration:
		checker.checkConditionalBranches(
			func() Type {
				checker.enterValueScope()
				defer checker.leaveValueScope(thenElement.EndPosition, true)

				checker.visitVariableDeclaration(test, true)
				thenElement.Accept(checker)

				return nil
			},
			func() Type {
				elseElement.Accept(checker)
				return nil
			},
		)

	default:
		panic(errors.NewUnreachableError())
	}

	return nil
}

func (checker *Checker) VisitConditionalExpression(expression *ast.ConditionalExpression) ast.Repr {

	expectedType := checker.expectedType

	checker.VisitExpression(expression.Test, BoolType)

	thenType, elseType := checker.checkConditionalBranches(
		func() Type {
			return checker.VisitExpression(expression.Then, expectedType)
		},
		func() Type {
			return checker.VisitExpression(expression.Else, expectedType)
		},
	)

	if thenType == nil || elseType == nil {
		panic(errors.NewUnreachableError())
	}

	if thenType.IsResourceType() {
		checker.report(
			&InvalidConditionalResourceOperandError{
				Range: ast.NewRangeFromPositioned(expression.Then),
			},
		)
	}
	if elseType.IsResourceType() {
		checker.report(
			&InvalidConditionalResourceOperandError{
				Range: ast.NewRangeFromPositioned(expression.Else),
			},
		)
	}

	if expectedType != nil {
		return expectedType
	}

	if thenType.Equal(elseType) {
		return thenType
	}

	return LeastCommonSuperType(thenType, elseType)
}

// visitConditional checks a conditional.
// The test expression must be a boolean.
// The "then" and "else" elements may be expressions, in which case their types are returned.
//
func (checker *Checker) visitConditional(
	test ast.Expression,
	thenElement ast.Element,
	elseElement ast.Element,
) (
	thenType, elseType Type,
) {

	checker.VisitExpression(test, BoolType)

	return checker.checkConditionalBranches(
		func() Type {
			thenResult, ok := thenElement.Accept(checker).(Type)
			if !ok || thenResult == nil {
				return nil
			}
			return thenResult
		},
		func() Type {
			elseResult, ok := elseElement.Accept(checker).(Type)
			if !ok || elseResult == nil {
				return nil
			}
			return elseResult
		},
	)
}

// checkConditionalBranches checks two conditional branches.
// It is assumed that either one of the branches is taken, so function returns,
// resource uses and invalidations, as well as field initializations,
// are only potential in each branch, but definite if they occur in both branches.
//
func (checker *Checker) checkConditionalBranches(
	checkThen TypeCheckFunc,
	checkElse TypeCheckFunc,
) (
	thenType, elseType Type,
) {
	functionActivation := checker.functionActivations.Current()

	initialReturnInfo := functionActivation.ReturnInfo
	thenReturnInfo := initialReturnInfo.Clone()
	elseReturnInfo := initialReturnInfo.Clone()

	var thenInitializedMembers *MemberSet
	var elseInitializedMembers *MemberSet
	if functionActivation.InitializationInfo != nil {
		initialInitializedMembers := functionActivation.InitializationInfo.InitializedFieldMembers
		thenInitializedMembers = initialInitializedMembers.Clone()
		elseInitializedMembers = initialInitializedMembers.Clone()
	}

	initialResources := checker.resources
	thenResources := initialResources.Clone()
	elseResources := initialResources.Clone()

	thenType = checker.checkBranch(
		checkThen,
		thenReturnInfo,
		thenInitializedMembers,
		thenResources,
	)

	elseType = checker.checkBranch(
		checkElse,
		elseReturnInfo,
		elseInitializedMembers,
		elseResources,
	)

	functionActivation.ReturnInfo.MergeBranches(thenReturnInfo, elseReturnInfo)

	if functionActivation.InitializationInfo != nil {

		// If one side definitely halted, the initializations in the other side can be considered definite

		if thenReturnInfo.DefinitelyHalted {
			functionActivation.InitializationInfo.InitializedFieldMembers = elseInitializedMembers
		} else if elseReturnInfo.DefinitelyHalted {
			functionActivation.InitializationInfo.InitializedFieldMembers = thenInitializedMembers
		} else {
			functionActivation.InitializationInfo.InitializedFieldMembers.
				AddIntersection(thenInitializedMembers, elseInitializedMembers)
		}
	}

	checker.resources.MergeBranches(thenResources, elseResources)

	return
}

// checkBranch checks a conditional branch.
// It is assumed that function returns, resource uses and invalidations,
// as well as field initializations, are only potential / temporary.
//
func (checker *Checker) checkBranch(
	check TypeCheckFunc,
	temporaryReturnInfo *ReturnInfo,
	temporaryInitializedMembers *MemberSet,
	temporaryResources *Resources,
) Type {
	return wrapTypeCheck(check,
		func(f TypeCheckFunc) Type {
			return checker.checkWithResources(f, temporaryResources)
		},
		func(f TypeCheckFunc) Type {
			return checker.checkWithInitializedMembers(f, temporaryInitializedMembers)
		},
		func(f TypeCheckFunc) Type {
			return checker.checkWithReturnInfo(f, temporaryReturnInfo)
		},
	)()
}
