package sema

import (
	"github.com/dapperlabs/cadence/runtime/ast"
	"github.com/dapperlabs/cadence/runtime/common"
	"github.com/dapperlabs/cadence/runtime/errors"
)

func (checker *Checker) VisitBinaryExpression(expression *ast.BinaryExpression) ast.Repr {

	// The left-hand side is always evaluated.
	// However, the right-hand side might not necessarily be evaluated,
	// e.g. in boolean logic or in nil-coalescing

	leftType := expression.Left.Accept(checker).(Type)
	leftIsInvalid := leftType.IsInvalidType()

	operation := expression.Operation
	operationKind := binaryOperationKind(operation)

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
		BinaryOperationKindConcatenation:

		// Right hand side will always be evaluated

		rightType := expression.Right.Accept(checker).(Type)
		rightIsInvalid := rightType.IsInvalidType()

		anyInvalid := leftIsInvalid || rightIsInvalid

		switch operationKind {
		case BinaryOperationKindArithmetic,
			BinaryOperationKindNonEqualityComparison:

			return checker.checkBinaryExpressionArithmeticOrNonEqualityComparison(
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

		case BinaryOperationKindConcatenation:
			return checker.checkBinaryExpressionConcatenation(
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
			return expression.Right.Accept(checker).(Type)
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

func (checker *Checker) checkBinaryExpressionArithmeticOrNonEqualityComparison(
	expression *ast.BinaryExpression,
	operation ast.Operation,
	operationKind BinaryOperationKind,
	leftType, rightType Type,
	leftIsInvalid, rightIsInvalid, anyInvalid bool,
) Type {
	// check both types are number subtypes

	leftIsNumber := IsSubType(leftType, &NumberType{})
	rightIsNumber := IsSubType(rightType, &NumberType{})

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
		}
	} else if !leftIsNumber {
		if !leftIsInvalid {
			checker.report(
				&InvalidBinaryOperandError{
					Operation:    operation,
					Side:         common.OperandSideLeft,
					ExpectedType: &NumberType{},
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
					ExpectedType: &NumberType{},
					ActualType:   rightType,
					Range:        ast.NewRangeFromPositioned(expression.Right),
				},
			)
		}
	}

	// check both types are equal
	if !anyInvalid && !leftType.Equal(rightType) {
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
	case BinaryOperationKindArithmetic:
		return leftType

	case BinaryOperationKindNonEqualityComparison:
		return &BoolType{}
	}

	panic(errors.NewUnreachableError())
}

func (checker *Checker) checkBinaryExpressionEquality(
	expression *ast.BinaryExpression,
	operation ast.Operation,
	operationKind BinaryOperationKind,
	leftType, rightType Type,
	leftIsInvalid, rightIsInvalid, anyInvalid bool,
) (resultType Type) {

	resultType = &BoolType{}

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

	leftIsBool := IsSubType(leftType, &BoolType{})
	rightIsBool := IsSubType(rightType, &BoolType{})

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
					ExpectedType: &BoolType{},
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
					ExpectedType: &BoolType{},
					ActualType:   rightType,
					Range:        ast.NewRangeFromPositioned(expression.Right),
				},
			)
		}
	}

	return &BoolType{}
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
			ResourceInvalidationKindMove,
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
		return &InvalidType{}
	}

	leftInner := leftOptional.Type

	if _, ok := leftInner.(*NeverType); ok {
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

func (checker *Checker) checkBinaryExpressionConcatenation(
	expression *ast.BinaryExpression,
	operation ast.Operation,
	operationKind BinaryOperationKind,
	leftType, rightType Type,
	leftIsInvalid, rightIsInvalid, anyInvalid bool,
) Type {

	// check both types are concatenatable
	leftIsConcat := IsConcatenatableType(leftType)
	rightIsConcat := IsConcatenatableType(rightType)

	if !leftIsConcat && !rightIsConcat {
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
	} else if !leftIsConcat {
		if !leftIsInvalid {
			checker.report(
				&InvalidBinaryOperandError{
					Operation:    operation,
					Side:         common.OperandSideLeft,
					ExpectedType: rightType,
					ActualType:   leftType,
					Range:        ast.NewRangeFromPositioned(expression.Left),
				},
			)
		}
	} else if !rightIsConcat {
		if !rightIsInvalid {
			checker.report(
				&InvalidBinaryOperandError{
					Operation:    operation,
					Side:         common.OperandSideRight,
					ExpectedType: leftType,
					ActualType:   rightType,
					Range:        ast.NewRangeFromPositioned(expression.Right),
				},
			)
		}
	}

	// check both types are equal
	if !leftType.Equal(rightType) {
		checker.report(
			&InvalidBinaryOperandsError{
				Operation: operation,
				LeftType:  leftType,
				RightType: rightType,
				Range:     ast.NewRangeFromPositioned(expression),
			},
		)
	}

	return leftType
}
