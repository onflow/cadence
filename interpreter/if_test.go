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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/onflow/cadence/tests/sema_utils"
	. "github.com/onflow/cadence/tests/utils"

	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

func TestInterpretIfStatement(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndInterpretWithOptions(t,
		`
           access(all) fun testTrue(): Int {
               if true {
                   return 2
               } else {
                   return 3
               }
               return 4
           }

           access(all) fun testFalse(): Int {
               if false {
                   return 2
               } else {
                   return 3
               }
               return 4
           }

           access(all) fun testNoElse(): Int {
               if true {
                   return 2
               }
               return 3
           }

           access(all) fun testElseIf(): Int {
               if false {
                   return 2
               } else if true {
                   return 3
               }
               return 4
           }
           
           access(all) fun testElseIfElse(): Int {
               if false {
                   return 2
               } else if false {
                   return 3
               } else {
                   return 4
               }
           }
        `,
		ParseCheckAndInterpretOptions{
			HandleCheckerError: func(err error) {
				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
				assert.IsType(t, &sema.UnreachableStatementError{}, errs[1])
			},
		},
	)
	require.NoError(t, err)

	for name, expected := range map[string]int64{
		"testTrue":   2,
		"testFalse":  3,
		"testNoElse": 2,
		"testElseIf": 3,
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
}

func TestInterpretIfStatementTestWithDeclaration(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var branch = 0

      fun test(x: Int?): Int {
          if var y = x {
              branch = 1
              return y
          } else {
              branch = 2
              return 0
          }
      }
    `)

	t.Run("2", func(t *testing.T) {
		value, err := inter.Invoke(
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(2),
		)
		require.NoError(t, err)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(2),
			value,
		)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(1),
			inter.Globals.Get("branch").GetValue(inter),
		)
	})

	t.Run("nil", func(t *testing.T) {
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
			interpreter.NewUnmeteredIntValueFromInt64(2),
			inter.Globals.Get("branch").GetValue(inter),
		)
	})
}

func TestInterpretIfStatementTestWithDeclarationAndElse(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var branch = 0

      fun test(x: Int?): Int {
          if var y = x {
              branch = 1
              return y
          }
          branch = 2
          return 0
      }
    `)

	t.Run("2", func(t *testing.T) {
		value, err := inter.Invoke(
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(2),
		)
		require.NoError(t, err)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(2),
			value,
		)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(1),
			inter.Globals.Get("branch").GetValue(inter),
		)
	})

	t.Run("nil", func(t *testing.T) {
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
			interpreter.NewUnmeteredIntValueFromInt64(2),
			inter.Globals.Get("branch").GetValue(inter),
		)

	})
}

func TestInterpretIfStatementTestWithDeclarationNestedOptionals(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var branch = 0

      fun test(x: Int??): Int? {
          if var y = x {
              branch = 1
              return y
          } else {
              branch = 2
              return 0
          }
      }
    `)

	t.Run("2", func(t *testing.T) {
		value, err := inter.Invoke(
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(2),
		)
		require.NoError(t, err)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(2),
			),
			value,
		)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(1),
			inter.Globals.Get("branch").GetValue(inter),
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
			interpreter.NewUnmeteredIntValueFromInt64(2),
			inter.Globals.Get("branch").GetValue(inter),
		)
	})
}

func TestInterpretIfStatementTestWithDeclarationNestedOptionalsExplicitAnnotation(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var branch = 0

      fun test(x: Int??): Int? {
          if var y: Int? = x {
              branch = 1
              return y
          } else {
              branch = 2
              return 0
          }
      }
    `)

	t.Run("2", func(t *testing.T) {
		value, err := inter.Invoke(
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(2),
		)
		require.NoError(t, err)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(2),
			),
			value,
		)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(1),
			inter.Globals.Get("branch").GetValue(inter),
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
			interpreter.NewUnmeteredIntValueFromInt64(2),
			inter.Globals.Get("branch").GetValue(inter),
		)
	})
}
