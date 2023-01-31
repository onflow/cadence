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

package checker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckTransactions(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, code string, expectedErrors []error) {
		t.Parallel()

		_, err := ParseAndCheck(t, code)

		errs := RequireCheckerErrors(t, err, len(expectedErrors))

		for i, err := range errs {
			if !assert.IsType(t, expectedErrors[i], err) {
				t.Log(err)
			}
		}
	}

	t.Run("Empty", func(t *testing.T) {
		test(
			t,
			`
              transaction {}
            `,
			nil,
		)
	})

	t.Run("No-op", func(t *testing.T) {
		test(
			t,
			`
              transaction {}
            `,
			nil,
		)
	})

	t.Run("Simple", func(t *testing.T) {
		test(
			t,
			`
              transaction {

                execute {
                   let x = 1 + 2
                }
              }
            `,
			nil,
		)
	})

	t.Run("ValidPrepareParameters", func(t *testing.T) {
		test(
			t,
			`
              transaction {

                  prepare(x: AuthAccount, y: AuthAccount) {}
              }
            `,
			nil,
		)
	})

	t.Run("InvalidPrepareParameters", func(t *testing.T) {
		test(
			t,
			`
              transaction {

                prepare(x: Int, y: Int) {}
              }
            `,
			[]error{
				&sema.InvalidTransactionPrepareParameterTypeError{},
				&sema.InvalidTransactionPrepareParameterTypeError{},
			},
		)
	})

	t.Run("field, missing prepare", func(t *testing.T) {
		test(
			t,
			`
              transaction {

                var x: Int

                execute {
                    let y = self.x + 1
                }
              }
            `,
			[]error{
				&sema.MissingPrepareForFieldError{},
			},
		)
	})

	t.Run("field, missing prepare", func(t *testing.T) {
		test(
			t,
			`
              transaction {

                var x: Int

                prepare() {}
              }
            `,
			[]error{
				&sema.FieldUninitializedError{},
			},
		)
	})

	t.Run("field, prepare, execute", func(t *testing.T) {
		test(t,
			`
              transaction {

                  var x: Int

                  prepare() {
                      self.x = 5
                  }

                  execute {
                      let y = self.x + 1
                  }
              }
            `,
			nil,
		)
	})

	t.Run("PreConditions", func(t *testing.T) {
		test(t,
			`
              transaction {

                  var x: Int
                  var y: Int

                  prepare() {
                      self.x = 5
                      self.y = 10
                  }

                  pre {
                      self.x > 2
                      self.y < 20
                  }

                  execute {
                      let z = self.x + self.y
                  }
              }
            `,
			nil,
		)
	})

	t.Run("PostConditions", func(t *testing.T) {
		test(t,
			`
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
            `,
			nil,
		)
	})

	t.Run("InvalidPostConditionsAccessExecuteScope", func(t *testing.T) {

		test(t,
			`
              transaction {

                  execute {
                      var x = 5
                  }

                  post {
                      x == 5
                  }
              }
            `,
			[]error{
				&sema.NotDeclaredError{},
			},
		)
	})

	// TODO: prevent self from being used in function
	// {
	//     "IllegalSelfUsage",
	//     `
	//        fun foo(x: AnyStruct) {}
	//
	//       transaction {
	//         execute {
	//           foo(x: self)
	//         }
	//       }
	//     `,
	//     []error{
	//         &sema.CheckerError{},
	//     },
	// },

	t.Run("ResourceField", func(t *testing.T) {
		test(t,
			`
              resource R {}

              transaction {

                var r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    destroy self.r
                }
              }
            `,
			nil,
		)
	})

	t.Run("InvalidResourceFieldLoss", func(t *testing.T) {
		test(t,
			`
              resource R {}

              transaction {

                  var r: @R

                  prepare() {
                      self.r <- create R()
                  }

                  execute {}
              }
            `,
			[]error{
				&sema.ResourceFieldNotInvalidatedError{},
			},
		)
	})

	t.Run("ParameterUse", func(t *testing.T) {
		test(t,
			`
              transaction(x: Bool) {

                  prepare() {
                      x
                  }

                  pre {
                      x
                  }

                  execute {
                      x
                  }

                  post {
                      x
                  }
              }
            `,
			nil,
		)
	})

	t.Run("InvalidParameterUseAfterDeclaration", func(t *testing.T) {
		test(t,
			`
              transaction(x: Bool) {}

              let y = x
            `,
			[]error{
				&sema.NotDeclaredError{},
			},
		)
	})

	t.Run("InvalidResourceParameter", func(t *testing.T) {
		test(t,
			`
              resource R {}

              transaction(rs: @[R]) {}
            `,
			[]error{
				&sema.InvalidNonImportableTransactionParameterTypeError{},
				&sema.ResourceLossError{},
			},
		)
	})

	t.Run("InvalidNonStorableParameter", func(t *testing.T) {
		test(t,
			`
              transaction(x: ((Int): Int)) {
                execute {
                  x(0)
                }
              }
            `,
			[]error{
				&sema.InvalidNonImportableTransactionParameterTypeError{},
			},
		)
	})

	t.Run("invalid access modifier for field", func(t *testing.T) {
		test(
			t,
			`
              transaction {

                priv var x: Int

                prepare() {
                    self.x = 1
                }
              }
            `,
			[]error{
				&sema.InvalidAccessModifierError{},
			},
		)
	})
}

func TestCheckTransactionRoles(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          transaction {

              role foo {}
          }
        `)

		require.NoError(t, err)
	})

	t.Run("field, prepare", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          transaction {

              role foo {

                  let bar: Int

                  prepare() {
                      self.bar = 1
                  }
              }
          }
        `)
		require.NoError(t, err)
	})

	t.Run("field, missing prepare", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          transaction {

              role foo {

                  let bar: Int
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.MissingPrepareForFieldError{}, errs[0])
	})

	t.Run("field, prepare, missing initialization", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          transaction {

              role foo {

                  let bar: Int

                  prepare() {}
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.FieldUninitializedError{}, errs[0])
	})

	t.Run("duplicate", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          transaction {

              role foo {}

              role foo {}
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.DuplicateTransactionRoleError{}, errs[0])
	})

	t.Run("field name conflict", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          transaction {

              let foo: Int

              prepare() {
                  self.foo = 1
              }

              role foo {}
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TransactionRoleWithFieldNameError{}, errs[0])
	})

	t.Run("multiple roles, one field each, execute", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          transaction {

              role foo {

                  let bar: Int

                  prepare() {
                      self.bar = 1
                  }
              }

              role baz {

                  let blub: String

                  prepare() {
                      self.blub = "2"
                  }
              }

              execute {
                  let bar: Int = self.foo.bar
                  let blub: String = self.baz.blub
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("invalid prepare parameter type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          transaction {

              prepare(signer: AuthAccount) {}

              role role1 {
                  let foo: Int

                  prepare(foo: Int) {
                      self.foo = foo
                  }
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidTransactionPrepareParameterTypeError{}, errs[0])
	})

	t.Run("missing role", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          transaction {

              prepare(first: AuthAccount, second: AuthAccount) {}

              role role1 {
                  prepare(signer: AuthAccount) {}
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.RoleCountMismatchError{}, errs[0])
	})

	t.Run("too many roles", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          transaction {

              prepare(signer: AuthAccount) {}

              role role1 {
                  prepare(signer: AuthAccount) {}
              }

              role role2 {
                  prepare(signer: AuthAccount) {}
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.RoleCountMismatchError{}, errs[0])
	})

	t.Run("correct number of roles", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          transaction {

              prepare(first: AuthAccount, second: AuthAccount) {}

              role role1 {
                  prepare(signer: AuthAccount) {}
              }

              role role2 {
                  prepare(signer: AuthAccount) {}
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("matching prepare", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          transaction {

              prepare(signer: AuthAccount) {}

              role role1 {
                  prepare(signer: AuthAccount) {}
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("missing prepare", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          transaction {

              prepare(signer: AuthAccount) {}

              role role1 {}
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.PrepareParameterCountMismatchError{}, errs[0])
	})

	t.Run("fewer prepare parameters", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          transaction {

              prepare(signer: AuthAccount) {}

              role role1 {
                  prepare() {}
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.PrepareParameterCountMismatchError{}, errs[0])
	})

	t.Run("more prepare parameters", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          transaction {

              prepare(signer: AuthAccount) {}

              role role1 {
                  prepare(firstSigner: AuthAccount, secondSigner: AuthAccount) {}
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.PrepareParameterCountMismatchError{}, errs[0])
	})

	t.Run("prepare, but no transaction prepare ", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          transaction {
              role role1 {
                  prepare(signer: AuthAccount) {}
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.PrepareParameterCountMismatchError{}, errs[0])
	})

	t.Run("roles are not required", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          transaction {

              prepare(first: AuthAccount, second: AuthAccount) {}
          }
        `)

		require.NoError(t, err)
	})

	t.Run("transaction parameter usage", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          transaction(foo: Int) {

              role role1 {
                  let foo: Int

                  prepare() {
                      self.foo = foo
                  }
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("resource field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun absorb(_ r: @R) {
              destroy r
          }

          transaction {

              role role1 {
                  var r: @R

                  prepare() {
                      self.r <- create R()
                  }
              }

              execute {
                  absorb(<-self.role1.r)
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("invalid resource field loss", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          transaction {

              role role1 {
                  var r: @R

                  prepare() {
                      self.r <- create R()
                  }
              }

              execute {}
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("resource field use after invalidation", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun absorb(_ r: @R) {
              destroy r
          }

          transaction {

              role role1 {
                  var r: @R

                  prepare() {
                      self.r <- create R()
                  }
              }

              execute {
                  absorb(<-self.role1.r)
                  absorb(<-self.role1.r)
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	})

	t.Run("resource field with nested resource", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R1 {
              var r2: @R2

              init() {
                  self.r2 <- create R2()
              }

              destroy() {
                  destroy self.r2
              }
          }

          resource R2 {}

          fun absorb(_ r: @AnyResource) {
              destroy r
          }

          transaction {

              role role1 {
                  var r1: @R1

                  prepare() {
                      self.r1 <- create R1()
                  }
              }

              execute {
                  absorb(<-self.role1.r1)
              }
          }
        `)
		require.NoError(t, err)
	})

	t.Run("nested resource field", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R1 {
              var r2: @R2

              init() {
                  self.r2 <- create R2()
              }

              destroy() {
                  destroy self.r2
              }
          }

          resource R2 {}

          fun absorb(_ r: @AnyResource) {
              destroy r
          }

          transaction {

              role role1 {
                  var r1: @R1

                  prepare() {
                      self.r1 <- create R1()
                  }
              }

              execute {
                  absorb(<-self.role1.r1.r2)
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		var invalidNestedResourceErr *sema.InvalidNestedResourceMoveError
		require.ErrorAs(t, errs[0], &invalidNestedResourceErr)
		assert.ErrorContains(t, invalidNestedResourceErr, "field `r2`")

		require.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})
}

func TestCheckTransactionExecuteScope(t *testing.T) {

	t.Parallel()

	// non-global variable declarations do not require access modifiers
	// execute block should be treated like function block

	_, err := ParseAndCheckWithOptions(
		t,
		`
          transaction {

              execute {
                  let code: Int = 1
              }
          }
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				AccessCheckMode: sema.AccessCheckModeStrict,
			},
		},
	)

	assert.NoError(t, err)
}

func TestCheckInvalidTransactionSelfMoveToFunction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      transaction {

          execute {
              use(self)
          }
      }

      fun use(_ any: AnyStruct) {}
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidMoveError{}, errs[0])
}

func TestCheckInvalidTransactionSelfMove(t *testing.T) {

	t.Parallel()

	t.Run("variable declaration", func(t *testing.T) {

		_, err := ParseAndCheck(t, `

          transaction {

              execute {
                  let x = self
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMoveError{}, errs[0])
	})

	t.Run("return from function", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          transaction {

              execute {
                  return self
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
		typeMismatchErr := errs[0].(*sema.TypeMismatchError)

		assert.Equal(t, sema.VoidType, typeMismatchErr.ExpectedType)
		assert.IsType(t, &sema.TransactionType{}, typeMismatchErr.ActualType)
	})

	t.Run("into array literal", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          transaction {

              execute {
                  let txs = [self]
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMoveError{}, errs[0])
	})

	t.Run("into dictionary literal", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          transaction {

              execute {
                  let txs = {"self": self}
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMoveError{}, errs[0])
	})

	t.Run("into function", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

              fun test(_ x: AnyStruct) {}

              transaction {

                  execute {
                      test(self)
                  }
              }
            `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidMoveError{}, errs[0])
	})
}

func TestCheckInvalidTransactionRoleSelfMove(t *testing.T) {

	t.Parallel()

	t.Run("direct", func(t *testing.T) {

		t.Parallel()

		t.Run("variable declaration", func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t, `

              transaction {

                  role role1 {
                      prepare() {
                          let x = self
                      }
                  }
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.InvalidMoveError{}, errs[0])
		})

		t.Run("into array literal", func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t, `

              transaction {

                  role role1 {
                      prepare() {
                          let txs = [self]
                      }
                  }
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.InvalidMoveError{}, errs[0])
		})

		t.Run("into dictionary literal", func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t, `

              transaction {

                  role role1 {
                      prepare() {
                          let txs = {"self": self}
                      }
                  }
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.InvalidMoveError{}, errs[0])
		})

		t.Run("into function", func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t, `

              fun test(_ x: AnyStruct) {}

              transaction {

                  role role1 {
                      prepare() {
                          test(self)
                      }
                  }
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.InvalidMoveError{}, errs[0])
		})
	})

	t.Run("indirect", func(t *testing.T) {
		t.Parallel()

		t.Run("variable declaration", func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t, `

              transaction {

                  role role1 {}

                  execute {
                      let x = self.role1
                  }
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.InvalidMoveError{}, errs[0])
		})

		t.Run("into array literal", func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t, `

              transaction {

                  role role1 {}

                  execute {
                      let roles = [self.role1]
                  }
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.InvalidMoveError{}, errs[0])
		})

		t.Run("into dictionary literal", func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t, `

              transaction {

                  role role1 {}

                  execute {
                      let roles = {"role1": self.role1}
                  }
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.InvalidMoveError{}, errs[0])
		})

		t.Run("into function", func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t, `

              fun test(_ x: AnyStruct) {}

              transaction {

                  role role1 {}

                  execute {
                      test(self.role1)
                  }
              }
            `)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.InvalidMoveError{}, errs[0])
		})
	})
}
