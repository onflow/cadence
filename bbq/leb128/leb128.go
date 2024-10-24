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
	"fmt"
)

// max32bitByteCount is the maximum number of bytes a 32-bit integer
// (signed or unsigned) may be encoded as. From
// https://webassembly.github.io/spec/core/binary/values.html#binary-int:
//
// "the total number of bytes encoding a value of type uN must not exceed ceil(N/7) bytes"
// "the total number of bytes encoding a value of type sN must not exceed ceil(N/7) bytes"
const max32bitByteCount = 5

// max64bitByteCount is the maximum number of bytes a 64-bit integer
// (signed or unsigned) may be encoded as. From
// https://webassembly.github.io/spec/core/binary/values.html#binary-int:
//
// "the total number of bytes encoding a value of type uN must not exceed ceil(N/7) bytes"
// "the total number of bytes encoding a value of type sN must not exceed ceil(N/7) bytes"
const max64bitByteCount = 10

// AppendUint32 encodes and writes the given unsigned 32-bit integer
// in canonical (with the fewest bytes possible) unsigned little-endian base-128 format
func AppendUint32(data []byte, v uint32) []byte {
	if v < 128 {
		data = append(data, uint8(v))
		return data
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
		data = append(data, c)
	}
	return data
}

// AppendUint64 encodes and writes the given unsigned 64-bit integer
// in canonical (with the fewest bytes possible) unsigned little-endian base-128 format
func AppendUint64(data []byte, v uint64) []byte {
	if v < 128 {
		data = append(data, uint8(v))
		return data
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
		data = append(data, c)
	}
	return data
}

// AppendUint32FixedLength encodes and writes the given unsigned 32-bit integer
// in non-canonical (fixed-size, instead of with the fewest bytes possible)
// unsigned little-endian base-128 format
func AppendUint32FixedLength(data []byte, v uint32, length int) ([]byte, error) {
	for i := 0; i < length; i++ {
		c := uint8(v & 0x7f)
		v >>= 7
		if i < length-1 {
			c |= 0x80
		}
		data = append(data, c)
	}
	if v != 0 {
		return nil, fmt.Errorf("length too small: %d", length)
	}
	return data, nil
}

// ReadUint32 reads and decodes an unsigned 32-bit integer
func ReadUint32(data []byte) (result uint32, count int, err error) {
	var shift uint
	// only read up to maximum number of bytes
	for i := 0; i < max32bitByteCount; i++ {
		if i >= len(data) {
			return 0, 0, fmt.Errorf("data too short: %d", len(data))
		}
		b := data[i]
		count++
		result |= (uint32(b & 0x7F)) << shift
		// check high order bit of byte
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}
	return result, count, nil
}

// ReadUint64 reads and decodes an unsigned 64-bit integer
func ReadUint64(data []byte) (result uint64, count int, err error) {
	var shift uint
	// only read up to maximum number of bytes
	for i := 0; i < max64bitByteCount; i++ {
		if i >= len(data) {
			return 0, 0, fmt.Errorf("data too short: %d", len(data))
		}
		b := data[i]
		count++
		result |= (uint64(b & 0x7F)) << shift
		// check high order bit of byte
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}
	return result, count, nil
}

// AppendInt32 encodes and writes the given signed 32-bit integer
// in canonical (with the fewest bytes possible) signed little-endian base-128 format
func AppendInt32(data []byte, v int32) []byte {
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
		data = append(data, c)
	}
	return data
}

// AppendInt64 encodes and writes the given signed 64-bit integer
// in canonical (with the fewest bytes possible) signed little-endian base-128 format
func AppendInt64(data []byte, v int64) []byte {
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
		data = append(data, c)
	}
	return data
}

// ReadInt32 reads and decodes a signed 32-bit integer
func ReadInt32(data []byte) (result int32, count int, err error) {
	var b byte = 0x80
	var signBits int32 = -1
	for i := 0; (b&0x80 == 0x80) && i < max32bitByteCount; i++ {
		if i >= len(data) {
			return 0, 0, fmt.Errorf("data too short: %d", len(data))
		}
		b = data[i]
		count++
		result += int32(b&0x7f) << (i * 7)
		signBits <<= 7
	}
	if ((signBits >> 1) & result) != 0 {
		result += signBits
	}
	return result, count, nil
}

// ReadInt64 reads and decodes a signed 64-bit integer
func ReadInt64(data []byte) (result int64, count int, err error) {
	var b byte = 0x80
	var signBits int64 = -1
	for i := 0; (b&0x80 == 0x80) && i < max64bitByteCount; i++ {
		if i >= len(data) {
			return 0, 0, fmt.Errorf("data too short: %d", len(data))
		}
		b = data[i]
		count++
		result += int64(b&0x7f) << (i * 7)
		signBits <<= 7
	}
	if ((signBits >> 1) & result) != 0 {
		result += signBits
	}
	return result, count, nil
}
