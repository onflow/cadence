/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
)

func TestCheckSwitchStatementTest(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          switch true {}
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidSwitchStatementTest(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct S {}

      fun test() {
          let s = S()

          switch s {}
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotEquatableTypeError{}, errs[0])
}

func TestCheckSwitchStatementCaseExpression(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(x: Int): String {
          switch x {
          case 1:
              return "one"
          }

          return "other"
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidSwitchStatementCaseExpression(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct S {}

      fun test(): Int {
          let s = S()

          switch true {
          case s:
              return 1
          }

          return 2
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidBinaryOperandsError{}, errs[0])
}

func TestCheckInvalidSwitchStatementCaseExpressionInvalidTest(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct S {}

      fun test() {
          let s = S()
          var y = 0
          switch x {
          case s:
              y = 1
          }
      }
    `)

	errs := ExpectCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	assert.IsType(t, &sema.NotEquatableTypeError{}, errs[1])
}

func TestCheckSwitchStatementDefaultDefinitiveReturn(t *testing.T) {

	t.Parallel()

	t.Run("with default", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          fun test(x: Int): String {
              switch x {
              case 1:
                  return "one"
              case 2:
                  return "two"
              default:
                  return "other"
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("no default", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          fun test(x: Int): String {
              switch x {
              case 1:
                  return "one"
              case 2:
                  return "two"
              }
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.MissingReturnStatementError{}, errs[0])
	})

	t.Run("unreachable code", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          fun test(x: Int): String {
              switch x {
              case 1:
                  return "one"
              case 2:
                  return "two"
              default:
                  return "other"
              }
              return "never"
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})
}

func TestCheckInvalidSwitchStatementCaseStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      fun test() {
          switch true {
          case true:
              x
          }
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckInvalidSwitchStatementDefaultStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      fun test() {
          switch true {
          default:
              x
          }
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckInvalidSwitchStatementDefaultPosition(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      fun test() {
          var x = 0
          switch true {
          default:
              x = 1
          case true:
              x = 2
          }
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.SwitchDefaultPositionError{}, errs[0])
}

func TestCheckInvalidSwitchStatementDefaultDuplicate(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      fun test() {
          var x = 0
          switch true {
          default:
              x = 1
          default:
              x = 2
          }
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.SwitchDefaultPositionError{}, errs[0])
}

func TestCheckSwitchStatementCaseScope(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      fun test(_ x: Int) {
          switch x {
          case 1:
              let y = true
          case 2:
              y
          }
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckSwitchStatementBreakStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      fun test(_ x: Int) {
          switch x {
          case 1:
              break
          default:
              break
          }
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidSwitchStatementContinueStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      fun test(_ x: Int) {
          switch x {
          case 1:
              continue
          default:
              continue
          }
      }
    `)

	errs := ExpectCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.ControlStatementError{}, errs[0])
	assert.IsType(t, &sema.ControlStatementError{}, errs[1])
}

func TestCheckInvalidSwitchStatementMissingStatements(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      fun test(_ x: Int) {
          switch x {
          case 1:
          }
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.MissingSwitchCaseStatementsError{}, errs[0])
}

func TestCheckSwitchStatementDuplicateCases(t *testing.T) {

	t.Parallel()

	t.Run("multiple duplicates", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
          fun test(): Int {
              let s: String? = nil

              switch s {
                  case "foo":
                      return 1
                  case "bar":
                      return 2
                  case "bar":
                      return 3
                  case "bar":
                      return 4
              }

              return -1
          }
        `)

		// Should only report two errors. i.e: second and the third
		// duplicate cases must not be compared with each other.
		errs := ExpectCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.DuplicateSwitchCaseError{}, errs[0])
		assert.IsType(t, &sema.DuplicateSwitchCaseError{}, errs[1])
	})

	t.Run("simple literals", func(t *testing.T) {
		type test struct {
			name string
			expr string
		}

		expressions := []test{
			{
				name: "string",
				expr: "\"hello\"",
			},
			{
				name: "integer",
				expr: "5",
			},
			{
				name: "fixedpoint",
				expr: "4.7",
			},
			{
				name: "boolean",
				expr: "true",
			},
		}

		for _, testCase := range expressions {

			t.Run(testCase.name, func(t *testing.T) {
				_, err := ParseAndCheck(t, fmt.Sprintf(`
                    fun test(): Int {
                        let x = %[1]s
                        switch x {
                            case %[1]s:
                                return 1
                            case %[1]s:
                                return 2
                        }
                        return -1
                    }`,
					testCase.expr),
				)

				errs := ExpectCheckerErrors(t, err, 1)
				assert.IsType(t, &sema.DuplicateSwitchCaseError{}, errs[0])
			})
		}
	})

	t.Run("identifier", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
          fun test(): Int {
              let x = 5
              let y = 5
              switch 4 {
                  case x:
                      return 1
                  case x:
                      return 2
                  case y:  // different identifier
                      return 3
              }
              return -1
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.DuplicateSwitchCaseError{}, errs[0])
	})

	t.Run("member access", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
          fun test(): Int {
              let x = Foo()

              switch x.a {
                  case x.a:
                      return 1
                  case x.a:
                      return 2
                  case x.b:
                      return 3
              }
              return -1
          }

          struct Foo {
              pub var a: String
              pub var b: String
              init() {
                  self.a = "foo"
                  self.b = "bar"
              }
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.DuplicateSwitchCaseError{}, errs[0])
	})

	t.Run("index access", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
          fun test(): Int {
              let x: [Int] = [1, 2, 3]
              let y: [Int] = [5, 6, 7]

              switch x[0] {
                  case x[1]:
                      return 1
                  case x[1]:
                      return 2
                  case x[2]:
                      return 3
                  case y[1]:
                      return 4
              }
              return -1
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.DuplicateSwitchCaseError{}, errs[0])
	})

	t.Run("conditional", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
          fun test(): Int {
              switch "foo" {
                  case true ? "foo" : "bar":
                      return 1
                  case true ? "foo" : "bar":
                      return 2
                  case true ? "baz" : "bar":  // different then expr
                      return 3
                  case true ? "foo" : "baz":  // different else expr
                      return 4
                  case false ? "foo" : "bar":  // different condition expr
                      return 5
              }
              return -1
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.DuplicateSwitchCaseError{}, errs[0])
	})

	t.Run("unary", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
          fun test(): Int {
              let x = 5
              let y = x
              switch x {
                  case -x:
                      return 1
                  case -x:
                      return 2
                  case -y:  // different rhs expr
                      return 3
              }
              return -1
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.DuplicateSwitchCaseError{}, errs[0])
	})

	t.Run("binary", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
          fun test(): Int {
              switch 4 {
                  case 3+5:
                      return 1
                  case 3+5:
                      return 2
                  case 3+7:  // different rhs expr
                      return 3
                  case 7+5:  // different lhs expr
                      return 4
                  case 3-5:  // different operator
                      return 5
              }
              return -1
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.DuplicateSwitchCaseError{}, errs[0])
	})

	t.Run("cast", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
          fun test(): Int {
              let x = 5
              let y = x as Integer
              switch y {
                  case x as Integer:
                      return 1
                  case x as Integer:
                      return 2
                  case x as! Integer:  // different operator
                      return 3
                  case y as Integer:  // different expr
                      return 4
              }
              return -1
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.DuplicateSwitchCaseError{}, errs[0])
	})

	t.Run("create", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
          fun test() {
              let x <- create Foo()
              switch x {
              }
              destroy x
          }

          resource Foo {}
        `)

		errs := ExpectCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.NotEquatableTypeError{}, errs[0])
	})

	t.Run("destroy", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
          fun test() {
              let x <- create Foo()
              switch destroy x {
              }
          }

          resource Foo {}
        `)

		errs := ExpectCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.NotEquatableTypeError{}, errs[0])
	})

	t.Run("reference", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
          fun test(): Int {
              let x: Int = 5
              let y: Int = 7
              switch (&x as &Int) {
                  case &x as &Int:
                      return 1
                  case &x as &Int:
                      return 2
                  case &y as &Int:  // different expr
                      return 2
              }
              return -1
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.DuplicateSwitchCaseError{}, errs[0])
	})

	t.Run("force", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
          fun test(): Int {
              let x: Int? = 5
              let y: Int? = 5
              switch 4 {
                  case x!:
                      return 1
                  case x!:
                      return 2
                  case y!:    // different expr
                      return 3
              }
              return -1
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.DuplicateSwitchCaseError{}, errs[0])
	})

	t.Run("path", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
          fun test() {
              switch /public/somepath {
              }
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.NotEquatableTypeError{}, errs[0])
	})

	t.Run("invocation", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
          fun test(): Int {
              switch "hello" {
                  case foo():
                      return 1
                  case foo():
                      return 2
              }
              return -1
          }

          fun foo(): String {
              return "hello"
          }
        `)

		assert.NoError(t, err)
	})
}
