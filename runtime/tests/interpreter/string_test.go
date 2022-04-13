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
			GetKey(inter, interpreter.ReturnEmptyLocationRange, interpreter.NewStringValue("mapRef")).
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
		interpreter.NewStringValue(""),
		result,
	)
}

func TestInterpretStringDecodeHex(t *testing.T) {

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
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeUInt8,
			},
			common.Address{},
			interpreter.UInt8Value(1),
			interpreter.UInt8Value(0xCA),
			interpreter.UInt8Value(0xDE),
		),
		result,
	)
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
		interpreter.NewStringValue("010203cade"),
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
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeUInt8,
			},
			common.Address{},
			// Flowers
			interpreter.UInt8Value(70),
			interpreter.UInt8Value(108),
			interpreter.UInt8Value(111),
			interpreter.UInt8Value(119),
			interpreter.UInt8Value(101),
			interpreter.UInt8Value(114),
			interpreter.UInt8Value(115),
			interpreter.UInt8Value(32),
			// Bouquet
			interpreter.UInt8Value(240),
			interpreter.UInt8Value(159),
			interpreter.UInt8Value(146),
			interpreter.UInt8Value(144),
			interpreter.UInt8Value(32),
			// are
			interpreter.UInt8Value(97),
			interpreter.UInt8Value(114),
			interpreter.UInt8Value(101),
			interpreter.UInt8Value(32),
			// beautiful
			interpreter.UInt8Value(98),
			interpreter.UInt8Value(101),
			interpreter.UInt8Value(97),
			interpreter.UInt8Value(117),
			interpreter.UInt8Value(116),
			interpreter.UInt8Value(105),
			interpreter.UInt8Value(102),
			interpreter.UInt8Value(117),
			interpreter.UInt8Value(108),
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
		interpreter.NewStringValue("flowers"),
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
		interpreter.NewStringValue("x"),
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
		interpreter.BoolValue(true),
		inter.Globals["x"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		inter.Globals["y"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(false),
		inter.Globals["z"].GetValue(),
	)
}
