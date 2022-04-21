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
	ufix64, _ := NewUnmeteredUFix64(parsedU)

	parsed, _ := ParseFix64("-32.11")
	fix64, _ := NewUnmeteredFix64(parsed)

	stringerTests := map[string]testCase{
		"UInt": {
			value:    NewUnmeteredUInt(10),
			expected: "10",
		},
		"UInt8": {
			value:    NewUnmeteredUInt8(8),
			expected: "8",
		},
		"UInt16": {
			value:    NewUnmeteredUInt16(16),
			expected: "16",
		},
		"UInt32": {
			value:    NewUnmeteredUInt32(32),
			expected: "32",
		},
		"UInt64": {
			value:    NewUnmeteredUInt64(64),
			expected: "64",
		},
		"UInt128": {
			value:    NewUnmeteredUInt128(128),
			expected: "128",
		},
		"UInt256": {
			value:    NewUnmeteredUInt256(256),
			expected: "256",
		},
		"Int": {
			value:    NewUnmeteredInt(1000000),
			expected: "1000000",
		},
		"Int8": {
			value:    NewUnmeteredInt8(-8),
			expected: "-8",
		},
		"Int16": {
			value:    NewUnmeteredInt16(-16),
			expected: "-16",
		},
		"Int32": {
			value:    NewUnmeteredInt32(-32),
			expected: "-32",
		},
		"Int64": {
			value:    NewUnmeteredInt64(-64),
			expected: "-64",
		},
		"Int128": {
			value:    NewUnmeteredInt128(-128),
			expected: "-128",
		},
		"Int256": {
			value:    NewUnmeteredInt256(-256),
			expected: "-256",
		},
		"Word8": {
			value:    NewUnmeteredWord8(8),
			expected: "8",
		},
		"Word16": {
			value:    NewUnmeteredWord16(16),
			expected: "16",
		},
		"Word32": {
			value:    NewUnmeteredWord32(32),
			expected: "32",
		},
		"Word64": {
			value:    NewUnmeteredWord64(64),
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
			value:    NewUnmeteredVoid(),
			expected: "()",
		},
		"true": {
			value:    NewUnmeteredBool(true),
			expected: "true",
		},
		"false": {
			value:    NewUnmeteredBool(false),
			expected: "false",
		},
		"some": {
			value:    NewUnmeteredOptional(ufix64),
			expected: "64.01000000",
		},
		"nil": {
			value:    NewUnmeteredOptional(nil),
			expected: "nil",
		},
		"String": {
			value:    String("Flow ridah!"),
			expected: "\"Flow ridah!\"",
		},
		"Array": {
			value: NewArray([]Value{
				NewUnmeteredInt(10),
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
			value:    NewUnmeteredAddress([8]byte{0, 0, 0, 0, 0, 0, 0, 1}),
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
			value: NewResource([]Value{NewUnmeteredInt(1)}).WithType(&ResourceType{
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
					NewUnmeteredInt(1),
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
				Address:    BytesToUnmeteredAddress([]byte{1, 2, 3, 4, 5}),
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
			NewUnmeteredInt(0):                  {0},
			NewUnmeteredInt(42):                 {42},
			NewUnmeteredInt(127):                {127},
			NewUnmeteredInt(128):                {0, 128},
			NewUnmeteredInt(200):                {0, 200},
			NewUnmeteredInt(-1):                 {255},
			NewUnmeteredInt(-200):               {255, 56},
			NewUnmeteredInt(-10000000000000000): {220, 121, 13, 144, 63, 0, 0},
		},
		"Int8": {
			NewUnmeteredInt8(0):    {0},
			NewUnmeteredInt8(42):   {42},
			NewUnmeteredInt8(127):  {127},
			NewUnmeteredInt8(-1):   {255},
			NewUnmeteredInt8(-127): {129},
			NewUnmeteredInt8(-128): {128},
		},
		"Int16": {
			NewUnmeteredInt16(0):      {0, 0},
			NewUnmeteredInt16(42):     {0, 42},
			NewUnmeteredInt16(32767):  {127, 255},
			NewUnmeteredInt16(-1):     {255, 255},
			NewUnmeteredInt16(-32767): {128, 1},
			NewUnmeteredInt16(-32768): {128, 0},
		},
		"Int32": {
			NewUnmeteredInt32(0):           {0, 0, 0, 0},
			NewUnmeteredInt32(42):          {0, 0, 0, 42},
			NewUnmeteredInt32(2147483647):  {127, 255, 255, 255},
			NewUnmeteredInt32(-1):          {255, 255, 255, 255},
			NewUnmeteredInt32(-2147483647): {128, 0, 0, 1},
			NewUnmeteredInt32(-2147483648): {128, 0, 0, 0},
		},
		"Int64": {
			NewUnmeteredInt64(0):                    {0, 0, 0, 0, 0, 0, 0, 0},
			NewUnmeteredInt64(42):                   {0, 0, 0, 0, 0, 0, 0, 42},
			NewUnmeteredInt64(9223372036854775807):  {127, 255, 255, 255, 255, 255, 255, 255},
			NewUnmeteredInt64(-1):                   {255, 255, 255, 255, 255, 255, 255, 255},
			NewUnmeteredInt64(-9223372036854775807): {128, 0, 0, 0, 0, 0, 0, 1},
			NewUnmeteredInt64(-9223372036854775808): {128, 0, 0, 0, 0, 0, 0, 0},
		},
		"Int128": {
			NewUnmeteredInt128(0):                  {0},
			NewUnmeteredInt128(42):                 {42},
			NewUnmeteredInt128(127):                {127},
			NewUnmeteredInt128(128):                {0, 128},
			NewUnmeteredInt128(200):                {0, 200},
			NewUnmeteredInt128(-1):                 {255},
			NewUnmeteredInt128(-200):               {255, 56},
			NewUnmeteredInt128(-10000000000000000): {220, 121, 13, 144, 63, 0, 0},
		},
		"Int256": {
			NewUnmeteredInt256(0):                  {0},
			NewUnmeteredInt256(42):                 {42},
			NewUnmeteredInt256(127):                {127},
			NewUnmeteredInt256(128):                {0, 128},
			NewUnmeteredInt256(200):                {0, 200},
			NewUnmeteredInt256(-1):                 {255},
			NewUnmeteredInt256(-200):               {255, 56},
			NewUnmeteredInt256(-10000000000000000): {220, 121, 13, 144, 63, 0, 0},
		},
		// UInt*
		"UInt": {
			NewUnmeteredUInt(0):   {0},
			NewUnmeteredUInt(42):  {42},
			NewUnmeteredUInt(127): {127},
			NewUnmeteredUInt(128): {128},
			NewUnmeteredUInt(200): {200},
		},
		"UInt8": {
			NewUnmeteredUInt8(0):   {0},
			NewUnmeteredUInt8(42):  {42},
			NewUnmeteredUInt8(127): {127},
			NewUnmeteredUInt8(128): {128},
			NewUnmeteredUInt8(255): {255},
		},
		"UInt16": {
			NewUnmeteredUInt16(0):     {0, 0},
			NewUnmeteredUInt16(42):    {0, 42},
			NewUnmeteredUInt16(32767): {127, 255},
			NewUnmeteredUInt16(32768): {128, 0},
			NewUnmeteredUInt16(65535): {255, 255},
		},
		"UInt32": {
			NewUnmeteredUInt32(0):          {0, 0, 0, 0},
			NewUnmeteredUInt32(42):         {0, 0, 0, 42},
			NewUnmeteredUInt32(2147483647): {127, 255, 255, 255},
			NewUnmeteredUInt32(2147483648): {128, 0, 0, 0},
			NewUnmeteredUInt32(4294967295): {255, 255, 255, 255},
		},
		"UInt64": {
			NewUnmeteredUInt64(0):                    {0, 0, 0, 0, 0, 0, 0, 0},
			NewUnmeteredUInt64(42):                   {0, 0, 0, 0, 0, 0, 0, 42},
			NewUnmeteredUInt64(9223372036854775807):  {127, 255, 255, 255, 255, 255, 255, 255},
			NewUnmeteredUInt64(9223372036854775808):  {128, 0, 0, 0, 0, 0, 0, 0},
			NewUnmeteredUInt64(18446744073709551615): {255, 255, 255, 255, 255, 255, 255, 255},
		},
		"UInt128": {
			NewUnmeteredUInt128(0):   {0},
			NewUnmeteredUInt128(42):  {42},
			NewUnmeteredUInt128(127): {127},
			NewUnmeteredUInt128(128): {128},
			NewUnmeteredUInt128(200): {200},
		},
		"UInt256": {
			NewUnmeteredUInt256(0):   {0},
			NewUnmeteredUInt256(42):  {42},
			NewUnmeteredUInt256(127): {127},
			NewUnmeteredUInt256(128): {128},
			NewUnmeteredUInt256(200): {200},
		},
		// Word*
		"Word8": {
			NewUnmeteredWord8(0):   {0},
			NewUnmeteredWord8(42):  {42},
			NewUnmeteredWord8(127): {127},
			NewUnmeteredWord8(128): {128},
			NewUnmeteredWord8(255): {255},
		},
		"Word16": {
			NewUnmeteredWord16(0):     {0, 0},
			NewUnmeteredWord16(42):    {0, 42},
			NewUnmeteredWord16(32767): {127, 255},
			NewUnmeteredWord16(32768): {128, 0},
			NewUnmeteredWord16(65535): {255, 255},
		},
		"Word32": {
			NewUnmeteredWord32(0):          {0, 0, 0, 0},
			NewUnmeteredWord32(42):         {0, 0, 0, 42},
			NewUnmeteredWord32(2147483647): {127, 255, 255, 255},
			NewUnmeteredWord32(2147483648): {128, 0, 0, 0},
			NewUnmeteredWord32(4294967295): {255, 255, 255, 255},
		},
		"Word64": {
			NewUnmeteredWord64(0):                    {0, 0, 0, 0, 0, 0, 0, 0},
			NewUnmeteredWord64(42):                   {0, 0, 0, 0, 0, 0, 0, 42},
			NewUnmeteredWord64(9223372036854775807):  {127, 255, 255, 255, 255, 255, 255, 255},
			NewUnmeteredWord64(9223372036854775808):  {128, 0, 0, 0, 0, 0, 0, 0},
			NewUnmeteredWord64(18446744073709551615): {255, 255, 255, 255, 255, 255, 255, 255},
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

	_, err := NewUnmeteredString(nonUTF8String)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "invalid UTF-8 in string")
}

func TestNewInt128FromBig(t *testing.T) {

	_, err := NewUnmeteredInt128FromBig(big.NewInt(1))
	require.NoError(t, err)

	belowMin := new(big.Int).Sub(
		sema.Int128TypeMinIntBig,
		big.NewInt(1),
	)
	_, err = NewUnmeteredInt128FromBig(belowMin)
	require.Error(t, err)

	aboveMax := new(big.Int).Add(
		sema.Int128TypeMaxIntBig,
		big.NewInt(1),
	)
	_, err = NewUnmeteredInt128FromBig(aboveMax)
	require.Error(t, err)
}

func TestNewInt256FromBig(t *testing.T) {

	_, err := NewUnmeteredInt256FromBig(big.NewInt(1))
	require.NoError(t, err)

	belowMin := new(big.Int).Sub(
		sema.Int256TypeMinIntBig,
		big.NewInt(1),
	)
	_, err = NewUnmeteredInt256FromBig(belowMin)
	require.Error(t, err)

	aboveMax := new(big.Int).Add(
		sema.Int256TypeMaxIntBig,
		big.NewInt(1),
	)
	_, err = NewUnmeteredInt256FromBig(aboveMax)
	require.Error(t, err)
}

func TestNewUIntFromBig(t *testing.T) {

	_, err := NewUnmeteredUIntFromBig(big.NewInt(1))
	require.NoError(t, err)

	belowMin := big.NewInt(-1)
	_, err = NewUnmeteredUIntFromBig(belowMin)
	require.Error(t, err)

	large := new(big.Int).Add(
		sema.UInt256TypeMaxIntBig,
		big.NewInt(1),
	)
	_, err = NewUnmeteredUIntFromBig(large)
	require.NoError(t, err)
}

func TestNewUInt128FromBig(t *testing.T) {

	_, err := NewUnmeteredUInt128FromBig(big.NewInt(1))
	require.NoError(t, err)

	belowMin := big.NewInt(-1)
	_, err = NewUnmeteredUInt128FromBig(belowMin)
	require.Error(t, err)

	aboveMax := new(big.Int).Add(
		sema.UInt128TypeMaxIntBig,
		big.NewInt(1),
	)
	_, err = NewUnmeteredUInt128FromBig(aboveMax)
	require.Error(t, err)
}

func TestNewUInt256FromBig(t *testing.T) {

	_, err := NewUnmeteredUInt256FromBig(big.NewInt(1))
	require.NoError(t, err)

	belowMin := big.NewInt(-1)
	_, err = NewUnmeteredUInt256FromBig(belowMin)
	require.Error(t, err)

	aboveMax := new(big.Int).Add(
		sema.UInt256TypeMaxIntBig,
		big.NewInt(1),
	)
	_, err = NewUnmeteredUInt256FromBig(aboveMax)
	require.Error(t, err)
}
