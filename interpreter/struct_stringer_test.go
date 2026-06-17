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

func TestStringerBasic(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
		access(all)
		struct Example: StructStringer {
			view fun toString(): String {
				return "example"
			}  
		}
		fun test(): String {
			return Example().toString()
		}
	`)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("example"),
		result,
	)
}

func TestStringerBuiltIn(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
		access(all)
		fun test(): String {
			let v = 1
			return v.toString()
		}
	`)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("1"),
		result,
	)
}

func TestStringerCast(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
		access(all)
		fun test(): String {
			var s = 1
			var somevalue = s as {StructStringer}
			return somevalue.toString()
		}
	`)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("1"),
		result,
	)
}

func TestStringerFixedPointAndAddress(t *testing.T) {

	t.Parallel()

	runTest := func(t *testing.T, code, expected string) {
		inter := parseCheckAndPrepare(t, code)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue(expected),
			result,
		)
	}

	// Static cast to `{StructStringer}`, then call `toString`.
	t.Run("static cast", func(t *testing.T) {
		t.Parallel()

		for _, test := range []struct {
			value    string
			expected string
		}{
			{"1.0 as Fix64", "1.00000000"},
			{"1.0 as UFix64", "1.00000000"},
			{"1.0 as Fix128", "1.000000000000000000000000"},
			{"1.0 as UFix128", "1.000000000000000000000000"},
			{"0x1 as Address", "0x0000000000000001"},
		} {
			value, expected := test.value, test.expected
			t.Run(value, func(t *testing.T) {
				t.Parallel()

				runTest(t, fmt.Sprintf(`
					access(all)
					fun test(): String {
						let v = %s as {StructStringer}
						return v.toString()
					}
				`, value), expected)
			})
		}
	})

	// Runtime cast (`as?`) to `{StructStringer}`, then call `toString`.
	t.Run("runtime cast", func(t *testing.T) {
		t.Parallel()

		for _, test := range []struct {
			value    string
			expected string
		}{
			{"1.0 as Fix64", "1.00000000"},
			{"1.0 as UFix64", "1.00000000"},
			{"1.0 as Fix128", "1.000000000000000000000000"},
			{"1.0 as UFix128", "1.000000000000000000000000"},
			{"0x1 as Address", "0x0000000000000001"},
		} {
			value, expected := test.value, test.expected
			t.Run(value, func(t *testing.T) {
				t.Parallel()

				runTest(t, fmt.Sprintf(`
					access(all)
					fun test(): String {
						let v = %s
						let s = (v as AnyStruct) as? {StructStringer}
						return s!.toString()
					}
				`, value), expected)
			})
		}
	})
}

func TestStringerAsValue(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
		access(all)
		fun test(): String {
			var v = Type<{StructStringer}>()
			return v.identifier
		}
	`)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("{StructStringer}"),
		result,
	)
}
