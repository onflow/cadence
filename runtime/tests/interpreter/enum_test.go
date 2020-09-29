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

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
)

func TestInterpretEnum(t *testing.T) {

	inter := parseCheckAndInterpret(t, `
      enum E: Int64 {
          case a
          case b
      }
    `)

	assert.IsType(t,
		interpreter.HostFunctionValue{},
		inter.Globals["E"].Value,
	)
}

func TestInterpretEnumCaseUse(t *testing.T) {

	inter := parseCheckAndInterpret(t, `
      enum E: Int64 {
          case a
          case b
      }

      let a = E.a
      let b = E.b
    `)

	a := inter.Globals["a"].Value
	require.IsType(t,
		&interpreter.CompositeValue{},
		a,
	)

	assert.Equal(t,
		common.CompositeKindEnum,
		a.(*interpreter.CompositeValue).Kind,
	)

	b := inter.Globals["b"].Value
	require.IsType(t,
		&interpreter.CompositeValue{},
		b,
	)

	assert.Equal(t,
		common.CompositeKindEnum,
		b.(*interpreter.CompositeValue).Kind,
	)
}

func TestInterpretEnumCaseRawValue(t *testing.T) {

	inter := parseCheckAndInterpret(t, `
      enum E: Int64 {
          case a
          case b
      }

      let a = E.a.rawValue
      let b = E.b.rawValue
    `)

	require.Equal(t,
		interpreter.Int64Value(0),
		inter.Globals["a"].Value,
	)

	require.Equal(t,
		interpreter.Int64Value(1),
		inter.Globals["b"].Value,
	)
}

func TestInterpretEnumCaseEquality(t *testing.T) {

	inter := parseCheckAndInterpret(t, `
      enum E: Int64 {
          case a
          case b
      }

      let res = [
          E.a == E.a,
          E.b == E.b,
          E.a != E.b
      ]
    `)

	require.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.BoolValue(true),
			interpreter.BoolValue(true),
			interpreter.BoolValue(true),
		),
		inter.Globals["res"].Value,
	)
}

func TestInterpretEnumConstructor(t *testing.T) {

	inter := parseCheckAndInterpret(t, `
      enum E: Int64 {
          case a
          case b
      }

      let res = [
          E(rawValue: 0)! == E.a,
          E(rawValue: 1)! == E.b,
          E(rawValue: -1) == nil,
          E(rawValue: 2) == nil
      ]
    `)

	require.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.BoolValue(true),
			interpreter.BoolValue(true),
			interpreter.BoolValue(true),
			interpreter.BoolValue(true),
		),
		inter.Globals["res"].Value,
	)
}

func TestInterpretEnumInstance(t *testing.T) {

	inter := parseCheckAndInterpret(t, `
      enum E: Int64 {
          case a
          case b
      }

      let res = [
         E.a.isInstance(Type<E>()),
         E.b.isInstance(Type<E>())
      ]
    `)

	require.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.BoolValue(true),
			interpreter.BoolValue(true),
		),
		inter.Globals["res"].Value,
	)
}
