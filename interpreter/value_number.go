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
	ToInt(locationRange LocationRange) int
	Negate(context NumberValueArithmeticContext, locationRange LocationRange) NumberValue
	Plus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue
	SaturatingPlus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue
	Minus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue
	SaturatingMinus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue
	Mod(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue
	Mul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue
	SaturatingMul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue
	Div(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue
	SaturatingDiv(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue
	ToBigEndianBytes() []byte
}

func getNumberValueFunctionMember(
	context MemberAccessibleContext,
	v NumberValue,
	name string,
	typ sema.Type,
	locationRange LocationRange,
) FunctionValue {
	switch name {

	case sema.ToStringFunctionName:
		return NewUnifiedBoundHostFunctionValue(
			context,
			v,
			sema.ToStringFunctionType,
			UnifiedNumberToStringFunction,
		)

	case sema.ToBigEndianBytesFunctionName:
		return NewUnifiedBoundHostFunctionValue(
			context,
			v,
			sema.ToBigEndianBytesFunctionType,
			UnifiedNumberToBigEndianBytesFunction,
		)

	case sema.NumericTypeSaturatingAddFunctionName:
		return NewUnifiedBoundHostFunctionValue(
			context,
			v,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			UnifiedNumberSaturatingAddFunction,
		)

	case sema.NumericTypeSaturatingSubtractFunctionName:
		return NewUnifiedBoundHostFunctionValue(
			context,
			v,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			UnifiedNumberSaturatingSubtractFunction,
		)

	case sema.NumericTypeSaturatingMultiplyFunctionName:
		return NewUnifiedBoundHostFunctionValue(
			context,
			v,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			UnifiedNumberSaturatingMultiplyFunction,
		)

	case sema.NumericTypeSaturatingDivideFunctionName:
		return NewUnifiedBoundHostFunctionValue(
			context,
			v,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			UnifiedNumberSaturatingDivideFunction,
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
	BitwiseOr(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue
	BitwiseXor(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue
	BitwiseAnd(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue
	BitwiseLeftShift(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue
	BitwiseRightShift(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue
}

// BigNumberValue is a number value with an integer value outside the range of int64
type BigNumberValue interface {
	NumberValue
	ByteLength() int
	ToBigInt(memoryGauge common.MemoryGauge) *big.Int
}

// all native number functions
var UnifiedNumberToStringFunction = UnifiedNativeFunction(
	func(
		context UnifiedFunctionContext,
		locationRange LocationRange,
		typeParameterGetter TypeParameterGetter,
		receiver Value,
		args ...Value,
	) Value {
		return NumberValueToString(context, receiver.(NumberValue))
	},
)

var UnifiedNumberToBigEndianBytesFunction = UnifiedNativeFunction(
	func(
		context UnifiedFunctionContext,
		locationRange LocationRange,
		typeParameterGetter TypeParameterGetter,
		receiver Value,
		args ...Value,
	) Value {
		return ByteSliceToByteArrayValue(context, receiver.(NumberValue).ToBigEndianBytes())
	},
)

var UnifiedNumberSaturatingAddFunction = UnifiedNativeFunction(
	func(
		context UnifiedFunctionContext,
		locationRange LocationRange,
		typeParameterGetter TypeParameterGetter,
		receiver Value,
		args ...Value,
	) Value {
		other := AssertValueOfType[NumberValue](args[0])
		return receiver.(NumberValue).SaturatingPlus(context, other, locationRange)
	},
)

var UnifiedNumberSaturatingSubtractFunction = UnifiedNativeFunction(
	func(
		context UnifiedFunctionContext,
		locationRange LocationRange,
		typeParameterGetter TypeParameterGetter,
		receiver Value,
		args ...Value,
	) Value {
		other := AssertValueOfType[NumberValue](args[0])
		return receiver.(NumberValue).SaturatingMinus(context, other, locationRange)
	},
)

var UnifiedNumberSaturatingMultiplyFunction = UnifiedNativeFunction(
	func(
		context UnifiedFunctionContext,
		locationRange LocationRange,
		typeParameterGetter TypeParameterGetter,
		receiver Value,
		args ...Value,
	) Value {
		other := AssertValueOfType[NumberValue](args[0])
		return receiver.(NumberValue).SaturatingMul(context, other, locationRange)
	},
)

var UnifiedNumberSaturatingDivideFunction = UnifiedNativeFunction(
	func(
		context UnifiedFunctionContext,
		locationRange LocationRange,
		typeParameterGetter TypeParameterGetter,
		receiver Value,
		args ...Value,
	) Value {
		other := AssertValueOfType[NumberValue](args[0])
		return receiver.(NumberValue).SaturatingDiv(context, other, locationRange)
	},
)
