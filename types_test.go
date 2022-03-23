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

package cadence

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
			CapabilityType{},
			"Capability",
		},
		{
			CapabilityType{
				BorrowType: IntType{},
			},
			"Capability<Int>",
		},
		{
			OptionalType{
				Type: StringType{},
			},
			"String?",
		},
		{
			VariableSizedArrayType{
				ElementType: StringType{},
			},
			"[String]",
		},
		{
			ConstantSizedArrayType{
				ElementType: StringType{},
				Size:        2,
			},
			"[String;2]",
		},
		{
			DictionaryType{
				KeyType:     StringType{},
				ElementType: IntType{},
			},
			"{String:Int}",
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
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooI",
			},
			"S.test.FooI",
		},
		{
			&ResourceType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "Foo",
			},
			"S.test.Foo",
		},
		{
			&ResourceInterfaceType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooI",
			},
			"S.test.FooI",
		},
		{
			RestrictedType{}.WithID("S.test.Foo{S.test.FooI}"),
			"S.test.Foo{S.test.FooI}",
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
