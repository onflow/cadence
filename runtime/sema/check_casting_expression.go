package sema

import (
	"github.com/dapperlabs/cadence/runtime/ast"
	"github.com/dapperlabs/cadence/runtime/errors"
)

func (checker *Checker) VisitCastingExpression(expression *ast.CastingExpression) ast.Repr {

	leftHandExpression := expression.Expression
	leftHandType := leftHandExpression.Accept(checker).(Type)

	checker.Elaboration.CastingStaticValueTypes[expression] = leftHandType

	rightHandTypeAnnotation := checker.ConvertTypeAnnotation(expression.TypeAnnotation)
	checker.checkTypeAnnotation(rightHandTypeAnnotation, expression.TypeAnnotation)

	rightHandType := rightHandTypeAnnotation.Type

	checker.Elaboration.CastingTargetTypes[expression] = rightHandType

	if leftHandType.IsResourceType() {
		checker.recordResourceInvalidation(
			leftHandExpression,
			leftHandType,
			ResourceInvalidationKindMove,
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
		}
	}

	switch expression.Operation {
	case ast.OperationFailableCast, ast.OperationForceCast:

		if !leftHandType.IsInvalidType() &&
			!rightHandType.IsInvalidType() {

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

			if !FailableCanSucceed(leftHandType, rightHandType) {

				checker.report(
					&TypeMismatchError{
						ActualType:   leftHandType,
						ExpectedType: rightHandType,
						Range:        ast.NewRangeFromPositioned(leftHandExpression),
					},
				)
			}
		}

		if expression.Operation == ast.OperationFailableCast {
			return &OptionalType{Type: rightHandType}
		}

		return rightHandType

	case ast.OperationCast:
		if !leftHandType.IsInvalidType() &&
			!rightHandType.IsInvalidType() &&
			!checker.checkTypeCompatibility(leftHandExpression, leftHandType, rightHandType) {

			checker.report(
				&TypeMismatchError{
					ActualType:   leftHandType,
					ExpectedType: rightHandType,
					Range:        ast.NewRangeFromPositioned(leftHandExpression),
				},
			)
		}

		return rightHandType

	default:
		panic(errors.NewUnreachableError())
	}
}

// FailableCanSucceed checks a failable (dynamic) cast, i.e. a cast that might succeed at run-time.
// It returns true if the cast from subType to superType could potentially succeed at run-time,
// and returns false if the cast will definitely always fail.
//
func FailableCanSucceed(subType, superType Type) bool {

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
				return FailableCanSucceed(typedSubType.Type, typedSuperType.Type)
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

			switch restrictedSuperType := typedSuperType.Type.(type) {

			case *AnyResourceType:
				// A restricted  type `T{Us}`
				// is a subtype of a restricted type `AnyResource{Vs}`:
				//
				// When `T == AnyResource`: if the run-time type conforms to `Vs`
				//
				// When `T != AnyResource`: if `T` conforms to `Vs`.
				// `Us` and `Vs` do *not* have to be subsets.

				if _, ok := typedSubType.Type.(*AnyResourceType); ok {
					return true
				} else {
					if typedInnerSubType, ok := typedSubType.Type.(*CompositeType); ok {

						return IsSubType(typedInnerSubType, restrictedSuperType) &&
							typedSuperType.RestrictionSet().
								IsSubsetOf(typedInnerSubType.ConformanceSet())
					}
				}

			case *AnyStructType:
				// A restricted  type `T{Us}`
				// is a subtype of a restricted type `AnyStruct{Vs}`:
				//
				// When `T == AnyStruct`: if the run-time type conforms to `Vs`
				//
				// When `T != AnyStruct`: if `T` conforms to `Vs`.
				// `Us` and `Vs` do *not* have to be subsets.

				if _, ok := typedSubType.Type.(*AnyStructType); ok {
					return true
				} else {
					if typedInnerSubType, ok := typedSubType.Type.(*CompositeType); ok {

						return IsSubType(typedInnerSubType, restrictedSuperType) &&
							typedSuperType.RestrictionSet().
								IsSubsetOf(typedInnerSubType.ConformanceSet())
					}
				}

			default:

				// A restricted type `T{Us}`
				// is a subtype of a restricted type `V{Ws}`:
				//
				// When `T == AnyResource || T == AnyStruct`: if the run-time type is `V`.
				//
				// When `T != AnyResource && T != AnyStruct`: if `T == V`.
				// `Us` and `Ws` do *not* have to be subsets:
				// The owner may freely restrict and unrestrict.
				//

				switch typedSubType.Type.(type) {
				case *AnyResourceType, *AnyStructType:
					return true
				default:
					return typedSubType.Type.Equal(typedSuperType.Type)
				}
			}

		case *CompositeType:

			switch typedSuperType.Type.(type) {
			case *AnyResourceType, *AnyStructType:

				// An unrestricted type `T`
				// is a subtype of a restricted type `AnyResource{Us}` / `AnyStruct{Us}`:
				//
				// When `T != AnyResource && T != AnyStruct`:
				// if `T` is a subtype of the restricted supertype,
				// and `T` conforms to `Us`.

				return IsSubType(typedSubType, typedSuperType.Type) &&
					typedSuperType.RestrictionSet().
						IsSubsetOf(typedSubType.ConformanceSet())

			default:

				// An unrestricted type `T`
				// is a subtype of a restricted type `U{Vs}`:
				//
				// When `T != AnyResource && T != AnyStruct`: if `T == U`.

				return typedSubType.Equal(typedSuperType.Type)
			}

		case *AnyResourceType, *AnyStructType:

			// An unrestricted type `T`
			// is a subtype of a restricted type `AnyResource{Us}` / `AnyStruct{Us}`:
			//
			// When `T == AnyResource || T == AnyStruct`: if the run-time type conforms to `Vs`

			// An unrestricted type `T`
			// is a subtype of a restricted type `U{Vs}`:
			//
			// When `T == AnyResource || T == AnyStruct`: if the run-time type is U.

			// NOTE: inverse!

			return IsSubType(typedSuperType.Type, typedSubType)
		}

	case *CompositeType:

		switch typedSubType := subType.(type) {
		case *RestrictedType:

			// A restricted type `T{Us}`
			// is a subtype of an unrestricted type `V`:
			//
			// When `T != AnyResource && T != AnyStruct``: if `T == V`.
			// The owner may freely unrestrict.
			//
			// When `T == AnyResource || T == AnyStruct`: if the run-time type is V.

			switch typedSubType.Type.(type) {
			case *AnyResourceType, *AnyStructType:
				return true

			default:
				return typedSubType.Type.Equal(typedSuperType)
			}
		}

	case *AnyResourceType, *AnyStructType:

		// A restricted type `T{Us}`
		// or unrestricted type `T`
		// is a subtype of the type `AnyResource`/`AnyStruct`: always.

		return IsSubType(subType, superType)
	}

	return true
}
