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

package wasm

import (
	"io"
)

type offset int

// Buffer is a byte buffer, which allows reading and writing.
type Buffer struct {
	data   []byte
	offset offset
}

func (buf *Buffer) WriteByte(b byte) error {
	if buf.offset < offset(len(buf.data)) {
		buf.data[buf.offset] = b
	} else {
		buf.data = append(buf.data, b)
	}
	buf.offset++
	return nil
}

func (buf *Buffer) WriteBytes(data []byte) error {
	for _, b := range data {
		err := buf.WriteByte(b)
		if err != nil {
			return err
		}
	}
	return nil
}

func (buf *Buffer) Read(data []byte) (int, error) {
	n := copy(data, buf.data[buf.offset:])
	if n == 0 && len(data) != 0 {
		return 0, io.EOF
	}
	buf.offset += offset(n)
	return n, nil
}

func (buf *Buffer) ReadByte() (byte, error) {
	if buf.offset >= offset(len(buf.data)) {
		return 0, io.EOF
	}
	b := buf.data[buf.offset]
	buf.offset++
	return b, nil
}

func (buf *Buffer) PeekByte() (byte, error) {
	if buf.offset >= offset(len(buf.data)) {
		return 0, io.EOF
	}
	b := buf.data[buf.offset]
	return b, nil
}

func (buf *Buffer) ReadBytesEqual(expected []byte) (bool, error) {
	off := buf.offset
	for _, b := range expected {
		if off >= offset(len(buf.data)) {
			return false, io.EOF
		}
		if buf.data[off] != b {
			return false, nil
		}
		off++
	}
	buf.offset = off
	return true, nil
}

func (buf *Buffer) Bytes() []byte {
	return buf.data
}
