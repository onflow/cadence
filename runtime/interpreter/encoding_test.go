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

package interpreter_test

import (
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/onflow/atree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

type encodeDecodeTest struct {
	value                atree.Value
	storable             atree.Storable
	encoded              []byte
	invalid              bool
	decodedValue         Value
	decodeOnly           bool
	deepEquality         bool
	storage              Storage
	slabStorageID        atree.StorageID
	maxInlineElementSize uint64
}

var testOwner = common.MustBytesToAddress([]byte{0x42})

func testEncodeDecode(t *testing.T, test encodeDecodeTest) {

	if test.storage == nil {
		test.storage = newUnmeteredInMemoryStorage()
	}

	var encoded []byte
	if (test.value != nil || test.storable != nil) && !test.decodeOnly {

		if test.value != nil {
			if test.storable == nil {
				maxInlineElementSize := test.maxInlineElementSize
				if maxInlineElementSize == 0 {
					maxInlineElementSize = math.MaxUint64
				}
				storable, err := test.value.Storable(
					test.storage,
					atree.Address(testOwner),
					maxInlineElementSize,
				)
				require.NoError(t, err)
				test.storable = storable
			}
		}

		var err error
		encoded, err = atree.Encode(test.storable, CBOREncMode)
		require.NoError(t, err)

		if test.encoded != nil {
			AssertEqualWithDiff(t, test.encoded, encoded)
		}
	} else {
		encoded = test.encoded
	}

	decoder := CBORDecMode.NewByteStreamDecoder(encoded)
	decodedStorable, err := DecodeStorable(decoder, test.slabStorageID, nil)

	if test.invalid {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
		inter, err := NewInterpreter(
			nil,
			TestLocation,
			&Config{
				Storage: test.storage,
			},
		)
		require.NoError(t, err)

		decodedValue, err := decodedStorable.StoredValue(test.storage)
		require.NoError(t, err)

		expectedValue := test.value
		if test.decodedValue != nil {
			expectedValue = test.decodedValue
		}

		if test.deepEquality {
			assert.Equal(t, expectedValue, decodedValue)
		} else {
			if expectedValue, ok := expectedValue.(Value); ok {
				storedValue, err := ConvertStoredValue(nil, decodedValue)
				require.NoError(t, err)
				AssertValuesEqual(t, inter, expectedValue, storedValue)
				return
			}
			assert.Equal(t, expectedValue, decodedValue)
		}
	}
}

func TestEncodeDecodeNilValue(t *testing.T) {

	t.Parallel()

	testEncodeDecode(t,
		encodeDecodeTest{
			value: Nil,
			encoded: []byte{
				// null
				0xf6,
			},
		},
	)
}

func TestEncodeDecodeVoidValue(t *testing.T) {

	t.Parallel()

	testEncodeDecode(t,
		encodeDecodeTest{
			value: Void,
			encoded: []byte{
				// tag
				0xd8, CBORTagVoidValue,
				// null
				0xf6,
			},
		},
	)
}

func TestEncodeDecodeBool(t *testing.T) {

	t.Parallel()

	t.Run("false", func(t *testing.T) {

		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: FalseValue,
				encoded: []byte{
					// false
					0xf4,
				},
			},
		)
	})

	t.Run("true", func(t *testing.T) {

		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: TrueValue,
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

		t.Parallel()

		expected := NewUnmeteredStringValue("")

		testEncodeDecode(t,
			encodeDecodeTest{
				value: expected,
				encoded: []byte{
					// tag
					0xd8, CBORTagStringValue,

					//  UTF-8 string, 0 bytes follow
					0x60,
				},
			})
	})

	t.Run("non-empty", func(t *testing.T) {

		t.Parallel()

		expected := NewUnmeteredStringValue("foo")

		testEncodeDecode(t,
			encodeDecodeTest{
				value: expected,
				encoded: []byte{
					// tag
					0xd8, CBORTagStringValue,

					// UTF-8 string, 3 bytes follow
					0x63,
					// f, o, o
					0x66, 0x6f, 0x6f,
				},
			},
		)
	})

	t.Run("larger than max inline size", func(t *testing.T) {

		t.Parallel()

		maxInlineElementSize := atree.MaxInlineArrayElementSize
		expected := NewUnmeteredStringValue(strings.Repeat("x", int(maxInlineElementSize+1)))

		testEncodeDecode(t,
			encodeDecodeTest{
				value:                expected,
				maxInlineElementSize: maxInlineElementSize,
				encoded: []byte{
					// tag
					0xd8, atree.CBORTagStorageID,

					// storage ID
					0x50, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x42, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
				},
			},
		)
	})
}

func TestEncodeDecodeStringAtreeValue(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {

		t.Parallel()

		expected := StringAtreeValue("")

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

		t.Parallel()

		expected := StringAtreeValue("foo")

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

	t.Run("larger than max inline size", func(t *testing.T) {

		t.Parallel()

		maxInlineElementSize := atree.MaxInlineArrayElementSize
		expected := StringAtreeValue(strings.Repeat("x", int(maxInlineElementSize+1)))

		testEncodeDecode(t,
			encodeDecodeTest{
				value:                expected,
				maxInlineElementSize: maxInlineElementSize,
				encoded: []byte{
					// tag
					0xd8, atree.CBORTagStorageID,

					// storage ID
					0x50, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x42, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
				},
			},
		)
	})
}

func TestEncodeDecodeUint64AtreeValue(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: Uint64AtreeValue(0),
				encoded: []byte{
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// negative integer 42
					0x38,
					0x29,
				},
				invalid: true,
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: Uint64AtreeValue(42),
				encoded: []byte{
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: Uint64AtreeValue(math.MaxUint64),
				encoded: []byte{
					// positive integer 0xffffffffffffffff
					0x1b,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})
}

func TestEncodeDecodeArray(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		expected := NewArrayValue(
			inter,
			EmptyLocationRange,
			&ConstantSizedStaticType{
				Type: PrimitiveStaticTypeAnyStruct,
				Size: 0,
			},
			common.ZeroAddress,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				storage: inter.Storage(),
				value:   expected,
				encoded: []byte{
					// tag
					0xd8, atree.CBORTagStorageID,

					// storage ID
					0x50, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
				},
			})
	})

	t.Run("string and bool", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		expectedString := NewUnmeteredStringValue("test")

		expected := NewArrayValue(
			inter,
			EmptyLocationRange,
			&VariableSizedStaticType{
				Type: PrimitiveStaticTypeAnyStruct,
			},
			common.ZeroAddress,
			expectedString,
			TrueValue,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				storage: inter.Storage(),
				value:   expected,
				encoded: []byte{
					// tag
					0xd8, atree.CBORTagStorageID,

					// storage ID
					0x50, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
				},
			},
		)
	})
}

func TestEncodeDecodeComposite(t *testing.T) {

	t.Parallel()

	t.Run("empty structure, string location, qualified identifier", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		expected := NewCompositeValue(
			inter,
			EmptyLocationRange,
			utils.TestLocation,
			"TestStruct",
			common.CompositeKindStructure,
			nil,
			testOwner,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				storage: inter.Storage(),
				value:   expected,
				encoded: []byte{
					// tag
					0xd8, atree.CBORTagStorageID,

					// storage ID
					0x50, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x42, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
				},
			},
		)
	})

	t.Run("non-empty resource, qualified identifier", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		stringValue := NewUnmeteredStringValue("test")

		fields := []CompositeField{
			{Name: "string", Value: stringValue},
			{Name: "true", Value: TrueValue},
		}

		expected := NewCompositeValue(
			inter,
			EmptyLocationRange,
			utils.TestLocation,
			"TestResource",
			common.CompositeKindResource,
			fields,
			testOwner,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				storage: inter.Storage(),
				value:   expected,
				encoded: []byte{
					// tag
					0xd8, atree.CBORTagStorageID,

					// storage ID
					0x50, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x42, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
				},
			},
		)
	})
}

func TestEncodeDecodeIntValue(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredIntValueFromInt64(0),
				encoded: []byte{
					0xd8, CBORTagIntValue,
					// positive bignum
					0xc2,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredIntValueFromInt64(42),
				encoded: []byte{
					0xd8, CBORTagIntValue,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredIntValueFromInt64(-1),
				encoded: []byte{
					0xd8, CBORTagIntValue,
					// negative bignum
					0xc3,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredIntValueFromInt64(-42),
				encoded: []byte{
					0xd8, CBORTagIntValue,
					// negative bignum
					0xc3,
					// byte string, length 1
					0x41,
					// `-42` in decimal is `0x2a` in hex.
					// CBOR requires negative values to be encoded as `-1-n`, which is `-n - 1`,
					// which is `0x2a - 0x01`, which equals to `0x29`.
					0x29,
				},
			},
		)
	})

	t.Run("negative, large (> 64 bit)", func(t *testing.T) {

		t.Parallel()

		setString, ok := new(big.Int).SetString("-18446744073709551617", 10)
		require.True(t, ok)

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredIntValueFromBigInt(setString),
				encoded: []byte{
					0xd8, CBORTagIntValue,
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

		t.Parallel()

		bigInt, ok := new(big.Int).SetString("18446744073709551616", 10)
		require.True(t, ok)

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredIntValueFromBigInt(bigInt),
				encoded: []byte{
					// tag
					0xd8, CBORTagIntValue,
					// positive bignum
					0xc2,
					// byte string, length 9
					0x49,
					0x01, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				},
			},
		)
	})

	t.Run("larger than max inline size", func(t *testing.T) {

		t.Parallel()

		inter, err := NewInterpreter(nil, nil, &Config{})
		require.NoError(t, err)

		expected := NewUnmeteredIntValueFromInt64(1_000_000_000)

		maxInlineElementSize := atree.MaxInlineArrayElementSize
		for len(expected.BigInt.Bytes()) < int(maxInlineElementSize+1) {
			expected = expected.Mul(inter, expected, EmptyLocationRange).(IntValue)
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:                expected,
				maxInlineElementSize: maxInlineElementSize,
				encoded: []byte{
					// tag
					0xd8, atree.CBORTagStorageID,

					// storage ID
					0x50, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x42, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
				},
			},
		)
	})
}

func TestEncodeDecodeInt8Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt8Value(0),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt8Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt8Value(-42),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt8Value,
					// negative integer 42
					0x38,
					0x29,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt8Value(42),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt8Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("min", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt8Value(math.MinInt8),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt8Value,
					// negative integer 0x7f
					0x38,
					0x7f,
				},
			},
		)
	})

	t.Run("<min", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagInt8Value,
					// negative integer 0xf00
					0x38,
					0xff,
				},
				invalid: true,
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt8Value(math.MaxInt8),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt8Value,
					// positive integer 0x7f00
					0x18,
					0x7f,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagInt8Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt16Value(0),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt16Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt16Value(-42),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt16Value,
					// negative integer 42
					0x38,
					0x29,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt16Value(42),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt16Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("min", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt16Value(math.MinInt16),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt16Value,
					// negative integer 0x7fff
					0x39,
					0x7f, 0xff,
				},
			},
		)
	})

	t.Run("<min", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagInt16Value,
					// negative integer 0xffff
					0x39,
					0xff, 0xff,
				},
				invalid: true,
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt16Value(math.MaxInt16),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt16Value,
					// positive integer 0x7fff
					0x19,
					0x7f, 0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagInt16Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt32Value(0),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt32Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt32Value(-42),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt32Value,
					// negative integer 42
					0x38,
					0x29,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt32Value(42),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt32Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("min", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt32Value(math.MinInt32),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt32Value,
					// negative integer 0x7fffffff
					0x3a,
					0x7f, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run("<min", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagInt32Value,
					// negative integer 0xffffffff
					0x3a,
					0xff, 0xff, 0xff, 0xff,
				},
				invalid: true,
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt32Value(math.MaxInt32),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt32Value,
					// positive integer 0x7fffffff
					0x1a,
					0x7f, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagInt32Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt64Value(0),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt64Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt64Value(-42),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt64Value,
					// negative integer 42
					0x38,
					0x29,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt64Value(42),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt64Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("min", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt64Value(math.MinInt64),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt64Value,
					// negative integer: 0x7fffffffffffffff
					0x3b,
					0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run("<min", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagInt64Value,
					// negative integer 0xffffffffffffffff
					0x3b,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
				invalid: true,
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt64Value(math.MaxInt64),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt64Value,
					// positive integer: 0x7fffffffffffffff
					0x1b,
					0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagInt64Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt128ValueFromInt64(0),
				encoded: []byte{
					0xd8, CBORTagInt128Value,
					// positive bignum
					0xc2,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt128ValueFromInt64(42),
				encoded: []byte{
					0xd8, CBORTagInt128Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt128ValueFromInt64(-1),
				encoded: []byte{
					0xd8, CBORTagInt128Value,
					// negative bignum
					0xc3,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt128ValueFromInt64(-42),
				encoded: []byte{
					0xd8, CBORTagInt128Value,
					// negative bignum
					0xc3,
					// byte string, length 1
					0x41,
					0x29,
				},
			},
		)
	})

	t.Run("min", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
				encoded: []byte{
					0xd8, CBORTagInt128Value,
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

	t.Run("<min", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, CBORTagInt128Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
				encoded: []byte{
					0xd8, CBORTagInt128Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, CBORTagInt128Value,
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

		t.Parallel()

		rfcValue, ok := new(big.Int).SetString("18446744073709551616", 10)
		require.True(t, ok)

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt128ValueFromBigInt(rfcValue),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt128Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt256ValueFromInt64(0),
				encoded: []byte{
					0xd8, CBORTagInt256Value,
					// positive bignum
					0xc2,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt256ValueFromInt64(42),
				encoded: []byte{
					0xd8, CBORTagInt256Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt256ValueFromInt64(-1),
				encoded: []byte{
					0xd8, CBORTagInt256Value,
					// negative bignum
					0xc3,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt256ValueFromInt64(-42),
				encoded: []byte{
					0xd8, CBORTagInt256Value,
					// negative bignum
					0xc3,
					// byte string, length 1
					0x41,
					0x29,
				},
			},
		)
	})

	t.Run("min", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
				encoded: []byte{
					0xd8, CBORTagInt256Value,
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

	t.Run("<min", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, CBORTagInt256Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
				encoded: []byte{
					0xd8, CBORTagInt256Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, CBORTagInt256Value,
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

		t.Parallel()

		rfcValue, ok := new(big.Int).SetString("18446744073709551616", 10)
		require.True(t, ok)

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredInt256ValueFromBigInt(rfcValue),
				encoded: []byte{
					// tag
					0xd8, CBORTagInt256Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUIntValueFromUint64(0),
				encoded: []byte{
					0xd8, CBORTagUIntValue,
					// positive bignum
					0xc2,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, CBORTagUIntValue,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUIntValueFromUint64(42),
				encoded: []byte{
					0xd8, CBORTagUIntValue,
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

		t.Parallel()

		rfcValue, ok := new(big.Int).SetString("18446744073709551616", 10)
		require.True(t, ok)

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUIntValueFromBigInt(rfcValue),
				encoded: []byte{
					// tag
					0xd8, CBORTagUIntValue,
					// positive bignum
					0xc2,
					// byte string, length 9
					0x49,
					0x01, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				},
			},
		)
	})

	t.Run("larger than max inline size", func(t *testing.T) {

		t.Parallel()

		inter, err := NewInterpreter(nil, nil, &Config{})
		require.NoError(t, err)

		expected := NewUnmeteredUIntValueFromUint64(1_000_000_000)

		maxInlineElementSize := atree.MaxInlineArrayElementSize
		for len(expected.BigInt.Bytes()) < int(maxInlineElementSize+1) {
			expected = expected.Mul(inter, expected, EmptyLocationRange).(UIntValue)
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:                expected,
				maxInlineElementSize: maxInlineElementSize,
				encoded: []byte{
					// tag
					0xd8, atree.CBORTagStorageID,

					// storage ID
					0x50, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x42, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
				},
			},
		)
	})
}

func TestEncodeDecodeUInt8Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUInt8Value(0),
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt8Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt8Value,
					// negative integer 42
					0x38,
					0x29,
				},
				invalid: true,
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUInt8Value(42),
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt8Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUInt8Value(math.MaxUint8),
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt8Value,
					// positive integer 0xff
					0x18,
					0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt8Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUInt16Value(0),
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt16Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt16Value,
					// negative integer 42
					0x38,
					0x29,
				},
				invalid: true,
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUInt16Value(42),
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt16Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUInt16Value(math.MaxUint16),
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt16Value,
					// positive integer 0xffff
					0x19,
					0xff, 0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt16Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUInt32Value(0),
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt32Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt32Value,
					// negative integer 42
					0x38,
					0x29,
				},
				invalid: true,
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUInt32Value(42),
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt32Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUInt32Value(math.MaxUint32),
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt32Value,
					// positive integer 0xffffffff
					0x1a,
					0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt32Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUInt64Value(0),
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt64Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt64Value,
					// negative integer 42
					0x38,
					0x29,
				},
				invalid: true,
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUInt64Value(42),
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt64Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUInt64Value(math.MaxUint64),
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt64Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUInt128ValueFromUint64(0),
				encoded: []byte{
					0xd8, CBORTagUInt128Value,
					// positive bignum
					0xc2,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUInt128ValueFromUint64(42),
				encoded: []byte{
					0xd8, CBORTagUInt128Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
				encoded: []byte{
					0xd8, CBORTagUInt128Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, CBORTagUInt128Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, CBORTagUInt128Value,
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

		t.Parallel()

		rfcValue, ok := new(big.Int).SetString("18446744073709551616", 10)
		require.True(t, ok)

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUInt128ValueFromBigInt(rfcValue),
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt128Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUInt256ValueFromUint64(0),
				encoded: []byte{
					0xd8, CBORTagUInt256Value,
					// positive bignum
					0xc2,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUInt256ValueFromUint64(42),
				encoded: []byte{
					0xd8, CBORTagUInt256Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, CBORTagUInt256Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, CBORTagUInt256Value,
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

		t.Parallel()

		rfcValue, ok := new(big.Int).SetString("18446744073709551616", 10)
		require.True(t, ok)

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUInt256ValueFromBigInt(rfcValue),
				encoded: []byte{
					// tag
					0xd8, CBORTagUInt256Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord8Value(0),
				encoded: []byte{
					// tag
					0xd8, CBORTagWord8Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagWord8Value,
					// negative integer 42
					0x38,
					0x29,
				},
				invalid: true,
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord8Value(42),
				encoded: []byte{
					// tag
					0xd8, CBORTagWord8Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord8Value(math.MaxUint8),
				encoded: []byte{
					// tag
					0xd8, CBORTagWord8Value,
					// positive integer 0xff
					0x18,
					0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagWord8Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord16Value(0),
				encoded: []byte{
					// tag
					0xd8, CBORTagWord16Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord16Value(42),
				encoded: []byte{
					// tag
					0xd8, CBORTagWord16Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord16Value(math.MaxUint16),
				encoded: []byte{
					// tag
					0xd8, CBORTagWord16Value,
					// positive integer 0xffff
					0x19,
					0xff, 0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagWord16Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord32Value(0),
				encoded: []byte{
					// tag
					0xd8, CBORTagWord32Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord32Value(42),
				encoded: []byte{
					// tag
					0xd8, CBORTagWord32Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord32Value(math.MaxUint32),
				encoded: []byte{
					// tag
					0xd8, CBORTagWord32Value,
					// positive integer 0xffffffff
					0x1a,
					0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagWord32Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord64Value(0),
				encoded: []byte{
					// tag
					0xd8, CBORTagWord64Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord64Value(42),
				encoded: []byte{
					// tag
					0xd8, CBORTagWord64Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord64Value(math.MaxUint64),
				encoded: []byte{
					// tag
					0xd8, CBORTagWord64Value,
					// positive integer 0xffffffffffffffff
					0x1b,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})
}

func TestEncodeDecodeWord128Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord128ValueFromUint64(0),
				encoded: []byte{
					0xd8, CBORTagWord128Value,
					// positive bignum
					0xc2,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord128ValueFromUint64(42),
				encoded: []byte{
					0xd8, CBORTagWord128Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord128ValueFromBigInt(sema.Word128TypeMaxIntBig),
				encoded: []byte{
					0xd8, CBORTagWord128Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, CBORTagWord128Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, CBORTagWord128Value,
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

		t.Parallel()

		rfcValue, ok := new(big.Int).SetString("18446744073709551616", 10)
		require.True(t, ok)

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord128ValueFromBigInt(rfcValue),
				encoded: []byte{
					// tag
					0xd8, CBORTagWord128Value,
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

func TestEncodeDecodeWord256Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord256ValueFromUint64(0),
				encoded: []byte{
					0xd8, CBORTagWord256Value,
					// positive bignum
					0xc2,
					// byte string, length 0
					0x40,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord256ValueFromUint64(42),
				encoded: []byte{
					0xd8, CBORTagWord256Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord256ValueFromBigInt(sema.Word256TypeMaxIntBig),
				encoded: []byte{
					0xd8, CBORTagWord256Value,
					// positive bignum
					0xc2,
					// byte string, length 32
					0x58, 0x20,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, CBORTagWord256Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					0xd8, CBORTagWord256Value,
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

		t.Parallel()

		rfcValue, ok := new(big.Int).SetString("18446744073709551616", 10)
		require.True(t, ok)

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredWord256ValueFromBigInt(rfcValue),
				encoded: []byte{
					// tag
					0xd8, CBORTagWord256Value,
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

func TestEncodeDecodeSomeValue(t *testing.T) {

	t.Parallel()

	t.Run("nil", func(t *testing.T) {

		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredSomeValueNonCopying(Nil),
				encoded: []byte{
					// tag
					0xd8, CBORTagSomeValue,
					// null
					0xf6,
				},
			},
		)
	})

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		expectedString := NewUnmeteredStringValue("test")

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredSomeValueNonCopying(expectedString),
				encoded: []byte{
					// tag
					0xd8, CBORTagSomeValue,

					// tag
					0xd8, CBORTagStringValue,

					// UTF-8 string, length 4
					0x64,
					// t, e, s, t
					0x74, 0x65, 0x73, 0x74,
				},
			},
		)
	})

	t.Run("bool", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredSomeValueNonCopying(TrueValue),
				encoded: []byte{
					// tag
					0xd8, CBORTagSomeValue,
					// true
					0xf5,
				},
			},
		)
	})

	t.Run("larger than max inline size", func(t *testing.T) {

		t.Parallel()

		// Generate a strings that has an encoding size just below the max inline element size.
		// It will not get inlined, but the outer value will

		var str *StringValue
		maxInlineElementSize := atree.MaxInlineArrayElementSize
		for i := uint64(0); i < maxInlineElementSize; i++ {
			str = NewUnmeteredStringValue(strings.Repeat("x", int(maxInlineElementSize-i)))
			size, err := StorableSize(str)
			require.NoError(t, err)
			if uint64(size) == maxInlineElementSize-1 {
				break
			}
		}

		expected := NewUnmeteredSomeValueNonCopying(str)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:                expected,
				maxInlineElementSize: maxInlineElementSize,
				encoded: []byte{
					// tag
					0xd8, atree.CBORTagStorageID,

					// storage ID
					0x50, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x42, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
				},
			},
		)
	})

	t.Run("inner path stored separately", func(t *testing.T) {

		t.Parallel()

		// Generate a string that has an encoding size just above the max inline element size

		var str *StringValue
		maxInlineElementSize := atree.MaxInlineArrayElementSize
		for i := uint64(0); i < maxInlineElementSize; i++ {
			str = NewUnmeteredStringValue(strings.Repeat("x", int(maxInlineElementSize-i)))
			size, err := StorableSize(str)
			require.NoError(t, err)
			if uint64(size) == maxInlineElementSize+1 {
				break
			}
		}

		expected := NewUnmeteredSomeValueNonCopying(str)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:                expected,
				maxInlineElementSize: maxInlineElementSize,
				encoded: []byte{
					// tag
					0xd8, CBORTagSomeValue,
					// value
					0xd8, atree.CBORTagStorageID,
					// storage ID
					0x50, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x42, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
				},
			},
		)
	})
}

func TestEncodeDecodeFix64Value(t *testing.T) {

	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredFix64Value(0),
				encoded: []byte{
					// tag
					0xd8, CBORTagFix64Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredFix64Value(-42),
				encoded: []byte{
					// tag
					0xd8, CBORTagFix64Value,
					// negative integer 42
					0x38,
					0x29,
				},
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredFix64Value(42),
				encoded: []byte{
					// tag
					0xd8, CBORTagFix64Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("min", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredFix64Value(math.MinInt64),
				encoded: []byte{
					// tag
					0xd8, CBORTagFix64Value,
					// negative integer: 0x7fffffffffffffff
					0x3b,
					0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run("<min", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagFix64Value,
					// negative integer 0xffffffffffffffff
					0x3b,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
				invalid: true,
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredFix64Value(math.MaxInt64),
				encoded: []byte{
					// tag
					0xd8, CBORTagFix64Value,
					// positive integer: 0x7fffffffffffffff
					0x1b,
					0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				},
			},
		)
	})

	t.Run(">max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagFix64Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUFix64Value(0),
				encoded: []byte{
					// tag
					0xd8, CBORTagUFix64Value,
					// integer 0
					0x0,
				},
			},
		)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagUFix64Value,
					// negative integer 42
					0x38,
					0x29,
				},
				invalid: true,
			},
		)
	})

	t.Run("positive", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUFix64Value(42),
				encoded: []byte{
					// tag
					0xd8, CBORTagUFix64Value,
					// positive integer 42
					0x18,
					0x2a,
				},
			},
		)
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: NewUnmeteredUFix64Value(math.MaxUint64),
				encoded: []byte{
					// tag
					0xd8, CBORTagUFix64Value,
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
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: AddressValue{},
				encoded: []byte{
					// tag
					0xd8, CBORTagAddressValue,
					// byte sequence, length 0
					0x40,
				},
			},
		)
	})

	t.Run("non-empty", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: AddressValue(common.MustBytesToAddress([]byte{0x42})),
				encoded: []byte{
					// tag
					0xd8, CBORTagAddressValue,
					// byte sequence, length 1
					0x41,
					// address
					0x42,
				},
			},
		)
	})

	t.Run("with leading zeros", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: AddressValue(common.MustBytesToAddress([]byte{0x0, 0x42})),
				encoded: []byte{
					// tag
					0xd8, CBORTagAddressValue,
					// byte sequence, length 1
					0x41,
					// address
					0x42,
				},
			},
		)
	})

	t.Run("with zeros in-between and at and", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				value: AddressValue(common.MustBytesToAddress([]byte{0x0, 0x42, 0x0, 0x43, 0x0})),
				encoded: []byte{
					// tag
					0xd8, CBORTagAddressValue,
					// byte sequence, length 4
					0x44,
					// address
					0x42, 0x0, 0x43, 0x0,
				},
			},
		)
	})

	t.Run("too long", func(t *testing.T) {
		t.Parallel()

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded: []byte{
					// tag
					0xd8, CBORTagAddressValue,
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

		t.Parallel()

		encoded := []byte{
			// tag
			0xd8, CBORTagPathValue,
			// array, 2 items follow
			0x82,
			// positive integer 2
			0x2,
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
	})

	t.Run("public", func(t *testing.T) {

		t.Parallel()

		encoded := []byte{
			// tag
			0xd8, CBORTagPathValue,
			// array, 2 items follow
			0x82,
			// positive integer 3
			0x3,
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
	})

	t.Run("larger than max inline size", func(t *testing.T) {

		t.Parallel()

		maxInlineElementSize := atree.MaxInlineArrayElementSize
		identifier := strings.Repeat("x", int(maxInlineElementSize+1))

		expected := PathValue{
			Domain:     common.PathDomainStorage,
			Identifier: identifier,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:                expected,
				maxInlineElementSize: maxInlineElementSize,
				encoded: []byte{
					// tag
					0xd8, atree.CBORTagStorageID,

					// storage ID
					0x50, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x42, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
				},
			},
		)
	})
}

func TestEncodeDecodeCapabilityValue(t *testing.T) {

	t.Parallel()

	t.Run("untyped capability", func(t *testing.T) {

		t.Parallel()

		encoded := []byte{
			// tag
			0xd8, CBORTagCapabilityValue,
			// array, 3 items follow
			0x83,
			// tag for address
			0xd8, CBORTagAddressValue,
			// byte sequence, length 1
			0x41,
			// address
			0x02,
			// positive integer 4
			0x4,
			// nil
			0xf6,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				encoded:    encoded,
				decodeOnly: true,
				invalid:    true,
			},
		)
	})

	t.Run("typed capability", func(t *testing.T) {

		t.Parallel()

		value := NewUnmeteredCapabilityValue(
			4,
			NewUnmeteredAddressValueFromBytes([]byte{0x2}),
			PrimitiveStaticTypeBool,
		)

		encoded := []byte{
			// tag
			0xd8, CBORTagCapabilityValue,
			// array, 3 items follow
			0x83,
			// tag for address
			0xd8, CBORTagAddressValue,
			// byte sequence, length 1
			0x41,
			// address
			0x02,
			// positive integer 4
			0x4,
			// tag for borrow type
			0xd8, CBORTagPrimitiveStaticType,
			// bool
			0x6,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)
	})

	t.Run("larger than max inline size", func(t *testing.T) {

		t.Parallel()

		// Generate an arbitrary, large static type
		maxInlineElementSize := atree.MaxInlineArrayElementSize
		var borrowType StaticType = PrimitiveStaticTypeNever

		for i := uint64(0); i < maxInlineElementSize; i++ {
			borrowType = &OptionalStaticType{
				Type: borrowType,
			}
		}

		expected := NewUnmeteredCapabilityValue(
			4,
			NewUnmeteredAddressValueFromBytes([]byte{0x3}),
			borrowType,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:                expected,
				maxInlineElementSize: maxInlineElementSize,
				encoded: []byte{
					// tag
					0xd8, atree.CBORTagStorageID,

					// storage ID
					0x50, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x42, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
				},
			},
		)
	})
}

func TestEncodeDecodeTypeValue(t *testing.T) {

	t.Parallel()

	t.Run("primitive, Bool", func(t *testing.T) {

		t.Parallel()

		value := TypeValue{
			Type: ConvertSemaToPrimitiveStaticType(nil, sema.BoolType),
		}

		encoded := []byte{
			// tag
			0xd8, CBORTagTypeValue,
			// array, 1 items follow
			0x81,
			// tag
			0xd8, CBORTagPrimitiveStaticType,
			// positive integer 0
			0x6,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)
	})

	t.Run("primitive, Int", func(t *testing.T) {

		t.Parallel()

		value := TypeValue{
			Type: ConvertSemaToPrimitiveStaticType(nil, sema.IntType),
		}

		encoded := []byte{
			// tag
			0xd8, CBORTagTypeValue,
			// array, 1 items follow
			0x81,
			// tag
			0xd8, CBORTagPrimitiveStaticType,
			// positive integer 36
			0x18, 0x24,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)
	})

	t.Run("InclusiveRange, Int", func(t *testing.T) {

		t.Parallel()

		value := TypeValue{
			Type: InclusiveRangeStaticType{
				ElementType: PrimitiveStaticTypeInt,
			},
		}

		encoded := []byte{
			// tag
			0xd8, CBORTagTypeValue,
			// array, 1 items follow
			0x81,
			// tag
			0xd8, CBORTagInclusiveRangeStaticType,
			// tag
			0xd8, CBORTagPrimitiveStaticType,
			// positive integer 36
			0x18, 0x24,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)
	})

	t.Run("InclusiveRange, UInt256", func(t *testing.T) {

		t.Parallel()

		value := TypeValue{
			Type: InclusiveRangeStaticType{
				ElementType: PrimitiveStaticTypeUInt256,
			},
		}

		encoded := []byte{
			// tag
			0xd8, CBORTagTypeValue,
			// array, 1 items follow
			0x81,
			// tag
			0xd8, CBORTagInclusiveRangeStaticType,
			// tag
			0xd8, CBORTagPrimitiveStaticType,
			// positive integer 50
			0x18, 0x32,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)
	})

	t.Run("without static type", func(t *testing.T) {

		t.Parallel()

		value := TypeValue{
			Type: nil,
		}

		encoded := []byte{
			// tag
			0xd8, CBORTagTypeValue,
			// array, 1 items follow
			0x81,
			// nil
			0xf6,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
				// type values without a static type are not semantically equal,
				// so check deep equality
				deepEquality: true,
			},
		)
	})

	t.Run("larger than max inline size", func(t *testing.T) {

		t.Parallel()

		maxInlineElementSize := atree.MaxInlineArrayElementSize
		identifier := strings.Repeat("x", int(maxInlineElementSize+1))

		expected := TypeValue{
			Type: NewCompositeStaticTypeComputeTypeID(
				nil,
				common.AddressLocation{},
				identifier,
			),
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:                expected,
				maxInlineElementSize: maxInlineElementSize,
				encoded: []byte{
					// tag
					0xd8, atree.CBORTagStorageID,

					// storage ID
					0x50, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x42, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
				},
			},
		)
	})
}

func staticTypeFromBytes(data []byte) (StaticType, error) {
	dec := CBORDecMode.NewByteStreamDecoder(data)
	return NewTypeDecoder(dec, nil).DecodeStaticType()
}

func TestEncodeDecodeStaticType(t *testing.T) {

	t.Parallel()

	t.Run("composite, struct, no location", func(t *testing.T) {

		t.Parallel()

		ty := NewCompositeStaticTypeComputeTypeID(nil, nil, "PublicKey")

		encoded := cbor.RawMessage{
			// tag
			0xd8, CBORTagCompositeStaticType,
			// array, 2 items follow
			0x82,
			// location: nil
			0xf6,
			// UTF-8 string, length 9
			0x69,
			// PublicKey
			0x50, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79,
		}

		actualEncoded, err := StaticTypeToBytes(ty)
		require.NoError(t, err)

		AssertEqualWithDiff(t, encoded, actualEncoded)

		actualType, err := staticTypeFromBytes(encoded)
		require.NoError(t, err)

		require.Equal(t, ty, actualType)
	})
}

func TestCBORTagValue(t *testing.T) {
	t.Parallel()

	t.Run("No new types added in between", func(t *testing.T) {
		require.Equal(t, byte(226), byte(CBORTag_Count))
	})
}

func TestEncodeDecodeStorageCapabilityControllerValue(t *testing.T) {

	t.Parallel()

	assemble := func(bytes ...byte) []byte {
		result := []byte{
			// tag
			0xd8, CBORTagStorageCapabilityControllerValue,
			// array, 3 items follow
			0x83,
		}
		result = append(result, bytes...)
		result = append(result,
			// positive integer 42
			0x18, 0x2A,
			0xd8, CBORTagPathValue,
			// array, 2 items follow
			0x82,
			// positive integer 3
			0x3,
			// UTF-8 string, length 3
			0x63,
			// b, a, r
			0x62, 0x61, 0x72,
		)
		return result
	}

	const capabilityID = 42

	t.Run("non-reference, primitive, Bool", func(t *testing.T) {

		t.Parallel()

		encoded := assemble(
			// tag
			0xd8, CBORTagPrimitiveStaticType,
			0x6,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				decodeOnly: true,
				invalid:    true,
				encoded:    encoded,
			},
		)
	})

	t.Run("unauthorized reference, primitive, Bool", func(t *testing.T) {

		t.Parallel()

		value := &StorageCapabilityControllerValue{
			TargetPath: publicPathValue,
			BorrowType: &ReferenceStaticType{
				ReferencedType: PrimitiveStaticTypeBool,
				Authorization:  UnauthorizedAccess,
			},
			CapabilityID: capabilityID,
		}

		encoded := assemble(
			// tag
			0xd8, CBORTagReferenceStaticType,
			// array, 2 items follow
			0x82,
			// authorization:
			// tag
			0xd8, CBORTagUnauthorizedStaticAuthorization,
			// null
			0xf6,
			// tag
			0xd8, CBORTagPrimitiveStaticType,
			0x6,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)
	})

	t.Run("unauthorized reference, optional, primitive, Bool", func(t *testing.T) {

		t.Parallel()

		value := &StorageCapabilityControllerValue{
			TargetPath: publicPathValue,
			BorrowType: &ReferenceStaticType{
				ReferencedType: &OptionalStaticType{
					Type: PrimitiveStaticTypeBool,
				},
				Authorization: UnauthorizedAccess,
			},
			CapabilityID: capabilityID,
		}

		encoded := assemble(
			// tag
			0xd8, CBORTagReferenceStaticType,
			// array, 2 items follow
			0x82,
			// authorization:
			// tag
			0xd8, CBORTagUnauthorizedStaticAuthorization,
			// null
			0xf6,
			// tag
			0xd8, CBORTagOptionalStaticType,
			// tag
			0xd8, CBORTagPrimitiveStaticType,
			0x6,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)
	})

	t.Run("unauthorized reference, composite, struct, qualified identifier", func(t *testing.T) {

		t.Parallel()

		value := &StorageCapabilityControllerValue{
			TargetPath: publicPathValue,
			BorrowType: &ReferenceStaticType{
				ReferencedType: NewCompositeStaticTypeComputeTypeID(
					nil,
					utils.TestLocation,
					"SimpleStruct",
				),
				Authorization: UnauthorizedAccess,
			},
			CapabilityID: capabilityID,
		}

		encoded := assemble(
			// tag
			0xd8, CBORTagReferenceStaticType,
			// array, 2 items follow
			0x82,
			// authorization:
			// tag
			0xd8, CBORTagUnauthorizedStaticAuthorization,
			// null
			0xf6,
			// tag
			0xd8, CBORTagCompositeStaticType,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, CBORTagStringLocation,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
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
	})

	t.Run("unauthorized reference, interface, struct, qualified identifier", func(t *testing.T) {

		t.Parallel()

		value := &StorageCapabilityControllerValue{
			TargetPath: publicPathValue,
			BorrowType: &ReferenceStaticType{
				ReferencedType: NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "SimpleInterface"),
				Authorization:  UnauthorizedAccess,
			},
			CapabilityID: capabilityID,
		}

		encoded := assemble(
			// tag
			0xd8, CBORTagReferenceStaticType,
			// array, 2 items follow
			0x82,
			// authorization:
			// tag
			0xd8, CBORTagUnauthorizedStaticAuthorization,
			// null
			0xf6,
			// tag
			0xd8, CBORTagInterfaceStaticType,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, CBORTagStringLocation,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
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
	})

	t.Run("unauthorized reference, variable-sized, bool", func(t *testing.T) {

		t.Parallel()

		value := &StorageCapabilityControllerValue{
			TargetPath: publicPathValue,
			BorrowType: &ReferenceStaticType{
				ReferencedType: &VariableSizedStaticType{
					Type: PrimitiveStaticTypeBool,
				},
				Authorization: UnauthorizedAccess,
			},
			CapabilityID: capabilityID,
		}

		encoded := assemble(
			// tag
			0xd8, CBORTagReferenceStaticType,
			// array, 2 items follow
			0x82,
			// authorization:
			// tag
			0xd8, CBORTagUnauthorizedStaticAuthorization,
			// null
			0xf6,
			// tag
			0xd8, CBORTagVariableSizedStaticType,
			// tag
			0xd8, CBORTagPrimitiveStaticType,
			0x6,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)
	})

	t.Run("unauthorized reference, constant-sized, bool", func(t *testing.T) {

		t.Parallel()

		value := &StorageCapabilityControllerValue{
			TargetPath: publicPathValue,
			BorrowType: &ReferenceStaticType{
				ReferencedType: &ConstantSizedStaticType{
					Type: PrimitiveStaticTypeBool,
					Size: 42,
				},
				Authorization: UnauthorizedAccess,
			},
			CapabilityID: capabilityID,
		}

		encoded := assemble(
			// tag
			0xd8, CBORTagReferenceStaticType,
			// array, 2 items follow
			0x82,
			// authorization:
			// tag
			0xd8, CBORTagUnauthorizedStaticAuthorization,
			// null
			0xf6,
			// tag
			0xd8, CBORTagConstantSizedStaticType,
			// array, 2 items follow
			0x82,
			// positive integer 42
			0x18, 0x2A,
			// tag
			0xd8, CBORTagPrimitiveStaticType,
			0x6,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)
	})

	t.Run("unauthorized reference, dictionary, bool, string", func(t *testing.T) {

		t.Parallel()

		value := &StorageCapabilityControllerValue{
			TargetPath: publicPathValue,
			BorrowType: &ReferenceStaticType{
				ReferencedType: &DictionaryStaticType{
					KeyType:   PrimitiveStaticTypeBool,
					ValueType: PrimitiveStaticTypeString,
				},
				Authorization: UnauthorizedAccess,
			},
			CapabilityID: capabilityID,
		}

		encoded := assemble(
			// tag
			0xd8, CBORTagReferenceStaticType,
			// array, 2 items follow
			0x82,
			// authorization:
			// tag
			0xd8, CBORTagUnauthorizedStaticAuthorization,
			// null
			0xf6,
			// tag
			0xd8, CBORTagDictionaryStaticType,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, CBORTagPrimitiveStaticType,
			0x6,
			// tag
			0xd8, CBORTagPrimitiveStaticType,
			0x8,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)
	})

	t.Run("unauthorized reference, intersection", func(t *testing.T) {

		t.Parallel()

		value := &StorageCapabilityControllerValue{
			TargetPath: publicPathValue,
			BorrowType: &ReferenceStaticType{
				ReferencedType: &IntersectionStaticType{
					Types: []*InterfaceStaticType{
						NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "I1"),
						NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "I2"),
					},
				},
				Authorization: UnauthorizedAccess,
			},
			CapabilityID: capabilityID,
		}

		encoded := assemble(
			// tag
			0xd8, CBORTagReferenceStaticType,
			// array, 2 items follow
			0x82,
			// authorization:
			// tag
			0xd8, CBORTagUnauthorizedStaticAuthorization,
			// null
			0xf6,
			// tag
			0xd8, CBORTagIntersectionStaticType,
			// array, length 2
			0x82,
			// nil
			0xf6,
			// array, length 2
			0x82,
			// tag
			0xd8, CBORTagInterfaceStaticType,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, CBORTagStringLocation,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
			// UTF-8 string, length 2
			0x62,
			// I1
			0x49, 0x31,
			// tag
			0xd8, CBORTagInterfaceStaticType,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, CBORTagStringLocation,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
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
	})

	t.Run("larger than max inline size", func(t *testing.T) {

		t.Parallel()

		maxInlineElementSize := atree.MaxInlineArrayElementSize
		identifier := strings.Repeat("x", int(maxInlineElementSize+1))

		path := PathValue{
			Domain:     common.PathDomainStorage,
			Identifier: identifier,
		}

		expected := &StorageCapabilityControllerValue{
			TargetPath: path,
			BorrowType: &ReferenceStaticType{
				ReferencedType: PrimitiveStaticTypeNever,
				Authorization:  UnauthorizedAccess,
			},
			CapabilityID: capabilityID,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:                expected,
				maxInlineElementSize: maxInlineElementSize,
				encoded: []byte{
					// tag
					0xd8, atree.CBORTagStorageID,

					// storage ID
					0x50, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x42, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
				},
			},
		)
	})
}

func TestEncodeDecodeAccountCapabilityControllerValue(t *testing.T) {

	t.Parallel()

	assemble := func(bytes ...byte) []byte {
		result := []byte{
			// tag
			0xd8, CBORTagAccountCapabilityControllerValue,
			// array, 2 items follow
			0x82,
		}
		result = append(result, bytes...)
		result = append(result,
			// positive integer 42
			0x18, 0x2A,
		)
		return result
	}

	const capabilityID = 42

	t.Run("non-reference, primitive, Bool", func(t *testing.T) {

		t.Parallel()

		encoded := assemble(
			// tag
			0xd8, CBORTagPrimitiveStaticType,
			0x6,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				decodeOnly: true,
				invalid:    true,
				encoded:    encoded,
			},
		)
	})

	t.Run("unauthorized reference, AuthAccount", func(t *testing.T) {

		t.Parallel()

		value := &AccountCapabilityControllerValue{
			BorrowType: &ReferenceStaticType{
				Authorization:  UnauthorizedAccess,
				ReferencedType: PrimitiveStaticTypeAuthAccount,
			},
			CapabilityID: capabilityID,
		}

		encoded := assemble(
			// tag
			0xd8, CBORTagReferenceStaticType,
			// array, 2 items follow
			0x82,
			// authorization:
			// tag
			0xd8, CBORTagUnauthorizedStaticAuthorization,
			// null
			0xf6,
			0xd8, CBORTagPrimitiveStaticType,
			// unsigned 90
			0x18, 0x5a,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)
	})

	t.Run("unauthorized reference, Account", func(t *testing.T) {

		t.Parallel()

		value := &AccountCapabilityControllerValue{
			BorrowType: &ReferenceStaticType{
				Authorization:  UnauthorizedAccess,
				ReferencedType: PrimitiveStaticTypeAccount,
			},
			CapabilityID: capabilityID,
		}

		encoded := assemble(
			// tag
			0xd8, CBORTagReferenceStaticType,
			// array, 2 items follow
			0x82,
			// authorization:
			// tag
			0xd8, CBORTagUnauthorizedStaticAuthorization,
			// null
			0xf6,
			0xd8, CBORTagPrimitiveStaticType,
			// unsigned 105
			0x18, 0x69,
		)

		testEncodeDecode(t,
			encodeDecodeTest{
				value:   value,
				encoded: encoded,
			},
		)
	})

	t.Run("unauthorized reference, intersection I1", func(t *testing.T) {

		t.Parallel()

		value := &AccountCapabilityControllerValue{
			BorrowType: &ReferenceStaticType{
				ReferencedType: &IntersectionStaticType{
					Types: []*InterfaceStaticType{
						NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "SimpleInterface"),
					},
				},
				Authorization: UnauthorizedAccess,
			},
			CapabilityID: capabilityID,
		}

		encoded := assemble(
			// tag
			0xd8, CBORTagReferenceStaticType,
			// array, 2 items follow
			0x82,
			// authorization:
			// tag
			0xd8, CBORTagUnauthorizedStaticAuthorization,
			// null
			0xf6,
			// tag
			0xd8, CBORTagIntersectionStaticType,
			// array, length 2
			0x82,
			// nil
			0xf6,
			// array, 1 item follows
			0x81,
			// tag
			0xd8, CBORTagInterfaceStaticType,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, CBORTagStringLocation,
			// UTF-8 string, length 4
			0x64,
			// t, e, s, t
			0x74, 0x65, 0x73, 0x74,
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
	})

	t.Run("larger than max inline size", func(t *testing.T) {

		t.Parallel()

		// Generate an arbitrary, large static type
		maxInlineElementSize := atree.MaxInlineArrayElementSize
		var borrowType StaticType = PrimitiveStaticTypeNever

		for i := uint64(0); i < maxInlineElementSize; i++ {
			borrowType = &OptionalStaticType{
				Type: borrowType,
			}
		}

		expected := &AccountCapabilityControllerValue{
			BorrowType: &ReferenceStaticType{
				ReferencedType: borrowType,
				Authorization:  UnauthorizedAccess,
			},
			CapabilityID: capabilityID,
		}

		testEncodeDecode(t,
			encodeDecodeTest{
				value:                expected,
				maxInlineElementSize: maxInlineElementSize,
				encoded: []byte{
					// tag
					0xd8, atree.CBORTagStorageID,

					// storage ID
					0x50, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x42, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
				},
			},
		)
	})
}
