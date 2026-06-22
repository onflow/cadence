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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
)

func TestInterpretWhileStatement(t *testing.T) {

	t.Parallel()

	invokable := parseCheckAndPrepare(t, `
       fun test(): Int {
           var x = 0
           while x < 5 {
               x = x + 2
           }
           return x
       }

    `)

	value, err := invokable.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		invokable,
		interpreter.NewUnmeteredIntValueFromInt64(6),
		value,
	)
}

func TestInterpretWhileStatementWithReturn(t *testing.T) {

	t.Parallel()

	invokable := parseCheckAndPrepare(t, `
       fun test(): Int {
           var x = 0
           while x < 10 {
               x = x + 2
               if x > 5 {
                   return x
               }
           }
           return x
       }
    `)

	value, err := invokable.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		invokable,
		interpreter.NewUnmeteredIntValueFromInt64(6),
		value,
	)
}

func TestInterpretWhileStatementWithContinue(t *testing.T) {

	t.Parallel()

	invokable := parseCheckAndPrepare(t, `
       fun test(): Int {
           var i = 0
           var x = 0
           while i < 10 {
               i = i + 1
               if i < 5 {
                   continue
               }
               x = x + 1
           }
           return x
       }
    `)

	value, err := invokable.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		invokable,
		interpreter.NewUnmeteredIntValueFromInt64(6),
		value,
	)
}

func TestInterpretWhileStatementWithBreak(t *testing.T) {

	t.Parallel()

	invokable := parseCheckAndPrepare(t, `
       fun test(): Int {
           var x = 0
           while x < 10 {
               x = x + 1
               if x == 5 {
                   break
               }
           }
           return x
       }
    `)

	value, err := invokable.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		invokable,
		interpreter.NewUnmeteredIntValueFromInt64(5),
		value,
	)
}

func TestInterpretWhileStatementCapturingLoopLocals(t *testing.T) {

	t.Parallel()

	test := func(name, middle string, expected ...int64) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf(`
              fun test(): [Int] {
                  let fs: [fun(): Int] = []

                  %s

                  let values: [Int] = []
                  for f in fs {
                      values.append(f())
                  }
                  return values
              }
            `,
				middle,
			)

			inter := parseCheckAndPrepare(t, code)

			value, err := inter.Invoke("test")
			require.NoError(t, err)

			require.IsType(t, value, &interpreter.ArrayValue{})
			arrayValue := value.(*interpreter.ArrayValue)

			expectedValues := make([]interpreter.Value, len(expected))
			for i, e := range expected {
				expectedValues[i] = interpreter.NewUnmeteredIntValueFromInt64(e)
			}

			AssertValueSlicesEqual(
				t,
				inter,
				expectedValues,
				ArrayElements(inter, arrayValue),
			)
		})
	}

	// Each iteration declares a fresh body local, captured by a single closure.
	test("body local",
		`var x = 0
         while x < 3 {
             x = x + 1
             let y = x * 10
             fs.append(fun (): Int { return y })
         }`,
		10, 20, 30,
	)

	// Two closures created in the same iteration share that iteration's binding,
	// so an update made between them is visible to both.
	test("multiple closures, update in between",
		`var x = 0
         while x < 3 {
             x = x + 1
             var y = x * 10
             fs.append(fun (): Int { return y })
             y = y + 1
             fs.append(fun (): Int { return y })
         }`,
		11, 11, 21, 21, 31, 31,
	)

	// The body local must get an independent binding even when the iteration's
	// back-edge is reached via an explicit `continue`.
	test("body local via continue",
		`var x = 0
         while x < 3 {
             x = x + 1
             let y = x * 10
             fs.append(fun (): Int { return y })
             if x == 2 {
                 continue
             }
         }`,
		10, 20, 30,
	)

	// Two sibling nested blocks in the same iteration, each capturing its own
	// local, must get independent bindings.
	test("sibling scopes",
		`var x = 0
         while x < 2 {
             x = x + 1
             if true {
                 let a = x * 10
                 fs.append(fun (): Int { return a })
             }
             if true {
                 let b = x * 100
                 fs.append(fun (): Int { return b })
             }
         }`,
		10, 100, 20, 200,
	)

	// Same as above, but the two sibling locals share the same name; they must
	// still resolve to independent slots.
	test("sibling scopes shadowed",
		`var x = 0
         while x < 2 {
             x = x + 1
             if true {
                 let a = x * 10
                 fs.append(fun (): Int { return a })
             }
             if true {
                 let a = x * 100
                 fs.append(fun (): Int { return a })
             }
         }`,
		10, 100, 20, 200,
	)

	// The captured local is declared only in some iterations. Closing the
	// upvalue on iterations that did not declare it must be a harmless no-op.
	test("conditional",
		`var x = 0
         while x < 4 {
             x = x + 1
             if x % 2 == 0 {
                 let a = x
                 fs.append(fun (): Int { return a })
             }
         }`,
		2, 4,
	)

	// A closure captures a local from the outer loop body AND a local from the
	// inner loop body. The outer local must be fresh per outer iteration, the
	// inner local fresh per inner iteration.
	test("nested loops",
		`var i = 0
         while i < 2 {
             i = i + 1
             let x = i * 100
             var j = 0
             while j < 2 {
                 j = j + 1
                 let y = j * 10
                 fs.append(fun (): Int { return x + y })
             }
         }`,
		110, 120, 210, 220,
	)

	// Two sibling loops in the same function. Each loop's back-edge must only
	// affect its own locals, exercising the single per-function locals list.
	test("sibling loops",
		`var i = 0
         while i < 2 {
             i = i + 1
             let x = i
             fs.append(fun (): Int { return x })
         }
         var j = 0
         while j < 2 {
             j = j + 1
             let y = j * 10
             fs.append(fun (): Int { return y })
         }`,
		1, 2, 10, 20,
	)

	// A local declared *before* the loop is captured alongside a per-iteration
	// local. The pre-loop local must NOT be reset per iteration: it is a single
	// shared binding, so all closures observe its final value, while the
	// per-iteration local stays distinct.
	test("pre-loop local",
		`var shared = 0
         var x = 0
         while x < 3 {
             x = x + 1
             shared = shared + x
             let local = x
             fs.append(fun (): Int { return shared + local })
         }`,
		7, 8, 9,
	)
}
