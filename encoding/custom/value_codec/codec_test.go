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

package value_codec_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/custom/common_codec"

	"github.com/onflow/cadence/encoding/custom/value_codec"
)

func TestValueCodecVoid(t *testing.T) {
	t.Parallel()

	encoder, decoder, buffer := NewTestCodec()

	value := cadence.NewVoid()

	err := encoder.Encode(value)
	require.NoError(t, err, "encoding error")

	assert.Equal(
		t,
		[]byte{byte(value_codec.EncodedValueVoid)},
		buffer.Bytes(), "encoded bytes differ")

	output, err := decoder.Decode()
	require.NoError(t, err, "decoding error")

	assert.Equal(t, value, output, "decoded value differs")
}

func TestValueCodecBool(t *testing.T) {
	t.Parallel()

	t.Run("false", func(t *testing.T) {
		t.Parallel()

		encoder, decoder, buffer := NewTestCodec()

		value := cadence.NewBool(false)

		err := encoder.Encode(value)
		require.NoError(t, err, "encoding error")

		assert.Equal(
			t,
			[]byte{
				byte(value_codec.EncodedValueBool),
				byte(common_codec.EncodedBoolFalse),
			},
			buffer.Bytes(), "encoded bytes differ")

		output, err := decoder.Decode()
		require.NoError(t, err, "decoding error")

		assert.Equal(t, value, output, "decoded value differs")
	})

	t.Run("true", func(t *testing.T) {
		t.Parallel()

		encoder, decoder, buffer := NewTestCodec()

		value := cadence.NewBool(true)

		err := encoder.Encode(value)
		require.NoError(t, err, "encoding error")

		assert.Equal(
			t,
			[]byte{
				byte(value_codec.EncodedValueBool),
				byte(common_codec.EncodedBoolTrue),
			},
			buffer.Bytes(), "encoded bytes differ")

		output, err := decoder.Decode()
		require.NoError(t, err, "decoding error")

		assert.Equal(t, value, output, "decoded value differs")
	})
}

func TestValueCodecOptional(t *testing.T) {
	t.Parallel()

	t.Run("Optional(Void)", func(t *testing.T) {
		t.Parallel()

		encoder, decoder, buffer := NewTestCodec()

		innerValue := cadence.NewVoid()
		value := cadence.NewOptional(innerValue)

		err := encoder.Encode(value)
		require.NoError(t, err, "encoding error")

		assert.Equal(
			t,
			[]byte{
				byte(value_codec.EncodedValueOptional),
				byte(common_codec.EncodedBoolFalse),
				byte(value_codec.EncodedValueVoid),
			},
			buffer.Bytes(), "encoded bytes differ")

		output, err := decoder.Decode()
		require.NoError(t, err, "decoding error")

		assert.Equal(t, value, output, "decoded value differs")
	})

	t.Run("Optional(bool)", func(t *testing.T) {
		t.Parallel()

		encoder, decoder, buffer := NewTestCodec()

		innerValue := cadence.NewBool(true)
		value := cadence.NewOptional(innerValue)

		err := encoder.Encode(value)
		require.NoError(t, err, "encoding error")

		assert.Equal(
			t,
			[]byte{
				byte(value_codec.EncodedValueOptional),
				byte(common_codec.EncodedBoolFalse),
				byte(value_codec.EncodedValueBool),
				byte(common_codec.EncodedBoolTrue),
			},
			buffer.Bytes(), "encoded bytes differ")

		output, err := decoder.Decode()
		require.NoError(t, err, "decoding error")

		assert.Equal(t, value, output, "decoded value differs")
	})

	t.Run("Optional(nil)", func(t *testing.T) {
		t.Parallel()

		encoder, decoder, buffer := NewTestCodec()

		value := cadence.NewOptional(nil)

		err := encoder.Encode(value)
		require.NoError(t, err, "encoding error")

		assert.Equal(
			t,
			[]byte{
				byte(value_codec.EncodedValueOptional),
				byte(common_codec.EncodedBoolTrue),
			},
			buffer.Bytes(), "encoded bytes differ")

		output, err := decoder.Decode()
		require.NoError(t, err, "decoding error")

		assert.Equal(t, value, output, "decoded value differs")
	})
}

func TestValueCodecArray(t *testing.T) {
	t.Parallel()

	t.Run("Variable Array, len=0", func(t *testing.T) {
		t.Parallel()

		encoder, decoder, buffer := NewTestCodec()

		elements := make([]cadence.Value, 0)

		value := cadence.NewArray(elements).
			WithType(cadence.NewVariableSizedArrayType(cadence.NewAnyType()))

		err := encoder.Encode(value)
		require.NoError(t, err, "encoding error")

		assert.Equal(
			t,
			[]byte{
				byte(value_codec.EncodedValueArray),
				byte(value_codec.EncodedArrayTypeVariable),
				byte(value_codec.EncodedTypeAnyType),
				0, 0, 0, byte(len(elements)),
			},
			buffer.Bytes(), "encoded bytes differ")

		output, err := decoder.Decode()
		require.NoError(t, err, "decoding error")

		assert.Equal(t, value, output, "decoded value differs")
	})

	t.Run("Variable Array, len=2", func(t *testing.T) {
		t.Parallel()

		encoder, decoder, buffer := NewTestCodec()

		elements := []cadence.Value{
			cadence.NewVoid(),
			cadence.NewBool(true),
		}

		value := cadence.NewArray(elements).
			WithType(cadence.NewVariableSizedArrayType(cadence.NewAnyType()))

		err := encoder.Encode(value)
		require.NoError(t, err, "encoding error")

		assert.Equal(
			t,
			[]byte{
				byte(value_codec.EncodedValueArray),
				byte(value_codec.EncodedArrayTypeVariable),
				byte(value_codec.EncodedTypeAnyType),
				0, 0, 0, byte(len(elements)),

				byte(value_codec.EncodedValueVoid),

				byte(value_codec.EncodedValueBool),
				byte(common_codec.EncodedBoolTrue),
			},
			buffer.Bytes(), "encoded bytes differ")

		output, err := decoder.Decode()
		require.NoError(t, err, "decoding error")

		assert.Equal(t, value, output, "decoded value differs")
	})

	t.Run("Constant Array, len=0", func(t *testing.T) {
		t.Parallel()

		encoder, decoder, buffer := NewTestCodec()

		elements := make([]cadence.Value, 0)

		value := cadence.NewArray(elements).
			WithType(cadence.NewConstantSizedArrayType(uint(len(elements)), cadence.NewAnyStructType()))

		err := encoder.Encode(value)
		require.NoError(t, err, "encoding error")

		assert.Equal(
			t,
			[]byte{
				byte(value_codec.EncodedValueArray),
				byte(value_codec.EncodedArrayTypeConstant),
				byte(value_codec.EncodedTypeAnyStructType),
				0, 0, 0, byte(len(elements)),
			},
			buffer.Bytes(), "encoded bytes differ")

		output, err := decoder.Decode()
		require.NoError(t, err, "decoding error")

		assert.Equal(t, value, output, "decoded value differs")
	})

	t.Run("Constant Array, len=2", func(t *testing.T) {
		t.Parallel()

		encoder, decoder, buffer := NewTestCodec()

		elements := []cadence.Value{
			cadence.NewVoid(),
			cadence.NewBool(true),
		}

		value := cadence.NewArray(elements).
			WithType(cadence.NewConstantSizedArrayType(uint(len(elements)), cadence.NewAnyStructType()))

		err := encoder.Encode(value)
		require.NoError(t, err, "encoding error")

		assert.Equal(
			t,
			[]byte{
				byte(value_codec.EncodedValueArray),
				byte(value_codec.EncodedArrayTypeConstant),
				byte(value_codec.EncodedTypeAnyStructType),
				0, 0, 0, byte(len(elements)),

				byte(value_codec.EncodedValueVoid),

				byte(value_codec.EncodedValueBool),
				byte(common_codec.EncodedBoolTrue),
			},
			buffer.Bytes(), "encoded bytes differ")

		output, err := decoder.Decode()
		require.NoError(t, err, "decoding error")

		assert.Equal(t, value, output, "decoded value differs")
	})
}

func NewTestCodec() (encoder *value_codec.Encoder, decoder *value_codec.Decoder, buffer *bytes.Buffer) {
	var w bytes.Buffer
	buffer = &w
	encoder = value_codec.NewEncoder(buffer)
	decoder = value_codec.NewDecoder(nil, buffer)
	return
}
