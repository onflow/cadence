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

package interpreter_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	. "github.com/onflow/cadence/runtime/interpreter"
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
				VariableSizedStaticType{
					Type: PrimitiveStaticTypeUInt64,
				},
				common.Address{},
				UInt64Value(500),
			),
			NewArrayValue(
				inter,
				VariableSizedStaticType{
					Type: PrimitiveStaticTypeInt256,
				},
				common.Address{},
				NewInt256ValueFromBigInt(largeBigInt),
			),
			UInt64Value(500),
			BoolValue(true),
			NewStringValue("test"),
		}

		for _, value := range invalid {
			_, err := ByteArrayValueToByteSlice(value)
			require.Error(t, err)
		}
	})

	t.Run("valid", func(t *testing.T) {

		inter := newTestInterpreter(t)

		invalid := map[Value][]byte{
			NewArrayValue(
				inter,
				VariableSizedStaticType{
					Type: PrimitiveStaticTypeInteger,
				},
				common.Address{},
			): {},
			NewArrayValue(
				inter,
				VariableSizedStaticType{
					Type: PrimitiveStaticTypeInteger,
				},
				common.Address{},
				UInt64Value(2),
				NewUInt128ValueFromUint64(3),
			): {2, 3},
			NewArrayValue(
				inter,
				VariableSizedStaticType{
					Type: PrimitiveStaticTypeInteger,
				},
				common.Address{},
				UInt8Value(4),
				NewIntValueFromInt64(5),
			): {4, 5},
		}

		for value, expected := range invalid {
			result, err := ByteArrayValueToByteSlice(value)
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
			UInt64Value(500),
			NewInt256ValueFromBigInt(largeBigInt),
		}

		for _, value := range invalid {
			_, err := ByteValueToByte(value)
			require.Error(t, err)
		}
	})

	t.Run("valid", func(t *testing.T) {

		const maxInt8Plus2 = math.MaxInt8 + 2

		invalid := map[Value]byte{
			UInt64Value(2):               2,
			NewUInt128ValueFromUint64(3): 3,
			UInt8Value(4):                4,
			NewIntValueFromInt64(5):      5,
			UInt8Value(maxInt8Plus2):     maxInt8Plus2,
		}

		for value, expected := range invalid {
			result, err := ByteValueToByte(value)
			require.NoError(t, err)
			require.Equal(t, expected, result)
		}
	})
}
