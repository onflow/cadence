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
	"math"
	"unicode/utf8"
)

// WASMReader allows reading WASM binaries
//
type WASMReader struct {
	buf    *buf
	Module Module
}

// readMagicAndVersion reads the magic byte sequence and version at the beginning of the WASM binary
//
// See https://webassembly.github.io/spec/core/binary/modules.html#binary-module:
//
// The encoding of a module starts with a preamble containing a 4-byte magic number [...] and a version field.
//
func (r *WASMReader) readMagicAndVersion() error {

	// Read the magic
	equal, err := r.buf.ReadBytesEqual(wasmMagic)
	if err != nil || !equal {
		return InvalidMagicError{
			Offset:    int(r.buf.offset),
			ReadError: err,
		}
	}

	// Read the version
	equal, err = r.buf.ReadBytesEqual(wasmVersion)
	if err != nil || !equal {
		return InvalidVersionError{
			Offset:    int(r.buf.offset),
			ReadError: err,
		}
	}

	return nil
}

// readSection reads a section in the WASM binary
//
func (r *WASMReader) readSection() error {
	// read the section ID
	sectionIDOffset := r.buf.offset
	b, err := r.buf.ReadByte()

	sectionID := sectionID(b)

	if err != nil {
		return InvalidSectionIDError{
			SectionID: sectionID,
			Offset:    int(sectionIDOffset),
			ReadError: err,
		}
	}

	invalidDuplicateSectionError := func() error {
		return InvalidDuplicateSectionError{
			SectionID: sectionID,
			Offset:    int(sectionIDOffset),
		}
	}

	switch sectionID {
	case sectionIDType:
		if r.Module.Types != nil {
			return invalidDuplicateSectionError()
		}

		err = r.readTypeSection()
		if err != nil {
			return err
		}

	case sectionIDImport:
		if r.Module.Imports != nil {
			return invalidDuplicateSectionError()
		}

		err = r.readImportSection()
		if err != nil {
			return err
		}

	case sectionIDFunction:
		if r.Module.functionTypeIndices != nil {
			return invalidDuplicateSectionError()
		}

		err = r.readFunctionSection()
		if err != nil {
			return err
		}
	case sectionIDExport:
		if r.Module.Exports != nil {
			return invalidDuplicateSectionError()
		}

		err = r.readExportSection()
		if err != nil {
			return err
		}

	case sectionIDCode:
		if r.Module.functionBodies != nil {
			return invalidDuplicateSectionError()
		}

		err = r.readCodeSection()
		if err != nil {
			return err
		}

	default:
		return InvalidSectionIDError{
			SectionID: sectionID,
			Offset:    int(sectionIDOffset),
		}
	}

	return nil
}

// readSectionSize reads the content size of a section
//
func (r *WASMReader) readSectionSize() error {
	// read the size
	sizeOffset := r.buf.offset
	// TODO: use size
	_, err := r.buf.readUint32LEB128()
	if err != nil {
		return InvalidSectionSizeError{
			Offset:    int(sizeOffset),
			ReadError: err,
		}
	}

	return nil
}

// readTypeSection reads the section that declares all function types
// so they can be referenced by index
//
func (r *WASMReader) readTypeSection() error {

	err := r.readSectionSize()
	if err != nil {
		return err
	}

	// read the number of types
	countOffset := r.buf.offset
	count, err := r.buf.readUint32LEB128()
	if err != nil {
		return InvalidTypeSectionTypeCountError{
			Offset:    int(countOffset),
			ReadError: err,
		}
	}

	funcTypes := make([]*FunctionType, count)

	// read each type
	for i := uint32(0); i < count; i++ {
		funcType, err := r.readFuncType()
		if err != nil {
			return err
		}
		funcTypes[i] = funcType
	}

	r.Module.Types = funcTypes

	return nil
}

// readFuncType reads a function type
//
func (r *WASMReader) readFuncType() (*FunctionType, error) {
	// read the function type indicator
	funcTypeIndicatorOffset := r.buf.offset
	funcTypeIndicator, err := r.buf.ReadByte()
	if err != nil || funcTypeIndicator != functionTypeIndicator {
		return nil, InvalidFuncTypeIndicatorError{
			Offset:            int(funcTypeIndicatorOffset),
			FuncTypeIndicator: funcTypeIndicator,
			ReadError:         err,
		}
	}

	// read the number of parameters
	parameterCountOffset := r.buf.offset
	parameterCount, err := r.buf.readUint32LEB128()
	if err != nil {
		return nil, InvalidFuncTypeParameterCountError{
			Offset:    int(parameterCountOffset),
			ReadError: err,
		}
	}

	// read the type of each parameter

	parameterTypes := make([]ValueType, parameterCount)

	for i := uint32(0); i < parameterCount; i++ {
		parameterType, err := r.readValType()
		if err != nil {
			return nil, InvalidFuncTypeParameterTypeError{
				Index:     int(i),
				ReadError: err,
			}
		}
		parameterTypes[i] = parameterType
	}

	// read the number of results
	resultCountOffset := r.buf.offset
	resultCount, err := r.buf.readUint32LEB128()
	if err != nil {
		return nil, InvalidFuncTypeResultCountError{
			Offset:    int(resultCountOffset),
			ReadError: err,
		}
	}

	// read the type of each result

	resultTypes := make([]ValueType, resultCount)

	for i := uint32(0); i < resultCount; i++ {
		resultType, err := r.readValType()
		if err != nil {
			return nil, InvalidFuncTypeResultTypeError{
				Index:     int(i),
				ReadError: err,
			}
		}
		resultTypes[i] = resultType
	}

	return &FunctionType{
		Params:  parameterTypes,
		Results: resultTypes,
	}, nil
}

// readValType reads a value type
//
func (r *WASMReader) readValType() (ValueType, error) {
	valTypeOffset := r.buf.offset
	b, err := r.buf.ReadByte()

	valType := ValueType(b)

	if err != nil {
		return 0, InvalidValTypeError{
			Offset:    int(valTypeOffset),
			ValType:   valType,
			ReadError: err,
		}
	}

	switch valType {
	case ValueTypeI32, ValueTypeI64:
		return valType, nil
	}

	return 0, InvalidValTypeError{
		Offset:  int(valTypeOffset),
		ValType: valType,
	}
}

// readImportSection reads the section that declares the imports
//
func (r *WASMReader) readImportSection() error {

	err := r.readSectionSize()
	if err != nil {
		return err
	}

	// read the number of imports
	countOffset := r.buf.offset
	count, err := r.buf.readUint32LEB128()
	if err != nil {
		return InvalidImportSectionImportCountError{
			Offset:    int(countOffset),
			ReadError: err,
		}
	}

	imports := make([]*Import, count)

	// read each import
	for i := uint32(0); i < count; i++ {
		im, err := r.readImport()
		if err != nil {
			return InvalidImportError{
				Index:     int(i),
				ReadError: err,
			}
		}
		imports[i] = im
	}

	r.Module.Imports = imports

	return nil
}

// readImport reads an import in the import section
//
func (r *WASMReader) readImport() (*Import, error) {

	// read the module
	module, err := r.readName()
	if err != nil {
		return nil, err
	}

	// read the name
	name, err := r.readName()
	if err != nil {
		return nil, err
	}

	// read the type indicator
	indicatorOffset := r.buf.offset
	b, err := r.buf.ReadByte()

	indicator := importIndicator(b)

	// TODO: add support for tables, memories, and globals

	if err != nil || indicator != importIndicatorFunction {
		return nil, InvalidImportIndicatorError{
			ImportIndicator: indicator,
			Offset:          int(indicatorOffset),
			ReadError:       err,
		}
	}

	// read the type index
	typeIndexOffset := r.buf.offset
	typeIndex, err := r.buf.readUint32LEB128()
	if err != nil {
		return nil, InvalidImportSectionTypeIndexError{
			Offset:    int(typeIndexOffset),
			ReadError: err,
		}
	}

	return &Import{
		Module:    module,
		Name:      name,
		TypeIndex: typeIndex,
	}, nil
}

// readFunctionSection reads the section that declares the types of functions.
// The bodies of these functions will later be provided in the code section
//
func (r *WASMReader) readFunctionSection() error {

	err := r.readSectionSize()
	if err != nil {
		return err
	}

	// read the number of functions
	countOffset := r.buf.offset
	count, err := r.buf.readUint32LEB128()
	if err != nil {
		return InvalidFunctionSectionFunctionCountError{
			Offset:    int(countOffset),
			ReadError: err,
		}
	}

	functionTypeIndices := make([]uint32, count)

	// read the type index for each function
	for i := uint32(0); i < count; i++ {
		typeIndexOffset := r.buf.offset
		typeIndex, err := r.buf.readUint32LEB128()
		if err != nil {
			return InvalidFunctionSectionTypeIndexError{
				Index:     int(i),
				Offset:    int(typeIndexOffset),
				ReadError: err,
			}
		}
		functionTypeIndices[i] = typeIndex
	}

	r.Module.functionTypeIndices = functionTypeIndices

	return nil
}

// readExportSection reads the section that declares the exports
//
func (r *WASMReader) readExportSection() error {

	err := r.readSectionSize()
	if err != nil {
		return err
	}

	// read the number of exports
	countOffset := r.buf.offset
	count, err := r.buf.readUint32LEB128()
	if err != nil {
		return InvalidExportSectionExportCountError{
			Offset:    int(countOffset),
			ReadError: err,
		}
	}

	exports := make([]*Export, count)

	// read each export
	for i := uint32(0); i < count; i++ {
		im, err := r.readExport()
		if err != nil {
			return InvalidExportError{
				Index:     int(i),
				ReadError: err,
			}
		}
		exports[i] = im
	}

	r.Module.Exports = exports

	return nil
}

// readExport reads an export in the export section
//
func (r *WASMReader) readExport() (*Export, error) {

	// read the name
	name, err := r.readName()
	if err != nil {
		return nil, err
	}

	// read the type indicator
	indicatorOffset := r.buf.offset
	b, err := r.buf.ReadByte()

	indicator := exportIndicator(b)

	// TODO: add support for tables, memories, and globals

	if err != nil || indicator != exportIndicatorFunction {
		return nil, InvalidExportIndicatorError{
			ExportIndicator: indicator,
			Offset:          int(indicatorOffset),
			ReadError:       err,
		}
	}

	// read the function type ID
	functionIndexOffset := r.buf.offset
	functionIndex, err := r.buf.readUint32LEB128()
	if err != nil {
		return nil, InvalidExportSectionFunctionIndexError{
			Offset:    int(functionIndexOffset),
			ReadError: err,
		}
	}

	return &Export{
		Name:          name,
		FunctionIndex: functionIndex,
	}, nil
}

// readCodeSection reads the section that provides the function bodies for the functions
// declared by the function section (which only provides the function types)
//
func (r *WASMReader) readCodeSection() error {

	err := r.readSectionSize()
	if err != nil {
		return err
	}

	// read the number of functions
	countOffset := r.buf.offset
	count, err := r.buf.readUint32LEB128()
	if err != nil {
		return InvalidCodeSectionFunctionCountError{
			Offset:    int(countOffset),
			ReadError: err,
		}
	}

	// read the code of each function

	functionBodies := make([]*Code, count)

	for i := uint32(0); i < count; i++ {
		functionBody, err := r.readFunctionBody()
		if err != nil {
			return InvalidFunctionCodeError{
				Index:     int(i),
				ReadError: err,
			}
		}
		functionBodies[i] = functionBody
	}

	r.Module.functionBodies = functionBodies

	return nil
}

// readFunctionBody reads the body (locals and instruction) of one function in the code section
//
func (r *WASMReader) readFunctionBody() (*Code, error) {

	// read the size
	sizeOffset := r.buf.offset
	// TODO: use size
	_, err := r.buf.readUint32LEB128()
	if err != nil {
		return nil, InvalidCodeSizeError{
			Offset:    int(sizeOffset),
			ReadError: err,
		}
	}

	// read the locals
	locals, err := r.readLocals()
	if err != nil {
		return nil, err
	}

	// read the instructions
	instructions, err := r.readInstructions()
	if err != nil {
		return nil, err
	}

	return &Code{
		Locals:       locals,
		Instructions: instructions,
	}, nil
}

// readLocals reads the locals for one function in the code sections
//
func (r *WASMReader) readLocals() ([]ValueType, error) {
	// read the number of locals
	localsCountOffset := r.buf.offset
	localsCount, err := r.buf.readUint32LEB128()
	if err != nil {
		return nil, InvalidCodeSectionLocalsCountError{
			Offset:    int(localsCountOffset),
			ReadError: err,
		}
	}

	locals := make([]ValueType, localsCount)

	// read each local
	for i := uint32(0); i < localsCount; {
		compressedLocalsCountOffset := r.buf.offset
		compressedLocalsCount, err := r.buf.readUint32LEB128()
		if err != nil {
			return nil, InvalidCodeSectionCompressedLocalsCountError{
				Offset:    int(compressedLocalsCountOffset),
				ReadError: err,
			}
		}

		localTypeOffset := r.buf.offset
		localType, err := r.readValType()
		if err != nil {
			return nil, InvalidCodeSectionLocalTypeError{
				Offset:    int(localTypeOffset),
				ReadError: err,
			}
		}

		locals[i] = localType

		i += compressedLocalsCount

		if i > localsCount {
			return nil, CodeSectionLocalsCountMismatchError{
				Actual:   i,
				Expected: localsCount,
			}
		}
	}

	return locals, nil
}

// readInstructions reads the instructions for one function in the code sections
//
func (r *WASMReader) readInstructions() (instructions []Instruction, err error) {
	for {
		instruction, err := r.readInstruction()
		if err != nil {
			return nil, err
		}

		if _, ok := instruction.(InstructionEnd); ok {
			return instructions, nil
		}

		instructions = append(instructions, instruction)
	}
}

// readName reads a name
//
func (r *WASMReader) readName() (string, error) {

	// read the length
	lengthOffset := r.buf.offset
	length, err := r.buf.readUint32LEB128()
	if err != nil {
		return "", InvalidNameLengthError{
			Offset:    int(lengthOffset),
			ReadError: err,
		}
	}

	// read the name
	nameOffset := r.buf.offset
	name := make([]byte, length)
	n, err := r.buf.Read(name)
	if err != nil {
		return "", InvalidNameError{
			Offset:    int(nameOffset),
			ReadError: err,
		}
	}

	readCount := uint32(n)

	// ensure the full name was read
	if readCount != length {
		return "", IncompleteNameError{
			Offset:   int(nameOffset),
			Expected: length,
			Actual:   readCount,
		}
	}

	// ensure the name is valid UTF-8
	if !utf8.Valid(name) {
		return "", InvalidNonUTF8NameError{
			Offset: int(nameOffset),
			Name:   string(name),
		}
	}

	return string(name), nil
}

// readUint32LEB128InstructionArgument reads a uint32 instruction argument
// (in LEB128 format)
//
func (r *WASMReader) readUint32LEB128InstructionArgument() (uint32, error) {
	offset := r.buf.offset
	v, err := r.buf.readUint32LEB128()
	if err != nil {
		return 0, InvalidInstructionArgumentError{
			Offset:    int(offset),
			ReadError: err,
		}
	}
	return v, nil
}

// readInt32LEB128InstructionArgument reads an int32 instruction argument
// (in LEB128 format)
//
func (r *WASMReader) readInt32LEB128InstructionArgument() (int32, error) {
	offset := r.buf.offset
	v, err := r.buf.readInt32LEB128()
	if err != nil {
		return 0, InvalidInstructionArgumentError{
			Offset:    int(offset),
			ReadError: err,
		}
	}
	return v, nil
}

// readInt64LEB128InstructionArgument reads an int64 instruction argument
// (in LEB128 format)
//
func (r *WASMReader) readInt64LEB128InstructionArgument() (int64, error) {
	offset := r.buf.offset
	v, err := r.buf.readInt64LEB128()
	if err != nil {
		return 0, InvalidInstructionArgumentError{
			Offset:    int(offset),
			ReadError: err,
		}
	}
	return v, nil
}

// readBlockInstructionArgument reads a block instruction argument
//
func (r *WASMReader) readBlockInstructionArgument(allowElse bool) (Block, error) {
	// read the block type.
	blockType, err := r.readBlockType()
	if err != nil {
		// TODO: improve error
		return Block{}, err
	}

	// read the first sequence of instructions.
	// `readInstructions` cannot be used, as it does not handle the `else` instruction / opcode.

	var instructions1 []Instruction

	sawElse := false

	for {
		b, err := r.buf.PeekByte()
		if err != nil {
			// TODO: improve error
			return Block{}, err
		}
		if b == byte(opcodeElse) {
			if !allowElse {
				return Block{}, InvalidBlockSecondInstructionsError{
					Offset: int(r.buf.offset),

				}
			}
			r.buf.offset++
			sawElse = true
			break
		}

		instruction, err := r.readInstruction()
		if err != nil {
			// TODO: improve error
			return Block{}, err
		}

		if _, ok := instruction.(InstructionEnd); ok {
			break
		}

		instructions1 = append(instructions1, instruction)
	}

	var instructions2 []Instruction
	if sawElse {
		// the first set of instructions contained an `else` instruction / opcode,
		// so read the second set of instructions. the validity was already checked above.

		instructions2, err = r.readInstructions()
		if err != nil {
			// TODO: improve error
			return Block{}, err
		}
	}

	return Block{
		BlockType: blockType,
		Instructions1: instructions1,
		Instructions2: instructions2,
	}, nil
}

func (r *WASMReader) readBlockType() (BlockType, error) {
	// it may be the empty block type
	b, err := r.buf.PeekByte()
	if err != nil {
		// TODO: improve error
		return nil, err
	}
	if b == emptyBlockType {
		r.buf.offset++
		return nil, nil
	}

	// it may be a value type
	blockType, err := r.readBlockTypeValueType()
	if err != nil {
		// TODO: improve error
		return nil, err
	}

	if blockType != nil {
		return blockType, nil
	}

	// the block type is not a value type,
	// it must be a type index.

	typeIndexOffset := r.buf.offset
	typeIndex, err := r.buf.readInt64LEB128()
	if err != nil {
		// TODO: improve error
		return nil, err
	}

	// the type index was read as a int64,
	// but must fit a uint32
	if typeIndex < 0 || typeIndex > math.MaxUint32 {
		return nil, InvalidBlockTypeTypeIndexError{
			Offset: int(typeIndexOffset),
			TypeIndex: typeIndex,
		}
	}

	return TypeIndexBlockType{
		TypeIndex: uint32(typeIndex),
	}, nil
}

// readBlockTypeValueType reads a value type block type
// and returns nil if the next byte is not a valid value type
//
func (r *WASMReader) readBlockTypeValueType() (BlockType, error) {
	b, err := r.buf.PeekByte()
	if err != nil {
		return nil, err
	}
	valueType := AsValueType(b)
	if valueType == 0 {
		return nil, nil
	}
	r.buf.offset++
	return valueType, nil
}
