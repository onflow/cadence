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

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMustBytesToAddress(t *testing.T) {

	t.Parallel()

	t.Run("short", func(t *testing.T) {

		t.Parallel()

		assert.NotPanics(t, func() {
			assert.Equal(t,
				Address{0, 0, 0, 0, 0, 0, 0, 0x1},
				MustBytesToAddress([]byte{0x1}),
			)
		})
	})

	t.Run("full", func(t *testing.T) {

		t.Parallel()

		assert.NotPanics(t, func() {
			assert.Equal(t,
				Address{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8},
				MustBytesToAddress([]byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8}),
			)
		})
	})

	t.Run("too long", func(t *testing.T) {

		t.Parallel()

		assert.Panics(t, func() {
			MustBytesToAddress([]byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9})
		})
	})
}

func TestBytesToAddress(t *testing.T) {

	t.Parallel()

	t.Run("short", func(t *testing.T) {

		t.Parallel()

		address, err := BytesToAddress([]byte{0x1})

		require.NoError(t, err)
		assert.Equal(t,
			Address{0, 0, 0, 0, 0, 0, 0, 0x1},
			address,
		)
	})

	t.Run("full", func(t *testing.T) {

		t.Parallel()

		address, err := BytesToAddress([]byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8})

		require.NoError(t, err)
		assert.Equal(t,
			Address{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8},
			address,
		)
	})

	t.Run("too long", func(t *testing.T) {

		t.Parallel()

		address, err := BytesToAddress([]byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9})

		require.Error(t, err)
		assert.Equal(t,
			Address{},
			address,
		)
	})
}

func TestAddress_Hex(t *testing.T) {

	t.Parallel()

	assert.Equal(t,
		"1234567890abcdef",
		Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}.Hex(),
	)

	assert.Equal(t,
		"0100000000000000",
		Address{0x1}.Hex(),
	)

	assert.Equal(t,
		"0000000000000001",
		Address{0, 0, 0, 0, 0, 0, 0, 0x1}.Hex(),
	)
}

func TestAddress_String(t *testing.T) {

	t.Parallel()

	assert.Equal(t,
		"1234567890abcdef",
		Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}.String(),
	)

	assert.Equal(t,
		"0100000000000000",
		Address{0x1}.String(),
	)

	assert.Equal(t,
		"0000000000000001",
		Address{0, 0, 0, 0, 0, 0, 0, 0x1}.String(),
	)
}

func TestAddress_SetBytes(t *testing.T) {

	t.Parallel()

	var address Address

	address.SetBytes([]byte{0x1})
	assert.Equal(t,
		Address{0, 0, 0, 0, 0, 0, 0, 0x1},
		address,
	)

	address.SetBytes([]byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8})
	assert.Equal(t,
		Address{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8},
		address,
	)

	address.SetBytes([]byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9})
	assert.Equal(t,
		Address{0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9},
		address,
	)
}

func TestAddress_Bytes(t *testing.T) {

	t.Parallel()

	assert.Equal(t,
		[]byte{0x1},
		Address{0, 0, 0, 0, 0, 0, 0, 0x1}.Bytes(),
	)

	assert.Equal(t,
		[]byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8},
		Address{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8}.Bytes(),
	)
}

func TestAddress_ShortHexWithPrefix(t *testing.T) {

	t.Parallel()

	assert.Equal(t,
		"0x1234567890abcdef",
		Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}.ShortHexWithPrefix(),
	)

	assert.Equal(t,
		"0x100000000000000",
		Address{0x1}.ShortHexWithPrefix(),
	)

	assert.Equal(t,
		"0x1",
		Address{0, 0, 0, 0, 0, 0, 0, 0x1}.ShortHexWithPrefix(),
	)
}

func TestAddress_HexWithPrefix(t *testing.T) {

	t.Parallel()

	assert.Equal(t,
		"0x1234567890abcdef",
		Address{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}.HexWithPrefix(),
	)

	assert.Equal(t,
		"0x0100000000000000",
		Address{0x1}.HexWithPrefix(),
	)

	assert.Equal(t,
		"0x0000000000000001",
		Address{0, 0, 0, 0, 0, 0, 0, 0x1}.HexWithPrefix(),
	)
}

func TestAddress_HexToAddress(t *testing.T) {

	t.Parallel()

	type testCase struct {
		literal string
		value   []byte
	}

	for _, test := range []testCase{
		{"123", []byte{0x1, 0x23}},
		{"1", []byte{0x1}},
		// leading zero
		{"01", []byte{0x1}},
	} {

		expected := MustBytesToAddress(test.value)

		address, err := HexToAddress(test.literal)
		assert.NoError(t, err)
		assert.Equal(t, expected, address)

		address, err = HexToAddress("0x" + test.literal)
		assert.NoError(t, err)
		assert.Equal(t, expected, address)
	}
}
