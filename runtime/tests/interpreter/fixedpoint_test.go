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

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/fixedpoint"
	"github.com/onflow/cadence/runtime/tests/utils"
	. "github.com/onflow/cadence/runtime/tests/utils"

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

func TestInterpretNegativeZeroFixedPoint(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = -0.42
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredFix64Value(-42000000),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretFixedPointConversionAndAddition(t *testing.T) {

	t.Parallel()

	tests := map[string]interpreter.Value{
		// Fix*
		"Fix64": interpreter.NewUnmeteredFix64Value(123000000),
		// UFix*
		"UFix64": interpreter.NewUnmeteredUFix64Value(123000000),
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

			inter := parseCheckAndInterpret(t,
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

var testFixedPointValues = map[string]interpreter.Value{
	"Fix64":  interpreter.NewUnmeteredFix64Value(50 * sema.Fix64Factor),
	"UFix64": interpreter.NewUnmeteredUFix64Value(50 * sema.Fix64Factor),
}

func init() {
	for _, fixedPointType := range sema.AllFixedPointTypes {
		// Only test leaf types
		switch fixedPointType {
		case sema.FixedPointType, sema.SignedFixedPointType:
			continue
		}

		if _, ok := testFixedPointValues[fixedPointType.String()]; !ok {
			panic(fmt.Sprintf("broken test: missing fixed-point type: %s", fixedPointType))
		}
	}
}

func TestInterpretFixedPointConversions(t *testing.T) {

	t.Parallel()

	// check conversion to integer types

	for fixedPointType, fixedPointValue := range testFixedPointValues {

		for integerType, integerValue := range testIntegerTypesAndValues {

			testName := fmt.Sprintf("valid %s to %s", fixedPointType, integerType)

			t.Run(testName, func(t *testing.T) {

				inter := parseCheckAndInterpret(t,
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
					inter.Globals["x"].GetValue(),
				)

				AssertValuesEqual(
					t,
					inter,
					integerValue,
					inter.Globals["y"].GetValue(),
				)
			})
		}
	}

	t.Run("valid UFix64 to UFix64", func(t *testing.T) {

		for _, value := range []uint64{
			50,
			sema.UFix64TypeMinInt,
			sema.UFix64TypeMaxInt,
		} {

			t.Run(fmt.Sprint(value), func(t *testing.T) {

				code := fmt.Sprintf(
					`
                          let x: UFix64 = %d.0
                          let y = UFix64(x)
                        `,
					value,
				)

				inter := parseCheckAndInterpret(t, code)

				expected := interpreter.NewUnmeteredUFix64Value(value * sema.Fix64Factor)

				AssertValuesEqual(
					t,
					inter,
					expected,
					inter.Globals["x"].GetValue(),
				)

				AssertValuesEqual(
					t,
					inter,
					expected,
					inter.Globals["y"].GetValue(),
				)
			})
		}
	})

	t.Run("valid Fix64 to Fix64", func(t *testing.T) {

		for _, value := range []int64{
			-50,
			50,
			sema.Fix64TypeMinInt,
			sema.Fix64TypeMaxInt,
		} {

			t.Run(fmt.Sprint(value), func(t *testing.T) {

				code := fmt.Sprintf(
					`
                          let x: Fix64 = %d.0
                          let y = Fix64(x)
                        `,
					value,
				)

				inter := parseCheckAndInterpret(t, code)

				expected := interpreter.NewUnmeteredFix64Value(value * sema.Fix64Factor)

				AssertValuesEqual(
					t,
					inter,
					expected,
					inter.Globals["x"].GetValue(),
				)

				AssertValuesEqual(
					t,
					inter,
					expected,
					inter.Globals["y"].GetValue(),
				)
			})
		}
	})

	t.Run("valid Fix64 to UFix64", func(t *testing.T) {

		for _, value := range []int64{0, 50, sema.Fix64TypeMaxInt} {

			t.Run(fmt.Sprint(value), func(t *testing.T) {

				code := fmt.Sprintf(
					`
                          let x: Fix64 = %d.0
                          let y = UFix64(x)
                        `,
					value,
				)

				inter := parseCheckAndInterpret(t, code)

				AssertValuesEqual(
					t,
					inter,
					interpreter.NewUnmeteredFix64Value(value*sema.Fix64Factor),
					inter.Globals["x"].GetValue(),
				)

				AssertValuesEqual(
					t,
					inter,
					interpreter.NewUnmeteredUFix64Value(uint64(value*sema.Fix64Factor)),
					inter.Globals["y"].GetValue(),
				)
			})
		}
	})

	t.Run("valid UFix64 to Fix64", func(t *testing.T) {

		for _, value := range []int64{0, 50, sema.Fix64TypeMaxInt} {

			t.Run(fmt.Sprint(value), func(t *testing.T) {

				code := fmt.Sprintf(
					`
                          let x: UFix64 = %d.0
                          let y = Fix64(x)
                        `,
					value,
				)

				inter := parseCheckAndInterpret(t, code)

				AssertValuesEqual(
					t,
					inter,
					interpreter.NewUnmeteredUFix64Value(uint64(value*sema.Fix64Factor)),
					inter.Globals["x"].GetValue(),
				)

				AssertValuesEqual(
					t,
					inter,
					interpreter.NewUnmeteredFix64Value(value*sema.Fix64Factor),
					inter.Globals["y"].GetValue(),
				)
			})
		}
	})

	t.Run("invalid negative Fix64 to UFix64", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
		  fun test(): UFix64 {
		      let x: Fix64 = -1.0
		      return UFix64(x)
		  }
		`)

		_, err := inter.Invoke("test")
		require.Error(t, err)
		_ = err.Error()

		require.ErrorAs(t, err, &interpreter.UnderflowError{})
	})

	t.Run("invalid UFix64 > max Fix64 int to Fix64", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			fmt.Sprintf(
				`
		          fun test(): Fix64 {
		              let x: UFix64 = %d.0
		              return Fix64(x)
		          }
		        `,
				sema.Fix64TypeMaxInt+1,
			),
		)

		_, err := inter.Invoke("test")
		require.Error(t, err)
		_ = err.Error()

		require.ErrorAs(t, err, &interpreter.OverflowError{})
	})

	t.Run("invalid negative integer to UFix64", func(t *testing.T) {

		for _, integerType := range sema.AllSignedIntegerTypes {

			t.Run(integerType.String(), func(t *testing.T) {

				inter := parseCheckAndInterpret(t,
					fmt.Sprintf(
						`
	                     fun test(): UFix64 {
	                         let x: %s = -1
	                         return UFix64(x)
	                     }
	                   `,
						integerType,
					),
				)

				_, err := inter.Invoke("test")
				require.Error(t, err)
				_ = err.Error()

				require.IsType(t,
					interpreter.Error{},
					err,
				)

				require.ErrorAs(t, err, &interpreter.UnderflowError{})
			})
		}
	})

	t.Run("invalid big integer (>uint64) to UFix64", func(t *testing.T) {

		bigIntegerTypes := []sema.Type{
			sema.Word64Type,
			sema.UInt64Type,
			sema.UInt128Type,
			sema.UInt256Type,
			sema.Int256Type,
			sema.Int128Type,
		}

		for _, integerType := range bigIntegerTypes {

			t.Run(integerType.String(), func(t *testing.T) {

				inter := parseCheckAndInterpret(t,
					fmt.Sprintf(
						`
	                     fun test(): UFix64 {
	                         let x: %s = %d
	                         return UFix64(x)
	                     }
	                   `,
						integerType,
						sema.UFix64TypeMaxInt+1,
					),
				)

				_, err := inter.Invoke("test")
				require.Error(t, err)
				_ = err.Error()

				require.ErrorAs(t, err, &interpreter.OverflowError{})
			})
		}
	})

	t.Run("invalid integer > UFix64 max int to UFix64", func(t *testing.T) {

		const testedValue = sema.UFix64TypeMaxInt + 1
		testValueBig := big.NewInt(0).SetUint64(testedValue)

		for _, integerType := range sema.AllIntegerTypes {

			// Only test for integer types that can hold testedValue

			maxInt := integerType.(sema.IntegerRangedType).MaxInt()
			if maxInt != nil && maxInt.Cmp(testValueBig) < 0 {
				continue
			}

			t.Run(integerType.String(), func(t *testing.T) {

				inter := parseCheckAndInterpret(t,
					fmt.Sprintf(
						`
	                     fun test(): UFix64 {
	                         let x: %s = %d
	                         return UFix64(x)
	                     }
	                   `,
						integerType,
						testedValue,
					),
				)

				_, err := inter.Invoke("test")
				require.Error(t, err)
				_ = err.Error()

				require.ErrorAs(t, err, &interpreter.OverflowError{})
			})
		}
	})

	t.Run("invalid integer > Fix64 max int to Fix64", func(t *testing.T) {

		const testedValue = sema.Fix64TypeMaxInt + 1
		testValueBig := big.NewInt(0).SetUint64(testedValue)

		for _, integerType := range sema.AllIntegerTypes {

			// Only test for integer types that can hold testedValue

			maxInt := integerType.(sema.IntegerRangedType).MaxInt()
			if maxInt != nil && maxInt.Cmp(testValueBig) < 0 {
				continue
			}

			t.Run(integerType.String(), func(t *testing.T) {

				inter := parseCheckAndInterpret(t,
					fmt.Sprintf(
						`
	                     fun test(): Fix64 {
	                         let x: %s = %d
	                         return Fix64(x)
	                     }
	                   `,
						integerType,
						testedValue,
					),
				)

				_, err := inter.Invoke("test")
				require.Error(t, err)
				_ = err.Error()

				require.ErrorAs(t, err, &interpreter.OverflowError{})
			})
		}
	})

	t.Run("invalid integer < Fix64 min int to Fix64", func(t *testing.T) {

		const testedValue = sema.Fix64TypeMinInt - 1
		testValueBig := big.NewInt(testedValue)

		for _, integerType := range sema.AllSignedIntegerTypes {

			// Only test for integer types that can hold testedValue

			minInt := integerType.(sema.IntegerRangedType).MinInt()
			if minInt != nil && minInt.Cmp(testValueBig) > 0 {
				continue
			}

			t.Run(integerType.String(), func(t *testing.T) {

				inter := parseCheckAndInterpret(t,
					fmt.Sprintf(
						`
	                     fun test(): Fix64 {
	                         let x: %s = %d
	                         return Fix64(x)
	                     }
	                   `,
						integerType,
						testedValue,
					),
				)

				_, err := inter.Invoke("test")
				require.Error(t, err)
				_ = err.Error()

				require.ErrorAs(t, err, &interpreter.UnderflowError{})
			})
		}
	})
}

func TestInterpretFixedPointMinMax(t *testing.T) {

	t.Parallel()

	type testCase struct {
		min interpreter.Value
		max interpreter.Value
	}

	test := func(t *testing.T, ty sema.Type, test testCase) {

		inter := parseCheckAndInterpret(t,
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
			inter.Globals["min"].GetValue(),
		)
		RequireValuesEqual(
			t,
			inter,
			test.max,
			inter.Globals["max"].GetValue(),
		)
	}

	testCases := map[sema.Type]testCase{
		sema.Fix64Type: {
			min: interpreter.NewUnmeteredFix64Value(math.MinInt64),
			max: interpreter.NewUnmeteredFix64Value(math.MaxInt64),
		},
		sema.UFix64Type: {
			min: interpreter.NewUnmeteredUFix64Value(0),
			max: interpreter.NewUnmeteredUFix64Value(math.MaxUint64),
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

func TestStringFixedpointConversion(t *testing.T) {
	t.Parallel()

	type testcase struct {
		decimal    *big.Int
		fractional *big.Int
	}

	type testsuite struct {
		name         string
		toFixedValue func(isNegative bool, decimal, fractional *big.Int, scale uint) (interpreter.Value, error)
		intBounds    []*big.Int
		fracBounds   []*big.Int
	}

	bigZero := big.NewInt(0)
	bigOne := big.NewInt(1)

	suites := []testsuite{
		{
			"Fix64",
			func(isNeg bool, decimal, fractional *big.Int, scale uint) (interpreter.Value, error) {
				fixedVal, err := fixedpoint.NewFix64(isNeg, decimal, fractional, scale)
				if err != nil {
					return nil, err
				}
				return interpreter.NewUnmeteredFix64Value(fixedVal.Int64()), nil
			},
			[]*big.Int{sema.Fix64TypeMinIntBig, sema.Fix64TypeMaxIntBig, bigZero, bigOne},
			[]*big.Int{sema.UFix64TypeMinFractionalBig, sema.Fix64TypeMaxFractionalBig, bigZero, bigOne},
		},
		{
			"UFix64",
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

	test := func(suite testsuite) {
		t.Run(suite.name, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf(`
				fun fromStringTest(s: String): %s? {
					return %s.fromString(s)
				}
			`, suite.name, suite.name)

			inter := parseCheckAndInterpret(t, code)

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

				utils.AssertEqualWithDiff(t, expectedVal, res)
			}

		})

	}
	for _, testsuite := range suites {
		test(testsuite)
	}

}
