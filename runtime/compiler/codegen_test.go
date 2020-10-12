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

package compiler

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/compiler/ir"
	"github.com/onflow/cadence/runtime/compiler/wasm"
)

func TestWasmCodeGenSimple(t *testing.T) {

	mod := GenerateWasm([]*ir.Func{
		{
			Name: "inc",
			Type: ir.FuncType{
				Params: []ir.ValType{
					ir.ValTypeInt,
				},
				Result: ir.ValTypeInt,
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
				{
					Params: []wasm.ValueType{
						wasm.ValueTypeI32,
					},
					Results: []wasm.ValueType{
						wasm.ValueTypeI32,
					},
				},
			},
			Functions: []*wasm.Function{
				{
					Name:      "inc",
					TypeIndex: 0,
					Code: &wasm.Code{
						Locals: []wasm.ValueType{
							wasm.ValueTypeI32,
							wasm.ValueTypeI32,
						},
						Instructions: []wasm.Instruction{
							wasm.InstructionI32Const{Value: 1},
							wasm.InstructionLocalSet{LocalIndex: 1},
							wasm.InstructionLocalGet{LocalIndex: 0},
							wasm.InstructionLocalGet{LocalIndex: 1},
							wasm.InstructionI32Add{},
							wasm.InstructionReturn{},
						},
					},
				},
			},
		},
		mod,
	)
}
