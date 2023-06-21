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

	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

type inclusiveRangeConstructionTest struct {
	ty         sema.Type
	s, e, step int64
}

func TestInclusiveRangeConstruction(t *testing.T) {
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
					   let r = InclusiveRange(s, e, step: step)

					   let rs = r.start
					   let re = r.end
					   let rstep = r.step
					   let contains_res = r.contains(s)
					`,
					testCase.ty.String(), testCase.s, testCase.ty.String(), testCase.e, testCase.ty.String(), testCase.step)
			} else {
				code = fmt.Sprintf(
					`
					   let s : %s = %d
					   let e : %s = %d
					   let r = InclusiveRange(s, e)

					   let rs = r.start
					   let re = r.end
					   let rstep = r.step
					   let contains_res = r.contains(s)
					`,
					testCase.ty.String(), testCase.s, testCase.ty.String(), testCase.e)
			}

			checker, err := ParseAndCheckWithOptions(t, code,
				ParseAndCheckOptions{
					Config: &sema.Config{
						BaseValueActivation: baseValueActivation,
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
