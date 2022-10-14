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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/onflow/cadence/runtime/tests/utils"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
)

func TestInterpretEnum(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      enum E: Int64 {
          case a
          case b
      }
    `)

	assert.IsType(t,
		&interpreter.HostFunctionValue{},
		inter.Globals.Get("E").GetValue(),
	)
}

func TestInterpretEnumCaseUse(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      enum E: Int64 {
          case a
          case b
      }

      let a = E.a
      let b = E.b
    `)

	a := inter.Globals.Get("a").GetValue()
	require.IsType(t,
		&interpreter.CompositeValue{},
		a,
	)

	assert.Equal(t,
		common.CompositeKindEnum,
		a.(*interpreter.CompositeValue).Kind,
	)

	b := inter.Globals.Get("b").GetValue()
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

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      enum E: Int64 {
          case a
          case b
      }

      let a = E.a.rawValue
      let b = E.b.rawValue
    `)

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredInt64Value(0),
		inter.Globals.Get("a").GetValue(),
	)

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredInt64Value(1),
		inter.Globals.Get("b").GetValue(),
	)
}

func TestInterpretEnumCaseEquality(t *testing.T) {

	t.Parallel()

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

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeBool,
			},
			common.Address{},
			interpreter.BoolValue(true),
			interpreter.BoolValue(true),
			interpreter.BoolValue(true),
		),
		inter.Globals.Get("res").GetValue(),
	)
}

func TestInterpretEnumConstructor(t *testing.T) {

	t.Parallel()

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

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeBool,
			},
			common.Address{},
			interpreter.BoolValue(true),
			interpreter.BoolValue(true),
			interpreter.BoolValue(true),
			interpreter.BoolValue(true),
		),
		inter.Globals.Get("res").GetValue(),
	)
}

func TestInterpretEnumInstance(t *testing.T) {

	t.Parallel()

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

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeBool,
			},
			common.Address{},
			interpreter.BoolValue(true),
			interpreter.BoolValue(true),
		),
		inter.Globals.Get("res").GetValue(),
	)
}

func TestInterpretEnumInContract(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndInterpretWithOptions(t,
		`
          contract C {
              enum E: UInt8 {
                  pub case a
                  pub case b
              }

              var e: E

              init() {
                  self.e = E.a
              }
          }
        `,
		ParseCheckAndInterpretOptions{
			Config: &interpreter.Config{
				ContractValueHandler: makeContractValueHandler(nil, nil, nil),
			},
		},
	)
	require.NoError(t, err)

	c := inter.Globals.Get("C").GetValue()
	require.IsType(t, &interpreter.CompositeValue{}, c)
	contract := c.(*interpreter.CompositeValue)

	eValue := contract.GetField(inter, interpreter.EmptyLocationRange, "e")
	require.NotNil(t, eValue)

	require.IsType(t, &interpreter.CompositeValue{}, eValue)
	enumCase := eValue.(*interpreter.CompositeValue)

	rawValue := enumCase.GetMember(
		inter,
		interpreter.EmptyLocationRange,
		"rawValue",
	)

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredUInt8Value(0),
		rawValue,
	)
}
