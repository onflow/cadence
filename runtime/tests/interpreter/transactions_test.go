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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/checker"
	. "github.com/onflow/cadence/runtime/tests/utils"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/interpreter"
)

func TestInterpretTransactions(t *testing.T) {

	t.Parallel()

	t.Run("NoPrepareFunction", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {
            execute {
              let x = 1 + 2
            }
          }
        `)

		err := inter.InvokeTransaction(0)
		assert.NoError(t, err)
	})

	t.Run("SetTransactionField", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {

            var x: Int

            prepare() {
              self.x = 5
            }

            execute {
              let y = self.x + 1
            }
          }
        `)

		err := inter.InvokeTransaction(0)
		assert.NoError(t, err)
	})

	t.Run("PreConditions", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {

            var x: Int

            prepare() {
              self.x = 5
            }

            pre {
              self.x > 1
            }
          }
        `)

		err := inter.InvokeTransaction(0)
		assert.NoError(t, err)
	})

	t.Run("FailingPreConditions", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {

            var x: Int

            prepare() {
              self.x = 5
            }

            pre {
              self.x > 10
            }
          }
        `)

		err := inter.InvokeTransaction(0)
		RequireError(t, err)

		var conditionErr interpreter.ConditionError
		require.ErrorAs(t, err, &conditionErr)

		assert.Equal(t,
			ast.ConditionKindPre,
			conditionErr.ConditionKind,
		)
	})

	t.Run("PostConditions", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {

            var x: Int

            prepare() {
              self.x = 5
            }

            execute {
              self.x = 10
            }

            post {
              self.x == 10
            }
          }
        `)

		err := inter.InvokeTransaction(0)
		assert.NoError(t, err)
	})

	t.Run("FailingPostConditions", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {

            var x: Int

            prepare() {
              self.x = 5
            }

            execute {
              self.x = 10
            }

            post {
              self.x == 5
            }
          }
        `)

		err := inter.InvokeTransaction(0)
		RequireError(t, err)

		var conditionErr interpreter.ConditionError
		require.ErrorAs(t, err, &conditionErr)

		assert.Equal(t,
			ast.ConditionKindPost,
			conditionErr.ConditionKind,
		)
	})

	t.Run("MultipleTransactions", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {
            execute {
              let x = 1 + 2
            }
          }

          transaction {
            execute {
              let y = 3 + 4
            }
          }
        `)

		// first transaction
		err := inter.InvokeTransaction(0)
		assert.NoError(t, err)

		// second transaction
		err = inter.InvokeTransaction(1)
		assert.NoError(t, err)

		// third transaction is not declared
		err = inter.InvokeTransaction(2)
		assert.IsType(t, interpreter.TransactionNotDeclaredError{}, err)
	})

	t.Run("TooFewArguments", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {
            prepare(signer: &Account) {}
          }
        `)

		err := inter.InvokeTransaction(0)
		assert.IsType(t, interpreter.ArgumentCountError{}, err)
	})

	t.Run("TooManyArguments", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {
            execute {}
          }

          transaction {
            prepare(signer: &Account) {}

            execute {}
          }
        `)

		signer1 := stdlib.NewAccountReferenceValue(nil, nil, interpreter.AddressValue{1}, interpreter.UnauthorizedAccess, interpreter.EmptyLocationRange)
		signer2 := stdlib.NewAccountReferenceValue(nil, nil, interpreter.AddressValue{2}, interpreter.UnauthorizedAccess, interpreter.EmptyLocationRange)

		// first transaction
		err := inter.InvokeTransaction(0, signer1)
		assert.IsType(t, interpreter.ArgumentCountError{}, err)

		// second transaction
		err = inter.InvokeTransaction(0, signer1, signer2)
		assert.IsType(t, interpreter.ArgumentCountError{}, err)
	})

	t.Run("Parameters", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let values: [AnyStruct] = []

          transaction(x: Int, y: Bool) {

            prepare(signer: &Account) {
              values.append(signer.address)
              values.append(y)
              values.append(x)
            }
          }
        `)

		arguments := []interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.TrueValue,
		}

		address := common.MustBytesToAddress([]byte{0x1})

		account := stdlib.NewAccountReferenceValue(
			nil,
			nil,
			interpreter.AddressValue(address),
			interpreter.UnauthorizedAccess,
			interpreter.EmptyLocationRange,
		)

		prepareArguments := []interpreter.Value{account}

		arguments = append(arguments, prepareArguments...)

		err := inter.InvokeTransaction(0, arguments...)
		require.NoError(t, err)

		values := inter.Globals.Get("values").GetValue(inter)

		require.IsType(t, &interpreter.ArrayValue{}, values)

		AssertValueSlicesEqual(
			t,
			inter,
			[]interpreter.Value{
				interpreter.AddressValue(address),
				interpreter.TrueValue,
				interpreter.NewUnmeteredIntValueFromInt64(1),
			},
			ArrayElements(inter, values.(*interpreter.ArrayValue)),
		)
	})
}

func TestRuntimeInvalidTransferInExecute(t *testing.T) {

	t.Parallel()

	inter, _ := parseCheckAndInterpretWithOptions(t, `
		access(all) resource Dummy {}

		transaction {
			var vaults: @[AnyResource]
			var account: auth(Storage) &Account

			prepare(account: auth(Storage) &Account) {
				self.vaults <- [<-create Dummy(), <-create Dummy()]
				self.account = account
			}

			execute {
				let x = fun(): @[AnyResource] {
					var x <- self.vaults <- [<-create Dummy()]
					return <-x
				}

				var t <-  self.vaults[0] <- self.vaults 
				destroy t
				self.account.storage.save(<- x(), to: /storage/x42)
			}
		}
	`, ParseCheckAndInterpretOptions{
		HandleCheckerError: func(err error) {
			errs := checker.RequireCheckerErrors(t, err, 3)
			require.IsType(t, &sema.ResourceCapturingError{}, errs[0])
			require.IsType(t, &sema.ResourceCapturingError{}, errs[1])
			require.IsType(t, &sema.ResourceCapturingError{}, errs[2])
		},
	})

	signer1 := stdlib.NewAccountReferenceValue(nil, nil, interpreter.AddressValue{1}, interpreter.UnauthorizedAccess, interpreter.EmptyLocationRange)
	err := inter.InvokeTransaction(0, signer1)
	require.ErrorAs(t, err, &interpreter.InvalidatedResourceError{})
}

func TestRuntimeInvalidRecursiveTransferInExecute(t *testing.T) {

	t.Parallel()

	t.Run("Array", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			transaction {
				var arr: @[AnyResource]

				prepare() {
					self.arr <- []
				}

				execute {
					self.arr.append(<-self.arr)
				}
			}
		`)

		err := inter.InvokeTransaction(0)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("Dictionary", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			transaction {
				var dict: @{String: AnyResource}

				prepare() {
					self.dict <- {}
				}

				execute {
					destroy self.dict.insert(key: "", <-self.dict)
				}
			}
		`)

		err := inter.InvokeTransaction(0)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			resource R {
				fun foo(_ r: @R) {
					destroy r
				}
			}

			transaction {
				var r: @R

				prepare() {
					self.r <- create R()
				}

				execute {
					self.r.foo(<-self.r) 
				}
			}
		`)

		err := inter.InvokeTransaction(0)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})
}
