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
	"encoding/binary"
	"testing"

	"github.com/onflow/atree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
)

func TestInterpretEnum(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
      enum E: Int64 {
          case a
          case b
      }
    `)

	var expectedType interpreter.Value
	if *compile {
		expectedType = vm.CompiledFunctionValue{}
	} else {
		expectedType = &interpreter.HostFunctionValue{}
	}

	assert.IsType(t,
		expectedType,
		inter.GetGlobal("E"),
	)
}

func TestInterpretEnumCaseUse(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
      enum E: Int64 {
          case a
          case b
      }

      let a = E.a
      let b = E.b
    `)

	a := inter.GetGlobal("a")
	require.IsType(t,
		&interpreter.CompositeValue{},
		a,
	)

	assert.Equal(t,
		common.CompositeKindEnum,
		a.(*interpreter.CompositeValue).Kind,
	)

	b := inter.GetGlobal("b")
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

	inter := parseCheckAndPrepare(t, `
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
		inter.GetGlobal("a"),
	)

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredInt64Value(1),
		inter.GetGlobal("b"),
	)
}

func TestInterpretEnumCaseEquality(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
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
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeBool,
			},
			common.ZeroAddress,
			interpreter.TrueValue,
			interpreter.TrueValue,
			interpreter.TrueValue,
		),
		inter.GetGlobal("res"),
	)
}

func TestInterpretEnumConstructor(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
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
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeBool,
			},
			common.ZeroAddress,
			interpreter.TrueValue,
			interpreter.TrueValue,
			interpreter.TrueValue,
			interpreter.TrueValue,
		),
		inter.GetGlobal("res"),
	)
}

func TestInterpretEnumInstance(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
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
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeBool,
			},
			common.ZeroAddress,
			interpreter.TrueValue,
			interpreter.TrueValue,
		),
		inter.GetGlobal("res"),
	)
}

func TestInterpretEnumInContract(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndPrepareWithOptions(t,
		`
          contract C {
              enum E: UInt8 {
                  access(all) case a
                  access(all) case b
              }

              var e: E

              init() {
                  self.e = E.a
              }
          }
        `,
		ParseCheckAndInterpretOptions{
			InterpreterConfig: &interpreter.Config{
				ContractValueHandler: makeContractValueHandler(nil, nil, nil),
			},
		},
	)
	require.NoError(t, err)

	c := inter.GetGlobal("C")
	contract, ok := c.(interpreter.MemberAccessibleValue)
	require.True(t, ok)

	eValue := contract.GetMember(inter, "e")
	require.NotNil(t, eValue)

	require.IsType(t, &interpreter.CompositeValue{}, eValue)
	enumCase := eValue.(*interpreter.CompositeValue)

	rawValue := enumCase.GetMember(inter, "rawValue")

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredUInt8Value(0),
		rawValue,
	)
}

func TestInterpretEnumLookup(t *testing.T) {
	t.Parallel()

	t.SkipNow()

	storage := NewUnmeteredInMemoryStorage()

	inter, err := parseCheckAndPrepareWithOptions(t,
		`
          enum E: UInt8 {
              case A
          }

          fun test(): E {
              return E(rawValue: 0)!
          }
        `,
		ParseCheckAndInterpretOptions{
			InterpreterConfig: &interpreter.Config{
				Storage: storage,
			},
		},
	)
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.NoError(t, err)

	slabID, err := storage.BasicSlabStorage.GenerateSlabID(atree.AddressUndefined)
	require.NoError(t, err)

	var expectedSlabIndex atree.SlabIndex
	binary.BigEndian.PutUint64(expectedSlabIndex[:], 4)

	require.Equal(
		t,
		atree.NewSlabID(
			atree.AddressUndefined,
			expectedSlabIndex,
		),
		slabID,
	)
}
