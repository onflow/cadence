/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

type Instruction interface {
	write(*WASMWriter) error
}

// InstructionLocalGet is the 'local.get' instruction
//
type InstructionLocalGet struct {
	Index uint32
}

func (i InstructionLocalGet) write(wasm *WASMWriter) error {
	err := wasm.writeOpcode(opcodeLocalGet)
	if err != nil {
		return err
	}
	return wasm.buf.writeULEB128(i.Index)
}

// InstructionI32Add is the 'i32.add' instruction
//
type InstructionI32Add struct{}

func (i InstructionI32Add) write(wasm *WASMWriter) error {
	return wasm.writeOpcode(opcodeI32Add)
}
