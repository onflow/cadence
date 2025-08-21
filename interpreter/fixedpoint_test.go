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

package interpreter_test

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/fixedpoint"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestInterpretNegativeZeroFixedPoint(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
      let x = -0.42
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredFix64Value(-42000000),
		inter.GetGlobal("x"),
	)
}

func TestInterpretFixedPointConversionAndAddition(t *testing.T) {

	t.Parallel()

	tests := map[string]interpreter.Value{
		// Fix*
		"Fix64":  interpreter.NewUnmeteredFix64Value(123000000),
		"Fix128": interpreter.NewUnmeteredFix128ValueWithIntegerAndScale(123, 22),

		// UFix*
		"UFix64":  interpreter.NewUnmeteredUFix64Value(123000000),
		"UFix128": interpreter.NewUnmeteredUFix128ValueWithIntegerAndScale(123, 22),
	}

	for _, fixedPointType := range sema.AllFixedPointTypes {
		// Only test leaf types
		switch fixedPointType {
		case sema.FixedPointType, sema.SignedFixedPointType:
			continue
		}

		if _, ok := tests[fixedPointType.String()]; !ok {
			panic(fmt.Sprintf("broken test: missing %s", fixedPointType))
		}
	}

	for fixedPointType, value := range tests {

		t.Run(fixedPointType, func(t *testing.T) {

			inter := parseCheckAndPrepare(t,
				fmt.Sprintf(
					`
                      let x: %[1]s = 1.23
                      let y = %[1]s(0.42) + %[1]s(0.81)
                      let z = y == x
                    `,
					fixedPointType,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				value,
				inter.GetGlobal("x"),
			)

			AssertValuesEqual(
				t,
				inter,
				value,
				inter.GetGlobal("y"),
			)

			AssertValuesEqual(
				t,
				inter,
				interpreter.TrueValue,
				inter.GetGlobal("z"),
			)

		})
	}
}

var testFixedPointValues = map[sema.Type]interpreter.Value{
	// Fix types
	sema.Fix64Type:  interpreter.NewUnmeteredFix64Value(50 * sema.Fix64Factor),
	sema.Fix128Type: interpreter.NewUnmeteredFix128ValueWithInteger(50, interpreter.EmptyLocationRange),

	// UFix types
	sema.UFix64Type:  interpreter.NewUnmeteredUFix64Value(50 * sema.Fix64Factor),
	sema.UFix128Type: interpreter.NewUnmeteredUFix128ValueWithInteger(50, interpreter.EmptyLocationRange),
}

func init() {
	for _, fixedPointType := range sema.AllFixedPointTypes {
		// Only test leaf types
		switch fixedPointType {
		case sema.FixedPointType, sema.SignedFixedPointType:
			continue
		}

		if _, ok := testFixedPointValues[fixedPointType]; !ok {
			panic(fmt.Sprintf("broken test: missing fixed-point type: %s", fixedPointType))
		}
	}
}

func TestInterpretFixedPointToIntegerConversions(t *testing.T) {
	t.Parallel()

	for fixedPointType, fixedPointValue := range testFixedPointValues {

		for integerType, integerValue := range testIntegerTypesAndValues {

			testName := fmt.Sprintf("valid %s to %s", fixedPointType, integerType)

			t.Run(testName, func(t *testing.T) {

				t.Parallel()

				inter := parseCheckAndPrepare(t,
					fmt.Sprintf(
						`
                          let x: %[1]s = 50.0
                          let y = %[2]s(x)
                        `,
						fixedPointType,
						integerType,
					),
				)

				AssertValuesEqual(
					t,
					inter,
					fixedPointValue,
					inter.GetGlobal("x"),
				)

				AssertValuesEqual(
					t,
					inter,
					integerValue,
					inter.GetGlobal("y"),
				)
			})
		}
	}
}

func TestInterpretIntegerToFixedPointConversions(t *testing.T) {
	t.Parallel()

	test := func(
		t *testing.T,
		targetFixType sema.Type,
		sourceIntegerType sema.Type,
		testedValueStr string,
		expectedError error,
	) {
		t.Run(sourceIntegerType.String(), func(t *testing.T) {

			t.Parallel()

			inter := parseCheckAndPrepare(t,
				fmt.Sprintf(
					`
                         fun test(): %[1]s {
                             let x: %[2]s = %[3]s
                             return %[1]s(x)
                         }
                       `,
					targetFixType,
					sourceIntegerType,
					testedValueStr,
				),
			)

			_, err := inter.Invoke("test")
			RequireError(t, err)
			require.ErrorAs(t, err, &expectedError)
		})
	}

	t.Run("Fix64", func(t *testing.T) {
		t.Parallel()

		t.Run("invalid integer > Fix64 max int", func(t *testing.T) {

			const testedValue = sema.Fix64TypeMaxInt + 1
			testValueBig := big.NewInt(0).SetUint64(testedValue)

			for _, integerType := range sema.AllIntegerTypes {

				// Only test for integer types that can hold testedValue
				maxInt := integerType.(sema.IntegerRangedType).MaxInt()
				if maxInt != nil && maxInt.Cmp(testValueBig) < 0 {
					continue
				}

				test(
					t,
					sema.Fix64Type,
					integerType,
					testValueBig.String(),
					&interpreter.OverflowError{},
				)
			}
		})

		t.Run("invalid integer < Fix64 min int", func(t *testing.T) {

			const testedValue = sema.Fix64TypeMinInt - 1
			testValueBig := big.NewInt(testedValue)

			for _, integerType := range sema.AllSignedIntegerTypes {

				// Only test for integer types that can hold testedValue

				minInt := integerType.(sema.IntegerRangedType).MinInt()
				if minInt != nil && minInt.Cmp(testValueBig) > 0 {
					continue
				}

				test(
					t,
					sema.Fix64Type,
					integerType,
					testValueBig.String(),
					&interpreter.OverflowError{},
				)
			}
		})
	})

	t.Run("Fix128", func(t *testing.T) {
		t.Parallel()

		t.Run("invalid integer > Fix128 max int", func(t *testing.T) {

			// max + 1
			testValueBig := new(big.Int).Add(fixedpoint.Fix128TypeMaxIntBig, big.NewInt(1))

			for _, integerType := range sema.AllIntegerTypes {

				// Only test for integer types that can hold testedValue
				maxInt := integerType.(sema.IntegerRangedType).MaxInt()
				if maxInt != nil && maxInt.Cmp(testValueBig) < 0 {
					continue
				}

				test(
					t,
					sema.Fix128Type,
					integerType,
					testValueBig.String(),
					&interpreter.OverflowError{},
				)
			}
		})

		t.Run("invalid integer < Fix128 min int", func(t *testing.T) {

			// min - 1
			testValueBig := new(big.Int).Sub(fixedpoint.Fix128TypeMinIntBig, big.NewInt(1))

			for _, integerType := range sema.AllSignedIntegerTypes {

				// Only test for integer types that can hold testedValue

				minInt := integerType.(sema.IntegerRangedType).MinInt()
				if minInt != nil && minInt.Cmp(testValueBig) > 0 {
					continue
				}

				test(
					t,
					sema.Fix128Type,
					integerType,
					testValueBig.String(),
					&interpreter.OverflowError{},
				)
			}
		})
	})

	t.Run("UFix64", func(t *testing.T) {
		t.Parallel()

		t.Run("invalid negative integer", func(t *testing.T) {

			for _, integerType := range sema.AllSignedIntegerTypes {
				test(
					t,
					sema.UFix64Type,
					integerType,
					"-1",
					&interpreter.OverflowError{},
				)
			}
		})

		t.Run("invalid big integer (>uint64)", func(t *testing.T) {

			bigIntegerTypes := []sema.Type{
				sema.Word64Type,
				sema.Word128Type,
				sema.Word256Type,
				sema.UInt64Type,
				sema.UInt128Type,
				sema.UInt256Type,
				sema.Int256Type,
				sema.Int128Type,
			}

			for _, integerType := range bigIntegerTypes {
				test(
					t,
					sema.UFix64Type,
					integerType,
					strconv.Itoa(int(sema.UFix64TypeMaxInt+1)),
					&interpreter.OverflowError{},
				)
			}
		})

		t.Run("invalid integer > UFix64 max", func(t *testing.T) {

			const testedValue = sema.UFix64TypeMaxInt + 1
			testValueBig := big.NewInt(0).SetUint64(testedValue)

			for _, integerType := range sema.AllIntegerTypes {

				// Only test for integer types that can hold testedValue

				maxInt := integerType.(sema.IntegerRangedType).MaxInt()
				if maxInt != nil && maxInt.Cmp(testValueBig) < 0 {
					continue
				}

				test(
					t,
					sema.UFix64Type,
					integerType,
					testValueBig.String(),
					&interpreter.OverflowError{},
				)
			}
		})
	})

	t.Run("UFix128", func(t *testing.T) {
		t.Parallel()

		t.Run("invalid negative integer", func(t *testing.T) {

			for _, integerType := range sema.AllSignedIntegerTypes {
				test(
					t,
					sema.UFix128Type,
					integerType,
					"-1",
					&interpreter.OverflowError{},
				)
			}
		})

		t.Run("invalid integer > UFix128 max int", func(t *testing.T) {

			// max + 1
			testValueBig := new(big.Int).Add(fixedpoint.UFix128TypeMaxIntBig, big.NewInt(1))

			for _, integerType := range sema.AllIntegerTypes {

				// Only test for integer types that can hold testedValue
				maxInt := integerType.(sema.IntegerRangedType).MaxInt()
				if maxInt != nil && maxInt.Cmp(testValueBig) < 0 {
					continue
				}

				test(
					t,
					sema.UFix128Type,
					integerType,
					testValueBig.String(),
					&interpreter.OverflowError{},
				)
			}
		})

		t.Run("invalid integer < Fix128 min int", func(t *testing.T) {

			// min - 1
			testValueBig := new(big.Int).Sub(fixedpoint.UFix128TypeMinIntBig, big.NewInt(1))

			for _, integerType := range sema.AllSignedIntegerTypes {

				// Only test for integer types that can hold testedValue

				minInt := integerType.(sema.IntegerRangedType).MinInt()
				if minInt != nil && minInt.Cmp(testValueBig) > 0 {
					continue
				}

				test(
					t,
					sema.UFix128Type,
					integerType,
					testValueBig.String(),
					&interpreter.OverflowError{},
				)
			}
		})
	})
}

func TestInterpretFixedPointToFixedPointConversions(t *testing.T) {
	t.Parallel()

	type values struct {
		positiveValue      interpreter.Value
		negativeValue      interpreter.Value
		largePositiveValue interpreter.Value
		largeNegativeValue interpreter.Value
		min                interpreter.Value
		max                interpreter.Value
	}

	testValues := map[*sema.FixedPointNumericType]values{
		// Fix types
		sema.Fix64Type: {
			positiveValue: interpreter.NewUnmeteredFix64ValueWithInteger(42, interpreter.EmptyLocationRange),
			negativeValue: interpreter.NewUnmeteredFix64ValueWithInteger(-42, interpreter.EmptyLocationRange),

			// Use the max(Fix64) as the large positive value
			largePositiveValue: interpreter.NewUnmeteredFix64Value(math.MaxInt64),
			// Use the min(Fix64) as the large negative value
			largeNegativeValue: interpreter.NewUnmeteredFix64Value(math.MinInt64),

			min: interpreter.NewUnmeteredFix64ValueWithInteger(sema.Fix64TypeMinInt, interpreter.EmptyLocationRange),
			max: interpreter.NewUnmeteredFix64ValueWithInteger(sema.Fix64TypeMaxInt, interpreter.EmptyLocationRange),
		},
		sema.Fix128Type: {
			positiveValue: interpreter.NewUnmeteredFix128ValueWithInteger(42, interpreter.EmptyLocationRange),
			negativeValue: interpreter.NewUnmeteredFix128ValueWithInteger(-42, interpreter.EmptyLocationRange),

			// Use the max(Fix64) as the large positive value
			largePositiveValue: interpreter.NewFix128ValueFromBigInt(nil, fixedpoint.Fix64TypeMaxScaledTo128),
			// Use the min(Fix64) as the large negative value
			largeNegativeValue: interpreter.NewFix128ValueFromBigInt(nil, fixedpoint.Fix64TypeMinScaledTo128),

			min: interpreter.NewUnmeteredFix128Value(fixedpoint.Fix128TypeMin),
			max: interpreter.NewUnmeteredFix128Value(fixedpoint.Fix128TypeMax),
		},

		// UFix types
		sema.UFix64Type: {
			positiveValue: interpreter.NewUnmeteredUFix64ValueWithInteger(42, interpreter.EmptyLocationRange),

			// Use the max(Fix64) as the large positive value
			largePositiveValue: interpreter.NewUnmeteredUFix64Value(math.MaxInt64),

			min: interpreter.NewUnmeteredUFix64ValueWithInteger(sema.UFix64TypeMinInt, interpreter.EmptyLocationRange),
			max: interpreter.NewUnmeteredUFix64ValueWithInteger(sema.UFix64TypeMaxInt, interpreter.EmptyLocationRange),
		},
		sema.UFix128Type: {
			positiveValue: interpreter.NewUnmeteredUFix128ValueWithInteger(42, interpreter.EmptyLocationRange),

			// Use the max(Fix64) as the large positive value
			largePositiveValue: interpreter.NewUFix128ValueFromBigInt(nil, fixedpoint.Fix64TypeMaxScaledTo128),

			min: interpreter.NewUnmeteredUFix128Value(fixedpoint.UFix128TypeMin),
			max: interpreter.NewUnmeteredUFix128Value(fixedpoint.UFix128TypeMax),
		},
	}

	for _, fixedPointType := range sema.AllFixedPointTypes {
		// Only test leaf types
		switch fixedPointType {
		case sema.FixedPointType, sema.SignedFixedPointType:
			continue
		}

		_, ok := testValues[fixedPointType.(*sema.FixedPointNumericType)]
		require.True(t, ok, "missing expected value for type %s", fixedPointType.String())
	}

	test := func(
		t *testing.T,
		sourceType sema.Type,
		targetType sema.Type,
		value interpreter.Value,
		expectedValue interpreter.Value,
		expectedError error,
	) {

		inter := parseCheckAndPrepare(t,
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
				expectedValueStr := value.String()
				actualValueStr := result.String()

				expectedValueStrLen := len(expectedValueStr)
				actualValueStrLen := len(actualValueStr)

				// Pad right (append zeros) to match the length.
				if expectedValueStrLen < actualValueStrLen {
					for i := 0; i < actualValueStrLen-expectedValueStrLen; i++ {
						expectedValueStr += "0"
					}
				} else if expectedValueStrLen > actualValueStrLen {
					for i := 0; i < expectedValueStrLen-actualValueStrLen; i++ {
						actualValueStr += "0"
					}
				}

				assert.Equal(t, expectedValueStr, actualValueStr)
			}
		}
	}

	for sourceType, sourceValues := range testValues {
		for targetType, targetValues := range testValues {

			t.Run(fmt.Sprintf("%s to %s", sourceType, targetType), func(t *testing.T) {

				// Check a "typical" positive value

				t.Run("valid positive", func(t *testing.T) {
					test(t,
						sourceType,
						targetType,
						sourceValues.positiveValue,
						targetValues.positiveValue,
						nil,
					)
				})

				// Check a "typical" negative value (only if both types are signed)

				sourceNegativeValue := sourceValues.negativeValue
				targetNegativeValue := targetValues.negativeValue
				if sourceNegativeValue != nil && targetNegativeValue != nil {
					t.Run("valid negative", func(t *testing.T) {
						test(t,
							sourceType,
							targetType,
							sourceNegativeValue,
							targetNegativeValue,
							nil,
						)
					})
				}

				t.Run("valid large positive", func(t *testing.T) {
					test(t,
						sourceType,
						targetType,
						sourceValues.largePositiveValue,
						targetValues.largePositiveValue,
						nil,
					)
				})

				sourceLargeNegativeValue := sourceValues.largeNegativeValue
				targetLargeNegativeValue := targetValues.largeNegativeValue
				if sourceLargeNegativeValue != nil && targetLargeNegativeValue != nil {
					t.Run("valid large negative", func(t *testing.T) {
						test(t,
							sourceType,
							targetType,
							sourceLargeNegativeValue,
							targetLargeNegativeValue,
							nil,
						)
					})
				}

				targetMaxInt := targetType.MaxInt()
				sourceMaxInt := sourceType.MaxInt()
				targetMinInt := targetType.MinInt()
				sourceMinInt := sourceType.MinInt()

				if targetMaxInt.Cmp(sourceMaxInt) >= 0 {
					// If the target type can accommodate source type's max value, check for conversion.
					t.Run("max", func(t *testing.T) {
						test(
							t,
							sourceType,
							targetType,
							sourceValues.max,
							nil,
							nil,
						)
					})
				} else {
					// Otherwise,check whether the overflow is handled correctly.
					t.Run("overflow", func(t *testing.T) {
						test(
							t,
							sourceType,
							targetType,
							sourceValues.max,
							nil,
							&interpreter.OverflowError{},
						)
					})
				}

				// Check the minimum value can be converted.

				if targetMinInt.Cmp(sourceMinInt) <= 0 {
					// If the target type can accommodate source type's min value, check for conversion.
					t.Run("min", func(t *testing.T) {
						test(
							t,
							sourceType,
							targetType,
							sourceValues.min,
							nil,
							nil,
						)
					})
				} else {
					// Otherwise, check whether the underflow is handled correctly.
					t.Run("underflow", func(t *testing.T) {
						test(
							t,
							sourceType,
							targetType,
							sourceValues.min,
							nil,
							&interpreter.UnderflowError{},
						)
					})
				}
			})
		}
	}

}

func TestInterpretFixedPointMinMax(t *testing.T) {

	t.Parallel()

	type testCase struct {
		min interpreter.Value
		max interpreter.Value
	}

	test := func(t *testing.T, ty sema.Type, test testCase) {

		inter := parseCheckAndPrepare(t,
			fmt.Sprintf(
				`
                  let min = %[1]s.min
                  let max = %[1]s.max
                `,
				ty,
			),
		)

		RequireValuesEqual(
			t,
			inter,
			test.min,
			inter.GetGlobal("min"),
		)
		RequireValuesEqual(
			t,
			inter,
			test.max,
			inter.GetGlobal("max"),
		)
	}

	testCases := map[sema.Type]testCase{
		sema.Fix64Type: {
			min: interpreter.NewUnmeteredFix64Value(math.MinInt64),
			max: interpreter.NewUnmeteredFix64Value(math.MaxInt64),
		},
		sema.Fix128Type: {
			min: interpreter.NewUnmeteredFix128Value(fixedpoint.Fix128TypeMin),
			max: interpreter.NewUnmeteredFix128Value(fixedpoint.Fix128TypeMax),
		},
		sema.UFix64Type: {
			min: interpreter.NewUnmeteredUFix64Value(0),
			max: interpreter.NewUnmeteredUFix64Value(math.MaxUint64),
		},
		sema.UFix128Type: {
			min: interpreter.NewUnmeteredUFix128Value(fixedpoint.UFix128TypeMin),
			max: interpreter.NewUnmeteredUFix128Value(fixedpoint.UFix128TypeMax),
		},
	}

	for _, ty := range sema.AllFixedPointTypes {
		// Only test leaf types
		switch ty {
		case sema.FixedPointType, sema.SignedFixedPointType:
			continue
		}

		if _, ok := testCases[ty]; !ok {
			require.Fail(t, "missing type: %s", ty.String())
		}
	}

	for ty, testCase := range testCases {

		t.Run(ty.String(), func(t *testing.T) {
			test(t, ty, testCase)
		})
	}
}

func TestInterpretStringFixedPointConversion(t *testing.T) {
	t.Parallel()

	type testcase struct {
		decimal    *big.Int
		fractional *big.Int
	}

	type testsuite struct {
		toFixedValue func(isNegative bool, decimal, fractional *big.Int, scale uint) (interpreter.Value, error)
		intBounds    []*big.Int
		fracBounds   []*big.Int
	}

	bigZero := big.NewInt(0)
	bigOne := big.NewInt(1)

	suites := map[sema.Type]testsuite{
		sema.Fix64Type: {
			func(isNeg bool, decimal, fractional *big.Int, scale uint) (interpreter.Value, error) {
				fixedVal, err := fixedpoint.NewFix64(isNeg, decimal, fractional, scale)
				if err != nil {
					return nil, err
				}
				return interpreter.NewUnmeteredFix64Value(fixedVal.Int64()), nil
			},
			[]*big.Int{sema.Fix64TypeMinIntBig, sema.Fix64TypeMaxIntBig, bigZero, bigOne},
			[]*big.Int{sema.Fix64TypeMinFractionalBig, sema.Fix64TypeMaxFractionalBig, bigZero, bigOne},
		},

		sema.Fix128Type: {
			func(isNeg bool, decimal, fractional *big.Int, scale uint) (interpreter.Value, error) {
				fixedVal, err := fixedpoint.NewFix128(isNeg, decimal, fractional, scale)
				if err != nil {
					return nil, err
				}
				return interpreter.NewFix128ValueFromBigInt(nil, fixedVal), nil
			},
			[]*big.Int{sema.Fix128TypeMinIntBig, sema.Fix128TypeMaxIntBig, bigZero, bigOne},
			[]*big.Int{sema.Fix128TypeMinFractionalBig, sema.Fix128TypeMaxFractionalBig, bigZero, bigOne},
		},

		sema.UFix64Type: {
			func(_ bool, decimal, fractional *big.Int, scale uint) (interpreter.Value, error) {
				fixedVal, err := fixedpoint.NewUFix64(decimal, fractional, scale)
				if err != nil {
					return nil, err
				}
				return interpreter.NewUnmeteredUFix64Value(fixedVal.Uint64()), nil
			},
			[]*big.Int{sema.UFix64TypeMinIntBig, sema.UFix64TypeMaxIntBig, bigZero, bigOne},
			[]*big.Int{sema.UFix64TypeMinFractionalBig, sema.UFix64TypeMaxFractionalBig, bigZero, bigOne},
		},

		sema.UFix128Type: {
			func(isNeg bool, decimal, fractional *big.Int, scale uint) (interpreter.Value, error) {
				fixedVal, err := fixedpoint.NewUFix128(decimal, fractional, scale)
				if err != nil {
					return nil, err
				}
				return interpreter.NewUFix128ValueFromBigInt(nil, fixedVal), nil
			},
			[]*big.Int{sema.UFix128TypeMinIntBig, sema.UFix128TypeMaxIntBig, bigZero, bigOne},
			[]*big.Int{sema.UFix128TypeMinFractionalBig, sema.UFix128TypeMaxFractionalBig, bigZero, bigOne},
		},
	}

	for _, ty := range sema.AllFixedPointTypes {
		// Only test leaf types
		switch ty {
		case sema.FixedPointType, sema.SignedFixedPointType:
			continue
		}

		if _, ok := suites[ty]; !ok {
			require.Fail(t, fmt.Sprintf("missing type: %s", ty.String()))
		}
	}

	genCases := func(intComponents, fracComponents []*big.Int) []testcase {
		testcases := []testcase{
			{big.NewInt(10), big.NewInt(11)},
			{big.NewInt(420), big.NewInt(840)},
			{big.NewInt(123), big.NewInt(45)},
		}

		for _, intPart := range intComponents {
			for _, fracPart := range fracComponents {
				clonedInt := new(big.Int).Set(intPart)
				clonedFrac := new(big.Int).Set(fracPart)
				testcases = append(testcases, testcase{clonedInt, clonedFrac})
			}
		}
		return testcases
	}

	getMagnitude := func(n *big.Int) uint {
		var m uint

		cloned := new(big.Int).Set(n)
		bigTen := big.NewInt(10)

		for m = 0; cloned.Cmp(bigZero) != 0; m++ {
			cloned.Div(cloned, bigTen)
		}

		return m
	}

	test := func(typeName string, suite testsuite) {
		t.Run(typeName, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf(
				`
                    fun fromStringTest(s: String): %[1]s? {
                        let a = %[1]s.fromString(s)
                        let f = %[1]s.fromString
                        let b = f(s)
                        if a != b {
                            return nil
                        }
                        return b
                    }
                `,
				typeName,
			)

			inter := parseCheckAndPrepare(t, code)

			testcases := genCases(suite.intBounds, suite.fracBounds)

			for _, tc := range testcases {
				isNegative := tc.decimal.Cmp(big.NewInt(0)) == -1
				scale := getMagnitude(tc.fractional)

				absDecimal := new(big.Int)
				tc.decimal.Abs(absDecimal)
				absFractional := new(big.Int)
				tc.fractional.Abs(absFractional)

				expectedNumericVal, err := suite.toFixedValue(isNegative, absDecimal, absFractional, scale)
				require.NoError(t, err)

				expectedVal := interpreter.NewUnmeteredSomeValueNonCopying(expectedNumericVal)

				stringified := fmt.Sprintf("%d.%d", absDecimal, absFractional)
				res, err := inter.Invoke("fromStringTest", interpreter.NewUnmeteredStringValue(stringified))

				require.NoError(t, err)

				AssertEqualWithDiff(t, expectedVal, res)
			}

		})

	}
	for typeName, testsuite := range suites {
		test(typeName.QualifiedString(), testsuite)
	}

}

func TestInterpretFixedPointLiteral(t *testing.T) {
	t.Parallel()

	type testValue struct {
		name          string
		literal       string
		expectedValue interpreter.Value
		expectedError error
	}

	testCases := map[sema.Type][]testValue{
		sema.Fix64Type: {
			{
				name:          "min",
				literal:       "-92233720368.54775808",
				expectedValue: interpreter.NewUnmeteredFix64Value(interpreter.Fix64MinValue),
			},
			{
				name:          "max",
				literal:       "92233720368.54775807",
				expectedValue: interpreter.NewUnmeteredFix64Value(interpreter.Fix64MaxValue),
			},
			{
				name:          "underflow",
				literal:       "-92233720368.54775809",
				expectedError: &sema.InvalidFixedPointLiteralRangeError{},
			},
			{
				name:          "overflow",
				literal:       "92233720368.54775808",
				expectedError: &sema.InvalidFixedPointLiteralRangeError{},
			},
		},

		sema.UFix64Type: {
			{
				name:          "min",
				literal:       "0.0",
				expectedValue: interpreter.NewUnmeteredUFix64Value(0 * sema.Fix64Scale),
			},
			{
				name:          "max",
				literal:       "184467440737.09551615",
				expectedValue: interpreter.NewUnmeteredUFix64Value(interpreter.UFix64MaxValue),
			},
			{
				name:          "underflow",
				literal:       "-0.00000001",
				expectedError: &sema.InvalidFixedPointLiteralRangeError{},
			},
			{
				name:          "overflow",
				literal:       "184467440737.09551616",
				expectedError: &sema.InvalidFixedPointLiteralRangeError{},
			},
		},

		sema.Fix128Type: {
			{
				name:          "min",
				literal:       "-170141183460469.231731687303715884105728",
				expectedValue: interpreter.NewUnmeteredFix128Value(fixedpoint.Fix128TypeMin),
			},
			{
				name:          "max",
				literal:       "170141183460469.231731687303715884105727",
				expectedValue: interpreter.NewUnmeteredFix128Value(fixedpoint.Fix128TypeMax),
			},
			{
				name:          "underflow",
				literal:       "-170141183460469.231731687303715884105729",
				expectedError: &sema.InvalidFixedPointLiteralRangeError{},
			},
			{
				name:          "overflow",
				literal:       "170141183460469.231731687303715884105728",
				expectedError: &sema.InvalidFixedPointLiteralRangeError{},
			},
		},

		sema.UFix128Type: {
			{
				name:          "min",
				literal:       "0.0",
				expectedValue: interpreter.NewUnmeteredUFix128Value(fixedpoint.UFix128TypeMin),
			},
			{
				name:          "max",
				literal:       "340282366920938.463463374607431768211455",
				expectedValue: interpreter.NewUnmeteredUFix128Value(fixedpoint.UFix128TypeMax),
			},
			{
				name:          "underflow",
				literal:       "-0.000000000000000000000001",
				expectedError: &sema.InvalidFixedPointLiteralRangeError{},
			},
			{
				name:          "overflow",
				literal:       "340282366920938.463463374607431768211456",
				expectedError: &sema.InvalidFixedPointLiteralRangeError{},
			},
		},
	}

	for _, ty := range sema.AllFixedPointTypes {
		// Only test leaf types
		switch ty {
		case sema.FixedPointType, sema.SignedFixedPointType:
			continue
		}

		if _, ok := testCases[ty]; !ok {
			require.Fail(t, fmt.Sprintf("missing type: %s", ty.String()))
		}
	}

	for ty, values := range testCases {
		for _, value := range values {
			ty := ty
			literal := value.literal
			expectedValue := value.expectedValue
			expectedError := value.expectedError

			t.Run(fmt.Sprintf("%s_%s (%s)", ty, value.name, literal), func(t *testing.T) {

				t.Parallel()

				code := fmt.Sprintf(`
                    fun main(): %[1]s {
                        return %[2]s
                    }
                    `,
					ty,
					literal,
				)

				if expectedError == nil {
					invokable := parseCheckAndPrepare(t, code)

					result, err := invokable.Invoke("main")
					require.NoError(t, err)
					assert.Equal(t, expectedValue, result)
				} else {
					_, err := ParseAndCheck(t, code)
					RequireError(t, err)
					errs := RequireCheckerErrors(t, err, 1)
					assert.IsType(t, errs[0], expectedError)
				}
			})
		}
	}
}

func TestInterpretFixedPointLeastSignificantDecimalHandling(t *testing.T) {
	t.Parallel()

	type testValue struct {
		a, b   string
		result string
	}

	test := func(tt *testing.T, testCases map[sema.Type][]testValue, operation ast.Operation) {
		for typ, values := range testCases {
			for _, value := range values {
				typ := typ
				a := value.a
				b := value.b
				expectedResult := value.result

				tt.Run(typ.String(), func(ttt *testing.T) {
					ttt.Parallel()

					code := fmt.Sprintf(`
                        fun main(): %[1]s {
                            return %[2]s %[4]s %[3]s
                        }`,
						typ,
						a,
						b,
						operation.Symbol(),
					)

					invokable := parseCheckAndPrepare(t, code)

					result, err := invokable.Invoke("main")
					require.NoError(t, err)

					assert.Equal(
						ttt,
						expectedResult,
						result.String(),
					)
				})
			}
		}
	}

	t.Run("multiplication", func(t *testing.T) {
		t.Parallel()

		t.Run("truncate", func(t *testing.T) {
			t.Parallel()

			// Should truncate the least significant decimal point,
			// but doesn't underflow (truncated value is big enough to be representable).

			testCases := map[sema.Type][]testValue{
				sema.Fix64Type: {
					{
						a:      "45.6",
						b:      "0.00000001",
						result: "0.00000045",
					},
				},
				sema.UFix64Type: {
					{
						a:      "45.6",
						b:      "0.00000001",
						result: "0.00000045",
					},
				},
				sema.Fix128Type: {
					{
						a:      "45.6",
						b:      "0.000000000000000000000001",
						result: "0.000000000000000000000045",
					},
				},
				sema.UFix128Type: {
					{
						a:      "45.6",
						b:      "0.000000000000000000000001",
						result: "0.000000000000000000000045",
					},
				},
			}

			test(t, testCases, ast.OperationMul)
		})

		t.Run("truncate and underflow", func(t *testing.T) {
			t.Parallel()

			// Underflows - Truncated value is too small to represent.
			// Result must always be zero.

			testCases := map[sema.Type][]testValue{
				sema.Fix64Type: {
					{
						a:      "0.6",
						b:      "0.00000001",
						result: "0.00000000",
					},
				},
				sema.UFix64Type: {
					{
						a:      "0.6",
						b:      "0.00000001",
						result: "0.00000000",
					},
				},
				sema.Fix128Type: {
					{
						a:      "0.6",
						b:      "0.000000000000000000000001",
						result: "0.000000000000000000000000",
					},
				},
				sema.UFix128Type: {
					{
						a:      "0.6",
						b:      "0.000000000000000000000001",
						result: "0.000000000000000000000000",
					},
				},
			}

			test(t, testCases, ast.OperationMul)
		})
	})

	t.Run("division", func(t *testing.T) {
		t.Parallel()

		t.Run("truncate", func(t *testing.T) {
			t.Parallel()

			// Should truncate the least significant decimal point,
			// but doesn't underflow (truncated value is big enough to be representable).

			testCases := map[sema.Type][]testValue{
				sema.Fix64Type: {
					{
						a:      "0.00000456",
						b:      "10.0",
						result: "0.00000045",
					},
				},
				sema.UFix64Type: {
					{
						a:      "0.00000456",
						b:      "10.0",
						result: "0.00000045",
					},
				},
				sema.Fix128Type: {
					{
						a:      "0.000000000000000000000456",
						b:      "10.0",
						result: "0.000000000000000000000045",
					},
				},
				sema.UFix128Type: {
					{
						a:      "0.000000000000000000000456",
						b:      "10.0",
						result: "0.000000000000000000000045",
					},
				},
			}

			test(t, testCases, ast.OperationDiv)
		})

		t.Run("truncate and underflow", func(t *testing.T) {
			t.Parallel()

			// Underflows - Truncated value is too small to represent.
			// Result must always be zero.

			testCases := map[sema.Type][]testValue{
				sema.Fix64Type: {
					{
						a:      "0.00000006",
						b:      "10.0",
						result: "0.00000000",
					},
				},
				sema.UFix64Type: {
					{
						a:      "0.00000006",
						b:      "10.0",
						result: "0.00000000",
					},
				},
				sema.Fix128Type: {
					{
						a:      "0.000000000000000000000006",
						b:      "10.0",
						result: "0.000000000000000000000000",
					},
				},
				sema.UFix128Type: {
					{
						a:      "0.000000000000000000000006",
						b:      "10.0",
						result: "0.000000000000000000000000",
					},
				},
			}

			test(t, testCases, ast.OperationDiv)
		})
	})
}
