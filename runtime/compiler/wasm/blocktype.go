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

const emptyBlockType byte = 0x40

type BlockType interface {
	isBlockType()
	write(writer *WASMWriter) error
}

type TypeIndexBlockType struct {
	TypeIndex uint32
}

func (t TypeIndexBlockType) write(w *WASMWriter) error {
	// "the type index in a block type is encoded as a positive signed integer,
	// so that its signed LEB128 bit pattern cannot collide with the encoding of value types or the special code 0x40,
	// which correspond to the LEB128 encoding of negative integers.
	// To avoid any loss in the range of allowed indices, it is treated as a 33 bit signed integer."
	return w.buf.writeInt64LEB128(int64(t.TypeIndex))
}

func (TypeIndexBlockType) isBlockType() {}
