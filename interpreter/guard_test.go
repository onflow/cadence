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

func TestInterpretGuardStatement(t *testing.T) {

	t.Parallel()

	t.Run("basic expressions", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			fun testTrue(): Int {
				guard true else {
					return 1
				}
				return 2
			}

			fun testFalse(): Int {
				guard false else {
					return 1
				}
				return 2
			}

			fun testMultipleGuards(): Int {
				guard true else {
					return 1
				}
				guard true else {
					return 2
				}
				return 3
			}

			fun testGuardInLoop(): Int {
				var i = 0
				while i < 10 {
					i = i + 1
					guard i != 5 else {
						return i
					}
				}
				return 0
			}
		`)

		for name, expected := range map[string]int64{
			"testTrue":           2,
			"testFalse":          1,
			"testMultipleGuards": 3,
			"testGuardInLoop":    5,
		} {
			t.Run(name, func(t *testing.T) {
				value, err := inter.Invoke(name)
				require.NoError(t, err)

				AssertValuesEqual(
					t,
					inter,
					interpreter.NewUnmeteredIntValueFromInt64(expected),
					value,
				)
			})
		}
	})

	t.Run("with boolean true", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			fun test(): Int {
				guard true else {
					return 1
				}
				return 2
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(2),
			value,
		)
	})

	t.Run("with boolean false", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			fun test(): Int {
				guard false else {
					return 1
				}
				return 2
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(1),
			value,
		)
	})

	t.Run("with break", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			fun test(): Int {
				var result = 0
				while true {
					result = result + 1
					guard result < 5 else {
						break
					}
				}
				return result
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(5),
			value,
		)
	})

	t.Run("with continue", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			fun test(): Int {
				var result = 0
				var i = 0
				while i < 10 {
					i = i + 1
					guard i % 2 == 0 else {
						continue
					}
					result = result + i
				}
				return result
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		// Sum of even numbers from 1 to 10: 2 + 4 + 6 + 8 + 10 = 30
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(30),
			value,
		)
	})
}

func TestInterpretGuardStatementWithOptionalBinding(t *testing.T) {

	t.Parallel()

	t.Run("let binding with Some value", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			var branch = 0

			fun test(x: Int?): Int {
				guard let y = x else {
					branch = 1
					return 0
				}
				// y should be available here and unwrapped
				branch = 2
				return y
			}
		`)

		value, err := inter.Invoke(
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(42),
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(42),
			value,
		)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(2),
			inter.GetGlobal("branch"),
		)
	})

	t.Run("let binding with nil", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			var branch = 0

			fun test(x: Int?): Int {
				guard let y = x else {
					branch = 1
					return 0
				}
				branch = 2
				return y
			}
		`)

		value, err := inter.Invoke("test", interpreter.Nil)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(0),
			value,
		)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(1),
			inter.GetGlobal("branch"),
		)
	})

	t.Run("var binding with Some value", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			fun test(x: Int?): Int {
				guard var y = x else {
					return 0
				}
				// y is mutable
				y = y + 10
				return y
			}
		`)

		value, err := inter.Invoke(
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(5),
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(15),
			value,
		)
	})

	t.Run("var binding with nil", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			fun test(x: Int?): Int {
				guard var y = x else {
					return 0
				}
				y = y + 10
				return y
			}
		`)

		value, err := inter.Invoke("test", interpreter.Nil)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(0),
			value,
		)
	})
}

func TestInterpretGuardStatementVariableScope(t *testing.T) {

	t.Parallel()

	t.Run("variable available after guard", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			fun test(x: String?): String {
				guard let name = x else {
					return "default"
				}
				// name should be available here
				return name
			}
		`)

		value, err := inter.Invoke(
			"test",
			interpreter.NewUnmeteredStringValue("Alice"),
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("Alice"),
			value,
		)
	})

	t.Run("variable available in subsequent code", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			fun test(a: Int?, b: Int?): Int {
				guard let x = a else {
					return 0
				}
				guard let y = b else {
					return 0
				}
				// Both x and y should be available
				return x + y
			}
		`)

		value, err := inter.Invoke(
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(10),
			interpreter.NewUnmeteredIntValueFromInt64(20),
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(30),
			value,
		)
	})
}

func TestInterpretGuardStatementWithResources(t *testing.T) {

	t.Parallel()

	t.Run("basic resource handling", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			resource R {
				let value: Int
				init(value: Int) {
					self.value = value
				}
			}

			fun test(shouldDestroy: Bool): Int {
				let r <- create R(value: 42)
				guard !shouldDestroy else {
					let value = r.value
					destroy r
					return value
				}
				let value = r.value
				destroy r
				return value
			}
		`)

		t.Run("main branch", func(t *testing.T) {
			value, err := inter.Invoke("test", interpreter.FalseValue)
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(42),
				value,
			)
		})

		t.Run("else branch", func(t *testing.T) {
			value, err := inter.Invoke("test", interpreter.TrueValue)
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(42),
				value,
			)
		})
	})

	t.Run("optional resource unwrapping with Some", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			resource R {
				let value: Int
				init(value: Int) {
					self.value = value
				}
			}

			fun test(): Int {
				let r: @R? <- create R(value: 100)
				guard let unwrapped <- r else {
					return 0
				}
				let value = unwrapped.value
				destroy unwrapped
				return value
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(100),
			value,
		)
	})

	t.Run("optional resource unwrapping with nil", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			resource R {
				let value: Int
				init(value: Int) {
					self.value = value
				}
			}

			fun test(): Int {
				let r: @R? <- nil
				guard let unwrapped <- r else {
					return 0
				}
				let value = unwrapped.value
				destroy unwrapped
				return value
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(0),
			value,
		)
	})

	t.Run("resource failable cast succeeds", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			resource R {}

			fun test(): Int {
				let r: @AnyResource <- create R()
				guard let typedR <- r as? @R else {
					destroy r
					return 1
				}
				destroy typedR
				return 2
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(2),
			value,
		)
	})

	t.Run("resource failable cast fails", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			resource R {}
			resource S {}

			fun test(): Int {
				let r: @AnyResource <- create S()
				guard let typedR <- r as? @R else {
					destroy r
					return 1
				}
				destroy typedR
				return 2
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(1),
			value,
		)
	})
}

func TestInterpretGuardStatementNestedOptionals(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
		var branch = 0

		fun test(x: Int??): Int? {
			guard var y = x else {
				branch = 1
				return 0
			}
			branch = 2
			return y
		}
	`)

	t.Run("some value", func(t *testing.T) {
		value, err := inter.Invoke(
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(42),
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(42),
			),
			value,
		)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(2),
			inter.GetGlobal("branch"),
		)
	})

	t.Run("nil", func(t *testing.T) {
		value, err := inter.Invoke("test", interpreter.Nil)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(0),
			),
			value,
		)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(1),
			inter.GetGlobal("branch"),
		)
	})
}

func TestInterpretGuardStatementNestedOptionalsExplicitAnnotation(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
		var branch = 0

		fun test(x: Int??): Int? {
			guard var y: Int? = x else {
				branch = 1
				return 0
			}
			branch = 2
			return y
		}
	`)

	t.Run("some value", func(t *testing.T) {
		value, err := inter.Invoke(
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(42),
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(42),
			),
			value,
		)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(2),
			inter.GetGlobal("branch"),
		)
	})

	t.Run("nil", func(t *testing.T) {
		value, err := inter.Invoke("test", interpreter.Nil)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(0),
			),
			value,
		)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(1),
			inter.GetGlobal("branch"),
		)
	})
}

func TestInterpretGuardStatementComplexConditions(t *testing.T) {

	t.Parallel()

	t.Run("guard with complex boolean expression", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			fun test(a: Int, b: Int): Int {
				guard a > 0 && b > 0 else {
					return -1
				}
				return a + b
			}
		`)

		t.Run("both positive", func(t *testing.T) {
			value, err := inter.Invoke(
				"test",
				interpreter.NewUnmeteredIntValueFromInt64(5),
				interpreter.NewUnmeteredIntValueFromInt64(10),
			)
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(15),
				value,
			)
		})

		t.Run("one negative", func(t *testing.T) {
			value, err := inter.Invoke(
				"test",
				interpreter.NewUnmeteredIntValueFromInt64(-5),
				interpreter.NewUnmeteredIntValueFromInt64(10),
			)
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(-1),
				value,
			)
		})
	})

	t.Run("chained guards", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			fun test(a: Int?, b: Int?, c: Int?): Int {
				guard let x = a else {
					return 0
				}
				guard let y = b else {
					return 0
				}
				guard let z = c else {
					return 0
				}
				return x + y + z
			}
		`)

		t.Run("all some", func(t *testing.T) {
			value, err := inter.Invoke(
				"test",
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
			)
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(6),
				value,
			)
		})

		t.Run("second nil", func(t *testing.T) {
			value, err := inter.Invoke(
				"test",
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.Nil,
				interpreter.NewUnmeteredIntValueFromInt64(3),
			)
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(0),
				value,
			)
		})
	})
}
