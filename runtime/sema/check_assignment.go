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
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

func (checker *Checker) VisitAssignmentStatement(assignment *ast.AssignmentStatement) (_ struct{}) {
	targetType, valueType := checker.checkAssignment(
		assignment,
		assignment.Target,
		assignment.Value,
		assignment.Transfer,
		false,
	)

	checker.Elaboration.SetAssignmentStatementTypes(
		assignment,
		AssignmentStatementTypes{
			ValueType:  valueType,
			TargetType: targetType,
		},
	)

	return
}

func (checker *Checker) checkAssignment(
	assignment ast.Statement,
	target, value ast.Expression,
	transfer *ast.Transfer,
	isSecondaryAssignment bool,
) (targetType, valueType Type) {

	targetType = checker.visitAssignmentValueType(target)

	// For all non-self member assignments,
	// require an explicit type annotation for references
	checkValue := checker.VisitExpression
	if checker.accessedSelfMember(target) == nil {
		checkValue = checker.VisitExpressionWithReferenceCheck
	}
	valueType = checkValue(value, assignment, targetType)

	// NOTE: Visiting the `value` checks the compatibility between value and target types.
	// Check for the *target* type, so that assignment using non-resource typed value (e.g. `nil`)
	// is possible

	checker.checkTransfer(transfer, targetType)

	// An assignment to a resource is generally invalid, as it would result in a loss

	if targetType.IsResourceType() {

		// However, there are two cases where it is allowed:
		//
		// 1. Force-assignment to an optional resource type.
		//
		// 2. Assignment to a `self` field in the initializer.
		//
		//    In this case the value that is assigned must be invalidated.
		//
		//    The check for a repeated assignment of a constant field after initialization
		//    is not part of this logic here, see `visitMemberExpressionAssignment`

		if transfer.Operation == ast.TransferOperationMoveForced {

			if _, ok := targetType.(*OptionalType); !ok {
				checker.report(
					&InvalidResourceAssignmentError{
						Range: ast.NewRangeFromPositioned(checker.memoryGauge, target),
					},
				)
			}

		} else {

			accessedSelfMember := checker.accessedSelfMember(target)

			if !isSecondaryAssignment &&
				(accessedSelfMember == nil ||
					checker.functionActivations.Current().InitializationInfo == nil) {

				checker.report(
					&InvalidResourceAssignmentError{
						Range: ast.NewRangeFromPositioned(checker.memoryGauge, target),
					},
				)
			}
		}
	}

	checker.enforceViewAssignment(assignment, target)

	checker.recordResourceInvalidation(
		value,
		valueType,
		ResourceInvalidationKindMoveDefinite,
	)

	checker.recordReferenceCreation(target, value)

	// Track nested resource moves.
	// Even though this is needed only for second value transfers, it is added here because:
	//  1) The second value transfers are checked as assignments,
	//     so the info needed (value's type etc.) is only available here.
	//     Adding it here covers second value transfers.
	//  2) Having it in assignment would cover all cases, even the ones that are statically rejected by the checker.
	//     So this would also act as a defensive check for all other cases.
	valueIsResource := valueType != nil && valueType.IsResourceType()
	if valueIsResource {
		checker.elaborateNestedResourceMoveExpression(value)
	}

	return
}

func (checker *Checker) rootOfAccessChain(target ast.Expression) (baseVariable *Variable, accessChain []Type) {
	var inAccessChain = true

	// seek the variable expression (if it exists) at the base of the access chain
	for inAccessChain {
		switch targetExp := target.(type) {
		case *ast.IdentifierExpression:
			baseVariable = checker.valueActivations.Find(targetExp.Identifier.Identifier)
			if baseVariable != nil {
				accessChain = append(accessChain, baseVariable.Type)
			}
			inAccessChain = false
		case *ast.IndexExpression:
			target = targetExp.TargetExpression
			indexExprTypes, ok := checker.Elaboration.IndexExpressionTypes(targetExp)
			var elementType Type
			if !ok {
				elementType = InvalidType
			} else {
				elementType = indexExprTypes.IndexedType.ElementType(true)
			}
			accessChain = append(accessChain, elementType)
		case *ast.MemberExpression:
			target = targetExp.Expression
			memberType, _, _, _ := checker.visitMember(targetExp, true)
			accessChain = append(accessChain, memberType)
		default:
			inAccessChain = false
		}
	}

	return
}

// We have to prevent any writes to references, since we cannot know where the value
// pointed to by the reference may have come from. Similarly, we can never safely assign
// to a resource; because resources are moved instead of copied, we cannot currently
// track the origin of a write target when it is a resource. Consider:
//
//	access(all) resource R {
//	  access(all) var x: Int
//	  init(x: Int) {
//	    self.x = x
//	  }
//	 }
//
//	view fun foo(_ f: @R): @R {
//	  let b <- f
//	  b.x = 3 // b was created in the current scope but modifies the resource value
//	  return <-b
//	}
func isWriteableInViewContext(t Type) bool {
	_, isReference := t.(*ReferenceType)
	return !isReference && !t.IsResourceType()
}

func (checker *Checker) enforceViewAssignment(assignment ast.Statement, target ast.Expression) {
	if !checker.CurrentPurityScope().EnforcePurity {
		return
	}

	baseVariable, accessChain := checker.rootOfAccessChain(target)

	// if the base of the access chain is not a variable, then we cannot make any static guarantees about
	// whether or not it is a local struct-kinded variable. E.g. in the case of `(b ? s1 : s2).x`, we can't
	// know whether `s1` or `s2` is being accessed here
	if baseVariable == nil {
		checker.ObserveImpureOperation(assignment)
		return
	}

	// `self` technically exists in param scope, but should still not be writeable outside of an initializer.
	// Within an initializer, writing to `self` is considered view:
	// whenever we call a constructor from inside a view scope, the value being constructed
	// (i.e. the one referred to by self in the constructor) is local to that scope,
	// so it is safe to create a new value from within a view scope.
	// This means that functions that just construct new values can technically be view
	// (in the same way that they pure are in a functional programming sense),
	// as long as they don't modify anything else while constructing those values.
	// They will still need a view annotation though (e.g. view init(...)).
	if baseVariable.DeclarationKind == common.DeclarationKindSelf {
		if checker.functionActivations.Current().InitializationInfo == nil {
			checker.ObserveImpureOperation(assignment)
		}
		return
	}

	// Check that all the types in the access chain are not resources or references
	for _, t := range accessChain {
		if !isWriteableInViewContext(t) {
			checker.ObserveImpureOperation(assignment)
			return
		}
	}

	// when the variable is neither a resource nor a reference,
	// we can write to if its activation depth is greater than or equal
	// to the depth at which the current purity scope was created;
	// i.e. if it is a parameter to the current function or was created within it
	if checker.CurrentPurityScope().ActivationDepth > baseVariable.ActivationDepth {
		checker.ObserveImpureOperation(assignment)
	}
}
func (checker *Checker) accessedSelfMember(expression ast.Expression) *Member {
	memberExpression, isMemberExpression := expression.(*ast.MemberExpression)
	if !isMemberExpression {
		return nil
	}

	identifierExpression, isIdentifierExpression := memberExpression.Expression.(*ast.IdentifierExpression)
	if !isIdentifierExpression {
		return nil
	}

	variable := checker.valueActivations.Find(identifierExpression.Identifier.Identifier)
	if variable == nil ||
		variable.DeclarationKind != common.DeclarationKindSelf {

		return nil
	}

	var members *StringMemberOrderedMap
	switch containerType := variable.Type.(type) {
	case *CompositeType:
		members = containerType.Members
	case *InterfaceType:
		members = containerType.Members
	case *TransactionType:
		members = containerType.Members
	case *ReferenceType:
		// self can only be a reference type if the container is an attachment, which is a composite
		members = containerType.Type.(*CompositeType).Members
	default:
		panic(errors.NewUnreachableError())
	}

	fieldName := memberExpression.Identifier.Identifier

	// caller handles the non-existing fieldNames
	member, _ := members.Get(fieldName)
	return member
}

func (checker *Checker) withAssignment(b bool, f func() Type) Type {
	inAssignment := checker.inAssignment
	checker.inAssignment = b
	defer func() {
		checker.inAssignment = inAssignment
	}()
	return f()
}

func (checker *Checker) visitAssignmentValueType(
	targetExpression ast.Expression,
) (targetType Type) {
	// Check the target is valid (e.g. identifier expression,
	// indexing expression, or member access expression)
	return checker.withAssignment(true, func() Type {
		if !IsValidAssignmentTargetExpression(targetExpression) {
			checker.report(
				&InvalidAssignmentTargetError{
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, targetExpression),
				},
			)

			return InvalidType
		}

		switch target := targetExpression.(type) {
		case *ast.IdentifierExpression:
			return checker.visitIdentifierExpressionAssignment(target)

		case *ast.IndexExpression:
			return checker.visitIndexExpressionAssignment(target)

		case *ast.MemberExpression:
			return checker.visitMemberExpressionAssignment(target)

		default:
			panic(errors.NewUnreachableError())
		}
	})
}

func (checker *Checker) visitIdentifierExpressionAssignment(
	target *ast.IdentifierExpression,
) (targetType Type) {
	identifier := target.Identifier

	// check identifier was declared before
	variable := checker.findAndCheckValueVariable(target, true)
	if variable == nil {
		return InvalidType
	}

	if variable.Type.IsResourceType() {
		checker.checkResourceVariableCapturingInFunction(variable, identifier)
	}

	// check identifier is not a constant
	if variable.IsConstant {
		checker.report(
			&AssignmentToConstantError{
				Name:  identifier.Identifier,
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, target),
			},
		)
	}

	return variable.Type
}

var mutateEntitledAccess = NewEntitlementSetAccess(
	[]*EntitlementType{MutateType},
	Disjunction,
)

var insertAndRemoveEntitledAccess = NewEntitlementSetAccess(
	[]*EntitlementType{InsertType, RemoveType},
	Conjunction,
)

func (checker *Checker) visitIndexExpressionAssignment(
	indexExpression *ast.IndexExpression,
) (elementType Type) {

	// in an statement like `ref.foo[i] = x`, the entire statement itself
	// is an assignment, but the evaluation of the index expression itself (i.e. `ref.foo`)
	// is not, so we temporarily clear the `inAssignment` status here before restoring it later.
	elementType = checker.withAssignment(false, func() Type {
		return checker.visitIndexExpression(indexExpression, true)
	})

	indexExprTypes, ok := checker.Elaboration.IndexExpressionTypes(indexExpression)
	if !ok {
		return InvalidType
	}

	indexedRefType, isReference := MaybeReferenceType(indexExprTypes.IndexedType)

	if isReference &&
		!mutateEntitledAccess.PermitsAccess(indexedRefType.Authorization) &&
		!insertAndRemoveEntitledAccess.PermitsAccess(indexedRefType.Authorization) {
		checker.report(&UnauthorizedReferenceAssignmentError{
			RequiredAccess: [2]Access{mutateEntitledAccess, insertAndRemoveEntitledAccess},
			FoundAccess:    indexedRefType.Authorization,
			Range:          ast.NewRangeFromPositioned(checker.memoryGauge, indexExpression),
		})
	}

	if elementType == nil {
		return InvalidType
	}

	return elementType
}

func (checker *Checker) visitMemberExpressionAssignment(
	target *ast.MemberExpression,
) (memberType Type) {

	_, memberType, member, isOptional := checker.visitMember(target, true)

	if member == nil {
		return InvalidType
	}

	if isOptional {
		checker.report(
			&UnsupportedOptionalChainingAssignmentError{
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, target),
			},
		)
	}

	if !checker.isWriteableMember(member) {
		checker.report(
			&InvalidAssignmentAccessError{
				Name:              member.Identifier.Identifier,
				ContainerType:     member.ContainerType,
				RestrictingAccess: member.Access,
				DeclarationKind:   member.DeclarationKind,
				Range:             ast.NewRangeFromPositioned(checker.memoryGauge, target.Identifier),
			},
		)
	}

	reportAssignmentToConstant := func() {
		checker.report(
			&AssignmentToConstantMemberError{
				Name:  target.Identifier.Identifier,
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, target.Identifier),
			},
		)
	}

	targetIsConstant := member.VariableKind != ast.VariableKindVariable

	// If this is an assignment to a `self` field, it needs special handling
	// depending on if the assignment is in an initializer or not

	accessedSelfMember := checker.accessedSelfMember(target)
	if accessedSelfMember != nil {

		functionActivation := checker.functionActivations.Current()

		// If this is an assignment to a `self` field in the initializer,
		// ensure it is only assigned once, to initialize it

		if functionActivation.InitializationInfo != nil {

			// If the function potentially returned before,
			// then the initialization is not definitive, and it must be ignored

			// NOTE: assignment can still be considered definitive if the function maybe halted

			if !functionActivation.ReturnInfo.MaybeReturned {

				// If the field is constant,
				// or it is variable and resource-kinded,
				// and it has already previously been initialized,
				// report an error for the repeated assignment / initialization
				//
				// Assigning to a variable, resource-kinded field is invalid,
				// because the initial value would get lost.

				initializedFieldMembers := functionActivation.InitializationInfo.InitializedFieldMembers

				if (targetIsConstant || memberType.IsResourceType()) &&
					initializedFieldMembers.Contains(accessedSelfMember) {

					checker.report(
						&FieldReinitializationError{
							Name:  target.Identifier.Identifier,
							Range: ast.NewRangeFromPositioned(checker.memoryGauge, target.Identifier),
						},
					)

				} else if !functionActivation.InitializationInfo.FieldMembers.Contains(accessedSelfMember) {
					// This member is not supposed to be initialized

					reportAssignmentToConstant()
				} else {
					// This is the initial assignment to the field, record it

					initializedFieldMembers.Add(accessedSelfMember)
				}
			}

		} else if targetIsConstant {

			// If this is an assignment outside the initializer,
			// an assignment to a constant field is invalid

			reportAssignmentToConstant()
		}

	} else if targetIsConstant {

		// The assignment is not to a `self` field. Report if there is an attempt
		// to assign to a constant field, which is always invalid,
		// independent of the location of the assignment (initializer or not)

		reportAssignmentToConstant()
	}

	if memberType.IsResourceType() {
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

	return
}

func IsValidAssignmentTargetExpression(expression ast.Expression) bool {
	switch expression := expression.(type) {
	case *ast.IdentifierExpression:
		return true

	case *ast.IndexExpression:
		return IsValidAssignmentTargetExpression(expression.TargetExpression)

	case *ast.MemberExpression:
		return IsValidAssignmentTargetExpression(expression.Expression)

	default:
		return false
	}
}
