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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestCheckInvalidUnknownDeclarationSwap(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          var x = 1
          x <-> y
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)
	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	assert.IsType(t, &sema.NotDeclaredError{}, errs[1])
}

func TestCheckInvalidLeftConstantSwap(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          let x = 2
          var y = 1
          x <-> y
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.AssignmentToConstantError{}, errs[0])
}

func TestCheckInvalidRightConstantSwap(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          var x = 2
          let y = 1
          x <-> y
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.AssignmentToConstantError{}, errs[0])
}

func TestCheckSwap(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          var x = 2
          var y = 3
          x <-> y
      }
    `)

	assert.NoError(t, err)
}

func TestCheckInvalidTypesSwap(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          var x = 2
          var y = "1"
          x <-> y
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
}

func TestCheckInvalidTypesSwap2(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          var x = "2"
          var y = 1
          x <-> y
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
}

func TestCheckInvalidSwapTargetExpressionLeft(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          var x = 1
          f() <-> x
      }

      fun f(): Int {
          return 2
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidSwapExpressionError{}, errs[0])
}

func TestCheckInvalidSwapTargetExpressionRight(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          var x = 1
          x <-> f()
      }

      fun f(): Int {
          return 2
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidSwapExpressionError{}, errs[0])
}

func TestCheckInvalidSwapTargetExpressions(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          f() <-> f()
      }

      fun f(): Int {
          return 2
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.InvalidSwapExpressionError{}, errs[0])
	assert.IsType(t, &sema.InvalidSwapExpressionError{}, errs[1])
}

func TestCheckSwapOptional(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          var x: Int? = 2
          var y: Int? = nil
          x <-> y
      }
    `)

	assert.NoError(t, err)
}

func TestCheckSwapResourceArrayElementAndVariable(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs <- [<-create X()]
          var x <- create X()
          x <-> xs[0]
          destroy x
          destroy xs
      }
    `)

	assert.NoError(t, err)
}

func TestCheckSwapResourceArrayElements(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs <- [<-create X(), <-create X()]
          xs[0] <-> xs[1]
          destroy xs
      }
    `)

	assert.NoError(t, err)
}

func TestCheckSwapResourceFields(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      resource Y {
          var x: @X

          init(x: @X) {
              self.x <- x
          }
      }

      fun test() {
          let y1 <- create Y(x: <-create X())
          let y2 <- create Y(x: <-create X())
          y1.x <-> y2.x
          destroy y1
          destroy y2
      }
    `)

	assert.NoError(t, err)
}

// TestCheckInvalidSwapConstantResourceFields tests that it is invalid
// to swap fields which are constant (`let`)
func TestCheckInvalidSwapConstantResourceFields(t *testing.T) {

	t.Parallel()

	for i := 0; i < 2; i++ {

		first := "var"
		second := "let"

		if i == 1 {
			first = "let"
			second = "var"
		}

		testName := fmt.Sprintf("%s_%s", first, second)

		t.Run(testName, func(t *testing.T) {

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      resource X {}

                      resource Y {
                          %[1]s x: @X

                          init(x: @X) {
                              self.x <- x
                          }
                      }

                      resource Z {
                          %[2]s x: @X

                          init(x: @X) {
                              self.x <- x
                          }
                      }

                      fun test() {
                          let y <- create Y(x: <-create X())
                          let z <- create Z(x: <-create X())
                          y.x <-> z.x
                          destroy y
                          destroy z
                      }
                    `,
					first,
					second,
				))

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.AssignmentToConstantMemberError{}, errs[0])
		})
	}
}

func TestCheckSwapResourceDictionaryElement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs: @{String: X} <- {}
          var x: @X? <- create X()
          xs["foo"] <-> x
          destroy xs
          destroy x
      }
    `)

	assert.NoError(t, err)
}

func TestCheckInvalidSwapResourceDictionaryElement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs: @{String: X} <- {}
          var x <- create X()
          xs["foo"] <-> x
          destroy xs
          destroy x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidTwoConstantsSwap(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        fun test() {
            let x = 1
            let y = 1

            x <-> y
        }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	require.IsType(t, &sema.AssignmentToConstantError{}, errs[0])
	assignmentError := errs[0].(*sema.AssignmentToConstantError)
	assert.Equal(t, "x", assignmentError.Name)

	require.IsType(t, &sema.AssignmentToConstantError{}, errs[1])
	assignmentError = errs[1].(*sema.AssignmentToConstantError)
	assert.Equal(t, "y", assignmentError.Name)
}

func TestCheckIndexSwapWithInvalidExpression(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          let xs = [1]

          // NOTE: ys is not declared
          xs[0] <-> ys[0]
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	require.IsType(t, &sema.NotDeclaredError{}, errs[0])
	require.IsType(t, &sema.NotDeclaredError{}, errs[1])
}
