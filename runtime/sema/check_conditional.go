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
	"github.com/onflow/cadence/runtime/common/persistent"
	"github.com/onflow/cadence/runtime/errors"
)

func (checker *Checker) VisitIfStatement(statement *ast.IfStatement) (_ struct{}) {

	thenElement := statement.Then

	switch test := statement.Test.(type) {
	case ast.Expression:
		checker.VisitExpression(test, BoolType)

		checker.checkConditionalBranches(
			func() Type {
				checker.checkBlock(statement.Then)
				return nil
			},
			func() Type {
				if statement.Else != nil {
					checker.checkBlock(statement.Else)
				}
				return nil
			},
		)

	case *ast.VariableDeclaration:
		declarationType := checker.visitVariableDeclarationValues(test, true)

		checker.checkConditionalBranches(
			func() Type {
				checker.enterValueScope()
				defer checker.leaveValueScope(thenElement.EndPosition, true)

				if castingExpression, ok := test.Value.(*ast.CastingExpression); ok &&
					castingExpression.Operation == ast.OperationFailableCast {

					castingTypes := checker.Elaboration.CastingExpressionTypes(castingExpression)
					leftHandType := castingTypes.StaticValueType
					if leftHandType.IsResourceType() {
						checker.recordResourceInvalidation(
							castingExpression.Expression,
							leftHandType,
							ResourceInvalidationKindMoveDefinite,
						)
					}
				}
				checker.declareVariableDeclaration(test, declarationType)

				checker.checkBlock(thenElement)
				return nil
			},
			func() Type {
				if statement.Else != nil {
					checker.checkBlock(statement.Else)
				}
				return nil
			},
		)

	default:
		panic(errors.NewUnreachableError())
	}

	return
}

func (checker *Checker) VisitConditionalExpression(expression *ast.ConditionalExpression) Type {

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
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, expression.Then),
			},
		)
	}
	if elseType.IsResourceType() {
		checker.report(
			&InvalidConditionalResourceOperandError{
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, expression.Else),
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

// checkConditionalBranches checks two conditional branches.
// It is assumed that either one of the branches is taken, so function returns,
// resource uses and invalidations, as well as field initializations,
// are only potential in each branch, but definite if they occur in both branches.
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

	var thenInitializedMembers *persistent.OrderedSet[*Member]
	var elseInitializedMembers *persistent.OrderedSet[*Member]
	if functionActivation.InitializationInfo != nil {
		initialInitializedMembers := functionActivation.InitializationInfo.InitializedFieldMembers
		thenInitializedMembers = initialInitializedMembers.Clone()
		elseInitializedMembers = initialInitializedMembers.Clone()
	}

	initialResources := checker.resources
	thenResources := initialResources.Clone()
	defer thenResources.Reclaim()
	elseResources := initialResources.Clone()
	defer elseResources.Reclaim()

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

	checker.resources.MergeBranches(
		thenResources,
		thenReturnInfo,
		elseResources,
		elseReturnInfo,
	)

	return
}

// checkBranch checks a conditional branch.
// It is assumed that function returns, resource uses and invalidations,
// as well as field initializations, are only potential / temporary.
func (checker *Checker) checkBranch(
	check TypeCheckFunc,
	temporaryReturnInfo *ReturnInfo,
	temporaryInitializedMembers *persistent.OrderedSet[*Member],
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
