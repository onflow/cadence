/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpretTransferCheck(t *testing.T) {

	t.Parallel()

	t.Run("String value as composite", func(t *testing.T) {

		t.Parallel()

		ty := &sema.CompositeType{
			Location:   utils.TestLocation,
			Identifier: "Fruit",
			Kind:       common.CompositeKindStructure,
		}

		valueDeclarations := stdlib.StandardLibraryValues{
			{
				Name: "fruit",
				Type: ty,
				// NOTE: not an instance of the type
				ValueFactory: func(_ *interpreter.Interpreter) interpreter.Value {
					return interpreter.NewUnmeteredStringValue("fruit")
				},
				Kind: common.DeclarationKindConstant,
			},
		}

		typeDeclarations := stdlib.StandardLibraryTypes{
			{
				Name: ty.Identifier,
				Type: ty,
				Kind: common.DeclarationKindStructure,
			},
		}

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              fun test() {
                  let alsoFruit: Fruit = fruit
              }
            `,
			ParseCheckAndInterpretOptions{
				CheckerOptions: []sema.Option{
					sema.WithPredeclaredValues(valueDeclarations.ToSemaValueDeclarations()),
					sema.WithPredeclaredTypes(typeDeclarations.ToTypeDeclarations()),
				},
				Options: []interpreter.Option{
					interpreter.WithPredeclaredValues(valueDeclarations.ToInterpreterValueDeclarations()),
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)

		require.ErrorAs(t, err, &interpreter.ValueTransferTypeError{})
	})

	t.Run("contract and restricted type", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
		      contract interface CI {
		          resource interface RI {}

		          resource R: RI {}

		          fun createR(): @R
		      }

              contract C: CI {
		          resource R: CI.RI {}

		          fun createR(): @R {
		              return <- create R()
		          }
		      }

              fun test() {
                  let r <- C.createR()
                  let r2: @CI.R <- r as @CI.R
                  let r3: @CI.R{CI.RI} <- r2
                  destroy r3
              }
            `,
			ParseCheckAndInterpretOptions{
				Options: []interpreter.Option{
					makeContractValueHandler(nil, nil, nil),
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("contract and restricted type, reference", func(t *testing.T) {

		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
		      contract interface CI {
		          resource interface RI {}

		          resource R: RI {}

		          fun createR(): @R
		      }

              contract C: CI {
		          resource R: CI.RI {}

		          fun createR(): @R {
		              return <- create R()
		          }
		      }

              fun test() {
                  let r <- C.createR()
                  let ref: &CI.R = &r as &CI.R
                  let restrictedRef: &CI.R{CI.RI} = ref
                  destroy r
              }
            `,
			ParseCheckAndInterpretOptions{
				Options: []interpreter.Option{
					makeContractValueHandler(nil, nil, nil),
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)
	})
}
