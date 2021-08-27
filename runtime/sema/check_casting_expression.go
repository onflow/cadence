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

func (checker *Checker) VisitCastingExpression(expression *ast.CastingExpression) ast.Repr {

	// Visit type annotation

	rightHandTypeAnnotation := checker.ConvertTypeAnnotation(expression.TypeAnnotation)
	checker.checkTypeAnnotation(rightHandTypeAnnotation, expression.TypeAnnotation)

	rightHandType := rightHandTypeAnnotation.Type

	checker.Elaboration.CastingTargetTypes[expression] = rightHandType

	// visit the expression

	leftHandExpression := expression.Expression

	// In simple casting expression, type annotation is used to infer the type for the expression.
	var expectedType Type
	if expression.Operation == ast.OperationCast {
		expectedType = rightHandType
	}

	beforeErrors := len(checker.errors)

	leftHandType, exprActualType := checker.visitExpression(leftHandExpression, expectedType)

	hasErrors := len(checker.errors) > beforeErrors

	checker.Elaboration.CastingStaticValueTypes[expression] = leftHandType

	if leftHandType.IsResourceType() {
		checker.recordResourceInvalidation(
			leftHandExpression,
			leftHandType,
			ResourceInvalidationKindMoveDefinite,
		)

		// If the failable casted type is a resource, the failable cast expression
		// must occur in an optional binding, i.e. inside a variable declaration
		// as the if-statement test element

		if expression.Operation == ast.OperationFailableCast {

			if expression.ParentVariableDeclaration == nil ||
				expression.ParentVariableDeclaration.ParentIfStatement == nil {

				checker.report(
					&InvalidFailableResourceDowncastOutsideOptionalBindingError{
						Range: ast.NewRangeFromPositioned(expression),
					},
				)
			}

			if _, ok := expression.Expression.(*ast.IdentifierExpression); !ok {
				checker.report(
					&InvalidNonIdentifierFailableResourceDowncast{
						Range: ast.NewRangeFromPositioned(expression.Expression),
					},
				)
			}
		}
	}

	bothValid := !leftHandType.IsInvalidType() &&
		!rightHandType.IsInvalidType()

	switch expression.Operation {
	case ast.OperationFailableCast, ast.OperationForceCast:

		if bothValid {

			if leftHandType.IsResourceType() {
				if !rightHandType.IsResourceType() {
					checker.report(
						&AlwaysFailingNonResourceCastingTypeError{
							ValueType:  leftHandType,
							TargetType: rightHandType,
							Range:      ast.NewRangeFromPositioned(expression.TypeAnnotation),
						},
					)
				}
			} else {
				if rightHandType.IsResourceType() {
					checker.report(
						&AlwaysFailingResourceCastingTypeError{
							ValueType:  leftHandType,
							TargetType: rightHandType,
							Range:      ast.NewRangeFromPositioned(expression.TypeAnnotation),
						},
					)
				}
			}

			if !FailableCastCanSucceed(leftHandType, rightHandType) {

				checker.report(
					&TypeMismatchError{
						ActualType:   leftHandType,
						ExpectedType: rightHandType,
						Range:        ast.NewRangeFromPositioned(leftHandExpression),
					},
				)
			} else if IsSubType(leftHandType, rightHandType) {

				switch expression.Operation {
				case ast.OperationFailableCast:
					checker.hint(
						&AlwaysSucceedingFailableCastHint{
							ValueType:  leftHandType,
							TargetType: rightHandType,
							Range:      ast.NewRangeFromPositioned(expression),
						},
					)

				case ast.OperationForceCast:
					checker.hint(
						&AlwaysSucceedingForceCastHint{
							ValueType:  leftHandType,
							TargetType: rightHandType,
							Range:      ast.NewRangeFromPositioned(expression),
						},
					)

				default:
					panic(errors.NewUnreachableError())
				}
			}
		}

		if expression.Operation == ast.OperationFailableCast {
			return &OptionalType{Type: rightHandType}
		}

		return rightHandType

	case ast.OperationCast:
		// If there are errors in the lhs-expr, then the target type is considered as
		// the inferred-type of the expression. i.e: exprActualType == rightHandType
		// Then, it is not possible to determine whether the target type is redundant.
		// Therefore don't check for redundant casts, if there are errors.
		if !hasErrors &&
			IsCastRedundant(leftHandExpression, exprActualType, rightHandType, checker.expectedType) {
			checker.hint(
				&UnnecessaryCastHint{
					TargetType: rightHandType,
					Range:      ast.NewRangeFromPositioned(expression.TypeAnnotation),
				},
			)
		}

		return rightHandType

	default:
		panic(errors.NewUnreachableError())
	}
}

// FailableCastCanSucceed checks a failable (dynamic) cast, i.e. a cast that might succeed at run-time.
// It returns true if the cast from subType to superType could potentially succeed at run-time,
// and returns false if the cast will definitely always fail.
//
func FailableCastCanSucceed(subType, superType Type) bool {

	// TODO: report impossible casts, e.g.
	//   - primitive/composite T -> composite U where T != U
	//   - array/dictionary where key or value cast is impossible
	//   => move checks from interpreter here

	switch typedSuperType := superType.(type) {
	case *ReferenceType:
		// References types are only subtypes of reference types

		if typedSubType, ok := subType.(*ReferenceType); ok {
			// An authorized reference type `auth &T`
			// is a subtype of a reference type `&U` (authorized or non-authorized),
			// if `T` is a subtype of `U`

			if typedSubType.Authorized {
				return FailableCastCanSucceed(typedSubType.Type, typedSuperType.Type)
			}

			// An unauthorized reference type is not a subtype of an authorized reference type.
			// Not even dynamically.
			//
			// The holder of the reference may not gain more permissions.

			if typedSuperType.Authorized {
				return false
			}

			// A failable cast from an unauthorized reference type
			// to an unauthorized reference type
			// has the same semantics as a static/non-failable cast

			return IsSubType(subType, superType)
		}

	case *RestrictedType:

		switch typedSubType := subType.(type) {
		case *RestrictedType:

			restrictedSuperType := typedSuperType.Type
			switch restrictedSuperType {

			case AnyResourceType:
				// A restricted type `T{Us}`
				// is a subtype of a restricted type `AnyResource{Vs}`:
				//
				// When `T == AnyResource || T == Any`:
				// if the run-time type conforms to `Vs`
				//
				// When `T != AnyResource && T != Any`:
				// if `T` conforms to `Vs`.
				// `Us` and `Vs` do *not* have to be subsets.

				switch typedSubType.Type {
				case AnyResourceType, AnyType:
					return true
				default:
					if typedInnerSubType, ok := typedSubType.Type.(*CompositeType); ok {

						return IsSubType(typedInnerSubType, restrictedSuperType) &&
							typedSuperType.RestrictionSet().
								IsSubsetOf(typedInnerSubType.ExplicitInterfaceConformanceSet())
					}
				}

			case AnyStructType:
				// A restricted type `T{Us}`
				// is a subtype of a restricted type `AnyStruct{Vs}`:
				//
				// When `T == AnyStruct || T == Any`: if the run-time type conforms to `Vs`
				//
				// When `T != AnyStruct && T != Any`: if `T` conforms to `Vs`.
				// `Us` and `Vs` do *not* have to be subsets.

				switch typedSubType.Type {
				case AnyStructType, AnyType:
					return true
				default:
					if typedInnerSubType, ok := typedSubType.Type.(*CompositeType); ok {

						return IsSubType(typedInnerSubType, restrictedSuperType) &&
							typedSuperType.RestrictionSet().
								IsSubsetOf(typedInnerSubType.ExplicitInterfaceConformanceSet())
					}
				}

			case AnyType:
				// A restricted type `T{Us}`
				// is a subtype of a restricted type `Any{Vs}`:
				//
				// When `T == AnyResource || T == AnyStruct || T == Any`:
				// if the run-time type conforms to `Vs`
				//
				// When `T != AnyResource && T != AnyStruct && T != Any`:
				// if `T` conforms to `Vs`.
				// `Us` and `Vs` do *not* have to be subsets.

				switch typedSubType.Type {
				case AnyResourceType, AnyStructType, AnyType:
					return true
				default:
					if typedInnerSubType, ok := typedSubType.Type.(*CompositeType); ok {

						return IsSubType(typedInnerSubType, restrictedSuperType) &&
							typedSuperType.RestrictionSet().
								IsSubsetOf(typedInnerSubType.ExplicitInterfaceConformanceSet())
					}
				}

			default:

				// A restricted type `T{Us}`
				// is a subtype of a restricted type `V{Ws}`:
				//
				// When `T == AnyResource || T == AnyStruct || T == Any`:
				// if the run-time type is `V`.
				//
				// When `T != AnyResource && T != AnyStruct && T != Any`:
				// if `T == V`.
				// `Us` and `Ws` do *not* have to be subsets:
				// The owner may freely restrict and unrestrict.

				switch typedSubType.Type {
				case AnyResourceType, AnyStructType, AnyType:
					return true
				default:
					return typedSubType.Type.Equal(typedSuperType.Type)
				}
			}

		case *CompositeType:

			switch typedSuperType.Type {
			case AnyResourceType, AnyStructType, AnyType:

				// An unrestricted type `T`
				// is a subtype of a restricted type `AnyResource{Us}` / `AnyStruct{Us}` / `Any{Us}`:
				//
				// When `T != AnyResource && T != AnyStruct && T != Any`:
				// if `T` is a subtype of the restricted supertype,
				// and `T` conforms to `Us`.

				return IsSubType(typedSubType, typedSuperType.Type) &&
					typedSuperType.RestrictionSet().
						IsSubsetOf(typedSubType.ExplicitInterfaceConformanceSet())

			default:

				// An unrestricted type `T`
				// is a subtype of a restricted type `U{Vs}`:
				//
				// When `T != AnyResource && T != AnyStruct && T != Any`:
				// if `T == U`.

				return typedSubType.Equal(typedSuperType.Type)
			}

		}

		switch subType {
		case AnyResourceType, AnyStructType, AnyType:

			switch typedSuperType.Type {
			case AnyResourceType, AnyStructType, AnyType:

				// An unrestricted type `T`
				// is a subtype of a restricted type `AnyResource{Us}` / `AnyStruct{Us}` / `Any{Us}`:
				//
				// When `T == AnyResource || T == AnyStruct` || T == Any`:
				// if the run-time type conforms to `Vs`

				return true

			default:

				// An unrestricted type `T`
				// is a subtype of a restricted type `U{Vs}`:
				//
				// When `T == AnyResource || T == AnyStruct || T == Any`:
				// if the run-time type is U.

				// NOTE: inverse!

				return IsSubType(typedSuperType.Type, subType)
			}
		}

	case *CompositeType:

		switch typedSubType := subType.(type) {
		case *RestrictedType:

			// A restricted type `T{Us}`
			// is a subtype of an unrestricted type `V`:
			//
			// When `T == AnyResource || T == AnyStruct || T == Any`:
			// if the run-time type is V.
			//
			// When `T != AnyResource && T != AnyStruct && T != Any`:
			// if `T == V`.
			// The owner may freely unrestrict.

			switch typedSubType.Type {
			case AnyResourceType, AnyStructType, AnyType:
				return true

			default:
				return typedSubType.Type.Equal(typedSuperType)
			}
		}

	}

	switch superType {
	case AnyResourceType, AnyStructType:

		// A restricted type `T{Us}`
		// or unrestricted type `T`
		// is a subtype of the type `AnyResource` / `AnyStruct`:
		// if `T` is `AnyType`, or `T` is a subtype of `AnyResource` / `AnyStruct`.

		innerSubtype := subType
		if restrictedSubType, ok := subType.(*RestrictedType); ok {
			innerSubtype = restrictedSubType.Type
		}

		return innerSubtype == AnyType ||
			IsSubType(innerSubtype, superType)

	case AnyType:
		return true

	}

	return true
}

// IsCastRedundant checks whether a simple cast is redundant.
// Checks for two cases:
//    - Case I: Contextually expected type is same as casted type (target type).
//    - Case II: Expression is self typed, and is same as the casted type (target type).
func IsCastRedundant(expr ast.Expression, exprInferredType, targetType, expectedType Type) bool {
	if expectedType != nil && expectedType.Equal(targetType) {
		return true
	}

	checkCastVisitor := &CheckCastVisitor{}

	return checkCastVisitor.isCastRedundant(expr, exprInferredType, targetType)
}

type CheckCastVisitor struct {
	exprInferredType Type
	targetType       Type
}

var _ ast.ExpressionVisitor = &CheckCastVisitor{}

func (d *CheckCastVisitor) isCastRedundant(expr ast.Expression, exprInferredType, targetType Type) bool {
	prevInferredType := d.exprInferredType
	prevTargetType := d.targetType

	defer func() {
		d.exprInferredType = prevInferredType
		d.targetType = prevTargetType
	}()

	d.exprInferredType = exprInferredType
	d.targetType = targetType

	result := expr.AcceptExp(d)
	return result.(bool)
}

func (d *CheckCastVisitor) VisitBoolExpression(_ *ast.BoolExpression) ast.Repr {
	return d.isTypeRedundant(BoolType, d.targetType)
}

func (d *CheckCastVisitor) VisitNilExpression(_ *ast.NilExpression) ast.Repr {
	return d.isTypeRedundant(TypeOfNil, d.targetType)
}

func (d *CheckCastVisitor) VisitIntegerExpression(_ *ast.IntegerExpression) ast.Repr {
	// For integer expressions, default inferred type is `Int`.
	// So, if the target type is not `Int`, then the cast is not redundant.
	return d.isTypeRedundant(IntType, d.targetType)
}

func (d *CheckCastVisitor) VisitFixedPointExpression(expr *ast.FixedPointExpression) ast.Repr {
	if expr.Negative {
		// Default inferred type for fixed-point expressions with sign is `Fix64Type`.
		return d.isTypeRedundant(Fix64Type, d.targetType)
	}

	// Default inferred type for fixed-point expressions without sign is `UFix64Type`.
	return d.isTypeRedundant(UFix64Type, d.targetType)
}

func (d *CheckCastVisitor) VisitArrayExpression(expr *ast.ArrayExpression) ast.Repr {
	// If the target type is `ConstantSizedType`, then it is not redundant.
	// Because array literals are always inferred to be `VariableSizedType`,
	// unless specified.
	targetArrayType, ok := d.targetType.(*VariableSizedType)
	if !ok {
		return false
	}

	inferredArrayType, ok := d.exprInferredType.(ArrayType)
	if !ok {
		return false
	}

	for _, element := range expr.Values {
		// If at-least one element uses the target-type to infer the expression type,
		// then the casting is not redundant.
		if !d.isCastRedundant(
			element,
			inferredArrayType.ElementType(false),
			targetArrayType.ElementType(false),
		) {
			return false
		}
	}

	return true
}

func (d *CheckCastVisitor) VisitDictionaryExpression(expr *ast.DictionaryExpression) ast.Repr {
	targetDictionaryType, ok := d.targetType.(*DictionaryType)
	if !ok {
		return false
	}

	inferredDictionaryType, ok := d.exprInferredType.(*DictionaryType)
	if !ok {
		return false
	}

	for _, entry := range expr.Entries {
		// If at-least one key or value uses the target-type to infer the expression type,
		// then the casting is not redundant.
		if !d.isCastRedundant(
			entry.Key,
			inferredDictionaryType.KeyType,
			targetDictionaryType.KeyType,
		) {
			return false
		}

		if !d.isCastRedundant(
			entry.Value,
			inferredDictionaryType.ValueType,
			targetDictionaryType.ValueType,
		) {
			return false
		}
	}

	return true
}

func (d *CheckCastVisitor) VisitIdentifierExpression(_ *ast.IdentifierExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitInvocationExpression(_ *ast.InvocationExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitMemberExpression(_ *ast.MemberExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitIndexExpression(_ *ast.IndexExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitConditionalExpression(_ *ast.ConditionalExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitUnaryExpression(_ *ast.UnaryExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitBinaryExpression(_ *ast.BinaryExpression) ast.Repr {
	// Binary expressions are not straight-forward to check.
	// Hence skip checking redundant casts for now.
	return false
}

func (d *CheckCastVisitor) VisitFunctionExpression(_ *ast.FunctionExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitStringExpression(_ *ast.StringExpression) ast.Repr {
	return d.isTypeRedundant(StringType, d.targetType)
}

func (d *CheckCastVisitor) VisitCastingExpression(_ *ast.CastingExpression) ast.Repr {
	// This is already covered under Case-I: where expected type is same as casted type.
	// So skip checking it here to avid duplicate errors.
	return false
}

func (d *CheckCastVisitor) VisitCreateExpression(_ *ast.CreateExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitDestroyExpression(_ *ast.DestroyExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitReferenceExpression(_ *ast.ReferenceExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitForceExpression(_ *ast.ForceExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitPathExpression(_ *ast.PathExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) isTypeRedundant(exprType, targetType Type) bool {
	// If there is no expected type (e.g: var-decl with no type annotation),
	// then the simple-cast might be used as a way of marking the type of the variable.
	// Therefore it is ok for the target type to be a super-type.
	// But being the exact type as expression's type is redundant.
	// e.g:
	//   var x: Int8 = 5
	//   var y = x as Int8     // <-- not ok: `y` will be of type `Int8` with/without cast
	//   var y = x as Integer  // <-- ok	: `y` will be of type `Integer`
	return exprType != nil &&
		exprType.Equal(targetType)
}
