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

	valueType := checker.VisitExpression(valueExpression, statement, expectedType)

	// Only get the element type if the array is not a resource array.
	// Otherwise, in addition to the `UnsupportedResourceForLoopError`,
	// the loop variable will be declared with the resource-typed element type,
	// leading to an additional `ResourceLossError`.
	loopVariableType := checker.loopVariableType(valueType, valueExpression)

	identifier := statement.Identifier.Identifier

	variable, err := checker.valueActivations.declare(variableDeclaration{
		identifier:               identifier,
		ty:                       loopVariableType,
		kind:                     common.DeclarationKindConstant,
		pos:                      statement.Identifier.Pos,
		isConstant:               true,
		argumentLabels:           nil,
		allowOuterScopeShadowing: false,
		access:                   PrimitiveAccess(ast.AccessNotSpecified),
	})
	checker.report(err)
	if checker.PositionInfo != nil && variable != nil {
		checker.recordVariableDeclarationOccurrence(identifier, variable)
	}

	var indexType Type

	if statement.Index != nil {
		index := statement.Index.Identifier
		indexType = IntType
		indexVariable, err := checker.valueActivations.declare(variableDeclaration{
			identifier:               index,
			ty:                       indexType,
			kind:                     common.DeclarationKindConstant,
			pos:                      statement.Index.Pos,
			isConstant:               true,
			argumentLabels:           nil,
			allowOuterScopeShadowing: false,
			access:                   PrimitiveAccess(ast.AccessNotSpecified),
		})
		checker.report(err)
		if checker.PositionInfo != nil && indexVariable != nil {
			checker.recordVariableDeclarationOccurrence(index, indexVariable)
		}
	}

	checker.Elaboration.SetForStatementType(statement, ForStatementTypes{
		IndexVariableType: indexType,
		ValueVariableType: loopVariableType,
	})

	// The body of the loop will maybe be evaluated.
	// That means that resource invalidations and
	// returns are not definite, but only potential.

	_ = checker.checkPotentiallyUnevaluated(func() Type {
		checker.functionActivations.Current().WithLoop(func() {
			checker.checkBlock(statement.Block)
		})

		// ignored
		return nil
	})

	return
}

func (checker *Checker) loopVariableType(valueType Type, hasPosition ast.HasPosition) Type {
	if valueType.IsInvalidType() {
		return InvalidType
	}

	// Resources cannot be looped.
	if valueType.IsResourceType() {
		checker.report(
			&UnsupportedResourceForLoopError{
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, hasPosition),
			},
		)
		return InvalidType
	}

	// If it's a reference, check whether the referenced type is iterable.
	// If yes, then determine the loop-var type depending on the
	// element-type of the referenced type.
	// If that element type is:
	//  a) A container type, then the loop-var is also a reference-type.
	//  b) A primitive type, then the loop-var is the concrete type itself.

	if referenceType, ok := valueType.(*ReferenceType); ok {
		referencedType := referenceType.Type
		referencedIterableElementType := checker.iterableElementType(referencedType, hasPosition)

		if referencedIterableElementType.IsInvalidType() {
			return referencedIterableElementType
		}

		// Case (a): Element type is a container type.
		// Then the loop-var must also be a reference type.
		if referencedIterableElementType.ContainFieldsOrElements() {
			return checker.getReferenceType(referencedIterableElementType, false, UnauthorizedAccess)
		}

		// Case (b): Element type is a primitive type.
		// Then the loop-var must be the concrete type.
		return referencedIterableElementType
	}

	// If it's not a reference, then simply get the element type.
	return checker.iterableElementType(valueType, hasPosition)
}

func (checker *Checker) iterableElementType(valueType Type, hasPosition ast.HasPosition) Type {
	switch valueType := valueType.(type) {
	case ArrayType:
		return valueType.ElementType(false)
	case *InclusiveRangeType:
		return valueType.MemberType
	}

	if valueType == StringType {
		return CharacterType
	}

	checker.report(
		&TypeMismatchWithDescriptionError{
			ExpectedTypeDescription: "array",
			ActualType:              valueType,
			Range:                   ast.NewRangeFromPositioned(checker.memoryGauge, hasPosition),
		},
	)

	return InvalidType
}
