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

package compiler

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/compiler/ir"
	"github.com/onflow/cadence/runtime/compiler/wasm"
)

func TestWasmCodeGenSimple(t *testing.T) {

	t.Skip("WIP")

	mod := GenerateWasm([]*ir.Func{
		{
			Name: "inc",
			Type: ir.FuncType{
				Params: []ir.ValType{
					ir.ValTypeInt,
				},
				Results: []ir.ValType{
					ir.ValTypeInt,
				},
			},
			Locals: []ir.Local{
				{Type: ir.ValTypeInt},
				{Type: ir.ValTypeInt},
			},
			Statement: &ir.Sequence{
				Stmts: []ir.Stmt{
					&ir.StoreLocal{
						LocalIndex: 1,
						Exp: &ir.Const{
							Constant: ir.Int{Value: []byte{1, 1}},
						},
					},
					&ir.Return{
						Exp: &ir.BinOpExpr{
							Op: ir.BinOpPlus,
							Left: &ir.CopyLocal{
								LocalIndex: 0,
							},
							Right: &ir.CopyLocal{
								LocalIndex: 1,
							},
						},
					},
				},
			},
		},
	})

	require.Equal(t,
		&wasm.Module{
			Types: []*wasm.FunctionType{
				// function type of crt.Int
				{
					Params: []wasm.ValueType{
						wasm.ValueTypeI32,
						wasm.ValueTypeI32,
					},
					Results: []wasm.ValueType{
						wasm.ValueTypeExternRef,
					},
				},
				// function type of crt.String
				{
					Params: []wasm.ValueType{
						wasm.ValueTypeI32,
						wasm.ValueTypeI32,
					},
					Results: []wasm.ValueType{
						wasm.ValueTypeExternRef,
					},
				},
				// function type of add
				{
					Params: []wasm.ValueType{
						wasm.ValueTypeExternRef,
						wasm.ValueTypeExternRef,
					},
					Results: []wasm.ValueType{
						wasm.ValueTypeExternRef,
					},
				},
				// function type of inc
				{
					Params: []wasm.ValueType{
						wasm.ValueTypeExternRef,
					},
					Results: []wasm.ValueType{
						wasm.ValueTypeExternRef,
					},
				},
			},
			Imports: []*wasm.Import{
				{
					Module:    RuntimeModuleName,
					Name:      "Int",
					TypeIndex: 0,
				},
				{
					Module:    RuntimeModuleName,
					Name:      "String",
					TypeIndex: 1,
				},
				{
					Module:    RuntimeModuleName,
					Name:      "add",
					TypeIndex: 2,
				},
			},
			Functions: []*wasm.Function{
				{
					Name:      "inc",
					TypeIndex: 3,
					Code: &wasm.Code{
						Locals: []wasm.ValueType{
							wasm.ValueTypeExternRef,
							wasm.ValueTypeExternRef,
						},
						Instructions: []wasm.Instruction{
							wasm.InstructionI32Const{Value: 0},
							wasm.InstructionI32Const{Value: 2},
							wasm.InstructionCall{FuncIndex: 0},
							wasm.InstructionLocalSet{LocalIndex: 1},
							wasm.InstructionLocalGet{LocalIndex: 0},
							wasm.InstructionLocalGet{LocalIndex: 1},
							wasm.InstructionCall{FuncIndex: 2},
							wasm.InstructionReturn{},
						},
					},
				},
			},
			Memories: []*wasm.Memory{
				{
					Min: 1,
					Max: nil,
				},
			},
			Data: []*wasm.Data{
				// load [0x1, 0x1] at offset 0
				{
					MemoryIndex: 0,
					Offset: []wasm.Instruction{
						wasm.InstructionI32Const{Value: 0},
					},
					Init: []byte{
						// positive flag
						0x1,
						// integer 1
						0x1,
					},
				},
			},
			Exports: []*wasm.Export{
				{
					Name: "inc",
					Descriptor: wasm.FunctionExport{
						FunctionIndex: 3,
					},
				},
				{
					Name: "mem",
					Descriptor: wasm.MemoryExport{
						MemoryIndex: 0,
					},
				},
			},
		},
		mod,
	)

	var buf wasm.Buffer
	w := wasm.NewWASMWriter(&buf)
	err := w.WriteModule(mod)
	require.NoError(t, err)

	_ = wasm.WASM2WAT(buf.Bytes())
}
