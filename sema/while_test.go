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

func TestCheckInvalidWhileTest(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          while 1 {}
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckWhileTest(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          while true {}
      }
    `)

	assert.NoError(t, err)
}

func TestCheckInvalidWhileBlock(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          while true { x }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckWhileBreakStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       fun test() {
           while true {
               break
           }
       }
    `)

	assert.NoError(t, err)
}

func TestCheckInvalidWhileBreakStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       fun test() {
           while true {
               fun () {
                   break
               }
           }
       }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ControlStatementError{}, errs[0])
}

func TestCheckWhileContinueStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       fun test() {
           while true {
               continue
           }
       }
    `)

	assert.NoError(t, err)
}

func TestCheckInvalidWhileContinueStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       fun test() {
           while true {
               fun () {
                   continue
               }
           }
       }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ControlStatementError{}, errs[0])
}

func TestCheckInvalidBreakStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          break
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.ControlStatementError{}, errs[0])
}

func TestCheckInvalidContinueStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          continue
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.ControlStatementError{}, errs[0])
}

func TestCheckBreakInWhileLoopBodyDoesNotPreventOuterReturn(t *testing.T) {

	t.Parallel()

	// A `break` inside the while-loop body targets the loop, not the enclosing function.
	// The trailing `return 1` must therefore still mark the function as definitely returning.

	_, err := ParseAndCheck(t, `
        fun test(): Int {
            while true {
                break
            }
            return 1
        }
    `)

	require.NoError(t, err)
}

func TestCheckContinueInWhileLoopBodyDoesNotPreventOuterReturn(t *testing.T) {

	t.Parallel()

	// A `continue` inside the while-loop body targets the loop, not the enclosing function.
	// The trailing `return 1` must therefore still mark the function as definitely returning.

	_, err := ParseAndCheck(t, `
        fun test(): Int {
            while true {
                continue
            }
            return 1
        }
    `)

	require.NoError(t, err)
}

// TestCheckWhileLoopBodyMixedExitVariants exercises every unique pair of distinct exit kinds
// (return, halt, break, continue) used as the two branches of an `if-else` inside the while-loop body.
// Every path through the if-else terminates control flow (in some way),
// so any trailing statement must be reported as unreachable.
func TestCheckWhileLoopBodyMixedExitVariants(t *testing.T) {

	t.Parallel()

	t.Run("break and continue", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test() {
                while true {
                    if true { break } else { continue }
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("break and halt", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckWithPanic(t, `
            fun test() {
                while true {
                    if true { break } else { panic("x") }
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("break and return", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test() {
                while true {
                    if true { break } else { return }
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("continue and halt", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckWithPanic(t, `
            fun test() {
                while true {
                    if true { continue } else { panic("x") }
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("continue and return", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test() {
                while true {
                    if true { continue } else { return }
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("halt and return", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckWithPanic(t, `
            fun test() {
                while true {
                    if true { panic("x") } else { return }
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})
}

// TestCheckWhileLoopConditionalJumpThenTermination covers the
// "maybe-jump on one path, definite terminator on the other" pattern in
// a while-loop body.
//
// For each (JUMP, TERMINATOR) combination, two assertions:
//   - Code AFTER the loop is reachable: the jump path falls past the
//     loop, so the loop body's `DefinitelyReturned`/`DefinitelyHalted`
//     claim must NOT propagate to the function.
//   - A statement AFTER the terminator inside the body is unreachable:
//     within the body, every path through the if-else does terminate.
func TestCheckWhileLoopConditionalJumpThenTermination(t *testing.T) {

	t.Parallel()

	t.Run("if break then return; code after loop reachable", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test(): Int {
                while true {
                    if true { break }
                    return 1
                }
                return 2
            }
        `)
		require.NoError(t, err)
	})

	t.Run("if break then return; statement after return is unreachable", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test() {
                while true {
                    if true { break }
                    return
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("if break then halt; code after loop reachable", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckWithPanic(t, `
            fun test() {
                while true {
                    if true { break }
                    panic("x")
                }
                let y = 1
            }
        `)
		require.NoError(t, err)
	})

	t.Run("if break then halt; statement after halt is unreachable", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckWithPanic(t, `
            fun test() {
                while true {
                    if true { break }
                    panic("x")
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("if continue then return; code after loop reachable", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test(): Int {
                while true {
                    if true { continue }
                    return 1
                }
                return 2
            }
        `)
		require.NoError(t, err)
	})

	t.Run("if continue then return; statement after return is unreachable", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test() {
                while true {
                    if true { continue }
                    return
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("if continue then halt; code after loop reachable", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckWithPanic(t, `
            fun test() {
                while true {
                    if true { continue }
                    panic("x")
                }
                let y = 1
            }
        `)
		require.NoError(t, err)
	})

	t.Run("if continue then halt; statement after halt is unreachable", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckWithPanic(t, `
            fun test() {
                while true {
                    if true { continue }
                    panic("x")
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})
}

func TestCheckNestedWhileLoopBreakDoesNotEscapeOuterLoop(t *testing.T) {

	t.Parallel()

	// A `break` inside the inner while-loop targets the inner loop only.
	// Code after the inner loop, but still in the outer loop body,
	// must remain reachable.

	_, err := ParseAndCheck(t, `
        fun test() {
            while true {
                while true {
                    break
                }
                let x = 1
            }
        }
    `)

	require.NoError(t, err)
}

func TestCheckNestedWhileLoopMaybeJumpedDoesNotEscape(t *testing.T) {

	t.Parallel()

	// A `MaybeJumpedLoop` set inside an inner while-loop body must not leak into the outer loop's body state.
	// WithLoop save/restores both `DefinitelyJumpedLoop` and `MaybeJumpedLoop`.

	_, err := ParseAndCheck(t, `
        fun test(): Int {
            while true {
                while true {
                    if true { break }
                    return 1
                }
                let x = 1
            }
            return 2
        }
    `)

	require.NoError(t, err)
}

// TestCheckWhileLoopWithSwitchInBody verifies that a switch nested in a while-loop body
// interacts correctly with the loop's control flow:
// switch-targeting `break` is consumed by the switch,
// `continue` propagates past the switch to the enclosing loop.
func TestCheckWhileLoopWithSwitchInBody(t *testing.T) {

	t.Parallel()

	t.Run("switch break does not escape loop body", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test() {
                while true {
                    switch 1 {
                    case 1:
                        break
                    default:
                        break
                    }
                    let x = 1
                }
            }
        `)
		require.NoError(t, err)
	})

	t.Run("all-cases continue makes post-switch in loop unreachable", func(t *testing.T) {
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
                    let x = 1
                }
            }
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("nested switch case with maybe-break does not affect outer return", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test(): Int {
                while true {
                    switch 1 {
                    case 1:
                        if true { break }
                        return 1
                    default:
                        return 2
                    }
                }
                return 3
            }
        `)
		require.NoError(t, err)
	})

	t.Run("nested switch case with maybe-continue does not over-claim", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheck(t, `
            fun test(): Int {
                while true {
                    switch 1 {
                    case 1:
                        if true { continue }
                        return 1
                    default:
                        return 2
                    }
                }
                return 3
            }
        `)
		require.NoError(t, err)
	})
}

func TestCheckResourceInWhileLoopBodyMaybeBreak(t *testing.T) {

	t.Parallel()

	// A while-loop body whose destroy/return path is guarded by a maybe-break:
	// on the break path, the resource is not destroyed and the loop is exited,
	// so the resource is potentially lost.

	_, err := ParseAndCheck(t, `
        resource R {}
        fun test(cond: Bool) {
            let r <- create R()
            while true {
                if cond { break }
                destroy r
                return
            }
        }
    `)

	errs := RequireCheckerErrors(t, err, 2)
	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	assert.IsType(t, &sema.ResourceLossError{}, errs[1])
}

func TestCheckGuardElseBreakInWhileLoop(t *testing.T) {

	t.Parallel()

	// A `guard ... else { break }` inside a while-loop body
	// must propagate the potential loop-targeting jump out of the (potentially-unevaluated) else block,
	// so code after the loop remains reachable.

	_, err := ParseAndCheck(t, `
        fun test(): Int {
            while true {
                guard let y = (nil as Int?) else { break }
                return y
            }
            return 3
        }
    `)

	require.NoError(t, err)
}
