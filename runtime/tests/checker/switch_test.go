/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

	errs := RequireCheckerErrors(t, err, 1)

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

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
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

	errs := RequireCheckerErrors(t, err, 2)

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

		errs := RequireCheckerErrors(t, err, 1)

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

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("break before return", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          fun test(x: Int): String {
              switch x {
              case 1:
                  return "one"
              default:
                  break
                  return "two"
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.MissingReturnStatementError{}, errs[1])
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

	errs := RequireCheckerErrors(t, err, 1)

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

	errs := RequireCheckerErrors(t, err, 1)

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

	errs := RequireCheckerErrors(t, err, 1)

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

	errs := RequireCheckerErrors(t, err, 1)

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

	errs := RequireCheckerErrors(t, err, 1)

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

	errs := RequireCheckerErrors(t, err, 2)

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

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.MissingSwitchCaseStatementsError{}, errs[0])
}

func TestCheckSwitchStatementWithUnreachableReturn(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(_ x: Int): Int {
          switch x {
          case 1:
              break
              return 1
          default:
              return 2
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	assert.IsType(t, &sema.MissingReturnStatementError{}, errs[1])
}

func TestCheckCaseExpressionTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(): Int {
              let x: UInt8 = 5

              switch x {
              case 1:
                  return 1
              default:
                  return 2
              }
          }
        `)

		assert.NoError(t, err)
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(): Int {
              let x: UInt8 = 5

              switch x {
              case "one":
                  return 1
              default:
                  return 2
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("unknown", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(): Int {
              switch x {
              case "one":
                  return 1
              default:
                  return 2
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("character literal", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun test(): Int {
                let c: Character = "a"
                switch c {
                case "b": return 0
                case "c": return 1
                case "d": return 2
                case "a": return 1337
                default: return -1
                }
            }
        `)

		require.NoError(t, err)
	})
}

func TestCheckSwitchResourceInvalidation(t *testing.T) {
	t.Parallel()

	t.Run("in first test", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun drop(_ r: @AnyResource): Bool {
              destroy r
              return true
          }

          fun test() {
              let r <- create R()
              switch true {
              case drop(<-r):
                return
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("in first case", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test() {
              let r <- create R()
              switch true {
              case false:
                destroy r
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("in second test, not invalidated in first", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun drop(_ r: @AnyResource): Bool {
              destroy r
              return true
          }

          fun test() {
              let r <- create R()
              switch true {
              case false:
                return
              case drop(<-r):
                return
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("in second test, but invalidated in first case", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun drop(_ r: @AnyResource): Bool {
              destroy r
              return true
          }

          fun test() {
              let r <- create R()
              switch true {
              case false:
                destroy r
                return
              case drop(<-r):
                return
              }
          }
        `)
		require.NoError(t, err)
	})

	t.Run("invalidations in multiple tests", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun drop(_ r: @AnyResource): Bool {
              destroy r
              return true
          }

          fun test() {
              let r <- create R()
              switch true {
              case drop(<-r):
                return
              case drop(<-r):
                return
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	})
}
