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
	"errors"
	"math"
)

// ModuleBuilder allows building modules
//
type ModuleBuilder struct {
	functionImports    []*Import
	types              []*FunctionType
	functions          []*Function
	data               []*Data
	requiredMemorySize uint32
	exports            []*Export
}

func (b *ModuleBuilder) AddFunction(name string, functionType *FunctionType, code *Code) uint32 {
	typeIndex := uint32(len(b.types))
	b.types = append(b.types, functionType)
	// function indices include function imports
	funcIndex := uint32(len(b.functionImports) + len(b.functions))
	b.functions = append(
		b.functions,
		&Function{
			Name:      name,
			TypeIndex: typeIndex,
			Code:      code,
		},
	)
	return funcIndex
}

func (b *ModuleBuilder) AddFunctionImport(module string, name string, functionType *FunctionType) (uint32, error) {
	if len(b.functions) > 0 {
		return 0, errors.New("cannot add function imports after adding functions")
	}

	typeIndex := uint32(len(b.types))
	b.types = append(b.types, functionType)
	funcIndex := uint32(len(b.functionImports))
	b.functionImports = append(
		b.functionImports,
		&Import{
			Module:    module,
			Name:      name,
			TypeIndex: typeIndex,
		},
	)

	return funcIndex, nil
}

func (b *ModuleBuilder) RequireMemory(size uint32) uint32 {
	offset := b.requiredMemorySize
	b.requiredMemorySize += size
	return offset
}

func (b *ModuleBuilder) AddData(offset uint32, value []byte) {
	b.data = append(b.data, &Data{
		// NOTE: currently only one memory is supported
		MemoryIndex: 0,
		Offset: []Instruction{
			InstructionI32Const{Value: int32(offset)},
		},
		Init: value,
	})
}

func (b *ModuleBuilder) Build() *Module {
	// NOTE: currently only one memory is supported
	memories := []*Memory{
		{
			Min: uint32(math.Ceil(float64(b.requiredMemorySize) / float64(MemoryPageSize))),
			Max: nil,
		},
	}

	return &Module{
		Types:     b.types,
		Imports:   b.functionImports,
		Functions: b.functions,
		Memories:  memories,
		Data:      b.data,
		Exports:   b.exports,
	}
}

func (b *ModuleBuilder) ExportMemory(name string) {
	b.AddExport(&Export{
		Name: name,
		Descriptor: MemoryExport{
			MemoryIndex: 0,
		},
	})
}

func (b *ModuleBuilder) AddExport(export *Export) {
	b.exports = append(b.exports, export)
}
