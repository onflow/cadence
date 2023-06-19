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

	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

func TestInclusiveRange(t *testing.T) {
	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.InclusiveRangeConstructorFunction)

	runValidCase := func(t *testing.T, memberType sema.Type, withStep bool) {
		t.Run(memberType.String(), func(t *testing.T) {
			t.Parallel()

			var code string
			if withStep {
				code = fmt.Sprintf(
					`
					   let s : %s = 10
					   let e : %s = 20
					   let step : %s = 2
					   let r = InclusiveRange(s, e, step: step)
					`,
					memberType.String(), memberType.String(), memberType.String())
			} else {
				code = fmt.Sprintf(
					`
					   let s : %s = 10
					   let e : %s = 20
					   let r = InclusiveRange(s, e)
					`,
					memberType.String(), memberType.String())
			}

			checker, err := ParseAndCheckWithOptions(t, code,
				ParseAndCheckOptions{
					Config: &sema.Config{
						BaseValueActivation: baseValueActivation,
					},
				},
			)

			require.NoError(t, err)
			resType := RequireGlobalValue(t, checker.Elaboration, "r")
			require.Equal(t,
				&sema.InclusiveRangeType{
					MemberType: memberType,
				},
				resType,
			)
		})
	}

	runValidCaseWithoutStep := func(t *testing.T, memberType sema.Type) {
		runValidCase(t, memberType, false)
	}
	runValidCaseWithStep := func(t *testing.T, memberType sema.Type) {
		runValidCase(t, memberType, true)
	}

	for _, integerType := range sema.AllIntegerTypes {
		switch integerType {
		case sema.IntegerType, sema.SignedIntegerType:
			continue
		}

		runValidCaseWithStep(t, integerType)
		runValidCaseWithoutStep(t, integerType)
	}
}
