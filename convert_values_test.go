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

package cadence

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

var convertTests = []struct {
	label       string
	value       interpreter.Value
	expected    Value
	skipReverse bool
}{
	{
		label:    "Void",
		value:    interpreter.VoidValue{},
		expected: NewVoid(),
	},
	{
		label:       "Nil",
		value:       interpreter.NilValue{},
		expected:    NewOptional(nil),
		skipReverse: true,
	},
	{
		label:    "SomeValue nil",
		value:    &interpreter.SomeValue{Value: nil},
		expected: NewOptional(nil),
	},
	{
		label:    "SomeValue non-nil",
		value:    &interpreter.SomeValue{Value: interpreter.NewIntValueFromInt64(42)},
		expected: NewOptional(NewInt(42)),
	},
	{
		label:    "Bool true",
		value:    interpreter.BoolValue(true),
		expected: NewBool(true),
	},
	{
		label:    "Bool false",
		value:    interpreter.BoolValue(false),
		expected: NewBool(false),
	},
	{
		label:    "String empty",
		value:    &interpreter.StringValue{Str: ""},
		expected: NewString(""),
	},
	{
		label:    "String non-empty",
		value:    &interpreter.StringValue{Str: "foo"},
		expected: NewString("foo"),
	},
	{
		label:    "Array empty",
		value:    &interpreter.ArrayValue{Values: []interpreter.Value{}},
		expected: NewArray([]Value{}),
	},
	{
		label: "Array non-empty",
		value: &interpreter.ArrayValue{
			Values: []interpreter.Value{
				interpreter.NewIntValueFromInt64(42),
				interpreter.NewStringValue("foo"),
			},
		},
		expected: NewArray([]Value{
			NewInt(42),
			NewString("foo"),
		}),
	},
	{
		label:    "Int",
		value:    interpreter.NewIntValueFromInt64(42),
		expected: NewInt(42),
	},
	{
		label:    "Int8",
		value:    interpreter.Int8Value(42),
		expected: NewInt8(42),
	},
	{
		label:    "Int16",
		value:    interpreter.Int16Value(42),
		expected: NewInt16(42),
	},
	{
		label:    "Int32",
		value:    interpreter.Int32Value(42),
		expected: NewInt32(42),
	},
	{
		label:    "Int64",
		value:    interpreter.Int64Value(42),
		expected: NewInt64(42),
	},
	{
		label:    "Int128",
		value:    interpreter.NewInt128ValueFromInt64(42),
		expected: NewInt128(42),
	},
	{
		label:    "Int256",
		value:    interpreter.NewInt256ValueFromInt64(42),
		expected: NewInt256(42),
	},
	{
		label:    "UInt",
		value:    interpreter.NewUIntValueFromUint64(42),
		expected: NewUInt(42),
	},
	{
		label:    "UInt8",
		value:    interpreter.UInt8Value(42),
		expected: NewUInt8(42),
	},
	{
		label:    "UInt16",
		value:    interpreter.UInt16Value(42),
		expected: NewUInt16(42),
	},
	{
		label:    "UInt32",
		value:    interpreter.UInt32Value(42),
		expected: NewUInt32(42),
	},
	{
		label:    "UInt64",
		value:    interpreter.UInt64Value(42),
		expected: NewUInt64(42),
	},
	{
		label:    "UInt128",
		value:    interpreter.NewUInt128ValueFromInt64(42),
		expected: NewUInt128(42),
	},
	{
		label:    "UInt256",
		value:    interpreter.NewUInt256ValueFromInt64(42),
		expected: NewUInt256(42),
	},
	{
		label:    "Word8",
		value:    interpreter.Word8Value(42),
		expected: NewWord8(42),
	},
	{
		label:    "Word16",
		value:    interpreter.Word16Value(42),
		expected: NewWord16(42),
	},
	{
		label:    "Word32",
		value:    interpreter.Word32Value(42),
		expected: NewWord32(42),
	},
	{
		label:    "Word64",
		value:    interpreter.Word64Value(42),
		expected: NewWord64(42),
	},
	{
		label:    "Fix64",
		value:    interpreter.Fix64Value(-123000000),
		expected: NewFix64(-123000000),
	},
	{
		label:    "UFix64",
		value:    interpreter.UFix64Value(123000000),
		expected: NewUFix64(123000000),
	},
}

func TestConvertValue(t *testing.T) {
	for _, tt := range convertTests {
		t.Run(tt.label, func(t *testing.T) {
			actual := convertValue(tt.value, nil)
			assert.Equal(t, tt.expected, actual)

			if !tt.skipReverse {
				original := actual.ToRuntimeValue()
				assert.Equal(t, tt.value, original.Value)
			}
		})
	}
}

func TestConvertIntegerValuesFromScript(t *testing.T) {
	for _, integerType := range sema.AllIntegerTypes {

		script := fmt.Sprintf(
			`
              pub fun main(): %s {
                  return 42
              }
            `,
			integerType,
		)

		assert.NotPanics(t, func() {
			convertValueFromScript(t, script)
		})
	}
}

func TestConvertFixedPointValuesFromScript(t *testing.T) {
	for _, fixedPointType := range sema.AllFixedPointTypes {
		script := fmt.Sprintf(
			`
              pub fun main(): %s {
                  return 1.23
              }
            `,
			fixedPointType,
		)

		assert.NotPanics(t, func() {
			convertValueFromScript(t, script)
		})
	}
}

func TestConvertDictionaryValueFromScript(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		script := `
            access(all) fun main(): {String: Int} {
                return {}
            }
        `

		actual := convertValueFromScript(t, script)
		expected := NewDictionary([]KeyValuePair{})

		assert.Equal(t, expected, actual)
	})

	t.Run("Non-empty", func(t *testing.T) {
		script := `
            access(all) fun main(): {String: Int} {
                return {
                    "a": 1,
                    "b": 2
                }
            }
        `

		actual := convertValueFromScript(t, script)
		expected := NewDictionary([]KeyValuePair{
			{
				Key:   NewString("a"),
				Value: NewInt(1),
			},
			{
				Key:   NewString("b"),
				Value: NewInt(2),
			},
		})

		assert.Equal(t, expected, actual)
	})
}

func TestConvertAddressValueFromScript(t *testing.T) {
	script := `
        access(all) fun main(): Address {
            return 0x42
        }
    `

	actual := convertValueFromScript(t, script)
	expected := NewAddressFromBytes(
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x42},
	)

	assert.Equal(t, expected, actual)
}

func TestConvertStructValueFromScript(t *testing.T) {
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

	actual := convertValueFromScript(t, script)
	expected := NewStruct([]Value{NewInt(42)}).WithType(fooStructType)

	assert.Equal(t, expected, actual)
}

func TestConvertResourceValueFromScript(t *testing.T) {
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

	actual := convertValueFromScript(t, script)
	expected :=
		NewResource([]Value{
			NewInt(42),
			NewUInt64(0),
		}).WithType(fooResourceType)

	assert.Equal(t, expected, actual)
}

func TestConvertResourceArrayValueFromScript(t *testing.T) {
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

	actual := convertValueFromScript(t, script)
	expected := NewArray([]Value{
		NewResource([]Value{
			NewInt(1),
			NewUInt64(0),
		}).WithType(fooResourceType),
		NewResource([]Value{
			NewInt(2),
			NewUInt64(0),
		}).WithType(fooResourceType),
	})

	assert.Equal(t, expected, actual)
}

func TestConvertResourceDictionaryValueFromScript(t *testing.T) {
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

	actual := convertValueFromScript(t, script)
	expected := NewDictionary([]KeyValuePair{
		{
			Key: NewString("a"),
			Value: NewResource([]Value{
				NewInt(1),
				NewUInt64(0),
			}).WithType(fooResourceType),
		},
		{
			Key: NewString("b"),
			Value: NewResource([]Value{
				NewInt(2),
				NewUInt64(0),
			}).WithType(fooResourceType),
		},
	})

	assert.Equal(t, expected, actual)
}

func TestConvertNestedResourceValueFromScript(t *testing.T) {
	barResourceType := ResourceType{
		TypeID:     "test.Bar",
		Identifier: "Bar",
		Fields: []Field{
			{
				Identifier: "uuid",
				Type:       UInt64Type{},
			},
			{
				Identifier: "x",
				Type:       IntType{},
			},
		},
	}

	fooResourceType := ResourceType{
		TypeID:     "test.Foo",
		Identifier: "Foo",
		Fields: []Field{
			{
				Identifier: "bar",
				Type:       barResourceType,
			},
			{
				Identifier: "uuid",
				Type:       UInt64Type{},
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

	actual := convertValueFromScript(t, script)
	expected := NewResource([]Value{
		NewResource([]Value{
			NewUInt64(0),
			NewInt(42),
		}).WithType(barResourceType),
		NewUInt64(0),
	}).WithType(fooResourceType)

	assert.Equal(t, expected, actual)
}

func TestConvertEventValueFromScript(t *testing.T) {
	script := `
        access(all) event Foo(bar: Int)

        access(all) fun main() {
            emit Foo(bar: 42)
        }
    `

	actual := convertEventFromScript(t, script)
	expected := NewEvent([]Value{NewInt(42)}).WithType(fooEventType)

	assert.Equal(t, expected, actual)
}

// mock runtime.Interface to capture events
type eventCapturingInterface struct {
	runtime.EmptyRuntimeInterface
	events []runtime.Event
}

func (t *eventCapturingInterface) EmitEvent(event runtime.Event) {
	t.events = append(t.events, event)
}

func convertEventFromScript(t *testing.T, script string) Event {
	rt := runtime.NewInterpreterRuntime()

	inter := &eventCapturingInterface{}

	_, err := rt.ExecuteScript(
		[]byte(script),
		inter,
		testLocation,
	)

	require.NoError(t, err)
	require.Len(t, inter.events, 1)

	event := inter.events[0]

	return ConvertEvent(event)
}

func convertValueFromScript(t *testing.T, script string) Value {
	rt := runtime.NewInterpreterRuntime()

	value, err := rt.ExecuteScript(
		[]byte(script),
		&runtime.EmptyRuntimeInterface{},
		testLocation,
	)

	require.NoError(t, err)

	return ConvertValue(value)
}

const testLocation = runtime.StringLocation("test")

const fooID = "Foo"

var fooTypeID = fmt.Sprintf("%s.%s", testLocation, fooID)
var fooFields = []Field{
	{
		Identifier: "bar",
		Type:       IntType{},
	},
}
var fooResourceFields = []Field{
	{
		Identifier: "bar",
		Type:       IntType{},
	},
	{
		Identifier: "uuid",
		Type:       UInt64Type{},
	},
}

var fooStructType = StructType{
	TypeID:     fooTypeID,
	Identifier: fooID,
	Fields:     fooFields,
}

var fooResourceType = ResourceType{
	TypeID:     fooTypeID,
	Identifier: fooID,
	Fields:     fooResourceFields,
}

var fooEventType = EventType{
	TypeID:     fooTypeID,
	Identifier: fooID,
	Fields:     fooFields,
}
