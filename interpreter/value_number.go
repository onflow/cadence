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
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
)

// NumberValue
type NumberValue interface {
	ComparableValue
	ToInt(locationRange LocationRange) int
	Negate(*Interpreter, LocationRange) NumberValue
	Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	ToBigEndianBytes() []byte
}

func getNumberValueMember(interpreter *Interpreter, v NumberValue, name string, typ sema.Type, locationRange LocationRange) Value {
	switch name {

	case sema.ToStringFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.ToStringFunctionType,
			func(v NumberValue, invocation Invocation) Value {
				interpreter := invocation.Interpreter

				memoryUsage := common.NewStringMemoryUsage(
					OverEstimateNumberStringLength(interpreter, v),
				)
				return NewStringValue(
					interpreter,
					memoryUsage,
					func() string {
						return v.String()
					},
				)
			},
		)

	case sema.ToBigEndianBytesFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.ToBigEndianBytesFunctionType,
			func(v NumberValue, invocation Invocation) Value {
				return ByteSliceToByteArrayValue(
					invocation.Interpreter,
					v.ToBigEndianBytes(),
				)
			},
		)

	case sema.NumericTypeSaturatingAddFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			func(v NumberValue, invocation Invocation) Value {
				other, ok := invocation.Arguments[0].(NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.SaturatingPlus(
					invocation.Interpreter,
					other,
					locationRange,
				)
			},
		)

	case sema.NumericTypeSaturatingSubtractFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			func(v NumberValue, invocation Invocation) Value {
				other, ok := invocation.Arguments[0].(NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.SaturatingMinus(
					invocation.Interpreter,
					other,
					locationRange,
				)
			},
		)

	case sema.NumericTypeSaturatingMultiplyFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			func(v NumberValue, invocation Invocation) Value {
				other, ok := invocation.Arguments[0].(NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.SaturatingMul(
					invocation.Interpreter,
					other,
					locationRange,
				)
			},
		)

	case sema.NumericTypeSaturatingDivideFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			func(v NumberValue, invocation Invocation) Value {
				other, ok := invocation.Arguments[0].(NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.SaturatingDiv(
					invocation.Interpreter,
					other,
					locationRange,
				)
			},
		)
	}

	return nil
}

type IntegerValue interface {
	NumberValue
	BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue
	BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue
	BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue
	BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue
	BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue
}

// BigNumberValue is a number value with an integer value outside the range of int64
type BigNumberValue interface {
	NumberValue
	ByteLength() int
	ToBigInt(memoryGauge common.MemoryGauge) *big.Int
}