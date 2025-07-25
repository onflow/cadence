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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	"github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestInterpretPathCapability(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, code string) (common_utils.Invokable, error) {
		borrowType := &sema.ReferenceType{
			Type:          sema.StringType,
			Authorization: sema.UnauthorizedAccess,
		}

		borrowStaticType := interpreter.ConvertSemaToStaticType(nil, borrowType)

		value := stdlib.StandardLibraryValue{
			Type: &sema.CapabilityType{
				BorrowType: borrowType,
			},
			Value: interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
				borrowStaticType,
				interpreter.AddressValue{0x42},
				interpreter.PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "foo",
				},
			),
			Name: "cap",
			Kind: common.DeclarationKindConstant,
		}

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(value)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, value)

		return parseCheckAndPrepareWithOptions(
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
	}

	t.Run("transfer", func(t *testing.T) {

		t.Parallel()

		_, err := test(t, `
          fun f(_ cap: Capability<&String>): Capability<&String>? {
              return cap
          }

	      let capOpt: Capability<&String>? = f(cap)
        `)
		require.NoError(t, err)
	})

	t.Run("borrow", func(t *testing.T) {

		t.Parallel()

		inter, err := test(t, `
          fun test(): &String? {
              return cap.borrow()
          }
        `)
		require.NoError(t, err)

		res, err := inter.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, interpreter.Nil, res)
	})

	t.Run("check", func(t *testing.T) {

		t.Parallel()

		inter, err := test(t, `
          fun test(): Bool {
              return cap.check()
          }
        `)
		require.NoError(t, err)

		res, err := inter.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, interpreter.FalseValue, res)
	})

	t.Run("id", func(t *testing.T) {

		t.Parallel()

		inter, err := test(t, `
          fun test(): UInt64 {
              return cap.id
          }
        `)
		require.NoError(t, err)

		res, err := inter.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, interpreter.UInt64Value(0), res)
	})
}
