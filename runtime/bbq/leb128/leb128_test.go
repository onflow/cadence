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

package leb128

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUint32(t *testing.T) {

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
			var b []byte
			b = AppendUint32(b, v)
			require.Equal(t, expected, b)

			actual, n, err := ReadUint32(b)
			require.NoError(t, err)
			require.Equal(t, v, actual)
			require.Equal(t, len(b), n)
		}
	})

	t.Run("write: max byte count", func(t *testing.T) {

		t.Parallel()

		// This test ensures that only up to the maximum number of bytes are written
		// when writing a LEB128-encoded 32-bit number (see max32bitByteCount),
		// i.e. test that only up to 5 bytes are written.

		var b []byte
		AppendUint32(b, math.MaxUint32)
		require.GreaterOrEqual(t, max32bitByteCount, len(b))
	})

	t.Run("read: max byte count", func(t *testing.T) {

		t.Parallel()

		// This test ensures that only up to the maximum number of bytes are read
		// when reading a LEB128-encoded 32-bit number (see max32bitByteCount),
		// i.e. test that only 5 of the 8 given bytes are read,
		// to ensure the LEB128 parser doesn't keep reading infinitely.

		b := []byte{0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88}
		_, n, err := ReadUint32(b)
		require.NoError(t, err)
		require.Equal(t, max32bitByteCount, n)
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
			var b []byte
			b = AppendUint64(b, v)
			require.Equal(t, expected, b)

			actual, n, err := ReadUint64(b)
			require.NoError(t, err)
			require.Equal(t, v, actual)
			require.Equal(t, len(b), n)
		}
	})

	t.Run("write: max byte count", func(t *testing.T) {

		t.Parallel()

		var b []byte
		b = AppendUint64(b, math.MaxUint64)
		require.GreaterOrEqual(t, max64bitByteCount, len(b))
	})

	t.Run("read: max byte count", func(t *testing.T) {

		t.Parallel()

		b := []byte{
			0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88,
			0x89, 0x8a, 0x8b, 0x8c, 0x8d, 0x8e, 0x8f, 0x90,
		}
		_, n, err := ReadUint64(b)
		require.NoError(t, err)
		require.Equal(t, max64bitByteCount, n)
	})
}

func TestBuf_Int32(t *testing.T) {

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
			var b []byte
			b = AppendInt32(b, v)
			require.Equal(t, expected, b)

			actual, n, err := ReadInt32(b)
			require.NoError(t, err)
			require.Equal(t, v, actual)
			require.Equal(t, len(b), n)
		}
	})

	t.Run("write: max byte count", func(t *testing.T) {

		t.Parallel()

		// This test ensures that only up to the maximum number of bytes are written
		// when writing a LEB128-encoded 32-bit number (see max32bitByteCount),
		// i.e. test that only up to 5 bytes are written.

		var b []byte
		b = AppendInt32(b, math.MaxInt32)
		require.GreaterOrEqual(t, max32bitByteCount, len(b))

		var b2 []byte
		b2 = AppendInt32(b2, math.MinInt32)
		require.GreaterOrEqual(t, max32bitByteCount, len(b2))
	})

	t.Run("read: max byte count", func(t *testing.T) {

		t.Parallel()

		// This test ensures that only up to the maximum number of bytes are read
		// when reading a LEB128-encoded 32-bit number (see max32bitByteCount),
		// i.e. test that only 5 of the 8 given bytes are read,
		// to ensure the LEB128 parser doesn't keep reading infinitely.

		b := []byte{0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88}
		_, n, err := ReadInt32(b)
		require.NoError(t, err)
		require.Equal(t, max32bitByteCount, n)
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
			var b []byte
			b = AppendInt64(b, v)
			require.Equal(t, expected, b)

			actual, n, err := ReadInt64(b)
			require.NoError(t, err)
			require.Equal(t, v, actual)
			require.Equal(t, len(b), n)
		}
	})

	t.Run("write: max byte count", func(t *testing.T) {

		t.Parallel()

		var b []byte
		b = AppendInt64(b, math.MaxInt64)
		require.GreaterOrEqual(t, max64bitByteCount, len(b))

		var b2 []byte
		b2 = AppendInt64(b2, math.MinInt64)
		require.GreaterOrEqual(t, max64bitByteCount, len(b2))
	})

	t.Run("read: max byte count", func(t *testing.T) {

		t.Parallel()

		b := []byte{
			0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88,
			0x89, 0x8a, 0x8b, 0x8c, 0x8d, 0x8e, 0x8f, 0x90,
		}
		_, n, err := ReadInt64(b)
		require.NoError(t, err)
		require.Equal(t, max64bitByteCount, n)
	})
}
