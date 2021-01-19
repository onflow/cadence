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

import (
	"unicode/utf8"
)

// WASMWriter allows writing WASM binaries
//
type WASMWriter struct {
	buf *buf
}

// writeMagicAndVersion writes the magic byte sequence and version at the beginning of the WASM binary
//
func (w *WASMWriter) writeMagicAndVersion() error {
	err := w.buf.WriteBytes(wasmMagic)
	if err != nil {
		return err
	}
	return w.buf.WriteBytes(wasmVersion)
}

// writeSection writes a section in the WASM binary, with the given section ID and the given content.
// The content is a function that writes the contents of the section.
//
func (w *WASMWriter) writeSection(sectionID sectionID, content func() error) error {
	// write the section ID
	err := w.buf.WriteByte(byte(sectionID))
	if err != nil {
		return err
	}

	// write the size and the content
	return w.writeContentWithSize(content)
}

// writeContentWithSize writes the size of the content,
// and the content itself
//
func (w *WASMWriter) writeContentWithSize(content func() error) error {

	// write the temporary placeholder for the size
	sizeOffset, err := w.buf.writeFixedUint32LEB128Space()
	if err != nil {
		return err
	}

	// write the content
	err = content()
	if err != nil {
		return err
	}

	// write the actual size into the size placeholder
	return w.buf.writeUint32LEB128SizeAt(sizeOffset)
}

// writeTypeSection writes the section that declares all function types
// so they can be referenced by index
//
func (w *WASMWriter) writeTypeSection(funcTypes []*FunctionType) error {
	return w.writeSection(sectionIDType, func() error {

		// write the number of types
		err := w.buf.writeUint32LEB128(uint32(len(funcTypes)))
		if err != nil {
			return err
		}

		// write each type
		for _, funcType := range funcTypes {
			err = w.writeFuncType(funcType)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// writeFuncType writes the function type
//
func (w *WASMWriter) writeFuncType(funcType *FunctionType) error {
	// write the type
	err := w.buf.WriteByte(functionTypeIndicator)
	if err != nil {
		return err
	}

	// write the number of parameters
	err = w.buf.writeUint32LEB128(uint32(len(funcType.Params)))
	if err != nil {
		return err
	}

	// write the type of each parameter
	for _, paramType := range funcType.Params {
		err = w.buf.WriteByte(byte(paramType))
		if err != nil {
			return err
		}
	}

	// write the number of results
	err = w.buf.writeUint32LEB128(uint32(len(funcType.Results)))
	if err != nil {
		return err
	}

	// write the type of each result
	for _, resultType := range funcType.Results {
		err = w.buf.WriteByte(byte(resultType))
		if err != nil {
			return err
		}
	}

	return nil
}

// writeImportSection writes the section that declares all imports
//
func (w *WASMWriter) writeImportSection(imports []*Import) error {
	return w.writeSection(sectionIDImport, func() error {

		// write the number of imports
		err := w.buf.writeUint32LEB128(uint32(len(imports)))
		if err != nil {
			return err
		}

		// write each import
		for _, im := range imports {
			err = w.writeImport(im)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// writeImport writes the import
//
func (w *WASMWriter) writeImport(im *Import) error {
	// write the module
	err := w.writeName(im.Module)
	if err != nil {
		return err
	}

	// write the name
	err = w.writeName(im.Name)
	if err != nil {
		return err
	}

	// TODO: add support for tables, memories, and globals

	// write the type indicator
	err = w.buf.WriteByte(byte(importIndicatorFunction))
	if err != nil {
		return err
	}

	// write the type index
	return w.buf.writeUint32LEB128(im.TypeIndex)
}

// writeFunctionSection writes the section that declares the types of functions.
// The bodies of these functions will later be provided in the code section
//
func (w *WASMWriter) writeFunctionSection(functions []*Function) error {
	return w.writeSection(sectionIDFunction, func() error {
		// write the number of functions
		err := w.buf.writeUint32LEB128(uint32(len(functions)))
		if err != nil {
			return err
		}

		// write the type index for each function
		for _, function := range functions {
			err = w.buf.writeUint32LEB128(function.TypeIndex)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// writeExportSection writes the section that declares all exports
//
func (w *WASMWriter) writeExportSection(exports []*Export) error {
	return w.writeSection(sectionIDExport, func() error {

		// write the number of exports
		err := w.buf.writeUint32LEB128(uint32(len(exports)))
		if err != nil {
			return err
		}

		// write each export
		for _, export := range exports {
			err = w.writeExport(export)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// writeExport writes the export
//
func (w *WASMWriter) writeExport(export *Export) error {

	// write the name
	err := w.writeName(export.Name)
	if err != nil {
		return err
	}

	// TODO: add support for tables, memories, and globals

	// write the type indicator
	err = w.buf.WriteByte(byte(exportIndicatorFunction))
	if err != nil {
		return err
	}

	// write the function index
	return w.buf.writeUint32LEB128(export.FunctionIndex)
}

// writeCodeSection writes the section that provides the function bodies for the functions
// declared by the function section (which only provides the function types)
//
func (w *WASMWriter) writeCodeSection(functions []*Function) error {
	return w.writeSection(sectionIDCode, func() error {
		// write the number of code entries (one for each function)
		err := w.buf.writeUint32LEB128(uint32(len(functions)))
		if err != nil {
			return err
		}

		// write the code for each function
		for _, function := range functions {

			err := w.writeFunctionBody(function.Code)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// writeFunctionBody writes the body of the function
//
func (w *WASMWriter) writeFunctionBody(code *Code) error {
	return w.writeContentWithSize(func() error {

		// write the number of locals
		err := w.buf.writeUint32LEB128(uint32(len(code.Locals)))
		if err != nil {
			return err
		}

		// TODO: run-length encode
		// write each local
		for _, localValType := range code.Locals {
			err = w.buf.writeUint32LEB128(1)
			if err != nil {
				return err
			}

			err = w.buf.WriteByte(byte(localValType))
			if err != nil {
				return err
			}
		}

		err = w.writeInstructions(code.Instructions)
		if err != nil {
			return err
		}

		return w.writeOpcode(opcodeEnd)
	})
}

// writeInstructions writes an instruction sequence
//
func (w *WASMWriter) writeInstructions(instructions []Instruction) error {
	for _, instruction := range instructions {
		err := instruction.write(w)
		if err != nil {
			return err
		}
	}
	return nil
}

// writeOpcode writes the opcode of an instruction
//
func (w *WASMWriter) writeOpcode(opcodes ...opcode) error {
	for _, b := range opcodes {
		err := w.buf.WriteByte(byte(b))
		if err != nil {
			return err
		}
	}
	return nil
}

// writeName writes a name, a UTF-8 byte sequence
//
func (w *WASMWriter) writeName(name string) error {

	// ensure the name is valid UTF-8
	if !utf8.ValidString(name) {
		return InvalidNonUTF8NameError{
			Name:   name,
			Offset: int(w.buf.offset),
		}
	}

	// write the length
	err := w.buf.writeUint32LEB128(uint32(len(name)))
	if err != nil {
		return err
	}

	// write the name
	return w.buf.WriteBytes([]byte(name))
}

// writeBlockInstructionArgument writes a block instruction argument
//
func (w *WASMWriter) writeBlockInstructionArgument(block Block, allowElse bool) error {

	// write the block type
	if block.BlockType != nil {
		err := block.BlockType.write(w)
		if err != nil {
			return err
		}
	} else {
		err := w.buf.WriteByte(emptyBlockType)
		if err != nil {
			return err
		}
	}

	// write the first sequence of instructions
	err := w.writeInstructions(block.Instructions1)
	if err != nil {
		return err
	}

	// write the second sequence of instructions.
	// in an if-instruction, this is the else branch.
	// in other instructions, it is not allowed.

	if len(block.Instructions2) > 0 {
		if !allowElse {
			return InvalidBlockSecondInstructionsError{
				Offset: int(w.buf.offset),
			}
		}

		err := w.writeOpcode(opcodeElse)
		if err != nil {
			return err
		}

		err = w.writeInstructions(block.Instructions2)
		if err != nil {
			return err
		}
	}

	// write the implicit end instruction / opcode

	return InstructionEnd{}.write(w)
}
