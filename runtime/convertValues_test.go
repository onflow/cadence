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
	"unicode/utf8"

	"github.com/onflow/cadence/encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestExportValue(t *testing.T) {

	t.Parallel()

	type exportTest struct {
		label    string
		value    interpreter.Value
		expected cadence.Value
	}

	test := func(tt exportTest) {

		t.Run(tt.label, func(t *testing.T) {

			t.Parallel()

			actual, err := exportValueWithInterpreter(tt.value, nil, seenReferences{})
			if tt.expected == nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				assert.Equal(t, tt.expected, actual)
			}
		})
	}

	for _, tt := range []exportTest{
		{
			label:    "Void",
			value:    interpreter.VoidValue{},
			expected: cadence.NewVoid(),
		},
		{
			label:    "Nil",
			value:    interpreter.NilValue{},
			expected: cadence.NewOptional(nil),
		},
		{
			label: "SomeValue",
			value: interpreter.NewSomeValueNonCopying(
				interpreter.NewIntValueFromInt64(42),
			),
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
			expected: cadence.String(""),
		},
		{
			label:    "String non-empty",
			value:    interpreter.NewStringValue("foo"),
			expected: cadence.String("foo"),
		},
		{
			label: "Array empty",
			value: interpreter.NewArrayValue(
				newTestInterpreter(t),
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeAnyStruct,
				},
			),
			expected: cadence.NewArray([]cadence.Value{}),
		},
		{
			label: "Array (non-empty)",
			value: interpreter.NewArrayValue(
				newTestInterpreter(t),
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeAnyStruct,
				},
				interpreter.NewIntValueFromInt64(42),
				interpreter.NewStringValue("foo"),
			),
			expected: cadence.NewArray([]cadence.Value{
				cadence.NewInt(42),
				cadence.String("foo"),
			}),
		},
		{
			label: "Dictionary",
			value: interpreter.NewDictionaryValue(
				newTestInterpreter(t),
				interpreter.DictionaryStaticType{
					KeyType:   interpreter.PrimitiveStaticTypeString,
					ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
				},
			),
			expected: cadence.NewDictionary([]cadence.KeyValuePair{}),
		},
		{
			label: "Dictionary (non-empty)",
			value: interpreter.NewDictionaryValue(
				newTestInterpreter(t),
				interpreter.DictionaryStaticType{
					KeyType:   interpreter.PrimitiveStaticTypeString,
					ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
				},
				interpreter.NewStringValue("a"),
				interpreter.NewIntValueFromInt64(1),
				interpreter.NewStringValue("b"),
				interpreter.NewIntValueFromInt64(2),
			),
			expected: cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("a"),
					Value: cadence.NewInt(1),
				},
				{
					Key:   cadence.String("b"),
					Value: cadence.NewInt(2),
				},
			}),
		},
		{
			label:    "Address",
			value:    interpreter.NewAddressValueFromBytes([]byte{0x1}),
			expected: cadence.NewAddress([8]byte{0, 0, 0, 0, 0, 0, 0, 1}),
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
		{
			label:    "Block (invalid)",
			value:    interpreter.BlockValue{},
			expected: nil,
		},
		{
			label:    "Interpreted Function (invalid)",
			value:    &interpreter.InterpretedFunctionValue{},
			expected: nil,
		},
		{
			label:    "Host Function (invalid)",
			value:    &interpreter.HostFunctionValue{},
			expected: nil,
		},
		{
			label:    "Bound Function (invalid)",
			value:    interpreter.BoundFunctionValue{},
			expected: nil,
		},
		{
			label: "Deployed contract",
			value: interpreter.DeployedContractValue{
				Address: interpreter.AddressValue{},
				Name:    interpreter.NewStringValue("C"),
				Code: interpreter.NewArrayValue(
					newTestInterpreter(t),
					interpreter.ByteArrayStaticType,
				),
			},
			expected: nil,
		},
	} {
		test(tt)
	}

}

func TestImportValue(t *testing.T) {

	t.Parallel()

	type importTest struct {
		label        string
		expected     interpreter.Value
		value        cadence.Value
		expectedType sema.Type
	}

	test := func(tt importTest) {

		t.Run(tt.label, func(t *testing.T) {

			t.Parallel()

			inter := newTestInterpreter(t)

			actual, err := importValue(inter, tt.value, tt.expectedType)

			if tt.expected == nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				AssertValuesEqual(t, tt.expected, actual)
			}
		})
	}

	for _, tt := range []importTest{
		{
			label:    "Void",
			expected: interpreter.VoidValue{},
			value:    cadence.NewVoid(),
		},
		{
			label:    "Nil",
			value:    cadence.NewOptional(nil),
			expected: interpreter.NilValue{},
		},
		{
			label: "SomeValue",
			value: cadence.NewOptional(cadence.NewInt(42)),
			expected: interpreter.NewSomeValueNonCopying(
				interpreter.NewIntValueFromInt64(42),
			),
		},
		{
			label:    "Bool true",
			value:    cadence.NewBool(true),
			expected: interpreter.BoolValue(true),
		},
		{
			label:    "Bool false",
			expected: interpreter.BoolValue(false),
			value:    cadence.NewBool(false),
		},
		{
			label:    "String empty",
			value:    cadence.String(""),
			expected: interpreter.NewStringValue(""),
		},
		{
			label:    "String non-empty",
			value:    cadence.String("foo"),
			expected: interpreter.NewStringValue("foo"),
		},
		{
			label: "Array empty",
			value: cadence.NewArray([]cadence.Value{}),
			expected: interpreter.NewArrayValue(
				newTestInterpreter(t),
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeAnyStruct,
				},
			),
			expectedType: &sema.VariableSizedType{
				Type: sema.AnyStructType,
			},
		},
		{
			label: "Array non-empty",
			value: cadence.NewArray([]cadence.Value{
				cadence.NewInt(42),
				cadence.String("foo"),
			}),
			expected: interpreter.NewArrayValue(
				newTestInterpreter(t),
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeAnyStruct,
				},
				interpreter.NewIntValueFromInt64(42),
				interpreter.NewStringValue("foo"),
			),
			expectedType: &sema.VariableSizedType{
				Type: sema.AnyStructType,
			},
		},
		{
			label: "Dictionary",
			expected: interpreter.NewDictionaryValue(
				newTestInterpreter(t),
				interpreter.DictionaryStaticType{
					KeyType:   interpreter.PrimitiveStaticTypeString,
					ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
				},
			),
			value: cadence.NewDictionary([]cadence.KeyValuePair{}),
			expectedType: &sema.DictionaryType{
				KeyType:   sema.StringType,
				ValueType: sema.AnyStructType,
			},
		},
		{
			label: "Dictionary (non-empty)",
			expected: interpreter.NewDictionaryValue(
				newTestInterpreter(t),
				interpreter.DictionaryStaticType{
					KeyType:   interpreter.PrimitiveStaticTypeString,
					ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
				},
				interpreter.NewStringValue("a"),
				interpreter.NewIntValueFromInt64(1),
				interpreter.NewStringValue("b"),
				interpreter.NewIntValueFromInt64(2),
			),
			value: cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("a"),
					Value: cadence.NewInt(1),
				},
				{
					Key:   cadence.String("b"),
					Value: cadence.NewInt(2),
				},
			}),
			expectedType: &sema.DictionaryType{
				KeyType:   sema.StringType,
				ValueType: sema.AnyStructType,
			},
		},
		{
			label:    "Address",
			expected: interpreter.NewAddressValueFromBytes([]byte{0x1}),
			value:    cadence.NewAddress([8]byte{0, 0, 0, 0, 0, 0, 0, 1}),
		},
		{
			label:    "Int",
			value:    cadence.NewInt(42),
			expected: interpreter.NewIntValueFromInt64(42),
		},
		{
			label:    "Int8",
			value:    cadence.NewInt8(42),
			expected: interpreter.Int8Value(42),
		},
		{
			label:    "Int16",
			value:    cadence.NewInt16(42),
			expected: interpreter.Int16Value(42),
		},
		{
			label:    "Int32",
			value:    cadence.NewInt32(42),
			expected: interpreter.Int32Value(42),
		},
		{
			label:    "Int64",
			value:    cadence.NewInt64(42),
			expected: interpreter.Int64Value(42),
		},
		{
			label:    "Int128",
			value:    cadence.NewInt128(42),
			expected: interpreter.NewInt128ValueFromInt64(42),
		},
		{
			label:    "Int256",
			value:    cadence.NewInt256(42),
			expected: interpreter.NewInt256ValueFromInt64(42),
		},
		{
			label:    "UInt",
			value:    cadence.NewUInt(42),
			expected: interpreter.NewUIntValueFromUint64(42),
		},
		{
			label:    "UInt8",
			value:    cadence.NewUInt8(42),
			expected: interpreter.UInt8Value(42),
		},
		{
			label:    "UInt16",
			value:    cadence.NewUInt16(42),
			expected: interpreter.UInt16Value(42),
		},
		{
			label:    "UInt32",
			value:    cadence.NewUInt32(42),
			expected: interpreter.UInt32Value(42),
		},
		{
			label:    "UInt64",
			value:    cadence.NewUInt64(42),
			expected: interpreter.UInt64Value(42),
		},
		{
			label:    "UInt128",
			value:    cadence.NewUInt128(42),
			expected: interpreter.NewUInt128ValueFromUint64(42),
		},
		{
			label:    "UInt256",
			value:    cadence.NewUInt256(42),
			expected: interpreter.NewUInt256ValueFromUint64(42),
		},
		{
			label:    "Word8",
			value:    cadence.NewWord8(42),
			expected: interpreter.Word8Value(42),
		},
		{
			label:    "Word16",
			value:    cadence.NewWord16(42),
			expected: interpreter.Word16Value(42),
		},
		{
			label:    "Word32",
			value:    cadence.NewWord32(42),
			expected: interpreter.Word32Value(42),
		},
		{
			label:    "Word64",
			value:    cadence.NewWord64(42),
			expected: interpreter.Word64Value(42),
		},
		{
			label:    "Fix64",
			value:    cadence.Fix64(-123000000),
			expected: interpreter.Fix64Value(-123000000),
		},
		{
			label:    "UFix64",
			value:    cadence.UFix64(123000000),
			expected: interpreter.UFix64Value(123000000),
		},
		{
			label: "Path",
			value: cadence.Path{
				Domain:     "storage",
				Identifier: "foo",
			},
			expected: interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
		},
		{
			label: "Link (invalid)",
			value: cadence.Link{
				TargetPath: cadence.Path{
					Domain:     "storage",
					Identifier: "test",
				},
				BorrowType: "Int",
			},
			expected: nil,
		},
		{
			label: "Capability (invalid)",
			value: cadence.Capability{
				Path: cadence.Path{
					Domain:     "public",
					Identifier: "test",
				},
				BorrowType: "Int",
			},
			expected: nil,
		},
	} {
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

	test := func(fixedPointType sema.Type, literal string) {

		t.Run(fixedPointType.String(), func(t *testing.T) {

			t.Parallel()

			script := fmt.Sprintf(
				`
                  pub fun main(): %s {
                      return %s
                  }
                `,
				fixedPointType,
				literal,
			)

			assert.NotPanics(t, func() {
				exportValueFromScript(t, script)
			})
		})
	}

	for _, fixedPointType := range sema.AllFixedPointTypes {

		var literal string
		if sema.IsSubType(fixedPointType, sema.SignedFixedPointType) {
			literal = "-1.23"
		} else {
			literal = "1.23"
		}

		test(fixedPointType, literal)
	}
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
			Key: cadence.String("a"),
			Value: cadence.NewResource([]cadence.Value{
				cadence.NewUInt64(0),
				cadence.NewInt(1),
			}).WithType(fooResourceType),
		},
		{
			Key: cadence.String("b"),
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
		Location:            TestLocation,
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
		Location:            TestLocation,
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

func exportEventFromScript(t *testing.T, script string) cadence.Event {
	rt := NewInterpreterRuntime()

	var events []cadence.Event

	inter := &testRuntimeInterface{
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	_, err := rt.ExecuteScript(
		Script{
			Source: []byte(script),
		},
		Context{
			Interface: inter,
			Location:  TestLocation,
		},
	)

	require.NoError(t, err)
	require.Len(t, events, 1)

	event := events[0]

	return event
}

func exportValueFromScript(t *testing.T, script string) cadence.Value {
	rt := NewInterpreterRuntime()

	value, err := rt.ExecuteScript(
		Script{
			Source: []byte(script),
		},
		Context{
			Interface: &testRuntimeInterface{},
			Location:  TestLocation,
		},
	)

	require.NoError(t, err)

	return value
}

func TestExportReferenceValue(t *testing.T) {

	t.Parallel()

	t.Run("ephemeral, Int", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main(): &Int {
                return &1 as &Int
            }
        `

		actual := exportValueFromScript(t, script)
		expected := cadence.NewInt(1)

		assert.Equal(t, expected, actual)
	})

	t.Run("ephemeral, recursive", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main(): [&AnyStruct] {
                let refs: [&AnyStruct] = []
                refs.append(&refs as &AnyStruct)
                return refs
            }
        `

		actual := exportValueFromScript(t, script)
		expected := cadence.NewArray([]cadence.Value{
			cadence.NewArray([]cadence.Value{
				nil,
			}),
		})

		assert.Equal(t, expected, actual)
	})

	t.Run("storage", func(t *testing.T) {

		t.Parallel()

		// Arrange

		rt := NewInterpreterRuntime()

		transaction := `
            transaction {
                prepare(signer: AuthAccount) {
                    signer.save(1, to: /storage/test)
                    signer.link<&Int>(
                        /public/test,
                        target: /storage/test
                    )
                }
            }
        `

		address, err := common.HexToAddress("0x1")
		require.NoError(t, err)

		runtimeInterface := &testRuntimeInterface{
			storage: newTestStorage(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return []Address{
					address,
				}, nil
			},
		}

		// Act

		err = rt.ExecuteTransaction(
			Script{
				Source: []byte(transaction),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.TransactionLocation{},
			},
		)
		require.NoError(t, err)

		script := `
            pub fun main(): &AnyStruct {
                return getAccount(0x1).getCapability(/public/test).borrow<&AnyStruct>()!
            }
        `

		actual, err := rt.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		expected := cadence.NewInt(1)

		assert.Equal(t, expected, actual)
	})
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
		actual, err := exportValueWithInterpreter(value, nil, seenReferences{})
		require.NoError(t, err)

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

		checker, err := sema.NewChecker(program, TestLocation)
		require.NoError(t, err)

		err = checker.Check()
		require.NoError(t, err)

		storage := interpreter.NewInMemoryStorage()

		inter, err := interpreter.NewInterpreter(
			interpreter.ProgramFromChecker(checker),
			TestLocation,
			interpreter.WithStorage(storage),
		)
		require.NoError(t, err)

		ty := interpreter.TypeValue{
			Type: &interpreter.RestrictedStaticType{
				Type: interpreter.CompositeStaticType{
					Location:            TestLocation,
					QualifiedIdentifier: "S",
				},
				Restrictions: []interpreter.InterfaceStaticType{
					{
						Location:            TestLocation,
						QualifiedIdentifier: "SI",
					},
				},
			},
		}

		actual, err := ExportValue(ty, inter)
		require.NoError(t, err)

		assert.Equal(t,
			cadence.TypeValue{
				StaticType: "S.test.S{S.test.SI}",
			},
			actual,
		)
	})

}

func TestExportCapabilityValue(t *testing.T) {

	t.Parallel()

	t.Run("Int", func(t *testing.T) {

		capability := &interpreter.CapabilityValue{
			Address: interpreter.AddressValue{0x1},
			Path: interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			BorrowType: interpreter.PrimitiveStaticTypeInt,
		}

		actual, err := exportValueWithInterpreter(capability, nil, seenReferences{})
		require.NoError(t, err)

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

		checker, err := sema.NewChecker(program, TestLocation)
		require.NoError(t, err)

		err = checker.Check()
		require.NoError(t, err)

		storage := interpreter.NewInMemoryStorage()

		inter, err := interpreter.NewInterpreter(
			interpreter.ProgramFromChecker(checker),
			TestLocation,
			interpreter.WithStorage(storage),
		)
		require.NoError(t, err)

		capability := &interpreter.CapabilityValue{
			Address: interpreter.AddressValue{0x1},
			Path: interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			BorrowType: interpreter.CompositeStaticType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
		}

		actual, err := exportValueWithInterpreter(capability, inter, seenReferences{})
		require.NoError(t, err)

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

		capability := &interpreter.CapabilityValue{
			Address: interpreter.AddressValue{0x1},
			Path: interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
		}

		actual, err := exportValueWithInterpreter(capability, nil, seenReferences{})
		require.NoError(t, err)

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

		actual, err := exportValueWithInterpreter(link, nil, seenReferences{})
		require.NoError(t, err)

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

		checker, err := sema.NewChecker(program, TestLocation)
		require.NoError(t, err)

		err = checker.Check()
		require.NoError(t, err)

		storage := interpreter.NewInMemoryStorage()

		inter, err := interpreter.NewInterpreter(
			interpreter.ProgramFromChecker(checker),
			TestLocation,
			interpreter.WithStorage(storage),
		)
		require.NoError(t, err)

		capability := interpreter.LinkValue{
			TargetPath: interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			Type: interpreter.CompositeStaticType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
		}

		actual, err := exportValueWithInterpreter(capability, inter, seenReferences{})
		require.NoError(t, err)

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
	Location:            TestLocation,
	QualifiedIdentifier: "Foo",
	Fields:              fooFields,
}

var fooResourceType = &cadence.ResourceType{
	Location:            TestLocation,
	QualifiedIdentifier: "Foo",
	Fields:              fooResourceFields,
}

var fooEventType = &cadence.EventType{
	Location:            TestLocation,
	QualifiedIdentifier: "Foo",
	Fields:              fooFields,
}

func TestRuntimeEnumValue(t *testing.T) {

	t.Parallel()

	enumValue := cadence.Enum{
		EnumType: &cadence.EnumType{
			Location:            TestLocation,
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

		actual, err := executeTestScript(t, script, enumValue)
		require.NoError(t, err)
		assert.Equal(t, enumValue, actual)
	})
}

func executeTestScript(t *testing.T, script string, arg cadence.Value) (cadence.Value, error) {
	encodedArg, err := json.Encode(arg)
	require.NoError(t, err)

	rt := NewInterpreterRuntime()

	storage := newTestStorage(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
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
			Location:  TestLocation,
		},
	)
}

func TestRuntimeArgumentPassing(t *testing.T) {

	t.Parallel()

	type argumentPassingTest struct {
		label         string
		typeSignature string
		exportedValue cadence.Value
		skipExport    bool
	}

	var argumentPassingTests = []argumentPassingTest{
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
			exportedValue: cadence.String(""),
		},
		{
			label:         "String non-empty",
			typeSignature: "String",
			exportedValue: cadence.String("foo"),
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
				cadence.String("foo"),
				cadence.String("bar"),
			}),
		},
		{
			label:         "Dictionary non-empty",
			typeSignature: "{String: String}",
			exportedValue: cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("foo"),
					Value: cadence.String("bar"),
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

		// TODO: Enable below once https://github.com/onflow/cadence/issues/712 is fixed.
		// TODO: Add a malformed argument test for capabilities
		//{
		//    label:         "Capability",
		//    typeSignature: "Capability<&Foo>",
		//    exportedValue: cadence.Capability{
		//        Path: cadence.Path{
		//            Domain:     "public",
		//            Identifier: "bar",
		//        },
		//        Address:    cadence.NewAddress([8]byte{0, 0, 0, 0, 0, 1, 0, 2}),
		//        BorrowType: "Foo",
		//    },
		//},

		// TODO: enable once https://github.com/onflow/cadence/issues/491 is fixed.
		//{
		//    label:         "Type",
		//    typeSignature: "Type",
		//    exportedValue: cadence.TypeValue{
		//        StaticType: "Foo",
		//    },
		//},

	}

	testArgumentPassing := func(test argumentPassingTest) {

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
                }`,
				test.typeSignature,
				returnSignature,
				returnStmt,
			)

			actual, err := executeTestScript(t, script, test.exportedValue)
			require.NoError(t, err)

			if !test.skipExport {
				assert.Equal(t, test.exportedValue, actual)
			}
		})
	}

	for _, testCase := range argumentPassingTests {
		testArgumentPassing(testCase)
	}
}

func TestRuntimeComplexStructArgumentPassing(t *testing.T) {

	t.Parallel()

	// Complex struct value
	complexStructValue := cadence.Struct{
		StructType: &cadence.StructType{
			Location:            TestLocation,
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
				{
					Identifier: "f",
					Type:       cadence.BoolType{},
				},
				{
					Identifier: "g",
					Type:       cadence.StoragePathType{},
				},
				{
					Identifier: "h",
					Type:       cadence.PublicPathType{},
				},
				{
					Identifier: "i",
					Type:       cadence.PrivatePathType{},
				},
				{
					Identifier: "j",
					Type:       cadence.AnyStructType{},
				},
			},
		},

		Fields: []cadence.Value{
			cadence.NewOptional(
				cadence.String("John"),
			),
			cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("name"),
					Value: cadence.String("Doe"),
				},
			}),
			cadence.NewArray([]cadence.Value{
				cadence.String("foo"),
				cadence.String("bar"),
			}),
			cadence.NewArray([]cadence.Value{
				cadence.String("foo"),
				cadence.String("bar"),
			}),
			cadence.NewAddress([8]byte{0, 0, 0, 0, 0, 1, 0, 2}),
			cadence.NewBool(true),
			cadence.Path{
				Domain:     "storage",
				Identifier: "foo",
			},
			cadence.Path{
				Domain:     "public",
				Identifier: "foo",
			},
			cadence.Path{
				Domain:     "private",
				Identifier: "foo",
			},
			cadence.String("foo"),
		},
	}

	script := fmt.Sprintf(
		`
          pub fun main(arg: %[1]s): %[1]s {

              if !arg.isInstance(Type<%[1]s>()) {
                  panic("Not a %[1]s value")
              }

              return arg
          }

          pub struct Foo {
              pub var a: String?
              pub var b: {String: String}
              pub var c: [String]
              pub var d: [String; 2]
              pub var e: Address
              pub var f: Bool
              pub var g: StoragePath
              pub var h: PublicPath
              pub var i: PrivatePath
              pub var j: AnyStruct

              init() {
                  self.a = "Hello"
                  self.b = {}
                  self.c = []
                  self.d = ["foo", "bar"]
                  self.e = 0x42
                  self.f = true
                  self.g = /storage/foo
                  self.h = /public/foo
                  self.i = /private/foo
                  self.j = nil
              }
          }
        `,
		"Foo",
	)

	actual, err := executeTestScript(t, script, complexStructValue)
	require.NoError(t, err)
	assert.Equal(t, complexStructValue, actual)

}

func TestRuntimeComplexStructWithAnyStructFields(t *testing.T) {

	t.Parallel()

	// Complex struct value
	complexStructValue := cadence.Struct{
		StructType: &cadence.StructType{
			Location:            TestLocation,
			QualifiedIdentifier: "Foo",
			Fields: []cadence.Field{
				{
					Identifier: "a",
					Type: cadence.OptionalType{
						Type: cadence.AnyStructType{},
					},
				},
				{
					Identifier: "b",
					Type: cadence.DictionaryType{
						KeyType:     cadence.StringType{},
						ElementType: cadence.AnyStructType{},
					},
				},
				{
					Identifier: "c",
					Type: cadence.VariableSizedArrayType{
						ElementType: cadence.AnyStructType{},
					},
				},
				{
					Identifier: "d",
					Type: cadence.ConstantSizedArrayType{
						ElementType: cadence.AnyStructType{},
						Size:        2,
					},
				},
				{
					Identifier: "e",
					Type:       cadence.AnyStructType{},
				},
			},
		},

		Fields: []cadence.Value{
			cadence.NewOptional(cadence.String("John")),
			cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("name"),
					Value: cadence.String("Doe"),
				},
			}),
			cadence.NewArray([]cadence.Value{
				cadence.String("foo"),
				cadence.String("bar"),
			}),
			cadence.NewArray([]cadence.Value{
				cadence.String("foo"),
				cadence.String("bar"),
			}),
			cadence.Path{
				Domain:     "storage",
				Identifier: "foo",
			},
		},
	}

	script := fmt.Sprintf(
		`
          pub fun main(arg: %[1]s): %[1]s {

              if !arg.isInstance(Type<%[1]s>()) {
                  panic("Not a %[1]s value")
              }

              return arg
          }

          pub struct Foo {
              pub var a: AnyStruct?
              pub var b: {String: AnyStruct}
              pub var c: [AnyStruct]
              pub var d: [AnyStruct; 2]
              pub var e: AnyStruct

              init() {
                  self.a = "Hello"
                  self.b = {}
                  self.c = []
                  self.d = ["foo", "bar"]
                  self.e = /storage/foo
              }
        }
        `,
		"Foo",
	)

	actual, err := executeTestScript(t, script, complexStructValue)
	require.NoError(t, err)
	assert.Equal(t, complexStructValue, actual)
}

func TestRuntimeMalformedArgumentPassing(t *testing.T) {

	t.Parallel()

	// Struct with wrong field type

	malformedStructType1 := &cadence.StructType{
		Location:            TestLocation,
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
			Location:            TestLocation,
			QualifiedIdentifier: "Foo",
			Fields: []cadence.Field{
				{
					Identifier: "nonExisting",
					Type:       cadence.StringType{},
				},
			},
		},
		Fields: []cadence.Value{
			cadence.String("John"),
		},
	}

	// Struct with nested malformed array value
	malformedStruct3 := cadence.Struct{
		StructType: &cadence.StructType{
			Location:            TestLocation,
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
			Location:            TestLocation,
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
					Key:   cadence.String("foo"),
					Value: malformedStruct1,
				},
			}),
		},
	}

	// Struct with nested array with mismatching element type
	malformedStruct5 := cadence.Struct{
		StructType: &cadence.StructType{
			Location:            TestLocation,
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
				cadence.String("mismatching value"),
			}),
		},
	}

	type argumentPassingTest struct {
		label           string
		typeSignature   string
		exportedValue   cadence.Value
		expectedErrType error
	}

	var argumentPassingTests = []argumentPassingTest{
		{
			label:           "Malformed Struct field type",
			typeSignature:   "Foo",
			exportedValue:   malformedStruct1,
			expectedErrType: &MalformedValueError{},
		},
		{
			label:           "Malformed Struct field name",
			typeSignature:   "Foo",
			exportedValue:   malformedStruct2,
			expectedErrType: &MalformedValueError{},
		},
		{
			label:           "Malformed AnyStruct",
			typeSignature:   "AnyStruct",
			exportedValue:   malformedStruct1,
			expectedErrType: &MalformedValueError{},
		},
		{
			label:           "Malformed nested struct array",
			typeSignature:   "Bar",
			exportedValue:   malformedStruct3,
			expectedErrType: &MalformedValueError{},
		},
		{
			label:           "Malformed nested struct dictionary",
			typeSignature:   "Baz",
			exportedValue:   malformedStruct4,
			expectedErrType: &MalformedValueError{},
		},
		{
			label:         "Array with malformed member",
			typeSignature: "[Foo]",
			exportedValue: cadence.NewArray([]cadence.Value{
				malformedStruct1,
			}),
			expectedErrType: &MalformedValueError{},
		},
		{
			label:         "Array with wrong size",
			typeSignature: "[String; 2]",
			exportedValue: cadence.NewArray([]cadence.Value{
				malformedStruct1,
			}),
			expectedErrType: &InvalidValueTypeError{},
		},
		{
			label:         "Nested array with mismatching element",
			typeSignature: "[[String]]",
			exportedValue: cadence.NewArray([]cadence.Value{
				cadence.NewArray([]cadence.Value{
					cadence.NewInt(5),
				}),
			}),
			expectedErrType: &InvalidValueTypeError{},
		},
		{
			label:           "Inner array with mismatching element",
			typeSignature:   "Bar",
			exportedValue:   malformedStruct5,
			expectedErrType: &MalformedValueError{},
		},
		{
			label:           "Malformed Optional",
			typeSignature:   "Foo?",
			exportedValue:   cadence.NewOptional(malformedStruct1),
			expectedErrType: &MalformedValueError{},
		},
		{
			label:         "Malformed dictionary",
			typeSignature: "{String: Foo}",
			exportedValue: cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("foo"),
					Value: malformedStruct1,
				},
			}),
			expectedErrType: &MalformedValueError{},
		},
	}

	testArgumentPassing := func(test argumentPassingTest) {

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

			_, err := executeTestScript(t, script, test.exportedValue)
			require.Error(t, err)

			require.IsType(t, Error{}, err)
			runtimeError := err.(Error)

			require.IsType(t, &InvalidEntryPointArgumentError{}, runtimeError.Err)
			argError := runtimeError.Err.(*InvalidEntryPointArgumentError)

			require.IsType(t, test.expectedErrType, argError.Err)
		})
	}

	for _, testCase := range argumentPassingTests {
		testArgumentPassing(testCase)
	}
}

func TestRuntimeImportExportArrayValue(t *testing.T) {

	t.Parallel()

	t.Run("export empty", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		value := interpreter.NewArrayValue(
			inter,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
		)

		actual, err := exportValueWithInterpreter(value, inter, seenReferences{})
		require.NoError(t, err)

		assert.Equal(t,
			cadence.NewArray([]cadence.Value{}),
			actual,
		)
	})

	t.Run("import empty", func(t *testing.T) {

		t.Parallel()

		value := cadence.NewArray([]cadence.Value{})

		inter := newTestInterpreter(t)

		actual, err := importValue(
			inter,
			value,
			&sema.VariableSizedType{
				Type: sema.UInt8Type,
			},
		)
		require.NoError(t, err)

		AssertValuesEqual(t,
			interpreter.NewArrayValue(
				inter,
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeUInt8,
				},
			),
			actual,
		)
	})

	t.Run("export non-empty", func(t *testing.T) {

		t.Parallel()

		value := interpreter.NewArrayValue(
			newTestInterpreter(t),
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			interpreter.NewIntValueFromInt64(42),
			interpreter.NewStringValue("foo"),
		)

		actual, err := exportValueWithInterpreter(value, nil, seenReferences{})
		require.NoError(t, err)

		assert.Equal(t,
			cadence.NewArray([]cadence.Value{
				cadence.NewInt(42),
				cadence.String("foo"),
			}),
			actual,
		)
	})

	t.Run("import non-empty", func(t *testing.T) {

		t.Parallel()

		value := cadence.NewArray([]cadence.Value{
			cadence.NewInt(42),
			cadence.String("foo"),
		})

		inter := newTestInterpreter(t)

		actual, err := importValue(
			inter,
			value,
			&sema.VariableSizedType{
				Type: sema.AnyStructType,
			},
		)
		require.NoError(t, err)

		AssertValuesEqual(t,
			interpreter.NewArrayValue(
				inter,
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeAnyStruct,
				},
				interpreter.NewIntValueFromInt64(42),
				interpreter.NewStringValue("foo"),
			),
			actual,
		)
	})

	t.Run("import nested array with broader expected type", func(t *testing.T) {

		t.Parallel()

		value := cadence.NewArray([]cadence.Value{
			cadence.NewArray([]cadence.Value{
				cadence.NewInt8(4),
				cadence.NewInt8(3),
			}),
			cadence.NewArray([]cadence.Value{
				cadence.NewInt8(42),
				cadence.NewInt8(54),
			}),
		})

		inter := newTestInterpreter(t)

		actual, err := importValue(
			inter,
			value,
			sema.AnyStructType,
		)
		require.NoError(t, err)

		AssertValuesEqual(t,
			interpreter.NewArrayValue(
				inter,
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeAnyStruct,
				},
				interpreter.NewArrayValue(
					inter,
					interpreter.VariableSizedStaticType{
						Type: interpreter.PrimitiveStaticTypeAnyStruct,
					},
					interpreter.Int8Value(4),
					interpreter.Int8Value(3),
				),
				interpreter.NewArrayValue(
					inter,
					interpreter.VariableSizedStaticType{
						Type: interpreter.PrimitiveStaticTypeAnyStruct,
					},
					interpreter.Int8Value(42),
					interpreter.Int8Value(54),
				),
			),
			actual,
		)
	})
}

func TestRuntimeImportExportDictionaryValue(t *testing.T) {

	t.Parallel()

	t.Run("export empty", func(t *testing.T) {

		t.Parallel()

		value := interpreter.NewDictionaryValue(
			newTestInterpreter(t),
			interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeString,
				ValueType: interpreter.PrimitiveStaticTypeInt,
			},
		)

		actual, err := exportValueWithInterpreter(value, nil, seenReferences{})
		require.NoError(t, err)

		assert.Equal(t,
			cadence.NewDictionary([]cadence.KeyValuePair{}),
			actual,
		)
	})

	t.Run("import empty", func(t *testing.T) {

		t.Parallel()

		value := cadence.NewDictionary([]cadence.KeyValuePair{})

		inter := newTestInterpreter(t)

		actual, err := importValue(
			inter,
			value,
			&sema.DictionaryType{
				KeyType:   sema.StringType,
				ValueType: sema.UInt8Type,
			},
		)
		require.NoError(t, err)

		AssertValuesEqual(t,
			interpreter.NewDictionaryValue(
				inter,
				interpreter.DictionaryStaticType{
					KeyType:   interpreter.PrimitiveStaticTypeString,
					ValueType: interpreter.PrimitiveStaticTypeUInt8,
				},
			),
			actual,
		)
	})

	t.Run("export non-empty", func(t *testing.T) {

		t.Parallel()

		value := interpreter.NewDictionaryValue(
			newTestInterpreter(t),
			interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeString,
				ValueType: interpreter.PrimitiveStaticTypeInt,
			},
			interpreter.NewStringValue("a"), interpreter.NewIntValueFromInt64(1),
			interpreter.NewStringValue("b"), interpreter.NewIntValueFromInt64(2),
		)

		actual, err := exportValueWithInterpreter(value, nil, seenReferences{})
		require.NoError(t, err)

		assert.Equal(t,
			cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("a"),
					Value: cadence.NewInt(1),
				},
				{
					Key:   cadence.String("b"),
					Value: cadence.NewInt(2),
				},
			}),
			actual,
		)
	})

	t.Run("import non-empty", func(t *testing.T) {

		t.Parallel()

		value := cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key:   cadence.String("a"),
				Value: cadence.NewInt(1),
			},
			{
				Key:   cadence.String("b"),
				Value: cadence.NewInt(2),
			},
		})

		inter := newTestInterpreter(t)

		actual, err := importValue(
			inter,
			value,
			&sema.DictionaryType{
				KeyType:   sema.StringType,
				ValueType: sema.IntType,
			},
		)
		require.NoError(t, err)

		AssertValuesEqual(t,
			interpreter.NewDictionaryValue(
				inter,
				interpreter.DictionaryStaticType{
					KeyType:   interpreter.PrimitiveStaticTypeString,
					ValueType: interpreter.PrimitiveStaticTypeInt,
				},
				interpreter.NewStringValue("a"), interpreter.NewIntValueFromInt64(1),
				interpreter.NewStringValue("b"), interpreter.NewIntValueFromInt64(2),
			),
			actual,
		)
	})

	t.Run("import nested dictionary with broader expected type", func(t *testing.T) {

		t.Parallel()

		value := cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key: cadence.String("a"),
				Value: cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.NewInt8(1),
						Value: cadence.NewInt(100),
					},
					{
						Key:   cadence.NewInt8(2),
						Value: cadence.String("hello"),
					},
				}),
			},
			{
				Key: cadence.String("b"),
				Value: cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.NewInt8(1),
						Value: cadence.String("foo"),
					},
					{
						Key:   cadence.NewInt(2),
						Value: cadence.NewInt(50),
					},
				}),
			},
		})

		inter := newTestInterpreter(t)

		actual, err := importValue(
			inter,
			value,
			sema.AnyStructType,
		)
		require.NoError(t, err)

		AssertValuesEqual(t,
			interpreter.NewDictionaryValue(
				inter,
				interpreter.DictionaryStaticType{
					KeyType:   interpreter.PrimitiveStaticTypeString,
					ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
				},

				interpreter.NewStringValue("a"),
				interpreter.NewDictionaryValue(
					inter,
					interpreter.DictionaryStaticType{
						KeyType:   interpreter.PrimitiveStaticTypeNumber,
						ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
					},
					interpreter.Int8Value(1), interpreter.NewIntValueFromInt64(100),
					interpreter.Int8Value(2), interpreter.NewStringValue("hello"),
				),

				interpreter.NewStringValue("b"),
				interpreter.NewDictionaryValue(
					inter,
					interpreter.DictionaryStaticType{
						KeyType:   interpreter.PrimitiveStaticTypeNumber,
						ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
					},
					interpreter.Int8Value(1), interpreter.NewStringValue("foo"),
					interpreter.NewIntValueFromInt64(2), interpreter.NewIntValueFromInt64(50),
				),
			),
			actual,
		)
	})

	t.Run("import dictionary with heterogeneous keys", func(t *testing.T) {
		t.Parallel()

		script :=
			`pub fun main(arg: Foo) {
            }

            pub struct Foo {
                pub var a: AnyStruct

                init() {
                    self.a = nil
                }
            }`

		// Struct with nested malformed dictionary value
		malformedStruct := cadence.Struct{
			StructType: &cadence.StructType{
				Location:            TestLocation,
				QualifiedIdentifier: "Foo",
				Fields: []cadence.Field{
					{
						Identifier: "a",
						Type:       cadence.AnyStructType{},
					},
				},
			},
			Fields: []cadence.Value{
				cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.String("foo"),
						Value: cadence.String("value1"),
					},
					{
						Key:   cadence.NewInt(5),
						Value: cadence.String("value2"),
					},
				}),
			},
		}

		_, err := executeTestScript(t, script, malformedStruct)
		require.Error(t, err)

		var argErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &argErr)

		assert.Contains(t, argErr.Error(), "cannot import dictionary: keys does not belong to the same type")
	})

	t.Run("nested dictionary with mismatching element", func(t *testing.T) {
		t.Parallel()

		script :=
			`pub fun main(arg: {String: {String: String}}) {
            }
            `

		dictionary := cadence.NewDictionary(
			[]cadence.KeyValuePair{
				{
					Key: cadence.String("hello"),
					Value: cadence.NewDictionary(
						[]cadence.KeyValuePair{
							{
								Key:   cadence.String("hello"),
								Value: cadence.NewInt(6),
							},
						},
					),
				},
			},
		)

		_, err := executeTestScript(t, script, dictionary)
		require.Error(t, err)

		var argErr interpreter.ContainerMutationError
		require.ErrorAs(t, err, &argErr)
	})
}

func TestRuntimeStringValueImport(t *testing.T) {

	t.Parallel()

	t.Run("non-utf8", func(t *testing.T) {

		t.Parallel()

		nonUTF8String := "\xbd\xb2\x3d\xbc\x20\xe2"
		require.False(t, utf8.ValidString(nonUTF8String))

		// Avoid using the `NewString()` constructor to skip the validation
		stringValue := cadence.String(nonUTF8String)

		script := `
            pub fun main(s: String) {
                log(s)
            }
        `

		encodedArg, err := json.Encode(stringValue)
		require.NoError(t, err)

		rt := NewInterpreterRuntime()

		validated := false

		runtimeInterface := &testRuntimeInterface{
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(b)
			},
			log: func(s string) {
				assert.True(t, utf8.ValidString(s))
				validated = true
			},
		}

		_, err = rt.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{encodedArg},
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)

		require.NoError(t, err)

		assert.True(t, validated)
	})
}

func TestRuntimePublicKeyImport(t *testing.T) {

	t.Parallel()

	executeScript := func(
		t *testing.T,
		script string,
		arg cadence.Value,
		runtimeInterface Interface,
	) (cadence.Value, error) {

		encodedArg, err := json.Encode(arg)
		require.NoError(t, err)

		rt := NewInterpreterRuntime()

		return rt.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{encodedArg},
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)
	}

	publicKeyBytes := cadence.NewArray([]cadence.Value{
		cadence.NewUInt8(1),
		cadence.NewUInt8(2),
	})

	t.Run("Test IsValid", func(t *testing.T) {
		t.Parallel()

		testPublicKeyImport := func(userSetValidity, publicKeyActualValidity bool) {
			t.Run(
				fmt.Sprintf("UserSet(%v)|Actual(%v)", userSetValidity, publicKeyActualValidity),
				func(t *testing.T) {

					t.Parallel()

					script := `
                        pub fun main(key: PublicKey): Bool {
                            return key.isValid
                        }
                    `

					publicKey := cadence.NewStruct(
						[]cadence.Value{
							// PublicKey bytes
							publicKeyBytes,

							// Sign algorithm
							cadence.NewEnum(
								[]cadence.Value{
									cadence.NewUInt8(0),
								},
							).WithType(SignAlgoType),

							// isValid
							cadence.NewBool(userSetValidity),
						},
					).WithType(PublicKeyType)

					publicKeyValidated := false

					storage := newTestStorage(nil, nil)

					runtimeInterface := &testRuntimeInterface{
						storage: storage,
						decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
							return json.Decode(b)
						},

						validatePublicKey: func(publicKey *PublicKey) (bool, error) {
							publicKeyValidated = true
							return publicKeyActualValidity, nil
						},
					}

					actual, err := executeScript(t, script, publicKey, runtimeInterface)
					require.NoError(t, err)

					// Check whether 'isValid' field returns the actual validity of
					// the public key, but not the one set by the user.
					assert.True(t, publicKeyValidated)
					assert.Equal(t, actual, cadence.NewBool(publicKeyActualValidity))
				},
			)
		}

		testPublicKeyImport(true, true)
		testPublicKeyImport(true, false)
		testPublicKeyImport(false, true)
		testPublicKeyImport(false, false)
	})

	t.Run("Test Verify", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(key: PublicKey): Bool {
                return key.verify(
                    signature: [],
                    signedData: [],
                    domainSeparationTag: "",
                    hashAlgorithm: HashAlgorithm.SHA2_256
                )
            }
        `

		publicKey := cadence.NewStruct(
			[]cadence.Value{
				// PublicKey bytes
				publicKeyBytes,

				// Sign algorithm
				cadence.NewEnum(
					[]cadence.Value{
						cadence.NewUInt8(0),
					},
				).WithType(SignAlgoType),

				// isValid
				cadence.NewBool(true),
			},
		).WithType(PublicKeyType)

		verifyInvoked := false

		storage := newTestStorage(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(b)
			},
			verifySignature: func(
				signature []byte,
				tag string,
				signedData []byte,
				publicKey []byte,
				signatureAlgorithm SignatureAlgorithm,
				hashAlgorithm HashAlgorithm,
			) (bool, error) {
				verifyInvoked = true
				return true, nil
			},
		}

		actual, err := executeScript(t, script, publicKey, runtimeInterface)
		require.NoError(t, err)

		assert.True(t, verifyInvoked)
		assert.Equal(t, actual, cadence.NewBool(true))
	})

	t.Run("Invalid raw public key", func(t *testing.T) {
		script := `
            pub fun main(key: PublicKey) {
            }
        `

		publicKey := cadence.NewStruct(
			[]cadence.Value{
				// Invalid value for 'publicKey' field
				cadence.NewBool(true),

				cadence.NewEnum(
					[]cadence.Value{
						cadence.NewUInt8(0),
					},
				).WithType(SignAlgoType),

				cadence.NewBool(true),
			},
		).WithType(PublicKeyType)

		storage := newTestStorage(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(b)
			},
		}

		_, err := executeScript(t, script, publicKey, runtimeInterface)
		require.Error(t, err)

		var argErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &argErr)
	})

	t.Run("Invalid content in public key", func(t *testing.T) {
		script := `
            pub fun main(key: PublicKey) {
            }
        `

		publicKey := cadence.NewStruct(
			[]cadence.Value{
				// Invalid content for 'publicKey' field
				cadence.NewArray([]cadence.Value{
					cadence.String("1"),
					cadence.String("2"),
				}),

				cadence.NewEnum(
					[]cadence.Value{
						cadence.NewUInt8(0),
					},
				).WithType(SignAlgoType),

				cadence.NewBool(true),
			},
		).WithType(PublicKeyType)

		storage := newTestStorage(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(b)
			},
		}

		_, err := executeScript(t, script, publicKey, runtimeInterface)
		require.Error(t, err)

		var valueErr *MalformedValueError
		require.ErrorAs(t, err, &valueErr)

		assert.Equal(t, sema.PublicKeyType, valueErr.ExpectedType)
	})

	t.Run("Invalid sign algo", func(t *testing.T) {
		script := `
            pub fun main(key: PublicKey) {
            }
        `

		publicKey := cadence.NewStruct(
			[]cadence.Value{
				publicKeyBytes,

				// Invalid value for 'signatureAlgorithm' field
				cadence.NewBool(true),

				cadence.NewBool(true),
			},
		).WithType(PublicKeyType)

		storage := newTestStorage(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(b)
			},
		}

		_, err := executeScript(t, script, publicKey, runtimeInterface)
		require.Error(t, err)

		var argErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &argErr)
	})

	t.Run("Invalid sign algo fields", func(t *testing.T) {
		script := `
            pub fun main(key: PublicKey) {
            }
        `

		publicKey := cadence.NewStruct(
			[]cadence.Value{
				publicKeyBytes,

				// Invalid value for fields of 'signatureAlgorithm'
				cadence.NewEnum(
					[]cadence.Value{
						cadence.String("hello"),
					},
				).WithType(SignAlgoType),

				cadence.NewBool(true),
			},
		).WithType(PublicKeyType)

		storage := newTestStorage(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(b)
			},
		}

		_, err := executeScript(t, script, publicKey, runtimeInterface)
		require.Error(t, err)

		var valueErr *MalformedValueError
		require.ErrorAs(t, err, &valueErr)

		assert.Equal(t, sema.PublicKeyType, valueErr.ExpectedType)
	})

	t.Run("Extra field", func(t *testing.T) {
		script := `
            pub fun main(key: PublicKey) {
            }
        `

		jsonCdc := `
            {
                "type":"Struct",
                "value":{
                    "id":"PublicKey",
                    "fields":[
                        {
                            "name":"publicKey",
                            "value":{
                                "type":"Array",
                                "value":[
                                    {
                                        "type":"UInt8",
                                        "value":"1"
                                    },
                                    {
                                        "type":"UInt8",
                                        "value":"2"
                                    }
                                ]
                            }
                        },
                        {
                            "name":"signatureAlgorithm",
                            "value":{
                                "type":"Enum",
                                "value":{
                                    "id":"SignatureAlgorithm",
                                    "fields":[
                                        {
                                            "name":"rawValue",
                                            "value":{
                                                "type":"UInt8",
                                                "value":"0"
                                            }
                                        }
                                    ]
                                }
                            }
                        },
                        {
                            "name":"isValid",
                            "value":{
                            "type":"Bool",
                            "value":true
                            }
                        },
                        {
                            "name":"extraField",
                            "value":{
                            "type":"Bool",
                            "value":true
                            }
                        }
                    ]
                }
            }
        `

		rt := NewInterpreterRuntime()

		storage := newTestStorage(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(b)
			},
		}

		_, err := rt.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: [][]byte{
					[]byte(jsonCdc),
				},
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)
		require.Error(t, err)

		var argErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &argErr)
	})

	t.Run("Missing raw public key", func(t *testing.T) {
		script := `
            pub fun main(key: PublicKey): Bool {
                return key.isValid
            }
        `

		jsonCdc := `
            {
                "type":"Struct",
                "value":{
                    "id":"PublicKey",
                    "fields":[
                        {
                            "name":"signatureAlgorithm",
                            "value":{
                                "type":"Enum",
                                "value":{
                                    "id":"SignatureAlgorithm",
                                    "fields":[
                                        {
                                            "name":"rawValue",
                                            "value":{
                                                "type":"UInt8",
                                                "value":"0"
                                            }
                                        }
                                    ]
                                }
                            }
                        },
                        {
                            "name":"isValid",
                            "value":{
                            "type":"Bool",
                            "value":true
                            }
                        }
                    ]
                }
            }
        `

		rt := NewInterpreterRuntime()

		storage := newTestStorage(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(b)
			},
		}

		_, err := rt.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: [][]byte{
					[]byte(jsonCdc),
				},
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)

		var argErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &argErr)
	})

	t.Run("Missing isValid", func(t *testing.T) {
		script := `
            pub fun main(key: PublicKey): Bool {
                return key.isValid
            }
        `

		jsonCdc := `
            {
                "type":"Struct",
                "value":{
                    "id":"PublicKey",
                    "fields":[
                        {
                            "name":"publicKey",
                            "value":{
                                "type":"Array",
                                "value":[
                                    {
                                        "type":"UInt8",
                                        "value":"1"
                                    },
                                    {
                                        "type":"UInt8",
                                        "value":"2"
                                    }
                                ]
                            }
                        },
                        {
                            "name":"signatureAlgorithm",
                            "value":{
                                "type":"Enum",
                                "value":{
                                    "id":"SignatureAlgorithm",
                                    "fields":[
                                        {
                                            "name":"rawValue",
                                            "value":{
                                                "type":"UInt8",
                                                "value":"0"
                                            }
                                        }
                                    ]
                                }
                            }
                        }
                    ]
                }
            }
        `

		rt := NewInterpreterRuntime()

		publicKeyValidated := false

		storage := newTestStorage(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(b)
			},
			validatePublicKey: func(publicKey *PublicKey) (bool, error) {
				publicKeyValidated = true
				return true, nil
			},
		}

		value, err := rt.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: [][]byte{
					[]byte(jsonCdc),
				},
			},
			Context{
				Interface: runtimeInterface,
				Location:  TestLocation,
			},
		)

		require.NoError(t, err)
		assert.True(t, publicKeyValidated)
		assert.Equal(t, value, cadence.NewBool(true))
	})
}

func TestRuntimeImportExportComplex(t *testing.T) {

	t.Parallel()

	inter := newTestInterpreter(t)

	// Array

	semaArrayType := &sema.VariableSizedType{
		Type: sema.AnyStructType,
	}

	staticArrayType := interpreter.VariableSizedStaticType{
		Type: interpreter.PrimitiveStaticTypeAnyStruct,
	}

	externalArrayType := cadence.VariableSizedArrayType{
		ElementType: cadence.AnyStructType{},
	}

	internalArrayValue := interpreter.NewArrayValue(
		inter,
		staticArrayType,
		interpreter.NewIntValueFromInt64(42),
		interpreter.NewStringValue("foo"),
	)

	externalArrayValue := cadence.NewArray([]cadence.Value{
		cadence.NewInt(42),
		cadence.String("foo"),
	})

	// Dictionary

	semaDictionaryType := &sema.DictionaryType{
		KeyType:   sema.StringType,
		ValueType: semaArrayType,
	}

	staticDictionaryType := interpreter.DictionaryStaticType{
		KeyType:   interpreter.PrimitiveStaticTypeString,
		ValueType: staticArrayType,
	}

	externalDictionaryType := cadence.DictionaryType{
		KeyType:     cadence.StringType{},
		ElementType: externalArrayType,
	}

	internalDictionaryValue := interpreter.NewDictionaryValue(
		inter,
		staticDictionaryType,
		interpreter.NewStringValue("a"), internalArrayValue,
	)

	externalDictionaryValue := cadence.NewDictionary([]cadence.KeyValuePair{
		{
			Key:   cadence.String("a"),
			Value: externalArrayValue,
		},
	})

	// Composite

	semaCompositeType := &sema.CompositeType{
		Location:   TestLocation,
		Identifier: "Foo",
		Kind:       common.CompositeKindStructure,
		Members:    sema.NewStringMemberOrderedMap(),
		Fields:     []string{"dictionary"},
	}

	semaCompositeType.Members.Set(
		"dictionary",
		sema.NewPublicConstantFieldMember(
			semaCompositeType,
			"dictionary",
			semaDictionaryType,
			"",
		),
	)

	externalCompositeType := &cadence.StructType{
		Location:            TestLocation,
		QualifiedIdentifier: "Foo",
		Fields: []cadence.Field{
			{
				Identifier: "dictionary",
				Type:       externalDictionaryType,
			},
		},
	}

	internalCompositeValueFields := interpreter.NewStringValueOrderedMap()
	internalCompositeValueFields.Set("dictionary", internalDictionaryValue)

	internalCompositeValue := interpreter.NewCompositeValue(
		inter.Storage,
		TestLocation,
		"Foo",
		common.CompositeKindStructure,
		internalCompositeValueFields,
		common.Address{},
	)

	externalCompositeValue := cadence.Struct{
		StructType: externalCompositeType,
		Fields: []cadence.Value{
			externalDictionaryValue,
		},
	}

	t.Run("export", func(t *testing.T) {

		t.Parallel()

		program := interpreter.Program{
			Elaboration: sema.NewElaboration(),
		}

		storage := interpreter.NewInMemoryStorage()

		inter, err := interpreter.NewInterpreter(
			&program,
			TestLocation,
			interpreter.WithStorage(storage),
		)
		require.NoError(t, err)

		program.Elaboration.CompositeTypes[semaCompositeType.ID()] = semaCompositeType

		actual, err := exportValueWithInterpreter(internalCompositeValue, inter, seenReferences{})
		require.NoError(t, err)

		assert.Equal(t,
			externalCompositeValue,
			actual,
		)
	})

	t.Run("import", func(t *testing.T) {

		t.Parallel()

		program := interpreter.Program{
			Elaboration: sema.NewElaboration(),
		}

		storage := interpreter.NewInMemoryStorage()

		inter, err := interpreter.NewInterpreter(
			&program,
			TestLocation,
			interpreter.WithStorage(storage),
		)
		require.NoError(t, err)

		program.Elaboration.CompositeTypes[semaCompositeType.ID()] = semaCompositeType

		actual, err := importValue(
			inter,
			externalCompositeValue,
			semaCompositeType,
		)
		require.NoError(t, err)

		AssertValuesEqual(t,
			internalCompositeValue,
			actual,
		)
	})
}

func TestRuntimeStaticTypeAvailability(t *testing.T) {

	t.Parallel()

	t.Run("inner array", func(t *testing.T) {
		script := `
            pub fun main(arg: Foo) {
            }

            pub struct Foo {
                pub var a: AnyStruct

                init() {
                    self.a = nil
                }
            }
        `

		structValue := cadence.Struct{
			StructType: &cadence.StructType{
				Location:            TestLocation,
				QualifiedIdentifier: "Foo",
				Fields: []cadence.Field{
					{
						Identifier: "a",
						Type:       cadence.AnyStructType{},
					},
				},
			},

			Fields: []cadence.Value{
				cadence.NewArray([]cadence.Value{
					cadence.String("foo"),
					cadence.String("bar"),
				}),
			},
		}

		_, err := executeTestScript(t, script, structValue)
		require.NoError(t, err)
	})

	t.Run("inner dictionary", func(t *testing.T) {
		script := `
            pub fun main(arg: Foo) {
            }

            pub struct Foo {
                pub var a: AnyStruct

                init() {
                    self.a = nil
                }
            }
        `

		structValue := cadence.Struct{
			StructType: &cadence.StructType{
				Location:            TestLocation,
				QualifiedIdentifier: "Foo",
				Fields: []cadence.Field{
					{
						Identifier: "a",
						Type:       cadence.AnyStructType{},
					},
				},
			},

			Fields: []cadence.Value{
				cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.String("foo"),
						Value: cadence.String("bar"),
					},
				}),
			},
		}

		_, err := executeTestScript(t, script, structValue)
		require.NoError(t, err)
	})
}

func newTestInterpreter(t *testing.T) *interpreter.Interpreter {
	storage := interpreter.NewInMemoryStorage()

	inter, err := interpreter.NewInterpreter(
		nil,
		TestLocation,
		interpreter.WithStorage(storage),
	)
	require.NoError(t, err)

	return inter
}
