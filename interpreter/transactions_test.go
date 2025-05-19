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

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestInterpretTransactions(t *testing.T) {

	t.Parallel()

	t.Run("no prepare", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
          transaction {
            execute {
              let x = 1 + 2
            }
          }
        `)

		err := inter.InvokeTransaction(nil)
		assert.NoError(t, err)
	})

	t.Run("prepare sets field", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
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

		err := inter.InvokeTransaction(nil)
		assert.NoError(t, err)
	})

	t.Run("succeeding pre-condition", func(t *testing.T) {

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

		err := inter.InvokeTransaction(nil)
		assert.NoError(t, err)
	})

	t.Run("failing pre-condition", func(t *testing.T) {

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

		err := inter.InvokeTransaction(nil)
		RequireError(t, err)

		var conditionErr *interpreter.ConditionError
		require.ErrorAs(t, err, &conditionErr)

		assert.Equal(t,
			ast.ConditionKindPre,
			conditionErr.ConditionKind,
		)
	})

	t.Run("succeeding post-condition", func(t *testing.T) {

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

		err := inter.InvokeTransaction(nil)
		assert.NoError(t, err)
	})

	t.Run("failing post-condition", func(t *testing.T) {

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

		err := inter.InvokeTransaction(nil)
		RequireError(t, err)

		var conditionErr *interpreter.ConditionError
		require.ErrorAs(t, err, &conditionErr)

		assert.Equal(t,
			ast.ConditionKindPost,
			conditionErr.ConditionKind,
		)
	})

	t.Run("too few signers", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
          transaction {
            prepare(signer: &Account) {}
          }
        `)

		signer := stdlib.NewAccountReferenceValue(
			inter,
			nil,
			interpreter.AddressValue{1},
			interpreter.UnauthorizedAccess,
			interpreter.EmptyLocationRange,
		)

		err := inter.InvokeTransaction(nil)
		assert.IsType(t, interpreter.ArgumentCountError{}, err)

		err = inter.InvokeTransaction(nil, signer)
		assert.NoError(t, err)
	})

	t.Run("too many signers, no prepare", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
          transaction {
            execute {}
          }
        `)

		signer1 := stdlib.NewAccountReferenceValue(
			inter,
			nil,
			interpreter.AddressValue{1},
			interpreter.UnauthorizedAccess,
			interpreter.EmptyLocationRange,
		)

		signer2 := stdlib.NewAccountReferenceValue(
			inter,
			nil,
			interpreter.AddressValue{2},
			interpreter.UnauthorizedAccess,
			interpreter.EmptyLocationRange,
		)

		err := inter.InvokeTransaction(nil)
		assert.NoError(t, err)

		err = inter.InvokeTransaction(nil, signer1)
		assert.IsType(t, interpreter.ArgumentCountError{}, err)

		err = inter.InvokeTransaction(nil, signer1, signer2)
		assert.IsType(t, interpreter.ArgumentCountError{}, err)
	})

	t.Run("too many signers, prepare requires one", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
          transaction {
            prepare(signer: &Account) {}

            execute {}
          }
        `)

		signer1 := stdlib.NewAccountReferenceValue(
			inter,
			nil,
			interpreter.AddressValue{1},
			interpreter.UnauthorizedAccess,
			interpreter.EmptyLocationRange,
		)

		signer2 := stdlib.NewAccountReferenceValue(
			inter,
			nil,
			interpreter.AddressValue{2},
			interpreter.UnauthorizedAccess,
			interpreter.EmptyLocationRange,
		)

		err := inter.InvokeTransaction(nil, signer1)
		require.NoError(t, err)

		err = inter.InvokeTransaction(nil, signer1, signer2)
		assert.IsType(t, interpreter.ArgumentCountError{}, err)
	})

	t.Run("parameters", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
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

		signer := stdlib.NewAccountReferenceValue(
			NoOpFunctionCreationContext{},
			nil,
			interpreter.AddressValue(address),
			interpreter.UnauthorizedAccess,
			interpreter.EmptyLocationRange,
		)

		err := inter.InvokeTransaction(arguments, signer)
		require.NoError(t, err)

		values := inter.GetGlobal("values")

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

	t.Run("enum", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            enum Alpha: Int {
                case A
                case B
            }

            let a = Alpha.A
            let b = Alpha.B

            let values: [AnyStruct] = []

            transaction(x: Alpha) {

                prepare(signer: &Account) {
                    values.append(signer.address)
                    values.append(x)
                    if x == Alpha.A {
                        values.append(Alpha.B)
                    } else {
                        values.append(-1)
                    }
                }
            }
        `)

		arguments := []interpreter.Value{
			inter.GetGlobal("a"),
		}

		address := common.MustBytesToAddress([]byte{0x1})

		signer := stdlib.NewAccountReferenceValue(
			NoOpFunctionCreationContext{},
			nil,
			interpreter.AddressValue(address),
			interpreter.UnauthorizedAccess,
			interpreter.EmptyLocationRange,
		)

		err := inter.InvokeTransaction(arguments, signer)
		require.NoError(t, err)

		values := inter.GetGlobal("values")

		require.IsType(t, &interpreter.ArrayValue{}, values)

		AssertValueSlicesEqual(
			t,
			inter,
			[]interpreter.Value{
				interpreter.AddressValue(address),
				inter.GetGlobal("a"),
				inter.GetGlobal("b"),
			},
			ArrayElements(inter, values.(*interpreter.ArrayValue)),
		)
	})
}

func TestInterpretInvalidTransferInExecute(t *testing.T) {

	t.Parallel()

	inter, _ := parseCheckAndInterpretWithOptions(t,
		`
          resource Dummy {}

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
        `,
		ParseCheckAndInterpretOptions{
			HandleCheckerError: func(err error) {
				errs := RequireCheckerErrors(t, err, 3)
				require.IsType(t, &sema.ResourceCapturingError{}, errs[0])
				require.IsType(t, &sema.ResourceCapturingError{}, errs[1])
				require.IsType(t, &sema.ResourceCapturingError{}, errs[2])
			},
		},
	)

	signer1 := stdlib.NewAccountReferenceValue(
		inter,
		nil,
		interpreter.AddressValue{1},
		interpreter.UnauthorizedAccess,
		interpreter.EmptyLocationRange,
	)

	err := inter.InvokeTransaction(nil, signer1)
	var invalidatedResourceError *interpreter.InvalidatedResourceError
	require.ErrorAs(t, err, &invalidatedResourceError)
}

func TestInterpretInvalidRecursiveTransferInExecute(t *testing.T) {

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

		err := inter.InvokeTransaction(nil)
		var invalidatedResourceReferenceError *interpreter.InvalidatedResourceReferenceError
		require.ErrorAs(t, err, &invalidatedResourceReferenceError)
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

		err := inter.InvokeTransaction(nil)
		var invalidatedResourceReferenceError *interpreter.InvalidatedResourceReferenceError
		require.ErrorAs(t, err, &invalidatedResourceReferenceError)
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

		err := inter.InvokeTransaction(nil)
		var invalidatedResourceReferenceError *interpreter.InvalidatedResourceReferenceError
		require.ErrorAs(t, err, &invalidatedResourceReferenceError)
	})
}
