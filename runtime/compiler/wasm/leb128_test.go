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

package wasm

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuf_Uint32LEB128(t *testing.T) {

	t.Parallel()

	t.Run("DWARF spec + more", func(t *testing.T) {

		t.Parallel()

		// DWARF Debugging Information Format, Version 3, page 140

		for v, expected := range map[uint32][]byte{
			0:     {0x00},
			1:     {0x01},
			2:     {2},
			63:    {0x3f},
			64:    {0x40},
			127:   {127},
			128:   {0 + 0x80, 1},
			129:   {1 + 0x80, 1},
			130:   {2 + 0x80, 1},
			0x90:  {0x90, 0x01},
			0x100: {0x80, 0x02},
			0x101: {0x81, 0x02},
			0xff:  {0xff, 0x01},
			12857: {57 + 0x80, 100},
		} {
			var b Buffer
			err := b.writeUint32LEB128(v)
			require.NoError(t, err)
			require.Equal(t, expected, b.data)

			b.offset = 0

			actual, err := b.readUint32LEB128()
			require.NoError(t, err)
			require.Equal(t, v, actual)
		}
	})

	t.Run("write: max byte count", func(t *testing.T) {

		t.Parallel()

		// This test ensures that only up to the maximum number of bytes are written
		// when writing a LEB128-encoded 32-bit number (see max32bitLEB128ByteCount),
		// i.e. test that only up to 5 bytes are written.

		var b Buffer
		err := b.writeUint32LEB128(math.MaxUint32)
		require.NoError(t, err)
		require.GreaterOrEqual(t, max32bitLEB128ByteCount, len(b.data))
	})

	t.Run("read: max byte count", func(t *testing.T) {

		t.Parallel()

		// This test ensures that only up to the maximum number of bytes are read
		// when reading a LEB128-encoded 32-bit number (see max32bitLEB128ByteCount),
		// i.e. test that only 5 of the 8 given bytes are read,
		// to ensure the LEB128 parser doesn't keep reading infinitely.

		b := Buffer{data: []byte{0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88}}
		_, err := b.readUint32LEB128()
		require.NoError(t, err)
		require.Equal(t, offset(max32bitLEB128ByteCount), b.offset)
	})
}

func TestBuf_Uint64LEB128(t *testing.T) {

	t.Parallel()

	t.Run("DWARF spec + more", func(t *testing.T) {

		t.Parallel()

		// DWARF Debugging Information Format, Version 3, page 140

		for v, expected := range map[uint64][]byte{
			0:     {0x00},
			1:     {0x01},
			2:     {2},
			63:    {0x3f},
			64:    {0x40},
			127:   {127},
			128:   {0 + 0x80, 1},
			129:   {1 + 0x80, 1},
			130:   {2 + 0x80, 1},
			0x90:  {0x90, 0x01},
			0x100: {0x80, 0x02},
			0x101: {0x81, 0x02},
			0xff:  {0xff, 0x01},
			12857: {57 + 0x80, 100},
		} {
			var b Buffer
			err := b.writeUint64LEB128(v)
			require.NoError(t, err)
			require.Equal(t, expected, b.data)

			b.offset = 0

			actual, err := b.readUint64LEB128()
			require.NoError(t, err)
			require.Equal(t, v, actual)
		}
	})

	t.Run("write: max byte count", func(t *testing.T) {

		t.Parallel()

		var b Buffer
		err := b.writeUint64LEB128(math.MaxUint64)
		require.NoError(t, err)
		require.GreaterOrEqual(t, max64bitLEB128ByteCount, len(b.data))
	})

	t.Run("read: max byte count", func(t *testing.T) {

		t.Parallel()

		b := Buffer{data: []byte{
			0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88,
			0x89, 0x8a, 0x8b, 0x8c, 0x8d, 0x8e, 0x8f, 0x90,
		}}
		_, err := b.readUint64LEB128()
		require.NoError(t, err)
		require.Equal(t, offset(max64bitLEB128ByteCount), b.offset)
	})
}

func TestBuf_Int32LEB128(t *testing.T) {

	t.Parallel()

	t.Run("DWARF spec + more", func(t *testing.T) {

		t.Parallel()

		// DWARF Debugging Information Format, Version 3, page 141

		for v, expected := range map[int32][]byte{
			0:      {0x00},
			1:      {0x01},
			-1:     {0x7f},
			2:      {2},
			-2:     {0x7e},
			63:     {0x3f},
			-63:    {0x41},
			64:     {0xc0, 0x00},
			-64:    {0x40},
			-65:    {0xbf, 0x7f},
			127:    {127 + 0x80, 0},
			-127:   {1 + 0x80, 0x7f},
			128:    {0 + 0x80, 1},
			-128:   {0 + 0x80, 0x7f},
			129:    {1 + 0x80, 1},
			-129:   {0x7f + 0x80, 0x7e},
			-12345: {0xc7, 0x9f, 0x7f},
		} {
			var b Buffer
			err := b.writeInt32LEB128(v)
			require.NoError(t, err)
			require.Equal(t, expected, b.data)

			b.offset = 0

			actual, err := b.readInt32LEB128()
			require.NoError(t, err)
			require.Equal(t, v, actual)
		}
	})

	t.Run("write: max byte count", func(t *testing.T) {

		t.Parallel()

		// This test ensures that only up to the maximum number of bytes are written
		// when writing a LEB128-encoded 32-bit number (see max32bitLEB128ByteCount),
		// i.e. test that only up to 5 bytes are written.

		var b Buffer
		err := b.writeInt32LEB128(math.MaxInt32)
		require.NoError(t, err)
		require.GreaterOrEqual(t, max32bitLEB128ByteCount, len(b.data))

		var b2 Buffer
		err = b2.writeInt32LEB128(math.MinInt32)
		require.NoError(t, err)
		require.GreaterOrEqual(t, max32bitLEB128ByteCount, len(b.data))
	})

	t.Run("read: max byte count", func(t *testing.T) {

		t.Parallel()

		// This test ensures that only up to the maximum number of bytes are read
		// when reading a LEB128-encoded 32-bit number (see max32bitLEB128ByteCount),
		// i.e. test that only 5 of the 8 given bytes are read,
		// to ensure the LEB128 parser doesn't keep reading infinitely.

		b := Buffer{data: []byte{0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88}}
		_, err := b.readInt32LEB128()
		require.NoError(t, err)
		require.Equal(t, offset(max32bitLEB128ByteCount), b.offset)
	})
}

func TestBuf_Int64LEB128(t *testing.T) {

	t.Parallel()

	t.Run("DWARF spec + more", func(t *testing.T) {

		t.Parallel()

		// DWARF Debugging Information Format, Version 3, page 141

		for v, expected := range map[int64][]byte{
			0:      {0x00},
			1:      {0x01},
			-1:     {0x7f},
			2:      {2},
			-2:     {0x7e},
			63:     {0x3f},
			-63:    {0x41},
			64:     {0xc0, 0x00},
			-64:    {0x40},
			-65:    {0xbf, 0x7f},
			127:    {127 + 0x80, 0},
			-127:   {1 + 0x80, 0x7f},
			128:    {0 + 0x80, 1},
			-128:   {0 + 0x80, 0x7f},
			129:    {1 + 0x80, 1},
			-129:   {0x7f + 0x80, 0x7e},
			-12345: {0xc7, 0x9f, 0x7f},
		} {
			var b Buffer
			err := b.writeInt64LEB128(v)
			require.NoError(t, err)
			require.Equal(t, expected, b.data)

			b.offset = 0

			actual, err := b.readInt64LEB128()
			require.NoError(t, err)
			require.Equal(t, v, actual)
		}
	})

	t.Run("write: max byte count", func(t *testing.T) {

		t.Parallel()

		var b Buffer
		err := b.writeInt64LEB128(math.MaxInt64)
		require.NoError(t, err)
		require.GreaterOrEqual(t, max64bitLEB128ByteCount, len(b.data))

		var b2 Buffer
		err = b2.writeInt64LEB128(math.MinInt64)
		require.NoError(t, err)
		require.GreaterOrEqual(t, max64bitLEB128ByteCount, len(b.data))
	})

	t.Run("read: max byte count", func(t *testing.T) {

		t.Parallel()

		b := Buffer{data: []byte{
			0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88,
			0x89, 0x8a, 0x8b, 0x8c, 0x8d, 0x8e, 0x8f, 0x90,
		}}
		_, err := b.readInt64LEB128()
		require.NoError(t, err)
		require.Equal(t, offset(max64bitLEB128ByteCount), b.offset)
	})
}

func TestBuf_WriteSpaceAndSize(t *testing.T) {

	t.Parallel()

	var b Buffer

	err := b.WriteByte(101)
	require.NoError(t, err)
	err = b.WriteByte(102)
	require.NoError(t, err)

	off, err := b.writeFixedUint32LEB128Space()
	require.NoError(t, err)
	require.Equal(t, offset(2), off)
	require.Equal(t,
		[]byte{
			101, 102,
			0, 0, 0, 0, 0,
		},
		b.data,
	)

	err = b.WriteByte(104)
	require.NoError(t, err)
	err = b.WriteByte(105)
	require.NoError(t, err)
	err = b.WriteByte(106)
	require.NoError(t, err)

	err = b.writeUint32LEB128SizeAt(off)
	require.NoError(t, err)
	require.Equal(t,
		[]byte{
			101, 102,
			0x83, 0x80, 0x80, 0x80, 0,
			104, 105, 106,
		},
		b.data,
	)
}
