/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

	"github.com/onflow/cadence/fixedpoint"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

// assignmentGetterSetter returns a getter/setter function pair
// for the target expression
//
func (interpreter *Interpreter) assignmentGetterSetter(expression ast.Expression) getterSetter {
	switch expression := expression.(type) {
	case *ast.IdentifierExpression:
		return interpreter.identifierExpressionGetterSetter(expression)

	case *ast.IndexExpression:
		return interpreter.indexExpressionGetterSetter(expression)

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
//
func (interpreter *Interpreter) identifierExpressionGetterSetter(identifierExpression *ast.IdentifierExpression) getterSetter {
	identifier := identifierExpression.Identifier.Identifier
	variable := interpreter.findVariable(identifier)

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

// indexExpressionGetterSetter returns a getter/setter function pair
// for the target index expression
//
func (interpreter *Interpreter) indexExpressionGetterSetter(indexExpression *ast.IndexExpression) getterSetter {
	target, ok := interpreter.evalExpression(indexExpression.TargetExpression).(ValueIndexableValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	getLocationRange := locationRangeGetter(interpreter, interpreter.Location, indexExpression)

	// Evaluate, transfer, and convert the indexing value,
	// as it is essentially an "argument" of the get/set operation

	elaboration := interpreter.Program.Elaboration

	indexedType := elaboration.IndexExpressionIndexedTypes[indexExpression]
	indexingType := elaboration.IndexExpressionIndexingTypes[indexExpression]

	transferredIndexingValue := interpreter.transferAndConvert(
		interpreter.evalExpression(indexExpression.IndexingExpression),
		indexingType,
		indexedType.IndexingType(),
		locationRangeGetter(
			interpreter,
			interpreter.Location,
			indexExpression.IndexingExpression,
		),
	)

	_, isNestedResourceMove := elaboration.IsNestedResourceMoveExpression[indexExpression]

	return getterSetter{
		target: target,
		get: func(_ bool) Value {
			if isNestedResourceMove {
				return target.RemoveKey(interpreter, getLocationRange, transferredIndexingValue)
			} else {
				return target.GetKey(interpreter, getLocationRange, transferredIndexingValue)
			}
		},
		set: func(value Value) {
			if isNestedResourceMove {
				target.InsertKey(interpreter, getLocationRange, transferredIndexingValue, value)
			} else {
				target.SetKey(interpreter, getLocationRange, transferredIndexingValue, value)
			}
		},
	}
}

// memberExpressionGetterSetter returns a getter/setter function pair
// for the target member expression
//
func (interpreter *Interpreter) memberExpressionGetterSetter(memberExpression *ast.MemberExpression) getterSetter {
	target := interpreter.evalExpression(memberExpression.Expression)
	identifier := memberExpression.Identifier.Identifier
	getLocationRange := locationRangeGetter(interpreter, interpreter.Location, memberExpression)
	_, isNestedResourceMove := interpreter.Program.Elaboration.IsNestedResourceMoveExpression[memberExpression]
	return getterSetter{
		target: target,
		get: func(allowMissing bool) Value {
			isOptional := memberExpression.Optional

			if isOptional {
				switch typedTarget := target.(type) {
				case NilValue:
					return typedTarget

				case *SomeValue:
					target = typedTarget.InnerValue(interpreter, getLocationRange)

				default:
					panic(errors.NewUnreachableError())
				}
			}

			var resultValue Value
			if isNestedResourceMove {
				resultValue = target.(MemberAccessibleValue).RemoveMember(interpreter, getLocationRange, identifier)
			} else {
				resultValue = interpreter.getMember(target, getLocationRange, identifier)
			}
			if resultValue == nil && !allowMissing {
				panic(MissingMemberValueError{
					Name:          identifier,
					LocationRange: getLocationRange(),
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
			interpreter.setMember(target, getLocationRange, identifier, value)
		},
	}
}

func (interpreter *Interpreter) VisitIdentifierExpression(expression *ast.IdentifierExpression) ast.Repr {
	name := expression.Identifier.Identifier
	variable := interpreter.findVariable(name)
	value := variable.GetValue()

	interpreter.checkInvalidatedResourceUse(value, variable, name, expression)

	return value
}

func (interpreter *Interpreter) evalExpression(expression ast.Expression) Value {
	return expression.Accept(interpreter).(Value)
}

func (interpreter *Interpreter) VisitBinaryExpression(expression *ast.BinaryExpression) ast.Repr {

	leftValue := interpreter.evalExpression(expression.Left)

	// We make this a thunk so that we can skip computing it for certain short-circuiting operations
	rightValue := func() Value {
		return interpreter.evalExpression(expression.Right)
	}

	error := func(right Value) {
		panic(InvalidOperandsError{
			Operation:     expression.Operation,
			LeftType:      leftValue.StaticType(interpreter),
			RightType:     right.StaticType(interpreter),
			LocationRange: locationRangeGetter(interpreter, interpreter.Location, expression)(),
		})
	}

	switch expression.Operation {
	case ast.OperationPlus:
		left, leftOk := leftValue.(NumberValue)
		right, rightOk := rightValue().(NumberValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.Plus(interpreter, right)

	case ast.OperationMinus:
		left, leftOk := leftValue.(NumberValue)
		right, rightOk := rightValue().(NumberValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.Minus(interpreter, right)

	case ast.OperationMod:
		left, leftOk := leftValue.(NumberValue)
		right, rightOk := rightValue().(NumberValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.Mod(interpreter, right)

	case ast.OperationMul:
		left, leftOk := leftValue.(NumberValue)
		right, rightOk := rightValue().(NumberValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.Mul(interpreter, right)

	case ast.OperationDiv:
		left, leftOk := leftValue.(NumberValue)
		right, rightOk := rightValue().(NumberValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.Div(interpreter, right)

	case ast.OperationBitwiseOr:
		left, leftOk := leftValue.(IntegerValue)
		right, rightOk := rightValue().(IntegerValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.BitwiseOr(interpreter, right)

	case ast.OperationBitwiseXor:
		left, leftOk := leftValue.(IntegerValue)
		right, rightOk := rightValue().(IntegerValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.BitwiseXor(interpreter, right)

	case ast.OperationBitwiseAnd:
		left, leftOk := leftValue.(IntegerValue)
		right, rightOk := rightValue().(IntegerValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.BitwiseAnd(interpreter, right)

	case ast.OperationBitwiseLeftShift:
		left, leftOk := leftValue.(IntegerValue)
		right, rightOk := rightValue().(IntegerValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.BitwiseLeftShift(interpreter, right)

	case ast.OperationBitwiseRightShift:
		left, leftOk := leftValue.(IntegerValue)
		right, rightOk := rightValue().(IntegerValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.BitwiseRightShift(interpreter, right)

	case ast.OperationLess:
		left, leftOk := leftValue.(NumberValue)
		right, rightOk := rightValue().(NumberValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.Less(interpreter, right)

	case ast.OperationLessEqual:
		left, leftOk := leftValue.(NumberValue)
		right, rightOk := rightValue().(NumberValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.LessEqual(interpreter, right)

	case ast.OperationGreater:
		left, leftOk := leftValue.(NumberValue)
		right, rightOk := rightValue().(NumberValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.Greater(interpreter, right)

	case ast.OperationGreaterEqual:
		left, leftOk := leftValue.(NumberValue)
		right, rightOk := rightValue().(NumberValue)
		if !leftOk || !rightOk {
			error(right)
		}
		return left.GreaterEqual(interpreter, right)

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
		getLocationRange := locationRangeGetter(interpreter, interpreter.Location, expression)

		// only evaluate right-hand side if left-hand side is nil
		if some, ok := leftValue.(*SomeValue); ok {
			return some.InnerValue(interpreter, getLocationRange)
		}

		value := rightValue()

		rightType := interpreter.Program.Elaboration.BinaryExpressionRightTypes[expression]
		resultType := interpreter.Program.Elaboration.BinaryExpressionResultTypes[expression]

		// NOTE: important to convert both any and optional
		return interpreter.ConvertAndBox(getLocationRange, value, rightType, resultType)
	}

	panic(&unsupportedOperation{
		kind:      common.OperationKindBinary,
		operation: expression.Operation,
		Range:     ast.NewUnmeteredRangeFromPositioned(expression),
	})
}

func (interpreter *Interpreter) testEqual(left, right Value, expression *ast.BinaryExpression) BoolValue {
	left = interpreter.Unbox(
		locationRangeGetter(interpreter, interpreter.Location, expression.Left),
		left,
	)

	right = interpreter.Unbox(
		locationRangeGetter(interpreter, interpreter.Location, expression.Right),
		right,
	)

	valueGetter := func() bool {
		leftEquatable, ok := left.(EquatableValue)
		if !ok {
			return false
		}

		return leftEquatable.Equal(
			interpreter,
			locationRangeGetter(interpreter, interpreter.Location, expression),
			right,
		)
	}

	return NewBoolValueFromConstructor(interpreter, valueGetter)
}

func (interpreter *Interpreter) VisitUnaryExpression(expression *ast.UnaryExpression) ast.Repr {
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
		return integerValue.Negate(interpreter)

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

func (interpreter *Interpreter) VisitBoolExpression(expression *ast.BoolExpression) ast.Repr {
	return NewBoolValue(interpreter, expression.Value)
}

func (interpreter *Interpreter) VisitNilExpression(_ *ast.NilExpression) ast.Repr {
	return NewNilValue(interpreter)
}

func (interpreter *Interpreter) VisitIntegerExpression(expression *ast.IntegerExpression) ast.Repr {
	typ := interpreter.Program.Elaboration.IntegerExpressionType[expression]

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
//
func (interpreter *Interpreter) NewIntegerValueFromBigInt(value *big.Int, integerSubType sema.Type) Value {
	memoryGauge := interpreter.memoryGauge

	// NOTE: cases meter manually and call the unmetered constructors to avoid allocating closures

	switch integerSubType {
	case sema.IntType, sema.IntegerType, sema.SignedIntegerType:
		common.UseMemory(
			memoryGauge,
			common.NewBigIntMemoryUsage(
				common.BigIntByteLength(value),
			),
		)
		return NewUnmeteredIntValueFromBigInt(value)
	case sema.UIntType:
		common.UseMemory(
			memoryGauge,
			common.NewBigIntMemoryUsage(
				common.BigIntByteLength(value),
			),
		)
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
		common.UseMemory(memoryGauge, Int128MemoryUsage)
		return NewUnmeteredInt128ValueFromBigInt(value)
	case sema.Int256Type:
		common.UseMemory(memoryGauge, Int256MemoryUsage)
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
		common.UseMemory(memoryGauge, Uint128MemoryUsage)
		return NewUnmeteredUInt128ValueFromBigInt(value)
	case sema.UInt256Type:
		common.UseMemory(memoryGauge, Uint256MemoryUsage)
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

func (interpreter *Interpreter) VisitFixedPointExpression(expression *ast.FixedPointExpression) ast.Repr {
	// TODO: adjust once/if we support more fixed point types

	fixedPointSubType := interpreter.Program.Elaboration.FixedPointExpression[expression]

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

func (interpreter *Interpreter) VisitStringExpression(expression *ast.StringExpression) ast.Repr {
	stringType := interpreter.Program.Elaboration.StringExpressionType[expression]

	switch stringType {
	case sema.CharacterType:
		return NewUnmeteredCharacterValue(expression.Value)
	}

	// NOTE: already metered in lexer/parser
	return NewUnmeteredStringValue(expression.Value)
}

func (interpreter *Interpreter) VisitArrayExpression(expression *ast.ArrayExpression) ast.Repr {
	values := interpreter.visitExpressionsNonCopying(expression.Values)

	argumentTypes := interpreter.Program.Elaboration.ArrayExpressionArgumentTypes[expression]
	arrayType := interpreter.Program.Elaboration.ArrayExpressionArrayType[expression]
	elementType := arrayType.ElementType(false)

	copies := make([]Value, len(values))
	for i, argument := range values {
		argumentType := argumentTypes[i]
		argumentExpression := expression.Values[i]
		getLocationRange := locationRangeGetter(interpreter, interpreter.Location, argumentExpression)
		copies[i] = interpreter.transferAndConvert(argument, argumentType, elementType, getLocationRange)
	}

	// TODO: cache
	arrayStaticType := ConvertSemaArrayTypeToStaticArrayType(interpreter, arrayType)

	return NewArrayValue(
		interpreter,
		arrayStaticType,
		common.Address{},
		copies...,
	)
}

func (interpreter *Interpreter) VisitDictionaryExpression(expression *ast.DictionaryExpression) ast.Repr {
	values := interpreter.visitEntries(expression.Entries)

	entryTypes := interpreter.Program.Elaboration.DictionaryExpressionEntryTypes[expression]
	dictionaryType := interpreter.Program.Elaboration.DictionaryExpressionType[expression]

	var keyValuePairs []Value

	for i, dictionaryEntryValues := range values {
		entryType := entryTypes[i]
		entry := expression.Entries[i]

		key := interpreter.transferAndConvert(
			dictionaryEntryValues.Key,
			entryType.KeyType,
			dictionaryType.KeyType,
			locationRangeGetter(interpreter, interpreter.Location, entry.Key),
		)

		value := interpreter.transferAndConvert(
			dictionaryEntryValues.Value,
			entryType.ValueType,
			dictionaryType.ValueType,
			locationRangeGetter(interpreter, interpreter.Location, entry.Value),
		)

		// TODO: panic for duplicate keys?

		keyValuePairs = append(
			keyValuePairs,
			key,
			value,
		)
	}

	dictionaryStaticType := ConvertSemaDictionaryTypeToStaticDictionaryType(interpreter, dictionaryType)

	return NewDictionaryValue(interpreter, dictionaryStaticType, keyValuePairs...)
}

func (interpreter *Interpreter) VisitMemberExpression(expression *ast.MemberExpression) ast.Repr {
	const allowMissing = false
	return interpreter.memberExpressionGetterSetter(expression).get(allowMissing)
}

func (interpreter *Interpreter) VisitIndexExpression(expression *ast.IndexExpression) ast.Repr {
	typedResult, ok := interpreter.evalExpression(expression.TargetExpression).(ValueIndexableValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	indexingValue := interpreter.evalExpression(expression.IndexingExpression)
	getLocationRange := locationRangeGetter(interpreter, interpreter.Location, expression)
	return typedResult.GetKey(interpreter, getLocationRange, indexingValue)
}

func (interpreter *Interpreter) VisitConditionalExpression(expression *ast.ConditionalExpression) ast.Repr {
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

func (interpreter *Interpreter) VisitInvocationExpression(invocationExpression *ast.InvocationExpression) ast.Repr {

	// tracing
	if interpreter.tracingEnabled {
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
				locationRangeGetter(interpreter, interpreter.Location, invocationExpression.InvokedExpression),
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
	argumentExpressions := make([]ast.Expression, len(invocationExpression.Arguments))
	for i, argument := range invocationExpression.Arguments {
		argumentExpressions[i] = argument.Expression
	}

	arguments := interpreter.visitExpressionsNonCopying(argumentExpressions)

	typeParameterTypes :=
		interpreter.Program.Elaboration.InvocationExpressionTypeArguments[invocationExpression]
	argumentTypes :=
		interpreter.Program.Elaboration.InvocationExpressionArgumentTypes[invocationExpression]
	parameterTypes :=
		interpreter.Program.Elaboration.InvocationExpressionParameterTypes[invocationExpression]

	line := invocationExpression.StartPosition().Line

	interpreter.reportFunctionInvocation(line)

	resultValue := interpreter.invokeFunctionValue(
		function,
		arguments,
		argumentExpressions,
		argumentTypes,
		parameterTypes,
		typeParameterTypes,
		invocationExpression,
	)

	interpreter.reportInvokedFunctionReturn(line)

	// If this is invocation is optional chaining, wrap the result
	// as an optional, as the result is expected to be an optional
	if isOptionalChaining {
		resultValue = NewSomeValueNonCopying(interpreter, resultValue)
	}

	return resultValue
}

func (interpreter *Interpreter) visitExpressionsNonCopying(expressions []ast.Expression) []Value {
	values := make([]Value, 0, len(expressions))

	for _, expression := range expressions {
		value := interpreter.evalExpression(expression)
		values = append(values, value)
	}

	return values
}

func (interpreter *Interpreter) visitEntries(entries []ast.DictionaryEntry) []DictionaryEntryValues {
	values := make([]DictionaryEntryValues, 0, len(entries))

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

	return values
}

func (interpreter *Interpreter) VisitFunctionExpression(expression *ast.FunctionExpression) ast.Repr {

	// lexical scope: variables in functions are bound to what is visible at declaration time
	lexicalScope := interpreter.activations.CurrentOrNew()

	functionType := interpreter.Program.Elaboration.FunctionExpressionFunctionType[expression]

	var preConditions ast.Conditions
	if expression.FunctionBlock.PreConditions != nil {
		preConditions = *expression.FunctionBlock.PreConditions
	}

	var beforeStatements []ast.Statement
	var rewrittenPostConditions ast.Conditions

	if expression.FunctionBlock.PostConditions != nil {
		postConditionsRewrite :=
			interpreter.Program.Elaboration.PostConditionsRewrite[expression.FunctionBlock.PostConditions]

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

func (interpreter *Interpreter) VisitCastingExpression(expression *ast.CastingExpression) ast.Repr {
	value := interpreter.evalExpression(expression.Expression)

	getLocationRange := locationRangeGetter(interpreter, interpreter.Location, expression.Expression)

	expectedType := interpreter.Program.Elaboration.CastingTargetTypes[expression]

	switch expression.Operation {
	case ast.OperationFailableCast, ast.OperationForceCast:
		isSubType := interpreter.IsSubTypeOfSemaType(value.StaticType(interpreter), expectedType)

		switch expression.Operation {
		case ast.OperationFailableCast:
			if !isSubType {
				return NewNilValue(interpreter)
			}

			// The failable cast may upcast to an optional type, e.g. `1 as? Int?`, so box
			value = interpreter.BoxOptional(getLocationRange, value, expectedType)

			return NewSomeValueNonCopying(interpreter, value)

		case ast.OperationForceCast:
			if !isSubType {
				getLocationRange := locationRangeGetter(interpreter, interpreter.Location, expression.Expression)
				panic(ForceCastTypeMismatchError{
					ExpectedType:  expectedType,
					LocationRange: getLocationRange(),
				})
			}

			// The failable cast may upcast to an optional type, e.g. `1 as? Int?`, so box
			return interpreter.BoxOptional(getLocationRange, value, expectedType)

		default:
			panic(errors.NewUnreachableError())
		}

	case ast.OperationCast:
		staticValueType := interpreter.Program.Elaboration.CastingStaticValueTypes[expression]
		// The cast may upcast to an optional type, e.g. `1 as Int?`, so box
		return interpreter.ConvertAndBox(getLocationRange, value, staticValueType, expectedType)

	default:
		panic(errors.NewUnreachableError())
	}
}

func (interpreter *Interpreter) VisitCreateExpression(expression *ast.CreateExpression) ast.Repr {
	return interpreter.evalExpression(expression.InvocationExpression)
}

func (interpreter *Interpreter) VisitDestroyExpression(expression *ast.DestroyExpression) ast.Repr {
	value := interpreter.evalExpression(expression.Expression)

	interpreter.invalidateResource(value)

	getLocationRange := locationRangeGetter(interpreter, interpreter.Location, expression)

	value.(ResourceKindedValue).Destroy(interpreter, getLocationRange)

	return NewVoidValue(interpreter)
}

func (interpreter *Interpreter) VisitReferenceExpression(referenceExpression *ast.ReferenceExpression) ast.Repr {

	borrowType := interpreter.Program.Elaboration.ReferenceExpressionBorrowTypes[referenceExpression]

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

			getLocationRange := locationRangeGetter(interpreter, interpreter.Location, referenceExpression.Expression)

			innerValue := result.InnerValue(interpreter, getLocationRange)
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
			return NewNilValue(interpreter)

		default:
			// If the referenced value is non-optional,
			// but the target type is optional,
			// then box the reference properly

			getLocationRange := locationRangeGetter(interpreter, interpreter.Location, referenceExpression)

			return interpreter.BoxOptional(
				getLocationRange,
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

func (interpreter *Interpreter) VisitForceExpression(expression *ast.ForceExpression) ast.Repr {
	result := interpreter.evalExpression(expression.Expression)

	switch result := result.(type) {
	case *SomeValue:
		getLocationRange := locationRangeGetter(interpreter, interpreter.Location, expression.Expression)
		return result.InnerValue(interpreter, getLocationRange)

	case NilValue:
		panic(
			ForceNilError{
				LocationRange: LocationRange{
					Location: interpreter.Location,
					Range: ast.NewUnmeteredRange(
						expression.EndPosition(nil),
						expression.EndPosition(nil),
					),
				},
			},
		)

	default:
		return result
	}
}

func (interpreter *Interpreter) VisitPathExpression(expression *ast.PathExpression) ast.Repr {
	domain := common.PathDomainFromIdentifier(expression.Domain.Identifier)

	// meter the Path's Identifier since path is just a container
	common.UseMemory(interpreter, common.NewRawStringMemoryUsage(len(expression.Identifier.Identifier)))

	return NewPathValue(
		interpreter,
		domain,
		expression.Identifier.Identifier,
	)
}
