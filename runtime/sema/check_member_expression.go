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

// NOTE: only called if the member expression is *not* an assignment
func (checker *Checker) VisitMemberExpression(expression *ast.MemberExpression) Type {
	accessedType, memberType, member, isOptional := checker.visitMember(expression, false)

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

	// If the member access is optional chaining, only wrap the result value
	// in an optional, if it is not already an optional value
	if isOptional {
		if _, ok := memberType.(*OptionalType); !ok {
			memberType = NewOptionalType(checker.memoryGauge, memberType)
		}
	}

	return memberType
}

// getReferenceType Returns a reference type to a given type.
// Reference to an optional should return an optional reference.
// This has to be done recursively for nested optionals.
// e.g.1: Given type T, this method returns &T.
// e.g.2: Given T?, this returns (&T)?
func (checker *Checker) getReferenceType(typ Type, substituteAuthorization bool, authorization Access) Type {
	if optionalType, ok := typ.(*OptionalType); ok {
		innerType := checker.getReferenceType(optionalType.Type, substituteAuthorization, authorization)
		return NewOptionalType(checker.memoryGauge, innerType)
	}

	auth := UnauthorizedAccess
	if substituteAuthorization && authorization != nil {
		auth = authorization
	}

	return NewReferenceType(checker.memoryGauge, auth, typ)
}

func shouldReturnReference(parentType, memberType Type, isAssignment bool) bool {
	if isAssignment {
		return false
	}

	if _, isReference := MaybeReferenceType(parentType); !isReference {
		return false
	}

	return memberType.ContainFieldsOrElements()
}

func MaybeReferenceType(typ Type) (*ReferenceType, bool) {
	unwrappedType := UnwrapOptionalType(typ)
	refType, isReference := unwrappedType.(*ReferenceType)
	return refType, isReference
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

	defer func() {
		checker.Elaboration.SetMemberExpressionMemberAccessInfo(
			expression,
			MemberAccessInfo{
				AccessedType:    accessedType,
				ResultingType:   resultingType,
				Member:          member,
				IsOptional:      isOptional,
				ReturnReference: returnReference,
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

		accessedType = checker.VisitExpression(accessedExpression, nil)
	}()

	checker.checkUnusedExpressionResourceLoss(accessedType, accessedExpression)

	// The access expression might have no name,
	// as the parser accepts invalid programs

	if expression.Identifier.Identifier == "" {
		return accessedType, resultingType, member, isOptional
	}

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

	getMemberForType := func(expressionType Type) {
		resolver, ok := expressionType.GetMembers()[identifier]
		if !ok {
			return
		}

		targetRange := ast.NewRangeFromPositioned(checker.memoryGauge, expression.Expression)
		member = resolver.Resolve(checker.memoryGauge, identifier, targetRange, checker.report)
		resultingType = member.TypeAnnotation.Type
	}

	// Get the member from the accessed value based
	// on the use of optional chaining syntax

	if expression.Optional {

		// If the member expression is using optional chaining,
		// check if the accessed type is optional

		if optionalExpressionType, ok := accessedType.(*OptionalType); ok {
			// The accessed type is optional, get the member from the wrapped type

			getMemberForType(optionalExpressionType.Type)
			isOptional = true
		} else {
			// Optional chaining was used on a non-optional type, report an error

			if !accessedType.IsInvalidType() {
				checker.report(
					&InvalidOptionalChainingError{
						Type:  accessedType,
						Range: ast.NewRangeFromPositioned(checker.memoryGauge, expression),
					},
				)
			}

			// NOTE: still try to get member for non-optional expression
			// to avoid spurious error that member does not exist,
			// even if the non-optional accessed type has the member

			getMemberForType(accessedType)
		}
	} else {
		// The member is accessed directly without optional chaining.
		// Get the member directly from the accessed type

		getMemberForType(accessedType)
		isOptional = false
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
					suggestMember: checker.Config.SuggestionsEnabled,
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
			accessedType,
			identifier,
			identifierStartPosition,
			identifierEndPosition,
		)
	}

	// Check access and report if inaccessible
	accessRange := func() ast.Range { return ast.NewRangeFromPositioned(checker.memoryGauge, expression) }
	isReadable, resultingAuthorization := checker.isReadableMember(accessedType, member, resultingType, accessRange)
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
				suggestEntitlements: checker.Config.SuggestionsEnabled,
				Range:               accessRange(),
			},
		)
	}

	// the resulting authorization was mapped through an entitlement map, so we need to substitute this new authorization into the resulting type
	// i.e. if the field was declared with `access(M) let x: auth(M) &T?`, and we computed that the output of the map would give entitlement `E`,
	// we substitute this entitlement in for the "variable" `M` to produce `auth(E) &T?`, the access with which the type is actually produced.
	// Equivalently, this can be thought of like generic instantiation.
	substituteConcreteAuthorization := func(resultingType Type) Type {
		switch ty := resultingType.(type) {
		case *ReferenceType:
			return NewReferenceType(checker.memoryGauge, resultingAuthorization, ty.Type)
		}
		return resultingType
	}

	shouldSubstituteAuthorization := !member.Access.Equal(resultingAuthorization)

	if shouldSubstituteAuthorization {
		resultingType = resultingType.Map(checker.memoryGauge, make(map[*TypeParameter]*TypeParameter), substituteConcreteAuthorization)
	}

	// Check that the member access is not to a function of resource type
	// outside of an invocation of it.
	//
	// This would result in a bound method for a resource, which is invalid.

	if !checker.inInvocation &&
		member.DeclarationKind == common.DeclarationKindFunction &&
		!accessedType.IsInvalidType() &&
		accessedType.IsResourceType() {

		checker.report(
			&ResourceMethodBindingError{
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, expression),
			},
		)
	}

	// If the member,
	//   1) is accessed via a reference, and
	//   2) is container-typed,
	// then the member type should also be a reference.

	// Note: For attachments, `self` is always a reference.
	// But we do not want to return a reference for `self.something`.
	// Otherwise, things like `destroy self.something` would become invalid.
	// Hence, special case `self`, and return a reference only if the member is not accessed via self.
	// i.e: `accessedSelfMember == nil`

	if accessedSelfMember == nil &&
		shouldReturnReference(accessedType, resultingType, isAssignment) &&
		member.DeclarationKind == common.DeclarationKindField {

		// Get a reference to the type
		resultingType = checker.getReferenceType(resultingType, shouldSubstituteAuthorization, resultingAuthorization)
		returnReference = true
	}

	return accessedType, resultingType, member, isOptional
}

// isReadableMember returns true if the given member can be read from
// in the current location of the checker, along with the authorzation with which the result can be used
func (checker *Checker) isReadableMember(accessedType Type, member *Member, resultingType Type, accessRange func() ast.Range) (bool, Access) {
	if checker.Config.AccessCheckMode.IsReadableAccess(member.Access) ||
		// only allow references unrestricted access to members in their own container that are not entitled
		// this prevents rights escalation attacks on entitlements
		(member.Access.IsPrimitiveAccess() && checker.containerTypes[member.ContainerType]) {

		if mappedAccess, isMappedAccess := member.Access.(*EntitlementMapAccess); isMappedAccess {
			return checker.mapAccess(mappedAccess, accessedType, resultingType, accessRange)
		}

		return true, member.Access
	}

	switch access := member.Access.(type) {
	case PrimitiveAccess:
		switch ast.PrimitiveAccess(access) {
		case ast.AccessContract:
			// If the member allows access from the containing contract,
			// check if the current location is contained in the member's contract

			contractType := containingContractKindedType(member.ContainerType)
			if checker.containerTypes[contractType] {
				return true, member.Access
			}

		case ast.AccessAccount:
			// If the member allows access from the containing account,
			// check if the current location is the same as the member's container location

			location := member.ContainerType.(LocatedType).GetLocation()
			if common.LocationsInSameAccount(checker.Location, location) {
				return true, member.Access
			}

			memberAccountAccessHandler := checker.Config.MemberAccountAccessHandler
			if memberAccountAccessHandler != nil {
				return memberAccountAccessHandler(checker, location), member.Access
			}
		}
	case EntitlementSetAccess:
		switch ty := accessedType.(type) {
		case *OptionalType:
			return checker.isReadableMember(ty.Type, member, resultingType, accessRange)
		case *ReferenceType:
			// when accessing a member on a reference, the read is allowed if
			// the member's access permits the reference's authorization
			return member.Access.PermitsAccess(ty.Authorization), member.Access
		default:
			// when accessing a member on a non-reference, the read is always
			// allowed as an owned value is considered fully authorized
			return true, member.Access
		}
	case *EntitlementMapAccess:
		return checker.mapAccess(access, accessedType, resultingType, accessRange)
	}

	return false, member.Access
}

func (checker *Checker) mapAccess(
	mappedAccess *EntitlementMapAccess,
	accessedType Type,
	resultingType Type,
	accessRange func() ast.Range,
) (bool, Access) {

	switch ty := accessedType.(type) {
	case *ReferenceType:
		// when accessing a member on a reference, the read is allowed, but the
		// granted entitlements are based on the image through the map of the reference's entitlements
		grantedAccess, err := mappedAccess.Image(ty.Authorization, accessRange)
		if err != nil {
			checker.report(err)
			// since we are already reporting an error that the map is unrepresentable,
			// pretend that the access succeeds to prevent a redundant access error report
			return true, UnauthorizedAccess
		}
		// when we are in an assignment statement,
		// we need full permissions to the map regardless of the input authorization of the reference
		// Consider:
		//
		//    entitlement X
		//    entitlement Y
		//    entitlement mapping M {
		//    	X -> Insert
		//    	Y -> Remove
		//    }
		//    struct S {
		//	    access(M) var member: auth(M) &[T]?
		//      ...
		//     }
		//
		//  If we were able to assign a `auth(Insert) &[T]` value to `ref.member` when `ref` has type `auth(X) &S`
		//  we could use this to then extract a `auth(Insert, Remove) &[T]` reference to that array by accessing `member`
		//  on an owned copy of `S`. As such, when in an assignment, we return the full codomain here as the "granted authorization"
		//  of the access expression, since the checker will later enforce that the incoming reference value is a subtype of that full codomain.
		if checker.inAssignment {
			return true, mappedAccess.Codomain()
		}
		return true, grantedAccess

	case *OptionalType:
		return checker.mapAccess(mappedAccess, ty.Type, resultingType, accessRange)

	default:
		if mappedAccess.Type == IdentityType {
			access := AllSupportedEntitlements(resultingType)
			if access != nil {
				return true, access
			}
		}

		// when accessing a member on a non-reference, the resulting mapped entitlement
		// should be the entire codomain of the map
		return true, mappedAccess.Codomain()
	}
}

func AllSupportedEntitlements(typ Type) Access {
	return allSupportedEntitlements(typ, false)
}

func allSupportedEntitlements(typ Type, isInnerType bool) Access {
	switch typ := typ.(type) {
	case *ReferenceType:
		return allSupportedEntitlements(typ.Type, true)
	case *OptionalType:
		return allSupportedEntitlements(typ.Type, true)
	case *FunctionType:
		// Entitlements must be returned only for function definitions.
		// Other than func-definitions, a member can be a function type in two ways:
		//  1) Function-typed field - Mappings are not allowed on function typed fields
		//  2) Function reference typed field - A function type inside a reference/optional-reference
		//     (i.e: an inner function type) should not be considered for entitlements.
		//
		if !isInnerType {
			return allSupportedEntitlements(typ.ReturnTypeAnnotation.Type, true)
		}
	case EntitlementSupportingType:
		supportedEntitlements := typ.SupportedEntitlements()
		if supportedEntitlements != nil && supportedEntitlements.Len() > 0 {
			access := EntitlementSetAccess{
				SetKind:      Conjunction,
				Entitlements: supportedEntitlements,
			}
			return access
		}
	}

	return nil
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
