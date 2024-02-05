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

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

func TestCheckMathSqrt(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.MathContract)

	runTest := func(t *testing.T, numberType sema.Type) {
		t.Run(fmt.Sprintf("Sqrt<%s>", numberType), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheckWithOptions(t,
				fmt.Sprintf(`
				   let l: UFix64 = Math.Sqrt(%s(1))
				`, numberType),
				ParseAndCheckOptions{
					Config: &sema.Config{
						BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
				},
			)

			require.NoError(t, err)
		})
	}

	for _, numberType := range sema.AllNumberTypes {
		switch numberType {
		// Test only leaf types.
		case sema.NumberType, sema.SignedNumberType,
			sema.IntegerType, sema.SignedIntegerType, sema.FixedSizeUnsignedIntegerType,
			sema.FixedPointType, sema.SignedFixedPointType:
			continue
		}

		runTest(t, numberType)
	}
}

func TestCheckInvalidTypeMathSqrt(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.MathContract)

	_, err := ParseAndCheckWithOptions(t,
		`
           let l = Math.Sqrt("string")
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
	)

	errs := RequireCheckerErrors(t, err, 1)
	require.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidNumberArgumentsMathSqrt(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.MathContract)

	_, err := ParseAndCheckWithOptions(t,
		`
           let x = Math.Sqrt()
           let l = Math.Sqrt(1, 2)
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
	)

	errs := RequireCheckerErrors(t, err, 3)
	require.IsType(t, &sema.InsufficientArgumentsError{}, errs[0])
	require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[1])
	require.IsType(t, &sema.ExcessiveArgumentsError{}, errs[2])
}
