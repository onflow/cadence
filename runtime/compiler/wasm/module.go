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

// Module represents a module
//
type Module struct {
	Name               string
	Types              []*FunctionType
	Imports            []*Import
	Functions          []*Function
	Memories           []*Memory
	Exports            []*Export
	StartFunctionIndex *uint32
	Data               []*Data
}

type ModuleBuilder struct {
	types     []*FunctionType
	functions []*Function
}

func (b *ModuleBuilder) AddFunction(name string, functionType *FunctionType, code *Code) {
	typeIndex := uint32(len(b.types))
	b.types = append(b.types, functionType)
	b.functions = append(b.functions,
		&Function{
			Name:      name,
			TypeIndex: typeIndex,
			Code:      code,
		},
	)
}

func (b *ModuleBuilder) Build() *Module {
	return &Module{
		Types:     b.types,
		Functions: b.functions,
	}
}
