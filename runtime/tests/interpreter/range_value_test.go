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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/activations"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/utils"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

type containsTestCase struct {
	param               int64
	expectedWithoutStep bool
	expectedWithStep    bool
}

type inclusiveRangeConstructionTest struct {
	ty            sema.Type
	s, e, step    int8
	containsTests []containsTestCase
}

func TestInclusiveRange(t *testing.T) {
	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.InclusiveRangeConstructorFunction)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, stdlib.InclusiveRangeConstructorFunction)

	unsignedContainsTestCases := []containsTestCase{
		{
			param:               1,
			expectedWithoutStep: true,
			expectedWithStep:    false,
		},
		{
			param:               12,
			expectedWithoutStep: false,
			expectedWithStep:    false,
		},
		{
			param:               10,
			expectedWithoutStep: true,
			expectedWithStep:    true,
		},
		{
			param:               0,
			expectedWithoutStep: true,
			expectedWithStep:    true,
		},
		{
			param:               2,
			expectedWithoutStep: true,
			expectedWithStep:    true,
		},
	}

	signedContainsTestCasesForward := []containsTestCase{
		{
			param:               1,
			expectedWithoutStep: true,
			expectedWithStep:    false,
		},
		{
			param:               100,
			expectedWithoutStep: false,
			expectedWithStep:    false,
		},
		{
			param:               -100,
			expectedWithoutStep: false,
			expectedWithStep:    false,
		},
		{
			param:               0,
			expectedWithoutStep: true,
			expectedWithStep:    true,
		},
		{
			param:               4,
			expectedWithoutStep: true,
			expectedWithStep:    true,
		},
		{
			param:               10,
			expectedWithoutStep: true,
			expectedWithStep:    true,
		},
	}
	signedContainsTestCasesBackward := []containsTestCase{
		{
			param:               1,
			expectedWithoutStep: true,
			expectedWithStep:    false,
		},
		{
			param:               12,
			expectedWithoutStep: false,
			expectedWithStep:    false,
		},
		{
			param:               -12,
			expectedWithoutStep: false,
			expectedWithStep:    false,
		},
		{
			param:               10,
			expectedWithoutStep: true,
			expectedWithStep:    true,
		},
		{
			param:               -10,
			expectedWithoutStep: true,
			expectedWithStep:    true,
		},
		{
			param:               -8,
			expectedWithoutStep: true,
			expectedWithStep:    true,
		},
	}

	validTestCases := []inclusiveRangeConstructionTest{
		// Int*
		{
			ty:            sema.IntType,
			s:             0,
			e:             10,
			step:          2,
			containsTests: signedContainsTestCasesForward,
		},
		{
			ty:            sema.IntType,
			s:             10,
			e:             -10,
			step:          -2,
			containsTests: signedContainsTestCasesBackward,
		},
		{
			ty:            sema.Int8Type,
			s:             0,
			e:             10,
			step:          2,
			containsTests: signedContainsTestCasesForward,
		},
		{
			ty:            sema.Int8Type,
			s:             10,
			e:             -10,
			step:          -2,
			containsTests: signedContainsTestCasesBackward,
		},
		{
			ty:            sema.Int16Type,
			s:             0,
			e:             10,
			step:          2,
			containsTests: signedContainsTestCasesForward,
		},
		{
			ty:            sema.Int16Type,
			s:             10,
			e:             -10,
			step:          -2,
			containsTests: signedContainsTestCasesBackward,
		},
		{
			ty:            sema.Int32Type,
			s:             0,
			e:             10,
			step:          2,
			containsTests: signedContainsTestCasesForward,
		},
		{
			ty:            sema.Int32Type,
			s:             10,
			e:             -10,
			step:          -2,
			containsTests: signedContainsTestCasesBackward,
		},
		{
			ty:            sema.Int64Type,
			s:             0,
			e:             10,
			step:          2,
			containsTests: signedContainsTestCasesForward,
		},
		{
			ty:            sema.Int64Type,
			s:             10,
			e:             -10,
			step:          -2,
			containsTests: signedContainsTestCasesBackward,
		},
		{
			ty:            sema.Int128Type,
			s:             0,
			e:             10,
			step:          2,
			containsTests: signedContainsTestCasesForward,
		},
		{
			ty:            sema.Int128Type,
			s:             10,
			e:             -10,
			step:          -2,
			containsTests: signedContainsTestCasesBackward,
		},
		{
			ty:            sema.Int256Type,
			s:             0,
			e:             10,
			step:          2,
			containsTests: signedContainsTestCasesForward,
		},
		{
			ty:            sema.Int256Type,
			s:             10,
			e:             -10,
			step:          -2,
			containsTests: signedContainsTestCasesBackward,
		},

		// UInt*
		{
			ty:            sema.UIntType,
			s:             0,
			e:             10,
			step:          2,
			containsTests: unsignedContainsTestCases,
		},
		{
			ty:            sema.UInt8Type,
			s:             0,
			e:             10,
			step:          2,
			containsTests: unsignedContainsTestCases,
		},
		{
			ty:            sema.UInt16Type,
			s:             0,
			e:             10,
			step:          2,
			containsTests: unsignedContainsTestCases,
		},
		{
			ty:            sema.UInt32Type,
			s:             0,
			e:             10,
			step:          2,
			containsTests: unsignedContainsTestCases,
		},
		{
			ty:            sema.UInt64Type,
			s:             0,
			e:             10,
			step:          2,
			containsTests: unsignedContainsTestCases,
		},
		{
			ty:            sema.UInt128Type,
			s:             0,
			e:             10,
			step:          2,
			containsTests: unsignedContainsTestCases,
		},
		{
			ty:            sema.UInt256Type,
			s:             0,
			e:             10,
			step:          2,
			containsTests: unsignedContainsTestCases,
		},

		// Word*
		{
			ty:            sema.Word8Type,
			s:             0,
			e:             10,
			step:          2,
			containsTests: unsignedContainsTestCases,
		},
		{
			ty:            sema.Word16Type,
			s:             0,
			e:             10,
			step:          2,
			containsTests: unsignedContainsTestCases,
		},
		{
			ty:            sema.Word32Type,
			s:             0,
			e:             10,
			step:          2,
			containsTests: unsignedContainsTestCases,
		},
		{
			ty:            sema.Word64Type,
			s:             0,
			e:             10,
			step:          2,
			containsTests: unsignedContainsTestCases,
		},
		{
			ty:            sema.Word128Type,
			s:             0,
			e:             10,
			step:          2,
			containsTests: unsignedContainsTestCases,
		},
		{
			ty:            sema.Word256Type,
			s:             0,
			e:             10,
			step:          2,
			containsTests: unsignedContainsTestCases,
		},
	}

	runValidCase := func(t *testing.T, testCase inclusiveRangeConstructionTest, withStep bool) {
		t.Run(testCase.ty.String(), func(t *testing.T) {
			t.Parallel()

			// Generate code for the contains calls.
			var containsCode string
			for i, tc := range testCase.containsTests {
				containsCode += fmt.Sprintf("\nlet c_%d = r.contains(%d)", i, tc.param)
			}

			var code string
			if withStep {
				code = fmt.Sprintf(
					`
					   let s : %s = %d
					   let e : %s = %d
					   let step : %s = %d
					   let r: InclusiveRange<%s> = InclusiveRange(s, e, step: step)

					   %s
					`,
					testCase.ty.String(),
					testCase.s,
					testCase.ty.String(),
					testCase.e,
					testCase.ty.String(),
					testCase.step,
					testCase.ty.String(),
					containsCode,
				)
			} else {
				code = fmt.Sprintf(
					`
					   let s : %s = %d
					   let e : %s = %d
					   let r = InclusiveRange(s, e)

					   %s
					`,
					testCase.ty.String(),
					testCase.s,
					testCase.ty.String(),
					testCase.e,
					containsCode,
				)
			}

			inter, err := parseCheckAndInterpretWithOptions(t, code,
				ParseCheckAndInterpretOptions{
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
					Config: &interpreter.Config{
						BaseActivationHandler: func(common.Location) *interpreter.VariableActivation {
							return baseActivation
						},
					},
				},
			)

			require.NoError(t, err)

			integerType := interpreter.ConvertSemaToStaticType(
				nil,
				testCase.ty,
			)

			rangeType := interpreter.NewInclusiveRangeStaticType(nil, integerType)
			rangeSemaType := sema.NewInclusiveRangeType(nil, testCase.ty)

			var expectedRangeValue *interpreter.CompositeValue

			if withStep {
				expectedRangeValue = interpreter.NewInclusiveRangeValueWithStep(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.GetSmallIntegerValue(testCase.s, integerType),
					interpreter.GetSmallIntegerValue(testCase.e, integerType),
					interpreter.GetSmallIntegerValue(testCase.step, integerType),
					rangeType,
					rangeSemaType,
				)
			} else {
				expectedRangeValue = interpreter.NewInclusiveRangeValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.GetSmallIntegerValue(testCase.s, integerType),
					interpreter.GetSmallIntegerValue(testCase.e, integerType),
					rangeType,
					rangeSemaType,
				)
			}

			utils.AssertValuesEqual(
				t,
				inter,
				expectedRangeValue,
				inter.Globals.Get("r").GetValue(inter),
			)

			// Check that contains returns correct information.
			for i, tc := range testCase.containsTests {
				var expectedValue interpreter.Value
				if withStep {
					expectedValue = interpreter.AsBoolValue(tc.expectedWithStep)
				} else {
					expectedValue = interpreter.AsBoolValue(tc.expectedWithoutStep)
				}

				utils.AssertValuesEqual(
					t,
					inter,
					expectedValue,
					inter.Globals.Get(fmt.Sprintf("c_%d", i)).GetValue(inter),
				)
			}
		})
	}

	// Run each test case with and without step.
	for _, testCase := range validTestCases {
		runValidCase(t, testCase, true)
		runValidCase(t, testCase, false)
	}
}

func TestGetValueForIntegerType(t *testing.T) {

	t.Parallel()

	// Ensure that GetValueForIntegerType handles every IntegerType

	for _, integerType := range sema.AllIntegerTypes {
		switch integerType {
		case sema.IntegerType,
			sema.SignedIntegerType,
			sema.FixedSizeUnsignedIntegerType:
			continue
		}

		integerStaticType := interpreter.ConvertSemaToStaticType(nil, integerType)

		// Panics if not handled.
		_ = interpreter.GetSmallIntegerValue(int8(1), integerStaticType)
	}
}

func TestInclusiveRangeConstructionInvalid(t *testing.T) {
	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.InclusiveRangeConstructorFunction)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, stdlib.InclusiveRangeConstructorFunction)

	runInvalidCase := func(t *testing.T, label, code string, expectedError error, expectedMessage string) {
		t.Run(label, func(t *testing.T) {
			t.Parallel()

			_, err := parseCheckAndInterpretWithOptions(t, code,
				ParseCheckAndInterpretOptions{
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
					Config: &interpreter.Config{
						BaseActivationHandler: func(common.Location) *interpreter.VariableActivation {
							return baseActivation
						},
					},
				},
			)

			RequireError(t, err)

			require.ErrorAs(t, err, expectedError)
			require.True(t, strings.Contains(err.Error(), expectedMessage))
		})
	}

	for _, integerType := range sema.AllIntegerTypes {
		// Only test leaf types
		switch integerType {
		case sema.IntegerType,
			sema.SignedIntegerType,
			sema.FixedSizeUnsignedIntegerType:
			continue
		}

		typeString := integerType.String()

		// step = 0.
		runInvalidCase(
			t,
			typeString,
			fmt.Sprintf("let r = InclusiveRange(%s(1), %s(2), step: %s(0))", typeString, typeString, typeString),
			&interpreter.InclusiveRangeConstructionError{},
			"step value cannot be zero",
		)

		// step takes sequence away from end.
		runInvalidCase(
			t,
			typeString,
			fmt.Sprintf("let r = InclusiveRange(%s(40), %s(2), step: %s(2))", typeString, typeString, typeString),
			&interpreter.InclusiveRangeConstructionError{},
			"sequence is moving away from end",
		)
	}

	// Additional invalid cases for signed integer types
	for _, integerType := range sema.AllSignedIntegerTypes {
		// Only test leaf types
		switch integerType {
		case sema.SignedIntegerType:
			continue
		}

		typeString := integerType.String()

		// step takes sequence away from end with step being negative.
		// This would be a checker error for unsigned integers but a
		// runtime error in signed integers.
		runInvalidCase(
			t,
			typeString,
			fmt.Sprintf("let r = InclusiveRange(%s(4), %s(100), step: %s(-2))", typeString, typeString, typeString),
			&interpreter.InclusiveRangeConstructionError{},
			"sequence is moving away from end",
		)
	}

	// Additional invalid cases for unsigned integer types
	for _, integerType := range sema.AllUnsignedIntegerTypes {
		// Only test leaf types
		switch integerType {
		case sema.IntegerType:
			continue
		}

		typeString := integerType.String()

		runInvalidCase(
			t,
			typeString,
			fmt.Sprintf("let r = InclusiveRange(%s(40), %s(1))", typeString, typeString),
			&interpreter.InclusiveRangeConstructionError{},
			"step value cannot be negative for unsigned integer type",
		)
	}
}
