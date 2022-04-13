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
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

var integerTestValues = map[string]interpreter.NumberValue{
	// Int*
	"Int":    interpreter.NewIntValueFromInt64(60),
	"Int8":   interpreter.Int8Value(60),
	"Int16":  interpreter.Int16Value(60),
	"Int32":  interpreter.Int32Value(60),
	"Int64":  interpreter.Int64Value(60),
	"Int128": interpreter.NewInt128ValueFromInt64(60),
	"Int256": interpreter.NewInt256ValueFromInt64(60),
	// UInt*
	"UInt":    interpreter.NewUIntValueFromUint64(60),
	"UInt8":   interpreter.UInt8Value(60),
	"UInt16":  interpreter.UInt16Value(60),
	"UInt32":  interpreter.UInt32Value(60),
	"UInt64":  interpreter.UInt64Value(60),
	"UInt128": interpreter.NewUInt128ValueFromUint64(60),
	"UInt256": interpreter.NewUInt256ValueFromUint64(60),
	// Word*
	"Word8":  interpreter.Word8Value(60),
	"Word16": interpreter.Word16Value(60),
	"Word32": interpreter.Word32Value(60),
	"Word64": interpreter.Word64Value(60),
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
				inter.Globals["c"].GetValue(),
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
				inter.Globals["c"].GetValue(),
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
				inter.Globals["c"].GetValue(),
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
				inter.Globals["c"].GetValue(),
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
				inter.Globals["c"].GetValue(),
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
					interpreter.Int8Value(math.MaxInt8),
					interpreter.Int8Value(2),
					interpreter.Int8Value(math.MaxInt8),
				},
				underflow: testCall{
					interpreter.Int8Value(math.MinInt8),
					interpreter.Int8Value(-2),
					interpreter.Int8Value(math.MinInt8),
				},
			},
			subtract: testCalls{
				overflow: testCall{
					interpreter.Int8Value(math.MaxInt8),
					interpreter.Int8Value(-2),
					interpreter.Int8Value(math.MaxInt8),
				},
				underflow: testCall{
					interpreter.Int8Value(math.MinInt8),
					interpreter.Int8Value(2),
					interpreter.Int8Value(math.MinInt8),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.Int8Value(math.MaxInt8),
					interpreter.Int8Value(2),
					interpreter.Int8Value(math.MaxInt8),
				},
				underflow: testCall{
					interpreter.Int8Value(math.MinInt8),
					interpreter.Int8Value(2),
					interpreter.Int8Value(math.MinInt8),
				},
			},
			divide: testCalls{
				overflow: testCall{
					interpreter.Int8Value(math.MinInt8),
					interpreter.Int8Value(-1),
					interpreter.Int8Value(math.MaxInt8),
				},
			},
		},
		sema.Int16Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.Int16Value(math.MaxInt16),
					interpreter.Int16Value(2),
					interpreter.Int16Value(math.MaxInt16),
				},
				underflow: testCall{
					interpreter.Int16Value(math.MinInt16),
					interpreter.Int16Value(-2),
					interpreter.Int16Value(math.MinInt16),
				},
			},
			subtract: testCalls{
				overflow: testCall{
					interpreter.Int16Value(math.MaxInt16),
					interpreter.Int16Value(-2),
					interpreter.Int16Value(math.MaxInt16),
				},
				underflow: testCall{
					interpreter.Int16Value(math.MinInt16),
					interpreter.Int16Value(2),
					interpreter.Int16Value(math.MinInt16),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.Int16Value(math.MaxInt16),
					interpreter.Int16Value(2),
					interpreter.Int16Value(math.MaxInt16),
				},
				underflow: testCall{
					interpreter.Int16Value(math.MinInt16),
					interpreter.Int16Value(2),
					interpreter.Int16Value(math.MinInt16),
				},
			},
			divide: testCalls{
				overflow: testCall{
					interpreter.Int16Value(math.MinInt16),
					interpreter.Int16Value(-1),
					interpreter.Int16Value(math.MaxInt16),
				},
			},
		},
		sema.Int32Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.Int32Value(math.MaxInt32),
					interpreter.Int32Value(2),
					interpreter.Int32Value(math.MaxInt32),
				},
				underflow: testCall{
					interpreter.Int32Value(math.MinInt32),
					interpreter.Int32Value(-2),
					interpreter.Int32Value(math.MinInt32),
				},
			},
			subtract: testCalls{
				overflow: testCall{
					interpreter.Int32Value(math.MaxInt32),
					interpreter.Int32Value(-2),
					interpreter.Int32Value(math.MaxInt32),
				},
				underflow: testCall{
					interpreter.Int32Value(math.MinInt32),
					interpreter.Int32Value(2),
					interpreter.Int32Value(math.MinInt32),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.Int32Value(math.MaxInt32),
					interpreter.Int32Value(2),
					interpreter.Int32Value(math.MaxInt32),
				},
				underflow: testCall{
					interpreter.Int32Value(math.MinInt32),
					interpreter.Int32Value(2),
					interpreter.Int32Value(math.MinInt32),
				},
			},
			divide: testCalls{
				overflow: testCall{
					interpreter.Int32Value(math.MinInt32),
					interpreter.Int32Value(-1),
					interpreter.Int32Value(math.MaxInt32),
				},
			},
		},
		sema.Int64Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.Int64Value(math.MaxInt64),
					interpreter.Int64Value(2),
					interpreter.Int64Value(math.MaxInt64),
				},
				underflow: testCall{
					interpreter.Int64Value(math.MinInt64),
					interpreter.Int64Value(-2),
					interpreter.Int64Value(math.MinInt64),
				},
			},
			subtract: testCalls{
				overflow: testCall{
					interpreter.Int64Value(math.MaxInt64),
					interpreter.Int64Value(-2),
					interpreter.Int64Value(math.MaxInt64),
				},
				underflow: testCall{
					interpreter.Int64Value(math.MinInt64),
					interpreter.Int64Value(2),
					interpreter.Int64Value(math.MinInt64),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.Int64Value(math.MaxInt64),
					interpreter.Int64Value(2),
					interpreter.Int64Value(math.MaxInt64),
				},
				underflow: testCall{
					interpreter.Int64Value(math.MinInt64),
					interpreter.Int64Value(2),
					interpreter.Int64Value(math.MinInt64),
				},
			},
			divide: testCalls{
				overflow: testCall{
					interpreter.Int64Value(math.MinInt64),
					interpreter.Int64Value(-1),
					interpreter.Int64Value(math.MaxInt64),
				},
			},
		},
		sema.Int128Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.NewInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
					interpreter.NewInt128ValueFromInt64(2),
					interpreter.NewInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
				},
				underflow: testCall{
					interpreter.NewInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
					interpreter.NewInt128ValueFromInt64(-2),
					interpreter.NewInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
				},
			},
			subtract: testCalls{
				overflow: testCall{
					interpreter.NewInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
					interpreter.NewInt128ValueFromInt64(-2),
					interpreter.NewInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
				},
				underflow: testCall{
					interpreter.NewInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
					interpreter.NewInt128ValueFromInt64(2),
					interpreter.NewInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.NewInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
					interpreter.NewInt128ValueFromInt64(2),
					interpreter.NewInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
				},
				underflow: testCall{
					interpreter.NewInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
					interpreter.NewInt128ValueFromInt64(2),
					interpreter.NewInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
				},
			},
			divide: testCalls{
				overflow: testCall{
					interpreter.NewInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
					interpreter.NewInt128ValueFromInt64(-1),
					interpreter.NewInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
				},
			},
		},
		sema.Int256Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.NewInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
					interpreter.NewInt256ValueFromInt64(2),
					interpreter.NewInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
				},
				underflow: testCall{
					interpreter.NewInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
					interpreter.NewInt256ValueFromInt64(-2),
					interpreter.NewInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
				},
			},
			subtract: testCalls{
				overflow: testCall{
					interpreter.NewInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
					interpreter.NewInt256ValueFromInt64(-2),
					interpreter.NewInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
				},
				underflow: testCall{
					interpreter.NewInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
					interpreter.NewInt256ValueFromInt64(2),
					interpreter.NewInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.NewInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
					interpreter.NewInt256ValueFromInt64(2),
					interpreter.NewInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
				},
				underflow: testCall{
					interpreter.NewInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
					interpreter.NewInt256ValueFromInt64(2),
					interpreter.NewInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
				},
			},
			divide: testCalls{
				overflow: testCall{
					interpreter.NewInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
					interpreter.NewInt256ValueFromInt64(-1),
					interpreter.NewInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
				},
			},
		},
		sema.Fix64Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.Fix64Value(math.MaxInt64),
					interpreter.NewFix64ValueWithInteger(2),
					interpreter.Fix64Value(math.MaxInt64),
				},
				underflow: testCall{
					interpreter.Fix64Value(math.MinInt64),
					interpreter.NewFix64ValueWithInteger(-2),
					interpreter.Fix64Value(math.MinInt64),
				},
			},
			subtract: testCalls{
				overflow: testCall{
					interpreter.Fix64Value(math.MaxInt64),
					interpreter.NewFix64ValueWithInteger(-2),
					interpreter.Fix64Value(math.MaxInt64),
				},
				underflow: testCall{
					interpreter.Fix64Value(math.MinInt64),
					interpreter.NewFix64ValueWithInteger(2),
					interpreter.Fix64Value(math.MinInt64),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.Fix64Value(math.MaxInt64),
					interpreter.NewFix64ValueWithInteger(2),
					interpreter.Fix64Value(math.MaxInt64),
				},
				underflow: testCall{
					interpreter.Fix64Value(math.MinInt64),
					interpreter.NewFix64ValueWithInteger(2),
					interpreter.Fix64Value(math.MinInt64),
				},
			},
			divide: testCalls{
				overflow: testCall{
					interpreter.Fix64Value(math.MinInt64),
					interpreter.NewFix64ValueWithInteger(-1),
					interpreter.Fix64Value(math.MaxInt64),
				},
			},
		},
		sema.UIntType: {
			subtract: testCalls{
				underflow: testCall{
					interpreter.NewUIntValueFromBigInt(sema.UIntTypeMin),
					interpreter.NewUIntValueFromUint64(2),
					interpreter.NewUIntValueFromBigInt(sema.UIntTypeMin),
				},
			},
		},
		sema.UInt8Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.UInt8Value(math.MaxUint8),
					interpreter.UInt8Value(2),
					interpreter.UInt8Value(math.MaxUint8),
				},
			},
			subtract: testCalls{
				underflow: testCall{
					interpreter.UInt8Value(0),
					interpreter.UInt8Value(2),
					interpreter.UInt8Value(0),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.UInt8Value(math.MaxUint8),
					interpreter.UInt8Value(2),
					interpreter.UInt8Value(math.MaxUint8),
				},
			},
		},
		sema.UInt16Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.UInt16Value(math.MaxUint16),
					interpreter.UInt16Value(2),
					interpreter.UInt16Value(math.MaxUint16),
				},
			},
			subtract: testCalls{
				underflow: testCall{
					interpreter.UInt16Value(0),
					interpreter.UInt16Value(2),
					interpreter.UInt16Value(0),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.UInt16Value(math.MaxUint16),
					interpreter.UInt16Value(2),
					interpreter.UInt16Value(math.MaxUint16),
				},
			},
		},
		sema.UInt32Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.UInt32Value(math.MaxUint32),
					interpreter.UInt32Value(2),
					interpreter.UInt32Value(math.MaxUint32),
				},
			},
			subtract: testCalls{
				underflow: testCall{
					interpreter.UInt32Value(0),
					interpreter.UInt32Value(2),
					interpreter.UInt32Value(0),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.UInt32Value(math.MaxUint32),
					interpreter.UInt32Value(2),
					interpreter.UInt32Value(math.MaxUint32),
				},
			},
		},
		sema.UInt64Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.UInt64Value(math.MaxUint64),
					interpreter.UInt64Value(2),
					interpreter.UInt64Value(math.MaxUint64),
				},
			},
			subtract: testCalls{
				underflow: testCall{
					interpreter.UInt64Value(0),
					interpreter.UInt64Value(2),
					interpreter.UInt64Value(0),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.UInt64Value(math.MaxUint64),
					interpreter.UInt64Value(2),
					interpreter.UInt64Value(math.MaxUint64),
				},
			},
		},
		sema.UInt128Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.NewUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
					interpreter.NewUInt128ValueFromUint64(2),
					interpreter.NewUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
				},
			},
			subtract: testCalls{
				underflow: testCall{
					interpreter.NewUInt128ValueFromBigInt(sema.UInt128TypeMinIntBig),
					interpreter.NewUInt128ValueFromUint64(2),
					interpreter.NewUInt128ValueFromBigInt(sema.UInt128TypeMinIntBig),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.NewUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
					interpreter.NewUInt128ValueFromUint64(2),
					interpreter.NewUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
				},
			},
		},
		sema.UInt256Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.NewUInt256ValueFromBigInt(sema.UInt256TypeMaxIntBig),
					interpreter.NewUInt256ValueFromUint64(2),
					interpreter.NewUInt256ValueFromBigInt(sema.UInt256TypeMaxIntBig),
				},
			},
			subtract: testCalls{
				underflow: testCall{
					interpreter.NewUInt256ValueFromBigInt(sema.UInt256TypeMinIntBig),
					interpreter.NewUInt256ValueFromUint64(2),
					interpreter.NewUInt256ValueFromBigInt(sema.UInt256TypeMinIntBig),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.NewUInt256ValueFromBigInt(sema.UInt256TypeMaxIntBig),
					interpreter.NewUInt256ValueFromUint64(2),
					interpreter.NewUInt256ValueFromBigInt(sema.UInt256TypeMaxIntBig),
				},
			},
		},
		sema.UFix64Type: {
			add: testCalls{
				overflow: testCall{
					interpreter.UFix64Value(math.MaxUint64),
					interpreter.NewUFix64ValueWithInteger(2),
					interpreter.UFix64Value(math.MaxUint64),
				},
			},
			subtract: testCalls{
				underflow: testCall{
					interpreter.UFix64Value(0),
					interpreter.NewUFix64ValueWithInteger(2),
					interpreter.UFix64Value(0),
				},
			},
			multiply: testCalls{
				overflow: testCall{
					interpreter.UFix64Value(math.MaxUint64),
					interpreter.NewUFix64ValueWithInteger(2),
					interpreter.UFix64Value(math.MaxUint64),
				},
			},
		},
	}

	// Verify all test cases exist

	for _, ty := range append(
		sema.AllSignedIntegerTypes[:],
		sema.AllSignedFixedPointTypes...,
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

	for _, ty := range append(
		sema.AllUnsignedIntegerTypes[:],
		sema.AllUnsignedFixedPointTypes...,
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
					call.expected.Equal(inter, interpreter.ReturnEmptyLocationRange, result),
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
