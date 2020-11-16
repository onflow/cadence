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
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func wasm2wat(binary []byte) string {
	f, err := ioutil.TempFile("", "wasm")
	if err != nil {
		panic(err)
	}

	defer os.Remove(f.Name())

	_, err = f.Write(binary)
	if err != nil {
		panic(err)
	}

	err = f.Close()
	if err != nil {
		panic(err)
	}

	cmd := exec.Command("wasm2wat", f.Name())
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	return string(out)
}

func TestWasmWriter_writeMagicAndVersion(t *testing.T) {

	t.Parallel()

	var b buf
	w := WASMWriter{&b}

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

func TestWasmWriter_writeTypeSection(t *testing.T) {

	t.Parallel()

	var b buf
	w := WASMWriter{&b}

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

func TestWasmWriter_writeFunctionSection(t *testing.T) {

	t.Parallel()

	var b buf
	w := WASMWriter{&b}

	functions := []*Function{
		{
			// not used, just for testing
			Name:   "add",
			TypeID: 0,
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
			// type ID of function: 0
			0x0,
		},
		b.data,
	)
}

func TestWasmWriter_writeCodeSection(t *testing.T) {

	t.Parallel()

	var b buf
	w := WASMWriter{&b}

	functions := []*Function{
		{
			// not used, just for testing
			Name: "add",
			// not used, just for testing
			TypeID: 0,
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

func TestWasmWriter(t *testing.T) {

	t.Parallel()

	var b buf

	w := WASMWriter{&b}

	err := w.writeMagicAndVersion()
	require.NoError(t, err)

	err = w.writeTypeSection([]*FunctionType{
		{
			Params:  []ValueType{ValueTypeI32, ValueTypeI32},
			Results: []ValueType{ValueTypeI32},
		},
	})
	require.NoError(t, err)

	functions := []*Function{
		{
			// not used, just for testing
			Name:   "add",
			TypeID: 0,
			Code: &Code{
				// not used, just for testing
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

	err = w.writeFunctionSection(functions)
	require.NoError(t, err)

	err = w.writeCodeSection(functions)
	require.NoError(t, err)

	require.Equal(t,
		[]byte{
			0x0, 0x61, 0x73, 0x6d, 0x1, 0x0, 0x0, 0x0,
			0x1, 0x87, 0x80, 0x80, 0x80, 0x0, 0x1, 0x60,
			0x2, 0x7f, 0x7f, 0x1, 0x7f, 0x3, 0x82, 0x80,
			0x80, 0x80, 0x0, 0x1, 0x0, 0xa, 0x8f, 0x80,
			0x80, 0x80, 0x0, 0x1, 0x89, 0x80, 0x80, 0x80,
			0x0, 0x1, 0x1, 0x7f, 0x20, 0x0, 0x20, 0x1,
			0x6a, 0xb,
		},
		b.data,
	)

	require.Equal(t,
		`(module
  (type (;0;) (func (param i32 i32) (result i32)))
  (func (;0;) (type 0) (param i32 i32) (result i32)
    (local i32)
    local.get 0
    local.get 1
    i32.add))
`,
		wasm2wat(b.data),
	)
}
