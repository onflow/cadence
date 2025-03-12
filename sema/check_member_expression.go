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

	checker.checkResourceMemberCapturingInFunction(expression, member, memberType)

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
func (checker *Checker) getReferenceType(typ Type, authorization Access) Type {
	if optionalType, ok := typ.(*OptionalType); ok {
		innerType := checker.getReferenceType(optionalType.Type, authorization)
		return NewOptionalType(checker.memoryGauge, innerType)
	}

	return NewReferenceType(checker.memoryGauge, authorization, typ)
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

		// in an statement like `a.b.c = x`, the entire statement itself
		// is an assignment, but the evaluation of the accessed exprssion itself (i.e. `a.b`)
		// is not, so we temporarily clear the `inAssignment` status here before restoring it later.
		accessedType = checker.withAssignment(false, func() Type {
			return checker.VisitExpression(accessedExpression, expression, nil)
		})
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

		member = resolver.Resolve(
			checker.memoryGauge,
			identifier,
			expression.Expression,
			checker.report,
		)
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
				suggestEntitlements: checker.Config.SuggestionsEnabled,
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

	// Note: For attachments, `self` is always a reference.
	// But we do not want to return a reference for `self.something`.
	// Otherwise, things like `destroy self.something` would become invalid.
	// Hence, special case `self`, and return a reference only if the member is not accessed via self.
	// i.e: `accessedSelfMember == nil`

	if accessedSelfMember == nil &&
		shouldReturnReference(accessedType, resultingType, isAssignment) &&
		member.DeclarationKind == common.DeclarationKindField {

		authorization := UnauthorizedAccess
		if mappedAccess, ok := member.Access.(*EntitlementMapAccess); ok {
			authorization = checker.mapAccessToAuthorization(mappedAccess, accessedType, expression)
		}

		resultingType = checker.getReferenceType(resultingType, authorization)
		returnReference = true
	}

	return accessedType, resultingType, member, isOptional
}

// isReadableMember returns true if the given member can be read from
// in the current location of the checker, along with the authorization with which the result can be used
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

			location := member.ContainerType.(LocatedType).GetLocation()
			if common.LocationsInSameAccount(checker.Location, location) {
				return true
			}

			memberAccountAccessHandler := checker.Config.MemberAccountAccessHandler
			if memberAccountAccessHandler != nil {
				return memberAccountAccessHandler(checker, location)
			}
		}

	case EntitlementSetAccess:
		switch ty := accessedType.(type) {
		case *OptionalType:
			return checker.isReadableMember(ty.Type, member)

		case *ReferenceType:
			// when accessing a member on a reference, the read is allowed if
			// the member's access permits the reference's authorization
			return member.Access.PermitsAccess(ty.Authorization)

		default:
			// when accessing a member on a non-reference, the read is always
			// allowed as an owned value is considered fully authorized
			return true
		}

	case *EntitlementMapAccess:
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
