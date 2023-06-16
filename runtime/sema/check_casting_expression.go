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
	"github.com/onflow/cadence/runtime/errors"
)

func (checker *Checker) VisitCastingExpression(expression *ast.CastingExpression) Type {

	// Visit type annotation

	rightHandTypeAnnotation := checker.ConvertTypeAnnotation(expression.TypeAnnotation)
	checker.checkTypeAnnotation(rightHandTypeAnnotation, expression.TypeAnnotation)

	rightHandType := rightHandTypeAnnotation.Type

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

	checker.Elaboration.SetCastingExpressionTypes(
		expression,
		CastingExpressionTypes{
			StaticValueType: leftHandType,
			TargetType:      rightHandType,
		},
	)

	if leftHandType.IsResourceType() {

		if expression.Operation == ast.OperationFailableCast {

			// If the failable casted type is a resource, the failable cast expression
			// must occur in an optional binding, i.e. inside a variable declaration
			// as the if-statement test element

			if expression.ParentVariableDeclaration == nil ||
				expression.ParentVariableDeclaration.ParentIfStatement == nil {

				checker.report(
					&InvalidFailableResourceDowncastOutsideOptionalBindingError{
						Range: ast.NewRangeFromPositioned(checker.memoryGauge, expression),
					},
				)
			}

			if _, ok := expression.Expression.(*ast.IdentifierExpression); !ok {
				checker.report(
					&InvalidNonIdentifierFailableResourceDowncast{
						Range: ast.NewRangeFromPositioned(checker.memoryGauge, expression.Expression),
					},
				)
			}

			// NOTE: Counter-intuitively, do not *always* invalidate the casted expression:
			// As the failable cast must occur in an if-statement, the statement itself
			// takes care of the invalidation:
			// - In the then-branch, the cast succeeded, so the casted variable becomes invalidated
			// - Whereas in the else-branch, the cast failed, and the casted variable is still available

		} else {

			// For non-failable casts of a resource,
			// always record an invalidation

			checker.recordResourceInvalidation(
				leftHandExpression,
				leftHandType,
				ResourceInvalidationKindMoveDefinite,
			)
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
							Range:      ast.NewRangeFromPositioned(checker.memoryGauge, expression.TypeAnnotation),
						},
					)
				}
			} else {
				if rightHandType.IsResourceType() {
					checker.report(
						&AlwaysFailingResourceCastingTypeError{
							ValueType:  leftHandType,
							TargetType: rightHandType,
							Range:      ast.NewRangeFromPositioned(checker.memoryGauge, expression.TypeAnnotation),
						},
					)
				}
			}

			if !FailableCastCanSucceed(leftHandType, rightHandType) {

				checker.report(
					&TypeMismatchError{
						ActualType:   leftHandType,
						ExpectedType: rightHandType,
						Range:        ast.NewRangeFromPositioned(checker.memoryGauge, leftHandExpression),
					},
				)
			} else if checker.Config.ExtendedElaborationEnabled {
				checker.Elaboration.SetRuntimeCastTypes(
					expression,
					RuntimeCastTypes{
						Left:  leftHandType,
						Right: rightHandType,
					},
				)
			}
		}

		if expression.Operation == ast.OperationFailableCast {
			return &OptionalType{Type: rightHandType}
		}

		return rightHandType

	case ast.OperationCast:
		if checker.Config.ExtendedElaborationEnabled && !hasErrors {
			checker.Elaboration.SetStaticCastTypes(
				expression,
				CastTypes{
					ExprActualType: exprActualType,
					TargetType:     rightHandType,
					ExpectedType:   checker.expectedType,
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
func FailableCastCanSucceed(subType, superType Type) bool {

	// TODO: report impossible casts, e.g.
	//   - primitive/composite T -> composite U where T != U
	//   - array/dictionary where key or value cast is impossible
	//   => move checks from interpreter here

	switch typedSuperType := superType.(type) {
	case *ReferenceType:
		// if both are references, the failability of this cast depends entirely on the referenced types;
		// entitlements do not factor in here. To see why, consider a case where you have a reference to `R`
		// value that dynamically possesses entitlements `X` and `Z`. Statically, this would be typed as
		// `auth(X, Z) &R`. This is statically upcastable to `auth(Z) &R`, since this decreases permissions,
		// and any use case that requires a `Z` will also permit an `X & Z`. Then, we wish to cast this `auth(Z) &R`-typed
		// value to `auth(X | Y) &R`. Statically, it would appear that these two types are unrelated since the two entitlement
		// sets are disjoint, but this cast would succeed dynamically because the value does indeed posses an `X` entitlement
		// at runtime, which does indeed satisfy the requirement to have either an `X` or a `Y`.
		if typedSubType, ok := subType.(*ReferenceType); ok {
			return FailableCastCanSucceed(typedSubType.Type, typedSuperType.Type)
		}

	case *IntersectionType:

		switch typedSubType := subType.(type) {
		case *IntersectionType:
			// A intersection type `{Us}`
			// is a subtype of a intersection type `{Vs}`:
			// if the run-time type conforms to `Vs`
			// `Us` and `Vs` do *not* have to be subsets.
			return true

		case *CompositeType:

			// A type `T` is a subtype of a intersection type `{Us}`:
			//
			// When `T != AnyResource && T != AnyStruct && T != Any`:
			// if `T` conforms to `Us`.

			return typedSuperType.EffectiveIntersectionSet().
				IsSubsetOf(typedSubType.EffectiveInterfaceConformanceSet())

		}

		switch subType {
		case AnyResourceType, AnyStructType, AnyType:
			// A type `T` is a subtype of a intersection type `{Us}`:
			// if the run-time type conforms to `Vs`

			return true
		}

	case *CompositeType:
		switch subType.(type) {
		case *IntersectionType:

			// A intersection type `{Us}` is a subtype of a type `V`:
			// if the run-time type is V.
			return true
		}

	}

	switch superType {
	case AnyResourceType, AnyStructType:

		// A intersection type `{Us}` or a type `T` s a subtype of the type `AnyResource` / `AnyStruct`:
		// if `T` is `AnyType`, or `T` is a subtype of `AnyResource` / `AnyStruct`, or if `Us` are subtypes of `AnyResource` / `AnyStruct`

		innerSubtype := subType
		if intersectionSubType, ok := subType.(*IntersectionType); ok {
			// because the intersection's types are guaranteed to be the same kind, we only need to check the first one
			innerSubtype = intersectionSubType.Types[0]
		}

		return innerSubtype == AnyType ||
			IsSubType(innerSubtype, superType)

	case AnyType:
		return true

	}

	return true
}
