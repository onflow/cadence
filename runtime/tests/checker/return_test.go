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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

func TestCheckInvalidReturnValue(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       fun test() {
           return 1
       }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	require.IsType(t, &sema.TypeMismatchError{}, errs[0])
	typeMismatchErr := errs[0].(*sema.TypeMismatchError)

	assert.Equal(t, sema.VoidType, typeMismatchErr.ExpectedType)
	assert.Equal(t, sema.IntType, typeMismatchErr.ActualType)
}

func TestCheckMissingReturnStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): Int {}
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.MissingReturnStatementError{}, errs[0])
}

func TestCheckReturnStatementMissingValue(t *testing.T) {

	t.Parallel()

	t.Run("valid return type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(): Int {
              return
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.MissingReturnValueError{}, errs[0])
	})

	t.Run("invalid return type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(): X {
              return
          }
        `)

		errs := ExpectCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.IsType(t, &sema.MissingReturnValueError{}, errs[1])
	})
}

func TestCheckReturnStatementTypeMismatch(t *testing.T) {

	t.Parallel()

	t.Run("invalid return type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(): X {
              return 1
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("invalid value type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(): Int {
              return x
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("invalid value type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(): Int {
              return true
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}

func TestCheckMissingReturnStatementInterfaceFunction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        struct interface Test {
            fun test(x: Int): Int {
                pre {
                    x != 0
                }
            }
        }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidMissingReturnStatementStructFunction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        struct Test {
            pub(set) var foo: Int

            init(foo: Int) {
                self.foo = foo
            }

            pub fun getFoo(): Int {
                if 2 > 1 {
                    return 0
                }
            }
        }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.MissingReturnStatementError{}, errs[0])
}

type exitTest struct {
	body              string
	exits             bool
	valueDeclarations []sema.ValueDeclaration
	errors            []error
}

func testExits(t *testing.T, test exitTest) {

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	for _, valueDeclaration := range test.valueDeclarations {
		baseValueActivation.DeclareValue(valueDeclaration)
	}

	code := fmt.Sprintf("fun test(): AnyStruct {\n%s\n}", test.body)
	_, err := ParseAndCheckWithOptions(
		t,
		code,
		ParseAndCheckOptions{
			Config: &sema.Config{
				BaseValueActivation: baseValueActivation,
			},
		},
	)

	if test.errors != nil {
		errs := ExpectCheckerErrors(t, err, len(test.errors))
		for i, err := range errs {
			assert.IsType(t, test.errors[i], err)
		}
	} else if test.exits {
		require.NoError(t, err)
	} else {
		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.MissingReturnStatementError{}, errs[0])
	}

}

func TestCheckReturnStatementExits(t *testing.T) {

	t.Parallel()

	t.Run("with value", func(t *testing.T) {

		t.Parallel()

		testExits(t, exitTest{
			body:  "return 1",
			exits: true,
		})
	})

	t.Run("without value", func(t *testing.T) {

		t.Parallel()

		testExits(t, exitTest{
			body:  "return",
			exits: true,
			errors: []error{
				&sema.MissingReturnValueError{},
			},
		})
	})
}

func TestCheckIfStatementExits(t *testing.T) {

	t.Parallel()

	t.Run("only in then branch", func(t *testing.T) {

		t.Parallel()

		testExits(t, exitTest{
			body: `
              if true {
                  return 1
              }
            `,
			exits: false,
		})
	})

	t.Run("only in else branch", func(t *testing.T) {

		t.Parallel()

		testExits(t, exitTest{
			body: `
              var x = 1
              if true {
                  x = 2
              } else {
                  return 2
              }
            `,
			exits: false,
		})
	})

	t.Run("in nested then branch", func(t *testing.T) {

		t.Parallel()

		testExits(t, exitTest{
			body: `
              if true {
                  if true {
                      return 1
                  }
              }
            `,
			exits: false,
		})
	})

	t.Run("return in both branches", func(t *testing.T) {

		t.Parallel()

		testExits(t, exitTest{
			body: `
              if 2 > 1 {
                  return 1
              } else {
                  return 2
              }
            `,
			exits: true,
		})
	})

	t.Run("return after then branch", func(t *testing.T) {

		t.Parallel()

		testExits(t, exitTest{
			body: `
              if 2 > 1 {
                  return 1
              }
              return 2
            `,
			exits: true,
		})
	})
}

func TestCheckWhileStatementExits(t *testing.T) {

	t.Parallel()

	t.Run("none", func(t *testing.T) {

		t.Parallel()

		testExits(t, exitTest{
			body: `
              var x = 1
              var y = 2
              while true {
                  x = y
              }
            `,
			exits: false,
		})
	})

	t.Run("break", func(t *testing.T) {

		t.Parallel()

		testExits(t, exitTest{
			body: `
              var x = 1
              var y = 2
              while true {
                  x = y
                  break
              }
            `,
			exits: false,
		})
	})

	t.Run("in body, missing value", func(t *testing.T) {

		t.Parallel()

		testExits(t, exitTest{
			body: `
              while 2 > 1 {
                  return
              }
            `,
			exits: false,
			errors: []error{
				&sema.MissingReturnValueError{},
				&sema.MissingReturnStatementError{},
			},
		})
	})

	t.Run("in body", func(t *testing.T) {

		t.Parallel()

		testExits(t, exitTest{
			body: `
              var x = 0
              while x < 10 {
                  return x
              }
            `,
			exits: false,
		})
	})
}

func TestCheckNeverInvocationExits(t *testing.T) {

	t.Parallel()

	valueDeclarations := []sema.ValueDeclaration{
		stdlib.PanicFunction,
	}

	t.Run("expression statement", func(t *testing.T) {

		t.Parallel()

		testExits(t, exitTest{
			body: `
              panic("")
            `,
			exits:             true,
			valueDeclarations: valueDeclarations,
		})
	})

	t.Run("expression in if-statement, definitely evaluated", func(t *testing.T) {

		t.Parallel()

		testExits(t, exitTest{
			body: `
              if panic("") {}
            `,
			exits:             true,
			valueDeclarations: valueDeclarations,
		})
	})

	t.Run("expression in while-statement, definitely evaluated", func(t *testing.T) {

		t.Parallel()

		testExits(t, exitTest{
			body: `
              while panic("") {}
            `,
			exits:             true,
			valueDeclarations: valueDeclarations,
		})
	})

	t.Run("expression, potentially evaluated", func(t *testing.T) {

		t.Parallel()

		testExits(t, exitTest{
			body: `
              let x: Int? = 1
              let y = x ?? panic("")
            `,
			exits:             false,
			valueDeclarations: valueDeclarations,
		})
	})

	t.Run("statement, potentially evaluated", func(t *testing.T) {

		t.Parallel()

		testExits(t, exitTest{
			body:`
              if true {
                  return nil
              }
              panic("")
            `,
			exits:             true,
			valueDeclarations: valueDeclarations,
		})
	})

}

// TestCheckNestedFunctionExits tests if a function with a return statement
// nested inside another function does not influence the containing function
func TestCheckNestedFunctionExits(t *testing.T) {

	t.Parallel()

	testExits(t, exitTest{
		body: `
          fun (): Int {
              return 1
          }
        `,
		// NOTE: inner function returns, but outer does not
		exits: false,
	})
}
