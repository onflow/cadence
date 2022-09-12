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
	"github.com/onflow/cadence/runtime/common"
)

func (checker *Checker) VisitForStatement(statement *ast.ForStatement) (_ struct{}) {

	checker.enterValueScope()
	defer checker.leaveValueScope(statement.EndPosition, true)

	valueExpression := statement.Value

	// iterations are only supported for non-resource arrays.
	// Hence, if the array is empty and no context type is available,
	// then default it to [AnyStruct].
	var expectedType Type
	arrayExpression, ok := valueExpression.(*ast.ArrayExpression)
	if ok && len(arrayExpression.Values) == 0 {
		expectedType = &VariableSizedType{
			Type: AnyStructType,
		}
	}

	valueType := checker.VisitExpression(valueExpression, expectedType)

	var elementType Type = InvalidType

	if !valueType.IsInvalidType() {

		// Only get the element type if the array is not a resource array.
		// Otherwise, in addition to the `UnsupportedResourceForLoopError`,
		// the loop variable will be declared with the resource-typed element type,
		// leading to an additional `ResourceLossError`.

		if valueType.IsResourceType() {
			checker.report(
				&UnsupportedResourceForLoopError{
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, valueExpression),
				},
			)
		} else if arrayType, ok := valueType.(ArrayType); ok {
			elementType = arrayType.ElementType(false)
		} else {
			checker.report(
				&TypeMismatchWithDescriptionError{
					ExpectedTypeDescription: "array",
					ActualType:              valueType,
					Range:                   ast.NewRangeFromPositioned(checker.memoryGauge, valueExpression),
				},
			)
		}
	}

	identifier := statement.Identifier.Identifier

	variable, err := checker.valueActivations.declare(variableDeclaration{
		identifier:               identifier,
		ty:                       elementType,
		kind:                     common.DeclarationKindConstant,
		pos:                      statement.Identifier.Pos,
		isConstant:               true,
		argumentLabels:           nil,
		allowOuterScopeShadowing: false,
	})
	checker.report(err)
	if checker.PositionInfo != nil {
		checker.recordVariableDeclarationOccurrence(identifier, variable)
	}

	if statement.Index != nil {
		index := statement.Index.Identifier
		indexVariable, err := checker.valueActivations.declare(variableDeclaration{
			identifier:               index,
			ty:                       IntType,
			kind:                     common.DeclarationKindConstant,
			pos:                      statement.Index.Pos,
			isConstant:               true,
			argumentLabels:           nil,
			allowOuterScopeShadowing: false,
		})
		checker.report(err)
		if checker.PositionInfo != nil {
			checker.recordVariableDeclarationOccurrence(index, indexVariable)
		}
	}

	// The body of the loop will maybe be evaluated.
	// That means that resource invalidations and
	// returns are not definite, but only potential.

	_ = checker.checkPotentiallyUnevaluated(func() Type {
		checker.functionActivations.WithLoop(func() {
			checker.checkBlock(statement.Block)
		})

		// ignored
		return nil
	})

	checker.reportResourceUsesInLoop(statement.StartPos, statement.EndPosition(checker.memoryGauge))

	return
}
