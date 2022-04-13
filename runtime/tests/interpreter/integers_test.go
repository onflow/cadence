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
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/onflow/cadence/runtime/tests/utils"

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

var testIntegerTypesAndValues = map[string]interpreter.Value{
	// Int*
	"Int":    interpreter.NewIntValueFromInt64(50),
	"Int8":   interpreter.Int8Value(50),
	"Int16":  interpreter.Int16Value(50),
	"Int32":  interpreter.Int32Value(50),
	"Int64":  interpreter.Int64Value(50),
	"Int128": interpreter.NewInt128ValueFromInt64(50),
	"Int256": interpreter.NewInt256ValueFromInt64(50),
	// UInt*
	"UInt":    interpreter.NewUIntValueFromUint64(50),
	"UInt8":   interpreter.UInt8Value(50),
	"UInt16":  interpreter.UInt16Value(50),
	"UInt32":  interpreter.UInt32Value(50),
	"UInt64":  interpreter.UInt64Value(50),
	"UInt128": interpreter.NewUInt128ValueFromUint64(50),
	"UInt256": interpreter.NewUInt256ValueFromUint64(50),
	// Word*
	"Word8":  interpreter.Word8Value(50),
	"Word16": interpreter.Word16Value(50),
	"Word32": interpreter.Word32Value(50),
	"Word64": interpreter.Word64Value(50),
}

func init() {
	for _, integerType := range sema.AllIntegerTypes {
		// Only test leaf types
		switch integerType {
		case sema.IntegerType, sema.SignedIntegerType:
			continue
		}

		if _, ok := testIntegerTypesAndValues[integerType.String()]; !ok {
			panic(fmt.Sprintf("broken test: missing %s", integerType))
		}
	}
}

func TestInterpretIntegerConversions(t *testing.T) {

	t.Parallel()

	for integerType, value := range testIntegerTypesAndValues {

		t.Run(integerType, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      let x: %[1]s = 50
                      let y = %[1]s(40) + %[1]s(10)
                      let z = y == x
                    `,
					integerType,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				value,
				inter.Globals["x"].GetValue(),
			)

			AssertValuesEqual(
				t,
				inter,
				value,
				inter.Globals["y"].GetValue(),
			)

			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(true),
				inter.Globals["z"].GetValue(),
			)

		})
	}
}

func TestInterpretAddressConversion(t *testing.T) {

	t.Parallel()

	t.Run("implicit through variable declaration", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let x: Address = 0x1
        `)

		AssertValuesEqual(
			t,
			inter,
			interpreter.AddressValue{
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
			},
			inter.Globals["x"].GetValue(),
		)

	})

	t.Run("conversion function", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let x = Address(0x2)
        `)

		AssertValuesEqual(
			t,
			inter,
			interpreter.AddressValue{
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2,
			},
			inter.Globals["x"].GetValue(),
		)
	})

	t.Run("conversion function, Int, overflow", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          fun test() {
              let y = 0x1111111111111111111
              let x = Address(y)
          }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		require.ErrorAs(t, err, &interpreter.OverflowError{})
	})

	t.Run("conversion function, underflow", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          fun test() {
              let y = -0x1
              let x = Address(y)
          }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		require.ErrorAs(t, err, &interpreter.UnderflowError{})
	})
}

func TestInterpretIntegerLiteralTypeConversionInVariableDeclaration(t *testing.T) {

	t.Parallel()

	for integerType, value := range testIntegerTypesAndValues {

		t.Run(integerType, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      let x: %s = 50
                    `,
					integerType,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				value,
				inter.Globals["x"].GetValue(),
			)

		})
	}
}

func TestInterpretIntegerLiteralTypeConversionInVariableDeclarationOptional(t *testing.T) {

	t.Parallel()

	for integerType, value := range testIntegerTypesAndValues {

		t.Run(integerType, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                        let x: %s? = 50
                    `,
					integerType,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewSomeValueNonCopying(value),
				inter.Globals["x"].GetValue(),
			)
		})
	}
}

func TestInterpretIntegerLiteralTypeConversionInAssignment(t *testing.T) {

	t.Parallel()

	for integerType, value := range testIntegerTypesAndValues {

		t.Run(integerType, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      var x: %s = 50
                      fun test() {
                          x = x + x
                      }
                    `,
					integerType,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				value,
				inter.Globals["x"].GetValue(),
			)

			_, err := inter.Invoke("test")
			require.NoError(t, err)

			numberValue := value.(interpreter.NumberValue)
			AssertValuesEqual(
				t,
				inter,
				numberValue.Plus(numberValue),
				inter.Globals["x"].GetValue(),
			)
		})
	}
}

func TestInterpretIntegerLiteralTypeConversionInAssignmentOptional(t *testing.T) {

	t.Parallel()

	for integerType, value := range testIntegerTypesAndValues {

		t.Run(integerType, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      var x: %s? = 50
                      fun test() {
                          x = 100
                      }
                    `,
					integerType,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewSomeValueNonCopying(value),
				inter.Globals["x"].GetValue(),
			)

			_, err := inter.Invoke("test")
			require.NoError(t, err)

			numberValue := value.(interpreter.NumberValue)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewSomeValueNonCopying(
					numberValue.Plus(numberValue),
				),
				inter.Globals["x"].GetValue(),
			)
		})
	}
}

func TestInterpretIntegerLiteralTypeConversionInFunctionCallArgument(t *testing.T) {

	t.Parallel()

	for integerType, value := range testIntegerTypesAndValues {

		t.Run(integerType, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      fun test(_ x: %[1]s): %[1]s {
                          return x
                      }
                      let x = test(50)
                    `,
					integerType,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				value,
				inter.Globals["x"].GetValue(),
			)
		})
	}
}

func TestInterpretIntegerLiteralTypeConversionInFunctionCallArgumentOptional(t *testing.T) {

	t.Parallel()

	for integerType, value := range testIntegerTypesAndValues {

		t.Run(integerType, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                        fun test(_ x: %[1]s?): %[1]s? {
                            return x
                        }
                        let x = test(50)
                    `,
					integerType,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewSomeValueNonCopying(value),
				inter.Globals["x"].GetValue(),
			)
		})
	}
}

func TestInterpretIntegerLiteralTypeConversionInReturn(t *testing.T) {

	t.Parallel()

	for integerType, value := range testIntegerTypesAndValues {

		t.Run(integerType, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      fun test(): %s {
                          return 50
                      }
                    `,
					integerType,
				),
			)

			result, err := inter.Invoke("test")
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				value,
				result,
			)
		})
	}
}

func TestInterpretIntegerLiteralTypeConversionInReturnOptional(t *testing.T) {

	t.Parallel()

	for integerType, value := range testIntegerTypesAndValues {

		t.Run(integerType, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      fun test(): %s? {
                          return 50
                      }
                    `,
					integerType,
				),
			)

			result, err := inter.Invoke("test")
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewSomeValueNonCopying(value),
				result,
			)
		})
	}
}

func TestInterpretIntegerConversion(t *testing.T) {

	t.Parallel()

	test := func(
		t *testing.T,
		sourceType sema.Type,
		targetType sema.Type,
		value interpreter.Value,
		expectedValue interpreter.Value,
		expectedError error,
	) {

		inter := parseCheckAndInterpret(t,
			fmt.Sprintf(
				`
                  fun test(value: %[1]s): %[2]s {
                      return %[2]s(value)
                  }
				`,
				sourceType,
				targetType,
			),
		)

		result, err := inter.Invoke("test", value)

		if expectedError != nil {
			require.Error(t, err)
			require.ErrorAs(t, err, &expectedError)
		} else {
			require.NoError(t, err)

			if expectedValue != nil {
				assert.Equal(t, expectedValue, result)
			} else {
				// Fall back to string comparison,
				// as it is too much work to construct the expected value
				assert.Equal(t, value.String(), result.String())
			}
		}
	}

	type values struct {
		fortyTwo interpreter.Value
		min      interpreter.Value
		max      interpreter.Value
	}

	testValues := map[*sema.NumericType]values{
		sema.IntType: {
			fortyTwo: interpreter.NewIntValueFromInt64(42),
			// Int does not actually have a minimum, but create a "large" value,
			// which can be used for testing against other types
			min: func() interpreter.Value {
				i := big.NewInt(-1)
				i.Lsh(i, 1000)
				return interpreter.NewIntValueFromBigInt(i)
			}(),
			// Int does not actually have a maximum, but create a "large" value,
			// which can be used for testing against other types
			max: func() interpreter.Value {
				i := big.NewInt(1)
				i.Lsh(i, 1000)
				return interpreter.NewIntValueFromBigInt(i)
			}(),
		},
		sema.UIntType: {
			fortyTwo: interpreter.NewUIntValueFromUint64(42),
			min:      interpreter.NewUIntValueFromUint64(0),
			// UInt does not actually have a maximum, but create a "large" value,
			// which can be used for testing against other types
			max: func() interpreter.Value {
				i := big.NewInt(1)
				i.Lsh(i, 1000)
				return interpreter.NewUIntValueFromBigInt(i)
			}(),
		},
		sema.UInt8Type: {
			fortyTwo: interpreter.UInt8Value(42),
			min:      interpreter.UInt8Value(0),
			max:      interpreter.UInt8Value(math.MaxUint8),
		},
		sema.UInt16Type: {
			fortyTwo: interpreter.UInt16Value(42),
			min:      interpreter.UInt16Value(0),
			max:      interpreter.UInt16Value(math.MaxUint16),
		},
		sema.UInt32Type: {
			fortyTwo: interpreter.UInt32Value(42),
			min:      interpreter.UInt32Value(0),
			max:      interpreter.UInt32Value(math.MaxUint32),
		},
		sema.UInt64Type: {
			fortyTwo: interpreter.UInt64Value(42),
			min:      interpreter.UInt64Value(0),
			max:      interpreter.UInt64Value(math.MaxUint64),
		},
		sema.UInt128Type: {
			fortyTwo: interpreter.NewUInt128ValueFromUint64(42),
			min:      interpreter.NewUInt128ValueFromUint64(0),
			max:      interpreter.NewUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
		},
		sema.UInt256Type: {
			fortyTwo: interpreter.NewUInt256ValueFromUint64(42),
			min:      interpreter.NewUInt256ValueFromUint64(0),
			max:      interpreter.NewUInt256ValueFromBigInt(sema.UInt256TypeMaxIntBig),
		},
		sema.Word8Type: {
			fortyTwo: interpreter.Word8Value(42),
			min:      interpreter.Word8Value(0),
			max:      interpreter.Word8Value(math.MaxUint8),
		},
		sema.Word16Type: {
			fortyTwo: interpreter.Word16Value(42),
			min:      interpreter.Word16Value(0),
			max:      interpreter.Word16Value(math.MaxUint16),
		},
		sema.Word32Type: {
			fortyTwo: interpreter.Word32Value(42),
			min:      interpreter.Word32Value(0),
			max:      interpreter.Word32Value(math.MaxUint32),
		},
		sema.Word64Type: {
			fortyTwo: interpreter.Word64Value(42),
			min:      interpreter.Word64Value(0),
			max:      interpreter.Word64Value(math.MaxUint64),
		},
		sema.Int8Type: {
			fortyTwo: interpreter.Int8Value(42),
			min:      interpreter.Int8Value(math.MinInt8),
			max:      interpreter.Int8Value(math.MaxInt8),
		},
		sema.Int16Type: {
			fortyTwo: interpreter.Int16Value(42),
			min:      interpreter.Int16Value(math.MinInt16),
			max:      interpreter.Int16Value(math.MaxInt16),
		},
		sema.Int32Type: {
			fortyTwo: interpreter.Int32Value(42),
			min:      interpreter.Int32Value(math.MinInt32),
			max:      interpreter.Int32Value(math.MaxInt32),
		},
		sema.Int64Type: {
			fortyTwo: interpreter.Int64Value(42),
			min:      interpreter.Int64Value(math.MinInt64),
			max:      interpreter.Int64Value(math.MaxInt64),
		},
		sema.Int128Type: {
			fortyTwo: interpreter.NewInt128ValueFromInt64(42),
			min:      interpreter.NewInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
			max:      interpreter.NewInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
		},
		sema.Int256Type: {
			fortyTwo: interpreter.NewInt256ValueFromInt64(42),
			min:      interpreter.NewInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
			max:      interpreter.NewInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
		},
	}

	for _, ty := range sema.AllIntegerTypes {
		// Only test leaf types
		switch ty {
		case sema.IntegerType, sema.SignedIntegerType:
			continue
		}

		_, ok := testValues[ty.(*sema.NumericType)]
		require.True(t, ok, "missing expected value for type %s", ty.String())
	}

	for sourceType, sourceValues := range testValues {
		for targetType, targetValues := range testValues {

			t.Run(fmt.Sprintf("%s to %s", sourceType, targetType), func(t *testing.T) {

				// Check underflow is handled correctly

				targetMinInt := targetType.MinInt()
				sourceMinInt := sourceType.MinInt()

				if targetMinInt != nil && (sourceMinInt == nil || sourceMinInt.Cmp(targetMinInt) < 0) {
					t.Run("underflow", func(t *testing.T) {
						test(t, sourceType, targetType, sourceValues.min, nil, interpreter.UnderflowError{})
					})
				}

				// Check a "typical" value can be converted

				t.Run("valid", func(t *testing.T) {
					test(t, sourceType, targetType, sourceValues.fortyTwo, targetValues.fortyTwo, nil)
				})

				// Check overflow is handled correctly

				targetMaxInt := targetType.MaxInt()
				sourceMaxInt := sourceType.MaxInt()

				if targetMaxInt != nil && (sourceMaxInt == nil || sourceMaxInt.Cmp(targetMaxInt) > 0) {
					t.Run("overflow", func(t *testing.T) {
						test(t, sourceType, targetType, sourceValues.max, nil, interpreter.OverflowError{})
					})
				}

				// Check the maximum value can be converted.
				// For example, this tests that BigNumberValue.ToBigInt() is used / correctly implemented,
				// instead of NumberValue.ToInt(), for example in the case Int(UInt64.max)

				if sourceMaxInt != nil && (targetMaxInt == nil || sourceMaxInt.Cmp(targetMaxInt) < 0) {
					t.Run("max", func(t *testing.T) {
						test(t, sourceType, targetType, sourceValues.max, nil, nil)
					})
				}
			})
		}
	}
}

func TestInterpretIntegerMinMax(t *testing.T) {

	t.Parallel()

	type testCase struct {
		min interpreter.Value
		max interpreter.Value
	}

	test := func(t *testing.T, ty sema.Type, field string, expected interpreter.Value) {

		inter := parseCheckAndInterpret(t,
			fmt.Sprintf(
				`
				  let x = %s.%s
				`,
				ty,
				field,
			),
		)

		RequireValuesEqual(
			t,
			inter,
			expected,
			inter.Globals["x"].GetValue(),
		)
	}

	testCases := map[sema.Type]testCase{
		sema.IntType: {},
		sema.UIntType: {
			min: interpreter.NewUIntValueFromUint64(0),
		},
		sema.UInt8Type: {
			min: interpreter.UInt8Value(0),
			max: interpreter.UInt8Value(math.MaxUint8),
		},
		sema.UInt16Type: {
			min: interpreter.UInt16Value(0),
			max: interpreter.UInt16Value(math.MaxUint16),
		},
		sema.UInt32Type: {
			min: interpreter.UInt32Value(0),
			max: interpreter.UInt32Value(math.MaxUint32),
		},
		sema.UInt64Type: {
			min: interpreter.UInt64Value(0),
			max: interpreter.UInt64Value(math.MaxUint64),
		},
		sema.UInt128Type: {
			min: interpreter.NewUInt128ValueFromUint64(0),
			max: interpreter.NewUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
		},
		sema.UInt256Type: {
			min: interpreter.NewUInt256ValueFromUint64(0),
			max: interpreter.NewUInt256ValueFromBigInt(sema.UInt256TypeMaxIntBig),
		},
		sema.Word8Type: {
			min: interpreter.Word8Value(0),
			max: interpreter.Word8Value(math.MaxUint8),
		},
		sema.Word16Type: {
			min: interpreter.Word16Value(0),
			max: interpreter.Word16Value(math.MaxUint16),
		},
		sema.Word32Type: {
			min: interpreter.Word32Value(0),
			max: interpreter.Word32Value(math.MaxUint32),
		},
		sema.Word64Type: {
			min: interpreter.Word64Value(0),
			max: interpreter.Word64Value(math.MaxUint64),
		},
		sema.Int8Type: {
			min: interpreter.Int8Value(math.MinInt8),
			max: interpreter.Int8Value(math.MaxInt8),
		},
		sema.Int16Type: {
			min: interpreter.Int16Value(math.MinInt16),
			max: interpreter.Int16Value(math.MaxInt16),
		},
		sema.Int32Type: {
			min: interpreter.Int32Value(math.MinInt32),
			max: interpreter.Int32Value(math.MaxInt32),
		},
		sema.Int64Type: {
			min: interpreter.Int64Value(math.MinInt64),
			max: interpreter.Int64Value(math.MaxInt64),
		},
		sema.Int128Type: {
			min: interpreter.NewInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
			max: interpreter.NewInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
		},
		sema.Int256Type: {
			min: interpreter.NewInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
			max: interpreter.NewInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
		},
	}

	for _, ty := range sema.AllIntegerTypes {
		// Only test leaf types
		switch ty {
		case sema.IntegerType, sema.SignedIntegerType:
			continue
		}

		if _, ok := testCases[ty]; !ok {
			require.Fail(t, "missing type: %s", ty.String())
		}
	}

	for ty, testCase := range testCases {

		t.Run(ty.String(), func(t *testing.T) {
			if testCase.min != nil {
				test(t, ty, sema.NumberTypeMinFieldName, testCase.min)
			}
			if testCase.max != nil {
				test(t, ty, sema.NumberTypeMaxFieldName, testCase.max)
			}
		})
	}
}
