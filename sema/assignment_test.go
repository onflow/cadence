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

func TestCheckInvalidUnknownDeclarationAssignment(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          x = 2
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckInvalidConstantAssignment(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          let x = 2
          x = 3
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.AssignmentToConstantError{}, errs[0])
}

func TestCheckAssignment(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          var x = 2
          x = 3
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidGlobalConstantAssignment(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let x = 2

      fun test() {
          x = 3
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.AssignmentToConstantError{}, errs[0])
}

func TestCheckGlobalVariableAssignment(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      var x = 2

      fun test(): Int {
          x = 3
          return x
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidAssignmentToParameter(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(x: Int8) {
           x = 2
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.AssignmentToConstantError{}, errs[0])
}

func TestCheckInvalidAssignmentTargetExpression(t *testing.T) {

	t.Parallel()

	t.Run("function invocation result", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun f() {}

          fun test() {
              f() = 2
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAssignmentTargetError{}, errs[0])
	})

	t.Run("index into function invocation result", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun f(): [Int] {
              return [1]
          }

          fun test() {
              f()[0] = 2
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAssignmentTargetError{}, errs[0])
	})

	t.Run("assess member of function invocation result", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {
              var x: Int

              init() {
                  self.x = 1
              }
          }

          let s = S()

          fun f(): S {
              return s
          }

          fun test() {
              f().x = 2
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAssignmentTargetError{}, errs[0])
	})

	t.Run("index into identifier", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let xs = [1]

          fun test() {
              xs[0] = 2
          }
        `)

		require.NoError(t, err)
	})

	t.Run("access member of identifier", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {
              var x: Int

              init() {
                  self.x = 1
              }
          }

          let s = S()

          fun test() {
              s.x = 2
          }
        `)

		require.NoError(t, err)
	})

	t.Run("index into array literal", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test() {
              [1][0] = 2
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAssignmentTargetError{}, errs[0])
	})

	t.Run("index into dictionary literal", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test() {
              {"a": 1}["a"] = 2
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAssignmentTargetError{}, errs[0])
	})
}
