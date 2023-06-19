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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/activations"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

func TestInclusiveRange(t *testing.T) {
	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.InclusiveRangeConstructorFunction)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, stdlib.InclusiveRangeConstructorFunction)

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

					   let count = r.count
					   
					   let containsTest = r.contains(s)
					`,
					memberType.String(), memberType.String(), memberType.String())
			} else {
				code = fmt.Sprintf(
					`
					   let s : %s = 10
					   let e : %s = 20
					   let r = InclusiveRange(s, e)

					   let count = r.count

					   let containsTest = r.contains(s)
					`,
					memberType.String(), memberType.String())
			}

			_, err := parseCheckAndInterpretWithOptions(t, code,
				ParseCheckAndInterpretOptions{
					CheckerConfig: &sema.Config{
						BaseValueActivation: baseValueActivation,
					},
					Config: &interpreter.Config{
						BaseActivation: baseActivation,
					},
				},
			)

			require.NoError(t, err)
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
