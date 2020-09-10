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
	"fmt"
)

// InvalidMagicError is returned when the WASM binary
// does not start with the magic byte sequence
//
type InvalidMagicError struct {
	Offset    int
	ReadError error
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
//
type InvalidVersionError struct {
	Offset    int
	ReadError error
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
//
type InvalidSectionIDError struct {
	Offset    int
	SectionID sectionID
	ReadError error
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
//
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

// InvalidSectionSizeError is returned when the WASM binary specifies
// an invalid section size
//
type InvalidSectionSizeError struct {
	Offset    int
	ReadError error
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
//
type InvalidValTypeError struct {
	Offset    int
	ValType   ValueType
	ReadError error
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
//
type InvalidFuncTypeIndicatorError struct {
	Offset            int
	FuncTypeIndicator byte
	ReadError         error
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
//
type InvalidFuncTypeParameterCountError struct {
	Offset    int
	ReadError error
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
//
type InvalidFuncTypeParameterTypeError struct {
	Index     int
	ReadError error
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
//
type InvalidFuncTypeResultCountError struct {
	Offset    int
	ReadError error
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
//
type InvalidFuncTypeResultTypeError struct {
	Index     int
	ReadError error
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
//
type InvalidTypeSectionTypeCountError struct {
	Offset    int
	ReadError error
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

// InvalidFunctionSectionFunctionCountError is returned when the WASM binary specifies
// an invalid count in the function section
//
type InvalidFunctionSectionFunctionCountError struct {
	Offset    int
	ReadError error
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

// InvalidFunctionSectionFunctionTypeIDError is returned when the WASM binary specifies
// an invalid function type ID in the function section
//
type InvalidFunctionSectionFunctionTypeIDError struct {
	Offset    int
	Index     int
	ReadError error
}

func (e InvalidFunctionSectionFunctionTypeIDError) Error() string {
	return fmt.Sprintf(
		"invalid function type ID at index %d at offset %d",
		e.Index,
		e.Offset,
	)
}

func (e InvalidFunctionSectionFunctionTypeIDError) Unwrap() error {
	return e.ReadError
}

// InvalidCodeSectionFunctionCountError is returned when the WASM binary specifies
// an invalid function count in the code section
//
type InvalidCodeSectionFunctionCountError struct {
	Offset    int
	ReadError error
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
//
type InvalidFunctionCodeError struct {
	Index     int
	ReadError error
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
//
type InvalidCodeSizeError struct {
	Offset    int
	ReadError error
}

func (e InvalidCodeSizeError) Error() string {
	return fmt.Sprintf(
		"invalid code size in code section at offset %d",
		e.Offset,
	)
}

// InvalidCodeSectionLocalsCountError is returned when the WASM binary specifies
// an invalid locals count in the code section
//
type InvalidCodeSectionLocalsCountError struct {
	Offset    int
	ReadError error
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
//
type InvalidCodeSectionCompressedLocalsCountError struct {
	Offset    int
	ReadError error
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
//
type InvalidCodeSectionLocalTypeError struct {
	Offset    int
	ReadError error
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
//
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
//
type InvalidOpcodeError struct {
	Offset    int
	Opcode    opcode
	ReadError error
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
//
type InvalidInstructionArgumentError struct {
	Offset    int
	Opcode    opcode
	ReadError error
}

func (e InvalidInstructionArgumentError) Error() string {
	return fmt.Sprintf(
		"invalid argument for instruction with opcode %x in code section at offset %d",
		e.Opcode,
		e.Offset,
	)
}

func (e InvalidInstructionArgumentError) Unwrap() error {
	return e.ReadError
}

// MissingEndInstructionError is returned when the WASM binary
// misses an end instruction for a function in the code section
//
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
//
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
//
type InvalidNameLengthError struct {
	Offset    int
	ReadError error
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
//
type InvalidNameError struct {
	Offset    int
	ReadError error
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
//
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
