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

package interpreter_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	. "github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestByteArrayValueToByteSlice(t *testing.T) {

	t.Parallel()

	t.Run("invalid", func(t *testing.T) {

		largeBigInt, ok := new(big.Int).SetString("1000000000000000000000000000000000000000000000", 10)
		require.True(t, ok)

		inter := newTestInterpreter(t)

		invalid := []Value{
			NewArrayValue(
				inter,
				EmptyLocationRange,
				VariableSizedStaticType{
					Type: PrimitiveStaticTypeUInt64,
				},
				common.ZeroAddress,
				NewUnmeteredUInt64Value(500),
			),
			NewArrayValue(
				inter,
				EmptyLocationRange,
				VariableSizedStaticType{
					Type: PrimitiveStaticTypeInt256,
				},
				common.ZeroAddress,
				NewUnmeteredInt256ValueFromBigInt(largeBigInt),
			),
			NewUnmeteredUInt64Value(500),
			TrueValue,
			NewUnmeteredStringValue("test"),
		}

		for _, value := range invalid {
			_, err := ByteArrayValueToByteSlice(inter, value, EmptyLocationRange)
			RequireError(t, err)
		}
	})

	t.Run("valid", func(t *testing.T) {

		inter := newTestInterpreter(t)

		invalid := map[Value][]byte{
			NewArrayValue(
				inter,
				EmptyLocationRange,
				VariableSizedStaticType{
					Type: PrimitiveStaticTypeInteger,
				},
				common.ZeroAddress,
			): nil,
			NewArrayValue(
				inter,
				EmptyLocationRange,
				VariableSizedStaticType{
					Type: PrimitiveStaticTypeInteger,
				},
				common.ZeroAddress,
				NewUnmeteredUInt64Value(2),
				NewUnmeteredUInt128ValueFromUint64(3),
			): {2, 3},
			NewArrayValue(
				inter,
				EmptyLocationRange,
				VariableSizedStaticType{
					Type: PrimitiveStaticTypeInteger,
				},
				common.ZeroAddress,
				NewUnmeteredUInt8Value(4),
				NewUnmeteredIntValueFromInt64(5),
			): {4, 5},
		}

		for value, expected := range invalid {
			result, err := ByteArrayValueToByteSlice(inter, value, EmptyLocationRange)
			require.NoError(t, err)
			require.Equal(t, expected, result)
		}
	})
}

func TestByteValueToByte(t *testing.T) {

	t.Parallel()

	t.Run("invalid", func(t *testing.T) {

		largeBigInt, ok := new(big.Int).SetString("1000000000000000000000000000000000000000000000", 10)
		require.True(t, ok)

		invalid := []Value{
			NewUnmeteredUInt64Value(500),
			NewUnmeteredInt256ValueFromBigInt(largeBigInt),
		}

		for _, value := range invalid {
			_, err := ByteValueToByte(nil, value, EmptyLocationRange)
			RequireError(t, err)
		}
	})

	t.Run("valid", func(t *testing.T) {

		const maxInt8Plus2 = math.MaxInt8 + 2

		invalid := map[Value]byte{
			NewUnmeteredUInt64Value(2):            2,
			NewUnmeteredUInt128ValueFromUint64(3): 3,
			NewUnmeteredUInt8Value(4):             4,
			NewUnmeteredIntValueFromInt64(5):      5,
			NewUnmeteredUInt8Value(maxInt8Plus2):  maxInt8Plus2,
		}

		for value, expected := range invalid {
			result, err := ByteValueToByte(nil, value, EmptyLocationRange)
			require.NoError(t, err)
			require.Equal(t, expected, result)
		}
	})
}

func TestByteSliceToArrayValue(t *testing.T) {
	t.Parallel()

	t.Run("variable sized", func(t *testing.T) {
		b := []byte{0, 1, 2}

		inter := newTestInterpreter(t)

		expectedType := VariableSizedStaticType{
			Type: PrimitiveStaticTypeUInt8,
		}

		expected := NewArrayValue(
			inter,
			EmptyLocationRange,
			expectedType,
			common.ZeroAddress,
			NewUnmeteredUInt8Value(0),
			NewUnmeteredUInt8Value(1),
			NewUnmeteredUInt8Value(2),
		)

		result := ByteSliceToByteArrayValue(inter, b)
		require.Equal(t, expectedType, result.Type)
		require.True(t, result.Equal(inter, EmptyLocationRange, expected))
	})

	t.Run("const sized", func(t *testing.T) {
		b := []byte{0, 1, 2}

		inter := newTestInterpreter(t)

		expectedType := ConstantSizedStaticType{
			Size: int64(len(b)),
			Type: PrimitiveStaticTypeUInt8,
		}

		expected := NewArrayValue(
			inter,
			EmptyLocationRange,
			expectedType,
			common.ZeroAddress,
			NewUnmeteredUInt8Value(0),
			NewUnmeteredUInt8Value(1),
			NewUnmeteredUInt8Value(2),
		)

		result := ByteSliceToConstantSizedByteArrayValue(inter, b)
		require.Equal(t, expectedType, result.Type)
		require.True(t, result.Equal(inter, EmptyLocationRange, expected))
	})
}
