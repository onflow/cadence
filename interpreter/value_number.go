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

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
)

type NumberValueArithmeticContext interface {
	ValueStaticTypeContext
}

var _ NumberValueArithmeticContext = &Interpreter{}

// NumberValue
type NumberValue interface {
	ComparableValue
	ToInt() int
	Negate(context NumberValueArithmeticContext) NumberValue
	Plus(context NumberValueArithmeticContext, other NumberValue) NumberValue
	SaturatingPlus(context NumberValueArithmeticContext, other NumberValue) NumberValue
	Minus(context NumberValueArithmeticContext, other NumberValue) NumberValue
	SaturatingMinus(context NumberValueArithmeticContext, other NumberValue) NumberValue
	Mod(context NumberValueArithmeticContext, other NumberValue) NumberValue
	Mul(context NumberValueArithmeticContext, other NumberValue) NumberValue
	SaturatingMul(context NumberValueArithmeticContext, other NumberValue) NumberValue
	Div(context NumberValueArithmeticContext, other NumberValue) NumberValue
	SaturatingDiv(context NumberValueArithmeticContext, other NumberValue) NumberValue
	ToBigEndianBytes() []byte
}

func getNumberValueFunctionMember(
	context MemberAccessibleContext,
	v NumberValue,
	name string,
	typ sema.Type,
) FunctionValue {
	switch name {

	case sema.ToStringFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ToStringFunctionType,
			NativeNumberToStringFunction,
		)

	case sema.ToBigEndianBytesFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ToBigEndianBytesFunctionType,
			NativeNumberToBigEndianBytesFunction,
		)

	case sema.NumericTypeSaturatingAddFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			NativeNumberSaturatingAddFunction,
		)

	case sema.NumericTypeSaturatingSubtractFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			NativeNumberSaturatingSubtractFunction,
		)

	case sema.NumericTypeSaturatingMultiplyFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			NativeNumberSaturatingMultiplyFunction,
		)

	case sema.NumericTypeSaturatingDivideFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			NativeNumberSaturatingDivideFunction,
		)
	}

	return nil
}

func NumberValueToString(
	memoryGauge common.MemoryGauge,
	v NumberValue,
) *StringValue {
	memoryUsage := common.NewStringMemoryUsage(
		OverEstimateNumberStringLength(memoryGauge, v),
	)
	return NewStringValue(
		memoryGauge,
		memoryUsage,
		v.String,
	)
}

type IntegerValue interface {
	NumberValue
	BitwiseOr(context ValueStaticTypeContext, other IntegerValue) IntegerValue
	BitwiseXor(context ValueStaticTypeContext, other IntegerValue) IntegerValue
	BitwiseAnd(context ValueStaticTypeContext, other IntegerValue) IntegerValue
	BitwiseLeftShift(context ValueStaticTypeContext, other IntegerValue) IntegerValue
	BitwiseRightShift(context ValueStaticTypeContext, other IntegerValue) IntegerValue
}

// BigNumberValue is a number value with an integer value outside the range of int64
type BigNumberValue interface {
	NumberValue
	ByteLength() int
	ToBigInt(memoryGauge common.MemoryGauge) *big.Int
}

// all native number functions
var NativeNumberToStringFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeParameterGetter,
		receiver Value,
		_ []Value,
	) Value {
		return NumberValueToString(context, receiver.(NumberValue))
	},
)

var NativeNumberToBigEndianBytesFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeParameterGetter,
		receiver Value,
		_ []Value,
	) Value {
		return ByteSliceToByteArrayValue(context, receiver.(NumberValue).ToBigEndianBytes())
	},
)

var NativeNumberSaturatingAddFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeParameterGetter,
		receiver Value,
		args []Value,
	) Value {
		other := AssertValueOfType[NumberValue](args[0])
		return receiver.(NumberValue).SaturatingPlus(context, other)
	},
)

var NativeNumberSaturatingSubtractFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeParameterGetter,
		receiver Value,
		args []Value,
	) Value {
		other := AssertValueOfType[NumberValue](args[0])
		return receiver.(NumberValue).SaturatingMinus(context, other)
	},
)

var NativeNumberSaturatingMultiplyFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeParameterGetter,
		receiver Value,
		args []Value,
	) Value {
		other := AssertValueOfType[NumberValue](args[0])
		return receiver.(NumberValue).SaturatingMul(context, other)
	},
)

var NativeNumberSaturatingDivideFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeParameterGetter,
		receiver Value,
		args []Value,
	) Value {
		other := AssertValueOfType[NumberValue](args[0])
		return receiver.(NumberValue).SaturatingDiv(context, other)
	},
)
