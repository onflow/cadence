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

package interpreter

import (
	"math/big"
	"time"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/fixedpoint"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

// assignmentGetterSetter returns a getter/setter function pair
// for the target expression
func (interpreter *Interpreter) assignmentGetterSetter(expression ast.Expression) getterSetter {
	switch expression := expression.(type) {
	case *ast.IdentifierExpression:
		return interpreter.identifierExpressionGetterSetter(expression)

	case *ast.IndexExpression:
		if attachmentType, ok := interpreter.Program.Elaboration.AttachmentAccessTypes(expression); ok {
			return interpreter.typeIndexExpressionGetterSetter(expression, attachmentType)
		}
		return interpreter.valueIndexExpressionGetterSetter(expression)

	case *ast.MemberExpression:
		return interpreter.memberExpressionGetterSetter(expression)

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
func (interpreter *Interpreter) identifierExpressionGetterSetter(identifierExpression *ast.IdentifierExpression) getterSetter {
	identifier := identifierExpression.Identifier.Identifier
	variable := interpreter.FindVariable(identifier)

	return getterSetter{
		get: func(_ bool) Value {
			value := variable.GetValue()
			interpreter.checkInvalidatedResourceUse(value, variable, identifier, identifierExpression)
			return value
		},
		set: func(value Value) {
			interpreter.startResourceTracking(value, variable, identifier, identifierExpression)
			variable.SetValue(value)
		},
	}
}

func (interpreter *Interpreter) typeIndexExpressionGetterSetter(
	indexExpression *ast.IndexExpression,
	attachmentType sema.Type,
) getterSetter {
	target, ok := interpreter.evalExpression(indexExpression.TargetExpression).(TypeIndexableValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: indexExpression,
	}

	return getterSetter{
		target: target,
		get: func(_ bool) Value {
			return target.GetTypeKey(interpreter, locationRange, attachmentType)
		},
		set: func(_ Value) {
			// writing to composites with indexing syntax is not supported
			panic(errors.NewUnreachableError())
		},
	}
}

// valueIndexExpressionGetterSetter returns a getter/setter function pair
// for the target index expression
func (interpreter *Interpreter) valueIndexExpressionGetterSetter(indexExpression *ast.IndexExpression) getterSetter {
	target, ok := interpreter.evalExpression(indexExpression.TargetExpression).(ValueIndexableValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: indexExpression,
	}

	// Evaluate, transfer, and convert the indexing value,
	// as it is essentially an "argument" of the get/set operation

	elaboration := interpreter.Program.Elaboration

	indexExpressionTypes := elaboration.IndexExpressionTypes(indexExpression)
	indexedType := indexExpressionTypes.IndexedType
	indexingType := indexExpressionTypes.IndexingType

	transferredIndexingValue := interpreter.transferAndConvert(
		interpreter.evalExpression(indexExpression.IndexingExpression),
		indexingType,
		indexedType.IndexingType(),
		LocationRange{
			Location:    interpreter.Location,
			HasPosition: indexExpression.IndexingExpression,
		},
	)

	isNestedResourceMove := elaboration.IsNestedResourceMoveExpression(indexExpression)

	return getterSetter{
		target: target,
		get: func(_ bool) Value {
			if isNestedResourceMove {
				return target.RemoveKey(interpreter, locationRange, transferredIndexingValue)
			} else {
				return target.GetKey(interpreter, locationRange, transferredIndexingValue)
			}
		},
		set: func(value Value) {
			if isNestedResourceMove {
				target.InsertKey(interpreter, locationRange, transferredIndexingValue, value)
			} else {
				target.SetKey(interpreter, locationRange, transferredIndexingValue, value)
			}
		},
	}
}

// memberExpressionGetterSetter returns a getter/setter function pair
// for the target member expression
func (interpreter *Interpreter) memberExpressionGetterSetter(memberExpression *ast.MemberExpression) getterSetter {
	target := interpreter.evalExpression(memberExpression.Expression)
	identifier := memberExpression.Identifier.Identifier
	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: memberExpression,
	}

	isNestedResourceMove := interpreter.Program.Elaboration.IsNestedResourceMoveExpression(memberExpression)

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
					target = typedTarget.InnerValue(interpreter, locationRange)

				default:
					panic(errors.NewUnreachableError())
				}
			}

			var resultValue Value
			if isNestedResourceMove {
				resultValue = target.(MemberAccessibleValue).RemoveMember(interpreter, locationRange, identifier)
			} else {
				resultValue = interpreter.getMember(target, locationRange, identifier)
			}
			if resultValue == nil && !allowMissing {
				panic(UseBeforeInitializationError{
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

			return resultValue
		},
		set: func(value Value) {
			interpreter.checkMemberAccess(memberExpression, target, locationRange)

			interpreter.setMember(target, locationRange, identifier, value)
		},
	}
}

func (interpreter *Interpreter) checkMemberAccess(
	memberExpression *ast.MemberExpression,
	target Value,
	locationRange LocationRange,
) {
	memberInfo, _ := interpreter.Program.Elaboration.MemberExpressionMemberInfo(memberExpression)
	expectedType := memberInfo.AccessedType

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

	if _, ok := target.(*StorageReferenceValue); ok {
		// NOTE: Storage reference value accesses are already checked in  StorageReferenceValue.dereference
		return
	}

	targetStaticType := target.StaticType(interpreter)

	if !interpreter.IsSubTypeOfSemaType(targetStaticType, expectedType) {
		targetSemaType := interpreter.MustConvertStaticToSemaType(targetStaticType)

		panic(MemberAccessTypeError{
			ExpectedType:  expectedType,
			ActualType:    targetSemaType,
			LocationRange: locationRange,
		})
	}
}

func (interpreter *Interpreter) VisitIdentifierExpression(expression *ast.IdentifierExpression) Value {
	name := expression.Identifier.Identifier
	variable := interpreter.FindVariable(name)
	value := variable.GetValue()

	interpreter.checkInvalidatedResourceUse(value, variable, name, expression)

	return value
}

func (interpreter *Interpreter) evalExpression(expression ast.Expression) Value {
	return ast.AcceptExpression[Value](expression, interpreter)
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
		panic(InvalidOperandsError{
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
		return interpreter.testEqual(leftValue, rightValue(), expression)

	case ast.OperationNotEqual:
		return !interpreter.testEqual(leftValue, rightValue(), expression)

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
			return some.InnerValue(interpreter, locationRange)
		}

		value := rightValue()

		binaryExpressionTypes := interpreter.Program.Elaboration.BinaryExpressionTypes(expression)
		rightType := binaryExpressionTypes.RightType
		resultType := binaryExpressionTypes.ResultType

		// NOTE: important to convert both any and optional
		return interpreter.ConvertAndBox(locationRange, value, rightType, resultType)
	}

	panic(&unsupportedOperation{
		kind:      common.OperationKindBinary,
		operation: expression.Operation,
		Range:     ast.NewUnmeteredRangeFromPositioned(expression),
	})
}

func (interpreter *Interpreter) testEqual(left, right Value, expression *ast.BinaryExpression) BoolValue {
	left = interpreter.Unbox(
		LocationRange{
			Location:    interpreter.Location,
			HasPosition: expression.Left,
		},
		left,
	)

	right = interpreter.Unbox(
		LocationRange{
			Location:    interpreter.Location,
			HasPosition: expression.Right,
		},
		right,
	)

	leftEquatable, ok := left.(EquatableValue)
	if !ok {
		return FalseValue
	}

	return AsBoolValue(
		leftEquatable.Equal(
			interpreter,
			LocationRange{
				Location:    interpreter.Location,
				HasPosition: expression,
			},
			right,
		),
	)
}

func (interpreter *Interpreter) testComparison(left, right Value, expression *ast.BinaryExpression) BoolValue {
	left = interpreter.Unbox(
		LocationRange{
			Location:    interpreter.Location,
			HasPosition: expression.Left,
		},
		left,
	)

	right = interpreter.Unbox(
		LocationRange{
			Location:    interpreter.Location,
			HasPosition: expression.Right,
		},
		right,
	)

	leftComparable, ok := left.(ComparableValue)
	if !ok {
		return FalseValue
	}

	rightComparable, ok := right.(ComparableValue)
	if !ok {
		return FalseValue
	}

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: expression,
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
		return integerValue.Negate(interpreter, LocationRange{
			Location:    interpreter.Location,
			HasPosition: expression,
		})

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
	return AsBoolValue(expression.Value)
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
	case sema.UInt256Type:
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

	// NOTE: already metered in lexer/parser
	return NewUnmeteredStringValue(expression.Value)
}

func (interpreter *Interpreter) VisitArrayExpression(expression *ast.ArrayExpression) Value {
	values := interpreter.visitExpressionsNonCopying(expression.Values)

	arrayExpressionTypes := interpreter.Program.Elaboration.ArrayExpressionTypes(expression)
	argumentTypes := arrayExpressionTypes.ArgumentTypes
	arrayType := arrayExpressionTypes.ArrayType
	elementType := arrayType.ElementType(false)

	var copies []Value

	count := len(values)
	if count > 0 {
		copies = make([]Value, count)
		for i, argument := range values {
			argumentType := argumentTypes[i]
			argumentExpression := expression.Values[i]
			locationRange := LocationRange{
				Location:    interpreter.Location,
				HasPosition: argumentExpression,
			}
			copies[i] = interpreter.transferAndConvert(argument, argumentType, elementType, locationRange)
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
		copies...,
	)
}

func (interpreter *Interpreter) VisitDictionaryExpression(expression *ast.DictionaryExpression) Value {
	values := interpreter.visitEntries(expression.Entries)

	dictionaryExpressionTypes := interpreter.Program.Elaboration.DictionaryExpressionTypes(expression)
	entryTypes := dictionaryExpressionTypes.EntryTypes
	dictionaryType := dictionaryExpressionTypes.DictionaryType

	var keyValuePairs []Value

	for i, dictionaryEntryValues := range values {
		entryType := entryTypes[i]
		entry := expression.Entries[i]

		key := interpreter.transferAndConvert(
			dictionaryEntryValues.Key,
			entryType.KeyType,
			dictionaryType.KeyType,
			LocationRange{
				Location:    interpreter.Location,
				HasPosition: entry.Key,
			},
		)

		value := interpreter.transferAndConvert(
			dictionaryEntryValues.Value,
			entryType.ValueType,
			dictionaryType.ValueType,
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
	return interpreter.memberExpressionGetterSetter(expression).get(allowMissing)
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
		return typedResult.GetKey(interpreter, locationRange, indexingValue)
	}
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

func (interpreter *Interpreter) visitInvocationExpressionWithImplicitArgument(invocationExpression *ast.InvocationExpression, implicitArg *Value) Value {
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
			result = typedResult.InnerValue(
				interpreter,
				LocationRange{
					Location:    interpreter.Location,
					HasPosition: invocationExpression.InvokedExpression,
				},
			)

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

	arguments := interpreter.visitExpressionsNonCopying(argumentExpressions)

	elaboration := interpreter.Program.Elaboration

	invocationExpressionTypes := elaboration.InvocationExpressionTypes(invocationExpression)

	typeParameterTypes := invocationExpressionTypes.TypeArguments
	argumentTypes := invocationExpressionTypes.ArgumentTypes
	parameterTypes := invocationExpressionTypes.TypeParameterTypes

	// add the implicit argument to the end of the argument list, if it exists
	if implicitArg != nil {
		arguments = append(arguments, *implicitArg)
		argumentTypes = append(argumentTypes, interpreter.MustSemaTypeOfValue(*implicitArg))
	}

	interpreter.reportFunctionInvocation()

	resultValue := interpreter.invokeFunctionValue(
		function,
		arguments,
		argumentExpressions,
		argumentTypes,
		parameterTypes,
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

func (interpreter *Interpreter) visitExpressionsNonCopying(expressions []ast.Expression) []Value {
	var values []Value

	count := len(expressions)
	if count > 0 {
		values = make([]Value, 0, count)
		for _, expression := range expressions {
			value := interpreter.evalExpression(expression)
			values = append(values, value)
		}
	}

	return values
}

func (interpreter *Interpreter) visitEntries(entries []ast.DictionaryEntry) []DictionaryEntryValues {
	var values []DictionaryEntryValues

	count := len(entries)
	if count > 0 {
		values = make([]DictionaryEntryValues, 0, count)

		for _, entry := range entries {
			key := interpreter.evalExpression(entry.Key)
			value := interpreter.evalExpression(entry.Value)

			values = append(
				values,
				DictionaryEntryValues{
					Key:   key,
					Value: value,
				},
			)
		}
	}

	return values
}

func (interpreter *Interpreter) VisitFunctionExpression(expression *ast.FunctionExpression) Value {

	// lexical scope: variables in functions are bound to what is visible at declaration time
	lexicalScope := interpreter.activations.CurrentOrNew()

	functionType := interpreter.Program.Elaboration.FunctionExpressionFunctionType(expression)

	var preConditions ast.Conditions
	if expression.FunctionBlock.PreConditions != nil {
		preConditions = *expression.FunctionBlock.PreConditions
	}

	var beforeStatements []ast.Statement
	var rewrittenPostConditions ast.Conditions

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

	expectedType := interpreter.Program.Elaboration.CastingExpressionTypes(expression).TargetType

	switch expression.Operation {
	case ast.OperationFailableCast, ast.OperationForceCast:
		valueStaticType := value.StaticType(interpreter)
		isSubType := interpreter.IsSubTypeOfSemaType(valueStaticType, expectedType)

		switch expression.Operation {
		case ast.OperationFailableCast:
			if !isSubType {
				return Nil
			}

			// The failable cast may upcast to an optional type, e.g. `1 as? Int?`, so box
			value = interpreter.BoxOptional(locationRange, value, expectedType)

			return NewSomeValueNonCopying(interpreter, value)

		case ast.OperationForceCast:
			if !isSubType {
				valueSemaType := interpreter.MustConvertStaticToSemaType(valueStaticType)

				locationRange := LocationRange{
					Location:    interpreter.Location,
					HasPosition: expression.Expression,
				}

				panic(ForceCastTypeMismatchError{
					ExpectedType:  expectedType,
					ActualType:    valueSemaType,
					LocationRange: locationRange,
				})
			}

			// The failable cast may upcast to an optional type, e.g. `1 as? Int?`, so box
			return interpreter.BoxOptional(locationRange, value, expectedType)

		default:
			panic(errors.NewUnreachableError())
		}

	case ast.OperationCast:
		staticValueType := interpreter.Program.Elaboration.CastingExpressionTypes(expression).StaticValueType
		// The cast may upcast to an optional type, e.g. `1 as Int?`, so box
		return interpreter.ConvertAndBox(locationRange, value, staticValueType, expectedType)

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

	if result, ok := result.(ReferenceTrackedResourceKindedValue); ok {
		interpreter.trackReferencedResourceKindedValue(result.StorageID(), result)
	}

	switch typ := borrowType.(type) {
	case *sema.OptionalType:
		innerBorrowType, ok := typ.Type.(*sema.ReferenceType)
		// we enforce this in the checker
		if !ok {
			panic(errors.NewUnreachableError())
		}

		switch result := result.(type) {
		case *SomeValue:
			// References to optionals are transformed into optional references,
			// so move the *SomeValue out to the reference itself

			locationRange := LocationRange{
				Location:    interpreter.Location,
				HasPosition: referenceExpression.Expression,
			}

			innerValue := result.InnerValue(interpreter, locationRange)
			if result, ok := innerValue.(ReferenceTrackedResourceKindedValue); ok {
				interpreter.trackReferencedResourceKindedValue(result.StorageID(), result)
			}

			return NewSomeValueNonCopying(
				interpreter,
				NewEphemeralReferenceValue(
					interpreter,
					innerBorrowType.Authorized,
					innerValue,
					innerBorrowType.Type,
				),
			)

		case NilValue:
			return Nil

		default:
			// If the referenced value is non-optional,
			// but the target type is optional,
			// then box the reference properly

			locationRange := LocationRange{
				Location:    interpreter.Location,
				HasPosition: referenceExpression,
			}

			return interpreter.BoxOptional(
				locationRange,
				NewEphemeralReferenceValue(
					interpreter,
					innerBorrowType.Authorized,
					result,
					innerBorrowType.Type,
				),
				borrowType,
			)
		}

	case *sema.ReferenceType:
		return NewEphemeralReferenceValue(interpreter, typ.Authorized, result, typ.Type)
	}
	panic(errors.NewUnreachableError())
}

func (interpreter *Interpreter) VisitForceExpression(expression *ast.ForceExpression) Value {
	result := interpreter.evalExpression(expression.Expression)

	switch result := result.(type) {
	case *SomeValue:
		locationRange := LocationRange{
			Location:    interpreter.Location,
			HasPosition: expression.Expression,
		}
		return result.InnerValue(interpreter, locationRange)

	case NilValue:
		panic(
			ForceNilError{
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

	// we enforce this in the checker, but check defensively anyways
	if !ok || !base.Kind.SupportsAttachments() {
		panic(InvalidAttachmentOperationTargetError{
			Value:         attachTarget,
			LocationRange: locationRange,
		})
	}

	if inIteration := interpreter.SharedState.inAttachmentIteration(base); inIteration {
		panic(AttachmentIterationMutationError{
			Value:         base,
			LocationRange: locationRange,
		})
	}

	// the `base` value must be accessible during the attachment's constructor, but we cannot
	// set it on the attachment's `CompositeValue` yet, because the value does not exist. Instead
	// we create an implicit constructor argument containing a reference to the base
	var baseValue Value = NewEphemeralReferenceValue(
		interpreter,
		false,
		base,
		interpreter.MustSemaTypeOfValue(base).(*sema.CompositeType),
	)
	interpreter.trackReferencedResourceKindedValue(base.StorageID(), base)

	attachment, ok := interpreter.visitInvocationExpressionWithImplicitArgument(
		attachExpression.Attachment,
		&baseValue,
	).(*CompositeValue)
	// attached expressions must be composite constructors, as enforced in the checker
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// Because `self` in attachments is a reference, we need to track the attachment if it's a resource
	interpreter.trackReferencedResourceKindedValue(attachment.StorageID(), attachment)

	base = base.Transfer(
		interpreter,
		locationRange,
		atree.Address{},
		false,
		nil,
	).(*CompositeValue)

	// we enforce this in the checker
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// when `v[A]` is executed, we set `A`'s base to `&v`
	attachment.setBaseValue(interpreter, base)

	base.SetTypeKey(
		interpreter,
		locationRange,
		interpreter.MustSemaTypeOfValue(attachment),
		attachment,
	)

	return base
}
