/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

package interpreter

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/common/orderedmap"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

type encodeDecodeTest struct {
	value                 Value
	encoded               []byte
	invalid               bool
	deferred              bool
	deferrals             *EncodingDeferrals
	decodedValue          Value
	decodeOnly            bool
	decodeVersionOverride bool
	decodeVersion         uint16
}

var testOwner = common.BytesToAddress([]byte{0x42})

func testEncodeDecode(t *testing.T, test encodeDecodeTest) {

	t.Parallel()

	var encoded []byte
	var deferrals *EncodingDeferrals
	if test.value != nil && !test.decodeOnly {
		test.value.SetOwner(&testOwner)

		var err error

		encoded, deferrals, err = EncodeValue(test.value, nil, test.deferred, nil)
		require.NoError(t, err)

		if test.encoded != nil {
			utils.AssertEqualWithDiff(t, test.encoded, encoded)
		}
	} else {
		encoded = test.encoded
	}

	version := CurrentEncodingVersion
	if test.decodeVersionOverride {
		version = test.decodeVersion
	}

	var decoded Value
	var err error
	if version <= 3 {
		decoded, err = DecodeValueV3(encoded, &testOwner, nil, version, nil)
	} else {
		decoded, err = DecodeValue(encoded, &testOwner, nil, version, nil)
	}

	if test.invalid {
		require.Error(t, err)
	} else {
		require.NoError(t, err)

		// Make sure the content is built.
		_ = decoded.String()

		if !test.deferred || (test.deferred && test.decodedValue != nil) {
			expectedValue := test.value
			if test.decodedValue != nil {
				test.decodedValue.SetOwner(&testOwner)
				expectedValue = test.decodedValue
			}
			utils.AssertEqualWithDiff(t, expectedValue, decoded)
		}
	}

	if test.value != nil && !test.decodeOnly {
		if test.deferred {
			utils.AssertEqualWithDiff(t, test.deferrals, deferrals)
		} else {
			require.Empty(t, deferrals.Values)
			require.Empty(t, deferrals.Moves)
		}
	}
}

func testEncodeDecodeOldFormat(t *testing.T, test encodeDecodeTest, oldFormatVersion uint16, oldFormatEncoded []byte) {

	if oldFormatVersion > 3 {
		require.FailNow(t, "oldFormatVersion must be <= 3 to test encode/decode backwards compatibility")
		return
	}

	test.value.SetOwner(&testOwner)

	// Decode oldFormatEncoded and compare decoded value with test.value or test.decodedValue
	decoded, err := DecodeValueV3(oldFormatEncoded, &testOwner, nil, oldFormatVersion, nil)
	require.NoError(t, err)

	if !test.deferred || (test.deferred && test.decodedValue != nil) {
		expectedValue := test.value
		if test.decodedValue != nil {
			test.decodedValue.SetOwner(&testOwner)
			expectedValue = test.decodedValue
		}
		utils.AssertEqualWithDiff(t, expectedValue, decoded)
	}

	// Encode decoded value to new format and compare data to test.encoded
	encoded, _, err := EncodeValue(decoded, nil, test.deferred, nil)
	require.NoError(t, err)

	if test.encoded != nil {
		utils.AssertEqualWithDiff(t, test.encoded, encoded)
	}
}

func TestEncodeDecodeNilValue(t *testing.T) {

	testEncodeDecode(t,
		encodeDecodeTest{
			value: NilValue{},
			encoded: []byte{
				// null
				0xf6,
			},
		},
	)
}

func TestEncodeDecodeVoidValue(t *testing.T) {

	testEncodeDecode(t,
		encodeDecodeTest{
			value: VoidValue{},
			encoded: []byte{
				// tag
				0xd8, cborTagVoidValue,
				// null
				0xf6,
			},
		},
	)
}

func TestEncodeDecodeBool(t *testing.T) {

	t.Parallel()

	t.Run("false", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: BoolValue(false),
				encoded: []byte{
					// false
					0xf4,
				},
			},
		)
	})

	t.Run("true", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: BoolValue(true),
				encoded: []byte{
					// true
					0xf5,
				},
			},
		)
	})
}

func TestEncodeDecodeString(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		expected := NewStringValue("")

		testEncodeDecode(t,
			encodeDecodeTest{
				value: expected,
				encoded: []byte{
					//  UTF-8 string, 0 bytes follow
					0x60,
				},
			})
	})

	t.Run("non-empty", func(t *testing.T) {
		expected := NewStringValue("foo")

		testEncodeDecode(t,
			encodeDecodeTest{
				value: expected,
				encoded: []byte{
					// UTF-8 string, 3 bytes follow
					0x63,
					// f, o, o
					0x66, 0x6f, 0x6f,
				},
			},
		)
	})
}

func TestEncodeDecodeArray(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		expected := NewArrayValueUnownedNonCopying(
			&sema.ConstantSizedType{
				Type: sema.AnyStructType,
				Size: 0,
			},
		)
		expected.modified = false

		testEncodeDecode(t,
			encodeDecodeTest{
				value: expected,
				encoded: []byte{
					// cbor Array Value tag
					0xd8, cborTagArrayValue,

					// array, 2 items follow
					0x82,

					// Type info

					// array type tag
					0xd8, cborTagConstantSizedStaticType,

					// array, 2 items follow
					0x82,

					// size (0)
					0x0,

					// element type
					0xd8, cborTagPrimitiveStaticType, byte(PrimitiveStaticTypeAnyStruct),

					// Elements

					// array, 0 items follow
					0x80,
				},
			})
	})

	t.Run("string and bool", func(t *testing.T) {
		expectedString := NewStringValue("test")

		expected := NewArrayValueUnownedNonCopying(
			&sema.VariableSizedType{
				Type: sema.AnyStructType,
			},
			expectedString,
			BoolValue(true),
		)
		expected.modified = false

		testEncodeDecode(t,
			encodeDecodeTest{
				value: expected,
				encoded: []byte{
					// cbor Array Value tag
					0xd8, cborTagArrayValue,

					// array, 2 items follow
					0x82,

					// Type info

					// array type tag
					0xd8, cborTagVariableSizedStaticType,

					// element type
					0xd8, cborTagPrimitiveStaticType, byte(PrimitiveStaticTypeAnyStruct),

					// Elements

					// array, 2 items follow
					0x82,
					// UTF-8 string, length 4
					0x64,
					// t, e, s, t
					0x74, 0x65, 0x73, 0x74,
					// true
					0xf5,
				},
			},
		)
	})
}

func TestEncodeDecodeDictionary(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {

		expected := NewDictionaryValueUnownedNonCopying(
			&sema.DictionaryType{
				KeyType:   sema.StringType,
				ValueType: sema.AnyStructType,
			},
		)
		expected.modified = false
		expected.Keys().modified = false

		encoded := []byte{
			// tag
			0xd8, cborTagDictionaryValue,
			// array, 2 items follow
			0x82,

			// cbor Array Value tag
			0xd8, cborTagArrayValue,

			// array, 2 items follow
			0x82,

			// Type info

			// array type tag
			0xd8, cborTagVariableSizedStaticType,

			// element type
			0xd8, cborTagPrimitiveStaticType, byte(PrimitiveStaticTypeString),

			// Element: array, 0 items follow
			0x80,

			// array, 0 items follow
			0x80,
		}

		version3Encoded := []byte{
			// tag
			0xd8, cborTagDictionaryValue,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x00,
			// array, 0 items follow
			0x80,
			// key 1
			0x01,
			// map, 0 pairs of items follow
			0xa0,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   expected,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   expected,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("non-empty", func(t *testing.T) {
		key1 := NewStringValue("test")
		value1 := NewArrayValueUnownedNonCopying(
			&sema.VariableSizedType{
				Type: sema.AnyStructType,
			},
		)

		key2 := BoolValue(true)
		value2 := BoolValue(false)

		key3 := NewStringValue("foo")
		value3 := NewStringValue("bar")

		expected := NewDictionaryValueUnownedNonCopying(
			// TODO: type
			&sema.DictionaryType{
				KeyType:   sema.AnyStructType,
				ValueType: sema.AnyStructType,
			},
			key1, value1,
			key2, value2,
			key3, value3,
		)

		expected.modified = false
		expected.Keys().modified = false

		value1.modified = false

		encoded := []byte{
			// tag
			0xd8, cborTagDictionaryValue,
			// array, 2 items follow
			0x82,

			// Keys

			// cbor Array Value tag
			0xd8, cborTagArrayValue,

			// array, 2 items follow
			0x82,

			// Type info

			// array type tag
			0xd8, cborTagVariableSizedStaticType,

			// element type
			0xd8, cborTagPrimitiveStaticType, byte(PrimitiveStaticTypeAnyStruct),

			// array, 3 items follow
			0x83,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// true
			0xf5,
			// UTF-8 string, length 3
			0x63,
			// f, o, o
			0x66, 0x6f, 0x6f,

			// Values

			// array, 3 items follow
			0x83,

			// cbor Array Value tag
			0xd8, cborTagArrayValue,

			// array, 2 items follow
			0x82,

			// Type info

			// array type tag
			0xd8, cborTagVariableSizedStaticType,

			// element type
			0xd8, cborTagPrimitiveStaticType, byte(PrimitiveStaticTypeAnyStruct),

			// Elements. array, 0 items follow
			0x80,

			// false
			0xf4,
			// UTF-8 string, length 3
			0x63,
			// b, a, r
			0x62, 0x61, 0x72,
		}

		version3Encoded := []byte{
			// tag
			0xd8, cborTagDictionaryValue,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// array, 3 items follow
			0x83,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// true
			0xf5,
			// UTF-8 string, length 3
			0x63,
			// f, o, o
			0x66, 0x6f, 0x6f,
			// key 1
			0x1,
			// map, 3 pairs of items follow
			0xa3,
			// UTF-8 string, length 3
			0x63,
			// f, o, o
			0x66, 0x6f, 0x6f,
			// UTF-8 string, length 3
			0x63,
			// b, a, r
			0x62, 0x61, 0x72,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// array, 0 items follow
			0x80,
			// UTF-8 string, length 4
			0x64,
			// t, r, u, e
			0x74, 0x72, 0x75, 0x65,
			// false
			0xf4,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   expected,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   expected,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("temporary address value key string change in format version 2", func(t *testing.T) {
		expected := NewDictionaryValueUnownedNonCopying(
			&sema.DictionaryType{
				KeyType:   &sema.AddressType{},
				ValueType: sema.Int8Type,
			},
			NewAddressValueFromBytes([]byte{0x42}),
			Int8Value(42),
		)
		expected.Keys().modified = false
		expected.modified = false

		testEncodeDecode(t,
			encodeDecodeTest{
				decodeOnly:            true,
				decodedValue:          expected,
				decodeVersionOverride: true,
				decodeVersion:         2,
				encoded: []byte{
					// tag
					0xd8, cborTagDictionaryValue,
					// map, 2 pairs of items follow
					0xa2,
					// key 0
					0x0,
					// array, 1 item follows
					0x81,
					// tag
					0xd8, cborTagAddressValue,
					// byte sequence, length 1
					0x41,
					// address
					0x42,
					// key 1
					0x1,
					// map, 1 pair of items follow
					0xa1,
					// UTF-8 string, length 16
					0x70,
					// "0000000000000042"
					0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x34, 0x32,
					// tag
					0xd8, cborTagInt8Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			})
	})

}

func TestEncodeDecodeComposite(t *testing.T) {

	t.Parallel()

	t.Run("empty structure, string location, qualified identifier", func(t *testing.T) {
		expected := NewCompositeValue(
			utils.TestLocation,
			"TestStruct",
			common.CompositeKindStructure,
			NewStringValueOrderedMap(),
			nil,
		)
		expected.modified = false

		encoded := []byte{
			// tag
			0xd8, cborTagCompositeValue,
			// array, 5 items follow
			0x85,

			// tag
			0xd8, cborTagStringLocation,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,

			// nil
			0xf6,

			// positive integer 1
			0x1,

			// array, 0 items follow
			0x80,

			// UTF-8 string, length 10
			0x6a,
			0x54, 0x65, 0x73, 0x74, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
		}

		version3Encoded := []byte{
			// tag
			0xd8, cborTagCompositeValue,
			// map, 4 pairs of items follow
			0xa4,
			// key 0
			0x0,
			// tag
			0xd8, cborTagStringLocation,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// key 2
			0x2,
			// positive integer 1
			0x1,
			// key 3
			0x3,
			// map, 0 pairs of items follow
			0xa0,
			// key 4
			0x4,
			// UTF-8 string, length 10
			0x6a,
			0x54, 0x65, 0x73, 0x74, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   expected,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   expected,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("empty structure, string location, type ID, version <= 3", func(t *testing.T) {
		expected := NewCompositeValue(
			utils.TestLocation,
			"TestStruct",
			common.CompositeKindStructure,
			NewStringValueOrderedMap(),
			nil,
		)
		expected.modified = false

		testEncodeDecode(t,
			encodeDecodeTest{
				decodeVersionOverride: true,
				decodeVersion:         3,
				decodeOnly:            true,
				decodedValue:          expected,
				encoded: []byte{
					// tag
					0xd8, cborTagCompositeValue,
					// map, 4 pairs of items follow
					0xa4,
					// key 0
					0x0,
					// tag
					0xd8, cborTagStringLocation,
					// UTF-8 string, length 4
					0x64,
					// t, e, s, t
					0x74, 0x65, 0x73, 0x74,
					// key 1
					0x1,
					// UTF-8 string, length 17
					0x71,
					0x53, 0x2e, 0x74, 0x65,
					0x73, 0x74, 0x2e, 0x54,
					0x65, 0x73, 0x74, 0x53,
					0x74, 0x72, 0x75, 0x63,
					0x74,
					// key 2
					0x2,
					// positive integer 1
					0x1,
					// key 3
					0x3,
					// map, 0 pairs of items follow
					0xa0,
				},
			},
		)
	})

	t.Run("empty structure, address location without name, version <= 3", func(t *testing.T) {
		expected := NewCompositeValue(
			common.AddressLocation{
				Address: common.BytesToAddress([]byte{0x1}),
				Name:    "SimpleStruct",
			},
			"SimpleStruct",
			common.CompositeKindStructure,
			NewStringValueOrderedMap(),
			nil,
		)
		expected.modified = false

		testEncodeDecode(t,
			encodeDecodeTest{
				decodeOnly:            true,
				decodedValue:          expected,
				decodeVersionOverride: true,
				decodeVersion:         3,
				encoded: []byte{
					// tag
					0xd8, cborTagCompositeValue,
					// map, 4 pairs of items follow
					0xa4,
					// key 0
					0x0,
					// tag
					0xd8, cborTagAddressLocation,
					// byte sequence, length 1
					0x41,
					// positive integer 1
					0x1,
					// key 1
					0x1,
					// UTF-8 string, length 31
					0x78, 0x1F,
					// A.0000000000000001.SimpleStruct
					0x41,
					0x2E,
					0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30,
					0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x31,
					0x2E,
					0x53, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
					// key 2
					0x2,
					// positive integer 1
					0x1,
					// key 3
					0x3,
					// map, 0 pairs of items follow
					0xa0,
				},
			},
		)
	})

	t.Run("non-empty resource, qualified identifier", func(t *testing.T) {
		stringValue := NewStringValue("test")

		members := NewStringValueOrderedMap()
		members.Set("string", stringValue)
		members.Set("true", BoolValue(true))

		expected := NewCompositeValue(
			utils.TestLocation,
			"TestResource",
			common.CompositeKindResource,
			members,
			nil,
		)
		expected.modified = false

		encoded := []byte{
			// tag
			0xd8, cborTagCompositeValue,
			// array, 5 items follow
			0x85,

			// tag
			0xd8, cborTagStringLocation,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,

			// nil
			0xf6,

			// positive integer 2
			0x2,

			// array, 4 items follow
			0x84,
			// UTF-8 string, length 6
			0x66,
			// s, t, r, i, n, g
			0x73, 0x74, 0x72, 0x69, 0x6e, 0x67,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// UTF-8 string, length 4
			0x64,
			// t, r, u, e
			0x74, 0x72, 0x75, 0x65,
			// true
			0xf5,

			// UTF-8 string, length 12
			0x6c,
			0x54, 0x65, 0x73, 0x74, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65,
		}

		version3Encoded := []byte{
			// tag
			0xd8, cborTagCompositeValue,
			// map, 4 pairs of items follow
			0xa4,
			// key 0
			0x0,
			// tag
			0xd8, cborTagStringLocation,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// key 2
			0x2,
			// positive integer 2
			0x2,
			// key 3
			0x3,
			// map, 2 pairs of items follow
			0xa2,
			// UTF-8 string, length 4
			0x64,
			// t, r, u, e
			0x74, 0x72, 0x75, 0x65,
			// true
			0xf5,
			// UTF-8 string, length 6
			0x66,
			// s, t, r, i, n, g
			0x73, 0x74, 0x72, 0x69, 0x6e, 0x67,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// key 4
			0x4,
			// UTF-8 string, length 12
			0x6c,
			0x54, 0x65, 0x73, 0x74, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   expected,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   expected,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("non-empty resource, type ID, version <= 3", func(t *testing.T) {
		stringValue := NewStringValue("test")

		members := NewStringValueOrderedMap()
		members.Set("string", stringValue)
		members.Set("true", BoolValue(true))

		expected := NewCompositeValue(
			utils.TestLocation,
			"TestResource",
			common.CompositeKindResource,
			members,
			nil,
		)
		expected.modified = false

		testEncodeDecode(t,
			encodeDecodeTest{
				decodeVersionOverride: true,
				decodeVersion:         3,
				decodeOnly:            true,
				decodedValue:          expected,
				encoded: []byte{
					// tag
					0xd8, cborTagCompositeValue,
					// map, 4 pairs of items follow
					0xa4,
					// key 0
					0x0,
					// tag
					0xd8, cborTagStringLocation,
					// UTF-8 string, length 4
					0x64,
					// t, e, s, t
					0x74, 0x65, 0x73, 0x74,
					// key 1
					0x1,
					// UTF-8 string, length 19
					0x73,
					0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x54,
					0x65, 0x73, 0x74, 0x52, 0x65, 0x73, 0x6f, 0x75,
					0x72, 0x63, 0x65,
					// key 2
					0x2,
					// positive integer 2
					0x2,
					// key 3
					0x3,
					// map, 2 pairs of items follow
					0xa2,
					// UTF-8 string, length 4
					0x64,
					// t, r, u, e
					0x74, 0x72, 0x75, 0x65,
					// true
					0xf5,
					// UTF-8 string, length 6
					0x66,
					// s, t, r, i, n, g
					0x73, 0x74, 0x72, 0x69, 0x6e, 0x67,
					// UTF-8 string, length 4
					0x64,
					// t, e, s, t
					0x74, 0x65, 0x73, 0x74,
				},
			},
		)
	})

	t.Run("empty, address location, nested, version <= 3", func(t *testing.T) {

		expected := NewCompositeValue(
			common.AddressLocation{
				Address: common.BytesToAddress([]byte{0x1}),
				// NOTE: not stored, inferred from type ID
				Name: "TestContract",
			},
			"TestContract.TestStruct",
			common.CompositeKindStructure,
			NewStringValueOrderedMap(),
			nil,
		)
		expected.modified = false

		testEncodeDecode(t,
			encodeDecodeTest{
				decodeVersionOverride: true,
				decodeVersion:         3,
				decodedValue:          expected,
				encoded: []byte{
					// tag
					0xd8, cborTagCompositeValue,
					// map, 4 pairs of items follow
					0xa4,
					// key 0
					0x0,
					// tag
					0xd8, cborTagAddressLocation,
					// byte sequence, length 1
					0x41,
					// positive integer 1
					0x1,
					// key 1
					0x1,
					// UTF-8 string, length 42
					0x78, 0x2a,
					0x41,
					0x2e,
					0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30,
					0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x31,
					0x2e,
					0x54, 0x65, 0x73, 0x74, 0x43, 0x6F, 0x6E, 0x74, 0x72, 0x61, 0x63, 0x74,
					0x2e,
					0x54, 0x65, 0x73, 0x74, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
					// key 2
					0x2,
					// positive integer 1
					0x1,
					// key 3
					0x3,
					// map, 0 pairs of items follow
					0xa0,
				},
				decodeOnly: true,
			},
		)
	})

	t.Run("empty, address location, address too long, version <= 3", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				decodeVersionOverride: true,
				decodeVersion:         3,
				encoded: []byte{
					// tag
					0xd8, cborTagCompositeValue,
					// map, 4 pairs of items follow
					0xa4,
					// key 0
					0x0,
					// tag
					0xd8, cborTagAddressLocation,
					// byte sequence, length 22
					0x56,
					// address
					0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
					0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
					0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
					// key 1
					0x1,
					// UTF-8 string, length 16
					0x70,
					0x41, 0x2e, 0x30, 0x78, 0x31, 0x2e, 0x54, 0x65,
					0x73, 0x74, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
					// key 2
					0x2,
					// positive integer 1
					0x1,
					// key 3
					0x3,
					// map, 0 pairs of items follow
					0xa0,
				},
				invalid: true,
			},
		)
	})

	t.Run("empty, address location", func(t *testing.T) {
		expected := NewCompositeValue(
			common.AddressLocation{
				Address: common.BytesToAddress([]byte{0x1}),
				Name:    "TestStruct",
			},
			"TestStruct",
			common.CompositeKindStructure,
			NewStringValueOrderedMap(),
			nil,
		)
		expected.modified = false

		encoded := []byte{
			// tag
			0xd8, cborTagCompositeValue,
			// array, 5 items follow
			0x85,

			// tag
			0xd8, cborTagAddressLocation,
			// array, 2 items follow
			0x82,
			// byte sequence, length 1
			0x41,
			// positive integer 1
			0x1,
			// UTF-8 string, length 10
			0x6a,
			0x54, 0x65, 0x73, 0x74, 0x53, 0x74, 0x72, 0x75,
			0x63, 0x74,

			// nil
			0xf6,

			// positive integer 1
			0x1,

			// array, 0 items follow
			0x80,

			// UTF-8 string, length 10
			0x6a,
			0x54, 0x65, 0x73, 0x74, 0x53, 0x74, 0x72, 0x75,
			0x63, 0x74,
		}

		version3Encoded := []byte{
			// tag
			0xd8, cborTagCompositeValue,
			// map, 4 pairs of items follow
			0xa4,
			// key 0
			0x0,
			// tag
			0xd8, cborTagAddressLocation,
			// map, 4 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// byte sequence, length 1
			0x41,
			// positive integer 1
			0x1,
			// key 1
			0x1,
			// UTF-8 string, length 10
			0x6a,
			0x54, 0x65, 0x73, 0x74, 0x53, 0x74, 0x72, 0x75,
			0x63, 0x74,
			// key 2
			0x2,
			// positive integer 1
			0x1,
			// key 3
			0x3,
			// map, 0 pairs of items follow
			0xa0,
			// key 4
			0x4,
			// UTF-8 string, length 10
			0x6a,
			0x54, 0x65, 0x73, 0x74, 0x53, 0x74, 0x72, 0x75,
			0x63, 0x74,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   expected,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   expected,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("empty, address location, address too long, version <= 3", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagCompositeValue,
					// map, 4 pairs of items follow
					0xa4,
					// key 0
					0x0,
					// tag
					0xd8, cborTagAddressLocation,
					// map, 2 pairs of items follow
					0xa2,
					// key 0
					0x0,
					// byte sequence, length 22
					0x56,
					// address
					0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
					0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
					0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
					// key 1
					0x1,
					// UTF-8 string, length 10
					0x6a,
					0x54, 0x65, 0x73, 0x74, 0x53, 0x74, 0x72, 0x75,
					0x63, 0x74,
					// key 1
					0x1,
					// UTF-8 string, length 17
					0x71,
					0x41, 0x43, 0x2e, 0x30, 0x78, 0x31, 0x2e, 0x54,
					0x65, 0x73, 0x74, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
					// key 2
					0x2,
					// positive integer 1
					0x1,
					// key 3
					0x3,
					// map, 0 pairs of items follow
					0xa0,
				},
				invalid:               true,
				decodeVersionOverride: true,
				decodeVersion:         3,
			},
		)
	})
}

func TestEncodeDecodeIntValue(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewIntValueFromInt64(0),
				encoded: []byte{
					0xd8, cborTagIntValue,
					// positive bignum
					0xc2,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewIntValueFromInt64(42),
				encoded: []byte{
					0xd8, cborTagIntValue,
					// positive bignum
					0xc2,
					// byte string, length 1
					0x41,
					0x2a,
				},
			},
		)
	})

	t.Run("negative one", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewIntValueFromInt64(-1),
				encoded: []byte{
					0xd8, cborTagIntValue,
					// negative bignum
					0xc3,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("negative one, version < 2", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewIntValueFromInt64(-1),
				encoded: []byte{
					0xd8, cborTagIntValue,
					// negative bignum
					0xc3,
					// byte string, length 1
					0x41,
					0x1,
				},
				decodeOnly:            true,
				decodeVersionOverride: true,
				decodeVersion:         1,
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewIntValueFromInt64(-42),
				encoded: []byte{
					0xd8, cborTagIntValue,
					// negative bignum
					0xc3,
					// byte string, length 1
					0x41,
					// `-42` in decimal is is `0x2a` in hex.
					// CBOR requires negative values to be encoded as `-1-n`, which is `-n - 1`,
					// which is `0x2a - 0x01`, which equals to `0x29`.
					0x29,
				},
			},
		)
	})

	t.Run("negative, version < 2", func(t *testing.T) {

		// negative bignums were encoded incorrectly in version < 2:
		// https://tools.ietf.org/html/rfc7049#section-2.4.2:
		// "For tag value 3, the value of the bignum is -1 - n."
		// However, the value was incorrectly encoded as just -n.

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewIntValueFromInt64(-42),
				encoded: []byte{
					0xd8, cborTagIntValue,
					// negative bignum
					0xc3,
					// byte string, length 1
					0x41,
					0x2a,
				},
				decodeOnly:            true,
				decodeVersionOverride: true,
				decodeVersion:         1,
			},
		)
	})

	t.Run("negative, large (> 64 bit)", func(t *testing.T) {
		setString, ok := new(big.Int).SetString("-18446744073709551617", 10)
		require.True(t, ok)

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewIntValueFromBigInt(setString),
				encoded: []byte{
					0xd8, cborTagIntValue,
					// negative bignum
					0xc3,
					// byte string, length 9
					0x49,
					0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				},
			},
		)
	})

	t.Run("positive, large (> 64 bit)", func(t *testing.T) {
		bigInt, ok := new(big.Int).SetString("18446744073709551616", 10)
		require.True(t, ok)

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewIntValueFromBigInt(bigInt),
				encoded: []byte{
					// tag
					0xd8, cborTagIntValue,
					// positive bignum
					0xc2,
					// byte string, length 9
					0x49,
					0x01, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				},
			},
		)
	})
}

func TestEncodeDecodeInt8Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int8Value(0),
				encoded: []byte{
					// tag
					0xd8, cborTagInt8Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int8Value(-42),
				encoded: []byte{
					// tag
					0xd8, cborTagInt8Value,
					// negative integer 42
					0x38,
					0x29,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int8Value(42),
				encoded: []byte{
					// tag
					0xd8, cborTagInt8Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("min", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int8Value(math.MinInt8),
				encoded: []byte{
					// tag
					0xd8, cborTagInt8Value,
					// negative integer 0x7f
					0x38,
					0x7f,
				},
			},
		)
	})

	t.Run("<min", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagInt8Value,
					// negative integer 0xf00
					0x38,
					0xff,
				},
				invalid: true,
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int8Value(math.MaxInt8),
				encoded: []byte{
					// tag
					0xd8, cborTagInt8Value,
					// positive integer 0x7f00
					0x18,
					0x7f,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagInt8Value,
					// positive integer 0xff
					0x18,
					0xff,
				},
				invalid: true,
			},
		)
	})
}

func TestEncodeDecodeInt16Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int16Value(0),
				encoded: []byte{
					// tag
					0xd8, cborTagInt16Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int16Value(-42),
				encoded: []byte{
					// tag
					0xd8, cborTagInt16Value,
					// negative integer 42
					0x38,
					0x29,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int16Value(42),
				encoded: []byte{
					// tag
					0xd8, cborTagInt16Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("min", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int16Value(math.MinInt16),
				encoded: []byte{
					// tag
					0xd8, cborTagInt16Value,
					// negative integer 0x7fff
					0x39,
					0x7f, 0xff,
				},
			},
		)
	})

	t.Run("<min", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagInt16Value,
					// negative integer 0xffff
					0x39,
					0xff, 0xff,
				},
				invalid: true,
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int16Value(math.MaxInt16),
				encoded: []byte{
					// tag
					0xd8, cborTagInt16Value,
					// positive integer 0x7fff
					0x19,
					0x7f, 0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagInt16Value,
					// positive integer 0xffff
					0x19,
					0xff, 0xff,
				},
				invalid: true,
			},
		)
	})
}

func TestEncodeDecodeInt32Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int32Value(0),
				encoded: []byte{
					// tag
					0xd8, cborTagInt32Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int32Value(-42),
				encoded: []byte{
					// tag
					0xd8, cborTagInt32Value,
					// negative integer 42
					0x38,
					0x29,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int32Value(42),
				encoded: []byte{
					// tag
					0xd8, cborTagInt32Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("min", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int32Value(math.MinInt32),
				encoded: []byte{
					// tag
					0xd8, cborTagInt32Value,
					// negative integer 0x7fffffff
					0x3a,
					0x7f, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run("<min", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagInt32Value,
					// negative integer 0xffffffff
					0x3a,
					0xff, 0xff, 0xff, 0xff,
				},
				invalid: true,
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int32Value(math.MaxInt32),
				encoded: []byte{
					// tag
					0xd8, cborTagInt32Value,
					// positive integer 0x7fffffff
					0x1a,
					0x7f, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagInt32Value,
					// positive integer 0xffffffff
					0x1a,
					0xff, 0xff, 0xff, 0xff,
				},
				invalid: true,
			},
		)
	})
}

func TestEncodeDecodeInt64Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int64Value(0),
				encoded: []byte{
					// tag
					0xd8, cborTagInt64Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int64Value(-42),
				encoded: []byte{
					// tag
					0xd8, cborTagInt64Value,
					// negative integer 42
					0x38,
					0x29,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int64Value(42),
				encoded: []byte{
					// tag
					0xd8, cborTagInt64Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("min", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int64Value(math.MinInt64),
				encoded: []byte{
					// tag
					0xd8, cborTagInt64Value,
					// negative integer: 0x7fffffffffffffff
					0x3b,
					0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run("<min", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagInt64Value,
					// negative integer 0xffffffffffffffff
					0x3b,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
				invalid: true,
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Int64Value(math.MaxInt64),
				encoded: []byte{
					// tag
					0xd8, cborTagInt64Value,
					// positive integer: 0x7fffffffffffffff
					0x1b,
					0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagInt64Value,
					// positive integer 0xffffffffffffffff
					0x1b,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
				invalid: true,
			},
		)
	})
}

func TestEncodeDecodeInt128Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt128ValueFromInt64(0),
				encoded: []byte{
					0xd8, cborTagInt128Value,
					// positive bignum
					0xc2,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt128ValueFromInt64(42),
				encoded: []byte{
					0xd8, cborTagInt128Value,
					// positive bignum
					0xc2,
					// byte string, length 1
					0x41,
					0x2a,
				},
			},
		)
	})

	t.Run("negative one", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt128ValueFromInt64(-1),
				encoded: []byte{
					0xd8, cborTagInt128Value,
					// negative bignum
					0xc3,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("negative one, version < 2", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt128ValueFromInt64(-1),
				encoded: []byte{
					0xd8, cborTagInt128Value,
					// negative bignum
					0xc3,
					// byte string, length 1
					0x41,
					0x1,
				},
				decodeOnly:            true,
				decodeVersionOverride: true,
				decodeVersion:         1,
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt128ValueFromInt64(-42),
				encoded: []byte{
					0xd8, cborTagInt128Value,
					// negative bignum
					0xc3,
					// byte string, length 1
					0x41,
					0x29,
				},
			},
		)
	})

	t.Run("negative, version < 2", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt128ValueFromInt64(-42),
				encoded: []byte{
					0xd8, cborTagInt128Value,
					// negative bignum
					0xc3,
					// byte string, length 1
					0x41,
					0x2a,
				},
				decodeOnly:            true,
				decodeVersionOverride: true,
				decodeVersion:         1,
			},
		)
	})

	t.Run("min", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
				encoded: []byte{
					0xd8, cborTagInt128Value,
					// negative bignum
					0xc3,
					// byte string, length 16
					0x50,
					0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run("min, version < 2", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
				encoded: []byte{
					0xd8, cborTagInt128Value,
					// negative bignum
					0xc3,
					// byte string, length 16
					0x50,
					0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				},
				decodeOnly:            true,
				decodeVersionOverride: true,
				decodeVersion:         1,
			},
		)
	})

	t.Run("<min", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, cborTagInt128Value,
					// negative bignum
					0xc3,
					// byte string, length 16
					0x50,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
				invalid: true,
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
				encoded: []byte{
					0xd8, cborTagInt128Value,
					// positive bignum
					0xc2,
					// byte string, length 16
					0x50,
					0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, cborTagInt128Value,
					// positive bignum
					0xc2,
					// byte string, length 16
					0x50,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
				invalid: true,
			},
		)
	})

	t.Run("RFC", func(t *testing.T) {
		rfcValue, ok := new(big.Int).SetString("18446744073709551616", 10)
		require.True(t, ok)

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt128ValueFromBigInt(rfcValue),
				encoded: []byte{
					// tag
					0xd8, cborTagInt128Value,
					// positive bignum
					0xc2,
					// byte string, length 9
					0x49,
					0x01, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				},
			},
		)
	})
}

func TestEncodeDecodeInt256Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt256ValueFromInt64(0),
				encoded: []byte{
					0xd8, cborTagInt256Value,
					// positive bignum
					0xc2,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt256ValueFromInt64(42),
				encoded: []byte{
					0xd8, cborTagInt256Value,
					// positive bignum
					0xc2,
					// byte string, length 1
					0x41,
					0x2a,
				},
			},
		)
	})

	t.Run("negative one", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt256ValueFromInt64(-1),
				encoded: []byte{
					0xd8, cborTagInt256Value,
					// negative bignum
					0xc3,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("negative one, version < 2", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt256ValueFromInt64(-1),
				encoded: []byte{
					0xd8, cborTagInt256Value,
					// negative bignum
					0xc3,
					// byte string, length 1
					0x41,
					0x1,
				},
				decodeOnly:            true,
				decodeVersionOverride: true,
				decodeVersion:         1,
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt256ValueFromInt64(-42),
				encoded: []byte{
					0xd8, cborTagInt256Value,
					// negative bignum
					0xc3,
					// byte string, length 1
					0x41,
					0x29,
				},
			},
		)
	})

	t.Run("negative, version < 2", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt256ValueFromInt64(-42),
				encoded: []byte{
					0xd8, cborTagInt256Value,
					// negative bignum
					0xc3,
					// byte string, length 1
					0x41,
					0x2a,
				},
				decodeOnly:            true,
				decodeVersionOverride: true,
				decodeVersion:         1,
			},
		)
	})

	t.Run("min", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
				encoded: []byte{
					0xd8, cborTagInt256Value,
					// negative bignum
					0xc3,
					// byte string, length 32
					0x58, 0x20,
					0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run("min, version < 2", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
				encoded: []byte{
					0xd8, cborTagInt256Value,
					// negative bignum
					0xc3,
					// byte string, length 32
					0x58, 0x20,
					0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				},
				decodeOnly:            true,
				decodeVersionOverride: true,
				decodeVersion:         1,
			},
		)
	})

	t.Run("<min", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, cborTagInt256Value,
					// negative bignum
					0xc3,
					// byte string, length 32
					0x58, 0x20,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
				invalid: true,
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
				encoded: []byte{
					0xd8, cborTagInt256Value,
					// positive bignum
					0xc2,
					// byte string, length 32
					0x58, 0x20,
					0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, cborTagInt256Value,
					// positive bignum
					0xc2,
					// byte string, length 32
					0x58, 0x20,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
				invalid: true,
			},
		)
	})

	t.Run("RFC", func(t *testing.T) {

		rfcValue, ok := new(big.Int).SetString("18446744073709551616", 10)
		require.True(t, ok)

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewInt256ValueFromBigInt(rfcValue),
				encoded: []byte{
					// tag
					0xd8, cborTagInt256Value,
					// positive bignum
					0xc2,
					// byte string, length 9
					0x49,
					0x01, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				},
			},
		)
	})
}

func TestEncodeDecodeUIntValue(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUIntValueFromUint64(0),
				encoded: []byte{
					0xd8, cborTagUIntValue,
					// positive bignum
					0xc2,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, cborTagUIntValue,
					// negative bignum
					0xc3,
					// byte string, length 1
					0x41,
					0x2a,
				},
				invalid: true,
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUIntValueFromUint64(42),
				encoded: []byte{
					0xd8, cborTagUIntValue,
					// positive bignum
					0xc2,
					// byte string, length 1
					0x41,
					0x2a,
				},
			},
		)
	})

	t.Run("RFC", func(t *testing.T) {

		rfcValue, ok := new(big.Int).SetString("18446744073709551616", 10)
		require.True(t, ok)

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUIntValueFromBigInt(rfcValue),
				encoded: []byte{
					// tag
					0xd8, cborTagUIntValue,
					// positive bignum
					0xc2,
					// byte string, length 9
					0x49,
					0x01, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				},
			},
		)
	})
}

func TestEncodeDecodeUInt8Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: UInt8Value(0),
				encoded: []byte{
					// tag
					0xd8, cborTagUInt8Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagUInt8Value,
					// negative integer 42
					0x38,
					0x29,
				},
				invalid: true,
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: UInt8Value(42),
				encoded: []byte{
					// tag
					0xd8, cborTagUInt8Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: UInt8Value(math.MaxUint8),
				encoded: []byte{
					// tag
					0xd8, cborTagUInt8Value,
					// positive integer 0xff
					0x18,
					0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagUInt8Value,
					// positive integer 0xffff
					0x19,
					0xff, 0xff,
				},
				invalid: true,
			},
		)
	})
}

func TestEncodeDecodeUInt16Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: UInt16Value(0),
				encoded: []byte{
					// tag
					0xd8, cborTagUInt16Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagUInt16Value,
					// negative integer 42
					0x38,
					0x29,
				},
				invalid: true,
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: UInt16Value(42),
				encoded: []byte{
					// tag
					0xd8, cborTagUInt16Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: UInt16Value(math.MaxUint16),
				encoded: []byte{
					// tag
					0xd8, cborTagUInt16Value,
					// positive integer 0xffff
					0x19,
					0xff, 0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagUInt16Value,
					// positive integer 0xffffffff
					0x1a,
					0xff, 0xff, 0xff, 0xff,
				},
				invalid: true,
			},
		)
	})
}

func TestEncodeDecodeUInt32Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: UInt32Value(0),
				encoded: []byte{
					// tag
					0xd8, cborTagUInt32Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagUInt32Value,
					// negative integer 42
					0x38,
					0x29,
				},
				invalid: true,
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: UInt32Value(42),
				encoded: []byte{
					// tag
					0xd8, cborTagUInt32Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: UInt32Value(math.MaxUint32),
				encoded: []byte{
					// tag
					0xd8, cborTagUInt32Value,
					// positive integer 0xffffffff
					0x1a,
					0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagUInt32Value,
					// positive integer 0xffffffffffffffff
					0x1b,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
				invalid: true,
			},
		)
	})
}

func TestEncodeDecodeUInt64Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: UInt64Value(0),
				encoded: []byte{
					// tag
					0xd8, cborTagUInt64Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagUInt64Value,
					// negative integer 42
					0x38,
					0x29,
				},
				invalid: true,
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: UInt64Value(42),
				encoded: []byte{
					// tag
					0xd8, cborTagUInt64Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: UInt64Value(math.MaxUint64),
				encoded: []byte{
					// tag
					0xd8, cborTagUInt64Value,
					// positive integer 0xffffffffffffffff
					0x1b,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})
}

func TestEncodeDecodeUInt128Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUInt128ValueFromUint64(0),
				encoded: []byte{
					0xd8, cborTagUInt128Value,
					// positive bignum
					0xc2,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUInt128ValueFromUint64(42),
				encoded: []byte{
					0xd8, cborTagUInt128Value,
					// positive bignum
					0xc2,
					// byte string, length 1
					0x41,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
				encoded: []byte{
					0xd8, cborTagUInt128Value,
					// positive bignum
					0xc2,
					// byte string, length 16
					0x50,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, cborTagUInt128Value,
					// negative bignum
					0xc3,
					// byte string, length 1
					0x41,
					0x2a,
				},
				invalid: true,
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, cborTagUInt128Value,
					// positive bignum
					0xc2,
					// byte string, length 17
					0x51,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff,
				},
				invalid: true,
			},
		)
	})

	t.Run("RFC", func(t *testing.T) {
		rfcValue, ok := new(big.Int).SetString("18446744073709551616", 10)
		require.True(t, ok)

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUInt128ValueFromBigInt(rfcValue),
				encoded: []byte{
					// tag
					0xd8, cborTagUInt128Value,
					// positive bignum
					0xc2,
					// byte string, length 9
					0x49,
					0x01, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				},
			},
		)
	})
}

func TestEncodeDecodeUInt256Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUInt256ValueFromUint64(0),
				encoded: []byte{
					0xd8, cborTagUInt256Value,
					// positive bignum
					0xc2,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUInt256ValueFromUint64(42),
				encoded: []byte{
					0xd8, cborTagUInt256Value,
					// positive bignum
					0xc2,
					// byte string, length 1
					0x41,
					0x2a,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, cborTagUInt256Value,
					// negative bignum
					0xc3,
					// byte string, length 1
					0x41,
					0x2a,
				},
				invalid: true,
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, cborTagUInt256Value,
					// positive bignum
					0xc2,
					// byte string, length 65
					0x58, 0x41,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff,
				},
				invalid: true,
			},
		)
	})

	t.Run("RFC", func(t *testing.T) {
		rfcValue, ok := new(big.Int).SetString("18446744073709551616", 10)
		require.True(t, ok)

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUInt256ValueFromBigInt(rfcValue),
				encoded: []byte{
					// tag
					0xd8, cborTagUInt256Value,
					// positive bignum
					0xc2,
					// byte string, length 9
					0x49,
					0x01, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				},
			},
		)
	})
}

func TestEncodeDecodeWord8Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Word8Value(0),
				encoded: []byte{
					// tag
					0xd8, cborTagWord8Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagWord8Value,
					// negative integer 42
					0x38,
					0x29,
				},
				invalid: true,
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Word8Value(42),
				encoded: []byte{
					// tag
					0xd8, cborTagWord8Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Word8Value(math.MaxUint8),
				encoded: []byte{
					// tag
					0xd8, cborTagWord8Value,
					// positive integer 0xff
					0x18,
					0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagWord8Value,
					// positive integer 0xffff
					0x19,
					0xff, 0xff,
				},
				invalid: true,
			},
		)
	})
}

func TestEncodeDecodeWord16Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Word16Value(0),
				encoded: []byte{
					// tag
					0xd8, cborTagWord16Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Word16Value(42),
				encoded: []byte{
					// tag
					0xd8, cborTagWord16Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Word16Value(math.MaxUint16),
				encoded: []byte{
					// tag
					0xd8, cborTagWord16Value,
					// positive integer 0xffff
					0x19,
					0xff, 0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagWord16Value,
					// positive integer 0xffffffff
					0x1a,
					0xff, 0xff, 0xff, 0xff,
				},
				invalid: true,
			},
		)
	})
}

func TestEncodeDecodeWord32Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Word32Value(0),
				encoded: []byte{
					// tag
					0xd8, cborTagWord32Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Word32Value(42),
				encoded: []byte{
					// tag
					0xd8, cborTagWord32Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Word32Value(math.MaxUint32),
				encoded: []byte{
					// tag
					0xd8, cborTagWord32Value,
					// positive integer 0xffffffff
					0x1a,
					0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagWord32Value,
					// positive integer 0xffffffffffffffff
					0x1b,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
				invalid: true,
			},
		)
	})
}

func TestEncodeDecodeWord64Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Word64Value(0),
				encoded: []byte{
					// tag
					0xd8, cborTagWord64Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Word64Value(42),
				encoded: []byte{
					// tag
					0xd8, cborTagWord64Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Word64Value(math.MaxUint64),
				encoded: []byte{
					// tag
					0xd8, cborTagWord64Value,
					// positive integer 0xffffffffffffffff
					0x1b,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})
}

func TestEncodeDecodeSomeValue(t *testing.T) {

	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: &SomeValue{
					Value: NilValue{},
				},
				encoded: []byte{
					// tag
					0xd8, cborTagSomeValue,
					// null
					0xf6,
				},
			},
		)
	})

	t.Run("string", func(t *testing.T) {
		expectedString := NewStringValue("test")

		testEncodeDecode(t,
			encodeDecodeTest{
				value: &SomeValue{
					Value: expectedString,
				},
				encoded: []byte{
					// tag
					0xd8, cborTagSomeValue,
					// UTF-8 string, length 4
					0x64,
					// t, e, s, t
					0x74, 0x65, 0x73, 0x74,
				},
			},
		)
	})

	t.Run("bool", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: &SomeValue{
					Value: BoolValue(true),
				},
				encoded: []byte{
					// tag
					0xd8, cborTagSomeValue,
					// true
					0xf5,
				},
			},
		)
	})
}

func TestEncodeDecodeFix64Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Fix64Value(0),
				encoded: []byte{
					// tag
					0xd8, cborTagFix64Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Fix64Value(-42),
				encoded: []byte{
					// tag
					0xd8, cborTagFix64Value,
					// negative integer 42
					0x38,
					0x29,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Fix64Value(42),
				encoded: []byte{
					// tag
					0xd8, cborTagFix64Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("min", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Fix64Value(math.MinInt64),
				encoded: []byte{
					// tag
					0xd8, cborTagFix64Value,
					// negative integer: 0x7fffffffffffffff
					0x3b,
					0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run("<min", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagFix64Value,
					// negative integer 0xffffffffffffffff
					0x3b,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
				invalid: true,
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: Fix64Value(math.MaxInt64),
				encoded: []byte{
					// tag
					0xd8, cborTagFix64Value,
					// positive integer: 0x7fffffffffffffff
					0x1b,
					0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagFix64Value,
					// positive integer 0xffffffffffffffff
					0x1b,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
				invalid: true,
			},
		)
	})

}

func TestEncodeDecodeUFix64Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: UFix64Value(0),
				encoded: []byte{
					// tag
					0xd8, cborTagUFix64Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagUFix64Value,
					// negative integer 42
					0x38,
					0x29,
				},
				invalid: true,
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: UFix64Value(42),
				encoded: []byte{
					// tag
					0xd8, cborTagUFix64Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: UFix64Value(math.MaxUint64),
				encoded: []byte{
					// tag
					0xd8, cborTagUFix64Value,
					// positive integer 0xffffffffffffffff
					0x1b,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})
}

func TestEncodeDecodeAddressValue(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: AddressValue{},
				encoded: []byte{
					// tag
					0xd8, cborTagAddressValue,
					// byte sequence, length 0
					0x40,
				},
			},
		)
	})

	t.Run("non-empty", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: AddressValue(common.BytesToAddress([]byte{0x42})),
				encoded: []byte{
					// tag
					0xd8, cborTagAddressValue,
					// byte sequence, length 1
					0x41,
					// address
					0x42,
				},
			},
		)
	})

	t.Run("with leading zeros", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: AddressValue(common.BytesToAddress([]byte{0x0, 0x42})),
				encoded: []byte{
					// tag
					0xd8, cborTagAddressValue,
					// byte sequence, length 1
					0x41,
					// address
					0x42,
				},
			},
		)
	})

	t.Run("with zeros in-between and at and", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				value: AddressValue(common.BytesToAddress([]byte{0x0, 0x42, 0x0, 0x43, 0x0})),
				encoded: []byte{
					// tag
					0xd8, cborTagAddressValue,
					// byte sequence, length 4
					0x44,
					// address
					0x42, 0x0, 0x43, 0x0,
				},
			},
		)
	})

	t.Run("too long", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, cborTagAddressValue,
					// byte sequence, length 22
					0x56,
					// address
					0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
					0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
					0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
				},
				invalid: true,
			},
		)
	})
}

var privatePathValue = PathValue{
	Domain:     common.PathDomainPrivate,
	Identifier: "foo",
}

var publicPathValue = PathValue{
	Domain:     common.PathDomainPublic,
	Identifier: "bar",
}

func TestEncodeDecodePathValue(t *testing.T) {

	t.Parallel()

	t.Run("private", func(t *testing.T) {
		encoded := []byte{
			// tag
			0xd8, cborTagPathValue,
			// array, 2 items follow
			0x82,
			// positive integer 2
			0x2,
			// UTF-8 string, 3 bytes follow
			0x63,
			// f, o, o
			0x66, 0x6f, 0x6f,
		}

		version3Encoded := []byte{
			// tag
			0xd8, cborTagPathValue,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// positive integer 2
			0x2,
			// key 1
			0x1,
			// UTF-8 string, 3 bytes follow
			0x63,
			// f, o, o
			0x66, 0x6f, 0x6f,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   privatePathValue,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   privatePathValue,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("public", func(t *testing.T) {
		encoded := []byte{
			// tag
			0xd8, cborTagPathValue,
			// array, 2 items follow
			0x82,
			// positive integer 3
			0x3,
			// UTF-8 string, 3 bytes follow
			0x63,
			// b, a, r
			0x62, 0x61, 0x72,
		}

		version3Encoded := []byte{
			// tag
			0xd8, cborTagPathValue,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// positive integer 3
			0x3,
			// key 1
			0x1,
			// UTF-8 string, 3 bytes follow
			0x63,
			// b, a, r
			0x62, 0x61, 0x72,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   publicPathValue,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   publicPathValue,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})
}

func TestEncodeDecodeCapabilityValue(t *testing.T) {

	t.Parallel()

	t.Run("private path, untyped capability, new format", func(t *testing.T) {

		value := CapabilityValue{
			Address: NewAddressValueFromBytes([]byte{0x2}),
			Path:    privatePathValue,
		}

		encoded := []byte{
			// tag
			0xd8, cborTagCapabilityValue,
			// array, 3 items follow
			0x83,
			// tag for address
			0xd8, cborTagAddressValue,
			// byte sequence, length 1
			0x41,
			// address
			0x02,
			// tag for address
			0xd8, cborTagPathValue,
			// array, 2 items follow
			0x82,
			// positive integer 2
			0x2,
			// UTF-8 string, length 3
			0x63,
			// f, o, o
			0x66, 0x6f, 0x6f,
			// nil
			0xf6,
		}

		version3Encoded := []byte{
			// tag
			0xd8, cborTagCapabilityValue,
			// map, 3 pairs of items follow
			0xa3,
			// key 0
			0x0,
			// tag for address
			0xd8, cborTagAddressValue,
			// byte sequence, length 1
			0x41,
			// address
			0x02,
			// key 1
			0x1,
			// tag for address
			0xd8, cborTagPathValue,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// positive integer 2
			0x2,
			// key 1
			0x1,
			// UTF-8 string, length 3
			0x63,
			// f, o, o
			0x66, 0x6f, 0x6f,
			// key 2
			0x2,
			// nil
			0xf6,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("private path, untyped capability, old format", func(t *testing.T) {

		testEncodeDecode(t,
			encodeDecodeTest{
				decodeVersionOverride: true,
				decodeVersion:         3,
				decodeOnly:            true,
				value: CapabilityValue{
					Address: NewAddressValueFromBytes([]byte{0x2}),
					Path:    privatePathValue,
				},
				encoded: []byte{
					// tag
					0xd8, cborTagCapabilityValue,
					// map, 2 pairs of items follow
					0xa2,
					// key 0
					0x0,
					// tag for address
					0xd8, cborTagAddressValue,
					// byte sequence, length 1
					0x41,
					// address
					0x02,
					// key 1
					0x1,
					// tag for address
					0xd8, cborTagPathValue,
					// map, 2 pairs of items follow
					0xa2,
					// key 0
					0x0,
					// positive integer 2
					0x2,
					// key 1
					0x1,
					// UTF-8 string, length 3
					0x63,
					// f, o, o
					0x66, 0x6f, 0x6f,
				},
			},
		)
	})

	t.Run("private path, typed capability", func(t *testing.T) {

		value := CapabilityValue{
			Address:    NewAddressValueFromBytes([]byte{0x2}),
			Path:       privatePathValue,
			BorrowType: PrimitiveStaticTypeBool,
		}

		encoded := []byte{
			// tag
			0xd8, cborTagCapabilityValue,
			// array, 3 items follow
			0x83,
			// tag for address
			0xd8, cborTagAddressValue,
			// byte sequence, length 1
			0x41,
			// address
			0x02,
			// tag for address
			0xd8, cborTagPathValue,
			// aray, 2 items follow
			0x82,
			// positive integer 2
			0x2,
			// UTF-8 string, length 3
			0x63,
			// f, o, o
			0x66, 0x6f, 0x6f,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			// bool
			0x6,
		}

		version3Encoded := []byte{
			// tag
			0xd8, cborTagCapabilityValue,
			// map, 3 pairs of items follow
			0xa3,
			// key 0
			0x0,
			// tag for address
			0xd8, cborTagAddressValue,
			// byte sequence, length 1
			0x41,
			// address
			0x02,
			// key 1
			0x1,
			// tag for address
			0xd8, cborTagPathValue,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// positive integer 2
			0x2,
			// key 1
			0x1,
			// UTF-8 string, length 3
			0x63,
			// f, o, o
			0x66, 0x6f, 0x6f,
			// key 2
			0x2,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			// bool
			0x6,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("public path, untyped capability, new format", func(t *testing.T) {
		value := CapabilityValue{
			Address: NewAddressValueFromBytes([]byte{0x3}),
			Path:    publicPathValue,
		}

		encoded := []byte{
			// tag
			0xd8, cborTagCapabilityValue,
			// array, 3 items follow
			0x83,
			// tag for address
			0xd8, cborTagAddressValue,
			// byte sequence, length 1
			0x41,
			// address
			0x03,
			// tag for address
			0xd8, cborTagPathValue,
			// array, 2 items follow
			0x82,
			// positive integer 3
			0x3,
			// UTF-8 string, length 3
			0x63,
			// b, a, r
			0x62, 0x61, 0x72,
			// nil
			0xf6,
		}

		version3Encoded := []byte{
			// tag
			0xd8, cborTagCapabilityValue,
			// map, 3 pairs of items follow
			0xa3,
			// key 0
			0x0,
			// tag for address
			0xd8, cborTagAddressValue,
			// byte sequence, length 1
			0x41,
			// address
			0x03,
			// key 1
			0x1,
			// tag for address
			0xd8, cborTagPathValue,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// positive integer 3
			0x3,
			// key 1
			0x1,
			// UTF-8 string, length 3
			0x63,
			// b, a, r
			0x62, 0x61, 0x72,
			// key 2
			0x2,
			// nil
			0xf6,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("public path, typed capability", func(t *testing.T) {

		value := CapabilityValue{
			Address:    NewAddressValueFromBytes([]byte{0x3}),
			Path:       publicPathValue,
			BorrowType: PrimitiveStaticTypeBool,
		}

		encoded := []byte{
			// tag
			0xd8, cborTagCapabilityValue,
			// array, 3 items follow
			0x83,
			// tag for address
			0xd8, cborTagAddressValue,
			// byte sequence, length 1
			0x41,
			// address
			0x03,
			// tag for address
			0xd8, cborTagPathValue,
			// array, 2 items follow
			0x82,
			// positive integer 3
			0x3,
			// UTF-8 string, length 3
			0x63,
			// b, a, r
			0x62, 0x61, 0x72,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			// bool
			0x6,
		}

		version3Encoded := []byte{
			// tag
			0xd8, cborTagCapabilityValue,
			// map, 3 pairs of items follow
			0xa3,
			// key 0
			0x0,
			// tag for address
			0xd8, cborTagAddressValue,
			// byte sequence, length 1
			0x41,
			// address
			0x03,
			// key 1
			0x1,
			// tag for address
			0xd8, cborTagPathValue,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// positive integer 3
			0x3,
			// key 1
			0x1,
			// UTF-8 string, length 3
			0x63,
			// b, a, r
			0x62, 0x61, 0x72,
			// key 2
			0x2,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			// bool
			0x6,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("public path, untyped capability, old format", func(t *testing.T) {

		testEncodeDecode(t,
			encodeDecodeTest{
				decodeVersionOverride: true,
				decodeVersion:         3,
				decodeOnly:            true,
				value: CapabilityValue{
					Address: NewAddressValueFromBytes([]byte{0x3}),
					Path:    publicPathValue,
				},
				encoded: []byte{
					// tag
					0xd8, cborTagCapabilityValue,
					// map, 2 pairs of items follow
					0xa2,
					// key 0
					0x0,
					// tag for address
					0xd8, cborTagAddressValue,
					// byte sequence, length 1
					0x41,
					// address
					0x03,
					// key 1
					0x1,
					// tag for address
					0xd8, cborTagPathValue,
					// map, 2 pairs of items follow
					0xa2,
					// key 0
					0x0,
					// positive integer 3
					0x3,
					// key 1
					0x1,
					// UTF-8 string, length 3
					0x63,
					// b, a, r
					0x62, 0x61, 0x72,
				},
			},
		)
	})

	// For testing backward compatibility for native composite types
	t.Run("public path, public account typed capability", func(t *testing.T) {

		capabilityValue := CapabilityValue{
			Address:    NewAddressValueFromBytes([]byte{0x3}),
			Path:       publicPathValue,
			BorrowType: PrimitiveStaticTypePublicAccount,
		}

		encoded := []byte{
			// tag
			0xd8, cborTagCapabilityValue,
			// array, 3 items follow
			0x83,
			// tag for address
			0xd8, cborTagAddressValue,
			// byte sequence, length 1
			0x41,
			// address
			0x03,
			// tag for address
			0xd8, cborTagPathValue,
			// array, 2 items follow
			0x82,
			// positive integer 3
			0x3,
			// UTF-8 string, length 3
			0x63,
			// b, a, r
			0x62, 0x61, 0x72,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			// positive integer to follow
			0x18,
			// public account (tag)
			0x5b,
		}

		version3Encoded := []byte{
			// tag
			0xd8, cborTagCapabilityValue,
			// map, 3 pairs of items follow
			0xa3,
			// key 0
			0x0,
			// tag for address
			0xd8, cborTagAddressValue,
			// byte sequence, length 1
			0x41,
			// address
			0x03,
			// key 1
			0x1,
			// tag for address
			0xd8, cborTagPathValue,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// positive integer 3
			0x3,
			// key 1
			0x1,
			// UTF-8 string, length 3
			0x63,
			// b, a, r
			0x62, 0x61, 0x72,
			// key 2
			0x2,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			// positive integer to follow
			0x18,
			// public account (tag)
			0x5b,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   capabilityValue,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   capabilityValue,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})
}

func TestEncodeDecodeLinkValue(t *testing.T) {

	t.Parallel()

	expectedLinkEncodingPrefix := []byte{
		// tag
		0xd8, cborTagLinkValue,
		// array, 2 items follow
		0x82,
		0xd8, cborTagPathValue,
		// array, 2 items follow
		0x82,
		// positive integer 3
		0x3,
		// UTF-8 string, length 3
		0x63,
		// b, a, r
		0x62, 0x61, 0x72,
	}

	expectedVersion3LinkEncodingPrefix := []byte{
		// tag
		0xd8, cborTagLinkValue,
		// map, 2 pairs of items follow
		0xa2,
		// key 0
		0x0,
		0xd8, cborTagPathValue,
		// map, 2 pairs of items follow
		0xa2,
		// key 0
		0x0,
		// positive integer 3
		0x3,
		// key 1
		0x1,
		// UTF-8 string, length 3
		0x63,
		// b, a, r
		0x62, 0x61, 0x72,
		// key 1
		0x1,
	}

	t.Run("primitive, Bool", func(t *testing.T) {
		value := LinkValue{
			TargetPath: publicPathValue,
			Type:       ConvertSemaToPrimitiveStaticType(sema.BoolType),
		}

		//nolint:gocritic
		encoded := append(
			expectedLinkEncodingPrefix[:],
			// tag
			0xd8, cborTagPrimitiveStaticType,
			0x6,
		)

		//nolint:gocritic
		version3Encoded := append(
			expectedVersion3LinkEncodingPrefix[:],
			// tag
			0xd8, cborTagPrimitiveStaticType,
			0x6,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("optional, primitive, bool", func(t *testing.T) {
		value := LinkValue{
			TargetPath: publicPathValue,
			Type: OptionalStaticType{
				Type: PrimitiveStaticTypeBool,
			},
		}

		encodedType := []byte{
			// tag
			0xd8, cborTagOptionalStaticType,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			0x6,
		}

		//nolint:gocritic
		encoded := append(
			expectedLinkEncodingPrefix[:],
			encodedType...,
		)

		//nolint:gocritic
		version3Encoded := append(
			expectedVersion3LinkEncodingPrefix[:],
			encodedType...,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("composite, struct, qualified identifier", func(t *testing.T) {
		value := LinkValue{
			TargetPath: publicPathValue,
			Type: CompositeStaticType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "SimpleStruct",
			},
		}

		//nolint:gocritic
		encoded := append(
			expectedLinkEncodingPrefix[:],
			// tag
			0xd8, cborTagCompositeStaticType,
			// array, 3 items follow
			0x83,
			// tag
			0xd8, cborTagStringLocation,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// nil
			0xf6,
			// UTF-8 string, length 12
			0x6c,
			// SimpleStruct
			0x53, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
		)

		//nolint:gocritic
		version3Encoded := append(
			expectedVersion3LinkEncodingPrefix[:],
			// tag
			0xd8, cborTagCompositeStaticType,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// tag
			0xd8, cborTagStringLocation,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// key 1
			0x2,
			// UTF-8 string, length 12
			0x6c,
			// SimpleStruct
			0x53, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("composite, struct, type ID", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				decodeVersionOverride: true,
				decodeVersion:         3,
				decodeOnly:            true,
				decodedValue: LinkValue{
					TargetPath: publicPathValue,
					Type: CompositeStaticType{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "SimpleStruct",
					},
				},
				encoded: append(
					expectedVersion3LinkEncodingPrefix[:],
					// tag
					0xd8, cborTagCompositeStaticType,
					// map, 2 pairs of items follow
					0xa2,
					// key 0
					0x0,
					// tag
					0xd8, cborTagStringLocation,
					// UTF-8 string, length 4
					0x64,
					// t, e, s, t
					0x74, 0x65, 0x73, 0x74,
					// key 1
					0x1,
					// UTF-8 string, length 19
					0x73,
					// S.test.SimpleStruct
					0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53,
					0x69, 0x6d, 0x70, 0x6c, 0x65, 0x53, 0x74, 0x72,
					0x75, 0x63, 0x74,
				),
			},
		)
	})

	t.Run("composite, struct, address location without name", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				decodeVersionOverride: true,
				decodeVersion:         3,
				decodeOnly:            true,
				decodedValue: LinkValue{
					TargetPath: publicPathValue,
					Type: CompositeStaticType{
						Location: common.AddressLocation{
							Address: common.BytesToAddress([]byte{0x1}),
							Name:    "SimpleStruct",
						},
						QualifiedIdentifier: "SimpleStruct",
					},
				},
				encoded: append(
					expectedVersion3LinkEncodingPrefix[:],
					// tag
					0xd8, cborTagCompositeStaticType,
					// map, 2 pairs of items follow
					0xa2,
					// key 0
					0x0,
					// tag
					0xd8, cborTagAddressLocation,
					// byte sequence, length 1
					0x41,
					// positive integer 1
					0x1,
					// key 1
					0x1,
					// UTF-8 string, length 31
					0x78, 0x1F,
					// A.0000000000000001.SimpleStruct
					0x41,
					0x2E,
					0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30,
					0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x31,
					0x2E,
					0x53, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
				),
			},
		)
	})

	t.Run("interface, struct, qualified identifier", func(t *testing.T) {
		value := LinkValue{
			TargetPath: publicPathValue,
			Type: InterfaceStaticType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "SimpleInterface",
			},
		}

		//nolint:gocritic
		encoded := append(
			expectedLinkEncodingPrefix[:],
			// tag
			0xd8, cborTagInterfaceStaticType,
			// array, 3 items follow
			0x83,
			// tag
			0xd8, cborTagStringLocation,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// nil
			0xf6,
			// UTF-8 string, length 22
			0x6F,
			// SimpleInterface
			0x53, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65,
		)

		//nolint:gocritic
		version3Encoded := append(
			expectedVersion3LinkEncodingPrefix[:],
			// tag
			0xd8, cborTagInterfaceStaticType,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// tag
			0xd8, cborTagStringLocation,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// key 1
			0x2,
			// UTF-8 string, length 22
			0x6F,
			// SimpleInterface
			0x53, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("interface, struct, type ID", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				decodeVersionOverride: true,
				decodeVersion:         3,
				decodeOnly:            true,
				decodedValue: LinkValue{
					TargetPath: publicPathValue,
					Type: InterfaceStaticType{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "SimpleInterface",
					},
				},
				encoded: append(
					expectedVersion3LinkEncodingPrefix[:],
					// tag
					0xd8, cborTagInterfaceStaticType,
					// map, 2 pairs of items follow
					0xa2,
					// key 0
					0x0,
					// tag
					0xd8, cborTagStringLocation,
					// UTF-8 string, length 4
					0x64,
					// t, e, s, t
					0x74, 0x65, 0x73, 0x74,
					// key 1
					0x1,
					// UTF-8 string, length 22
					0x76,
					// S.test.SimpleInterface
					0x53,
					0x2e,
					0x74, 0x65, 0x73, 0x74,
					0x2e,
					0x53, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65,
				),
			},
		)
	})

	t.Run("interface, struct, address location without name", func(t *testing.T) {
		testEncodeDecode(t,
			encodeDecodeTest{
				decodeVersionOverride: true,
				decodeVersion:         3,
				decodeOnly:            true,
				decodedValue: LinkValue{
					TargetPath: publicPathValue,
					Type: InterfaceStaticType{
						Location: common.AddressLocation{
							Address: common.BytesToAddress([]byte{0x1}),
							Name:    "SimpleInterface",
						},
						QualifiedIdentifier: "SimpleInterface",
					},
				},
				encoded: append(
					expectedVersion3LinkEncodingPrefix[:],
					// tag
					0xd8, cborTagInterfaceStaticType,
					// map, 2 pairs of items follow
					0xa2,
					// key 0
					0x0,
					// tag
					0xd8, cborTagAddressLocation,
					// byte sequence, length 1
					0x41,
					// positive integer 1
					0x1,
					// key 1
					0x1,
					// UTF-8 string, length 34
					0x78, 0x22,
					// A.0000000000000001.SimpleInterface
					0x41,
					0x2E,
					0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30,
					0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x31,
					0x2E,
					0x53, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65,
				),
			},
		)
	})

	t.Run("variable-sized, bool", func(t *testing.T) {
		value := LinkValue{
			TargetPath: publicPathValue,
			Type: VariableSizedStaticType{
				Type: PrimitiveStaticTypeBool,
			},
		}

		//nolint:gocritic
		encoded := append(
			expectedLinkEncodingPrefix[:],
			// tag
			0xd8, cborTagVariableSizedStaticType,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			0x6,
		)

		//nolint:gocritic
		version3Encoded := append(
			expectedVersion3LinkEncodingPrefix[:],
			// tag
			0xd8, cborTagVariableSizedStaticType,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			0x6,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("constant-sized, bool", func(t *testing.T) {
		value := LinkValue{
			TargetPath: publicPathValue,
			Type: ConstantSizedStaticType{
				Type: PrimitiveStaticTypeBool,
				Size: 42,
			},
		}

		//nolint:gocritic
		encoded := append(
			expectedLinkEncodingPrefix[:],
			// tag
			0xd8, cborTagConstantSizedStaticType,
			// array, 2 items follow
			0x82,
			// positive integer 42
			0x18, 0x2A,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			0x6,
		)

		//nolint:gocritic
		version3Encoded := append(
			expectedVersion3LinkEncodingPrefix[:],
			// tag
			0xd8, cborTagConstantSizedStaticType,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// positive integer 42
			0x18, 0x2A,
			// key 1
			0x1,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			0x6,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("reference type, authorized, bool", func(t *testing.T) {
		value := LinkValue{
			TargetPath: publicPathValue,
			Type: ReferenceStaticType{
				Authorized: true,
				Type:       PrimitiveStaticTypeBool,
			},
		}

		//nolint:gocritic
		encoded := append(
			expectedLinkEncodingPrefix[:],
			// tag
			0xd8, cborTagReferenceStaticType,
			// array, 2 items follow
			0x82,
			// true
			0xf5,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			0x6,
		)

		//nolint:gocritic
		version3Encoded := append(
			expectedVersion3LinkEncodingPrefix[:],
			// tag
			0xd8, cborTagReferenceStaticType,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// true
			0xf5,
			// key 1
			0x1,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			0x6,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("reference type, unauthorized, bool", func(t *testing.T) {
		value := LinkValue{
			TargetPath: publicPathValue,
			Type: ReferenceStaticType{
				Authorized: false,
				Type:       PrimitiveStaticTypeBool,
			},
		}

		//nolint:gocritic
		encoded := append(
			expectedLinkEncodingPrefix[:],
			// tag
			0xd8, cborTagReferenceStaticType,
			// array, 2 items follow
			0x82,
			// false
			0xf4,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			0x6,
		)

		//nolint:gocritic
		version3Encoded := append(
			expectedVersion3LinkEncodingPrefix[:],
			// tag
			0xd8, cborTagReferenceStaticType,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// false
			0xf4,
			// key 1
			0x1,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			0x6,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("dictionary, bool, string", func(t *testing.T) {
		value := LinkValue{
			TargetPath: publicPathValue,
			Type: DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeBool,
				ValueType: PrimitiveStaticTypeString,
			},
		}

		//nolint:gocritic
		encoded := append(
			expectedLinkEncodingPrefix[:],
			// tag
			0xd8, cborTagDictionaryStaticType,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			0x6,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			0x8,
		)

		//nolint:gocritic
		version3Encoded := append(
			expectedVersion3LinkEncodingPrefix[:],
			// tag
			0xd8, cborTagDictionaryStaticType,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			0x6,
			// key 1
			0x1,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			0x8,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("restricted", func(t *testing.T) {
		value := LinkValue{
			TargetPath: publicPathValue,
			Type: &RestrictedStaticType{
				Type: CompositeStaticType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "S",
				},
				Restrictions: []InterfaceStaticType{
					{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "I1",
					},
					{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "I2",
					},
				},
			},
		}

		//nolint:gocritic
		encoded := append(
			expectedLinkEncodingPrefix[:],
			// tag
			0xd8, cborTagRestrictedStaticType,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, cborTagCompositeStaticType,
			// array, 3 items follow
			0x83,
			// tag
			0xd8, cborTagStringLocation,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// nil
			0xf6,
			// UTF-8 string, length 1
			0x61,
			// S
			0x53,
			// array, length 2
			0x82,
			// tag
			0xd8, cborTagInterfaceStaticType,
			// array, 3 items follow
			0x83,
			// tag
			0xd8, cborTagStringLocation,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// nil
			0xf6,
			// UTF-8 string, length 2
			0x62,
			// I1
			0x49, 0x31,
			// tag
			0xd8, cborTagInterfaceStaticType,
			// array, 3 items follow
			0x83,
			// tag
			0xd8, cborTagStringLocation,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// nil
			0xf6,
			// UTF-8 string, length 2
			0x62,
			// I2
			0x49, 0x32,
		)

		//nolint:gocritic
		version3Encoded := append(
			expectedVersion3LinkEncodingPrefix[:],
			// tag
			0xd8, cborTagRestrictedStaticType,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// tag
			0xd8, cborTagCompositeStaticType,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// tag
			0xd8, cborTagStringLocation,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// key 2
			0x2,
			// UTF-8 string, length 1
			0x61,
			// S
			0x53,
			// key 1
			0x1,
			// array, length 2
			0x82,
			// tag
			0xd8, cborTagInterfaceStaticType,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// tag
			0xd8, cborTagStringLocation,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// key 2
			0x2,
			// UTF-8 string, length 2
			0x62,
			// I1
			0x49, 0x31,
			// tag
			0xd8, cborTagInterfaceStaticType,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// tag
			0xd8, cborTagStringLocation,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// key 2
			0x2,
			// UTF-8 string, length 2
			0x62,
			// I2
			0x49, 0x32,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("capability, none", func(t *testing.T) {
		value := LinkValue{
			TargetPath: publicPathValue,
			Type:       CapabilityStaticType{},
		}

		//nolint:gocritic
		encoded := append(
			expectedLinkEncodingPrefix[:],
			// tag
			0xd8, cborTagCapabilityStaticType,
			// null
			0xf6,
		)

		//nolint:gocritic
		version3Encoded := append(
			expectedVersion3LinkEncodingPrefix[:],
			// tag
			0xd8, cborTagCapabilityStaticType,
			// null
			0xf6,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("capability, primitive, bool", func(t *testing.T) {
		value := LinkValue{
			TargetPath: publicPathValue,
			Type: CapabilityStaticType{
				BorrowType: PrimitiveStaticTypeBool,
			},
		}

		//nolint:gocritic
		encoded := append(
			expectedLinkEncodingPrefix[:],
			// tag
			0xd8, cborTagCapabilityStaticType,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			0x6,
		)

		//nolint:gocritic
		version3Encoded := append(
			expectedVersion3LinkEncodingPrefix[:],
			// tag
			0xd8, cborTagCapabilityStaticType,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			0x6,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})
}

func TestEncodeDecodeDictionaryDeferred(t *testing.T) {

	t.Run("resource values", func(t *testing.T) {

		key1 := NewStringValue("test")
		value1 := NewCompositeValue(
			utils.TestLocation,
			"R",
			common.CompositeKindResource,
			NewStringValueOrderedMap(),
			nil,
		)
		value1.modified = false

		key2 := BoolValue(true)
		value2 := NewCompositeValue(
			utils.TestLocation,
			"R2",
			common.CompositeKindResource,
			NewStringValueOrderedMap(),
			nil,
		)
		value2.modified = false

		expected := NewDictionaryValueUnownedNonCopying(
			// TODO: type
			&sema.DictionaryType{
				KeyType:   sema.AnyStructType,
				ValueType: sema.AnyResourceType,
			},
			key1, value1,
			key2, value2,
		)
		expected.modified = false
		expected.Keys().modified = false

		deferredKeys := orderedmap.NewStringStructOrderedMap()
		deferredKeys.Set("test", struct{}{})
		deferredKeys.Set("true", struct{}{})

		deferrals := &EncodingDeferrals{
			Values: []EncodingDeferralValue{
				{
					Key:   "v\x1ftest",
					Value: value1,
				},
				{
					Key:   "v\x1ftrue",
					Value: value2,
				},
			},
		}

		decodedValue := &DictionaryValue{
			Type:                   expected.Type,
			keys:                   expected.Keys(),
			entries:                NewStringValueOrderedMap(),
			deferredOwner:          &testOwner,
			deferredKeys:           deferredKeys,
			deferredStorageKeyBase: "v",
		}

		encoded := []byte{
			// tag
			0xd8, cborTagDictionaryValue,
			// array, 2 items follow
			0x82,
			// keys to follow
			// cbor Array Value tag
			0xd8, cborTagArrayValue,
			// array, 2 items follow
			0x82,
			// array type tag
			0xd8, cborTagVariableSizedStaticType,
			// element type
			0xd8, cborTagPrimitiveStaticType, byte(PrimitiveStaticTypeAnyStruct),
			// keys: array, 2 items follow
			0x82,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// true
			0xf5,
			// array, 0 items follow
			0x80,
		}

		version3Encoded := []byte{
			// tag
			0xd8, cborTagDictionaryValue,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// array, 2 items follow
			0x82,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// true
			0xf5,
			// key 1
			0x1,
			// map, 0 pairs of items follow
			0xa0,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				deferred:     true,
				value:        expected,
				encoded:      encoded,
				deferrals:    deferrals,
				decodedValue: decodedValue,
			},
		)

		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				deferred:     true,
				value:        expected,
				encoded:      encoded,
				deferrals:    deferrals,
				decodedValue: decodedValue,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("non-resource values", func(t *testing.T) {

		key1 := NewStringValue("test")
		value1 := NewStringValue("xyz")

		key2 := BoolValue(true)
		value2 := BoolValue(false)

		expected := NewDictionaryValueUnownedNonCopying(
			// TODO: type
			&sema.DictionaryType{
				KeyType:   sema.AnyStructType,
				ValueType: sema.AnyStructType,
			},
			key1, value1,
			key2, value2,
		)
		expected.modified = false
		expected.Keys().modified = false

		deferrals := &EncodingDeferrals{}

		encoded := []byte{
			// tag
			0xd8, cborTagDictionaryValue,
			// array, 2 items follow
			0x82,

			// cbor Array Value tag
			0xd8, cborTagArrayValue,
			// keys
			// array, 2 items follow
			0x82,
			// array type tag
			0xd8, cborTagVariableSizedStaticType,
			// element type
			0xd8, cborTagPrimitiveStaticType, byte(PrimitiveStaticTypeAnyStruct),
			// keys: array, 2 items follow
			0x82,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// true
			0xf5,
			// array, 2 items follow
			0x82,
			// UTF-8 string, length 3
			0x63,
			// x, y, z
			0x78, 0x79, 0x7a,
			// false
			0xf4,
		}

		version3Encoded := []byte{
			// tag
			0xd8, cborTagDictionaryValue,
			// map, 2 pairs of items follow
			0xa2,
			// key 0
			0x0,
			// array, 2 items follow
			0x82,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// true
			0xf5,
			// key 1
			0x1,
			// map, 2 pairs of items follow
			0xa2,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// UTF-8 string, length 3
			0x63,
			// x, y, z
			0x78, 0x79, 0x7a,
			// UTF-8 string, length 4
			0x64,
			// t, r, u, e
			0x74, 0x72, 0x75, 0x65,
			// false
			0xf4,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				deferred:  true,
				value:     expected,
				deferrals: deferrals,
				encoded:   encoded,
			},
		)
		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				deferred:  true,
				value:     expected,
				deferrals: deferrals,
				encoded:   encoded,
			},
			3,
			version3Encoded,
		)
	})
}

func TestEncodeDecodeTypeValue(t *testing.T) {

	t.Parallel()

	t.Run("primitive, Bool", func(t *testing.T) {
		value := TypeValue{
			Type: ConvertSemaToPrimitiveStaticType(sema.BoolType),
		}
		encoded := []byte{
			// tag
			0xd8, cborTagTypeValue,
			// array, 1 items follow
			0x81,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			// positive integer 0
			0x6,
		}
		version3Encoded := []byte{
			// tag
			0xd8, cborTagTypeValue,
			// map, 1 pair of items follow
			0xa1,
			// key 0
			0x0,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			// positive integer 0
			0x6,
		}
		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)
		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("primitive, Int", func(t *testing.T) {
		value := TypeValue{
			Type: ConvertSemaToPrimitiveStaticType(sema.IntType),
		}
		encoded := []byte{
			// tag
			0xd8, cborTagTypeValue,
			// array, 1 items follow
			0x81,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			// positive integer 36
			0x18, 0x24,
		}
		version3Encoded := []byte{
			// tag
			0xd8, cborTagTypeValue,
			// map, 1 pair of items follow
			0xa1,
			// key 0
			0x0,
			// tag
			0xd8, cborTagPrimitiveStaticType,
			// positive integer 36
			0x18, 0x24,
		}
		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)
		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})

	t.Run("without static type", func(t *testing.T) {
		value := TypeValue{
			Type: nil,
		}
		encoded := []byte{
			// tag
			0xd8, cborTagTypeValue,
			// array, 1 items follow
			0x81,
			// nil
			0xf6,
		}
		version3Encoded := []byte{
			// tag
			0xd8, cborTagTypeValue,
			// map, 0 pairs of items follow
			0xa0,
		}
		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)
		testEncodeDecodeOldFormat(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
			3,
			version3Encoded,
		)
	})
}

func TestEncodePrepareCallback(t *testing.T) {

	value := NewArrayValueUnownedNonCopying(
		&sema.VariableSizedType{
			Type: sema.Int8Type,
		},
		Int8Value(42),
	)

	type prepareCallback struct {
		value Value
		path  []string
	}

	var prepareCallbacks []prepareCallback

	data, _, err := EncodeValue(value, nil, false, func(value Value, path []string) {
		prepareCallbacks = append(prepareCallbacks, prepareCallback{
			value: value,
			path:  path,
		})
	})
	require.NoError(t, err)

	require.Equal(t,
		[]prepareCallback{
			{
				value: value,
				path:  nil,
			},
			{
				value: value.Elements()[0],
				path:  []string{"0"},
			},
		},
		prepareCallbacks,
	)

	utils.AssertEqualWithDiff(t,
		[]byte{
			// cbor Array Value tag
			0xd8, cborTagArrayValue,

			// array, 2 items follow
			0x82,

			// Type info

			// array type tag
			0xd8, cborTagVariableSizedStaticType,

			// element type
			0xd8, cborTagPrimitiveStaticType, 0x18, byte(PrimitiveStaticTypeInt8),

			// elements: array with 1 item follow
			0x81,
			// tag
			0xd8, cborTagInt8Value,
			// positive integer 42
			0x18,
			0x2a,
		},
		data,
	)
}

func TestDecodeCallback(t *testing.T) {

	data := []byte{
		// cbor Array Value tag
		0xd8, cborTagArrayValue,

		// array, 2 items follow
		0x82,

		// Type info
		// array type tag
		0xd8, cborTagVariableSizedStaticType,

		// element type
		0xd8, cborTagPrimitiveStaticType, 0x18, byte(PrimitiveStaticTypeInt8),

		// array with 1 item follow
		0x81,
		// tag
		0xd8, cborTagInt8Value,
		// positive integer 42
		0x18,
		0x2a,
	}

	type decodeCallback struct {
		value interface{}
		path  []string
	}

	var decodeCallbacks []decodeCallback

	decoded, err := DecodeValue(data, nil, nil, CurrentEncodingVersion, func(value interface{}, path []string) {
		decodeCallbacks = append(decodeCallbacks, decodeCallback{
			value: value,
			path:  path,
		})
	})
	require.NoError(t, err)

	// build the content
	_ = decoded.String()

	require.Equal(t,
		[]decodeCallback{
			{
				value: &ArrayValue{
					values: []Value{Int8Value(42)},
				},
				path: nil,
			},
			{
				value: Int8Value(42),
				path:  []string{"0"},
			},
		},
		decodeCallbacks,
	)
}

func BenchmarkEncoding(b *testing.B) {

	value := prepareLargeTestValue()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, err := EncodeValue(value, nil, false, nil)
		require.NoError(b, err)
	}
}

func BenchmarkDecoding(b *testing.B) {

	value := prepareLargeTestValue()

	encoded, _, err := EncodeValue(value, nil, false, nil)
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err = DecodeValue(encoded, nil, nil, CurrentEncodingVersion, nil)
		require.NoError(b, err)
	}
}

func prepareLargeTestValue() Value {
	values := NewArrayValueUnownedNonCopying(
		&sema.VariableSizedType{
			Type: sema.AnyStructType,
		},
	)
	for i := 0; i < 100; i++ {
		dict := NewDictionaryValueUnownedNonCopying(
			&sema.DictionaryType{
				KeyType:   sema.StringType,
				ValueType: sema.Int256Type,
			},
		)
		for i := 0; i < 100; i++ {
			key := NewStringValue(fmt.Sprintf("hello world %d", i))
			value := NewInt256ValueFromInt64(int64(i))
			dict.Set(nil, ReturnEmptyLocationRange, key, NewSomeValueOwningNonCopying(value))
		}
		values.Append(dict)
	}
	return values
}
