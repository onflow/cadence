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

	t.Run("InvalidFieldUninitialized", func(t *testing.T) {
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
				&sema.TransactionMissingPrepareError{},
			},
		)
	})

	t.Run("FieldInitialized", func(t *testing.T) {
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

	t.Run("PreConditions must be view", func(t *testing.T) {
		test(t,
			`
              transaction {
				  var foo: ((): Int)

                  prepare() {
					  self.foo = fun (): Int {
						return 40
					  }
                  }

                  pre {
					  self.foo() > 30
                  }
              }
            `,
			[]error{
				&sema.PurityError{},
			},
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

	t.Run("PostConditions must be view", func(t *testing.T) {
		test(t,
			`
              transaction {
				  var foo: ((): Int)

                  prepare() {
					  self.foo = fun (): Int {
						return 40
					  }
                  }

                  post {
					  self.foo() > 30
                  }
              }
            `,
			[]error{
				&sema.PurityError{},
			},
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
	// 	"IllegalSelfUsage",
	// 	`
	//  	  fun foo(x: AnyStruct) {}
	//
	// 	  transaction {
	// 	    execute {
	// 		  foo(x: self)
	// 		}
	// 	  }
	// 	`,
	// 	[]error{
	// 		&sema.CheckerError{},
	// 	},
	// },

	t.Run("ResourceField", func(t *testing.T) {
		test(t,
			`
              resource R {}

              transaction {

                var x: @R

                prepare() {
                    self.x <- create R()
                }

                execute {
                    destroy self.x
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

                  var x: @R

                  prepare() {
                      self.x <- create R()
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
		      transaction(x: fun(Int): Int) {
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

func TestCheckInvalidTransactionSelfMoveInVariableDeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

     transaction {

         execute {
             let x = self
         }
     }
   `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidMoveError{}, errs[0])
}

func TestCheckInvalidTransactionSelfMoveReturnFromFunction(t *testing.T) {

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
}

func TestCheckInvalidTransactionSelfMoveIntoArrayLiteral(t *testing.T) {

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
}

func TestCheckInvalidTransactionSelfMoveIntoDictionaryLiteral(t *testing.T) {

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
}
