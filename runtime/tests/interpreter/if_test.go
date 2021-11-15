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

	. "github.com/onflow/cadence/runtime/tests/utils"

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/checker"
)

func TestInterpretIfStatement(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndInterpretWithOptions(t,
		`
           pub fun testTrue(): Int {
               if true {
                   return 2
               } else {
                   return 3
               }
               return 4
           }

           pub fun testFalse(): Int {
               if false {
                   return 2
               } else {
                   return 3
               }
               return 4
           }

           pub fun testNoElse(): Int {
               if true {
                   return 2
               }
               return 3
           }

           pub fun testElseIf(): Int {
               if false {
                   return 2
               } else if true {
                   return 3
               }
               return 4
           }
           
           pub fun testElseIfElse(): Int {
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
				errs := checker.ExpectCheckerErrors(t, err, 2)

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
				interpreter.NewIntValueFromInt64(expected),
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
			interpreter.NewIntValueFromInt64(2),
		)
		require.NoError(t, err)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewIntValueFromInt64(2),
			value,
		)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewIntValueFromInt64(1),
			inter.Globals["branch"].GetValue(),
		)
	})

	t.Run("nil", func(t *testing.T) {
		value, err := inter.Invoke("test", interpreter.NilValue{})
		require.NoError(t, err)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewIntValueFromInt64(0),
			value,
		)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewIntValueFromInt64(2),
			inter.Globals["branch"].GetValue(),
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
			interpreter.NewIntValueFromInt64(2),
		)
		require.NoError(t, err)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewIntValueFromInt64(2),
			value,
		)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewIntValueFromInt64(1),
			inter.Globals["branch"].GetValue(),
		)
	})

	t.Run("nil", func(t *testing.T) {
		value, err := inter.Invoke("test", interpreter.NilValue{})
		require.NoError(t, err)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewIntValueFromInt64(0),
			value,
		)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewIntValueFromInt64(2),
			inter.Globals["branch"].GetValue(),
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
			interpreter.NewIntValueFromInt64(2),
		)
		require.NoError(t, err)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewSomeValueNonCopying(
				interpreter.NewIntValueFromInt64(2),
			),
			value,
		)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewIntValueFromInt64(1),
			inter.Globals["branch"].GetValue(),
		)
	})

	t.Run("nil", func(t *testing.T) {
		value, err := inter.Invoke("test", interpreter.NilValue{})
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewSomeValueNonCopying(
				interpreter.NewIntValueFromInt64(0),
			),
			value,
		)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewIntValueFromInt64(2),
			inter.Globals["branch"].GetValue(),
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
			interpreter.NewIntValueFromInt64(2),
		)
		require.NoError(t, err)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewSomeValueNonCopying(
				interpreter.NewIntValueFromInt64(2),
			),
			value,
		)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewIntValueFromInt64(1),
			inter.Globals["branch"].GetValue(),
		)

	})

	t.Run("nil", func(t *testing.T) {
		value, err := inter.Invoke("test", interpreter.NilValue{})
		require.NoError(t, err)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewSomeValueNonCopying(
				interpreter.NewIntValueFromInt64(0),
			),
			value,
		)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewIntValueFromInt64(2),
			inter.Globals["branch"].GetValue(),
		)
	})
}
