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
	"github.com/onflow/cadence/runtime/errors"
)

func (checker *Checker) VisitBinaryExpression(expression *ast.BinaryExpression) ast.Repr {

	// The left-hand side is always evaluated.
	// However, the right-hand side might not necessarily be evaluated,
	// e.g. in boolean logic or in nil-coalescing

	operation := expression.Operation
	operationKind := binaryOperationKind(operation)

	// Of all binary expressions, only arithmetic expressions require the
	// rhsExpr and lhsExpr to be in the same type as the parent expression.
	// i.e: `var x: Int8 = a + b`. Here both `a` and `b` needs to be of the type `Int8`.
	//
	// This is also true, even if the expected type is `Int8?` e.g: `var x: Int8? = a + b`
	// So here we take the 'optional-type' out of the way.
	//
	// For the rest of the binary-expressions, this is not the case. e.g: logical-expressions.
	// i.e: `var x: Bool = a > b`. Here `a` and `b` can be anything, and doesn't have to be `Bool`.
	// For them, the type is inferred from the expressions themselves, and the compatibility check
	// is done based on the operation kind.

	var expectedType Type
	switch operationKind {
	case BinaryOperationKindArithmetic,
		BinaryOperationKindBitwise:
		expectedType = UnwrapOptionalType(checker.expectedType)
	}

	// Visit the expression, with contextually expected type. Use the expected type
	// only for inferring wherever possible, but do not check for compatibility.
	// Compatibility is checked separately for each operand kind.
	leftType := checker.VisitExpressionWithForceType(expression.Left, expectedType, false)

	leftIsInvalid := leftType.IsInvalidType()

	unsupportedOperation := func() Type {
		panic(&unsupportedOperation{
			kind:      common.OperationKindBinary,
			operation: operation,
			Range:     ast.NewRangeFromPositioned(expression),
		})
	}

	switch operationKind {
	case BinaryOperationKindArithmetic,
		BinaryOperationKindNonEqualityComparison,
		BinaryOperationKindEquality,
		BinaryOperationKindBitwise:

		// Right hand side will always be evaluated

		// Visit the expression, with contextually expected type.
		// Use the expected type only for inferring wherever possible,
		// but do not check for compatibility.
		// Compatibility is checked separately for each operand kind.
		//
		// If there is no contextually expected type,
		// then expect the right type to have the type of the left side.
		// For example, this allows a declaration like this to type-check:
		//
		// ```
		// let x = 1 as UInt8
		// let y = x + 1
		// ```
		//
		// Also, if there is a contextually expected type,
		// but the left type is a subtype and more specific (i.e not the same),
		// then use it instead as the expected type for the right type.
		// For example, this allows declarations like the following to type-check:
		//
		// ```
		// let x = 1 as UInt8
		// let y: Integer = x + 1
		// ```
		//
		// ```
		// let string = "this is a test"
		// let index = 1 as UInt8
		// let character = string[index + 1]
		// ```

		if expectedType == nil ||
			(leftType != expectedType && IsProperSubType(leftType, expectedType)) {

			expectedType = leftType
		}

		rightType := checker.VisitExpressionWithForceType(expression.Right, expectedType, false)

		rightIsInvalid := rightType.IsInvalidType()

		anyInvalid := leftIsInvalid || rightIsInvalid

		switch operationKind {
		case BinaryOperationKindArithmetic,
			BinaryOperationKindNonEqualityComparison,
			BinaryOperationKindBitwise:

			return checker.checkBinaryExpressionArithmeticOrNonEqualityComparisonOrBitwise(
				expression, operation, operationKind,
				leftType, rightType,
				leftIsInvalid, rightIsInvalid, anyInvalid,
			)

		case BinaryOperationKindEquality:
			return checker.checkBinaryExpressionEquality(
				expression, operation, operationKind,
				leftType, rightType,
				leftIsInvalid, rightIsInvalid, anyInvalid,
			)

		default:
			return unsupportedOperation()
		}

	case BinaryOperationKindBooleanLogic,
		BinaryOperationKindNilCoalescing:

		// The evaluation of the right-hand side is not guaranteed.
		// That means that resource invalidation and returns
		// are not definite, but only potential.

		rightType := checker.checkPotentiallyUnevaluated(func() Type {
			var expectedType Type
			if !leftIsInvalid {
				if optionalLeftType, ok := leftType.(*OptionalType); ok {
					expectedType = optionalLeftType.Type
				}
			}
			return checker.VisitExpressionWithForceType(expression.Right, expectedType, false)
		})

		rightIsInvalid := rightType.IsInvalidType()

		anyInvalid := leftIsInvalid || rightIsInvalid

		switch operationKind {
		case BinaryOperationKindBooleanLogic:
			return checker.checkBinaryExpressionBooleanLogic(
				expression, operation, operationKind,
				leftType, rightType,
				leftIsInvalid, rightIsInvalid, anyInvalid,
			)

		case BinaryOperationKindNilCoalescing:
			resultType := checker.checkBinaryExpressionNilCoalescing(
				expression, operation, operationKind,
				leftType, rightType,
				leftIsInvalid, rightIsInvalid, anyInvalid,
			)

			checker.Elaboration.BinaryExpressionResultTypes[expression] = resultType
			checker.Elaboration.BinaryExpressionRightTypes[expression] = rightType

			return resultType

		default:
			return unsupportedOperation()
		}
	default:
		return unsupportedOperation()
	}
}

func (checker *Checker) checkBinaryExpressionArithmeticOrNonEqualityComparisonOrBitwise(
	expression *ast.BinaryExpression,
	operation ast.Operation,
	operationKind BinaryOperationKind,
	leftType, rightType Type,
	leftIsInvalid, rightIsInvalid, anyInvalid bool,
) Type {
	// check both types are number/integer subtypes

	var expectedSuperType Type

	switch operationKind {
	case BinaryOperationKindArithmetic,
		BinaryOperationKindNonEqualityComparison:

		expectedSuperType = NumberType

	case BinaryOperationKindBitwise:
		expectedSuperType = IntegerType

	default:
		panic(errors.NewUnreachableError())
	}

	leftIsNumber := IsSameTypeKind(leftType, expectedSuperType)
	rightIsNumber := IsSameTypeKind(rightType, expectedSuperType)

	reportedInvalidOperands := false

	if !leftIsNumber && !rightIsNumber {
		if !anyInvalid {
			checker.report(
				&InvalidBinaryOperandsError{
					Operation: operation,
					LeftType:  leftType,
					RightType: rightType,
					Range:     ast.NewRangeFromPositioned(expression),
				},
			)
			reportedInvalidOperands = true
		}
	} else if !leftIsNumber {
		if !leftIsInvalid {
			checker.report(
				&InvalidBinaryOperandError{
					Operation:    operation,
					Side:         common.OperandSideLeft,
					ExpectedType: expectedSuperType,
					ActualType:   leftType,
					Range:        ast.NewRangeFromPositioned(expression.Left),
				},
			)
		}
	} else if !rightIsNumber {
		if !rightIsInvalid {
			checker.report(
				&InvalidBinaryOperandError{
					Operation:    operation,
					Side:         common.OperandSideRight,
					ExpectedType: expectedSuperType,
					ActualType:   rightType,
					Range:        ast.NewRangeFromPositioned(expression.Right),
				},
			)
		}
	}

	// check both types are equal

	if !reportedInvalidOperands &&
		!anyInvalid &&
		!checker.validNumericTypesForOperator(operationKind, leftType, rightType) {

		checker.report(
			&InvalidBinaryOperandsError{
				Operation: operation,
				LeftType:  leftType,
				RightType: rightType,
				Range:     ast.NewRangeFromPositioned(expression),
			},
		)
	}

	switch operationKind {
	case BinaryOperationKindArithmetic,
		BinaryOperationKindBitwise:

		return leftType

	case BinaryOperationKindNonEqualityComparison:
		return BoolType

	default:
		panic(errors.NewUnreachableError())
	}
}

// validNumericTypesForOperator checks whether the operand types are valid for the given operator.
// This method assumes that the two types: `leftType` and `rightType` are always numeric types.
func (checker *Checker) validNumericTypesForOperator(operationKind BinaryOperationKind, leftType, rightType Type) bool {
	switch operationKind {
	case BinaryOperationKindArithmetic,
		BinaryOperationKindBitwise:
		if !leftType.Equal(rightType) {
			return false
		}

		// So if it is not a super-type, then it's a valid numeric type for
		// arithmetic and bitwise operations.
		return !checker.isNumericSuperType(leftType)

	case BinaryOperationKindNonEqualityComparison:
		return leftType.Equal(rightType)

	default:
		// Ideally unreachable. However, return `false` instead of panicking to gracefully handle.
		return false
	}
}

func (*Checker) isNumericSuperType(numberType Type) bool {
	switch numberType {
	case NumberType,
		SignedNumberType,
		IntegerType,
		SignedIntegerType,
		FixedPointType,
		SignedFixedPointType:
		return true
	default:
		return false
	}
}

func (checker *Checker) checkBinaryExpressionEquality(
	expression *ast.BinaryExpression,
	operation ast.Operation,
	operationKind BinaryOperationKind,
	leftType, rightType Type,
	leftIsInvalid, rightIsInvalid, anyInvalid bool,
) (resultType Type) {

	resultType = BoolType

	if anyInvalid {
		return
	}

	if !AreCompatibleEquatableTypes(leftType, rightType) {
		checker.report(
			&InvalidBinaryOperandsError{
				Operation: operation,
				LeftType:  leftType,
				RightType: rightType,
				Range:     ast.NewRangeFromPositioned(expression),
			},
		)
	}

	checker.checkUnusedExpressionResourceLoss(leftType, expression.Left)
	checker.checkUnusedExpressionResourceLoss(rightType, expression.Right)

	return
}

func (checker *Checker) checkBinaryExpressionBooleanLogic(
	expression *ast.BinaryExpression,
	operation ast.Operation,
	operationKind BinaryOperationKind,
	leftType, rightType Type,
	leftIsInvalid, rightIsInvalid, anyInvalid bool,
) Type {
	// check both types are boolean subtypes

	leftIsBool := IsSameTypeKind(leftType, BoolType)
	rightIsBool := IsSameTypeKind(rightType, BoolType)

	if !leftIsBool && !rightIsBool {
		if !anyInvalid {
			checker.report(
				&InvalidBinaryOperandsError{
					Operation: operation,
					LeftType:  leftType,
					RightType: rightType,
					Range:     ast.NewRangeFromPositioned(expression),
				},
			)
		}
	} else if !leftIsBool {
		if !leftIsInvalid {
			checker.report(
				&InvalidBinaryOperandError{
					Operation:    operation,
					Side:         common.OperandSideLeft,
					ExpectedType: BoolType,
					ActualType:   leftType,
					Range:        ast.NewRangeFromPositioned(expression.Left),
				},
			)
		}
	} else if !rightIsBool {
		if !rightIsInvalid {
			checker.report(
				&InvalidBinaryOperandError{
					Operation:    operation,
					Side:         common.OperandSideRight,
					ExpectedType: BoolType,
					ActualType:   rightType,
					Range:        ast.NewRangeFromPositioned(expression.Right),
				},
			)
		}
	}

	return BoolType
}

func (checker *Checker) checkBinaryExpressionNilCoalescing(
	expression *ast.BinaryExpression,
	operation ast.Operation,
	operationKind BinaryOperationKind,
	leftType, rightType Type,
	leftIsInvalid, rightIsInvalid, anyInvalid bool,
) Type {
	leftOptional, leftIsOptional := leftType.(*OptionalType)

	if !leftIsInvalid {

		checker.recordResourceInvalidation(
			expression.Left,
			leftType,
			ResourceInvalidationKindMoveDefinite,
		)

		if !leftIsOptional {

			checker.report(
				&InvalidBinaryOperandError{
					Operation:    operation,
					Side:         common.OperandSideLeft,
					ExpectedType: &OptionalType{},
					ActualType:   leftType,
					Range:        ast.NewRangeFromPositioned(expression.Left),
				},
			)
		}
	}

	if leftIsInvalid || !leftIsOptional {
		return InvalidType
	}

	leftInner := leftOptional.Type

	if leftInner == NeverType {
		return rightType
	}
	canNarrow := false

	if !rightIsInvalid {

		if rightType.IsResourceType() {

			checker.report(
				&InvalidNilCoalescingRightResourceOperandError{
					Range: ast.NewRangeFromPositioned(expression.Right),
				},
			)
		}

		if !IsSubType(rightType, leftOptional) {

			checker.report(
				&InvalidBinaryOperandError{
					Operation:    operation,
					Side:         common.OperandSideRight,
					ExpectedType: leftOptional,
					ActualType:   rightType,
					Range:        ast.NewRangeFromPositioned(expression.Right),
				},
			)
		} else {
			canNarrow = IsSubType(rightType, leftInner)
		}
	}

	if !canNarrow {
		return leftOptional
	}
	return leftInner
}
