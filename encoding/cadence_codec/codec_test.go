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

package cadence_codec_test

import (
	"fmt"
	"testing"

	"github.com/onflow/cadence/encoding/cadence_codec"
	"github.com/onflow/cadence/encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/cbf/cbf_codec"
)

func TestCadenceCodecCBF(t *testing.T) {
	t.Parallel()

	codec := cadence_codec.NewCadenceCodec(cbf_codec.CadenceBinaryFormatCodec{})

	expectedEncoding := []byte{byte(cbf_codec.EncodedValueVoid)}

	t.Run("EncodeValue then DecodeValue", func(t *testing.T) {
		t.Parallel()

		value := cadence.Void{}

		b, err := codec.Encode(value)
		require.NoError(t, err, "encoding error")

		assert.Equal(t, expectedEncoding, b, "encoded wrong")

		v, err := codec.Decode(nil, b)
		require.NoError(t, err, "decoding error")

		assert.Equal(t, value, v, "decoded wrong")
	})

	t.Run("MustEncode", func(t *testing.T) {
		t.Parallel()

		value := cadence.Void{}

		b := codec.MustEncode(value)

		assert.Equal(t, expectedEncoding, b, "encoded wrong")

		v := codec.MustDecode(nil, b)

		assert.Equal(t, value, v, "decoded wrong")
	})
}

func TestCadenceCodecJSON(t *testing.T) {
	t.Parallel()

	codec := cadence_codec.NewCadenceCodec(json.JsonCodec{})

	expectedEncoding := []byte(fmt.Sprint(`{"type":"Void"}`, "\n"))

	t.Run("Encode then Decode", func(t *testing.T) {
		t.Parallel()

		value := cadence.Void{}

		b, err := codec.Encode(value)
		require.NoError(t, err, "encoding error")

		assert.Equal(t, expectedEncoding, b, "encoded wrong")

		v, err := codec.Decode(nil, b)
		require.NoError(t, err, "decoding error")

		assert.Equal(t, value, v, "decoded wrong")
	})

	t.Run("MustEncode then MustDecode", func(t *testing.T) {
		t.Parallel()

		value := cadence.Void{}

		b := codec.MustEncode(value)

		assert.Equal(t, expectedEncoding, b, "encoded wrong")

		v := codec.MustDecode(nil, b)

		assert.Equal(t, value, v, "decoded wrong")
	})
}

func TestCadenceCodecErrors(t *testing.T) {
	t.Parallel()

	t.Run("cannot choose codec: empty bytes", func(t *testing.T) {
		t.Parallel()

		codec := cadence_codec.CadenceCodec{}

		_, err := codec.Decode(nil, []byte{})
		assert.ErrorContains(t, err, "cannot decode empty bytes")
	})

	t.Run("cannot choose codec: unknown codec version", func(t *testing.T) {
		t.Parallel()

		codec := cadence_codec.CadenceCodec{}

		_, err := codec.Decode(nil, []byte{0x00}) // no codec denoted by 0x00 byte
		assert.ErrorContains(t, err, "unknown codec version")
	})

	t.Run("cannot choose codec for MustDecode: empty bytes", func(t *testing.T) {
		t.Parallel()

		codec := cadence_codec.CadenceCodec{}

		assert.PanicsWithError(t, "cannot decode empty bytes", func() {
			codec.MustDecode(nil, []byte{})
		})
	})
}
