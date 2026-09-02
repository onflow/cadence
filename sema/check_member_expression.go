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
	"github.com/onflow/cadence/errors"
)

// NOTE: only called if the member expression is *not* an assignment
func (checker *Checker) VisitMemberExpression(expression *ast.MemberExpression) Type {
	accessedType, memberType, member, _ := checker.visitMember(expression, false)

	if !accessedType.IsInvalidType() {
		memberAccessType := accessedType

		if expression.Optional {
			if memberAccessOptionalType, ok := memberAccessType.(*OptionalType); ok {
				memberAccessType = memberAccessOptionalType.Type
			}
		}

		if checker.PositionInfo != nil {
			checker.PositionInfo.recordMemberAccess(
				checker.memoryGauge,
				expression,
				memberAccessType,
			)
		}
	}

	if member == nil {
		return InvalidType
	}

	accessedSelfMember := checker.accessedSelfMember(expression)
	if accessedSelfMember != nil {

		functionActivation := checker.functionActivations.Current()

		// Prevent an access to a field before it was initialized.
		//
		// If this is not an assignment to a `self` member, and the member is a field
		// which must be initialized, ensure the field has been initialized.
		//
		// An access of a member which is not a field / which must not be initialized, is safe
		// (e.g. a composite function call)

		info := functionActivation.InitializationInfo
		isInInitializer := info != nil

		if isInInitializer {
			fieldInitialized := info.InitializedFieldMembers.Contains(accessedSelfMember)

			field, _ := info.FieldMembers.Get(accessedSelfMember)
			if field != nil && !fieldInitialized {

				checker.report(
					&UninitializedFieldAccessError{
						Name: expression.Identifier.Identifier,
						Pos:  expression.Identifier.Pos,
					},
				)
			}
		}
	}

	checker.checkResourceMemberCapturingInFunction(expression, member, memberType)

	return memberType
}

// GetDescendantReferenceType Returns a reference type to a given descendant's (member/element) type.
// Reference to an optional should return an optional reference.
// This has to be done recursively for nested optionals.
// e.g.1: Given type T, this method returns &T.
// e.g.2: Given T?, this returns (&T)?
//
// When the descendant is already a reference, the outer reference's authorization
// is intersected with the inner reference's authorization. This prevents authorization
// escalation when accessing references stored in referenced containers.
// e.g.: auth(E, F) &[auth(F, G) &T] → auth(F) &T (intersection of {E,F} and {F,G})
func GetDescendantReferenceType(
	memoryGauge common.MemoryGauge,
	descendantType Type,
	authorization Access,
	outerAuthorization Access,
) Type {
	switch typ := descendantType.(type) {
	case *OptionalType:
		innerType := GetDescendantReferenceType(
			memoryGauge,
			typ.Type,
			authorization,
			outerAuthorization,
		)
		return NewOptionalType(memoryGauge, innerType)

	case *ReferenceType:
		// Intersect the outer reference's authorization with the inner reference's authorization.
		// This prevents gaining authorization through nested reference access.
		// Also, recursively intersect any references within the referenced type,
		// using the effective (intersected) authorization — not the original outer.
		// This cascades: if the top-level intersection strips auth,
		// inner references are also stripped accordingly.
		intersected := IntersectAccess(outerAuthorization, typ.Authorization)
		innerType := intersectReferenceAuthorizationsInType(memoryGauge, typ.Type, intersected)
		if intersected.Equal(typ.Authorization) && innerType == typ.Type {
			return typ
		}
		return NewReferenceType(memoryGauge, intersected, innerType)

	default:
		// Recursively intersect any references within the element type
		// before wrapping in a reference.
		innerType := intersectReferenceAuthorizationsInType(memoryGauge, typ, outerAuthorization)
		return NewReferenceType(memoryGauge, authorization, innerType)

	}
}

// intersectReferenceAuthorizationsInType recursively traverses a type and intersects
// all inner reference type authorizations with outerAuthorization.
// Returns the original type unchanged if no intersection was applied.
//
// Only references in covariant positions are intersected,
// i.e. positions a reader obtains a value from.
// A reference in a contravariant position is one the reader has to supply
// rather than one it receives, so capping it grants the reader nothing,
// and instead weakens what callers are required to pass.
// See intersectReferenceAuthorizationsInTypeAtPosition.
func intersectReferenceAuthorizationsInType(
	memoryGauge common.MemoryGauge,
	typ Type,
	outerAuthorization Access,
) Type {
	const covariant = true
	return intersectReferenceAuthorizationsInTypeAtPosition(
		memoryGauge,
		typ,
		outerAuthorization,
		covariant,
	)
}

// intersectReferenceAuthorizationsInTypeAtPosition implements
// intersectReferenceAuthorizationsInType for a type occurring
// in a covariant or a contravariant position.
//
// Variance only ever flips at function parameters,
// so it has to be tracked rather than derived locally:
// the parameter of a parameter is covariant again.
// For example, when reading a member of type
// `fun(callback: fun(reference: auth(E) &T): Void): Void`
// through a weaker reference,
// `callback` is supplied by the reader and must not be capped,
// but `reference` is passed *to* the reader's callback,
// and therefore must be.
func intersectReferenceAuthorizationsInTypeAtPosition(
	memoryGauge common.MemoryGauge,
	typ Type,
	outerAuthorization Access,
	covariant bool,
) Type {
	switch t := typ.(type) {
	case *ReferenceType:
		if !covariant {
			// The reader supplies this reference, rather than obtaining it,
			// so its authorization is left as declared.
			// Positions nested inside it are still traversed,
			// as variance can flip back to covariant further in.
			innerType := intersectReferenceAuthorizationsInTypeAtPosition(
				memoryGauge,
				t.Type,
				outerAuthorization,
				covariant,
			)
			if innerType == t.Type {
				return t
			}
			return NewReferenceType(memoryGauge, t.Authorization, innerType)
		}

		intersected := IntersectAccess(outerAuthorization, t.Authorization)
		// Cascade: use the effective (intersected) auth for inner recursion
		innerType := intersectReferenceAuthorizationsInTypeAtPosition(
			memoryGauge,
			t.Type,
			intersected,
			covariant,
		)
		if intersected.Equal(t.Authorization) && innerType == t.Type {
			return t
		}
		return NewReferenceType(memoryGauge, intersected, innerType)

	case *OptionalType:
		innerType := intersectReferenceAuthorizationsInTypeAtPosition(memoryGauge, t.Type, outerAuthorization, covariant)
		if innerType == t.Type {
			return t
		}
		return NewOptionalType(memoryGauge, innerType)

	case *VariableSizedType:
		elementType := intersectReferenceAuthorizationsInTypeAtPosition(memoryGauge, t.Type, outerAuthorization, covariant)
		if elementType == t.Type {
			return t
		}
		return NewVariableSizedType(memoryGauge, elementType)

	case *ConstantSizedType:
		elementType := intersectReferenceAuthorizationsInTypeAtPosition(memoryGauge, t.Type, outerAuthorization, covariant)
		if elementType == t.Type {
			return t
		}
		return NewConstantSizedType(memoryGauge, elementType, t.Size)

	case *DictionaryType:
		keyType := intersectReferenceAuthorizationsInTypeAtPosition(memoryGauge, t.KeyType, outerAuthorization, covariant)
		valueType := intersectReferenceAuthorizationsInTypeAtPosition(memoryGauge, t.ValueType, outerAuthorization, covariant)
		if keyType == t.KeyType && valueType == t.ValueType {
			return t
		}
		return NewDictionaryType(memoryGauge, keyType, valueType)

	case *CapabilityType:
		borrowType := t.BorrowType
		if borrowType == nil {
			return t
		}

		newBorrowType := intersectReferenceAuthorizationsInTypeAtPosition(
			memoryGauge,
			borrowType,
			outerAuthorization,
			covariant,
		)
		if newBorrowType == borrowType {
			return t
		}
		return NewCapabilityType(memoryGauge, newBorrowType)

	case *FunctionType:
		changed := false

		newReturnType := intersectReferenceAuthorizationsInTypeAtPosition(
			memoryGauge,
			t.ReturnTypeAnnotation.Type,
			outerAuthorization,
			covariant,
		)
		if newReturnType != t.ReturnTypeAnnotation.Type {
			changed = true
		}

		// The type parameters are preserved unchanged, rather than copied with
		// intersected bounds. A type parameter bound constrains type arguments; it is
		// not a value-bearing position and cannot hide a recoverable authorized
		// reference. Preserving the original binders also keeps the `GenericType`
		// references in the parameter and return types, and the `TypeArgumentsCheck`
		// callback, consistent, since all identify the binders by pointer identity.

		// Parameters, and the default arguments feeding them, flip the variance.
		parameterCovariant := !covariant

		newParameters := t.Parameters
		parametersCopied := false
		for i, parameter := range t.Parameters {
			newParameterType := intersectReferenceAuthorizationsInTypeAtPosition(
				memoryGauge,
				parameter.TypeAnnotation.Type,
				outerAuthorization,
				parameterCovariant,
			)

			newDefaultArgument := parameter.DefaultArgument
			if parameter.DefaultArgument != nil {
				newDefaultArgument = intersectReferenceAuthorizationsInTypeAtPosition(
					memoryGauge,
					parameter.DefaultArgument,
					outerAuthorization,
					parameterCovariant,
				)
			}

			if newParameterType == parameter.TypeAnnotation.Type &&
				newDefaultArgument == parameter.DefaultArgument {
				continue
			}
			if !parametersCopied {
				newParameters = make([]Parameter, len(t.Parameters))
				copy(newParameters, t.Parameters)
				parametersCopied = true
				changed = true
			}
			newParameters[i] = Parameter{
				TypeAnnotation:  NewTypeAnnotation(newParameterType),
				DefaultArgument: newDefaultArgument,
				Label:           parameter.Label,
				Identifier:      parameter.Identifier,
			}
		}

		if !changed {
			return t
		}

		return &FunctionType{
			Purity:                   t.Purity,
			ReturnTypeAnnotation:     NewTypeAnnotation(newReturnType),
			Arity:                    t.Arity,
			ArgumentExpressionsCheck: t.ArgumentExpressionsCheck,
			TypeArgumentsCheck:       t.TypeArgumentsCheck,
			Members:                  t.Members,
			TypeParameters:           t.TypeParameters,
			Parameters:               newParameters,
			IsConstructor:            t.IsConstructor,
			TypeFunctionType:         t.TypeFunctionType,
		}

	default:
		return typ
	}
}

func ShouldReturnReference(parentType, memberType Type, isAssignment bool) bool {
	if isAssignment {
		return false
	}

	if _, isReference := MaybeReferenceType(parentType); !isReference {
		return false
	}

	// If the member is already a reference, then a reference must be returned.
	if _, elementTypeIsReference := MaybeReferenceType(memberType); elementTypeIsReference {
		return true
	}

	return memberType.ContainFieldsOrElements()
}

func MaybeReferenceType(typ Type) (*ReferenceType, bool) {
	unwrappedType := UnwrapOptionalType(typ)
	refType, isReference := unwrappedType.(*ReferenceType)
	return refType, isReference
}

// GetDescendantTypeForAccess returns the type that a descendant (member or element)
// should have when read through `accessedType`, and whether the descendant was
// wrapped in a reference (equivalent to ShouldReturnReference's result for the same inputs).
//
// Two independent decisions are made here, and must not be coupled:
//
//  1. Whether the descendant value must be wrapped in a reference.
//     This is ShouldReturnReference's decision, and it drives the returned boolean.
//     Only fielded/element-containing descendants (or descendants that are already
//     references) are wrapped, and only when read through a reference.
//
//  2. Whether references nested inside the descendant's type must be
//     authorization-intersected with the outer reference's authorization.
//     This applies whenever the container is read through a reference, regardless
//     of decision (1), so that a weak reference to a container caps every
//     authorization exported through the descendant.
//
// Coupling the second decision to the first is unsound for descendants that carry
// references but report no fields or elements — notably capability and function
// values (`CapabilityType` and `FunctionType` both return false from
// ContainFieldsOrElements). Those take ShouldReturnReference's early exit even
// when they nest authorized references (e.g. `fun(): auth(E) &S`), so without a
// separate intersection step their nested authorizations would leak through
// unchanged.
//
// When the descendant warrants becoming a reference per ShouldReturnReference,
// it is wrapped via GetDescendantReferenceType with an unauthorized wrapping
// authorization, intersecting any inner reference authorizations with the outer
// reference's authorization, and the returned boolean is true.
//
// When the descendant is not wrapped but is read through a reference, its nested
// reference authorizations are still intersected via intersectContainerElementReferences,
// and the returned boolean is false.
//
// Otherwise (for owned access, or in assignment contexts, where the descendant is
// written rather than read out): `descendantType` is returned unchanged and the
// returned boolean is false.
//
// This encapsulates the cascading rule that applies uniformly when reading element/member data
// out of a referenced container or composite.
// Call sites that need a custom wrapping authorization (e.g. mapped field access)
// keep using GetDescendantReferenceType directly.
//
// The returned boolean is useful for callers (notably the interpreter's container-method
// implementations) that also need to materialize fresh reference values for each iterated
// element via getReferenceValue when the cascade wraps. Callers that only need the type
// can discard the boolean.
func GetDescendantTypeForAccess(
	memoryGauge common.MemoryGauge,
	accessedType Type,
	descendantType Type,
	isAssignment bool,
) (Type, bool) {
	if !ShouldReturnReference(accessedType, descendantType, isAssignment) {
		// The descendant is not wrapped in a reference.
		// In assignment contexts the descendant is being written, not read out,
		// so no intersection applies and the type is returned unchanged.
		if isAssignment {
			return descendantType, false
		}
		// When the container is read through a reference, references nested inside
		// the descendant's type must still be intersected with the outer reference's
		// authorization, even though the descendant itself is not wrapped.
		// This handles descendants that carry references but
		// report no fields or elements (capability and function values).
		// For owned access, or descendants without nested references, this is a no-op.
		return intersectContainerElementReferences(memoryGauge, accessedType, descendantType), false
	}
	outerRef, isRef := MaybeReferenceType(accessedType)
	if !isRef {
		panic(errors.NewUnreachableError())
	}
	return GetDescendantReferenceType(
		memoryGauge,
		descendantType,
		UnauthorizedAccess,
		outerRef.Authorization,
	), true
}

func (checker *Checker) visitMember(expression *ast.MemberExpression, isAssignment bool) (
	accessedType Type,
	resultingType Type,
	member *Member,
	isOptional bool,
) {
	memberInfo, ok := checker.Elaboration.MemberExpressionMemberAccessInfo(expression)
	if ok {
		return memberInfo.AccessedType, memberInfo.ResultingType, memberInfo.Member, memberInfo.IsOptional
	}

	returnReference := false
	cappedNestedReferences := false

	defer func() {
		checker.Elaboration.SetMemberExpressionMemberAccessInfo(
			expression,
			MemberAccessInfo{
				AccessedType:           accessedType,
				ResultingType:          resultingType,
				Member:                 member,
				IsOptional:             isOptional,
				ReturnReference:        returnReference,
				CappedNestedReferences: cappedNestedReferences,
			},
		)
	}()

	accessedExpression := expression.Expression

	func() {
		previousMemberExpression := checker.currentMemberExpression
		checker.currentMemberExpression = expression
		defer func() {
			checker.currentMemberExpression = previousMemberExpression
		}()

		// in an statement like `a.b.c = x`, the entire statement itself
		// is an assignment, but the evaluation of the accessed exprssion itself (i.e. `a.b`)
		// is not, so we temporarily clear the `inAssignment` status here before restoring it later.
		accessedType = checker.withAssignment(false, func() Type {
			return checker.VisitExpression(accessedExpression, expression, nil)
		})
	}()

	checker.checkUnusedExpressionResourceLoss(accessedType, accessedExpression)

	// If the access is to a member of `self` and a resource,
	// its use must be recorded/checked, so that it isn't used after it was invalidated

	accessedSelfMember := checker.accessedSelfMember(expression)
	if accessedSelfMember != nil &&
		accessedSelfMember.TypeAnnotation.Type.IsResourceType() {

		// NOTE: Preventing the capturing of the resource field is already implicitly handled:
		// By definition, the resource field can only be nested in a resource,
		// so `self` is a resource, and the capture of it is checked separately

		res := Resource{Member: accessedSelfMember}

		checker.checkResourceUseAfterInvalidation(res, expression.Identifier)
	}

	identifier := expression.Identifier.Identifier
	identifierStartPosition := expression.Identifier.StartPosition()
	identifierEndPosition := expression.Identifier.EndPosition(checker.memoryGauge)

	// Check if the type instance actually has members. For most types (e.g. composite types)
	// this is known statically (in the sense of this host language (Go), not the implemented language),
	// i.e. a Go type switch would be sufficient.
	// However, for some types (e.g. reference types) this depends on what type is referenced

	findAndSetResultingType := func(expressionType Type, optional bool) {
		resolver, ok := expressionType.GetMembers()[identifier]
		if !ok {
			return
		}

		member = resolver.Resolve(
			checker.memoryGauge,
			identifier,
			expression.Expression,
			checker.report,
		)
		resultingType = member.TypeAnnotation.Type

		// If the member is accessed using optional-chaining, then the resulting type also should be optional.
		// However, if the member is already optional, then no need to double-wrap from optionals.
		if optional {
			if _, memberIsOptional := resultingType.(*OptionalType); !memberIsOptional {
				resultingType = NewOptionalType(checker.memoryGauge, resultingType)
			}
		}

		isOptional = optional
	}

	// Get the member from the accessed value based
	// on the use of optional chaining syntax

	if expression.Optional {

		// If the member expression is using optional chaining,
		// check if the accessed type is optional

		if optionalExpressionType, ok := accessedType.(*OptionalType); ok {
			// The accessed type is optional, get the member from the wrapped type

			findAndSetResultingType(optionalExpressionType.Type, true)
		} else {
			// Optional chaining was used on a non-optional type, report an error

			if !accessedType.IsInvalidType() {

				// The length of the optional chaining operator `?.` is 2
				const optionalChainingOperatorLength = 2

				checker.report(
					&InvalidOptionalChainingError{
						Type: accessedType,
						Range: ast.NewRange(
							checker.memoryGauge,
							expression.AccessEndPos.Shifted(
								checker.memoryGauge,
								-(optionalChainingOperatorLength-1),
							),
							expression.AccessEndPos,
						),
					},
				)
			}

			// NOTE: still try to get member for non-optional expression
			// to avoid spurious error that member does not exist,
			// even if the non-optional accessed type has the member

			findAndSetResultingType(accessedType, false)
		}
	} else {
		// The member is accessed directly without optional chaining.
		// Get the member directly from the accessed type
		findAndSetResultingType(accessedType, false)
	}

	if member == nil {
		if !accessedType.IsInvalidType() {

			if checker.Config.ExtendedElaborationEnabled {
				checker.Elaboration.SetMemberExpressionExpectedType(
					expression,
					checker.expectedType,
				)
			}

			checker.report(
				&NotDeclaredMemberError{
					Type:          accessedType,
					Name:          identifier,
					SuggestMember: checker.Config.SuggestionsEnabled,
					Expression:    expression,
					Range: ast.NewRange(
						checker.memoryGauge,
						identifierStartPosition,
						identifierEndPosition,
					),
				},
			)
		}

		return
	}

	if checker.PositionInfo != nil {
		checker.PositionInfo.recordMemberOccurrence(
			checker.memoryGauge,
			accessedType,
			identifier,
			identifierStartPosition,
			identifierEndPosition,
		)
	}

	// Check access and report if inaccessible
	isReadable := checker.isReadableMember(accessedType, member)
	if !isReadable {
		// if the member being accessed has entitled access,
		// also report the authorization possessed by the reference so that developers
		// can more easily see what access is missing
		var possessedAccess Access
		if _, ok := member.Access.(PrimitiveAccess); !ok {
			if ty, ok := accessedType.(*ReferenceType); ok {
				possessedAccess = ty.Authorization
			}
		}
		checker.report(
			&InvalidAccessError{
				Name:                member.Identifier.Identifier,
				RestrictingAccess:   member.Access,
				PossessedAccess:     possessedAccess,
				DeclarationKind:     member.DeclarationKind,
				SuggestEntitlements: checker.Config.SuggestionsEnabled,
				Range:               ast.NewRangeFromPositioned(checker.memoryGauge, expression),
			},
		)
	}

	// Check that the member access is not to a function of resource type
	// outside of an invocation of it.
	//
	// This would result in a bound method for a resource, which is invalid.

	if member.DeclarationKind == common.DeclarationKindFunction &&
		!accessedType.IsInvalidType() &&
		accessedType.IsResourceType() {

		parent := checker.parent
		parentInvocationExpr, parentIsInvocation := parent.(*ast.InvocationExpression)

		if !parentIsInvocation ||
			expression != parentInvocationExpr.InvokedExpression {
			checker.report(
				&ResourceMethodBindingError{
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, expression),
				},
			)
		}
	}

	// If the member,
	//   1) is accessed via a reference, and
	//   2) is container-typed,
	// then the member type should also be a reference.
	// Otherwise, if the member is already a reference, then again, a reference must be returned.

	// Note: For attachments, `self` is always a reference.
	// But we do not want to return a reference for `self.something`.
	// Otherwise, things like `destroy self.something` would become invalid.
	// Hence, special case `self`, and return a reference only if the member is not accessed via self.
	// i.e: `accessedSelfMember == nil`

	if accessedSelfMember == nil &&
		member.DeclarationKind == common.DeclarationKindField {

		switch {
		case ShouldReturnReference(accessedType, resultingType, isAssignment):
			// ShouldReturnReference only holds when the member is accessed
			// through a reference, which grantedMemberAuthorization requires.
			grantedAuthorization := checker.grantedMemberAuthorization(
				member,
				accessedType,
				expression,
			)

			// For non-reference members, the granted authorization is also
			// the authorization of the reference the member gets wrapped in,
			// but only for a field with mapping access:
			// any other field is wrapped in an unauthorized reference.
			// For reference members, the granted authorization is intersected
			// with the member's own authorization instead, and it is not wrapped again.
			wrappingAuthorization := UnauthorizedAccess
			if _, isMappedAccess := member.Access.(*EntitlementMapAccess); isMappedAccess {
				wrappingAuthorization = grantedAuthorization
			}

			resultingType = GetDescendantReferenceType(
				checker.memoryGauge,
				resultingType,
				wrappingAuthorization,
				grantedAuthorization,
			)
			returnReference = true

		case !isAssignment:
			// The member is not wrapped in a reference,
			// but references nested inside its type must still be intersected
			// with the authorization reading the member grants,
			// so that a weak reference to the container caps every authorization
			// exported through the member.
			//
			// This handles members that carry references
			// but report no fields or elements,
			// namely capability and function values:
			// both `CapabilityType` and `FunctionType` return false
			// from `ContainFieldsOrElements`, so they take
			// `ShouldReturnReference`'s early exit
			// even when they nest authorized references
			// (e.g. `Capability<auth(E) &S>` or `fun(): auth(E) &S`).
			//
			// In assignment contexts the member is being written, not read out,
			// so no intersection applies.
			// For owned access, or members without nested references, this is a no-op.
			_, isRef := MaybeReferenceType(accessedType)
			if isRef && memberAuthorizationGatesReads(member) {
				cappedType := intersectReferencesWithAuthorization(
					checker.memoryGauge,
					resultingType,
					checker.grantedMemberAuthorization(member, accessedType, expression),
				)
				// Only record the cap when it actually narrowed the type,
				// so that the runtime converts the value exactly when needed.
				if cappedType != resultingType {
					resultingType = cappedType
					cappedNestedReferences = true
				}
			}
		}
	}

	return accessedType, resultingType, member, isOptional
}

// isReadableVariable returns true if the given variable can be read
// in the current location of the checker.
// Only applicable for variables with a ContainerType (e.g. constructor variables).
func (checker *Checker) isReadableVariable(variable *Variable) bool {

	if checker.Config.AccessCheckMode.IsReadableAccess(variable.Access) {
		return true
	}

	switch access := variable.Access.(type) {
	case PrimitiveAccess:
		if checker.containerTypes[variable.ContainerType] {
			return true
		}

		switch ast.PrimitiveAccess(access) {
		case ast.AccessContract:
			// If the variable allows access from the containing contract,
			// check if the current location is contained in the variable's contract

			contractType := containingContractKindedType(variable.ContainerType)
			if checker.containerTypes[contractType] {
				return true
			}

		case ast.AccessAccount:
			// If the variable allows access from the containing account,
			// check if the current location is the same as the variable's container location

			if locatedType, ok := variable.ContainerType.(LocatedType); ok {
				location := locatedType.GetLocation()
				if common.LocationsInSameAccount(checker.Location, location) {
					return true
				}

				memberAccountAccessHandler := checker.Config.MemberAccountAccessHandler
				if memberAccountAccessHandler != nil {
					return memberAccountAccessHandler(checker, location)
				}
			}
		}
	}

	return false
}

// isReadableMember returns true if the given member can be read from in the current location of the checker
func (checker *Checker) isReadableMember(accessedType Type, member *Member) bool {

	// TODO: check if this is correct
	if checker.Config.AccessCheckMode.IsReadableAccess(member.Access) {
		return true
	}

	switch access := member.Access.(type) {
	case PrimitiveAccess:
		if checker.containerTypes[member.ContainerType] {
			return true
		}

		switch ast.PrimitiveAccess(access) {
		case ast.AccessContract:
			// If the member allows access from the containing contract,
			// check if the current location is contained in the member's contract

			contractType := containingContractKindedType(member.ContainerType)
			if checker.containerTypes[contractType] {
				return true
			}

		case ast.AccessAccount:
			// If the member allows access from the containing account,
			// check if the current location is the same as the member's container location

			if locatedType, ok := member.ContainerType.(LocatedType); ok {
				location := locatedType.GetLocation()
				if common.LocationsInSameAccount(checker.Location, location) {
					return true
				}

				memberAccountAccessHandler := checker.Config.MemberAccountAccessHandler
				if memberAccountAccessHandler != nil {
					return memberAccountAccessHandler(checker, location)
				}
			}
		}

	case EntitlementSetAccess:
		switch ty := accessedType.(type) {
		case *OptionalType:
			return checker.isReadableMember(ty.Type, member)

		case *ReferenceType:
			// When accessing a member on a reference, the read is allowed if
			// the member's access permits the reference's authorization
			return member.Access.PermitsAccess(ty.Authorization)

		default:
			// When accessing a member on a non-reference, the read is always
			// allowed as an owned value is considered fully authorized
			return true
		}

	case *EntitlementMapAccess:
		// Accessing a member with mapping access, on a non-reference or a reference,
		// is always allowed. Only the entitlements granted through the mapping will differ
		return true
	}

	return false
}

func (checker *Checker) mapAccessToAuthorization(
	mappedAccess *EntitlementMapAccess,
	accessedType Type,
	pos ast.HasPosition,
) Access {

	switch accessedType := accessedType.(type) {
	case *ReferenceType:
		grantedAccess, err := mappedAccess.Image(
			checker.memoryGauge,
			accessedType.Authorization,
			pos,
		)
		if err != nil {
			checker.report(err)
			return UnauthorizedAccess
		}

		return grantedAccess

	case *OptionalType:
		return checker.mapAccessToAuthorization(
			mappedAccess,
			accessedType.Type,
			pos,
		)

	default:
		return UnauthorizedAccess
	}
}

// grantedMemberAuthorization returns the authorization
// that reading the given member through accessedType grants.
//
// For a field with mapping access it is the image
// of the accessed reference's authorization through the map,
// and for any other member it is the accessed reference's authorization itself.
//
// This is what caps the authorizations of the references nested in the member's type.
// A field with mapping access has to be capped with the mapped authorization,
// rather than with the accessed reference's raw authorization:
// given `entitlement mapping M { G -> E }`,
// reading an `access(mapping M)` field through an `auth(G)` reference grants `E`,
// so a nested `auth(E)` reference has to survive,
// even though `G` and `E` are disjoint.
//
// accessedType must be a reference type, or an optional reference type.
// An owned value's members are read as values rather than as references,
// so there is no authorization to grant,
// and in particular the result must not be used as a cap:
// it would strip every nested authorization
// from a member read from an owned value.
//
// NOTE: For a mapping whose image is not representable, an error is reported,
// so this must be called at most once per member access.
func (checker *Checker) grantedMemberAuthorization(
	member *Member,
	accessedType Type,
	pos ast.HasPosition,
) Access {
	outerRef, isRef := MaybeReferenceType(accessedType)
	if !isRef {
		panic(errors.NewUnreachableError())
	}

	if mappedAccess, ok := member.Access.(*EntitlementMapAccess); ok {
		return checker.mapAccessToAuthorization(mappedAccess, accessedType, pos)
	}

	return outerRef.Authorization
}

// memberAuthorizationGatesReads returns whether the authorization
// of the reference a member is read through is what gates reading it,
// and therefore also has to cap the references nested in the member's type.
//
// It does for entitlement-based and publicly readable members,
// which code anywhere may read.
//
// It does not for `access(self)`, `access(contract)` and `access(account)` members,
// which are gated by where the reading code is instead.
// Capping those would only restrict the declaring code itself,
// as no other code can read them through a reference of any authorization.
//
// An unspecified access modifier counts as publicly readable:
// that is what it means outside a restricted access check mode,
// and it is the conservative answer, because it caps rather than not.
func memberAuthorizationGatesReads(member *Member) bool {
	primitiveAccess, isPrimitiveAccess := member.Access.(PrimitiveAccess)
	if !isPrimitiveAccess {
		return true
	}

	switch ast.PrimitiveAccess(primitiveAccess) {
	case ast.AccessSelf,
		ast.AccessContract,
		ast.AccessAccount:
		return false

	default:
		return true
	}
}

// isWriteableMember returns true if the given member can be written to
// in the current location of the checker
func (checker *Checker) isWriteableMember(member *Member) bool {
	return checker.Config.AccessCheckMode.IsWriteableAccess(member.Access) ||
		checker.containerTypes[member.ContainerType]
}

// containingContractKindedType returns the containing contract-kinded type
// of the given type, if any.
//
// The given type itself might be the result.
func containingContractKindedType(t Type) CompositeKindedType {
	for {
		if compositeKindedType, ok := t.(CompositeKindedType); ok &&
			compositeKindedType.GetCompositeKind() == common.CompositeKindContract {

			return compositeKindedType
		}

		if containedType, ok := t.(ContainedType); ok {
			t = containedType.GetContainerType()
			continue
		}

		return nil
	}
}
