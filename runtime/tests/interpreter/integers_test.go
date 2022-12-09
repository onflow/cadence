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
	"testing/quick"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/onflow/cadence/runtime/tests/utils"

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

var testIntegerTypesAndValues = map[string]interpreter.Value{
	// Int*
	"Int":    interpreter.NewUnmeteredIntValueFromInt64(50),
	"Int8":   interpreter.NewUnmeteredInt8Value(50),
	"Int16":  interpreter.NewUnmeteredInt16Value(50),
	"Int32":  interpreter.NewUnmeteredInt32Value(50),
	"Int64":  interpreter.NewUnmeteredInt64Value(50),
	"Int128": interpreter.NewUnmeteredInt128ValueFromInt64(50),
	"Int256": interpreter.NewUnmeteredInt256ValueFromInt64(50),
	// UInt*
	"UInt":    interpreter.NewUnmeteredUIntValueFromUint64(50),
	"UInt8":   interpreter.NewUnmeteredUInt8Value(50),
	"UInt16":  interpreter.NewUnmeteredUInt16Value(50),
	"UInt32":  interpreter.NewUnmeteredUInt32Value(50),
	"UInt64":  interpreter.NewUnmeteredUInt64Value(50),
	"UInt128": interpreter.NewUnmeteredUInt128ValueFromUint64(50),
	"UInt256": interpreter.NewUnmeteredUInt256ValueFromUint64(50),
	// Word*
	"Word8":  interpreter.NewUnmeteredWord8Value(50),
	"Word16": interpreter.NewUnmeteredWord16Value(50),
	"Word32": interpreter.NewUnmeteredWord32Value(50),
	"Word64": interpreter.NewUnmeteredWord64Value(50),
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
				inter.Globals.Get("x").GetValue(),
			)

			AssertValuesEqual(
				t,
				inter,
				value,
				inter.Globals.Get("y").GetValue(),
			)

			AssertValuesEqual(
				t,
				inter,
				interpreter.TrueValue,
				inter.Globals.Get("z").GetValue(),
			)

		})
	}
}

func TestInterpretWordOverflowConversions(t *testing.T) {

	t.Parallel()

	words := map[string]*big.Int{
		"Word8":  sema.UInt8TypeMaxInt,
		"Word16": sema.UInt16TypeMaxInt,
		"Word32": sema.UInt32TypeMaxInt,
		"Word64": sema.UInt64TypeMaxInt,
	}

	for typeName, value := range words {

		t.Run(typeName, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      let x = %s
                      let y = %s(x + 1)
                    `,
					value.String(),
					typeName,
				),
			)

			require.Equal(
				t,
				"0",
				inter.Globals.Get("y").GetValue().String(),
			)
		})
	}
}

func TestInterpretWordUnderflowConversions(t *testing.T) {

	t.Parallel()

	words := map[string]*big.Int{
		"Word8":  sema.UInt8TypeMaxInt,
		"Word16": sema.UInt16TypeMaxInt,
		"Word32": sema.UInt32TypeMaxInt,
		"Word64": sema.UInt64TypeMaxInt,
	}

	for typeName, value := range words {

		t.Run(typeName, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      let x = 0
                      let y = %s(x - 1)
                    `,
					typeName,
				),
			)

			require.Equal(
				t,
				value.String(),
				inter.Globals.Get("y").GetValue().String(),
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
			inter.Globals.Get("x").GetValue(),
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
			inter.Globals.Get("x").GetValue(),
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
		RequireError(t, err)

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
		RequireError(t, err)

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
				inter.Globals.Get("x").GetValue(),
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
				interpreter.NewUnmeteredSomeValueNonCopying(value),
				inter.Globals.Get("x").GetValue(),
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
				inter.Globals.Get("x").GetValue(),
			)

			_, err := inter.Invoke("test")
			require.NoError(t, err)

			numberValue := value.(interpreter.NumberValue)
			AssertValuesEqual(
				t,
				inter,
				numberValue.Plus(inter, numberValue, interpreter.EmptyLocationRange),
				inter.Globals.Get("x").GetValue(),
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
				interpreter.NewUnmeteredSomeValueNonCopying(value),
				inter.Globals.Get("x").GetValue(),
			)

			_, err := inter.Invoke("test")
			require.NoError(t, err)

			numberValue := value.(interpreter.NumberValue)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredSomeValueNonCopying(
					numberValue.Plus(inter, numberValue, interpreter.EmptyLocationRange),
				),
				inter.Globals.Get("x").GetValue(),
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
				inter.Globals.Get("x").GetValue(),
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
				interpreter.NewUnmeteredSomeValueNonCopying(value),
				inter.Globals.Get("x").GetValue(),
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
				interpreter.NewUnmeteredSomeValueNonCopying(value),
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
			RequireError(t, err)

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
			fortyTwo: interpreter.NewUnmeteredIntValueFromInt64(42),
			// Int does not actually have a minimum, but create a "large" value,
			// which can be used for testing against other types
			min: func() interpreter.Value {
				i := big.NewInt(-1)
				i.Lsh(i, 1000)
				return interpreter.NewUnmeteredIntValueFromBigInt(i)
			}(),
			// Int does not actually have a maximum, but create a "large" value,
			// which can be used for testing against other types
			max: func() interpreter.Value {
				i := big.NewInt(1)
				i.Lsh(i, 1000)
				return interpreter.NewUnmeteredIntValueFromBigInt(i)
			}(),
		},
		sema.UIntType: {
			fortyTwo: interpreter.NewUnmeteredUIntValueFromUint64(42),
			min:      interpreter.NewUnmeteredUIntValueFromUint64(0),
			// UInt does not actually have a maximum, but create a "large" value,
			// which can be used for testing against other types
			max: func() interpreter.Value {
				i := big.NewInt(1)
				i.Lsh(i, 1000)
				return interpreter.NewUnmeteredUIntValueFromBigInt(i)
			}(),
		},
		sema.UInt8Type: {
			fortyTwo: interpreter.NewUnmeteredUInt8Value(42),
			min:      interpreter.NewUnmeteredUInt8Value(0),
			max:      interpreter.NewUnmeteredUInt8Value(math.MaxUint8),
		},
		sema.UInt16Type: {
			fortyTwo: interpreter.NewUnmeteredUInt16Value(42),
			min:      interpreter.NewUnmeteredUInt16Value(0),
			max:      interpreter.NewUnmeteredUInt16Value(math.MaxUint16),
		},
		sema.UInt32Type: {
			fortyTwo: interpreter.NewUnmeteredUInt32Value(42),
			min:      interpreter.NewUnmeteredUInt32Value(0),
			max:      interpreter.NewUnmeteredUInt32Value(math.MaxUint32),
		},
		sema.UInt64Type: {
			fortyTwo: interpreter.NewUnmeteredUInt64Value(42),
			min:      interpreter.NewUnmeteredUInt64Value(0),
			max:      interpreter.NewUnmeteredUInt64Value(math.MaxUint64),
		},
		sema.UInt128Type: {
			fortyTwo: interpreter.NewUnmeteredUInt128ValueFromUint64(42),
			min:      interpreter.NewUnmeteredUInt128ValueFromUint64(0),
			max:      interpreter.NewUnmeteredUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
		},
		sema.UInt256Type: {
			fortyTwo: interpreter.NewUnmeteredUInt256ValueFromUint64(42),
			min:      interpreter.NewUnmeteredUInt256ValueFromUint64(0),
			max:      interpreter.NewUnmeteredUInt256ValueFromBigInt(sema.UInt256TypeMaxIntBig),
		},
		sema.Word8Type: {
			fortyTwo: interpreter.NewUnmeteredWord8Value(42),
			min:      interpreter.NewUnmeteredWord8Value(0),
			max:      interpreter.NewUnmeteredWord8Value(math.MaxUint8),
		},
		sema.Word16Type: {
			fortyTwo: interpreter.NewUnmeteredWord16Value(42),
			min:      interpreter.NewUnmeteredWord16Value(0),
			max:      interpreter.NewUnmeteredWord16Value(math.MaxUint16),
		},
		sema.Word32Type: {
			fortyTwo: interpreter.NewUnmeteredWord32Value(42),
			min:      interpreter.NewUnmeteredWord32Value(0),
			max:      interpreter.NewUnmeteredWord32Value(math.MaxUint32),
		},
		sema.Word64Type: {
			fortyTwo: interpreter.NewUnmeteredWord64Value(42),
			min:      interpreter.NewUnmeteredWord64Value(0),
			max:      interpreter.NewUnmeteredWord64Value(math.MaxUint64),
		},
		sema.Int8Type: {
			fortyTwo: interpreter.NewUnmeteredInt8Value(42),
			min:      interpreter.NewUnmeteredInt8Value(math.MinInt8),
			max:      interpreter.NewUnmeteredInt8Value(math.MaxInt8),
		},
		sema.Int16Type: {
			fortyTwo: interpreter.NewUnmeteredInt16Value(42),
			min:      interpreter.NewUnmeteredInt16Value(math.MinInt16),
			max:      interpreter.NewUnmeteredInt16Value(math.MaxInt16),
		},
		sema.Int32Type: {
			fortyTwo: interpreter.NewUnmeteredInt32Value(42),
			min:      interpreter.NewUnmeteredInt32Value(math.MinInt32),
			max:      interpreter.NewUnmeteredInt32Value(math.MaxInt32),
		},
		sema.Int64Type: {
			fortyTwo: interpreter.NewUnmeteredInt64Value(42),
			min:      interpreter.NewUnmeteredInt64Value(math.MinInt64),
			max:      interpreter.NewUnmeteredInt64Value(math.MaxInt64),
		},
		sema.Int128Type: {
			fortyTwo: interpreter.NewUnmeteredInt128ValueFromInt64(42),
			min:      interpreter.NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
			max:      interpreter.NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
		},
		sema.Int256Type: {
			fortyTwo: interpreter.NewUnmeteredInt256ValueFromInt64(42),
			min:      interpreter.NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
			max:      interpreter.NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
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
					// words wrap instead of underflow
					switch targetType {
					case sema.Word8Type,
						sema.Word16Type,
						sema.Word32Type,
						sema.Word64Type:
					default:
						t.Run("underflow", func(t *testing.T) {
							test(t, sourceType, targetType, sourceValues.min, nil, interpreter.UnderflowError{})
						})
					}
				}

				// Check a "typical" value can be converted

				t.Run("valid", func(t *testing.T) {
					test(t, sourceType, targetType, sourceValues.fortyTwo, targetValues.fortyTwo, nil)
				})

				// Check overflow is handled correctly

				targetMaxInt := targetType.MaxInt()
				sourceMaxInt := sourceType.MaxInt()

				if targetMaxInt != nil && (sourceMaxInt == nil || sourceMaxInt.Cmp(targetMaxInt) > 0) {
					// words wrap instead of overflow
					switch targetType {
					case sema.Word8Type,
						sema.Word16Type,
						sema.Word32Type,
						sema.Word64Type:
					default:
						t.Run("overflow", func(t *testing.T) {
							test(t, sourceType, targetType, sourceValues.max, nil, interpreter.OverflowError{})
						})
					}
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
			inter.Globals.Get("x").GetValue(),
		)
	}

	testCases := map[sema.Type]testCase{
		sema.IntType: {},
		sema.UIntType: {
			min: interpreter.NewUnmeteredUIntValueFromUint64(0),
		},
		sema.UInt8Type: {
			min: interpreter.NewUnmeteredUInt8Value(0),
			max: interpreter.NewUnmeteredUInt8Value(math.MaxUint8),
		},
		sema.UInt16Type: {
			min: interpreter.NewUnmeteredUInt16Value(0),
			max: interpreter.NewUnmeteredUInt16Value(math.MaxUint16),
		},
		sema.UInt32Type: {
			min: interpreter.NewUnmeteredUInt32Value(0),
			max: interpreter.NewUnmeteredUInt32Value(math.MaxUint32),
		},
		sema.UInt64Type: {
			min: interpreter.NewUnmeteredUInt64Value(0),
			max: interpreter.NewUnmeteredUInt64Value(math.MaxUint64),
		},
		sema.UInt128Type: {
			min: interpreter.NewUnmeteredUInt128ValueFromUint64(0),
			max: interpreter.NewUnmeteredUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
		},
		sema.UInt256Type: {
			min: interpreter.NewUnmeteredUInt256ValueFromUint64(0),
			max: interpreter.NewUnmeteredUInt256ValueFromBigInt(sema.UInt256TypeMaxIntBig),
		},
		sema.Word8Type: {
			min: interpreter.NewUnmeteredWord8Value(0),
			max: interpreter.NewUnmeteredWord8Value(math.MaxUint8),
		},
		sema.Word16Type: {
			min: interpreter.NewUnmeteredWord16Value(0),
			max: interpreter.NewUnmeteredWord16Value(math.MaxUint16),
		},
		sema.Word32Type: {
			min: interpreter.NewUnmeteredWord32Value(0),
			max: interpreter.NewUnmeteredWord32Value(math.MaxUint32),
		},
		sema.Word64Type: {
			min: interpreter.NewUnmeteredWord64Value(0),
			max: interpreter.NewUnmeteredWord64Value(math.MaxUint64),
		},
		sema.Int8Type: {
			min: interpreter.NewUnmeteredInt8Value(math.MinInt8),
			max: interpreter.NewUnmeteredInt8Value(math.MaxInt8),
		},
		sema.Int16Type: {
			min: interpreter.NewUnmeteredInt16Value(math.MinInt16),
			max: interpreter.NewUnmeteredInt16Value(math.MaxInt16),
		},
		sema.Int32Type: {
			min: interpreter.NewUnmeteredInt32Value(math.MinInt32),
			max: interpreter.NewUnmeteredInt32Value(math.MaxInt32),
		},
		sema.Int64Type: {
			min: interpreter.NewUnmeteredInt64Value(math.MinInt64),
			max: interpreter.NewUnmeteredInt64Value(math.MaxInt64),
		},
		sema.Int128Type: {
			min: interpreter.NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
			max: interpreter.NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
		},
		sema.Int256Type: {
			min: interpreter.NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
			max: interpreter.NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
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

func TestStringIntegerConversion(t *testing.T) {
	t.Parallel()

	test := func(t *testing.T, typ sema.Type) {
		t.Parallel()

		numericType := typ.(*sema.NumericType)
		low := numericType.MinInt()
		if low == nil {
			low = big.NewInt(0)
		}
		high := numericType.MaxInt()
		if high == nil {
			high = big.NewInt(math.MaxInt64)
		}

		code := fmt.Sprintf(`
			fun testFromString(_ input: String): Int? {
				return %s.fromString(input).map(Int)
			}
		`, typ.String())
		inter := parseCheckAndInterpret(t, code)

		placeInRange := func(x *big.Int) *big.Int {
			z := big.NewInt(0).Sub(high, low)
			z.Mod(x, z)
			z.Add(low, z)
			return z
		}

		prop := func(x int64) bool {
			normalized := placeInRange(big.NewInt(x))
			strInput := interpreter.NewUnmeteredStringValue(normalized.String())
			expected := interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromBigInt(normalized),
			)

			result, err := inter.Invoke("testFromString", strInput)
			return err == nil && ValuesAreEqual(inter, expected, result)
		}

		if err := quick.Check(prop, nil); err != nil {
			t.Error(err)
		}
	}

	for _, typ := range append(sema.AllSignedIntegerTypes, sema.AllUnsignedIntegerTypes...) {
		t.Run(typ.String(), func(t *testing.T) { test(t, typ) })
	}
}
