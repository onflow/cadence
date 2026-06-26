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

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
)

func TestInterpretSwitchStatement(t *testing.T) {

	t.Parallel()

	t.Run("Bool", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
          fun test(_ x: Bool): Int {
              switch x {
              case true:
                  return 1
              case false:
                  return 2
              default:
                  return 3
              }
          }
        `)

		for argument, expected := range map[interpreter.Value]interpreter.Value{
			interpreter.TrueValue:  interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.FalseValue: interpreter.NewUnmeteredIntValueFromInt64(2),
		} {

			actual, err := inter.Invoke("test", argument)
			require.NoError(t, err)

			AssertValuesEqual(t, inter, expected, actual)
		}
	})

	t.Run("Int", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
          fun test(_ x: Int): String {
              switch x {
              case 1:
                  return "1"
              case 2:
                  return "2"
              default:
                  return "3"
              }
          }
        `)

		for argument, expected := range map[interpreter.Value]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1): interpreter.NewUnmeteredStringValue("1"),
			interpreter.NewUnmeteredIntValueFromInt64(2): interpreter.NewUnmeteredStringValue("2"),
			interpreter.NewUnmeteredIntValueFromInt64(3): interpreter.NewUnmeteredStringValue("3"),
			interpreter.NewUnmeteredIntValueFromInt64(4): interpreter.NewUnmeteredStringValue("3"),
		} {

			actual, err := inter.Invoke("test", argument)
			require.NoError(t, err)

			AssertValuesEqual(t, inter, expected, actual)
		}
	})

	t.Run("break", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
          fun test(_ x: Int): String {
              switch x {
              case 1:
                  break
              case 2:
                  return "2"
              default:
                  return "3"
              }
			  return "4"
          }
        `)

		for argument, expected := range map[interpreter.Value]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1): interpreter.NewUnmeteredStringValue("4"),
			interpreter.NewUnmeteredIntValueFromInt64(2): interpreter.NewUnmeteredStringValue("2"),
			interpreter.NewUnmeteredIntValueFromInt64(3): interpreter.NewUnmeteredStringValue("3"),
			interpreter.NewUnmeteredIntValueFromInt64(4): interpreter.NewUnmeteredStringValue("3"),
		} {

			actual, err := inter.Invoke("test", argument)
			require.NoError(t, err)

			AssertValuesEqual(t, inter, expected, actual)
		}
	})

	t.Run("no-implicit fallthrough", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
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

		for argument, expectedValues := range map[interpreter.Value][]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1): {
				interpreter.NewUnmeteredStringValue("1"),
			},
			interpreter.NewUnmeteredIntValueFromInt64(2): {
				interpreter.NewUnmeteredStringValue("2"),
			},
			interpreter.NewUnmeteredIntValueFromInt64(3): {
				interpreter.NewUnmeteredStringValue("3"),
			},
			interpreter.NewUnmeteredIntValueFromInt64(4): {
				interpreter.NewUnmeteredStringValue("3"),
			},
		} {

			actual, err := inter.Invoke("test", argument)
			require.NoError(t, err)

			require.IsType(t, &interpreter.ArrayValue{}, actual)
			arrayValue := actual.(*interpreter.ArrayValue)

			AssertValueSlicesEqual(
				t,
				inter,
				expectedValues,
				ArrayElements(inter, arrayValue),
			)
		}
	})

	t.Run("no default, no match falls through", func(t *testing.T) {

		t.Parallel()

		// A switch without a `default` case where no case matches must
		// continue execution after the switch, leaving `result` unchanged.
		inter := parseCheckAndPrepare(t, `
          fun test(_ x: Int): Int {
              var result = 0
              switch x {
              case 1:
                  result = 10
              case 2:
                  result = 20
              }
              return result
          }
        `)

		for argument, expected := range map[interpreter.Value]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1): interpreter.NewUnmeteredIntValueFromInt64(10),
			interpreter.NewUnmeteredIntValueFromInt64(2): interpreter.NewUnmeteredIntValueFromInt64(20),
			interpreter.NewUnmeteredIntValueFromInt64(3): interpreter.NewUnmeteredIntValueFromInt64(0),
			interpreter.NewUnmeteredIntValueFromInt64(0): interpreter.NewUnmeteredIntValueFromInt64(0),
		} {

			actual, err := inter.Invoke("test", argument)
			require.NoError(t, err)

			AssertValuesEqual(t, inter, expected, actual)
		}
	})

	t.Run("String", func(t *testing.T) {

		t.Parallel()

		// Switch case matching uses `==`. For a String subject this exercises
		// string equality, a distinct code path from integer equality.
		inter := parseCheckAndPrepare(t, `
          fun test(_ x: String): Int {
              switch x {
              case "one":
                  return 1
              case "two":
                  return 2
              default:
                  return 3
              }
          }
        `)

		for argument, expected := range map[interpreter.Value]interpreter.Value{
			interpreter.NewUnmeteredStringValue("one"):   interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredStringValue("two"):   interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredStringValue("three"): interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredStringValue(""):      interpreter.NewUnmeteredIntValueFromInt64(3),
		} {

			actual, err := inter.Invoke("test", argument)
			require.NoError(t, err)

			AssertValuesEqual(t, inter, expected, actual)
		}
	})

	t.Run("enum", func(t *testing.T) {

		t.Parallel()

		// Switch over an enum subject matches cases by enum equality.
		inter := parseCheckAndPrepare(t, `
          enum Color: UInt8 {
              case red
              case green
              case blue
          }

          fun test(_ raw: UInt8): Int {
              let c = Color(rawValue: raw)!
              switch c {
              case Color.red:
                  return 10
              case Color.green:
                  return 20
              case Color.blue:
                  return 30
              default:
                  return 0
              }
          }
        `)

		for argument, expected := range map[interpreter.Value]interpreter.Value{
			interpreter.NewUnmeteredUInt8Value(0): interpreter.NewUnmeteredIntValueFromInt64(10),
			interpreter.NewUnmeteredUInt8Value(1): interpreter.NewUnmeteredIntValueFromInt64(20),
			interpreter.NewUnmeteredUInt8Value(2): interpreter.NewUnmeteredIntValueFromInt64(30),
		} {

			actual, err := inter.Invoke("test", argument)
			require.NoError(t, err)

			AssertValuesEqual(t, inter, expected, actual)
		}
	})

	t.Run("optional", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
          fun test(_ x: Int?, _ y: Int?): String {
              switch x {
              case y:
                  return "1"
              case nil:
                  return "2"
              default:
                  return "3"
              }
          }
        `)

		type testCase struct {
			arguments []interpreter.Value
			expected  interpreter.Value
		}

		for _, testCase := range []testCase{
			{
				[]interpreter.Value{
					interpreter.NewUnmeteredSomeValueNonCopying(
						interpreter.NewUnmeteredIntValueFromInt64(1),
					),
					interpreter.NewUnmeteredSomeValueNonCopying(
						interpreter.NewUnmeteredIntValueFromInt64(1),
					),
				},
				interpreter.NewUnmeteredStringValue("1"),
			},
			{
				[]interpreter.Value{
					interpreter.Nil,
					interpreter.NewUnmeteredSomeValueNonCopying(
						interpreter.NewUnmeteredIntValueFromInt64(1),
					),
				},
				interpreter.NewUnmeteredStringValue("2"),
			},
			{
				[]interpreter.Value{
					interpreter.NewUnmeteredSomeValueNonCopying(
						interpreter.NewUnmeteredIntValueFromInt64(1),
					),
					interpreter.NewUnmeteredSomeValueNonCopying(
						interpreter.NewUnmeteredIntValueFromInt64(2),
					),
				},
				interpreter.NewUnmeteredStringValue("3"),
			},
		} {
			actual, err := inter.Invoke("test", testCase.arguments...)
			require.NoError(t, err)

			AssertValuesEqual(t, inter, testCase.expected, actual)
		}
	})
}

func TestInterpretSwitchStatementControlFlowInLoop(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, code string, expected int64) {
		inter := parseCheckAndPrepare(t, code)
		actual, err := inter.Invoke("test")
		require.NoError(t, err)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(expected),
			actual,
		)
	}

	t.Run("continue in switch in while", func(t *testing.T) {
		t.Parallel()

		test(t, `
            fun test(): Int {
                var sum = 0
                var i = 0
                while i < 5 {
                    i = i + 1
                    switch i {
                        case 3:
                            continue
                        case 4:
                            break
                    }
                    sum = sum + i
                }
                return sum
            }
        `,
			1+2+4+5,
		)
	})

	t.Run("continue in switch in for", func(t *testing.T) {
		t.Parallel()

		test(t, `
            fun test(): Int {
                var sum = 0
                for i in [1, 2, 3, 4, 5] {
                    switch i {
                        case 3:
                            continue
                        case 4:
                            break
                    }
                    sum = sum + i
                }
                return sum
            }
        `,
			1+2+4+5,
		)
	})

	t.Run("continue in switch in for with index", func(t *testing.T) {
		t.Parallel()

		test(t, `
            fun test(): Int {
                var sum = 0
                for i, x in [10, 20, 30, 40] {
                    switch i {
                        case 1:
                            continue
                    }
                    sum = sum + x
                }
                return sum
            }
        `,
			10+30+40,
		)
	})

	t.Run("continue in switch in nested loops", func(t *testing.T) {
		t.Parallel()

		test(t, `
            fun test(): Int {
                var sum = 0
                var i = 0
                while i < 3 {
                    i = i + 1
                    var j = 0
                    while j < 3 {
                        j = j + 1
                        switch j {
                            case 2:
                                continue
                        }
                        sum = sum + (i * 10 + j)
                    }
                }
                return sum
            }
        `,
			(11+13)+(21+23)+(31+33),
		)
	})

	t.Run("continue in nested switches in loop", func(t *testing.T) {
		t.Parallel()

		test(t, `
            fun test(): Int {
                var sum = 0
                for i in [1, 2, 3, 4] {
                    switch i % 2 {
                        case 0:
                            switch i {
                                case 4:
                                    continue
                            }
                    }
                    sum = sum + i
                }
                return sum
            }
        `,
			1+2+3,
		)
	})

	t.Run("break in switch only exits switch", func(t *testing.T) {
		t.Parallel()

		test(
			t,
			`
            fun test(): Int {
                var sum = 0
                var i = 0
                while i < 3 {
                    i = i + 1
                    switch i {
                        case 2:
                            break
                    }
                    sum = sum + i
                }
                return sum
            }
        `,
			6,
		)
	})
}
