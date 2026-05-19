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

func TestCheckResourceInvalidationInWhileLoop(t *testing.T) {

	t.Parallel()

	t.Run("break", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                while true {
                    let r <- create R()
                    break
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("continue", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                while true {
                    let r <- create R()
                    continue
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("return", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                while true {
                    let r <- create R()
                    return
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})
}

func TestCheckResourceInvalidationInWhileLoopWithIfElse(t *testing.T) {

	t.Parallel()

	t.Run("if break", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                while true {
                    let r <- create R()
                    if true {
                        break
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("if continue", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                while true {
                    let r <- create R()
                    if true {
                        continue
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("if-else both break", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                while true {
                    let r <- create R()
                    if true {
                        break
                    } else {
                        break
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("if break, destroy after", func(t *testing.T) {
		t.Parallel()

		// The `destroy r` only runs on the non-break path,
		// so `r` leaks if the `break` is taken.

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                while true {
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

	t.Run("if break else destroy", func(t *testing.T) {
		t.Parallel()

		// The `destroy r` only runs on the else path,
		// so `r` leaks if the `break` is taken.

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                while true {
                    let r <- create R()
                    if true {
                        break
                    } else {
                        destroy r
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("if destroy else break", func(t *testing.T) {
		t.Parallel()

		// The `destroy r` only runs on the then path,
		// so `r` leaks if the `break` is taken.

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                while true {
                    let r <- create R()
                    if true {
                        destroy r
                    } else {
                        break
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("if destroy else destroy break", func(t *testing.T) {
		t.Parallel()

		// `r` is destroyed in both branches before the `break`, so no loss.

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                while true {
                    let r <- create R()
                    if true {
                        destroy r
                        break
                    } else {
                        destroy r
                    }
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("nested if break in inner if", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                while true {
                    let r <- create R()
                    if true {
                        if true {
                            break
                        }
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("resource outside loop, if break", func(t *testing.T) {
		t.Parallel()

		// `r` is declared outside the loop and destroyed after the loop,
		// so the `break` does not leak it.

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                let r <- create R()
                while true {
                    if true {
                        break
                    }
                }
                destroy r
            }
        `)

		require.NoError(t, err)
	})
}

func TestCheckResourceInvalidationInNestedWhileLoops(t *testing.T) {

	t.Parallel()

	t.Run("break in inner loop, destroy in outer", func(t *testing.T) {
		t.Parallel()

		// `r` is declared in the outer loop and destroyed there.
		// The inner `break` only exits the inner loop, so no leak.

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                while true {
                    let r <- create R()
                    while true {
                        break
                    }
                    destroy r
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("resource in inner loop, inner break", func(t *testing.T) {
		t.Parallel()

		// `r` is declared in the inner loop body and the `break` exits it
		// without destroying `r`.

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                while true {
                    while true {
                        let r <- create R()
                        break
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("outer break inside inner loop", func(t *testing.T) {
		t.Parallel()

		// `break` always targets the innermost loop,
		// so the inner `break` only exits the inner loop.
		// The outer loop's `r` is destroyed at the end of each outer iteration.

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                while true {
                    let r <- create R()
                    while true {
                        if true {
                            break
                        }
                    }
                    destroy r
                }
            }
        `)

		require.NoError(t, err)
	})
}
