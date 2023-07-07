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
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

var integerTestValues = map[string]interpreter.NumberValue{
	// Int*
	"Int":    interpreter.NewUnmeteredIntValueFromInt64(60),
	"Int8":   interpreter.NewUnmeteredInt8Value(60),
	"Int16":  interpreter.NewUnmeteredInt16Value(60),
	"Int32":  interpreter.NewUnmeteredInt32Value(60),
	"Int64":  interpreter.NewUnmeteredInt64Value(60),
	"Int128": interpreter.NewUnmeteredInt128ValueFromInt64(60),
	"Int256": interpreter.NewUnmeteredInt256ValueFromInt64(60),
	// UInt*
	"UInt":    interpreter.NewUnmeteredUIntValueFromUint64(60),
	"UInt8":   interpreter.NewUnmeteredUInt8Value(60),
	"UInt16":  interpreter.NewUnmeteredUInt16Value(60),
	"UInt32":  interpreter.NewUnmeteredUInt32Value(60),
	"UInt64":  interpreter.NewUnmeteredUInt64Value(60),
	"UInt128": interpreter.NewUnmeteredUInt128ValueFromUint64(60),
	"UInt256": interpreter.NewUnmeteredUInt256ValueFromUint64(60),
	// Word*
	"Word8":   interpreter.NewUnmeteredWord8Value(60),
	"Word16":  interpreter.NewUnmeteredWord16Value(60),
	"Word32":  interpreter.NewUnmeteredWord32Value(60),
	"Word64":  interpreter.NewUnmeteredWord64Value(60),
	"Word128": interpreter.NewUnmeteredWord128ValueFromUint64(60),
	"Word256": interpreter.NewUnmeteredWord256ValueFromUint64(60),
}

func init() {

	for _, integerType := range sema.AllIntegerTypes {
		switch integerType {
		case sema.IntegerType, sema.SignedIntegerType:
			continue
		}

		if _, ok := integerTestValues[integerType.String()]; !ok {
			panic(fmt.Sprintf("broken test: missing %s", integerType))
		}
	}
}

func TestInterpretPlusOperator(t *testing.T) {

	t.Parallel()

	for ty, value := range integerTestValues {

		t.Run(ty, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      let a: %[1]s = 20
                      let b: %[1]s = 40
                      let c = a + b
                    `,
					ty,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				value,
				inter.Globals.Get("c").GetValue(),
			)
		})
	}
}

func TestInterpretMinusOperator(t *testing.T) {

	t.Parallel()

	for ty, value := range integerTestValues {

		t.Run(ty, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      let a: %[1]s = 80
                      let b: %[1]s = 20
                      let c = a - b
                    `,
					ty,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				value,
				inter.Globals.Get("c").GetValue(),
			)
		})
	}
}

func TestInterpretMulOperator(t *testing.T) {

	t.Parallel()

	for ty, value := range integerTestValues {

		t.Run(ty, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      let a: %[1]s = 20
                      let b: %[1]s = 3
                      let c = a * b
                    `,
					ty,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				value,
				inter.Globals.Get("c").GetValue(),
			)
		})
	}
}

func TestInterpretDivOperator(t *testing.T) {

	t.Parallel()

	for ty, value := range integerTestValues {

		t.Run(ty, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      let a: %[1]s = 120
                      let b: %[1]s = 2
                      let c = a / b
                    `,
					ty,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				value,
				inter.Globals.Get("c").GetValue(),
			)
		})
	}
}

func TestInterpretModOperator(t *testing.T) {

	t.Parallel()

	for ty, value := range integerTestValues {

		t.Run(ty, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      let a: %[1]s = 126
                      let b: %[1]s = 66
                      let c = a %% b
                    `,
					ty,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				value,
				inter.Globals.Get("c").GetValue(),
			)
		})
	}
}

func TestInterpretSaturatedArithmeticFunctions(t *testing.T) {

	t.Parallel()

	type testCall struct {
		left, right interpreter.Value
		expected    interpreter.EquatableValue
	}

	type testCalls struct {
		underflow, overflow testCall
	}

	type testCase struct {
		add, subtract, multiply, divide testCalls
	}

	testCases := map[sema.Type]testCase{
		sema.Int8Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt8Value(math.MaxInt8),
					interpreter.NewUnmeteredInt8Value(2),
					interpreter.NewUnmeteredInt8Value(math.MaxInt8),
				},
				underflow: testCall{
					interpreter.NewUnmeteredInt8Value(math.MinInt8),
					interpreter.NewUnmeteredInt8Value(-2),
					interpreter.NewUnmeteredInt8Value(math.MinInt8),
				},
			},
			subtract: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt8Value(math.MaxInt8),
					interpreter.NewUnmeteredInt8Value(-2),
					interpreter.NewUnmeteredInt8Value(math.MaxInt8),
				},
				underflow: testCall{
					interpreter.NewUnmeteredInt8Value(math.MinInt8),
					interpreter.NewUnmeteredInt8Value(2),
					interpreter.NewUnmeteredInt8Value(math.MinInt8),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt8Value(math.MaxInt8),
					interpreter.NewUnmeteredInt8Value(2),
					interpreter.NewUnmeteredInt8Value(math.MaxInt8),
				},
				underflow: testCall{
					interpreter.NewUnmeteredInt8Value(math.MinInt8),
					interpreter.NewUnmeteredInt8Value(2),
					interpreter.NewUnmeteredInt8Value(math.MinInt8),
				},
			},
			divide: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt8Value(math.MinInt8),
					interpreter.NewUnmeteredInt8Value(-1),
					interpreter.NewUnmeteredInt8Value(math.MaxInt8),
				},
			},
		},
		sema.Int16Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt16Value(math.MaxInt16),
					interpreter.NewUnmeteredInt16Value(2),
					interpreter.NewUnmeteredInt16Value(math.MaxInt16),
				},
				underflow: testCall{
					interpreter.NewUnmeteredInt16Value(math.MinInt16),
					interpreter.NewUnmeteredInt16Value(-2),
					interpreter.NewUnmeteredInt16Value(math.MinInt16),
				},
			},
			subtract: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt16Value(math.MaxInt16),
					interpreter.NewUnmeteredInt16Value(-2),
					interpreter.NewUnmeteredInt16Value(math.MaxInt16),
				},
				underflow: testCall{
					interpreter.NewUnmeteredInt16Value(math.MinInt16),
					interpreter.NewUnmeteredInt16Value(2),
					interpreter.NewUnmeteredInt16Value(math.MinInt16),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt16Value(math.MaxInt16),
					interpreter.NewUnmeteredInt16Value(2),
					interpreter.NewUnmeteredInt16Value(math.MaxInt16),
				},
				underflow: testCall{
					interpreter.NewUnmeteredInt16Value(math.MinInt16),
					interpreter.NewUnmeteredInt16Value(2),
					interpreter.NewUnmeteredInt16Value(math.MinInt16),
				},
			},
			divide: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt16Value(math.MinInt16),
					interpreter.NewUnmeteredInt16Value(-1),
					interpreter.NewUnmeteredInt16Value(math.MaxInt16),
				},
			},
		},
		sema.Int32Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt32Value(math.MaxInt32),
					interpreter.NewUnmeteredInt32Value(2),
					interpreter.NewUnmeteredInt32Value(math.MaxInt32),
				},
				underflow: testCall{
					interpreter.NewUnmeteredInt32Value(math.MinInt32),
					interpreter.NewUnmeteredInt32Value(-2),
					interpreter.NewUnmeteredInt32Value(math.MinInt32),
				},
			},
			subtract: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt32Value(math.MaxInt32),
					interpreter.NewUnmeteredInt32Value(-2),
					interpreter.NewUnmeteredInt32Value(math.MaxInt32),
				},
				underflow: testCall{
					interpreter.NewUnmeteredInt32Value(math.MinInt32),
					interpreter.NewUnmeteredInt32Value(2),
					interpreter.NewUnmeteredInt32Value(math.MinInt32),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt32Value(math.MaxInt32),
					interpreter.NewUnmeteredInt32Value(2),
					interpreter.NewUnmeteredInt32Value(math.MaxInt32),
				},
				underflow: testCall{
					interpreter.NewUnmeteredInt32Value(math.MinInt32),
					interpreter.NewUnmeteredInt32Value(2),
					interpreter.NewUnmeteredInt32Value(math.MinInt32),
				},
			},
			divide: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt32Value(math.MinInt32),
					interpreter.NewUnmeteredInt32Value(-1),
					interpreter.NewUnmeteredInt32Value(math.MaxInt32),
				},
			},
		},
		sema.Int64Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt64Value(math.MaxInt64),
					interpreter.NewUnmeteredInt64Value(2),
					interpreter.NewUnmeteredInt64Value(math.MaxInt64),
				},
				underflow: testCall{
					interpreter.NewUnmeteredInt64Value(math.MinInt64),
					interpreter.NewUnmeteredInt64Value(-2),
					interpreter.NewUnmeteredInt64Value(math.MinInt64),
				},
			},
			subtract: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt64Value(math.MaxInt64),
					interpreter.NewUnmeteredInt64Value(-2),
					interpreter.NewUnmeteredInt64Value(math.MaxInt64),
				},
				underflow: testCall{
					interpreter.NewUnmeteredInt64Value(math.MinInt64),
					interpreter.NewUnmeteredInt64Value(2),
					interpreter.NewUnmeteredInt64Value(math.MinInt64),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt64Value(math.MaxInt64),
					interpreter.NewUnmeteredInt64Value(2),
					interpreter.NewUnmeteredInt64Value(math.MaxInt64),
				},
				underflow: testCall{
					interpreter.NewUnmeteredInt64Value(math.MinInt64),
					interpreter.NewUnmeteredInt64Value(2),
					interpreter.NewUnmeteredInt64Value(math.MinInt64),
				},
			},
			divide: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt64Value(math.MinInt64),
					interpreter.NewUnmeteredInt64Value(-1),
					interpreter.NewUnmeteredInt64Value(math.MaxInt64),
				},
			},
		},
		sema.Int128Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
					interpreter.NewUnmeteredInt128ValueFromInt64(2),
					interpreter.NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
				},
				underflow: testCall{
					interpreter.NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
					interpreter.NewUnmeteredInt128ValueFromInt64(-2),
					interpreter.NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
				},
			},
			subtract: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
					interpreter.NewUnmeteredInt128ValueFromInt64(-2),
					interpreter.NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
				},
				underflow: testCall{
					interpreter.NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
					interpreter.NewUnmeteredInt128ValueFromInt64(2),
					interpreter.NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
					interpreter.NewUnmeteredInt128ValueFromInt64(2),
					interpreter.NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
				},
				underflow: testCall{
					interpreter.NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
					interpreter.NewUnmeteredInt128ValueFromInt64(2),
					interpreter.NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
				},
			},
			divide: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
					interpreter.NewUnmeteredInt128ValueFromInt64(-1),
					interpreter.NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
				},
			},
		},
		sema.Int256Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
					interpreter.NewUnmeteredInt256ValueFromInt64(2),
					interpreter.NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
				},
				underflow: testCall{
					interpreter.NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
					interpreter.NewUnmeteredInt256ValueFromInt64(-2),
					interpreter.NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
				},
			},
			subtract: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
					interpreter.NewUnmeteredInt256ValueFromInt64(-2),
					interpreter.NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
				},
				underflow: testCall{
					interpreter.NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
					interpreter.NewUnmeteredInt256ValueFromInt64(2),
					interpreter.NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
					interpreter.NewUnmeteredInt256ValueFromInt64(2),
					interpreter.NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
				},
				underflow: testCall{
					interpreter.NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
					interpreter.NewUnmeteredInt256ValueFromInt64(2),
					interpreter.NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
				},
			},
			divide: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
					interpreter.NewUnmeteredInt256ValueFromInt64(-1),
					interpreter.NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
				},
			},
		},
		sema.Fix64Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredFix64Value(math.MaxInt64),
					interpreter.NewUnmeteredFix64ValueWithInteger(2, interpreter.EmptyLocationRange),
					interpreter.NewUnmeteredFix64Value(math.MaxInt64),
				},
				underflow: testCall{
					interpreter.NewUnmeteredFix64Value(math.MinInt64),
					interpreter.NewUnmeteredFix64ValueWithInteger(-2, interpreter.EmptyLocationRange),
					interpreter.NewUnmeteredFix64Value(math.MinInt64),
				},
			},
			subtract: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredFix64Value(math.MaxInt64),
					interpreter.NewUnmeteredFix64ValueWithInteger(-2, interpreter.EmptyLocationRange),
					interpreter.NewUnmeteredFix64Value(math.MaxInt64),
				},
				underflow: testCall{
					interpreter.NewUnmeteredFix64Value(math.MinInt64),
					interpreter.NewUnmeteredFix64ValueWithInteger(2, interpreter.EmptyLocationRange),
					interpreter.NewUnmeteredFix64Value(math.MinInt64),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredFix64Value(math.MaxInt64),
					interpreter.NewUnmeteredFix64ValueWithInteger(2, interpreter.EmptyLocationRange),
					interpreter.NewUnmeteredFix64Value(math.MaxInt64),
				},
				underflow: testCall{
					interpreter.NewUnmeteredFix64Value(math.MinInt64),
					interpreter.NewUnmeteredFix64ValueWithInteger(2, interpreter.EmptyLocationRange),
					interpreter.NewUnmeteredFix64Value(math.MinInt64),
				},
			},
			divide: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredFix64Value(math.MinInt64),
					interpreter.NewUnmeteredFix64ValueWithInteger(-1, interpreter.EmptyLocationRange),
					interpreter.NewUnmeteredFix64Value(math.MaxInt64),
				},
			},
		},
		sema.UIntType: {
			subtract: testCalls{
				underflow: testCall{
					interpreter.NewUnmeteredUIntValueFromBigInt(sema.UIntTypeMin),
					interpreter.NewUnmeteredUIntValueFromUint64(2),
					interpreter.NewUnmeteredUIntValueFromBigInt(sema.UIntTypeMin),
				},
			},
		},
		sema.UInt8Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredUInt8Value(math.MaxUint8),
					interpreter.NewUnmeteredUInt8Value(2),
					interpreter.NewUnmeteredUInt8Value(math.MaxUint8),
				},
			},
			subtract: testCalls{
				underflow: testCall{
					interpreter.NewUnmeteredUInt8Value(0),
					interpreter.NewUnmeteredUInt8Value(2),
					interpreter.NewUnmeteredUInt8Value(0),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredUInt8Value(math.MaxUint8),
					interpreter.NewUnmeteredUInt8Value(2),
					interpreter.NewUnmeteredUInt8Value(math.MaxUint8),
				},
			},
		},
		sema.UInt16Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredUInt16Value(math.MaxUint16),
					interpreter.NewUnmeteredUInt16Value(2),
					interpreter.NewUnmeteredUInt16Value(math.MaxUint16),
				},
			},
			subtract: testCalls{
				underflow: testCall{
					interpreter.NewUnmeteredUInt16Value(0),
					interpreter.NewUnmeteredUInt16Value(2),
					interpreter.NewUnmeteredUInt16Value(0),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredUInt16Value(math.MaxUint16),
					interpreter.NewUnmeteredUInt16Value(2),
					interpreter.NewUnmeteredUInt16Value(math.MaxUint16),
				},
			},
		},
		sema.UInt32Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredUInt32Value(math.MaxUint32),
					interpreter.NewUnmeteredUInt32Value(2),
					interpreter.NewUnmeteredUInt32Value(math.MaxUint32),
				},
			},
			subtract: testCalls{
				underflow: testCall{
					interpreter.NewUnmeteredUInt32Value(0),
					interpreter.NewUnmeteredUInt32Value(2),
					interpreter.NewUnmeteredUInt32Value(0),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredUInt32Value(math.MaxUint32),
					interpreter.NewUnmeteredUInt32Value(2),
					interpreter.NewUnmeteredUInt32Value(math.MaxUint32),
				},
			},
		},
		sema.UInt64Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredUInt64Value(math.MaxUint64),
					interpreter.NewUnmeteredUInt64Value(2),
					interpreter.NewUnmeteredUInt64Value(math.MaxUint64),
				},
			},
			subtract: testCalls{
				underflow: testCall{
					interpreter.NewUnmeteredUInt64Value(0),
					interpreter.NewUnmeteredUInt64Value(2),
					interpreter.NewUnmeteredUInt64Value(0),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredUInt64Value(math.MaxUint64),
					interpreter.NewUnmeteredUInt64Value(2),
					interpreter.NewUnmeteredUInt64Value(math.MaxUint64),
				},
			},
		},
		sema.UInt128Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
					interpreter.NewUnmeteredUInt128ValueFromUint64(2),
					interpreter.NewUnmeteredUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
				},
			},
			subtract: testCalls{
				underflow: testCall{
					interpreter.NewUnmeteredUInt128ValueFromBigInt(sema.UInt128TypeMinIntBig),
					interpreter.NewUnmeteredUInt128ValueFromUint64(2),
					interpreter.NewUnmeteredUInt128ValueFromBigInt(sema.UInt128TypeMinIntBig),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
					interpreter.NewUnmeteredUInt128ValueFromUint64(2),
					interpreter.NewUnmeteredUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
				},
			},
		},
		sema.UInt256Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredUInt256ValueFromBigInt(sema.UInt256TypeMaxIntBig),
					interpreter.NewUnmeteredUInt256ValueFromUint64(2),
					interpreter.NewUnmeteredUInt256ValueFromBigInt(sema.UInt256TypeMaxIntBig),
				},
			},
			subtract: testCalls{
				underflow: testCall{
					interpreter.NewUnmeteredUInt256ValueFromBigInt(sema.UInt256TypeMinIntBig),
					interpreter.NewUnmeteredUInt256ValueFromUint64(2),
					interpreter.NewUnmeteredUInt256ValueFromBigInt(sema.UInt256TypeMinIntBig),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredUInt256ValueFromBigInt(sema.UInt256TypeMaxIntBig),
					interpreter.NewUnmeteredUInt256ValueFromUint64(2),
					interpreter.NewUnmeteredUInt256ValueFromBigInt(sema.UInt256TypeMaxIntBig),
				},
			},
		},
		sema.UFix64Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredUFix64Value(math.MaxUint64),
					interpreter.NewUnmeteredUFix64ValueWithInteger(2, interpreter.EmptyLocationRange),
					interpreter.NewUnmeteredUFix64Value(math.MaxUint64),
				},
			},
			subtract: testCalls{
				underflow: testCall{
					interpreter.NewUnmeteredUFix64Value(0),
					interpreter.NewUnmeteredUFix64ValueWithInteger(2, interpreter.EmptyLocationRange),
					interpreter.NewUnmeteredUFix64Value(0),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.NewUnmeteredUFix64Value(math.MaxUint64),
					interpreter.NewUnmeteredUFix64ValueWithInteger(2, interpreter.EmptyLocationRange),
					interpreter.NewUnmeteredUFix64Value(math.MaxUint64),
				},
			},
		},
	}

	// Verify all test cases exist

	for _, ty := range common.Concat(
		sema.AllSignedIntegerTypes,
		sema.AllSignedFixedPointTypes,
	) {

		testCase, ok := testCases[ty]

		if ty == sema.IntType {
			require.False(t, ok, "invalid test case for %s", ty)
		} else {
			require.True(t, ok, "missing test case for %s", ty)

			require.NotNil(t, testCase.add.overflow.expected)
			require.NotNil(t, testCase.add.underflow.expected)

			require.NotNil(t, testCase.subtract.overflow.expected)
			require.NotNil(t, testCase.subtract.underflow.expected)

			require.NotNil(t, testCase.multiply.overflow.expected)
			require.NotNil(t, testCase.multiply.underflow.expected)

			require.NotNil(t, testCase.divide.overflow.expected)
			require.Nil(t, testCase.divide.underflow.expected)
		}
	}

	for _, ty := range common.Concat(
		sema.AllUnsignedIntegerTypes,
		sema.AllUnsignedFixedPointTypes,
	) {

		if strings.HasPrefix(ty.String(), "Word") {
			continue
		}

		testCase, ok := testCases[ty]
		require.True(t, ok, "missing test case for %s", ty)

		if ty == sema.UIntType {

			require.Nil(t, testCase.add.overflow.expected)
			require.Nil(t, testCase.add.underflow.expected)

			require.Nil(t, testCase.subtract.overflow.expected)
			require.NotNil(t, testCase.subtract.underflow.expected)

			require.Nil(t, testCase.multiply.overflow.expected)
			require.Nil(t, testCase.multiply.underflow.expected)

			require.Nil(t, testCase.divide.overflow.expected)
			require.Nil(t, testCase.divide.underflow.expected)
		} else {
			require.NotNil(t, testCase.add.overflow.expected)
			require.Nil(t, testCase.add.underflow.expected)

			require.Nil(t, testCase.subtract.overflow.expected)
			require.NotNil(t, testCase.subtract.underflow.expected)

			require.NotNil(t, testCase.multiply.overflow.expected)
			require.Nil(t, testCase.multiply.underflow.expected)

			require.Nil(t, testCase.divide.overflow.expected)
			require.Nil(t, testCase.divide.underflow.expected)
		}
	}

	test := func(ty sema.Type, method string, calls testCalls) {

		method = fmt.Sprintf("saturating%s", method)

		for kind, call := range map[string]testCall{
			"overflow":  calls.overflow,
			"underflow": calls.underflow,
		} {

			if call.expected == nil {
				continue
			}

			t.Run(fmt.Sprintf("%s %s %s", ty, method, kind), func(t *testing.T) {

				inter := parseCheckAndInterpret(t,
					fmt.Sprintf(
						`
                          fun test(a: %[1]s, b: %[1]s): %[1]s {
                              return a.%[2]s(b)
                          }
                        `,
						ty,
						method,
					),
				)

				result, err := inter.Invoke("test", call.left, call.right)
				require.NoError(t, err)

				require.True(t,
					call.expected.Equal(inter, interpreter.EmptyLocationRange, result),
					fmt.Sprintf(
						"%s(%s, %s) = %s != %s",
						method, call.left, call.right, result, call.expected,
					),
				)
			})
		}
	}

	for ty, testCase := range testCases {
		test(ty, "Add", testCase.add)
		test(ty, "Subtract", testCase.subtract)
		test(ty, "Multiply", testCase.multiply)
		test(ty, "Divide", testCase.divide)
	}
}
