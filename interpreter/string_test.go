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

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
)

func TestInterpretRecursiveValueString(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
      fun test(): AnyStruct {
          let map: {String: AnyStruct} = {}
          let mapRef = &map as auth(Mutate) &{String: AnyStruct}
          mapRef["mapRef"] = mapRef
          return map
      }
    `)

	mapValue, err := inter.Invoke("test")
	require.NoError(t, err)

	require.Equal(t,
		`{"mapRef": {"mapRef": ...}}`,
		mapValue.String(),
	)

	require.IsType(t, &interpreter.DictionaryValue{}, mapValue)
	require.Equal(t,
		`{"mapRef": ...}`,
		mapValue.(*interpreter.DictionaryValue).
			GetKey(inter, interpreter.EmptyLocationRange, interpreter.NewUnmeteredStringValue("mapRef")).
			String(),
	)
}

func TestInterpretStringFunction(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
      fun test(): String {
          return String()
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
}

func TestInterpretStringDecodeHex(t *testing.T) {

	t.Parallel()

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
          fun test(): [UInt8] {
              return "01CADE".decodeHex()
          }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeUInt8,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredUInt8Value(1),
				interpreter.NewUnmeteredUInt8Value(0xCA),
				interpreter.NewUnmeteredUInt8Value(0xDE),
			),
			result,
		)

	})

	t.Run("invalid: invalid byte", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
          fun test(): [UInt8] {
              return "0x".decodeHex()
          }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var typedErr interpreter.InvalidHexByteError
		require.ErrorAs(t, err, &typedErr)
		require.Equal(t, byte('x'), typedErr.Byte)
	})

	t.Run("invalid: invalid length", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
          fun test(): [UInt8] {
              return "0".decodeHex()
          }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var typedErr interpreter.InvalidHexLengthError
		require.ErrorAs(t, err, &typedErr)
	})
}

func TestInterpretStringEncodeHex(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
      fun test(): String {
          return String.encodeHex([1, 2, 3, 0xCA, 0xDE])
      }
    `)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("010203cade"),
		result,
	)
}

func TestInterpretStringFromUtf8(t *testing.T) {
	t.Parallel()

	type Testcase struct {
		expr     string
		expected any
	}

	testCases := [...]Testcase{
		// String.fromUTF(str.utf8) = str
		{`"omae wa mou shindeiru".utf8`, "omae wa mou shindeiru"},
		{`"would you still use cadence if i was a worm 🥺😳 👉👈".utf8`, "would you still use cadence if i was a worm 🥺😳 👉👈"},
		// ¥: yen symbol
		{"[0xC2, 0xA5]", "¥"},
		// cyrillic multiocular O
		{"[0xEA, 0x99, 0xAE]", "ꙮ"},
		// chinese biangbiang noodles, doesn't render in 99% of fonts
		{"[0xF0, 0xB0, 0xBB, 0x9E]", "𰻞"},
		{"[0xF0, 0x9F, 0x98, 0x94]", "😔"},
		{"[]", ""},
		// invalid codepoint
		{"[0xc3, 0x28]", nil},
	}

	for _, testCase := range testCases {

		code := fmt.Sprintf(`
			fun testString(): String? {
				return String.fromUTF8(%s)
			}
		`, testCase.expr)

		inter := parseCheckAndPrepare(t, code)

		var expected interpreter.Value
		strValue, ok := testCase.expected.(string)
		// assume that a nil expected means that conversion should fail
		if ok {
			expected = interpreter.NewSomeValueNonCopying(inter,
				interpreter.NewUnmeteredStringValue(strValue))
		} else {
			expected = interpreter.Nil
		}

		result, err := inter.Invoke("testString")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			expected,
			result,
		)
	}
}

func TestInterpretStringFromCharacters(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
      fun test(): String {
          return String.fromCharacters(["👪", "❤️"])
      }
	`)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("👪❤️"),
		result,
	)
}

func TestInterpretStringUtf8Field(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
      fun test(): [UInt8] {
          return "Flowers \u{1F490} are beautiful".utf8
      }
    `)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeUInt8,
			},
			common.ZeroAddress,
			// Flowers
			interpreter.NewUnmeteredUInt8Value(70),
			interpreter.NewUnmeteredUInt8Value(108),
			interpreter.NewUnmeteredUInt8Value(111),
			interpreter.NewUnmeteredUInt8Value(119),
			interpreter.NewUnmeteredUInt8Value(101),
			interpreter.NewUnmeteredUInt8Value(114),
			interpreter.NewUnmeteredUInt8Value(115),
			interpreter.NewUnmeteredUInt8Value(32),
			// Bouquet
			interpreter.NewUnmeteredUInt8Value(240),
			interpreter.NewUnmeteredUInt8Value(159),
			interpreter.NewUnmeteredUInt8Value(146),
			interpreter.NewUnmeteredUInt8Value(144),
			interpreter.NewUnmeteredUInt8Value(32),
			// are
			interpreter.NewUnmeteredUInt8Value(97),
			interpreter.NewUnmeteredUInt8Value(114),
			interpreter.NewUnmeteredUInt8Value(101),
			interpreter.NewUnmeteredUInt8Value(32),
			// beautiful
			interpreter.NewUnmeteredUInt8Value(98),
			interpreter.NewUnmeteredUInt8Value(101),
			interpreter.NewUnmeteredUInt8Value(97),
			interpreter.NewUnmeteredUInt8Value(117),
			interpreter.NewUnmeteredUInt8Value(116),
			interpreter.NewUnmeteredUInt8Value(105),
			interpreter.NewUnmeteredUInt8Value(102),
			interpreter.NewUnmeteredUInt8Value(117),
			interpreter.NewUnmeteredUInt8Value(108),
		),
		result,
	)
}

func TestInterpretStringToLower(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
      fun test(): String {
          return "Flowers".toLower()
      }
    `)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	require.Equal(t,
		interpreter.NewUnmeteredStringValue("flowers"),
		result,
	)
}

func TestInterpretStringAccess(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
    fun test(): Type {
        let c: Character = "x"[0]
        return c.getType() 
    }
    `)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	require.Equal(t,
		interpreter.TypeValue{Type: interpreter.PrimitiveStaticTypeCharacter},
		result,
	)
}

func TestInterpretCharacterLiteralType(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
    fun test(): Type {
        let c: Character = "x"
        return c.getType() 
    }
    `)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	require.Equal(t,
		interpreter.TypeValue{Type: interpreter.PrimitiveStaticTypeCharacter},
		result,
	)
}

func TestInterpretOneCharacterStringLiteralType(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
    fun test(): Type {
        let c: String = "x"
        return c.getType() 
    }
    `)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	require.Equal(t,
		interpreter.TypeValue{Type: interpreter.PrimitiveStaticTypeString},
		result,
	)
}

func TestInterpretCharacterLiteralTypeNoAnnotation(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
    fun test(): Type {
        let c = "x"
        return c.getType() 
    }
    `)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	require.Equal(t,
		interpreter.TypeValue{Type: interpreter.PrimitiveStaticTypeString},
		result,
	)
}

func TestInterpretConvertCharacterToString(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
    fun test(): String {
        let c: Character = "x"
        return c.toString()
    }
    `)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	require.Equal(t,
		interpreter.NewUnmeteredStringValue("x"),
		result,
	)
}

func TestInterpretCompareCharacters(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
        let a: Character = "ü"
        let b: Character = "\u{FC}"
        let c: Character = "\u{75}\u{308}"
        let d: Character = "y"
        let x = a == b
        let y = a == c
        let z = a == d
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.GetGlobal("x"),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.GetGlobal("y"),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		inter.GetGlobal("z"),
	)
}

func TestInterpretStringJoin(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
		fun test(): String {
			return String.join(["👪", "❤️"], separator: "//")
		}

		fun testEmptyArray(): String {
			return String.join([], separator: "//")
		}

		fun testSingletonArray(): String {
			return String.join(["pqrS"], separator: "//")
		}
	`)

	testCase := func(t *testing.T, funcName string, expected *interpreter.StringValue) {
		t.Run(funcName, func(t *testing.T) {
			result, err := inter.Invoke(funcName)
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				expected,
				result,
			)
		})
	}

	testCase(t, "test", interpreter.NewUnmeteredStringValue("👪//❤️"))
	testCase(t, "testEmptyArray", interpreter.NewUnmeteredStringValue(""))
	testCase(t, "testSingletonArray", interpreter.NewUnmeteredStringValue("pqrS"))
}

func TestInterpretStringSplit(t *testing.T) {

	t.Parallel()

	type test struct {
		str    string
		sep    string
		result []string
	}

	var abcd = "abcd"
	var faces = "☺☻☹"
	var commas = "1,2,3,4"
	var dots = "1....2....3....4"

	tests := []test{
		{"", "", []string{}},
		{abcd, "", []string{"a", "b", "c", "d"}},
		{faces, "", []string{"☺", "☻", "☹"}},
		{"☺�☹", "", []string{"☺", "�", "☹"}},
		{abcd, "a", []string{"", "bcd"}},
		{abcd, "z", []string{"abcd"}},
		{commas, ",", []string{"1", "2", "3", "4"}},
		{dots, "...", []string{"1", ".2", ".3", ".4"}},
		{faces, "☹", []string{"☺☻", ""}},
		{faces, "~", []string{faces}},
		{
			"\\u{1F46A}////\\u{2764}\\u{FE0F}",
			"////",
			[]string{"\U0001F46A", "\u2764\uFE0F"},
		},
		{
			"\\u{1F46A} \\u{2764}\\u{FE0F} Abc6 ;123",
			" ",
			[]string{"\U0001F46A", "\u2764\uFE0F", "Abc6", ";123"},
		},
		{
			"Caf\\u{65}\\u{301}ABc",
			"\\u{e9}",
			[]string{"Caf", "ABc"},
		},
		{
			"",
			"//",
			[]string{""},
		},
		{
			"pqrS;asdf",
			";;",
			[]string{"pqrS;asdf"},
		},
		{
			// U+1F476 U+1F3FB is 👶🏻
			" \\u{1F476}\\u{1F3FB} ascii \\u{D}\\u{A}",
			" ",
			[]string{"", "\U0001F476\U0001F3FB", "ascii", "\u000D\u000A"},
		},
		// 🇪🇸🇸🇪🇪🇪 is "ES", "SE", "EE"
		{
			"\\u{1F1EA}\\u{1F1F8}\\u{1F1F8}\\u{1F1EA}\\u{1F1EA}\\u{1F1EA}",
			"\\u{1F1F8}\\u{1F1EA}",
			[]string{"\U0001F1EA\U0001F1F8", "\U0001F1EA\U0001F1EA"},
		},
	}

	runTest := func(test test) {

		name := fmt.Sprintf("%s, %s", test.str, test.sep)

		t.Run(name, func(t *testing.T) {

			t.Parallel()

			inter := parseCheckAndPrepare(t,
				fmt.Sprintf(
					`
                      fun test(): [String] {
                        let s = "%s"
                        return s.split(separator: "%s")
                      }
                    `,
					test.str,
					test.sep,
				),
			)

			value, err := inter.Invoke("test")
			require.NoError(t, err)

			require.IsType(t, &interpreter.ArrayValue{}, value)
			actual := value.(*interpreter.ArrayValue)

			require.Equal(t, len(test.result), actual.Count())

			for partIndex, expected := range test.result {
				actualPart := actual.Get(
					inter,
					interpreter.EmptyLocationRange,
					partIndex,
				)

				require.IsType(t, &interpreter.StringValue{}, actualPart)
				actualPartString := actualPart.(*interpreter.StringValue)

				require.Equal(t, expected, actualPartString.Str)
			}
		})
	}

	for _, test := range tests {
		runTest(test)
	}
}

func TestInterpretStringReplaceAll(t *testing.T) {

	t.Parallel()

	type test struct {
		str    string
		old    string
		new    string
		result string
	}

	tests := []test{
		{"hello", "l", "L", "heLLo"},
		{"hello", "x", "X", "hello"},
		{"", "x", "X", ""},
		{"radar", "r", "<r>", "<r>ada<r>"},
		{"", "", "<>", "<>"},
		{"banana", "a", "<>", "b<>n<>n<>"},
		{"banana", "an", "<>", "b<><>a"},
		{"banana", "ana", "<>", "b<>na"},
		{"banana", "", "<>", "<>b<>a<>n<>a<>n<>a<>"},
		{"banana", "a", "a", "banana"},
		{"☺☻☹", "", "<>", "<>☺<>☻<>☹<>"},

		{"\\u{1F46A}////\\u{2764}\\u{FE0F}", "////", "||", "\U0001F46A||\u2764\uFE0F"},
		{"👪 ❤️ Abc6 ;123", " ", "  ", "👪  ❤️  Abc6  ;123"},
		{"Caf\\u{65}\\u{301}ABc", "\\u{e9}", "X", "CafXABc"},
		{"", "//", "abc", ""},
		{"abc", "", "1", "1a1b1c1"},
		{"pqrS;asdf", ";;", "does_not_matter", "pqrS;asdf"},

		{
			// 🇪🇸🇪🇪 ("ES", "EE") does NOT contain 🇸🇪 ("SE")
			"\\u{1F1EA}\\u{1F1F8}\\u{1F1EA}\\u{1F1EA}",
			"\\u{1F1F8}\\u{1F1EA}",
			"XX",
			"\U0001F1EA\U0001F1F8\U0001F1EA\U0001F1EA",
		},
		{
			// 🇪🇸🇪🇪🇪🇸 ("ES", "EE", "ES")
			"\\u{1F1EA}\\u{1F1F8}\\u{1F1EA}\\u{1F1EA}\\u{1F1EA}\\u{1F1F8}",
			"\\u{1F1EA}\\u{1F1EA}",
			"XX",
			"\U0001F1EA\U0001F1F8XX\U0001F1EA\U0001F1F8",
		},
		{
			// 🇪🇸🇪🇪🇪🇸 ("ES", "EE", "ES")
			"\\u{1F1EA}\\u{1F1F8}\\u{1F1EA}\\u{1F1EA}\\u{1F1EA}\\u{1F1F8}",
			"\\u{1F1EA}\\u{1F1F8}",
			"XX",
			"XX\U0001F1EA\U0001F1EAXX",
		},
		{
			// 🇪🇸🇪🇪🇪🇸 ("ES", "EE", "ES")
			"\\u{1F1EA}\\u{1F1F8}\\u{1F1EA}\\u{1F1EA}\\u{1F1EA}\\u{1F1F8}",
			"",
			"<>",
			"<>\U0001F1EA\U0001F1F8<>\U0001F1EA\U0001F1EA<>\U0001F1EA\U0001F1F8<>",
		},
	}

	runTest := func(test test) {

		name := fmt.Sprintf("%s, %s, %s", test.str, test.old, test.new)

		t.Run(name, func(t *testing.T) {

			t.Parallel()

			inter := parseCheckAndPrepare(t,
				fmt.Sprintf(
					`
                      fun test(): String {
                        let s = "%s"
                        return s.replaceAll(of: "%s", with: "%s")
                      }
                    `,
					test.str,
					test.old,
					test.new,
				),
			)

			value, err := inter.Invoke("test")
			require.NoError(t, err)

			require.IsType(t, &interpreter.StringValue{}, value)
			actual := value.(*interpreter.StringValue)

			require.Equal(t, test.result, actual.Str)
		})
	}

	for _, test := range tests {
		runTest(test)
	}
}

func TestInterpretStringContains(t *testing.T) {

	t.Parallel()

	type test struct {
		str    string
		subStr string
		result bool
	}

	tests := []test{
		{"abcdef", "", true},
		{"abcdef", "a", true},
		{"abcdef", "ab", true},
		{"abcdef", "ac", false},
		{"abcdef", "b", true},
		{"abcdef", "bc", true},
		{"abcdef", "bcd", true},
		{"abcdef", "c", true},
		{"abcdef", "cd", true},
		{"abcdef", "cdef", true},
		{"abcdef", "cdefg", false},
		{"abcdef", "abcdef", true},
		{"abcdef", "abcdefg", false},

		// U+1F476 U+1F3FB is 👶🏻
		{" \\u{1F476}\\u{1F3FB} ascii \\u{D}\\u{A}", " \\u{1F476}", false},
		{" \\u{1F476}\\u{1F3FB} ascii \\u{D}\\u{A}", "\\u{1F3FB}", false},
		{" \\u{1F476}\\u{1F3FB} ascii \\u{D}\\u{A}", " \\u{1F476}\\u{1F3FB}", true},
		{" \\u{1F476}\\u{1F3FB} ascii \\u{D}\\u{A}", "\\u{1F476}\\u{1F3FB}", true},
		{" \\u{1F476}\\u{1F3FB} ascii \\u{D}\\u{A}", "\\u{1F476}\\u{1F3FB} ", true},
		{" \\u{1F476}\\u{1F3FB} ascii \\u{D}\\u{A}", "\\u{D}", false},
		{" \\u{1F476}\\u{1F3FB} ascii \\u{D}\\u{A}", "\\u{A}", false},
		{" \\u{1F476}\\u{1F3FB} ascii \\u{D}\\u{A}", " ascii ", true},

		// 🇪🇸🇪🇪 ("ES", "EE") contains 🇪🇸("ES")
		{"\\u{1F1EA}\\u{1F1F8}\\u{1F1EA}\\u{1F1EA}", "\\u{1F1EA}\\u{1F1F8}", true},
		// 🇪🇸🇪🇪 ("ES", "EE") contains 🇪🇪 ("EE")
		{"\\u{1F1EA}\\u{1F1F8}\\u{1F1EA}\\u{1F1EA}", "\\u{1F1EA}\\u{1F1EA}", true},
		// 🇪🇸🇪🇪 ("ES", "EE") does NOT contain 🇸🇪 ("SE")
		{"\\u{1F1EA}\\u{1F1F8}\\u{1F1EA}\\u{1F1EA}", "\\u{1F1F8}\\u{1F1EA}", false},
		// neither prefix nor suffix of codepoints are valid
		{"\\u{1F1EA}\\u{1F1F8}\\u{1F1EA}\\u{1F1EA}", "\\u{1F1EA}\\u{1F1F8}\\u{1F1EA}", false},
		{"\\u{1F1EA}\\u{1F1F8}\\u{1F1EA}\\u{1F1EA}", "\\u{1F1F8}\\u{1F1EA}\\u{1F1EA}", false},
	}

	runTest := func(test test) {

		name := fmt.Sprintf("%s, %s", test.str, test.subStr)

		t.Run(name, func(t *testing.T) {

			t.Parallel()

			inter := parseCheckAndPrepare(t,
				fmt.Sprintf(
					`
                      fun test(): Bool {
                        let s = "%s"
                        return s.contains("%s")
                      }
                    `,
					test.str,
					test.subStr,
				),
			)

			value, err := inter.Invoke("test")
			require.NoError(t, err)

			require.IsType(t, interpreter.BoolValue(true), value)
			actual := value.(interpreter.BoolValue)
			require.Equal(t, test.result, bool(actual))
		})
	}

	for _, test := range tests {
		runTest(test)
	}
}

func TestInterpretStringIndex(t *testing.T) {

	t.Parallel()

	type test struct {
		str    string
		subStr string
		result int
	}

	tests := []test{
		{"abcdef", "", 0},
		{"abcdef", "a", 0},
		{"abcdef", "ab", 0},
		{"abcdef", "ac", -1},
		{"abcdef", "b", 1},
		{"abcdef", "bc", 1},
		{"abcdef", "bcd", 1},
		{"abcdef", "c", 2},
		{"abcdef", "cd", 2},
		{"abcdef", "cdef", 2},
		{"abcdef", "cdefg", -1},
		{"abcdef", "abcdef", 0},
		{"abcdef", "abcdefg", -1},

		// U+1F476 U+1F3FB is 👶🏻
		{" \\u{1F476}\\u{1F3FB} ascii \\u{D}\\u{A}", " \\u{1F476}", -1},
		{" \\u{1F476}\\u{1F3FB} ascii \\u{D}\\u{A}", "\\u{1F3FB}", -1},
		{" \\u{1F476}\\u{1F3FB} ascii \\u{D}\\u{A}", " \\u{1F476}\\u{1F3FB}", 0},
		{" \\u{1F476}\\u{1F3FB} ascii \\u{D}\\u{A}", "\\u{1F476}\\u{1F3FB}", 1},
		{" \\u{1F476}\\u{1F3FB} ascii \\u{D}\\u{A}", "\\u{1F476}\\u{1F3FB} ", 1},
		{" \\u{1F476}\\u{1F3FB} ascii \\u{D}\\u{A}", "\\u{D}", -1},
		{" \\u{1F476}\\u{1F3FB} ascii \\u{D}\\u{A}", "\\u{A}", -1},
		{" \\u{1F476}\\u{1F3FB} ascii \\u{D}\\u{A}", " ascii ", 2},

		// 🇪🇸🇪🇪 ("ES", "EE") contains 🇪🇸("ES")
		{"\\u{1F1EA}\\u{1F1F8}\\u{1F1EA}\\u{1F1EA}", "\\u{1F1EA}\\u{1F1F8}", 0},
		// 🇪🇸🇪🇪 ("ES", "EE") contains 🇪🇪 ("EE")
		{"\\u{1F1EA}\\u{1F1F8}\\u{1F1EA}\\u{1F1EA}", "\\u{1F1EA}\\u{1F1EA}", 1},
		// 🇪🇸🇪🇪 ("ES", "EE") does NOT contain 🇸🇪 ("SE")
		{"\\u{1F1EA}\\u{1F1F8}\\u{1F1EA}\\u{1F1EA}", "\\u{1F1F8}\\u{1F1EA}", -1},
		// neither prefix nor suffix of codepoints are valid
		{"\\u{1F1EA}\\u{1F1F8}\\u{1F1EA}\\u{1F1EA}", "\\u{1F1EA}\\u{1F1F8}\\u{1F1EA}", -1},
		{"\\u{1F1EA}\\u{1F1F8}\\u{1F1EA}\\u{1F1EA}", "\\u{1F1F8}\\u{1F1EA}\\u{1F1EA}", -1},
	}

	runTest := func(test test) {

		name := fmt.Sprintf("%s, %s", test.str, test.subStr)

		t.Run(name, func(t *testing.T) {

			t.Parallel()

			inter := parseCheckAndPrepare(t,
				fmt.Sprintf(
					`
                      fun test(): Int {
                        let s = "%s"
                        return s.index(of: "%s")
                      }
                    `,
					test.str,
					test.subStr,
				),
			)

			value, err := inter.Invoke("test")
			require.NoError(t, err)

			require.IsType(t, interpreter.IntValue{}, value)
			actual := value.(interpreter.IntValue)
			require.Equal(t, test.result, actual.ToInt(interpreter.EmptyLocationRange))
		})
	}

	for _, test := range tests {
		runTest(test)
	}
}

func TestInterpretStringCount(t *testing.T) {

	t.Parallel()

	type test struct {
		str    string
		subStr string
		result int
	}

	tests := []test{
		{"", "", 1},
		{"abcdef", "", 7},

		{"", "notempty", 0},
		{"notempty", "", 9},
		{"smaller", "not smaller", 0},
		{"12345678987654321", "6", 2},
		{"611161116", "6", 3},
		{"notequal", "NotEqual", 0},
		{"equal", "equal", 1},
		{"abc1231231123q", "123", 3},
		{"11111", "11", 2},

		// 🇪🇸🇪🇪🇪🇸 ("ES", "EE", "ES") contains 🇪🇸("ES") twice
		{"\\u{1F1EA}\\u{1F1F8}\\u{1F1EA}\\u{1F1EA}\\u{1F1EA}\\u{1F1F8}", "\\u{1F1EA}\\u{1F1F8}", 2},
	}

	runTest := func(test test) {

		name := fmt.Sprintf("%s, %s", test.str, test.subStr)

		t.Run(name, func(t *testing.T) {

			t.Parallel()

			inter := parseCheckAndPrepare(t,
				fmt.Sprintf(
					`
                      fun test(): Int {
                        let s = "%s"
                        return s.count("%s")
                      }
                    `,
					test.str,
					test.subStr,
				),
			)

			value, err := inter.Invoke("test")
			require.NoError(t, err)

			require.IsType(t, interpreter.IntValue{}, value)
			actual := value.(interpreter.IntValue)
			require.Equal(t, test.result, actual.ToInt(interpreter.EmptyLocationRange))
		})
	}

	for _, test := range tests {
		runTest(test)
	}
}
