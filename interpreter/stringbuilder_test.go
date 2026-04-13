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

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
)

func TestInterpretStringBuilder(t *testing.T) {

	t.Parallel()

	t.Run("empty toString", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): String {
                let sb = StringBuilder()
                return sb.toString()
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue(""),
			result,
		)
	})

	t.Run("append strings", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): String {
                let sb = StringBuilder()
                sb.append("hello")
                sb.append(" ")
                sb.append("world")
                return sb.toString()
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("hello world"),
			result,
		)
	})

	t.Run("append character", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): String {
                let sb = StringBuilder()
                sb.append("ab")
                sb.appendCharacter("c")
                sb.appendCharacter("d")
                return sb.toString()
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("abcd"),
			result,
		)
	})

	t.Run("length", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): Int {
                let sb = StringBuilder()
                sb.append("hello")
                return sb.length
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewIntValueFromInt64(nil, 5),
			result,
		)
	})

	t.Run("clear and reuse", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): String {
                let sb = StringBuilder()
                sb.append("first")
                sb.clear()
                sb.append("second")
                return sb.toString()
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("second"),
			result,
		)
	})

	t.Run("length after clear", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): Int {
                let sb = StringBuilder()
                sb.append("hello")
                sb.clear()
                return sb.length
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewIntValueFromInt64(nil, 0),
			result,
		)
	})

	t.Run("multiple toString calls", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): [String] {
                let sb = StringBuilder()
                sb.append("hello")
                let s1 = sb.toString()
                sb.append(" world")
                let s2 = sb.toString()
                return [s1, s2]
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredStringValue("hello"),
				interpreter.NewUnmeteredStringValue("hello world"),
			),
			result,
		)
	})

	t.Run("pass to function", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun addGreeting(_ sb: StringBuilder) {
                sb.append("hello")
            }

            fun test(): String {
                let sb = StringBuilder()
                addGreeting(sb)
                return sb.toString()
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("hello"),
			result,
		)
	})

	t.Run("empty length", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): Int {
                let sb = StringBuilder()
                return sb.length
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewIntValueFromInt64(nil, 0),
			result,
		)
	})

	t.Run("constructor type", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): StringBuilder {
                return StringBuilder()
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		compositeValue := result.(*interpreter.SimpleCompositeValue)
		assert.Equal(t, sema.StringBuilderType.ID(), compositeValue.TypeID)
	})

	t.Run("unicode", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): String {
                let sb = StringBuilder()
                sb.append("Hello")
                sb.append(" ")
                sb.append("\u{1F30D}")
                sb.append(" ")
                sb.append("\u{4E16}\u{754C}")
                return sb.toString()
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("Hello 🌍 世界"),
			result,
		)
	})

	t.Run("unicode character", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): String {
                let sb = StringBuilder()
                sb.append("Test")
                sb.appendCharacter("\u{1F389}")
                sb.appendCharacter("\u{2728}")
                return sb.toString()
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("Test🎉✨"),
			result,
		)
	})

	t.Run("multiple clear and reuse", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): [String] {
                let sb = StringBuilder()
                let results: [String] = []

                sb.append("First")
                results.append(sb.toString())

                sb.clear()
                sb.append("Second")
                results.append(sb.toString())

                sb.clear()
                sb.append("Third")
                results.append(sb.toString())

                return results
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredStringValue("First"),
				interpreter.NewUnmeteredStringValue("Second"),
				interpreter.NewUnmeteredStringValue("Third"),
			),
			result,
		)
	})

	t.Run("in loop", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): String {
                let sb = StringBuilder()
                var i = 0
                while i < 5 {
                    sb.append(i.toString())
                    if i < 4 {
                        sb.append(",")
                    }
                    i = i + 1
                }
                return sb.toString()
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("0,1,2,3,4"),
			result,
		)
	})

	t.Run("large string", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): Int {
                let sb = StringBuilder()
                var i = 0
                while i < 100 {
                    sb.append("Hello World ")
                    i = i + 1
                }
                return sb.length
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		// "Hello World " is 12 characters, repeated 100 times = 1200
		RequireValuesEqual(
			t,
			inter,
			interpreter.NewIntValueFromInt64(nil, 1200),
			result,
		)
	})

	t.Run("not storable", func(t *testing.T) {
		t.Parallel()

		assert.False(t, sema.StringBuilderType.IsStorable(map[*sema.Member]bool{}))
	})

	t.Run("not importable", func(t *testing.T) {
		t.Parallel()

		assert.False(t, sema.StringBuilderType.IsImportable(map[*sema.Member]bool{}))
	})

	t.Run("mixed append methods", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): String {
                let sb = StringBuilder()
                sb.append("A")
                sb.append("B")
                sb.appendCharacter("C")
                sb.append("D")
                return sb.toString()
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("ABCD"),
			result,
		)
	})

	t.Run("toString idempotent", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): Bool {
                let sb = StringBuilder()
                sb.append("Hello")
                let first = sb.toString()
                let second = sb.toString()
                return first == second
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			result,
		)
	})
}
