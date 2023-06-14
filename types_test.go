/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestType_ID(t *testing.T) {

	t.Parallel()

	type testCase struct {
		ty       Type
		expected string
	}

	stringerTests := []testCase{
		{AnyType{}, "Any"},
		{AnyStructType{}, "AnyStruct"},
		{AnyResourceType{}, "AnyResource"},
		{NumberType{}, "Number"},
		{SignedNumberType{}, "SignedNumber"},
		{IntegerType{}, "Integer"},
		{SignedIntegerType{}, "SignedInteger"},
		{FixedPointType{}, "FixedPoint"},
		{SignedFixedPointType{}, "SignedFixedPoint"},
		{UIntType{}, "UInt"},
		{UInt8Type{}, "UInt8"},
		{UInt16Type{}, "UInt16"},
		{UInt32Type{}, "UInt32"},
		{UInt64Type{}, "UInt64"},
		{UInt128Type{}, "UInt128"},
		{UInt256Type{}, "UInt256"},
		{IntType{}, "Int"},
		{Int8Type{}, "Int8"},
		{Int16Type{}, "Int16"},
		{Int32Type{}, "Int32"},
		{Int64Type{}, "Int64"},
		{Int128Type{}, "Int128"},
		{Int256Type{}, "Int256"},
		{Word8Type{}, "Word8"},
		{Word16Type{}, "Word16"},
		{Word32Type{}, "Word32"},
		{Word64Type{}, "Word64"},
		{Word128Type{}, "Word128"},
		{Word256Type{}, "Word256"},
		{UFix64Type{}, "UFix64"},
		{Fix64Type{}, "Fix64"},
		{VoidType{}, "Void"},
		{BoolType{}, "Bool"},
		{CharacterType{}, "Character"},
		{NeverType{}, "Never"},
		{StringType{}, "String"},
		{BytesType{}, "Bytes"},
		{AddressType{}, "Address"},
		{PathType{}, "Path"},
		{StoragePathType{}, "StoragePath"},
		{CapabilityPathType{}, "CapabilityPath"},
		{PublicPathType{}, "PublicPath"},
		{PrivatePathType{}, "PrivatePath"},
		{BlockType{}, "Block"},
		{MetaType{}, "Type"},
		{
			&CapabilityType{},
			"Capability",
		},
		{
			&CapabilityType{
				BorrowType: IntType{},
			},
			"Capability<Int>",
		},
		{
			&OptionalType{
				Type: StringType{},
			},
			"String?",
		},
		{
			&VariableSizedArrayType{
				ElementType: StringType{},
			},
			"[String]",
		},
		{
			&ConstantSizedArrayType{
				ElementType: StringType{},
				Size:        2,
			},
			"[String;2]",
		},
		{
			&DictionaryType{
				KeyType:     StringType{},
				ElementType: IntType{},
			},
			"{String:Int}",
		},
		{
			&StructType{
				QualifiedIdentifier: "Foo",
			},
			"Foo",
		},
		{
			&StructType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "Foo",
			},
			"S.test.Foo",
		},
		{
			&StructInterfaceType{
				QualifiedIdentifier: "FooI",
			},
			"FooI",
		},
		{
			&StructInterfaceType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooI",
			},
			"S.test.FooI",
		},
		{
			&ResourceType{
				QualifiedIdentifier: "Bar",
			},
			"Bar",
		},
		{
			&ResourceType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "Bar",
			},
			"S.test.Bar",
		},
		{
			&ResourceInterfaceType{
				QualifiedIdentifier: "BarI",
			},
			"BarI",
		},
		{
			&ResourceInterfaceType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "BarI",
			},
			"S.test.BarI",
		},
		{
			&RestrictedType{
				Type: &ResourceType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "Foo",
				},
				Restrictions: []Type{
					&ResourceInterfaceType{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "FooI",
					},
				},
			},
			"S.test.Foo{S.test.FooI}",
		},
		{
			&FunctionType{
				Parameters: []Parameter{
					{Type: IntType{}},
				},
				ReturnType: StringType{},
			},
			"((Int):String)",
		},
		{
			&EventType{
				QualifiedIdentifier: "Event",
			},
			"Event",
		},
		{
			&EventType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "Event",
			},
			"S.test.Event",
		},
		{
			&EnumType{
				QualifiedIdentifier: "Enum",
			},
			"Enum",
		},
		{
			&EnumType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "Enum",
			},
			"S.test.Enum",
		},
		{
			&ContractType{
				QualifiedIdentifier: "Contract",
			},
			"Contract",
		},
		{
			&ContractType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "Contract",
			},
			"S.test.Contract",
		},
		{
			&ContractInterfaceType{
				QualifiedIdentifier: "ContractI",
			},
			"ContractI",
		},
		{
			&ContractInterfaceType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "ContractI",
			},
			"S.test.ContractI",
		},
	}

	test := func(ty Type, expected string) {

		id := ty.ID()

		t.Run(id, func(t *testing.T) {

			assert.Equal(t, expected, id)
		})
	}

	for _, testCase := range stringerTests {
		test(testCase.ty, testCase.expected)
	}
}

func TestTypeEquality(t *testing.T) {

	t.Parallel()

	t.Run("simple types", func(t *testing.T) {

		t.Parallel()

		types := []Type{
			AnyType{},
			AnyStructType{},
			AnyResourceType{},
			NumberType{},
			SignedNumberType{},
			IntegerType{},
			SignedIntegerType{},
			FixedPointType{},
			SignedFixedPointType{},
			UIntType{},
			UInt8Type{},
			UInt16Type{},
			UInt32Type{},
			UInt64Type{},
			UInt128Type{},
			UInt256Type{},
			IntType{},
			Int8Type{},
			Int16Type{},
			Int32Type{},
			Int64Type{},
			Int128Type{},
			Int256Type{},
			Word8Type{},
			Word16Type{},
			Word32Type{},
			Word64Type{},
			Word128Type{},
			Word256Type{},
			UFix64Type{},
			Fix64Type{},
			VoidType{},
			BoolType{},
			CharacterType{},
			NeverType{},
			StringType{},
			BytesType{},
			AddressType{},
			PathType{},
			StoragePathType{},
			CapabilityPathType{},
			PublicPathType{},
			PrivatePathType{},
			BlockType{},
			MetaType{},
			AuthAccountType{},
			AuthAccountKeysType{},
			AuthAccountContractsType{},
			PublicAccountType{},
			PublicAccountKeysType{},
			PublicAccountContractsType{},
			AccountKeyType{},
			DeployedContractType{},
		}

		for i, source := range types {
			for j, target := range types {
				if i == j {
					assert.True(t, source.Equal(target))
				} else {
					assert.False(t, source.Equal(target))
				}
			}
		}
	})

	t.Run("typeId type", func(t *testing.T) {
		t.Parallel()

		t.Run("equal", func(t *testing.T) {
			t.Parallel()

			source := TypeID("Foo")
			target := TypeID("Foo")
			assert.True(t, source.Equal(target))
		})

		t.Run("not equal", func(t *testing.T) {
			t.Parallel()

			source := TypeID("Foo")
			target := TypeID("Bar")
			assert.False(t, source.Equal(target))
		})
	})

	t.Run("capability type", func(t *testing.T) {
		t.Parallel()

		t.Run("equal with borrow type", func(t *testing.T) {
			t.Parallel()

			source := &CapabilityType{
				BorrowType: IntType{},
			}
			target := &CapabilityType{
				BorrowType: IntType{},
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("equal without borrow type", func(t *testing.T) {
			t.Parallel()

			source := &CapabilityType{}
			target := &CapabilityType{}
			assert.True(t, source.Equal(target))
		})

		t.Run("not equal", func(t *testing.T) {
			t.Parallel()

			source := &CapabilityType{
				BorrowType: IntType{},
			}
			target := &CapabilityType{
				BorrowType: StringType{},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("source missing borrow type", func(t *testing.T) {
			t.Parallel()

			source := &CapabilityType{}
			target := &CapabilityType{
				BorrowType: StringType{},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("target missing borrow type", func(t *testing.T) {
			t.Parallel()

			source := &CapabilityType{
				BorrowType: IntType{},
			}
			target := &CapabilityType{}
			assert.False(t, source.Equal(target))
		})

		t.Run("different type", func(t *testing.T) {
			t.Parallel()

			source := &CapabilityType{
				BorrowType: IntType{},
			}
			target := AnyType{}
			assert.False(t, source.Equal(target))
		})
	})

	t.Run("optional type", func(t *testing.T) {
		t.Parallel()

		t.Run("equal", func(t *testing.T) {
			t.Parallel()

			source := &OptionalType{
				Type: IntType{},
			}
			target := &OptionalType{
				Type: IntType{},
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("not equal", func(t *testing.T) {
			t.Parallel()

			source := &OptionalType{
				Type: IntType{},
			}
			target := &OptionalType{
				Type: StringType{},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different type", func(t *testing.T) {
			t.Parallel()

			source := &OptionalType{
				Type: IntType{},
			}
			target := AnyType{}
			assert.False(t, source.Equal(target))
		})
	})

	t.Run("variable sized type", func(t *testing.T) {
		t.Parallel()

		t.Run("equal", func(t *testing.T) {
			t.Parallel()

			source := &VariableSizedArrayType{
				ElementType: IntType{},
			}
			target := &VariableSizedArrayType{
				ElementType: IntType{},
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("not equal", func(t *testing.T) {
			t.Parallel()

			source := &VariableSizedArrayType{
				ElementType: IntType{},
			}
			target := &VariableSizedArrayType{
				ElementType: StringType{},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different type", func(t *testing.T) {
			t.Parallel()

			source := &VariableSizedArrayType{
				ElementType: IntType{},
			}
			target := AnyType{}
			assert.False(t, source.Equal(target))
		})
	})

	t.Run("constant sized type", func(t *testing.T) {
		t.Parallel()

		t.Run("equal", func(t *testing.T) {
			t.Parallel()

			source := &ConstantSizedArrayType{
				ElementType: IntType{},
				Size:        3,
			}
			target := &ConstantSizedArrayType{
				ElementType: IntType{},
				Size:        3,
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("different inner types", func(t *testing.T) {
			t.Parallel()

			source := &ConstantSizedArrayType{
				ElementType: IntType{},
				Size:        3,
			}
			target := &ConstantSizedArrayType{
				ElementType: StringType{},
				Size:        3,
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different sizes", func(t *testing.T) {
			t.Parallel()

			source := &ConstantSizedArrayType{
				ElementType: IntType{},
				Size:        3,
			}
			target := &ConstantSizedArrayType{
				ElementType: IntType{},
				Size:        4,
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different type", func(t *testing.T) {
			t.Parallel()

			source := &ConstantSizedArrayType{
				ElementType: IntType{},
				Size:        3,
			}
			target := AnyType{}
			assert.False(t, source.Equal(target))
		})
	})

	t.Run("dictionary type", func(t *testing.T) {
		t.Parallel()

		t.Run("equal", func(t *testing.T) {
			t.Parallel()

			source := &DictionaryType{
				KeyType:     IntType{},
				ElementType: BoolType{},
			}
			target := &DictionaryType{
				KeyType:     IntType{},
				ElementType: BoolType{},
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("different key types", func(t *testing.T) {
			t.Parallel()

			source := &DictionaryType{
				KeyType:     IntType{},
				ElementType: BoolType{},
			}
			target := &DictionaryType{
				KeyType:     UIntType{},
				ElementType: BoolType{},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different element types", func(t *testing.T) {
			t.Parallel()

			source := &DictionaryType{
				KeyType:     IntType{},
				ElementType: BoolType{},
			}
			target := &DictionaryType{
				KeyType:     IntType{},
				ElementType: StringType{},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different key and element types", func(t *testing.T) {
			t.Parallel()

			source := &DictionaryType{
				KeyType:     IntType{},
				ElementType: BoolType{},
			}
			target := &DictionaryType{
				KeyType:     UIntType{},
				ElementType: StringType{},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different type", func(t *testing.T) {
			t.Parallel()

			source := &DictionaryType{
				KeyType:     IntType{},
				ElementType: BoolType{},
			}
			target := AnyType{}
			assert.False(t, source.Equal(target))
		})
	})

	t.Run("struct type", func(t *testing.T) {
		t.Parallel()

		t.Run("equal", func(t *testing.T) {
			t.Parallel()

			source := &StructType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &StructType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("different location name", func(t *testing.T) {
			t.Parallel()

			source := &StructType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &StructType{
				Location: common.AddressLocation{
					Name:    "Test",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different address", func(t *testing.T) {
			t.Parallel()

			source := &StructType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &StructType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0, 0, 0, 0, 0, 0, 0, 0x01},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different qualified identifier", func(t *testing.T) {
			t.Parallel()

			source := &StructType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &StructType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Baz",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different type", func(t *testing.T) {
			t.Parallel()

			source := &StructType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := AnyType{}
			assert.False(t, source.Equal(target))
		})
	})

	t.Run("struct interface type", func(t *testing.T) {
		t.Parallel()

		t.Run("equal", func(t *testing.T) {
			t.Parallel()

			source := &StructInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &StructInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("different location name", func(t *testing.T) {
			t.Parallel()

			source := &StructInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &StructInterfaceType{
				Location: common.AddressLocation{
					Name:    "Test",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different address", func(t *testing.T) {
			t.Parallel()

			source := &StructInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &StructInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0, 0, 0, 0, 0, 0, 0, 0x01},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different qualified identifier", func(t *testing.T) {
			t.Parallel()

			source := &StructInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &StructInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Baz",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different type", func(t *testing.T) {
			t.Parallel()

			source := &StructInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := AnyType{}
			assert.False(t, source.Equal(target))
		})
	})

	t.Run("resource type", func(t *testing.T) {
		t.Parallel()

		t.Run("equal", func(t *testing.T) {
			t.Parallel()

			source := &ResourceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &ResourceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("different location name", func(t *testing.T) {
			t.Parallel()

			source := &ResourceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &ResourceType{
				Location: common.AddressLocation{
					Name:    "Test",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different address", func(t *testing.T) {
			t.Parallel()

			source := &ResourceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &ResourceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0, 0, 0, 0, 0, 0, 0, 0x01},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different qualified identifier", func(t *testing.T) {
			t.Parallel()

			source := &ResourceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &ResourceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Baz",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different type", func(t *testing.T) {
			t.Parallel()

			source := &ResourceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := AnyType{}
			assert.False(t, source.Equal(target))
		})
	})

	t.Run("resource interface type", func(t *testing.T) {
		t.Parallel()

		t.Run("equal", func(t *testing.T) {
			t.Parallel()

			source := &ResourceInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &ResourceInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("different location name", func(t *testing.T) {
			t.Parallel()

			source := &ResourceInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &ResourceInterfaceType{
				Location: common.AddressLocation{
					Name:    "Test",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different address", func(t *testing.T) {
			t.Parallel()

			source := &ResourceInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &ResourceInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0, 0, 0, 0, 0, 0, 0, 0x01},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different qualified identifier", func(t *testing.T) {
			t.Parallel()

			source := &ResourceInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &ResourceInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Baz",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different type", func(t *testing.T) {
			t.Parallel()

			source := &ResourceInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := AnyType{}
			assert.False(t, source.Equal(target))
		})
	})

	t.Run("contract type", func(t *testing.T) {
		t.Parallel()

		t.Run("equal", func(t *testing.T) {
			t.Parallel()

			source := &ContractType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &ContractType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("different location name", func(t *testing.T) {
			t.Parallel()

			source := &ContractType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &ContractType{
				Location: common.AddressLocation{
					Name:    "Test",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different address", func(t *testing.T) {
			t.Parallel()

			source := &ContractType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &ContractType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0, 0, 0, 0, 0, 0, 0, 0x01},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different qualified identifier", func(t *testing.T) {
			t.Parallel()

			source := &ContractType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &ContractType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Baz",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different type", func(t *testing.T) {
			t.Parallel()

			source := &ContractType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := AnyType{}
			assert.False(t, source.Equal(target))
		})
	})

	t.Run("contract interface type", func(t *testing.T) {
		t.Parallel()

		t.Run("equal", func(t *testing.T) {
			t.Parallel()

			source := &ContractInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &ContractInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("different location name", func(t *testing.T) {
			t.Parallel()

			source := &ContractInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &ContractInterfaceType{
				Location: common.AddressLocation{
					Name:    "Test",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different address", func(t *testing.T) {
			t.Parallel()

			source := &ContractInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &ContractInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0, 0, 0, 0, 0, 0, 0, 0x01},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different qualified identifier", func(t *testing.T) {
			t.Parallel()

			source := &ContractInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &ContractInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Baz",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different type", func(t *testing.T) {
			t.Parallel()

			source := &ContractInterfaceType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := AnyType{}
			assert.False(t, source.Equal(target))
		})
	})

	t.Run("event type", func(t *testing.T) {
		t.Parallel()

		t.Run("equal", func(t *testing.T) {
			t.Parallel()

			source := &EventType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &EventType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("different location name", func(t *testing.T) {
			t.Parallel()

			source := &EventType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &EventType{
				Location: common.AddressLocation{
					Name:    "Test",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different address", func(t *testing.T) {
			t.Parallel()

			source := &EventType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &EventType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0, 0, 0, 0, 0, 0, 0, 0x01},
				},
				QualifiedIdentifier: "Bar",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different qualified identifier", func(t *testing.T) {
			t.Parallel()

			source := &EventType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := &EventType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Baz",
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different type", func(t *testing.T) {
			t.Parallel()

			source := &EventType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
			}
			target := AnyType{}
			assert.False(t, source.Equal(target))
		})
	})

	t.Run("function type", func(t *testing.T) {
		t.Parallel()

		t.Run("equal", func(t *testing.T) {
			t.Parallel()

			source := &FunctionType{
				ReturnType: StringType{},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
					{
						Type: BoolType{},
					},
				},
			}
			target := &FunctionType{
				ReturnType: StringType{},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
					{
						Type: BoolType{},
					},
				},
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("different return type", func(t *testing.T) {
			t.Parallel()

			source := &FunctionType{
				ReturnType: StringType{},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
				},
			}
			target := &FunctionType{
				ReturnType: BoolType{},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
				},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different param type", func(t *testing.T) {
			t.Parallel()

			source := &FunctionType{
				ReturnType: StringType{},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
				},
			}
			target := &FunctionType{
				ReturnType: StringType{},
				Parameters: []Parameter{
					{
						Type: StringType{},
					},
				},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different param type count", func(t *testing.T) {
			t.Parallel()

			source := &FunctionType{
				ReturnType: StringType{},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
				},
			}
			target := &FunctionType{
				ReturnType: StringType{},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
					{
						Type: StringType{},
					},
				},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different type param count", func(t *testing.T) {
			t.Parallel()

			source := &FunctionType{
				TypeParameters: []TypeParameter{
					{
						Name: "T",
					},
				},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
				},
				ReturnType: StringType{},
			}
			target := &FunctionType{
				TypeParameters: []TypeParameter{},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
				},
				ReturnType: StringType{},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different type param name", func(t *testing.T) {
			t.Parallel()

			source := &FunctionType{
				TypeParameters: []TypeParameter{
					{
						Name: "T",
					},
				},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
				},
				ReturnType: StringType{},
			}
			target := &FunctionType{
				TypeParameters: []TypeParameter{
					{
						Name: "U",
					},
				},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
				},
				ReturnType: StringType{},
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("different type param bound: nil, some", func(t *testing.T) {
			t.Parallel()

			source := &FunctionType{
				TypeParameters: []TypeParameter{
					{
						Name: "T",
					},
				},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
				},
				ReturnType: StringType{},
			}
			target := &FunctionType{
				TypeParameters: []TypeParameter{
					{
						Name:      "T",
						TypeBound: AnyStructType{},
					},
				},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
				},
				ReturnType: StringType{},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different type param bound: some, nil", func(t *testing.T) {
			t.Parallel()

			source := &FunctionType{
				TypeParameters: []TypeParameter{
					{
						Name:      "T",
						TypeBound: AnyStructType{},
					},
				},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
				},
				ReturnType: StringType{},
			}
			target := &FunctionType{
				TypeParameters: []TypeParameter{
					{
						Name: "T",
					},
				},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
				},
				ReturnType: StringType{},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different type param bounds", func(t *testing.T) {
			t.Parallel()

			source := &FunctionType{
				TypeParameters: []TypeParameter{
					{
						Name:      "T",
						TypeBound: AnyResourceType{},
					},
				},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
				},
				ReturnType: StringType{},
			}
			target := &FunctionType{
				TypeParameters: []TypeParameter{
					{
						Name:      "T",
						TypeBound: AnyStructType{},
					},
				},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
				},
				ReturnType: StringType{},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("same type param bounds", func(t *testing.T) {
			t.Parallel()

			source := &FunctionType{
				TypeParameters: []TypeParameter{
					{
						Name:      "T",
						TypeBound: AnyResourceType{},
					},
				},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
				},
				ReturnType: StringType{},
			}
			target := &FunctionType{
				TypeParameters: []TypeParameter{
					{
						Name:      "T",
						TypeBound: AnyResourceType{},
					},
				},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
				},
				ReturnType: StringType{},
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("different type", func(t *testing.T) {
			t.Parallel()

			source := &FunctionType{
				ReturnType: StringType{},
				Parameters: []Parameter{
					{
						Type: IntType{},
					},
				},
			}
			target := AnyType{}
			assert.False(t, source.Equal(target))
		})
	})

	t.Run("reference type", func(t *testing.T) {
		t.Parallel()

		t.Run("equal", func(t *testing.T) {
			t.Parallel()

			source := &ReferenceType{
				Type:       IntType{},
				Authorized: false,
			}
			target := &ReferenceType{
				Type:       IntType{},
				Authorized: false,
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("different referenced type", func(t *testing.T) {
			t.Parallel()

			source := &ReferenceType{
				Type:       IntType{},
				Authorized: false,
			}
			target := &ReferenceType{
				Type:       StringType{},
				Authorized: false,
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("auth vs non-auth", func(t *testing.T) {
			t.Parallel()

			source := &ReferenceType{
				Type:       IntType{},
				Authorized: false,
			}
			target := &ReferenceType{
				Type:       IntType{},
				Authorized: true,
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("non-auth vs auth", func(t *testing.T) {
			t.Parallel()

			source := &ReferenceType{
				Type:       IntType{},
				Authorized: true,
			}
			target := &ReferenceType{
				Type:       IntType{},
				Authorized: false,
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different type", func(t *testing.T) {
			t.Parallel()

			source := &ReferenceType{
				Type:       IntType{},
				Authorized: true,
			}
			target := AnyType{}
			assert.False(t, source.Equal(target))
		})
	})

	t.Run("restricted type", func(t *testing.T) {
		t.Parallel()

		t.Run("equal", func(t *testing.T) {
			t.Parallel()

			source := &RestrictedType{
				Type: IntType{},
				Restrictions: []Type{
					AnyType{},
					IntType{},
				},
			}
			target := &RestrictedType{
				Type: IntType{},
				Restrictions: []Type{
					AnyType{},
					IntType{},
				},
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("different restrictions order", func(t *testing.T) {
			t.Parallel()

			source := &RestrictedType{
				Type: IntType{},
				Restrictions: []Type{
					AnyType{},
					IntType{},
				},
			}
			target := &RestrictedType{
				Type: IntType{},
				Restrictions: []Type{
					IntType{},
					AnyType{},
				},
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("duplicate restrictions", func(t *testing.T) {
			t.Parallel()

			source := &RestrictedType{
				Type: IntType{},
				Restrictions: []Type{
					IntType{},
					AnyType{},
					IntType{},
				},
			}
			target := &RestrictedType{
				Type: IntType{},
				Restrictions: []Type{
					IntType{},
					AnyType{},
				},
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("different inner type", func(t *testing.T) {
			t.Parallel()

			source := &RestrictedType{
				Type: IntType{},
				Restrictions: []Type{
					AnyType{},
					IntType{},
				},
			}
			target := &RestrictedType{
				Type: StringType{},
				Restrictions: []Type{
					AnyType{},
					IntType{},
				},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different restrictions", func(t *testing.T) {
			t.Parallel()

			source := &RestrictedType{
				Type: IntType{},
				Restrictions: []Type{
					AnyType{},
					IntType{},
				},
			}
			target := &RestrictedType{
				Type: IntType{},
				Restrictions: []Type{
					AnyType{},
					StringType{},
				},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different restrictions length", func(t *testing.T) {
			t.Parallel()

			source := &RestrictedType{
				Type: IntType{},
				Restrictions: []Type{
					AnyType{},
				},
			}
			target := &RestrictedType{
				Type: IntType{},
				Restrictions: []Type{
					AnyType{},
					StringType{},
				},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different type", func(t *testing.T) {
			t.Parallel()

			source := &RestrictedType{
				Type: IntType{},
				Restrictions: []Type{
					AnyType{},
				},
			}
			target := AnyType{}
			assert.False(t, source.Equal(target))
		})
	})

	t.Run("enum type", func(t *testing.T) {
		t.Parallel()

		t.Run("equal", func(t *testing.T) {
			t.Parallel()

			source := &EnumType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
				RawType:             IntType{},
			}
			target := &EnumType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
				RawType:             IntType{},
			}
			assert.True(t, source.Equal(target))
		})

		t.Run("different location name", func(t *testing.T) {
			t.Parallel()

			source := &EnumType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
				RawType:             IntType{},
			}
			target := &EnumType{
				Location: common.AddressLocation{
					Name:    "Test",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
				RawType:             IntType{},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different address", func(t *testing.T) {
			t.Parallel()

			source := &EnumType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
				RawType:             IntType{},
			}
			target := &EnumType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0, 0, 0, 0, 0, 0, 0, 0x01},
				},
				QualifiedIdentifier: "Bar",
				RawType:             IntType{},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different qualified identifier", func(t *testing.T) {
			t.Parallel()

			source := &EnumType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
				RawType:             IntType{},
			}
			target := &EnumType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Baz",
				RawType:             IntType{},
			}
			assert.False(t, source.Equal(target))
		})

		t.Run("different type", func(t *testing.T) {
			t.Parallel()

			source := &EnumType{
				Location: common.AddressLocation{
					Name:    "Foo",
					Address: common.Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
				},
				QualifiedIdentifier: "Bar",
				RawType:             IntType{},
			}
			target := AnyType{}
			assert.False(t, source.Equal(target))
		})
	})

}

func TestDecodeFields(t *testing.T) {
	t.Parallel()

	simpleEvent := NewEvent(
		[]Value{
			NewInt(1),
			String("foo"),
			NewOptional(nil),
			NewOptional(NewInt(2)),
			NewDictionary([]KeyValuePair{
				{Key: String("k"), Value: NewInt(3)},
			}),
			NewDictionary([]KeyValuePair{
				{Key: String("k"), Value: NewOptional(NewInt(4))},
				{Key: String("nilK"), Value: NewOptional(nil)},
			}),
			NewDictionary([]KeyValuePair{
				{Key: String("k"), Value: NewInt(3)},
				{Key: String("k2"), Value: String("foo")},
			}),
			NewOptional(NewInt(2)),
			String("bar"),
			Array{ArrayType: NewVariableSizedArrayType(IntType{}), Values: []Value{NewInt(1), NewInt(2)}},
			Array{ArrayType: NewVariableSizedArrayType(&OptionalType{Type: IntType{}}), Values: []Value{
				NewOptional(NewInt(1)),
				NewOptional(NewInt(2)),
				NewOptional(nil),
			}},
		},
	).WithType(&EventType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: "SimpleEvent",
		Fields: []Field{
			{
				Identifier: "intField",
				Type:       IntType{},
			},
			{
				Identifier: "stringField",
				Type:       StringType{},
			},
			{
				Identifier: "nilOptionalIntField",
				Type:       &OptionalType{Type: IntType{}},
			},
			{
				Identifier: "optionalIntField",
				Type:       &OptionalType{Type: IntType{}},
			},
			{
				Identifier: "dictField",
				Type:       &DictionaryType{KeyType: StringType{}, ElementType: IntType{}},
			},
			{
				Identifier: "dictOptionalField",
				Type:       &OptionalType{Type: &DictionaryType{KeyType: StringType{}, ElementType: &OptionalType{Type: IntType{}}}},
			},
			{
				Identifier: "dictAnyStructField",
				Type:       &DictionaryType{KeyType: StringType{}, ElementType: AnyStructType{}},
			},
			{
				Identifier: "optionalAnyStructField",
				Type:       &OptionalType{Type: AnyStructType{}},
			},
			{
				Identifier: "anyStructField",
				Type:       AnyStructType{},
			},
			{
				Identifier: "variableArrayIntField",
				Type:       NewVariableSizedArrayType(IntType{}),
			},
			{
				Identifier: "variableArrayOptionalIntField",
				Type:       NewVariableSizedArrayType(&OptionalType{Type: IntType{}}),
			},
		},
	})

	type eventStruct struct {
		Int                   Int                    `cadence:"intField"`
		String                String                 `cadence:"stringField"`
		NilOptionalInt        *Int                   `cadence:"nilOptionalIntField"`
		OptionalInt           *Int                   `cadence:"optionalIntField"`
		DictAnyStruct         map[String]interface{} `cadence:"dictAnyStructField"`
		Dict                  map[String]Int         `cadence:"dictField"`
		DictOptional          map[String]*Int        `cadence:"dictOptionalField"`
		OptionalAnyStruct     *interface{}           `cadence:"optionalAnyStructField"`
		AnyStructString       interface{}            `cadence:"anyStructField"`
		ArrayInt              []Int                  `cadence:"variableArrayIntField"`
		VariableArrayOptional []*Int                 `cadence:"variableArrayOptionalIntField"`
		NonCadenceField       Int
	}

	evt := eventStruct{}
	err := DecodeFields(simpleEvent, &evt)
	require.NoError(t, err)

	int1 := NewInt(1)
	int2 := NewInt(2)
	int4 := NewInt(4)

	assert.Nil(t, evt.NilOptionalInt)

	require.NotNil(t, evt.OptionalInt)
	assert.EqualValues(t, int2, *evt.OptionalInt)

	assert.Equal(t, Int{}, evt.NonCadenceField)

	assert.EqualValues(t, map[String]Int{"k": NewInt(3)}, evt.Dict)

	assert.EqualValues(t, map[String]*Int{"k": &int4}, evt.DictOptional)

	assert.EqualValues(t, map[String]interface{}{
		"k":  NewInt(3),
		"k2": String("foo"),
	}, evt.DictAnyStruct)

	evtOptionalAnyStruct := evt.OptionalAnyStruct
	require.NotNil(t, evtOptionalAnyStruct)
	assert.EqualValues(t, int2, *evtOptionalAnyStruct)

	assert.Equal(t, String("bar"), evt.AnyStructString)

	assert.EqualValues(t, []Int{int1, int2}, evt.ArrayInt)

	assert.EqualValues(t, []*Int{&int1, &int2, nil}, evt.VariableArrayOptional)

	type ErrCases struct {
		Struct      interface{}
		ExpectedErr string
		Description string
	}

	errCases := []ErrCases{
		{Struct: struct {
			A Int `cadence:"intField"`
		}{},
			ExpectedErr: "s must be a pointer to a struct",
			Description: "should err when mapping to non-pointer",
		},
		{Struct: &struct {
			A String `cadence:"intField"`
		}{},
			ExpectedErr: "cannot convert cadence field intField of type Int to struct field A of type cadence.String",
			Description: "should err when mapping to invalid type",
		},
		{Struct: &struct {
			a Int `cadence:"intField"` // nolint: unused
		}{},
			ExpectedErr: "cannot set field a",
			Description: "should err when mapping to private field",
		},
		{Struct: &struct {
			A Int `cadence:"notFoundField"`
		}{},
			ExpectedErr: "notFoundField field not found",
			Description: "should err when mapping to non-existing field",
		},
		{Struct: &struct {
			O *String `cadence:"optionalIntField"`
		}{},
			ExpectedErr: "cannot decode optional field O: cannot set field: expected cadence.String, got cadence.Int",
			Description: "should err when mapping to optional field with wrong type",
		},
		{Struct: &struct {
			DOptional map[*String]*Int `cadence:"dictOptionalField"`
		}{},
			ExpectedErr: "cannot decode dictionary field DOptional: map key cannot be a pointer (optional) type",
			Description: "should err when mapping to dictionary field with ptr key type",
		},
		{Struct: &struct {
			D map[String]String `cadence:"dictField"`
		}{},
			ExpectedErr: "cannot decode dictionary field D: map value type mismatch: expected cadence.String, got cadence.Int",
			Description: "should err when mapping to dictionary field with wrong value type",
		},
		{Struct: &struct {
			A []String `cadence:"intField"`
		}{},
			ExpectedErr: "cannot decode slice field A: field is not an array",
			Description: "should err when mapping to array field with wrong type",
		},
		{Struct: &struct {
			A []String `cadence:"variableArrayIntField"`
		}{},
			ExpectedErr: "cannot decode slice field A: array element type mismatch at index 0: expected cadence.String, got cadence.Int",
			Description: "should err when mapping to array field with wrong element type",
		},
		{Struct: &struct {
			A []*String `cadence:"variableArrayOptionalIntField"`
		}{},
			ExpectedErr: "cannot decode slice field A: error decoding array element optional: cannot set field: expected cadence.String, got cadence.Int",
			Description: "should err when mapping to array field with wrong type",
		},
		{Struct: &struct {
			A map[Int]Int `cadence:"dictField"`
		}{},
			ExpectedErr: "cannot decode dictionary field A: map key type mismatch: expected cadence.Int, got cadence.String",
			Description: "should err when mapping to map field with mismatching key type",
		},
		{Struct: &struct {
			A map[String]*String `cadence:"dictOptionalField"`
		}{},
			ExpectedErr: "cannot decode dictionary field A: cannot decode optional map value for key \"k\": cannot set field: expected cadence.String, got cadence.Int",
			Description: "should err when mapping to map field with mismatching value type",
		},
		{Struct: &struct {
			A map[String]Int `cadence:"intField"`
		}{},
			ExpectedErr: "cannot decode dictionary field A: field is not a dictionary",
			Description: "should err when mapping to map with mismatching field type",
		},
		{Struct: &struct {
			A *Int `cadence:"intField"`
		}{},
			ExpectedErr: "cannot decode optional field A: field is not an optional",
			Description: "should err when mapping to optional field with mismatching type",
		},
	}
	for _, errCase := range errCases {
		t.Run(errCase.Description, func(t *testing.T) {
			//t.Parallel()
			err := DecodeFields(simpleEvent, errCase.Struct)
			assert.Equal(t, errCase.ExpectedErr, err.Error())
		})
	}
}
