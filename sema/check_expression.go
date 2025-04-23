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

func (checker *Checker) VisitIdentifierExpression(expression *ast.IdentifierExpression) Type {
	identifier := expression.Identifier
	variable := checker.findAndCheckValueVariable(expression, true)
	if variable == nil {
		return InvalidType
	}

	valueType := variable.Type

	if valueType.IsResourceType() {
		res := Resource{Variable: variable}
		checker.checkResourceVariableCapturingInFunction(variable, identifier)
		checker.checkResourceUseAfterInvalidation(res, identifier)
	}

	checker.checkSelfVariableUseInInitializer(variable, identifier.Pos)

	checker.checkReferenceValidity(variable, expression)

	if checker.inInvocation {
		checker.Elaboration.SetIdentifierInInvocationType(expression, valueType)
	}

	checker.checkVariableMove(expression, variable)

	return valueType
}

func (checker *Checker) checkVariableMove(identifierExpression *ast.IdentifierExpression, variable *Variable) {

	reportMaybeInvalidMove := func(declarationKind common.DeclarationKind) {
		// If the parent is member-access or index-access, then it's OK.
		// e.g: `v.foo`, `v.bar()`, `v[T]` are OK.
		switch parent := checker.parent.(type) {
		case *ast.MemberExpression:
			// TODO: No need for below check? i.e: should always be true
			if parent.Expression == identifierExpression {
				return
			}
		case *ast.IndexExpression:
			// Only `v[foo]` is OK, `foo[v]` is not.
			if parent.TargetExpression == identifierExpression {
				return
			}
		}

		checker.report(
			&InvalidMoveError{
				Name:            variable.Identifier,
				DeclarationKind: declarationKind,
				Pos:             identifierExpression.StartPosition(),
			},
		)
	}

	switch ty := variable.Type.(type) {
	case *TransactionType:
		reportMaybeInvalidMove(common.DeclarationKindTransaction)

	case CompositeKindedType:
		kind := ty.GetCompositeKind()
		if kind == common.CompositeKindContract {
			reportMaybeInvalidMove(common.DeclarationKindContract)
		}
	}
}

func (checker *Checker) checkReferenceValidity(variable *Variable, hasPosition ast.HasPosition) {
	typ := UnwrapOptionalType(variable.Type)
	if _, ok := typ.(*ReferenceType); !ok {
		return
	}

	// Here it is not required to find the root of the reference chain,
	// because it is already done at the time of recoding the reference.
	// i.e: It is always the roots of the chain that is being stored as the `referencedResourceVariables`.
	for _, referencedVar := range variable.referencedResourceVariables {
		resourceInfo := checker.resources.Get(Resource{Variable: referencedVar})
		invalidation := resourceInfo.Invalidation()
		if invalidation == nil {
			continue
		}

		checker.report(&InvalidatedResourceReferenceError{
			Invalidation: *invalidation,
			Range:        ast.NewRangeFromPositioned(checker.memoryGauge, hasPosition),
		})
	}
}

// checkSelfVariableUseInInitializer checks uses of `self` in the initializer
// and ensures it is properly initialized
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

func (checker *Checker) checkResourceMemberCapturingInFunction(target *ast.MemberExpression, member *Member, memberType Type) {
	if !memberType.IsResourceType() {
		return
	}
	// if the member is a resource, check that it is not captured in a function,
	// based off the activation depth of the root of the access chain, i.e. `a` in `a.b.c`
	// we only want to make this check for transactions, as they are the only "resource-like" types
	// (that can contain resources and must destroy them in their `execute` blocks), that are themselves
	// not checked by the capturing logic, since they are not themselves resources.
	baseVariable, _ := checker.rootOfAccessChain(target)

	if baseVariable == nil {
		return
	}

	if _, isTransaction := baseVariable.Type.(*TransactionType); isTransaction {
		checker.checkResourceVariableCapturingInFunction(baseVariable, member.Identifier)
	}
}

// checkResourceVariableCapturingInFunction checks if a resource variable is captured in a function
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

func (checker *Checker) VisitExpressionStatement(statement *ast.ExpressionStatement) (_ struct{}) {
	expression := statement.Expression

	ty := checker.VisitExpression(expression, statement, nil)

	if ty.IsResourceType() {
		checker.report(
			&ResourceLossError{
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, expression),
			},
		)
	}

	return
}

func (checker *Checker) VisitVoidExpression(_ *ast.VoidExpression) Type {
	return VoidType
}

func (checker *Checker) VisitBoolExpression(_ *ast.BoolExpression) Type {
	return BoolType
}

var NilType = &OptionalType{
	Type: NeverType,
}

func (checker *Checker) VisitNilExpression(_ *ast.NilExpression) Type {
	return NilType
}

func (checker *Checker) VisitIntegerExpression(expression *ast.IntegerExpression) Type {
	expectedType := UnwrapOptionalType(checker.expectedType)

	var actualType Type
	isAddress := false

	// If the contextually expected type is a subtype of Integer or Address, then take that.
	if IsSameTypeKind(expectedType, IntegerType) {
		actualType = expectedType
	} else if IsSameTypeKind(expectedType, TheAddressType) {
		isAddress = true
		CheckAddressLiteral(checker.memoryGauge, expression, checker.report)
		actualType = expectedType
	} else {
		// Otherwise infer the type as `Int` which can represent any integer.
		actualType = IntType
	}

	if !isAddress {
		CheckIntegerLiteral(checker.memoryGauge, expression, actualType, checker.report)
	}

	checker.Elaboration.SetIntegerExpressionType(expression, actualType)

	return actualType
}

func (checker *Checker) VisitFixedPointExpression(expression *ast.FixedPointExpression) Type {
	// TODO: adjust once/if we support more fixed point types

	// If the contextually expected type is a subtype of FixedPoint, then take that.
	// Otherwise, infer the type from the expression itself.

	expectedType := UnwrapOptionalType(checker.expectedType)

	var actualType Type

	if IsSameTypeKind(expectedType, FixedPointType) {
		actualType = expectedType
	} else if expression.Negative {
		actualType = Fix64Type
	} else {
		actualType = UFix64Type
	}

	CheckFixedPointLiteral(checker.memoryGauge, expression, actualType, checker.report)

	checker.Elaboration.SetFixedPointExpression(expression, actualType)

	return actualType
}

func (checker *Checker) VisitStringExpression(expression *ast.StringExpression) Type {
	expectedType := UnwrapOptionalType(checker.expectedType)

	var actualType Type = StringType

	if IsSameTypeKind(expectedType, CharacterType) {
		checker.checkCharacterLiteral(expression)
		actualType = expectedType
	}

	checker.Elaboration.SetStringExpressionType(expression, actualType)

	return actualType
}

func (checker *Checker) VisitIndexExpression(expression *ast.IndexExpression) Type {
	return checker.visitIndexExpression(expression, false)
}

// visitIndexExpression checks if the indexed expression is indexable,
// checks if the indexing expression can be used to index into the indexed expression,
// and returns the expected element type
func (checker *Checker) visitIndexExpression(
	indexExpression *ast.IndexExpression,
	isAssignment bool,
) Type {

	targetExpression := indexExpression.TargetExpression
	targetType := checker.VisitExpression(targetExpression, indexExpression, nil)

	// NOTE: check indexed type first for UX reasons

	// check indexed expression's type is indexable
	// by getting the expected element

	if targetType.IsInvalidType() {
		return InvalidType
	}

	reportNonIndexable := func(t Type) {
		checker.report(
			&NotIndexableTypeError{
				Type:  t,
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, targetExpression),
			},
		)
	}

	// Check if the type instance is actually indexable. For most types (e.g. arrays and dictionaries)
	// this is known statically (in the sense of this host language (Go), not the implemented language),
	// i.e. a Go type switch would be sufficient.
	// However, for some types (e.g. reference types) this depends on what type is referenced
	// we cannot use a switch here because in the reference type case we need to be able to "fallthrough",
	// and Go does not allow fallthroughs in typed switches
	valueIndexedType, isValueIndexableType := targetType.(ValueIndexableType)

	if isValueIndexableType && valueIndexedType.isValueIndexableType() {
		if isAssignment && !valueIndexedType.AllowsValueIndexingAssignment() {
			checker.report(
				&NotIndexingAssignableTypeError{
					Type:  valueIndexedType,
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, targetExpression),
				},
			)
		}
		indexingType := checker.VisitExpression(
			indexExpression.IndexingExpression,
			indexExpression,
			valueIndexedType.IndexingType(),
		)

		elementType := valueIndexedType.ElementType(isAssignment)

		checker.checkUnusedExpressionResourceLoss(elementType, targetExpression)

		// If the element,
		//   1) is accessed via a reference, and
		//   2) is container-typed,
		// then the element type should also be a reference.
		returnReference := false
		if shouldReturnReference(valueIndexedType, elementType, isAssignment) {
			// For index expressions, element are un-authorized.
			elementType = checker.getReferenceType(elementType, false, UnauthorizedAccess)

			// Store the result in elaboration, so the interpreter can re-use this.
			returnReference = true
		}

		checker.Elaboration.SetIndexExpressionTypes(
			indexExpression,
			IndexExpressionTypes{
				IndexedType:     valueIndexedType,
				IndexingType:    indexingType,
				ResultType:      elementType,
				ReturnReference: returnReference,
			},
		)

		return elementType
	}

	typeIndexedType, isTypeIndexableType := targetType.(TypeIndexableType)

	if isTypeIndexableType && typeIndexedType.isTypeIndexableType() {
		if isAssignment {
			checker.report(
				&NotIndexingAssignableTypeError{
					Type:  typeIndexedType,
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, targetExpression),
				},
			)
			return InvalidType
		}

		return checker.checkTypeIndexingExpression(typeIndexedType, indexExpression)
	}

	reportNonIndexable(targetType)
	return InvalidType
}

func (checker *Checker) checkTypeIndexingExpression(
	targetType TypeIndexableType,
	indexExpression *ast.IndexExpression,
) Type {

	targetExpression := indexExpression.TargetExpression

	reportInvalid := func() {
		checker.report(
			&InvalidTypeIndexingError{
				BaseType:           targetType,
				IndexingExpression: indexExpression.IndexingExpression,
				Range: ast.NewRangeFromPositioned(
					checker.memoryGauge,
					indexExpression.IndexingExpression,
				),
			},
		)
	}

	expressionType := ast.ExpressionAsType(indexExpression.IndexingExpression)
	if expressionType == nil {
		reportInvalid()
		return InvalidType
	}

	nominalTypeExpression, isNominalType := expressionType.(*ast.NominalType)
	if !isNominalType {
		reportInvalid()
		return InvalidType
	}

	nominalType := checker.convertNominalType(nominalTypeExpression)

	if !targetType.IsValidIndexingType(nominalType) {
		reportInvalid()
		return InvalidType
	}

	checker.Elaboration.SetAttachmentAccessTypes(indexExpression, nominalType)

	checker.checkUnusedExpressionResourceLoss(targetType, targetExpression)

	// at this point, the base is known to be a struct/resource,
	// and the attachment is known to be a valid attachment for that base
	indexedType, err := targetType.TypeIndexingElementType(nominalType, func() ast.Range { return ast.NewRangeFromPositioned(checker.memoryGauge, indexExpression) })
	if err != nil {
		checker.report(err)
		return InvalidType
	}
	return indexedType
}
