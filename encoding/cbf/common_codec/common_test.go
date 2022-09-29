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

package common_codec_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/encoding/cbf/common_codec"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/stdlib"
)

func TestCodecMiscValues(t *testing.T) {
	t.Parallel()

	t.Run("length (1 byte)", func(t *testing.T) {
		t.Parallel()

		length := 10

		var w bytes.Buffer
		err := common_codec.EncodeLength(&w, 10)
		require.NoError(t, err, "encoding error")

		assert.Equal(t, w.Bytes(), []byte{0, 0, 0, byte(length)}, "encoded bytes differ")

		output, err := common_codec.DecodeLength(&w, 0xFFFFFFFF)
		require.NoError(t, err, "decoding error")

		assert.Equal(t, length, output)
	})

	t.Run("length (2 bytes)", func(t *testing.T) {
		t.Parallel()

		length0 := 5
		length1 := 10
		length := length0 + (length1 << 8)

		var w bytes.Buffer

		err := common_codec.EncodeLength(&w, length)
		require.NoError(t, err, "encoding error")

		assert.Equal(t, w.Bytes(), []byte{0, 0, byte(length1), byte(length0)}, "encoded bytes differ")

		output, err := common_codec.DecodeLength(&w, 0xFFFFFFFF)
		require.NoError(t, err, "decoding error")

		assert.Equal(t, length, output)
	})

	t.Run("length error: negative", func(t *testing.T) {
		t.Parallel()

		var w bytes.Buffer

		err := common_codec.EncodeLength(&w, -1)
		assert.ErrorContains(t, err, "cannot encode length below zero: -1")
	})

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		var w bytes.Buffer

		s := "some string \x00 foo \t \n\r\n $ 5"

		err := common_codec.EncodeString(&w, s)
		require.NoError(t, err, "encoding error")

		assert.Equal(t, w.Bytes(), common_codec.Flatten(
			[]byte{0, 0, 0, byte(len(s))},
			[]byte(s),
		), "encoded bytes differ")

		output, err := common_codec.DecodeString(&w, 0xFFFFFFFF)
		require.NoError(t, err, "decoding error")

		assert.Equal(t, s, output)
	})

	t.Run("string len=0", func(t *testing.T) {
		t.Parallel()

		var w bytes.Buffer

		s := ""

		err := common_codec.EncodeString(&w, s)
		require.NoError(t, err, "encoding error")

		assert.Equal(t, w.Bytes(), common_codec.Flatten(
			[]byte{0, 0, 0, byte(len(s))},
			[]byte(s),
		), "encoded bytes differ")

		output, err := common_codec.DecodeString(&w, 0xFFFFFFFF)
		require.NoError(t, err, "decoding error")

		assert.Equal(t, s, output)
	})

	t.Run("bytes", func(t *testing.T) {
		t.Parallel()

		var w bytes.Buffer

		s := []byte("some string \x00 foo \t \n\r\n $ 5")

		err := common_codec.EncodeBytes(&w, s)
		require.NoError(t, err, "encoding error")

		assert.Equal(t, w.Bytes(), common_codec.Flatten(
			[]byte{0, 0, 0, byte(len(s))},
			s,
		), "encoded bytes differ")

		output, err := common_codec.DecodeBytes(&w, 0xFFFFFFFF)
		require.NoError(t, err, "decoding error")

		assert.Equal(t, s, output)
	})

	t.Run("bool true", func(t *testing.T) {
		t.Parallel()

		var w bytes.Buffer

		var b = true

		err := common_codec.EncodeBool(&w, b)
		require.NoError(t, err, "encoding error")

		assert.Equal(t, w.Bytes(), []byte{byte(common_codec.EncodedBoolTrue)}, "encoded bytes differ")

		output, err := common_codec.DecodeBool(&w)
		require.NoError(t, err, "decoding error")

		assert.Equal(t, b, output)
	})

	t.Run("bool false", func(t *testing.T) {
		t.Parallel()

		var w bytes.Buffer

		var b = false

		err := common_codec.EncodeBool(&w, b)
		require.NoError(t, err, "encoding error")

		assert.Equal(t, w.Bytes(), []byte{byte(common_codec.EncodedBoolFalse)}, "encoded bytes differ")

		output, err := common_codec.DecodeBool(&w)
		require.NoError(t, err, "decoding error")

		assert.Equal(t, b, output)
	})

	t.Run("address", func(t *testing.T) {
		t.Parallel()

		var w bytes.Buffer

		addr := common.MustBytesToAddress([]byte{0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00})

		err := common_codec.EncodeAddress(&w, addr)
		require.NoError(t, err, "encoding error")

		assert.Equal(t, w.Bytes(), addr.Bytes(), "encoded bytes differ")

		output, err := common_codec.DecodeAddress(&w)
		require.NoError(t, err, "decoding error")

		assert.Equal(t, addr, output)
	})

	t.Run("uint64", func(t *testing.T) {
		t.Parallel()

		var w bytes.Buffer

		i := uint64(1<<63) + 17

		err := common_codec.EncodeNumber(&w, i)
		require.NoError(t, err, "encoding error")

		assert.Equal(t, w.Bytes(), []byte{128, 0, 0, 0, 0, 0, 0, 17}, "encoded bytes differ")

		output, err := common_codec.DecodeNumber[uint64](&w)
		require.NoError(t, err, "decoding error")

		assert.Equal(t, i, output)
	})

	t.Run("int64 positive", func(t *testing.T) {
		t.Parallel()

		var w bytes.Buffer

		i := int64(1<<62) + 17

		err := common_codec.EncodeNumber(&w, i)
		require.NoError(t, err, "encoding error")

		assert.Equal(t, w.Bytes(), []byte{64, 0, 0, 0, 0, 0, 0, 17}, "encoded bytes differ")

		output, err := common_codec.DecodeNumber[int64](&w)
		require.NoError(t, err, "decoding error")

		assert.Equal(t, i, output)
	})

	t.Run("int64 negative", func(t *testing.T) {
		t.Parallel()

		var w bytes.Buffer

		i := -(int64(1<<62) + 17)

		err := common_codec.EncodeNumber(&w, i)
		require.NoError(t, err, "encoding error")

		assert.Equal(t, w.Bytes(), []byte{0xff - 64, 0xff - 0, 0xff - 0, 0xff - 0, 0xff - 0, 0xff - 0, 0xff - 0, 0xff - 17 + 1}, "encoded bytes differ")

		output, err := common_codec.DecodeNumber[int64](&w)
		require.NoError(t, err, "decoding error")

		assert.Equal(t, i, output)
	})
}

func TestLocationPrefixCodec(t *testing.T) {
	t.Parallel()

	t.Run("length (1 byte)", func(t *testing.T) {
		t.Parallel()

		length := 10

		var w bytes.Buffer
		err := common_codec.EncodeLength(&w, 10)
		require.NoError(t, err, "encoding error")

		assert.Equal(t, w.Bytes(), []byte{0, 0, 0, byte(length)}, "encoded bytes differ")

		output, err := common_codec.DecodeLength(&w, 0xFFFFFFFF)
		require.NoError(t, err, "decoding error")

		assert.Equal(t, length, output)
	})

	for _, prefix := range []string{
		common.AddressLocationPrefix,
		common.IdentifierLocationPrefix,
		common.ScriptLocationPrefix,
		common.StringLocationPrefix,
		common.TransactionLocationPrefix,
		common.REPLLocationPrefix,
		stdlib.FlowLocationPrefix,
		common_codec.NilLocationPrefix,
	} {
		func(prefix string) {
			t.Run(fmt.Sprintf("prefix: %s", prefix), func(t *testing.T) {
				t.Parallel()

				var w bytes.Buffer

				err := common_codec.EncodeLocationPrefix(&w, prefix)
				require.NoError(t, err, "encoding error")

				assert.Equal(t, w.Bytes(), []byte{prefix[0]}, "encoded bytes differ")

				output, err := common_codec.DecodeLocationPrefix(&w)
				require.NoError(t, err, "decoding error")

				assert.Equal(t, prefix[0], output[0], "bad decoding")
			})
		}(prefix)
	}
}

func TestLocationCodec(t *testing.T) {
	type testcase struct {
		location common.Location
		expected []byte
	}

	addrLoc := common.AddressLocation{
		Address: common.Address{12, 13, 14},
		Name:    "foo-bar",
	}

	identLoc := common.IdentifierLocation("id \x01 \x00\n\rsomeid\n")

	scriptLoc := common.ScriptLocation{'i', 'd', ' ', 1, 0, '\n', '\r', 's', 'o', 'm', 'e', 'i', 'd', '\n'}

	stringLoc := common.StringLocation("id \x01 \x00\n\rsomeid\n")

	txnLoc := common.TransactionLocation{'i', 'd', ' ', 1, 0, '\n', '\r', 's', 'o', 'm', 'e', 'i', 'd', '\n'}

	testcases := []testcase{
		{nil, []byte{common_codec.NilLocationPrefix[0]}},
		{
			addrLoc,
			common_codec.Flatten(
				[]byte{common.AddressLocationPrefix[0]},
				addrLoc.Address.Bytes(),
				[]byte{0, 0, 0, byte(len(addrLoc.Name))},
				[]byte(addrLoc.Name),
			),
		},
		{
			identLoc,
			common_codec.Flatten(
				[]byte{common.IdentifierLocationPrefix[0]},
				[]byte{0, 0, 0, byte(len(identLoc))},
				[]byte(identLoc),
			),
		},
		{
			scriptLoc,
			common_codec.Flatten(
				[]byte{common.ScriptLocationPrefix[0]},
				scriptLoc[:],
			),
		},
		{
			stringLoc,
			common_codec.Flatten(
				[]byte{common.StringLocationPrefix[0]},
				[]byte{0, 0, 0, byte(len(stringLoc))},
				[]byte(stringLoc),
			),
		},
		{
			txnLoc,
			common_codec.Flatten(
				[]byte{common.TransactionLocationPrefix[0]},
				txnLoc[:],
			),
		},
		{
			common.REPLLocation{},
			[]byte{common.REPLLocationPrefix[0]},
		},
		{
			stdlib.FlowLocation{},
			[]byte{stdlib.FlowLocationPrefix[0]},
		},
	}

	for _, tc := range testcases {
		func(tc testcase) {
			name := fmt.Sprintf("EncodeLocation(%T)", tc.location)
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				var w bytes.Buffer

				err := common_codec.EncodeLocation(&w, tc.location)
				require.NoError(t, err, "encoding error")

				assert.Equal(t, w.Bytes(), common_codec.Flatten(
					[]byte{common.REPLLocationPrefix[0]},
				), "encoded bytes differ")

				output, err := common_codec.DecodeLocation(&w, 0xFFFFFFFF, nil)
				require.NoError(t, err, "decoding error")
				assert.Equal(t, tc.location, output, "bad decoding")
			})
		}(tc)
	}
}
