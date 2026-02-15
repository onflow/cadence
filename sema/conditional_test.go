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

func TestCheckConditionalExpressionTest(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          let x = true ? 1 : 2
      }
	`)

	require.NoError(t, err)
}

func TestCheckInvalidConditionalExpressionTest(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          let x = 1 ? 2 : 3
      }
	`)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidConditionalExpressionElse(t *testing.T) {

	t.Parallel()

	t.Run("undeclared variable", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            let x = true ? 2 : y
	    `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.IsType(t, &sema.TypeAnnotationRequiredError{}, errs[1])

		xType := RequireGlobalValue(t, checker.Elaboration, "x")
		assert.Equal(t, sema.InvalidType, xType)
	})

	t.Run("mismatching type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            let x: Int8 = true ? 2 : "hello"
	    `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
		typeMismatchError := errs[0].(*sema.TypeMismatchError)

		assert.Equal(t, sema.Int8Type, typeMismatchError.ExpectedType)
		assert.Equal(t, sema.StringType, typeMismatchError.ActualType)
	})
}

func TestCheckConditionalExpressionTypeInferring(t *testing.T) {

	t.Parallel()

	t.Run("different simple types", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            let x = true ? 2 : false
        `)

		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")
		assert.Equal(t, sema.HashableStructType, xType)
	})

	t.Run("optional", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            let x = true ? 1 : nil
        `)

		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")
		assert.Equal(
			t,
			&sema.OptionalType{
				Type: sema.IntType,
			},
			xType,
		)
	})

	t.Run("chained optional", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            let x = true ? Int8(1) : (false ? Int(5) : nil)
        `)

		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")
		assert.Equal(
			t,
			&sema.OptionalType{
				Type: sema.SignedIntegerType,
			},
			xType,
		)
	})
}

func TestCheckGuardStatementWithBooleanExpression(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          guard true else {
              return
          }
      }
	`)

	require.NoError(t, err)
}

func TestCheckInvalidGuardStatementBooleanExpression(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          guard 1 else {
              return
          }
      }
	`)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckGuardStatementWithOptionalBinding(t *testing.T) {

	t.Parallel()

	t.Run("let binding with variable available after", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(x: Int?): Int {
              guard let y = x else {
                  return 0
              }
              // y should be available here and unwrapped to Int
              return y
          }
	    `)

		require.NoError(t, err)
	})

	t.Run("var binding", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(x: String?): String {
              guard var y = x else {
                  return ""
              }
              return y
          }
	    `)

		require.NoError(t, err)
	})

	t.Run("binding not available in else branch", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(x: Int?) {
              guard let y = x else {
                  // y should not be available in the else block
                  y
                  return
              }
          }
	    `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("let binding cannot be assigned", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(x: Int?) {
              guard let y = x else {
                  return
              }
              // y is a constant, cannot be assigned
              y = 42
          }
	    `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AssignmentToConstantError{}, errs[0])
	})

	t.Run("var binding can be assigned", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(x: Int?): Int {
              guard var y = x else {
                  return 0
              }
              // y is a variable, can be assigned
              y = 42
              return y
          }
	    `)

		require.NoError(t, err)
	})

	t.Run("else declarations not visible outside", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(x: Int?) {
              guard let y = x else {
                  let z = 10
                  return
              }
              // z should not be visible here
              z
          }
	    `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})
}

func TestCheckInvalidGuardStatementElseBlockMustExit(t *testing.T) {

	t.Parallel()

	t.Run("else without exit", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test() {
              guard true else {}
          }
	    `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.GuardStatementElseBlockMustExitError{}, errs[0])
	})

	t.Run("else with optional binding without exit", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(x: Int?) {
              guard let y = x else {}
          }
	    `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.GuardStatementElseBlockMustExitError{}, errs[0])
	})

	t.Run("else with exit", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(): Int {
              guard true else {
                  return 42
              }
              return 0
          }
        `)
		require.NoError(t, err)
	})

	t.Run("else with break", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(): Bool {
              while true {
                  guard false else {
                      break
                  }
              }
              return true
          }
	    `)
		require.NoError(t, err)
	})

	t.Run("else with continue", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(): Bool {
              while true {
                  guard false else {
                      continue
                  }
              }
              return true
          }
        `)
		require.NoError(t, err)
	})

	t.Run("else with panic", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
          fun test(): Bool {
              guard true else {
                  panic("error")
              }
              return true
          }
	    `)

		require.NoError(t, err)
	})
}

func TestCheckGuardStatementResourceTracking(t *testing.T) {

	t.Parallel()

	t.Run("resource used in else is still available after guard", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test(r: @R) {
              guard true else {
                  destroy r
                  return
              }
              destroy r
          }
	    `)

		require.NoError(t, err)
	})

	t.Run("resource must be handled in both paths", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test(r: @R) {
              guard true else {
                  return
              }
          }
	    `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("resource handled in else but not in main branch", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test(r: @R) {
              guard true else {
                  destroy r
                  return
              }
          }
	    `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("resource handled in main branch but not in else", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test(r: @R) {
              guard true else {
                  return
              }
              destroy r
          }
	    `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})
}

func TestCheckGuardStatementWithNonOptionalBinding(t *testing.T) {
	t.Parallel()

	t.Run("nil case has no resource loss", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test(r: @R?) {
              guard let unwrapped <- r else {
                  return
              }
              destroy unwrapped
          }
	    `)

		require.NoError(t, err)
	})

	t.Run("use in else", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
          resource R {}

          fun test(r: @R?): @R {
              guard let unwrapped <- r else {
                  panic("r is nil: \(r == nil)")
              }
              return <-unwrapped
          }
	    `)

		errs := RequireCheckerErrors(t, err, 1)

		var resourceErr *sema.ResourceUseAfterInvalidationError
		require.ErrorAs(t, errs[0], &resourceErr)

		assert.Equal(t, resourceErr.StartPos.Line, 6)
	})

	t.Run("original not available after binding", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test(r: @R?) {
              guard let unwrapped <- r else {
                  return
              }

              // r was consumed by the binding
              destroy r

              destroy unwrapped
          }
	    `)

		errs := RequireCheckerErrors(t, err, 1)

		var resourceErr *sema.ResourceUseAfterInvalidationError
		require.ErrorAs(t, errs[0], &resourceErr)

		assert.Equal(t, resourceErr.StartPos.Line, 10)
	})
}

func TestCheckGuardStatementWithResourceFailableCast(t *testing.T) {

	t.Parallel()

	t.Run("return", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test(r: @AnyResource): @R? {
              guard let typedR <- r as? @R else {
                  destroy r
                  return nil
              }
              return <-typedR
          }
	    `)

		require.NoError(t, err)
	})

	t.Run("destroy", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
	      resource R {}

	      fun test(r: @AnyResource) {
	          guard let typedR <- r as? @R else {
	              destroy r
	              return
	          }
	          destroy typedR
	      }
		`)

		require.NoError(t, err)
	})

	t.Run("resource not available after successful binding", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test(r: @AnyResource) {
              guard let typedR <- r as? @R else {
                  destroy r
                  return
              }
              destroy r
              destroy typedR
          }
	    `)

		errs := RequireCheckerErrors(t, err, 1)

		var resourceErr *sema.ResourceUseAfterInvalidationError
		require.ErrorAs(t, errs[0], &resourceErr)

		assert.Equal(t, resourceErr.StartPos.Line, 9)
	})

	t.Run("must handle resource in else", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test(r: @AnyResource) {
              guard let typedR <- r as? @R else {
                  return
              }
              destroy typedR
          }
	    `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("must handle resource in main branch", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
	      resource R {}

	      fun test(r: @AnyResource) {
	          guard let typedR <- r as? @R else {
	              destroy r
	              return
	          }
	      }
		`)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})
}
