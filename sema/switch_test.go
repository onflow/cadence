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

package sema_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/sema_utils"
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

func TestCheckSwitchBreakDoesNotEscapeOuterLoop(t *testing.T) {

	t.Parallel()

	t.Run("default only", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(xs: [Int]) {
              for x in xs {
                  switch x {
                  default:
                      break
                  }
                  let y = 1
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("all cases break", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(xs: [Int]) {
              for x in xs {
                  switch x {
                  case 1:
                      break
                  default:
                      break
                  }
                  let y = 1
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("nested switch inner break", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(a: Int, b: Int) {
              switch a {
              default:
                  switch b {
                  default:
                      break
                  }
                  let x = 1
              }
              let y = 1
          }
        `)

		require.NoError(t, err)
	})

	t.Run("continue still propagates as unreachable", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(xs: [Int]) {
              for x in xs {
                  switch x {
                  default:
                      continue
                  }
                  let y = 1
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("mixed break and continue does not propagate", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(xs: [Int]) {
              for x in xs {
                  switch x {
                  case 1:
                      break
                  default:
                      continue
                  }
                  let y = 1
              }
          }
        `)

		require.NoError(t, err)
	})
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

func TestCheckFoo(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(cond: Bool): Int {
          switch 1 {
          case 1:
              if cond { break }
              return 2
          default:
              return 0
          }
          return 3   // sema (incorrectly) flags this as unreachable
      }
    `)

	require.NoError(t, err)
}

func TestCheckSwitchMaybeBreakDoesNotSuppressUnreachableInCase(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(cond: Bool): Int {
          switch 1 {
          case 1:
              if cond { break }
              return 2
              let x = 1
          default:
              return 0
          }
          return 3
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
}

func TestCheckSwitchConditionalBreak(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        fun test(): Int {
            switch 1 {
            case 1:
                if true { break }
                return 2
            default:
                return 0
            }
            return 3   // Shouldn't be marked as unreachable
        }
    `)

	require.NoError(t, err)
}

func TestCheckSwitchConditionalHalt(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithPanic(t, `
        fun test() {
            switch 1 {
            case 1:
                if true { break }
                panic("unreachable")
            default:
                return
            }
            let x = 1   // Shouldn't be marked as unreachable
        }
    `)

	require.NoError(t, err)
}

func TestCheckSwitchGuardElseBreak(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        fun test(): Int {
            switch 1 {
            case 1:
                guard let y = (nil as Int?) else { break }
                return y
            default:
                return 0
            }
            return 3   // Shouldn't be marked as unreachable
        }
    `)

	require.NoError(t, err)
}

func TestCheckSwitchUnreachableAfterReturnFollowingConditionalBreak(t *testing.T) {

	t.Parallel()

	// Inside a single case body, `if cond { break }; return` exits via either path.
	// Any statement following the return must therefore be reported as unreachable,
	// even though the case as a whole is not DefinitelyReturned (one path broke from the switch).

	_, err := ParseAndCheck(t, `
        fun test(): Int {
            switch 1 {
            case 1:
                if true { break }
                return 2
                let x = 1
            default:
                return 0
            }
            return 3
        }
    `)

	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
}

func TestCheckSwitchUnreachableAfterMixedExits(t *testing.T) {

	t.Parallel()

	// Inside a case body, both branches of an `if-else` exit,
	// but in different ways: the `then` branch breaks from the switch,
	// while the `else` branch returns from the function.
	// Neither branch alone gives DefinitelyReturned, but every path has
	// exited (DefinitelyExited holds via AND-merge), so subsequent
	// statements must be reported as unreachable.
	_, err := ParseAndCheck(t, `
        fun test() {
            switch 1 {
            case 1:
                if true {
                    break
                } else {
                    return
                }
                let x = 1
            default:
                return
            }
        }
    `)

	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
}

// TestCheckSwitchMixedExitVariants exercises the various combinations
// of mixed exits inside a single switch case body.
// In each case, every path through the if-else terminates,
// so any trailing statement must be reported as unreachable,
// but the switch as a whole does not definitely terminate (one path breaks out),
// so code after the switch remains reachable.
func TestCheckSwitchMixedExitVariants(t *testing.T) {

	t.Parallel()

	t.Run("return then break", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun test() {
                switch 1 {
                case 1:
                    if true { return } else { break }
                    let x = 1
                default:
                    return
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("break then halt", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            fun test() {
                switch 1 {
                case 1:
                    if true { break } else { panic("x") }
                    let x = 1
                default:
                    return
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("halt then break", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            fun test() {
                switch 1 {
                case 1:
                    if true { panic("x") } else { break }
                    let x = 1
                default:
                    return
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("return then halt", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            fun test() {
                switch 1 {
                case 1:
                    if true { return } else { panic("x") }
                    let x = 1
                default:
                    return
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})
}

// TestCheckSwitchCaseConditionalJumpThenTermination covers the "maybe-jump on one path,
// definite terminator on the other" pattern in a switch case body.
// The case is not a definite-return for the switch merge (the break path falls past the switch),
// and any statement after the trailing terminator is unreachable.
func TestCheckSwitchCaseConditionalJumpThenTermination(t *testing.T) {

	t.Parallel()

	t.Run("if break then halt; code after switch reachable", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            fun test(): Int {
                switch 1 {
                case 1:
                    if true { break }
                    panic("x")
                default:
                    return 0
                }
                return 3
            }
        `)
		require.NoError(t, err)
	})

	t.Run("if break then halt; statement after halt is unreachable", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            fun test() {
                switch 1 {
                case 1:
                    if true { break }
                    panic("x")
                    let x = 1
                default:
                    return
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})
}

// TestCheckSwitchAllCasesMaybeBreak verifies that when every case body "maybe breaks" (and otherwise returns),
// the switch is correctly not treated as a definite return, so code after it remains reachable.
func TestCheckSwitchAllCasesMaybeBreak(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        fun test(cond: Bool): Int {
            switch 1 {
            case 1:
                if cond { break }
                return 1
            case 2:
                if cond { break }
                return 2
            default:
                if cond { break }
                return 3
            }
            return 4
        }
    `)

	require.NoError(t, err)
}

// TestCheckSwitchNestedSwitchInnerBreak verifies that a break inside an inner switch
// is consumed by that inner switch and does not affect the outer case body's "definitely returns" status.
// The outer case definitely returns because the inner switch's break-path falls through to the trailing `return`.
func TestCheckSwitchNestedSwitchInnerBreak(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        fun test(a: Int, b: Int): Int {
            switch a {
            case 1:
                switch b {
                case 1:
                    break
                default:
                    return 10
                }
                return 1
            default:
                return 2
            }
            return 3   // unreachable: outer case 1 and default both return
        }
    `)

	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
}

// TestCheckSwitchLoopInCaseInnerBreak verifies that a `break` inside a loop nested in a switch case
// targets the loop (the innermost construct), not the switch.
// The case is therefore a definite return.
func TestCheckSwitchLoopInCaseInnerBreak(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        fun test(): Int {
            switch 1 {
            case 1:
                while true {
                    break
                }
                return 1
            default:
                return 2
            }
            return 3   // unreachable
        }
    `)

	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
}

// TestCheckSwitchContinueInSwitchInLoop verifies that a `continue` inside a switch case
// (where the switch is inside a loop) targets the enclosing loop,
// not the switch — the continue propagates past the switch as a loop-targeting jump.
func TestCheckSwitchContinueInSwitchInLoop(t *testing.T) {

	t.Parallel()

	t.Run("continue in all cases makes post-switch unreachable", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test() {
                while true {
                    switch 1 {
                    case 1:
                        continue
                    default:
                        continue
                    }
                    let x = 1   // unreachable
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("mixed break-switch and continue: post-switch reachable", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test() {
                while true {
                    switch 1 {
                    case 1:
                        break
                    default:
                        continue
                    }
                    let x = 1   // reachable on the break path
                }
            }
        `)
		require.NoError(t, err)
	})
}

// TestCheckSwitchResourceMaybeBreak verifies that a switch case whose body destroys a resource
// on the non-break path correctly reports the resource as potentially lost:
// the break path leaves the resource undestroyed.
func TestCheckSwitchResourceMaybeBreak(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        resource R {}
        fun test(cond: Bool) {
            let r <- create R()
            switch 1 {
            case 1:
                if cond { break }
                destroy r
                return
            default:
                destroy r
                return
            }
        }
    `)

	// The break path in case 1 does not destroy r,
	// so r escapes the switch on that path.
	// After the switch, r may still be alive.
	errs := RequireCheckerErrors(t, err, 2)
	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	assert.IsType(t, &sema.ResourceLossError{}, errs[1])
}

func TestCheckResourceInvalidationInSwitch(t *testing.T) {

	t.Parallel()

	t.Run("break in case", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}
            fun test() {
                switch true {
                case true:
                    let r <- create R()
                    break
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("break in default case", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}
            fun test() {
                switch true {
                default:
                    let r <- create R()
                    break
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("return in case", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}
            fun test() {
                switch true {
                case true:
                    let r <- create R()
                    return
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("resource outside switch, break in case", func(t *testing.T) {
		t.Parallel()

		// `break` only exits the switch, not any enclosing scope.
		// `r` is declared outside and destroyed after the switch,
		// so no leak.

		_, err := ParseAndCheck(t, `
            resource R {}
            fun test() {
                let r <- create R()
                switch true {
                case true:
                    break
                }
                destroy r
            }
        `)

		require.NoError(t, err)
	})

	t.Run("break in nested if, destroy after", func(t *testing.T) {
		t.Parallel()

		// The `destroy r` only runs on the non-break path,
		// so `r` leaks if the `break` is taken.

		_, err := ParseAndCheck(t, `
            resource R {}
            fun test() {
                switch true {
                case true:
                    let r <- create R()
                    if true {
                        break
                    }
                    destroy r
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("break in switch inside loop, resource in loop body", func(t *testing.T) {
		t.Parallel()

		// `break` targets the innermost switch (not the loop),
		// so the loop's `r` is destroyed at the end of each iteration.

		_, err := ParseAndCheck(t, `
            resource R {}
            fun test() {
                while true {
                    let r <- create R()
                    switch true {
                    case true:
                        break
                    }
                    destroy r
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("break in switch inside loop, resource in case", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}
            fun test() {
                while true {
                    switch true {
                    case true:
                        let r <- create R()
                        break
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})
}
