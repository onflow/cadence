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
	"fmt"
)

// max32bitLEB128ByteCount is the maximum number of bytes a 32-bit integer
// (signed or unsigned) may be encoded as. From
// https://webassembly.github.io/spec/core/binary/values.html#binary-int:
//
// "the total number of bytes encoding a value of type uN must not exceed ceil(N/7) bytes"
// "the total number of bytes encoding a value of type sN must not exceed ceil(N/7) bytes"
//
const max32bitLEB128ByteCount = 5

// max64bitLEB128ByteCount is the maximum number of bytes a 64-bit integer
// (signed or unsigned) may be encoded as. From
// https://webassembly.github.io/spec/core/binary/values.html#binary-int:
//
// "the total number of bytes encoding a value of type uN must not exceed ceil(N/7) bytes"
// "the total number of bytes encoding a value of type sN must not exceed ceil(N/7) bytes"
//
const max64bitLEB128ByteCount = 10

// writeUint32LEB128 encodes and writes the given unsigned 32-bit integer
// in canonical (with the fewest bytes possible) unsigned little endian base 128 format
func (buf *Buffer) writeUint32LEB128(v uint32) error {
	if v < 128 {
		err := buf.WriteByte(uint8(v))
		if err != nil {
			return err
		}
		return nil
	}
	more := true
	for more {
		// low order 7 bits of value
		c := uint8(v & 0x7f)
		v >>= 7
		// more bits to come?
		more = v != 0
		if more {
			// set high order bit of byte
			c |= 0x80
		}
		// emit byte
		err := buf.WriteByte(c)
		if err != nil {
			return err
		}
	}
	return nil
}

// writeUint64LEB128 encodes and writes the given unsigned 64-bit integer
// in canonical (with the fewest bytes possible) unsigned little endian base 128 format
func (buf *Buffer) writeUint64LEB128(v uint64) error {
	if v < 128 {
		err := buf.WriteByte(uint8(v))
		if err != nil {
			return err
		}
		return nil
	}
	more := true
	for more {
		// low order 7 bits of value
		c := uint8(v & 0x7f)
		v >>= 7
		// more bits to come?
		more = v != 0
		if more {
			// set high order bit of byte
			c |= 0x80
		}
		// emit byte
		err := buf.WriteByte(c)
		if err != nil {
			return err
		}
	}
	return nil
}

// writeUint32LEB128FixedLength encodes and writes the given unsigned 32-bit integer
// in non-canonical (fixed-size, instead of with the fewest bytes possible)
// unsigned little endian base 128 format
//
func (buf *Buffer) writeUint32LEB128FixedLength(v uint32, length int) error {
	for i := 0; i < length; i++ {
		c := uint8(v & 0x7f)
		v >>= 7
		if i < length-1 {
			c |= 0x80
		}
		err := buf.WriteByte(c)
		if err != nil {
			return err
		}
	}
	if v != 0 {
		return fmt.Errorf("writeUint32LEB128FixedLength: length too small: %d", length)
	}
	return nil
}

// readUint32LEB128 reads and decodes an unsigned 32-bit integer
//
func (buf *Buffer) readUint32LEB128() (uint32, error) {
	var result uint32
	var shift, i uint
	// only read up to maximum number of bytes
	for i < max32bitLEB128ByteCount {
		b, err := buf.ReadByte()
		if err != nil {
			return 0, err
		}
		result |= (uint32(b & 0x7F)) << shift
		// check high order bit of byte
		if b&0x80 == 0 {
			break
		}
		shift += 7
		i++
	}
	return result, nil
}

// readUint64LEB128 reads and decodes an unsigned 64-bit integer
func (buf *Buffer) readUint64LEB128() (uint64, error) {
	var result uint64
	var shift, i uint
	// only read up to maximum number of bytes
	for i < max64bitLEB128ByteCount {
		b, err := buf.ReadByte()
		if err != nil {
			return 0, err
		}
		result |= (uint64(b & 0x7F)) << shift
		// check high order bit of byte
		if b&0x80 == 0 {
			break
		}
		shift += 7
		i++
	}
	return result, nil
}

// writeInt32LEB128 encodes and writes the given signed 32-bit integer
// in canonical (with the fewest bytes possible) signed little endian base 128 format
func (buf *Buffer) writeInt32LEB128(v int32) error {
	more := true
	for more {
		// low order 7 bits of value
		c := uint8(v & 0x7f)
		sign := uint8(v & 0x40)
		v >>= 7
		more = !((v == 0 && sign == 0) || (v == -1 && sign != 0))
		if more {
			c |= 0x80
		}
		err := buf.WriteByte(c)
		if err != nil {
			return err
		}
	}
	return nil
}

// writeInt64LEB128 encodes and writes the given signed 64-bit integer
// in canonical (with the fewest bytes possible) signed little endian base 128 format
func (buf *Buffer) writeInt64LEB128(v int64) error {
	more := true
	for more {
		// low order 7 bits of value
		c := uint8(v & 0x7f)
		sign := uint8(v & 0x40)
		v >>= 7
		more = !((v == 0 && sign == 0) || (v == -1 && sign != 0))
		if more {
			c |= 0x80
		}
		err := buf.WriteByte(c)
		if err != nil {
			return err
		}
	}
	return nil
}

// readInt32LEB128 reads and decodes a signed 32-bit integer
//
func (buf *Buffer) readInt32LEB128() (int32, error) {
	var result int32
	var i uint
	var b byte = 0x80
	var signBits int32 = -1
	var err error
	for (b&0x80 == 0x80) && i < max32bitLEB128ByteCount {
		b, err = buf.ReadByte()
		if err != nil {
			return 0, err
		}
		result += int32(b&0x7f) << (i * 7)
		signBits <<= 7
		i++
	}
	if ((signBits >> 1) & result) != 0 {
		result += signBits
	}
	return result, nil
}

// readInt64LEB128 reads and decodes a signed 64-bit integer
//
func (buf *Buffer) readInt64LEB128() (int64, error) {
	var result int64
	var i uint
	var b byte = 0x80
	var signBits int64 = -1
	var err error
	for (b&0x80 == 0x80) && i < max64bitLEB128ByteCount {
		b, err = buf.ReadByte()
		if err != nil {
			return 0, err
		}
		result += int64(b&0x7f) << (i * 7)
		signBits <<= 7
		i++
	}
	if ((signBits >> 1) & result) != 0 {
		result += signBits
	}
	return result, nil
}

// writeFixedUint32LEB128Space writes a non-canonical 5-byte fixed-size space
// (instead of the minimal size if canonical encoding would be used)
//
func (buf *Buffer) writeFixedUint32LEB128Space() (offset, error) {
	off := buf.offset
	for i := 0; i < max32bitLEB128ByteCount; i++ {
		err := buf.WriteByte(0)
		if err != nil {
			return 0, err
		}
	}
	return off, nil
}

// writeUint32LEB128SizeAt writes the size, the number of bytes
// between the given offset and the current offset,
// as an uint32 in non-canonical 5-byte fixed-size format
// (instead of the minimal size if canonical encoding would be used)
// at the given offset
//
func (buf *Buffer) writeUint32LEB128SizeAt(off offset) error {
	currentOff := buf.offset
	if currentOff < max32bitLEB128ByteCount || currentOff-max32bitLEB128ByteCount < off {
		return fmt.Errorf("writeUint32LEB128SizeAt: invalid offset: %d", off)
	}
	size := uint32(currentOff - off - max32bitLEB128ByteCount)
	buf.offset = off
	defer func() {
		buf.offset = currentOff
	}()
	return buf.writeUint32LEB128FixedLength(size, max32bitLEB128ByteCount)
}
