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
	"github.com/onflow/cadence/runtime/common"
)

func (checker *Checker) VisitIdentifierExpression(expression *ast.IdentifierExpression) ast.Repr {
	identifier := expression.Identifier
	variable := checker.findAndCheckValueVariable(expression, true)
	if variable == nil {
		return InvalidType
	}

	valueType := variable.Type

	if valueType.IsResourceType() {
		checker.checkResourceVariableCapturingInFunction(variable, identifier)
		checker.checkResourceUseAfterInvalidation(variable, identifier)
		checker.resources.AddUse(variable, identifier.Pos)
	}

	checker.checkSelfVariableUseInInitializer(variable, identifier.Pos)

	if checker.inInvocation {
		checker.Elaboration.IdentifierInInvocationTypes[expression] = valueType
	}

	return valueType
}

// checkSelfVariableUseInInitializer checks uses of `self` in the initializer
// and ensures it is properly initialized
//
func (checker *Checker) checkSelfVariableUseInInitializer(variable *Variable, position ast.Position) {

	// Is this a use of `self`?

	if variable.DeclarationKind != common.DeclarationKindSelf {
		return
	}

	// Is this use of `self` in an initializer?

	initializationInfo := checker.functionActivations.Current().InitializationInfo
	if initializationInfo == nil {
		return
	}

	// The use of `self` is inside the initializer

	checkInitializationComplete := func() {
		if initializationInfo.InitializationComplete() {
			return
		}

		checker.report(
			&UninitializedUseError{
				Name: variable.Identifier,
				Pos:  position,
			},
		)
	}

	if checker.currentMemberExpression != nil {

		// The use of `self` is inside a member access

		// If the member expression refers to a field that must be initialized,
		// it must be initialized. This check is handled in `VisitMemberExpression`

		accessedSelfMember := checker.accessedSelfMember(checker.currentMemberExpression)

		// If the member access is to a predeclared field, it can be considered
		// initialized and its use is valid

		if accessedSelfMember == nil || !accessedSelfMember.Predeclared {

			// If the member access is to a non-field, e.g. a function,
			// *all* fields must have been initialized

			field, _ := initializationInfo.FieldMembers.Get(accessedSelfMember)
			if field == nil {
				checkInitializationComplete()
			}
		}

	} else {
		// The use of `self` is *not* inside a member access, i.e. `self` is used
		// as a standalone expression, e.g. to pass it as an argument to a function.
		// Ensure that *all* fields were initialized

		checkInitializationComplete()
	}
}

// checkResourceVariableCapturingInFunction checks if a resource variable is captured in a function
//
func (checker *Checker) checkResourceVariableCapturingInFunction(variable *Variable, useIdentifier ast.Identifier) {
	currentFunctionDepth := -1
	currentFunctionActivation := checker.functionActivations.Current()
	if currentFunctionActivation != nil {
		currentFunctionDepth = currentFunctionActivation.ValueActivationDepth
	}

	if currentFunctionDepth == -1 ||
		variable.ActivationDepth > currentFunctionDepth {

		return
	}

	checker.report(
		&ResourceCapturingError{
			Name: useIdentifier.Identifier,
			Pos:  useIdentifier.Pos,
		},
	)
}

func (checker *Checker) VisitExpressionStatement(statement *ast.ExpressionStatement) ast.Repr {
	expression := statement.Expression

	ty := checker.VisitExpression(expression, nil)

	if ty.IsResourceType() {
		checker.report(
			&ResourceLossError{
				Range: ast.NewRangeFromPositioned(expression),
			},
		)
	}

	return nil
}

func (checker *Checker) VisitBoolExpression(_ *ast.BoolExpression) ast.Repr {
	return BoolType
}

var TypeOfNil = &OptionalType{
	Type: NeverType,
}

func (checker *Checker) VisitNilExpression(_ *ast.NilExpression) ast.Repr {
	return TypeOfNil
}

func (checker *Checker) VisitIntegerExpression(expression *ast.IntegerExpression) ast.Repr {
	expectedType := UnwrapOptionalType(checker.expectedType)

	var actualType Type
	isAddress := false

	// If the contextually expected type is a subtype of Integer or Address, then take that.
	if expectedType == nil || IsSubType(expectedType, NeverType) {
		actualType = IntType
	} else if IsSubType(expectedType, IntegerType) {
		actualType = expectedType
	} else if IsSubType(expectedType, &AddressType{}) {
		isAddress = true
		CheckAddressLiteral(expression, checker.report)
		actualType = expectedType
	} else {
		// Otherwise infer the type as `Int` which can represent any integer.
		actualType = IntType
	}

	if !isAddress {
		CheckIntegerLiteral(expression, actualType, checker.report)
	}

	checker.Elaboration.IntegerExpressionType[expression] = actualType

	return actualType
}

func (checker *Checker) VisitFixedPointExpression(expression *ast.FixedPointExpression) ast.Repr {
	// TODO: adjust once/if we support more fixed point types

	// If the contextually expected type is a subtype of FixedPoint, then take that.
	// Otherwise, infer the type from the expression itself.

	expectedType := UnwrapOptionalType(checker.expectedType)

	var actualType Type

	if expectedType != nil &&
		!IsSubType(expectedType, NeverType) &&
		IsSubType(expectedType, FixedPointType) {
		actualType = expectedType
	} else if expression.Negative {
		actualType = Fix64Type
	} else {
		actualType = UFix64Type
	}

	CheckFixedPointLiteral(expression, actualType, checker.report)

	checker.Elaboration.FixedPointExpression[expression] = actualType

	return actualType
}

func (checker *Checker) VisitStringExpression(expression *ast.StringExpression) ast.Repr {
	expectedType := UnwrapOptionalType(checker.expectedType)

	if expectedType != nil && IsSubType(expectedType, CharacterType) {
		checker.checkCharacterLiteral(expression)
		return expectedType
	}

	return StringType
}

func (checker *Checker) VisitIndexExpression(expression *ast.IndexExpression) ast.Repr {
	return checker.visitIndexExpression(expression, false)
}

// visitIndexExpression checks if the indexed expression is indexable,
// checks if the indexing expression can be used to index into the indexed expression,
// and returns the expected element type
//
func (checker *Checker) visitIndexExpression(
	indexExpression *ast.IndexExpression,
	isAssignment bool,
) Type {

	targetExpression := indexExpression.TargetExpression
	targetType := checker.VisitExpression(targetExpression, nil)

	// NOTE: check indexed type first for UX reasons

	// check indexed expression's type is indexable
	// by getting the expected element

	if targetType.IsInvalidType() {
		return InvalidType
	}

	// Check if the type instance is actually indexable. For most types (e.g. arrays and dictionaries)
	// this is known statically (in the sense of this host language (Go), not the implemented language),
	// i.e. a Go type switch would be sufficient.
	// However, for some types (e.g. reference types) this depends on what type is referenced

	indexedType, ok := targetType.(ValueIndexableType)
	if !ok || !indexedType.isValueIndexableType() {
		checker.report(
			&NotIndexableTypeError{
				Type:  targetType,
				Range: ast.NewRangeFromPositioned(targetExpression),
			},
		)

		return InvalidType
	}

	elementType := checker.visitValueIndexingExpression(
		indexedType,
		indexExpression.IndexingExpression,
		isAssignment,
	)

	if isAssignment && !indexedType.AllowsValueIndexingAssignment() {
		checker.report(
			&NotIndexingAssignableTypeError{
				Type:  indexedType,
				Range: ast.NewRangeFromPositioned(targetExpression),
			},
		)
	}

	checker.checkUnusedExpressionResourceLoss(elementType, targetExpression)

	return elementType
}

func (checker *Checker) visitValueIndexingExpression(
	indexedType ValueIndexableType,
	indexingExpression ast.Expression,
	isAssignment bool,
) Type {
	checker.VisitExpression(indexingExpression, indexedType.IndexingType())

	return indexedType.ElementType(isAssignment)
}
