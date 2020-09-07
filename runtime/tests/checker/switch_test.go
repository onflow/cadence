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

      fun test() {
          let s = S()

          switch true {
          case s:
          }
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

          switch x {
          case s:
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
          switch true {
          default:
              // ...
          case true:
              // ...
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
          switch true {
          default:
              // ...
          default:
              // ...
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
