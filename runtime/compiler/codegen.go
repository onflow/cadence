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
	"fmt"

	"github.com/onflow/cadence/runtime/compiler/ir"
	"github.com/onflow/cadence/runtime/compiler/wasm"
	"github.com/onflow/cadence/runtime/errors"
)

const RuntimeModuleName = "crt"

type wasmCodeGen struct {
	mod                        *wasm.ModuleBuilder
	code                       *wasm.Code
	runtimeFunctionIndexInt    uint32
	runtimeFunctionIndexString uint32
	runtimeFunctionIndexAdd    uint32
}

func (codeGen *wasmCodeGen) VisitInt(i ir.Int) ir.Repr {
	codeGen.emitConstantCall(
		codeGen.runtimeFunctionIndexInt,
		i.Value,
	)
	return nil
}

func (codeGen *wasmCodeGen) VisitString(s ir.String) ir.Repr {
	codeGen.emitConstantCall(
		codeGen.runtimeFunctionIndexString,
		[]byte(s.Value),
	)
	return nil
}

func (codeGen *wasmCodeGen) VisitSequence(sequence *ir.Sequence) ir.Repr {
	for _, stmt := range sequence.Stmts {
		stmt.Accept(codeGen)
	}
	return nil
}

func (codeGen *wasmCodeGen) VisitBlock(_ *ir.Block) ir.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (codeGen *wasmCodeGen) VisitLoop(_ *ir.Loop) ir.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (codeGen *wasmCodeGen) VisitIf(_ *ir.If) ir.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (codeGen *wasmCodeGen) VisitBranch(_ *ir.Branch) ir.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (codeGen *wasmCodeGen) VisitBranchIf(_ *ir.BranchIf) ir.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (codeGen *wasmCodeGen) VisitStoreLocal(storeLocal *ir.StoreLocal) ir.Repr {
	storeLocal.Exp.Accept(codeGen)
	codeGen.emit(wasm.InstructionLocalSet{
		LocalIndex: storeLocal.LocalIndex,
	})
	return nil
}

func (codeGen *wasmCodeGen) VisitDrop(_ *ir.Drop) ir.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (codeGen *wasmCodeGen) VisitReturn(r *ir.Return) ir.Repr {
	r.Exp.Accept(codeGen)
	codeGen.emit(wasm.InstructionReturn{})
	return nil
}

func (codeGen *wasmCodeGen) VisitConst(c *ir.Const) ir.Repr {
	c.Constant.Accept(codeGen)
	return nil
}

func (codeGen *wasmCodeGen) VisitCopyLocal(c *ir.CopyLocal) ir.Repr {
	// TODO: copy
	codeGen.emit(wasm.InstructionLocalGet{
		LocalIndex: c.LocalIndex,
	})
	return nil
}

func (codeGen *wasmCodeGen) VisitMoveLocal(_ *ir.MoveLocal) ir.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (codeGen *wasmCodeGen) VisitUnOpExpr(_ *ir.UnOpExpr) ir.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (codeGen *wasmCodeGen) VisitBinOpExpr(expr *ir.BinOpExpr) ir.Repr {
	expr.Left.Accept(codeGen)
	expr.Right.Accept(codeGen)
	// TODO: add remaining operations, take types into account
	switch expr.Op {
	case ir.BinOpPlus:
		codeGen.emit(wasm.InstructionCall{
			FuncIndex: codeGen.runtimeFunctionIndexAdd,
		})
		return nil
	}
	panic(errors.NewUnreachableError())
}

func (codeGen *wasmCodeGen) VisitCall(_ *ir.Call) ir.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (codeGen *wasmCodeGen) VisitFunc(f *ir.Func) ir.Repr {
	codeGen.code = &wasm.Code{}
	codeGen.code.Locals = generateWasmLocalTypes(f.Locals)
	f.Statement.Accept(codeGen)
	functionType := generateWasmFunctionType(f.Type)
	funcIndex := codeGen.mod.AddFunction(f.Name, functionType, codeGen.code)
	// TODO: make export dependent on visibility modifier
	codeGen.mod.AddExport(&wasm.Export{
		Name: f.Name,
		Descriptor: wasm.FunctionExport{
			FunctionIndex: funcIndex,
		},
	})
	return nil
}

func (codeGen *wasmCodeGen) emit(inst wasm.Instruction) {
	codeGen.code.Instructions = append(codeGen.code.Instructions, inst)
}

func (codeGen *wasmCodeGen) addConstant(value []byte) uint32 {
	offset := codeGen.mod.RequireMemory(uint32(len(value)))
	// TODO: optimize:
	//   let module builder generate one data entry of all constants,
	//   instead of one data entry for each constant
	codeGen.mod.AddData(offset, value)
	return offset
}

func (codeGen *wasmCodeGen) emitConstantCall(funcIndex uint32, value []byte) {
	memoryOffset := codeGen.addConstant(value)
	codeGen.emit(wasm.InstructionI32Const{Value: int32(memoryOffset)})

	length := int32(len(value))
	codeGen.emit(wasm.InstructionI32Const{Value: length})

	codeGen.emit(wasm.InstructionCall{FuncIndex: funcIndex})
}

var constantFunctionType = &wasm.FunctionType{
	Params: []wasm.ValueType{
		// memory offset
		wasm.ValueTypeI32,
		// length
		wasm.ValueTypeI32,
	},
	Results: []wasm.ValueType{
		wasm.ValueTypeExternRef,
	},
}

var addFunctionType = &wasm.FunctionType{
	Params: []wasm.ValueType{
		wasm.ValueTypeExternRef,
		wasm.ValueTypeExternRef,
	},
	Results: []wasm.ValueType{
		wasm.ValueTypeExternRef,
	},
}

func (codeGen *wasmCodeGen) addRuntimeImports() {
	// NOTE: ensure to update the imports in the vm
	codeGen.runtimeFunctionIndexInt = codeGen.addRuntimeImport("Int", constantFunctionType)
	codeGen.runtimeFunctionIndexString = codeGen.addRuntimeImport("String", constantFunctionType)
	codeGen.runtimeFunctionIndexAdd = codeGen.addRuntimeImport("add", addFunctionType)
}

func (codeGen *wasmCodeGen) addRuntimeImport(name string, funcType *wasm.FunctionType) uint32 {
	funcIndex, err := codeGen.mod.AddFunctionImport(RuntimeModuleName, name, funcType)
	if err != nil {
		panic(fmt.Errorf("failed to add runtime import of function %s: %w", name, err))
	}
	return funcIndex
}

func GenerateWasm(funcs []*ir.Func) *wasm.Module {
	g := &wasmCodeGen{
		mod: &wasm.ModuleBuilder{},
	}

	g.addRuntimeImports()

	for _, f := range funcs {
		f.Accept(g)
	}

	g.mod.ExportMemory("mem")

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
	case ir.ValTypeInt,
		ir.ValTypeString:

		return wasm.ValueTypeExternRef
	}

	panic(errors.NewUnreachableError())
}

func generateWasmFunctionType(funcType ir.FuncType) *wasm.FunctionType {
	// generate parameter types
	params := make([]wasm.ValueType, len(funcType.Params))
	for i, param := range funcType.Params {
		params[i] = generateWasmValType(param)
	}

	// generate result types
	results := make([]wasm.ValueType, len(funcType.Results))
	for i, result := range funcType.Results {
		results[i] = generateWasmValType(result)
	}

	return &wasm.FunctionType{
		Params:  params,
		Results: results,
	}
}
