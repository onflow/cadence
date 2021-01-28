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
	"github.com/onflow/cadence/runtime/compiler/ir"
	"github.com/onflow/cadence/runtime/compiler/wasm"
	"github.com/onflow/cadence/runtime/errors"
)

type WasmCodeGen struct {
	mod  *wasm.ModuleBuilder
	code *wasm.Code
}

func (codeGen *WasmCodeGen) VisitInt(i ir.Int) ir.Repr {
	// TODO: box, treated as uint8 for now
	codeGen.emit(wasm.InstructionI32Const{Value: int32(i.Value[1])})
	return nil
}

func (codeGen *WasmCodeGen) VisitSequence(sequence *ir.Sequence) ir.Repr {
	for _, stmt := range sequence.Stmts {
		stmt.Accept(codeGen)
	}
	return nil
}

func (codeGen *WasmCodeGen) VisitBlock(_ *ir.Block) ir.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (codeGen *WasmCodeGen) VisitLoop(_ *ir.Loop) ir.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (codeGen *WasmCodeGen) VisitIf(_ *ir.If) ir.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (codeGen *WasmCodeGen) VisitBranch(_ *ir.Branch) ir.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (codeGen *WasmCodeGen) VisitBranchIf(_ *ir.BranchIf) ir.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (codeGen *WasmCodeGen) VisitStoreLocal(storeLocal *ir.StoreLocal) ir.Repr {
	storeLocal.Exp.Accept(codeGen)
	codeGen.emit(wasm.InstructionLocalSet{
		LocalIndex: storeLocal.LocalIndex,
	})
	return nil
}

func (codeGen *WasmCodeGen) VisitDrop(_ *ir.Drop) ir.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (codeGen *WasmCodeGen) VisitReturn(r *ir.Return) ir.Repr {
	r.Exp.Accept(codeGen)
	codeGen.emit(wasm.InstructionReturn{})
	return nil
}

func (codeGen *WasmCodeGen) VisitConst(c *ir.Const) ir.Repr {
	c.Constant.Accept(codeGen)
	return nil
}

func (codeGen *WasmCodeGen) VisitCopyLocal(c *ir.CopyLocal) ir.Repr {
	// TODO: copy
	codeGen.emit(wasm.InstructionLocalGet{
		LocalIndex: c.LocalIndex,
	})
	return nil
}

func (codeGen *WasmCodeGen) VisitMoveLocal(_ *ir.MoveLocal) ir.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (codeGen *WasmCodeGen) VisitUnOpExpr(_ *ir.UnOpExpr) ir.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (codeGen *WasmCodeGen) VisitBinOpExpr(expr *ir.BinOpExpr) ir.Repr {
	expr.Left.Accept(codeGen)
	expr.Right.Accept(codeGen)
	// TODO: add remaining operations, take types into account
	switch expr.Op {
	case ir.BinOpPlus:
		// TODO: take types into account
		codeGen.emit(wasm.InstructionI32Add{})
		return nil
	}
	panic(errors.NewUnreachableError())
}

func (codeGen *WasmCodeGen) VisitCall(_ *ir.Call) ir.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (codeGen *WasmCodeGen) VisitFunc(f *ir.Func) ir.Repr {
	codeGen.code = &wasm.Code{}
	codeGen.code.Locals = generateWasmLocalTypes(f.Locals)
	f.Statement.Accept(codeGen)
	functionType := generateWasmFunctionType(f.Type)
	codeGen.mod.AddFunction(f.Name, functionType, codeGen.code)
	return nil
}

func (codeGen *WasmCodeGen) emit(inst wasm.Instruction) {
	codeGen.code.Instructions = append(codeGen.code.Instructions, inst)
}

func GenerateWasm(funcs []*ir.Func) *wasm.Module {
	g := &WasmCodeGen{
		mod: &wasm.ModuleBuilder{},
	}
	for _, f := range funcs {
		f.Accept(g)
	}
	return g.mod.Build()
}

func generateWasmLocalTypes(locals []ir.Local) []wasm.ValueType {
	result := make([]wasm.ValueType, len(locals))
	for i, local := range locals {
		result[i] = generateWasmValType(local.Type)
	}
	return result
}

func generateWasmValType(valType ir.ValType) wasm.ValueType {
	// TODO: add remaining types
	switch valType {
	case ir.ValTypeInt:
		// TODO: box, return ref
		return wasm.ValueTypeI32
	}

	panic(errors.NewUnreachableError())
}

func generateWasmFunctionType(funcType ir.FuncType) *wasm.FunctionType {
	params := make([]wasm.ValueType, len(funcType.Params))
	for i, param := range funcType.Params {
		params[i] = generateWasmValType(param)
	}

	// TODO: handle void, no results
	result := generateWasmValType(funcType.Result)

	return &wasm.FunctionType{
		Params: params,
		Results: []wasm.ValueType{
			result,
		},
	}
}
