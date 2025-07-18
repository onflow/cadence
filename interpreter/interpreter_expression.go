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

package interpreter

import (
	"math/big"
	"strings"
	"time"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/fixedpoint"
	"github.com/onflow/cadence/sema"
)

// assignmentGetterSetter returns a getter/setter function pair
// for the target expression
func (interpreter *Interpreter) assignmentGetterSetter(expression ast.Expression, locationRange LocationRange) getterSetter {
	switch expression := expression.(type) {
	case *ast.IdentifierExpression:
		return interpreter.identifierExpressionGetterSetter(expression, locationRange)

	case *ast.IndexExpression:
		if attachmentType, ok := interpreter.Program.Elaboration.AttachmentAccessTypes(expression); ok {
			return interpreter.typeIndexExpressionGetterSetter(expression, attachmentType, locationRange)
		}
		return interpreter.valueIndexExpressionGetterSetter(expression, locationRange)

	case *ast.MemberExpression:
		return interpreter.memberExpressionGetterSetter(expression, locationRange)

	default:
		return getterSetter{
			get: func(_ bool) Value {
				return interpreter.evalExpression(expression)
			},
			set: func(_ Value) {
				panic(errors.NewUnreachableError())
			},
		}
	}
}

// identifierExpressionGetterSetter returns a getter/setter function pair
// for the target identifier expression
func (interpreter *Interpreter) identifierExpressionGetterSetter(
	identifierExpression *ast.IdentifierExpression,
	locationRange LocationRange,
) getterSetter {
	identifier := identifierExpression.Identifier.Identifier
	variable := interpreter.FindVariable(identifier)

	return getterSetter{
		get: func(_ bool) Value {
			value := variable.GetValue(interpreter)
			interpreter.checkInvalidatedResourceUse(value, variable, identifier, identifierExpression)
			return value
		},
		set: func(value Value) {
			interpreter.startResourceTracking(value, variable, identifier, identifierExpression)
			variable.SetValue(
				interpreter,
				locationRange,
				value,
			)
		},
	}
}

func (interpreter *Interpreter) typeIndexExpressionGetterSetter(
	indexExpression *ast.IndexExpression,
	attachmentType sema.Type,
	locationRange LocationRange,
) getterSetter {
	target, ok := interpreter.evalExpression(indexExpression.TargetExpression).(TypeIndexableValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return getterSetter{
		target: target,
		get: func(_ bool) Value {
			CheckInvalidatedResourceOrResourceReference(target, locationRange, interpreter)
			return target.GetTypeKey(interpreter, locationRange, attachmentType)
		},
		set: func(_ Value) {
			CheckInvalidatedResourceOrResourceReference(target, locationRange, interpreter)
			// writing to composites with indexing syntax is not supported
			panic(errors.NewUnreachableError())
		},
	}
}

// valueIndexExpressionGetterSetter returns a getter/setter function pair
// for the target index expression
func (interpreter *Interpreter) valueIndexExpressionGetterSetter(
	indexExpression *ast.IndexExpression,
	locationRange LocationRange,
) getterSetter {

	// Use getter/setter functions to evaluate the target expression,
	// instead of evaluating it directly.
	//
	// In a swap statement, the left or right side may be an index expression,
	// and the indexed type (type of the target expression) may be a resource type.
	// In that case, the target expression must be considered as a nested resource move expression,
	// i.e. needs to be temporarily moved out (get)
	// and back in (set) after the index expression got evaluated.
	//
	// This is because the evaluation of the index expression
	// should not be able to access/move the target resource.
	//
	// For example, if a side is `a.b[c()]`, then `a.b` is the target expression.
	// If `a.b` is a resource, then `c()` should not be able to access/move it.

	targetExpression := indexExpression.TargetExpression
	targetGetterSetter := interpreter.assignmentGetterSetter(targetExpression, locationRange)
	const allowMissing = false
	target, ok := targetGetterSetter.get(allowMissing).(ValueIndexableValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// Evaluate, transfer, and convert the indexing value,
	// as it is essentially an "argument" of the get/set operation

	elaboration := interpreter.Program.Elaboration

	indexExpressionTypes, ok := elaboration.IndexExpressionTypes(indexExpression)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	indexedType := indexExpressionTypes.IndexedType
	indexingType := indexExpressionTypes.IndexingType

	transferredIndexingValue := TransferAndConvert(
		interpreter,
		interpreter.evalExpression(indexExpression.IndexingExpression),
		indexingType,
		indexedType.IndexingType(),
		LocationRange{
			Location:    interpreter.Location,
			HasPosition: indexExpression.IndexingExpression,
		},
	)

	isTargetNestedResourceMove := elaboration.IsNestedResourceMoveExpression(targetExpression)
	if isTargetNestedResourceMove {
		targetGetterSetter.set(target)
	}

	// Normally, moves of nested resources (e.g `let r <- rs[0]`) are statically rejected.
	//
	// However, there are cases in which we do allow moves of nested resources:
	//
	// - In a swap statement (e.g. `rs[0] <-> rs[1]`)
	// - In a variable declaration with two values/assignments (e.g. `let r <- rs["foo"] <- nil`)
	//
	// In both cases we know that a move of the nested resource is immediately followed by a replacement.
	// This notion of an expression that moves a nested resource is tracked in the elaboration.
	//
	// When indexing is a move of a nested resource, we need to remove the key/value from the container.
	// However, for some containers, like arrays, the removal influences other values in the container.
	// In case of an array, the removal of an element shifts all following elements.
	//
	// A removal alone would thus result in subsequent code being executed incorrectly.
	// For example, in the case where a swap operation through indexing is performed on the same array,
	// e.g. `rs[0] <-> rs[1]`, once the first removal was performed, the second operates on a modified container.
	//
	// Prevent this problem by temporarily writing a placeholder value after the removal.
	// Only perform the replacement with a placeholder in the case of a nested resource move.
	// We know that in that case the get operation will be followed by a set operation,
	// which will replace the temporary placeholder.

	isNestedResourceMove := elaboration.IsNestedResourceMoveExpression(indexExpression)

	var get func(allowMissing bool) Value

	if isNestedResourceMove {
		get = func(_ bool) Value {
			CheckInvalidatedResourceOrResourceReference(target, locationRange, interpreter)
			value := target.RemoveKey(interpreter, locationRange, transferredIndexingValue)
			target.InsertKey(interpreter, locationRange, transferredIndexingValue, placeholder)
			return value
		}
	} else {
		get = func(_ bool) Value {
			CheckInvalidatedResourceOrResourceReference(target, locationRange, interpreter)
			value := target.GetKey(interpreter, locationRange, transferredIndexingValue)

			// If the indexing value is a reference, then return a reference for the resulting value.
			return interpreter.maybeGetReference(indexExpression, value)
		}
	}

	return getterSetter{
		target: target,
		get:    get,
		set: func(value Value) {
			CheckInvalidatedResourceOrResourceReference(target, locationRange, interpreter)
			target.SetKey(interpreter, locationRange, transferredIndexingValue, value)
		},
	}
}

// memberExpressionGetterSetter returns a getter/setter function pair
// for the target member expression
func (interpreter *Interpreter) memberExpressionGetterSetter(
	memberExpression *ast.MemberExpression,
	locationRange LocationRange,
) getterSetter {

	target := interpreter.evalExpression(memberExpression.Expression)
	identifier := memberExpression.Identifier.Identifier

	isNestedResourceMove := interpreter.Program.Elaboration.IsNestedResourceMoveExpression(memberExpression)

	memberAccessInfo, ok := interpreter.Program.Elaboration.MemberExpressionMemberAccessInfo(memberExpression)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return getterSetter{
		target: target,
		get: func(allowMissing bool) Value {

			interpreter.checkMemberAccess(memberExpression, target, locationRange)

			isOptional := memberExpression.Optional

			if isOptional {
				switch typedTarget := target.(type) {
				case NilValue:
					return typedTarget

				case *SomeValue:
					target = typedTarget.InnerValue()

				default:
					panic(errors.NewUnreachableError())
				}
			}

			var resultValue Value
			if isNestedResourceMove {
				resultValue = target.(MemberAccessibleValue).RemoveMember(interpreter, locationRange, identifier)
			} else {
				resultValue = getMember(interpreter, target, locationRange, identifier)
			}

			if resultValue == nil && !allowMissing {
				panic(&UseBeforeInitializationError{
					Name:          identifier,
					LocationRange: locationRange,
				})
			}

			// If the member access is optional chaining, only wrap the result value
			// in an optional, if it is not already an optional value

			if isOptional {
				if _, ok := resultValue.(OptionalValue); !ok {
					resultValue = NewSomeValueNonCopying(interpreter, resultValue)
				}
			}

			// Return a reference, if the member is accessed via a reference.
			// This is pre-computed at the checker.
			if memberAccessInfo.ReturnReference {
				// Get a reference to the value
				resultValue = getReferenceValue(
					interpreter,
					resultValue,
					memberAccessInfo.ResultingType,
					locationRange,
				)
			}

			return resultValue
		},
		set: func(value Value) {
			interpreter.checkMemberAccess(memberExpression, target, locationRange)
			setMember(interpreter, target, locationRange, identifier, value)
		},
	}
}

// getReferenceValue Returns a reference to a given value.
// Reference to an optional should return an optional reference.
// This has to be done recursively for nested optionals.
// e.g.1: Given type T, this method returns &T.
// e.g.2: Given T?, this returns (&T)?
func getReferenceValue(
	context GetReferenceContext,
	value Value,
	resultType sema.Type,
	locationRange LocationRange,
) Value {
	return CreateReferenceValue(context, resultType, value, locationRange, true)
}

func (interpreter *Interpreter) checkMemberAccess(
	memberExpression *ast.MemberExpression,
	target Value,
	locationRange LocationRange,
) {

	CheckInvalidatedResourceOrResourceReference(target, locationRange, interpreter)

	memberInfo, _ := interpreter.Program.Elaboration.MemberExpressionMemberAccessInfo(memberExpression)
	expectedType := memberInfo.AccessedType

	CheckMemberAccessTargetType(
		interpreter,
		target,
		expectedType,
		locationRange,
	)
}

func CheckMemberAccessTargetType(
	context ValueStaticTypeContext,
	target Value,
	expectedType sema.Type,
	locationRange LocationRange,
) {
	switch expectedType := expectedType.(type) {
	case *sema.TransactionType:
		// TODO: maybe also check transactions.
		//   they are composites with a type ID which has an empty qualified ID, i.e. no type is available

		return

	case *sema.CompositeType:
		// TODO: also check built-in values.
		//   blocked by standard library values (RLP, BLS, etc.),
		//   which are implemented as contracts, but currently do not have their type registered

		if expectedType.Location == nil {
			return
		}
	}

	// NOTE: accesses of (optional) storage reference values
	// are already checked in StorageReferenceValue.dereference
	_, isStorageReference := target.(*StorageReferenceValue)
	if !isStorageReference {
		if optional, ok := target.(*SomeValue); ok {
			_, isStorageReference = optional.value.(*StorageReferenceValue)
		}
	}
	if isStorageReference {
		return
	}

	targetStaticType := target.StaticType(context)

	if _, ok := expectedType.(*sema.OptionalType); ok {
		if _, ok := targetStaticType.(*OptionalStaticType); !ok {
			targetSemaType := MustConvertStaticToSemaType(targetStaticType, context)

			panic(&MemberAccessTypeError{
				ExpectedType:  expectedType,
				ActualType:    targetSemaType,
				LocationRange: locationRange,
			})
		}
	}

	if !IsSubTypeOfSemaType(context, targetStaticType, expectedType) {
		targetSemaType := MustConvertStaticToSemaType(targetStaticType, context)

		panic(&MemberAccessTypeError{
			ExpectedType:  expectedType,
			ActualType:    targetSemaType,
			LocationRange: locationRange,
		})
	}
}

func (interpreter *Interpreter) VisitIdentifierExpression(expression *ast.IdentifierExpression) Value {
	name := expression.Identifier.Identifier
	variable := interpreter.FindVariable(name)
	value := variable.GetValue(interpreter)

	interpreter.checkInvalidatedResourceUse(value, variable, name, expression)

	return value
}

func (interpreter *Interpreter) evalExpression(expression ast.Expression) Value {
	result := ast.AcceptExpression[Value](expression, interpreter)
	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: expression,
	}
	CheckInvalidatedResourceOrResourceReference(
		result,
		locationRange,
		interpreter,
	)
	return result
}

func CheckInvalidatedResourceOrResourceReference(
	value Value,
	locationRange LocationRange,
	context ValueStaticTypeContext,
) {
	// Unwrap SomeValue, to access references wrapped inside optionals.
	someValue, isSomeValue := value.(*SomeValue)
	for isSomeValue && someValue.value != nil {
		value = someValue.value
		someValue, isSomeValue = value.(*SomeValue)
	}

	switch value := value.(type) {
	case ResourceKindedValue:
		if value.isInvalidatedResource(context) {
			panic(&InvalidatedResourceError{
				LocationRange: locationRange,
			})
		}
	case *EphemeralReferenceValue:
		if value.Value == nil {
			panic(&InvalidatedResourceReferenceError{
				LocationRange: locationRange,
			})
		} else {
			// If the value is there, check whether the referenced value is an invalidated one.
			// This step is not really needed, since reference tracking is supposed to clear the
			// `value.Value` if the referenced-value was moved/deleted.
			// However, have this as a second layer of defensive.
			CheckInvalidatedResourceOrResourceReference(
				value.Value,
				locationRange,
				context,
			)
		}
	}
}

func (interpreter *Interpreter) VisitBinaryExpression(expression *ast.BinaryExpression) Value {

	leftValue := interpreter.evalExpression(expression.Left)

	// We make this a thunk so that we can skip computing it for certain short-circuiting operations
	rightValue := func() Value {
		return interpreter.evalExpression(expression.Right)
	}

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: expression,
	}

	error := func(right Value) {
		panic(&InvalidOperandsError{
			Operation:     expression.Operation,
			LeftType:      leftValue.StaticType(interpreter),
			RightType:     right.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	switch expression.Operation {
	case ast.OperationPlus:
		left, leftOk := leftValue.(NumberValue)
		right, rightOk := rightValue().(NumberValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.Plus(interpreter, right, locationRange)

	case ast.OperationMinus:
		left, leftOk := leftValue.(NumberValue)
		right, rightOk := rightValue().(NumberValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.Minus(interpreter, right, locationRange)

	case ast.OperationMod:
		left, leftOk := leftValue.(NumberValue)
		right, rightOk := rightValue().(NumberValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.Mod(interpreter, right, locationRange)

	case ast.OperationMul:
		left, leftOk := leftValue.(NumberValue)
		right, rightOk := rightValue().(NumberValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.Mul(interpreter, right, locationRange)

	case ast.OperationDiv:
		left, leftOk := leftValue.(NumberValue)
		right, rightOk := rightValue().(NumberValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.Div(interpreter, right, locationRange)

	case ast.OperationBitwiseOr:
		left, leftOk := leftValue.(IntegerValue)
		right, rightOk := rightValue().(IntegerValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.BitwiseOr(interpreter, right, locationRange)

	case ast.OperationBitwiseXor:
		left, leftOk := leftValue.(IntegerValue)
		right, rightOk := rightValue().(IntegerValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.BitwiseXor(interpreter, right, locationRange)

	case ast.OperationBitwiseAnd:
		left, leftOk := leftValue.(IntegerValue)
		right, rightOk := rightValue().(IntegerValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.BitwiseAnd(interpreter, right, locationRange)

	case ast.OperationBitwiseLeftShift:
		left, leftOk := leftValue.(IntegerValue)
		right, rightOk := rightValue().(IntegerValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.BitwiseLeftShift(interpreter, right, locationRange)

	case ast.OperationBitwiseRightShift:
		left, leftOk := leftValue.(IntegerValue)
		right, rightOk := rightValue().(IntegerValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.BitwiseRightShift(interpreter, right, locationRange)

	case ast.OperationLess,
		ast.OperationLessEqual,
		ast.OperationGreater,
		ast.OperationGreaterEqual:
		return interpreter.testComparison(leftValue, rightValue(), expression)

	case ast.OperationEqual:
		return TestValueEqual(
			interpreter,
			LocationRange{
				Location:    interpreter.Location,
				HasPosition: expression,
			},
			leftValue,
			rightValue(),
		)

	case ast.OperationNotEqual:
		return !TestValueEqual(
			interpreter,
			LocationRange{
				Location:    interpreter.Location,
				HasPosition: expression,
			},
			leftValue,
			rightValue(),
		)

	case ast.OperationOr:
		// interpret the left-hand side
		left, leftOk := leftValue.(BoolValue)
		if !leftOk {
			// ok to evaluate the right value here because we will abort afterwards
			error(rightValue())
		}
		// only interpret right-hand side if left-hand side is false
		if left {
			return left
		}

		// after interpreting the left-hand side,
		// interpret the right-hand side
		right, rightOk := rightValue().(BoolValue)
		if !rightOk {
			error(right)
		}
		return right

	case ast.OperationAnd:
		// interpret the left-hand side
		left, leftOk := leftValue.(BoolValue)
		if !leftOk {
			// ok to evaluate the right value here because we will abort afterwards
			error(rightValue())
		}
		// only interpret right-hand side if left-hand side is true
		if !left {
			return left
		}

		// after interpreting the left-hand side,
		// interpret the right-hand side
		right, rightOk := rightValue().(BoolValue)
		if !rightOk {
			error(right)
		}
		return right

	case ast.OperationNilCoalesce:
		locationRange := LocationRange{
			Location:    interpreter.Location,
			HasPosition: expression,
		}

		// only evaluate right-hand side if left-hand side is nil
		if some, ok := leftValue.(*SomeValue); ok {
			return some.InnerValue()
		}

		value := rightValue()

		binaryExpressionTypes := interpreter.Program.Elaboration.BinaryExpressionTypes(expression)
		rightType := binaryExpressionTypes.RightType
		resultType := binaryExpressionTypes.ResultType

		// NOTE: important to convert both any and optional
		return ConvertAndBox(interpreter, locationRange, value, rightType, resultType)
	}

	panic(&unsupportedOperation{
		kind:      common.OperationKindBinary,
		operation: expression.Operation,
		Range:     ast.NewUnmeteredRangeFromPositioned(expression),
	})
}

func TestValueEqual(
	context ValueComparisonContext,
	locationRange LocationRange,
	left, right Value,
) BoolValue {
	left = Unbox(left)

	right = Unbox(right)

	leftEquatable, ok := left.(EquatableValue)
	if !ok {
		return FalseValue
	}

	return BoolValue(
		leftEquatable.Equal(
			context,
			locationRange,
			right,
		),
	)
}

func (interpreter *Interpreter) testComparison(left, right Value, expression *ast.BinaryExpression) BoolValue {
	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: expression,
	}

	leftComparable, leftOk := left.(ComparableValue)
	rightComparable, rightOk := right.(ComparableValue)

	if !leftOk || !rightOk {
		panic(&InvalidOperandsError{
			Operation:     expression.Operation,
			LeftType:      left.StaticType(interpreter),
			RightType:     right.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	switch expression.Operation {
	case ast.OperationLess:
		return leftComparable.Less(
			interpreter,
			rightComparable,
			locationRange,
		)

	case ast.OperationLessEqual:
		return leftComparable.LessEqual(
			interpreter,
			rightComparable,
			locationRange,
		)

	case ast.OperationGreater:
		return leftComparable.Greater(
			interpreter,
			rightComparable,
			locationRange,
		)

	case ast.OperationGreaterEqual:
		return leftComparable.GreaterEqual(
			interpreter,
			rightComparable,
			locationRange,
		)

	default:
		panic(&unsupportedOperation{
			kind:      common.OperationKindBinary,
			operation: expression.Operation,
			Range:     ast.NewUnmeteredRangeFromPositioned(expression),
		})
	}
}

func (interpreter *Interpreter) VisitUnaryExpression(expression *ast.UnaryExpression) Value {
	value := interpreter.evalExpression(expression.Expression)

	switch expression.Operation {
	case ast.OperationNegate:
		boolValue, ok := value.(BoolValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		return boolValue.Negate(interpreter)

	case ast.OperationMinus:
		integerValue, ok := value.(NumberValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		return integerValue.Negate(
			interpreter,
			LocationRange{
				Location:    interpreter.Location,
				HasPosition: expression,
			},
		)

	case ast.OperationMul:
		locationRange := LocationRange{
			Location:    interpreter.Location,
			HasPosition: expression,
		}

		return DereferenceValue(interpreter, locationRange, value)

	case ast.OperationMove:
		interpreter.invalidateResource(value)
		return value
	}

	panic(&unsupportedOperation{
		kind:      common.OperationKindUnary,
		operation: expression.Operation,
		Range:     ast.NewUnmeteredRangeFromPositioned(expression),
	})
}

func (interpreter *Interpreter) VisitVoidExpression(_ *ast.VoidExpression) Value {
	return Void
}

func (interpreter *Interpreter) VisitBoolExpression(expression *ast.BoolExpression) Value {
	return BoolValue(expression.Value)
}

func (interpreter *Interpreter) VisitNilExpression(_ *ast.NilExpression) Value {
	return Nil
}

func (interpreter *Interpreter) VisitIntegerExpression(expression *ast.IntegerExpression) Value {
	typ := interpreter.Program.Elaboration.IntegerExpressionType(expression)

	value := expression.Value

	if _, ok := typ.(*sema.AddressType); ok {
		return NewAddressValueFromBytes(interpreter, value.Bytes)
	}

	// The ranges are checked at the checker level.
	// Hence, it is safe to create the value without validation.
	return interpreter.NewIntegerValueFromBigInt(value, typ)

}

// NewIntegerValueFromBigInt creates a Cadence interpreter value of a given subtype.
// This method assumes the range validations are done prior to calling this method. (i.e: at semantic level)
func (interpreter *Interpreter) NewIntegerValueFromBigInt(value *big.Int, integerSubType sema.Type) Value {
	config := interpreter.SharedState.Config
	memoryGauge := config.MemoryGauge

	// NOTE: cases meter manually and call the unmetered constructors to avoid allocating closures

	switch integerSubType {
	case sema.IntType, sema.IntegerType, sema.SignedIntegerType:
		// BigInt value is already metered at parser.
		return NewUnmeteredIntValueFromBigInt(value)
	case sema.UIntType:
		// BigInt value is already metered at parser.
		return NewUnmeteredUIntValueFromBigInt(value)

	// Int*
	case sema.Int8Type:
		common.UseMemory(memoryGauge, Int8MemoryUsage)
		return NewUnmeteredInt8Value(int8(value.Int64()))
	case sema.Int16Type:
		common.UseMemory(memoryGauge, Int16MemoryUsage)
		return NewUnmeteredInt16Value(int16(value.Int64()))
	case sema.Int32Type:
		common.UseMemory(memoryGauge, Int32MemoryUsage)
		return NewUnmeteredInt32Value(int32(value.Int64()))
	case sema.Int64Type:
		common.UseMemory(memoryGauge, Int64MemoryUsage)
		return NewUnmeteredInt64Value(value.Int64())
	case sema.Int128Type:
		// BigInt value is already metered at parser.
		return NewUnmeteredInt128ValueFromBigInt(value)
	case sema.Int256Type:
		// BigInt value is already metered at parser.
		return NewUnmeteredInt256ValueFromBigInt(value)

	// UInt*
	case sema.UInt8Type:
		common.UseMemory(memoryGauge, UInt8MemoryUsage)
		return NewUnmeteredUInt8Value(uint8(value.Uint64()))
	case sema.UInt16Type:
		common.UseMemory(memoryGauge, UInt16MemoryUsage)
		return NewUnmeteredUInt16Value(uint16(value.Uint64()))
	case sema.UInt32Type:
		common.UseMemory(memoryGauge, UInt32MemoryUsage)
		return NewUnmeteredUInt32Value(uint32(value.Uint64()))
	case sema.UInt64Type:
		common.UseMemory(memoryGauge, UInt64MemoryUsage)
		return NewUnmeteredUInt64Value(value.Uint64())
	case sema.UInt128Type:
		// BigInt value is already metered at parser.
		return NewUnmeteredUInt128ValueFromBigInt(value)
	case sema.UInt256Type, sema.FixedSizeUnsignedIntegerType:
		// BigInt value is already metered at parser.
		return NewUnmeteredUInt256ValueFromBigInt(value)

	// Word*
	case sema.Word8Type:
		common.UseMemory(memoryGauge, word8MemoryUsage)
		return NewUnmeteredWord8Value(uint8(value.Int64()))
	case sema.Word16Type:
		common.UseMemory(memoryGauge, word16MemoryUsage)
		return NewUnmeteredWord16Value(uint16(value.Int64()))
	case sema.Word32Type:
		common.UseMemory(memoryGauge, word32MemoryUsage)
		return NewUnmeteredWord32Value(uint32(value.Int64()))
	case sema.Word64Type:
		common.UseMemory(memoryGauge, word64MemoryUsage)
		return NewUnmeteredWord64Value(uint64(value.Int64()))
	case sema.Word128Type:
		// BigInt value is already metered at parser.
		return NewUnmeteredWord128ValueFromBigInt(value)
	case sema.Word256Type:
		// BigInt value is already metered at parser.
		return NewUnmeteredWord256ValueFromBigInt(value)

	default:
		panic(errors.NewUnreachableError())
	}
}

func (interpreter *Interpreter) VisitFixedPointExpression(expression *ast.FixedPointExpression) Value {
	// TODO: adjust once/if we support more fixed point types

	fixedPointSubType := interpreter.Program.Elaboration.FixedPointExpression(expression)

	value := fixedpoint.ConvertToFixedPointBigInt(
		expression.Negative,
		expression.UnsignedInteger,
		expression.Fractional,
		expression.Scale,
		sema.Fix64Scale,
	)
	switch fixedPointSubType {
	case sema.Fix64Type, sema.SignedFixedPointType:
		return NewFix64Value(interpreter, value.Int64)
	case sema.UFix64Type:
		return NewUFix64Value(interpreter, value.Uint64)
	case sema.FixedPointType:
		if expression.Negative {
			return NewFix64Value(interpreter, value.Int64)
		} else {
			return NewUFix64Value(interpreter, value.Uint64)
		}
	default:
		panic(errors.NewUnreachableError())
	}
}

func (interpreter *Interpreter) VisitStringExpression(expression *ast.StringExpression) Value {
	stringType := interpreter.Program.Elaboration.StringExpressionType(expression)

	switch stringType {
	case sema.CharacterType:
		return NewUnmeteredCharacterValue(expression.Value)
	}

	// Optimization: If the string is empty, return the empty string singleton
	// to avoid allocating a new string value.
	if len(expression.Value) == 0 {
		return EmptyString
	}

	// NOTE: already metered in lexer/parser
	return NewUnmeteredStringValue(expression.Value)
}

func BuildStringTemplate(values []string, exprs []Value) Value {
	var builder strings.Builder
	for i, str := range values {
		builder.WriteString(str)
		if i < len(exprs) {
			// switch on value instead of type
			switch expr := exprs[i].(type) {
			case *StringValue:
				builder.WriteString(expr.Str)
			case CharacterValue:
				builder.WriteString(expr.Str)
			default:
				builder.WriteString(expr.String())
			}
		}
	}
	return NewUnmeteredStringValue(builder.String())
}

func (interpreter *Interpreter) VisitStringTemplateExpression(expression *ast.StringTemplateExpression) Value {

	var values []Value
	if len(expression.Expressions) > 0 {
		// TODO: optimize: avoid allocation
		values = make([]Value, 0, len(expression.Expressions))

		for _, expr := range expression.Expressions {
			value := interpreter.evalExpression(expr)
			values = append(values, value)
		}
	}

	return BuildStringTemplate(expression.Values, values)
}

func (interpreter *Interpreter) VisitArrayExpression(expression *ast.ArrayExpression) Value {
	arrayExpressionTypes := interpreter.Program.Elaboration.ArrayExpressionTypes(expression)
	argumentTypes := arrayExpressionTypes.ArgumentTypes
	arrayType := arrayExpressionTypes.ArrayType
	elementType := arrayType.ElementType(false)

	var values []Value

	count := len(expression.Values)
	if count > 0 {

		// TODO: optimize: avoid allocation, use NewArrayValueWithIterator
		values = make([]Value, count)

		for i, valueExpression := range expression.Values {
			argumentType := argumentTypes[i]
			value := interpreter.evalExpression(valueExpression)

			values[i] = TransferAndConvert(
				interpreter,
				value,
				argumentType,
				elementType,
				LocationRange{
					Location:    interpreter.Location,
					HasPosition: valueExpression,
				},
			)
		}
	}

	// TODO: cache
	arrayStaticType := ConvertSemaArrayTypeToStaticArrayType(interpreter, arrayType)

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: expression,
	}

	return NewArrayValue(
		interpreter,
		locationRange,
		arrayStaticType,
		common.ZeroAddress,
		values...,
	)
}

func (interpreter *Interpreter) VisitDictionaryExpression(expression *ast.DictionaryExpression) Value {

	dictionaryExpressionTypes := interpreter.Program.Elaboration.DictionaryExpressionTypes(expression)
	entryTypes := dictionaryExpressionTypes.EntryTypes
	dictionaryType := dictionaryExpressionTypes.DictionaryType

	var keyValuePairs []Value
	if len(expression.Entries) > 0 {

		// TODO: optimize: avoid allocation, use newDictionaryValueWithIterator
		keyValuePairs = make([]Value, 0, len(expression.Entries)*2)

		keyType := dictionaryType.KeyType
		valueType := dictionaryType.ValueType

		for i, entry := range expression.Entries {
			entryType := entryTypes[i]

			key := interpreter.evalExpression(entry.Key)

			key = TransferAndConvert(
				interpreter,
				key,
				entryType.KeyType,
				keyType,
				LocationRange{
					Location:    interpreter.Location,
					HasPosition: entry.Key,
				},
			)

			value := interpreter.evalExpression(entry.Value)

			value = TransferAndConvert(
				interpreter,
				value,
				entryType.ValueType,
				valueType,
				LocationRange{
					Location:    interpreter.Location,
					HasPosition: entry.Value,
				},
			)

			keyValuePairs = append(
				keyValuePairs,
				key,
				value,
			)
		}
	}

	dictionaryStaticType := ConvertSemaDictionaryTypeToStaticDictionaryType(interpreter, dictionaryType)

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: expression,
	}

	return NewDictionaryValue(
		interpreter,
		locationRange,
		dictionaryStaticType,
		keyValuePairs...,
	)
}

func (interpreter *Interpreter) VisitMemberExpression(expression *ast.MemberExpression) Value {
	const allowMissing = false

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: expression,
	}

	return interpreter.memberExpressionGetterSetter(expression, locationRange).get(allowMissing)
}

func (interpreter *Interpreter) VisitIndexExpression(expression *ast.IndexExpression) Value {
	// note that this check in `AttachmentAccessTypes` must proceed the casting to the `TypeIndexableValue`
	// or `ValueIndexableValue` interfaces. A `*EphemeralReferenceValue` value is both a `TypeIndexableValue`
	// and a `ValueIndexableValue` statically, but at runtime can only be used as one or the other. Whether
	// or not an expression is present in this map allows us to disambiguate between these two cases.
	if attachmentType, ok := interpreter.Program.Elaboration.AttachmentAccessTypes(expression); ok {
		typedResult, ok := interpreter.evalExpression(expression.TargetExpression).(TypeIndexableValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		locationRange := LocationRange{
			Location:    interpreter.Location,
			HasPosition: expression,
		}
		return typedResult.GetTypeKey(interpreter, locationRange, attachmentType)
	} else {
		typedResult, ok := interpreter.evalExpression(expression.TargetExpression).(ValueIndexableValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		indexingValue := interpreter.evalExpression(expression.IndexingExpression)
		locationRange := LocationRange{
			Location:    interpreter.Location,
			HasPosition: expression,
		}
		value := typedResult.GetKey(interpreter, locationRange, indexingValue)

		// If the indexing value is a reference, then return a reference for the resulting value.
		return interpreter.maybeGetReference(expression, value)
	}
}

func (interpreter *Interpreter) maybeGetReference(
	expression *ast.IndexExpression,
	memberValue Value,
) Value {
	indexExpressionTypes, _ := interpreter.Program.Elaboration.IndexExpressionTypes(expression)

	if indexExpressionTypes.ReturnReference {
		expectedType := indexExpressionTypes.ResultType

		locationRange := LocationRange{
			Location:    interpreter.Location,
			HasPosition: expression,
		}

		// Get a reference to the value
		memberValue = getReferenceValue(
			interpreter,
			memberValue,
			expectedType,
			locationRange,
		)
	}

	return memberValue
}

func (interpreter *Interpreter) VisitConditionalExpression(expression *ast.ConditionalExpression) Value {
	value, ok := interpreter.evalExpression(expression.Test).(BoolValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	if value {
		return interpreter.evalExpression(expression.Then)
	} else {
		return interpreter.evalExpression(expression.Else)
	}
}

func (interpreter *Interpreter) VisitInvocationExpression(invocationExpression *ast.InvocationExpression) Value {
	return interpreter.visitInvocationExpressionWithImplicitArgument(invocationExpression, nil)
}

func (interpreter *Interpreter) visitInvocationExpressionWithImplicitArgument(invocationExpression *ast.InvocationExpression, implicitArg Value) Value {
	config := interpreter.SharedState.Config

	// tracing
	if config.TracingEnabled {
		startTime := time.Now()
		invokedExpression := invocationExpression.InvokedExpression.String()
		defer func() {
			interpreter.reportFunctionTrace(
				invokedExpression,
				time.Since(startTime),
			)
		}()
	}

	// interpret the invoked expression
	result := interpreter.evalExpression(invocationExpression.InvokedExpression)

	// Handle optional chaining on member expression, if any:
	// - If the member expression is nil, finish execution
	// - If the member expression is some value, the wrapped value
	//   is the function value that should be invoked

	isOptionalChaining := false

	if invokedMemberExpression, ok :=
		invocationExpression.InvokedExpression.(*ast.MemberExpression); ok && invokedMemberExpression.Optional {

		isOptionalChaining = true

		switch typedResult := result.(type) {
		case NilValue:
			return typedResult

		case *SomeValue:
			result = typedResult.InnerValue()

		default:
			panic(errors.NewUnreachableError())
		}
	}

	function, ok := result.(FunctionValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// NOTE: evaluate all argument expressions in call-site scope, not in function body

	var argumentExpressions []ast.Expression

	argumentCount := len(invocationExpression.Arguments)
	if argumentCount > 0 {
		argumentExpressions = make([]ast.Expression, argumentCount)
		for i, argument := range invocationExpression.Arguments {
			argumentExpressions[i] = argument.Expression
		}
	}

	elaboration := interpreter.Program.Elaboration

	invocationExpressionTypes := elaboration.InvocationExpressionTypes(invocationExpression)

	typeParameterTypes := invocationExpressionTypes.TypeArguments
	argumentTypes := invocationExpressionTypes.ArgumentTypes
	parameterTypes := invocationExpressionTypes.ParameterTypes
	returnType := invocationExpressionTypes.ReturnType

	interpreter.reportFunctionInvocation()

	resultValue := invokeFunctionValueWithEval(
		interpreter,
		function,
		argumentExpressions,
		func(expression ast.Expression) Value {
			return interpreter.evalExpression(expression)
		},
		implicitArg,
		argumentExpressions,
		argumentTypes,
		parameterTypes,
		returnType,
		typeParameterTypes,
		invocationExpression,
	)

	interpreter.reportInvokedFunctionReturn()

	// If this is invocation is optional chaining, wrap the result
	// as an optional, as the result is expected to be an optional
	if isOptionalChaining {
		resultValue = NewSomeValueNonCopying(interpreter, resultValue)
	}

	return resultValue
}

func (interpreter *Interpreter) VisitFunctionExpression(expression *ast.FunctionExpression) Value {

	// lexical scope: variables in functions are bound to what is visible at declaration time.
	lexicalScope := interpreter.activations.CurrentOrNew()

	// Variables which are declared after this function declaration
	// should not be visible or even overwrite the variables captured by the closure
	/// (e.g. through shadowing).
	//
	// For example:
	//
	//     fun foo(a: Int): Int {
	//         let bar = fun(): Int {
	//             return a
	//             //     ^ should refer to the `a` parameter of `foo`,
	//             //     not to the `a` variable declared after `bar`
	//         }
	//         let a = 2
	//         return bar()
	//     }
	//
	// As variable declarations mutate the current activation in place, capture a clone of the current activation,
	// so that the mutations are not performed on the captured activation.

	lexicalScope = lexicalScope.Clone()

	functionType := interpreter.Program.Elaboration.FunctionExpressionFunctionType(expression)

	var preConditions []ast.Condition
	if expression.FunctionBlock.PreConditions != nil {
		preConditions = expression.FunctionBlock.PreConditions.Conditions
	}

	var beforeStatements []ast.Statement
	var rewrittenPostConditions []ast.Condition

	if expression.FunctionBlock.PostConditions != nil {
		postConditionsRewrite :=
			interpreter.Program.Elaboration.PostConditionsRewrite(expression.FunctionBlock.PostConditions)

		rewrittenPostConditions = postConditionsRewrite.RewrittenPostConditions
		beforeStatements = postConditionsRewrite.BeforeStatements
	}

	statements := expression.FunctionBlock.Block.Statements

	return NewInterpretedFunctionValue(
		interpreter,
		expression.ParameterList,
		functionType,
		lexicalScope,
		beforeStatements,
		preConditions,
		statements,
		rewrittenPostConditions,
	)
}

func (interpreter *Interpreter) VisitCastingExpression(expression *ast.CastingExpression) Value {
	value := interpreter.evalExpression(expression.Expression)

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: expression.Expression,
	}

	castingExpressionTypes := interpreter.Program.Elaboration.CastingExpressionTypes(expression)
	expectedType := castingExpressionTypes.TargetType

	switch expression.Operation {
	case ast.OperationFailableCast, ast.OperationForceCast:
		// if the value itself has a mapped entitlement type in its authorization
		// (e.g. if it is a reference to `self` or `base`  in an attachment function with mapped access)
		// substitution must also be performed on its entitlements
		//
		// we do this here (as opposed to in `IsSubTypeOfSemaType`) because casting is the only way that
		// an entitlement can "traverse the boundary", so to speak, between runtime and static types, and
		// thus this is the only place where it becomes necessary to "instantiate" the result of a map to its
		// concrete outputs. In other places (e.g. interface conformance checks) we want to leave maps generic,
		// so we don't substitute them.

		// if the target is anystruct or anyresource we want to preserve optionals
		unboxedExpectedType := sema.UnwrapOptionalType(expectedType)
		if !(unboxedExpectedType == sema.AnyStructType || unboxedExpectedType == sema.AnyResourceType) {
			// otherwise dynamic cast now always unboxes optionals
			value = Unbox(value)
		}
		valueSemaType := MustSemaTypeOfValue(value, interpreter)
		valueStaticType := ConvertSemaToStaticType(interpreter, valueSemaType)
		isSubType := IsSubTypeOfSemaType(interpreter, valueStaticType, expectedType)

		switch expression.Operation {
		case ast.OperationFailableCast:
			if !isSubType {
				return Nil
			}

		case ast.OperationForceCast:
			if !isSubType {
				locationRange := LocationRange{
					Location:    interpreter.Location,
					HasPosition: expression.Expression,
				}

				panic(&ForceCastTypeMismatchError{
					ExpectedType:  expectedType,
					ActualType:    valueSemaType,
					LocationRange: locationRange,
				})
			}

		default:
			panic(errors.NewUnreachableError())
		}

		// The failable cast may upcast to an optional type, e.g. `1 as? Int?`, so box
		value = ConvertAndBox(interpreter, locationRange, value, valueSemaType, expectedType)

		if expression.Operation == ast.OperationFailableCast {
			// Failable casting is a resource invalidation
			interpreter.invalidateResource(value)

			value = NewSomeValueNonCopying(interpreter, value)
		}

		return value

	case ast.OperationCast:
		staticValueType := castingExpressionTypes.StaticValueType
		// The cast may upcast to an optional type, e.g. `1 as Int?`, so box
		return ConvertAndBox(interpreter, locationRange, value, staticValueType, expectedType)

	default:
		panic(errors.NewUnreachableError())
	}
}

func (interpreter *Interpreter) VisitCreateExpression(expression *ast.CreateExpression) Value {
	return interpreter.evalExpression(expression.InvocationExpression)
}

func (interpreter *Interpreter) VisitDestroyExpression(expression *ast.DestroyExpression) Value {
	value := interpreter.evalExpression(expression.Expression)

	interpreter.invalidateResource(value)

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: expression,
	}

	value.(ResourceKindedValue).Destroy(interpreter, locationRange)

	return Void
}

func (interpreter *Interpreter) VisitReferenceExpression(referenceExpression *ast.ReferenceExpression) Value {

	borrowType := interpreter.Program.Elaboration.ReferenceExpressionBorrowType(referenceExpression)

	result := interpreter.evalExpression(referenceExpression.Expression)

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: referenceExpression,
	}

	return CreateReferenceValue(
		interpreter,
		borrowType,
		result,
		locationRange,
		false,
	)
}

func CreateReferenceValue(
	context ReferenceCreationContext,
	borrowType sema.Type,
	value Value,
	locationRange LocationRange,
	isImplicit bool,
) Value {

	// There are four potential cases:
	// (1) Target type is optional, actual value is also optional
	//     (1.a) value is SomeValue
	//     (1.b) value is nil
	// (2) Target type is optional, actual value is non-optional
	// (3) Target type is non-optional, actual value is optional
	//     (3.a) value is SomeValue
	//     (3.b) value is nil
	// (4) Target type is non-optional, actual value is non-optional

	switch typ := borrowType.(type) {
	case *sema.OptionalType:

		innerType := typ.Type

		switch value := value.(type) {
		case *SomeValue:
			// Case (1):
			// References to optionals are transformed into optional references.

			// (1.a) value is SomeValue
			// Move the *SomeValue out to the reference itself.

			innerValue := value.InnerValue()

			referenceValue := CreateReferenceValue(context, innerType, innerValue, locationRange, false)

			// Wrap the reference with an optional (since an optional is expected).
			return NewSomeValueNonCopying(context, referenceValue)

		case NilValue:
			// Case (1.b) value is nil.
			// Since the resulting type can accommodate a nil (optional-reference), return nil,
			return Nil

		default:
			// Case (2):
			// If the referenced value is non-optional,
			// but the target type is optional.
			referenceValue := CreateReferenceValue(context, innerType, value, locationRange, false)

			// Wrap the reference with an optional (since an optional is expected).
			return NewSomeValueNonCopying(context, referenceValue)
		}

	case *sema.ReferenceType:

		switch value := value.(type) {
		case *SomeValue:
			// Case (3.a): target type is non-optional, actual value is optional.
			innerValue := value.InnerValue()

			return CreateReferenceValue(context, typ, innerValue, locationRange, false)

		case NilValue:
			// Case (3.b) value is nil.
			// Since the resulting type can NOT accommodate a nil (non-optional reference), error-out.
			panic(&NonOptionalReferenceToNilError{
				ReferenceType: typ,
				LocationRange: locationRange,
			})

		case ReferenceValue:
			if isImplicit {
				// During implicit reference creation (e.g: member/index access on a reference),
				// if the value is already a reference then return the same reference.
				// However, we need to make sure that this reference is actually a subtype of the resultType,
				// since the checker may not be aware that we are "short-circuiting" in this case.
				// Additionally, it is only safe to "compress" reference types like this when the desired
				// result reference type is unauthorized
				staticType := value.StaticType(context)
				if typ.Authorization != sema.UnauthorizedAccess || !IsSubTypeOfSemaType(context, staticType, typ) {
					panic(&InvalidMemberReferenceError{
						ExpectedType:  typ,
						ActualType:    MustConvertStaticToSemaType(staticType, context),
						LocationRange: locationRange,
					})
				}

				return value
			}

			// break
		}

		// Case (4): target type is non-optional, actual value is also non-optional.
		return newEphemeralReference(context, value, typ, locationRange)

	default:
		panic(errors.NewUnreachableError())
	}
}

func newEphemeralReference(
	context ReferenceCreationContext,
	value Value,
	typ *sema.ReferenceType,
	locationRange LocationRange,
) *EphemeralReferenceValue {

	auth := ConvertSemaAccessToStaticAuthorization(context, typ.Authorization)

	return NewEphemeralReferenceValue(
		context,
		auth,
		value,
		typ.Type,
		locationRange,
	)
}

func (interpreter *Interpreter) VisitForceExpression(expression *ast.ForceExpression) Value {
	result := interpreter.evalExpression(expression.Expression)

	switch result := result.(type) {
	case *SomeValue:
		return result.InnerValue()

	case NilValue:
		panic(
			&ForceNilError{
				LocationRange: LocationRange{
					Location:    interpreter.Location,
					HasPosition: expression,
				},
			},
		)

	default:
		return result
	}
}

func (interpreter *Interpreter) VisitPathExpression(expression *ast.PathExpression) Value {
	domain := common.PathDomainFromIdentifier(expression.Domain.Identifier)

	// meter the Path's Identifier since path is just a container
	common.UseMemory(interpreter, common.NewRawStringMemoryUsage(len(expression.Identifier.Identifier)))

	return NewPathValue(
		interpreter,
		domain,
		expression.Identifier.Identifier,
	)
}

func (interpreter *Interpreter) VisitAttachExpression(attachExpression *ast.AttachExpression) Value {

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: attachExpression,
	}

	attachTarget := interpreter.evalExpression(attachExpression.Base)
	base, ok := attachTarget.(*CompositeValue)

	// we enforce this in the checker, but check defensively anyway
	if !ok || !base.Kind.SupportsAttachments() {
		panic(&InvalidAttachmentOperationTargetError{
			Value:         attachTarget,
			LocationRange: locationRange,
		})
	}

	if inIteration := interpreter.SharedState.inAttachmentIteration(base); inIteration {
		panic(&AttachmentIterationMutationError{
			Value:         base,
			LocationRange: locationRange,
		})
	}

	// the `base` value must be accessible during the attachment's constructor, but we cannot
	// set it on the attachment's `CompositeValue` yet, because the value does not exist.
	// Instead, we create an implicit constructor argument containing a reference to the base.

	// within the constructor, the attachment's base and self references should be fully entitled,
	// as the constructor of the attachment is only callable by the owner of the base
	baseType := MustSemaTypeOfValue(base, interpreter).(sema.EntitlementSupportingType)
	baseAccess := baseType.SupportedEntitlements().Access()
	auth := ConvertSemaAccessToStaticAuthorization(interpreter, baseAccess)

	attachmentType := interpreter.Program.Elaboration.AttachTypes(attachExpression)

	baseValue := NewEphemeralReferenceValue(
		interpreter,
		auth,
		base,
		MustSemaTypeOfValue(base, interpreter).(*sema.CompositeType),
		locationRange,
	)

	attachment, ok := interpreter.visitInvocationExpressionWithImplicitArgument(
		attachExpression.Attachment,
		baseValue,
	).(*CompositeValue)
	// attached expressions must be composite constructors, as enforced in the checker
	if !ok {
		panic(errors.NewUnreachableError())
	}

	base = base.Transfer(
		interpreter,
		locationRange,
		atree.Address{},
		false,
		nil,
		nil,
		true, // base is standalone.
	).(*CompositeValue)

	attachment.setBaseValue(base)

	// we enforce this in the checker
	if !ok {
		panic(errors.NewUnreachableError())
	}

	base.SetTypeKey(
		interpreter,
		locationRange,
		attachmentType,
		attachment,
	)

	return base
}
