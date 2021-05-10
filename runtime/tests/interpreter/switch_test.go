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

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/checker"
)

func TestInterpretSwitchStatement(t *testing.T) {

	t.Parallel()

	t.Run("Bool", func(t *testing.T) {

		inter := parseCheckAndInterpretWithOptions(t,
			`
              fun test(_ x: Bool): Int {
                  switch x {
                  case true:
                      return 1
                  case false:
                      return 2
                  default:
                      return 3
                  }
                  return 4
              }
            `,
			ParseCheckAndInterpretOptions{
				HandleCheckerError: func(err error) {
					errs := checker.ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
				},
			},
		)

		for argument, expected := range map[interpreter.Value]interpreter.Value{
			interpreter.BoolValue(true):  interpreter.NewIntValueFromInt64(1),
			interpreter.BoolValue(false): interpreter.NewIntValueFromInt64(2),
		} {

			actual, err := inter.Invoke("test", argument)
			require.NoError(t, err)

			assert.Equal(t, expected, actual)
		}
	})

	t.Run("Int", func(t *testing.T) {

		inter := parseCheckAndInterpretWithOptions(t,
			`
              fun test(_ x: Int): String {
                  switch x {
                  case 1:
                      return "1"
                  case 2:
                      return "2"
                  default:
                      return "3"
                  }
                  return "4"
              }
            `,
			ParseCheckAndInterpretOptions{
				HandleCheckerError: func(err error) {
					errs := checker.ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
				},
			},
		)

		for argument, expected := range map[interpreter.Value]interpreter.Value{
			interpreter.NewIntValueFromInt64(1): interpreter.NewStringValue("1"),
			interpreter.NewIntValueFromInt64(2): interpreter.NewStringValue("2"),
			interpreter.NewIntValueFromInt64(3): interpreter.NewStringValue("3"),
			interpreter.NewIntValueFromInt64(4): interpreter.NewStringValue("3"),
		} {

			actual, err := inter.Invoke("test", argument)
			require.NoError(t, err)

			assert.Equal(t, expected, actual)
		}
	})

	t.Run("break", func(t *testing.T) {

		inter := parseCheckAndInterpretWithOptions(t,
			`
              fun test(_ x: Int): String {
                  switch x {
                  case 1:
                      break
                      return "1"
                  case 2:
                      return "2"
                  default:
                      return "3"
                  }
                  return "4"
              }
            `,
			ParseCheckAndInterpretOptions{
				HandleCheckerError: func(err error) {
					errs := checker.ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
				},
			},
		)

		for argument, expected := range map[interpreter.Value]interpreter.Value{
			interpreter.NewIntValueFromInt64(1): interpreter.NewStringValue("4"),
			interpreter.NewIntValueFromInt64(2): interpreter.NewStringValue("2"),
			interpreter.NewIntValueFromInt64(3): interpreter.NewStringValue("3"),
			interpreter.NewIntValueFromInt64(4): interpreter.NewStringValue("3"),
		} {

			actual, err := inter.Invoke("test", argument)
			require.NoError(t, err)

			assert.Equal(t, expected, actual)
		}
	})

	t.Run("no-implicit fallthrough", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
          fun test(_ x: Int): [String] {
              let results: [String] = []
              switch x {
              case 1:
                  results.append("1")
              case 2:
                  results.append("2")
              default:
                  results.append("3")
              }
              return results
          }
        `)

		for argument, expected := range map[interpreter.Value]interpreter.Value{
			interpreter.NewIntValueFromInt64(1): interpreter.NewArrayValueUnownedNonCopying(
				interpreter.NewStringValue("1"),
			),
			interpreter.NewIntValueFromInt64(2): interpreter.NewArrayValueUnownedNonCopying(
				interpreter.NewStringValue("2"),
			),
			interpreter.NewIntValueFromInt64(3): interpreter.NewArrayValueUnownedNonCopying(
				interpreter.NewStringValue("3"),
			),
			interpreter.NewIntValueFromInt64(4): interpreter.NewArrayValueUnownedNonCopying(
				interpreter.NewStringValue("3"),
			),
		} {

			actual, err := inter.Invoke("test", argument)
			require.NoError(t, err)

			assert.Equal(t, expected, actual)
		}
	})

	t.Run("optional", func(t *testing.T) {

		inter := parseCheckAndInterpretWithOptions(t,
			`
              fun test(_ x: Int?, _ y: Int?): String {
                  switch x {
                  case y:
                      return "1"
                  case nil:
                      return "2"
                  default:
                      return "3"
                  }
                  return "4"
              }
            `,
			ParseCheckAndInterpretOptions{
				HandleCheckerError: func(err error) {
					errs := checker.ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
				},
			},
		)

		type testCase struct {
			arguments []interpreter.Value
			expected  interpreter.Value
		}

		for _, testCase := range []testCase{
			{
				[]interpreter.Value{
					interpreter.NewSomeValueOwningNonCopying(
						interpreter.NewIntValueFromInt64(1),
					),
					interpreter.NewSomeValueOwningNonCopying(
						interpreter.NewIntValueFromInt64(1),
					),
				},
				interpreter.NewStringValue("1"),
			},
			{
				[]interpreter.Value{
					interpreter.NilValue{},
					interpreter.NewSomeValueOwningNonCopying(
						interpreter.NewIntValueFromInt64(1),
					),
				},
				interpreter.NewStringValue("2"),
			},
			{
				[]interpreter.Value{
					interpreter.NewSomeValueOwningNonCopying(
						interpreter.NewIntValueFromInt64(1),
					),
					interpreter.NewSomeValueOwningNonCopying(
						interpreter.NewIntValueFromInt64(2),
					),
				},
				interpreter.NewStringValue("3"),
			},
		} {
			actual, err := inter.Invoke("test", testCase.arguments...)
			require.NoError(t, err)

			assert.Equal(t, testCase.expected, actual)
		}
	})
}
