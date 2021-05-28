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

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/stretchr/testify/require"
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
		`{"mapRef": {"mapRef": {"mapRef": ...}}}`,
		mapValue.String(interpreter.StringResults{}),
	)

	require.IsType(t, &interpreter.DictionaryValue{}, mapValue)
	require.Equal(t,
		`{"mapRef": {"mapRef": ...}}`,
		mapValue.(*interpreter.DictionaryValue).
			Get(inter, nil, interpreter.NewStringValue("mapRef")).
			String(interpreter.StringResults{}),
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

	require.Equal(t,
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

	require.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
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
          return String.encodeHex([1 as UInt8, 2, 3, 0xCA, 0xDE])
      }
	`)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	require.Equal(t,
		interpreter.NewStringValue("010203cade"),
		result,
	)
}
