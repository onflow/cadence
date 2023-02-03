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

// ValueType is the type of a value
type ValueType byte

const (
	// ValueTypeI32 is the `i32` type,
	// the type of 32 bit integers.
	// The value is the byte used in the WASM binary
	ValueTypeI32 ValueType = 0x7F

	// ValueTypeI64 is the `i64` type,
	// the type of 64 bit integers.
	// The value is the byte used in the WASM binary
	ValueTypeI64 ValueType = 0x7E

	// ValueTypeFuncRef is the `funcref` type,
	// the type of first-class references to functions.
	// The value is the byte used in the WASM binary
	ValueTypeFuncRef ValueType = 0x70

	// ValueTypeExternRef is the `funcref` type,
	// the type of first-class references to objects owned by the embedder.
	// The value is the byte used in the WASM binary
	ValueTypeExternRef ValueType = 0x6F
)

// AsValueType returns the value type for the given byte,
// or 0 if the byte is not a valid value type
func AsValueType(b byte) ValueType {
	switch ValueType(b) {
	case ValueTypeI32:
		return ValueTypeI32

	case ValueTypeI64:
		return ValueTypeI64

	case ValueTypeFuncRef:
		return ValueTypeFuncRef

	case ValueTypeExternRef:
		return ValueTypeExternRef
	}

	return 0
}

func (ValueType) isBlockType() {}

func (t ValueType) write(w *WASMWriter) error {
	return w.buf.WriteByte(byte(t))
}
