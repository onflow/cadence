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

package runtime_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
)

// Maximum integer whose square-root can be computed and stored in a UFix64.
var maxSquareInteger *big.Int

func init() {
	// Maximum value supported by a UFix64 is 184467440737.09551615.
	// So the smallest number that overflows is 184467440737.09551616.
	// (184467440737.09551616)^2 = 34028236692093846346337.4607431768211456
	//
	// Note that we have opted of rounding mode ToZero (IEEE 754-2008 roundTowardZero).
	// So we can support Sqrt till 34028236692093846346337 since
	// Sqrt(34028236692093846346337) = 184467440737.09551615999875115311
	// which gets rounded down to 184467440737.09551615
	// Sqrt(34028236692093846346338) = 184467440737.09551616000146165854
	// which gets rounded to 184467440737.09551616 which overflows.
	maxSquareInteger, _ = new(big.Int).SetString("34028236692093846346337", 10)
}

func TestRuntimeMathSqrt(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	script := `

      access(all) fun main(_ data: %s): UFix64 {
          return Math.Sqrt(data)
      }
    `

	newUFix64 := func(t *testing.T, integer int, fraction uint) cadence.UFix64 {
		value, err := cadence.NewUFix64FromParts(integer, fraction)
		require.NoError(t, err)
		return value
	}

	tests := map[string]map[cadence.NumberValue]cadence.UFix64{
		// Int*
		"Int": {
			cadence.NewInt(0):                       newUFix64(t, 0, 0),
			cadence.NewInt(42):                      newUFix64(t, 6, 48074069),
			cadence.NewInt(127):                     newUFix64(t, 11, 26942766),
			cadence.NewIntFromBig(maxSquareInteger): newUFix64(t, 184467440737, 9551615),
		},
		"Int8": {
			cadence.Int8(0):   newUFix64(t, 0, 0),
			cadence.Int8(40):  newUFix64(t, 6, 32455532),
			cadence.Int8(124): newUFix64(t, 11, 13552872),
			cadence.Int8(127): newUFix64(t, 11, 26942766),
		},
		"Int16": {
			cadence.NewInt16(0):     newUFix64(t, 0, 0),
			cadence.NewInt16(40):    newUFix64(t, 6, 32455532),
			cadence.NewInt16(32767): newUFix64(t, 181, 1657382),
			cadence.NewInt16(10000): newUFix64(t, 100, 0),
		},
		"Int32": {
			cadence.NewInt32(0):          newUFix64(t, 0, 0),
			cadence.NewInt32(42):         newUFix64(t, 6, 48074069),
			cadence.NewInt32(10000):      newUFix64(t, 100, 0),
			cadence.NewInt32(2147483647): newUFix64(t, 46340, 95000105),
		},
		"Int64": {
			cadence.NewInt64(0):                   newUFix64(t, 0, 0),
			cadence.NewInt64(42):                  newUFix64(t, 6, 48074069),
			cadence.NewInt64(10000):               newUFix64(t, 100, 0),
			cadence.NewInt64(9223372036854775807): newUFix64(t, 3037000499, 97604969),
		},
		"Int128": {
			cadence.NewInt128(0):                    newUFix64(t, 0, 0),
			cadence.NewInt128(42):                   newUFix64(t, 6, 48074069),
			cadence.NewInt128(127):                  newUFix64(t, 11, 26942766),
			cadence.NewInt128(128):                  newUFix64(t, 11, 31370849),
			cadence.NewInt128(200):                  newUFix64(t, 14, 14213562),
			cadence.NewInt128(10000):                newUFix64(t, 100, 0),
			cadence.Int128{Value: maxSquareInteger}: newUFix64(t, 184467440737, 9551615),
		},
		"Int256": {
			cadence.NewInt256(0):                    newUFix64(t, 0, 0),
			cadence.NewInt256(42):                   newUFix64(t, 6, 48074069),
			cadence.NewInt256(127):                  newUFix64(t, 11, 26942766),
			cadence.NewInt256(128):                  newUFix64(t, 11, 31370849),
			cadence.NewInt256(200):                  newUFix64(t, 14, 14213562),
			cadence.Int256{Value: maxSquareInteger}: newUFix64(t, 184467440737, 9551615),
		},
		// UInt*
		"UInt": {
			cadence.NewUInt(0):                    newUFix64(t, 0, 0),
			cadence.NewUInt(42):                   newUFix64(t, 6, 48074069),
			cadence.NewUInt(127):                  newUFix64(t, 11, 26942766),
			cadence.UInt{Value: maxSquareInteger}: newUFix64(t, 184467440737, 9551615),
		},
		"UInt8": {
			cadence.UInt8(0):   newUFix64(t, 0, 0),
			cadence.UInt8(40):  newUFix64(t, 6, 32455532),
			cadence.UInt8(124): newUFix64(t, 11, 13552872),
			cadence.UInt8(127): newUFix64(t, 11, 26942766),
		},
		"UInt16": {
			cadence.NewUInt16(0):     newUFix64(t, 0, 0),
			cadence.NewUInt16(40):    newUFix64(t, 6, 32455532),
			cadence.NewUInt16(32767): newUFix64(t, 181, 1657382),
			cadence.NewUInt16(10000): newUFix64(t, 100, 0),
		},
		"UInt32": {
			cadence.NewUInt32(0):          newUFix64(t, 0, 0),
			cadence.NewUInt32(42):         newUFix64(t, 6, 48074069),
			cadence.NewUInt32(10000):      newUFix64(t, 100, 0),
			cadence.NewUInt32(2147483647): newUFix64(t, 46340, 95000105),
		},
		"UInt64": {
			cadence.NewUInt64(0):                   newUFix64(t, 0, 0),
			cadence.NewUInt64(42):                  newUFix64(t, 6, 48074069),
			cadence.NewUInt64(10000):               newUFix64(t, 100, 0),
			cadence.NewUInt64(9223372036854775807): newUFix64(t, 3037000499, 97604969),
		},
		"UInt128": {
			cadence.NewUInt128(0):                    newUFix64(t, 0, 0),
			cadence.NewUInt128(42):                   newUFix64(t, 6, 48074069),
			cadence.NewUInt128(127):                  newUFix64(t, 11, 26942766),
			cadence.NewUInt128(128):                  newUFix64(t, 11, 31370849),
			cadence.NewUInt128(200):                  newUFix64(t, 14, 14213562),
			cadence.NewUInt128(10000):                newUFix64(t, 100, 0),
			cadence.UInt128{Value: maxSquareInteger}: newUFix64(t, 184467440737, 9551615),
		},
		"UInt256": {
			cadence.NewUInt256(0):                    newUFix64(t, 0, 0),
			cadence.NewUInt256(42):                   newUFix64(t, 6, 48074069),
			cadence.NewUInt256(127):                  newUFix64(t, 11, 26942766),
			cadence.NewUInt256(128):                  newUFix64(t, 11, 31370849),
			cadence.NewUInt256(200):                  newUFix64(t, 14, 14213562),
			cadence.UInt256{Value: maxSquareInteger}: newUFix64(t, 184467440737, 9551615),
		},
		// Word*
		"Word8": {
			cadence.Word8(0):   newUFix64(t, 0, 0),
			cadence.Word8(40):  newUFix64(t, 6, 32455532),
			cadence.Word8(124): newUFix64(t, 11, 13552872),
			cadence.Word8(127): newUFix64(t, 11, 26942766),
		},
		"Word16": {
			cadence.NewWord16(0):     newUFix64(t, 0, 0),
			cadence.NewWord16(40):    newUFix64(t, 6, 32455532),
			cadence.NewWord16(32767): newUFix64(t, 181, 1657382),
			cadence.NewWord16(10000): newUFix64(t, 100, 0),
		},
		"Word32": {
			cadence.NewWord32(0):          newUFix64(t, 0, 0),
			cadence.NewWord32(42):         newUFix64(t, 6, 48074069),
			cadence.NewWord32(10000):      newUFix64(t, 100, 0),
			cadence.NewWord32(2147483647): newUFix64(t, 46340, 95000105),
		},
		"Word64": {
			cadence.NewWord64(0):                   newUFix64(t, 0, 0),
			cadence.NewWord64(42):                  newUFix64(t, 6, 48074069),
			cadence.NewWord64(10000):               newUFix64(t, 100, 0),
			cadence.NewWord64(9223372036854775807): newUFix64(t, 3037000499, 97604969),
		},
		"Word128": {
			cadence.NewWord128(0):                    newUFix64(t, 0, 0),
			cadence.NewWord128(42):                   newUFix64(t, 6, 48074069),
			cadence.NewWord128(127):                  newUFix64(t, 11, 26942766),
			cadence.NewWord128(128):                  newUFix64(t, 11, 31370849),
			cadence.NewWord128(200):                  newUFix64(t, 14, 14213562),
			cadence.NewWord128(10000):                newUFix64(t, 100, 0),
			cadence.Word128{Value: maxSquareInteger}: newUFix64(t, 184467440737, 9551615),
		},
		"Word256": {
			cadence.NewWord256(0):                    newUFix64(t, 0, 0),
			cadence.NewWord256(42):                   newUFix64(t, 6, 48074069),
			cadence.NewWord256(127):                  newUFix64(t, 11, 26942766),
			cadence.NewWord256(128):                  newUFix64(t, 11, 31370849),
			cadence.NewWord256(200):                  newUFix64(t, 14, 14213562),
			cadence.Word256{Value: maxSquareInteger}: newUFix64(t, 184467440737, 9551615),
		},
		// Fix*
		"Fix64": {
			cadence.Fix64(0):                    newUFix64(t, 0, 0),
			cadence.Fix64(42_00000000):          newUFix64(t, 6, 48074069),
			cadence.Fix64(92233720368_54775807): newUFix64(t, 303700, 4999760),
		},
		// UFix*
		"UFix64": {
			cadence.UFix64(0):                     newUFix64(t, 0, 0),
			cadence.UFix64(42_00000000):           newUFix64(t, 6, 48074069),
			cadence.UFix64(184467440737_09551615): newUFix64(t, 429496, 72959999),
		},
	}

	// Ensure the test cases are complete

	for _, numberType := range sema.AllNumberTypes {
		switch numberType {
		case sema.NumberType, sema.SignedNumberType,
			sema.IntegerType, sema.SignedIntegerType, sema.FixedSizeUnsignedIntegerType,
			sema.FixedPointType, sema.SignedFixedPointType:
			continue
		}

		if _, ok := tests[numberType.String()]; !ok {
			panic(fmt.Sprintf("broken test: missing %s", numberType))
		}
	}

	test := func(
		ty string,
		input cadence.NumberValue,
		expectedResult cadence.UFix64,
	) {
		t.Run(fmt.Sprintf("Sqrt<%s>(%s)", ty, input), func(t *testing.T) {

			t.Parallel()

			runtimeInterface := &TestRuntimeInterface{
				Storage: NewTestLedger(nil, nil),
				OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
					return json.Decode(nil, b)
				},
			}

			result, err := runtime.ExecuteScript(
				Script{
					Source: []byte(fmt.Sprintf(script, ty)),
					Arguments: encodeArgs([]cadence.Value{
						input,
					}),
				},
				Context{
					Interface: runtimeInterface,
					Location:  common.ScriptLocation{},
				},
			)

			require.NoError(t, err)
			assert.Equal(t,
				expectedResult,
				result,
			)
		})
	}

	for ty, tests := range tests {
		for value, expected := range tests {
			test(ty, value, expected)
		}
	}
}

func TestRuntimeMathSqrtInvalid(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	script := `

      access(all) fun main(_ data: %s): UFix64 {
          return Math.Sqrt(data)
      }
    `

	type testCase struct {
		name          string
		ty            string
		input         cadence.NumberValue
		expectedError error
	}

	test := func(tc *testCase) {
		t.Run(fmt.Sprintf("Sqrt<%s>(%s)", tc.ty, tc.input), func(t *testing.T) {

			t.Parallel()

			runtimeInterface := &TestRuntimeInterface{
				Storage: NewTestLedger(nil, nil),
				OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
					return json.Decode(nil, b)
				},
			}

			_, err := runtime.ExecuteScript(
				Script{
					Source: []byte(fmt.Sprintf(script, tc.ty)),
					Arguments: encodeArgs([]cadence.Value{
						tc.input,
					}),
				},
				Context{
					Interface: runtimeInterface,
					Location:  common.ScriptLocation{},
				},
			)

			require.Error(t, err)
			assert.ErrorAs(t, err, tc.expectedError)
		})
	}

	tests := []*testCase{
		{
			name:          "Overflow",
			ty:            "Int",
			input:         cadence.NewIntFromBig(new(big.Int).Add(maxSquareInteger, new(big.Int).SetInt64(1))),
			expectedError: &interpreter.OverflowError{},
		},
		{
			name:          "Negative param underflows",
			ty:            "Int32",
			input:         cadence.NewInt32(-1),
			expectedError: &interpreter.UnderflowError{},
		},
	}

	for _, tc := range tests {
		test(tc)
	}
}
