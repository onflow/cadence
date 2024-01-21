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

package checker

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

type inclusiveRangeConstructionTest struct {
	ty         sema.Type
	s, e, step int64
}

func TestCheckInclusiveRangeConstructionValid(t *testing.T) {
	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.InclusiveRangeConstructorFunction)

	validTestCases := []inclusiveRangeConstructionTest{
		// Int*
		{
			ty:   sema.IntType,
			s:    0,
			e:    10,
			step: 2,
		},
		{
			ty:   sema.IntType,
			s:    10,
			e:    -10,
			step: -2,
		},
		{
			ty:   sema.Int8Type,
			s:    0,
			e:    10,
			step: 2,
		},
		{
			ty:   sema.Int8Type,
			s:    10,
			e:    -10,
			step: -2,
		},
		{
			ty:   sema.Int16Type,
			s:    0,
			e:    10,
			step: 2,
		},
		{
			ty:   sema.Int16Type,
			s:    10,
			e:    -10,
			step: -2,
		},
		{
			ty:   sema.Int32Type,
			s:    0,
			e:    10,
			step: 2,
		},
		{
			ty:   sema.Int32Type,
			s:    10,
			e:    -10,
			step: -2,
		},
		{
			ty:   sema.Int64Type,
			s:    0,
			e:    10,
			step: 2,
		},
		{
			ty:   sema.Int64Type,
			s:    10,
			e:    -10,
			step: -2,
		},
		{
			ty:   sema.Int128Type,
			s:    0,
			e:    10,
			step: 2,
		},
		{
			ty:   sema.Int128Type,
			s:    10,
			e:    -10,
			step: -2,
		},
		{
			ty:   sema.Int256Type,
			s:    0,
			e:    10,
			step: 2,
		},
		{
			ty:   sema.Int256Type,
			s:    10,
			e:    -10,
			step: -2,
		},

		// UInt*
		{
			ty:   sema.UIntType,
			s:    0,
			e:    10,
			step: 2,
		},
		{
			ty:   sema.UInt8Type,
			s:    0,
			e:    10,
			step: 2,
		},
		{
			ty:   sema.UInt16Type,
			s:    0,
			e:    10,
			step: 2,
		},
		{
			ty:   sema.UInt32Type,
			s:    0,
			e:    10,
			step: 2,
		},
		{
			ty:   sema.UInt64Type,
			s:    0,
			e:    10,
			step: 2,
		},
		{
			ty:   sema.UInt128Type,
			s:    0,
			e:    10,
			step: 2,
		},
		{
			ty:   sema.UInt256Type,
			s:    0,
			e:    10,
			step: 2,
		},

		// Word*
		{
			ty:   sema.Word8Type,
			s:    0,
			e:    10,
			step: 2,
		},
		{
			ty:   sema.Word16Type,
			s:    0,
			e:    10,
			step: 2,
		},
		{
			ty:   sema.Word32Type,
			s:    0,
			e:    10,
			step: 2,
		},
		{
			ty:   sema.Word64Type,
			s:    0,
			e:    10,
			step: 2,
		},
		{
			ty:   sema.Word128Type,
			s:    0,
			e:    10,
			step: 2,
		},
		{
			ty:   sema.Word256Type,
			s:    0,
			e:    10,
			step: 2,
		},
	}

	runValidCase := func(t *testing.T, testCase inclusiveRangeConstructionTest, withStep bool) {
		t.Run(testCase.ty.String(), func(t *testing.T) {
			t.Parallel()

			var code string
			if withStep {
				code = fmt.Sprintf(
					`
					   let s : %s = %d
					   let e : %s = %d
					   let step : %s = %d
					   let r: InclusiveRange<%s> = InclusiveRange(s, e, step: step)

					   let rs = r.start
					   let re = r.end
					   let rstep = r.step
					   let contains_res = r.contains(s)
					`,
					testCase.ty.String(), testCase.s, testCase.ty.String(), testCase.e, testCase.ty.String(), testCase.step, testCase.ty.String())
			} else {
				code = fmt.Sprintf(
					`
					   let s : %s = %d
					   let e : %s = %d
					   let r: InclusiveRange<%s> = InclusiveRange(s, e)

					   let rs = r.start
					   let re = r.end
					   let rstep = r.step
					   let contains_res = r.contains(s)
					`,
					testCase.ty.String(), testCase.s, testCase.ty.String(), testCase.e, testCase.ty.String())
			}

			checker, err := ParseAndCheckWithOptions(t, code,
				ParseAndCheckOptions{
					Config: &sema.Config{
						BaseValueActivationHandler: func(common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
				},
			)

			require.NoError(t, err)

			checkType := func(t *testing.T, name string, expectedType sema.Type) {
				resType := RequireGlobalValue(t, checker.Elaboration, name)
				assert.IsType(t, expectedType, resType)
			}

			checkType(t, "r", &sema.InclusiveRangeType{
				MemberType: testCase.ty,
			})
			checkType(t, "rs", testCase.ty)
			checkType(t, "re", testCase.ty)
			checkType(t, "rstep", testCase.ty)
			checkType(t, "contains_res", sema.BoolType)
		})
	}

	// Run each test case with and without step.
	for _, testCase := range validTestCases {
		runValidCase(t, testCase, true)
		runValidCase(t, testCase, false)
	}
}

func TestCheckInclusiveRangeConstructionInvalid(t *testing.T) {
	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.InclusiveRangeConstructorFunction)

	runInvalidCase := func(t *testing.T, label, code string, expectedErrorTypes []error) {
		t.Run(label, func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheckWithOptions(t, code,
				ParseAndCheckOptions{
					Config: &sema.Config{
						BaseValueActivationHandler: func(common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
				},
			)

			errs := RequireCheckerErrors(t, err, len(expectedErrorTypes))
			for i, err := range expectedErrorTypes {
				assert.IsType(t, err, errs[i])
			}
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

		// Wrong type of arguments

		// Any integer type name other than the integerType is sufficient.
		// There is nothing special about the Int128 and Int256 types here.
		differentTypeString := sema.Int128TypeName
		if typeString == differentTypeString {
			differentTypeString = sema.Int256TypeName
		}

		runInvalidCase(
			t,
			typeString,
			fmt.Sprintf("let r = InclusiveRange(%s(1), %s(2))", typeString, differentTypeString),
			[]error{
				&sema.TypeMismatchError{},
			},
		)
		runInvalidCase(
			t,
			typeString,
			fmt.Sprintf("let r = InclusiveRange(%s(1), %s(10), step: %s(2))", typeString, typeString, differentTypeString),
			[]error{
				&sema.TypeMismatchError{},
			},
		)

		// Not enough arguments
		runInvalidCase(
			t,
			typeString,
			fmt.Sprintf("let r = InclusiveRange(%s(1))", typeString),
			[]error{&sema.InsufficientArgumentsError{}},
		)

		// Label for step not provided
		runInvalidCase(
			t,
			typeString,
			fmt.Sprintf("let r = InclusiveRange(%s(1), %s(0), %s(10))", typeString, typeString, typeString),
			[]error{&sema.MissingArgumentLabelError{}},
		)

		// Label for start and end provided
		runInvalidCase(
			t,
			typeString,
			fmt.Sprintf("let r = InclusiveRange(start: %s(1), %s(0))", typeString, typeString),
			[]error{&sema.IncorrectArgumentLabelError{}},
		)
		runInvalidCase(
			t,
			typeString,
			fmt.Sprintf("let r = InclusiveRange(%s(1), end: %s(0))", typeString, typeString),
			[]error{&sema.IncorrectArgumentLabelError{}},
		)
	}

	runInvalidCase(
		t,
		"without_inner_type_annotation",
		"let r: InclusiveRange<> = InclusiveRange(1, 10)",
		[]error{&sema.InvalidTypeArgumentCountError{}, &sema.MissingTypeArgumentError{}},
	)

	runInvalidCase(
		t,
		"without instantiation type",
		"let r: InclusiveRange = InclusiveRange(1, 10)",
		[]error{&sema.MissingTypeArgumentError{}},
	)

	runInvalidCase(
		t,
		"same_supertype_different_subtype_start_end",
		`
			let a: Integer = UInt8(0)
			let b: Integer = Int16(10)
			let r = InclusiveRange(a, b)
		`,
		[]error{&sema.InvalidTypeArgumentError{}},
	)
	runInvalidCase(
		t,
		"same_supertype_different_subtype_start_step",
		`
			let a: Integer = UInt8(0)
			let b: Integer = UInt8(10)
			let s: Integer = UInt16(2)
			let r = InclusiveRange(a, b, step: s)
		`,
		[]error{&sema.InvalidTypeArgumentError{}},
	)
}

func TestInclusiveRangeNonLeafIntegerTypes(t *testing.T) {

	t.Parallel()

	newOptions := func() ParseAndCheckOptions {
		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(stdlib.InclusiveRangeConstructorFunction)

		return ParseAndCheckOptions{
			Config: &sema.Config{
				BaseValueActivationHandler: func(common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		}
	}

	test := func(t *testing.T, ty sema.Type) {
		t.Run(fmt.Sprintf("InclusiveRange<%s> infer from args", ty), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheckWithOptions(t, fmt.Sprintf(`
				let a: %[1]s = 0
				let b: %[1]s = 10
				var range = InclusiveRange<%[1]s>(a, b)
			`, ty), newOptions())

			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.InvalidTypeArgumentError{}, errs[0])
		})

		t.Run(fmt.Sprintf("InclusiveRange<%s> infer from lhs", ty), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheckWithOptions(t, fmt.Sprintf(`
				let a: %[1]s = 0
				let b: %[1]s = 10
				var range: InclusiveRange<%[1]s> = InclusiveRange<%[1]s>(a, b)
			`, ty), newOptions())

			// One for the invocation and another for the type.
			errs := RequireCheckerErrors(t, err, 2)
			assert.IsType(t, &sema.InvalidTypeArgumentError{}, errs[0])
			assert.IsType(t, &sema.InvalidTypeArgumentError{}, errs[1])
		})

		t.Run(fmt.Sprintf("InclusiveRange<%s> assignment", ty), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheckWithOptions(t, fmt.Sprintf(`
				let a: InclusiveRange<Int> = InclusiveRange(0, 10)
				let b: InclusiveRange<%s> = a
			`, ty), newOptions())

			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.InvalidTypeArgumentError{}, errs[0])
		})
	}

	for _, ty := range []sema.Type{
		sema.IntegerType,
		sema.SignedIntegerType,
	} {
		test(t, ty)
	}
}
