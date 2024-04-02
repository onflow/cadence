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
	"math"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

func ByteArrayValueToByteSlice(interpreter *Interpreter, value Value, locationRange LocationRange) ([]byte, error) {
	array, ok := value.(*ArrayValue)
	if !ok {
		return nil, errors.NewDefaultUserError("value is not an array")
	}

	var result []byte

	count := array.Count()
	if count > 0 {
		result = make([]byte, 0, count)

		var err error
		array.Iterate(
			interpreter,
			func(element Value) (resume bool) {
				var b byte
				b, err = ByteValueToByte(interpreter, element, locationRange)
				if err != nil {
					return false
				}

				result = append(result, b)

				return true
			},
			false,
			locationRange,
		)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func ByteValueToByte(memoryGauge common.MemoryGauge, element Value, locationRange LocationRange) (byte, error) {
	var b byte

	switch element := element.(type) {
	case BigNumberValue:
		bigInt := element.ToBigInt(memoryGauge)
		if !bigInt.IsUint64() {
			return 0, errors.NewDefaultUserError("value is not in byte range (0-255)")
		}

		integer := bigInt.Uint64()

		if integer > math.MaxUint8 {
			return 0, errors.NewDefaultUserError("value is not in byte range (0-255)")
		}

		b = byte(integer)

	case NumberValue:
		integer := element.ToInt(locationRange)

		if integer < 0 || integer > math.MaxUint8 {
			return 0, errors.NewDefaultUserError("value is not in byte range (0-255)")
		}

		b = byte(integer)

	default:
		return 0, errors.NewDefaultUserError("value is not an integer")
	}

	return b, nil
}

func ByteSliceToByteArrayValue(interpreter *Interpreter, buf []byte) *ArrayValue {

	common.UseMemory(interpreter, common.NewBytesMemoryUsage(len(buf)))

	var values []Value

	count := len(buf)
	if count > 0 {
		values = make([]Value, count)
		for i, b := range buf {
			values[i] = UInt8Value(b)
		}
	}

	return NewArrayValue(
		interpreter,
		EmptyLocationRange,
		ByteArrayStaticType,
		common.ZeroAddress,
		values...,
	)
}

func ByteSliceToConstantSizedByteArrayValue(interpreter *Interpreter, buf []byte) *ArrayValue {

	common.UseMemory(interpreter, common.NewBytesMemoryUsage(len(buf)))

	var values []Value

	count := len(buf)
	if count > 0 {
		values = make([]Value, count)
		for i, b := range buf {
			values[i] = UInt8Value(b)
		}
	}

	constantSizedByteArrayStaticType := NewConstantSizedStaticType(
		interpreter,
		PrimitiveStaticTypeUInt8,
		int64(len(buf)),
	)

	return NewArrayValue(
		interpreter,
		EmptyLocationRange,
		constantSizedByteArrayStaticType,
		common.ZeroAddress,
		values...,
	)
}
