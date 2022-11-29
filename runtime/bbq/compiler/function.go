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

package compiler

import (
	"github.com/onflow/cadence/runtime/activations"
	"github.com/onflow/cadence/runtime/bbq/opcode"
	"github.com/onflow/cadence/runtime/bbq/registers"
)

type function struct {
	name       string
	localCount registers.RegisterCounts
	// TODO: use byte.Buffer?
	code           []opcode.Opcode
	locals         *activations.Activations[*local]
	parameterCount uint16
}

func newFunction(name string, parameterCount uint16) *function {
	return &function{
		name:           name,
		parameterCount: parameterCount,
		locals:         activations.NewActivations[*local](nil),
	}
}

func (f *function) emit(opcode opcode.Opcode) int {
	offset := len(f.code)
	f.code = append(f.code, opcode)
	return offset
}

func (f *function) emitAt(index int, opcode opcode.Opcode) {
	f.code[index] = opcode
}

func (f *function) declareLocal(name string, registerType registers.RegisterType) *local {
	index := f.localCount.NextIndex(registerType)

	local := &local{
		index:   index,
		regType: registerType,
	}

	f.locals.Set(name, local)

	return local
}

func (f *function) findLocal(name string) *local {
	return f.locals.Find(name)
}
