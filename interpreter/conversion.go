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
	goerrors "errors"
	"math"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
)

func ByteArrayValueToByteSlice(context ContainerMutationContext, value Value) ([]byte, error) {
	array, ok := value.(*ArrayValue)
	if !ok {
		return nil, errors.NewDefaultUserError("value is not an array")
	}

	count := array.Count()
	if count == 0 {
		return nil, nil
	}

	common.UseMemory(context, common.NewBytesMemoryUsage(count))

	// Optimize conversion from cadence byte array to Go []byte.
	if array.Type.ElementType() == PrimitiveStaticTypeUInt8 {
		b, err := atree.ByteArrayToByteSlice[UInt8Value](array.array)
		if err == nil {
			return b, nil
		}

		var unexpectedElementTypeError *atree.UnexpectedElementTypeError
		if !goerrors.As(err, &unexpectedElementTypeError) {
			return nil, err
		}

		// If error is atree.UnexpectedElementTypeError, try again using iteration and element conversion approach.
		// NOTE: This should never happen because we check array element type before calling atree.ByteArrayToByteSlice.
	}

	result := make([]byte, 0, count)

	var err error
	array.Iterate(
		context,
		func(element Value) (resume bool) {
			var b byte
			b, err = ByteValueToByte(context, element)
			if err != nil {
				return false
			}

			result = append(result, b)

			return true
		},
		false,
	)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func ByteValueToByte(memoryGauge common.MemoryGauge, element Value) (byte, error) {
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
		integer := element.ToInt()

		if integer < 0 || integer > math.MaxUint8 {
			return 0, errors.NewDefaultUserError("value is not in byte range (0-255)")
		}

		b = byte(integer)

	default:
		return 0, errors.NewDefaultUserError("value is not an integer")
	}

	return b, nil
}

var estimatedEncodedUint8Size = UInt8Value(math.MaxUint8).ByteSize()

func ByteSliceToByteArrayValue(context ArrayCreationContext, bytes []byte) *ArrayValue {
	return ByteSliceToByteArrayValueWithType(context, ByteArrayStaticType, bytes)
}

func ByteSliceToConstantSizedByteArrayValue(context ArrayCreationContext, bytes []byte) *ArrayValue {
	count := len(bytes)

	constantSizedByteArrayStaticType := NewConstantSizedStaticType(
		context,
		PrimitiveStaticTypeUInt8,
		int64(count),
	)

	return ByteSliceToByteArrayValueWithType(context, constantSizedByteArrayStaticType, bytes)
}

func ByteSliceToByteArrayValueWithType(
	context ArrayCreationContext,
	arrayType ArrayStaticType,
	bytes []byte,
) *ArrayValue {
	atreeArray, err := atree.ByteSliceToByteArray[UInt8Value](
		context.Storage(),
		atree.Address(common.ZeroAddress),
		arrayType,
		bytes,
		estimatedEncodedUint8Size,
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return newArrayValueFromAtreeArray(context, arrayType, ArrayElementSize(ByteArrayStaticType), atreeArray)
}
