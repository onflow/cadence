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
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWASMReader_readMagicAndVersion(t *testing.T) {

	t.Parallel()

	read := func(data []byte) error {
		b := buf{data: data}
		r := WASMReader{buf: &b}
		return r.readMagicAndVersion()
	}

	t.Run("invalid magic, too short", func(t *testing.T) {

		t.Parallel()

		err := read([]byte{0x0, 0x61})
		require.Error(t, err)
		assert.Equal(t,
			InvalidMagicError{
				Offset:    0,
				ReadError: io.EOF,
			},
			err,
		)
	})

	t.Run("invalid magic, incorrect", func(t *testing.T) {

		t.Parallel()

		err := read([]byte{0x0, 0x61, 0x73, 0xFF})
		require.Error(t, err)
		assert.Equal(t,
			InvalidMagicError{
				Offset:    0,
				ReadError: nil,
			},
			err,
		)
	})

	t.Run("invalid version, too short", func(t *testing.T) {

		t.Parallel()

		err := read([]byte{0x0, 0x61, 0x73, 0x6d, 0x1, 0x0})
		require.Error(t, err)
		assert.Equal(t,
			InvalidVersionError{
				Offset:    4,
				ReadError: io.EOF,
			},
			err,
		)
	})

	t.Run("invalid version, incorrect", func(t *testing.T) {

		t.Parallel()

		err := read([]byte{0x0, 0x61, 0x73, 0x6d, 0x2, 0x0, 0x0, 0x0})
		require.Error(t, err)
		assert.Equal(t,
			InvalidVersionError{
				Offset:    4,
				ReadError: nil,
			},
			err,
		)
	})

	t.Run("valid magic and version", func(t *testing.T) {

		t.Parallel()

		err := read([]byte{0x0, 0x61, 0x73, 0x6d, 0x1, 0x0, 0x0, 0x0})
		require.NoError(t, err)
	})
}

func TestWASMReader_readValType(t *testing.T) {

	t.Parallel()

	read := func(data []byte) (ValueType, error) {
		b := buf{data: data}
		r := WASMReader{buf: &b}
		return r.readValType()
	}

	t.Run("too short", func(t *testing.T) {

		t.Parallel()

		valType, err := read([]byte{})
		require.Error(t, err)
		assert.Equal(t,
			InvalidValTypeError{
				Offset:    0,
				ValType:   valType,
				ReadError: io.EOF,
			},
			err,
		)
		assert.Equal(t, ValueType(0), valType)
	})

	t.Run("invalid", func(t *testing.T) {

		t.Parallel()

		valType, err := read([]byte{0xFF})
		require.Error(t, err)
		assert.Equal(t,
			InvalidValTypeError{
				Offset:    0,
				ValType:   0xFF,
				ReadError: nil,
			},
			err,
		)
		assert.Equal(t, ValueType(0), valType)
	})

	t.Run("i32", func(t *testing.T) {

		t.Parallel()

		valType, err := read([]byte{byte(ValueTypeI32)})
		require.NoError(t, err)
		assert.Equal(t, ValueTypeI32, valType)
	})

	t.Run("i64", func(t *testing.T) {

		t.Parallel()

		valType, err := read([]byte{byte(ValueTypeI64)})
		require.NoError(t, err)
		assert.Equal(t, ValueTypeI64, valType)
	})
}

func TestWASMReader_readTypeSection(t *testing.T) {

	t.Parallel()

	read := func(data []byte) ([]*FunctionType, error) {
		b := buf{data: data}
		r := WASMReader{buf: &b}
		err := r.readTypeSection()
		if err != nil {
			return nil, err
		}
		return r.Module.Types, nil
	}

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 7 (LEB128)
			0x87, 0x80, 0x80, 0x80, 0x0,
			// type count
			0x1,
			// function type
			0x60,
			// parameter count: 2
			0x2,
			// type of parameter 1: i32
			0x7f,
			// type of parameter 2: i32
			0x7f,
			// return value count: 1
			0x1,
			// type of return value 1: i32
			0x7f,
		})
		require.NoError(t, err)
		assert.Equal(t,
			[]*FunctionType{
				{
					Params:  []ValueType{ValueTypeI32, ValueTypeI32},
					Results: []ValueType{ValueTypeI32},
				},
			},
			funcTypes,
		)
	})

	t.Run("invalid size", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			0x87, 0x80, 0x80,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidSectionSizeError{
				Offset:    0,
				ReadError: io.EOF,
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid count", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 0 (LEB128)
			0x80, 0x80, 0x80, 0x80, 0x0,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidTypeSectionTypeCountError{
				Offset:    5,
				ReadError: io.EOF,
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid indicator", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 2 (LEB128)
			0x82, 0x80, 0x80, 0x80, 0x0,
			// type count
			0x1,
			// function type
			0xFF,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidFuncTypeIndicatorError{
				Offset:            6,
				FuncTypeIndicator: 0xff,
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid parameter count", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 4 (LEB128)
			0x82, 0x80, 0x80, 0x80, 0x0,
			// type count
			0x1,
			// function type
			0x60,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidFuncTypeParameterCountError{
				Offset:    7,
				ReadError: io.EOF,
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid parameter type", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 4 (LEB128)
			0x84, 0x80, 0x80, 0x80, 0x0,
			// type count
			0x1,
			// function type
			0x60,
			// parameter count: 1
			0x1,
			// type of parameter 1
			0xff,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidFuncTypeParameterTypeError{
				Index: 0,
				ReadError: InvalidValTypeError{
					Offset:  8,
					ValType: 0xFF,
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid, parameter type missing", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 4 (LEB128)
			0x84, 0x80, 0x80, 0x80, 0x0,
			// type count
			0x1,
			// function type
			0x60,
			// parameter count: 2
			0x2,
			// type of parameter 1: i32
			0x7f,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidFuncTypeParameterTypeError{
				Index: 1,
				ReadError: InvalidValTypeError{
					Offset:    9,
					ValType:   0,
					ReadError: io.EOF,
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid result count", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 4 (LEB128)
			0x83, 0x80, 0x80, 0x80, 0x0,
			// type count
			0x1,
			// function type
			0x60,
			// parameter count
			0x0,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidFuncTypeResultCountError{
				Offset:    8,
				ReadError: io.EOF,
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid result type", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 4 (LEB128)
			0x85, 0x80, 0x80, 0x80, 0x0,
			// type count
			0x1,
			// function type
			0x60,
			// parameter count: 0
			0x0,
			// result count: 1
			0x1,
			// type of result 1
			0xff,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidFuncTypeResultTypeError{
				Index: 0,
				ReadError: InvalidValTypeError{
					Offset:  9,
					ValType: 0xFF,
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid, result type missing", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 4 (LEB128)
			0x85, 0x80, 0x80, 0x80, 0x0,
			// type count
			0x1,
			// function type
			0x60,
			// parameter count: 0
			0x0,
			// result count: 2
			0x2,
			// type of result 1: i32
			0x7f,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidFuncTypeResultTypeError{
				Index: 1,
				ReadError: InvalidValTypeError{
					Offset:    10,
					ValType:   0,
					ReadError: io.EOF,
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})
}

func TestWASMReader_readImportSection(t *testing.T) {

	t.Parallel()

	read := func(data []byte) ([]*Import, error) {
		b := buf{data: data}
		r := WASMReader{buf: &b}
		err := r.readImportSection()
		if err != nil {
			return nil, err
		}
		return r.Module.Imports, nil
	}

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		typeIDs, err := read([]byte{
			// section size: 11 (LEB128)
			0x8b, 0x80, 0x80, 0x80, 0x0,
			// import count: 1
			0x1,
			// module length
			0x3,
			// module = "foo"
			0x66, 0x6f, 0x6f,
			// name length
			0x3,
			// name = "bar"
			0x62, 0x61, 0x72,
			// type indicator: function = 0
			0x0,
			// type ID of function: 0
			0x1,
		})
		require.NoError(t, err)
		assert.Equal(t,
			[]*Import{
				{
					Module: "foo",
					Name:   "bar",
					TypeID: 1,
				},
			},
			typeIDs,
		)
	})

	t.Run("invalid size", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			0x87, 0x80, 0x80,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidSectionSizeError{
				Offset:    0,
				ReadError: io.EOF,
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid count", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 7 (LEB128)
			0x80, 0x80, 0x80, 0x80, 0x0,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidImportSectionImportCountError{
				Offset:    5,
				ReadError: io.EOF,
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid module", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 7 (LEB128)
			0x81, 0x80, 0x80, 0x80, 0x0,
			// import count
			0x1,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidImportError{
				Index: 0,
				ReadError: InvalidNameLengthError{
					Offset:    6,
					ReadError: io.EOF,
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid name", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 7 (LEB128)
			0x81, 0x80, 0x80, 0x80, 0x0,
			// import count
			0x1,
			// module length
			0x3,
			// module = "foo"
			0x66, 0x6f, 0x6f,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidImportError{
				Index: 0,
				ReadError: InvalidNameLengthError{
					Offset:    10,
					ReadError: io.EOF,
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("missing type indicator", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 7 (LEB128)
			0x81, 0x80, 0x80, 0x80, 0x0,
			// import count
			0x1,
			// module length
			0x3,
			// module = "foo"
			0x66, 0x6f, 0x6f,
			// name length
			0x3,
			// name = "bar"
			0x62, 0x61, 0x72,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidImportError{
				Index: 0,
				ReadError: InvalidImportTypeIndicatorError{
					Offset:    14,
					ReadError: io.EOF,
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid type indicator", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 7 (LEB128)
			0x81, 0x80, 0x80, 0x80, 0x0,
			// import count
			0x1,
			// module length
			0x3,
			// module = "foo"
			0x66, 0x6f, 0x6f,
			// name length
			0x3,
			// name = "bar"
			0x62, 0x61, 0x72,
			// type indicator
			0x1,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidImportError{
				Index: 0,
				ReadError: InvalidImportTypeIndicatorError{
					Offset:        14,
					TypeIndicator: 0x1,
					ReadError:     nil,
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid function type index", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 7 (LEB128)
			0x81, 0x80, 0x80, 0x80, 0x0,
			// import count
			0x1,
			// module length
			0x3,
			// module = "foo"
			0x66, 0x6f, 0x6f,
			// name length
			0x3,
			// name = "bar"
			0x62, 0x61, 0x72,
			// type indicator: function = 0
			0x0,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidImportError{
				Index: 0,
				ReadError: InvalidImportSectionFunctionTypeIDError{
					Offset:    15,
					ReadError: io.EOF,
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})
}

func TestWASMReader_readFunctionSection(t *testing.T) {

	t.Parallel()

	read := func(data []byte) ([]uint32, error) {
		b := buf{data: data}
		r := WASMReader{buf: &b}
		err := r.readFunctionSection()
		if err != nil {
			return nil, err
		}
		return r.Module.functionTypeIDs, nil
	}

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		typeIDs, err := read([]byte{
			// section size: 2 (LEB128)
			0x82, 0x80, 0x80, 0x80, 0x0,
			// function count: 1
			0x1,
			// type ID of function: 0x23
			0x23,
		})
		require.NoError(t, err)
		assert.Equal(t,
			[]uint32{0x23},
			typeIDs,
		)
	})

	t.Run("invalid size", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			0x87, 0x80, 0x80,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidSectionSizeError{
				Offset:    0,
				ReadError: io.EOF,
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid count", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 7 (LEB128)
			0x80, 0x80, 0x80, 0x80, 0x0,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidFunctionSectionFunctionCountError{
				Offset:    5,
				ReadError: io.EOF,
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid function type ID", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 7 (LEB128)
			0x81, 0x80, 0x80, 0x80, 0x0,
			// function count
			0x1,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidFunctionSectionFunctionTypeIDError{
				Offset:    6,
				Index:     0,
				ReadError: io.EOF,
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})
}

func TestWASMReader_readCodeSection(t *testing.T) {

	t.Parallel()

	read := func(data []byte) ([]*Code, error) {
		b := buf{data: data}
		r := WASMReader{buf: &b}
		err := r.readCodeSection()
		if err != nil {
			return nil, err
		}
		return r.Module.functionBodies, nil
	}

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		codes, err := read([]byte{
			// section size: 15 (LEB128)
			0x8f, 0x80, 0x80, 0x80, 0x0,
			// function count: 1
			0x1,
			// code size: 9 (LEB128)
			0x89, 0x80, 0x80, 0x80, 0x0,
			// number of locals: 1
			0x1,
			// number of locals with this type: 1
			0x1,
			// local type: i32
			0x7f,
			// opcode: local.get, 0
			0x20, 0x0,
			// opcode: local.get 1
			0x20, 0x1,
			// opcode: i32.add
			0x6a,
			// opcode: end
			0xb,
		})
		require.NoError(t, err)
		assert.Equal(t,
			[]*Code{
				{
					Locals: []ValueType{
						ValueTypeI32,
					},
					Instructions: []Instruction{
						InstructionLocalGet{0},
						InstructionLocalGet{1},
						InstructionI32Add{},
					},
				},
			},
			codes,
		)
	})

	t.Run("invalid size", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			0x87, 0x80, 0x80,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidSectionSizeError{
				Offset:    0,
				ReadError: io.EOF,
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid count", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 7 (LEB128)
			0x80, 0x80, 0x80, 0x80, 0x0,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidCodeSectionFunctionCountError{
				Offset:    5,
				ReadError: io.EOF,
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid code size", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 15 (LEB128)
			0x8f, 0x80, 0x80, 0x80, 0x0,
			// function count: 1
			0x1,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidFunctionCodeError{
				Index: 0,
				ReadError: InvalidCodeSizeError{
					Offset:    6,
					ReadError: io.EOF,
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid locals count", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 15 (LEB128)
			0x8f, 0x80, 0x80, 0x80, 0x0,
			// function count: 1
			0x1,
			// code size: 9 (LEB128)
			0x89, 0x80, 0x80, 0x80, 0x0,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidFunctionCodeError{
				Index: 0,
				ReadError: InvalidCodeSectionLocalsCountError{
					Offset:    11,
					ReadError: io.EOF,
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid compressed locals count", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 15 (LEB128)
			0x8f, 0x80, 0x80, 0x80, 0x0,
			// function count: 1
			0x1,
			// code size: 9 (LEB128)
			0x89, 0x80, 0x80, 0x80, 0x0,
			// number of locals: 1
			0x1,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidFunctionCodeError{
				Index: 0,
				ReadError: InvalidCodeSectionCompressedLocalsCountError{
					Offset:    12,
					ReadError: io.EOF,
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid local type", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 15 (LEB128)
			0x8f, 0x80, 0x80, 0x80, 0x0,
			// function count: 1
			0x1,
			// code size: 9 (LEB128)
			0x89, 0x80, 0x80, 0x80, 0x0,
			// number of locals: 1
			0x1,
			// number of locals with this type: 1
			0x1,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidFunctionCodeError{
				Index: 0,
				ReadError: InvalidCodeSectionLocalTypeError{
					Offset: 13,
					ReadError: InvalidValTypeError{
						Offset:    13,
						ValType:   0,
						ReadError: io.EOF,
					},
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid instruction", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 15 (LEB128)
			0x8f, 0x80, 0x80, 0x80, 0x0,
			// function count: 1
			0x1,
			// code size: 9 (LEB128)
			0x89, 0x80, 0x80, 0x80, 0x0,
			// number of locals: 1
			0x1,
			// number of locals with this type: 1
			0x1,
			// local type: i32
			0x7f,
			// invalid opcode
			0xff,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidFunctionCodeError{
				Index: 0,
				ReadError: InvalidOpcodeError{
					Offset:    14,
					Opcode:    0xff,
					ReadError: nil,
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("missing end", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 15 (LEB128)
			0x8f, 0x80, 0x80, 0x80, 0x0,
			// function count: 1
			0x1,
			// code size: 9 (LEB128)
			0x89, 0x80, 0x80, 0x80, 0x0,
			// number of locals: 1
			0x1,
			// number of locals with this type: 1
			0x1,
			// local type: i32
			0x7f,
			// opcode: local.get, 0
			0x20, 0x0,
			// opcode: local.get 1
			0x20, 0x1,
			// opcode: i32.add
			0x6a,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidFunctionCodeError{
				Index: 0,
				ReadError: MissingEndInstructionError{
					Offset: 19,
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})
}

func TestWASMReader_readName(t *testing.T) {

	t.Parallel()

	read := func(data []byte) (string, error) {
		b := buf{data: data}
		r := WASMReader{buf: &b}
		return r.readName()
	}

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		name, err := read([]byte{
			// length
			0x5,
			// "hello"
			0x68, 0x65, 0x6c, 0x6c, 0x6f,
		})
		require.NoError(t, err)

		require.Equal(t, "hello", name)
	})

	t.Run("invalid length", func(t *testing.T) {

		t.Parallel()

		name, err := read(nil)
		require.Error(t, err)

		assert.Equal(t,
			InvalidNameLengthError{
				Offset:    0,
				ReadError: io.EOF,
			},
			err,
		)
		assert.Empty(t, name)
	})

	t.Run("invalid name", func(t *testing.T) {

		t.Parallel()

		name, err := read([]byte{
			// length
			0x5,
			// "he"
			0x68, 0x65,
		})
		require.Error(t, err)

		assert.Equal(t,
			IncompleteNameError{
				Offset:   1,
				Expected: 5,
				Actual:   2,
			},
			err,
		)
		assert.Empty(t, name)
	})

	t.Run("invalid non UTF-8", func(t *testing.T) {

		t.Parallel()

		name, err := read([]byte{
			// length
			0x3,
			// name
			0xff, 0xfe, 0xfd,
		})
		require.Error(t, err)

		assert.Equal(t,
			InvalidNonUTF8NameError{
				Name:   "\xff\xfe\xfd",
				Offset: 1,
			},
			err,
		)
		assert.Empty(t, name)
	})
}
