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

package runtime

import (
	"fmt"
	"testing"

	"github.com/onflow/cadence/encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

type exportTest struct {
	label       string
	value       interpreter.Value
	expected    cadence.Value
	skipReverse bool
}

var exportTests = []exportTest{
	{
		label:    "Void",
		value:    interpreter.VoidValue{},
		expected: cadence.NewVoid(),
	},
	{
		label:       "Nil",
		value:       interpreter.NilValue{},
		expected:    cadence.NewOptional(nil),
		skipReverse: true,
	},
	{
		label:    "SomeValue",
		value:    interpreter.NewSomeValueOwningNonCopying(interpreter.NewIntValueFromInt64(42)),
		expected: cadence.NewOptional(cadence.NewInt(42)),
	},
	{
		label:    "Bool true",
		value:    interpreter.BoolValue(true),
		expected: cadence.NewBool(true),
	},
	{
		label:    "Bool false",
		value:    interpreter.BoolValue(false),
		expected: cadence.NewBool(false),
	},
	{
		label:    "String empty",
		value:    interpreter.NewStringValue(""),
		expected: cadence.NewString(""),
	},
	{
		label:    "String non-empty",
		value:    interpreter.NewStringValue("foo"),
		expected: cadence.NewString("foo"),
	},
	{
		label:    "Array empty",
		value:    interpreter.NewArrayValueUnownedNonCopying([]interpreter.Value{}...),
		expected: cadence.NewArray([]cadence.Value{}),
	},
	{
		label: "Array non-empty",
		value: interpreter.NewArrayValueUnownedNonCopying(
			[]interpreter.Value{
				interpreter.NewIntValueFromInt64(42),
				interpreter.NewStringValue("foo"),
			}...,
		),
		expected: cadence.NewArray([]cadence.Value{
			cadence.NewInt(42),
			cadence.NewString("foo"),
		}),
	},
	{
		label:    "Int",
		value:    interpreter.NewIntValueFromInt64(42),
		expected: cadence.NewInt(42),
	},
	{
		label:    "Int8",
		value:    interpreter.Int8Value(42),
		expected: cadence.NewInt8(42),
	},
	{
		label:    "Int16",
		value:    interpreter.Int16Value(42),
		expected: cadence.NewInt16(42),
	},
	{
		label:    "Int32",
		value:    interpreter.Int32Value(42),
		expected: cadence.NewInt32(42),
	},
	{
		label:    "Int64",
		value:    interpreter.Int64Value(42),
		expected: cadence.NewInt64(42),
	},
	{
		label:    "Int128",
		value:    interpreter.NewInt128ValueFromInt64(42),
		expected: cadence.NewInt128(42),
	},
	{
		label:    "Int256",
		value:    interpreter.NewInt256ValueFromInt64(42),
		expected: cadence.NewInt256(42),
	},
	{
		label:    "UInt",
		value:    interpreter.NewUIntValueFromUint64(42),
		expected: cadence.NewUInt(42),
	},
	{
		label:    "UInt8",
		value:    interpreter.UInt8Value(42),
		expected: cadence.NewUInt8(42),
	},
	{
		label:    "UInt16",
		value:    interpreter.UInt16Value(42),
		expected: cadence.NewUInt16(42),
	},
	{
		label:    "UInt32",
		value:    interpreter.UInt32Value(42),
		expected: cadence.NewUInt32(42),
	},
	{
		label:    "UInt64",
		value:    interpreter.UInt64Value(42),
		expected: cadence.NewUInt64(42),
	},
	{
		label:    "UInt128",
		value:    interpreter.NewUInt128ValueFromUint64(42),
		expected: cadence.NewUInt128(42),
	},
	{
		label:    "UInt256",
		value:    interpreter.NewUInt256ValueFromUint64(42),
		expected: cadence.NewUInt256(42),
	},
	{
		label:    "Word8",
		value:    interpreter.Word8Value(42),
		expected: cadence.NewWord8(42),
	},
	{
		label:    "Word16",
		value:    interpreter.Word16Value(42),
		expected: cadence.NewWord16(42),
	},
	{
		label:    "Word32",
		value:    interpreter.Word32Value(42),
		expected: cadence.NewWord32(42),
	},
	{
		label:    "Word64",
		value:    interpreter.Word64Value(42),
		expected: cadence.NewWord64(42),
	},
	{
		label:    "Fix64",
		value:    interpreter.Fix64Value(-123000000),
		expected: cadence.Fix64(-123000000),
	},
	{
		label:    "UFix64",
		value:    interpreter.UFix64Value(123000000),
		expected: cadence.UFix64(123000000),
	},
	{
		label: "Path",
		value: interpreter.PathValue{
			Domain:     common.PathDomainStorage,
			Identifier: "foo",
		},
		expected: cadence.Path{
			Domain:     "storage",
			Identifier: "foo",
		},
	},
}

func TestExportValue(t *testing.T) {

	t.Parallel()

	test := func(tt exportTest) {

		t.Run(tt.label, func(t *testing.T) {

			t.Parallel()

			actual := exportValueWithInterpreter(tt.value, nil, exportResults{})
			assert.Equal(t, tt.expected, actual)

			if !tt.skipReverse {
				original := importValue(actual)
				assert.Equal(t, tt.value, original)
			}
		})
	}

	for _, tt := range exportTests {
		test(tt)
	}
}

func TestExportIntegerValuesFromScript(t *testing.T) {

	t.Parallel()

	test := func(integerType sema.Type) {

		t.Run(integerType.String(), func(t *testing.T) {

			t.Parallel()

			script := fmt.Sprintf(
				`
                  pub fun main(): %s {
                      return 42
                  }
                `,
				integerType,
			)

			assert.NotPanics(t, func() {
				exportValueFromScript(t, script)
			})
		})
	}

	for _, integerType := range sema.AllIntegerTypes {
		test(integerType)
	}
}

func TestExportFixedPointValuesFromScript(t *testing.T) {

	t.Parallel()

	test := func(fixedPointType sema.Type) {

		t.Run(fixedPointType.String(), func(t *testing.T) {

			t.Parallel()

			script := fmt.Sprintf(
				`
                  pub fun main(): %s {
                      return 1.23
                  }
                `,
				fixedPointType,
			)

			assert.NotPanics(t, func() {
				exportValueFromScript(t, script)
			})
		})
	}

	for _, fixedPointType := range sema.AllFixedPointTypes {
		test(fixedPointType)
	}
}

func TestExportDictionaryValue(t *testing.T) {

	t.Parallel()

	t.Run("Empty", func(t *testing.T) {

		t.Parallel()

		script := `
            access(all) fun main(): {String: Int} {
                return {}
            }
        `

		actual := exportValueFromScript(t, script)
		expected := cadence.NewDictionary([]cadence.KeyValuePair{})

		assert.Equal(t, expected, actual)
	})

	t.Run("Non-empty", func(t *testing.T) {

		t.Parallel()

		script := `
            access(all) fun main(): {String: Int} {
                return {
                    "a": 1,
                    "b": 2
                }
            }
        `

		actual := exportValueFromScript(t, script)
		expected := cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key:   cadence.NewString("a"),
				Value: cadence.NewInt(1),
			},
			{
				Key:   cadence.NewString("b"),
				Value: cadence.NewInt(2),
			},
		})

		assert.Equal(t, expected, actual)
	})
}

func TestExportAddressValue(t *testing.T) {

	t.Parallel()

	script := `
        access(all) fun main(): Address {
            return 0x42
        }
    `

	actual := exportValueFromScript(t, script)
	expected := cadence.BytesToAddress(
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x42},
	)

	assert.Equal(t, expected, actual)
}

func TestExportStructValue(t *testing.T) {

	t.Parallel()

	script := `
        access(all) struct Foo {
            access(all) let bar: Int

            init(bar: Int) {
                self.bar = bar
            }
        }

        access(all) fun main(): Foo {
            return Foo(bar: 42)
        }
    `

	actual := exportValueFromScript(t, script)
	expected := cadence.NewStruct([]cadence.Value{cadence.NewInt(42)}).WithType(fooStructType)

	assert.Equal(t, expected, actual)
}

func TestExportResourceValue(t *testing.T) {

	t.Parallel()

	script := `
        access(all) resource Foo {
            access(all) let bar: Int

            init(bar: Int) {
                self.bar = bar
            }
        }

        access(all) fun main(): @Foo {
            return <- create Foo(bar: 42)
        }
    `

	actual := exportValueFromScript(t, script)
	expected :=
		cadence.NewResource([]cadence.Value{
			cadence.NewUInt64(0),
			cadence.NewInt(42),
		}).WithType(fooResourceType)

	assert.Equal(t, expected, actual)
}

func TestExportResourceArrayValue(t *testing.T) {

	t.Parallel()

	script := `
        access(all) resource Foo {
            access(all) let bar: Int

            init(bar: Int) {
                self.bar = bar
            }
        }

        access(all) fun main(): @[Foo] {
            return <- [<- create Foo(bar: 1), <- create Foo(bar: 2)]
        }
    `

	actual := exportValueFromScript(t, script)
	expected := cadence.NewArray([]cadence.Value{
		cadence.NewResource([]cadence.Value{
			cadence.NewUInt64(0),
			cadence.NewInt(1),
		}).WithType(fooResourceType),
		cadence.NewResource([]cadence.Value{
			cadence.NewUInt64(0),
			cadence.NewInt(2),
		}).WithType(fooResourceType),
	})

	assert.Equal(t, expected, actual)
}

func TestExportResourceDictionaryValue(t *testing.T) {

	t.Parallel()

	script := `
        access(all) resource Foo {
            access(all) let bar: Int

            init(bar: Int) {
                self.bar = bar
            }
        }

        access(all) fun main(): @{String: Foo} {
            return <- {
                "a": <- create Foo(bar: 1),
                "b": <- create Foo(bar: 2)
            }
        }
    `

	actual := exportValueFromScript(t, script)
	expected := cadence.NewDictionary([]cadence.KeyValuePair{
		{
			Key: cadence.NewString("a"),
			Value: cadence.NewResource([]cadence.Value{
				cadence.NewUInt64(0),
				cadence.NewInt(1),
			}).WithType(fooResourceType),
		},
		{
			Key: cadence.NewString("b"),
			Value: cadence.NewResource([]cadence.Value{
				cadence.NewUInt64(0),
				cadence.NewInt(2),
			}).WithType(fooResourceType),
		},
	})

	assert.Equal(t, expected, actual)
}

func TestExportNestedResourceValueFromScript(t *testing.T) {

	t.Parallel()

	barResourceType := &cadence.ResourceType{
		TypeID:     "S.test.Bar",
		Identifier: "Bar",
		Fields: []cadence.Field{
			{
				Identifier: "uuid",
				Type:       cadence.UInt64Type{},
			},
			{
				Identifier: "x",
				Type:       cadence.IntType{},
			},
		},
	}

	fooResourceType := &cadence.ResourceType{
		TypeID:     "S.test.Foo",
		Identifier: "Foo",
		Fields: []cadence.Field{
			{
				Identifier: "uuid",
				Type:       cadence.UInt64Type{},
			},
			{
				Identifier: "bar",
				Type:       barResourceType,
			},
		},
	}

	script := `
        access(all) resource Bar {
            access(all) let x: Int

            init(x: Int) {
                self.x = x
            }
        }

        access(all) resource Foo {
            access(all) let bar: @Bar

            init(bar: @Bar) {
                self.bar <- bar
            }

            destroy() {
                destroy self.bar
            }
        }

        access(all) fun main(): @Foo {
            return <- create Foo(bar: <- create Bar(x: 42))
        }
    `

	actual := exportValueFromScript(t, script)
	expected := cadence.NewResource([]cadence.Value{
		cadence.NewUInt64(0),
		cadence.NewResource([]cadence.Value{
			cadence.NewUInt64(0),
			cadence.NewInt(42),
		}).WithType(barResourceType),
	}).WithType(fooResourceType)

	assert.Equal(t, expected, actual)
}

func TestExportEventValue(t *testing.T) {

	t.Parallel()

	script := `
        access(all) event Foo(bar: Int)

        access(all) fun main() {
            emit Foo(bar: 42)
        }
    `

	actual := exportEventFromScript(t, script)
	expected := cadence.NewEvent([]cadence.Value{cadence.NewInt(42)}).WithType(fooEventType)

	assert.Equal(t, expected, actual)
}

// mock runtime.Interface to capture events
type eventCapturingInterface struct {
	EmptyRuntimeInterface
	events []cadence.Event
}

func (t *eventCapturingInterface) EmitEvent(event cadence.Event) {
	t.events = append(t.events, event)
}

func exportEventFromScript(t *testing.T, script string) cadence.Event {
	rt := NewInterpreterRuntime()

	inter := &eventCapturingInterface{}

	_, err := rt.ExecuteScript(
		[]byte(script),
		nil,
		inter,
		utils.TestLocation,
	)

	require.NoError(t, err)
	require.Len(t, inter.events, 1)

	event := inter.events[0]

	return event
}

func exportValueFromScript(t *testing.T, script string) cadence.Value {
	rt := NewInterpreterRuntime()

	value, err := rt.ExecuteScript(
		[]byte(script),
		nil,
		&EmptyRuntimeInterface{},
		utils.TestLocation,
	)

	require.NoError(t, err)

	return value
}

func TestExportTypeValue(t *testing.T) {

	t.Parallel()

	script := `
        access(all) fun main(): Type {
            return Type<Int>()
        }
    `

	actual := exportValueFromScript(t, script)
	expected := cadence.TypeValue{
		StaticType: "Int",
	}

	assert.Equal(t, expected, actual)
}

func TestExportCapabilityValue(t *testing.T) {

	t.Parallel()

	capability := interpreter.CapabilityValue{
		Address: interpreter.AddressValue{0x1},
		Path: interpreter.PathValue{
			Domain:     common.PathDomainStorage,
			Identifier: "foo",
		},
		BorrowType: interpreter.PrimitiveStaticTypeInt,
	}
	actual := exportValueWithInterpreter(capability, nil, exportResults{})
	expected := cadence.Capability{
		Path: cadence.Path{
			Domain:     "storage",
			Identifier: "foo",
		},
		Address:    cadence.Address{0x1},
		BorrowType: "Int",
	}

	assert.Equal(t, expected, actual)
}

func TestExportJsonDeterministic(t *testing.T) {

	// exported order of field in a dictionary depends on the execution ,
	// however the deterministic code should generate deterministic type

	script := `
        access(all) event Foo(bar: Int, aaa: {Int: {Int: String}})

        access(all) fun main() {

			let dict0 = {
				3: "c",
				2: "c",
				1: "a",
				0: "a"
			}

			let dict2 = {
				7: "d"
			}

			dict2[1] = "c"
			dict2[3] = "b"

            emit Foo(
				bar: 2,
				aaa: {
					2: dict2,
					1: {
						3: "a",
						7: "b",
						2: "a",
						1: ""
					},
					0: dict0
				}
			)
        }
    `

	event := exportEventFromScript(t, script)

	bytes, err := json.Encode(event)

	assert.NoError(t, err)
	assert.Equal(t, "{\"type\":\"Event\",\"value\":{\"id\":\"S.test.Foo\",\"fields\":[{\"name\":\"bar\",\"value\":{\"type\":\"Int\",\"value\":\"2\"}},{\"name\":\"aaa\",\"value\":{\"type\":\"Dictionary\",\"value\":[{\"key\":{\"type\":\"Int\",\"value\":\"2\"},\"value\":{\"type\":\"Dictionary\",\"value\":[{\"key\":{\"type\":\"Int\",\"value\":\"7\"},\"value\":{\"type\":\"String\",\"value\":\"d\"}},{\"key\":{\"type\":\"Int\",\"value\":\"1\"},\"value\":{\"type\":\"String\",\"value\":\"c\"}},{\"key\":{\"type\":\"Int\",\"value\":\"3\"},\"value\":{\"type\":\"String\",\"value\":\"b\"}}]}},{\"key\":{\"type\":\"Int\",\"value\":\"1\"},\"value\":{\"type\":\"Dictionary\",\"value\":[{\"key\":{\"type\":\"Int\",\"value\":\"3\"},\"value\":{\"type\":\"String\",\"value\":\"a\"}},{\"key\":{\"type\":\"Int\",\"value\":\"7\"},\"value\":{\"type\":\"String\",\"value\":\"b\"}},{\"key\":{\"type\":\"Int\",\"value\":\"2\"},\"value\":{\"type\":\"String\",\"value\":\"a\"}},{\"key\":{\"type\":\"Int\",\"value\":\"1\"},\"value\":{\"type\":\"String\",\"value\":\"\"}}]}},{\"key\":{\"type\":\"Int\",\"value\":\"0\"},\"value\":{\"type\":\"Dictionary\",\"value\":[{\"key\":{\"type\":\"Int\",\"value\":\"3\"},\"value\":{\"type\":\"String\",\"value\":\"c\"}},{\"key\":{\"type\":\"Int\",\"value\":\"2\"},\"value\":{\"type\":\"String\",\"value\":\"c\"}},{\"key\":{\"type\":\"Int\",\"value\":\"1\"},\"value\":{\"type\":\"String\",\"value\":\"a\"}},{\"key\":{\"type\":\"Int\",\"value\":\"0\"},\"value\":{\"type\":\"String\",\"value\":\"a\"}}]}}]}}]}}\n", string(bytes))
}

const fooID = "Foo"

var fooTypeID = fmt.Sprintf("S.%s.%s", utils.TestLocation, fooID)
var fooFields = []cadence.Field{
	{
		Identifier: "bar",
		Type:       cadence.IntType{},
	},
}
var fooResourceFields = []cadence.Field{
	{
		Identifier: "uuid",
		Type:       cadence.UInt64Type{},
	},
	{
		Identifier: "bar",
		Type:       cadence.IntType{},
	},
}

var fooStructType = &cadence.StructType{
	TypeID:     fooTypeID,
	Identifier: fooID,
	Fields:     fooFields,
}

var fooResourceType = &cadence.ResourceType{
	TypeID:     fooTypeID,
	Identifier: fooID,
	Fields:     fooResourceFields,
}

var fooEventType = &cadence.EventType{
	TypeID:     fooTypeID,
	Identifier: fooID,
	Fields:     fooFields,
}
