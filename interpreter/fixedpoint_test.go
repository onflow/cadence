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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/fixedpoint"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func fix128BigInt(s string) *big.Int {
	// Parse a decimal string like "33.333333333333333333333333"
	// into the raw scaled big.Int (removing the decimal point).
	// The fractional part must have exactly Fix128Scale (24) digits.
	parts := strings.SplitN(s, ".", 2)
	if len(parts) == 1 {
		// No decimal point — treat as integer, scale up
		v, ok := new(big.Int).SetString(s, 10)
		if !ok {
			panic("invalid fix128 string: " + s)
		}
		return v.Mul(v, sema.Fix128FactorIntBig)
	}
	if len(parts[1]) != sema.Fix128Scale {
		panic(fmt.Sprintf("expected %d fractional digits, got %d: %s", sema.Fix128Scale, len(parts[1]), s))
	}
	raw := parts[0] + parts[1]
	v, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		panic("invalid fix128 string: " + s)
	}
	return v
}

func TestInterpretFixedPointMultiplyDivide(t *testing.T) {

	t.Parallel()

	t.Run("UFix64", func(t *testing.T) {

		t.Parallel()

		type testCase struct {
			a, b, c       string
			rounding      string
			expected      uint64
			expectedError bool
		}

		// Expected values pre-computed using the fixed-point library's UFix64.FMD().
		testCases := []testCase{
			// Basic: 2*3/1 = 6
			{a: "2.00000000", b: "3.00000000", c: "1.00000000", rounding: "towardZero", expected: 600000000},
			// Rounding modes: 10*10/3
			{a: "10.00000000", b: "10.00000000", c: "3.00000000", rounding: "towardZero", expected: 3333333333},
			{a: "10.00000000", b: "10.00000000", c: "3.00000000", rounding: "awayFromZero", expected: 3333333334},
			{a: "10.00000000", b: "10.00000000", c: "3.00000000", rounding: "nearestHalfAway", expected: 3333333333},
			// Fractional: 0.5*0.5/1.0 = 0.25
			{a: "0.50000000", b: "0.50000000", c: "1.00000000", rounding: "towardZero", expected: 25000000},
			// Zero factor
			{a: "0.00000000", b: "5.00000000", c: "2.00000000", rounding: "towardZero", expected: 0},
			{a: "5.00000000", b: "0.00000000", c: "2.00000000", rounding: "towardZero", expected: 0},
			// Larger values: 100*200/50 = 400
			{a: "100.00000000", b: "200.00000000", c: "50.00000000", rounding: "towardZero", expected: 40000000000},
			// 1*1/3 with different rounding
			{a: "1.00000000", b: "1.00000000", c: "3.00000000", rounding: "towardZero", expected: 33333333},
			{a: "1.00000000", b: "1.00000000", c: "3.00000000", rounding: "awayFromZero", expected: 33333334},
			// Division by zero
			{a: "1.00000000", b: "2.00000000", c: "0.00000000", rounding: "towardZero", expectedError: true},
		}

		for _, tc := range testCases {

			testName := fmt.Sprintf("%s * %s / %s (%s)", tc.a, tc.b, tc.c, tc.rounding)

			t.Run(testName, func(t *testing.T) {
				t.Parallel()

				code := fmt.Sprintf(
					`
					fun test(): UFix64 {
						let a: UFix64 = %s
						let b: UFix64 = %s
						let c: UFix64 = %s
						return a.multiplyDivide(b, c, rounding: RoundingRule.%s)
					}
					`,
					tc.a, tc.b, tc.c, tc.rounding,
				)

				inter := parseCheckAndPrepareWithRoundingRule(t, code)

				if tc.expectedError {
					_, err := inter.Invoke("test")
					require.Error(t, err)
				} else {
					result, err := inter.Invoke("test")
					require.NoError(t, err)

					expected := interpreter.NewUnmeteredUFix64Value(tc.expected)
					AssertValuesEqual(t, inter, expected, result)
				}
			})
		}
	})

	t.Run("Fix64", func(t *testing.T) {

		t.Parallel()

		type testCase struct {
			a, b, c       string
			rounding      string
			expected      int64
			expectedError bool
		}

		testCases := []testCase{
			// Basic: 2*3/1 = 6
			{a: "2.00000000", b: "3.00000000", c: "1.00000000", rounding: "towardZero", expected: 600000000},
			// Signed: (-2)*3/1 = -6
			{a: "-2.00000000", b: "3.00000000", c: "1.00000000", rounding: "towardZero", expected: -600000000},
			// (-5)*(-3)/2 = 7.5
			{a: "-5.00000000", b: "-3.00000000", c: "2.00000000", rounding: "towardZero", expected: 750000000},
			// Rounding: 10*10/3
			{a: "10.00000000", b: "10.00000000", c: "3.00000000", rounding: "towardZero", expected: 3333333333},
			{a: "10.00000000", b: "10.00000000", c: "3.00000000", rounding: "awayFromZero", expected: 3333333334},
			// 1/3 truncated
			{a: "1.00000000", b: "1.00000000", c: "3.00000000", rounding: "towardZero", expected: 33333333},
			// Division by zero
			{a: "1.00000000", b: "2.00000000", c: "0.00000000", rounding: "towardZero", expectedError: true},
		}

		for _, tc := range testCases {

			testName := fmt.Sprintf("%s * %s / %s (%s)", tc.a, tc.b, tc.c, tc.rounding)

			t.Run(testName, func(t *testing.T) {
				t.Parallel()

				code := fmt.Sprintf(
					`
					fun test(): Fix64 {
						let a: Fix64 = %s
						let b: Fix64 = %s
						let c: Fix64 = %s
						return a.multiplyDivide(b, c, rounding: RoundingRule.%s)
					}
					`,
					tc.a, tc.b, tc.c, tc.rounding,
				)

				inter := parseCheckAndPrepareWithRoundingRule(t, code)

				if tc.expectedError {
					_, err := inter.Invoke("test")
					require.Error(t, err)
				} else {
					result, err := inter.Invoke("test")
					require.NoError(t, err)

					expected := interpreter.NewUnmeteredFix64Value(tc.expected)
					AssertValuesEqual(t, inter, expected, result)
				}
			})
		}
	})

	t.Run("UFix128", func(t *testing.T) {

		t.Parallel()

		type testCase struct {
			a, b, c       string
			rounding      string
			expected      string
			expectedError bool
		}

		testCases := []testCase{
			{a: "2.000000000000000000000000", b: "3.000000000000000000000000", c: "1.000000000000000000000000", rounding: "towardZero", expected: "6.000000000000000000000000"},
			{a: "10.000000000000000000000000", b: "10.000000000000000000000000", c: "3.000000000000000000000000", rounding: "towardZero", expected: "33.333333333333333333333333"},
			{a: "10.000000000000000000000000", b: "10.000000000000000000000000", c: "3.000000000000000000000000", rounding: "awayFromZero", expected: "33.333333333333333333333334"},
			{a: "0.500000000000000000000000", b: "0.500000000000000000000000", c: "1.000000000000000000000000", rounding: "towardZero", expected: "0.250000000000000000000000"},
			{a: "100.000000000000000000000000", b: "200.000000000000000000000000", c: "50.000000000000000000000000", rounding: "towardZero", expected: "400.000000000000000000000000"},
			// Division by zero
			{a: "1.000000000000000000000000", b: "2.000000000000000000000000", c: "0.000000000000000000000000", rounding: "towardZero", expectedError: true},
		}

		for _, tc := range testCases {

			testName := fmt.Sprintf("%s * %s / %s (%s)", tc.a, tc.b, tc.c, tc.rounding)

			t.Run(testName, func(t *testing.T) {
				t.Parallel()

				code := fmt.Sprintf(
					`
					fun test(): UFix128 {
						let a: UFix128 = %s
						let b: UFix128 = %s
						let c: UFix128 = %s
						return a.multiplyDivide(b, c, rounding: RoundingRule.%s)
					}
					`,
					tc.a, tc.b, tc.c, tc.rounding,
				)

				inter := parseCheckAndPrepareWithRoundingRule(t, code)

				if tc.expectedError {
					_, err := inter.Invoke("test")
					require.Error(t, err)
				} else {
					result, err := inter.Invoke("test")
					require.NoError(t, err)

					expected := interpreter.NewUFix128ValueFromBigInt(nil, fix128BigInt(tc.expected))
					AssertValuesEqual(t, inter, expected, result)
				}
			})
		}
	})

	t.Run("Fix128", func(t *testing.T) {

		t.Parallel()

		type testCase struct {
			a, b, c       string
			rounding      string
			expected      string
			expectedError bool
		}

		testCases := []testCase{
			{a: "2.000000000000000000000000", b: "3.000000000000000000000000", c: "1.000000000000000000000000", rounding: "towardZero", expected: "6.000000000000000000000000"},
			{a: "-2.000000000000000000000000", b: "3.000000000000000000000000", c: "1.000000000000000000000000", rounding: "towardZero", expected: "-6.000000000000000000000000"},
			{a: "-2.000000000000000000000000", b: "-3.000000000000000000000000", c: "1.000000000000000000000000", rounding: "towardZero", expected: "6.000000000000000000000000"},
			{a: "10.000000000000000000000000", b: "10.000000000000000000000000", c: "3.000000000000000000000000", rounding: "towardZero", expected: "33.333333333333333333333333"},
			{a: "10.000000000000000000000000", b: "10.000000000000000000000000", c: "3.000000000000000000000000", rounding: "awayFromZero", expected: "33.333333333333333333333334"},
			// Division by zero
			{a: "1.000000000000000000000000", b: "2.000000000000000000000000", c: "0.000000000000000000000000", rounding: "towardZero", expectedError: true},
		}

		for _, tc := range testCases {

			testName := fmt.Sprintf("%s * %s / %s (%s)", tc.a, tc.b, tc.c, tc.rounding)

			t.Run(testName, func(t *testing.T) {
				t.Parallel()

				code := fmt.Sprintf(
					`
					fun test(): Fix128 {
						let a: Fix128 = %s
						let b: Fix128 = %s
						let c: Fix128 = %s
						return a.multiplyDivide(b, c, rounding: RoundingRule.%s)
					}
					`,
					tc.a, tc.b, tc.c, tc.rounding,
				)

				inter := parseCheckAndPrepareWithRoundingRule(t, code)

				if tc.expectedError {
					_, err := inter.Invoke("test")
					require.Error(t, err)
				} else {
					result, err := inter.Invoke("test")
					require.NoError(t, err)

					expected := interpreter.NewFix128ValueFromBigInt(nil, fix128BigInt(tc.expected))
					AssertValuesEqual(t, inter, expected, result)
				}
			})
		}
	})

	t.Run("default rounding (truncate)", func(t *testing.T) {

		t.Parallel()

		t.Run("UFix64", func(t *testing.T) {
			t.Parallel()

			inter := parseCheckAndPrepareWithRoundingRule(t, `
				fun test(): UFix64 {
					let a: UFix64 = 10.0
					let b: UFix64 = 10.0
					let c: UFix64 = 3.0
					return a.multiplyDivide(b, c)
				}
			`)
			result, err := inter.Invoke("test")
			require.NoError(t, err)

			// 10*10/3 truncated = 33.33333333
			expected := interpreter.NewUnmeteredUFix64Value(3333333333)
			AssertValuesEqual(t, inter, expected, result)
		})

		t.Run("Fix64", func(t *testing.T) {
			t.Parallel()

			inter := parseCheckAndPrepareWithRoundingRule(t, `
				fun test(): Fix64 {
					let a: Fix64 = 10.0
					let b: Fix64 = 10.0
					let c: Fix64 = 3.0
					return a.multiplyDivide(b, c)
				}
			`)
			result, err := inter.Invoke("test")
			require.NoError(t, err)

			// 10*10/3 truncated = 33.33333333
			expected := interpreter.NewUnmeteredFix64Value(3333333333)
			AssertValuesEqual(t, inter, expected, result)
		})

		t.Run("UFix128", func(t *testing.T) {
			t.Parallel()

			inter := parseCheckAndPrepareWithRoundingRule(t, `
				fun test(): UFix128 {
					let a: UFix128 = 10.0
					let b: UFix128 = 10.0
					let c: UFix128 = 3.0
					return a.multiplyDivide(b, c)
				}
			`)
			result, err := inter.Invoke("test")
			require.NoError(t, err)

			// 10*10/3 truncated = 33.333333333333333333333333
			expected := interpreter.NewUFix128ValueFromBigInt(nil, fix128BigInt("33.333333333333333333333333"))
			AssertValuesEqual(t, inter, expected, result)
		})

		t.Run("Fix128", func(t *testing.T) {
			t.Parallel()

			inter := parseCheckAndPrepareWithRoundingRule(t, `
				fun test(): Fix128 {
					let a: Fix128 = 10.0
					let b: Fix128 = 10.0
					let c: Fix128 = 3.0
					return a.multiplyDivide(b, c)
				}
			`)
			result, err := inter.Invoke("test")
			require.NoError(t, err)

			// 10*10/3 truncated = 33.333333333333333333333333
			expected := interpreter.NewFix128ValueFromBigInt(nil, fix128BigInt("33.333333333333333333333333"))
			AssertValuesEqual(t, inter, expected, result)
		})
	})
}

func TestInterpretFixedPointPow(t *testing.T) {

	t.Parallel()

	t.Run("UFix64", func(t *testing.T) {

		t.Parallel()

		type testCase struct {
			base          string
			exponent      string
			expected      uint64
			expectedError bool
		}

		// Expected values were pre-computed using the fixed-point library's UFix64.Pow(Fix64).
		testCases := []testCase{
			// Edge cases
			{base: "0.00000000", exponent: "0.00000000", expected: 100000000}, // 0^0 = 1
			{base: "0.00000000", exponent: "2.00000000", expected: 0},         // 0^2 = 0
			{base: "1.00000000", exponent: "0.00000000", expected: 100000000}, // 1^0 = 1
			{base: "1.00000000", exponent: "5.00000000", expected: 100000000}, // 1^5 = 1
			{base: "2.00000000", exponent: "0.00000000", expected: 100000000}, // 2^0 = 1
			{base: "2.00000000", exponent: "1.00000000", expected: 200000000}, // 2^1 = 2

			// Integer exponents
			{base: "2.00000000", exponent: "3.00000000", expected: 800000000},     // 2^3 = 8
			{base: "5.00000000", exponent: "2.00000000", expected: 2500000000},    // 5^2 = 25
			{base: "10.00000000", exponent: "3.00000000", expected: 100000000000}, // 10^3 = 1000

			// Negative exponents
			{base: "2.00000000", exponent: "-1.00000000", expected: 50000000}, // 2^(-1) = 0.5
			{base: "4.00000000", exponent: "-1.00000000", expected: 25000000}, // 4^(-1) = 0.25
			{base: "10.00000000", exponent: "-2.00000000", expected: 1000000}, // 10^(-2) = 0.01

			// Fractional bases
			{base: "0.50000000", exponent: "2.00000000", expected: 25000000},  // 0.5^2 = 0.25
			{base: "1.50000000", exponent: "2.00000000", expected: 225000000}, // 1.5^2 = 2.25
			{base: "0.25000000", exponent: "3.00000000", expected: 1562500},   // 0.25^3 = 0.015625

			// Fractional exponents
			{base: "4.00000000", exponent: "0.50000000", expected: 200000000}, // 4^0.5 = 2
			{base: "9.00000000", exponent: "0.50000000", expected: 300000000}, // 9^0.5 = 3
			{base: "8.00000000", exponent: "0.33333333", expected: 199999999}, // 8^(1/3) ≈ 2

			// Values from library test data
			{base: "0.11111111", exponent: "2.00000000", expected: 1234568},      // (1/9)^2
			{base: "0.33333333", exponent: "3.00000000", expected: 3703704},      // (1/3)^3
			{base: "2.71828183", exponent: "1.00000000", expected: 271828183},    // e^1
			{base: "3.14159265", exponent: "-0.50000000", expected: 56418958},    // pi^(-0.5)
			{base: "0.14285714", exponent: "2.00000000", expected: 2040816},      // (1/7)^2
			{base: "123.45678901", exponent: "0.50000000", expected: 1111111106}, // 123.45678901^0.5

			// Repeating decimal bases with negative exponents
			{base: "0.66666666", exponent: "-1.00000000", expected: 150000002}, // (2/3)^(-1)
			{base: "0.50000000", exponent: "-2.00000000", expected: 400000000}, // 0.5^(-2) = 4

			// Overflow
			{base: "429496.72960000", exponent: "2.00000000", expectedError: true}, // sqrt(MaxUFix64)^2 overflows
			{base: "10.00000000", exponent: "20.00000000", expectedError: true},    // 10^20 overflows

			// Underflow (truncated to 0 by handleFixedpointError)
			{base: "0.00000003", exponent: "2.00000000", expected: 0}, // 0.00000003^2 underflows to 0
		}

		for _, tc := range testCases {

			testName := fmt.Sprintf("%s ^ %s", tc.base, tc.exponent)

			t.Run(testName, func(t *testing.T) {
				t.Parallel()

				code := fmt.Sprintf(
					`
					fun test(): UFix64 {
						let base: UFix64 = %s
						let exponent: Fix64 = %s
						return base.pow(exponent)
					}
					`,
					tc.base,
					tc.exponent,
				)

				inter := parseCheckAndPrepare(t, code)

				if tc.expectedError {
					_, err := inter.Invoke("test")
					require.Error(t, err)
				} else {
					result, err := inter.Invoke("test")
					require.NoError(t, err)

					expected := interpreter.NewUnmeteredUFix64Value(tc.expected)
					AssertValuesEqual(t, inter, expected, result)
				}
			})
		}
	})

	t.Run("UFix128", func(t *testing.T) {

		t.Parallel()

		type testCase struct {
			base          string
			exponent      string
			expected      string
			expectedError bool
		}

		// Expected values were pre-computed using the fixed-point library's UFix128.Pow(Fix128).
		testCases := []testCase{
			// Edge cases
			{base: "0.000000000000000000000000", exponent: "0.000000000000000000000000", expected: "1.000000000000000000000000"}, // 0^0 = 1
			{base: "1.000000000000000000000000", exponent: "0.000000000000000000000000", expected: "1.000000000000000000000000"}, // 1^0 = 1
			{base: "1.000000000000000000000000", exponent: "5.000000000000000000000000", expected: "1.000000000000000000000000"}, // 1^5 = 1
			{base: "2.000000000000000000000000", exponent: "0.000000000000000000000000", expected: "1.000000000000000000000000"}, // 2^0 = 1
			{base: "2.000000000000000000000000", exponent: "1.000000000000000000000000", expected: "2.000000000000000000000000"}, // 2^1 = 2

			// Integer exponents
			{base: "2.000000000000000000000000", exponent: "3.000000000000000000000000", expected: "8.000000000000000000000000"},     // 2^3 = 8
			{base: "5.000000000000000000000000", exponent: "2.000000000000000000000000", expected: "25.000000000000000000000000"},    // 5^2 = 25
			{base: "10.000000000000000000000000", exponent: "3.000000000000000000000000", expected: "1000.000000000000000000000000"}, // 10^3 = 1000

			// Negative exponents
			{base: "2.000000000000000000000000", exponent: "-1.000000000000000000000000", expected: "0.500000000000000000000000"}, // 2^(-1) = 0.5
			{base: "4.000000000000000000000000", exponent: "-1.000000000000000000000000", expected: "0.250000000000000000000000"}, // 4^(-1) = 0.25

			// Fractional base
			{base: "0.500000000000000000000000", exponent: "2.000000000000000000000000", expected: "0.250000000000000000000000"}, // 0.5^2 = 0.25

			// Fractional exponent
			{base: "4.000000000000000000000000", exponent: "0.500000000000000000000000", expected: "2.000000000000000000000000"}, // 4^0.5 = 2
			{base: "9.000000000000000000000000", exponent: "0.500000000000000000000000", expected: "3.000000000000000000000000"}, // 9^0.5 = 3
		}

		for _, tc := range testCases {

			testName := fmt.Sprintf("%s ^ %s", tc.base, tc.exponent)

			t.Run(testName, func(t *testing.T) {
				t.Parallel()

				code := fmt.Sprintf(
					`
					fun test(): UFix128 {
						let base: UFix128 = %s
						let exponent: Fix128 = %s
						return base.pow(exponent)
					}
					`,
					tc.base,
					tc.exponent,
				)

				inter := parseCheckAndPrepare(t, code)

				if tc.expectedError {
					_, err := inter.Invoke("test")
					require.Error(t, err)
				} else {
					result, err := inter.Invoke("test")
					require.NoError(t, err)

					expected := interpreter.NewUFix128ValueFromBigInt(nil, fix128BigInt(tc.expected))
					AssertValuesEqual(t, inter, expected, result)
				}
			})
		}
	})
}

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
	sema.Fix128Type: interpreter.NewUnmeteredFix128ValueWithInteger(50),

	// UFix types
	sema.UFix64Type:  interpreter.NewUnmeteredUFix64Value(50 * sema.Fix64Factor),
	sema.UFix128Type: interpreter.NewUnmeteredUFix128ValueWithInteger(50),
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
			positiveValue: interpreter.NewUnmeteredFix64ValueWithInteger(42),
			negativeValue: interpreter.NewUnmeteredFix64ValueWithInteger(-42),

			// Use the max(Fix64) as the large positive value
			largePositiveValue: interpreter.NewUnmeteredFix64Value(math.MaxInt64),
			// Use the min(Fix64) as the large negative value
			largeNegativeValue: interpreter.NewUnmeteredFix64Value(math.MinInt64),

			min: interpreter.NewUnmeteredFix64ValueWithInteger(sema.Fix64TypeMinInt),
			max: interpreter.NewUnmeteredFix64ValueWithInteger(sema.Fix64TypeMaxInt),
		},
		sema.Fix128Type: {
			positiveValue: interpreter.NewUnmeteredFix128ValueWithInteger(42),
			negativeValue: interpreter.NewUnmeteredFix128ValueWithInteger(-42),

			// Use the max(Fix64) as the large positive value
			largePositiveValue: interpreter.NewFix128ValueFromBigInt(nil, fixedpoint.Fix64TypeMaxScaledTo128),
			// Use the min(Fix64) as the large negative value
			largeNegativeValue: interpreter.NewFix128ValueFromBigInt(nil, fixedpoint.Fix64TypeMinScaledTo128),

			min: interpreter.NewUnmeteredFix128Value(fixedpoint.Fix128TypeMin),
			max: interpreter.NewUnmeteredFix128Value(fixedpoint.Fix128TypeMax),
		},

		// UFix types
		sema.UFix64Type: {
			positiveValue: interpreter.NewUnmeteredUFix64ValueWithInteger(42),

			// Use the max(Fix64) as the large positive value
			largePositiveValue: interpreter.NewUnmeteredUFix64Value(math.MaxInt64),

			min: interpreter.NewUnmeteredUFix64ValueWithInteger(sema.UFix64TypeMinInt),
			max: interpreter.NewUnmeteredUFix64ValueWithInteger(sema.UFix64TypeMaxInt),
		},
		sema.UFix128Type: {
			positiveValue: interpreter.NewUnmeteredUFix128ValueWithInteger(42),

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
		operation ast.Operation
		a, b      interpreter.FixedPointValue
		result    interpreter.FixedPointValue
	}

	test := func(tt *testing.T, typ sema.Type, a, b, expectedResult interpreter.Value, operation ast.Operation) {
		testName := fmt.Sprintf("%s (%s%s%s)",
			typ,
			a,
			operation.Symbol(),
			b,
		)

		tt.Run(testName, func(ttt *testing.T) {
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

			invokable := parseCheckAndPrepare(ttt, code)

			result, err := invokable.Invoke("main")
			require.NoError(t, err)

			assert.Equal(
				ttt,
				expectedResult,
				result,
			)
		})
	}

	t.Run("unsigned", func(t *testing.T) {
		t.Parallel()

		testCases := map[sema.Type][]testValue{
			sema.UFix64Type: {
				{
					// Multiplication truncates.
					// 45.6 * 1e-8 =  0.00000045
					a:         interpreter.NewUnmeteredUFix64Value(4560000000),
					b:         interpreter.NewUnmeteredUFix64Value(1),
					operation: ast.OperationMul,
					result:    interpreter.NewUnmeteredUFix64Value(45),
				},
				{
					// Multiplication truncates and underflows.
					// 0.6 * 1e-8 =  0.00000000
					a:         interpreter.NewUnmeteredUFix64Value(60000000),
					b:         interpreter.NewUnmeteredUFix64Value(1),
					operation: ast.OperationMul,
					result:    interpreter.NewUnmeteredUFix64Value(0),
				},
				{
					// Division truncates.
					// 0.00000456 / 10 =  0.00000045
					a:         interpreter.NewUnmeteredUFix64Value(456),
					b:         interpreter.NewUnmeteredUFix64Value(1000000000),
					operation: ast.OperationDiv,
					result:    interpreter.NewUnmeteredUFix64Value(45),
				},
				{
					// Division truncates and underflows.
					// 0.00000006 / 10 =  0.00000000
					a:         interpreter.NewUnmeteredUFix64Value(6),
					b:         interpreter.NewUnmeteredUFix64Value(1000000000),
					operation: ast.OperationDiv,
					result:    interpreter.NewUnmeteredUFix64Value(0),
				},
			},

			sema.UFix128Type: {
				{
					// Multiplication truncates.
					// 45.6 * 1e-24 =  0.00..045
					a:         interpreter.NewUnmeteredUFix128ValueWithIntegerAndScale(456, sema.Fix128Scale-1),
					b:         interpreter.NewUnmeteredUFix128ValueWithIntegerAndScale(1, 0),
					operation: ast.OperationMul,
					result:    interpreter.NewUnmeteredUFix128ValueWithIntegerAndScale(45, 0),
				},
				{
					// Multiplication truncates and underflows.
					// 0.6 * 1e-24 =  0.00..000
					a:         interpreter.NewUnmeteredUFix128ValueWithIntegerAndScale(6, sema.Fix128Scale-1),
					b:         interpreter.NewUnmeteredUFix128ValueWithIntegerAndScale(1, 0),
					operation: ast.OperationMul,
					result:    interpreter.NewUnmeteredUFix128ValueWithIntegerAndScale(0, 0),
				},
				{
					// Division truncates.
					// 456 / 1e25 =  0.00..045
					a:         interpreter.NewUnmeteredUFix128ValueWithIntegerAndScale(456, 0),
					b:         interpreter.NewUnmeteredUFix128ValueWithIntegerAndScale(1, sema.Fix128Scale+1),
					operation: ast.OperationDiv,
					result:    interpreter.NewUnmeteredUFix128ValueWithIntegerAndScale(45, 0),
				},
				{
					// Division truncates and underflows.
					// 6 / 1e25 =  0.00..000
					a:         interpreter.NewUnmeteredUFix128ValueWithIntegerAndScale(6, 0),
					b:         interpreter.NewUnmeteredUFix128ValueWithIntegerAndScale(1, sema.Fix128Scale+1),
					operation: ast.OperationDiv,
					result:    interpreter.NewUnmeteredUFix128ValueWithIntegerAndScale(0, 0),
				},
			},
		}

		for typ, testValues := range testCases {
			typ := typ

			for _, testCase := range testValues {
				test(
					t,
					typ,
					testCase.a,
					testCase.b,
					testCase.result,
					testCase.operation,
				)
			}
		}
	})

	t.Run("signed", func(t *testing.T) {
		t.Parallel()

		testCases := map[sema.Type][]testValue{
			sema.Fix64Type: {
				{
					// Multiplication truncates.
					// 45.6 * 1e-8 =  0.00000045
					a:         interpreter.NewUnmeteredFix64Value(4560000000),
					b:         interpreter.NewUnmeteredFix64Value(1),
					operation: ast.OperationMul,
					result:    interpreter.NewUnmeteredFix64Value(45),
				},
				{
					// Multiplication truncates and underflows.
					// 0.6 * 1e-8 =  0.00000000
					a:         interpreter.NewUnmeteredFix64Value(60000000),
					b:         interpreter.NewUnmeteredFix64Value(1),
					operation: ast.OperationMul,
					result:    interpreter.NewUnmeteredFix64Value(0),
				},
				{
					// Division truncates.
					// 0.00000456 / 10 =  0.00000045
					a:         interpreter.NewUnmeteredFix64Value(456),
					b:         interpreter.NewUnmeteredFix64Value(1000000000),
					operation: ast.OperationDiv,
					result:    interpreter.NewUnmeteredFix64Value(45),
				},
				{
					// Division truncates and underflows.
					// 0.00000006 / 10 =  0.00000000
					a:         interpreter.NewUnmeteredFix64Value(6),
					b:         interpreter.NewUnmeteredFix64Value(1000000000),
					operation: ast.OperationDiv,
					result:    interpreter.NewUnmeteredFix64Value(0),
				},
			},

			sema.Fix128Type: {
				{
					// Multiplication truncates.
					// 45.6 * 1e-24 =  0.00..045
					a:         interpreter.NewUnmeteredFix128ValueWithIntegerAndScale(456, sema.Fix128Scale-1),
					b:         interpreter.NewUnmeteredFix128ValueWithIntegerAndScale(1, 0),
					operation: ast.OperationMul,
					result:    interpreter.NewUnmeteredFix128ValueWithIntegerAndScale(45, 0),
				},
				{
					// Multiplication truncates and underflows.
					// 0.6 * 1e-24 =  0.00..000
					a:         interpreter.NewUnmeteredFix128ValueWithIntegerAndScale(6, sema.Fix128Scale-1),
					b:         interpreter.NewUnmeteredFix128ValueWithIntegerAndScale(1, 0),
					operation: ast.OperationMul,
					result:    interpreter.NewUnmeteredFix128ValueWithIntegerAndScale(0, 0),
				},
				{
					// Division truncates.
					// 456 / 1e25 =  0.00..045
					a:         interpreter.NewUnmeteredFix128ValueWithIntegerAndScale(456, 0),
					b:         interpreter.NewUnmeteredFix128ValueWithIntegerAndScale(1, sema.Fix128Scale+1),
					operation: ast.OperationDiv,
					result:    interpreter.NewUnmeteredFix128ValueWithIntegerAndScale(45, 0),
				},
				{
					// Division truncates and underflows.
					// 6 / 1e25 =  0.00..000
					a:         interpreter.NewUnmeteredFix128ValueWithIntegerAndScale(6, 0),
					b:         interpreter.NewUnmeteredFix128ValueWithIntegerAndScale(1, sema.Fix128Scale+1),
					operation: ast.OperationDiv,
					result:    interpreter.NewUnmeteredFix128ValueWithIntegerAndScale(0, 0),
				},
			},
		}

		for typ, testValues := range testCases {
			typ := typ

			for _, testCase := range testValues {
				a := testCase.a
				b := testCase.b
				operation := testCase.operation
				result := testCase.result

				// Both `a` and `b` are positive.
				test(
					t,
					typ,
					a,
					b,
					result,
					operation,
				)

				// Both `a` and `b` are negative.
				test(
					t,
					typ,
					a.Negate(nil),
					b.Negate(nil),
					result,
					operation,
				)

				// `a` is positive and `b` is negative.
				test(
					t,
					typ,
					a,
					b.Negate(nil),
					result.Negate(nil),
					operation,
				)

				// `a` is negative and `b` is positive.
				test(
					t,
					typ,
					a.Negate(nil),
					b,
					result.Negate(nil),
					operation,
				)
			}
		}
	})
}

func parseCheckAndPrepareWithRoundingRule(t *testing.T, code string) Invokable {
	t.Helper()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)

	valueDeclaration := stdlib.InterpreterRoundingRuleConstructor
	baseValueActivation.DeclareValue(valueDeclaration)
	interpreter.Declare(baseActivation, valueDeclaration)

	invokable, err := parseCheckAndPrepareWithOptions(
		t,
		code,
		ParseCheckAndInterpretOptions{
			ParseAndCheckOptions: &ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
			InterpreterConfig: &interpreter.Config{
				BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
					return baseActivation
				},
			},
		},
	)
	require.NoError(t, err)

	return invokable
}

func TestInterpretFix64WithRoundingRule(t *testing.T) {
	t.Parallel()

	// Fix128 has 24 decimal places, Fix64 has 8.
	// Rounding applies to the 9th+ decimal places when converting Fix128 → Fix64.
	//
	// Test values:
	//   1.000000003... (9th digit < 5): non-halfway, fractional part below midpoint
	//   1.000000005... (9th digit = 5): exact halfway between 1.00000000 and 1.00000001
	//   1.000000007... (9th digit > 5): non-halfway, fractional part above midpoint
	//
	// Expected results per rounding mode:
	//   towardZero:     always truncate → 1.00000000 (positive), -1.00000000 (negative)
	//   awayFromZero:   always round up magnitude → 1.00000001 (positive), -1.00000001 (negative)
	//   nearestHalfAway: < 5 → 1.00000000, = 5 → 1.00000001 (away), > 5 → 1.00000001
	//   nearestHalfEven: < 5 → 1.00000000, = 5 → 1.00000000 (even), > 5 → 1.00000001

	type testCase struct {
		name     string
		code     string
		expected interpreter.Fix64Value
	}

	tests := []testCase{
		// towardZero: truncates fractional part beyond 8 decimals
		{
			name: "towardZero, positive, non-halfway below",
			code: `fun main(): Fix64 {
                let x: Fix128 = 1.000000003000000000000000
                return Fix64(x, rounding: RoundingRule.towardZero)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(100000000), // 1.00000000
		},
		{
			name: "towardZero, positive, exact halfway",
			code: `fun main(): Fix64 {
                let x: Fix128 = 1.000000005000000000000000
                return Fix64(x, rounding: RoundingRule.towardZero)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(100000000), // 1.00000000
		},
		{
			name: "towardZero, positive, non-halfway above",
			code: `fun main(): Fix64 {
                let x: Fix128 = 1.000000007000000000000000
                return Fix64(x, rounding: RoundingRule.towardZero)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(100000000), // 1.00000000
		},
		{
			name: "towardZero, negative, non-halfway below",
			code: `fun main(): Fix64 {
                let x: Fix128 = -1.000000003000000000000000
                return Fix64(x, rounding: RoundingRule.towardZero)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(-100000000), // -1.00000000
		},
		{
			name: "towardZero, negative, exact halfway",
			code: `fun main(): Fix64 {
                let x: Fix128 = -1.000000005000000000000000
                return Fix64(x, rounding: RoundingRule.towardZero)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(-100000000), // -1.00000000
		},
		{
			name: "towardZero, negative, non-halfway above",
			code: `fun main(): Fix64 {
                let x: Fix128 = -1.000000007000000000000000
                return Fix64(x, rounding: RoundingRule.towardZero)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(-100000000), // -1.00000000
		},

		// awayFromZero: rounds up magnitude for any nonzero fractional part
		{
			name: "awayFromZero, positive, non-halfway below",
			code: `fun main(): Fix64 {
                let x: Fix128 = 1.000000003000000000000000
                return Fix64(x, rounding: RoundingRule.awayFromZero)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(100000001), // 1.00000001
		},
		{
			name: "awayFromZero, positive, exact halfway",
			code: `fun main(): Fix64 {
                let x: Fix128 = 1.000000005000000000000000
                return Fix64(x, rounding: RoundingRule.awayFromZero)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(100000001), // 1.00000001
		},
		{
			name: "awayFromZero, positive, non-halfway above",
			code: `fun main(): Fix64 {
                let x: Fix128 = 1.000000007000000000000000
                return Fix64(x, rounding: RoundingRule.awayFromZero)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(100000001), // 1.00000001
		},
		{
			name: "awayFromZero, negative, non-halfway below",
			code: `fun main(): Fix64 {
                let x: Fix128 = -1.000000003000000000000000
                return Fix64(x, rounding: RoundingRule.awayFromZero)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(-100000001), // -1.00000001
		},
		{
			name: "awayFromZero, negative, exact halfway",
			code: `fun main(): Fix64 {
                let x: Fix128 = -1.000000005000000000000000
                return Fix64(x, rounding: RoundingRule.awayFromZero)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(-100000001), // -1.00000001
		},
		{
			name: "awayFromZero, negative, non-halfway above",
			code: `fun main(): Fix64 {
                let x: Fix128 = -1.000000007000000000000000
                return Fix64(x, rounding: RoundingRule.awayFromZero)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(-100000001), // -1.00000001
		},

		// nearestHalfAway: nearest, tie breaks away from zero
		{
			name: "nearestHalfAway, positive, non-halfway below",
			code: `fun main(): Fix64 {
                let x: Fix128 = 1.000000003000000000000000
                return Fix64(x, rounding: RoundingRule.nearestHalfAway)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(100000000), // 1.00000000
		},
		{
			name: "nearestHalfAway, positive, exact halfway",
			code: `fun main(): Fix64 {
                let x: Fix128 = 1.000000005000000000000000
                return Fix64(x, rounding: RoundingRule.nearestHalfAway)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(100000001), // 1.00000001 (tie → away)
		},
		{
			name: "nearestHalfAway, positive, non-halfway above",
			code: `fun main(): Fix64 {
                let x: Fix128 = 1.000000007000000000000000
                return Fix64(x, rounding: RoundingRule.nearestHalfAway)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(100000001), // 1.00000001
		},
		{
			name: "nearestHalfAway, negative, exact halfway",
			code: `fun main(): Fix64 {
                let x: Fix128 = -1.000000005000000000000000
                return Fix64(x, rounding: RoundingRule.nearestHalfAway)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(-100000001), // -1.00000001 (tie → away)
		},

		// nearestHalfEven: nearest, tie breaks to even
		{
			name: "nearestHalfEven, positive, non-halfway below",
			code: `fun main(): Fix64 {
                let x: Fix128 = 1.000000003000000000000000
                return Fix64(x, rounding: RoundingRule.nearestHalfEven)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(100000000), // 1.00000000
		},
		{
			name: "nearestHalfEven, positive, exact halfway (last digit 0 is even)",
			code: `fun main(): Fix64 {
                let x: Fix128 = 1.000000005000000000000000
                return Fix64(x, rounding: RoundingRule.nearestHalfEven)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(100000000), // 1.00000000 (tie → even, 0 is even)
		},
		{
			name: "nearestHalfEven, positive, non-halfway above",
			code: `fun main(): Fix64 {
                let x: Fix128 = 1.000000007000000000000000
                return Fix64(x, rounding: RoundingRule.nearestHalfEven)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(100000001), // 1.00000001
		},
		{
			name: "nearestHalfEven, positive, exact halfway (last digit 1 is odd)",
			code: `fun main(): Fix64 {
                let x: Fix128 = 1.000000015000000000000000
                return Fix64(x, rounding: RoundingRule.nearestHalfEven)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(100000002), // 1.00000002 (tie → even, 2 is even)
		},
		{
			name: "nearestHalfEven, negative, exact halfway",
			code: `fun main(): Fix64 {
                let x: Fix128 = -1.000000005000000000000000
                return Fix64(x, rounding: RoundingRule.nearestHalfEven)
            }`,
			expected: interpreter.NewUnmeteredFix64Value(-100000000), // -1.00000000 (tie → even)
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			invokable := parseCheckAndPrepareWithRoundingRule(t, tc.code)
			result, err := invokable.Invoke("main")
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}

	t.Run("backward compat, no rounding truncates", func(t *testing.T) {
		t.Parallel()

		invokable := parseCheckAndPrepare(t, `
            fun main(): Fix64 {
                let x: Fix128 = 1.000000005000000000000000
                return Fix64(x)
            }
        `)

		result, err := invokable.Invoke("main")
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredFix64Value(100000000), result)
	})

	t.Run("integer conversion ignores rounding", func(t *testing.T) {
		t.Parallel()

		invokable := parseCheckAndPrepareWithRoundingRule(t, `
            fun main(): Fix64 {
                return Fix64(42, rounding: RoundingRule.awayFromZero)
            }
        `)

		result, err := invokable.Invoke("main")
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredFix64ValueWithInteger(42), result)
	})
}

func TestInterpretUFix64WithRoundingRule(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		code     string
		expected interpreter.UFix64Value
	}

	tests := []testCase{
		// towardZero
		{
			name: "towardZero, non-halfway below",
			code: `fun main(): UFix64 {
                let x: UFix128 = 1.000000003000000000000000
                return UFix64(x, rounding: RoundingRule.towardZero)
            }`,
			expected: interpreter.NewUnmeteredUFix64Value(100000000),
		},
		{
			name: "towardZero, exact halfway",
			code: `fun main(): UFix64 {
                let x: UFix128 = 1.000000005000000000000000
                return UFix64(x, rounding: RoundingRule.towardZero)
            }`,
			expected: interpreter.NewUnmeteredUFix64Value(100000000),
		},
		{
			name: "towardZero, non-halfway above",
			code: `fun main(): UFix64 {
                let x: UFix128 = 1.000000007000000000000000
                return UFix64(x, rounding: RoundingRule.towardZero)
            }`,
			expected: interpreter.NewUnmeteredUFix64Value(100000000),
		},

		// awayFromZero
		{
			name: "awayFromZero, non-halfway below",
			code: `fun main(): UFix64 {
                let x: UFix128 = 1.000000003000000000000000
                return UFix64(x, rounding: RoundingRule.awayFromZero)
            }`,
			expected: interpreter.NewUnmeteredUFix64Value(100000001),
		},
		{
			name: "awayFromZero, exact halfway",
			code: `fun main(): UFix64 {
                let x: UFix128 = 1.000000005000000000000000
                return UFix64(x, rounding: RoundingRule.awayFromZero)
            }`,
			expected: interpreter.NewUnmeteredUFix64Value(100000001),
		},
		{
			name: "awayFromZero, non-halfway above",
			code: `fun main(): UFix64 {
                let x: UFix128 = 1.000000007000000000000000
                return UFix64(x, rounding: RoundingRule.awayFromZero)
            }`,
			expected: interpreter.NewUnmeteredUFix64Value(100000001),
		},

		// nearestHalfAway
		{
			name: "nearestHalfAway, non-halfway below",
			code: `fun main(): UFix64 {
                let x: UFix128 = 1.000000003000000000000000
                return UFix64(x, rounding: RoundingRule.nearestHalfAway)
            }`,
			expected: interpreter.NewUnmeteredUFix64Value(100000000),
		},
		{
			name: "nearestHalfAway, exact halfway",
			code: `fun main(): UFix64 {
                let x: UFix128 = 1.000000005000000000000000
                return UFix64(x, rounding: RoundingRule.nearestHalfAway)
            }`,
			expected: interpreter.NewUnmeteredUFix64Value(100000001), // tie → away
		},
		{
			name: "nearestHalfAway, non-halfway above",
			code: `fun main(): UFix64 {
                let x: UFix128 = 1.000000007000000000000000
                return UFix64(x, rounding: RoundingRule.nearestHalfAway)
            }`,
			expected: interpreter.NewUnmeteredUFix64Value(100000001),
		},

		// nearestHalfEven
		{
			name: "nearestHalfEven, non-halfway below",
			code: `fun main(): UFix64 {
                let x: UFix128 = 1.000000003000000000000000
                return UFix64(x, rounding: RoundingRule.nearestHalfEven)
            }`,
			expected: interpreter.NewUnmeteredUFix64Value(100000000),
		},
		{
			name: "nearestHalfEven, exact halfway (last digit 0 is even)",
			code: `fun main(): UFix64 {
                let x: UFix128 = 1.000000005000000000000000
                return UFix64(x, rounding: RoundingRule.nearestHalfEven)
            }`,
			expected: interpreter.NewUnmeteredUFix64Value(100000000), // tie → even, 0 is even
		},
		{
			name: "nearestHalfEven, non-halfway above",
			code: `fun main(): UFix64 {
                let x: UFix128 = 1.000000007000000000000000
                return UFix64(x, rounding: RoundingRule.nearestHalfEven)
            }`,
			expected: interpreter.NewUnmeteredUFix64Value(100000001),
		},
		{
			name: "nearestHalfEven, exact halfway (last digit 1 is odd)",
			code: `fun main(): UFix64 {
                let x: UFix128 = 1.000000015000000000000000
                return UFix64(x, rounding: RoundingRule.nearestHalfEven)
            }`,
			expected: interpreter.NewUnmeteredUFix64Value(100000002), // tie → even, 2 is even
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			invokable := parseCheckAndPrepareWithRoundingRule(t, tc.code)
			result, err := invokable.Invoke("main")
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}

	t.Run("backward compat, no rounding truncates", func(t *testing.T) {
		t.Parallel()

		invokable := parseCheckAndPrepare(t, `
            fun main(): UFix64 {
                let x: UFix128 = 1.000000005000000000000000
                return UFix64(x)
            }
        `)

		result, err := invokable.Invoke("main")
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredUFix64Value(100000000), result)
	})
}

func TestInterpretFix64WithRoundingRuleOverflow(t *testing.T) {
	t.Parallel()

	t.Run("Fix128 overflow", func(t *testing.T) {
		t.Parallel()

		invokable := parseCheckAndPrepareWithRoundingRule(t, `
            fun main(): Fix64 {
                // Fix128 max is much larger than Fix64 max
                let x: Fix128 = Fix128.max
                return Fix64(x, rounding: RoundingRule.towardZero)
            }
        `)

		_, err := invokable.Invoke("main")
		RequireError(t, err)
		var expectedError *interpreter.OverflowError
		require.ErrorAs(t, err, &expectedError)
	})

	t.Run("Fix128 negative overflow", func(t *testing.T) {
		t.Parallel()

		invokable := parseCheckAndPrepareWithRoundingRule(t, `
            fun main(): Fix64 {
                let x: Fix128 = Fix128.min
                return Fix64(x, rounding: RoundingRule.towardZero)
            }
        `)

		_, err := invokable.Invoke("main")
		RequireError(t, err)
		var expectedError *interpreter.UnderflowError
		require.ErrorAs(t, err, &expectedError)
	})

	t.Run("rounding causes overflow", func(t *testing.T) {
		t.Parallel()

		invokable := parseCheckAndPrepareWithRoundingRule(t, `
            fun main(): Fix64 {
                // Fix64.max as Fix128, plus a fraction that would round up
                let x: Fix128 = 92233720368.547758079999999999999999
                return Fix64(x, rounding: RoundingRule.awayFromZero)
            }
        `)

		_, err := invokable.Invoke("main")
		RequireError(t, err)
		var expectedError *interpreter.OverflowError
		require.ErrorAs(t, err, &expectedError)
	})
}

func TestInterpretUFix64WithRoundingRuleOverflow(t *testing.T) {
	t.Parallel()

	t.Run("UFix128 overflow", func(t *testing.T) {
		t.Parallel()

		invokable := parseCheckAndPrepareWithRoundingRule(t, `
            fun main(): UFix64 {
                let x: UFix128 = UFix128.max
                return UFix64(x, rounding: RoundingRule.towardZero)
            }
        `)

		_, err := invokable.Invoke("main")
		RequireError(t, err)
		var expectedError *interpreter.OverflowError
		require.ErrorAs(t, err, &expectedError)
	})

	t.Run("Fix128 negative to UFix64", func(t *testing.T) {
		t.Parallel()

		invokable := parseCheckAndPrepareWithRoundingRule(t, `
            fun main(): UFix64 {
                let x: Fix128 = -1.0
                return UFix64(x, rounding: RoundingRule.towardZero)
            }
        `)

		_, err := invokable.Invoke("main")
		RequireError(t, err)
		var expectedError *interpreter.UnderflowError
		require.ErrorAs(t, err, &expectedError)
	})
}

func TestInterpretRoundingRuleEnum(t *testing.T) {
	t.Parallel()

	invokable := parseCheckAndPrepareWithRoundingRule(t, `
        fun main(): [UInt8] {
            return [
                RoundingRule.towardZero.rawValue,
                RoundingRule.awayFromZero.rawValue,
                RoundingRule.nearestHalfAway.rawValue,
                RoundingRule.nearestHalfEven.rawValue
            ]
        }
    `)

	result, err := invokable.Invoke("main")
	require.NoError(t, err)

	arrayValue := result.(*interpreter.ArrayValue)
	require.Equal(t, 4, arrayValue.Count())

	assert.Equal(t, interpreter.UInt8Value(0), arrayValue.Get(nil, 0))
	assert.Equal(t, interpreter.UInt8Value(1), arrayValue.Get(nil, 1))
	assert.Equal(t, interpreter.UInt8Value(2), arrayValue.Get(nil, 2))
	assert.Equal(t, interpreter.UInt8Value(3), arrayValue.Get(nil, 3))
}
