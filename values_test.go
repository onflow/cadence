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
	"fmt"
	"math/big"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestStringer(t *testing.T) {

	t.Parallel()

	type testCase struct {
		value    Value
		expected string
	}

	parsedU, _ := ParseUFix64("64.01")
	ufix64, _ := NewUFix64(parsedU)

	parsed, _ := ParseFix64("-32.11")
	fix64, _ := NewFix64(parsed)

	stringerTests := map[string]testCase{
		"UInt": {
			value:    NewUInt(10),
			expected: "10",
		},
		"UInt8": {
			value:    NewUInt8(8),
			expected: "8",
		},
		"UInt16": {
			value:    NewUInt16(16),
			expected: "16",
		},
		"UInt32": {
			value:    NewUInt32(32),
			expected: "32",
		},
		"UInt64": {
			value:    NewUInt64(64),
			expected: "64",
		},
		"UInt128": {
			value:    NewUInt128(128),
			expected: "128",
		},
		"UInt256": {
			value:    NewUInt256(256),
			expected: "256",
		},
		"Int": {
			value:    NewInt(1000000),
			expected: "1000000",
		},
		"Int8": {
			value:    NewInt8(-8),
			expected: "-8",
		},
		"Int16": {
			value:    NewInt16(-16),
			expected: "-16",
		},
		"Int32": {
			value:    NewInt32(-32),
			expected: "-32",
		},
		"Int64": {
			value:    NewInt64(-64),
			expected: "-64",
		},
		"Int128": {
			value:    NewInt128(-128),
			expected: "-128",
		},
		"Int256": {
			value:    NewInt256(-256),
			expected: "-256",
		},
		"Word8": {
			value:    NewWord8(8),
			expected: "8",
		},
		"Word16": {
			value:    NewWord16(16),
			expected: "16",
		},
		"Word32": {
			value:    NewWord32(32),
			expected: "32",
		},
		"Word64": {
			value:    NewWord64(64),
			expected: "64",
		},
		"UFix64": {
			value:    ufix64,
			expected: "64.01000000",
		},
		"Fix64": {
			value:    fix64,
			expected: "-32.11000000",
		},
		"Void": {
			value:    NewVoid(),
			expected: "()",
		},
		"true": {
			value:    NewBool(true),
			expected: "true",
		},
		"false": {
			value:    NewBool(false),
			expected: "false",
		},
		"some": {
			value:    NewOptional(ufix64),
			expected: "64.01000000",
		},
		"nil": {
			value:    NewOptional(nil),
			expected: "nil",
		},
		"String": {
			value:    String("Flow ridah!"),
			expected: "\"Flow ridah!\"",
		},
		"Array": {
			value: NewArray([]Value{
				NewInt(10),
				String("TEST"),
			}),
			expected: "[10, \"TEST\"]",
		},
		"Dictionary": {
			value: NewDictionary([]KeyValuePair{
				{
					Key:   String("key"),
					Value: String("value"),
				},
			}),
			expected: "{\"key\": \"value\"}",
		},
		"Bytes": {
			value:    NewBytes([]byte{0x1, 0x2}),
			expected: "[0x1, 0x2]",
		},
		"Address": {
			value:    NewAddress([8]byte{0, 0, 0, 0, 0, 0, 0, 1}),
			expected: "0x0000000000000001",
		},
		"struct": {
			value: NewStruct([]Value{String("bar")}).WithType(&StructType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooStruct",
				Fields: []Field{
					{
						Identifier: "y",
						Type:       StringType{},
					},
				},
			}),
			expected: "S.test.FooStruct(y: \"bar\")",
		},
		"resource": {
			value: NewResource([]Value{NewInt(1)}).WithType(&ResourceType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooResource",
				Fields: []Field{
					{
						Identifier: "bar",
						Type:       IntType{},
					},
				},
			}),
			expected: "S.test.FooResource(bar: 1)",
		},
		"event": {
			value: NewEvent(
				[]Value{
					NewInt(1),
					String("foo"),
				},
			).WithType(&EventType{
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
			}),
			expected: "S.test.FooEvent(a: 1, b: \"foo\")",
		},
		"contract": {
			value: NewContract([]Value{String("bar")}).WithType(&ContractType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooContract",
				Fields: []Field{
					{
						Identifier: "y",
						Type:       StringType{},
					},
				},
			}),
			expected: "S.test.FooContract(y: \"bar\")",
		},
		"Link": {
			value: NewLink(
				Path{
					Domain:     "storage",
					Identifier: "foo",
				},
				"Int",
			),
			expected: "Link<Int>(/storage/foo)",
		},
		"Path": {
			value: Path{
				Domain:     "storage",
				Identifier: "foo",
			},
			expected: "/storage/foo",
		},
		"Type": {
			value:    TypeValue{StaticType: IntType{}},
			expected: "Type<Int>()",
		},
		"Capability": {
			value: Capability{
				Path:       Path{Domain: "storage", Identifier: "foo"},
				Address:    BytesToAddress([]byte{1, 2, 3, 4, 5}),
				BorrowType: IntType{},
			},
			expected: "Capability<Int>(address: 0x0000000102030405, path: /storage/foo)",
		},
	}

	test := func(name string, testCase testCase) {

		t.Run(name, func(t *testing.T) {

			t.Parallel()

			assert.Equal(t,
				testCase.expected,
				testCase.value.String(),
			)
		})
	}

	for name, testCase := range stringerTests {
		test(name, testCase)
	}
}

func TestToBigEndianBytes(t *testing.T) {

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

	t.Run("none", func(t *testing.T) {

		require.Equal(t,
			OptionalType{
				Type: NeverType{},
			},
			Optional{}.Type(),
		)
	})

	t.Run("some", func(t *testing.T) {

		require.Equal(t,
			OptionalType{
				Type: Int8Type{},
			},
			Optional{
				Value: Int8(2),
			}.Type(),
		)
	})
}

func TestNonUTF8String(t *testing.T) {
	nonUTF8String := "\xbd\xb2\x3d\xbc\x20\xe2"

	// Make sure it is an invalid utf8 string
	assert.False(t, utf8.ValidString(nonUTF8String))

	_, err := NewString(nonUTF8String)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "invalid UTF-8 in string")
}

func TestNewInt128FromBig(t *testing.T) {

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
