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

import (
	"io"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWASMReader_readMagicAndVersion(t *testing.T) {

	t.Parallel()

	read := func(data []byte) error {
		b := Buffer{data: data}
		r := NewWASMReader(&b)
		err := r.readMagicAndVersion()
		if err != nil {
			return err
		}
		require.Equal(t, offset(len(b.data)), b.offset)
		return nil
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
		b := Buffer{data: data}
		r := NewWASMReader(&b)
		valueType, err := r.readValType()
		if err != nil {
			return 0, err
		}
		require.Equal(t, offset(len(b.data)), b.offset)
		return valueType, nil
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
		b := Buffer{data: data}
		r := NewWASMReader(&b)
		err := r.readTypeSection()
		if err != nil {
			return nil, err
		}
		require.Equal(t, offset(len(b.data)), b.offset)
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
			// section size: 2 (LEB128)
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
			// section size: 3 (LEB128)
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
			// section size: 5 (LEB128)
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
			// section size: 5 (LEB128)
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
		b := Buffer{data: data}
		r := NewWASMReader(&b)
		err := r.readImportSection()
		if err != nil {
			return nil, err
		}
		require.Equal(t, offset(len(b.data)), b.offset)
		return r.Module.Imports, nil
	}

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		typeIndices, err := read([]byte{
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
			// indicator: function = 0
			0x0,
			// type index of function: 0
			0x1,
		})
		require.NoError(t, err)
		assert.Equal(t,
			[]*Import{
				{
					Module:    "foo",
					Name:      "bar",
					TypeIndex: 1,
				},
			},
			typeIndices,
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
			// section size: 1 (LEB128)
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
			// section size: 5 (LEB128)
			0x85, 0x80, 0x80, 0x80, 0x0,
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

	t.Run("missing indicator", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 9 (LEB128)
			0x89, 0x80, 0x80, 0x80, 0x0,
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
				ReadError: InvalidImportIndicatorError{
					Offset:    14,
					ReadError: io.EOF,
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid indicator", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 10 (LEB128)
			0x8a, 0x80, 0x80, 0x80, 0x0,
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
			// indicator
			0x1,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidImportError{
				Index: 0,
				ReadError: InvalidImportIndicatorError{
					Offset:          14,
					ImportIndicator: 0x1,
					ReadError:       nil,
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid type index", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 10 (LEB128)
			0x8a, 0x80, 0x80, 0x80, 0x0,
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
			// indicator: function = 0
			0x0,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidImportError{
				Index: 0,
				ReadError: InvalidImportSectionTypeIndexError{
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

	read := func(data []byte) ([]*Function, error) {
		b := Buffer{data: data}
		r := NewWASMReader(&b)
		err := r.readFunctionSection()
		if err != nil {
			return nil, err
		}
		require.Equal(t, offset(len(b.data)), b.offset)
		return r.Module.Functions, nil
	}

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		typeIDs, err := read([]byte{
			// section size: 2 (LEB128)
			0x82, 0x80, 0x80, 0x80, 0x0,
			// function count: 1
			0x1,
			// type index of function: 0x23
			0x23,
		})
		require.NoError(t, err)
		assert.Equal(t,
			[]*Function{
				{
					TypeIndex: 0x23,
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
			// section size: 0 (LEB128)
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
			// section size: 1 (LEB128)
			0x81, 0x80, 0x80, 0x80, 0x0,
			// function count
			0x1,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidFunctionSectionTypeIndexError{
				Offset:    6,
				Index:     0,
				ReadError: io.EOF,
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})
}

func TestWASMReader_readMemorySection(t *testing.T) {

	t.Parallel()

	read := func(data []byte) ([]*Memory, error) {
		b := Buffer{data: data}
		r := NewWASMReader(&b)
		err := r.readMemorySection()
		if err != nil {
			return nil, err
		}
		require.Equal(t, offset(len(b.data)), b.offset)
		return r.Module.Memories, nil
	}

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		codes, err := read([]byte{
			// section size: 6 (LEB128)
			0x86, 0x80, 0x80, 0x80, 0x0,
			// memory count: 2
			0x2,
			// memory type / limit: no max
			0x0,
			// limit 1 min
			0x0,
			// memory type / limit: max
			0x1,
			// limit 2 min
			0x1,
			// limit 2 max
			0x2,
		})
		require.NoError(t, err)
		assert.Equal(t,
			[]*Memory{
				{
					Min: 0,
					Max: nil,
				},
				{
					Min: 1,
					Max: func() *uint32 {
						var max uint32 = 2
						return &max
					}(),
				},
			},
			codes,
		)
	})

	t.Run("invalid size", func(t *testing.T) {

		t.Parallel()

		segments, err := read([]byte{
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
		assert.Nil(t, segments)
	})

	t.Run("invalid count", func(t *testing.T) {

		t.Parallel()

		segments, err := read([]byte{
			// section size: 7 (LEB128)
			0x80, 0x80, 0x80, 0x80, 0x0,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidMemorySectionMemoryCountError{
				Offset:    5,
				ReadError: io.EOF,
			},
			err,
		)
		assert.Nil(t, segments)
	})

	t.Run("missing limit indicator", func(t *testing.T) {

		t.Parallel()

		segments, err := read([]byte{
			// section size: 10 (LEB128)
			0x8a, 0x80, 0x80, 0x80, 0x0,
			// segment count
			0x1,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidMemoryError{
				Index: 0,
				ReadError: InvalidLimitIndicatorError{
					Offset:    6,
					ReadError: io.EOF,
				},
			},
			err,
		)
		assert.Nil(t, segments)
	})

	t.Run("invalid limit indicator", func(t *testing.T) {

		t.Parallel()

		segments, err := read([]byte{
			// section size: 10 (LEB128)
			0x8a, 0x80, 0x80, 0x80, 0x0,
			// segment count
			0x1,
			// limit indicator
			0xFF,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidMemoryError{
				Index: 0,
				ReadError: InvalidLimitIndicatorError{
					Offset:         6,
					LimitIndicator: 0xFF,
				},
			},
			err,
		)
		assert.Nil(t, segments)
	})

	t.Run("invalid min", func(t *testing.T) {

		t.Parallel()

		segments, err := read([]byte{
			// section size: 10 (LEB128)
			0x8a, 0x80, 0x80, 0x80, 0x0,
			// segment count
			0x1,
			// limit indicator: no max = 0x0
			0x0,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidMemoryError{
				Index: 0,
				ReadError: InvalidLimitMinError{
					Offset:    7,
					ReadError: io.EOF,
				},
			},
			err,
		)
		assert.Nil(t, segments)
	})

	t.Run("invalid max", func(t *testing.T) {

		t.Parallel()

		segments, err := read([]byte{
			// section size: 10 (LEB128)
			0x8a, 0x80, 0x80, 0x80, 0x0,
			// segment count
			0x1,
			// limit indicator: max = 0x1
			0x1,
			// min
			0x1,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidMemoryError{
				Index: 0,
				ReadError: InvalidLimitMaxError{
					Offset:    8,
					ReadError: io.EOF,
				},
			},
			err,
		)
		assert.Nil(t, segments)
	})
}

func TestWASMReader_readExportSection(t *testing.T) {

	t.Parallel()

	read := func(data []byte) ([]*Export, error) {
		b := Buffer{data: data}
		r := NewWASMReader(&b)
		err := r.readExportSection()
		if err != nil {
			return nil, err
		}
		require.Equal(t, offset(len(b.data)), b.offset)
		return r.Module.Exports, nil
	}

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		typeIndices, err := read([]byte{
			// section size: 7 (LEB128)
			0x87, 0x80, 0x80, 0x80, 0x0,
			// export count: 1
			0x1,
			// name length
			0x3,
			// name = "foo"
			0x66, 0x6f, 0x6f,
			// indicator: function = 0
			0x0,
			// index of function: 0
			0x1,
		})
		require.NoError(t, err)
		assert.Equal(t,
			[]*Export{
				{
					Name: "foo",
					Descriptor: FunctionExport{
						FunctionIndex: 1,
					},
				},
			},
			typeIndices,
		)
	})

	t.Run("invalid size", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			0x80, 0x80, 0x80,
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
			InvalidExportSectionExportCountError{
				Offset:    5,
				ReadError: io.EOF,
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid name", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 1 (LEB128)
			0x81, 0x80, 0x80, 0x80, 0x0,
			// export count
			0x1,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidExportError{
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

	t.Run("missing indicator", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 5 (LEB128)
			0x85, 0x80, 0x80, 0x80, 0x0,
			// export count
			0x1,
			// name length
			0x3,
			// name = "foo"
			0x66, 0x6f, 0x6f,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidExportError{
				Index: 0,
				ReadError: InvalidExportIndicatorError{
					Offset:    10,
					ReadError: io.EOF,
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid indicator", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 6 (LEB128)
			0x86, 0x80, 0x80, 0x80, 0x0,
			// import count
			0x1,
			// name length
			0x3,
			// name = "foo"
			0x66, 0x6f, 0x6f,
			// indicator: invalid
			0xFF,
			// index
			0x1,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidExportError{
				Index: 0,
				ReadError: InvalidExportIndicatorError{
					Offset:          10,
					ExportIndicator: 0xFF,
					ReadError:       nil,
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})

	t.Run("invalid function index", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 6 (LEB128)
			0x86, 0x80, 0x80, 0x80, 0x0,
			// import count
			0x1,
			// name length
			0x3,
			// name = "bar"
			0x62, 0x61, 0x72,
			// indicator: function = 0
			0x0,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidExportError{
				Index: 0,
				ReadError: InvalidExportSectionIndexError{
					Offset:    11,
					ReadError: io.EOF,
				},
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})
}

func TestWASMReader_readStartSection(t *testing.T) {

	t.Parallel()

	read := func(data []byte) (*uint32, error) {
		b := Buffer{data: data}
		r := NewWASMReader(&b)
		err := r.readStartSection()
		if err != nil {
			return nil, err
		}
		require.Equal(t, offset(len(b.data)), b.offset)
		return r.Module.StartFunctionIndex, nil
	}

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		typeIDs, err := read([]byte{
			// section size: 1 (LEB128)
			0x81, 0x80, 0x80, 0x80, 0x0,
			// function index: 1
			0x1,
		})
		require.NoError(t, err)
		assert.Equal(t,
			func() *uint32 {
				var funcIndex uint32 = 1
				return &funcIndex
			}(),
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

	t.Run("invalid function type ID", func(t *testing.T) {

		t.Parallel()

		funcTypes, err := read([]byte{
			// section size: 7 (LEB128)
			0x81, 0x80, 0x80, 0x80, 0x0,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidStartSectionFunctionIndexError{
				Offset:    5,
				ReadError: io.EOF,
			},
			err,
		)
		assert.Nil(t, funcTypes)
	})
}

func TestWASMReader_readCodeSection(t *testing.T) {

	t.Parallel()

	read := func(data []byte) ([]*Function, error) {
		b := Buffer{data: data}
		r := NewWASMReader(&b)
		err := r.readCodeSection()
		if err != nil {
			return nil, err
		}
		require.Equal(t, offset(len(b.data)), b.offset)
		return r.Module.Functions, nil
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
			[]*Function{
				{
					Code: &Code{
						Locals: []ValueType{
							ValueTypeI32,
						},
						Instructions: []Instruction{
							InstructionLocalGet{LocalIndex: 0},
							InstructionLocalGet{LocalIndex: 1},
							InstructionI32Add{},
						},
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
			// section size: 0 (LEB128)
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
			// section size: 1 (LEB128)
			0x81, 0x80, 0x80, 0x80, 0x0,
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
			// section size: 6 (LEB128)
			0x86, 0x80, 0x80, 0x80, 0x0,
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
			// section size: 7 (LEB128)
			0x87, 0x80, 0x80, 0x80, 0x0,
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
			// section size: 8 (LEB128)
			0x88, 0x80, 0x80, 0x80, 0x0,
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
			// section size: 10 (LEB128)
			0x8a, 0x80, 0x80, 0x80, 0x0,
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
			// section size: 14 (LEB128)
			0x8e, 0x80, 0x80, 0x80, 0x0,
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

func TestWASMReader_readDataSection(t *testing.T) {

	t.Parallel()

	read := func(data []byte) ([]*Data, error) {
		b := Buffer{data: data}
		r := NewWASMReader(&b)
		err := r.readDataSection()
		if err != nil {
			return nil, err
		}
		require.Equal(t, offset(len(b.data)), b.offset)
		return r.Module.Data, nil
	}

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		codes, err := read([]byte{
			// section size: 9 (LEB128)
			0x89, 0x80, 0x80, 0x80, 0x0,
			// segment count: 1
			0x1,
			// memory index
			0x1,
			// i32.const 2
			0x41, 0x2,
			// end
			0xb,
			// byte count
			0x3,
			// init (bytes 0x3, 0x4, 0x5)
			0x3, 0x4, 0x5,
		})
		require.NoError(t, err)
		assert.Equal(t,
			[]*Data{
				{
					MemoryIndex: 1,
					Offset: []Instruction{
						InstructionI32Const{Value: 2},
					},
					Init: []byte{3, 4, 5},
				},
			},
			codes,
		)
	})

	t.Run("invalid size", func(t *testing.T) {

		t.Parallel()

		segments, err := read([]byte{
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
		assert.Nil(t, segments)
	})

	t.Run("invalid count", func(t *testing.T) {

		t.Parallel()

		segments, err := read([]byte{
			// section size: 7 (LEB128)
			0x80, 0x80, 0x80, 0x80, 0x0,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidDataSectionSegmentCountError{
				Offset:    5,
				ReadError: io.EOF,
			},
			err,
		)
		assert.Nil(t, segments)
	})

	t.Run("invalid memory index", func(t *testing.T) {

		t.Parallel()

		segments, err := read([]byte{
			// section size: 10 (LEB128)
			0x8a, 0x80, 0x80, 0x80, 0x0,
			// segment count
			0x1,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidDataSegmentError{
				Index: 0,
				ReadError: InvalidDataSectionMemoryIndexError{
					Offset:    6,
					ReadError: io.EOF,
				},
			},
			err,
		)
		assert.Nil(t, segments)
	})

	t.Run("invalid instruction", func(t *testing.T) {

		t.Parallel()

		segments, err := read([]byte{
			// section size: 10 (LEB128)
			0x8a, 0x80, 0x80, 0x80, 0x0,
			// segment count
			0x1,
			// memory index
			0x0,
			// invalid opcode
			0xff,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidDataSegmentError{
				Index: 0,
				ReadError: InvalidOpcodeError{
					Offset:    7,
					Opcode:    0xff,
					ReadError: nil,
				},
			},
			err,
		)
		assert.Nil(t, segments)
	})

	t.Run("missing end", func(t *testing.T) {

		t.Parallel()

		segments, err := read([]byte{
			// section size: 9 (LEB128)
			0x89, 0x80, 0x80, 0x80, 0x0,
			// segment count: 1
			0x1,
			// memory index
			0x1,
			// i32.const 2
			0x41, 0x2,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidDataSegmentError{
				Index: 0,
				ReadError: MissingEndInstructionError{
					Offset: 9,
				},
			},
			err,
		)
		assert.Nil(t, segments)
	})

	t.Run("invalid init byte count", func(t *testing.T) {

		t.Parallel()

		segments, err := read([]byte{
			// section size: 9 (LEB128)
			0x89, 0x80, 0x80, 0x80, 0x0,
			// segment count: 1
			0x1,
			// memory index
			0x1,
			// i32.const 2
			0x41, 0x2,
			// end
			0xb,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidDataSegmentError{
				Index: 0,
				ReadError: InvalidDataSectionInitByteCountError{
					Offset:    10,
					ReadError: io.EOF,
				},
			},
			err,
		)
		assert.Nil(t, segments)
	})

	t.Run("invalid init bytes", func(t *testing.T) {

		t.Parallel()

		segments, err := read([]byte{
			// section size: 9 (LEB128)
			0x89, 0x80, 0x80, 0x80, 0x0,
			// segment count: 1
			0x1,
			// memory index
			0x1,
			// i32.const 2
			0x41, 0x2,
			// end
			0xb,
			// byte count
			0x2,
		})
		require.Error(t, err)
		assert.Equal(t,
			InvalidDataSegmentError{
				Index:     0,
				ReadError: io.EOF,
			},
			err,
		)
		assert.Nil(t, segments)
	})
}

func TestWASMReader_readName(t *testing.T) {

	t.Parallel()

	read := func(data []byte) (string, error) {
		b := Buffer{data: data}
		r := NewWASMReader(&b)
		name, err := r.readName()
		if err != nil {
			return "", err
		}
		require.Equal(t, offset(len(b.data)), b.offset)
		return name, nil
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

func TestWASMReader_readInstruction(t *testing.T) {

	t.Parallel()

	t.Run("block, i32 result", func(t *testing.T) {

		t.Parallel()

		b := Buffer{
			data: []byte{
				// block
				0x02,
				// i32
				0x7f,
				// i32.const
				0x41,
				0x01,
				// end
				0x0b,
			},
			offset: 0,
		}
		r := NewWASMReader(&b)

		expected := InstructionBlock{
			Block: Block{
				BlockType: ValueTypeI32,
				Instructions1: []Instruction{
					InstructionI32Const{Value: 1},
				},
				Instructions2: nil,
			},
		}
		actual, err := r.readInstruction()
		require.NoError(t, err)

		require.Equal(t, expected, actual)
		require.Equal(t, offset(len(b.data)), b.offset)
	})

	t.Run("block, type index result", func(t *testing.T) {

		t.Parallel()

		b := Buffer{
			data: []byte{
				// block
				0x02,
				// type index: 2
				0x2,
				// unreachable
				0x0,
				// end
				0x0b,
			},
			offset: 0,
		}
		r := NewWASMReader(&b)

		expected := InstructionBlock{
			Block: Block{
				BlockType: TypeIndexBlockType{TypeIndex: 2},
				Instructions1: []Instruction{
					InstructionUnreachable{},
				},
				Instructions2: nil,
			},
		}
		actual, err := r.readInstruction()
		require.NoError(t, err)

		require.Equal(t, expected, actual)
		require.Equal(t, offset(len(b.data)), b.offset)
	})

	t.Run("block, type index result, type index too large", func(t *testing.T) {

		t.Parallel()

		b := Buffer{
			data: []byte{
				// block
				0x02,
				// type index: math.MaxUint32 + 1
				0x80, 0x80, 0x80, 0x80, 0x10,
				// unreachable
				0x0,
				// end
				0x0b,
			},
			offset: 0,
		}
		r := NewWASMReader(&b)

		_, err := r.readInstruction()
		require.Equal(t,
			InvalidBlockTypeTypeIndexError{
				TypeIndex: math.MaxUint32 + 1,
				Offset:    1,
			},
			err,
		)
	})

	t.Run("block, i32 result, second instructions", func(t *testing.T) {

		t.Parallel()

		b := Buffer{
			data: []byte{
				// block
				0x02,
				// i32
				0x7f,
				// i32.const
				0x41,
				0x01,
				// else
				0x05,
				// i32.const
				0x41,
				0x02,
				// end
				0x0b,
			},
			offset: 0,
		}
		r := NewWASMReader(&b)

		_, err := r.readInstruction()
		require.Equal(t, InvalidBlockSecondInstructionsError{
			Offset: 4,
		}, err)
	})

	t.Run("loop, i32 result", func(t *testing.T) {

		t.Parallel()

		b := Buffer{
			data: []byte{
				// loop
				0x03,
				// i32
				0x7f,
				// i32.const
				0x41,
				0x01,
				// end
				0x0b,
			},
			offset: 0,
		}
		r := NewWASMReader(&b)

		expected := InstructionLoop{
			Block: Block{
				BlockType: ValueTypeI32,
				Instructions1: []Instruction{
					InstructionI32Const{Value: 1},
				},
				Instructions2: nil,
			},
		}
		actual, err := r.readInstruction()
		require.NoError(t, err)

		require.Equal(t, expected, actual)
		require.Equal(t, offset(len(b.data)), b.offset)
	})

	t.Run("loop, i32 result, second instructions", func(t *testing.T) {

		t.Parallel()

		b := Buffer{
			data: []byte{
				// loop
				0x03,
				// i32
				0x7f,
				// i32.const
				0x41,
				0x01,
				// else
				0x05,
				// i32.const
				0x41,
				0x02,
				// end
				0x0b,
			},
			offset: 0,
		}
		r := NewWASMReader(&b)

		_, err := r.readInstruction()
		require.Equal(t, InvalidBlockSecondInstructionsError{
			Offset: 4,
		}, err)
	})

	t.Run("if, i32 result", func(t *testing.T) {

		t.Parallel()

		b := Buffer{
			data: []byte{
				// if
				0x04,
				// i32
				0x7f,
				// i32.const
				0x41,
				0x01,
				// end
				0x0b,
			},
			offset: 0,
		}
		r := NewWASMReader(&b)

		expected := InstructionIf{
			Block: Block{
				BlockType: ValueTypeI32,
				Instructions1: []Instruction{
					InstructionI32Const{Value: 1},
				},
				Instructions2: nil,
			},
		}
		actual, err := r.readInstruction()
		require.NoError(t, err)

		require.Equal(t, expected, actual)
		require.Equal(t, offset(len(b.data)), b.offset)
	})

	t.Run("if-else, i32 result", func(t *testing.T) {

		t.Parallel()

		b := Buffer{
			data: []byte{
				// loop
				0x04,
				// i32
				0x7f,
				// i32.const
				0x41,
				0x01,
				// else
				0x05,
				// i32.const
				0x41,
				0x02,
				// end
				0x0b,
			},
			offset: 0,
		}
		r := NewWASMReader(&b)

		expected := InstructionIf{
			Block: Block{
				BlockType: ValueTypeI32,
				Instructions1: []Instruction{
					InstructionI32Const{Value: 1},
				},
				Instructions2: []Instruction{
					InstructionI32Const{Value: 2},
				},
			},
		}
		actual, err := r.readInstruction()
		require.NoError(t, err)

		require.Equal(t, expected, actual)
		require.Equal(t, offset(len(b.data)), b.offset)
	})

	t.Run("br_table", func(t *testing.T) {

		t.Parallel()

		b := Buffer{
			data: []byte{
				// br_table
				0x0e,
				// number of branch depths
				0x04,
				// 1. branch depth
				0x03,
				// 2. branch depth
				0x02,
				// 3. branch depth
				0x01,
				// 4. branch depth
				0x00,
				// default branch depth
				0x04,
			},
			offset: 0,
		}
		r := NewWASMReader(&b)

		expected := InstructionBrTable{
			LabelIndices:      []uint32{3, 2, 1, 0},
			DefaultLabelIndex: 4,
		}
		actual, err := r.readInstruction()
		require.NoError(t, err)

		require.Equal(t, expected, actual)
		require.Equal(t, offset(len(b.data)), b.offset)
	})
}

func TestWASMReader_readNameSection(t *testing.T) {

	t.Parallel()

	b := Buffer{
		data: []byte{
			// section size: 37 (LEB128)
			0xa5, 0x80, 0x80, 0x80, 0x0,
			// name length
			0x4,
			// name = "name"
			0x6e, 0x61, 0x6d, 0x65,
			// sub-section ID: module name = 0
			0x0,
			// sub-section size: 5 (LEB128)
			0x85, 0x80, 0x80, 0x80, 0x0,
			// name length
			0x4,
			// name = "test"
			0x74, 0x65, 0x73, 0x74,
			// sub-section ID: function names = 1
			0x1,
			// sub-section size: 15 (LEB128)
			0x8f, 0x80, 0x80, 0x80, 0x0,
			// name count
			0x2,
			// function index = 0
			0x0,
			// name length
			0x7,
			// name = "foo.bar"
			0x66, 0x6f, 0x6f, 0x2e, 0x62, 0x61, 0x72,
			// function index = 1
			0x1,
			// name length
			0x3,
			// name = "add"
			0x61, 0x64, 0x64,
		},
		offset: 0,
	}

	r := NewWASMReader(&b)

	err := r.readCustomSection()
	require.NoError(t, err)

	require.Equal(t, offset(len(b.data)), b.offset)
}
