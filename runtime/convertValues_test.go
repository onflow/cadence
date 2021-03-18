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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser2"
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
            pub fun main(): {String: Int} {
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
            pub fun main(): {String: Int} {
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
        pub fun main(): Address {
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
        pub struct Foo {
            pub let bar: Int

            init(bar: Int) {
                self.bar = bar
            }
        }

        pub fun main(): Foo {
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
        pub resource Foo {
            pub let bar: Int

            init(bar: Int) {
                self.bar = bar
            }
        }

        pub fun main(): @Foo {
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
        pub resource Foo {
            pub let bar: Int

            init(bar: Int) {
                self.bar = bar
            }
        }

        pub fun main(): @[Foo] {
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
        pub resource Foo {
            pub let bar: Int

            init(bar: Int) {
                self.bar = bar
            }
        }

        pub fun main(): @{String: Foo} {
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
		Location:            utils.TestLocation,
		QualifiedIdentifier: "Bar",
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
		Location:            utils.TestLocation,
		QualifiedIdentifier: "Foo",
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
        pub resource Bar {
            pub let x: Int

            init(x: Int) {
                self.x = x
            }
        }

        pub resource Foo {
            pub let bar: @Bar

            init(bar: @Bar) {
                self.bar <- bar
            }

            destroy() {
                destroy self.bar
            }
        }

        pub fun main(): @Foo {
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
        pub event Foo(bar: Int)

        pub fun main() {
            emit Foo(bar: 42)
        }
    `

	actual := exportEventFromScript(t, script)
	expected := cadence.NewEvent([]cadence.Value{cadence.NewInt(42)}).WithType(fooEventType)

	assert.Equal(t, expected, actual)
}

// mock runtime.Interface to capture events
type eventCapturingInterface struct {
	emptyRuntimeInterface
	events []cadence.Event
}

func (t *eventCapturingInterface) EmitEvent(event cadence.Event) error {
	t.events = append(t.events, event)
	return nil
}

func exportEventFromScript(t *testing.T, script string) cadence.Event {
	rt := NewInterpreterRuntime()

	inter := &eventCapturingInterface{}
	inter.programs = map[common.LocationID]*interpreter.Program{}

	_, err := rt.ExecuteScript(
		Script{
			Source: []byte(script),
		},
		Context{
			Interface: inter,
			Location:  utils.TestLocation,
		},
	)

	require.NoError(t, err)
	require.Len(t, inter.events, 1)

	event := inter.events[0]

	return event
}

func exportValueFromScript(t *testing.T, script string) cadence.Value {
	rt := NewInterpreterRuntime()

	value, err := rt.ExecuteScript(
		Script{
			Source: []byte(script),
		},
		Context{
			Interface: NewEmptyRuntimeInterface(),
			Location:  utils.TestLocation,
		},
	)

	require.NoError(t, err)

	return value
}

func TestExportTypeValue(t *testing.T) {

	t.Parallel()

	t.Run("Int", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main(): Type {
                return Type<Int>()
            }
        `

		actual := exportValueFromScript(t, script)
		expected := cadence.TypeValue{
			StaticType: "Int",
		}

		assert.Equal(t, expected, actual)
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		script := `
            pub struct S {}

            pub fun main(): Type {
                return Type<S>()
            }
        `

		actual := exportValueFromScript(t, script)
		expected := cadence.TypeValue{
			StaticType: "S.test.S",
		}

		assert.Equal(t, expected, actual)
	})

	t.Run("without static type", func(t *testing.T) {

		t.Parallel()

		value := interpreter.TypeValue{
			Type: nil,
		}
		actual := exportValueWithInterpreter(value, nil, exportResults{})
		expected := cadence.TypeValue{
			StaticType: "",
		}

		assert.Equal(t, expected, actual)
	})

	t.Run("with restricted static type", func(t *testing.T) {

		t.Parallel()

		program, err := parser2.ParseProgram(`
          pub struct interface SI {}

          pub struct S: SI {}

        `)
		require.NoError(t, err)

		checker, err := sema.NewChecker(program, utils.TestLocation)
		require.NoError(t, err)

		err = checker.Check()
		require.NoError(t, err)

		inter, err := interpreter.NewInterpreter(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		require.NoError(t, err)

		ty := interpreter.TypeValue{
			Type: &interpreter.RestrictedStaticType{
				Type: interpreter.CompositeStaticType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "S",
				},
				Restrictions: []interpreter.InterfaceStaticType{
					{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "SI",
					},
				},
			},
		}

		assert.Equal(t,
			cadence.TypeValue{
				StaticType: "S.test.S{S.test.SI}",
			},
			ExportValue(ty, inter),
		)
	})

}

func TestExportCapabilityValue(t *testing.T) {

	t.Parallel()

	t.Run("Int", func(t *testing.T) {

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

	})

	t.Run("Struct", func(t *testing.T) {

		program, err := parser2.ParseProgram(`pub struct S {}`)
		require.NoError(t, err)

		checker, err := sema.NewChecker(program, utils.TestLocation)
		require.NoError(t, err)

		err = checker.Check()
		require.NoError(t, err)

		inter, err := interpreter.NewInterpreter(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		require.NoError(t, err)

		capability := interpreter.CapabilityValue{
			Address: interpreter.AddressValue{0x1},
			Path: interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			BorrowType: interpreter.CompositeStaticType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "S",
			},
		}
		actual := exportValueWithInterpreter(capability, inter, exportResults{})
		expected := cadence.Capability{
			Path: cadence.Path{
				Domain:     "storage",
				Identifier: "foo",
			},
			Address:    cadence.Address{0x1},
			BorrowType: "S.test.S",
		}

		assert.Equal(t, expected, actual)
	})

	t.Run("no borrow type", func(t *testing.T) {

		capability := interpreter.CapabilityValue{
			Address: interpreter.AddressValue{0x1},
			Path: interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
		}
		actual := exportValueWithInterpreter(capability, nil, exportResults{})
		expected := cadence.Capability{
			Path: cadence.Path{
				Domain:     "storage",
				Identifier: "foo",
			},
			Address: cadence.Address{0x1},
		}

		assert.Equal(t, expected, actual)
	})
}

func TestExportLinkValue(t *testing.T) {

	t.Parallel()

	t.Run("Int", func(t *testing.T) {

		link := interpreter.LinkValue{
			TargetPath: interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			Type: interpreter.PrimitiveStaticTypeInt,
		}
		actual := exportValueWithInterpreter(link, nil, exportResults{})
		expected := cadence.Link{
			TargetPath: cadence.Path{
				Domain:     "storage",
				Identifier: "foo",
			},
			BorrowType: "Int",
		}

		assert.Equal(t, expected, actual)
	})

	t.Run("Struct", func(t *testing.T) {

		program, err := parser2.ParseProgram(`pub struct S {}`)
		require.NoError(t, err)

		checker, err := sema.NewChecker(program, utils.TestLocation)
		require.NoError(t, err)

		err = checker.Check()
		require.NoError(t, err)

		inter, err := interpreter.NewInterpreter(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		require.NoError(t, err)

		capability := interpreter.LinkValue{
			TargetPath: interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			Type: interpreter.CompositeStaticType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "S",
			},
		}
		actual := exportValueWithInterpreter(capability, inter, exportResults{})
		expected := cadence.Link{
			TargetPath: cadence.Path{
				Domain:     "storage",
				Identifier: "foo",
			},
			BorrowType: "S.test.S",
		}

		assert.Equal(t, expected, actual)
	})
}

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
	Location:            utils.TestLocation,
	QualifiedIdentifier: "Foo",
	Fields:              fooFields,
}

var fooResourceType = &cadence.ResourceType{
	Location:            utils.TestLocation,
	QualifiedIdentifier: "Foo",
	Fields:              fooResourceFields,
}

var fooEventType = &cadence.EventType{
	Location:            utils.TestLocation,
	QualifiedIdentifier: "Foo",
	Fields:              fooFields,
}

func TestEnumValue(t *testing.T) {

	t.Parallel()

	enumValue := cadence.Enum{
		EnumType: &cadence.EnumType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "Direction",
			Fields: []cadence.Field{
				{
					Identifier: sema.EnumRawValueFieldName,
					Type:       cadence.IntType{},
				},
			},
			RawType: cadence.IntType{},
		},
		Fields: []cadence.Value{
			cadence.NewInt(3),
		},
	}

	t.Run("test export", func(t *testing.T) {
		script := `
			pub fun main(): Direction {
				return Direction.RIGHT
			}

			pub enum Direction: Int {
				pub case UP
				pub case DOWN
				pub case LEFT
				pub case RIGHT
			}
		`

		actual := exportValueFromScript(t, script)
		assert.Equal(t, enumValue, actual)
	})

	t.Run("test import", func(t *testing.T) {
		script := `
			pub fun main(dir: Direction): Direction {
				if !dir.isInstance(Type<Direction>()) {
					panic("Not a Direction value")
				}

				return dir
			}

			pub enum Direction: Int {
				pub case UP
				pub case DOWN
				pub case LEFT
				pub case RIGHT
			}
		`

		actual, err := importAndExportValuesFromScript(t, script, enumValue)
		require.NoError(t, err)
		assert.Equal(t, enumValue, actual)
	})
}

func importAndExportValuesFromScript(t *testing.T, script string, arg cadence.Value) (cadence.Value, error) {
	encodedArg, err := json.Encode(arg)
	require.NoError(t, err)

	rt := NewInterpreterRuntime()

	runtimeInterface := &testRuntimeInterface{
		decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(b)
		},
	}

	return rt.ExecuteScript(
		Script{
			Source:    []byte(script),
			Arguments: [][]byte{encodedArg},
		},
		Context{
			Interface: runtimeInterface,
			Location:  utils.TestLocation,
		},
	)
}

type importValueTest struct {
	label         string
	typeSignature string
	exportedValue cadence.Value
	skipExport    bool
}

func TestImportValue(t *testing.T) {

	t.Parallel()

	// Complex struct value
	fooStructValue := cadence.Struct{
		StructType: &cadence.StructType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "Foo",
			Fields: []cadence.Field{
				{
					Identifier: "a",
					Type: cadence.OptionalType{
						Type: cadence.StringType{},
					},
				},
				{
					Identifier: "b",
					Type: cadence.DictionaryType{
						KeyType:     cadence.StringType{},
						ElementType: cadence.StringType{},
					},
				},
				{
					Identifier: "c",
					Type: cadence.VariableSizedArrayType{
						ElementType: cadence.StringType{},
					},
				},
				{
					Identifier: "d",
					Type: cadence.ConstantSizedArrayType{
						ElementType: cadence.StringType{},
						Size:        2,
					},
				},
				{
					Identifier: "e",
					Type:       cadence.AddressType{},
				},
			},
		},
		Fields: []cadence.Value{
			cadence.NewString("John"),
			cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.NewString("name"),
					Value: cadence.NewString("Doe"),
				},
			}),
			cadence.NewArray([]cadence.Value{
				cadence.NewString("foo"),
				cadence.NewString("bar"),
			}),
			cadence.NewArray([]cadence.Value{
				cadence.NewString("foo"),
				cadence.NewString("bar"),
			}),
			cadence.NewAddress([8]byte{0, 0, 0, 0, 0, 1, 0, 2}),
		},
	}

	var importValueTests = []importValueTest{
		{
			label:         "Nil",
			typeSignature: "String?",
			exportedValue: cadence.NewOptional(nil),
		},
		{
			label:         "Bool true",
			typeSignature: "Bool",
			exportedValue: cadence.NewBool(true),
		},
		{
			label:         "Bool false",
			typeSignature: "Bool",
			exportedValue: cadence.NewBool(false),
		},
		{
			label:         "String empty",
			typeSignature: "String",
			exportedValue: cadence.NewString(""),
		},
		{
			label:         "String non-empty",
			typeSignature: "String",
			exportedValue: cadence.NewString("foo"),
		},
		{
			label:         "Array empty",
			typeSignature: "[String]",
			exportedValue: cadence.NewArray([]cadence.Value{}),
		},
		{
			label:         "Array non-empty",
			typeSignature: "[String]",
			exportedValue: cadence.NewArray([]cadence.Value{
				cadence.NewString("foo"),
				cadence.NewString("bar"),
			}),
		},
		{
			label:         "Dictionary non-empty",
			typeSignature: "{String: Foo}",
			exportedValue: cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.NewString("foo"),
					Value: fooStructValue,
				},
			}),
		},
		{
			label:         "Int",
			typeSignature: "Int",
			exportedValue: cadence.NewInt(42),
		},
		{
			label:         "Int8",
			typeSignature: "Int8",
			exportedValue: cadence.NewInt8(42),
		},
		{
			label:         "Int16",
			typeSignature: "Int16",
			exportedValue: cadence.NewInt16(42),
		},
		{
			label:         "Int32",
			typeSignature: "Int32",
			exportedValue: cadence.NewInt32(42),
		},
		{
			label:         "Int64",
			typeSignature: "Int64",
			exportedValue: cadence.NewInt64(42),
		},
		{
			label:         "Int128",
			typeSignature: "Int128",
			exportedValue: cadence.NewInt128(42),
		},
		{
			label:         "Int256",
			typeSignature: "Int256",
			exportedValue: cadence.NewInt256(42),
		},
		{
			label:         "UInt",
			typeSignature: "UInt",
			exportedValue: cadence.NewUInt(42),
		},
		{
			label:         "UInt8",
			typeSignature: "UInt8",
			exportedValue: cadence.NewUInt8(42),
		},
		{
			label:         "UInt16",
			typeSignature: "UInt16",
			exportedValue: cadence.NewUInt16(42),
		},
		{
			label:         "UInt32",
			typeSignature: "UInt32",
			exportedValue: cadence.NewUInt32(42),
		},
		{
			label:         "UInt64",
			typeSignature: "UInt64",
			exportedValue: cadence.NewUInt64(42),
		},
		{
			label:         "UInt128",
			typeSignature: "UInt128",
			exportedValue: cadence.NewUInt128(42),
		},
		{
			label:         "UInt256",
			typeSignature: "UInt256",
			exportedValue: cadence.NewUInt256(42),
		},
		{
			label:         "Word8",
			typeSignature: "Word8",
			exportedValue: cadence.NewWord8(42),
		},
		{
			label:         "Word16",
			typeSignature: "Word16",
			exportedValue: cadence.NewWord16(42),
		},
		{
			label:         "Word32",
			typeSignature: "Word32",
			exportedValue: cadence.NewWord32(42),
		},
		{
			label:         "Word64",
			typeSignature: "Word64",
			exportedValue: cadence.NewWord64(42),
		},
		{
			label:         "Fix64",
			typeSignature: "Fix64",
			exportedValue: cadence.Fix64(-123000000),
		},
		{
			label:         "UFix64",
			typeSignature: "UFix64",
			exportedValue: cadence.UFix64(123000000),
		},
		{
			label:         "Struct",
			typeSignature: "Foo",
			exportedValue: fooStructValue,
		},
		{
			label:         "StoragePath",
			typeSignature: "StoragePath",
			exportedValue: cadence.Path{
				Domain:     "storage",
				Identifier: "foo",
			},
			skipExport: true,
		},
		{
			label:         "PrivatePath",
			typeSignature: "PrivatePath",
			exportedValue: cadence.Path{
				Domain:     "private",
				Identifier: "foo",
			},
			skipExport: true,
		},
		{
			label:         "PublicPath",
			typeSignature: "PublicPath",
			exportedValue: cadence.Path{
				Domain:     "public",
				Identifier: "foo",
			},
			skipExport: true,
		},
		{
			label:         "Address",
			typeSignature: "Address",
			exportedValue: cadence.NewAddress([8]byte{0, 0, 0, 0, 0, 1, 0, 2}),
		},

		// TODO: Enable once https://github.com/onflow/cadence/issues/712 is fixed.
		//{
		//	label:         "Capability",
		//	typeSignature: "Capability<&Foo>",
		//	exportedValue: cadence.Capability{
		//		Path: cadence.Path{
		//			Domain:     "public",
		//			Identifier: "bar",
		//		},
		//		Address:    cadence.NewAddress([8]byte{0, 0, 0, 0, 0, 1, 0, 2}),
		//		BorrowType: "Foo",
		//	},
		//},

		// TODO: enable once https://github.com/onflow/cadence/issues/491 is fixed.
		//{
		//	label:         "Type",
		//	typeSignature: "Type",
		//	exportedValue: cadence.TypeValue{
		//		StaticType: "Foo",
		//	},
		//},

	}

	testImport := func(test importValueTest) {

		t.Run(test.label, func(t *testing.T) {

			t.Parallel()

			returnSignature := ""
			returnStmt := ""
			if !test.skipExport {
				returnSignature = fmt.Sprintf(": %[1]s", test.typeSignature)
				returnStmt = "return arg"
			}

			script := fmt.Sprintf(
				`pub fun main(arg: %[1]s)%[2]s {

					if !arg.isInstance(Type<%[1]s>()) {
						panic("Not a %[1]s value")
					}

					%[3]s
				}

				pub struct Foo {
					pub var a: String?
					pub var b: {String: String}
					pub var c: [String]
					pub var d: [String; 2]
					pub var e: Address

					init() {
						self.a = "Hello"
						self.b = {}
						self.c = []
						self.d = ["foo", "bar"]
						self.e = 0x42
					}
				}`,
				test.typeSignature,
				returnSignature,
				returnStmt,
			)

			actual, err := importAndExportValuesFromScript(t, script, test.exportedValue)
			require.NoError(t, err)
			if !test.skipExport {
				assert.Equal(t, test.exportedValue, actual)
			}
		})
	}

	for _, testCase := range importValueTests {
		testImport(testCase)
	}
}

func TestMalformedImportValue(t *testing.T) {

	t.Parallel()

	// Struct with wrong field type

	malformedStructType1 := &cadence.StructType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: "Foo",
		Fields: []cadence.Field{
			{
				Identifier: "a",
				Type:       cadence.IntType{},
			},
		},
	}

	malformedStruct1 := cadence.Struct{
		StructType: malformedStructType1,
		Fields: []cadence.Value{
			cadence.NewInt(3),
		},
	}

	// Struct with wrong field name

	malformedStruct2 := cadence.Struct{
		StructType: &cadence.StructType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "Foo",
			Fields: []cadence.Field{
				{
					Identifier: "nonExisting",
					Type:       cadence.StringType{},
				},
			},
		},
		Fields: []cadence.Value{
			cadence.NewString("John"),
		},
	}

	// Struct with nested malformed array value
	malformedStruct3 := cadence.Struct{
		StructType: &cadence.StructType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "Bar",
			Fields: []cadence.Field{
				{
					Identifier: "a",
					Type: cadence.VariableSizedArrayType{
						ElementType: malformedStructType1,
					},
				},
			},
		},
		Fields: []cadence.Value{
			cadence.NewArray([]cadence.Value{
				malformedStruct1,
			}),
		},
	}

	// Struct with nested malformed dictionary value
	malformedStruct4 := cadence.Struct{
		StructType: &cadence.StructType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "Baz",
			Fields: []cadence.Field{
				{
					Identifier: "a",
					Type: cadence.DictionaryType{
						KeyType:     cadence.StringType{},
						ElementType: malformedStructType1,
					},
				},
			},
		},
		Fields: []cadence.Value{
			cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.NewString("foo"),
					Value: malformedStruct1,
				},
			}),
		},
	}

	var importValueTests = []importValueTest{
		{
			label:         "Malformed Struct field type",
			typeSignature: "Foo",
			exportedValue: malformedStruct1,
		},
		{
			label:         "Malformed Struct field name",
			typeSignature: "Foo",
			exportedValue: malformedStruct2,
		},
		{
			label:         "Malformed AnyStruct",
			typeSignature: "AnyStruct",
			exportedValue: malformedStruct1,
		},
		{
			label:         "Malformed nested struct array",
			typeSignature: "Bar",
			exportedValue: malformedStruct3,
		},
		{
			label:         "Malformed nested struct dictionary",
			typeSignature: "Baz",
			exportedValue: malformedStruct4,
		},
		{
			label:         "Array with malformed member",
			typeSignature: "[Foo]",
			exportedValue: cadence.NewArray([]cadence.Value{
				malformedStruct1,
			}),
		},
		{
			label:         "Array with wrong size",
			typeSignature: "[String; 2]",
			exportedValue: cadence.NewArray([]cadence.Value{
				malformedStruct1,
			}),
		},
		{
			label:         "Malformed Optional",
			typeSignature: "Foo?",
			exportedValue: cadence.NewOptional(malformedStruct1),
		},
		{
			label:         "Malformed Map",
			typeSignature: "{String : Foo}",
			exportedValue: cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.NewString("foo"),
					Value: malformedStruct1,
				},
			}),
		},
	}

	testImport := func(test importValueTest) {

		t.Run(test.label, func(t *testing.T) {

			t.Parallel()

			script := fmt.Sprintf(
				`pub fun main(arg: %[1]s): %[1]s {

					if !arg.isInstance(Type<%[1]s>()) {
						panic("Not a %[1]s value")
					}

					return arg
				}

				pub struct Foo {
					pub var a: String

					init() {
						self.a = "Hello"
					}
				}

				pub struct Bar {
					pub var a: [Foo]

					init() {
						self.a = []
					}
				}

				pub struct Baz {
					pub var a: {String: Foo}

					init() {
						self.a = {}
					}
				}`,
				test.typeSignature,
			)

			_, err := importAndExportValuesFromScript(t, script, test.exportedValue)
			require.Error(t, err)
			assert.Contains(
				t,
				err.Error(),
				fmt.Sprintf("malformed value `%s`", test.exportedValue.String()),
			)
		})
	}

	for _, testCase := range importValueTests {
		testImport(testCase)
	}
}
