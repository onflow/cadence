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

func TestCheckOptionalChainingNonOptionalFieldRead(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      struct Test {
          let x: Int

          init(x: Int) {
              self.x = x
          }
      }

      let test: Test? = Test(x: 1)
      let x = test?.x
    `)

	require.NoError(t, err)

	assert.Equal(t,
		&sema.OptionalType{Type: sema.IntType},
		RequireGlobalValue(t, checker.Elaboration, "x"),
	)
}

func TestCheckOptionalChainingOptionalFieldRead(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      struct Test {
          let x: Int?

          init(x: Int?) {
              self.x = x
          }
      }

      let test: Test? = Test(x: 1)
      let x = test?.x
    `)

	require.NoError(t, err)

	assert.Equal(t,
		&sema.OptionalType{Type: sema.IntType},
		RequireGlobalValue(t, checker.Elaboration, "x"),
	)
}

func TestCheckOptionalChainingNonOptionalFieldAccess(t *testing.T) {

	t.Parallel()

	t.Run("function", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
              fun test() {
                  let bar = Bar()
                  // field Bar.foo is not optional but try to access it through optional chaining
                  bar.foo?.getContent()
              }

              struct Bar {
                  var foo: Foo
                  init() {
                      self.foo = Foo()
                  }
              }

              struct Foo {
                  fun getContent(): String {
                      return "hello"
                  }
              }
            `,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidOptionalChainingError{}, errs[0])

	})

	t.Run("non-function", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
              fun test() {
                  let bar = Bar()
                  // Two issues:
                  //    - Field Bar.foo is not optional, but access through optional chaining
                  //    - Field Foo.id is not a function, yet invoke as a function
                  bar.foo?.id()
              }

              struct Bar {
                  var foo: Foo
                  init() {
                      self.foo = Foo()
                  }
              }

              struct Foo {
                  var id: String

                  init() {
                      self.id = ""
                  }
              }
            `,
		)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidOptionalChainingError{}, errs[0])
		assert.IsType(t, &sema.NotCallableError{}, errs[1])
	})
}

func TestCheckOptionalChainingFunctionRead(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      struct Test {
          fun x(): Int {
              return 42
          }
      }

      let test: Test? = Test()
      let x = test?.x
    `)

	require.NoError(t, err)

	xType := RequireGlobalValue(t, checker.Elaboration, "x")

	expectedType := &sema.OptionalType{
		Type: &sema.FunctionType{
			Purity:               sema.FunctionPurityImpure,
			ReturnTypeAnnotation: sema.IntTypeAnnotation,
		},
	}

	assert.True(t, xType.Equal(expectedType))
}

func TestCheckOptionalChainingFunctionCall(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      struct Test {
          fun x(): Int {
              return 42
          }
      }

      let test: Test? = Test()
      let x = test?.x()
    `)

	require.NoError(t, err)

	assert.True(t,
		RequireGlobalValue(t, checker.Elaboration, "x").Equal(
			&sema.OptionalType{Type: sema.IntType},
		),
	)
}

func TestCheckInvalidOptionalChainingNonOptional(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct Test {
          let x: Int

          init(x: Int) {
              self.x = x
          }
      }

      let test = Test(x: 1)
      let x = test?.x
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidOptionalChainingError{}, errs[0])
}

func TestCheckInvalidOptionalChainingFieldAssignment(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct Test {
          var x: Int
          init(x: Int) {
              self.x = x
          }
      }

      fun test() {
          let test: Test? = Test(x: 1)
          test?.x = 2
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.UnsupportedOptionalChainingAssignmentError{}, errs[0])
}

func TestCheckFunctionTypeReceiverType(t *testing.T) {

	t.Parallel()

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          struct S {
              fun f() {}
          }

          let s = S()
          let f = s.f
        `)

		require.NoError(t, err)

		assert.Equal(t,
			&sema.FunctionType{
				Purity:               sema.FunctionPurityImpure,
				Parameters:           []sema.Parameter{},
				ReturnTypeAnnotation: sema.VoidTypeAnnotation,
			},
			RequireGlobalValue(t, checker.Elaboration, "f"),
		)
	})

	t.Run("cast bound function type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {
              fun f() {}
          }

          let s = S()
          let f = s.f as ((): Void)
        `)

		require.NoError(t, err)
	})
}

func TestCheckMemberNotDeclaredSecondaryError(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithOptions(t, `
            struct Test {
                fun foo(): Int { return 3 }
            }

            let test: Test = Test()
            let x = test.foop()
        `, ParseAndCheckOptions{
			Config: &sema.Config{
				SuggestionsEnabled: true,
			},
		})

		errs := RequireCheckerErrors(t, err, 1)

		var memberErr *sema.NotDeclaredMemberError
		require.ErrorAs(t, errs[0], &memberErr)
		assert.Equal(t, "did you mean `foo`?", memberErr.SecondaryError())
	})

	t.Run("without option", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct Test {
                fun foo(): Int { return 3 }
            }

            let test: Test = Test()
            let x = test.foop()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		var memberErr *sema.NotDeclaredMemberError
		require.ErrorAs(t, errs[0], &memberErr)
		assert.Equal(t, "unknown member", memberErr.SecondaryError())
	})

	t.Run("selects closest", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithOptions(t, `
            struct Test {
                fun fou(): Int { return 1 }
                fun bar(): Int { return 2 }
                fun foo(): Int { return 3 }
            }

            let test: Test = Test()
            let x = test.foop()
        `, ParseAndCheckOptions{
			Config: &sema.Config{
				SuggestionsEnabled: true,
			},
		})

		errs := RequireCheckerErrors(t, err, 1)

		var memberErr *sema.NotDeclaredMemberError
		require.ErrorAs(t, errs[0], &memberErr)
		assert.Equal(t, "did you mean `foo`?", memberErr.SecondaryError())
	})

	t.Run("no members = no suggestion", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithOptions(t, `
            struct Test {
                
            }

            let test: Test = Test()
            let x = test.foop()
        `, ParseAndCheckOptions{
			Config: &sema.Config{
				SuggestionsEnabled: true,
			},
		})

		errs := RequireCheckerErrors(t, err, 1)

		var memberErr *sema.NotDeclaredMemberError
		require.ErrorAs(t, errs[0], &memberErr)
		assert.Equal(t, "unknown member", memberErr.SecondaryError())
	})

	t.Run("no similarity = no suggestion", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithOptions(t, `
            struct Test {
                fun bar(): Int { return 1 }
            }

            let test: Test = Test()
            let x = test.foop()
        `, ParseAndCheckOptions{
			Config: &sema.Config{
				SuggestionsEnabled: true,
			},
		})

		errs := RequireCheckerErrors(t, err, 1)

		var memberErr *sema.NotDeclaredMemberError
		require.ErrorAs(t, errs[0], &memberErr)
		assert.Equal(t, "unknown member", memberErr.SecondaryError())
	})
}
