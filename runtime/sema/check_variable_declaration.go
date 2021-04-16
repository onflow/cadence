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
	"github.com/onflow/cadence/runtime/errors"
)

func (checker *Checker) VisitVariableDeclaration(declaration *ast.VariableDeclaration) ast.Repr {
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
	if declaration.TypeAnnotation != nil {
		typeAnnotation := checker.ConvertTypeAnnotation(declaration.TypeAnnotation)
		checker.checkTypeAnnotation(typeAnnotation, declaration.TypeAnnotation)
		declarationType = typeAnnotation.Type
	}

	valueType := checker.VisitExpression(declaration.Value, declarationType)

	checker.Elaboration.VariableDeclarationValueTypes[declaration] = valueType

	if declarationType == nil {
		if isOptionalBinding {
			if optionalType, isOptional := valueType.(*OptionalType); isOptional {
				declarationType = optionalType.Type
			} else {
				checker.report(
					&TypeMismatchError{
						ExpectedType: &OptionalType{},
						ActualType:   valueType,
						Range:        ast.NewRangeFromPositioned(declaration.Value),
					},
				)
			}
		} else {
			declarationType = valueType
		}
	}

	//valueIsInvalid := valueType.IsInvalidType()

	// If the variable declaration is an optional binding, the value must be optional

	//var valueIsOptional bool
	//var optionalValueType *OptionalType
	//
	//if isOptionalBinding && !valueIsInvalid {
	//	optionalValueType, valueIsOptional = valueType.(*OptionalType)
	//	if !valueIsOptional {
	//		checker.report(
	//			&TypeMismatchError{
	//				ExpectedType: &OptionalType{},
	//				ActualType:   valueType,
	//				Range:        ast.NewRangeFromPositioned(declaration.Value),
	//			},
	//		)
	//	}
	//}

	// Determine the declaration type based on the value type and the optional type annotation

	//declarationType := valueType

	// If the declaration has an explicit type annotation, take it into account:
	// Check it and ensure the value type is *compatible* with the type annotation

	// TODO: Remove this check
	//if declaration.TypeAnnotation != nil {
	//
	//	typeAnnotation := checker.ConvertTypeAnnotation(declaration.TypeAnnotation)
	//	checker.checkTypeAnnotation(typeAnnotation, declaration.TypeAnnotation)
	//
	//	declarationType = typeAnnotation.Type
	//
	//	// check the value type is a subtype of the declaration type
	//	if declarationType != nil && valueType != nil && !valueIsInvalid && !declarationType.IsInvalidType() {
	//
	//		if isOptionalBinding {
	//			if optionalValueType != nil &&
	//				(optionalValueType.Equal(declarationType) ||
	//					!IsSubType(optionalValueType.Type, declarationType)) {
	//
	//				checker.report(
	//					&TypeMismatchError{
	//						ExpectedType: declarationType,
	//						ActualType:   optionalValueType.Type,
	//						Range:        ast.NewRangeFromPositioned(declaration.Value),
	//					},
	//				)
	//			}
	//
	//		}
	//	}
	//} else if isOptionalBinding && optionalValueType != nil {
	//	declarationType = optionalValueType.Type
	//}

	checker.Elaboration.VariableDeclarationTargetTypes[declaration] = declarationType

	//if declarationType != nil &&
	//	valueType != nil &&
	//	!valueIsInvalid &&
	//	!declarationType.IsInvalidType() &&
	//	!isOptionalBinding &&
	//	!checker.checkTypeCompatibility(declaration.Value, valueType, declarationType) {
	//
	//	checker.report(
	//		&TypeMismatchError{
	//			ExpectedType: declarationType,
	//			ActualType:   valueType,
	//			Range:        ast.NewRangeFromPositioned(declaration.Value),
	//		},
	//	)
	//}

	checker.checkVariableDeclarationUsability(declaration)

	checker.checkTransfer(declaration.Transfer, declarationType)

	// The variable declaration might have a second transfer and second expression.
	//
	// In that case the declaration transfers not only the value of the first expression
	// to the identifier (new variable), but the declaration also transfers the value
	// of the second expression to the first expression (which must be a target expression).
	//
	// This is only valid for resources, i.e. the declaration type, first value type,
	// and the second value type must be resource types, and all transfers must be moves.

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
					Range: ast.NewRangeFromPositioned(declaration.Value),
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
						Range:      ast.NewRangeFromPositioned(declaration.Value),
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

			_, secondValueType := checker.checkAssignment(
				declaration.Value,
				declaration.SecondValue,
				declaration.SecondTransfer,
				true,
			)

			checker.Elaboration.VariableDeclarationSecondValueTypes[declaration] = secondValueType

			if valueIsResource {
				checker.elaborateIndexExpressionResourceMove(declaration.Value)
			}
		}
	}

	// Finally, declare the variable in the current value activation

	identifier := declaration.Identifier.Identifier

	variable, err := checker.valueActivations.Declare(variableDeclaration{
		identifier:               identifier,
		ty:                       declarationType,
		access:                   declaration.Access,
		kind:                     declaration.DeclarationKind(),
		pos:                      declaration.Identifier.Pos,
		isConstant:               declaration.IsConstant,
		argumentLabels:           nil,
		allowOuterScopeShadowing: true,
	})
	checker.report(err)
	if checker.originsAndOccurrencesEnabled {
		checker.recordVariableDeclarationOccurrence(identifier, variable)
	}
}

func (checker *Checker) checkVariableDeclarationUsability(declaration *ast.VariableDeclaration) {

	// If the variable declaration has no type annotation
	// and the value is an empty array literal,
	// then the type is inferred to `[Never]`,
	// which is effectively useless
	// (it is an empty array to which no values can be added).
	//
	// Require an explicit type annotation

	if declaration.TypeAnnotation == nil {
		switch value := declaration.Value.(type) {
		case *ast.ArrayExpression:
			if len(value.Values) == 0 {
				checker.report(
					&TypeAnnotationRequiredError{
						Cause: "empty array literal",
						Pos:   declaration.Identifier.EndPosition().Shifted(1),
					},
				)
			}

		case *ast.DictionaryExpression:
			if len(value.Entries) == 0 {
				checker.report(
					&TypeAnnotationRequiredError{
						Cause: "empty dictionary literal",
						Pos:   declaration.Identifier.EndPosition().Shifted(1),
					},
				)
			}
		}
	}
}

func (checker *Checker) elaborateIndexExpressionResourceMove(expression ast.Expression) {
	indexExpression, ok := expression.(*ast.IndexExpression)
	if !ok {
		return
	}

	checker.Elaboration.IsResourceMoveIndexExpression[indexExpression] = true
}
