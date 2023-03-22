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
	"github.com/onflow/cadence/runtime/errors"
)

// NOTE: only called if the member expression is *not* an assignment
func (checker *Checker) VisitMemberExpression(expression *ast.MemberExpression) Type {
	accessedType, member, isOptional := checker.visitMember(expression)

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

	memberType := member.TypeAnnotation.Type

	// If the member access is optional chaining, only wrap the result value
	// in an optional, if it is not already an optional value

	if isOptional {
		if _, ok := memberType.(*OptionalType); !ok {
			return &OptionalType{Type: memberType}
		}
	}

	return memberType
}

func (checker *Checker) visitMember(expression *ast.MemberExpression) (accessedType Type, member *Member, isOptional bool) {
	memberInfo, ok := checker.Elaboration.MemberExpressionMemberInfo(expression)
	if ok {
		return memberInfo.AccessedType, memberInfo.Member, memberInfo.IsOptional
	}

	defer func() {
		checker.Elaboration.SetMemberExpressionMemberInfo(
			expression,
			MemberInfo{
				AccessedType: accessedType,
				Member:       member,
				IsOptional:   isOptional,
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
		return accessedType, member, isOptional
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

		if !checker.isAvailableMember(expressionType, identifier) {
			return
		}

		targetRange := ast.NewRangeFromPositioned(checker.memoryGauge, expression.Expression)
		member = resolver.Resolve(checker.memoryGauge, identifier, targetRange, checker.report)
		if resolver.Mutating {
			if targetExpression, ok := accessedExpression.(*ast.MemberExpression); ok {
				// visitMember caches its result, so visiting the target expression again,
				// after it had been previously visited to get the resolver,
				// performs no computation
				_, subMember, _ := checker.visitMember(targetExpression)
				if subMember != nil && !checker.isMutatableMember(subMember) {
					checker.report(
						&ExternalMutationError{
							Name:            subMember.Identifier.Identifier,
							DeclarationKind: subMember.DeclarationKind,
							Range:           ast.NewRangeFromPositioned(checker.memoryGauge, targetRange),
							ContainerType:   subMember.ContainerType,
						},
					)
				}
			}
		}
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
	} else {

		if checker.PositionInfo != nil {
			checker.PositionInfo.recordMemberOccurrence(
				accessedType,
				identifier,
				identifierStartPosition,
				identifierEndPosition,
			)
		}

		// Check access and report if inaccessible

		if !checker.isReadableMember(accessedType, member) {
			checker.report(
				&InvalidAccessError{
					Name:              member.Identifier.Identifier,
					RestrictingAccess: member.Access,
					DeclarationKind:   member.DeclarationKind,
					Range:             ast.NewRangeFromPositioned(checker.memoryGauge, expression),
				},
			)
		}

		// Check that the member access is not to a function of resource type
		// outside of an invocation of it.
		//
		// This would result in a bound method for a resource, which is invalid.

		if !checker.inAssignment &&
			!checker.inInvocation &&
			member.DeclarationKind == common.DeclarationKindFunction &&
			!accessedType.IsInvalidType() &&
			accessedType.IsResourceType() {

			checker.report(
				&ResourceMethodBindingError{
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, expression),
				},
			)
		}
	}
	return accessedType, member, isOptional
}

// isReadableMember returns true if the given member can be read from
// in the current location of the checker
func (checker *Checker) isReadableMember(accessedType Type, member *Member) bool {
	if checker.Config.AccessCheckMode.IsReadableAccess(member.Access) ||
		checker.containerTypes[member.ContainerType] {

		return true
	}

	switch access := member.Access.(type) {
	case PrimitiveAccess:
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
		case *ReferenceType:
			// when accessing a member on a reference, the read is allowed if
			// the member's access permits the reference's authorization
			return member.Access.PermitsAccess(ty.Authorization)
		default:
			// when accessing a member on a non-reference, the read is always
			// allowed as an owned value is considered fully authorized
			return true
		}
	case EntitlementMapAccess:
		// ENTITLEMENT TODO: fill this out
		panic(errors.NewUnreachableError())
	}

	return false
}

// isWriteableMember returns true if the given member can be written to
// in the current location of the checker
func (checker *Checker) isWriteableMember(member *Member) bool {
	return checker.Config.AccessCheckMode.IsWriteableAccess(member.Access) ||
		checker.containerTypes[member.ContainerType]
}

// isMutatableMember returns true if the given member can be mutated
// in the current location of the checker. Currently equivalent to
// isWriteableMember above, but separate in case this changes
func (checker *Checker) isMutatableMember(member *Member) bool {
	return checker.isWriteableMember(member)
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
