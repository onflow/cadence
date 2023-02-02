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
	"fmt"
)

// InvalidMagicError is returned when the WASM binary
// does not start with the magic byte sequence
type InvalidMagicError struct {
	ReadError error
	Offset    int
}

func (e InvalidMagicError) Error() string {
	return fmt.Sprintf(
		"invalid magic at offset %d",
		e.Offset,
	)
}

func (e InvalidMagicError) Unwrap() error {
	return e.ReadError
}

// InvalidMagicError is returned when the WASM binary
// does not have the expected version
type InvalidVersionError struct {
	ReadError error
	Offset    int
}

func (e InvalidVersionError) Error() string {
	return fmt.Sprintf(
		"invalid version at offset %d",
		e.Offset,
	)
}

func (e InvalidVersionError) Unwrap() error {
	return e.ReadError
}

// InvalidSectionIDError is returned when the WASM binary specifies
// an invalid section ID
type InvalidSectionIDError struct {
	ReadError error
	Offset    int
	SectionID sectionID
}

func (e InvalidSectionIDError) Error() string {
	return fmt.Sprintf(
		"invalid section ID %d at offset %d",
		e.SectionID,
		e.Offset,
	)
}

func (e InvalidSectionIDError) Unwrap() error {
	return e.ReadError
}

// InvalidDuplicateSectionError is returned when the WASM binary specifies
// a duplicate section
type InvalidDuplicateSectionError struct {
	Offset    int
	SectionID sectionID
}

func (e InvalidDuplicateSectionError) Error() string {
	return fmt.Sprintf(
		"invalid duplicate section with ID %d at offset %d",
		e.SectionID,
		e.Offset,
	)
}

// InvalidSectionOrderError is returned when the WASM binary specifies
// a non-custom section out-of-order
type InvalidSectionOrderError struct {
	Offset    int
	SectionID sectionID
}

func (e InvalidSectionOrderError) Error() string {
	return fmt.Sprintf(
		"out-of-order section with ID %d at offset %d",
		e.SectionID,
		e.Offset,
	)
}

// InvalidSectionSizeError is returned when the WASM binary specifies
// an invalid section size
type InvalidSectionSizeError struct {
	ReadError error
	Offset    int
}

func (e InvalidSectionSizeError) Error() string {
	return fmt.Sprintf(
		"invalid section size at offset %d: %s",
		e.Offset,
		e.ReadError,
	)
}

func (e InvalidSectionSizeError) Unwrap() error {
	return e.ReadError
}

// InvalidValTypeError is returned when the WASM binary specifies
// an invalid value type
type InvalidValTypeError struct {
	ReadError error
	Offset    int
	ValType   ValueType
}

func (e InvalidValTypeError) Error() string {
	return fmt.Sprintf(
		"invalid value type %d at offset %d",
		e.ValType,
		e.Offset,
	)
}

func (e InvalidValTypeError) Unwrap() error {
	return e.ReadError
}

// InvalidFuncTypeIndicatorError is returned when the WASM binary specifies
// an invalid function type indicator
type InvalidFuncTypeIndicatorError struct {
	ReadError         error
	Offset            int
	FuncTypeIndicator byte
}

func (e InvalidFuncTypeIndicatorError) Error() string {
	return fmt.Sprintf(
		"invalid function type indicator at offset %d: got %x, expected %x",
		e.Offset,
		e.FuncTypeIndicator,
		functionTypeIndicator,
	)
}

func (e InvalidFuncTypeIndicatorError) Unwrap() error {
	return e.ReadError
}

// InvalidFuncTypeParameterCountError is returned when the WASM binary specifies
// an invalid func type parameter count
type InvalidFuncTypeParameterCountError struct {
	ReadError error
	Offset    int
}

func (e InvalidFuncTypeParameterCountError) Error() string {
	return fmt.Sprintf(
		"invalid function type parameter count at offset %d",
		e.Offset,
	)
}

func (e InvalidFuncTypeParameterCountError) Unwrap() error {
	return e.ReadError
}

// InvalidFuncTypeParameterTypeError is returned when the WASM binary specifies
// an invalid function type parameter type
type InvalidFuncTypeParameterTypeError struct {
	ReadError error
	Index     int
}

func (e InvalidFuncTypeParameterTypeError) Error() string {
	return fmt.Sprintf(
		"invalid function type parameter type at index %d",
		e.Index,
	)
}

func (e InvalidFuncTypeParameterTypeError) Unwrap() error {
	return e.ReadError
}

// InvalidFuncTypeResultCountError is returned when the WASM binary specifies
// an invalid func type result count
type InvalidFuncTypeResultCountError struct {
	ReadError error
	Offset    int
}

func (e InvalidFuncTypeResultCountError) Error() string {
	return fmt.Sprintf(
		"invalid function type result count at offset %d",
		e.Offset,
	)
}

func (e InvalidFuncTypeResultCountError) Unwrap() error {
	return e.ReadError
}

// InvalidFuncTypeResultTypeError is returned when the WASM binary specifies
// an invalid function type result type
type InvalidFuncTypeResultTypeError struct {
	ReadError error
	Index     int
}

func (e InvalidFuncTypeResultTypeError) Error() string {
	return fmt.Sprintf(
		"invalid function type result type at index %d",
		e.Index,
	)
}

func (e InvalidFuncTypeResultTypeError) Unwrap() error {
	return e.ReadError
}

// InvalidTypeSectionTypeCountError is returned when the WASM binary specifies
// an invalid count in the type section
type InvalidTypeSectionTypeCountError struct {
	ReadError error
	Offset    int
}

func (e InvalidTypeSectionTypeCountError) Error() string {
	return fmt.Sprintf(
		"invalid type count in type section at offset %d",
		e.Offset,
	)
}

func (e InvalidTypeSectionTypeCountError) Unwrap() error {
	return e.ReadError
}

// InvalidImportSectionImportCountError is returned when the WASM binary specifies
// an invalid count in the import section
type InvalidImportSectionImportCountError struct {
	ReadError error
	Offset    int
}

func (e InvalidImportSectionImportCountError) Error() string {
	return fmt.Sprintf(
		"invalid import count in import section at offset %d",
		e.Offset,
	)
}

func (e InvalidImportSectionImportCountError) Unwrap() error {
	return e.ReadError
}

// InvalidImportError is returned when the WASM binary specifies
// invalid import in the import section
type InvalidImportError struct {
	ReadError error
	Index     int
}

func (e InvalidImportError) Error() string {
	return fmt.Sprintf(
		"invalid import at index %d",
		e.Index,
	)
}

func (e InvalidImportError) Unwrap() error {
	return e.ReadError
}

// InvalidImportIndicatorError is returned when the WASM binary specifies
// an invalid type indicator in the import section
type InvalidImportIndicatorError struct {
	ReadError       error
	Offset          int
	ImportIndicator importIndicator
}

func (e InvalidImportIndicatorError) Error() string {
	return fmt.Sprintf(
		"invalid import indicator %d at offset %d",
		e.ImportIndicator,
		e.Offset,
	)
}

func (e InvalidImportIndicatorError) Unwrap() error {
	return e.ReadError
}

// InvalidImportSectionTypeIndexError is returned when the WASM binary specifies
// an invalid type index in the import section
type InvalidImportSectionTypeIndexError struct {
	ReadError error
	Offset    int
}

func (e InvalidImportSectionTypeIndexError) Error() string {
	return fmt.Sprintf(
		"invalid type index in import section at offset %d",
		e.Offset,
	)
}

func (e InvalidImportSectionTypeIndexError) Unwrap() error {
	return e.ReadError
}

// InvalidFunctionSectionFunctionCountError is returned when the WASM binary specifies
// an invalid count in the function section
type InvalidFunctionSectionFunctionCountError struct {
	ReadError error
	Offset    int
}

func (e InvalidFunctionSectionFunctionCountError) Error() string {
	return fmt.Sprintf(
		"invalid function count in function section at offset %d",
		e.Offset,
	)
}

func (e InvalidFunctionSectionFunctionCountError) Unwrap() error {
	return e.ReadError
}

// InvalidFunctionSectionTypeIndexError is returned when the WASM binary specifies
// an invalid type index in the function section
type InvalidFunctionSectionTypeIndexError struct {
	ReadError error
	Offset    int
	Index     int
}

func (e InvalidFunctionSectionTypeIndexError) Error() string {
	return fmt.Sprintf(
		"invalid type index in function section at index %d at offset %d",
		e.Index,
		e.Offset,
	)
}

func (e InvalidFunctionSectionTypeIndexError) Unwrap() error {
	return e.ReadError
}

// FunctionCountMismatchError is returned when the WASM binary specifies
// information for a different number of functions than previously specified
type FunctionCountMismatchError struct {
	Offset int
}

func (e FunctionCountMismatchError) Error() string {
	return fmt.Sprintf(
		"function count mismatch at offset %d",
		e.Offset,
	)
}

// InvalidExportSectionExportCountError is returned when the WASM binary specifies
// an invalid count in the export section
type InvalidExportSectionExportCountError struct {
	ReadError error
	Offset    int
}

func (e InvalidExportSectionExportCountError) Error() string {
	return fmt.Sprintf(
		"invalid export count in export section at offset %d",
		e.Offset,
	)
}

func (e InvalidExportSectionExportCountError) Unwrap() error {
	return e.ReadError
}

// InvalidExportError is returned when the WASM binary specifies
// invalid export in the export section
type InvalidExportError struct {
	ReadError error
	Index     int
}

func (e InvalidExportError) Error() string {
	return fmt.Sprintf(
		"invalid export at index %d",
		e.Index,
	)
}

func (e InvalidExportError) Unwrap() error {
	return e.ReadError
}

// InvalidExportIndicatorError is returned when the WASM binary specifies
// an invalid type indicator in the export section
type InvalidExportIndicatorError struct {
	ReadError       error
	Offset          int
	ExportIndicator exportIndicator
}

func (e InvalidExportIndicatorError) Error() string {
	return fmt.Sprintf(
		"invalid export indicator %d at offset %d",
		e.ExportIndicator,
		e.Offset,
	)
}

func (e InvalidExportIndicatorError) Unwrap() error {
	return e.ReadError
}

// InvalidExportSectionIndexError is returned when the WASM binary specifies
// an invalid index in the export section
type InvalidExportSectionIndexError struct {
	ReadError error
	Offset    int
}

func (e InvalidExportSectionIndexError) Error() string {
	return fmt.Sprintf(
		"invalid index in export section at offset %d",
		e.Offset,
	)
}

func (e InvalidExportSectionIndexError) Unwrap() error {
	return e.ReadError
}

// InvalidCodeSectionFunctionCountError is returned when the WASM binary specifies
// an invalid function count in the code section
type InvalidCodeSectionFunctionCountError struct {
	ReadError error
	Offset    int
}

func (e InvalidCodeSectionFunctionCountError) Error() string {
	return fmt.Sprintf(
		"invalid function count in code section at offset %d",
		e.Offset,
	)
}

func (e InvalidCodeSectionFunctionCountError) Unwrap() error {
	return e.ReadError
}

// InvalidFunctionCodeError is returned when the WASM binary specifies
// invalid code for a function in the code section
type InvalidFunctionCodeError struct {
	ReadError error
	Index     int
}

func (e InvalidFunctionCodeError) Error() string {
	return fmt.Sprintf(
		"invalid code for function at index %d",
		e.Index,
	)
}

func (e InvalidFunctionCodeError) Unwrap() error {
	return e.ReadError
}

// InvalidCodeSizeError is returned when the WASM binary specifies
// an invalid code size in the code section
type InvalidCodeSizeError struct {
	ReadError error
	Offset    int
}

func (e InvalidCodeSizeError) Error() string {
	return fmt.Sprintf(
		"invalid code size in code section at offset %d",
		e.Offset,
	)
}

// InvalidCodeSectionLocalsCountError is returned when the WASM binary specifies
// an invalid locals count in the code section
type InvalidCodeSectionLocalsCountError struct {
	ReadError error
	Offset    int
}

func (e InvalidCodeSectionLocalsCountError) Error() string {
	return fmt.Sprintf(
		"invalid locals count in code section at offset %d",
		e.Offset,
	)
}

func (e InvalidCodeSectionLocalsCountError) Unwrap() error {
	return e.ReadError
}

// InvalidCodeSectionCompressedLocalsCountError is returned when the WASM binary specifies
// an invalid local type in the code section
type InvalidCodeSectionCompressedLocalsCountError struct {
	ReadError error
	Offset    int
}

func (e InvalidCodeSectionCompressedLocalsCountError) Error() string {
	return fmt.Sprintf(
		"invalid compressed local type count in code section at offset %d",
		e.Offset,
	)
}

func (e InvalidCodeSectionCompressedLocalsCountError) Unwrap() error {
	return e.ReadError
}

// InvalidCodeSectionLocalTypeError is returned when the WASM binary specifies
// an invalid local type in the code section
type InvalidCodeSectionLocalTypeError struct {
	ReadError error
	Offset    int
}

func (e InvalidCodeSectionLocalTypeError) Error() string {
	return fmt.Sprintf(
		"invalid local type in code section at offset %d",
		e.Offset,
	)
}

func (e InvalidCodeSectionLocalTypeError) Unwrap() error {
	return e.ReadError
}

// CodeSectionLocalsCountMismatchError is returned when
// the sum of the compressed locals locals count in the code section does not match
// the number of locals in the code section of the WASM binary
type CodeSectionLocalsCountMismatchError struct {
	Offset   int
	Expected uint32
	Actual   uint32
}

func (e CodeSectionLocalsCountMismatchError) Error() string {
	return fmt.Sprintf(
		"local count mismatch in code section at offset %d: expected %d, got %d",
		e.Offset,
		e.Expected,
		e.Actual,
	)
}

// InvalidOpcodeError is returned when the WASM binary specifies
// an invalid opcode in the code section
type InvalidOpcodeError struct {
	ReadError error
	Offset    int
	Opcode    opcode
}

func (e InvalidOpcodeError) Error() string {
	return fmt.Sprintf(
		"invalid opcode in code section at offset %d: %x",
		e.Offset,
		e.Opcode,
	)
}

func (e InvalidOpcodeError) Unwrap() error {
	return e.ReadError
}

// InvalidInstructionArgumentError is returned when the WASM binary specifies
// an invalid argument for an instruction in the code section
type InvalidInstructionArgumentError struct {
	ReadError error
	Offset    int
}

func (e InvalidInstructionArgumentError) Error() string {
	return fmt.Sprintf(
		"invalid argument in code section at offset %d",
		e.Offset,
	)
}

func (e InvalidInstructionArgumentError) Unwrap() error {
	return e.ReadError
}

// MissingEndInstructionError is returned when the WASM binary
// misses an end instruction for a function in the code section
type MissingEndInstructionError struct {
	Offset int
}

func (e MissingEndInstructionError) Error() string {
	return fmt.Sprintf(
		"missing end instruction in code section at offset %d",
		e.Offset,
	)
}

// InvalidNonUTF8NameError is returned when the WASM binary specifies
// or the writer is given a name which is not properly UTF-8 encoded
type InvalidNonUTF8NameError struct {
	Name   string
	Offset int
}

func (e InvalidNonUTF8NameError) Error() string {
	return fmt.Sprintf(
		"invalid non UTF-8 string at offset %d: %s",
		e.Offset,
		e.Name,
	)
}

// InvalidNameLengthError is returned the WASM binary specifies
// an invalid name length
type InvalidNameLengthError struct {
	ReadError error
	Offset    int
}

func (e InvalidNameLengthError) Error() string {
	return fmt.Sprintf(
		"invalid name length at offset %d",
		e.Offset,
	)
}

func (e InvalidNameLengthError) Unwrap() error {
	return e.ReadError
}

// InvalidNameError is returned the WASM binary specifies
// an invalid name
type InvalidNameError struct {
	ReadError error
	Offset    int
}

func (e InvalidNameError) Error() string {
	return fmt.Sprintf(
		"invalid name at offset %d",
		e.Offset,
	)
}

func (e InvalidNameError) Unwrap() error {
	return e.ReadError
}

// IncompleteNameError is returned the WASM binary specifies
// an incomplete name
type IncompleteNameError struct {
	Offset   int
	Expected uint32
	Actual   uint32
}

func (e IncompleteNameError) Error() string {
	return fmt.Sprintf(
		"incomplete name at offset %d. expected %d bytes, got %d",
		e.Offset,
		e.Expected,
		e.Actual,
	)
}

// InvalidBlockSecondInstructionsError is returned when the WASM binary specifies
// or the writer is given a second set of instructions in a block that
// is not allowed to have it (only the 'if' instruction may have it)
type InvalidBlockSecondInstructionsError struct {
	Offset int
}

func (e InvalidBlockSecondInstructionsError) Error() string {
	return fmt.Sprintf(
		"invalid second set of instructions at offset %d",
		e.Offset,
	)
}

// InvalidInstructionVectorArgumentCountError is returned when the WASM binary specifies
// an invalid count for a vector argument of an instruction
type InvalidInstructionVectorArgumentCountError struct {
	ReadError error
	Offset    int
}

func (e InvalidInstructionVectorArgumentCountError) Error() string {
	return fmt.Sprintf(
		"invalid vector count for argument of instruction at offset %d",
		e.Offset,
	)
}

func (e InvalidInstructionVectorArgumentCountError) Unwrap() error {
	return e.ReadError
}

// InvalidBlockTypeTypeIndexError is returned when the WASM binary specifies
// an invalid type index as a block type
type InvalidBlockTypeTypeIndexError struct {
	TypeIndex int64
	Offset    int
}

func (e InvalidBlockTypeTypeIndexError) Error() string {
	return fmt.Sprintf(
		"invalid type index in block type at offset %d: %d",
		e.Offset,
		e.TypeIndex,
	)
}

// InvalidDataSectionSegmentCountError is returned when the WASM binary specifies
// an invalid count in the data section
type InvalidDataSectionSegmentCountError struct {
	ReadError error
	Offset    int
}

func (e InvalidDataSectionSegmentCountError) Error() string {
	return fmt.Sprintf(
		"invalid segment count in data section at offset %d",
		e.Offset,
	)
}

func (e InvalidDataSectionSegmentCountError) Unwrap() error {
	return e.ReadError
}

// InvalidDataSegmentError is returned when the WASM binary specifies
// invalid segment in the data section
type InvalidDataSegmentError struct {
	ReadError error
	Index     int
}

func (e InvalidDataSegmentError) Error() string {
	return fmt.Sprintf(
		"invalid data segment at index %d",
		e.Index,
	)
}

func (e InvalidDataSegmentError) Unwrap() error {
	return e.ReadError
}

// InvalidDataSectionMemoryIndexError is returned when the WASM binary specifies
// an invalid memory index in the data section
type InvalidDataSectionMemoryIndexError struct {
	ReadError error
	Offset    int
}

func (e InvalidDataSectionMemoryIndexError) Error() string {
	return fmt.Sprintf(
		"invalid memory index in data section at offset %d",
		e.Offset,
	)
}

func (e InvalidDataSectionMemoryIndexError) Unwrap() error {
	return e.ReadError
}

// InvalidDataSectionInitByteCountError is returned when the WASM binary specifies
// an invalid init byte count in the data section
type InvalidDataSectionInitByteCountError struct {
	ReadError error
	Offset    int
}

func (e InvalidDataSectionInitByteCountError) Error() string {
	return fmt.Sprintf(
		"invalid init byte count in data section at offset %d",
		e.Offset,
	)
}

func (e InvalidDataSectionInitByteCountError) Unwrap() error {
	return e.ReadError
}

// InvalidMemorySectionMemoryCountError is returned when the WASM binary specifies
// an invalid count in the memory section
type InvalidMemorySectionMemoryCountError struct {
	ReadError error
	Offset    int
}

func (e InvalidMemorySectionMemoryCountError) Error() string {
	return fmt.Sprintf(
		"invalid memories count in memory section at offset %d",
		e.Offset,
	)
}

func (e InvalidMemorySectionMemoryCountError) Unwrap() error {
	return e.ReadError
}

// InvalidMemoryError is returned when the WASM binary specifies
// invalid memory in the memory section
type InvalidMemoryError struct {
	ReadError error
	Index     int
}

func (e InvalidMemoryError) Error() string {
	return fmt.Sprintf(
		"invalid memory at index %d",
		e.Index,
	)
}

func (e InvalidMemoryError) Unwrap() error {
	return e.ReadError
}

// InvalidLimitIndicatorError is returned when the WASM binary specifies
// an invalid limit indicator
type InvalidLimitIndicatorError struct {
	ReadError      error
	Offset         int
	LimitIndicator byte
}

func (e InvalidLimitIndicatorError) Error() string {
	return fmt.Sprintf(
		"invalid limit indicator at offset %d: %x",
		e.Offset,
		e.LimitIndicator,
	)
}

func (e InvalidLimitIndicatorError) Unwrap() error {
	return e.ReadError
}

// InvalidLimitMinError is returned when the WASM binary specifies
// an invalid limit minimum
type InvalidLimitMinError struct {
	ReadError error
	Offset    int
}

func (e InvalidLimitMinError) Error() string {
	return fmt.Sprintf(
		"invalid limit minimum at offset %d",
		e.Offset,
	)
}

func (e InvalidLimitMinError) Unwrap() error {
	return e.ReadError
}

// InvalidLimitMaxError is returned when the WASM binary specifies
// an invalid limit maximum
type InvalidLimitMaxError struct {
	ReadError error
	Offset    int
}

func (e InvalidLimitMaxError) Error() string {
	return fmt.Sprintf(
		"invalid limit maximum at offset %d",
		e.Offset,
	)
}

func (e InvalidLimitMaxError) Unwrap() error {
	return e.ReadError
}

// InvalidStartSectionFunctionIndexError is returned when the WASM binary specifies
// an invalid function index in the start section
type InvalidStartSectionFunctionIndexError struct {
	ReadError error
	Offset    int
}

func (e InvalidStartSectionFunctionIndexError) Error() string {
	return fmt.Sprintf(
		"invalid function index in start section at offset %d",
		e.Offset,
	)
}

func (e InvalidStartSectionFunctionIndexError) Unwrap() error {
	return e.ReadError
}
