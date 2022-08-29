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

func (checker *Checker) VisitVariableDeclaration(declaration *ast.VariableDeclaration) Type {
	checker.visitVariableDeclaration(declaration, false)
	return nil
}

func (checker *Checker) visitVariableDeclaration(declaration *ast.VariableDeclaration, isOptionalBinding bool) {

	checker.checkDeclarationAccessModifier(
		declaration.Access,
		declaration.DeclarationKind(),
		declaration.StartPos,
		declaration.IsConstant,
	)

	// Determine the type of the initial value of the variable declaration
	// and save it in the elaboration

	var declarationType Type
	var expectedValueType Type

	if declaration.TypeAnnotation != nil {
		typeAnnotation := checker.ConvertTypeAnnotation(declaration.TypeAnnotation)
		checker.checkTypeAnnotation(typeAnnotation, declaration.TypeAnnotation)
		declarationType = typeAnnotation.Type

		// If the variable declaration is an optional binding (`if let`),
		// then the value is expected to be an optional of the declaration's type.
		if isOptionalBinding {
			expectedValueType = &OptionalType{
				Type: declarationType,
			}
		} else {
			expectedValueType = declarationType
		}
	}

	valueType := checker.VisitExpression(declaration.Value, expectedValueType)

	if isOptionalBinding {
		optionalType, isOptional := valueType.(*OptionalType)

		if !isOptional || optionalType.Equal(declarationType) {
			if !valueType.IsInvalidType() {
				checker.report(
					&TypeMismatchError{
						ExpectedType: &OptionalType{},
						ActualType:   valueType,
						Range:        ast.NewRangeFromPositioned(checker.memoryGauge, declaration.Value),
					},
				)
			}
		} else if declarationType == nil {
			declarationType = optionalType.Type
		}
	}

	if declarationType == nil {
		declarationType = valueType
	}

	checker.checkTransfer(declaration.Transfer, declarationType)

	// The variable declaration might have a second transfer and second expression.
	//
	// In that case the declaration transfers not only the value of the first expression
	// to the identifier (new variable), but the declaration also transfers the value
	// of the second expression to the first expression (which must be a target expression).
	//
	// This is only valid for resources, i.e. the declaration type, first value type,
	// and the second value type must be resource types, and all transfers must be moves.

	var secondValueType Type

	if declaration.SecondTransfer == nil {
		if declaration.SecondValue != nil {
			panic(errors.NewUnreachableError())
		}

		checker.checkVariableMove(declaration.Value)

		// If only one value expression is provided, it is invalidated (if it has a resource type)

		checker.recordResourceInvalidation(
			declaration.Value,
			declarationType,
			ResourceInvalidationKindMoveDefinite,
		)
	} else {

		// The first expression must be a target expression (e.g. identifier expression,
		// indexing expression, or member access expression)

		if !IsValidAssignmentTargetExpression(declaration.Value) {
			checker.report(
				&InvalidAssignmentTargetError{
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, declaration.Value),
				},
			)
		} else {
			// The assignment is valid (i.e. to a target expression)

			// NOTE: Check that the *first* value type is a resource type –
			// The assignment check below will also ensure that the second value type
			// is a resource.
			//
			// The first value is checked instead of the second value,
			// so that second value types that are standalone not considered resource typed
			// are still admitted if they are type compatible (e.g. `nil`).

			valueIsResource := valueType != nil && valueType.IsResourceType()

			if valueType != nil &&
				!valueType.IsInvalidType() &&
				!valueIsResource {

				checker.report(
					&NonResourceTypeError{
						ActualType: valueType,
						Range:      ast.NewRangeFromPositioned(checker.memoryGauge, declaration.Value),
					},
				)
			}

			// Check the assignment of the second value to the first expression

			// The check of the assignment of the second value to the first also:
			// - Invalidates the second resource
			// - Checks the second transfer
			// - Checks the second value type is a subtype of value type
			// etc.

			// NOTE: already performs resource invalidation

			_, secondValueType = checker.checkAssignment(
				declaration.Value,
				declaration.SecondValue,
				declaration.SecondTransfer,
				true,
			)

			if valueIsResource {
				checker.elaborateNestedResourceMoveExpression(declaration.Value)
			}
		}
	}

	checker.Elaboration.VariableDeclarationTypes[declaration] =
		VariableDeclarationTypes{
			TargetType:      declarationType,
			ValueType:       valueType,
			SecondValueType: secondValueType,
		}

	// Finally, declare the variable in the current value activation

	identifier := declaration.Identifier.Identifier

	variable, err := checker.valueActivations.Declare(variableDeclaration{
		identifier:               identifier,
		ty:                       declarationType,
		docString:                declaration.DocString,
		access:                   declaration.Access,
		kind:                     declaration.DeclarationKind(),
		pos:                      declaration.Identifier.Pos,
		isConstant:               declaration.IsConstant,
		argumentLabels:           nil,
		allowOuterScopeShadowing: true,
	})
	checker.report(err)

	if checker.PositionInfo != nil {
		checker.recordVariableDeclarationOccurrence(identifier, variable)
		checker.recordVariableDeclarationRange(declaration, identifier, declarationType)
	}
}

func (checker *Checker) recordVariableDeclarationRange(
	declaration *ast.VariableDeclaration,
	identifier string,
	declarationType Type,
) {
	activation := checker.valueActivations.Current()
	activation.LeaveCallbacks = append(
		activation.LeaveCallbacks,
		func(getEndPosition EndPositionGetter) {
			if getEndPosition == nil || checker.PositionInfo == nil {
				return
			}

			memoryGauge := checker.memoryGauge

			endPosition := getEndPosition(memoryGauge)

			checker.PositionInfo.recordVariableDeclarationRange(
				memoryGauge,
				declaration,
				endPosition,
				identifier,
				declarationType,
			)
		},
	)
}

func (checker *Checker) elaborateNestedResourceMoveExpression(expression ast.Expression) {
	switch expression.(type) {
	case *ast.IndexExpression, *ast.MemberExpression:
		checker.Elaboration.IsNestedResourceMoveExpression[expression] = struct{}{}
	}
}
