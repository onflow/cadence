/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

	"github.com/onflow/cadence/runtime/activations"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	. "github.com/onflow/cadence/runtime/tests/utils"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/interpreter"
)

func TestInterpretTransactions(t *testing.T) {

	t.Parallel()

	t.Run("no prepare function", func(t *testing.T) {

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

	t.Run("field and prepare", func(t *testing.T) {

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

		err := inter.InvokeTransaction(0)
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

		err := inter.InvokeTransaction(0)
		RequireError(t, err)

		var conditionErr interpreter.ConditionError
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

		err := inter.InvokeTransaction(0)
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

		err := inter.InvokeTransaction(0)
		RequireError(t, err)

		var conditionErr interpreter.ConditionError
		require.ErrorAs(t, err, &conditionErr)

		assert.Equal(t,
			ast.ConditionKindPost,
			conditionErr.ConditionKind,
		)
	})

	t.Run("multiple transactions", func(t *testing.T) {

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

	t.Run("invocation with too few arguments", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {
            prepare(signer: AuthAccount) {}
          }
        `)

		err := inter.InvokeTransaction(0)
		assert.IsType(t, interpreter.ArgumentCountError{}, err)
	})

	t.Run("invocation with too many arguments", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {
            execute {}
          }

          transaction {
            prepare(signer: AuthAccount) {}

            execute {}
          }
        `)

		signer1 := newTestAuthAccountValue(
			nil,
			interpreter.AddressValue{0, 0, 0, 0, 0, 0, 0, 1},
		)
		signer2 := newTestAuthAccountValue(
			nil,
			interpreter.AddressValue{0, 0, 0, 0, 0, 0, 0, 2},
		)

		// first transaction
		err := inter.InvokeTransaction(0, signer1)
		assert.IsType(t, interpreter.ArgumentCountError{}, err)

		// second transaction
		err = inter.InvokeTransaction(0, signer1, signer2)
		assert.IsType(t, interpreter.ArgumentCountError{}, err)
	})

	t.Run("transaction parameters", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let values: [AnyStruct] = []

          transaction(x: Int, y: Bool) {

            prepare(signer: AuthAccount) {
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

		prepareArguments := []interpreter.Value{
			newTestAuthAccountValue(
				nil,
				interpreter.AddressValue{},
			),
		}

		arguments = append(arguments, prepareArguments...)

		err := inter.InvokeTransaction(0, arguments...)
		assert.NoError(t, err)

		values := inter.Globals.Get("values").GetValue()

		require.IsType(t, &interpreter.ArrayValue{}, values)

		AssertValueSlicesEqual(
			t,
			inter,
			[]interpreter.Value{
				interpreter.AddressValue{},
				interpreter.TrueValue,
				interpreter.NewUnmeteredIntValueFromInt64(1),
			},
			arrayElements(inter, values.(*interpreter.ArrayValue)),
		)
	})
}

func TestInterpretTransactionRoles(t *testing.T) {

	t.Parallel()

	t.Run("single role with field", func(t *testing.T) {

		t.Parallel()

		var logs []string

		valueDeclaration := stdlib.NewStandardLibraryFunction(
			"log",
			stdlib.LogFunctionType,
			"",
			func(invocation interpreter.Invocation) interpreter.Value {
				firstArgument := invocation.Arguments[0]
				message := firstArgument.(*interpreter.StringValue).Str
				logs = append(logs, message)
				return interpreter.Void
			},
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(valueDeclaration)

		baseActivation := activations.NewActivation[*interpreter.Variable](nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, valueDeclaration)

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              transaction(a: String, b: String) {

                  let foo: String

                  prepare(signer: AuthAccount) {
                      log("a 1")
                      log(a)
                      log("b 1")
                      log(b)
                      self.foo = signer.address.toString()
                      log("self.foo 1")
                      log(self.foo)
                  }

                  role role1 {
                      let bar: String

                      prepare(signer: AuthAccount) {
                          log("a 2")
                          log(a)
                          log("b 2")
                          log(b)
                          self.bar = signer.address.toString()
                          log("self.bar")
                          log(self.bar)
                      }
                  }

                  execute {
                      log("self.foo 2")
                      log(self.foo)
                      log("self.role1.bar")
                      log(self.role1.bar)
                  }
              }
            `,
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

		signer := newTestAuthAccountValue(
			nil,
			interpreter.AddressValue{0, 0, 0, 0, 0, 0, 0, 1},
		)

		err = inter.InvokeTransaction(
			0,
			interpreter.NewUnmeteredStringValue("A"),
			interpreter.NewUnmeteredStringValue("B"),
			signer,
		)
		assert.NoError(t, err)

		assert.Equal(t,
			[]string{
				// transaction prepare
				"a 1",
				"A",
				"b 1",
				"B",
				"self.foo 1",
				"0x0000000000000001",
				// role prepare
				"a 2",
				"A",
				"b 2",
				"B",
				"self.bar",
				"0x0000000000000001",
				// execute
				"self.foo 2",
				"0x0000000000000001",
				"self.role1.bar",
				"0x0000000000000001",
			},
			logs,
		)
	})

	t.Run("multiple roles, each with a field", func(t *testing.T) {

		t.Parallel()

		var logs []string

		valueDeclaration := stdlib.NewStandardLibraryFunction(
			"log",
			stdlib.LogFunctionType,
			"",
			func(invocation interpreter.Invocation) interpreter.Value {
				firstArgument := invocation.Arguments[0]
				message := firstArgument.(*interpreter.StringValue).Str
				logs = append(logs, message)
				return interpreter.Void
			},
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(valueDeclaration)

		baseActivation := activations.NewActivation[*interpreter.Variable](nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, valueDeclaration)

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              transaction(a: String, b: String) {

                  let foo: String

                  prepare(signer: AuthAccount) {
                      log("a 1")
                      log(a)
                      log("b 1")
                      log(b)
                      self.foo = signer.address.toString()
                      log("self.foo 1")
                      log(self.foo)
                  }

                  role role1 {
                      let bar: String

                      prepare(signer: AuthAccount) {
                          log("a 2")
                          log(a)
                          log("b 2")
                          log(b)
                          self.bar = signer.address.toString()
                          log("self.bar")
                          log(self.bar)
                      }
                  }

                  role role2 {
                      let baz: String

                      prepare(signer: AuthAccount) {
                          log("a 3")
                          log(a)
                          log("b 3")
                          log(b)
                          self.baz = signer.address.toString()
                          log("self.baz")
                          log(self.baz)
                      }
                  }

                  execute {
                      log("self.foo 2")
                      log(self.foo)
                      log("self.role1.bar")
                      log(self.role1.bar)
                      log("self.role2.baz")
                      log(self.role2.baz)
                  }
              }
            `,
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

		signer := newTestAuthAccountValue(
			nil,
			interpreter.AddressValue{0, 0, 0, 0, 0, 0, 0, 1},
		)

		err = inter.InvokeTransaction(
			0,
			interpreter.NewUnmeteredStringValue("A"),
			interpreter.NewUnmeteredStringValue("B"),
			signer,
		)
		assert.NoError(t, err)

		assert.Equal(t,
			[]string{
				// transaction prepare
				"a 1",
				"A",
				"b 1",
				"B",
				"self.foo 1",
				"0x0000000000000001",
				// role1 prepare
				"a 2",
				"A",
				"b 2",
				"B",
				"self.bar",
				"0x0000000000000001",
				// role2 prepare
				"a 3",
				"A",
				"b 3",
				"B",
				"self.baz",
				"0x0000000000000001",
				// execute
				"self.foo 2",
				"0x0000000000000001",
				"self.role1.bar",
				"0x0000000000000001",
				"self.role2.baz",
				"0x0000000000000001",
			},
			logs,
		)
	})

}
