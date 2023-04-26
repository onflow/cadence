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
	"fmt"
	"math/big"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

type valueTestCase struct {
	value        Value
	string       string
	exampleType  Type
	expectedType Type
	withType     func(value Value, ty Type) Value
	noType       bool
}

func newValueTestCases() map[string]valueTestCase {
	ufix64, _ := NewUFix64("64.01")
	fix64, _ := NewFix64("-32.11")

	testFunctionType := NewFunctionType(
		nil,
		[]Parameter{
			{
				Type: StringType{},
			},
		},
		UInt8Type{},
	)

	return map[string]valueTestCase{
		"UInt": {
			value:        NewUInt(10),
			string:       "10",
			expectedType: UIntType{},
		},
		"UInt8": {
			value:        NewUInt8(8),
			string:       "8",
			expectedType: UInt8Type{},
		},
		"UInt16": {
			value:        NewUInt16(16),
			string:       "16",
			expectedType: UInt16Type{},
		},
		"UInt32": {
			value:        NewUInt32(32),
			string:       "32",
			expectedType: UInt32Type{},
		},
		"UInt64": {
			value:        NewUInt64(64),
			string:       "64",
			expectedType: UInt64Type{},
		},
		"UInt128": {
			value:        NewUInt128(128),
			string:       "128",
			expectedType: UInt128Type{},
		},
		"UInt256": {
			value:        NewUInt256(256),
			string:       "256",
			expectedType: UInt256Type{},
		},
		"Int": {
			value:        NewInt(1000000),
			string:       "1000000",
			expectedType: IntType{},
		},
		"Int8": {
			value:        NewInt8(-8),
			string:       "-8",
			expectedType: Int8Type{},
		},
		"Int16": {
			value:        NewInt16(-16),
			string:       "-16",
			expectedType: Int16Type{},
		},
		"Int32": {
			value:        NewInt32(-32),
			string:       "-32",
			expectedType: Int32Type{},
		},
		"Int64": {
			value:        NewInt64(-64),
			string:       "-64",
			expectedType: Int64Type{},
		},
		"Int128": {
			value:        NewInt128(-128),
			string:       "-128",
			expectedType: Int128Type{},
		},
		"Int256": {
			value:        NewInt256(-256),
			string:       "-256",
			expectedType: Int256Type{},
		},
		"Word8": {
			value:        NewWord8(8),
			string:       "8",
			expectedType: Word8Type{},
		},
		"Word16": {
			value:        NewWord16(16),
			string:       "16",
			expectedType: Word16Type{},
		},
		"Word32": {
			value:        NewWord32(32),
			string:       "32",
			expectedType: Word32Type{},
		},
		"Word64": {
			value:        NewWord64(64),
			string:       "64",
			expectedType: Word64Type{},
		},
		"UFix64": {
			value:        ufix64,
			string:       "64.01000000",
			expectedType: UFix64Type{},
		},
		"Fix64": {
			value:        fix64,
			string:       "-32.11000000",
			expectedType: Fix64Type{},
		},
		"Void": {
			value:        NewVoid(),
			string:       "()",
			expectedType: VoidType{},
		},
		"Bool": {
			value:        NewBool(true),
			string:       "true",
			expectedType: BoolType{},
		},
		"some": {
			value:        NewOptional(ufix64),
			string:       "64.01000000",
			expectedType: NewOptionalType(UFix64Type{}),
		},
		"nil": {
			value:        NewOptional(nil),
			string:       "nil",
			expectedType: NewOptionalType(NeverType{}),
		},
		"String": {
			value:        String("Flow ridah!"),
			string:       "\"Flow ridah!\"",
			expectedType: StringType{},
		},
		"Character": {
			value:        Character("✌️"),
			string:       "\"\\u{270c}\\u{fe0f}\"",
			expectedType: CharacterType{},
		},
		"Array": {
			value: NewArray([]Value{
				NewInt(10),
				String("TEST"),
			}),
			exampleType: NewConstantSizedArrayType(2, AnyType{}),
			withType: func(value Value, ty Type) Value {
				return value.(Array).WithType(ty.(ArrayType))
			},
			string: "[10, \"TEST\"]",
		},
		"Dictionary": {
			value: NewDictionary([]KeyValuePair{
				{
					Key:   String("key"),
					Value: String("value"),
				},
			}),
			exampleType: NewDictionaryType(StringType{}, StringType{}),
			withType: func(value Value, ty Type) Value {
				return value.(Dictionary).WithType(ty.(*DictionaryType))
			},
			string: "{\"key\": \"value\"}",
		},
		"Bytes": {
			value:        NewBytes([]byte{0x1, 0x2}),
			string:       "[0x1, 0x2]",
			expectedType: BytesType{},
		},
		"Address": {
			value:        NewAddress([8]byte{0, 0, 0, 0, 0, 0, 0, 1}),
			string:       "0x0000000000000001",
			expectedType: AddressType{},
		},
		"struct": {
			value: NewStruct([]Value{String("bar")}),
			exampleType: &StructType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooStruct",
				Fields: []Field{
					{
						Identifier: "y",
						Type:       StringType{},
					},
				},
			},
			withType: func(value Value, ty Type) Value {
				return value.(Struct).WithType(ty.(*StructType))
			},
			string: "S.test.FooStruct(y: \"bar\")",
		},
		"resource": {
			value: NewResource([]Value{NewInt(1)}),
			exampleType: &ResourceType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooResource",
				Fields: []Field{
					{
						Identifier: "bar",
						Type:       IntType{},
					},
				},
			},
			withType: func(value Value, ty Type) Value {
				return value.(Resource).WithType(ty.(*ResourceType))
			},
			string: "S.test.FooResource(bar: 1)",
		},
		"event": {
			value: NewEvent(
				[]Value{
					NewInt(1),
					String("foo"),
				},
			),
			exampleType: &EventType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooEvent",
				Fields: []Field{
					{
						Identifier: "a",
						Type:       IntType{},
					},
					{
						Identifier: "b",
						Type:       StringType{},
					},
				},
			},
			withType: func(value Value, ty Type) Value {
				return value.(Event).WithType(ty.(*EventType))
			},
			string: "S.test.FooEvent(a: 1, b: \"foo\")",
		},
		"contract": {
			value: NewContract([]Value{String("bar")}),
			exampleType: &ContractType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooContract",
				Fields: []Field{
					{
						Identifier: "y",
						Type:       StringType{},
					},
				},
			},
			withType: func(value Value, ty Type) Value {
				return value.(Contract).WithType(ty.(*ContractType))
			},
			string: "S.test.FooContract(y: \"bar\")",
		},
		"enum": {
			value: NewEnum([]Value{UInt8(1)}),
			exampleType: &EnumType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooEnum",
				Fields: []Field{
					{
						Identifier: sema.EnumRawValueFieldName,
						Type:       UInt8Type{},
					},
				},
			},
			withType: func(value Value, ty Type) Value {
				return value.(Enum).WithType(ty.(*EnumType))
			},
			string: "S.test.FooEnum(rawValue: 1)",
		},
		"attachment": {
			value: NewAttachment([]Value{NewInt(1)}),
			exampleType: &AttachmentType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooAttachment",
				Fields: []Field{
					{
						Identifier: "bar",
						Type:       IntType{},
					},
				},
			},
			withType: func(value Value, ty Type) Value {
				return value.(Attachment).WithType(ty.(*AttachmentType))
			},
			string: "S.test.FooAttachment(bar: 1)",
		},
		"PathLink": {
			value: NewPathLink(
				Path{
					Domain:     common.PathDomainStorage,
					Identifier: "foo",
				},
				"Int",
			),
			string: "PathLink<Int>(/storage/foo)",
			noType: true,
		},
		"AccountLink": {
			value:  NewAccountLink(),
			string: "AccountLink()",
			noType: true,
		},
		"StoragePath": {
			value: Path{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			expectedType: TheStoragePathType,
			string:       "/storage/foo",
		},
		"PrivatePath": {
			value: Path{
				Domain:     common.PathDomainPrivate,
				Identifier: "foo",
			},
			expectedType: ThePrivatePathType,
			string:       "/private/foo",
		},
		"PublicPath": {
			value: Path{
				Domain:     common.PathDomainPublic,
				Identifier: "foo",
			},
			expectedType: ThePublicPathType,
			string:       "/public/foo",
		},
		"Type": {
			value:        TypeValue{StaticType: IntType{}},
			expectedType: NewMetaType(),
			string:       "Type<Int>()",
		},
		"Capability (Path)": {
			value: NewPathCapability(
				BytesToAddress([]byte{1, 2, 3, 4, 5}),
				Path{
					Domain:     common.PathDomainPublic,
					Identifier: "foo",
				},
				IntType{},
			),
			expectedType: NewCapabilityType(IntType{}),
			string:       "Capability<Int>(address: 0x0000000102030405, path: /public/foo)",
		},
		"Capability (Path, no borrow type)": {
			value: NewPathCapability(
				BytesToAddress([]byte{1, 2, 3, 4, 5}),
				Path{
					Domain:     common.PathDomainPublic,
					Identifier: "foo",
				},
				nil,
			),
			expectedType: NewCapabilityType(nil),
			string:       "Capability(address: 0x0000000102030405, path: /public/foo)",
		},
		"Capability (ID)": {
			value: NewIDCapability(
				3,
				BytesToAddress([]byte{1, 2, 3, 4, 5}),
				IntType{},
			),
			expectedType: NewCapabilityType(IntType{}),
			string:       "Capability<Int>(address: 0x0000000102030405, id: 3)",
		},
		"Function": {
			value: NewFunction(
				testFunctionType,
			),
			expectedType: testFunctionType,
			string:       "Function(...)",
		},
	}
}

func TestValue_String(t *testing.T) {

	t.Parallel()

	test := func(name string, testCase valueTestCase) {

		t.Run(name, func(t *testing.T) {

			t.Parallel()

			withType := testCase.withType
			if withType != nil {
				testCase.value = withType(testCase.value, testCase.exampleType)
			}

			assert.Equal(t,
				testCase.string,
				testCase.value.String(),
			)
		})
	}

	for name, testCase := range newValueTestCases() {
		test(name, testCase)
	}
}

func TestNumberValue_ToBigEndianBytes(t *testing.T) {

	t.Parallel()

	typeTests := map[string]map[NumberValue][]byte{
		// Int*
		"Int": {
			NewInt(0):                  {0},
			NewInt(42):                 {42},
			NewInt(127):                {127},
			NewInt(128):                {0, 128},
			NewInt(200):                {0, 200},
			NewInt(-1):                 {255},
			NewInt(-200):               {255, 56},
			NewInt(-10000000000000000): {220, 121, 13, 144, 63, 0, 0},
		},
		"Int8": {
			NewInt8(0):    {0},
			NewInt8(42):   {42},
			NewInt8(127):  {127},
			NewInt8(-1):   {255},
			NewInt8(-127): {129},
			NewInt8(-128): {128},
		},
		"Int16": {
			NewInt16(0):      {0, 0},
			NewInt16(42):     {0, 42},
			NewInt16(32767):  {127, 255},
			NewInt16(-1):     {255, 255},
			NewInt16(-32767): {128, 1},
			NewInt16(-32768): {128, 0},
		},
		"Int32": {
			NewInt32(0):           {0, 0, 0, 0},
			NewInt32(42):          {0, 0, 0, 42},
			NewInt32(2147483647):  {127, 255, 255, 255},
			NewInt32(-1):          {255, 255, 255, 255},
			NewInt32(-2147483647): {128, 0, 0, 1},
			NewInt32(-2147483648): {128, 0, 0, 0},
		},
		"Int64": {
			NewInt64(0):                    {0, 0, 0, 0, 0, 0, 0, 0},
			NewInt64(42):                   {0, 0, 0, 0, 0, 0, 0, 42},
			NewInt64(9223372036854775807):  {127, 255, 255, 255, 255, 255, 255, 255},
			NewInt64(-1):                   {255, 255, 255, 255, 255, 255, 255, 255},
			NewInt64(-9223372036854775807): {128, 0, 0, 0, 0, 0, 0, 1},
			NewInt64(-9223372036854775808): {128, 0, 0, 0, 0, 0, 0, 0},
		},
		"Int128": {
			NewInt128(0):                  {0},
			NewInt128(42):                 {42},
			NewInt128(127):                {127},
			NewInt128(128):                {0, 128},
			NewInt128(200):                {0, 200},
			NewInt128(-1):                 {255},
			NewInt128(-200):               {255, 56},
			NewInt128(-10000000000000000): {220, 121, 13, 144, 63, 0, 0},
		},
		"Int256": {
			NewInt256(0):                  {0},
			NewInt256(42):                 {42},
			NewInt256(127):                {127},
			NewInt256(128):                {0, 128},
			NewInt256(200):                {0, 200},
			NewInt256(-1):                 {255},
			NewInt256(-200):               {255, 56},
			NewInt256(-10000000000000000): {220, 121, 13, 144, 63, 0, 0},
		},
		// UInt*
		"UInt": {
			NewUInt(0):   {0},
			NewUInt(42):  {42},
			NewUInt(127): {127},
			NewUInt(128): {128},
			NewUInt(200): {200},
		},
		"UInt8": {
			NewUInt8(0):   {0},
			NewUInt8(42):  {42},
			NewUInt8(127): {127},
			NewUInt8(128): {128},
			NewUInt8(255): {255},
		},
		"UInt16": {
			NewUInt16(0):     {0, 0},
			NewUInt16(42):    {0, 42},
			NewUInt16(32767): {127, 255},
			NewUInt16(32768): {128, 0},
			NewUInt16(65535): {255, 255},
		},
		"UInt32": {
			NewUInt32(0):          {0, 0, 0, 0},
			NewUInt32(42):         {0, 0, 0, 42},
			NewUInt32(2147483647): {127, 255, 255, 255},
			NewUInt32(2147483648): {128, 0, 0, 0},
			NewUInt32(4294967295): {255, 255, 255, 255},
		},
		"UInt64": {
			NewUInt64(0):                    {0, 0, 0, 0, 0, 0, 0, 0},
			NewUInt64(42):                   {0, 0, 0, 0, 0, 0, 0, 42},
			NewUInt64(9223372036854775807):  {127, 255, 255, 255, 255, 255, 255, 255},
			NewUInt64(9223372036854775808):  {128, 0, 0, 0, 0, 0, 0, 0},
			NewUInt64(18446744073709551615): {255, 255, 255, 255, 255, 255, 255, 255},
		},
		"UInt128": {
			NewUInt128(0):   {0},
			NewUInt128(42):  {42},
			NewUInt128(127): {127},
			NewUInt128(128): {128},
			NewUInt128(200): {200},
		},
		"UInt256": {
			NewUInt256(0):   {0},
			NewUInt256(42):  {42},
			NewUInt256(127): {127},
			NewUInt256(128): {128},
			NewUInt256(200): {200},
		},
		// Word*
		"Word8": {
			NewWord8(0):   {0},
			NewWord8(42):  {42},
			NewWord8(127): {127},
			NewWord8(128): {128},
			NewWord8(255): {255},
		},
		"Word16": {
			NewWord16(0):     {0, 0},
			NewWord16(42):    {0, 42},
			NewWord16(32767): {127, 255},
			NewWord16(32768): {128, 0},
			NewWord16(65535): {255, 255},
		},
		"Word32": {
			NewWord32(0):          {0, 0, 0, 0},
			NewWord32(42):         {0, 0, 0, 42},
			NewWord32(2147483647): {127, 255, 255, 255},
			NewWord32(2147483648): {128, 0, 0, 0},
			NewWord32(4294967295): {255, 255, 255, 255},
		},
		"Word64": {
			NewWord64(0):                    {0, 0, 0, 0, 0, 0, 0, 0},
			NewWord64(42):                   {0, 0, 0, 0, 0, 0, 0, 42},
			NewWord64(9223372036854775807):  {127, 255, 255, 255, 255, 255, 255, 255},
			NewWord64(9223372036854775808):  {128, 0, 0, 0, 0, 0, 0, 0},
			NewWord64(18446744073709551615): {255, 255, 255, 255, 255, 255, 255, 255},
		},
		// Fix*
		"Fix64": {
			Fix64(0):           {0, 0, 0, 0, 0, 0, 0, 0},
			Fix64(42_00000000): {0, 0, 0, 0, 250, 86, 234, 0},
			Fix64(42_24000000): {0, 0, 0, 0, 251, 197, 32, 0},
			Fix64(-1_00000000): {255, 255, 255, 255, 250, 10, 31, 0},
		},
		// UFix*
		"UFix64": {
			Fix64(0):           {0, 0, 0, 0, 0, 0, 0, 0},
			Fix64(42_00000000): {0, 0, 0, 0, 250, 86, 234, 0},
			Fix64(42_24000000): {0, 0, 0, 0, 251, 197, 32, 0},
		},
	}

	// Ensure the test cases are complete

	for _, integerType := range sema.AllNumberTypes {
		switch integerType {
		case sema.NumberType, sema.SignedNumberType,
			sema.IntegerType, sema.SignedIntegerType,
			sema.FixedPointType, sema.SignedFixedPointType:
			continue
		}

		if _, ok := typeTests[integerType.String()]; !ok {
			panic(fmt.Sprintf("broken test: missing %s", integerType))
		}
	}

	for ty, tests := range typeTests {

		for value, expected := range tests {

			t.Run(fmt.Sprintf("%s: %s", ty, value), func(t *testing.T) {

				assert.Equal(t,
					expected,
					value.ToBigEndianBytes(),
				)
			})
		}
	}
}

func TestOptional_Type(t *testing.T) {
	t.Parallel()

	t.Run("none", func(t *testing.T) {

		require.Equal(t,
			&OptionalType{
				Type: NeverType{},
			},
			Optional{}.Type(),
		)
	})

	t.Run("some", func(t *testing.T) {

		require.Equal(t,
			&OptionalType{
				Type: Int8Type{},
			},
			Optional{
				Value: Int8(2),
			}.Type(),
		)
	})
}

func TestNonUTF8String(t *testing.T) {
	t.Parallel()

	nonUTF8String := "\xbd\xb2\x3d\xbc\x20\xe2"

	// Make sure it is an invalid utf8 string
	assert.False(t, utf8.ValidString(nonUTF8String))

	_, err := NewString(nonUTF8String)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "invalid UTF-8 in string")
}

func TestNewInt128FromBig(t *testing.T) {
	t.Parallel()

	_, err := NewInt128FromBig(big.NewInt(1))
	require.NoError(t, err)

	belowMin := new(big.Int).Sub(
		sema.Int128TypeMinIntBig,
		big.NewInt(1),
	)
	_, err = NewInt128FromBig(belowMin)
	require.Error(t, err)

	aboveMax := new(big.Int).Add(
		sema.Int128TypeMaxIntBig,
		big.NewInt(1),
	)
	_, err = NewInt128FromBig(aboveMax)
	require.Error(t, err)
}

func TestNewInt256FromBig(t *testing.T) {
	t.Parallel()

	_, err := NewInt256FromBig(big.NewInt(1))
	require.NoError(t, err)

	belowMin := new(big.Int).Sub(
		sema.Int256TypeMinIntBig,
		big.NewInt(1),
	)
	_, err = NewInt256FromBig(belowMin)
	require.Error(t, err)

	aboveMax := new(big.Int).Add(
		sema.Int256TypeMaxIntBig,
		big.NewInt(1),
	)
	_, err = NewInt256FromBig(aboveMax)
	require.Error(t, err)
}

func TestNewUIntFromBig(t *testing.T) {
	t.Parallel()

	_, err := NewUIntFromBig(big.NewInt(1))
	require.NoError(t, err)

	belowMin := big.NewInt(-1)
	_, err = NewUIntFromBig(belowMin)
	require.Error(t, err)

	large := new(big.Int).Add(
		sema.UInt256TypeMaxIntBig,
		big.NewInt(1),
	)
	_, err = NewUIntFromBig(large)
	require.NoError(t, err)
}

func TestNewUInt128FromBig(t *testing.T) {
	t.Parallel()

	_, err := NewUInt128FromBig(big.NewInt(1))
	require.NoError(t, err)

	belowMin := big.NewInt(-1)
	_, err = NewUInt128FromBig(belowMin)
	require.Error(t, err)

	aboveMax := new(big.Int).Add(
		sema.UInt128TypeMaxIntBig,
		big.NewInt(1),
	)
	_, err = NewUInt128FromBig(aboveMax)
	require.Error(t, err)
}

func TestNewUInt256FromBig(t *testing.T) {
	t.Parallel()

	_, err := NewUInt256FromBig(big.NewInt(1))
	require.NoError(t, err)

	belowMin := big.NewInt(-1)
	_, err = NewUInt256FromBig(belowMin)
	require.Error(t, err)

	aboveMax := new(big.Int).Add(
		sema.UInt256TypeMaxIntBig,
		big.NewInt(1),
	)
	_, err = NewUInt256FromBig(aboveMax)
	require.Error(t, err)
}

func TestValue_Type(t *testing.T) {

	t.Parallel()

	checkedTypes := map[Type]struct{}{}

	test := func(name string, testCase valueTestCase) {

		t.Run(name, func(t *testing.T) {

			value := testCase.value

			returnedType := value.Type()

			expectedType := testCase.expectedType
			if expectedType != nil {
				require.NotNil(t, returnedType)
				require.True(t, returnedType != nil)
				require.Equal(t, expectedType, returnedType)
			} else if !testCase.noType {
				exampleType := testCase.exampleType
				require.NotNil(t, exampleType)

				// Ensure the nil type is an *untyped nil*
				require.Nil(t, returnedType)
				require.True(t, returnedType == nil)

				// Once a type is set, it should be returned
				value = testCase.withType(value, exampleType)

				returnedType = value.Type()

				require.NotNil(t, returnedType)
				require.Equal(t, exampleType, returnedType)
			}

			if !testCase.noType {
				// Check if the type is not a duplicate of some other type
				// i.e: two values can't return the same type.
				//
				// Current known exceptions:
				// - Capability: PathCapabilityValue | IDCapabilityValue

				var ignoreDuplicateType bool

				if _, ok := returnedType.(*CapabilityType); ok {
					switch value.(type) {
					case IDCapability, PathCapability:
						ignoreDuplicateType = true
					}
				}

				if !ignoreDuplicateType {
					require.NotContains(t, checkedTypes, returnedType)
				}
				checkedTypes[returnedType] = struct{}{}
			}
		})
	}

	for name, testCase := range newValueTestCases() {
		test(name, testCase)
	}
}
