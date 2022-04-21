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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWASMWriter_writeMagicAndVersion(t *testing.T) {

	t.Parallel()

	var b Buffer
	w := NewWASMWriter(&b)

	err := w.writeMagicAndVersion()
	require.NoError(t, err)

	require.Equal(t,
		[]byte{
			// magic
			0x0, 0x61, 0x73, 0x6d,
			// version
			0x1, 0x0, 0x0, 0x0,
		},
		b.data,
	)
}

func TestWASMWriter_writeTypeSection(t *testing.T) {

	t.Parallel()

	var b Buffer
	w := NewWASMWriter(&b)

	err := w.writeTypeSection([]*FunctionType{
		{
			Params:  []ValueType{ValueTypeI32, ValueTypeI32},
			Results: []ValueType{ValueTypeI32},
		},
	})
	require.NoError(t, err)

	require.Equal(t,
		[]byte{
			// section ID: Type = 1
			0x1,
			// section size: 7 (LEB128)
			0x87, 0x80, 0x80, 0x80, 0x0,
			// type count
			0x1,
			// function type indicator
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
		},
		b.data,
	)
}

func TestWASMWriter_writeImportSection(t *testing.T) {

	t.Parallel()

	var b Buffer
	w := NewWASMWriter(&b)

	imports := []*Import{
		{
			Module:    "foo",
			Name:      "bar",
			TypeIndex: 1,
		},
	}

	err := w.writeImportSection(imports)
	require.NoError(t, err)

	require.Equal(t,
		[]byte{
			// section ID: Import = 2
			0x2,
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
			// type index of function: 0
			0x1,
		},
		b.data,
	)
}

func TestWASMWriter_writeFunctionSection(t *testing.T) {

	t.Parallel()

	var b Buffer
	w := NewWASMWriter(&b)

	functions := []*Function{
		{
			// not used, just for testing
			Name:      "add",
			TypeIndex: 0,
			// not used, just for testing
			Code: &Code{
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
	}

	err := w.writeFunctionSection(functions)
	require.NoError(t, err)

	require.Equal(t,
		[]byte{
			// section ID: Function = 3
			0x3,
			// section size: 2 (LEB128)
			0x82, 0x80, 0x80, 0x80, 0x0,
			// function count: 1
			0x1,
			// type index of function: 0
			0x0,
		},
		b.data,
	)
}

func TestWASMWriter_writeMemorySection(t *testing.T) {

	t.Parallel()

	var b Buffer
	w := NewWASMWriter(&b)

	memories := []*Memory{
		{
			Min: 1024,
			Max: nil,
		},
		{
			Min: 2048,
			Max: func() *uint32 {
				var max uint32 = 2
				return &max
			}(),
		},
	}

	err := w.writeMemorySection(memories)
	require.NoError(t, err)

	require.Equal(t,
		[]byte{
			// section ID: Import = 5
			0x5,
			// section size: 8 (LEB128)
			0x88, 0x80, 0x80, 0x80, 0x0,
			// memory count: 2
			0x2,
			// memory type / limit: no max
			0x0,
			// limit 1 min: 1024 (LEB128)
			0x80, 0x8,
			// memory type / limit: max
			0x1,
			// limit 2 min: 2048 (LEB128)
			0x80, 0x10,
			// limit 2 max
			0x2,
		},
		b.data,
	)
}

func TestWASMWriter_writeExportSection(t *testing.T) {

	t.Parallel()

	var b Buffer
	w := NewWASMWriter(&b)

	exports := []*Export{
		{
			Name: "foo",
			Descriptor: FunctionExport{
				FunctionIndex: 1,
			},
		},
	}

	err := w.writeExportSection(exports)
	require.NoError(t, err)

	require.Equal(t,
		[]byte{
			// section ID: Export = 7
			0x7,
			// section size: 7 (LEB128)
			0x87, 0x80, 0x80, 0x80, 0x0,
			// import count: 1
			0x1,
			// name length
			0x3,
			// name = "foo"
			0x66, 0x6f, 0x6f,
			// type indicator: function = 0
			0x0,
			// index of function: 1
			0x1,
		},
		b.data,
	)
}

func TestWASMWriter_writeStartSection(t *testing.T) {

	t.Parallel()

	var b Buffer
	w := NewWASMWriter(&b)

	err := w.writeStartSection(1)
	require.NoError(t, err)

	require.Equal(t,
		[]byte{
			// Section ID: Code = 8
			0x8,
			// section size: 15 (LEB128)
			0x81, 0x80, 0x80, 0x80, 0x0,
			// function index: 1
			0x1,
		},
		b.data,
	)
}

func TestWASMWriter_writeCodeSection(t *testing.T) {

	t.Parallel()

	var b Buffer
	w := NewWASMWriter(&b)

	functions := []*Function{
		{
			// not used, just for testing
			Name: "add",
			// not used, just for testing
			TypeIndex: 0,
			Code: &Code{
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
	}

	err := w.writeCodeSection(functions)
	require.NoError(t, err)

	require.Equal(t,
		[]byte{
			// Section ID: Code = 10
			0xa,
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
		},
		b.data,
	)
}

func TestWASMWriter_writeDataSection(t *testing.T) {

	t.Parallel()

	var b Buffer
	w := NewWASMWriter(&b)

	dataSegments := []*Data{
		{
			MemoryIndex: 1,
			Offset: []Instruction{
				InstructionI32Const{Value: 2},
			},
			Init: []byte{3, 4, 5},
		},
	}

	err := w.writeDataSection(dataSegments)
	require.NoError(t, err)

	require.Equal(t,
		[]byte{
			// section ID: Import = 11
			0xB,
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
		},
		b.data,
	)
}

func TestWASMWriter_writeName(t *testing.T) {

	t.Parallel()

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		var b Buffer
		w := NewWASMWriter(&b)

		err := w.writeName("hello")
		require.NoError(t, err)

		require.Equal(t,
			[]byte{
				// length
				0x5,
				// "hello"
				0x68, 0x65, 0x6c, 0x6c, 0x6f,
			},
			b.data,
		)
	})

	t.Run("invalid", func(t *testing.T) {

		t.Parallel()

		var b Buffer
		w := NewWASMWriter(&b)

		name := string([]byte{0xff, 0xfe, 0xfd})
		err := w.writeName(name)
		require.Error(t, err)

		assert.Equal(t,
			InvalidNonUTF8NameError{
				Name:   name,
				Offset: 0,
			},
			err,
		)

		assert.Empty(t, b.data)
	})
}

func TestWASMWriter_writeNameSection(t *testing.T) {

	t.Parallel()

	var b Buffer
	w := NewWASMWriter(&b)

	imports := []*Import{
		{
			Module: "foo",
			Name:   "bar",
		},
	}

	functions := []*Function{
		{
			Name: "add",
		},
	}

	err := w.writeNameSection("test", imports, functions)
	require.NoError(t, err)

	require.Equal(t,
		[]byte{
			// Section ID: Custom = 0
			0x0,
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
		b.data,
	)
}

func TestWASMWriterReader(t *testing.T) {

	t.Skip("WIP")

	t.Parallel()

	var b Buffer

	w := NewWASMWriter(&b)
	w.WriteNames = true

	module := &Module{
		Name: "test",
		Types: []*FunctionType{
			{
				Params:  nil,
				Results: nil,
			},
			{
				Params:  []ValueType{ValueTypeI32, ValueTypeI32},
				Results: []ValueType{ValueTypeI32},
			},
		},
		Imports: []*Import{
			{
				Module:    "env",
				Name:      "add",
				TypeIndex: 1,
			},
		},
		Functions: []*Function{
			{
				Name:      "start",
				TypeIndex: 0,
				Code: &Code{
					Instructions: []Instruction{
						InstructionReturn{},
					},
				},
			},
			{
				Name:      "add",
				TypeIndex: 1,
				Code: &Code{
					// not used, just for testing
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
		Memories: []*Memory{
			{
				Min: 1024,
				Max: func() *uint32 {
					var max uint32 = 2048
					return &max
				}(),
			},
		},
		Exports: []*Export{
			{
				Name: "add",
				Descriptor: FunctionExport{
					FunctionIndex: 0,
				},
			},
			{
				Name: "mem",
				Descriptor: MemoryExport{
					MemoryIndex: 0,
				},
			},
		},
		StartFunctionIndex: func() *uint32 {
			var funcIndex uint32 = 1
			return &funcIndex
		}(),
		Data: []*Data{
			{
				MemoryIndex: 0,
				Offset: []Instruction{
					InstructionI32Const{Value: 0},
				},
				Init: []byte{0x0, 0x1, 0x2, 0x3},
			},
		},
	}

	err := w.WriteModule(module)
	require.NoError(t, err)

	expected := []byte{
		// magic
		0x0, 0x61, 0x73, 0x6d,
		// version
		0x1, 0x0, 0x0, 0x0,
		// type section
		0x1,
		0x8a, 0x80, 0x80, 0x80, 0x0,
		0x2,
		0x60, 0x0, 0x0,
		0x60, 0x2, 0x7f, 0x7f, 0x1, 0x7f,
		// import section
		0x2,
		0x8b, 0x80, 0x80, 0x80, 0x0,
		0x1,
		0x3, 0x65, 0x6e, 0x76, 0x3, 0x61, 0x64, 0x64, 0x0, 0x1,
		// function section
		0x3,
		0x83, 0x80, 0x80, 0x80, 0x0,
		0x2,
		0x0,
		0x1,
		// memory section
		0x5,
		0x86, 0x80, 0x80, 0x80, 0x0,
		0x1,
		0x1, 0x80, 0x8, 0x80, 0x10,
		// export section
		0x07,
		0x8d, 0x80, 0x80, 0x80, 0x00,
		0x02,
		0x03, 0x61, 0x64, 0x64,
		0x00, 0x00,
		0x03, 0x6d, 0x65, 0x6d,
		0x02, 0x00,
		// start section
		0x8,
		0x81, 0x80, 0x80, 0x80, 0x0,
		0x1,
		// code section
		0xa,
		0x97, 0x80, 0x80, 0x80, 0x0,
		0x2,
		0x83, 0x80, 0x80, 0x80, 0x0, 0x0, 0xf, 0xb,
		0x89, 0x80, 0x80, 0x80, 0x0, 0x1, 0x1, 0x7f, 0x20, 0x0, 0x20, 0x1, 0x6a, 0xb,
		// data section
		0xb,
		0x8a, 0x80, 0x80, 0x80, 0x0,
		0x1,
		0x0,
		0x41, 0x0, 0xb,
		0x4,
		0x0, 0x1, 0x2, 0x3,
		// name section
		0x0,
		0xac, 0x80, 0x80, 0x80, 0x0,
		0x4, 0x6e, 0x61, 0x6d, 0x65, 0x0, 0x85, 0x80,
		0x80, 0x80, 0x0, 0x4, 0x74, 0x65, 0x73, 0x74,
		0x1, 0x96, 0x80, 0x80, 0x80, 0x0, 0x3, 0x0,
		0x7, 0x65, 0x6e, 0x76, 0x2e, 0x61, 0x64, 0x64,
		0x1, 0x5, 0x73, 0x74, 0x61, 0x72, 0x74,
		0x2, 0x3, 0x61, 0x64, 0x64,
	}
	require.Equal(t,
		expected,
		b.data,
	)

	require.Equal(t,
		`(module $test
  (type (;0;) (func))
  (type (;1;) (func (param i32 i32) (result i32)))
  (import "env" "add" (func $env.add (type 1)))
  (func $start (type 0)
    return)
  (func $add (type 1) (param i32 i32) (result i32)
    (local i32)
    local.get 0
    local.get 1
    i32.add)
  (memory (;0;) 1024 2048)
  (export "add" (func $env.add))
  (export "mem" (memory 0))
  (start $start)
  (data (;0;) (i32.const 0) "\00\01\02\03"))
`,
		WASM2WAT(b.data),
	)

	b.offset = 0

	r := NewWASMReader(&b)
	err = r.ReadModule()
	require.NoError(t, err)

	// prepare the expected module:
	// remove all names, as the name section is not read yet

	module.Name = ""
	for _, function := range module.Functions {
		function.Name = ""
	}

	require.Equal(t,
		module,
		&r.Module,
	)

	require.Equal(t,
		offset(len(expected)),
		b.offset,
	)
}

func TestWASMWriter_writeInstruction(t *testing.T) {

	t.Parallel()

	t.Run("block, i32 result", func(t *testing.T) {

		t.Parallel()

		var b Buffer
		w := NewWASMWriter(&b)

		instruction := InstructionBlock{
			Block: Block{
				BlockType: ValueTypeI32,
				Instructions1: []Instruction{
					InstructionI32Const{Value: 1},
				},
				Instructions2: nil,
			},
		}
		err := instruction.write(w)
		require.NoError(t, err)

		require.Equal(t,
			[]byte{
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
			b.data,
		)
	})

	t.Run("block, type index result", func(t *testing.T) {

		t.Parallel()

		var b Buffer
		w := NewWASMWriter(&b)

		instruction := InstructionBlock{
			Block: Block{
				BlockType: TypeIndexBlockType{TypeIndex: 2},
				Instructions1: []Instruction{
					InstructionUnreachable{},
				},
				Instructions2: nil,
			},
		}
		err := instruction.write(w)
		require.NoError(t, err)

		require.Equal(t,
			[]byte{
				// block
				0x02,
				// type index: 2
				0x2,
				// unreachable
				0x0,
				// end
				0x0b,
			},
			b.data,
		)
	})

	t.Run("block, i32 result, second instructions", func(t *testing.T) {

		t.Parallel()

		var b Buffer
		w := NewWASMWriter(&b)

		instruction := InstructionBlock{
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
		err := instruction.write(w)
		require.Equal(t, InvalidBlockSecondInstructionsError{
			Offset: 4,
		}, err)
	})

	t.Run("loop, i32 result", func(t *testing.T) {

		t.Parallel()

		var b Buffer
		w := NewWASMWriter(&b)

		instruction := InstructionLoop{
			Block: Block{
				BlockType: ValueTypeI32,
				Instructions1: []Instruction{
					InstructionI32Const{Value: 1},
				},
				Instructions2: nil,
			},
		}
		err := instruction.write(w)
		require.NoError(t, err)

		require.Equal(t,
			[]byte{
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
			b.data,
		)
	})

	t.Run("loop, i32 result, second instructions", func(t *testing.T) {

		t.Parallel()

		var b Buffer
		w := NewWASMWriter(&b)

		instruction := InstructionLoop{
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
		err := instruction.write(w)
		require.Equal(t, InvalidBlockSecondInstructionsError{
			Offset: 4,
		}, err)
	})

	t.Run("if, i32 result", func(t *testing.T) {

		t.Parallel()

		var b Buffer
		w := NewWASMWriter(&b)

		instruction := InstructionIf{
			Block: Block{
				BlockType: ValueTypeI32,
				Instructions1: []Instruction{
					InstructionI32Const{Value: 1},
				},
				Instructions2: nil,
			},
		}
		err := instruction.write(w)
		require.NoError(t, err)

		require.Equal(t,
			[]byte{
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
			b.data,
		)
	})

	t.Run("if-else, i32 result", func(t *testing.T) {

		t.Parallel()

		var b Buffer
		w := NewWASMWriter(&b)

		instruction := InstructionIf{
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
		err := instruction.write(w)
		require.NoError(t, err)

		require.Equal(t,
			[]byte{
				// ii
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
			b.data,
		)
	})

	t.Run("br_table", func(t *testing.T) {

		t.Parallel()

		var b Buffer
		w := NewWASMWriter(&b)

		instruction := InstructionBrTable{
			LabelIndices:      []uint32{3, 2, 1, 0},
			DefaultLabelIndex: 4,
		}
		err := instruction.write(w)
		require.NoError(t, err)

		require.Equal(t,
			[]byte{
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
			b.data,
		)
	})
}
