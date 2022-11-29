/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpretRecursiveValueString(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): AnyStruct {
          let map: {String: AnyStruct} = {}
          let mapRef = &map as &{String: AnyStruct}
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

	inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, `
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
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeUInt8,
				},
				common.Address{},
				interpreter.NewUnmeteredUInt8Value(1),
				interpreter.NewUnmeteredUInt8Value(0xCA),
				interpreter.NewUnmeteredUInt8Value(0xDE),
			),
			result,
		)

	})

	t.Run("invalid: invalid byte", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, `
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

	inter := parseCheckAndInterpret(t, `
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

		inter := parseCheckAndInterpret(t, code)

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

	inter := parseCheckAndInterpret(t, `
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

	inter := parseCheckAndInterpret(t, `
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
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeUInt8,
			},
			common.Address{},
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

	inter := parseCheckAndInterpret(t, `
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

	inter := parseCheckAndInterpret(t, `
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

	inter := parseCheckAndInterpret(t, `
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

	inter := parseCheckAndInterpret(t, `
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

	inter := parseCheckAndInterpret(t, `
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

	inter := parseCheckAndInterpret(t, `
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

	inter := parseCheckAndInterpret(t, `
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
		inter.Globals.Get("x").GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("y").GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		inter.Globals.Get("z").GetValue(),
	)
}
